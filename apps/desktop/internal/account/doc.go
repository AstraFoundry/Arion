// Package account is the desktop hosted-sync sign-in service.
//
// It is an OIDC public client (RFC 8252): one in-flight sign-in, a one-shot
// listener on 127.0.0.1 with a random port, PKCE S256, and the system browser.
// The access token stays in memory. The refresh token is written only through
// RefreshTokenStore, which the composition root binds to the OS credential
// store. Sign-out revokes that refresh token and deletes only that item.
//
// Constructing a Service does not dial the issuer. Dictation stays offline
// until the user starts sign-in, refresh, waitlist join, or sign-out.
//
// The UI must not import this package. It calls runtime.Actions and reads
// runtime.State.Account. See docs/project/account-signin.md.
package account
