# MyGo architecture

This guide explains how MyGo is put together: the layers, the threading
model, how Go talks to the native toolkits without cgo, how pages and Go
exchange typed messages, how MyGo draws native UI, and how to extend the
framework safely. Read it before changing anything under `internal/`.

## Goals and constraints

- **Low overhead.** A hello-world app is a ~7 MB binary with a ~35 MB
  physical footprint on macOS (mostly AppKit/WebKit), ~62 MB with the
  processes WKWebView runs for its page, GPU and network, and idles at 0%
  CPU. A window of native UI starts no webview process: idle, a small one
  takes 44 MB on macOS, and the counter example 65 to 67 MB on Linux, tiled
  to half of a 4K display at scale 2, 15 MB of it the window's buffer,
  without loading Mesa (122 to 155 MB with it). Nothing
  polls: all work is driven by native events or explicit wake-ups.
- **No cgo.** Everything builds with `CGO_ENABLED=0`, so any platform can be
  cross-compiled from any machine. Native APIs are called at run time through
  [purego](https://github.com/ebitengine/purego) (`dlopen` + assembly
  trampolines) on macOS and Linux and through the `syscall` package on
  Windows, never through `import "C"`.
- **Two kinds of windows.** Web pages show in the system webview:
  WKWebView on macOS, WebKitGTK 4.1 (4.0 as a fallback) on Linux, WebView2
  on Windows (amd64 and arm64); no browser engine is bundled. Native UI is
  drawn by MyGo itself, in Go, with Metal, Direct3D 11 or OpenGL, or on the
  CPU, and text from the system's own engines (see [Native UI](#native-ui-ui));
  an app whose windows all show it needs no webview at all, and runs in
  browsers too, compiled to WebAssembly, on canvases drawn with WebGL 2
  (see [Browsers](#browsers-internalweb)). On unsupported
  platforms the `internal/unsupported` backend makes `App.Run` fail with a
  clear error while everything still compiles.
- **Bun is dev tooling only.** It builds and tests the TypeScript bridge, and
  installs and runs the web template's tools (Vite, TypeScript, the
  mygo-cli package); projects of native UI need none. Nothing Bun-related
  ships in an app.
- **Great DX over API parity.** The API has the feel of Electron (app
  lifecycle, windows, menus, dialogs) but is Go-first: typed IPC with a
  generated TypeScript client, `http.Handler` for custom protocols, typed
  event structs, blocking calls that are safe from any goroutine, and native
  UI written in Go alone.

## Repository layout

```
.                       package mygo: the public API
├── app.go              lifecycle, quit sequence, Dock, paths (paths.go)
├── window.go           Window: the native window, its options, state and events
├── flyout.go           NewFlyout: windows owned by a parent, placed next to an anchor
├── page.go             Page: the web page a window shows, its loading, Eval and events
├── content.go          Content: windows showing native UI instead of a page
├── webview.go          WebView: web pages in windows of native UI, under it
├── ipc.go              Bind/BindAs, method calls, Event[T], CallerWindow
├── plugin.go           Plugin and Use: services bound as "plugin:<name>"
├── channel.go          Channel[T]: values streamed to a call's page
├── typescript.go       GenerateTypeScript / WriteTypeScript (uses internal/tsgen)
├── protocol.go         custom schemes served by http.Handler, FileServer
├── frontend.go         the app's frontend: relative URLs, devUrl, mygo://localhost
├── menu.go             Menu/MenuItem model, roles, native item updates
├── dialog.go modules.go clipboard.go: shell, clipboard, screen, theme, tray, shortcuts, notifications
├── loop.go             main-thread queue: postMain / onMain / await
├── events.go           listener lists and the Preventable event types
├── single_instance.go  RequestSingleInstanceLock over a Unix socket
├── dev.go signal_*.go  IsDev, the `mygo dev` ready signal, quitting on SIGINT/SIGTERM
├── backend_*.go        picks the backend per GOOS
├── internal/
│   ├── platform/       the contract every backend implements
│   ├── darwin/         macOS: AppKit + WKWebView through the Objective-C runtime
│   ├── linux/          Linux: GTK 3 + WebKitGTK through dlopen
│   ├── windows/        Windows: Win32 + WebView2 through syscall and COM
│   ├── web/            browsers (js/wasm): native UI on canvases, through syscall/js
│   ├── unsupported/    stub for other platforms
│   ├── fake/           in-memory backend for unit tests
│   ├── bridge/         embeds bridge.js, built from packages/bridge
│   ├── surface/        the connection between a window and its native UI
│   ├── scene/          display lists and glyph atlases, what renderers draw
│   ├── text/           fonts, shaping, line breaking, editing, glyph rasterization
│   ├── vec/            the coverage rasterizer of glyphs and paths
│   ├── raster/         the CPU renderer of scenes
│   ├── gpu/            the instances GPU renderers draw, and gputest/ for their tests
│   ├── gpu/d3d11/      the Direct3D 11 renderer of scenes, and device/ for its devices
│   ├── gpu/metal/      the Metal renderer of scenes
│   ├── gpu/gl/         the OpenGL and WebGL 2 renderer of scenes
│   ├── svg/            SVG parsing and drawing, for icons and images
│   ├── tsgen/          TypeScript client generator
│   ├── accelerator/    parses "CmdOrCtrl+Shift+K"
│   ├── update/         update manifests, signatures, archives and delta updates
│   ├── idlemem/        measures apps' memory once idle, for the benchmarks
│   └── e2e/            GUI tests and benchmarks against the real backend (MYGO_E2E=1)
├── packages/           Bun workspace (with the examples' frontends):
│   ├── bridge/         the runtime injected into pages (→ internal/bridge/bridge.js)
│   ├── runtime/        mygo-runtime, the npm package apps and generated clients import
│   └── cli/            mygo-cli, the npm package of the CLI, and in npm/ its
│                       per-platform binary packages
├── plugins/            official plugins, each a Go package and its npm
│                       package (@mygo-plugins/<name>) side by side: fetch,
│                       websocket, sqlite; and Go only: updater, the update window,
│                       a web page or native UI (updater/native), and
│                       terminal, a view of native UI running programs with
│                       libghostty-vt
├── ui/                 native UI: views, layout, widgets, text editing, Tester;
│                       ui/chart: charts without a look, on ui's public API
├── transfer/           immutable data items, representations, lazy providers and drag effects
├── audio/              sound: players, the mixer, the output of each platform, WAV;
│                       audio/vorbis and audio/mp3 decode Ogg Vorbis and MP3
├── cmd/mygo/           the CLI: init, generate, dev, build, doctor
├── examples/           hello, todo, frameless, native; counter-native, vibrancy,
│                       effort-slider, flyout, menubar-flyout, piano and
│                       gallery (native UI); webview (web views in native UI)
├── docs/               the user guides, the official plugins' pages
│                       (plugins/), and this architecture guide
└── website/            the website, with these docs: TanStack Start, prerendered
                        and served by Cloudflare Workers as static assets
```

## Layers

```
 user code ──► package mygo ──► platform.Backend ──► darwin | linux | windows | web | unsupported
                 ▲    │               ▲                  │
                 │    └─ handlers ────┘ (AppHandler,     └─ purego ─► AppKit/WebKit, GTK/WebKitGTK
                 │                       WindowHandler)
 page (JS) ◄── bridge.js ◄── Eval (batched) ─┘
     └──── postMessage(JSON) ──► WindowHandler.Message
```

- **`package mygo`** owns all behavior that is not platform specific: option
  defaults, the window registry, event listeners, the quit sequence, IPC
  routing and encoding, menus, protocol handling, trust checks. It never
  calls a native API directly.
- **`internal/platform`** is a small, synchronous contract: a `Backend`
  (event loop, window factory, menus, dialogs, clipboard, …), a `Window`
  (chrome + webview), and two handler interfaces the core implements to
  receive events (`AppHandler`, `WindowHandler`). Options arrive fully
  defaulted; backends never invent policy.
- **Backends** translate the contract to native calls. They keep native
  objects alive, map native callbacks back to Go objects and report events.

Keeping policy in the core is what lets `internal/fake` test almost all
behavior without a GUI, and keeps each backend a thin translation layer.

## Threading model

Cocoa and GTK must be driven from the thread that started the process. The
rules are:

1. **The main goroutine is locked to the main thread** (`runtime.LockOSThread`
   in `mygo.go`'s `init`), and `App.Run` must be called from it. `Run`
   initializes the backend and blocks in the native event loop. Before
   that no backend is initialized (the Linux one has not even loaded GTK),
   yet `main` already runs on the main thread, where `onMain` calls
   functions directly, and generate mode never initializes it. So a public
   method that reaches the backend either keeps its value in the core
   until the backend can take it, for settings (`SetActivationPolicy`
   through `AppOptions`, `SetMenu` once ready, `Theme.SetSource` and
   `Dock.SetMenu` right after `Init`), or starts with `needsApp`, which
   panics on the main thread before `Run` with the name of the call,
   rather than letting it crash on Linux, hang in `await` or answer wrong
   on macOS. Only calls that work before `Init` on every backend (`Locale`,
   packaging info, login items, URL schemes, `IsOnBattery`) do neither.
2. **Every `platform` method is called on the main thread, and every handler
   callback runs there.** The backend never needs locks for its own state.
3. **Every public method is safe from any goroutine.** `loop.go` provides:
   - `postMain(fn)` appends to a FIFO queue and calls `Backend.Signal`, which
     makes the backend call `AppHandler.Dispatch` (→ `loop.drain`) on the main
     thread. macOS uses a `CFRunLoopSource` added in all common modes (so it
     fires while menus track, windows resize and modal panels run); Linux uses
     `g_idle_add`, deduplicated with an atomic flag.
   - `onMain(fn)` runs `fn` directly when already on the main thread, else posts
     it and waits. After shutdown posted work is dropped instead of
     deadlocking.
   - `await(ch)` waits for an asynchronous native result. Off the main thread
     it is a plain channel receive. **On the main thread it pumps native events
     with `Backend.Step`** until the value arrives; whoever produces the value
     calls `deliver`, which sends and then `Backend.Wake`s the loop. This is
     what makes `Page.Eval`, `CapturePage` and dialogs usable from event
     listeners without deadlocking. Nested runs of the dispatch source are
     expected and safe.
4. **Event listeners run on the main thread**, synchronously, so cancelable
   events (`OnClose`, `OnBeforeQuit`, `OnWillNavigate`, …) can answer the
   native toolkit immediately. **IPC calls run on their own goroutines**, so
   bound methods may block.
5. **Validate in the caller's goroutine.** Anything that can panic on bad
   input (for example `WindowOptions.BackgroundColor`) is checked before
   hopping to the main thread, so the panic points at the user's call.

Main-thread-only fields are marked as such in comments (for example
`Window.native`, `Window.trusted`). Fields shared with other goroutines are
guarded by a mutex or atomic.

## Native interop without cgo

purego gives three primitives, used everywhere:

- `purego.SyscallN(fn, args...)` for integer/pointer arguments. Fast, no
  reflection; it allocates only the array of its arguments, which escapes.
  Floats cannot be passed this way, nor structs, but for those C passes in
  integer registers, as Core Foundation's `CFRange`, which goes as its two
  fields, and comes back as `r1` and `r2`.
- `purego.RegisterFunc(&typedFn, addr)` for signatures with floats or
  structs (`NSRect`, `CGFloat`, `GdkRGBA*` …). These are created once at
  startup; calling them uses reflection, so they are kept off hot paths.
- `purego.NewCallback(fn)` turns a Go function into a C function pointer. At
  most ~2000 callbacks can exist per process and they are never freed, so
  **callbacks are created once per signature at startup** and user data (a
  window id, a request id) identifies the target. Never create callbacks per
  window, per request or per call.

### macOS (`internal/darwin`)

- `objc.go` loads the frameworks, caches selectors and classes, and wraps
  `objc_msgSend`: `send(obj, "selector:", args...)` for integer arguments and
  pre-registered typed variants (`msgRect`, `msgSetFloat`, `msgInitWindow`,
  …) for the rest. On amd64, methods returning structs larger than 16 bytes go
  through `objc_msgSend_stret`.
- Classes are defined at run time with `objc.RegisterClass`: `MyGoWindow`
  (borderless-friendly `NSWindow`), `MyGoWebView` (records the last
  `mouseDown:` for drag regions), `MyGoWindowDelegate` (window, navigation,
  UI and script-message delegate in one object per window), the app delegate,
  scheme handler, menu and tray targets. Only protocols that exist at run
  time are adopted (`WKScriptMessageHandler` is not registered, and WebKit
  does not need it).
- **Overrides call super with `sendSuper`** (`sendSuperSize` for an
  `NSSize`), naming the class they are defined in, never purego's
  `objc.ID.SendSuper`: that resolves super from the object's class, which
  key-value observing replaces with a generated subclass, so the override
  would call itself until the stack overflows. `TestSuperFromDefinedClass`
  rejects it.
- **The web view is not the content view.** A plain `NSView` is, holding the
  web view and, with vibrancy, an `NSVisualEffectView` behind it. WebKit
  docks the inspector next to the web view in its superview; were that the
  window's frame view, AppKit would draw a broken legacy title bar from then
  on.
- **Memory is managed by hand.** Objects created with `alloc`/`init` are owned
  (+1) and must be released; convenience constructors return autoreleased
  objects. Code that creates temporary objects runs inside `withPool`,
  which keeps the goroutine on its thread until it pops the pool: a pool
  belongs to the thread that pushed it, and a goroutine other than the
  main one, as those reading the bundle for `App.Name`, may otherwise
  resume on another thread, where popping it crashes.
  Delegates and windows are released with `autorelease` from
  `windowWillClose:` because AppKit still uses them while closing.
- **Blocks.** Completion handlers passed to Apple APIs are created with
  `objc.NewBlock` and released right after the call (the callee copies
  them). Blocks received from Apple (decision and completion handlers) are
  invoked with `callBlock`; when they are called later they are
  `_Block_copy`d first and released after use.
- **Coordinates.** AppKit's origin is the bottom-left of the primary screen;
  MyGo's is the top-left. `rectToMac`/`rectFromMac` convert.
- **Quitting.** Cmd+Q ends in `applicationShouldTerminate:`, which runs the
  core quit sequence and then stops the run loop so `App.Run` returns (the
  reply is `NSTerminateCancel`; AppKit never calls `exit`). Quit Apple Events
  (Dock > Quit, AppleScript, logout) are handled by our own handler installed
  in `applicationWillFinishLaunching:`, so senders get a success reply, or
  error -128 when a listener cancels.

### Linux (`internal/linux`)

- `ffi.go` `dlopen`s GLib, GObject, GIO, GDK, GTK 3, cairo, GdkPixbuf
  and, when installed, AppIndicator. WebKitGTK 4.1 (4.0), JavaScriptCore
  and libsoup 3 (2.4), which only windows showing web pages need, load
  with the first such window (`webKit`), or to clear browsing data: an app
  that shows only native UI never maps them and the 75 or so libraries
  they bring, some 7 MB and 20 ms of startup. Without them, `errWebKit`
  says what to install when such a window is created. Symbols from newer
  WebKitGTK versions are bound optionally and feature-detected
  (`webkitWebViewCallAsyncJavascriptFunction != nil`).
- Signals are connected with `g_signal_connect_data`, passing the window id as
  user data; `Backend.window(data)` resolves it and ignores closed windows.
  GDK event structs are read at fixed 64-bit offsets (`field[T]`).
- The main thread is the one that called `gtk_init_check`; `IsMainThread`
  compares `gettid`. `Step` is `g_main_context_iteration(NULL, TRUE)` and
  `Wake` is `g_main_context_wakeup`.
- `Bounds` are those of the frame a reparenting X11 window manager puts
  around the window (`gdk_window_get_frame_extents`): GTK positions the
  frame (NORTH_WEST gravity) and sizes the window inside it, without the
  frame. `ContentBounds` are where the content widget is
  (`gdk_window_get_origin` plus its offset below the menu bar). The
  frame's extents are known once the window manager framed the window
  that shows, which its first synthetic configure report tells; until
  then a window expects the frame it had, or the one seen last around a
  decorated window that is resizable or not as it is (openbox: 1, 20, 1
  and 5 px at the bottom, or 1). Wayland has no frame in `Bounds`.
- GTK geometry changes are asynchronous: `SetBounds` remembers the requested
  rectangle, within the sizes GTK gives the window (`constrain`), with the
  frame it expects, as does a window about to show, which X has where GTK
  created it, or where it was, until the window manager places it;
  `Center` moves that rectangle along. `Bounds` reports that rectangle
  until a configure event reports its size and, on X11, its position, or
  the window manager's own (synthetic) report of the size comes, or
  another size, which the window manager or the user chose. A window being
  placed waits for the window manager's report: a reparenting one first
  puts its frame where it created it. If the frame is not the one
  expected, the rectangle grows with it around the size asked for inside
  it (`SetContentBounds`, `UseContentSize`, a window that showed before),
  or GTK is asked for the size inside the new frame. A report of the
  previous size was sent before the window manager took the request
  (openbox sends one when the size hints change); GTK would ask for that
  size again from it, so the backend asks for the new one again.
- On Wayland, the configure event that activates a window can carry the
  size of the last buffer it drew, from before a resize GTK already took
  (Mutter sends one when native UI drawing with GL resizes from its first
  frames), and GTK goes back to that size. So a window drawing with GL
  keeps the latest bounds asked for while it is not focused, and once it
  is, an idle callback asks for them again if the window went back to the
  size it had, unless it is maximized, full screen or tiled. GTK skips a
  request for the size it asked for last: the callback first takes the
  configure and asks for the size the window has.
- A window the user cannot resize is never smaller than its default size,
  which `SetBounds` sets too, and `SetResizable(false)` to the size it has,
  or than its natural size, which GTK makes 200x200 when the window's child
  has none, as the web view: the box around the web view asks for 1x1.
- GTK gives windows without decorations no resize borders, so the outer
  5 px of the page of a frameless window, or one with a hidden title bar,
  resize it (16 px along the edges from a corner resize the corner). The web view's `motion-notify-event` shows
  a resize cursor there, keeping WebKit's cursor to restore, and its
  `button-press-event` calls `gtk_window_begin_resize_drag` with the press,
  as a Wayland compositor requires; neither event reaches WebKit. Nothing
  resizes a maximized or full screen window, and a tiled one only resizes
  at the edges the window manager allows, as with GTK's own decorations.
- `gtk_window_realize` announces server-side decorations to a Wayland
  compositor that speaks `org_kde_kwin_server_decoration` (KWin, COSMIC,
  Sway) for every window GTK does not decorate itself, one without
  decorations too, and the compositor draws its title bar on it. A
  frameless window is realized before it shows and announced client-side
  instead (`gdk_wayland_window_announce_csd`, looked up at startup): GDK
  keeps that for the `GdkWindow` and requests it again whenever the window
  maps or the compositor answers with another mode, so the window gets no
  title bar unless the compositor decorates every window. Without the
  protocol (Mutter), or on X11, which reads the hint of
  `gtk_window_set_decorated`, nothing decorates the window anyway.
- A hidden title bar (`titlebar.go`) is a window without decorations whose
  web view is in a `GtkOverlay`, under a `GtkHeaderBar` for each side of
  `gtk-decoration-layout` that names window buttons. Each bar shows only
  that side's minimize, maximize and close (`decoration-layout`), and CSS
  clears its background and, with `TitleBarHeight`, its minimum height.
  GTK makes the buttons (they need the bar inside a `GtkWindow`) and hides
  those the window cannot use, and they act as a header bar's: iconify,
  toggle maximized, close. The bars are measured before the web view
  exists, from their natural size, for the script that tells the first
  page, then from their allocations; a change of the layout setting
  rebuilds them. A Wayland compositor that decorates windows itself
  (`gdk_wayland_display_prefers_ssd`: the default mode of
  `org_kde_kwin_server_decoration_manager`, server on KWin, Hyprland and
  Sway) gets no bars: such a window keeps GTK's announcement of server-side
  decorations, unlike a frameless one, so the compositor's title bar, or a
  tiling compositor's lack of one, stands for the buttons, whatever the
  layout says.
- `WEBKIT_DISABLE_DMABUF_RENDERER=1` is set unless the user set it, which
  avoids blank webviews on NVIDIA drivers, VMs and containers.
- XDG desktop portal calls go through `portalCall` (`portal.go`), which
  first registers the app's ID, the name of its desktop entry, with
  `org.freedesktop.host.portal.Registry`: the portal only accepts that
  before any other call, and needs it for apps outside a sandbox that the
  desktop did not launch. Methods that may involve the user answer with a
  `Response` signal on a request object; `portalRequest` routes it to a
  callback by the object's path.
- On Wayland, where apps grab no keys, global shortcuts go through the
  `GlobalShortcuts` portal (`hotkey_portal.go`). A session binds its
  shortcuts once, so every change closes the session and creates one that
  binds them all; the ids are canonical accelerators and the session token
  is named after the app, so desktops keep the keys the user picked. The
  portal may ask the user to confirm new shortcuts and needs the app's ID
  (portals from 1.21 refuse without it), so the app must be installed.
  KDE Plasma 5 binds the shortcuts listed in `CreateSession`'s options, as
  the portal's drafts had it: portals before 1.17 pass them on, later ones
  drop them and Plasma 5 then binds nothing; MyGo logs the shortcuts the
  desktop bound no keys to. The activation token of an `Activated` signal
  becomes the display's startup notification id while the callback runs,
  so the window it shows or focuses may take the focus.

### Windows (`internal/windows`)

- Win32 is called through `syscall` (`LazyDLL` procs; system DLLs are loaded
  by absolute path from the system directory). `IsMainThread` compares the
  thread id recorded during package initialization. The loop is
  `GetMessageW`; `Signal` and `Wake` post messages to a hidden top-level
  application window, which also receives tray, hot key, theme
  (`WM_SETTINGCHANGE`) and display broadcasts. `Quit` posts `WM_QUIT`, which
  modal loops (dialogs, menus) forward, so quitting works while they run.
- **COM.** `comCall` calls vtable methods; its indices come from
  `WebView2.h` (constants in `webview2.go`). Handler objects (event and
  completion handlers) share one vtable whose callbacks are created once and
  route `Invoke` to a Go function per object; the objects live in a map until
  COM releases them. Call wrappers are `go:uintptrescapes`, so
  `uintptr(unsafe.Pointer(&x))` arguments stay valid.
- **WebView2 without a loader DLL.** `createEnvironment` finds the Evergreen
  runtime in the registry (`EdgeUpdate\ClientState\<channel>\EBWebView`)
  and calls `CreateWebViewEnvironmentWithOptionsInternal` of its
  `EmbeddedBrowserWebView.dll`, as `WebView2Loader.dll` does; a loader next to
  the executable wins. `internal/webview2` finds the runtime, for the backend
  and for `mygo doctor`: a directory holding that library for this
  architecture and the browser process `msedgewebview2.exe` at its root (the
  library alone is the Edge browser's, whose environments fail with
  `ERROR_NOT_SUPPORTED`), and EdgeUpdate's record of a runtime outlives its
  files. The first window that shows a web page starts
  the environment (`startEnvironment`), and fails without a runtime: windows
  that show native UI need none, so an app whose windows all do runs
  without WebView2. The environment and each controller are created
  asynchronously: window methods that need the webview wait in `pending`.
  A creation that fails is tried again, three times in all, a second
  then two apart, as Microsoft advises unless it failed with
  `ERROR_INVALID_STATE`: under load, WebView2 fails some with `ERROR_BUSY`
  or `CO_E_SERVER_EXEC_FAILURE`. When the webview never comes, the calls
  that report a result (Eval, CapturePage, PrintToPDF) get the reason, as
  later ones do: the window closed, or WebView2 could not create it.
  User data lives in `%LOCALAPPDATA%\<name>\WebView2`.
- **Custom schemes** load from `http://<scheme>.localhost/`, which WebView2
  lets the app answer through `WebResourceRequested`. Chromium treats
  `*.localhost` as a secure origin, and unlike `https`, `http` blocks no
  mixed content, so pages reach `ws://` and `http://` URLs as with the
  custom schemes of the other backends. The backend maps URLs both ways, so
  the core only sees `<scheme>://localhost/`.
  Responses are buffered: WebView2 takes a whole stream. `LoadHTML` with a
  base URL serves the document from that URL the same way.
- **Eval** goes through the DevTools protocol (`Runtime.evaluate` with
  `awaitPromise`), which reports syntax errors and ignores the page's CSP.
  Edit roles run `document.execCommand` with a user gesture; paste inserts
  the clipboard text.
- Scripts injected at document creation run in every frame, so the
  main-frame-only ones (the bridge) are wrapped in `window === window.top`.
  Iframes get no `chrome.webview` handler.
- **ABI.** On x64, structs over 8 bytes are passed by reference and Go
  mirrors the first integer arguments into the XMM registers, so doubles can
  be passed as bits. On ARM64, 16-byte structs travel in two registers and
  floats cannot be passed, so zoom falls back to CSS (`abi_*.go`). Go
  callbacks read only integer registers: the two methods of UI Automation
  that take doubles enter through thunks in assembly (`uia_*.s`), which
  move the doubles' bits to the integer registers of their positions,
  touching only registers a call may change, and jump to the callbacks.
- DPI: the process is per-monitor aware (v2); the backend converts between
  pixels and DIPs with the window's or monitor's DPI. Frameless windows, and
  those with a hidden title bar, drop the caption in `WM_NCCALCSIZE` but
  keep the side and bottom borders, which Windows 10+ draws invisibly
  outside the window, so they still resize. On Windows 11 a hidden title
  bar keeps one pixel of its top, under the window's border, without which
  the maximize button opens no snap layouts while the window is not
  maximized. Windows 10, which has no snap layouts, keeps none: over a
  window that keeps any of its top it draws its whole title bar. The build
  comes from `RtlGetVersion`, which no manifest changes.
- A hidden title bar (`titlebar.go`) gets its controls from two child
  windows above the webview's. The buttons are a layered window
  (`UpdateLayeredWindow`, premultiplied BGRA): Segoe Fluent Icons glyphs
  (Segoe MDL2 Assets on Windows 10) drawn white on black with GDI give
  their coverage, over backplates of Chromium's alphas. It answers
  `WM_NCHITTEST` with `HTMINBUTTON`, `HTMAXBUTTON` and `HTCLOSE`, passes
  `WM_NCMOUSEMOVE` to `DefWindowProc`, which opens Windows 11's snap
  layouts over maximize, and runs the buttons itself from the non-client
  button messages, sending the window `WM_SYSCOMMAND`. The top edge is a
  layered window without a bitmap (`WS_EX_NOREDIRECTIONBITMAP`), as tall as
  the sizing frame, that answers `HTTOP` and sends the window's presses on,
  so it resizes; Windows Terminal's caption works the same way. The
  webview's window is created later, so the controls go back to the top of
  the z-order when it appears. A window without a caption has no room for
  its menu bar: Alt and F10 open a popup holding the bar's own submenus.
- **Vibrancy** sets `DWMWA_SYSTEMBACKDROP_TYPE` (Windows 11 22H2) and
  extends the frame over the client area, behind a transparent webview or
  native UI.
  The material shows only in a window created without a redirection
  bitmap (`WS_EX_NOREDIRECTIONBITMAP`, which Windows neither adds nor
  removes later), whose opaque surface would cover it, so `SetVibrancy`
  shows one only in windows created with `Vibrancy`. Such a window shows
  nothing GDI draws: it has no menu bar, like a window without a caption,
  and the buttons of its hidden title bar, layered without a bitmap too,
  show their pixels through DirectComposition (`compositor.go`), on a
  Direct3D 11 device the windows share (`internal/gpu/d3d11/device`,
  which links none of the renderer). A device found invalid after a
  failed draw, or whose GPU device signals its removed event
  (`ID3D11Device4::RegisterDeviceRemovedEvent`, waited for by a
  goroutine), is made again, and every window's buttons draw again;
  buttons that could not draw retry on a timer of the application
  window, a second later and longer after each failure in a row. Without
  DirectComposition, a window with a hidden title bar or native UI keeps
  its bitmap: what it draws shows, and the material does not. Native UI
  there draws on the GPU into a swap chain for composition, premultiplied
  by its alpha (`d3d11.NewComposed`), on a DirectComposition device of the
  renderer's own, and its frames drawn in memory show as the window
  controls do. A renderer about to draw takes the surface's window over
  from those (`Surface.Native`): a window has one DirectComposition target
  at most. `ui.Context.Vibrancy` tells the view whether the material shows
  (`platform.MaterialSurface`).
- Message boxes are task dialogs (comctl32 v6, activated from shell32's
  manifest for executables without one); their structs are packed and laid
  out by hand. Notifications are notification-area balloons, which Windows
  10+ shows as toasts.

### Browsers (`internal/web`)

With `GOOS=js GOARCH=wasm`, `backend_js.go` picks `internal/web`, which
shows windows of native UI (`WindowOptions.Surface`) on a page; windows of
web pages, menus, trays, notifications, global shortcuts and file dialogs
report `errUnsupported` or do nothing, as the browser has nothing to stand
for them. Building for browsers adds nothing to desktop builds: the
backend, `load_js.go` and `canvas_js.go` are `js` only, and the contract
gained one field, `SurfaceNative.Canvas`.

- **Size.** What only web pages use would make most of a build for
  browsers, which servers send whole (Cloudflare's static files are at
  most 25 MiB): the root's files for pages leave it out. `ipc.go`,
  `channel.go`, `bridge.go` (what windows say to the bridge: its script,
  messages, `Eval`), `protocol.go` (custom schemes, the frontend's, their
  downloads), `typescript.go`, `updater.go` (and `relaunchNow`),
  `single_instance.go`, `window_state.go` and `random.go` have twins
  (`_js.go`) that keep their API and do nothing, or report that there is
  no page, without the packages they need: net/http, net, crypto,
  encoding/json, internal/tsgen with go/parser, internal/update. Only
  `Protocol.Handle`, `HandleFunc` and `FileServer`, which take net/http's
  types, are missing there. `Bind` binds nothing, so nothing calls
  `reflect.Value.Method`, which would make the linker keep every exported
  method of every type in an interface. `internal/platform` takes an
  `http.Header` as its type, `map[string][]string`, so that ui does not
  import net/http either (importing it alone adds 2.3 MB), and
  `gpu.SourceSum`, for shaders compiled ahead of time, is not built for
  browsers, with crypto/sha256. The root's tests and `internal/e2e` are
  for desktops. What stays is mostly the runtime, ui and its packages,
  and the image codecs (`ui.Image`): `website/docs-demo` comes to about
  9.8 MB (2.6 MB with gzip).

- **Threading.** Go's main goroutine, which calls `Run`, is the main
  thread, told by its goroutine ID as `internal/fake` does. JavaScript
  calls Go's functions (`js.FuncOf`) on goroutines of their own, while
  the browser's event loop waits: `Backend.sync` hands the event to the
  main goroutine through `tasks` and waits for it, so that a handler can
  still call `preventDefault`. A callback that a call from Go to
  JavaScript sets off (a `focus()` firing `focus`) runs on the calling
  goroutine, the main one, and so runs at once. `Step`, for nested loops
  (`await`, `PopupMenu`), takes tasks as `Run` does, so the browser's
  events go on meanwhile; a nested loop must not start in a task, whose
  callback holds the browser's event loop, but from `Dispatch`.
- **Windows** are elements of class `mygo-window` filling the element with
  the ID `mygo`, or else the page, each holding a canvas and a textarea.
  The page sets their size; `Bounds` reads the element's. The surface's
  size is the canvas's CSS size (DIPs) and `devicePixelRatio`, which a
  `ResizeObserver` and the window's `resize` watch; its backing store is
  set before each frame to the frame's size, `ceil(width × scale)` as
  package ui computes it. `RequestFrame` asks `requestAnimationFrame`;
  `Occluded` is `document.hidden`, where browsers give no frames, and
  `visibilitychange` brings `SurfaceShown`.
- **Drawing.** The canvas's WebGL 2 context (`gl.NewContext`: premultiplied
  alpha, no depth, stencil or antialiasing) is current while a frame is
  handled, as a GtkGLArea's is in its render signal, and
  `SurfaceNative.Canvas` names it for `ui/gpu_js.go`. WebGL names its
  objects by JavaScript objects and GL by numbers: `load_js.go` numbers
  them (the number also a property of the object, which reads
  `FRAMEBUFFER_BINDING` back) and implements the GL functions `gl.go`
  calls on the current context, copying Go's bytes into one reused
  `Uint8Array` for uploads, which the pixel store's unpacking (tracked
  from `glPixelStorei`) sizes. With `WEBGL_blend_func_extended` the
  shader writes the second color as on desktops; without it (one color
  blended) subpixel glyphs could not draw, which browsers' text never
  has. A lost context is kept (`preventDefault`) and, once restored,
  renumbered (`Context.Renew`), which makes the renderer start over.
  Where the browser has no WebGL 2, the canvas takes a 2D context and
  `PresentPixels` puts the frames drawn in memory into an `ImageData`.
- **Text** (`internal/text/canvas_js.go`) is the browser's: a 2D context
  measures and draws it, so that it finds fonts, falls back and shapes as
  for pages. Canvases give widths, not glyphs: the engine's glyphs are
  grapheme clusters (go-text's segmenter), each ending where the text of
  its word through it ends and starting its own width before, so that the
  word's kerning moves the cluster it applies to; they are drawn alone,
  with the font at the size they were measured at and the canvas scaled
  (at the pixels' size the system's font may take another optical size,
  as SF Pro Display for Text, whose glyphs the advances would not fit),
  their coverage read back (`getImageData` of a `willReadFrequently`
  context) and cropped to the ink. A line of text draws as the browser's
  `fillText` draws it whole, pixel for pixel. Fonts are shaded: a glyph is
  drawn in a gray of its text's shade, as Chrome's text, like AppKit's,
  is bolder the lighter its color (dark text drawn with white glyphs looks
  too bold), Thick in white; emoji take an unshaded twin of their font.
  Runs of scripts whose letters join or reorder (Arabic, the scripts of
  India, Thai, …) are one glyph each, shaped whole. Lines break at
  UAX #14's opportunities, and go-text's bidi package gives runs' levels,
  which the engine reorders (UAX #9's L2). Glyph IDs number the clusters'
  texts. Glyphs blend in turn (`join` false), at quarter-pixel positions,
  as Skia draws them. Chrome picks the optical size of the system font
  inconsistently between pages and canvases, so text of native UI can be
  wider or narrower than a page's in the same font.
- **Input.** Pointer events (captured on press) are the pointer, with
  clicks counted from presses close in time and place, buttons 3 and 4
  the back and forward keys; `wheel` events scroll, lines and pages
  turned into DIPs, notches of 120 not `Precise`. The textarea, out of
  sight at the caret (`SetTextInput`), has the keyboard: `keydown` and
  `keyup` are keys, by what they type (`key`) and else by where they are
  (`code`); `beforeinput` brings typed text, which it cancels so that the
  textarea stays empty; composition events are the input method's. The
  browser keeps its defaults for keys that type, its own keys (F5, F11,
  F12) and those copying and pasting: the content copies on the key and
  the `copy` event that follows carries its text (the backend's
  clipboard) to the system's, while the clipboard's text comes with
  `paste`, which then sends the paste command. On a Mac, Command is
  `ModCtrl` and Control `ModSuper`, as package ui takes browsers for
  another system than macOS.
- **Files dropped** (`files.go`): `dragover` on the canvas, with files,
  is `FileDragOver`, whose answer is the `dropEffect`, and the browser's
  default (opening the file, leaving the app) is always prevented; a
  `drop` reads the files' bytes (`arrayBuffer`, settled together; folders
  have none) before `FileDrop` gives their paths, `/mygo-drop/<n>/<name>`.
  Go's syscall package calls the `fs` object of `wasm_exec.js`, whose
  functions fail in browsers: the backend wraps `open`, `read`, `fstat`,
  `close`, `stat`, `lstat` and `readdir` to answer for those paths,
  synchronously, as an app reads a drop in its handler, while the browser
  waits on the callback that sent the drop, and passes other paths on.
- **Context menus** (`PopupMenu`) are HTML, styled as macOS's, in a nested
  loop until an item is chosen, a press lands elsewhere, Escape is pressed
  or the page loses focus; the choice goes to `MenuItemClicked`.
- **Themes**: `IsDark` and `Preferences` read `matchMedia`, whose changes
  call `ThemeChanged`; the root's `data-mygo-theme` (watched by a
  `MutationObserver`) chooses the appearance over the media's. `ShowMessageBox` uses `alert` and `confirm`.
- **History** (`ui/router_js.go`): the first `ui.Router` a window (not a
  `Tester`) shows claims the tab's history, through hooks the router calls
  (`browserHistory`, nil elsewhere). Its location goes in the URL's
  fragment and the number of its entry (`Router.base + at`, `base`
  counting the entries dropped past `maxHistory`) in `history.state`:
  `Push` is `pushState`, `Replace` `replaceState`, `Go` `history.go`,
  whose `popstate` finds the router there already. A `popstate` of the
  browser's own buttons goes to the entry of its number; one just before
  the router's first or after its last, as after a reload (which takes
  the number of the browser's entry for the router's), becomes the
  router's; a fragment typed in, without a state, is pushed.

## IPC

### The page runtime (`packages/bridge` → `internal/bridge/bridge.js`)

`packages/bridge/src/bridge.ts` is bundled by Bun into an IIFE
and committed, so building an app never needs Bun. The core wraps it with the
window's configuration (`bridge.Script`) and every backend injects it at
document start into the main frame. It installs:

- `window.mygo`: `call(method, ...args)`, `on(event, fn)`, `once`, the
  `window` controls (`minimize`, `toggleMaximize`, `close`, …), `platform`,
  `windowId`, `version`. Frozen, so pages cannot tamper with it.
- `window.__mygo.receive(messages)`, used by Go to deliver messages.
- dom-ready notification and `--app-region: drag` handling for frameless
  windows (the mousedown is reported, the backend starts a native window drag
  from the recorded mouse event).
- The `--mygo-titlebar-*` CSS variables of a window with a hidden title
  bar, in a constructed style sheet (a `<style>` element where engines
  lack them), from the `mygo:title-bar` event: the room its controls take
  (`Window.TitleBar` of the backend, in CSS pixels). Each backend delivers
  the event at document start, in a script after the bridge, so the first
  page lays out around the controls before it paints; the core sends it
  again when the room changes and once the DOM of any later page is ready.
- File drops (`Window.OnFileDrop`, `onFileDrop` in mygo-runtime). Unlike
  Tauri, whose native drop handler takes drops away from the page (on
  Windows, HTML5 drag and drop needs it turned off), MyGo leaves the page's
  drag and drop alone and reads the paths next to it. A capture listener on
  `drop` posts `{t:"drop", x, y}` when the drop carries files; the core then
  takes the paths the backend recorded just before the page saw the drop:
  `performDragOperation:` of the web view subclass (calling super) on
  macOS, `drag-data-received` then `drag-drop` (before WebKit's own
  handlers) on Linux, and on Windows the File objects the bridge posts with
  `postMessageWithAdditionalObjects` (`ICoreWebView2File.Path`), with
  WebView2's own drop handling left on. Drags that start in the page are
  ignored (`dragstart`/`dragend` tracking), and so are drops the page
  handles; where the page does not handle dragged files, bubbling
  `dragover`/`drop` listeners cancel them, so a drop reaches Go instead of
  the engine replacing the page with the file. Paths go to Go listeners and,
  as the `mygo:file-drop` event, to trusted pages only. Event names starting
  with `mygo:` are reserved.

- Find in page (`find.ts`, `Page.FindInPage`): the bridge walks the
  visible text nodes, marks matches with the CSS Custom Highlight API (a
  constructed style sheet, which a Content Security Policy allows) and
  scrolls to the active one; engines without the API get the active match
  selected instead. Being JavaScript, it works the same in every engine.

The transport is `window.webkit.messageHandlers.mygo.postMessage` on both
WebKit platforms and `window.chrome.webview.postMessage` on WebView2.

### The `mygo-runtime` package (`packages/runtime`)

Apps reach the injected runtime through the `mygo-runtime` npm package:
`call`, `on`/`once`, typed `event<T>(name)`, `currentWindow` controls,
`isMyGo`, `isCallError`, `runtime()` (which throws a helpful error outside a
MyGo window) and the public types (`Runtime`, `WindowControls`, `Platform`),
which the bridge shares. It holds no transport of its own: it delegates to
`window.mygo`, so the injected script stays the single implementation of the
protocol. The template depends on `^<version>` from npm, released in step
with the Go module.

<!-- repository-only:start -->

Its `dist/` is not committed. Run `bun run build` before using a checkout
with `mygo init --mygo <checkout>` and its `file:` dependency. CI and releases
build the package too.

<!-- repository-only:end -->

### The `mygo-cli` package (`packages/cli`)

The CLI is also published to npm, so that projects pin it in package.json
and run it from their scripts. Like esbuild, each platform's binary is a
package of its own, `@egoist/mygo-cli-<os>-<cpu>` in `packages/cli/npm` (npm
takes unscoped names such as `mygo-cli-win32-x64` for spam), with `os`
and `cpu` fields; `mygo-cli` lists them all as optional dependencies, so
package managers install only the matching one, and its `bin/mygo.js`
resolves that package and replaces itself with the binary
(`process.execve` in Node.js 23.11 and later and in Bun; elsewhere it spawns
the binary, waits and passes its exit status on). `MYGO_CLI_BINARY` points
it at another build.

<!-- repository-only:start -->

In this repository's checkout, platform packages hold no binary, so the
workspace examples build `cmd/mygo` from source.

`bun run --cwd packages/cli binaries [platform...]` cross-compiles the
binaries (ignored by git) and writes the manifests with the version of
`mygo.Version`; `bun scripts/publish.ts` publishes the packages (see
[Releasing](#releasing)). The binary is named `mygo`, like an unrelated npm
package: docs say `bunx mygo-cli`, never `bunx mygo`, outside a project.

<!-- repository-only:end -->

### Wire protocol

Page → Go (a JSON string per message, prefixed with the window's secret; see
Trust):

| message | meaning |
|---|---|
| `{"t":"call","id":N,"k":token,"m":"Service.Method","a":[...]}` | call a bound method |
| `{"t":"chan-ack","c":N,"k":token,"n":S}` | the page took the values of channel N up to the S-th |
| `{"t":"chan-close","c":N,"k":token}` | the page closed channel N |
| `{"t":"dom-ready"}` | DOMContentLoaded fired |
| `{"t":"drag"}` / `{"t":"dblclick"}` | mousedown / double click on a drag region |

Go → page, batched into one `__mygo.receive([...])` evaluation per
main-loop turn. JavaScriptCore runs a script of that shape, `a.b(JSON)`,
with its JSON parser instead of compiling it, which is several times
faster, unless the inspector is enabled (development builds):

| message | meaning |
|---|---|
| `{"t":"reply","id":N,"k":token,"ok":true,"v":value}` | successful call |
| `{"t":"reply","id":N,"k":token,"ok":false,"e":"message"}` | error or panic |
| `{"t":"chan","c":N,"k":token,"p":value}` | a value of channel N; `"a":1` asks for an acknowledgment |
| `{"t":"chan","c":N,"k":token,"end":true}` | Go closed channel N |
| `{"t":"event","n":"name","p":payload}` | typed event |

`k` is a random per-page token: a reply meant for a page that has since
navigated away can never resolve a promise of the new page.

### Calls

1. `WindowHandler.Message` runs on the main thread. Messages starting with
   `{"t":"call",` are handed to a goroutine (`handleCall`) together with the
   page context and the page's trust decision, captured on the main thread so
   a racing navigation cannot change them. Everything else is decoded on the
   main thread.
2. `handleCall` decodes the envelope with `encoding/json/v2`, looks up the
   method, decodes each argument into the parameter type, calls it and
   encodes the result. Panics are recovered, logged with a stack trace and
   returned as errors. Large values are copied as little as possible: the
   arguments are raw values that share the message (`rawValue`), decoded
   from it in place, and the result is encoded into a part of the reply
   (`message`), which the flush copies into the script once.
3. The reply is queued with `Window.enqueue(msg, false)` and flushed on the
   main thread.

Bound methods may take a `context.Context` first: it carries the calling
window (`CallerWindow`) and is canceled when the page navigates or the window
closes. `Bind` validates every parameter and result type up front
(`tsgen.Validate`), so unsupported types fail at startup, not at call time.

JSON options (`jsonOptions` in `ipc.go`) are shared by calls and events:
json/v2 defaults (nil slices encode as `[]`, case-sensitive names),
`time.Duration` as nanoseconds, and U+2028/2029 escaped for JavaScript.

### Events

`NewEvent[T](name)` registers a typed event. `Emit` and `Broadcast` encode
once and enqueue per window. **Events are held until the page reports
dom-ready** (bounded to 1024 per window) and the hold resets on every
navigation, so events sent right after creating a window, or during a
navigation, are delivered once listeners exist. Replies are never held:
module scripts may `await` a call at top level, and DOMContentLoaded waits for
them.

### Channels

A `*Channel[T]` parameter (`channel.go`) streams values to the page that
made the call. The page's `Channel` goes into the call's arguments as an
id it chose; `method.call` creates the Go side for that page, registered
in `Window.channels`, and closes it when the method returns, before the
reply is queued, so the page gets the values, the end, then the result.
The page may close it earlier, which cancels the call's context, a context
derived from the page's for calls with channels; so does the page going
away. The page may even close it before the call's goroutine made it
(aborting a request right after starting it): the window remembers such
closes (`closedEarly`, bounded, reset with the page) and the call closes
the channel as soon as it makes it. Values are queued in the window's outbox like replies, so they are
batched and stay in order with them.

Flow control keeps a producer faster than the page from piling up
messages: every half MiB of messages, a value asks for an acknowledgment,
which the page sends once it took that value (handled it, or yielded it to
an iterator), and `Send` waits while more than a MiB is unacknowledged. It
never waits on the main thread, which receives the acknowledgments.

### Eval

`Page.Eval(code)` uses the webview's native async-function API
(`callAsyncJavaScript` on macOS, `webkit_web_view_call_async_javascript_function`
on Linux), which awaits promises and ignores the page's Content Security
Policy. The code is first wrapped as `return JSON.stringify({ok, v: await (code)})`;
if that fails to compile the code is not an expression, and it is run again as
a function body. A compile error means nothing executed, so code never runs
twice. The result travels as a JSON string and is decoded in Go.

### Trust

Every page gets the runtime, but only trusted pages may call Go: the
frontend (`mygo:` and the `mygo dev` server), custom-scheme pages
(`Protocol.Handle`), `file:` and `about:` pages, loopback `http(s)` dev
servers in development, and origins listed in `PageOptions.TrustedOrigins`
(`"*"` trusts everything). The decision is
recomputed on every committed navigation. Untrusted calls are rejected
without running any Go code.

The message handler is reachable from every frame, while trust is decided
by the main frame's URL, so an iframe of another origin must not be able to
talk to Go: each window has a random secret (`crypto/rand`), passed only to
the bridge's closure in its configuration, which prefixes every message;
`handleMessage` drops messages without it. macOS additionally only accepts
messages from the main frame (`WKScriptMessage.frameInfo.isMainFrame`);
WebKitGTK exposes no frame information, so Linux relies on the secret.

### Plugins

`Use` binds a plugin's service as an internal service named
`plugin:<name>`: calls reach it like any bound method (trust checks,
contexts, channels), but `GenerateTypeScript` skips it, since the plugin's
own npm package is its client. A call to a plugin that is not used fails
with an error naming `mygo.Use`. The official plugins in `plugins/` keep
each Go package next to its npm package, built into `dist/` by `bun run
build` like mygo-runtime and released with the same version.

- **fetch** streams a response through a `Channel`: the head first (status,
  headers, final URL), then base64 chunks of the body as Go reads them. The
  JavaScript side builds a `Response` around a pull-based `ReadableStream`
  over the channel's iterator, so the page's reading paces Go through the
  channel's flow control. Aborting, canceling the body or the page going
  away closes the channel, which cancels the request's context. Request
  headers go through a plain `Headers`, which, unlike a `Request`'s, drops
  no forbidden names.
- **websocket** is an RFC 6455 client of its own (no dependency) on top of
  `net/http`, which keeps upgrade requests on HTTP/1.1 and hands the
  connection over as the body of the 101 response, so proxies and the
  client's TLS settings apply. `Connect` streams the connection's events
  through a channel for as long as it lasts; the page sends with `Send`
  calls numbered in order, since calls run on goroutines of their own and
  would otherwise race, and Go writes them in that order. Connections are
  keyed by window and a random id the page chooses.
- **updater** is the update window, in the manner of Sparkle, built on
  `mygo.Updater` alone. A *session* is a check and what follows it (the
  release notes, the download, the offer to relaunch): a goroutine that
  sets the session's *view* (title, message, progress, release notes,
  buttons) and waits for responses, whether or not the window shows, so
  that a background check shows it only when it has something to offer and
  a "Check for Updates…" during one just shows it. The window's page, one
  embedded HTML file loaded with `LoadHTML` (an `about:blank` page, so
  trusted), watches the views through a `Channel` (`Watch`) and answers
  with `Respond`; each set of buttons has a prompt number and only the
  first answer to the current prompt counts, so a double click cannot
  answer the next view. The plugin's service rejects calls from other
  windows. Release notes are Markdown parsed in Go (`internal/markdown`),
  which keeps all HTML as text and only links http(s) and mailto URLs; the
  page gets them rendered as escaped HTML, and a Content Security Policy
  with a nonce runs only the page's own script: the page may call Go, and
  the notes come from the unsigned manifest. Links open in the browser
  (`OnWillNavigate`). Views with release notes have a fixed size; status
  views ask for the height of their text (`Fit`). The window gets an empty
  menu of its own, so it has no menu bar on Linux and Windows.
  A window sees its session through `internal/frontend`: the session
  creates it (`NewWindow`: its size, its menu, the session canceled when
  it closes), and the window shows the views (`Current`), answers
  (`Respond`) and asks for a size (`Fit`). `plugins/updater/native` is
  the same window in native UI: `native.New` builds the plugin with a
  window of its own through a hook package updater sets
  (`frontend.Plugin`), so package updater links no native UI and has no
  API for other windows. Its view rebuilds when the session's view
  changes, measures the boxes of a layout in the next frame (`Bounds`) to
  ask for the size the page's script would, and builds the notes'
  paragraphs of inline text elements, their links of `ui.Link`. `updater.json` in `PathUserData` keeps the
  preferences, the skipped version and the time of the last check, and
  a change that alters them calls the functions of `OnChange` on the main
  thread, from a goroutine of its own so that no caller waits for the main
  thread; the next check is due an interval after it, or an hour after a failure, and
  is rescheduled on resume since timers stop while the computer sleeps. An
  update installed while the app runs is remembered, so that checks offer
  to relaunch instead of installing it again. The texts are `Strings` in
  the language that best matches `Options.Language` or `App.Locale`
  (`matchLanguage`: language, script, region; Chinese scripts inferred
  from regions such as TW; another variant of the language before
  English), among the plugin's translations (`translations.go`) and the
  app's, whose empty fields fall back to the plugin's, then English. The
  page gets `lang`, which picks CJK fonts, and `dir`; status texts use
  `unicode-bidi: plaintext` and the notes `dir="auto"`, as either may be
  in another language than the window. The page reports the width its
  buttons need too, as translations can be long.

- **sqlite** compiles SQLite's pinned C amalgamation with Zig 0.16 and
  loads it through purego. A C shim passes doubles as bits on every ABI,
  binds a per-operation atomic cancellation token to progress and busy
  handlers, and authorizes SQL without Go callbacks. Connections serialize
  operations; transactions hold one connection through BEGIN IMMEDIATE,
  COMMIT or rollback. The page client preserves int64 and BLOB values with
  tagged cells and confines database files to the Go-configured directory.
  Connections belong to their page and close on navigation or app quit.
  `mygo-plugin.json` names the six native-library assets, which the CLI
  bundles like libghostty-vt, with checksums for the library assets. Go and native UI apps can also open connections directly.

- **terminal** is a terminal for native UI: a `Terminal` runs a program in
  a pseudo-terminal and emulates it with libghostty-vt, Ghostty's terminal
  emulator, and `View` draws it with package ui and takes its input.
  - *The library* is loaded with purego (`internal/vt`): `dlopen`, or
    `LoadLibrary`, then a symbol table; functions taking structs or floats
    by value go through `purego.RegisterFunc`, the rest through `SyscallN`.
    The Go mirrors of the C structs are checked against the layouts the
    library describes in JSON (`ghostty_type_json`) as it loads, as are the
    enum values the package hardcodes, and packed cells (`GhosttyCell`) are
    read by the bit positions it reports, so that a library of another
    version fails to load. Its callbacks (writing to the pty, the title,
    the bell, the clipboard, synchronized output, device attributes, …) are
    made once and find their terminal by the userdata.
  - *Pseudo-terminals* (`internal/pty`) are opened in pure Go: `/dev/ptmx`
    with the ioctls of `grantpt`, `unlockpt` and `ptsname` (`TIOCPTYGNAME`
    on macOS, `TIOCGPTN` on Linux), the program in a session of its own
    with the slave as its controlling terminal; ConPTY on Windows
    (`CreatePseudoConsole`, a process attribute list), whose console the
    terminal closes once the program exits. The user's shell starts as a
    login shell (`-zsh`), when the first frame of a view sizes the screen
    (or after half a second without one), so that it starts at its size.
    A key the view takes tells Windows' backend to open no menu for the
    `WM_SYSCHAR` of Alt and the key.
  - *Threads.* A goroutine reads the program's output and writes it to the
    emulator under the terminal's lock, then asks the view for a frame;
    another writes what is typed, from a queue, so that a program that does
    not read cannot block the reader, which answers its queries. Callbacks
    of the app run after the lock is released. A frame takes the lock only
    to copy the emulator's state into a render state (`render_state`'s
    first phase); a second lock guards the render state, which a program's
    synchronized update (mode 2026) holds: the render hold callback copies
    the screen as the update starts, and frames show that copy until it
    ends, or for a second at most. Locks are taken in that order.
  - *Drawing.* The view sizes the grid in device pixels from the font
    (cells as wide as the widest ASCII glyph, as tall as the line) and
    rebuilds only the rows the render state marks dirty: runs of cells of
    one font and color, shaped with `ui.Shape` (through an LRU of shaped
    text, as output repeats) and placed in their cells, the glyphs of a
    cell centered there and shrunk when too wide, as emoji are; backgrounds
    and lines merge across cells. Box drawing, blocks and Powerline's
    separators are drawn, as in Ghostty, so that lines join across cells.
    Frames paint every row from these caches, the cursor (its cell's text
    again in the cursor's text color under a block), the selection (its
    text clipped to its cells and drawn again in the selected text's color
    when the theme has one), an input method's composition and a scroll
    bar.
  - *Themes.* `ghostty_themes.go` holds the themes the Ghostty it binds
    ships (iTerm2-Color-Schemes' archive that its `build.zig.zon` names),
    as 22 colors each in the generated theme data.
    `GhosttyTheme` looks for a theme file of the user's first, in
    Ghostty's themes directory, as Ghostty does.
  - *Input* comes as it happens (`ui.Element.HandleInput`): keys are
    encoded by libghostty-vt's key encoder, set from the terminal's modes
    (legacy, modifyOtherKeys, the Kitty keyboard protocol). A key that
    types text waits for the text, which macOS and Windows send after it,
    to encode both; text that comes alone, as on Linux or from input
    methods, finds its key from US layout. Mouse reports go through the
    mouse encoder while the program asks for them; otherwise a selection
    gesture of libghostty-vt turns presses and drags into selections, with
    the clicks it counts itself.
  - *Shipping the library.* `mygo-plugin.json` names its build for each
    platform, published as release assets with their SHA-256 checksums.
    The CLI puts them into apps
    (see the CLI's resources); other programs download theirs into the
    user's cache once, which packaged apps never do.
- **glass** is Liquid Glass for native UI, as macOS 26 and later draw it:
  `glass.Glass`, a `ui.Material` (interactive glass builds with its
  element, following its press with `Animate`, and paints itself grown
  while pressed, as macOS 27's: by a fixed 1.1 DIPs left and right and
  0.45 above and below, measured from `NSGlassEffectView`), paints an
  effect (`glass.Effect`, see Effects under Native UI), whose parameters
  are a pane's material in device pixels. Its shader (`glass.metal`,
  `glass.hlsl`, `glass.glsl`, and `pixels.go` for the CPU) takes what is
  behind the pane, blurred, sampled where the curved surface refracts it:
  displaced toward the middle within the bezel by Snell's law through
  Apple's squircle profile (`lens`, along `normal`), which mirrors what is
  just inside the rim; then maps its lightness and colors (`tone`; the CPU
  keeps its curve in a table, as its power is slow there), tints it, and
  lights the rim by how its normal faces the light. The defaults follow
  macOS 27's, measured from
  `NSGlassEffectView` over test patterns; the optics follow the
  open-source reproductions of Liquid Glass. `glass.Blur`, a backdrop
  blur, is a second effect (`blurEffect`): what is behind the element,
  blurred, with its alpha (the effect samples the backdrop's texels
  itself, as the heads' `sampleBackdrop` gives no alpha), so over nothing
  it stays transparent. Masked by a `LinearGradient`, the blur varies as
  the gradient's alpha (`blurLevels`): the element paints levels in turn,
  each blurring what the levels before painted (blurs add as their
  variances do), shown where the blur wanted is above its band, by how far
  into the band, so a level's top doubles from at most 2 pixels to the
  blur's most and a pixel between two levels mixes their blurs. A level of
  a square element covers only where it shows (`blurWanted`), and so
  reads less. Bounding the levels after the first to the element, so that
  they read no unblurred content around it, measured worse against a blur
  varying per pixel than leaving them reading it: the repeated edge biases
  more than what is around. A level may tone what it shows (`blurTone`:
  saturation, an offset, an opaque color over it), which the hard style of
  `glass.ScrollEdge`, macOS's scroll edge effect, frosts with in one
  level; the soft style is a gradient of the background, as macOS 27's
  replays its window's background under a mask, with no blur (measured
  from SwiftUI's `safeAreaBar` over test patterns, its layers dumped).
  Both effects' shaders are compiled ahead of time on macOS
  (`shaders_darwin.go`) and on Windows (`shaders_windows.go`), with
  `internal/gen`.

## Typed client generation (`internal/tsgen`)

`mygo generate` builds the app and runs it with `MYGO_GENERATE=<file>`;
`App.Run` then writes the client (`WriteTypeScript`) and returns before
touching the GUI. The generator combines:

- **Reflection** for correctness: types follow json/v2 encoding (embedded
  structs are inlined with v2's conflict rules, `omitempty`/`omitzero` become
  optional fields, pointers become `T | null`, `[]byte` is a base64 string,
  `time.Time` a string, types with JSON methods `unknown`, text marshalers
  `string`, generic instantiations get names like `PageTask`).
- **Source code** for readability, when available: the entry PC of each bound
  method (`runtime.FuncForPC`) locates its file, which is parsed with
  `go/parser` to recover parameter names and doc comments (JSDoc). Named
  string/number types with constants become union types (`"all" | "done"`),
  including simple `iota` sequences. Packages not reached through a PC are
  located with `go list`. Without sources (e.g. `-trimpath` binaries) the
  client is still correct, just with `arg0` names and no docs.

The output imports `call` and `event` from `mygo-runtime` and declares the
interfaces, one object per service with camelCased methods, and an `events`
object. `WriteTypeScript` only rewrites the file when its
content changes, so dev servers don't reload needlessly.

## Custom protocols

`Protocol.Handle(scheme, http.Handler)` lets pages load `<scheme>://localhost/…`
like a web origin (fetch, ES modules, relative URLs). `Protocol.serve` runs the
handler on a goroutine with a `schemeWriter`: headers and body are buffered
and handed to the main thread in 256 KiB chunks (and on `Flush`), the content
type is sniffed when missing, panics become 500 responses. Backends that
implement `platform.SchemeBodyWriter` take the body on the handler's
goroutine instead, where they may block it, and the chunk buffer is reused.
Backends turn responses into native ones:

- macOS: `WKURLSchemeHandler`; `didReceiveResponse:`/`didReceiveData:`/
  `didFinish`. A task stopped by WebKit cancels the request context and later
  writes are ignored (touching a stopped task raises an Objective-C exception).
- Linux: WebKitGTK wants a `GInputStream`, so the response body is streamed
  through a pipe (`g_unix_input_stream_new`), which WebKit reads on the main
  loop, 8 KiB at a time. The handler's goroutine writes the body into the
  pipe (`WriteBody`), waiting while it is full, so the main loop never
  blocks and a response WebKit reads slowly does not pile up in memory.
  Custom schemes are registered as secure and CORS-enabled.

Schemes are registered per webview at creation time, so call
`Protocol.Handle` before creating windows. `FileServer(fsys)` serves an
`fs.FS` with an index.html fallback for client-side routing.

## The frontend (`frontend.go`)

Apps load their web UI with URLs without a scheme (`WindowOptions.URL: "/"`,
`LoadURL("/settings")`), which `resolveURL` resolves against the frontend,
as in Tauri:

- during `mygo dev`, the dev server: `MYGO_DEV_URL`, from `devUrl` of the
  configuration (only honored when `IsDev`);
- otherwise `mygo://localhost/`. The `mygo` scheme is registered with every
  webview and, unless the app handles it with `Protocol.Handle`, serves the
  files given to `SetFrontend`. `mygo build` calls `SetFrontend` from a
  generated file that embeds `frontendDist` (see CLI), so app code has no
  `//go:embed` and development builds need no built frontend. During
  `mygo dev` without a dev server it serves `MYGO_FRONTEND_DIST` from disk,
  and with nothing to serve, a page explaining how to get a frontend.

## Windows, lifecycle and quitting

- `NewWindow` validates options, waits for readiness when called off the main
  thread, builds `platform.WindowOptions` (all defaults applied, bridge and
  preload scripts, registered schemes) and registers the window.
- A window's page API is on `Page`, a handle on the window that
  `Window.Page` returns unless the window shows native UI; the page's state
  stays in `Window`, where IPC and the backend's callbacks reach it.
- The user's close (`WindowHandler.ShouldClose`) and `Window.Close` both emit
  `OnClose`, which can be prevented. `Destroy` skips it. The backend reports
  `Closed` synchronously; the core unregisters the window, closes child
  windows, cancels the page context and, when it was the last window and no
  quit is in progress, runs `OnWindowAllClosed` listeners or quits.
- The quit sequence (`Application.prepareQuit`) is: `OnBeforeQuit` → close
  every window (any `OnClose` can cancel) → `OnWillQuit` → stop the loop →
  `App.Run` returns → `OnQuit`. It is used for `App.Quit`, Cmd+Q, quit
  Apple Events, and SIGINT/SIGTERM alike (`signal_unix.go`; a second signal
  exits immediately).
- Taskbar state on Windows (`taskbar.go`) lives on the window and is
  applied when Explorer sends `TaskbarButtonCreated` (first show, Explorer
  restarts), so progress and a hidden button set before showing stick.
- `WindowOptions.StateKey` (`window_state.go`) remembers a window's normal
  bounds and maximized/full screen state in `window-state.json` in
  `PathUserData`, like Tauri's window-state plugin, or in the app's
  `WindowStateStore` (`App.SetWindowStateStore`), which loads all the
  states once and saves them all, on the main thread. The state is captured
  300 ms after the last move/resize/state event (transitions resize the
  window on the way, and the normal bounds only change in the normal
  state) and when the window closes; it is saved when a window closes
  and when the app quits, if it changed, and kept for the next save when
  saving fails. A saved window that would not show on any
  display keeps its size and is centered. `WindowOptions.Maximized` starts
  a window maximized: `zoom:` on macOS, `gtk_window_maximize` before
  mapping on Linux, `SW_SHOWMAXIMIZED` on the first show on Windows, where
  `Maximize` on a hidden window also waits for it to be shown.
- Deep links (`deeplink.go`). `mygo build` and `mygo dev` link the name,
  version, identifier and `urlSchemes` of the configuration into the binary
  (`-X …packageName=…`), which is how Linux builds know them (`IsPackaged`,
  `Name`, `Version`) and every platform knows which launch arguments are
  deep links. URLs of those schemes, and of ones registered with
  `RegisterURLScheme`, reach `OnOpenURL`: from Apple Events on macOS, from
  the launch arguments once the app is ready, and from the arguments a
  second instance forwards. `RegisterURLScheme` writes
  `HKCU\Software\Classes\<scheme>` on Windows and, on Linux, a hidden
  `<id>.url-handler.desktop` in `$XDG_DATA_HOME/applications` made the
  default in `$XDG_CONFIG_HOME/mimeapps.list` (what `xdg-mime default`
  does); on macOS it calls `LSSetDefaultHandlerForURLScheme` for a scheme
  the Info.plist declares. The Linux `.desktop` of `mygo build` also
  declares the schemes (`Exec=… %u`, `MimeType=x-scheme-handler/…`).
- Starting at login (`login.go`): `SetOpenAtLogin` registers the bundle
  with `SMAppService.mainAppService` on macOS 13+ (a launch agent running
  `open -a` on macOS 12; both need a bundle), writes an XDG autostart entry
  on Linux, and a value of `HKCU\…\CurrentVersion\Run` on Windows, where
  `OpenAtLogin` also honors Task Manager's `StartupApproved` (odd first byte:
  disabled). The Linux, Windows and launch agent commands pass
  `--mygo-opened-at-login`, which the package removes from `os.Args` at
  init and `WasOpenedAtLogin` reports; macOS login items are recognized by
  `keyAELaunchedAsLogInItem` in the launch Apple Event instead.
- `Relaunch` runs the quit sequence, then (after `OnQuit`, which releases
  the single instance lock) starts the executable again with the same
  arguments and working directory. A development build exits with code 75
  instead, and `mygo dev` starts the same build again, also when that
  happens before it was ready.
- Updates (`updater.go`, `internal/update`, `cmd/mygo/updates.go`), in pure
  Go on every platform. `mygo keygen` creates an Ed25519 key pair; with
  `updates` in the configuration (the public key, and a GitHub repository or a base
  URL) `mygo build` links the manifest URL of the target and the public key
  into the app, and when the private key is available
  (`MYGO_UPDATER_PRIVATE_KEY` or `updates.privateKey`) archives the app as
  installed (the bundle, else everything next to the executable) into
  `<name>-<version>-<target>.tar.gz` (Linux builds always get it, for
  `install.sh`), signs its SHA-256, and writes
  `update-<target>.json` with the `## <version>` section of CHANGELOG.md as
  notes. `Updater.Check` fetches the manifest (HTTPS only, loopback HTTP for
  tests) and compares versions semantically; `Update.Install` streams the
  archive next to the app, verifies size and signature, unpacks it (files,
  directories and relative links only) and swaps it in: the bundle is
  renamed on macOS, the entries of the app directory on Linux and Windows,
  where the running executable is renamed away and removed at the next
  launch. `App.Relaunch` then starts the new version from the path the app
  started from. Development builds are never updated. Delta updates work
  as Sparkle's (`internal/update/delta.go`): `mygo build` fetches the
  published manifest of the target, downloads the archives of up to
  `updates.deltas` versions (that manifest's, and those it lists as
  `previous`, which the new manifest passes on), checks their signatures
  and writes a signed delta from each. Its index lists the tree of the new
  app, inside the bundle on macOS, with each file's size and SHA-256 and
  how to make it: a copy of a file of the old app with the same content
  (at its path, or elsewhere for moved files), else a patch of the file at
  its path or its own bytes, whichever is smaller. Patches are bsdiff's
  (`bsdiff.go`, with the qsufsort suffix array of `suffix.go`), their
  three streams compressed apart so that they are read from the delta
  file in place. `Update.Install` takes the delta whose `from` is the
  running version, makes the new app next to the old one through
  `os.Root`s, and fails on any file that does not match, then downloads
  the archive; the delta also names the versions it updates and makes.
  The tree made is the signed one byte for byte, so bundles keep their
  code signature. `mygo build -upload` publishes to `updates.github` with
  the `gh` CLI: it creates the release `<tagPrefix><version>` as a draft
  with the changelog section as notes, uploads installers, update
  archives and deltas, then the manifests, and leaves
  publishing the draft (which makes the manifests "latest") to the
  developer once every platform is there. With a `tagPrefix` other than
  `v` the repository may hold other releases, so apps are built with a
  feed naming the manifest in the release `<tagPrefix>{version}`
  (`update.TaggedFeed`), which `update.ResolveFeed` turns into the manifest
  of the newest published, non-prerelease release with that prefix, from
  the GitHub API's list of releases, newest first. `Updater.Check` and the
  delta step resolve it in Go, and `install.sh` in sh (`latest_tag`, which
  reads the `tag_name`, `draft` and `prerelease` fields in order). Such
  drafts are made with `--latest=false`, and published the same way, so
  the repository's latest release stays the others'. With `updates.s3` it puts the
  same files, manifests last, into a bucket of S3 or a compatible service
  (`cmd/mygo/s3.go`): one `PUT` per object, signed with AWS Signature
  Version 4 in pure Go (`crypto/hmac`), its SHA-256 payload hash checked
  by the service; the bucket goes in the host name on AWS, and in the path
  with a custom `endpoint` or a bucket name with dots, unless `pathStyle`
  says otherwise.
- File associations (`fileAssociations` in the configuration) are declared by the
  packages (see the table below) and their extensions linked into the
  binary; like deep links, files of those extensions among the launch
  arguments and those a second instance forwards reach `OnOpenFile`
  (paths relative to the working directory, and file URLs from `%U`).
- Downloads (`downloads.go`): links with `download`, and responses a page
  cannot show or that are attachments, become downloads. `OnWillDownload`
  gets the URL, the suggested name and a default path in Downloads (a
  unique name), which it may change or cancel; `OnDownloadDone` reports
  the result. WebKit cannot turn what a custom scheme handler serves into a
  download, and WebView2 would fetch it over the network, so those are
  canceled and the core serves the URL again through the scheme's handler
  into the file (`SchemeDownload`).
- Permissions (`permissions.go`): the backends route camera, microphone,
  location and notification requests to `WindowHandler.PermissionRequested`;
  `SetPermissionHandler` decides them, and by default trusted pages (the
  app's own, `TrustedOrigins`) are granted and others denied. Custom scheme
  pages are secure contexts with a real origin on every platform, which
  these APIs need. `macos.infoPlist` adds keys such as the camera and
  microphone usage descriptions macOS requires.
- Flyouts (`flyout.go`, `internal/platform/flyout.go`): `NewFlyout`
  creates a window with `platform.WindowOptions.Flyout`, which backends
  make a borderless, transparent popup owned by its parent, and place
  with `Window.PlaceFlyout` against an anchor in the parent's content
  area. `Flyout.Place` resolves the placement in the work area of a
  display as GDK does for `gdk_window_move_to_rect` (`maybe_flip_position`,
  then slide, then resize, per axis), and `WorkAreaFor` picks the display
  as GDK's `get_monitor_for_rect`; macOS, X11 and Windows (in physical
  pixels) use it, Wayland hands the anchor to the compositor's
  positioner. The core places a window's flyouts again as it moves or
  resizes (owned windows on Windows and X11 popups do not follow), and
  as `SetAnchor` and `SetSize` change them. A focusable flyout closes
  through `WindowHandler.ShouldClose` when it loses the keyboard: as its
  panel resigns key (macOS), on a press outside it or a broken grab, or
  as the compositor dismisses the popup (GTK, as `GtkMenu`), and as
  another window is activated (Windows), whose activating click its
  owner eats (`MA_ACTIVATEANDEAT`), as menus' and WinUI's flyouts'. On
  macOS a local monitor eats a press on a panel's anchor, which only
  closes it, as `NSPopover` eats one on its positioning rect: the press
  would make the parent key, which closes the flyout before the release
  clicks the anchor and opens it again. An
  Escape the content leaves closes a flyout too: native UI reports one
  no shortcut took (`routeKeys`, `surface.Conn.Escape`), and the bridge
  one no handler of the page cancelled (an `escape` message). `Flyout.Popover` shows one in an
  `NSPopover` (`internal/darwin/popover.go`), whose window AppKit makes
  as it shows, and whose transient behavior asks `popoverShouldClose:`.
  `Flyout.Tray` anchors one to a tray icon's `Bounds` instead, without a
  parent: the panel at `NSPopUpMenuWindowLevel` or an `NSPopover` shown
  from the status item's button (macOS), a topmost unowned popup
  (Windows), and on Linux, whose AppIndicator tells no bounds, an
  undecorated utility toplevel in the middle of the work area, which a
  focusable one closes as it loses the keyboard. A press on the icon
  does not dismiss its flyout (`tray.pressed` in `popoverShouldClose:`
  and on resigning key, the cursor over `Shell_NotifyIconGetRect` on
  Windows), so its click toggles it. A global monitor of mouse downs
  closes macOS's focusable panels on clicks outside the app, which a
  panel taking the keyboard of an app that was not activated never
  hears as the key window changing. The design follows GPUI's
  `WindowKind::AnchoredPopup` and winit's `WindowType::Popup`, which
  model xdg_positioner too.
- `window.open()` and `target=_blank` go through `SetWindowOpenHandler`. By
  default http(s) URLs open in the default browser. Allowing one creates a
  window around the configuration or related view WebKit provides, with its
  own content manager so scripts and messages never leak between windows.

## Menus

`Menu`/`MenuItem` are plain Go values built from templates. Roles expand into
labels, accelerators and submenus per platform (`roleDefaults`,
`roleSubmenu`); macOS-only roles are hidden elsewhere. Items get a process
unique id and are registered with weak pointers, so discarded context menus
can be collected. The core sends immutable snapshots (`platform.Menu`) to the
backend, which:

- builds native menus and tracks native items per owner (app menu, window menu
  bar, tray, popup) so `UpdateMenuItem` can change label/state in place and
  rebuilt menus release their items,
- performs edit roles natively (first responder on macOS,
  `webkit_web_view_execute_editing_command` on Linux), or sends them to a
  window showing native UI as a `SurfaceCommand`, and reports everything
  else through `WindowHandler.MenuItemClicked` for a window's menu, or
  `AppHandler.MenuItemClicked` for menus without one; the core toggles
  checkbox/radio state, performs window and view roles and calls `Click`
  with the menu's window, even while a submenu has focus instead.

macOS gets a default menu bar (App, File, Edit, View, Window), which is what
makes Cmd+C/V/Q work; other platforms get none unless the app sets one.

## Clipboard and drag data (`transfer`)

`transfer.Data` is the immutable serialized model shared by clipboard writes
and native drags: ordered items with alternative byte representations, MIME
formats, file/URI lists, and lazy providers. Each clipboard write or drag
starts an independent provider cache; discovery reads no bytes, and success,
errors and recovered provider panics are cached once. Typed `Drag(value)`
remains process-local, resolved by the core's temporary drag registry; its
token is rejected by clipboard writes.

`clipboard.go` owns validation, main-thread dispatch, provider reentry guards,
and deferred, exactly-once release hooks. `platform.Clipboard` accepts a
snapshot, returns materialized reads, discovers formats without rendering,
and flushes/closes native ownership. Clipboard providers belong to the app,
not a source window. Quit flushes them before stopping the loop, and finish
releases remaining providers. Linux requires a clipboard manager for data
to remain after exit; a failed provider prevents flushing its offer.

macOS owns NSPasteboardItemDataProvider objects until AppKit finishes them,
with eager NSPasteboardItems after flush. GTK uses application-owned selection
get/clear callbacks and owner-change generation checks; clipboard-only apps
bind selection primitives without requiring a surface. Windows shares the
drag adapter's IDataObject, vtables, native conversions and reference-counted
objects with OleSetClipboard/OleFlushClipboard; foreign reads hold a Win32
clipboard lock. Callback signatures are allocated once and routed by native
object or user data. Native reads copy bytes so application data outlives
native handles. See [Clipboard and drag data](data-transfer.md) for API and
format mappings.

## Audio (`audio`)

Package `audio` is independent of windows and the main thread: it neither
goes through `internal/platform` nor needs the app running. Players are
voices (`audio/mixer.go`) under one mutex. Each reads frames from a source,
an in-memory sound or a stream that a goroutine of its own decodes about a
second ahead into a ring (`audio/source.go`); the mixer never waits on a
stream, playing silence for one that is late. Voices resample to 48 kHz
stereo with a Catmull-Rom spline at their rate times their speed, and ramp
their gains (a 2×2 matrix of volume and pan) over 5 ms on every change,
pause and resume. Looping happens in the source, so a loop has no gap; the
voice counts frames past it modulo the length for `Position`.

The output (`audio/output.go`) is a goroutine locked to one thread that
opens the device when a voice starts, mixes 10 to 21 ms at a time and
writes it with a call that blocks while the device holds enough, and
closes the device after five idle seconds. Each platform gives one
`openDevice`:

- macOS: an Audio Queue of four buffers of 512 frames, started once all
  hold audio; its callback, created once, hands buffers back over a channel.
- Linux: `pa_simple_write` of PulseAudio's simple API (`libpulse-simple`,
  which PipeWire serves too) with a 40 ms target, else ALSA's
  `snd_pcm_writei` on `default`, both through purego.
- Windows: WASAPI in shared mode, event driven, with a 40 ms buffer,
  `AUTOCONVERTPCM` and the default-quality resampler; it compares the
  default endpoint's ID every 500 ms and fails the write when it changed,
  which reopens the device.
- Browsers: an AudioWorklet from a Blob URL that queues three chunks and
  posts a request whenever it used one; the output waits for a request
  before mixing the next. One AudioContext serves the page; closing
  suspends it, and the first pointer or key input resumes it.

Where `openDevice` fails, a null device takes the frames on the wall
clock and fails after two seconds, so the output tries the device again;
`audio.Err` reports the failure. Formats register a magic prefix and a
constructor as `image` does: WAV is built in, `audio/vorbis` and
`audio/mp3` wrap pure Go decoders.

## Native UI (`ui`)

The public API keeps one stable `*Context` per window. Child builders share
it and temporarily change its parent. `Element` is a 16-byte checked value:
a direct owner record, arena slot and 32-bit generation. The owner detaches
from the engine on close and retires before generation wrap. Old elements
cannot alias recycled nodes. Development builds and `Tester` diagnose stale
use; production methods return empty results or ignore it. `Handle` stores
persistent control identity separately for each window. Focus queries read
identity without depending on construction order; focus requests wait for a
hidden control. `Services` offers persistent clipboard, URL and redraw access.

Constructors build and style controls eagerly. `Context.Key` supplies an ID
before state initialization. `Changed` and `Submitted` apply pending input
to controls built so far, once per pass, before returning the response. This
lets polling commit a local bound value immediately. Input options must be
configured before response queries. Public click and shortcut queries also
apply pending bound input before returning, so inline actions see the latest
edits. `ComboboxParts.Chosen` applies input and reports a choice in the same
pass; it is not carried into a later rebuild. Private widget queries do not
finalize input during construction, before fluent configuration. Remaining
input applies after construction. `OnChange` and `OnSubmit` run after construction and bound
input, before rebuilding. Notices are cleared before another pass so an
edit cannot be reported again on a fresh local binding. Callback actions
consume input once and rebuild before paint.
The private render tree uses `node` and `context`; `internal/uigen` generates
the checked public facade. `FocusBind` binds desired focus to app data; `FocusedValue` reads actual focus independently of a hidden
control's pending request.

See the [migration guide](ui/migration.md) for the breaking element API,
`mygo migrate-ui`, and the Go type-aware lifetime checks in `mygo vet`.

A window with `WindowOptions.Content` shows a user interface MyGo draws
itself instead of a web page. The layers stay as everywhere else: the
toolkit (`ui`) is plain Go above the platform contract, a backend only
provides a surface to draw on and its input, and renderers know nothing of
either.

```
 view (Go) ──► ui: build, layout, paint ──► internal/scene ──► internal/gpu: d3d11 | metal | gl, or internal/raster
                 ▲       └─ internal/text: DirectWrite | Core Text | Pango | canvas, glyph and mask atlases
                 │
 internal/surface.Conn ◄── content.go (package mygo) ◄── platform.Surface: darwin | linux | windows | web | fake
```

- **The surface.** With `platform.WindowOptions.Surface`, a backend creates
  a view MyGo draws in place of the webview: a layer-backed NSView on
  macOS, a GtkGLArea on Linux (a GtkDrawingArea where OpenGL would not
  run on a GPU), a child window of class `MyGoSurface` on Windows.
  `platform.Surface` gives its native handles (for a swap chain, a layer,
  or the GtkGLArea while its `render` signal draws a frame, with its
  context current), size and scale, and the refresh rate of the display
  showing it (`RefreshRate`: the screen's `maximumFramesPerSecond`, GDK's
  `gdk_monitor_get_refresh_rate`, the display mode's frequency from
  `EnumDisplaySettingsW`); asks for a frame (`RequestFrame`: a
  paused `CADisplayLink`, whose ticks draw frames that follow each other
  themselves while a frame after a pause, or after a resize, draws when
  AppKit displays the view, before macOS 14 an `NSTimer` at the display's
  rate, `gtk_widget_queue_draw` or a tick callback of GTK's frame clock,
  `InvalidateRect`); presents
  pixels drawn on the CPU (`PresentPixels`: a CGImage as the layer's
  contents, cairo in the `draw` signal, `SetDIBitsToDevice` in `WM_PAINT`;
  on Linux `PresentDamage` too, with what changed);
  sets the cursor; and turns the input method on and off at the caret
  (NSTextInputClient, GtkIMContext, IMM32), with the text around it, up to
  512 runes on each side (`SetTextInput`). Everything else comes through
  `WindowHandler.SurfaceEvent`, in DIPs: frames, resizes, the pointer, the
  wheel, keys, text, compositions, focus, the edit roles of menus
  (`SurfaceCommand`), files dragged and dropped, and assistive technology.
  The back and forward buttons of a mouse come as the keys `KeyBack` and
  `KeyForward`, as do the keys of keyboards that have them: buttons 3 and 4
  of `otherMouseDown:` on macOS, buttons 8 and 9 and `XF86Back` and
  `XF86Forward` on Linux, and `WM_XBUTTONUP` (taken, so that
  DefWindowProc sends no `WM_APPCOMMAND` as well), `VK_BROWSER_BACK` and
  `VK_BROWSER_FORWARD`, and `WM_APPCOMMAND`'s browser commands on Windows.
  A key typed while the input method composes is the input method's
  alone, as Enter choosing a candidate: on macOS `keyDown:` sends no
  `KeyPressed` while there is marked text, as GTK's input method filters
  such keys on Linux and IMM32 makes them `VK_PROCESSKEY` on Windows.
  Input methods name what text and compositions replace in the text
  around the caret (`Replace`, `From`, `To`): NSTextInputClient's
  replacement ranges, GtkIMContext's `delete-surrounding`, IMM32's
  reconversion (`IMR_CONFIRMRECONVERTSTRING`). Files dragged over the
  surface (NSDraggingDestination, a GTK drag destination, an OLE
  `IDropTarget`) are `FileDragOver` events, whose answer the drag source
  shows, and a drop a `FileDrop`; files the content does not take go to
  `OnFileDrop`.
- **Assistive technology.** Once it asks for a surface's content, the
  backend sends `AccessibilityOn`, and the engine describes every frame
  (`ui/access.go`) as a `platform.AccessTree`: nodes in pre-order with
  roles, names, values, ranges, states, bounds and actions, under
  `UpdateAccessibility`. Backends keep a native object per node ID and
  tell the system what changed between trees: subclasses of
  `NSAccessibilityElement` and AppKit's notifications on macOS; on Linux,
  ATK objects of GObject types registered through purego below the
  accessible of a `GtkGLArea` or `GtkDrawingArea` subclass, with ATK's
  signals, which GTK bridges to AT-SPI; on Windows, UI Automation
  fragments, COM objects with control patterns, answering `WM_GETOBJECT`,
  with UI Automation's events.
  What assistive technology does comes back as `AccessAction` events,
  which the engine performs as the pointer or the keyboard would.
  `AccessTree.Announcements` are texts to read out once
  (`Context.Announce`, toasts, routers): `AXAnnouncementRequested` posted
  on the application with a medium priority on macOS, AtkObject's
  `announcement` signal (ATK 2.46), which AT-SPI carries as
  `Object:Announcement`, on Linux, and on Windows a polite live region at
  the end of the tree holding the text, raising `LiveRegionChanged`, as
  Flutter's alerts: `UiaRaiseNotificationEvent` returned `S_OK` but its
  events reached no client, unlike the provider's automation events.
  Statuses, as toasts, are polite live regions too (`LiveSetting`).
  Widgets of an app's own set what the bases set with element methods
  (`Checked`, `Mixed`, `Expanded`, `Value`, `Range`, `Level`,
  `ActiveDescendant`). Menus drawn in the window are AppKit's `AXMenu`,
  `AXMenuBar` and `AXMenuItem`, whose choice shows as
  `AXMenuItemMarkChar` (answered through `accessibilityAttributeValue:`,
  as `AXInvalid` is), ATK's menu, menu bar, menu item, check menu item and
  radio menu item, and UI Automation's Menu, MenuBar and MenuItem, with
  Toggle for the items showing a choice; headings are WebKit's
  `AXHeading`, whose value is their level, ATK's heading with the `level`
  attribute, and UI Automation's Text with `HeadingLevel`. A vertical
  slider says so (`AccessVertical`: `AXOrientation`, ATK's vertical state,
  UI Automation's Orientation). A range's step, how far the keys move
  its value (`AccessNode.Step`), is UI Automation's SmallChange (and
  LargeChange at least as much) and ATK's minimum increment, through
  `get_minimum_increment`, as purego's callbacks return no floats for
  `get_increment`; AppKit has none, and its increments go through the
  keys. A read-only text input is
  `AccessReadOnly`, without `ActionSetValue`. AppKit finds an element's
  value settable when its class overrides the setter, whatever
  `isAccessibilitySelectorAllowed:` says, so elements answer the older
  `accessibilityIsAttributeSettable:` themselves, as AppKit would but for
  that value: `NSAccessibilityElement` has no implementation to call.
  The rows of a `List` (list items) and a `Table` (rows) are named by
  their content and say which of all the rows they are (`PosInSet`,
  `SetSize`, the list giving its total), as only those in view are built;
  rows built beyond the view are `AccessOffscreen`, and what is in a scroll
  container scrolls into view on request (`AccessScrollIntoView`), a row
  of a list by its place, which builds it. A list choosing rows has the
  focus for them: the tree's `Focus` is the chosen row while the list has
  the keyboard focus, so the arrows are read as they move the choice, and
  focusing a row chooses it, as screen readers expect of an active
  descendant. On macOS, lists are tables (`AXTable`) of rows
  (`AXTableRow`), as AppKit's, with `AXIndex`, `AXRowCount`,
  `AXVisibleRows`, `AXSelectedRows` and `AXSelectedRowsChanged`, and
  `AXScrollToVisible`, an action of AppKit's older API, which elements
  then answer for all their actions; on Linux, a list choosing rows is a
  list box with `AtkSelection` and `selection-changed`, rows carry the
  `posinset` and `setsize` object attributes, and `AtkComponent`'s
  `scroll_to` (ATK 2.30) scrolls; on Windows, rows have `PositionInSet`,
  `SizeOfSet`, `IsOffscreen`, `SelectionItem` (raising `ElementSelected`)
  in their list's `Selection`, and `ScrollItem`. A list choosing several
  rows is `AccessMultiselectable` (ATK's `multiselectable`, UIA's
  `CanSelectMultiple`, rows then raising `ElementAddedToSelection` and
  `ElementRemovedFromSelection` unless chosen alone), and its focus is on
  the row last chosen.
- **The connection.** `content.go` attaches the content to its window
  through `internal/surface.Conn`, which carries the surface and, as
  functions, what the content needs of the app (the clipboard, dragging
  the window, the appearance, the room of a hidden title bar's controls,
  opening URLs, context menus), so `ui` imports neither
  `mygo` nor a backend, and an app without native UI links none of it.
  `Window.Update` and `Invalidate` coalesce redraws asked from any goroutine
  into one frame on the main thread, which `Conn.Changed` asks the content
  for, so that it builds anew. Page methods return `errNoPage` or do
  nothing.
- **Frames.** The engine (`ui/runtime.go`) calls the view to build a frame,
  again (up to three times) when a handler changed the state while it built,
  so the frame shows the outcome. A window that closes during a frame (a
  close button's click; backends report `Closed` at once) keeps its engine
  until the frame ends: the rest of the view builds with a working
  `Context`, no further callback runs, nothing is laid out or painted, and
  the engine closes as the frame ends. Observing `Pressed` when a pointer press
  begins also rebuilds before painting, so selection made on press and its
  focus colors appear together; holding the pointer asks for no further
  build passes. The engine lays it out with flexbox (`layout.go`) or
  as a grid (`grid.go`, CSS grid's placement and track sizing for fixed,
  fractional and content-sized tracks); commits the boxes to the elements'
  states with the hit list in paint order, the focus order and the labels;
  paints a `scene.Scene`; and presents it. Input between frames goes to the
  states of the last frame's elements. An element's identity hashes its
  parent's with its position or `Key`, so focus, scroll offsets, editors and
  animations survive rebuilding. `Context` is stable for the window; `Element` values expire between build
  passes and validate their owner, slot and generation before accessing storage. `ListState` resolves
  its focus owner only for the current context, frame and pass, exposing
  focus and shortcuts without retaining an element in app state. Scroll
  offsets move in the layout too
  (`ui/scroll.go`): elements that asked to `ScrollIntoView`, and the focus,
  come into view before the boxes are placed, and placing keeps each offset
  within its content; a `ScrollState` mirrors an offset both ways, and a
  frame that moved one builds another for what read the old one. Frames
  happen only when asked: input, `Invalidate`, `After`, or `AnimationFrame`
  while something moves.
  A frame asked for only by drawings that move (`Painter.AnimationFrame`,
  `Painter.After`, as spinners and progress bars of unknown length do)
  paints the elements of the last frame again at its own time
  (`repaintFrame`), without building or laying out; elements out of view
  are not painted, so they ask for none. Such a frame is asked for with
  `redraw` set, which every event of the surface clears, so that the frame
  builds anew in case the event changed what the view shows; anything
  asking for a frame clears it too, and the frame `Painter.After` has due
  with it, which the frame asked for replaces: `requestFrame`,
  `Conn.Changed` (`Window.Update`, `Invalidate`, and `After`'s timer
  through them) and a change of the appearance. An event that asks for no
  frame, as the pointer moving over elements that do not look at it,
  leaves that frame due. A frame of another size, or after the text
  system forgot its layouts (`text.System.Generation`, as it lets go of
  fonts the elements' layouts hold), builds anew all the same. The timers
  the last frame built armed stay; `Painter.After` has a timer of its
  own, which posts to the main thread.
  While nothing of the window shows (`platform.OccludableSurface`: a macOS
  window hidden, minimized or covered by other windows, whose display link
  still ticks, at the display's rate for half a minute, then at about 40
  Hz), what moves asks for no frame (`held`), as browsers pause the
  animation frames of windows out of sight; changes of the state still
  build frames, so that assistive technology and captures follow them.
  `SurfaceShown`, sent as some of the window shows again
  (`windowDidChangeOcclusionState:`), has a held animation go on from
  where the time puts it.
  The window gaining or losing the keyboard focus draws a frame only when
  the last frame read it (`windowFocus`, as a focus ring, a caret, a
  selection's color or a focused text input's IME state do): a window
  hiding or showing again draws nothing it would draw the same. Each
  frame matters on macOS, where the Metal driver takes 30 to 45 MB of the
  process's footprint ("owned unmapped (graphics)") for the first command
  buffer encoding anything after a pause, and gives them back a second or
  two after the last, as a bare Metal app does.
  A frame allocates next to nothing once the view builds what it built
  before: elements come from the context's arena, the default theme is
  copied for each pass, the states that pruning frees go to new elements
  (as rows coming into a list's view take those of rows that went out of
  it), without keeping their local resources, editors or callbacks.
  The arena uses chunks of 32 elements, releases unused chunks after a
  smaller frame, and keeps one empty chunk for growth; unused elements
  and the spare arena of exit transitions give up their references after
  the frame copied what it needs. A text keeps in its state what it made
  of its spans
  (`spanCache`: their text, the styles of their layout, encoded, and where
  each ends), which a frame compares rather than makes again. With
  `MYGO_FRAME_STATS` set, frames slower than its threshold log how long
  each part took, which path drew them, what the process allocated and
  whether the collector ran (`ui/framestats.go`).
- **Visual transforms and motion** (`ui/transform.go`, `ui/motion.go`). An
  element's transform, 2D, or 3D in a 4×4 matrix (`mat4`) that it flattens
  into its parent's plane as CSS's flat transform-style does, composes
  with its ancestors after layout as a projective map
  (`scene.Affine`, with G and H in perspective); boxes keep their original
  layout space. Painting stores the composed transform on scene
  operations and clips, and committing keeps it on the state for
  inverse-mapped pointer input, carets, accessibility bounds, and overlays
  attached to visual bounds. The CPU renderer inverse maps pixels for
  transformed primitives and clips, and leaves what a projective map puts
  behind the viewer. GPU builders retain local geometry and send three
  transform rows with every instance; the vertex shaders project the quad
  in homogeneous coordinates, so that local coordinates interpolate in
  perspective, and fragment shaders integrate coverage in device space:
  from screen derivatives for affine maps, and, in perspective, at the
  CPU renderer's sample points, mapped back from the pixel's center
  through the inverse. Nested
  transformed or rounded clips use cached masks, independent of content
  moving within them; hard clips in the drawing's coordinates evaluate on
  the GPU. Backdrop effects map local samples into the frame. With a GPU,
  every frame draws there, including resting changes to transformed content.
  Metal retains wide colors under transforms. See
  [Transforms](ui/transforms.md#rendering) for rendering details.
- **Motion** (`ui/motion.go`): every animated value of an element
  (`AnimateTo`, `Animate`, `AnimateWith`, `Spring`, `Jump`) is a `channel`
  by key, with a value, a velocity and the time they hold for, moving
  along a curve or a spring. Springs solve damped oscillators analytically
  (`solveSpring`), so frames however far apart follow one path; a channel
  samples its path at the frame's time before taking a new target or kind
  of animation, so position and velocity survive interruptions (a curve's
  velocity is its easing's slope). A delayed target waits in the channel
  while the old path goes on. `Jump` sets values and derives a velocity
  from successive ones; presses keep their last moves (`dragTrack`), in
  the element's axes and the window's, for `DragVelocity` and
  `DragEnded`. Keyframe sequences keep one sequence per
  key with an explicit replay token. All ask for frames only while moving,
  and end at their targets under reduced motion. Poses (`ui/pose.go`,
  `AnimatePose`) are channels sampled as frames commit, into a transform
  after the element's own (`poseM`) and a fade; frames that only paint
  (`repaintFrame`) move the posed elements of the last frame built on
  (`repose`) and set their subtrees' transforms again (`updateWorld`), so
  they move without the view running. `Draggable` (`ui/draggable.go`)
  keeps two channels of an offset (`dragRec`) that `showPose` adds to the
  pose's translation: as the pointer drags the element, they hold where
  the press's move in the window (`dragTrack`), mapped into the parent's
  axes through the inverse of its last world transform, takes the offset
  from where the press caught it, so they are the same in every build
  pass of a frame; once let go, a spring starts at the release's velocity
  for where `Snap` chooses from the projected throw, and goes on in frames
  that only paint, the view building once more when it comes to rest. Content transitions (`ui/content.go`)
  keep what an element showed, and fade or roll it into what it shows as
  it paints, counting paintings (`engine.paints`) rather than frames
  built, so frames that only paint move them on too.
- **Transitions** (`ui/transition.go`) animate elements FLIP-style, after
  the layout of each frame and before `place` turns boxes into window
  coordinates: an element given a `Transition` keeps, by its ID, where the
  layout put it relative to its parent, its size, and what it shows, and
  goes from what it showed to a new place or size over the transition
  (parents first, as one resizing lays out its content anew at each size,
  which then moves itself). Moving with a parent or a scroll offset is no
  change; resizing the window animates nothing, and reduced motion only
  fades and colors. Each moving component (position, size, opacity, and a
  `Motion`'s scale and rotation) is a motion `channel`, so a spring keeps
  its velocity through changes and a delay holds them. Colors move as
  elements paint, after their `styleFn`. A `Match` moves an element
  appearing where another of the same identity showed the frame before
  (`rt.matched`, in window coordinates) after `place` (`animateMatches`),
  laying it out anew at each box on the way, across parents and scroll
  offsets. An element appears with
  its `Enter` only in a parent the frame before had. One no longer built,
  whose parent still is, goes with its `Exit` as an inert copy, laid out
  and painted under its siblings without taking room: frames that built
  elements with an `Exit` leave their arena to the next frame, which
  builds in the other (`Context.spare`), so that the next one copies those
  gone (`ghostOf`), with states of their own and nothing referring to the
  app's state. Siblings are not laid out around a size moving; they move
  with their own transitions, in step when they share it.
- **The inspector** (`ui/inspector.go`, `inspector_details.go`,
  `inspector_panel.go`, `inspector_overlay.go`), as Chrome's developer
  tools. A window whose DevTools are on (`surface.Conn.DevTools`, from
  `PageOptions.DevTools`) opens it for the Toggle Developer Tools role
  (`Conn.ToggleDevTools`), F12, or Alt+Cmd+I (Ctrl+Shift+I), and picks
  for Shift+Cmd+C (Ctrl+Shift+C). It is a panel built with the view's
  elements as the last child of the root, keyed, which is narrower by
  the panel's width (the view sees a narrower window); hits and paint
  cover the whole window. Its theme is a copy of the default one in
  Chrome's colors, light or dark, whose accent colors the tree's chosen
  row. After each frame is laid out, it notes the tree of elements, the
  keys given them while it is open (`rekey`), and the details of the one
  chosen (`describe`: its styles as CSS rules, own and inherited, its box
  model, computed values and properties), and asks for another frame when
  a hash of them changed, which the panel then shows; frames whose
  elements stay the same draw no more, and the times of frames, which
  the Performance tab draws against the display's refresh, ask for
  none. The tree is a `List` of the opening and closing tags of the nodes
  shown. Picking takes the pointer
  over the content before the elements do. Painting highlights the
  element hovered, or chosen while the tree has the focus, over the
  content, with a tooltip. The source of the element chosen is the first
  frame outside package ui on the stack as it is created or keyed
  (`callSite`), only for that one. Two elements keyed alike under one
  parent share one state: `rekey` reports it (`duplicateKey`), logged
  once and listed as an issue, and in a `Tester` it panics
  (`engine.strict`), without package testing in apps.
  Production builds of `mygo build` leave the inspector out with the build
  tag `mygo_noinspector` (`inspector_off.go` stands in for it, never
  open), added to the tags of `GOFLAGS`, unless `-debug` or
  `MYGO_INSPECTOR=1`.
- **Input taken as it comes.** An element with `HandleInput` gets its
  input on the main thread as the backend reports it, before the frame
  (`ui/handler.go`): keys (with their releases, which ui otherwise
  ignores), text, compositions and edit commands while it has the focus,
  unless a `Shortcut` around it claims the key, and the pointer pressed on
  it, moving over it or scrolling over it; once it takes a press, the
  moves and the release come to it, as to a pressed element. Its
  `TextCaret` turns on the input method at that caret, with no text around
  it. `ui.Shape` lays out text without the cache of layouts, for widgets
  that keep their glyphs, and `Painter.Glyphs` draws them where they
  placed them.
  `HandleTextInput` connects a custom element's application-owned
  `TextInputClient` to the same platform services. A stable, focus-checked
  adapter supplies absolute UTF-16 text ranges, selection, marked text,
  mutation callbacks, range geometry and hit testing. It is invalidated on
  replacement/disposal, and focus changes unmark the old client and reset
  native composition. GTK and IMM32 use bounded surrounding context while
  macOS can query arbitrary document ranges directly. `ShapeText` and
  `ShapeRichText` retain per-paragraph `TextLayout` geometry; an application
  owns its buffer, selections, rendering, editing and undo policy.
- **Preferences.** `platform.Theme.Preferences` reads the settings of the
  desktop that controls follow: on macOS, `controlAccentColor` and
  `NSWorkspace`'s accessibility display options, with their notifications;
  on Windows, DWM's `AccentColor`, `SPI_GETCLIENTAREAANIMATION`,
  `SPI_GETHIGHCONTRAST` and Accessibility's `TextScaleFactor`, with
  `WM_SETTINGCHANGE`; on Linux, GTK's `gtk-enable-animations` and the
  settings portal's accent, contrast and GNOME's text scaling factor
  (`ReadAll`, `SettingChanged`). A change goes through
  `Handler.ThemeChanged`, as the appearance's does; package `ui` reads them
  once until the next, the default theme follows them (`Theme.follow`), and
  `Animate` follows reduced motion.
- **Lists** (`ui/list.go`) build only the rows in view and keep their
  place by a row, the anchor, and how far its top is above where the
  content starts, not by an offset into their content: rows are measured,
  grow, come and go (found again by key, `ListState.Key`, within a
  thousand rows of where they were), and those in view stay put. Building,
  a list makes the rows its place shows as far as the heights it knows
  tell, and the row holding the focus and the header it pinned; laying out
  (`layoutList`, in place of flexbox), it measures them, places them from
  the anchor, a request (`ScrollTo`) or the end it follows, builds the rows
  still missing, so that no frame shows a gap, and keeps the content's
  ends at the list's. Heights measured live in blocks of 64 rows, made as
  rows are measured, and the others are estimated as their average; the
  content's size, and its offset, are where those heights put the rows,
  in float64, as all scroll offsets are, so that millions of rows scroll
  by fractions of a DIP. Rows are placed relative to the list, the offset
  they were placed at kept as `scrollBase` for placing and revealing. An
  offset the wheel, the keys, the scroll bar or the app changed moves the
  place by as much when it is a step (up to two views), so rows not
  measured yet scroll by the step; a jump goes where the heights known put
  it once the rows built are measured, and the end to the end. A reveal
  (the focus, `ScrollIntoView`) lays a list out anew from the row it
  shows, the window's ScrollState-set offset of a list built anew is a
  jump, and each element remembers the `ListState` that placed it last. A
  scroll bar's thumb keeps the content's size of when it was grabbed until
  it is let go, as rows measured meanwhile change it, and a frame that
  moved a list whose `Visible` or `AtEnd` the view read builds another.
  Rows built as a list lays out handle their input as the view's do,
  which `layoutTree` then forgets, asking for a frame if they changed
  anything, and lays out what they put in the overlay. Rows a `ScrollTo`
  shows at the top go below the header pinned over their section. A list
  without a size is as high as its rows, as the heights known tell, and
  one without rows lays out what else was built in it, as a scroll
  container. `Table` is a header of columns over a list, whose rows are as
  high as their tallest cell, and which takes the focus and the keys for
  the list. Lists choosing several rows hold them by key in a
  `Selection[K]`, through the unexported methods of the `Selector`
  interface, so that the app reads its own keys back typed; the row last
  chosen (the cursor: `Selected`, or the list's own), and the row Shift
  extends from, follow their items by key as the choice does. Typing to
  choose a row goes through the engine: letters, digits and spaces that no
  shortcut takes add up in the focused list's state (`typed`, reset after
  a second), which the list matches against `ListState.Label`.
- **Menus.** Context menus and menu buttons share the machinery of
  `ui/menu.go`: an event asks for an element's menu (`askMenu`, with
  whether it is a menu button's), the next frame runs the element's menu
  function to collect the items, the host shows the system's menu after
  the frame (`popupMenu`), and the item chosen comes back to the function
  in the frame after, matched by place and label. A menu button opens as
  the primary button goes down on it, taking the release, as AppKit's and
  GTK's pop-up buttons do, and for Enter, Space and Down.
- **Routers** (`ui/router.go`). A `Router` is a history of entries, each
  a location (an escaped path and its query) and the pages showing it, one
  per level of views: `View`'s, then each `Route.View` inside a layout's
  page, made as first built. An entry pushed shares the pages of the
  layouts whose part of the path it keeps (each entry notes how many parts
  each of its layouts took), and all of them for another query of the same
  path. Each view keeps which entry it showed (the router, or the layout
  page's element for a `Route.View`), so a change transitions only the
  level whose page changed. A view builds the page shown in a column keyed
  by the page, flagged `flagPage`: committing marks
  each element's state with the page around it (`state.page`), and
  `prune` keeps the states the frame did not build when a page around
  them, through nested pages, is one the routers built in the frame kept
  (`engine.kept`): the ten pages of the history nearest the one shown, a
  layout's view keeping those inside the same layout.
  The focus does not stay in a page kept out of sight. As the page shown
  changes, the router remembers the element of the page left that had the
  focus, and moves the focus into the new page, to the element it
  remembers there, else to the page (`flagFocusTarget`: focusable, not a
  stop of Tab), unless an element of the page took it as it was built
  (`AutoFocus`), or the focus was outside the router, where it stays and
  the page's title is announced. A page deeper in the paths slides in
  over the page left, the one going away built too, inert (`flagInert`:
  painted, without hits, focus, labels, accessibility nor `Context`
  shortcuts), absolutely positioned over the one in flow, both offset by
  their insets as relative positioning; others fade in alone. The first
  router of a frame takes the back and forward keys for the window, each
  one also for the focus inside it, which comes first.
- **Comboboxes** (`ui/combobox.go`). A combobox is a text input with a
  popup of options in the overlay, without a backdrop: the input keeps
  the focus while the user types and picks, as presses in the popup do not
  move the focus (`flagKeepFocus`), and the popup closes as the input loses
  it. The option the arrows are on is the input's active descendant,
  which has the focus in the accessibility tree, as a list's chosen row
  does, and reads as which of how many it is. Fields around an input (a
  combobox's, a search field's, a token field's) name it with their
  Label (`nameFrom`). Backends treat combo boxes as text fields (macOS
  `AXComboBox`, ATK combo box with AtkEditableText, UIA ComboBox with
  Value and ExpandCollapse); `AccessSearch` makes a text field AppKit's
  `AXSearchField`.
- **Disclosures** (`ui/collapsible.go`). A collapsible's trigger is a
  `RoleDisclosure`, expanded while open: AppKit's `AXDisclosureTriangle`,
  whose value is 1 while open; a toggle button that is expandable, and
  checked and expanded while open, as GTK's expanders; a UIA Button with
  ExpandCollapse. Its panel animates `Progress` with `Animate`, which
  follows reduced motion: while it moves, the panel is clipped to that
  share of the height its content had in the last frame, and its content
  is built until it has closed.
- **Text areas** (`ui/textarea.go`, `ui/textbuffer.go`). An editor
  holds its text as the string of the app's value, which it shares, and
  where each paragraph starts in runes and in bytes (`buffer`): the frame
  compares the value with it at once while it is the same string, and an
  edit copies the bytes once, where converting the whole text between
  runes and a string took milliseconds for a few megabytes. Undo keeps the
  changes of each step (`undoStep`), not copies of the text; a text the app
  sets makes the last step one change from the text before it, which
  undoing takes back, so that a log the app keeps setting holds two texts
  there, not one a frame. Deleted fragments own their bytes, and
  discarded history and input events give up their references, so a
  one-character deletion does not retain an entire old document.
  The paragraph index is allocated for the newline count up front;
  scanning ASCII counts runes a word at a time, with the standard UTF-8
  decoder for other text. A much smaller replacement releases the large
  paragraph and height indexes. Grapheme boundaries come from the
  paragraph of the caret. A text area lays its
  text out a paragraph at a time (`area`), as the text system breaks
  lines anyway, so that the lines are those of the text laid out whole:
  each paragraph keeps its layout until an edit changes it or the width
  does, those in view are laid out from the paragraph the view starts in
  (the anchor) down, those far from view give their layouts up and keep
  their heights, and layouts own their paragraphs' text rather than
  keeping old document strings alive. The heights not measured are
  estimated by those measured. Two Fenwick trees, of the heights measured and of how many are
  not, give the top of a paragraph and the paragraph at a height in
  O(log n) whatever the estimate. The area scrolls as a scroll container,
  its offset the state's (`flagScrollY`, the content as high as its
  paragraphs), kept by the anchor as heights above the view are measured;
  an edit, a move of the caret or a press reveals the caret once.
- **Editing layers.** `ui/editor.go` holds the widget's editing state;
  `editor_history.go` owns delta undo, `editor_selection.go` the visual
  range set and caret affinity, `editor_navigation.go` keys and pointer
  gestures, and `editor_layout.go` layout and painting. `textinput.go`
  builds the public string widgets. They use the same `TextInputClient`
  contract as custom controls through `editor_input.go`: native callbacks
  query and mutate state synchronously, while the widget publishes its
  bound string during its next build, preserving change propagation in
  composed controls. Preedit is a virtual document insertion, and its
  replacement and commit form one delta undo transaction. The paragraph
  index carries rune, byte and UTF-16 starts; bounded native queries own
  their bytes so they do not retain old document allocations.
  Single-line controls retain their current layout themselves, keeping
  changing input strings out of the system's cache of display text.
  `internal/text/caret.go` keeps both logical edges of bidi boundaries and
  wrapped lines, and moves between whole graphemes in visual order.
  Selection gestures produce logical range sets: highlight, copy,
  replacement and undo use the same ranges, preserving unselected gaps.
- **Indexed text storage.** `ui/text_buffer.go` is a persistent AVL tree of
  bounded, owned UTF-8 chunks, summarized by byte, rune, UTF-16 and newline
  counts. Edits copy affected chunks and tree paths; snapshots share other
  chunks, and export can stream them. `TextInputBuffer`/`TextAreaBuffer` in
  `textbuffer_input.go` bind those roots to the existing editing client.
  Native mutations publish a new root immediately, using a version check
  to preserve concurrent program edits. External root changes refresh the
  widget and clear stale history. The indexed buffer adapter derives line
  starts from the text tree rather than moving every later paragraph's
  absolute offsets. Paragraph layout and height caches remain with the
  widget; newline-count changes update those indexes. Full text is produced
  for explicit export/value queries, never as a binding update per edit.
- **Text selection** (`ui/textselection.go`). `Selectable` on a text
  selects that paragraph; on a container it gives its text descendants
  one selection. The window keeps endpoints as stable element IDs and
  rune offsets, and each frame collects participating paragraphs in
  build order. After layout the shared range is projected onto their
  editors, whose layouts paint the highlights. Pointer gestures,
  keyboard extension, native editing commands and context menus use
  the same endpoints. Nested containers have independent scopes;
  controls and `Unselectable` subtrees do not participate. A drag near
  a scroll edge asks for frames until scrolling stops. Inline children
  contribute to their paragraph once, and preparation waits for their
  final text so a rebuild preserves the selection.
- **Tables** (`ui/table.go`, `ui/editable.go`). A table's rows are a
  `List`'s that scrolls both ways: the list lays its rows out at least as
  wide as the columns ask (`rowMinW`), and the header, outside the list,
  follows its horizontal offset as it is placed (`followX`). The order
  and the widths the user gives the columns live in the `ListState`
  (`TableLayout`), by column ID. Dragging a header moves its column once
  the pointer went a few DIPs, past a neighbor whose middle it passes; a
  double click on a header's edge fits the column as the table lays
  out, from the intrinsic widths of the cells the frame built
  (`tableFit`). Sorting is the app's: the table sets `ListState.Sort`
  and shows it, to assistive technology as well (`AccessSortAscending`
  and `AccessSortDescending`: AppKit's `AXSortDirection`, ATK's `sort`
  attribute, UIA's ItemStatus). An `EditableText` in a row reads the row
  being built (`Context.row`): the list, which keeps the focus, asks for
  the chosen row's text to be edited on Return (macOS) or F2, which it
  takes only once a row has such a text; a click on the text of a row
  chosen before the click edits it once the double-click time has
  passed.
- **Outlines** (`ui/outline.go`). An outline flattens the open part of
  its tree into rows every frame (`OutlineState.flatten`), which a `List`
  or a `Table` builds as they show, keyed by their items. Rows are tree
  items, with how deep they are (`AccessNode.Level`) and whether they
  have children (`AccessExpandable`), open or not: on macOS, the rows of
  an `AXOutline` with `AXDisclosureLevel` and `AXDisclosing`, which keeps
  its rows as a table does; on Linux, the `level` attribute and the
  expandable state; on Windows, Level, and ExpandCollapse with LeafNode
  for items without children.
- **Drag and drop within a window** (`ui/dragdrop.go`). An element
  dragging a value notes it in its state (`Drag`), or a function giving it
  as a drag starts, as a list's rows chosen. The engine starts the drag
  once the pointer pressing it moves a few DIPs (`dragMove`), and follows
  the innermost element under the pointer whose `accepts` takes the value,
  which `Drop` and `DragOver` set from their type. Releasing drops the
  value on it and is no click (`dragEnd`); Escape gives up. Each frame
  scrolls the scroll container under the pointer near its edges, and
  paints the source's element of the frame again under the pointer,
  shifting it there and back, with a count beside the pointer for
  several rows. Lists and grids take their own rows (`rowDrag`,
  `itemDrag`), placing them by the middles of the rows of the last frame.
- **Native data drags** (`transfer`, `drag.go`, `ui/data_drag.go`). An
  element with `DragData` promotes the gesture to the native drag tracker.
  `transfer.Data` holds immutable items with MIME-tagged representations;
  providers are advertised lazily and cached per session. `DropData`
  accepts matching formats with a negotiated copy/move effect. The core
  owns one live source, identified by a random token: backends transfer the
  token as data, never a Go pointer. Only the core resolves it to the
  original items and optional typed value, while the source lives. A
  completion or cancellation invalidates the token and calls Done once.
  `Surface.StartDataDrag`, `CancelDataDrag` and `SetDropFormats` translate
  to AppKit sessions/pasteboard item providers, GTK selections, or OLE
  IDataObject/IDropSource/IDropTarget. GTK collects accepted formats through
  asynchronous callbacks after drop; OLE runs its native modal tracker
  after the source input callback returns. None waits on the main thread
  for data that needs that thread. Dropped bytes outlive native objects;
  unhandled file transfers still reach the existing file-drop path, with
  copy semantics. The unsupported backend cannot create windows/surfaces,
  and continues to fail at Init without linking any native drag code.
- **Grid views** (`ui/gridview.go`). A grid view is a `List` of rows of
  items, whose columns it takes from the width its rows had in the last
  frame: the layout asks for another frame when the width calls for
  others (`gridFit`). The list's own state holds the rows' place; the
  grid chooses items itself, takes the keys, and tells assistive
  technology about its items rather than its rows, which are RoleNone:
  a list of all its items, the item chosen its active descendant.
- **Sidebars** (`ui/sidebar.go`). A sidebar is one element taking the
  focus, whose items note their IDs as they are built: the keys choose
  among those of the last frame, and the item chosen is the sidebar's
  active descendant, with the focus for assistive technology, as a
  combobox's option. Sections and items are tree items, the sections'
  RoleNone containers leaving them all in the sidebar's tree, whose rows
  macOS keeps as an outline's.
- **Feedback** (`ui/feedback.go`, `ui/toast.go`). An alert dialog is a
  modal overlay as a dialog's, which a click outside leaves, as AppKit's
  alerts beep: an alert role (AppKit's `AXApplicationAlertDialog`, ATK's
  alert, UIA's dialog pane) named by its title and described by its
  message. A find bar is keyed, for its state, which focuses its field
  as it opens, to go as it closes. A checkbox group finds its check boxes
  as they register with it (`Context.checks`), then applies a click to
  them all. A text input notes the modifiers of the Enter submitting it
  (`submitMods`), as the editor takes Shift+Enter: the find bar goes back
  with it.
- **Dates, times and colors** (`ui/date.go`, `ui/timeinput.go`,
  `ui/colorpicker.go`). `DateInput` and `Calendar` share one month's grid,
  whose day under the keys is its active descendant. A time input's
  hours and minutes are spin buttons named after the field, typed into
  through type-to-choose (`flagTypeSelect`). A color picker keeps its
  color as hue, saturation, value and alpha, as a color of red, green
  and blue loses the hue of its grays, and takes the app's color anew
  when the app sets another. A color well is AppKit's `AXColorWell`, a
  button on Linux and Windows, its value the color in hex.
- **Indicators** (`ui/indicators.go`). Meters and steppers have a value
  in a range, as sliders and progress bars do (`AccessRole.Ranged`):
  a meter is AppKit's `AXLevelIndicator`, ATK's level bar and UIA's
  ProgressBar; a stepper AppKit's `AXIncrementor`, ATK's spin button and
  UIA's Spinner, taking increments. A range slider's knobs are sliders of
  their own, named after it (`nameJoin`).
- **Forms** (`ui/form.go`). A form notes the label boxes of its fields as
  they are built, and makes them as wide as the widest before the layout
  first measures or lays it out (`intrinsic` and `boxLayout` call
  `alignLabels`), its fieldsets' fields too. The row of a field in a form
  lines up the first baselines of its label and control after laying them
  out (`alignBaselines`), from the text layouts of the frame; a control
  without text centers on the label's line. Fields name their control
  with their label (`nameFrom`). `Description` and `Error` reach
  assistive technology as `AccessNode.Description` and `AccessInvalid`:
  `AXHelp` and `AXInvalid` on macOS (the newer API has no invalid state,
  so the element answers it through `accessibilityAttributeValue:`), the
  ATK description and `invalid-entry` on Linux, UIA's FullDescription,
  HelpText (unless a placeholder takes it) and IsDataValidForForm on
  Windows. Tooltips describe their element as AppKit's and GTK's do.
  Committing a frame marks disabled the state of what is inside a
  disabled element, which a fieldset disables after building it: input
  in the next frame then finds it disabled (`Element.disabled`).
- **Focus groups** (`ui/focusgroup.go`). Committing a frame notes the
  outermost focus group (`FocusGroup`) each element of the focus order is
  in, beside its dialog. Tab skips the elements of a group but its entry,
  the one that had the focus last, else the radio button or tab chosen,
  else the first, and leaves the group; the arrows, Home and End move the
  focus among its elements when no element around the focus takes them as
  a shortcut, and click the element they reach in a radio group.
  `Toolbar` lays out (`layoutToolbar`) the controls that fit, takes the
  others out of the flow (absolute and invisible, `collapsed`), and lists
  them, by label, for its overflow menu, whose choices click them after
  the pass (`clickLater`).
- **Overlays** (`ui/scope.go`) scope the keyboard. Committing a frame
  notes the dialog each focusable element is in (an element made `Modal`,
  as `DialogBase`'s backdrop, `flagModal`; a popover is in its anchor's),
  the dialog on top (`engine.modal`, and `modalLayer`, the element at the
  top of the overlay holding it), and moves a popover's elements after its
  anchor in the focus order. Tab cycles the dialog on top, the focus moves
  into it while it is outside, the window's shortcuts built outside it do
  not fire, and the accessibility tree is the dialog's layer and what is
  above it. Overlays register Escape as overlay shortcuts
  (`OverlayShortcut`), which the keys the focus and the elements around it
  leave reach, the last registered (the overlay on top) first. Each element
  built at the top of `Overlay` notes the focus as it opens (`openers`),
  which pruning gives back once it is gone with the focus that was in it.
  An element attached to another (`AttachTo`, which `PopoverBase` and the
  popups of selects and comboboxes use) is that element's popover
  (`Element.popover`, and `state.anchor` once committed): its layout
  (`attachTo`) finds where the target will be in the window from the boxes
  laid out before it, the scroll offsets around it, and for an inline
  target the paragraph's layout (`laidOutBox`), so that a popover follows
  its anchor in the frame that moves it; then it goes to the other side, or
  the other way along the target, where that leaves the window less, and
  moves along the target into the window (`alongTarget`). Popovers have no
  backdrop: the engine notes what each press went down on (`downs`, for a
  pass), and `PressedOutside` walks from the last through parents, and from
  a popover to its anchor, which keeps presses in popovers of elements
  inside the panel, and on its anchor, inside it, as the press goes on to
  what is under it; a press on the anchor after one elsewhere in the same
  pass, as on a web view's page, leaves open the popover it opens. A select's popup keeps a backdrop taking the presses outside,
  as the system's pop-up menus do.
- **Web views** (`webview.go`, `ui/webview.go`, `webview.go` of each
  backend). `Window.NewWebView` makes a web view of a window showing
  `Content`, whose page lives in a `Window` of its own that no list holds,
  with `host` set: calls, channels, events, `Eval`, custom schemes,
  downloads and permissions go through the code of windows' pages
  unchanged, `CallerWindow` returns the host, `CallerPage` the page, and
  the bridge has the host's id, so the window controls of the page act on
  it. Its native side (`webViewNative`) takes page methods from the
  backend's `platform.WebView` and window methods from the host's, which
  drag regions use. `ui.WebView` places it: committing a frame notes where
  each web view's element shows (`noteWebView`: its content box on whole
  pixels, clipped as its hits are), and painting it makes a hole
  (`OpHole`) through what the window painted, rounded as the border's
  inner edge and cut by the clips around it, so that what paints after
  the element paints over the page. The native web view is under the
  surface, which shows it through its transparent frames; what shows over
  it blends with it as over any pixels. After presenting a frame, the
  engine gives the surface each web view's frame and clip and the boxes
  of the hits after its own within its clip (`Covers`): the pointer is the
  page's where it shows and nothing painted over it takes the pointer
  (`WebViewPlacement.At`), as a dialog's backdrop does over all of it. A
  press there goes to the page and comes to the content as `WebViewPress`,
  which notes it for `PressedOutside` and takes the focus from the
  content; the engine sets no cursor while the pointer is over a web view
  (`cursorUnknown`), whose page sets it. Web views the frame does not build
  hide and keep their pages. Glyphs over a hole are not on an opaque
  background (`Painter.holes`), so that subpixel ones do not color the
  page. Below full opacity a hole takes only that much of what is under
  it away (`OpHole`'s `Opacity`), so the page blends over it. The element
  is a Tab stop: focusing it gives the page the keyboard
  (`WebView.TabInto`, from its end going back), and Tab past the page's
  ends comes back as `WebViewTabOut`, from which the focus moves on. The
  access tree has a node for each page, inside its element
  (`AccessNode.WebView`), which the backends fill with the page's own
  accessibility, as SwiftUI's `WebView` is in place. Backends:
  - macOS: the WKWebView is in a flipped clip view (`MyGoClipView`, its
    layer masking to its bounds) placed at the clip, under the surface in
    the content view, in paint order; the Metal layer is not opaque, and
    frames drawn in memory have their alpha. The surface's `hitTest:`
    returns nil where a web view takes the pointer, so AppKit finds the
    page under it. A web view's tracking areas get the mouse's moves
    wherever its frame is: `MyGoWebView` drops `mouseMoved:`,
    `mouseEntered:` and `cursorUpdate:` where the content painted over it,
    and as the pointer goes there from the page sends WebKit an exit
    outside the view, which WebKit takes as a move there, unhovering the
    page; WebKit sets its cursor only where `hitTest:` finds the web view,
    and the surface sets none over a page. `TabInto` puts the web view
    after (before) the surface in the window's key view loop and selects
    it, so WebKit focuses the page's first (last) element; as Tab leaves
    the page WebKit selects the next key view, which `MyGoWindow` turns
    into `WebViewTabOut`. The page's node lists the WKWebView, whose
    `accessibilityParent` is that node; the clip view lists none.
  - Linux: the first web view puts the surface's area in a `GtkOverlay`,
    over a `GtkLayout` holding the WebKitWebViews (which, unlike a
    `GtkFixed`, asks for no size), as an overlay, which GTK realizes again
    with a new GL context (the GL renderer makes its objects again). The
    GtkGLArea has an alpha channel from then on, and frames drawn in
    memory paint over the web views (`CAIRO_OPERATOR_OVER`). An input
    shape on the overlay's window of the area
    (`gdk_window_input_shape_combine_region`) lets the pointer through to
    the pages, whose windows then set the cursor; a web view's own
    `button-press-event` tells the content. WebKitGTK focuses no element
    as a view takes the focus, so `TabInto` blurs the page's and passes the
    Tab being handled on to the view (`gtk_widget_event`); WebKit moves the
    focus out as Tab leaves the page through GTK's `focus` signal, which
    the view's handler turns into `WebViewTabOut`. The page's node lists
    the view's accessible, an `AtkSocket` the web process's tree is
    embedded in; the overlay is a subclass whose accessible lists the
    surface's alone.
  - Windows: WebView2 in visual hosting, a composition controller
    (`ICoreWebView2Environment3`) whose visual is in a DirectComposition
    target of the surface's window that is not topmost (a window has one
    of each), on a device of its own without a rendering device
    (`DCompositionCreateDevice2`): the surface's frames, in the topmost
    target, are over it. The surface composes its frames from its first
    web view on, as in a window without a redirection bitmap: it sends
    `SurfaceRenew`, and the content makes its renderer again for a swap
    chain for composition, or shows frames drawn in memory through the
    compositor; GDI would draw under the web views. Each web view's
    visual is in one clipped to its clip, the page at its bounds in the
    surface's pixels. The surface gets the mouse everywhere and sends it on
    (`SendMouseInput`) where a web view takes the pointer, or to the one a
    press went to until the buttons are let go (with the capture), with a
    leave as the pointer goes; it sets the cursor the page asks for
    (`CursorChanged`), and a press moves the focus into the page.
    `TabInto` is `MoveFocus` (next or previous), and `MoveFocusRequested`
    becomes `WebViewTabOut`. The surface's drop target sends drags over a
    page to its composition controller (`ICoreWebView2CompositionController3`'s
    `DragEnter` and the others). For UI Automation the root also answers
    `IRawElementProviderHwndOverride`: for the window WebView2 shows a page
    with, it returns the page's node, which answers with WebView2's
    provider of the page (`get_AutomationProvider`), as WinUI 3 does. That
    provider has no children: the page's elements are under a top-level
    window of WebView2's (`Chrome_WidgetWin_1`), which
    `GetAutomationProviderForWindow` does not map to it, so UI Automation
    finds them there (seen with a runtime of October 2026).
- **Context menus** (`ui/menu.go`) open in two frames. A right-click or the
  menu key marks the element, from the states of the last frame, and the
  next frame runs its `ContextMenu` function to collect a `platform.Menu`;
  after the frame, package mygo shows it with `Menu.popup`, posted to the
  main loop, since native menus wait in a loop of their own. A choice
  comes back as the item's place and label, which the next frame's run of
  the function matches to report `Chosen`; text inputs' menus queue their
  editing commands instead.
- **Scenes** are flat lists of operations in device pixels: rounded
  rectangles with borders of a width per side, solid or dashed, filled
  with a color, a linear gradient mixed in sRGB or Oklab, or stripes;
  shadows (blurred rounded rectangles, cut by the box casting them, as
  CSS's box-shadow is, or inner ones, inside the box where the hole they
  leave, blurred, does not cover it, as box-shadow: inset is); runs of
  glyphs, whose masks may take a gradient (paths drawn with one); images,
  in color or gray; effects (`OpEffect`); holes (`OpHole`), rounded
  rectangles made transparent, what was painted there gone, for web
  views; and pushed and popped clips.
  Renderers draw the whole scene each frame and retain only textures.
  Wavy underlines are stroked paths.
- **Effects** (`scene.Effect`) are drawings that packages outside the
  renderers define, as the official plugins do (the glass plugin's Liquid
  Glass): a fragment shader for each GPU renderer, in Metal Shading
  Language, HLSL and GLSL, which the renderer puts between a head and a
  tail of its own (`effect.metal`, `effect.hlsl`, `effect.glsl`, after its
  shader's common part, with `EFFECT` defined: `EffectSource`), and a twin
  for the CPU renderer (`EffectPixels`), which must draw the same pixels.
  An effect paints its op's shape from five float4s of parameters
  (`EffectOp.Params`, which travel in the instance's slots fills use for
  colors and gradients), and from its backdrop when it reads one: what
  the operations before it painted, the area its blur reaches
  (`scene.BackdropOf`), averaged over squares of 1, 2, 4 or 8 pixels a
  side so that the blur stays a few texels wide, then blurred by a
  Gaussian along rows and along columns, each step kept in 8 bits a
  channel, as textures hold it, which the shaders sample bilinearly
  (`sampleBackdrop`, `BackdropImage.Sample`). The renderer multiplies an
  effect's color by its shape's coverage, the clip and the opacity.
  Effects carry the code compiled ahead of time from what their renderer
  makes of their shader, with its SHA-256: renderers compile the source
  when it no longer matches, as when their head or tail changed since,
  and an effect's tests fail then. Package `ui` gives packages a hook,
  `Element.Material` (a `ui.Material`, built with its element when it is
  a `MaterialBuilder`), and `Painter.Effect`, which only this module's
  packages can call, since effects are internal types. An effect's shape
  has continuous corners where its op does, which the renderer covers as
  such, while the effect gets positive radii and takes them as circular.
- **Corners.** On macOS, rounded corners are continuous, as AppKit's and
  SwiftUI's (`scene.Op.Continuous`), and drawn as Core Animation draws
  them on screen, with the function of its shaders (QuartzCore's
  `supercircle_sdf`), which differs from the Béziers of SwiftUI's paths
  by up to half a percent of the radius: a quarter circle around the
  diagonal that bends less and less toward the edges, which it meets
  1.528665 radii from the corner, a quartic of the ratio of the point's
  coordinates in the box of the curve. On sides too short for both
  corners' curves, the curves blend toward quarter circles by the side's
  clamp factor, from the side's length over its corners' mean radius, so a
  circle's corners are quarter circles; a shape is the intersection of its
  corners', so a curve may reach past the middle of a side whose other
  corner is smaller; and edges are antialiased as Core Animation does, by
  the value over the sum of its derivatives (`internal/raster/corner.go`).
  Against captures of layers on screen, every pixel is within 1/255 for
  corners that fit and for circles, and within 6/255 for pills and other
  short sides at the sizes of controls. The CPU's renderer finds where a
  curve crosses a row by regula falsi between the quarter circles of the
  radius and of the curve's extent, and keeps the rows' spans of a clip
  with continuous corners for the operations within it; a whole frame of
  `BenchmarkFrame` takes a fifth longer. Elsewhere corners are circular,
  as Windows and GTK draw them, and only the CPU's renderer and Metal's
  draw continuous ones.
- **Text.** `internal/text` lays out text with the system's own text stack,
  behind a small `engine` interface: DirectWrite on Windows
  (`IDWriteTextLayout`, with an `IDWriteTextRenderer` implemented in Go
  collecting its glyph runs), Core Text on macOS (`CTTypesetter`, with
  `NSFont` for the system font's weights) and Pango on Linux (with
  fontconfig and HarfBuzz underneath, as in GTK); in browsers, a canvas's
  2D context (`canvas_js.go`, see [Browsers](#browsers-internalweb)). They find the fonts, fall
  back to other fonts for what one lacks, shape, run the bidirectional
  algorithm and break lines; font files stay memory mapped and shared with
  every other process. The package assembles their lines into layouts, with
  its own whitespace trimming, ellipsis, alignment, carets and selection, so
  those behave the same everywhere, and caches layouts by their parameters.
  The engines rasterize glyphs too (`IDWriteGlyphRunAnalysis`,
  `CTFontDrawGlyphs`, cairo) into a coverage atlas, color glyphs (emoji)
  into a color atlas, and subpixel glyphs, the coverage of each pixel's
  red, green and blue, into the color atlas as well. Text looks as each
  platform's own toolkit draws it, pixel for pixel, which tests against
  AppKit, Direct2D and GTK 3 labels showed:
  - *Placement.* `System.Glyph` takes a glyph's pen position and places
    it as the engine does: Core Graphics' subpixel quantization, which
    AppKit leaves on, at `min(5, ⌈100/(3·px)⌉)` positions a pixel for
    glyphs of `px` pixels an em, left of the pen; DirectWrite's six
    positions a pixel for ClearType, eight in natural, four in natural
    symmetric and two in downsampled rendering, whole pixels in GDI and
    aliased rendering, the nearest to the pen; GTK's whole pixels, as Pango rounds glyph
    positions. `System.Baseline` snaps baselines likewise: down to the
    next pixel on macOS (Core Graphics floors in its y-up space),
    to the nearest on Windows, up on Linux.
  - *macOS.* Core Text draws with font smoothing, as AppKit does unless
    the user turned it off (`AppleFontSmoothing`, read through
    `NSUserDefaults` as AppKit does: Core Graphics' bitmap contexts ignore
    it), which emboldens glyphs more the lighter their color is, in four
    steps of its luminance: its glyphs are rasterized for each
    `text.Shade` of the text's color. `text.Thick`, for text a
    `ui.Font` thickens (the terminal's `Font.Thicken`, as Ghostty's
    font-thicken), smooths at the strongest step whatever the color and
    the user's setting, and `text.Flat`, for text a `ui.Font` draws
    `Antialiased` (as browsers draw `-webkit-font-smoothing:
    antialiased`), does not smooth; other engines draw both as other
    text.
  - *Windows.* Glyphs take the rendering mode and grid fitting DirectWrite
    recommends for their font and size (`IDWriteFontFace3`'s, which from
    about 1.9 pixels a DIP downsamples natural symmetric rendering, as
    Direct2D draws), ClearType where the system smooths
    fonts with it (`SPI_GETFONTSMOOTHINGTYPE`, with the pixel geometry and
    ClearType level of the system's rendering parameters) and the glyph is
    on an opaque background (the root's, or that of an element around it
    holding all of it that shows, as a pane beside a sidebar over a
    material), and are aliased where the system does not
    smooth fonts. Renderers blend them as Direct2D does, with the gamma
    and enhanced contrasts of the system's rendering parameters
    (`scene.TextParams`, Windows Terminal's reproduction of Direct2D's
    shaders), and half again as much contrast for the families Direct2D
    finds too thin for antialiasing, such as Courier New.
  - *Linux.* Pango takes GTK's font options, which package ui reads from
    `gtk-xft-*` (`platform.Theme.FontRendering`, given to
    `System.SetFontRendering` with the interface font): the desktop's
    antialiasing, subpixel order and hint style, with metrics hinted to
    whole pixels and glyph positions rounded, as a GTK 3 label has them.
    Subpixel antialiasing draws subpixel glyphs.
  - *Decorations.* `System.Decorate` places underlines and strikethroughs
    as the platform does: Core Text's own geometry for AppKit, snapped in
    points, an underline as low and thick as the fonts under it want,
    skipping the ink of descenders, a strikethrough through the middle of
    the x-height, from the text's start to its width rounded up to a
    point; Direct2D's, an underline at the font's offset rounded to whole
    pixels from the snapped baseline, a strikethrough from the baseline as
    laid out, with edges where its eight samples across and down a pixel
    put them; GTK's, at Pango's metrics in whole pixels, the underline
    along a run's ink and advances, the strikethrough along its ink.
  - *Joined glyphs.* Renderers blend each glyph in turn, as Core Graphics
    and Direct2D do, but cairo adds the coverage of a run's glyphs before
    blending it, which differs where antialiased edges of neighbors share
    a pixel. There, on Linux, package ui has the text system join them
    (`System.GlyphRun`): their coverage added and clamped, cached as one
    bitmap.
  Paths drawn with
  `Painter` are masks in the coverage atlas, rasterized by `internal/vec`,
  cached by their shape relative to the pixel grid in quarters of a pixel,
  to which their points move first: a path looks the same whichever path
  of its key drew its mask first.
  So are icons: `internal/svg` parses SVG documents (with `encoding/xml`
  and its own small CSS cascade) into nodes of paths, paints and layers,
  and draws them on the CPU with `internal/vec`, compositing gradients,
  group opacity, clip paths and masks in layers of premultiplied pixels;
  `ui.Icon` keeps the coverage of that drawing for each size, which the
  GPU tints like a glyph, and `ui.Image` the colored pixels, as images
  whose textures it updates in place when only the text color changes.
  An atlas logs the rectangles that change, so renderers upload only those.
  DirectWrite methods taking floats are called through purego, which sets
  the floating point registers that `syscall` leaves alone on ARM64.
  Letter spacing and OpenType features go to each engine's own, so lines
  break with them: `IDWriteTextLayout1`'s character spacing and an
  `IDWriteTypography`, Core Text's tracking and a font copied with feature
  settings, Pango's attributes. So do the spans of rich text, which style
  ranges of runes (`text.Span`): DirectWrite's setters over a range, the
  fonts of a Core Text attributed string's ranges, Pango's attributes
  between byte indices. `Params.Spans` holds them encoded as a string, so
  `Params` stays a key of the layout cache. Laying out reuses what shaping
  throws away: engines take the glyphs, runs and lines `shape` returns from
  a scratch each layout resets (`shapeScratch`), as the layout copies them
  into its lines, which share one allocation; Core Text keeps its buffers,
  the attributes of each font, and calls the functions of each layout
  through `SyscallN` rather than reflection. Colors, underlines and
  strikethroughs of spans only paint, by the rune each glyph starts.
  Text elements built inside a text (`ui/inline.go`) are inline: the
  paragraph takes their text when its `Children` returns, and their
  styles as spans over its own when it lays out. At commit, each gets the
  boxes of its runes in the paragraph's layout (`Selection`), one hit
  each, so the pointer, the focus order, tooltips and menus work as for
  any element; they paint only their focus ring, and assistive
  technology sees those that are more than text, such as links, as nodes
  inside the paragraph's.
  Fontconfig knows no desktop's interface font, so on Linux `system-ui`
  stands for the family of GTK's `gtk-font-name` first
  (`platform.Theme.UIFont`, which package ui gives `SetUIFamily` with each
  change of the appearance).
- **Atlas lifetime.** A mask drawn for the first time goes to the
  transient zone at the bottom of the atlas, which the next frame frees,
  and moves to the lasting zone at the top once another frame draws it, so
  animated shapes never fill the atlas. When an atlas fills up during a
  frame anyway, the engine calls `MakeRoom`, which repacks what the frame
  drew, grows the atlas when that is much of it and forgets the rest, and
  paints the frame again: no frame shows with glyphs missing. The mask
  atlas starts at 256×256, and the color atlas at one transparent texel
  until a color or subpixel glyph draws. Growth accounts for the width
  and height of missing bitmaps as well as their area, so long thin
  paths fit on the repaint too. The first glyph can grow an empty atlas
  immediately, without invalidating anything already painted.
- **Renderers.** `internal/gpu` turns a scene into one instanced quad per
  operation, in batches that share a scissor rectangle and an image, for one
  shader that computes the signed distance to rounded rectangles, Evan
  Wallace's blurred rounded box, gradients, stripes, the dashes of borders,
  atlas coverage and the innermost rounded clip; outer clips are scissor
  rectangles. An effect draws in a batch of its own, with a pipeline of
  its own, which the renderer makes the first time the effect draws; one
  reading its backdrop draws after a backdrop step (`Builder.Backdrops`):
  the renderer ends its pass, reads the backdrop's area of what it drew
  (Metal from the drawable, which is not `framebufferOnly`; Direct3D 11
  and OpenGL from a copy of the area, `CopySubresourceRegion` and
  `glCopyTexSubImage2D`, whose rows go up), averages and blurs it with two
  small pipelines (`down` and `blur`, scissored to the texels they
  compute, reading them with `Load`, `read` or `texelFetch`) into textures
  as large as the frame's largest backdrop, and goes on with the pass, the
  backdrop bound for the effect. Each renderer draws scenes offscreen for
  the tests of packages defining effects (`RenderOffscreen`,
  `NewOffscreen`).
  Near square corners, coverage is exact, the area of a pixel
  inside the box, so lines thinner than a pixel cover as much as they
  should, as text decorations need. A fill's border widths travel in its
  texture rectangle, which fills do not use, and a glyph's gamma ratios
  and contrast in its radii. Each instance is thirteen float4s, including
  its two affine rows. The shader writes a second color, the source's alpha of each channel, and
  renderers blend with it (dual-source blending): it is the color's alpha
  but for subpixel glyphs, whose subpixels cover each channel by its own.
  OpenGL ES without `EXT_blend_func_extended` blends the mean of their
  subpixels instead. A hole is an opaque fill in a batch of its own
  (`Batch.Hole`), blended with a source factor of zero: the destination
  keeps one minus its coverage, as the CPU renderer's `erase` does. Every GPU renderer draws these instances:
  - `internal/gpu/d3d11` with a shader compiled to DXBC ahead of time
    (on Windows, with the system's `d3dcompiler_47.dll`), so apps carry no shader compiler. The
    generated file records the SHA-256 of the source it came from, line
    endings aside (`gpu.SourceSum`), as Metal's does: bytecode older than
    `shader.hlsl` falls back to compiling that with the same DLL, which
    Windows has, and its test fails. Both compilation paths enable IEEE
    strictness with optimization, avoiding a native compiler optimizer crash
    on transformed coverage sampling. It draws into a flip-model swap
    chain on the surface's window, or, over a material, one for
    DirectComposition shown on it, with WARP when no hardware device
    works. At most one frame waits ahead of the screen, not DXGI's three,
    so frames that follow each other, as when scrolling or animating,
    show their input two frames sooner. A frame of a new size is
    presented at once (sync interval 0), not after the frame queued
    before it, and waits until DWM shows it (`DwmFlush`), so a live
    resize shows no stretched frames: the window takes its next size
    with the frame of that size. Resizing the buffers takes DWM longer
    than a frame for a window about as large as a 4K screen, so while
    the window changes size the swap chain has three buffers a quarter
    larger than the window, and shows the window's size of them
    (`SetSourceSize`), until a frame a second after the last change,
    which a timer asks for, gives it two of the window's size again;
  - `internal/gpu/metal` with a shader in Metal Shading Language
    compiled into a Metal library ahead of time (on macOS, with
    Xcode's `metal` tools), which
    spares a first launch the 100 to 150 ms Metal takes to compile the
    source until it has cached it; a library older than `shader.metal`
    falls back to that, and its test fails. It draws into a CAMetalLayer
    it adds to the surface view's layer. The command queue, shaders,
    sampler and placeholder texture are made on demand. A frame that
    changes the layer (its size, scale or pixel format), or shows web
    views, which move in the window's transaction, presents with the Core
    Animation transaction (`presentsWithTransaction`), so a live resize
    shows no stretched frames; the others present as soon as the GPU has
    them (`presentDrawable:`), sparing a pass of AppKit's display cycle
    and a commit of the transaction each: an animation at 120 Hz takes
    about half the main thread's time it would. A frame presented so
    that waited for its drawable, as the display held both until its next
    refresh, has the next skip a refresh (`Behind`, at most once a
    second), or the frames after would wait as long, most of each
    refresh, blocking the main thread. Each frame waits for the GPU to
    finish the last before it updates the textures and the instance
    buffer the last read, so two drawables do rather than the three a
    layer makes while a window animates. Two seconds after the last frame, as the
    driver frees its own memory of frames, a timer shrinks the drawables,
    which frees all but the one shown, until the next frame makes them
    again, and releases GPU instance buffers, atlas/image textures and
    backdrop textures: an idle window keeps one frame of memory. Its
    drawables are not `framebufferOnly`, as effects sample the drawable
    for their backdrops. Colors
    outside the sRGB gamut (`ui.Oklch`, as CSS's `oklch()`) are in a
    table of the scene (`scene.Scene.Wide`), which ops and glyphs point
    into with a 16-bit index (`Op.Wide`, in room the op's other fields
    leave); their `Color`, `Color2` and `BorderColor` hold the nearest
    sRGB colors, found as CSS Color 4 does (`internal/gamut`), which the
    CPU, OpenGL and Direct3D renderers draw, as `gpu.Builder` does unless
    `Wide`. The window host decides when a window draws them
    (`wideHold`): on a screen showing more than sRGB
    (`platform.WideGamutSurface`, `canRepresentDisplayGamut:` on macOS),
    from a frame with some until none came for two seconds, so that a
    blinking caret does not switch formats at every blink; those frames
    are the GPU's, and a window on an sRGB screen keeps drawing the
    nearest colors on the CPU. The Metal renderer (`SetWide`) then
    switches the layer to `RGBA16Float` in the extended sRGB color space,
    which keeps components outside 0 to 1 and which the system shows in
    the display's gamut, and draws with pipelines and backdrop textures
    of that format, made the first time. A uniform of the frame (the z of
    the vertex and fragment globals) tells the shaders: Oklab gradients
    then keep what their mix has outside sRGB, and effects read it as
    `e.wide`, as the glass does to take its tint in extended sRGB
    (`ui.Painter.EffectColor`);
  - `internal/gpu/gl` with the shader in GLSL 3.30 or GLSL ES 3.00, which
    the driver compiles when the renderer starts, since these versions
    have no compiled form every driver takes, and drivers keep what they
    compiled between launches, as Mesa and NVIDIA's do. It draws into
    the framebuffer of the GtkGLArea, which GTK shows. GL functions come
    from libepoxy, as GTK's do. Instances draw from per-instance
    attributes, pointed at each batch's first since OpenGL ES 3.0 has no
    base instance. The area's `create-context` asks GDK for OpenGL 3.3,
    else OpenGL ES 3.0, as GPUs with OpenGL ES alone have, and so does
    the probe below (`glContext`). In browsers the same renderer draws
    with WebGL 2, which is OpenGL ES 3.0 (see
    [Browsers](#browsers-internalweb)): only where its functions come from
    differs, `load_linux.go` or `load_js.go`.

  Linux draws with OpenGL only where it runs on a GPU. A GL context loads
  Mesa for good: some 50 MB of libraries (LLVM, which distributions' Mesa
  links, takes 19 MB as it loads), its threads and heap, as much as the
  rest of a small app; the first window of native UI loads it.
  Before a surface has one, the backend makes one context, on a window
  that never shows, and reads its renderer. A software
  renderer such as Mesa's llvmpipe (in virtual machines, in WSL without
  `GALLIUM_DRIVER=d3d12`) redraws every pixel of every frame on the CPU,
  ten times the CPU renderer's work on an animated page; and once a window
  has a GL context GTK composites it with OpenGL, so the choice is made
  before any surface has one; `MYGO_GPU=1` skips it and draws with OpenGL
  wherever GDK makes a context, from the first frame, as the GUI tests of
  CI do on llvmpipe. The devices answer first where they can: no GtkGLArea
  without NVIDIA's devices or a render node of a DRM driver with 3D
  (simpledrm, bochs, VirtualBox's or Hyper-V's only show what the CPU
  drew), nor on WSL's device without `GALLIUM_DRIVER=d3d12`. A GtkGLArea shows only
  what OpenGL draws into it: when its GL renderer fails all the same, the
  frames drawn in memory go through OpenGL too (`gl.Presenter` uploads
  them and blits them into the area's framebuffer).

  `internal/raster` draws the same scene with the same formulas on the CPU,
  solid spans inside shapes and only the edges of shadows computed, and
  blends ordinary opaque mask glyphs in integers, retaining the general
  path for gradients, corrected/subpixel text and rounded clip edges. It
  redraws only what differs from the last scene (`raster.Renderer`), with
  the effects reading their backdrops that meets and those backdrops, in
  rectangles apart from each other (`addBackdrops`): it is
  the renderer of tests, of `MYGO_GPU=0` and of Linux where OpenGL would not
  run on a GPU, and the one a window falls back to when its GPU renderer
  fails.
  Frames that are not the surface's (a capture before the first frame) are
  kept, not drawn: OpenGL's context is current only in the surface's. An
  area drawn goes through the operations whose bounds (those the damage is
  found from) it meets; a large one is drawn on up to eight cores, in
  bands of 64 rows that each takes in turn, as rows differ in how much
  they draw, each pixel as drawing the area whole gives it. An effect is
  readied (`EffectPixels.Begin`) before its pixels, and one reading its
  backdrop reads pixels other bands draw, so an area is drawn up to each
  effect, its backdrop computed (`backdrop.read`, its rows and columns on
  several cores too), then from the effect to the next, in bands of 16
  rows, as effects are costly and often short. The glass plugin's
  `BenchmarkGlass`, a window of 1360×720 pixels with four panes, takes
  6.4 ms drawn whole on an M5, 22 ms on one core. A whole frame
  of `BenchmarkFrame`'s view at 1844×2044 pixels, built and drawn, takes
  0.7 ms rather than 3.2 on a Ryzen 7 8745HS: memory bandwidth and the
  lower clock of all cores keep it from scaling further.

  On Linux, `RequestFrame` of a surface drawing in memory adds a tick
  callback to the area, which GTK's frame clock runs in its update phase,
  before painting: the frame is drawn there, and `PresentDamage` keeps its
  pixels and asks GTK to repaint what changed (`gtk_widget_queue_draw_area`),
  which the `draw` signal paints from them in the same frame. GTK keeps
  the rest of its buffer, copying it from the last one into a new buffer
  when the compositor still holds the last, and tells the compositor
  only that changed: a digit of the counter copies 24 KB of pixels instead
  of 15 MB for half of a 4K display, in 0.01 ms instead of 1. A draw GTK
  asks for by itself, as a resize does, draws a frame in the `draw`
  signal, and repaints in the next frame what it changed outside GTK's
  clip.

  With a GPU renderer, every frame draws on the GPU, small changes too, so
  a gesture never switches renderers mid-way: on the CPU, a frame
  redrawing glass or a blur that content scrolls under takes 30 to 70 ms
  on a large window, where Metal takes 2 to 3.

  Two seconds after the last frame (`frameIdle`), the host frees the frame
  drawn in memory, as large as the window, which the next frame draws whole,
  and tells the surface it is idle (`platform.IdleSurface`): Linux's calls
  glibc's `malloc_trim`. GTK paints a window it composites with OpenGL into
  an image as large as what it repaints, the whole window for a whole
  frame, and glibc, raising its
  threshold for giving large blocks back as they are freed, kept one or two
  of them, 15 to 30 MB for half of a 4K display. These timers run on the
  main thread through `surface.Conn.Post`.

  A GPU renderer that fails (a driver reset, a GPU unplugged, sleep) is
  released and another made in the same frame (`ui/window.go`): the GPU
  may be back already, and on Windows WARP stands in, presenting through a
  swap chain, as GDI cannot show frames once a flip-model swap chain has
  drawn into a window. Without one, frames are drawn in memory, and a
  frame tries again after a wait that doubles from one second to half a
  minute; a renderer drawing in software in place of a GPU the window had
  tries for it the same way. A window without a GPU at its first frame
  draws in memory for good.
- **Tests.** `ui.Tester` runs views against a host in memory
  (`ui/headless.go`) with the CPU renderer; the fake backend's surface lets
  the core's tests drive content windows through `package mygo`.
  `BenchmarkFrame` in `ui` measures a frame of a large window on the CPU,
  and `BenchmarkListScroll` the frames scrolling a list of a million rows;
  `BenchmarkDiff*` the frames of a window as godiff's, two lists of texts
  and rich texts, unchanged or scrolling, with their allocations
  (`TestDiffFrameAllocs` logs them).
  The GPU renderers' tests draw `gputest.Scene` and compare it with the
  CPU renderer's drawing: on Windows in a hidden window, on macOS into an
  offscreen texture, on Linux into a framebuffer of a context EGL makes
  without a window (Mesa's surfaceless platform, llvmpipe without a GPU),
  once with OpenGL 3.3 and once with OpenGL ES 3.0.

A new widget composes elements (`ui/widgets.go`), keeps what it needs from
frame to frame with `Local` or in `state`, declares its interaction with
element flags (`flagClickable`, `flagFocusable`, …), paints in `Draw`, and
gets a test with `Tester`. A new GPU renderer draws the instances of
`gpu.Builder` with a port of the shader, syncs atlases with
`Atlas.Changes`, is created in `ui/gpu_<os>.go` behind `gpuRenderer`, and
has a test that compares its drawing of `gputest.Scene` with the CPU
renderer's (`gputest.Compare`).

## CLI (`cmd/mygo`)

- Browsers (`web.go`): `build -platform web` (or `js/wasm`) compiles the
  app with `GOOS=js GOARCH=wasm` and the production flags into `<out>/web`,
  with a page of its own (`webPage`: a spinner while the WebAssembly loads)
  and the `wasm_exec.js` of the Go that compiled it, which must match.
  `dev -web` serves the page, the loader and the last build that compiled
  from memory (`webDev`), rebuilds on the watcher's changes as the desktop
  loop does, and tells pages through server-sent events (`/__mygo/events`):
  `reload` after a build, `build-error` with the compiler's output, which
  the page shows over the app until a build compiles.
- `init` renders `cmd/mygo/template`: a Go module and a TypeScript frontend
  built with Vite, side by side at the project root like the examples, draws
  a default icon at `resources/icon.png`, fetches modules, installs the
  JavaScript dependencies with Bun and generates the client. package.json
  runs the CLI from mygo-cli (`bun run dev`, `bun run build`), or with
  `go run github.com/egoist/mygo/cmd/mygo` for `-mygo <checkout>`, whose
  go.mod replaces the module with the checkout. mygo.config.ts imports
  `defineConfig` from mygo-cli, or from the checkout's `packages/cli` by a
  relative path between real locations, and sets `devUrl`, `devCommand` and
  `buildCommand` (the `dev:web` and `build:web` scripts, which run Vite and
  never mygo), `frontendDist` (Vite's `dist`) and `out` (`build`, so the two
  do not meet); `vite.config.ts` pins the dev server to the port of `devUrl`
  and does not watch the development app and builds. Names reach the
  templates escaped for their language (`json`, `printf "%q"`, `html`).
  `-template native` renders `cmd/mygo/template/native` instead (the
  frontend's is `template/web`): an app of native UI with a test of its
  view and a `mygo.json`, whose module has the CLI as a tool (`go get
  -tool`, or a `tool` line beside the `replace` of a checkout), for
  `go tool mygo dev` and `build`; no Bun. Without a frontend in the
  configuration, `bindings` stays empty and the CLI writes no client.
- Both templates install the agent skills embedded under
  `cmd/mygo/template/shared/.agents/skills`. `install-skills [dir]` refreshes
  the same bundled files in an existing directory, without reading project
  configuration or replacing unrelated skills and extra custom files.
- The configuration is `mygo.config.ts`, or `mygo.json` (`config.go`,
  `config_ts.go`). For the former, Bun, else Node.js 22.6 or later (with
  `--experimental-strip-types` before 22.18 and 23.6), runs a loader that
  imports it, awaits its default export or calls it with `{ command }`, and
  writes JSON to a temporary file; either way the JSON goes through the same
  checks. The loader goes to the runtime's standard input, never its
  command line: on Windows the runtime may be a batch file (npm's
  `bun.cmd`), and cmd.exe cuts the command line at a line break. Errors name the file in use. `defineConfig` and the types of the
  configuration come from `packages/cli/index.d.ts`; `TestConfigTypes`
  keeps its interfaces in step with the `Config` struct.
- `generate` builds the app for the host and runs it in generate mode
  (`MYGO_GENERATE`; `RequestSingleInstanceLock` then returns true at once).
- `dev` (`dev.go`, `watch.go`) writes the client as `generate` does (the
  frontend imports it), runs `devCommand` in the project directory, waits
  for `devUrl` to answer and runs a development build, pointed at it
  with `MYGO_DEV_URL` (or at `frontendDist` on disk without `devUrl`). The
  build is packaged like a release: on macOS a bundle named
  "<name> Dev" with identifier "<id>.dev" in `.mygo/dev/<goos>-<goarch>`, so
  bundle-only features (notifications, URL schemes) work and its data stays
  apart from the production app's. Builds are assembled in a staging
  directory and renamed into place, so the running build keeps its files.
  - Builds run with `MYGO_DEV=1`, which the core takes out of its
    environment (`dev.go`): `App.Relaunch` then exits with code 75, which
    has `mygo dev` start the same build again. Production builds ignore it.
    A build counts as started once its process is: one that crashes on
    start is reported as it exits.
  - *Reload.* A change (polling every 250 ms, debounced) rebuilds;
    when the executable, Info.plist and icon are unchanged nothing restarts.
    Otherwise the running build is sent SIGTERM (quit sequence), then
    SIGKILL after 3 s, and the new one starts once it has exited: builds
    never overlap, so the single-instance lock, the web view's profile and
    the app's files are free. Only the app gets the SIGTERM, which lets it
    end the processes it started; those left after it exits are killed. A
    build that fails to compile leaves the old one running; one that fails
    to start leaves none until the next change.
  - *Watching.* Exactly what the build reads, from `go list -deps` after
    every build: the directories of the compiled packages outside GOROOT and
    the module cache (so local `replace` modules too), embedded files,
    go.mod/go.sum, the configuration, the icon and the resources, but not
    the platform directories of other platforms. Frontend sources
    are the dev server's business and never rebuild the app. A build keeps
    the watcher's baseline unless it changed what is watched, so edits made
    during a build trigger another one. On Windows the watcher opens
    directories sharing them for deletion (`openDir`), which `os.Open`
    does not, so that deleting or renaming one it lists does not fail.
  - Quitting the app ends `mygo dev`; a crash waits for the next change.
    Ctrl+C stops the app as a change does; a second one kills it and the
    dev server at once. The watcher reports the paths that changed, which
    the rebuild names. Lines typed on the terminal are shortcuts (`r`
    restarts even an unchanged build, `c`, `q`, `h`); on Unix, mygo
    ignores SIGTTIN, so that reading them in the background fails rather
    than stopping it.
- *Output* (`console.go`), on standard error: a line per step of a
  command. On a terminal (Windows: a console that takes VT sequences once
  `enableVT` asks), the step in progress has a spinner, redrawn in place
  at 12 fps and cut to the width (`termWidth`, autowrap off besides), with
  its elapsed time and what it does: `goBuild` runs `go build -v` and
  shows the import paths it prints, and modules it downloads, keeping
  the other lines, the compiler's errors, for its error; downloads show
  their bytes. Steps nest (signing inside a build). What the programs the
  CLI runs print goes through `labeled` writers, which erase the spinner's
  line, write whole lines with a label (`web │`, `app │`) and draw it
  again; the app writes to an `os.Pipe`, not a writer of `exec.Cmd`,
  whose `Wait` would wait for the web view's processes holding it, and
  commands with writers get a `WaitDelay`. Without a terminal, each step
  has a start line and an end line, without colors unless `FORCE_COLOR`
  (`NO_COLOR` turns them off anyway). `dev` and `build` open with a
  banner whose five exclamation marks are those of MyGO!!!!!, in its
  members' colors, which light up one after the other on a terminal
  (redrawn two lines up with the cursor saved, until a line moves it).
- `build` generates the client, runs `buildCommand`, then compiles each
  target with `-trimpath -ldflags "-s -w -X …production=1"` (`-H=windowsgui`
  on Windows) and `-tags mygo_noinspector`, which leaves the inspector of
  native UI out (unless `MYGO_INSPECTOR=1`), into a staging directory, so a
  failed build keeps the previous artifacts. `frontendDist` is embedded without touching the project
  (`embed.go`): `go build -overlay` adds a generated `mygo_frontend_gen.go`
  to the main package, with `//go:embed all:mygo_frontend` and a call to
  `SetFrontend`, and maps every `frontendDist` file into that virtual
  directory, so the frontend may live anywhere. macOS targets become `.app`
  bundles (Info.plist, `.icns` rendered in pure Go, and with `macos.icon`
  an `Assets.car`, as it is or compiled from an Icon Composer `.icon` by
  `xcrun actool`), signed with
  `macos.signingIdentity` (hardened runtime and timestamp for real
  identities, ad hoc by default); `darwin/universal` combines both
  architectures with a pure-Go fat-binary writer. Linux gets a `.desktop`
  entry and icon. The production flag makes `IsDev` false, which disables
  the web inspector by default.
- *Resources* (`resources.go`), as in quickgui: the contents of the
  project's `resources/` directory, plus the files and directories listed
  in `resources` in the configuration under their base names, are copied into
  `Contents/Resources` of macOS bundles and next to the executable on
  Linux and Windows, by `build` and `dev` alike; apps find them with
  `App.Path(PathResources)` (under `go run`, `./resources`). The platform
  directories of `resources/`, named `<goos>` or `<goos>-<goarch>`
  (`platformDir`; names other tools use, such as `darwin-x64`, are
  errors), ship with the apps of that target only, their entries merged
  with the shared ones: `merger` groups what goes to one path, ignoring
  case, and only descends into directories that several sources share, so
  a whole tree from one source stays one entry. For `darwin/universal`,
  `darwin-arm64` and `darwin-amd64` are walked as pairs whose names must
  match: identical files and links ship once, an arm64 and an x86_64
  Mach-O file become a universal binary (`writeUniversal`) when copied,
  anything else fails. Names starting with a dot are skipped; the entries
  of these directories and listed paths are followed when they are links,
  links inside them are copied as links, permissions are kept. Installed
  paths must be unique ignoring case (listed resources never merge), and
  top-level names must not replace the packaging's own files
  (`AppIcon.icns`, `Assets.car`, the executable, the `.desktop` entry). Code
  among them is signed before the app (`signNestedCode`), since
  `codesign --deep` only covers code directories and notarization rejects
  unsigned code: Mach-O files, then the bundles holding them, deepest
  first, as a bundle's signature seals what it holds (and signing its main
  executable signs the whole bundle). A real identity signs all of it,
  keeping the entitlements of each (`--preserve-metadata`) unless
  `macos.helperEntitlements` names others; ad hoc signing only signs code
  without an `LC_CODE_SIGNATURE`, and the bundles around it, so vendors'
  signatures stay. Windows builds sign the PE images among the resources
  that have no certificate table, with the app's certificate or command.
  The packages of the app may name native libraries in a
  `mygo-plugin.json` (`natives.go`), with a file per platform, its URL and
  its SHA-256, as the terminal plugin names libghostty-vt: `go list -deps`
  of the target finds them, the CLI downloads each once into
  `<user cache>/mygo/natives/<sha256>/`, checking its SHA-256, and installs
  it at the top of the resources like a listed resource, both
  architectures of a universal app combined.
  `resources/icon.png` is the default icon. Each platform's output
  directory is assembled in a staging directory that replaces it whole, so
  removed resources do not linger; development builds on Linux and Windows
  remove what the previous build placed and this one lacks.
- Packages for the other platforms. Windows gets "<name> Setup
  <version>.exe", made with NSIS (`nsis.go`): a per-user install in
  `%LOCALAPPDATA%\Programs\<name>`, where the updater can write, a Start
  menu shortcut, a desktop shortcut that the finish page's second check
  box (MUI's "show readme" one) creates, and an uninstaller registered
  under `HKCU\…\Uninstall\<identifier>`; `/S /D=<dir>` installs
  silently, without the desktop shortcut. The uninstaller runs from a
  copy of itself that nothing waits for, so it removes the app's folder
  last: the folder is gone once the uninstall is done. It tries deleting
  each shortcut again for up to 5 s, as Explorer opens a new shortcut a
  few seconds after it appears, not sharing it for deletion, and `Delete`
  fails meanwhile.
  `makensis` comes from an installation of NSIS or, on Windows, where NSIS
  is rarely installed, from the official zip of the release `nsisRelease`
  pins, which the CLI downloads once, checks against its SHA-256 and
  unpacks into `<user cache>/mygo`, as Tauri does. It tries the copies of
  the zip in `nsisRelease.urls` in order, until one answers with the right
  SHA-256: the asset of this repository's `nsis-<version>` release, then
  SourceForge, where NSIS publishes it. SourceForge's download host has
  been down for hours at a time, and its mirrors then redirect to it, so
  it is not the only source; a host that sends no response headers within
  30 s gives way to the next. Updating NSIS means publishing the copy (see
  [Releasing](https://github.com/egoist/mygo/blob/main/docs/architecture.md#releasing)). Other systems skip the
  installer without NSIS: its zip holds Windows programs only. A signed
  app gets a signed uninstaller too, as with Tauri: `!uninstfinalize`
  (NSIS 3.08 and later) makes makensis run `mygo sign-uninstaller` on the
  uninstaller it generates, before it puts it into the installer, and the
  CLI signs it as it signs the app. The executable and the Windows
  configuration reach it through the environment (`MYGO_SIGNER`,
  `MYGO_SIGN_SETTINGS`), since NSIS reads `$` in the script as its own
  syntax. Linux
  gets a Debian package written in pure Go (`deb.go`), whose maintainer
  is `linux.maintainer`, else the `author` of package.json, else the name
  of the app: the app in `/opt/<name>`, a `/usr/bin` link named
  `linux.command` when there is one, the desktop entry (categories,
  comment, URL schemes; `Exec` is the app's path) and hicolor icons,
  depending on GTK 3 and WebKitGTK 4.1. Packages hold the same files as
  the update archive; apps installed by a package manager do not update
  themselves (`Updater.Enabled` checks that the app can write where it is
  installed). Every Linux build also gets `install.sh`
  (`installscript.go`), a POSIX sh script, the same for every
  architecture, that installs the archive for the user in
  `~/.local/<name>.app` (which updates can replace), links
  `~/.local/bin/<linux.command>` when there is one and no other program
  is there (and removes a `~/.local/bin/<name>` link of its own, which
  earlier scripts made), and registers the app's `<name>.desktop`, with
  absolute `Exec` and `Icon` paths, and `<name>.xml`, the shared-mime-info
  package of the types it defines, under `$XDG_DATA_HOME`. It then warns
  when `ldconfig -p` lists no WebKitGTK (4.1 or 4.0, the libraries
  `internal/linux` loads), with the command that installs it on the
  distribution `/etc/os-release` names, and stays silent when it cannot
  tell (no `ldconfig`, or a cache without libc, as on NixOS or musl).
  `Update.Install` registers them again from the new version
  (`update.RefreshDesktopEntry`, the same rewrite in Go, which a test
  compares with the script's) when the user's entry runs the app it
  updated. It takes the archive it is given, else the one of its
  version next to it, else, with updates, the `url` of the target's
  manifest, read with sed from the indented JSON `mygo build` writes.
  `--uninstall` removes only what points into its install, including the
  URL handler entry the app registers.
- On a macOS host, macOS targets also get "<name> <version> <arch>.dmg",
  or "<name> <version>.dmg" for darwin/universal
  (`dmg.go`): the CLI puts a copy of the app (`ditto --noclone`, since a
  clone of a sparse file takes fewer blocks than its copy on HFS+ will),
  the `/Applications` link, the volume icon and a `.DS_Store` written in
  pure Go (`dsstore.go`, byte-identical to dmgbuild's `ds_store` package)
  that lays out the Finder window in a folder, `hdiutil` creates a writable
  HFS+ image of it, as large as its files need, the CLI sets the volume's
  Finder flags, then `hdiutil convert` compresses it with LZMA.
  No AppleScript or Finder automation is involved, so it works headless and
  in CI. The image is signed with a real identity and, with `macos.notarize`,
  notarized with `notarytool` and stapled.

- `keygen` writes the update signing keys (see Updates above).

Configuration lives in an optional `mygo.config.ts` or `mygo.json`
(`cmd/mygo/config.go`): app
metadata (the icon defaults to `resources/icon.png`), extra `resources`,
the frontend (`devUrl`, `devCommand`, `buildCommand`, `frontendDist`,
`bindings`) and the `macos` section (minimum system version, signing
identity, entitlements of the app and of helpers, DMG title, notarization
profile).

<!-- repository-only:start -->

## Testing

Regenerate checked UI wrappers with `go generate ./ui`. Native library and
shader generation commands are listed in [AGENTS.md](../AGENTS.md).

| suite | command | covers |
|---|---|---|
| core | `go test .` | lifecycle, quit, IPC, channels, events, Eval, protocol, frontend URLs and serving, menus, trust, single instance, dev ready signal (fake backend); `go test -run '^$' -bench .` measures the Go side of IPC and custom schemes |
| generator | `go test ./internal/tsgen` | TS output, json/v2 rules, source lookup; type-checks the output with `tsc` when `bun install` was run |
| CLI | `go test ./cmd/mygo` | config, Info.plist, icons, universal binaries, template, dev launch/ready/stop (the test binary plays the app), watcher and `go list` inputs, the console's output (replayed as a terminal shows it), resources (platform directories, universal pairs, staging, conflicts, dev placement; builds for every OS), frontend embedding (compiles an app with the overlay), `.DS_Store` against a dmgbuild golden file, a real DMG (`hdiutil`); builds and tools are skipped with `-short` |
| runtime | `bun run test` | the injected runtime, `mygo-runtime` and the plugins' packages (against a fake Go side on the real runtime, `plugins/fake-go.ts`) |
| plugins | `go test ./plugins/...` | the fetch plugin against `httptest` servers, the WebSocket client against a test server (ordering, fragments, pings, closing handshakes); the terminal's binding of libghostty-vt (layouts, rendering, encoders, selections), its pseudo-terminals, and its view through `Tester` with real shells: typing, keys as programs ask, input methods, mouse reports, selecting and copying, pasting, scrollback, exits (the library is downloaded, or named by `MYGO_GHOSTTY_VT`; `-short` skips them) |
| native UI | `go test ./ui ./internal/text ./internal/scene ./internal/raster ./internal/svg ./internal/gpu/...` | the GPU renderers against the CPU renderer (Direct3D on Windows, Metal on macOS, OpenGL on Linux); views through `Tester`: input, focus, editing, lists, overlays, frames that fill the glyph atlas; text layout and caret geometry; atlas zones and repacking; the CPU renderer against its formulas; SVG parsing and drawing, with `FuzzParse`; `go test -run '^$' -bench . ./ui` times a frame |
| GUI | `MYGO_E2E=1 go test ./internal/e2e` | the real backend: IPC, channels, protocol, Eval, geometry, capture, menus, window.open, native UI (frames, clicks, input methods replacing typed text, file drops, assistive technology reading and acting; on macOS typing, skipped while an input method is selected, and composing; on Linux with `MYGO_GPU=1`, what OpenGL drew in the GtkGLArea, from the first frame and once a window drawing in memory asks for the GPU); on Windows too (a GitHub Actions `windows-latest` runner has WebView2) |

Browsers run the `js` tests of the renderer, the text engine and package
ui: `GOOS=js GOARCH=wasm go test -c` builds a test binary, which a page
loads with `wasm_exec.js`, its arguments in `go.argv`, and logs to the
console. Under Node (`$(go env GOROOT)/lib/wasm/go_js_wasm_exec`), with no
page, the WebGL tests skip and text falls back to the stub engine.

The XDG variables let the URL scheme test check that GLib opens the scheme
with the handler it registered; without them it writes to temporary
directories, which GLib does not see.

`internal/fake` runs its loop on the goroutine that calls `Run` and records
evaluated scripts, so tests can assert on exactly what the page would
receive. The unit tests run `App.Run` on the main goroutine from `TestMain`,
like a real program.

Linux GUI tests run in a container, since no cgo means the test binary
cross-compiles:

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go test -c -o e2e.test ./internal/e2e
docker run --rm -v "$PWD:/work" -w /work -e MYGO_E2E=1 \
  -e XDG_DATA_HOME=/tmp/xdg-data -e XDG_CONFIG_HOME=/tmp/xdg-config \
  <image with libwebkit2gtk-4.1-0, xvfb, dbus> \
  dbus-run-session -- xvfb-run -a ./e2e.test
```

`TestGlobalShortcutPortal` binds global shortcuts through the desktop portal
as on Wayland, and skips without one. KDE's portal works on X11, so an image
that adds `xdg-desktop-portal`, `xdg-desktop-portal-kde` and `kglobalacceld`
(Plasma 6; `libkf5globalaccel-bin` for Plasma 5, which D-Bus activates) runs
it for real: start `xvfb-run` outside `dbus-run-session` so the portals
D-Bus activates get `DISPLAY`, set `XDG_CURRENT_DESKTOP=KDE`, start
`/usr/lib/*/libexec/kglobalacceld` before the tests, and install
`$XDG_DATA_HOME/applications/e2e.test.desktop` so that the portal accepts
the test binary's app ID. It passes on Debian 13 (portal 1.20, Plasma 6.3)
and Debian 12 (portal 1.16, Plasma 5.27); Ubuntu 24.04 (portal 1.18, Plasma
5.27) binds no shortcuts for any app.

### Benchmarks

`.github/workflows/bench.yml` measures every push to main on GitHub's
macOS, Linux and Windows runners, and the website shows the results at
[/benchmarks](https://mygo.egoist.dev/benchmarks). `scripts/bench.ts`
runs them:

```sh
bun scripts/bench.ts run                 # every package's Go benchmarks, and app sizes
bun scripts/bench.ts run --e2e           # internal/e2e's too, in a desktop session
bun scripts/bench.ts run ./ui --count 3  # one package, fewer runs
```

It finds the packages with benchmarks and runs them one at a time
(`-p 1`), each benchmark six times for half a second (`-count 6
-benchtime 500ms`), and keeps the median of each metric: `ns/op`,
`B/op`, `allocs/op` and those a benchmark reports; `MB/s` follows from
`ns/op`. It also builds `examples/hello` and `examples/counter-native` as
`mygo build` builds a release, without resources, and records their size,
and with `--e2e` their memory six seconds after they start
(`go run ./internal/idlemem app...`, as many times as `--count`): an
app's process with the processes it runs, and `hello`'s own as
`hello/app`. Memory is what each platform shows: the physical footprint
on macOS, the proportional set size on Linux, the private working set on
Windows. On macOS a bare executable's WebKit processes belong to the
process responsible for it (the terminal, or the runner), so idlemem
counts those that started with the app and share its responsible
process. GitHub's runners are virtual machines with a 1024×768
display at scale 1: there, counter-native's footprint is 22 MB on macOS,
where an M5 MacBook with a Retina and a 4K display gives 37 MB, AppKit's
own allocations taking half (`footprint -p <pid>` breaks it down); on
Windows its private working set is 14 MB, of a working set of 39 MB with
the pages it shares, and 62 MB committed. Compare a series with itself.
`--out` writes the results as JSON, with the commit and the runner's CPU.
The benchmarks of `internal/e2e` time the real backend: a page calling Go
(`PageCall`, `PageCallItems`), streaming (`PageChannel`), receiving events
(`PageEvent`), fetching from the app's scheme (`PageFetch1MB`), and
windows opening until their DOM is ready or their native UI has built a
frame (`WindowOpen`, `ContentWindowOpen`).

The workflow's last job merges the results into the `benchmarks` branch
(`bun scripts/bench.ts merge`): `history/<yyyy-mm-dd>.json` keeps every
commit measured that day, a line each, and `latest.json` the last 300
commits by series, which the page fetches from raw.githubusercontent.com
when it opens, so results show without a deploy. Its commits say
`[skip ci]`, which Cloudflare's builds of the website honor too, and a
push that races another run's merges again onto it. Started by hand with a
`ref`, the workflow measures that commit with the current script, to fill
in history.

The page finds the commits that moved each series (`findSteps` in
`website/src/lib/benchmarks.ts`): a commit whose value, and the median of
the five from it, differ from the median of the five before by more than
the series' noise, five median absolute deviations of the twenty before
(their range while fewer than fifteen; at least 5% for time, 1% for
memory and allocations, 0.2% for sizes, and 16 bytes or an allocation).
After a step, the values before it are left behind. A series that moves
needs ten values before a step and two commits after it; a steady one,
as allocations mostly are, five and none. Charts mark steps with ▲ and ▼,
the summary lists the commits shown that moved results, and a card
compares the last value with the first ones shown, so that a regression
stays in sight after later commits. GitHub's runners get one of several
CPUs from run to run (Windows' EPYC 9V74 runs some benchmarks 40% faster
than the 7763), so timings are compared only among commits measured on
the same CPU, and their charts draw the last commit's CPU as the line and
others in gray.

Benchmarks are described on the page by their doc comments, which the
site reads when it is built: start them with the benchmark's name.

## Releasing

The Go module, the CLI and the npm packages share one version:

```sh
bun scripts/version.ts 0.2.0   # mygo.Version, the CLI, every package.json and bun.lock
git commit -am "Release 0.2.0" && git tag v0.2.0 && git push origin main v0.2.0
```

The tag starts `.github/workflows/release.yml`, which checks that the tag
matches the versions (`bun scripts/version.ts --check`) and runs
`bun scripts/publish.ts --provenance`: it publishes mygo-runtime and the
plugins' packages while it builds the CLI's binaries, each platform package
once its binary is built, then mygo-cli once npm serves them to package
managers, skipping versions already on npm, prereleases under the `next`
dist-tag. npm serves a package minutes after accepting it ("Your package
is being processed"), and a package manager installs mygo-cli without a platform
package it cannot fetch; bun then keeps that package out while its lockfile
lacks it, even with `--force`. The workflow then asks the Go module proxy
for the tag and creates the GitHub release.

It does not run the tests again: `ci.yml` runs them on every push to main,
for every platform, GUI tests included, so tag a commit whose CI passed.
Its Windows job keeps the NSIS that the installer tests download in the
Actions cache, under a key of `cmd/mygo/nsis.go`.

The NSIS that `mygo build` downloads (`nsisRelease` in `cmd/mygo/nsis.go`)
has a copy on a release of this repository, which a new version of NSIS
needs before a MyGo release that pins it. Download the official zip,
check that its SHA-256 is the one `nsisRelease` pins, and publish it,
without making the release the latest:

```sh
curl -fLO https://downloads.sourceforge.net/project/nsis/NSIS%203/3.13/nsis-3.13.zip
sha256sum nsis-3.13.zip
gh release create nsis-3.13 nsis-3.13.zip --latest=false --title "NSIS 3.13" \
  --notes "The official NSIS 3.13 zip, which mygo build downloads on Windows."
```

The tag starts no workflow, and the Go module proxy ignores it.

npm authenticates the workflow as a trusted publisher of each package
(`release.yml` of this repository, set in the package's settings on npm),
which npm allows only for packages that exist: the first release uses an
`NPM_TOKEN` secret of the repository instead. `bun scripts/publish.ts
--dry-run` shows what would be published.

## Adding a feature

1. **Design the public API first** in `package mygo`: typed, goroutine-safe,
   documented, with sensible zero values. Keep policy (defaults, validation,
   state machines) here.
2. **Extend `internal/platform`** with the smallest mechanism the backends
   need: main-thread only, synchronous, or with a callback that runs on the
   main thread exactly once.
3. **Implement it in every backend**: `darwin`, `linux`, `windows`, `web`,
   `fake` and `unsupported` (return `platform.ErrUnsupported` or a zero
   value).
   Create callbacks once, never per call.
4. **Wire the core**: hop with `onMain`/`onMainValue`, or `postMain` + `await`
   + `deliver` for asynchronous native results. Start with `needsApp` when
   the call needs the running app, or keep a setting until `Run` applies it
   (see the threading model).
5. **Test**: unit test through `internal/fake`, a GUI test in `internal/e2e`
   when behavior depends on the toolkit, and run the Linux GUI tests in the
   container.
6. **If the page runtime changes**, edit `packages/bridge` or
   `packages/runtime`, run `bun run test`, `bun run typecheck` and
   `bun run build`, and commit `internal/bridge/bridge.js`. If the generated client changes, update
   `internal/tsgen/generate.go` and its tests.
7. **Document** the behavior in the Go doc comments and platform
   differences in the README.

<!-- repository-only:end -->

## Platform differences

| feature | macOS | Linux | Windows |
|---|---|---|---|
| menu bar | application menu bar, default menu installed | per-window GTK menu bar, none by default | per-window Win32 menu bar, none by default |
| auto-hide menu bar | ignored | the bar widget hides; `can-activate-accel` keeps its shortcuts; Alt alone or F10 show it and open its first menu until it deactivates | the menu is attached only for the `SC_KEYMENU` menu loop that Alt alone or F10 start; shortcuts come from the webview |
| tray | NSStatusItem, click events | AppIndicator (menu only, no click events) | notification area icon, click events |
| global shortcuts | Carbon hot keys | X11: `XGrabKey` on the root window (with Caps/Num Lock variants), key presses from a GDK filter. Wayland: the XDG `GlobalShortcuts` portal (see [Linux](#linux-internallinux)) | `RegisterHotKey` |
| notifications | UserNotifications, packaged apps only; `Group` is the `threadIdentifier`; the delegate is attached at launch, for the click that launched the app | org.freedesktop.Notifications over D-Bus; no `Group` | notification-area balloons (toasts); no `Group` |
| notification removal | `removeDeliveredNotificationsWithIdentifiers:`; `ClearNotifications` removes all, earlier runs' too | `CloseNotification` on the bus, for those of this run | hides the balloon, which goes away by itself anyway |
| vibrancy | all materials, behind pages and native UI | ignored | Windows 11 22H2 Mica, Acrylic, Tabbed, behind pages and native UI, in windows created with a material, which have no menu bar |
| traffic lights, Dock | yes | ignored | ignored |
| hidden title bar | AppKit's traffic lights over a full-size content view | GTK's title buttons in header bars over the page, per `gtk-decoration-layout`; none where the Wayland compositor decorates windows | caption buttons drawn in a layered child window, through DirectComposition over a material; snap layouts; a top edge that resizes |
| progress bar | Dock tile content view (NSBoxes: NSProgressIndicator does not draw there), app-wide | Unity launcher API over D-Bus (`com.canonical.Unity.LauncherEntry`), app-wide | `ITaskbarList3`, per window |
| badge count | Dock tile label | Unity launcher API count | not shown |
| skip taskbar | ignored | skip-taskbar hint | `ITaskbarList::DeleteTab` (the window style is untouched) |
| FlashFrame | informational Dock bounce | urgency hint | `FlashWindowEx` until focused |
| visible on all workspaces | `NSWindowCollectionBehaviorCanJoinAllSpaces` | `gtk_window_stick` | ignored |
| window icon | ignored | `gtk_window_set_icon` | `WM_SETICON` at the window's DPI |
| URL schemes | Info.plist (`urlSchemes`); `RegisterURLScheme` makes the app the default handler | desktop entry + `mimeapps.list` | `HKCU\Software\Classes` |
| downloads | `shouldPerformDownload`, non-displayable or attachment responses → `WKDownload` delegate | `download-started` / `decide-destination` on the web context; response policy for attachments | `DownloadStarting` (`ICoreWebView2_4`), replacing WebView2's download UI |
| ClearBrowsingData | default `WKWebsiteDataStore`, all types | the web context's website data manager | the WebView2 profile's `ClearBrowsingDataAll` (needs a window) |
| permissions | `WKUIDelegate` media capture (camera, microphone) | `permission-request` (camera, microphone, geolocation, notifications) | `PermissionRequested` (the same four; WebView2 asks about others) |
| file associations | `CFBundleDocumentTypes`; files arrive with `application:openURLs:` | desktop entry `MimeType` (`%U`), a shared-mime-info package in the .deb for types the app defines | ProgIDs and `OpenWithProgids` written by the installer |
| Dock menu | `applicationDockMenu:` | ignored | ignored |
| PrintToPDF | `printOperationWithPrintInfo:` save job (`NSJobSavingURL`), fit to width | `WebKitPrintOperation` to GTK's "Print to File" | DevTools `Page.printToPDF` |
| power events | NSWorkspace sleep/wake, `com.apple.screenIsLocked` distributed notifications | logind `PrepareForSleep` (system bus), screen saver `ActiveChanged` (GNOME, freedesktop) | `WM_POWERBROADCAST`, `WM_WTSSESSION_CHANGE` |
| KeepAwake | `NSProcessInfo` activity (shows in `pmset -g assertions`) | XDG portal `Inhibit`, else `org.freedesktop.ScreenSaver.Inhibit` | `PowerCreateRequest` |
| IsOnBattery, IdleTime | IOKit power sources, `CGEventSourceSecondsSinceLastEventType` | `/sys/class/power_supply`; Mutter idle monitor or `GetSessionIdleTime` | `GetSystemPowerStatus`, `GetLastInputInfo` |
| window position | honored | ignored by Wayland compositors | honored |
| resize borders without a title bar | the window's own | the outer 5 px of the page | invisible, outside the window; along the top of a hidden title bar, a child window |
| content protection, click-through | yes | ignored | yes |
| flyouts | borderless nonactivating `NSPanel` child window, or an `NSPopover`; placed in the visible frame | `GTK_WINDOW_POPUP` (override-redirect, placed in the work area, on X11; `xdg_popup` with `gdk_window_move_to_rect` on Wayland); focusable ones grab the seat | owned `WS_POPUP` + `WS_EX_TOOLWINDOW`, `WS_EX_NOACTIVATE` unless focusable, `CS_DROPSHADOW` for a shadow, with corners rounded by DWM on Windows 11 |
| custom scheme origin | `<scheme>://localhost` | `<scheme>://localhost` | `http://<scheme>.localhost` (the page's `location`) |
| `LoadHTML` documents | a reload loads the base URL (`about:blank` without one), so the core cancels it (`Navigation.Reload`) and loads the HTML again; not in the history | as macOS | as macOS, but in the history: going back to one with a base URL loads that URL |
| window.open | keeps the opener | independent window | independent window |
| native UI surface | layer-backed NSView, frames from `CADisplayLink` (a timer at the display's rate before macOS 14), input methods through NSTextInputClient | GtkGLArea (GtkDrawingArea without a GPU), GtkIMMulticontext | `MyGoSurface` child window, IMM32 |
| native UI data drags | NSDraggingSession, NSPasteboardItemDataProvider, NSDraggingDestination | GTK drag contexts, MIME selections, text/uri-list | OLE IDataObject, IDropSource, IDropTarget, Shell drag images |
| native UI accessibility | `NSAccessibilityElement` subclasses | ATK objects (GObject types registered through purego), bridged to AT-SPI by GTK | UI Automation fragments (COM objects; assembly thunks for the methods taking doubles) |
| native UI rendering | Metal, into a CAMetalLayer presenting as soon as drawn, or with the Core Animation transaction for frames that resize it or show web views | OpenGL 3.3 or ES 3.0 in the GtkGLArea's render signal; on the CPU, painted with cairo, where OpenGL runs on the CPU | Direct3D 11 (WARP without a GPU), flip-model swap chain |
| web views in native UI | WKWebView in a clip view under the surface; `hitTest:` lets the pointer through | WebKitWebView in a `GtkLayout` under the area in a `GtkOverlay`; an input shape lets the pointer through | WebView2 composition controller in a DirectComposition target under the frames'; the surface sends it the mouse |
| audio output | Audio Queue on the default output | PulseAudio simple API (PipeWire too), else ALSA `default` | WASAPI shared mode, reopened when the default output changes |
