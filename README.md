# Arion

Local-first voice input for macOS and Windows, built with [MyGo](https://github.com/egoist/mygo) native Go UI.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

[![X @AstraFoundry](https://img.shields.io/badge/X-%40AstraFoundry-000000?logo=x&logoColor=white)](https://x.com/AstraFoundry)
[![Discord](https://img.shields.io/badge/Discord-Join_Chat-5865F2?logo=discord&logoColor=white)](https://discord.com/invite/72kh465mMm)
[![Feishu Group](https://img.shields.io/badge/Feishu-Group_Chat-3370FF)](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go 1.27.1](https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white)](go.work)
[![MyGo v0.4.6](https://img.shields.io/badge/MyGo-v0.4.6-4B5563)](apps/desktop/go.mod)
[![Native Qwen3-ASR](https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E)](docs/decisions/020-native-qwen-asr.md)
[![Node 24](https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white)](.nvmrc)
[![pnpm 10.33.0](https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white)](package.json)

![Arion voice input window and floating dictation pill](docs/readme/hero.png)

![Arion voice dictation in action](docs/readme/demo.gif)

## Overview

Arion is a local-first desktop voice input tool that combines streaming on-device speech recognition, deterministic text cleanup, personal vocabulary learning, context-aware rewriting, and encrypted memory sync.

- **Offline by default** — After initial model provisioning, recognition and local rewriting run entirely on your machine.
- **Strict privacy boundary** — Raw microphone audio stays on the device. Enabled context is sent only when cloud polishing is explicitly selected ([005 Privacy](specs/005-privacy.md)).
- **Focused desktop targets** — macOS on Apple silicon (`arm64`) and Windows (`x64`) are the primary `P0` targets. Intel macOS is unsupported ([ADR 021](docs/decisions/021-apple-silicon-only-macos.md)); iOS and Android are future `P2` targets.

## Key Features

### Voice Input & System Integration

- **Global dictation key** — Hold the dictation key (default right `⌥` on macOS, right `Alt` on Windows) in any app to speak Mandarin, English, or mixed technical text, and release to insert once into the focused editor. A quick tap keeps listening until the next press; `Esc` cancels ([DSK-02](specs/001-desktop-input.md)).
- **macOS InputMethodKit host (`ArionInput.app`)** — Integrates directly as a system input source with voice composition and an offline Simplified Chinese full-Pinyin keyboard ([ADR 006](docs/decisions/006-macos-input-method-host.md), [ADR 017](docs/decisions/017-offline-pinyin-keyboard.md)).
- **Native Go desktop shell** — Built with [MyGo](https://github.com/egoist/mygo) (`v0.4.6`) and the Sunbeam design system using Feature-Sliced Design in Go, with no Electron or webview ([ADR 001](docs/decisions/001-desktop-native-boundaries.md), [ADR 007](docs/decisions/007-sunbeam-design-system.md)). Includes a menu bar popover, a click-through floating pill that joins fullscreen Spaces, onboarding, settings, local diagnostic ZIP export, and live English / Simplified Chinese localization ([i18n](docs/frontend/i18n.md)).

### Speech Recognition, Correction & Rewriting

- **Native on-device ASR** — Runs Qwen3-ASR 1.7B locally via native Swift MLX (8-bit) on Apple silicon and native `speech.cpp` (Q8_0) on Windows x64, with automatic digest-verified model provisioning ([ADR 004](docs/decisions/004-automatic-model-selection.md), [ADR 020](docs/decisions/020-native-qwen-asr.md)).
- **Conservative correction & edit learning** — Applies deterministic rules (`arion-rules-v3`) and user-approved vocabulary while keeping recognition drafts separate from committed text. On macOS, edits made to inserted text are learned on-device with one-click undo ([ADR 014](docs/decisions/014-edit-learning.md)).
- **Staged local rewriting (macOS Apple silicon)** — Polishes speech using a local two-stage GGUF pipeline (`Qwen3.5-2B` via `llama.cpp`) with scene-aware formatting (**Match the app**) and on-device cursor, screen, and recent dictation context ([ADR 011](docs/decisions/011-live-local-rewriting.md), [ADR 012](docs/decisions/012-context-scene-selection.md), [ADR 013](docs/decisions/013-rewrite-context-on-device.md), [ADR 018](docs/decisions/018-staged-gguf-rewriting.md)).
- **Optional cloud polishing** — Explicit opt-in final-text polishing via [Republic](https://github.com/AstraFoundry/republic) using ChatGPT sign-in or one provider API key (OpenAI, Anthropic, Google Gemini, DeepSeek, OpenRouter). Uses the same staged pipeline and context switches as local polishing. Enabled context and recalled vocabulary inform spelling selection; audio stays local ([ADR 022](docs/decisions/022-cloud-polishing.md), [cloud polishing](docs/project/cloud-polishing.md)).

### Security, Sync & Updates

- **Local encryption & self-hostable sync relay** — Personal vocabulary is encrypted locally with AES-256-GCM using keys stored in the macOS login Keychain or Windows DPAPI ([ADR 005](docs/decisions/005-local-encryption.md)). Includes `services/sync`, a self-hostable Go ciphertext relay with `noop` and `oidc` account gates ([ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)), and desktop loopback PKCE sign-in ([account sign-in](docs/project/account-signin.md)).
- **Center-served signed updates** — The in-app updater checks `stable` and `beta` channels on the Arion center (`api.arion.astrafoundry.si`, `services/updates`) and verifies Ed25519 package signatures before installation ([ADR 010](docs/decisions/010-in-app-updates.md), [ADR 019](docs/decisions/019-center-update-channels.md), [deploy/center](deploy/center/README.md)).

## Current Status & Open Gates

Arion is a working macOS (`arm64`) menu bar dictation app. Windows (`amd64`) code compiles and passes headless tests as well as CI window lifecycle, `SendInput`, and hold-key hook E2E tests on Windows Server 2022, though full interactive dictation has not yet been verified end-to-end on a Windows host.

Items still open or not yet active:

- **Local candidate ranking (`RNK-01`)** — The ranking contract and evaluation harness exist, but current recognizers expose a single hypothesis, so candidate ranking is inactive.
- **Desktop memory sync client & enrollment (`MEM-01` / `SYNC-10`)** — The `services/sync` relay and desktop OIDC sign-in flow are implemented; the desktop SQLite outbox, device enrollment, and end-to-end key sync remain open.
- **Windows TSF & local rewrite worker (`TSF-01` / `RWT-RUN-01`)** — Windows Text Services Framework (TSF) IME integration, Windows edit learning, and a Windows local rewrite runtime are not yet implemented.
- **Interactive OS permission & IMK acceptance** — Hold-to-talk gesture logic and the Windows low-level hook pass CI E2E, while macOS event-tap Input Monitoring and clean-machine IMK registration require interactive desktop verification.
- **Rewrite blind human evaluation (`LRW-REVIEW-01` / `COR-01`)** — Development benchmarks are recorded in [`benchmarks/results`](benchmarks/README.md), while blind human ratings (`LRW-A06`) remain open; standalone LLM correction outside the rewrite pipeline is not adopted.
- **Signed public releases (`REL-01`)** — The in-app updater and release staging pipeline are implemented, while production Developer ID signing, notarization, and final bundle identifiers remain pending.

See the [agent handoff](docs/project/agent-handoff.md) and [validation report](docs/quality/validation.md) for detailed verification evidence and task gates.

## Quick Start

### Prerequisites

- **Go 1.27.1** with `cgo` enabled (`go.work` uses Go automatic toolchain selection; `packages/core` and `services/sync` also check standalone with `GOWORK=off`).
- **macOS (`arm64`)** — Xcode Command Line Tools and macOS SDK.
- **Windows (`amd64`)** — Native toolchain required by MyGo, plus MinGW-w64 (`CC=x86_64-w64-mingw32-gcc`) when cross-packaging `cgo`, and NSIS for installer builds.
- **Node 24** (`.nvmrc`) & **pinned `pnpm@10.33.0`** (`package.json`) — Used for repository checks, evaluation tooling, and the static marketing site (`apps/site`). `npm` and `yarn` are rejected.

### Common Commands

```sh
make install        # install the pinned pnpm workspace (--frozen-lockfile)
make check          # docs, JSON, Go layer boundaries, core/sync and evaluation tests
make desktop-check  # desktop tests and go vet (cgo)
make desktop-dev    # stage native runtimes and launch the desktop app in dev mode
make build desktop  # package Arion.app and a DMG under apps/desktop/build/
make e2e            # native window lifecycle E2E on a real desktop session
make e2e-macos      # owned product scenarios against the packaged Arion.app
make site-check     # marketing site (apps/site): i18n, types, lint, tests, static export
```

On first launch, `make desktop-dev` opens onboarding, requests microphone and Accessibility permissions, and downloads the default recognition model bundle into `~/Library/Application Support/Arion/models/` (macOS) or `%AppData%\Arion\models\` (Windows). Once provisioned, dictation works offline. Run `make help` to list all benchmark, evaluation, and diagnostic targets.

## Documentation Map

| Area | Entry Points |
| --- | --- |
| **Guidelines & Index** | [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) · [Documentation Index (`docs/README.md`)](docs/README.md) |
| **Product & Architecture** | [Product Scope (`docs/project/README.md`)](docs/project/README.md) · [Architecture (`docs/project/architecture.md`)](docs/project/architecture.md) · [Repository Ownership (`docs/project/repository.md`)](docs/project/repository.md) |
| **Specs & Decisions** | [Specifications (`specs/README.md`)](specs/README.md) · [Architecture Decision Records (`docs/decisions/README.md`)](docs/decisions/README.md) · [Agent Handoff (`docs/project/agent-handoff.md`)](docs/project/agent-handoff.md) |
| **Frontend & Backend** | [Frontend FSD Conventions (`docs/frontend/README.md`)](docs/frontend/README.md) · [UI Localization (`docs/frontend/i18n.md`)](docs/frontend/i18n.md) · [Backend DDD Conventions (`docs/backend/README.md`)](docs/backend/README.md) |
| **Operations & Deployment** | [Local Development (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) · [Deployment & Release Gates (`docs/operations/deployment.md`)](docs/operations/deployment.md) · [Sync Relay Runbook (`deploy/sync/README.md`)](deploy/sync/README.md) · [Center Deploy (`deploy/center/README.md`)](deploy/center/README.md) |
| **Quality & Benchmarks** | [Quality & Testing (`docs/quality/README.md`)](docs/quality/README.md) · [Validation Report (`docs/quality/validation.md`)](docs/quality/validation.md) · [Native E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md) · [Research (`docs/research/README.md`)](docs/research/README.md) · [Methodology (`docs/research/benchmark-methodology.md`)](docs/research/benchmark-methodology.md) · [Evaluation Runners (`tools/evaluation/README.md`)](tools/evaluation/README.md) · [Benchmark Results (`benchmarks/README.md`)](benchmarks/README.md) |

## Contributors

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star History

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## License

Arion is developed by [AstraFoundry](https://github.com/AstraFoundry) and licensed under [AGPL-3.0](LICENSE) ([ADR 009](docs/decisions/009-agpl-license.md)). Follow [@AstraFoundry on X](https://x.com/AstraFoundry), join the community on [Discord](https://discord.com/invite/72kh465mMm) or [Feishu](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae), and report bugs in [GitHub Issues](https://github.com/AstraFoundry/Arion/issues).
