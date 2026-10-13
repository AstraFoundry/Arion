# Arion

A local-first voice input monorepo for macOS and Windows, built with [MyGo](https://github.com/egoist/mygo) native Go UI. iOS and Android are future targets.

The product combines streaming speech recognition, conservative correction, local candidate ranking, and synchronized personal memory. Users can choose shortcut dictation or system IME integration. A self-hosted service relays encrypted memory; dictation remains usable offline after model provisioning.

## Current state

Arion is a working macOS menu bar dictation app (Windows code compiles; not yet run on Windows):

- Press the shortcut (default ⌥Space) in any app, speak Chinese, English or both, press it again: a floating pill shows the live draft, and the corrected text is inserted once into the app you were using. Esc cancels.
- Recognition runs on the device with sherpa-onnx. The model bundle with the best measured accuracy that passes the latency gates is **downloaded, verified and loaded automatically**; no model choice is required ([ADR 004](docs/decisions/004-automatic-model-selection.md)).
- First run walks through microphone and Accessibility permission, model setup, the shortcut and a practice dictation. Settings cover permissions, shortcut, input mode (shortcut or the Arion Voice input method), insertion method, microphone, model, encrypted vocabulary, privacy and about.
- Local-only: no audio or text leaves the device; model downloads are the only network use.

Not yet implemented: candidate ranking, local LLM correction, memory sync, hold-to-talk, Windows TSF and signed releases. See the [agent handoff](docs/project/agent-handoff.md) and [validation](docs/quality/validation.md).

## Start here

1. Read [AGENTS.md](AGENTS.md), the [product scope](docs/project/README.md), and [architecture](docs/project/architecture.md).
2. Read the [specification index](specs/README.md) and [research evidence](docs/research/README.md).
3. Choose an unclaimed task from the [agent handoff](docs/project/agent-handoff.md).
4. Follow [local development](docs/operations/local-dev.md) and run checks before handing work off.

```sh
make install
make check          # docs, boundaries, core/catalog tests, evaluation tooling
make desktop-check  # desktop tests and vet (cgo)
make desktop-dev    # run the app
make desktop-build  # package Arion.app and a DMG
make e2e            # native window lifecycle on a real desktop session
make e2e-macos      # owned product scenarios against the packaged app
```

JavaScript tooling uses pinned pnpm 10.33.0 and Node 24; npm installation is rejected. Native desktop builds use Go 1.27.1 with cgo and the pinned MyGo CLI. Model selection evidence comes from the [evaluation runners](tools/evaluation/README.md) and [paper-grounded metrics](docs/research/benchmark-methodology.md); aggregate reports are in [`benchmarks/results`](benchmarks/README.md). See [repository ownership](docs/project/repository.md) for the workspace layout and [release gates](docs/operations/deployment.md) before distribution.
