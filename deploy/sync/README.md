# Sync relay

Compose stack for the ciphertext relay. It is a separate PostgreSQL 17 from the desktop and from the AstraFoundry platform database. Bind the API to loopback and terminate TLS in front of it for `api.arion.astrafoundry.si`.

```sh
cd deploy/sync
cp .env.example .env
# set ARION_SYNC_POSTGRES_PASSWORD
docker compose up --build
```

`migrate` applies `services/sync/infrastructure/postgres/migrations` once, then `api` serves `:8080`. `GET /healthz` is process liveness. `GET /readyz` checks the database.

`ARION_SYNC_ACCOUNT_GATE=noop` is the self-host default: device signatures only, no account tables are read, and `POST /v1/security-events` is not registered. `oidc` verifies `at+jwt` access tokens from the configured issuer and accepts security events. See [services/sync](../../services/sync/README.md).

Secrets may be supplied as `ARION_SYNC_DATABASE_URL_FILE` instead of `ARION_SYNC_DATABASE_URL`. Setting both is a startup error. The image runs as uid 65532 and does not include a shell.

Revocation bumps the key epoch. Clients re-encrypt live records; this process stores ciphertext and cannot erase plaintext a device already decrypted. A non-empty `recovery` field on enrollment is rejected.

The in-process rate limit is per device id and is not shared across replicas.
