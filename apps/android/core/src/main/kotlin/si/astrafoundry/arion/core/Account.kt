package si.astrafoundry.arion.core

import kotlinx.coroutines.flow.StateFlow

/**
 * The AstraFoundry account the app signs in to, as on the desktop
 * (docs/project/account-signin.md), with a mobile public client and an app
 * redirect instead of a loopback.
 */
data class AccountConfig(
    val issuer: String = "https://astrafoundry.si",
    val clientId: String = "arion-android",
    val redirect: String = "arion://auth/callback",
    val scopes: List<String> = listOf("openid", "profile", "offline_access", "arion.sync"),
    val resource: String = "https://api.arion.astrafoundry.si",
) {
    /** The account portal, `{issuer}/account`. */
    val portal: String get() = "$issuer/account"
}

enum class AccountPhase { SignedOut, Restoring, SigningIn, SignedIn, SigningOut }

/** The signed-in account's answer to sharing usage data, held by the platform. */
enum class UsageSharing { Unknown, Loading, Undecided, Shared, NotShared, Saving }

/** Content-free failures, mapped to copy by the views. A cancelled sign-in shows nothing. */
enum class AccountFailure {
    InProgress, Denied, Authorization, Token, Discovery, Store, SessionEnded, Refresh,
    /** Signed out here, but the server did not confirm the revocation. */
    RevokeUnconfirmed,
    UsageConsent, Generic,
}

data class AccountProfile(val subject: String, val name: String, val email: String, val picture: String?) {
    /** One or two letters for the avatar when there is no picture. */
    val initials: String
        get() = (name.ifEmpty { email }).split(' ', '@', '.').filter { it.isNotEmpty() }
            .take(2).joinToString("") { it.first().uppercase() }
}

data class AccountState(
    val phase: AccountPhase = AccountPhase.SignedOut,
    /** Kept while restoring, so a restart shows the profile at once. */
    val profile: AccountProfile? = null,
    val usageSharing: UsageSharing = UsageSharing.Unknown,
    val failure: AccountFailure? = null,
) {
    val signedIn: Boolean get() = phase == AccountPhase.SignedIn
    val busy: Boolean
        get() = phase in setOf(AccountPhase.Restoring, AccountPhase.SigningIn, AccountPhase.SigningOut) ||
            usageSharing == UsageSharing.Saving
}

/** What the views call. Every method returns at once; results arrive in [state]. */
interface AccountServing {
    val state: StateFlow<AccountState>
    /** Restores a stored account at launch, without a browser. */
    fun restore()
    /**
     * Opens the browser sign-in (Custom Tabs). A checked usage box records
     * consent only after a successful sign-in (ADR 044).
     */
    fun signIn(shareUsage: Boolean)
    fun cancel()
    /** Revokes the refresh token, then deletes it. Local data stays. */
    fun signOut()
    fun setUsageSharing(share: Boolean)
    /** Sends changed daily usage snapshots when signed in and sharing. */
    fun reportUsage()
}
