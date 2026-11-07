# Account sign-in

Desktop hosted-sync sign-in. The UI calls `runtime.Actions` and reads `runtime.State.Account`. It does not import `arion.local/desktop/internal/account`. The composition root (`bootstrap`) owns the service; product notifications and a future sync client read the access token from `App.Account().Session()`.

Behavior is specified in [ADR 015](../decisions/015-hosted-relay-account-authorization.md), SYNC-10, and MEM-A17. This page is the call surface.

## Startup

`account.New` and `bootstrap.New` remain idle. After native readiness, the composition root reads the OS credential store off the main thread and automatically restores an existing account with its refresh token. Dictation and window startup never wait for account networking. Device-only, offline and scripted E2E profiles do not start background maintenance.

Maintenance checks wall-clock deadlines every five seconds, including after sleep. Tokens renew before expiry (10% of the remaining lifetime, capped at one minute). Missing expiry schedules no proactive refresh. Transient transport failures, HTTP 408/429/5xx, OAuth `server_error`/`temporarily_unavailable`, and credential read/write errors retry with exponential backoff and jitter, starting at five seconds and capped at five minutes. Existing account identity stays visible; transient refresh failures show no prompt or toast. A restart while offline shows automatic restoration until connectivity returns. Display name and email for the restored subject are read from preferences; picture bytes are not persisted.

Missing credentials or `invalid_grant` stop recovery and allow sign-in. Other permanent failures stop automatic retries and show the existing localized error. Refresh, sign-in and sign-out are serialized by account phase and service locks; generation checks discard stale results. Sign-out invalidates queued recovery before revocation, and app quit stops the maintenance timer. No background operation opens a browser.

## What views call

These `runtime.Actions` methods return immediately. Work continues off the main thread, and the result is posted onto `runtime.State.Account`.

| Method | Effect |
| --- | --- |
| `SignIn(shareUsage)` | System browser, one loopback callback. Phase `signing_in`, then `signed_in` or `signed_out`. `shareUsage` is an explicit consent argument. Onboarding passes its default-off checkbox choice; Settings passes `false`. |
| `CancelSignIn()` | Invalidates the in-flight sign-in or refresh. A callback that arrives afterwards is not exchanged. A refresh token already stored for that attempt is deleted and revoked. |
| `RefreshAccount()` | Explicit entitlement recheck; rotates the stored refresh token and replaces the in-memory access token. Phase `refreshing`. |
| `SignOut()` | Revokes the refresh token, then deletes that credential item. Local keys and local memory stay. |
| `SetUsageSharing(share)` | Records the signed-in account's answer to sharing usage data, from the explicitly checked onboarding choice after sign-in or the switch in Settings › Account. `UsageSaving` while it runs; the platform's confirmation sets `UsageSharing` and starts or stops reporting. |
| `CheckStoredAccount()` | Reads whether a refresh token is stored, with `Service.Stored`. This does nothing unless `Phase` is `signed_out` and `Credential` is unknown. A result is dropped when a sign-in, refresh, cancel or sign-out started during the read. |

One sign-in or refresh runs at a time. A second request fails with `SIGN_IN_IN_PROGRESS`. `JoinHostedSyncWaitlist` does the same while one is running; views do not call it.

## What views read

`State.Account` has no access token and no refresh token. The last signed-in `Subject`, `Name` and `Email` are also stored in preference schema v1 as additive `accountIssuer` / `accountSubject` / `accountName` / `accountEmail` so Settings can show the profile line after a restart. They are display cache, not credentials. A matching issuer and `Subject` fills empty session fields; a different account or sign-out clears them. Picture bytes stay in memory only.

| Field | Values |
| --- | --- |
| `Phase` | `signed_out`, `signing_in`, `signed_in`, `refreshing`, `signing_out` |
| `Subject`, `Name`, `Email`, `Picture`, `Locale` | Profile from the ID token or UserInfo, for display. Empty name and email for the same subject are filled from preferences |
| `HostedSync` | `ent` contains `arion.sync.hosted`. The sync server remains the authority |
| `ErrorCode`, `ErrorDetail` | Content-free failure. Map them through the copy catalog |
| `Waitlist.Phase` | empty, `joining`, `joined_pending`, `already_entitled`, `closed`, `error` |
| `Waitlist.EntryID`, `Code`, `HTTPStatus` | Join payload |
| `SignOut` | `Attempted`, `RefreshTokenDeleted`, `RevokeAttempted`, `Revoked`, `RevokeDetail` |
| `Credential` | While signed out: empty (not read since the last account change), `checking`, `absent`, `stored`, or `unreadable`. A failed read sets `unreadable` and `ErrorCode` `SECURE_STORE` with detail `read` or `unsupported`; it is never shown as `absent`. Sign-in, refresh, cancel and sign-out reset it to empty, so the view reads the store again |

`ErrorDetail` is an OAuth error code (`access_denied`, `invalid_grant`, `server_error`), an `http_N` label, or another short token such as `issuer_mismatch` or `s256_required`. It is not `error_description`, a token, or an authorization code. Show catalog copy for `ErrorCode`, not `Error()` from the service.

The loopback answers `200` with `Cache-Control: no-store`, `Referrer-Policy: no-referrer` and a CSP that allows no scripts. The body is `Dependencies.CallbackPage`, rendered when the callback arrives; nil leaves it empty. `bootstrap` supplies a static page in the interface language (`AccCallbackTitle`, `AccCallbackBody`, `AccCallbackOpen`) with an **Open Arion** link to `arion://open`. The page does not report the outcome; the app does. `arion` is declared as `urlSchemes` in `desktop.json`, so the packaged app registers it (macOS Info.plist, Windows installer). An `arion:` URL focuses onboarding while it is open and Settings otherwise, except `arion://feedback`, which opens the [feedback window](agent-feedback.md); a second instance started with such a URL does not also open Settings.

## Views

Account setup is optional and shares the welcome page. That page is still shown when the account is disabled, without sign-in controls. Signed out, it offers **Continue with Google**, **Continue with GitHub** and **Continue with email**, or one **Sign in** button when the issuer is custom. Each passes the onboarding usage-sharing checkbox to `SignIn(shareUsage)`. Below the buttons, the terms checkbox starts checked; its single-line label contains underlined **Terms of Use** and **Privacy Policy** links. Unchecking it disables every browser sign-in method. A separate optional usage-sharing checkbox starts unchecked and uses an underlined **usage data** keyword within its single-line label to open the localized disclosure on the Arion site. The muted footer button is **Continue locally**, which moves on with no account and no network, independently of both choices. While sign-in is running, the page shows the browser wait with Cancel and does not move on. After sign-in it shows the account card, without a separate sharing page or switch. Only an explicitly checked usage choice records consent after successful sign-in; cancel/failure never carries that request into a later refresh ([ADR 044](../decisions/044-onboarding-consent.md)).

The settings sidebar ends with an account card, as in macOS System Settings: the profile picture (or initials) with name and email, or the person glyph and "Account" while signed out. Its hover and selected fills are the sidebar's neutral grays. It opens Settings › Account, which shows the account card: Sign in, or the profile with Sign out. When `Credential` is `stored`, the card reports automatic restoration and offers **Sign out**, which revokes and deletes that token without requiring a session in memory. There is no Reconnect control. While the store is being read, the card shows the checking row. When the store can't be read, the card offers Sign in with the store error under it. When signed in with the hosted-sync entitlement, a Hosted sync card follows. Without the entitlement, views do not offer a waitlist. Links to the AstraFoundry account portal follow (`{Portal}/profile`, `/security`, `/connections`, `/products`, `/data`), opened in the browser. `Account.Portal` is `{issuer}/account` for the standard profile only.

`features/signin` draws the account card and, when the entitlement is present, the hosted-sync card. `entities/account` maps state to copy keys.

| State | Shown |
| --- | --- |
| `HostedSync` false | No hosted-sync card. Local features stay available. Views do not call `JoinHostedSyncWaitlist` |
| `HostedSync` true | Hosted sync is available. Setting up this device is not in this version |
| `ErrorCode` | One notice per code. `ErrorDetail` picks a variant: `access_denied`; `invalid_grant` or `not_found`; a network detail (`transport`, `request`, `http_N`) or an issuer configuration detail (`issuer_mismatch`, `s256_required`, …); browser `open` or the loopback. `SIGN_IN_CANCELLED` shows nothing. A `WAITLIST` code uses the same notice |
| `SignOut.RevokeAttempted` and not `Revoked` | Signed out on this device, but AstraFoundry did not confirm the revocation |

`runtime.AccountCodes` lists the codes views know. `TestAccountCodesMatchService` parses `internal/account/errors.go`, so a new service code fails until it is listed and has copy (`TestEveryErrorCodeHasCopy`).

The profile picture is downloaded once per `Picture` URL (`https` only, `image/*`, at most 2 MiB, 15 s) into `Account.PictureData`; a failure leaves the initials.

Not built yet:

- **Device enrollment (SYNC-02) after the entitlement.** No desktop sync client exists. The entitled row in `features/signin` is where enrollment starts.
- **Turn off sync and Delete server data (SYNC-07).** No actions back them, so Settings shows neither. They stay two separate operations.

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
func (s *Service) Stored() (bool, error)
```

`Stored` calls only `RefreshTokenStore.Get`, under the same lock that commits a token. `ErrNotFound` means false. Any other failure is a `SECURE_STORE` error.

`Dependencies.Store` is a `RefreshTokenStore` (`Put`, `Get`, `Delete`). `Delete` removes the refresh token only. `Dependencies.Browser` opens the authorization URL. `New` returns an error when either is nil. `Dependencies.CallbackPage` is optional.

`Session.AccessToken` is for the composition root's bearer calls. `TokenType` is `Bearer` when the token response omits it. `HasHostedSync()` reports the entitlement. The refresh token is never on `Session`.

`SignOut`'s `armed` callback runs after the in-flight attempt is marked as sign-out and before `SignOut` waits for it. `bootstrap` cancels the sign-in context there. Pass nil from tests. The returned `SignOutResult.RevokeError` is also the error from `SignOut` when revocation fails; the local item is still deleted.

`Config` zero value (direct service construction):

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

The access token is decoded for `sub`, `ent`, and the ID-token profile only. After sign-in and refresh, the service reads `userinfo_endpoint` from discovery with the access token and fills name, email, picture and locale from it, because Better Auth 1.7 leaves profile claims out of the ID token. A UserInfo failure or a mismatched `sub` keeps the ID-token claims and does not fail sign-in. Email needs the `email` scope, which the `arion-desktop` client is not granted, so no email shows. The signature is not checked on device. A missing `ent` means no hosted entitlement. A non-array `ent` is `ENTITLEMENT` / `ent_type`. A malformed ID token does not fail sign-in.

Refresh replaces the stored refresh token when the response includes a new one, and keeps the current one when it does not. Rotation is persisted before the optional UserInfo request; missing profile claims retain the same subject’s in-memory display fields. `invalid_grant` deletes the stored token and clears the session.

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

`*Error` implements `Is` by code. `CodeOf` and `DetailOf` are the UI mapping. Codes: `SIGN_IN_IN_PROGRESS`, `SIGN_IN_TIMEOUT`, `SIGN_IN_CANCELLED`, `STATE_MISMATCH`, `AUTHORIZATION`, `TOKEN`, `DISCOVERY`, `SECURE_STORE`, `NOT_SIGNED_IN`, `REFRESH`, `REVOKE`, `WAITLIST`, `ENTITLEMENT`, `BROWSER`, `USAGE_CONSENT`.

Sentinels: `ErrSignInInProgress`, `ErrSignInTimeout`, `ErrSignInCancelled`, `ErrStateMismatch`, `ErrNotSignedIn`, `ErrNotFound`, `ErrUnsupported`, `ErrRefreshTokenTooLarge`.

## Credential store

`infrastructure/keystore` implements `RefreshTokenStore`.

| OS | Item | Mechanism |
| --- | --- | --- |
| macOS | login Keychain, service `Arion`, account `account-refresh-token/<profile fingerprint>` | `/usr/bin/security` add/find/delete-generic-password |
| Windows | Credential Manager target `Arion/account-refresh-token/<profile fingerprint>`, generic, local-machine persistence | `CredWriteW` / `CredReadW` / `CredDeleteW` |
| other | — | `ErrUnsupported` when a method is called. `Open` still succeeds, so dictation starts |

The vocabulary encryption key is a different item: account `vocabulary-key`, written by `infrastructure/secrets` with SecItem on macOS and DPAPI on Windows ([ADR 005](../decisions/005-local-encryption.md)). Sign-out does not address that account. macOS passes the refresh token as an argument of `security` for the duration of the call, visible to other processes of the same user; at rest it is a Keychain generic password. This package uses the `security` tool, the same approach as `github.com/zalando/go-keyring`, so it does not add that module, does not link the SecItem helper that owns the vocabulary key, and `GOOS=darwin go vet` does not need the macOS SDK. Windows blobs are at most 2560 bytes (`ErrRefreshTokenTooLarge`).

`infrastructure/browser` opens `http` and `https` URLs only: `/usr/bin/open` on macOS, `rundll32 url.dll,FileProtocolHandler` on Windows, `xdg-open` elsewhere.

Native Keychain and Credential Manager behavior is verified by compiling the platform files. It has not been executed on those operating systems.

## Packaged center profiles

The composition root supplies the immutable [compiled center profile](central-sync-design.md#implemented-profile-boundaries), including custom issuer, client, resource, scopes and entitlement. Refresh-token storage is keyed by its fingerprint. Legacy unscoped tokens are retained without migration; sign in again. Device mode disables all account operations and removes the account setup path. Custom OIDC disables the AstraFoundry catalog/waitlist flow. Neither profile selection nor credential-store construction contacts a server or enables sync.

OS-authorized [product notifications](product-notifications.md) use the current session without requiring hosted-sync entitlement or an application preference. Sign-out invalidates the network/action generation before revocation; restoring a session resumes authenticated catchup when OS permission allows. This does not enable encrypted memory sync.

## Consented usage reporting

Signing in identifies the account; sharing usage data is the account's own
answer, held by the platform, so the web portal, other devices and this app see
the same choice. Arion never treats signing in alone as permission.

Onboarding sign-in passes its explicit default-off usage checkbox choice. A checked choice records consent after successful authentication; an unchecked choice reads the existing account answer without changing it. Settings › Account retains the switch for later changes. The disclosure at `/usage-data/` lists the exact fields, account association, exclusions and stopping behavior.

After successful sign-in or refresh, `Service.ReadUsageConsent`
reads the answer with `GET {issuer}/api/v1/usage/consent?productSlug=arion` and
the access token (`UsageLoading` meanwhile). It is `undecided`, `shared` or
`not_shared` (`Account.UsageSharing`) and replaces the token's claim for
reporting. Reporting waits for this answer: a valid claim alone never starts
it, and a failed read leaves it off, so reporting runs only while the switch
shows sharing. Settings › Account shows a **Share data to join the product improvement program** switch while signed
in.

The switch records the answer with `Service.RecordUsageConsent`:
`PUT {issuer}/api/v1/usage/consent` and the access token, body
`{"productSlug":"arion","accepted":bool}`. The platform stores the notice
version and a timestamp and keeps a history of acceptance and withdrawal. An
acceptance must name `usage-v1`, and a recorded answer must echo the choice and
be decided; otherwise the session is left unchanged and the view shows
`USAGE_CONSENT`. A confirmed choice updates the session at once, clears an
earlier authorization refusal of the same token, and reporting starts or stops
with it. Another account signing in starts unknown.

The access token carries `usage_consent_version=usage-v1` and a valid
`usage_consent_at` timestamp only after the platform records explicit agreement.
Missing/older claims leave reporting disabled until the platform's read says
otherwise. A choice made later on the web reaches an existing desktop session
at its next token refresh, which reads the choice again; the server checks
current consent on every report.

The same committed usage event that updates Overview also updates a separate
account-scoped ledger. Only events committed while signed in and consented are
counted; earlier Overview history and signed-out dictation are not backfilled.
Visits record active signed-in calendar days, including days without dictation.
The payload contains date, visited, sessions, microphone milliseconds and word
count, plus the fixed product slug and a random account/installation source ID.
It contains no recording, text, scene, application identity, host or session time.

At sign-in and every five minutes, a background worker sends changed daily
snapshots to the issuer's `/api/v1/usage`. The persisted random source ID survives
restarts; the server takes monotonic snapshots per source and sums distinct
sources, so retries and multiple devices do not overwrite each other. Pending
snapshots are bounded to 31 local calendar days. Local ledger paths are keyed by
issuer and subject, and contain no credentials. Account switches and sign-out
cancel requests and invalidate queued acknowledgements. Only the exact sent
snapshot is acknowledged, preserving commits made while a request is in flight.
A successful response must name Arion, match the sent calendar day, retain its
visit flag and confirm each counter at or above the sent value. Missing/wrong
products or lower counters are payload errors; they leave the snapshot pending
and use the normal retry backoff instead of acknowledging unconfirmed usage.
Transient failures retry after 15 seconds with exponential delay capped at five
minutes. Authorization refusals stop collection and sending until a fresh token
allows them; token expiry waits for the existing account maintenance flow.
Offline and scripted E2E profiles do not start reporting.

The private evaluation repository checks this flow against a local
astrafoundry.si (its `usage:local-e2e` script): sign-in on a new account, the
switch, and a second install of the same account, asserting throughout that
reporting never runs on an answer the switch does not show. See its README.

The account portal's Data link opens `/account/data` (standard issuer only),
where users can also withdraw or re-enable sharing.
The platform caps sources per account day and accepts dates within 32 days back
and one day ahead of its UTC day; the 31-day ledger stays inside that window.

Display caches are also keyed by issuer and subject. Legacy unscoped display
caches are cleared rather than reused for a potentially different account.
