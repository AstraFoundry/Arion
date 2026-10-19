# Desktop app

MyGo native Go module: the menu bar app, its windows (overlay pill, menu bar panel, settings, onboarding, macOS consent guide), OS adapters and the composition root.

| Path | Role |
| --- | --- |
| `main.go`, `bootstrap/` | Composition root: wires the core coordinator, model runtime, platform adapters, tray, windows, hotkeys and sounds; E2E control mode behind `ARION_E2E_CONTROL` |
| `ui/` | FSD native UI (`app → pages → widgets → features → entities → shared`); all copy in `ui/shared/copy`, Sunbeam tokens in `ui/shared/theme`, embedded images in `ui/shared/assets` ([design system](../../docs/frontend/design-system.md)) |
| `infrastructure/audio` | miniaudio capture (CoreAudio/WASAPI, 16 kHz mono) and real-time fixture replay |
| `infrastructure/asr` | sherpa-onnx streaming recognizer and offline finalizer adapters |
| `infrastructure/provision`, `modelruntime` | Verified model downloads and the resident bundle chosen by evaluation |
| `infrastructure/macos`, `windows` | Permissions, target capture and insertion routes ([ADR 003](../../docs/decisions/003-insertion-routes.md)); hold-key listeners (listen-only event tap, low-level keyboard hook) |
| `infrastructure/holdkey` | Pure interpretation of raw key events into hold-key down/up/chord, tested on every OS |
| `infrastructure/settings`, `secrets`, `vocabulary` | Preferences JSON, OS-kept key, encrypted vocabulary with learned entries and their counts |
| `cmd/arion-eval-driver` | Benchmark driver running the production pipeline |
| `cmd/arion-e2e-driver`, `e2e/` | Product scenario driver, owned editor fixture and native window tests |
| `cmd/arion-models`, `cmd/arion-icon` | Catalog review/offline import tool; app icon renderer (ink bars on Sunbeam yellow) |

Speech runtimes are cgo libraries: build with `make build desktop` / `make desktop-dev`, which stage the pinned dylibs/DLLs into `resources/` and set the rpath. See [local development](../../docs/operations/local-dev.md) and [architecture](../../docs/project/architecture.md).

Settings › About checks for updates and installs them: the latest published [GitHub release](https://github.com/AstraFoundry/Arion/releases) is the only source, and an archive must carry the Ed25519 signature of the key in `desktop.json` ([ADR 010](../../docs/decisions/010-in-app-updates.md)). Checks run at launch and every 6 hours while "Check automatically" is on; the menu bar panel shows a pending update. Development runs and offline/E2E profiles never check.

The interface defaults to English; choose English or Simplified Chinese in General settings or onboarding. The choice persists and updates existing windows without restarting dictation. Copy keys, formatting, native permission resources and verification commands are documented in [i18n](../../docs/frontend/i18n.md).

First launch opens onboarding. Completing it opens and focuses Settings, keeping Arion visible after setup. Later launches with completed onboarding remain in the menu bar or system tray; Settings can be reopened there.

Dictation starts from the toggle shortcut (default ⌥Space / Alt+Space) or, optionally, a hold key chosen in Dictation settings: fn, right Option or right Command on macOS; right Alt or right Ctrl on Windows. Holding the key past 300 ms dictates until release; a shorter tap keeps dictating until the next press; another key pressed during a hold that started the session cancels it (a chord such as fn+arrow inserts nothing). On macOS the hold key needs Input Monitoring consent; without it settings shows the missing consent with a way to grant it, and the toggle shortcut keeps working.

On macOS, granting Accessibility or Input Monitoring from a packaged `Arion.app` opens the privacy pane in System Settings with a small guide under its window: drag the Arion tile into the list (or switch Arion on if it is listed). The guide follows the window, hides while another app is in front, and closes once consent arrives, returning to the Arion window that asked. A build run outside an app bundle (`go run`, tests) has nothing to drag and uses the system prompts.
