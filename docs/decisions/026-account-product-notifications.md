# 026: Account-targeted product notifications

## Status

Accepted for running-app reception and next-launch catchup.

## Decision

The standard AstraFoundry package may poll an authenticated, product-scoped
account inbox when the user explicitly enables Product notifications in General
settings and OS authorization is available. New and migrated profiles keep the
choice off. Custom/device-only centers and offline/E2E profiles make no product
notification requests. Local dictation remains independent of sign-in.

The inbox and selected-recipient admin workflow belong to AstraFoundry. Arion
uses the existing account bearer token without acquiring a new scope or hosted
sync entitlement. A poll interval reads the newest page and at most one catchup
page, at most twice per minute, while input/resource/maintenance work is idle.
No shortcut/input event initiates a request. Exiting Arion stops reception;
there is no APNs, Windows push service or claim of immediate delivery while quit.

Delivery uses the existing native MyGo adapter. At most two labeled buttons may
dismiss or open an exact-allowlisted HTTPS destination. Persistent account-hashed
records resolve native ID/button-index activations. A successful current-server
receipt must validate recipient, product, publication, expiry, configuration and
button before browser launch. Account/generation checks run again at launch;
network failure, revocation and sign-out cannot open a previous account's URL.

A durable reservation precedes native submission. Successful OS submission
produces `received`; explicit body interaction produces `read`, a URL button
produces `action`, and dismissal produces `dismissed`. Submission never proves
banner visibility or reading. An interrupted reservation remains uncertain and
is not automatically repeated. The bounded private ledger holds action metadata
and account hashes, never message bodies or credentials. Received server receipts
prevent repeat submissions across devices and after acknowledged local pruning.

## Consequences

Administrators can select recipients, author title/body/buttons and publish via
the central product workflow. Desktop consent and current OS settings remain
independent. Published payloads remain immutable; withdrawing a destination host
suppresses affected inbox notices and rejects stale actions. Already displayed
OS banners are removed only on known local cleanup; absence from a paginated
response cannot prove withdrawal. Current-server validation protects old buttons.

See [implementation and API contract](../project/product-notifications.md) and
DSK-18 / DSK-A20–22. Native banner/action acceptance remains separate from HTTP,
policy, persistence and headless view tests.
