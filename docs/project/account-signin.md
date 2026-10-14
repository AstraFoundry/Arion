# Account sign-in

Desktop hosted-sync sign-in. The UI calls `runtime.Actions` and reads `runtime.State.Account`. It does not import `arion.local/desktop/internal/account`. The composition root (`bootstrap`) owns the service; a future sync client reads the access token from `App.Account().Session()`.

Behavior is specified in [ADR 015](../decisions/015-hosted-relay-account-authorization.md), SYNC-10, and MEM-A17. This page is the call surface.

## Startup

`account.New` and `bootstrap.New` do not discover the issuer, open a port, read the credential store, or refresh a token. Dictation does not wait on this service. A process restart leaves any stored refresh token where it is and shows `AccountSignedOut` until the user signs in or the UI calls `RefreshAccount`.

## What views call

These `runtime.Actions` methods return immediately. Work continues off the main thread, and the result is posted onto `runtime.State.Account`.

| Method | Effect |
| --- | --- |
| `SignIn()` | System browser, one loopback callback. Phase `signing_in`, then `signed_in` or `signed_out`. |
| `CancelSignIn()` | Invalidates the in-flight sign-in or refresh. A callback that arrives afterwards is not exchanged. A refresh token already stored for that attempt is deleted and revoked. |
| `RefreshAccount()` | Rotates the stored refresh token and replaces the in-memory access token. Phase `refreshing`. |
| `SignOut()` | Revokes the refresh token, then deletes that credential item. Local keys and local memory stay. |
| `JoinHostedSyncWaitlist()` | Resolves and joins the hosted-sync waitlist. |

One sign-in or refresh runs at a time. A second request fails with `SIGN_IN_IN_PROGRESS`. Join does the same while one is running.

## What views read

`State.Account` has no access token and no refresh token.

| Field | Values |
| --- | --- |
| `Phase` | `signed_out`, `signing_in`, `signed_in`, `refreshing`, `signing_out` |
| `Subject`, `Name`, `Email`, `Picture`, `Locale` | Profile from the ID token, for display |
| `HostedSync` | `ent` contains `arion.sync.hosted`. The sync server remains the authority |
| `ErrorCode`, `ErrorDetail` | Content-free failure. Map them through the copy catalog |
| `Waitlist.Phase` | empty, `joining`, `joined_pending`, `already_entitled`, `closed`, `error` |
| `Waitlist.EntryID`, `Code`, `HTTPStatus` | Join payload |
| `SignOut` | `Attempted`, `RefreshTokenDeleted`, `RevokeAttempted`, `Revoked`, `RevokeDetail` |

`ErrorDetail` is an OAuth error code (`access_denied`, `invalid_grant`, `server_error`), an `http_N` label, or another short token such as `issuer_mismatch` or `s256_required`. It is not `error_description`, a token, or an authorization code. Show catalog copy for `ErrorCode`, not `Error()` from the service.

The loopback response is an empty `200` with `Cache-Control: no-store`. The onboarding copy that tells the user to return to Arion lives in the app, not on that page.

## Service API

Package `arion.local/desktop/internal/account`. `bootstrap` is the only production caller.

```go
func New(cfg Config, deps Dependencies) (*Service, error)
func (s *Service) SignIn(ctx context.Context) (Session, error)
func (s *Service) Cancel()
func (s *Service) Refresh(ctx context.Context) (Session, error)
func (s *Service) SignOut(ctx context.Context, armed func()) (SignOutResult, error)
func (s *Service) JoinHostedSyncWaitlist(ctx context.Context) (WaitlistResult, error)
func (s *Service) Session() (Session, bool)
```

`Dependencies.Store` is a `RefreshTokenStore` (`Put`, `Get`, `Delete`). `Delete` removes the refresh token only. `Dependencies.Browser` opens the authorization URL. `New` returns an error when either is nil.

`Session.AccessToken` is for the composition root's bearer calls. `TokenType` is `Bearer` when the token response omits it. `HasHostedSync()` reports the entitlement. The refresh token is never on `Session`.

`SignOut`'s `armed` callback runs after the in-flight attempt is marked as sign-out and before `SignOut` waits for it. `bootstrap` cancels the sign-in context there. Pass nil from tests. The returned `SignOutResult.RevokeError` is also the error from `SignOut` when revocation fails; the local item is still deleted.

`Config` zero value:

| Field | Default |
| --- | --- |
| `Issuer` | `https://astrafoundry.si` |
| `ClientID` | `arion-desktop` |
| `Scopes` | `openid`, `profile`, `offline_access`, `arion.sync`, `waitlist:join` |
| `Resource` | `https://api.arion.astrafoundry.si` (RFC 8707 `resource`) |
| `WaitlistID` | empty: resolve key `hosted-sync` from `GET {issuer}/api/v1/catalog/products/arion` |
| `SignInTimeout` | 5 minutes. Negative: wait only on the caller's context |
| `HTTP` | 30s client that does not follow redirects |

There is no client secret. The public client is registered at the issuer. Discovery is `{issuer}/.well-known/openid-configuration`; the issuer string must match the document. Authorize, token, and revocation URLs come from that document. PKCE is S256. If the document lists code-challenge methods and omits S256, sign-in fails with `s256_required` and the browser stays closed.

The access token is decoded for `sub`, `ent`, and the ID-token profile only. The signature is not checked on device. A missing `ent` means no hosted entitlement. A non-array `ent` is `ENTITLEMENT` / `ent_type`. A malformed ID token does not fail sign-in.

Refresh replaces the stored refresh token when the response includes a new one, and keeps the current one when it does not. `invalid_grant` deletes the stored token and clears the session.

### Waitlist

`JoinHostedSyncWaitlist` requires an in-memory access token (`NOT_SIGNED_IN` otherwise). With an empty `WaitlistID` it reads the catalog without a bearer token. Catalog status `closed`, `paused`, `draft`, `not_open`, or `waitlist_closed` returns `WaitlistClosed` and does not POST.

Otherwise it `POST`s `{issuer}/api/v1/me/waitlists/{id}/entries` with `Authorization: Bearer` and body `{"answers":{}}`.

| Response | `WaitlistResult.Status` | Error |
| --- | --- | --- |
| `201` with status `pending` or empty | `joined_pending` | nil |
| `409` `already_entitled`, or `entitled: true` | `already_entitled` | nil |
| `409` `already_pending`, status `pending`, or an empty `409` while `ent` lacks hosted sync | `joined_pending` | nil |
| `409` with an empty body while `ent` contains hosted sync | `already_entitled` | nil |
| body code/status closed, at any HTTP status | `closed` | nil |
| anything else | `error` | `*Error` `WAITLIST` |

An empty `409` has no discriminator in the platform contract, so the access token's `ent` claim splits it. An explicit body code wins over that claim.

### Errors

`*Error` implements `Is` by code. `CodeOf` and `DetailOf` are the UI mapping. Codes: `SIGN_IN_IN_PROGRESS`, `SIGN_IN_TIMEOUT`, `SIGN_IN_CANCELLED`, `STATE_MISMATCH`, `AUTHORIZATION`, `TOKEN`, `DISCOVERY`, `SECURE_STORE`, `NOT_SIGNED_IN`, `REFRESH`, `REVOKE`, `WAITLIST`, `ENTITLEMENT`, `BROWSER`.

Sentinels: `ErrSignInInProgress`, `ErrSignInTimeout`, `ErrSignInCancelled`, `ErrStateMismatch`, `ErrNotSignedIn`, `ErrNotFound`, `ErrUnsupported`, `ErrRefreshTokenTooLarge`.

## Credential store

`infrastructure/keystore` implements `RefreshTokenStore`.

| OS | Item | Mechanism |
| --- | --- | --- |
| macOS | login Keychain, service `Arion`, account `account-refresh-token` | `/usr/bin/security` add/find/delete-generic-password |
| Windows | Credential Manager target `Arion/account-refresh-token`, generic, local-machine persistence | `CredWriteW` / `CredReadW` / `CredDeleteW` |
| other | — | `ErrUnsupported` when a method is called. `Open` still succeeds, so dictation starts |

The vocabulary encryption key is a different item: account `vocabulary-key`, written by `infrastructure/secrets` with SecItem on macOS and DPAPI on Windows ([ADR 005](../decisions/005-local-encryption.md)). Sign-out does not address that account. macOS passes the refresh token as an argument of `security` for the duration of the call, visible to other processes of the same user; at rest it is a Keychain generic password. This package uses the `security` tool, the same approach as `github.com/zalando/go-keyring`, so it does not add that module, does not link the SecItem helper that owns the vocabulary key, and `GOOS=darwin go vet` does not need the macOS SDK. Windows blobs are at most 2560 bytes (`ErrRefreshTokenTooLarge`).

`infrastructure/browser` opens `http` and `https` URLs only: `/usr/bin/open` on macOS, `rundll32 url.dll,FileProtocolHandler` on Windows, `xdg-open` elsewhere.

Native Keychain and Credential Manager behavior is verified by compiling the platform files. It has not been executed on those operating systems.
