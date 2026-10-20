# Arion center

The center is the public origin `api.arion.astrafoundry.si`: the standard desktop profile's `ARION_CENTER_URL` ([center profile](../../docs/project/central-sync-design.md#distribution-and-build-time-center-profile)). One Compose stack runs:

| Service | Routes | Source |
| --- | --- | --- |
| `caddy` | TLS for `ARION_CENTER_DOMAIN`, path routing | [`Caddyfile`](Caddyfile) |
| `updates` | `/v1/updates/*`, `/v1/releases/*`, `/v1/downloads/*` | [`services/updates`](../../services/updates/README.md) |
| `sync` + `migrate` + `postgres` | `/v1/sync/*`, `/v1/security-events` | [`services/sync`](../../services/sync/README.md) (`oidc` gate by default here) |

```sh
cd deploy/center
cp .env.example .env            # set the domain, database password and bucket
mkdir -p secrets && printf '%s' "$READ_ONLY_SECRET" > secrets/updates-s3-secret
docker compose up -d --build
curl -fsS "https://$ARION_CENTER_DOMAIN/v1/releases/beta"
```

Give `updates` a bucket token that can only read the product prefix. The platform admin holds the write credentials and publishes channels ([astrafoundry.si release admin](https://github.com/AstraFoundry/astrafoundry.si/blob/main/docs/backend/releases.md)); CI holds credentials that can write `files/` and `releases/` only. The sync database is the relay's own, never the platform's. `secrets/` is ignored by Git.

The stack has been validated with `docker compose config`; it has not been brought up against a live bucket in this repository's CI.
