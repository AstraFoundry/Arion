# MyGo source investigation

## Scope and source identity

This is source and documentation analysis of the user-selected framework, not
reverse engineering of a shipped MyGo app. The inspected revision is
[`v0.4.6`, `bc111b6d2cca221716ad99ddea1580449ad8c18e`](https://github.com/egoist/mygo/tree/bc111b6d2cca221716ad99ddea1580449ad8c18e).
`git ls-remote` resolved both `HEAD` and `refs/tags/v0.4.6` to this revision.
The [Go module proxy record](https://proxy.golang.org/github.com/egoist/mygo/@v/v0.4.6.info)
independently identifies that tag and commit. Do not use `@latest` in the
project's dependency or tool declaration.

Context7 resolution was attempted for `mygo egoist` and `egoist/mygo`. Both
returned `/zjutjh/mygo`, an unrelated Go HTTP framework. No documentation query
was made against that incorrect match. The evidence below is the selected
repository's primary source, inspected in a temporary checkout. No framework
GUI or app integration test was run during this investigation.

## What "native" means here

MyGo supports two interfaces. Its native `ui` package is a Go toolkit drawn by
MyGo on the GPU; it is not a webview or a wrapper around a complete set of
AppKit/WinUI controls. macOS rendering uses Metal and Windows uses Direct3D 11,
with Windows WARP support. Platform text engines shape text. The OS still owns
windows, menus, event dispatch and accessibility integration.

The other interface loads web pages through WKWebView, WebView2 or WebKitGTK.
A single app can mix these interfaces. Arion should start from the **native
template**, because the user selected MyGo as the native desktop framework.
Native UI calls Go directly; a TypeScript bridge, React, Vite and Bun are not
required for this mode. Sources:
[architecture](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/architecture.md),
[rendering](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/ui/rendering.md),
[native template](https://github.com/egoist/mygo/tree/bc111b6d2cca221716ad99ddea1580449ad8c18e/cmd/mygo/template/native).

## Capability and gap matrix

| Need | Source observation | Arion implication |
|---|---|---|
| Desktop shell | macOS and Windows backend implementations, windows, menus, tray, login items and single-instance support exist. | Use MyGo for the shell and configuration UI; validate tray lifecycle on both OSes. |
| Non-focusing overlay | `Window.ShowInactive`, `AlwaysOnTop`, `SkipTaskbar` and transparent-window options exist. `FocusedWindow` refers to this application's windows. | An overlay is feasible; discovering and preserving a foreign editor's focus still needs a platform adapter. |
| Global shortcut | Public `GlobalShortcut.Register(acc, func())` invokes callbacks on the main thread. macOS uses Carbon `RegisterEventHotKey`; Windows uses `RegisterHotKey` with `MOD_NOREPEAT`. | Toggle dictation can use it. The API does not provide global key-release or modifier-only hold events; push-to-talk needs a separately designed adapter. |
| Audio | `audio` decodes, mixes and **plays** sound. The platform outputs are Audio Queue and WASAPI. | Playback is available; microphone capture, input selection, resampling, device failure handling and capture permission requests must be implemented elsewhere. Do not mistake playback for recording. |
| Clipboard | Text and richer transfer formats have public read/write APIs. | Clipboard-backed commit is possible only with a separate paste/event adapter and explicit clipboard ownership policy. It is not itself text injection. |
| UI accessibility | Native widgets publish their own accessibility roles, names and values for assistive technology. | This does not read another application's field. Foreign-app context capture requires macOS AX and Windows UI Automation or another verified OS adapter. |
| Text input methods | Native editable widgets implement the receiving side: `NSTextInputClient` on macOS and IMM32 document-feed/reconversion on Windows. | This is **not** an InputMethodKit input method or Windows TSF text service. The user-configurable system IME needs a separate integration and packaging design. |
| ASR and correction | No speech-model inference contract is supplied by the selected framework. | Keep ASR, correction, inference-process lifetime and model distribution outside the MyGo UI layer. |
| Mobile | Backends select desktop OSes and browser `js/wasm`; unsupported targets use a stub backend. | No iOS/Android application or keyboard-extension support is established. Reuse portable domain/contracts later; design mobile shells separately. |

Evidence for the matrix:

- [Desktop APIs and global shortcut public contract](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/native.md).
- [Window API](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/window.go).
- [macOS hotkey implementation](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/internal/darwin/hotkey.go).
- [Windows hotkey implementation in `tray.go`](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/internal/windows/tray.go).
- [Audio contract](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/audio.md).
- [Clipboard transfer contract](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/data-transfer.md).
- [UI accessibility](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/ui/accessibility.md).
- [Text input client contract](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/ui/text-input-client.md).
- [Windows backend selection](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/backend_windows.go) and [unsupported backend selection](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/backend_other.go).

The negative findings are bounded to the public documentation and targeted
source searches for capture, global release events, foreign-app accessibility,
`SendInput`, InputMethodKit and TSF. They are not a claim that such functionality
could never be added. Some internal E2E helpers synthesize input for framework
testing; those are not a supported production integration API.

## Toolchain and development workflow

The selected [go.mod](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/go.mod)
declares `module github.com/egoist/mygo` and **`go 1.27.1`**. The native app
should pin the framework and its CLI together:

```go
go 1.27.1

require github.com/egoist/mygo v0.4.6

tool github.com/egoist/mygo/cmd/mygo
```

This is the relevant dependency/tool portion, not a complete app module. Run
`go mod tidy` to resolve transitive dependencies and commit `go.sum`. Native
configuration belongs in `mygo.json`; the template contains `name`,
`identifier`, `version` and `out`, with an icon in `resources/icon.png`.

```sh
cd apps/desktop
go tool mygo dev
go tool mygo build -platform darwin/universal,windows/amd64
```

The native template's boot sequence is `App.WhenReady` -> `NewWindow` with
`Content: ui.View(view)` -> `App.Run` on the main goroutine. Long capture,
inference or storage operations must not run in shortcut callbacks or views.
Use workers and schedule UI state updates through `Window.Update`. Sources:
[application threading](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/app.md),
[getting started and build commands](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/getting-started.md).

At investigation time the host's `go version` was `go1.25.7 darwin/arm64`,
which cannot compile this dependency with `GOTOOLCHAIN=local`. The
[Go download feed](https://go.dev/dl/?mode=json) lists a stable Go 1.27 toolchain;
the Go 1.27.1 Darwin ARM64 auto-toolchain module info endpoint returned HTTP
200. This establishes remote availability, not successful local installation
or a desktop build. Configure the desktop build environment for Go 1.27.1 or
newer and record actual build results separately. Do not lower the dependency's
`go` requirement to bypass this constraint.

MyGo itself needs no cgo and uses purego/Objective-C runtime on macOS and
syscall/COM on Windows. That does not guarantee Arion's speech libraries will
be cgo-free. Native inference dylibs/DLLs or helper executables introduce their
own ABI, architecture, signing and redistribution requirements. Its CLI also
defaults `CGO_ENABLED=0` unless the environment sets it; an in-process cgo
inference adapter needs an explicit build configuration. Source:
[CLI Go build environment](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/cmd/mygo/proc.go).

## Distribution and platform coverage

MyGo packages macOS `.app`/DMG and Windows executables/NSIS installers, and
documents signing, notarization, resources and signed updates. macOS defaults
to minimum OS 12.0. Native UI does not need WebView2; it is needed for Windows
web pages. Windows backend selection covers `amd64` and `arm64`; a backend's
existence does not validate every Arion model/runtime on both architectures.
Linux and browser UI support exist upstream but are outside Arion's initial
delivery scope. Source:
[distribution](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/docs/distribution.md).

For microphone access, set a real
[`NSMicrophoneUsageDescription`](https://developer.apple.com/documentation/bundleresources/information-property-list/nsmicrophoneusagedescription)
and the hardened-runtime
[`com.apple.security.device.audio-input` entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.device.audio-input)
in the packaged macOS app.
MyGo's web permission APIs do not grant a native capture adapter permission.
Stable development and release bundle identifiers matter because MyGo dev
uses a separate `.dev` identifier and permission/data identity. Verify this
with packaged native-capture builds, rather than treating a successful
`go run` as permission coverage.

The framework is MIT licensed; retain the required license notice in
redistribution. Its `v0.x` release series should be treated as an evolving API.
Upstream has unit tests and GUI E2E workflows for desktop platforms, but their
existence does not prove Arion's foreign-editor operations work. Sources:
[license](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/LICENSE),
[upstream CI](https://github.com/egoist/mygo/blob/bc111b6d2cca221716ad99ddea1580449ad8c18e/.github/workflows/ci.yml).

## Recommended repository boundaries and implementation gates

Keep `apps/desktop` as the MyGo module with `mygo.json`, native UI, resources
and OS adapters. Place portable speech-session, correction and memory domain
logic in a separate Go module under `packages/`; it must not import MyGo.
Maintain model/runtime assets and licensing manifests separately from source.
The docs and specs are shared product contracts, not an embedded web frontend.
These paths are project design recommendations, not upstream requirements.

Before starting feature implementation, verify these gates in dependency order:

1. P0: resolve the pinned desktop module with a compatible Go toolchain; build
   and run one native window and tray lifecycle on macOS and Windows.
2. P0: establish native microphone capture and permission behavior with a
   bounded PCM stream; test device changes and cancellation.
3. P0: preserve the external editor's focus; probe context and text-commit
   adapters against representative editors. Treat permission denial and
   unsupported controls as explicit capability results.
4. P0: choose a verified ASR/correction process or FFI boundary, model licenses
   and per-platform artifacts. Benchmark inference outside the UI thread.
5. P1: add hold-key behavior only after a release-event adapter is tested;
   MyGo global shortcuts alone satisfy toggle operation.
6. P1: validate packaging/signing with microphone and accessibility behavior,
   including separate development and release identifiers.

Unresolved here: Arion's exact minimum Windows version, native capture backend,
cross-application text-injection API, system-IME registration and model ABI.
The framework investigation establishes where to build these adapters; it
does not establish that they already exist.

## Arion foundation validation

The repository subsequently resolved MyGo v0.4.6 with automatic Go 1.27.1 selection. Its minimal native module passed Go compile/tests/vet on this macOS arm64 host, and the pinned CLI produced an ad hoc signed `.app` and DMG. Direct Windows amd64 cross-compilation and the pinned CLI executable build also passed; NSIS installer creation was skipped on this host. The generated shell plist declares macOS 12.0 minimum; this is not a minimum-version promise for future inference/IME adapters. No interactive UI, microphone, IMK/TSF or Windows runtime behavior was exercised. CI native jobs are configured but have not run remotely in this investigation. See [local development](../operations/local-dev.md).

### Observed on Windows (CI, windows-2022)

The native E2E job runs on the GitHub-hosted Windows Server 2022 desktop. Overlay, settings and onboarding windows open, draw and close; Notepad insertion through `SendInput` and the focus-change guard pass. Two platform facts surfaced:

- Layered child windows, which MyGo uses for the controls of a hidden title bar, need the Windows 8+ `supportedOS` entry in the executable manifest. `mygo build` embeds it; a bare `go test` binary needs the external `e2e/windows.manifest`.
- `Window.SetContentSize` on a `TitleBarHidden` window sets the outer size without frame adjustment, and Windows 10 still excludes the 8 px resize frame from the client area: a request for 820x600 yields 804x592 content. Arion does not resize windows programmatically; the E2E test resizes by outer bounds.
