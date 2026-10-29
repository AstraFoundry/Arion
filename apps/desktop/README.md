# Desktop app

MyGo native Go module: the menu bar app, its windows (overlay pill, menu bar panel, settings, onboarding, tray hint, update reminder, macOS consent guide), OS adapters and the composition root.

| Path | Role |
| --- | --- |
| `main.go`, `bootstrap/` | Composition root: wires the core coordinator, model runtime, platform adapters, tray, windows, hotkeys and sounds; E2E control mode behind `ARION_E2E_CONTROL` |
| `ui/` | FSD native UI (`app → pages → widgets → features → entities → shared`); all copy in `ui/shared/copy`, Sunbeam tokens in `ui/shared/theme`, embedded images in `ui/shared/assets` ([design system](../../docs/frontend/design-system.md)) |
| `infrastructure/audio` | miniaudio capture (CoreAudio/WASAPI, 16 kHz mono) and real-time fixture replay |
| `infrastructure/asr` | Resident native Qwen3-ASR workers, cancellable chunked previews and final-tail flush |
| `infrastructure/provision`, `modelruntime` | Verified model downloads and the resident bundle chosen by evaluation |
| `infrastructure/macos`, `windows` | Permissions, target capture and insertion routes ([ADR 003](../../docs/decisions/003-insertion-routes.md)); hold-key listeners (listen-only event tap, low-level keyboard hook) |
| `infrastructure/holdkey` | Pure interpretation of raw key events into dictation and translation shortcut gestures (single keys, modifier combinations, hold, tap, chord), tested on every OS |
| `infrastructure/settings`, `secrets`, `vocabulary`, `usage` | Preferences JSON, OS-kept key, encrypted vocabulary with learned entries and their counts, per-day usage counts |
| `infrastructure/wordseg` | Local OS word boundaries for correction learning |
| `infrastructure/diagnostics` | Bounded, content-free local support logs and their export |
| `infrastructure/imebridge`, `tsf` | Local IPC with the input-method hosts; Windows TSF installation and source observation |
| `infrastructure/updatestate`, `netproxy` | Update handles and reminder decisions; proxy choice for downloads and update checks |
| `infrastructure/llm`, `rewriting`, `cloudpolish`, `keystore`, `productnotifications`, `browser`, `internal/` | Local and cloud polishing, credential items, account notices and sign-in helpers; see [repository ownership](../../docs/project/repository.md) |
| `cmd/arion-eval-driver` | Benchmark driver running the production pipeline |
| `cmd/arion-e2e-driver`, `e2e/` | Product scenario driver, owned editor fixture and native window tests |
| `cmd/arion-models`, `cmd/arion-icon` | Catalog review/offline import tool; app icon renderer (ink bars on Sunbeam yellow) |
| `cmd/arion-llm` and other `cmd/arion-*` tools | The `arion-llm` rewrite worker (no cgo); live, rewrite, scene and memory benchmarks, IME and context probes, OCR |

Build with `make build desktop` / `make desktop-dev`, which stage the native ASR/rewrite runtimes and keyboard assets into `resources/`. On Apple silicon, staging compiles `ArionASR` with `xcodebuild`, so full Xcode 26.4.1 (`xcode-select` pointed at `Xcode.app`) and the Metal toolchain (`xcodebuild -downloadComponent MetalToolchain`) are required. `make desktop-dev` outputs `Arion Dev.app` under `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/`. See [local development](../../docs/operations/local-dev.md) and [architecture](../../docs/project/architecture.md).

Settings › About checks for updates and installs them: standard builds query the Arion center (`https://api.arion.astrafoundry.si/v1/updates`, `stable` or `beta` channel, [ADR 019](../../docs/decisions/019-center-update-channels.md)), and an archive must carry the Ed25519 signature of the key in `desktop.json` ([ADR 010](../../docs/decisions/010-in-app-updates.md)). Checks run at launch and every 6 hours while "Check automatically" is on; the menu bar panel shows a pending update. Development runs, custom-center builds, and offline/E2E profiles never check the official feed.

The interface defaults to English; choose English or Simplified Chinese in General settings or onboarding. The choice persists and updates existing windows without restarting dictation. Copy keys, formatting, native permission resources and verification commands are documented in [i18n](../../docs/frontend/i18n.md).

First launch opens onboarding. Completing it opens and focuses Settings, keeping Arion visible after setup, and floats a dismissible card beside the menu bar or tray icon saying Arion stays there. Later launches with completed onboarding remain in the menu bar or system tray; Settings can be reopened there.

Dictation starts globally from the dictation shortcut recorded in Dictation settings or onboarding's dictation key step: a single key (right Option by default, fn, left/right Option, Command, Ctrl, Shift or F1–F20 on macOS; right Alt by default, left/right Alt, Ctrl, Shift, Menu or F1–F24 on Windows) or a modifier combination with a trigger key. An optional translation shortcut (right Command on macOS and right Ctrl on Windows by default) translates into a chosen target language. There is no separate accelerator shortcut. Holding the shortcut past 300 ms dictates until release; a shorter tap keeps dictating until the next press; another key pressed during a hold that started the session cancels it (a chord such as fn+arrow inserts nothing). On macOS the key needs Input Monitoring consent; without it settings and onboarding show the missing consent with a way to grant it, and onboarding waits until the key listens. Profiles that stored the retired accelerator or an "off" hold key load the platform default key. Choices, conflicts and storage are in [dictation shortcuts](../../docs/project/dictation-shortcuts.md).

On macOS, granting Accessibility or Input Monitoring from a packaged `Arion.app` opens the privacy pane in System Settings with a small guide under its window: drag the Arion tile into the list (or switch Arion on if it is listed). The guide follows the window, hides while another app is in front, and closes once consent arrives, returning to the Arion window that asked. A build run outside an app bundle (`go run`, tests) has nothing to drag and uses the system prompts.
