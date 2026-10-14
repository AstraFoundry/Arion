# Memory relay

Go module `arion.local/sync`. Follow [003 Memory sync](../../specs/003-memory-sync.md), [005 Privacy](../../specs/005-privacy.md), and Go DDD: domain → application ← infrastructure, thin interfaces. `cmd/sync` is the composition root (`serve` or `migrate`).

The relay stores authenticated ciphertext and routing revisions. It never processes speech or reads plaintext memory. Clients encrypt. Login is not decryption.

One core serves self-hosted and hosted deployments ([ADR 015](../../docs/decisions/015-hosted-relay-account-authorization.md)). The core (envelopes, revisions, cursors, devices, revocation) does not know about accounts. The application port `AccountGate` resolves each request to a tenant first:

- `noop` (self-host default, `ARION_SYNC_ACCOUNT_GATE=noop`): one implicit scope `local`, device authentication only. No account table is consulted and `POST /v1/security-events` is not registered.
- `oidc`: Bearer `at+jwt` for `aud` `https://api.arion.astrafoundry.si`, `ent` containing `arion.sync.hosted`, signature from the issuer JWKS (`alg` allow-list, default `EdDSA`, `RS256`, `ES256`; `none` and HMAC rejected). `(issuer, subject)` maps to `account_link` / `tenant`. Device authentication is still required. A missing entitlement, invalid token, or frozen tenant is an explicit rejection and never falls through to `noop`.

Issuer, client, token and event values are normative in the [platform contracts](https://github.com/AstraFoundry/astrafoundry.si/blob/main/docs/project/platform-contracts.md). The hosted deployment uses its own PostgreSQL ([deploy/sync](../../deploy/sync/README.md)), never the platform database.

## Device authentication

Ed25519. Headers `X-Arion-Device-Id`, `X-Arion-Timestamp` (unix milliseconds), `X-Arion-Signature` (raw base64url). The signed bytes are:

```text
arion-sync-request-v1
<device id>
<unix milliseconds>
<METHOD>
<path and raw query>
<hex sha256 of the raw body>
```

The first device creates the group at key epoch 1. A later device needs an active sponsor signature over `arion-sync-enroll-v1\n<group>\n<device>\n<public key>`. A non-empty `recovery` field returns `RECOVERY_UNSUPPORTED`. Revocation is idempotent and rotates the key epoch once. Clients re-encrypt live records; the relay cannot.

## HTTP

`POST /v1/sync/devices`, `POST /v1/sync/devices/{id}/revoke` with body `{}`, `POST /v1/sync/push`, `GET /v1/sync/pull?cursor=`, `GET /v1/sync/snapshot`, `POST /v1/sync/ack`, `GET /healthz`, `GET /readyz`.

Push bodies are JSON. Ciphertext, nonce, tag and AAD are raw base64url. Schema version is 1. A cursor is an opaque encoding of a server revision. Revision 0 means the start of the log. An unknown or expired cursor is `410` with `resync` `/v1/sync/snapshot`. The snapshot does not delete client memory.

Sibling revisions are kept. A delete beside an edit stays suppressed until a resolution names every current head. Tombstones stay until every active device has acknowledged them, then that record's envelopes are removed. Wall clock does not pick a winner.

With `oidc`, `POST /v1/security-events` accepts `application/secevent+jwt` (RFC 8417 over RFC 8935), verified with the same issuer JWKS, `typ` `secevent+jwt`, `aud` `arion-sync`, deduplicated on persisted `jti`. Success is `202` with an empty body. Failure is `400` `{"err","description"}` using only `invalid_request`, `invalid_key`, `invalid_issuer`, or `invalid_audience`. A bad signature or a key the JWKS cannot supply is `invalid_key`. A malformed token, wrong `typ`, or rejected claim is `invalid_request`.

| Event | Effect |
| --- | --- |
| `https://schemas.openid.net/secevent/risc/event-type/account-purged` | Delete the tenant's ciphertext, routing metadata, devices and account link. A later token cannot recreate it. |
| `https://schemas.openid.net/secevent/risc/event-type/account-disabled` | Freeze the tenant and revoke sessions. Data is kept. |
| `https://schemas.openid.net/secevent/risc/event-type/account-enabled` | Unfreeze. Tokens issued at or before the revocation floor stay rejected. |
| `https://schemas.openid.net/secevent/caep/event-type/session-revoked` | Reject tokens issued at or before the event until a newer token is presented. |
| `https://astrafoundry.si/secevent/entitlements-changed` | Accept `{ "entitlements": ["…"] }` and log. The payload is not stored. `ent` is read from the next access token. |

## Configuration

`ARION_SYNC_*` environment variables. A `*_FILE` sibling supplies `ARION_SYNC_DATABASE_URL` and `ARION_SYNC_OIDC_ISSUER`; setting both the variable and the file is a startup error. Invalid gate, duration, size, log level or issuer URL fails the process. Logs are JSON and drop credential and ciphertext attribute names. The default quota is 1 GiB (`0` means unlimited). The rate limit is in-process, per device id.

```sh
make sync-check
cd services/sync && GOWORK=off go test ./... && GOWORK=off go vet ./...
```

`make sync-integration` runs the PostgreSQL test behind the `integration` build tag and requires `ARION_SYNC_TEST_DATABASE_URL`. The default test run uses the in-memory port double, which is not wired into `cmd/sync`. The wire fixture [push-v1.json](../../packages/contracts/sync/push-v1.json) is posted by the HTTP tests.
