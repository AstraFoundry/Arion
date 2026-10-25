<div align="center">

# Arion

Local-first voice dictation and Pinyin input for macOS and Windows, built with [MyGo](https://github.com/egoist/mygo) native Go UI.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md)

<p align="center">
  <a href="https://x.com/AstraFoundry"><img src="https://img.shields.io/badge/X-%40AstraFoundry-000000?logo=x&logoColor=white" alt="X @AstraFoundry" style="display: inline-block; margin: 2px;" /></a>
  <a href="https://discord.com/invite/72kh465mMm"><img src="https://img.shields.io/badge/Discord-Join_Chat-5865F2?logo=discord&logoColor=white" alt="Discord" style="display: inline-block; margin: 2px;" /></a>
  <a href="https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae"><img src="https://img.shields.io/badge/Feishu-Group_Chat-3370FF" alt="Feishu Group" style="display: inline-block; margin: 2px;" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue.svg" alt="License: AGPL-3.0" style="display: inline-block; margin: 2px;" /></a>
</p>

<p align="center">
  <a href="go.work"><img src="https://img.shields.io/badge/Go-1.27.1-00ADD8?logo=go&logoColor=white" alt="Go 1.27.1" style="display: inline-block; margin: 2px;" /></a>
  <a href="apps/desktop/go.mod"><img src="https://img.shields.io/badge/MyGo-v0.4.6-4B5563" alt="MyGo v0.4.6" style="display: inline-block; margin: 2px;" /></a>
  <a href="docs/decisions/020-native-qwen-asr.md"><img src="https://img.shields.io/badge/ASR-Qwen3_1.7B-0F766E" alt="Native Qwen3-ASR" style="display: inline-block; margin: 2px;" /></a>
  <a href=".nvmrc"><img src="https://img.shields.io/badge/Node-24-339933?logo=nodedotjs&logoColor=white" alt="Node 24" style="display: inline-block; margin: 2px;" /></a>
  <a href="package.json"><img src="https://img.shields.io/badge/pnpm-10.33.0-F69220?logo=pnpm&logoColor=white" alt="pnpm 10.33.0" style="display: inline-block; margin: 2px;" /></a>
</p>

![Arion voice dictation in action](docs/readme/demo.gif)

</div>

## Overview

Cloud dictation tools require sending raw microphone audio and window context to remote servers, while built-in OS offline dictation struggles with mid-sentence Chinese/English code-switching, domain terminology, and application-specific formatting.

[AstraFoundry](https://astrafoundry.si) built **Arion** as an open-source, local-first, self-hostable voice and Pinyin input method. Speech recognition, offline full-Pinyin typing, correction learning, and contextual polishing run on your device while keeping raw audio and encryption keys on your machine.

## Features

- **Hold-to-Talk & Native System IME** — Hold the global dictation key (default right `⌥` on macOS, right `Alt` on Windows) in any focused editor to speak Mandarin, English, or mixed technical terms (e.g. *"把这个 PR merge 到 main 之前，先跑一下 CI"*), and release to insert once; tap once to keep listening until the next press (`Esc` cancels). Also includes native system IME hosts (macOS `ArionInput.app` and Windows `ArionTSF.dll`) with offline Simplified Chinese full-Pinyin typing and `Shift` language toggling.
- **On-Device Speech Recognition** — Runs Qwen3-ASR 1.7B locally via Swift MLX (8-bit) on Apple silicon and `speech.cpp` (Q8_0, CPU/Vulkan) on Windows x64 ([ADR 020](docs/decisions/020-native-qwen-asr.md)). After the model bundle is downloaded and SHA-256 verified, dictation runs completely offline. Microphone audio stays in memory only during the active session, password fields are skipped automatically, and stepped software input gain (`−6 dB` to `+12 dB`) supports quiet dictation at shared desks.
- **Edit Learning & Draft Isolation** — Streaming partial drafts stay inside Arion's floating pill and are committed to the editor only on release. Deterministic cleanup rules and user-approved vocabulary are applied locally; on macOS, manual edits made to inserted text in native editors are learned automatically with one-click undo ([ADR 014](docs/decisions/014-edit-learning.md)).
- **Two-Stage Contextual Polishing** — Optional polishing runs after key release ([ADR 023](docs/decisions/023-final-only-polishing.md)) to strip spoken fillers, resolve mid-sentence self-corrections, disambiguate terms against visible window/cursor context ([ADR 013](docs/decisions/013-rewrite-context-on-device.md)), and format text for the active app (**Match the app**, [ADR 012](docs/decisions/012-context-scene-selection.md)). Runs Qwen3.5-2B (`Q4_K_M` GGUF via `llama.cpp`) locally on macOS Apple silicon and Windows x64 ([ADR 018](docs/decisions/018-staged-gguf-rewriting.md), [ADR 025](docs/decisions/025-windows-local-polishing.md)), or connects to an explicit opt-in cloud model (ChatGPT sign-in or OpenAI, Anthropic, Gemini, DeepSeek, OpenRouter API key) while keeping raw audio on-device ([ADR 022](docs/decisions/022-cloud-polishing.md)).
- **Native Go Desktop Shell** — Built with [MyGo](https://github.com/egoist/mygo) (`v0.4.6`) using Feature-Sliced Design in Go without Electron or a webview ([ADR 001](docs/decisions/001-desktop-native-boundaries.md), [ADR 007](docs/decisions/007-sunbeam-design-system.md)). Runs in the menu bar / system tray with a click-through floating pill that joins fullscreen Spaces without stealing focus, plus live-switched English and Simplified Chinese UI ([i18n](docs/frontend/i18n.md)).
- **Zero-Knowledge Sync & Signed Updates** — Personal vocabulary is encrypted locally with AES-256-GCM using keys stored in macOS Keychain or Windows DPAPI ([ADR 005](docs/decisions/005-local-encryption.md)). The repository includes `services/sync` (a Docker self-hostable Go ciphertext relay that never holds decryption keys, [ADR 015](docs/decisions/015-hosted-relay-account-authorization.md)) and `services/updates` (Ed25519-signed `stable` and `beta` update channels, [ADR 019](docs/decisions/019-center-update-channels.md)).

## Platform Support

| Platform | Architecture | Status |
| --- | --- | --- |
| **macOS** | Apple silicon (`arm64`) | Working menu bar dictation app, native MLX ASR, local GGUF polishing, and `ArionInput.app` IMK host (Intel macOS is unsupported, [ADR 021](docs/decisions/021-apple-silicon-only-macos.md)) |
| **Windows** | 64-bit (`x64` / `amd64`) | Early preview with native `speech.cpp` ASR, local GGUF polishing, and `ArionTSF.dll` IME host |

See [Release Status](docs/operations/release-status.md), [Agent Handoff](docs/project/agent-handoff.md), and [Validation Report](docs/quality/validation.md) for current milestone progress and open release gates.

## Quick Start

### Prerequisites

- **Go 1.27.1** with `cgo` enabled (`go.work` uses automatic toolchain selection).
- **macOS (`arm64`)**: Full **Xcode 26.4.1** application (Swift 6.3.1) and the **Metal Toolchain** component:
  ```sh
  sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
  sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
  xcodebuild -downloadComponent MetalToolchain
  ```
- **Windows (`amd64`)**: Native C/C++ toolchain, CMake, MinGW-w64 GCC (`CC=x86_64-w64-mingw32-gcc`), and NSIS.
- **Node 24** (`.nvmrc`) and **pinned `pnpm@10.33.0`** (`package.json`).

### Build and Run

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

On first launch, `make desktop-dev` opens onboarding, requests microphone and Accessibility permissions, and downloads the default recognition model bundle into `~/Library/Application Support/Arion/models/` (macOS) or `%AppData%\Arion\models\` (Windows). For macOS ad hoc `CDHash` permission behavior and development troubleshooting, see [Local Development (`docs/operations/local-dev.md`)](docs/operations/local-dev.md#macos-development-build-and-tcc-permission-diagnostics).

## Documentation

| Area | Entry Points |
| --- | --- |
| **Overview & Guidelines** | [Documentation Index (`docs/README.md`)](docs/README.md) · [Product Scope (`docs/project/README.md`)](docs/project/README.md) · [Agentic Coding Guidelines (`AGENTS.md`)](AGENTS.md) |
| **Architecture & Specs** | [Architecture (`docs/project/architecture.md`)](docs/project/architecture.md) · [Repository Ownership (`docs/project/repository.md`)](docs/project/repository.md) · [Specifications (`specs/README.md`)](specs/README.md) · [ADRs (`docs/decisions/README.md`)](docs/decisions/README.md) |
| **Frontend & Backend** | [Frontend FSD (`docs/frontend/README.md`)](docs/frontend/README.md) · [UI Localization (`docs/frontend/i18n.md`)](docs/frontend/i18n.md) · [Backend DDD (`docs/backend/README.md`)](docs/backend/README.md) |
| **Operations & Deployment** | [Local Development (`docs/operations/local-dev.md`)](docs/operations/local-dev.md) · [Deployment (`docs/operations/deployment.md`)](docs/operations/deployment.md) · [Release Status (`docs/operations/release-status.md`)](docs/operations/release-status.md) · [Arion Center (`deploy/center/README.md`)](deploy/center/README.md) · [Sync Relay (`deploy/sync/README.md`)](deploy/sync/README.md) |
| **Quality & Benchmarks** | [Quality & Testing (`docs/quality/README.md`)](docs/quality/README.md) · [Native E2E (`docs/quality/e2e.md`)](docs/quality/e2e.md) · [Research (`docs/research/README.md`)](docs/research/README.md) · [Evaluation Runners (`tools/evaluation/README.md`)](tools/evaluation/README.md) · [Benchmark Results (`benchmarks/README.md`)](benchmarks/README.md) |

## Contributors

[![AstraFoundry/Arion contributors](https://contrib.rocks/image?repo=AstraFoundry/Arion)](https://github.com/AstraFoundry/Arion/graphs/contributors)

## Star History

[![AstraFoundry/Arion Star History Chart](https://api.star-history.com/svg?repos=AstraFoundry/Arion&type=Date)](https://star-history.com/#AstraFoundry/Arion&Date)

## License

Arion is developed by [AstraFoundry](https://github.com/AstraFoundry) and licensed under [AGPL-3.0](LICENSE) ([ADR 009](docs/decisions/009-agpl-license.md)). Follow [@AstraFoundry on X](https://x.com/AstraFoundry), join the community on [Discord](https://discord.com/invite/72kh465mMm) or [Feishu](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae), and report bugs in [GitHub Issues](https://github.com/AstraFoundry/Arion/issues).
