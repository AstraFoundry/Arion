# Arion

Local-first voice dictation and Pinyin input for macOS and Windows, built with [MyGo](https://github.com/egoist/mygo) native Go UI.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![Native Qwen3-ASR](https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E)](docs/decisions/020-native-qwen-asr.md)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion voice input window and floating dictation pill](docs/readme/hero.png)

![Arion voice dictation in action](docs/readme/demo.gif)

## Background and problem statement

An input method sits at the lowest layer of desktop text entry, directly handling chat messages, documents, search queries, and coding prompts. Existing desktop voice input tools generally involve two trade-offs:

1. **Cloud dictation services rely on remote speech recognition and server-side text post-processing**, requiring raw microphone audio, edit history, or window context to leave the machine.
2. **Built-in OS offline dictation lacks context adaptation**, struggling with mid-sentence Chinese/English code-switching, domain vocabulary learning, spoken filler cleanup, and application-specific formatting.

[AstraFoundry](https://astrafoundry.si) built Arion to provide an **open-source, local-first, self-hostable** voice and Pinyin input method: running streaming speech recognition, offline full-Pinyin typing, correction learning, and optional contextual polishing on the device while keeping raw audio and encryption keys on the user's machine.

### What it addresses

- **Offline recognition and field safety boundaries**
  After the recognition model bundle is downloaded and verified against pinned SHA-256 digests, dictation runs offline on the device. Microphone audio is held in memory only for the active session and dropped when it ends (never written to disk or uploaded). Password fields and inputs with unknown security attributes are skipped automatically ([DSK-08](specs/001-desktop-input.md)).
- **Distance-calibrated software input gain for quiet dictation**
  Includes stepped software input gain from `−6 dB` (15 cm close talk) to `+12 dB` (120 cm) in 3 dB increments ([DSK-12](specs/001-desktop-input.md)). Users can dictate quietly at a shared desk while scaling only Arion's capture stream, without modifying the OS input volume.
- **Mid-sentence Chinese/English code-switching and draft isolation**
  Recognizes Mandarin, English, and mixed technical terms in a single utterance (e.g. *"把这个 PR merge 到 main 之前，先跑一下 CI"*) without pre-selecting a language. Streaming partial drafts stay inside Arion's floating pill and are committed to the focused editor once on release.
- **On-device edit learning and optional two-stage polishing**
  - **Edit learning**: In native macOS editors that expose `AXValue`, when a user manually edits a misrecognized word after insertion, Arion extracts the correction pair locally, encrypts it into the on-device vocabulary, and provides a one-click undo in the pill ([ADR 014](docs/decisions/014-edit-learning.md)).
  - **Two-stage contextual polishing**: When enabled, releasing the dictation key invokes an isolated `arion-llm` (`llama.cpp`) worker (or an explicitly configured cloud text model) to remove non-semantic fillers, apply mid-sentence spoken self-corrections, disambiguate terms against visible window/cursor context, and format output for chat, email, docs, or coding prompts.

## Key advantages

| Area | Implementation |
| --- | --- |
| **On-device inference & zero-knowledge sync** | Speech recognition and vocabulary learning run locally. Personal vocabulary is encrypted with AES-256-GCM using a key stored in macOS Keychain or Windows DPAPI. The sync relay (`services/sync`) stores ciphertext and routing metadata only, supports Docker self-hosting, and never holds decryption keys. |
| **Unified global dictation & native IME** | Hold the global dictation key (default right `⌥` on macOS, right `Alt` on Windows) in any focused editor, or use the built-in system IME hosts (macOS `ArionInput.app` / Windows TSF `ArionTSF.dll`) for offline full-Pinyin typing with standalone `Shift` language toggling. |
| **Native Go desktop shell** | Built with [MyGo](https://github.com/egoist/mygo) native Go UI without Electron or a webview. Runs as a menu bar / system tray app with a click-through floating pill that joins macOS fullscreen Spaces without taking editor focus. |
| **Deterministic session & error boundaries** | Recognition drafts are isolated from committed text; cancellation or focus changes immediately invalidate in-flight generations; missing models or permissions surface explicit error codes and never silently fall back to cloud ASR. |
| **Full-stack open source** | Licensed under [AGPL-3.0](LICENSE). The desktop app, native ASR/LLM workers, encrypted sync relay, update channel service, and evaluation benchmarks all live in this repository. |

## Architecture and technical features

macOS on Apple silicon and Windows x64 are the primary desktop targets (`P0`). Intel macOS is unsupported ([ADR 021](docs/decisions/021-apple-silicon-only-macos.md)); iOS and Android are future targets (`P2`).

- **Global dictation key and native IME hosts** — Hold the dictation key (default right `⌥` on macOS, right `Alt` on Windows) to dictate and release to insert once; a quick tap keeps listening until the next press (`Esc` cancels, [DSK-02](specs/001-desktop-input.md)). Includes the `ArionInput.app` macOS InputMethodKit host with offline full-Pinyin typing and encrypted learning ([ADR 006](docs/decisions/006-macos-input-method-host.md), [ADR 017](docs/decisions/017-offline-pinyin-keyboard.md)) and the Windows x64 TSF input-method host ([Windows input method](docs/operations/windows-input-method.md)).
- **On-device native speech recognition** — Qwen3-ASR 1.7B runs through native Swift MLX 8-bit on Apple silicon and native `speech.cpp` Q8_0 (CPU/Vulkan) on Windows x64 ([ADR 020](docs/decisions/020-native-qwen-asr.md)).
- **Conservative correction and edit learning** — Applies deterministic rules (`arion-rules-v4`) and user-approved encrypted vocabulary; on macOS, edits made to inserted text in native editors are observed through Accessibility and learned on-device ([ADR 014](docs/decisions/014-edit-learning.md)).
- **Optional staged local or cloud polishing** — Runs once after key release ([ADR 023](docs/decisions/023-final-only-polishing.md)) using a shared two-stage selector/formatter pipeline ([ADR 024](docs/decisions/024-shared-polishing-pipeline.md)), scene-aware formatting (**Match the app**, [ADR 012](docs/decisions/012-context-scene-selection.md)), and on-device cursor/screen/history term extraction ([ADR 013](docs/decisions/013-rewrite-context-on-device.md)). Local polishing runs Qwen3.5-2B (`Q4_K_M` GGUF) in an isolated `arion-llm` (`llama.cpp`) worker on macOS Apple silicon and Windows x64 ([ADR 018](docs/decisions/018-staged-gguf-rewriting.md), [ADR 025](docs/decisions/025-windows-local-polishing.md), [local rewrite](docs/operations/local-rewrite.md)); optional cloud polishing requires an explicit provider choice while keeping raw audio on-device ([ADR 022](docs/decisions/022-cloud-polishing.md)).
- **Native Go desktop shell (FSD)** — Built with [MyGo](https://github.com/egoist/mygo) (`v0.4.6`) using Feature-Sliced Design in Go ([ADR 001](docs/decisions/001-desktop-native-boundaries.md), [ADR 007](docs/decisions/007-sunbeam-design-system.md)), with first-run onboarding, local diagnostic ZIP export, and live-switched English / Simplified Chinese UI localization ([i18n](docs/frontend/i18n.md)).
- **Local encryption, self-hosted sync relay, and signed update channels** — AES-256-GCM local vocabulary encryption ([ADR 005](docs/decisions/005-local-encryption.md)), `services/sync` Go ciphertext relay with `noop` and `oidc` account gates ([ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)), desktop loopback PKCE sign-in ([account sign-in](docs/project/account-signin.md)), and Ed25519-signed `stable`/`beta` update channels served by `services/updates` ([ADR 019](docs/decisions/019-center-update-channels.md), [deploy/center](deploy/center/README.md)).

## Current state and limitations

Arion is a working macOS (`arm64`) menu bar dictation app and has an early Windows (`amd64`) preview ([release status](docs/operations/release-status.md)). Native speech, local polishing, and IME primitives pass on development hosts and CI, while clean-machine and broad third-party editor acceptance remain open.

Not yet implemented or still open:

- **Local candidate ranking (`RNK-01`)** — The ranking contract and evaluation harness exist, but current recognizers expose a single hypothesis, so candidate ranking is not active.
- **Desktop memory sync client & enrollment (`MEM-01` / `SYNC-10`)** — The `services/sync` relay and desktop OIDC sign-in views/service are implemented, but the desktop SQLite outbox and device enrollment remain open.
- **Broad IME & Windows host acceptance (`IMK-01` / `TSF-01` / `RWT-RUN-01`)** — macOS `ArionInput.app` and Windows `ArionTSF.dll` / local GGUF polishing pass isolated and development-host checks, while clean-machine installation, broad editor matrix acceptance, and Windows edit learning remain open.
- **Rewrite blind human evaluation & general LLM correction (`LRW-REVIEW-01` / `COR-01`)** — Development benchmarks are recorded in [`benchmarks/results`](benchmarks/README.md), while blind human ratings (`LRW-A06`) remain open; standalone LLM correction outside the polishing pipeline is not adopted.
- **Windows publisher signing & published upgrade acceptance (`REL-01`)** — Canonical macOS `arm64` Developer ID signing, notarization, and IMK provisioning profile validation are in place ([deployment](docs/operations/deployment.md)); Windows Authenticode publisher signing and clean-host old-to-new channel upgrade verification remain open.

See the [agent handoff](docs/project/agent-handoff.md), [release status](docs/operations/release-status.md), and [validation report](docs/quality/validation.md) for exact verification evidence and task gates.

## Quick start

### Prerequisites

- **Go 1.27.1** with `cgo` enabled (`go.work` uses Go automatic toolchain selection; `packages/core` and `services/sync` also check standalone with `GOWORK=off`).
- **macOS (`arm64`)**: Full **Xcode 26.4.1** application / Swift 6.3.1 and the **Metal Toolchain** component. Command Line Tools alone cannot compile `ArionASR`:
  ```sh
  sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
  sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
  xcodebuild -downloadComponent MetalToolchain
  ```
- **Windows (`amd64`)**: Native C/C++ toolchain, CMake, MinGW-w64 GCC (`CC=x86_64-w64-mingw32-gcc`) for native keyboard/cgo builds, and NSIS for installer builds.
- **Node 24** (`.nvmrc`) and **pinned `pnpm@10.33.0`** (`package.json`) for repository checks, evaluation tooling, and the static marketing site (`apps/site`). `npm` and `yarn` are rejected.

### Build and run

```sh
make install        # install the pinned pnpm workspace (--frozen-lockfile)
make check          # docs, JSON, Go layer boundaries, core/sync and evaluation tests
make desktop-check  # desktop tests, keyboard checks, and go vet (cgo)
make desktop-dev    # stage native runtimes and launch Arion Dev in dev mode
make build desktop  # package Arion.app + DMG (macOS) or NSIS installer (Windows) under apps/desktop/build/
make e2e            # native window lifecycle E2E on a real desktop session
make e2e-macos      # owned product scenarios against the packaged Arion.app
make site-check     # marketing site (apps/site): i18n, types, lint, tests, static export
```

On first run, `make desktop-dev` opens onboarding, requests microphone and Accessibility permissions, and downloads the default recognition model bundle into `~/Library/Application Support/Arion/models/` (macOS) or `%AppData%\Arion\models\` (Windows). After model provisioning, dictation works offline. Run `make help` to list all benchmark, evaluation, and diagnostic targets.

### macOS development build and TCC permission diagnostics (`make desktop-dev`)

> Full operational reference: [`docs/operations/local-dev.md` — macOS development build and TCC permission diagnostics](docs/operations/local-dev.md#macos-development-build-and-tcc-permission-diagnostics).

- **1–3. Toolchain & 2–5 minute silent cold build**: `make desktop-dev` compiles `ArionASR` via `xcodebuild -quiet`. If you see `tool 'xcodebuild' requires Xcode` or `missing Metal Toolchain`, switch `xcode-select` to `/Applications/Xcode.app/Contents/Developer`, run `sudo xcodebuild -runFirstLaunch`, and run `xcodebuild -downloadComponent MetalToolchain`. The first run fetches 11 SwiftPM packages and compiles `mlx-swift` + Metal shaders silently for 2–5 minutes before printing `Staged native ASR for darwin-arm64.`.
- **4–6. Bundle location, silent `open`, and avoiding the ad hoc `CDHash` permission loop**: `make desktop-dev` builds and launches `Arion Dev` (`si.astrafoundry.arion.dev`) at `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app` (printed at startup). Every `make desktop-dev` run re-signs `Arion Dev.app` ad hoc and invalidates prior TCC grants (`CDHash`). After enabling a permission in System Settings, quit `Arion Dev` from the menu bar and reopen the **already-built cached bundle** instead of rerunning `make desktop-dev`:
  ```sh
  open "~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app"
  ```
  `open` exits `0` with no terminal output and `Arion Dev` is a menu bar app (`LSUIElement`, no Dock icon or window); check the top-right menu bar icon. Do not open any leftover `apps/desktop/.mygo/dev/.../Arion Dev.app` (delete `apps/desktop/.mygo` if present).
- **7–8. Resetting Microphone, Accessibility, Input Monitoring, and Screen Recording TCC state**:
  - **Microphone** (no `+` button in System Settings; does not re-prompt after denial): run `tccutil reset Microphone si.astrafoundry.arion.dev`, reopen `Arion Dev.app` via `open`, and click **Authorize** in **Settings › General** (not the Microphone device tab).
  - **Accessibility / Input Monitoring** (switch is on but app reports unauthorized or `hold_register PERMISSION_DENIED`): run `tccutil reset Accessibility si.astrafoundry.arion.dev && tccutil reset ListenEvent si.astrafoundry.arion.dev`, re-authorize, and reopen via `open`.
  - **Screen Recording** (no drag tile; cache path is hidden): in System Settings › Screen & System Audio Recording, click **+**, press `⌘⇧G`, paste `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64`, select `Arion Dev.app` in the file list (press `⌘2` for List view; do not click the top parent-folder dropdown), enable the switch, and reopen via `open`.
- **9. Ad hoc limits (`ARTIFACT_UNVERIFIED` & edit learning)**: Installing the system input method (`ArionInput.app`) returns `ARTIFACT_UNVERIFIED` in ad hoc dev builds (requires a Developer ID signed `make build desktop` build with `MACOS_IME_PROVISIONING_PROFILE`). Dictation edit learning ([ADR 014](docs/decisions/014-edit-learning.md)) reads `AXValue` and works in native Cocoa editors (**Notes**, **TextEdit**), not in Electron apps (Cursor, VS Code) or terminals.

## Documentation

### Project and specifications

- [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) — rules, boundaries, and workflows for contributors and coding agents.
- [Documentation Index (`docs/README.md`)](docs/README.md) — top-level map of all documentation domains.
- [Product Scope (`docs/project/README.md`)](docs/project/README.md) — confirmed scope, explicit boundaries, and stack status table.
- [Architecture (`docs/project/architecture.md`)](docs/project/architecture.md) — data flow, polishing pipeline, runtime shape, and privacy boundaries.
- [Repository Ownership (`docs/project/repository.md`)](docs/project/repository.md) — Go workspace modules, FSD/DDD package boundaries, and working rules.
- [Specifications (`specs/README.md`)](specs/README.md) — normative requirements (`001` through `009`) and acceptance scenario IDs.
- [Agent Handoff (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) — implemented milestones, open `P0`/`P1`/`P2` tasks, and completion gates.
- [Architecture Decision Records (`docs/decisions/README.md`)](docs/decisions/README.md) — ADRs `001` through `025`.

### Engineering, operations, and evaluation

- [Frontend Conventions (`docs/frontend/README.md`)](docs/frontend/README.md) — Feature-Sliced Design (FSD) in native Go and [Interface Localization (`docs/frontend/i18n.md`)](docs/frontend/i18n.md).
- [Backend Conventions (`docs/backend/README.md`)](docs/backend/README.md) — Domain-Driven Design (DDD) rules for `packages/core` and `services/sync`.
- [Local Development (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) — toolchains, data locations, environment variables, and troubleshooting.
- [Deployment & Release Gates (`docs/operations/deployment.md`)](docs/operations/deployment.md), [Release Status (`docs/operations/release-status.md`)](docs/operations/release-status.md), [Arion Center (`deploy/center/README.md`)](deploy/center/README.md), and [Sync Relay Runbook (`deploy/sync/README.md`)](deploy/sync/README.md).
- [Quality & Testing (`docs/quality/README.md`)](docs/quality/README.md), [Validation Report (`docs/quality/validation.md`)](docs/quality/validation.md), and [Native E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md).
- [Research Evidence (`docs/research/README.md`)](docs/research/README.md), [Benchmark Methodology (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md), [Evaluation Runners (`tools/evaluation/README.md`)](tools/evaluation/README.md), and [Benchmark Results (`benchmarks/README.md`)](benchmarks/README.md).

## Contributors

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star History

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## License

Arion is developed by [AstraFoundry](https://github.com/AstraFoundry) and licensed under [AGPL-3.0](LICENSE) ([ADR 009](docs/decisions/009-agpl-license.md)). Report bugs in [GitHub Issues](https://github.com/AstraFoundry/Arion/issues).
