# Desktop app

MyGo native Go module: the menu bar app, its windows (overlay pill, menu bar panel, settings, onboarding), OS adapters and the composition root.

| Path | Role |
| --- | --- |
| `main.go`, `bootstrap/` | Composition root: wires the core coordinator, model runtime, platform adapters, tray, windows, hotkeys and sounds; E2E control mode behind `ARION_E2E_CONTROL` |
| `ui/` | FSD native UI (`app → pages → widgets → features → entities → shared`); all copy in `ui/shared/copy` |
| `infrastructure/audio` | miniaudio capture (CoreAudio/WASAPI, 16 kHz mono) and real-time fixture replay |
| `infrastructure/asr` | sherpa-onnx streaming recognizer and offline finalizer adapters |
| `infrastructure/provision`, `modelruntime` | Verified model downloads and the resident bundle chosen by evaluation |
| `infrastructure/macos`, `windows` | Permissions, target capture and insertion routes ([ADR 003](../../docs/decisions/003-insertion-routes.md)) |
| `infrastructure/settings`, `secrets`, `vocabulary` | Preferences JSON, OS-kept key, encrypted vocabulary |
| `cmd/arion-eval-driver` | Benchmark driver running the production pipeline |
| `cmd/arion-e2e-driver`, `e2e/` | Product scenario driver, owned editor fixture and native window tests |
| `cmd/arion-models`, `cmd/arion-icon` | Catalog review/offline import tool; app icon renderer |

Speech runtimes are cgo libraries: build with `make desktop-build` / `make desktop-dev`, which stage the pinned dylibs/DLLs into `resources/` and set the rpath. See [local development](../../docs/operations/local-dev.md) and [architecture](../../docs/project/architecture.md).
