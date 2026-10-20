# Release bucket contract

One S3-compatible bucket holds Arion's signed release files. Three parties use it, each with its own credentials:

| Party | Writes | Reads |
| --- | --- | --- |
| Release CI (`scripts/release.mjs stage`) | `files/*`, `releases/<version>/*` | `releases/<version>/release.json` (refuses to overwrite) |
| AstraFoundry platform admin (`astrafoundry.si` `releases` module) | `channels/*`, imported `files/*` and `releases/<version>/*` | everything |
| Arion center (`services/updates`) | nothing | `files/*`, `channels/*` |

All keys are relative to the configured prefix (standard: `arion`).

| Key | Content | Mutability |
| --- | --- | --- |
| `files/<name>` | Installers, MyGo update archives and deltas; flat names that carry the version | Immutable |
| `releases/<version>/release.json` | Staged release record ([`staged-release-v1.json`](staged-release-v1.json), schema `astrafoundry.release.v1`); written last | Immutable |
| `releases/<version>/SHA256SUMS`, `RELEASE-STATUS.md` | Release documents | Immutable |
| `channels/<stable\|beta>/update-<target>.json` | The MyGo manifest the channel serves for a target, copied verbatim from a staged record | Replaced on publication |
| `channels/<stable\|beta>/index.json` | Channel index ([`channel-index-v1.json`](channel-index-v1.json), schema `astrafoundry.release-channel.v1`) | Replaced on publication |

Rules:

- A staged record is not offered to anyone. Only the platform publishes, by writing channel objects: manifests first, the index last.
- `stable` holds, per target, the newest published release whose version has no prerelease part. `beta` holds the newest published release of either kind. A prerelease version can never enter `stable`.
- Manifest URLs point at the center, `https://api.arion.astrafoundry.si/v1/updates/<name>`; the center redirects to a short-lived presigned URL of `files/<name>`. Rewriting a URL never touches a signature: MyGo signs the SHA-256 of each archive and delta, and the app verifies it with its built-in key.
- `sha256` is lowercase hex of the file bytes; `size` is in bytes. Targets are `<darwin|windows|linux>-<arm64|amd64|universal>`.
