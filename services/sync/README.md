# Memory relay

Follow [003 Memory sync](../../specs/003-memory-sync.md), [005 Privacy](../../specs/005-privacy.md), and Go DDD: domain → application ← infrastructure, thin interfaces.

The relay stores authenticated ciphertext and routing revisions. It never processes speech or reads plaintext memory. Add its Go module, migrations, concrete wire fixtures and service runbook together when implementing task SYNC-01.

One core serves self-hosted and hosted deployments ([ADR 015](../../docs/decisions/015-hosted-relay-account-authorization.md)). The core (envelopes, revisions, cursors, devices, revocation) does not know about accounts. The application port `AccountGate` resolves each request to a tenant first: `noop` (self-host default, device authentication only) or `oidc` (hosted: Bearer JWT with the `arion.sync.hosted` entitlement, `(issuer, subject)` → tenant, device authentication still required). With `oidc`, `POST /v1/security-events` receives account purge, disable, enable and session-revoked events. The hosted deployment uses its own PostgreSQL, never the platform database.
