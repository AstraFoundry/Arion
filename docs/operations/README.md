# Operations

- [Local rewriting](local-rewrite.md) — selected GGUF model, two-stage prompts, private runtime and delivery policy.
- [Local development](local-dev.md) — actual repository commands and module toolchains.
- [Evaluation commands](../../tools/evaluation/README.md) — Make/pnpm benchmarks and real drivers.
- [Deployment](deployment.md) — desktop/model/IME and self-hosted relay release gates, and the marketing site.
- [Release status](release-status.md) — shipped platform capabilities, preview signing status and host acceptance limits.

## Build components

Use `make build <component>` with exactly one component per invocation:

| Command | Output |
| --- | --- |
| `make build desktop` | Native app and installer for the current host; configured ENV enables macOS signing/notarization and signed updates |
| `make build site` | Static marketing site in `apps/site/out` |

Both dispatch through `scripts/build.mjs`, also used by desktop CI. Missing, unsupported or extra components fail before building. Mobile and server builds are not registered yet. `make site-deploy` uses the same site build before deployment. The former `desktop-build` and `site-build` targets have been removed.

The foundation uses Node 24 and pinned pnpm 10.33.0 for workspace/check/evaluation tooling, Go 1.25.7+ for isolated portable core checks, Go 1.27.1+ for the sync relay and MyGo desktop. Go automatic toolchain selection may download the pinned newer toolchain. Native shell compilation is distinct from functional speech/IME support.

The desktop shell does not need a database. The marketing site builds from committed public configuration (`apps/site/.env.production`); only `make site-deploy` needs Cloudflare credentials, supplied by the shell running it. The sync relay has its own PostgreSQL, configuration and runbook under [deploy/sync](../../deploy/sync/README.md). Do not point it at another product's database.

CI installs with a frozen pnpm lockfile. `make updater-check` / `pnpm updater:check` verifies the maintained MyGo update-channel patch and is a portable CI gate. The portable job also runs `pnpm site:check` (site dictionaries, types, lint, tests and static export). Native jobs run actual window E2E and verify packages on macOS arm64 and Windows x64. Tagged releases require Apple signing/notarization and update credentials, reuse these jobs at the tag commit and upload only a complete verified set for their selected platforms (alpha: macOS ARM64; other tags: both supported platforms); ordinary CI retains installers without signing keys. The manual [benchmark workflow](../../.github/workflows/benchmark.yml) requires an explicitly provisioned `self-hosted, arion-benchmark` runner, reviewed local driver and licensed private corpus; no hosted-runner hardware score or automatic weights download is implied. Restrict access to the self-hosted runner and retain raw traces privately.
