# ADR 015: Hosted relay account authorization layer

## Status

Accepted. Amends SYNC-02 of [003](../../specs/003-memory-sync.md), which assumed no account login, and adds SYNC-08–10.

## Context

[003](../../specs/003-memory-sync.md) defines a self-hosted relay that stores only ciphertext and authenticates devices. Astra Foundry will also run a hosted relay at `api.arion.astrafoundry.si`. A hosted relay needs to know who may use it, which tenant a request belongs to and what quota applies, and it must act when the account behind a tenant is deleted, banned or revokes Arion. Access to the hosted relay is an entitlement (`arion.sync.hosted`) granted from the Astra Foundry waitlist; Astra Foundry accounts are an OpenID Connect provider at issuer `https://astrafoundry.si`. Endpoints, claims, clients and event payloads are defined normatively by the [platform contracts](https://github.com/AstraFoundry/astrafoundry.si/blob/main/docs/project/platform-contracts.md) (`astrafoundry.si` repository, `docs/project/platform-contracts.md`); this ADR records Arion's side.

Self-hosting stays the sync priority, so none of this may become a requirement for a self-hosted relay, and an account must never become a way to read memory.

## Decision

**One core, two gates.** The relay core (envelopes, revisions, cursors, devices, revocation) does not know about accounts. An application port, `AccountGate`, resolves each request to a sync tenant before the core runs. It has two implementations, chosen by configuration:

- **`noop`** — the self-host default. The deployment has one implicit tenant; only device authentication applies.
- **`oidc`** — the hosted edition. The request carries `Authorization: Bearer <access token>`. The gate verifies a `typ: at+jwt` token's signature against the issuer's JWKS (from discovery), taking `alg` from the key (EdDSA/Ed25519 by default; `none` and HMAC are rejected), then `iss`, `aud` (the relay's resource identifier, `https://api.arion.astrafoundry.si`), `exp`, and that the `ent` claim contains `arion.sync.hosted`. It maps `(issuer, subject)` to a tenant, creating the link on the first authorized request. Issuer, audience and entitlement key are configuration, so a self-hoster may point `oidc` at any IdP.

Device signature authentication is still required inside the tenant. A valid token alone never reads or writes envelopes, and a device credential is valid only in the tenant it enrolled in. A missing entitlement, an expired or invalid token and a frozen tenant are explicit outcomes (SYNC-06), never a silent fallback to `noop`.

**Linking data.** Accounts are keyed on `(issuer, subject)`, never on email:

```text
account_link(issuer, subject, tenant_id, created_at)   unique (issuer, subject)
tenant(id, quota, state, created_at)                   state: active | frozen
```

Groups, devices and envelopes belong to a tenant as defined by 003.

**Security events.** With `oidc`, the relay receives Security Event Tokens (RFC 8417) pushed per RFC 8935 at `POST /v1/security-events`. It verifies the `secevent+jwt` signature with the same JWKS, `iss`, `aud` (the relay's server client, `arion-sync`), dedupes on `jti`, and answers `202` with an empty body. It rejects with `400` and `{ "err", "description" }`, where `err` is `invalid_request`, `invalid_key`, `invalid_issuer` or `invalid_audience` (a bad signature is `invalid_key`). Handlers are idempotent because delivery is at least once.

| Event | Action on the tenant linked to `sub_id` |
| --- | --- |
| `risc/account-purged` | Delete its ciphertext, routing metadata, devices and account link |
| `risc/account-disabled` | Freeze (reject push, pull and enrollment; keep data) and revoke sessions as for `session-revoked` |
| `risc/account-enabled` | Unfreeze |
| `caep/session-revoked` | Reject that subject's access tokens issued before the event until the client re-authorizes |
| `entitlements-changed` (`{ "entitlements": […] }`) | Accept; `ent` is checked on every request, so no state changes |

With `noop` the endpoint does not exist.

**Login is not decryption.** The account answers only whether a client may use the hosted relay, which tenant it is and what quota applies. Reading memory depends only on a device holding the group key. Neither the relay nor Astra Foundry sees plaintext or holds keys, and account recovery does not recover data: losing every device without the recovery key loses the synced memory. Onboarding, Settings and the privacy notice say so.

**Desktop sign-in** follows RFC 8252. Arion opens the system browser to the issuer's authorize endpoint (from discovery) as the public client `arion-desktop`, with PKCE S256 and a `state`, and receives the code on a one-shot listener at `http://127.0.0.1:{random port}/callback`. It requests `openid profile offline_access arion.sync waitlist:join`. The refresh token is kept in the macOS Keychain or Windows Credential Manager; access tokens stay in memory.

- Onboarding offers two paths, the first being the default: **Continue locally**, with no account and every local feature, and **Sign in and set up sync**. A signed-in user without `arion.sync.hosted` can join the hosted-sync waitlist from the app and is offered a self-hosted relay. The app resolves the waitlist id from `GET {issuer}/api/v1/catalog/products/arion` (key `hosted-sync`) rather than hardcoding it; joining returns `201` pending, and `409` means already entitled or already pending; local features are unaffected. With the entitlement, device enrollment follows SYNC-02.
- Signing out revokes and deletes the refresh token and keeps local keys and memory. Turning sync off and deleting server data remain separate actions (SYNC-07).

**Deployment.** The hosted relay is its own Docker compose stack with its own PostgreSQL, kept in this repository's `deploy/`. It never shares the Astra Foundry platform database; it talks to the platform only through OIDC discovery, JWKS, the security events it receives and, optionally, `POST {issuer}/api/v1/entitlements/introspect` as the confidential client `arion-sync` (`client_credentials`, `client_secret_basic`, scope `entitlements:introspect`).

## Consequences

- Self-hosted relays run exactly as 003 describes, with no IdP, account table or events endpoint.
- The hosted relay can meter and freeze tenants, and account deletion reaches it without a shared secret: SETs are signed with the keys it already trusts for access tokens.
- Two independent credentials guard the hosted relay: a stolen access token cannot read or write without an enrolled device, and a stolen device credential stops working when its tenant is frozen or purged.
- A hosted request carries both a Bearer token and a device signature, and the desktop must refresh tokens before sync calls.
- If the event endpoint is unreachable, the platform retries for 72 h. Until delivery, a purged account's ciphertext remains, but new tokens can no longer be issued for that subject.
- An issuer change would orphan every link; the issuer is fixed by the platform and changes only with a dual-issuer migration.

## Alternatives considered

- **Account login instead of device authentication.** One credential would then grant ciphertext access, and self-hosting would need an IdP. Rejected.
- **Per-tenant API keys.** No link to the account, so no entitlement check and no deletion signal.
- **Linking by email.** Emails change and can be reassigned; `(issuer, subject)` is stable.
- **Key escrow or account-based recovery.** It would let the account, the platform or the relay decrypt memory, which contradicts 003 and 005.
- **Sharing the platform database.** Couples deployments and widens the blast radius; the relay needs only the link table.
- **Polling entitlement introspection instead of receiving events.** Slower deletion and extra load. `ent` in the token already covers entitlement changes; introspection stays an optional on-demand check.

## References

- [003 Memory sync](../../specs/003-memory-sync.md), [005 Privacy](../../specs/005-privacy.md), [ADR 002](002-local-pipeline-and-memory.md).
- [`services/sync`](../../services/sync/README.md), [`deploy/`](../../deploy/README.md).
- [Astra Foundry platform contracts](https://github.com/AstraFoundry/astrafoundry.si/blob/main/docs/project/platform-contracts.md) — normative issuer, client, token, waitlist and security-event values.
- RFC 8252 (OAuth 2.0 for native apps), RFC 7636 (PKCE), RFC 8417 (Security Event Token), RFC 8935 (push delivery), OpenID RISC and CAEP event types.
