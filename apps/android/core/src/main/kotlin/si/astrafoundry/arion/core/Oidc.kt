package si.astrafoundry.arion.core

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.net.URLEncoder
import java.security.MessageDigest
import java.security.SecureRandom
import java.util.Base64

/** The parts of OpenID Connect discovery the account needs, as the desktop's `oidc.go`. */
@Serializable
data class Discovery(
    val issuer: String,
    val authorization_endpoint: String,
    val token_endpoint: String,
    val revocation_endpoint: String? = null,
    val userinfo_endpoint: String? = null,
    val code_challenge_methods_supported: List<String>? = null,
)

@Serializable
data class TokenResponse(val access_token: String, val refresh_token: String? = null, val id_token: String? = null, val expires_in: Double? = null)

object Oidc {
    val json = Json { ignoreUnknownKeys = true }

    fun verifier(): String = ByteArray(32).also { SecureRandom().nextBytes(it) }.let(::base64Url)

    fun challenge(verifier: String): String = base64Url(MessageDigest.getInstance("SHA-256").digest(verifier.toByteArray()))

    private fun base64Url(bytes: ByteArray): String = Base64.getUrlEncoder().withoutPadding().encodeToString(bytes)

    private fun encode(value: String) = URLEncoder.encode(value, Charsets.UTF_8).replace("+", "%20")

    fun form(values: Map<String, String>): String = values.entries.joinToString("&") { "${encode(it.key)}=${encode(it.value)}" }

    fun authorizeUrl(discovery: Discovery, config: AccountConfig, state: String, challenge: String): String {
        val query = form(linkedMapOf(
            "client_id" to config.clientId,
            "redirect_uri" to config.redirect,
            "response_type" to "code",
            "scope" to config.scopes.joinToString(" "),
            "state" to state,
            "code_challenge" to challenge,
            "code_challenge_method" to "S256",
            "resource" to config.resource,
        ))
        val separator = if ('?' in discovery.authorization_endpoint) "&" else "?"
        return discovery.authorization_endpoint + separator + query
    }

    /** Whether the discovery document is this issuer's and supports S256. */
    fun accepts(discovery: Discovery, config: AccountConfig): Boolean =
        discovery.issuer.trimEnd('/') == config.issuer.trimEnd('/') && (discovery.code_challenge_methods_supported?.contains("S256") ?: true)

    /** The profile claims of a JWT; the signature is not checked on the device. */
    fun claims(token: String?): Map<String, String> {
        val payload = token?.split('.')?.getOrNull(1) ?: return emptyMap()
        val text = runCatching { String(Base64.getUrlDecoder().decode(payload.padEnd((payload.length + 3) / 4 * 4, '='))) }.getOrNull() ?: return emptyMap()
        return fields(text)
    }

    /** The string fields of a JSON object, such as a UserInfo response. */
    fun fields(text: String): Map<String, String> {
        val obj = runCatching { json.parseToJsonElement(text).jsonObject }.getOrNull() ?: return emptyMap()
        return obj.mapNotNull { (k, v) -> runCatching { v.jsonPrimitive.contentOrNull }.getOrNull()?.let { k to it } }.toMap()
    }

    /**
     * The platform's usage consent answer, as the desktop's `usageConsent`:
     * true when sharing; null when the account has not decided. Throws on a
     * malformed or mismatched answer.
     */
    fun consent(body: String, want: Boolean?): Boolean? {
        val data = json.parseToJsonElement(body).jsonObject["data"]?.jsonObject ?: throw IllegalStateException("consent")
        val accepted = data["accepted"]?.jsonPrimitive?.booleanOrNull ?: false
        val decided = data["decided"]?.jsonPrimitive?.booleanOrNull ?: false
        val version = data["version"]?.jsonPrimitive?.contentOrNull
        if (accepted && (version != "usage-v1" || !decided)) throw IllegalStateException("consent")
        if (want != null && (accepted != want || !decided)) throw IllegalStateException("consent")
        if (!decided) return null
        return accepted
    }

    /** Whether a usage report response confirms at least what was sent. */
    fun confirmsReport(body: String, day: UsageDay): Boolean = runCatching {
        val root = json.parseToJsonElement(body).jsonObject
        if (root["success"]?.jsonPrimitive?.booleanOrNull != true) return false
        val data = root["data"]?.jsonObject ?: return false
        data["productSlug"]?.jsonPrimitive?.contentOrNull == "arion" && data["date"]?.jsonPrimitive?.contentOrNull == day.date &&
            (data["sessions"]?.jsonPrimitive?.contentOrNull?.toIntOrNull() ?: -1) >= day.sessions &&
            (data["speechMs"]?.jsonPrimitive?.contentOrNull?.toLongOrNull() ?: -1) >= day.speechMs &&
            (data["words"]?.jsonPrimitive?.contentOrNull?.toIntOrNull() ?: -1) >= day.words
    }.getOrDefault(false)
}
