# Repository and ownership

## Workspace

| Path | Owner and dependency boundary |
| --- | --- |
| `apps/desktop` | Separate Go module; MyGo lifecycle and FSD native UI; depends on core |
| `apps/desktop/ui` | `app → pages → widgets → features → entities → shared` Go packages; same-layer slices isolated; shared segments may import each other |
| `apps/desktop/bootstrap` | Composition root: wiring, windows, tray, hotkeys, main-thread state, E2E control mode |
| `apps/desktop/infrastructure` | OS input/capture/context and model/storage adapters; implements domain-owned ports (package list in the [desktop README](../../apps/desktop/README.md)) |
| `apps/desktop/internal/center` | Compiled public center profile, validation and fingerprint; no runtime ENV or credential access |
| `apps/desktop/internal/account` | OIDC public client for hosted sync: loopback PKCE, in-memory access token, refresh rotation, waitlist join ([account sign-in](account-signin.md)) |
| `apps/desktop/internal/productnotifications` | Account-scoped inbox validation, native submission dedupe and authorized action policy ([product notifications](product-notifications.md)) |
| `apps/desktop/infrastructure/productnotifications` | Bounded authenticated HTTPS client and private atomic notification ledger |
| `apps/desktop/internal/oauthloopback` | Native-app OAuth helpers shared by both sign-ins: PKCE S256, state, one-shot loopback callback listener |
| `apps/desktop/infrastructure/keystore` | Credential items in the macOS Keychain or Windows Credential Manager: the refresh token, the active cloud-provider pointer, and one cloud polishing credential per provider |
| `apps/desktop/infrastructure/cloudpolish` | Optional cloud polishing ([ADR 022](../decisions/022-cloud-polishing.md)): ChatGPT sign-in, API key check, model list and a `Rewriter` over Republic ([cloud polishing](cloud-polishing.md)) |
| `apps/desktop/infrastructure/browser` | System browser for http(s) authorization URLs |
| `apps/desktop/cmd` | Evaluation and E2E drivers, the `arion-llm` rewrite worker (no cgo), live rewrite benchmarks, model catalog tool, icon renderer |
| `apps/desktop/infrastructure/llm/rewrite` | Shared staged polishing: cleanup, context-backed bounded spelling choices and scene formatting; model adapters supply stage generation ([ADR 024](../decisions/024-shared-polishing-pipeline.md)) |
| `apps/desktop/infrastructure/llm` | Pinned GGUF adapter (`gguf`), selected staged templates (`prompt`, with `prompt.go`/`live.json`/`context.json` retained for historical replay), pronunciation lexicon (`spelling`), pipe worker protocol (`protocol`), worker engine (`engine`), process client (`client`, `internal/subprocess`), and the ID-only local keyboard ranking adapter (`ranking`, [ADR 031](../decisions/031-local-keyboard-neural-ranking.md)) |
| `apps/desktop/infrastructure/rewriting` | Rewrite capability service: bundled runtime location, model provisioning, worker lifecycle, `Rewriter` port |
| `packages/core/domain` | Session and candidate invariants; standard-library only |
| `packages/core/contracts` | Versioned transport-neutral DTOs; no SDK or platform handles |
| `packages/core/application` | Dictation coordinator and preferences; depends on domain-owned ports |
| `packages/contracts` | Shared wire examples/documentation, no duplicate business types |
| `apps/site` | Static Next.js marketing site (pnpm, TypeScript FSD, en/zh/ja); independent of the Go modules and the desktop shell ([ADR 016](../decisions/016-marketing-site.md)) |
| `services/updates` | Go DDD update-channel service: serves `stable`/`beta` manifests, release summaries and presigned downloads from the release bucket ([ADR 019](../decisions/019-center-update-channels.md)) |
| `services/sync` | Go DDD relay; account-unaware core, `AccountGate` port with `noop` and `oidc` adapters ([ADR 015](../decisions/015-hosted-relay-account-authorization.md)) |
| `deploy` | Relay images, self-host compose and the hosted relay's compose stack with its own PostgreSQL |
| `models` | Go module embedding the reviewed catalog: provenance, digests, licenses, bundles, evaluation summaries; no weights |
| `third_party/mygo` | Maintained MyGo v0.4.6 snapshot with prerelease/URL-channel and staged-update, Windows rollback, signing-metadata, native-notification and Windows native-transparency patches ([third_party](../../third_party/README.md)); original hashes and licence retained |
| `scripts` | Dependency-free repository checks and platform-neutral workflow entry points |
| `tools/driver` | Public JSONL product-driver transport, artifact verification and owned scenario execution; no private corpus or scoring dependency |
| `Arion-evaluation/tools/evaluation` | Private study orchestration, metrics, report replay and development inference adapters |
| `Arion-evaluation/benchmarks` | Private Git archive of corpora, raw observations, reproduction sources, provenance and reports; no weights, runtime environments or personal recordings |
| `tests/e2e` | Product action/assertion scenarios; requires live OS adapters |
| `apps/desktop/e2e` | Build-tagged real native window lifecycle test |
| `specs` | Normative requirements, acceptance scenarios and task traceability |
| `docs/research` | Evidence and technical proposals; cannot override accepted specs |

The five Go modules (`packages/core`, `models`, `apps/desktop`, `services/sync`, `services/updates`) are connected by `go.work`. `services/sync` and `services/updates` are checked with `GOWORK=off`, the same way as the portable core. Their domain packages are standard-library only and does not import the application or infrastructure layers. `cmd/arion-llm` and the `infrastructure/llm` packages build with `CGO_ENABLED=0` and must never import the speech runtime; `scripts/stage-desktop-resources.mjs` prepares packages in local builds and CI. It calls `scripts/stage-llm-runtime.mjs` for the worker, pinned llama.cpp executable/dependency closure and pronunciation notices in `resources/<platform>/llm`, and collects module licenses and model provenance in `resources/licenses`. Windows x64 stages the same pinned GGUF model with a CPU-dispatched runtime ([ADR 025](../decisions/025-windows-local-polishing.md)); Intel macOS is unsupported ([ADR 021](../decisions/021-apple-silicon-only-macos.md)). `apps/desktop/go.mod` pins MyGo, miniaudio and Republic (`github.com/AstraFoundry/republic`, Apache-2.0, a pseudo-version until it is tagged), and replaces core/models with the local modules; reconsider the local `replace` before publishing modules. `GOWORK=off` allows portable core and catalog checks without upgrading their toolchain. The desktop module builds with cgo (`CGO_ENABLED=1`) because audio capture and the native UI use cgo.

The [two-repository boundary](repository-split.md) keeps public checks and packaging independent of private access. Private paths in the table are logical paths in [Arion evaluation](https://github.com/AstraFoundry/Arion-evaluation), not directories required by this checkout. Public native and unit-test inputs live in provenance-bound product `testdata`.

## FSD in native Go

Slice directories become Go packages. Only exported package symbols are the public API; there is no `index.ts`. Internal implementation uses unexported names or Go `internal/` packages. The original TypeScript snippets remain explanatory examples; they are not a requirement to add TypeScript or React.

UI slices can consume core contract DTOs through a shared adapter boundary. UI must not implement ASR/correction/ranking rules or import an infrastructure adapter directly. The app composition root wires dependencies. `ui/entities` owns UI-facing domain projections without widget code or transport clients.

All application copy is centralized in the English/Simplified Chinese catalogs under `ui/shared/copy`; `runtime.Store.Copy()` selects immutable translations from persisted preferences. Entity projections and transient errors carry typed message keys ([i18n](../frontend/i18n.md)). Native theme/layout values stay in shared theme packages when needed. Apply UI craft skills using MyGo's existing native primitives, not CSS-only instructions.

## DDD in desktop and relay

Core domain defines invariants and abstractions; core application orchestrates them. Desktop infrastructure provides local implementations. The relay domain owns device authorization, opaque revisions, retention and conflict validation, never speech, plaintext memory, or accounts. Its HTTP package calls application services rather than SQL. `infrastructure/memory` implements the same ports as PostgreSQL for tests and is not composed into `cmd/sync`.

## Working rules

Do not create empty implementations that return success, mock a recognizer as production behavior, or silently use cloud services. Every adapter requires a capability statement, explicit unsupported result and acceptance evidence. OS target claims require tests on that OS. Test doubles live only in `_test.go` files; the E2E control mode replaces the microphone with fixture audio but runs the production recognizer, correction and insertion.

JavaScript tooling is a pnpm workspace pinned by the root package manager and lockfile. `make` exposes recurring workflows; do not introduce npm commands or a second lockfile. Go modules remain independently versioned. Benchmark/OS drivers are concrete adapter implementations, not frontend business logic.

Native ASR workers live in `apps/desktop/native/mlx-asr` (Swift) and the pinned speech.cpp build configured by `scripts/asr-runtime.json`. `stage-asr-runtime.mjs` prepares the platform executable, shader/Swift runtime closure and dependency notices in ignored `resources/<platform>/asr`. The domain port never imports either runtime. See [ADR 020](../decisions/020-native-qwen-asr.md).

The optional fast keyboard model runs in `apps/desktop/runtime/keyboard-ranker`, a small locked Rust worker using the upstream reference inference crate. `scripts/stage-keyboard-ranker.mjs` builds it for the native supported platform and collects dependency notices under `resources/<platform>/keyboard-ranker`; the Go adapter owns provisioning and process lifetime ([ADR 036](../decisions/036-two-stage-keyboard-ranking.md)).
