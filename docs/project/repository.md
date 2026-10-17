# Repository and ownership

## Workspace

| Path | Owner and dependency boundary |
| --- | --- |
| `apps/desktop` | Separate Go module; MyGo lifecycle and FSD native UI; depends on core |
| `apps/desktop/ui` | `app → pages → widgets → features → entities → shared` Go packages; same-layer slices isolated; shared segments may import each other |
| `apps/desktop/bootstrap` | Composition root: wiring, windows, tray, hotkeys, main-thread state, E2E control mode |
| `apps/desktop/infrastructure` | OS input/capture/context and model/storage adapters; implements domain-owned ports |
| `apps/desktop/internal/center` | Compiled public center profile, validation and fingerprint; no runtime ENV or credential access |
| `apps/desktop/internal/account` | OIDC public client for hosted sync: loopback PKCE, in-memory access token, refresh rotation, waitlist join ([account sign-in](account-signin.md)) |
| `apps/desktop/infrastructure/keystore` | Refresh-token item in the macOS Keychain or Windows Credential Manager |
| `apps/desktop/infrastructure/browser` | System browser for http(s) authorization URLs |
| `apps/desktop/cmd` | Evaluation and E2E drivers, the `arion-llm` rewrite worker (no cgo), live rewrite benchmarks, model catalog tool, icon renderer |
| `apps/desktop/infrastructure/llm` | GenAI purego binding, prompt template, KV-reusing engine and worker protocol (worker side), process client (app side) |
| `apps/desktop/infrastructure/rewriting` | Rewrite capability service: bundled runtime location, model provisioning, worker lifecycle, `Rewriter` port |
| `packages/core/domain` | Session and candidate invariants; standard-library only |
| `packages/core/contracts` | Versioned transport-neutral DTOs; no SDK or platform handles |
| `packages/core/application` | Dictation coordinator and preferences; depends on domain-owned ports |
| `packages/contracts` | Shared wire examples/documentation, no duplicate business types |
| `apps/site` | Static Next.js marketing site (pnpm, TypeScript FSD, en/zh/ja); independent of the Go modules and the desktop shell ([ADR 016](../decisions/016-marketing-site.md)) |
| `services/sync` | Go DDD relay; account-unaware core, `AccountGate` port with `noop` and `oidc` adapters ([ADR 015](../decisions/015-hosted-relay-account-authorization.md)) |
| `deploy` | Relay images, self-host compose and the hosted relay's compose stack with its own PostgreSQL |
| `models` | Go module embedding the reviewed catalog: provenance, digests, licenses, bundles, evaluation summaries; no weights |
| `scripts` | Dependency-free repository checks and platform-neutral workflow entry points |
| `tools/evaluation` | pnpm workspace: metrics, driver protocols, real-run orchestration and isolated Python development inference adapters |
| `benchmarks` | Versioned research corpora/audio, raw observations, reproduction sources, provenance and reports; no weights/runtime environments or personal recordings |
| `tests/e2e` | Product action/assertion scenarios; requires live OS adapters |
| `apps/desktop/e2e` | Build-tagged real native window lifecycle test |
| `specs` | Normative requirements, acceptance scenarios and task traceability |
| `docs/research` | Evidence and technical proposals; cannot override accepted specs |

The four Go modules (`packages/core`, `models`, `apps/desktop`, `services/sync`) are connected by `go.work`. `services/sync` is checked with `GOWORK=off`, the same way as the portable core. Its domain package is standard-library only and does not import the application or infrastructure layers. `cmd/arion-llm` and the `infrastructure/llm` packages build with `CGO_ENABLED=0` and must never import the speech runtime; `scripts/stage-desktop-resources.mjs` prepares packages in local builds and CI. It calls `scripts/stage-llm-runtime.mjs` for the worker, pinned GenAI/ONNX Runtime libraries and notices in `resources/darwin-arm64/llm`, and collects module licenses and model provenance in `resources/licenses`. Intel and Windows have no approved rewrite runtime/model pair yet. `apps/desktop/go.mod` pins MyGo, sherpa-onnx Go bindings and miniaudio, and replaces core/models with the local modules; reconsider the local `replace` before publishing modules. `GOWORK=off` allows portable core and catalog checks without upgrading their toolchain. The desktop module builds with cgo (`CGO_ENABLED=1`) because the speech runtimes are native libraries.

## FSD in native Go

Slice directories become Go packages. Only exported package symbols are the public API; there is no `index.ts`. Internal implementation uses unexported names or Go `internal/` packages. The original TypeScript snippets remain explanatory examples; they are not a requirement to add TypeScript or React.

UI slices can consume core contract DTOs through a shared adapter boundary. UI must not implement ASR/correction/ranking rules or import an infrastructure adapter directly. The app composition root wires dependencies. `ui/entities` owns UI-facing domain projections without widget code or transport clients.

All application copy is centralized in the English/Simplified Chinese catalogs under `ui/shared/copy`; `runtime.Store.Copy()` selects immutable translations from persisted preferences. Entity projections and transient errors carry typed message keys ([i18n](../frontend/i18n.md)). Native theme/layout values stay in shared theme packages when needed. Apply UI craft skills using MyGo's existing native primitives, not CSS-only instructions.

## DDD in desktop and relay

Core domain defines invariants and abstractions; core application orchestrates them. Desktop infrastructure provides local implementations. The relay domain owns device authorization, opaque revisions, retention and conflict validation, never speech, plaintext memory, or accounts. Its HTTP package calls application services rather than SQL. `infrastructure/memory` implements the same ports as PostgreSQL for tests and is not composed into `cmd/sync`.

## Working rules

Do not create empty implementations that return success, mock a recognizer as production behavior, or silently use cloud services. Every adapter requires a capability statement, explicit unsupported result and acceptance evidence. OS target claims require tests on that OS. Test doubles live only in `_test.go` files; the E2E control mode replaces the microphone with fixture audio but runs the production recognizer, correction and insertion.

JavaScript tooling is a pnpm workspace pinned by the root package manager and lockfile. `make` exposes recurring workflows; do not introduce npm commands or a second lockfile. Go modules remain independently versioned. Benchmark/OS drivers are concrete adapter implementations, not frontend business logic.
