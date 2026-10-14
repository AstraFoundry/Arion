# Operations

- [Local development](local-dev.md) — actual repository commands and module toolchains.
- [Evaluation commands](../../tools/evaluation/README.md) — Make/pnpm benchmarks and real drivers.
- [Deployment](deployment.md) — desktop/model/IME and self-hosted relay release gates, and the marketing site.

The foundation uses Node 24 and pinned pnpm 10.33.0 for workspace/check/evaluation tooling, Go 1.25.7+ for isolated portable core checks, and Go 1.27.1+ for MyGo desktop. Go automatic toolchain selection may download the pinned newer toolchain. Native shell compilation is distinct from functional speech/IME support.

No database/server/cache or environment credentials are needed for the current shell. The marketing site builds from committed public configuration (`apps/site/.env.production`); only `make site-deploy` needs Cloudflare credentials, supplied by the shell running it. Do not start placeholder cloud services. Future self-hosted sync needs its own concrete module, config and runbook before deployment.

CI installs with a frozen pnpm lockfile. The portable job also runs `pnpm site:check` (site dictionaries, types, lint, tests and static export). Native jobs run actual window E2E. The manual [benchmark workflow](../../.github/workflows/benchmark.yml) requires an explicitly provisioned `self-hosted, arion-benchmark` runner, reviewed local driver and licensed private corpus; no hosted-runner hardware score or automatic weights download is implied. Restrict access to the self-hosted runner and retain raw traces privately.
