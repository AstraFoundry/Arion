package si.astrafoundry.arion.account

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import si.astrafoundry.arion.cloud.http
import si.astrafoundry.arion.core.AccountConfig
import si.astrafoundry.arion.core.AccountFailure
import si.astrafoundry.arion.core.AccountPhase
import si.astrafoundry.arion.core.AccountProfile
import si.astrafoundry.arion.core.AccountServing
import si.astrafoundry.arion.core.AccountState
import si.astrafoundry.arion.core.Discovery
import si.astrafoundry.arion.core.Oidc
import si.astrafoundry.arion.core.TokenResponse
import si.astrafoundry.arion.core.UsageSharing
import si.astrafoundry.arion.storage.Secrets
import si.astrafoundry.arion.usage.UsageLedger
import java.io.IOException
import java.time.LocalDate
import java.util.UUID

private class AccountError(val failure: AccountFailure?) : Exception(failure?.name)

/**
 * The AstraFoundry account on the phone: sign-in in the browser with PKCE,
 * a refresh token encrypted with an Android Keystore key, the platform's
 * usage consent and the consented daily usage report, as the desktop's
 * account service.
 */
class AccountService private constructor(private val context: Context) : AccountServing {
    private val config = AccountConfig()
    private val secrets = Secrets(context)
    private val prefs = context.getSharedPreferences("arion.account", Context.MODE_PRIVATE)
    private val ledger = UsageLedger(context)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private val _state = MutableStateFlow(AccountState(profile = storedProfile()))
    override val state: StateFlow<AccountState> = _state

    private var accessToken: String? = null
    private var expiry = 0L
    private var generation = 0
    private var reporting = false
    /** The sign-in waiting for its browser callback. */
    private var pending: Pending? = null

    private class Pending(val generation: Int, val discovery: Discovery, val verifier: String, val state: String, val shareUsage: Boolean)

    private fun update(change: AccountState.() -> AccountState) { _state.value = _state.value.change() }

    override fun restore() {
        if (_state.value.phase != AccountPhase.SignedOut) return
        if (!secrets.has(TOKEN)) {
            update { copy(profile = null) }
            return
        }
        val current = generation
        update { copy(phase = AccountPhase.Restoring) }
        scope.launch {
            try {
                refresh()
                if (current != generation) return@launch
                update { copy(phase = AccountPhase.SignedIn, failure = null) }
                readConsent()
                reportUsage()
            } catch (e: AccountError) {
                if (current != generation) return@launch
                // A transient failure keeps the account; the next launch tries again.
                if (e.failure == AccountFailure.SessionEnded) signedOut(AccountFailure.SessionEnded)
                else update { copy(phase = AccountPhase.SignedIn, failure = e.failure) }
            }
        }
    }

    override fun signIn(shareUsage: Boolean) {
        if (_state.value.busy) {
            update { copy(failure = AccountFailure.InProgress) }
            return
        }
        val current = ++generation
        update { copy(phase = AccountPhase.SigningIn, failure = null) }
        scope.launch {
            try {
                val discovery = discover()
                val verifier = Oidc.verifier()
                val stateToken = Oidc.verifier()
                pending = Pending(current, discovery, verifier, stateToken, shareUsage)
                val url = Oidc.authorizeUrl(discovery, config, stateToken, Oidc.challenge(verifier))
                context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
            } catch (e: AccountError) {
                if (current == generation) signedOut(e.failure)
            } catch (e: android.content.ActivityNotFoundException) {
                if (current == generation) signedOut(AccountFailure.Generic)
            }
        }
    }

    /** The browser returned to `arion://auth/callback`. */
    fun complete(uri: Uri) {
        val pending = pending ?: return
        this.pending = null
        if (pending.generation != generation) return
        scope.launch {
            try {
                uri.getQueryParameter("error")?.let {
                    throw AccountError(if (it == "access_denied") AccountFailure.Denied else AccountFailure.Authorization)
                }
                if (uri.getQueryParameter("state") != pending.state) throw AccountError(AccountFailure.Authorization)
                val code = uri.getQueryParameter("code") ?: throw AccountError(AccountFailure.Authorization)
                val tokens = token(pending.discovery, mapOf(
                    "grant_type" to "authorization_code", "code" to code, "redirect_uri" to config.redirect,
                    "client_id" to config.clientId, "code_verifier" to pending.verifier, "resource" to config.resource,
                ))
                if (pending.generation != generation) return@launch
                accept(tokens, pending.discovery)
                update { copy(phase = AccountPhase.SignedIn) }
                if (pending.shareUsage) setUsageSharing(true) else readConsent()
                reportUsage()
            } catch (e: AccountError) {
                if (pending.generation == generation) signedOut(e.failure)
            }
        }
    }

    override fun cancel() {
        generation++
        pending = null
        if (_state.value.phase == AccountPhase.SigningIn) signedOut(null)
    }

    override fun signOut() {
        generation++
        val refresh = runCatching { secrets.read(TOKEN) }.getOrNull()
        update { copy(phase = AccountPhase.SigningOut) }
        scope.launch {
            var confirmed = refresh == null
            if (refresh != null) {
                runCatching {
                    val endpoint = discover().revocation_endpoint ?: return@runCatching
                    val result = http(endpoint, "POST", body = Oidc.form(mapOf("token" to refresh, "token_type_hint" to "refresh_token", "client_id" to config.clientId)),
                        contentType = "application/x-www-form-urlencoded")
                    confirmed = result.status == 200
                }
            }
            val deleted = runCatching { secrets.delete(TOKEN) }.isSuccess
            prefs.edit().clear().apply()
            signedOut(if (!deleted) AccountFailure.Store else if (confirmed) null else AccountFailure.RevokeUnconfirmed)
            update { copy(profile = null) }
        }
    }

    override fun setUsageSharing(share: Boolean) {
        if (!_state.value.signedIn) return
        update { copy(usageSharing = UsageSharing.Saving) }
        scope.launch {
            try {
                val answer = consent("PUT", share)
                update { copy(usageSharing = if (answer == true) UsageSharing.Shared else UsageSharing.NotShared, failure = null) }
                if (answer == true) reportUsage()
            } catch (e: Exception) {
                update { copy(failure = AccountFailure.UsageConsent) }
                readConsent()
            }
        }
    }

    /**
     * Sends the days whose counts changed since they were last confirmed,
     * within the platform's 31-day window, as the desktop's usage upload.
     */
    override fun reportUsage() {
        if (!_state.value.signedIn || _state.value.usageSharing != UsageSharing.Shared || reporting) return
        reporting = true
        scope.launch {
            try {
                val token = currentToken()
                val today = LocalDate.now()
                val sent = prefs.getStringSet(SENT, emptySet())!!.associate { it.substringBefore('=') to it.substringAfter('=') }.toMutableMap()
                for (day in ledger.days(today.toString(), 31)) {
                    if (day.sessions == 0 && day.date != today.toString()) continue
                    val snapshot = "${day.sessions},${day.speechMs},${day.words}"
                    if (sent[day.date] == snapshot) continue
                    val body = buildJsonObject {
                        put("sourceId", sourceId); put("productSlug", "arion"); put("date", day.date); put("visited", true)
                        put("sessions", day.sessions); put("speechMs", day.speechMs); put("words", day.words)
                    }.toString()
                    val result = http("${config.issuer}/api/v1/usage", "PUT", mapOf("Authorization" to "Bearer $token"), body)
                    if (result.status !in 200..299 || !Oidc.confirmsReport(result.body, day)) break
                    sent[day.date] = snapshot
                }
                val oldest = today.minusDays(31).toString()
                prefs.edit().putStringSet(SENT, sent.filterKeys { it >= oldest }.map { "${it.key}=${it.value}" }.toSet()).apply()
            } catch (e: Exception) {
                // The next launch or return to the app sends what is left.
            } finally {
                reporting = false
            }
        }
    }

    private suspend fun discover(): Discovery {
        val result = try {
            http("${config.issuer}/.well-known/openid-configuration")
        } catch (e: IOException) {
            throw AccountError(AccountFailure.Discovery)
        }
        if (result.status != 200) throw AccountError(AccountFailure.Discovery)
        val discovery = runCatching { Oidc.json.decodeFromString(Discovery.serializer(), result.body) }.getOrNull()
            ?: throw AccountError(AccountFailure.Discovery)
        if (!Oidc.accepts(discovery, config)) throw AccountError(AccountFailure.Discovery)
        return discovery
    }

    private suspend fun token(discovery: Discovery, form: Map<String, String>): TokenResponse {
        val result = try {
            http(discovery.token_endpoint, "POST", body = Oidc.form(form), contentType = "application/x-www-form-urlencoded")
        } catch (e: IOException) {
            throw AccountError(AccountFailure.Token)
        }
        if (result.status == 400 && result.body.contains("\"invalid_grant\"")) throw AccountError(AccountFailure.SessionEnded)
        if (result.status != 200) throw AccountError(AccountFailure.Token)
        return runCatching { Oidc.json.decodeFromString(TokenResponse.serializer(), result.body) }.getOrNull()
            ?: throw AccountError(AccountFailure.Token)
    }

    private suspend fun accept(tokens: TokenResponse, discovery: Discovery) {
        accessToken = tokens.access_token
        expiry = tokens.expires_in?.let { System.currentTimeMillis() + (it * 1000).toLong() } ?: 0L
        tokens.refresh_token?.let {
            try { secrets.write(TOKEN, it) } catch (e: Exception) { throw AccountError(AccountFailure.Store) }
        }
        val claims = Oidc.claims(tokens.id_token)
        var profile = AccountProfile(claims["sub"] ?: Oidc.claims(tokens.access_token)["sub"].orEmpty(), claims["name"].orEmpty(), claims["email"].orEmpty(), claims["picture"])
        discovery.userinfo_endpoint?.let { endpoint ->
            runCatching {
                val result = http(endpoint, headers = mapOf("Authorization" to "Bearer ${tokens.access_token}"))
                if (result.status == 200) {
                    val info = Oidc.fields(result.body)
                    if (profile.subject.isEmpty() || info["sub"] == profile.subject) {
                        profile = profile.copy(subject = info["sub"] ?: profile.subject, name = info["name"] ?: profile.name,
                            email = info["email"] ?: profile.email, picture = info["picture"] ?: profile.picture)
                    }
                }
            }
        }
        // The same subject keeps a known name and email when the new claims omit them.
        _state.value.profile?.takeIf { it.subject == profile.subject }?.let { previous ->
            profile = profile.copy(name = profile.name.ifEmpty { previous.name }, email = profile.email.ifEmpty { previous.email })
        }
        update { copy(profile = profile) }
        prefs.edit().putString("sub", profile.subject).putString("name", profile.name).putString("email", profile.email)
            .putString("picture", profile.picture).apply()
    }

    private suspend fun refresh() {
        val refresh = runCatching { secrets.read(TOKEN) }.getOrNull() ?: throw AccountError(AccountFailure.SessionEnded)
        val discovery = discover()
        try {
            accept(token(discovery, mapOf("grant_type" to "refresh_token", "refresh_token" to refresh, "client_id" to config.clientId, "resource" to config.resource)), discovery)
        } catch (e: AccountError) {
            if (e.failure == AccountFailure.SessionEnded) runCatching { secrets.delete(TOKEN) }
            throw if (e.failure == AccountFailure.Token) AccountError(AccountFailure.Refresh) else e
        }
    }

    private suspend fun currentToken(): String {
        accessToken?.let { if (expiry == 0L || expiry - System.currentTimeMillis() > 60_000) return it }
        refresh()
        return accessToken ?: throw AccountError(AccountFailure.Refresh)
    }

    private suspend fun readConsent() {
        update { copy(usageSharing = UsageSharing.Loading) }
        try {
            val answer = consent("GET", null)
            update { copy(usageSharing = when (answer) { true -> UsageSharing.Shared; false -> UsageSharing.NotShared; null -> UsageSharing.Undecided }) }
        } catch (e: Exception) {
            update { copy(usageSharing = UsageSharing.Unknown, failure = AccountFailure.UsageConsent) }
        }
    }

    private suspend fun consent(method: String, accepted: Boolean?): Boolean? {
        val token = currentToken()
        val url = "${config.issuer}/api/v1/usage/consent" + if (method == "GET") "?productSlug=arion" else ""
        val body = accepted?.let { buildJsonObject { put("productSlug", "arion"); put("accepted", it) }.toString() }
        val result = http(url, method, mapOf("Authorization" to "Bearer $token"), body)
        if (result.status != 200) throw AccountError(AccountFailure.UsageConsent)
        return Oidc.consent(result.body, accepted)
    }

    /** A random source ID per installation, 32 hex characters. */
    private val sourceId: String
        get() = prefs.getString(SOURCE, null)?.takeIf { it.length == 32 }
            ?: UUID.randomUUID().toString().replace("-", "").also { prefs.edit().putString(SOURCE, it).apply() }

    private fun signedOut(failure: AccountFailure?) {
        accessToken = null
        expiry = 0L
        pending = null
        update { copy(phase = AccountPhase.SignedOut, usageSharing = UsageSharing.Unknown, failure = failure) }
    }

    private fun storedProfile(): AccountProfile? {
        val subject = prefs.getString("sub", null) ?: return null
        return AccountProfile(subject, prefs.getString("name", "")!!, prefs.getString("email", "")!!, prefs.getString("picture", null))
    }

    companion object {
        private const val TOKEN = "account-refresh-token"
        private const val SOURCE = "usage.sourceId"
        private const val SENT = "usage.sent"

        @Volatile private var instance: AccountService? = null

        fun get(context: Context): AccountService =
            instance ?: synchronized(this) { instance ?: AccountService(context.applicationContext).also { instance = it } }
    }
}

/** Receives `arion://auth/callback` from the browser, hands it over and closes. */
class AuthCallbackActivity : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        intent?.data?.let { AccountService.get(this).complete(it) }
        startActivity(packageManager.getLaunchIntentForPackage(packageName)?.addFlags(Intent.FLAG_ACTIVITY_REORDER_TO_FRONT))
        finish()
    }
}
