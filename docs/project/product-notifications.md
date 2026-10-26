# Product notifications

[ADR 026](../decisions/026-account-product-notifications.md) covers account notices
sent from the AstraFoundry product admin. This is independent of app-update
reminders. General settings has a separate opt-in switch, off for new/migrated
profiles, with explicit notification authorization. The standard center and an
authenticated account are required; hosted-sync entitlement is not required.

While Arion runs, its five-second scheduling tick evaluates a one-minute network
deadline. A network interval reads the newest page plus at most one saved catchup
page. This discovers new notices during a large history replay without unbounded
requests. Empty filtered pages can still have a continuation cursor. Input,
managed resource work and update/IME maintenance defer fetching/submission; no
input event queries the server. Disabling, signing out or switching accounts
cancels in-flight requests and removes known old native IDs. Token refresh also
invalidates work. Existing account maintenance renews tokens before expiry through
the shared refresh controller; product polling never rotates refresh tokens itself. Quit stops polling. Missed notices are fetched after the next
launch and account restoration, subject to expiry and current publication.

## Contract

Requests go to the immutable compiled standard issuer, `https://astrafoundry.si`,
using the in-memory account bearer token; redirects are refused. Other builds do
not silently substitute another service. The server validates active recipient,
product-linked client and product-resource audience. No new scope is requested.

`GET /api/v1/notifications?productSlug=arion&limit=20&cursor=...` returns the standard
`{success,data,error}` envelope, whose data is:

```json
{
  "enabled": true,
  "allowedActionHosts": ["astrafoundry.si"],
  "items": [{
    "id": "12345678-1234-1234-1234-123456789abc",
    "productSlug": "arion",
    "title": "Product announcement",
    "body": "Announcement text",
    "buttons": [{"id":"learn-more","label":"Learn more","action":"open_url","url":"https://astrafoundry.si/"}],
    "publishedAt": "2026-01-01T00:00:00Z",
    "expiresAt": null,
    "receipt": {"receivedAt":null,"readAt":null,"dismissedAt":null,"actionIds":[]}
  }],
  "nextCursor": null
}
```

IDs are UUIDs;
product must be `arion`; title/body/labels have limits 120/2000/40 characters.
At most two unique button IDs match `[A-Za-z0-9_-]{1,32}`. Button actions are
`open_url` with a safe URL, or `dismiss` without a URL. Cursors are base64url up to
256 characters. Client response bytes and collection sizes are bounded. Missing
configuration fields, malformed payloads and cross-product items fail explicitly.

URL destinations must be HTTPS on an exact lowercase DNS allowlist, without
credentials, private/IP/localhost-style hosts or a nonstandard port (443 is
permitted). Activation arguments never contain a URL or command. Native IDs are
account/message hashes; `button-0` and `button-1` resolve only persisted metadata
for the current `(issuer, subject, product)`. Body activation opens Account
settings after authorized read acknowledgement. URL buttons launch the browser
only after an authorized action acknowledgement and final generation check.

`POST /api/v1/notifications/{id}/receipt` accepts
`{productSlug:"arion",event:"received"|"read"|"dismissed"|"action",actionId?:string}`.
`actionId` is required for `action`. The success envelope data is the updated
receipt. Replays are idempotent. The client uses `received` only after successful
OS submission, not fetching; it never sends `read` automatically. A backend
failure prevents browser launch. `401`/`403` reject authentication/product access,
revoked/expired/nonrecipient IDs yield `404`, removed destination hosts yield
`409`, and invalid requests yield `422`. These responses are content-free UI
failures; response bodies/tokens are not diagnostic data.

Received/read/dismissed receipts are user-level across devices, so a submission
on another device suppresses another automatic native submission. Inbox history
still includes those entries. Admin withdrawal and host removal stop future
fetches/actions; already displayed OS banners cannot be inferred from paginated
absence and are not guaranteed remotely retractable.

## Local ownership

- `internal/productnotifications`: SDK-free validation, generation, receipt and
  per-account submission/action policy.
- `infrastructure/productnotifications`: bounded HTTPS client and schema-checked
  private atomic `product-notifications.json` ledger.
- `bootstrap/product_notifications.go`: account lifecycle, scheduling, input and
  maintenance checks, native MyGo transport and system browser composition.
- `ui/features/productnotifications`: localized opt-in control and permission,
  account, failure and uncertain-submission states; no adapter imports.

A reservation is saved before `Show`, ensuring cold buttons have durable lookup.
Successful submission is saved before its idempotent receipt. Receipt retries
never repeat native submission. A process interrupted around `Show` cannot prove
submission; the retained reservation reports uncertainty and is not automatically
repeated. Definite submission refusal permits a later authorized retry.

The ledger stores account hashes, message/native IDs, button labels/actions/URLs,
allowed hosts, expiry and submission/receipt state. It holds no title/body,
transcript, raw subject or token, uses private file modes and rejects corruption,
unknown schema, oversized files and symlinks. At 4096 records it can prune the
oldest acknowledged entry and remove its native ID; durable server receipts still
prevent replay. Unacknowledged/uncertain entries are not evicted to manufacture
success. Server transport data is not claimed to be end-to-end encrypted.

## Verification

Run `go -C apps/desktop test -race ./internal/productnotifications ./infrastructure/productnotifications`
and `go -C apps/desktop test ./ui/app -run TestProductNotification` for validation,
receipt ordering, account switch/sign-out cancellation, active-input deferral,
interval/cursor behavior, cold metadata, storage corruption/privacy and bilingual
view behavior. Core defaults/round trips are covered by
`TestProductNotificationsAreExplicitAndPersisted`.

`TestProductNotificationBackendIntegration` is opt-in through
`ARION_NOTIFICATION_E2E_CONFIG`, an isolated loopback fixture config containing
`origin`, `token`, `subject`, `notificationId`. Only that test rewrites its HTTPS
logical transport to the literal loopback origin. It uses real inbox/receipt
requests; it does not show an OS notification or claim a real OS submission.

Native MyGo permission/submission observations and banner/action/cold-click gaps
remain in [notification/update validation](../quality/update-validation.md).
No production send, deployment or real recipient notification is part of these
local tests.
