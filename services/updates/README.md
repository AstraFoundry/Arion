# Update channels

Go module `arion.local/updates`. It serves Arion's update checks and downloads on the center origin (`https://api.arion.astrafoundry.si`), so installed apps and the landing site never call the GitHub API. It reads only the objects the AstraFoundry platform publishes in the release bucket ([bucket contract](../../packages/contracts/releases/README.md)); it has no database and decides nothing about which version a channel offers. Layers follow Go DDD: `domain` (channels, targets, file names, channel index) → `application` (`Bucket` port, short cache) ← `infrastructure` (`s3` SigV4 reader, `config`) and thin `interfaces/http`. `cmd/updates` is the composition root.

## HTTP

| Route | Answer |
| --- | --- |
| `GET /v1/updates/update-<target>.json?channel=stable\|beta` | The channel's MyGo manifest, verbatim, `Cache-Control: no-cache`. No `channel` means `stable`. `404` `NO_RELEASE` when the channel has no release for the target |
| `GET /v1/updates/<file>` | `302` to a presigned URL of `files/<file>` (installers, archives, deltas) |
| `GET /v1/releases/<channel>` | Public channel summary: per target version, prerelease flag, date, notes and installer name/size/SHA-256/URL. CORS `*`, cached 60 s. An unpublished channel has no targets |
| `GET /v1/downloads/<channel>/<target>` | `302` to the channel's current installer for the target |
| `GET /healthz`, `GET /readyz` | `ok` |

Invalid channels, targets and file names are `400`. A storage failure or an object that breaks the contract is `502` `STORAGE`, never an empty channel. The desktop selects the channel with the `url-channel-v1` MyGo patch ([third_party](../../third_party/README.md)): **Allow beta updates** sends `channel=beta`.

Signatures are not this service's concern: the app verifies every archive and delta against the Ed25519 key built into it ([ADR 010](../../docs/decisions/010-in-app-updates.md)). A compromised bucket or center can withhold updates but cannot install unsigned code.

## Configuration

| Variable | Meaning |
| --- | --- |
| `ARION_UPDATES_PUBLIC_URL` | Required. Public origin, used for installer URLs in summaries (https; http only on loopback) |
| `ARION_UPDATES_S3_ENDPOINT`, `_REGION`, `_BUCKET`, `_ACCESS_KEY_ID`, `_SECRET_ACCESS_KEY` | Required. Read-only credentials; the secret also as `_SECRET_ACCESS_KEY_FILE` |
| `ARION_UPDATES_S3_PREFIX` | Key prefix of the product (standard `arion`) |
| `ARION_UPDATES_S3_PATH_STYLE` | `true` (default, R2/MinIO) or `false` |
| `ARION_UPDATES_S3_SESSION_TOKEN` | Temporary credentials (`_FILE` supported) |
| `ARION_UPDATES_PRESIGN_TTL` | Download URL lifetime, default `10m`, at most `168h` |
| `ARION_UPDATES_CACHE_TTL` | How long channel objects are cached, default `30s`: a publication reaches clients within this time |
| `ARION_UPDATES_HTTP_ADDR`, `ARION_UPDATES_LOG_LEVEL` | Default `:8081`, `info` |

```sh
make updates-check   # gofmt, go test and go vet with GOWORK=off
```

Tests cover the AWS Signature Version 4 presign reference example, path-style keys with a prefix, cache expiry, empty versus failed channels, traversal rejection and the contract fixture. The service has not yet been run against a live R2 or S3 bucket. Deployment: [deploy/center](../../deploy/center/README.md).
