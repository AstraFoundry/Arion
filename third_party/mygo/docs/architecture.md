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
  polls: all work is driven by native events or explicit wake-ups. A
  window of native UI whose content settled asks for no frame and arms no
  timer (`TestSettledWindowSleeps`): the display link pauses, Go's
  scheduler sleeps, and the counter example idles at about 0.005% of a
  core on macOS (an AppKit window, 0.002%) and wakes no thread under X11.
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
  platforms, 32-bit Windows and Linux among them, the `internal/unsupported`
  backend makes `App.Run` fail with a clear error while everything still
  compiles, so every file of a backend, and of the text engine it pairs
  with, names the backend's architectures in its build constraint.
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
├── cookies.go          Cookies: the pages' cookies, checked, keyed by name, domain and path, filtered by CookieFilter
├── session.go          Session: website data stores chosen per page (PageOptions.Session), ClearData by kind
├── recent_docs.go jumplist.go: recent documents (Open Recent and the system's lists), the jump list
├── dialog.go modules.go clipboard.go: shell, clipboard, screen, theme, tray, shortcuts, notifications
├── watch.go watch_scan.go: WatchFiles, and what a watch saw of its directory
├── document*.go        NewDocuments: documents, their saving, recovery copies, watches, menus
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
│   ├── schemehost/     maps custom scheme URLs to the http://*.localhost URLs WebView2 serves them at
│   ├── web/            browsers (js/wasm): native UI on canvases, through syscall/js
│   ├── unsupported/    stub for other platforms
│   ├── fake/           in-memory backend for unit tests
│   ├── automationcli/  mygo-automation, the command line client of plugins/automation's socket
│   ├── watchdriver/    the goroutine that runs a backend's file watcher
│   ├── bridge/         embeds bridge.js, built from packages/bridge
│   ├── surface/        the connection between a window and its native UI
│   ├── scene/          display lists and glyph atlases, what renderers draw
│   ├── text/           fonts, shaping, line breaking, editing, glyph rasterization
│   ├── rope/           the persistent tree of text chunks text buffers keep
│   ├── diff/           line and word diffs: histogram, Myers, git's indent heuristic
│   ├── vec/            the coverage rasterizer of glyphs and paths
│   ├── raster/         the CPU renderer of scenes
│   ├── gpu/            the instances GPU renderers draw, and gputest/ for their tests
│   ├── gpu/d3d11/      the Direct3D 11 renderer of scenes, and device/ for its devices
│   ├── gpu/metal/      the Metal renderer of scenes
│   ├── gpu/gl/         the OpenGL and WebGL 2 renderer of scenes
│   ├── shader/         the compiler of shaders in Go's syntax, forked from Ebitengine's Kage
│   ├── shaderir/       its IR; glsl/, hlsl/ and msl/ write it for the GPU renderers, interp/ runs it on the CPU
│   ├── svg/            SVG parsing and drawing, for icons and images
│   ├── tsgen/          TypeScript client generator
│   ├── i18ngen/        checks catalogs of messages, writes their Go and TypeScript functions
│   ├── i18nlink/       what package i18n gives MyGo's own catalog only: a catalog that links no dates
│   ├── l10n/           MyGo's own strings: the catalog of the controls and menu roles
│   ├── accelerator/    parses "CmdOrCtrl+Shift+K"
│   ├── menuicon/       menu items' PNG icons as premultiplied, scaled pixels (Windows)
│   ├── update/         update manifests, signatures, archives and delta updates
│   ├── idlemem/        measures apps' memory once idle, for the benchmarks
│   ├── inputlatency/   measures how long input takes to reach the screen in native UI (macOS)
│   └── e2e/            GUI tests and benchmarks against the real backend (MYGO_E2E=1)
├── packages/           Bun workspace (with the examples' frontends):
│   ├── bridge/         the runtime injected into pages (→ internal/bridge/bridge.js)
│   ├── runtime/        mygo-runtime, the npm package apps and generated clients import
│   └── cli/            mygo-cli, the npm package of the CLI, and in npm/ its
│                       per-platform binary packages
├── plugins/            official plugins, each a Go package and its npm
│                       package (@mygo-plugins/<name>) side by side: fetch,
│                       websocket, sqlite, watch; and Go only: updater, the
│                       update window, a web page or native UI (updater/native),
│                       terminal, a view of native UI running programs with
│                       libghostty-vt, glass, game, what games need
│                       beyond what apps do (game/gamepad), loading,
│                       loading.dev's indicators for native UI,
│                       automation, a socket other processes drive the
│                       app through, for end-to-end tests, with its
│                       command line client for agents
│                       (automation/cmd/mygo-automation), and
│                       permissionflow, the system's permissions and a
│                       guide to System Settings, after PermissionFlow
├── ui/                 native UI: views, layout, widgets, text editing, Tester;
│                       ui/chart: charts without a look, on ui's public API;
│                       ui/editor: the code editor and the diff viewer;
│                       ui/shader: shaders in Go's syntax, on every renderer
├── i18n/               languages: matching, CLDR plural rules and parent locales,
│                       formatting of numbers and dates (generated by
│                       i18n/internal/gen), ICU MessageFormat, catalogs
├── transfer/           immutable data items, representations, lazy providers and drag effects
├── netproxy/           the proxy of the system's settings, for Go's HTTP clients
├── audio/              sound: players, the mixer, the output of each platform, WAV;
│                       audio/vorbis and audio/mp3 decode Ogg Vorbis and MP3
├── cmd/mygo/           the CLI: init, generate, dev, build, doctor
├── examples/           hello, todo, frameless, native; counter-native, vibrancy,
│                       effort-slider, flyout, menubar-flyout, piano, farm
│                       and mayoi (games), code-editor and gallery (native
│                       UI), timeline (motion on one playhead); webview (web
│                       views in native UI); launcher (a Spotlight-like panel,
│                       native UI); wind (a texture of glyphs lit by a
│                       wind, two shaders of ui/shader); loading (the
│                       loading plugin's indicators); video (a GIF
│                       looping behind a window's content); permissionflow
│                       (the permissions, asked for and guided to)
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
   through `AppOptions`, `SetMenu` once ready, `Theme.SetSource`,
   `Theme.SetGTKCSS` and `Dock.SetMenu` right after `Init`), or starts
   with `needsApp`, which
   panics on the main thread before `Run` with the name of the call,
   rather than letting it crash on Linux, hang in `await` or answer wrong
   on macOS. Only calls that work before `Init` on every backend (`Locale`,
   packaging info, login items, URL schemes, `IsOnBattery`) do neither.
2. **Platform methods and handler callbacks run on the main thread**, except
   `Clipboard.WithFileAccess`: this synchronous worker-safe operation touches
   no clipboard state and holds a local autorelease pool and file scopes only
   for its callback. The backend never needs locks for its own clipboard state.
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
     expected and safe. A nested loop started inside a run loop observer
     is not: CFRunLoop never runs that observer again meanwhile, and in
     the one AppKit displays views and Core Animation commits in, no
     window, panel or sheet would show. So native UI's frames, whose
     handlers (`Clicked`) may show a dialog, draw outside it on macOS,
     but those AppKit asks for, as a resize's (`drawSoon` in
     `internal/darwin/surface.go`).
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

File watches (`watch.go`) don't use the main thread: each runs on a
goroutine of its own, and the backend's watcher on another (see
[Watching files](#watching-files-watchgo)).

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
  `mouseDown:` for drag regions; `window_drag.go` follows the press with a
  local event monitor: `StartDrag` asks the window server
  (`performWindowDragWithEvent:`) while the pointer has not moved, and
  moves the window itself when it has, or when the window server did not
  begin, which `windowWillMove:` tells, not the window's frame: it lags),
  `MyGoWindowDelegate` (window, navigation,
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
  on. Liquid Glass is an `NSGlassEffectView` there instead, which shows
  what is under it in the window rather than behind it: an opaque window
  is made clear while it shows one, and so is the content view a rounded
  window's color went to (`clearForGlass`), the color given back once the
  glass goes.
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
- On Wayland, GTK decorates a frameless window, or one with a hidden title
  bar, itself (`useCSD`): `gtk_window_set_titlebar` with a `GtkBox` that
  never shows gives it client-side decorations without a title bar, so GTK
  draws its shadow and border around the page, resizes the window from
  them, and drops them while the window is maximized or in full screen.
  GTK sizes the window without them (`gtk_window_get_size`, the default
  size, the geometry hints), so the page keeps the size asked for, and
  Wayland has no positions they would shift. A window with `DisableShadow`,
  a transparent one (the decorations would outline the rectangle around a
  page that shows its own shape; GTK draws them on an app-paintable window
  too), panels and flyouts, a hidden title bar on a compositor that
  decorates windows itself, and every window on X11 stay undecorated
  (`gtk_window_set_decorated`): an X11 window that decorates itself has
  its shadow inside its X window, which the frame extents that `Bounds`
  counts do not tell, and Mutter puts its own shadow around opaque
  undecorated windows there. GTK styles the decorations once, so the
  shadow comes with the window: `SetHasShadow` changes nothing on Linux.
- GTK gives windows without decorations no resize borders, so the outer
  5 px of the page of an undecorated frameless window, or one with a
  hidden title bar, resize it (16 px along the edges from a corner resize the corner). The web view's `motion-notify-event` shows
  a resize cursor there, keeping WebKit's cursor to restore, and its
  `button-press-event` calls `gtk_window_begin_resize_drag` with the press,
  as a Wayland compositor requires; neither event reaches WebKit. Nothing
  resizes a maximized or full screen window, and a tiled one only resizes
  at the edges the window manager allows, as with GTK's own decorations.
- `gtk_window_realize` announces server-side decorations to a Wayland
  compositor that speaks `org_kde_kwin_server_decoration` (KWin, COSMIC,
  Sway) for every window GTK does not decorate itself, one without
  decorations too, and the compositor draws its title bar on it. A
  frameless window GTK does not decorate itself is realized before it
  shows and announced client-side
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
- The app's GTK style sheet (`Theme.SetGTKCSS`, `style.go`) is a
  `GtkCssProvider` of its own for the default screen, at
  `GTK_STYLE_PROVIDER_PRIORITY_USER`: GTK 3 looks providers up from the
  highest priority, and among equal ones from the last added, so it wins
  over the theme, MyGo's own provider (600) and the user's `gtk.css`.
  Each sheet loads into a new provider, which replaces the one before only
  once it parsed (`gtk_css_provider_load_from_data` with a `GError`, which
  takes GTK's first parse error instead of a warning). The core reaches it
  through `platform.StyleBackend`, which only Linux and the fake backend
  implement, so apps that never set a sheet do not link it; one set on the
  main goroutine before `Run` waits in `pendingGTKCSS` until `Init`.
- `WEBKIT_DISABLE_DMABUF_RENDERER=1` is set before GTK starts unless the
  user set it, which avoids blank webviews on NVIDIA drivers, VMs and
  containers, at the cost of drawing pages on the CPU. `App.SetDMABUFRenderer`
  (`AppOptions.DMABUFRenderer`) leaves it unset: always (`on`), or where
  `dmabufSafe` (`dmabuf.go`) finds Wayland and a GPU of Mesa's outside
  containers and virtual machines (`auto`, whose call links that probe in
  with `LinkDMABUFProbe`).
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

#### Why not GTK 4

The backend uses GTK 3 because a GTK 4 backend would do less, run on
fewer systems and give apps nothing they lack on GTK 3, at the cost of
rewriting most of `internal/linux`.

**It would do less.** Some of what MyGo's API promises has no GTK 4
counterpart, not even a renamed one:

- **Window placement.** GTK 4 has no `gtk_window_move`,
  `gtk_window_get_position`, `gtk_window_set_position` or
  `gdk_window_get_frame_extents`, on X11 too: an app cannot put a window
  anywhere or ask where it is. `Bounds`, `SetBounds`, `Center` and the
  frame handling above rest on them, and flyouts are anchored to a
  rectangle with `gdk_window_move_to_rect`, of a `GdkWindow` GTK 4 has no
  more.
- **Window hints.** `gtk_window_set_keep_above`,
  `gtk_window_set_skip_taskbar_hint` and `gtk_window_set_type_hint` are
  gone: no always on top, no windows kept out of the taskbar, no utility
  or popup menu windows for flyouts and panels.
- **Tray icons.** AppIndicator, which draws them, takes a `GtkMenu`,
  which GTK 4 removed with `GtkMenuBar`, and has no GTK 4 version: the
  backend would serve StatusNotifierItem and dbusmenu over D-Bus itself.

Only X11 could get placement and hints back, through Xlib and EWMH beside
GTK (`XMoveWindow`, `_NET_WM_STATE_ABOVE`, `_NET_WM_STATE_SKIP_TASKBAR`),
and GTK 4.18 deprecated its X11 backend, to be removed in GTK 5: a
GTK 4 backend would rest those features on what GTK is dropping. Wayland is no
reason to move either: no GTK version places windows there.

**It would run on fewer systems.** GTK 4 replaced ATK, which native UI's
accessible tree implements (`surface_access.go`) and GTK 3 bridges to
AT-SPI, with `GtkAccessible`, and only GTK 4.14 lets an app expose text it
draws itself (`GtkAccessibleText`), which screen readers need to read
native UI's text fields and editors. Ubuntu 22.04 ships GTK 4.6 and
Debian 12 GTK 4.8, both still supported, where GTK 3 gives native UI the
same accessibility as everywhere else. GTK 4 also means WebKitGTK 6.0,
which replaces 4.1 and 4.0 rather than adding to them, so apps could no
longer fall back to the WebKitGTK 4.0 of older distributions.

**It would give apps nothing.** A MyGo window shows a web page or native
UI, which MyGo draws itself; GTK's own widgets show only in title bar
buttons, menus and dialogs. WebKitGTK 6.0 is the same WebKit as 4.1,
built from the same releases, so pages render and behave the same.

**And it would cost the rewrite.** Beyond the above, GTK 4 replaced the
`button-press-event`, `motion-notify-event`, `key-press-event` and
`configure-event` signals with event controllers and made `GdkEvent`
opaque, where the backend reads its fields at fixed offsets (`field[T]`);
removed `gtk_widget_event`, which hands Tab to the web view, and
`GtkLayout`, which holds the web views over native UI; turned
`GtkClipboard` and `gtk_drag_*` into `GdkClipboard` and
`GtkDragSource`/`GtkDropTarget`; made menus `GMenuModel`s in popovers; and
removed `gtk_dialog_run` and `gtk_native_dialog_run`, which run the main
loop until a message or file dialog is answered so that the call that
opened it returns the answer. The window manager quirks the bounds
handling above works around would have to be found again on GTK 4, across
the same X11 window managers and Wayland compositors.

GTK 3 is still maintained, installed on every desktop MyGo targets, and
what WebKitGTK 4.1, AppIndicator and gtk-layer-shell are built against.
The trade turns once WebKitGTK stops releasing 4.1, GTK 3 leaves the
distributions apps ship to, or those that are still supported all have
GTK 4.14: then a GTK 4 backend loses nothing GTK 3 still gives, and the
features GTK 4 lacks become X11-only code beside it.

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
  User data lives in `%LOCALAPPDATA%\<name>\WebView2`. The direct path applies
  `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` and `WEBVIEW2_USER_DATA_FOLDER` as
  the loader does; it does not read the channel variables or policy keys.
- **Custom schemes** load from `http://<scheme>.localhost/`, and their
  other hosts from `http://<host>.<scheme>.localhost/` (`internal/schemehost`:
  hosts of DNS labels, lower-cased, without a port), which WebView2
  lets the app answer through `WebResourceRequested`. Chromium treats
  `*.localhost` as a secure origin, and unlike `https`, `http` blocks no
  mixed content, so pages reach `ws://` and `http://` URLs as with the
  custom schemes of the other backends. The backend maps URLs both ways, so
  the core only sees `<scheme>://<host>/`.
  Responses are buffered: WebView2 takes a whole stream. `LoadHTML` with a
  base URL serves the document from that URL the same way.
- **Eval** goes through the DevTools protocol (`Runtime.evaluate` with
  `awaitPromise`), which reports syntax errors and ignores the page's CSP.
  Edit roles run `document.execCommand` with a user gesture, except Paste:
  `Input.dispatchKeyEvent` carries Chromium's `paste` command on an
  `Unidentified` raw key-down, followed by key-up once it completes. The
  page sees these events without modifiers, avoiding its Ctrl+V handlers,
  and a paste event with the clipboard's formats. The controller gets
  focus first, after the menu closes. Paste and Match Style still inserts
  the clipboard's text with `Input.insertText`.
- Scripts injected at document creation run in every frame, so the
  main-frame-only ones (the bridge) are wrapped in `window === window.top`.
  Iframes get no `chrome.webview` handler.
- **ABI.** On x64, structs over 8 bytes are passed by reference and Go
  mirrors the first integer arguments into the XMM registers, so doubles can
  be passed as bits. On ARM64, 16-byte structs travel in two registers and
  `syscall` cannot pass floats, so zoom falls back to CSS (`abi_*.go`);
  `comCallFloat` calls methods taking a double through purego instead,
  as `put_Expires` of cookies. Go callbacks read only integer registers:
  the two methods of UI Automation that take doubles enter through thunks
  in assembly (`uia_*.s`), which move the doubles' bits to the integer
  registers of their positions, touching only registers a call may
  change, and jump to the callbacks.
- DPI: the process is per-monitor aware (v2); the backend converts sizes
  between pixels and DIPs with the window's or monitor's DPI, and points
  and rectangles of the screen (windows' bounds, the displays, the cursor,
  tray icons) through one space of DIPs in which displays never overlap
  and a point names one display (`dipSpace`, in `dips.go`, which builds
  on every platform for its tests): a display's DIPs start at its
  top-left pixel scaled by the smallest scale of the displays, as Qt
  anchors them at that pixel, and go on in its own scale. The primary
  display's DIPs, and those of displays that all have one scale, are the
  screen's pixels at that scale. A window takes the scale of the display
  it is the most on, whose DPI Windows gives it; it is created on its
  display, at its rectangle, and `place` moves it, keeping its rectangle
  in `WM_DPICHANGED`, whose suggested one would scale the size, already
  in the new display's scale, a second time (#1039). Frameless windows, and
  those with a hidden title bar, drop the caption in `WM_NCCALCSIZE` but
  keep the side and bottom borders, which Windows 10+ draws invisibly
  outside the window, so they still resize. On Windows 11 a hidden title
  bar keeps one pixel of its top, under the window's border, without which
  the maximize button opens no snap layouts while the window is not
  maximized. Windows 10, which has no snap layouts, keeps none: over a
  window that keeps any of its top it draws its whole title bar. A
  resizable frameless window on Windows 10 takes the visible pixel of its
  borders into its client area too, where the content covers the line DWM
  draws there, white whatever the content (#916), and its frame is not
  extended for the shadow, which its invisible borders cast: extended, it
  shows as a white line along the top. Other windows without a caption
  extend it by a pixel, without which a window with no borders has no
  shadow. The build comes from `RtlGetVersion`, which no manifest changes.
- The top edge of a window without a caption (`topedge.go`), which
  `WM_NCCALCSIZE` gives to the client area, where the webview's or the
  surface's window answers `HTCLIENT`, is a child window above them: a
  layered window without a bitmap (`WS_EX_NOREDIRECTIONBITMAP`), as tall
  as the sizing frame, that answers `HTTOP`, and `HTTOPLEFT` and
  `HTTOPRIGHT` near the corners it reaches, and sends the window's presses
  on, so it resizes; Windows Terminal's caption works the same way. Below
  it, `--app-region: drag` still moves the window. It spans a frameless
  window and stops at a hidden title bar's buttons, and hides while the
  window is maximized, in full screen or not resizable.
- A hidden title bar (`titlebar.go`) gets its controls from a child window
  above the webview's, beside the top edge. The buttons are a layered
  window (`UpdateLayeredWindow`, premultiplied BGRA): Segoe Fluent Icons
  glyphs (Segoe MDL2 Assets on Windows 10) drawn white on black with GDI
  give their coverage, over backplates of Chromium's alphas. It answers
  `WM_NCHITTEST` with `HTMINBUTTON`, `HTMAXBUTTON` and `HTCLOSE`, passes
  `WM_NCMOUSEMOVE` to `DefWindowProc`, which opens Windows 11's snap
  layouts over maximize, and runs the buttons itself from the non-client
  button messages, sending the window `WM_SYSCOMMAND`. The webview's
  window is created later, so the controls and the top edge go back to the
  top of the z-order when it appears. A window without a caption has no
  room for its menu bar: Alt and F10 open a popup holding the bar's own
  submenus.
- **Menus** are Win32's (`menu.go`). Popup menus follow the appearance
  through uxtheme's preferred app mode (ordinal 135); the menu bar follows
  nothing, so the window draws it, and covers the light line below it when
  dark, from the UAH messages menus send their window (`darkmenu.go`).
  Windows 11 still draws Win32 menus as Windows 10 did, so there MyGo draws
  the items in the style of its own menus (WinUI's `MenuFlyout`, File
  Explorer's), in either appearance (`menudraw.go`): popup items are
  owner-drawn (`MFT_OWNERDRAW`, measured and drawn in `WM_MEASUREITEM` and
  `WM_DRAWITEM` of the window the menu shows for, or of the application
  window for trays), each into a bitmap with its highlight's rounded
  corners antialiased, then copied, and excluded from the DC after, which
  keeps Windows from drawing the submenus' arrows over the chevrons;
  `WM_INITMENUPOPUP` gives the menu the background of the items. Windows
  widens every owner-drawn item past the width it asks for, by as much as
  it doesn't tell (not `SM_CXMENUCHECK`, #922): items ask for all they need
  until one draws, whose width less the widest asked for in its menu is
  that widening at the DPI, and for that much less after. Labels are
  measured by their advance and drawn unclipped (`DT_NOCLIP`), into the
  item's padding where their ink reaches past it (Microsoft YaHei UI,
  Chinese Windows' menu font, by a pixel, #1028). The items
  keep their strings, which mnemonics, screen readers and the tests read.
  A menu with owner-drawn items loses Windows 11's frame, its window square
  under its class's drop shadow (`CS_DROPSHADOW`) along two sides: while
  menus show (`WM_ENTERMENULOOP` to `WM_EXITMENULOOP`, and around
  `TrackPopupMenuEx`, which creates its window before), a `WH_CALLWNDPROC`
  hook of the main thread rounds the windows of class `#32768` in their
  `WM_CREATE` (`DWMWCP_ROUND`, WinUI's 8 pixels), gives them one outline
  (`DWMWA_BORDER_COLOR`: a pixel of WinUI's `SurfaceStrokeColorFlyout`
  over the menu's background, `#00000019` in the light appearance and
  `#FFFFFF19` in the dark, a color of its own rather than the accent) and
  takes the shadow off the class, for DWM's around the window. It also
  subclasses them: on Windows 11 22H2 with transparency effects on, no
  contrast theme, energy saver off, outside remote sessions and with a
  GPU (`glassConditions`: DWM draws no material when the first DXGI
  adapter is Microsoft's Basic Render Driver or a software one, unless
  its `ForceEffectMode` policy is 2), a window that says (`MN_GETHMENU`, before it shows
  or paints) that it shows a menu MyGo draws gets Acrylic
  (`DWMSBT_TRANSIENTWINDOW`, in the appearance through
  `DWMWA_USE_IMMERSIVE_DARK_MODE`) with the frame extended over all of
  it, and is framed active (`WM_NCACTIVATE` to `DefWindowProc` as active
  with an lParam of -1, again once shown and whatever it is told after,
  as Firefox's popups with Acrylic are): DWM draws a system backdrop as a
  flat grey in a window it frames inactive, as a menu's window, never
  activated, otherwise is. Its background brush turns black, which DWM
  shows as transparent, its border too (`WM_NCPAINT`, and `WM_PRINT` for
  animations), and its items wear the veil of WinUI's Acrylic tint over
  the material: the fallback color at the opacity of its luminosity layer
  (`acrylicLuminosity`, 85% in either appearance), so that a menu is as
  light as the shell's flyouts over a dark desktop, the material showing
  a seventh of the way through, and the highlights and separators are
  WinUI's translucent fills over it, their rounded corners leaving DWM's
  window to show. The labels, the shortcuts and the icons go over the
  veil's own color, opaque, and once: what shows is the text as GDI draws
  it on the color the shell's menus draw theirs on, its ClearType
  blending with it, rather than the grayscale antialiasing of text drawn
  over black and over white for unblend to tell its alpha, and the pixels
  GDI leaves no alpha to take the item's. A window that shows another
  menu, as Windows' window menu, stays as it is, and so do its items.
  Elsewhere a menu MyGo draws has WinUI's solid background, the fallback
  of its Acrylic (`#F9F9F9`, dark `#2C2C2C`), its window's own border
  painted in it, for that one outline to be the menu's only border. Tray
  menus open on Windows 11 as the system's do, centered on the icon 12
  DIPs from the taskbar (`ABM_GETTASKBARPOS`, aligned by
  `TrackPopupMenuEx` and kept off the taskbar by its `rcExclude`), and at
  the pointer on Windows 10 or for an icon not on the taskbar.
  The bar grows from `WM_UAHMEASUREMENUITEM` and its items get inset
  backplates with rounded corners. A contrast theme keeps the system's
  colors: the bar is Windows' again, the popup items take the system's.
  Windows 10 keeps its menus.
- **Vibrancy** sets `DWMWA_SYSTEMBACKDROP_TYPE` (Windows 11 22H2) and
  extends the frame over the client area, behind a transparent webview or
  native UI.
  The material shows only in a window created without a redirection
  bitmap (`WS_EX_NOREDIRECTIONBITMAP`, which Windows neither adds nor
  removes later), whose opaque surface would cover it, so `SetVibrancy`
  shows one only in windows created with `Vibrancy`. A transparent window
  of native UI (`Transparent` with `Content`) is created without one too,
  as a flyout is, where DirectComposition is available: what its frames
  leave transparent shows what is behind the window. Such a window shows
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
  out by hand. Notifications use native WinRT toast with per-user installed
  AUMID/shortcut and COM activation registration. Permission queries and explicit
  requests are separate from delivery; setup failures never select balloons.

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
  `visibilitychange` brings `SurfaceShown`, or `SurfaceHidden`.
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
  the back and forward keys. Those of a pointer that is not the primary
  one (`isPrimary`), as a second finger, are left out, as no second finger
  presses on the desktops; a button going down or up while another is
  held, which comes as a `pointermove` with its `button`, is a
  `PointerDown` or `PointerUp` as on the desktops (`chord`); their
  `pointerType` and `pressure` are the device and the pressure
  (`device`). `pointercancel`, the browser taking the pointer, as a touch
  it scrolls the page with, is a `PointerCancel`, not a release. `wheel` events scroll, lines and pages
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
  windows (the mousedown is reported, the backend starts a window drag
  from the recorded mouse event while the button is down).
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

- The page's theme color (`theme-color.ts`, `Page.ThemeColor`): the first
  `<meta name="theme-color">` of the head whose `media` matches and whose
  content is a color, which a canvas's `fillStyle` parses (a color it does
  not serialize as sRGB is drawn and read back), posted as
  `{t:"theme-color", v:"#rrggbbaa"}` (`""` for none) when it changes. A
  `MutationObserver` watches the head, not the body, so pages that change
  their body pay nothing, and `matchMedia` the metas' media, so a change of
  appearance picks another. Before DOMContentLoaded it posts only colors,
  then whatever it found, before dom-ready: the core keeps the previous
  page's color from the commit until then, so pages of the same color
  change nothing, and drops it at the load's end for a page that never
  told one (no bridge, as an image). The same in every engine: WKWebView's
  `themeColor` (macOS 12+) would need KVO, and WebView2 and WebKitGTK have
  none.
- Fitting a window to its content (`fit-content.ts`,
  `FlyoutOptions.FitContent`, `WindowOptions.FitContent`): the bridge
  configuration's `fit` (the limits, in the CSS pixels of the page's zoom,
  which `mygo:fit-content` updates as the zoom changes) adds
  `:where(:root){width:max-content;min-width;max-width;min-height}` to
  MyGo's style sheet, without specificity so that a page's own size for
  `html` wins, with the room of a vertical scrollbar left out of the
  widths while the content is taller than the maximum height. At DOMContentLoaded, before dom-ready, and then from a
  `ResizeObserver` on the root and the window's `resize` events, it posts
  `{t:"content-size", w, h, sw, sh}`, the root's border box and the
  scrollbars' room in CSS pixels, when it changes; `sh` only while the
  content is wider than the root (the body overflows), as at the maximum
  width. The core turns it into DIPs at the page's zoom (a fraction of a
  pixel takes one) and clamps it. The vertical scrollbar's room counts
  only where the content is taller than the window may be (its maximum
  height or the work area): the page shows one for a moment whenever the
  content grows before the window does, and counting it then would undo a
  size the user just gave the window. It resizes a
  flyout where it goes, or a window's content bounds, only in the
  dimension that changed and not while maximized or in full screen,
  within its display's work area; a size equal to the last it took does
  nothing, so a user's resize holds. The same in every engine: WebKit's
  private `-[WKWebView _setSizeToContentAutoSizeMaximumSize:]` is not
  used. A flyout of native UI fits without the bridge: its
  `surface.Conn` gets `FitWidth` (the width's limits and the work area's
  width, from the display the flyout is on) and `ContentSize`, through
  `fitContent.native`, which `NewFlyout` sets so that apps without such a
  flyout link none of the core's side, and
  `ui/fit.go` measures the root as each frame lays out, right after
  `layoutTree`: `contentWidth(root, true)` within those widths, and
  `contentHeight` at that width, the non-committing layout flex columns
  measure their items with, so percentages and `Grow` measure as their
  content, as `height: auto`; absolutely placed elements and the
  inspector take none of it. It tells a size only when it differs from
  the last it told, and the core clamps it as a page's (`fitNative`),
  then resizes and places the flyout after the frame (`postMain`, as the
  backends' resizes may send the surface events at once): the surface's
  `SurfaceResize` brings the frame laid out at the new size. A frame laid
  out at a new size that measures the size told before the last one
  holds, so that a view building its content after `Context.Size` does
  not resize the flyout back and forth. A hidden window draws no frames,
  so `Show` and `ShowInactive` of a hidden flyout first call
  `fitContent.show` (`fitBeforeShow`), which `NewFlyout` sets with
  `native`: where the content waits to draw a frame, `surface.Conn.Fit`
  builds and lays it out at once, as a capture does, and the flyout
  takes the size it measured before the platform shows it; the
  surface's next frame draws it at that size. `Fit` builds nothing
  within the content's frame or before its first, so the view never
  runs within `NewFlyout`.
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
| `{"t":"theme-color","v":"#rrggbbaa"}` | the page's theme color changed; `""` for none |
| `{"t":"content-size","w":W,"h":H,"sw":SW,"sh":SH}` | the size of the content of a window that fits it, and the room of its scrollbars, in CSS pixels |
| `{"t":"drag"}` / `{"t":"dblclick"}` | mousedown / double click on a drag region |
| `{"t":"context-menu","x":X,"y":Y,"l":link,"lt":text,"i":image,"m":media,"s":selection,"e":true}` | the page's `contextmenu` event, not cancelled, at X, Y in CSS pixels: what it opens on, empty parts left out (kept only while `OnContextMenu` has a listener) |

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

`Bind` lists a service's methods with `reflect.Type.Method`, and a program
that does that keeps every exported method of every type in an interface,
since the linker cannot tell which it calls: about 2.5 MB of a native UI
app. So only apps that call `Bind` (or `Use` a plugin with a service) link
it. MyGo's own service, `mygo:window` behind `window.mygo.window`, is bound
by `bindFuncs`, from funcs, without it; `go build -ldflags=-dumpdep`
prints `<ReflectMethod>` on what does it.

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
twice. The result travels as a JSON string and is decoded in Go. A page whose
window closed never answers, so a window closing fails the evals waiting for
it, and an eval with the quit under way fails at once.

### Trust

Every page gets the runtime, but only trusted pages may call Go: the
frontend (`mygo:` and the `mygo dev` server), custom-scheme pages
(`Protocol.Handle`), `file:` and `about:` pages, loopback `http(s)` dev
servers in development, and origins listed in `PageOptions.TrustedOrigins`
(`"*"` trusts everything). The decision is
recomputed on every committed navigation. Untrusted calls are rejected
without running any Go code.

`PageOptions.Untrusted` (`Window.untrusted`, fixed when the window or web
view is made) makes `isTrusted` false whatever the URL, for a plugin's UI
served from a custom scheme. The core then drops the page's messages above
`maxUntrustedMessage` before stripping the secret, rejects its calls on
the main thread instead of a goroutine each, ignores its drag regions,
leaves it out of `Broadcast` and of `Emit` to the window its web view is
in, and refuses what other pages get by default without a handler:
permissions (through `isTrusted`), new windows and the browser, downloads,
and main-frame navigations the page starts to another origin than its
committed page's (`Window.origin`). `serveScheme` puts the asking page in
the request's context, under the keys of `CallerWindow` and `CallerPage`,
so a scheme's handler knows which page sent a request: that's the channel
of untrusted pages to the app.

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
- **watch** gives pages `mygo.WatchFiles` watches of the directories and
  files the app names, by their names (`Roots`, or `Resolve` for those
  granted while it runs). `Watch` streams a part per batch through a
  channel for as long as the watch lasts, and returns how it ended, a code
  the page turns into a `WatchError`. A page's watches count against its
  quota by `mygo.CallerPage` while their call's context is not done, as
  navigating ends the page context before the watches have stopped; they
  stop with it.

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
    the clicks it counts itself. Keys meet the terminal's keybinds before
    the encoder: Ghostty's defaults for the platform with the app's over
    them, as Ghostty layers its config. A paste reads the clipboard on the
    main thread as it starts (every format for `OnPaste` or a program in
    mode 5522, else the text), and the terminal keeps a snapshot: the
    program's read (OSC 5522) is answered during `Write`, on the reader's
    goroutine under the terminal's lock, where waiting for the main
    thread could deadlock. The binding gathers the library's reply and
    hands it over in one piece, which the input queue writes in pieces
    that report their progress. Any other read of the clipboard
    (`ClipboardRead`: OSC 52's "?" and OSC 5522, unless a paste granted
    it) is queued instead: the binding holds the library's own refusal
    back, and the view, on the main thread, reads the clipboard and
    writes the answer, which the reader's lock makes safe.
  - *Shipping the library.* `mygo-plugin.json` names its build for each
    platform, published as release assets with their SHA-256 checksums.
    The CLI puts them into apps
    (see the CLI's resources); other programs download theirs into the
    user's cache once, which packaged apps never do.
- **game** has what games need beyond what apps do, opt-in, out of core
  `ui`, on its public API only. `View` is a focusable box that keeps its
  loop in `ui.Local`: its input handler gathers keys (by `PhysicalKey`),
  buttons and wheel into the `Input` of the next step, and its draw
  function asks for the next frame (`AnimationFrame`), adds the time since
  the last frame (`Painter.Now`, at most a quarter of a second) and runs
  the steps it covers before `Draw`. `Camera` is a `ui.Transform`, snapped
  to device pixels when unturned.
  Its package `gamepad`, like `audio`, stands apart from windows and the
  main thread. One mutex guards the pads. `Pads` asks the platform's `scan` for
  the devices connected at most once a second, handing it the devices it
  has so that it reuses them, and a device that is gone, from a scan or
  when read, disconnects its pad. `State` reads a device on the calling
  goroutine into the standard layout (`State.Buttons` in the W3C's order,
  `Axes` with y down):
  - macOS: GameController.framework through purego's objc: `GCController`
    `controllers` with an `extendedGamepad`, each element's `value` read
    inside an autorelease pool, after `respondsToSelector:` for the buttons
    older systems lack. The framework finds controllers on the main run
    loop.
  - Linux: evdev, opened non-blocking where `EVIOCGBIT` shows `BTN_SOUTH`;
    reading drains the events waiting into key and axis arrays, with axes
    scaled by `EVIOCGABS`'s range. Xbox controllers (Microsoft's and Valve's
    vendor IDs, or the `xpad` driver) report X and Y under BTN_NORTH's and
    BTN_WEST's codes, which are swapped back.
  - Windows: XInput's four slots, through `XInputGetStateEx` (ordinal 100)
    for the Guide button where it exists. Empty slots, slow to ask, are
    asked only by scans.
  - Browsers: `navigator.getGamepads()` by index, the standard mapping
    matching the package's order.
  Tests replace `scanDevices` with fake devices; on Linux, `TestEvdev`
  (`MYGO_UINPUT=1`) makes uinput pads in the test container, and on macOS
  `TestGameController` reads a controller from `controllerWithExtendedGamepad`.
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
  `internal/gen`. `glass.Group` paints one surface over a row of elements:
  a shader of package `ui/shader` (Kage, compiled for every renderer and
  the CPU) takes up to 8 rectangles and radii, unions them with a smooth
  minimum (so panes that nearly touch bridge), and draws the glass once
  around the union's bezel. `Glass{Native: true}` asks the window's
  surface for the system's glass: `ui.NativeGlassMaterial`, noted when the
  frame commits (`ui/nativeglass.go`), tells the surface where the glass
  is (`platform.GlassSurface.PlaceGlass`). On macOS the surface puts an
  `NSGlassEffectView` under the content and above the web views, and the
  material leaves a hole where it shows, so the glass samples a web view's
  page or the window's material, never the frame's own pixels; elsewhere,
  and with Reduce Transparency, the shader draws it. The materials read
  `ui.Painter.Preferences`: Reduce Transparency draws opaque surfaces,
  Increase Contrast lights the rim. Window-level glass takes a tint
  through `NSGlassEffectView`'s `tintColor`.
- **loading** is loading.dev's 29 indicators (`loading-dev`, React and
  CSS) for native UI, on ui's public API only. Each is a box of
  `RoleProgress` whose draw function asks for the next frame
  (`AnimationFrame`) and computes its pose from how many cycles it has
  run: since it first showed (its start in `ui.Local`, as a CSS animation
  starts with its element), or held while `Paused`. Parts are staggered as
  loading-dev's negative `animation-delay`s, keyframes eased stretch by
  stretch with CSS's `cubic-bezier` (`frames`, `bezier`), and shapes built
  as `ui.Path`s in the SVG's viewBox through an affine transform (`affine`,
  `shape`), arcs as cubic Béziers, which an affine transform keeps exact:
  Atom's rings turning in 3D are ellipses, Flip's square is projected in
  perspective point by point. Round caps are `StrokePath`'s; flat ones
  fill the stroke's outline. Gradients are conic ones (Comet, Orbit), and
  Radar's radial fade is sectors of growing radii over each other, each as
  opaque as makes its band the gradient's opacity, so that no two edges
  meet inside the beam. With `ReduceMotion` an indicator draws
  loading-dev's pose for reduced motion and asks for no frames. The
  indicators were checked against loading-dev rendered by a browser at the
  same points of their cycles, with every option.

## Typed client generation (`internal/tsgen`)

`mygo generate` builds the app and runs it with `MYGO_GENERATE=<file>`;
`App.Run` then writes the client (`WriteTypeScript`) and returns before
touching the GUI. It reaches the generator through what `Bind`, `BindAs`
and `NewEvent` register (`ipc.writeClient`): an app that binds nothing
writes no client, and links neither the generator nor go/parser (375 KB).
The generator combines:

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

## The language the app shows (`locale.go`)

The app shows one language, which `localeState.shown` holds with the
user's languages and regional formats: an atomic pointer the main thread
replaces (`updateLocale`), so that `App.Locale`, the menus, the catalogs'
`i18n.Locale` and native UI read it from any goroutine for nothing. The
first read works it out on the main thread. `updateLocale` reads the
backend's `PreferredLanguages`, `Region` and, once, the saved language
(`SavedLocale`, where the system keeps one, else the file `locale` in
PathUserData; `MYGO_LOCALE` over both), resolves it with `i18n.Match`
against `App.Languages()`, and, when it changed, tells: windows of native
UI (`Conn.LocaleChanged`, after which `ui` reads `Conn.Locale` and
`Conn.Region` and sets the text system's language), pages (the event, and
`Page.SetUserScripts` with a bridge configuration of the new language, for
pages loaded later), menus (`relabelMenus`: items whose label is still
their role's take `internal/l10n`'s), the backend (`SetLanguages`: the app's
language, then the user's others) and the listeners. `SetLocale` calls the
main thread may not have reached yet show the last tag only (a
generation counter). A backend's `LocaleChanged` reports changes of the
system's languages or formats, which `updateLocale` compares.

The text engine takes a language per style and span (`text.Style.Lang`),
and one of the system for those without (`System.SetLanguage`, a new
generation of layouts): Core Text's `kCTLanguageAttributeName`,
DirectWrite's locale of text formats and of ranges
(`IDWriteTextLayout::SetLocaleName`), Pango's language attributes. Han
characters in a language that does not write them get the engine's own
choice (`langRuns`), as Chromium's `LocaleForHan` does; Core Text does so
by itself.

WebView2's environment takes the language through
`ICoreWebView2EnvironmentOptions`, a COM object of ours
(`webview2_options.go`) whose vtable's callbacks are made once; a runtime
that fails with it is asked for an environment without it.

## Catalogs of messages (`i18n`, `internal/i18ngen`, `internal/l10n`)

Package `i18n` is a leaf (it imports golang.org/x/text/language only):

- **CLDR data** is generated, never written by hand: `go generate ./i18n`
  runs `i18n/internal/gen`, which reads the npm tarball of cldr-core at the
  version it pins, checked against its SHA-256, and writes `tables.go`.
  Plural rules are compiled to Go, a `switch` per kind (cardinal, ordinal)
  over the distinct rule sets with a sorted table of languages, so nothing
  is built at init; an operand of more than 18 digits keeps its last 18
  plus 10^18, which keeps its remainders by powers of ten and equals no
  constant of the rules. Rules are looked up by truncation, as ICU does
  (CLDR's parent locales of plurals are empty); messages fall back along
  `Parent`, CLDR's main parent locales with its nonlikelyScript rule. The
  tests check every sample CLDR gives and ICU's categories for 200 numbers
  in every language (`scripts/plurals-golden.ts`, run with Node, whose ICU
  is upstream; Bun on macOS uses Apple's, which differs).
- **Formatting data** of CLDR's modern locales (effective coverage
  "modern", 430 locales) comes from cldr-dates-full, cldr-numbers-full and
  cldr-localenames-full, pinned like cldr-core (`format.go` of the
  generator writes `format_tables.go`). A locale's data is a record of
  slots (month and weekday names by context and width, day periods, eras,
  date, time and date-time patterns, the availableFormats of the
  skeletons MyGo formats, time zones' GMT formats, number symbols,
  patterns and digits of the default numbering system and of latn); each
  locale stores only the slots in which it differs from its nearest
  ancestor of the list (or from a synthetic root of the values most
  locales have), as indexes into a pool of strings. The slots come in
  two halves, each with its own entries and pool, packed with DEFLATE in
  a constant of its own: the numbers with the locales and their parents
  (`fmtPackedNumbers`, 3 KB), which every app links, as MyGo's own strings
  format numbers, and the dates with the names of languages
  (`fmtPackedDates`, 77 KB), which only apps that format dates or name
  languages link. At run time `cldr.go` inflates each once and puts a
  locale's record together on its first use (an atomic pointer per
  locale), its dates on the first date it formats (`withDates`); `keyOf` caches what a tag formats with (its data locale, by
  `Parent` and likely scripts, and its `-u-hc`, `-u-fw`, `-u-nu`). Week
  data, time data and day period rules are small tables by region.
  `dtpg.go` ports ICU's DateTimePatternGenerator (dtTypes, distances,
  the pattern map keyed by base skeleton and types in ICU's order,
  adjustFieldTypes, date and time put together with the atTime glue), and
  `date.go` V8's handling of styles and hour cycles; `number.go` writes
  decimal strings, so that rounding is decimal as ICU's. Formatters are
  kept per locale and options in maps under a read lock.
  `scripts/format-golden.ts` records Node's `Intl` (upstream ICU) for
  every locale; `TestFormatMatchesICU` compares, and `knownDifference`
  lists, with reasons, the cases that differ on purpose.
- **Relative times** (`relative.go`) use dateFields from the same pinned
  cldr-dates-full tarball, written to `relative_tables.go` in a compressed
  blob of their own: runs of slots as `recordSections` writes them, by
  style, unit, and words for offsets -2 to 2 or past/future plural form.
  Only `NewRelativeTimeFormatter` reaches it, so formatting dates or
  numbers links none of it. The formatter uses the number formatter
  (half-even, as ICU's RelativeDateTimeFormatter does), the plural form
  of its rounded number, and ICU's rounding of offsets to hundredths
  when choosing words with `Auto`. `scripts/relative-time-golden.ts`
  records Node's Intl for every formatting locale, with the full matrix
  compressed in `testdata/relative_time_icu.json.gz`; the tests name the
  CLDR/ICU data differences explicitly.
- **MessageFormat** (`message.go`, `format.go`) parses ICU's syntax as
  ICU's MessagePattern does (apostrophe mode DOUBLE_OPTIONAL, `#` only in
  a plural's own cases), and formats numbers and dates with the
  formatting above; a plural case is picked on the number as `#` writes
  it. Dates go through `timeFormat`, which `NewCatalog`,
  `NewLibraryCatalog` and `ParseMessage` set (`linkTimes`): MyGo's own
  catalog, which every app formats and which has no dates (its test
  checks), is made through `internal/i18nlink`, which does not, so an app
  that formats no date links no date formatting. `mygo-runtime/i18n` is its TypeScript twin over `Intl` (skeletons
  turned into `Intl`'s options, as V8 turns them back);
  `i18n/testdata/messages.json` holds the cases both test.
- **Catalogs** read a language's file once, on its first message
  (`sync.Once`), and cache the tags a locale looks keys up in. A library's
  catalog (`NewLibraryCatalog`) looks a key up in the registered catalogs
  of the app under `<namespace>.<key>` before its own, at each tag of the
  chain. `Register` declares the languages, which package mygo reads with
  `Languages`; `SetLocaleFunc` is how package mygo tells the language the
  app shows to `Locale`, which the generated functions read.

A catalog keeps each message formatted without arguments, by locale and
key, as of the registry's version: the controls of native UI say theirs
every frame (`l10n.In(c.Locale()).Undo()`), for a map lookup under a
read lock and no allocation. `ui`'s `context.formats` works out, once per
change of the language or the system's formats, the region the controls
follow (CLDR's for the language when the system says none) and its
`Region.FormatLocale`; `context.formatted` keeps the dates and numbers the
calendar, the date and time fields and the color picker write, so a frame
formats nothing it formatted before.

Backends say a few words of their own (the buttons of pages' alert,
confirm and prompt panels, file dialogs' titles, the about panel's
title): package mygo gives them in the app's language through
`platform.SetTexts` as the language changes, so that `internal/<os>`
does not import `internal/l10n`.

`internal/i18ngen` checks catalogs and writes their functions:
`catalog_gen.go` in the catalogs' directory (an embed, the catalog,
registered for apps, and a function and a method of `Messages` per key)
and the TypeScript module, which carries the messages with each
language's fallback chain. `internal/l10n` is MyGo's own catalog,
generated by the same code (`go generate ./internal/l10n`, with
`i18ngen.Options.Own`), whose keys apps change as `mygo.<key>`; the CLI
checks those keys against it (`l10n.FS`, the files as written, which only
the CLI links). It embeds English only, as `messages/en.json`, which the
generator writes from `en.json` without the descriptions (7 KB of 28): each translation is a package,
`locales/<tag>`, that adds its file to the catalog as it is imported
(`i18n.Catalog.Add`), so that an app links only the languages it ships.
The generator writes those packages and `l10n.Languages`; the CLI passes
that list to `internal/i18ngen`, whose Go code imports the translations
that suit the app's languages, and package mygo tells an app that ships
one of them without the translation, as it starts. The
CLI (`cmd/mygo/i18n.go`) writes the functions before anything compiles,
in `generate`, `build` and every build of `dev`, where the catalogs'
directory is watched and the generated Go file is not, so writing it does
not trigger another build.

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
- Called on the main thread, `NewWindow` leaves showing a window of
  native UI to the main loop's next turn (`showLater`, `settleShow`), so
  that its view first runs once `NewWindow` returned and can use the
  window kept from it (#1053): shown within `NewWindow`, a window draws
  its first frame at once on
  some systems, as AppKit may display a window it orders in, and Windows
  a window maximized as it shows (`WM_SIZE`). The frame the content asked
  for draws on that turn too, while the window is still hidden, as a
  window made `Hidden` draws its first (`drawSoon`, `drawHidden`), or as
  it shows. A method acting on the native window (`do`, `get`) shows it
  first, so that it finds the window as `NewWindow` returned it, but not
  from within a frame (`framing`), whose view may call one (`Title`): a
  frame drawn as the window shows would run inside it. `Show`,
  `ShowInactive` and `Hide` take the deferred show's place, so `Hide`
  right after `NewWindow` keeps the window from showing. Called from
  another goroutine, which `NewWindow` returns to while the main thread
  goes on, `NewWindow` shows the window before it returns, as it does a
  window of a page, whose page loads later anyway.
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
  display keeps its size and is centered. On Windows, where a window's
  position in DIPs changes with the displays' scales (`dipSpace`), and
  earlier versions' could not tell displays of different scales apart,
  the state also keeps the window's
  rectangle in pixels (`platform.WindowStateWindow`), and the backend
  creates the restored window at it, on its display
  (`platform.WindowStateBackend`, `WindowOptions.StateBounds`); a state
  without pixels is restored in DIPs where every display has the same
  scale, and centered elsewhere. `WindowOptions.Maximized` starts
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
  launch. There the app directory holds `update.FileList` (`.mygo-files`),
  its top-level entries as installed, one per line: `mygo build` writes it
  into the app (so the archive, installer and packages have it), and
  `swapFiles` rewrites it in the update with the update's entries before
  swapping it in with them. The entries that the installed list names and
  the update lacks are renamed aside first, as replaced ones are, and
  deleted once the swap succeeds, so a failed swap puts them back; entries
  that no list names (the user's, the uninstaller) and all of an app
  without a list stay, and so does the running executable. Deltas never
  make the list from the installed one, which the updater rewrote.
  `App.Relaunch` then starts the new version from the path the app
  started from. Development builds are never updated. Delta updates work
  as Sparkle's (`internal/update/delta.go`): `mygo build` fetches the
  published manifest of the target, downloads the archives of up to
  `updates.deltas` versions (that manifest's, and those it lists as
  `previous`, which the new manifest passes on), checks their signatures
  and writes a signed delta from each. Its index lists the tree of the new
  app, inside the bundle on macOS, with each file's size and SHA-256 and
  how to make it: a copy of a file of the old app with the same content
  (at its path, or elsewhere for moved files), else a patch of the file at
  its path or its own bytes, whichever is smaller. The stapled ticket
  (`Contents/CodeResources`) is never made from the old app: the archives'
  bundles are stapled, but the bundle a user copies from the disk image is
  not, as `buildDMG` staples the app after copying it into the image.
  Patches are bsdiff's
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
  the repository's latest release stays the others'. While the app turns
  prereleases on (`Updater.SetPrereleases`, an `atomic.Bool` that `Check`
  reads once), `Check` resolves the feed of the latest release (with the
  prefix `v`) and tagged feeds alike with `update.ResolvePrereleaseFeed`
  instead: the manifest of the release with the highest version
  (`update.Compare`) that is not a draft, among the pages of releases up to
  the one holding the newest release that is not a prerelease. With `updates.s3` it puts the
  same files, manifests last, into a bucket of S3 or a compatible service
  (`cmd/mygo/s3.go`): one `PUT` per object, signed with AWS Signature
  Version 4 in pure Go (`crypto/hmac`), its SHA-256 payload hash checked
  by the service; the bucket goes in the host name on AWS, and in the path
  with a custom `endpoint` or a bucket name with dots, unless `pathStyle`
  says otherwise. An app with an update service of its own gives
  `Updater.SetSource` an `UpdateSource` (an `atomic.Pointer` that `Check`
  reads once): `Check` reads the manifest from its `Manifest` in place of
  the feed, and the `Update` keeps the source, whose `Open` downloads the
  archive and deltas in place of `httpGet`. The configured feed is itself
  a source (`feedSource`), so parsing, sizes, signatures, staging and the
  swap are the same either way; `Enabled` still needs a build with
  `updates`, for the public key.
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
- Sign-in and certificates (`auth.go`): the backends route HTTP
  authentication challenges (Basic, Digest, NTLM, Negotiate, the server's
  or a proxy's) to `WindowHandler.AuthRequested`, which `SetAuthHandler`
  answers with credentials, or none to get the server's refusal; without
  a handler the answer is `Default`, the web view's own handling.
  Server certificates that do not verify go to
  `WindowHandler.CertificateError`, false unless `SetCertificateErrorHandler`
  allows them; macOS checks certificates itself
  (`SecTrustEvaluateAsyncWithError`) only while
  `HandlesCertificateErrors` is true, and Linux and Windows extract the
  certificate only then. Client certificates are left to the platform.
- Active windows (`active.go`): a window is drawn as active, the one the
  user's input goes to, while it or a flyout of it (through flyouts'
  parents) has the keyboard (`windows.focused`), always while
  `SetActiveAppearance` keeps it so, and so are panels and flyouts
  without a parent; a flyout with one is as active as its parent, and a
  window that never had the keyboard nor showed without it
  (`focusKnown`) counts as active, so that the first frame of a window
  about to take it draws active, which package `ui` assumes too. No
  backend reports it: `Focused` and `Blurred` post `updateActive` once
  (`scheduleActive`), which works out every window's state after the
  native event returns, so that a window blurred by its flyout taking
  the keyboard in the same event (AppKit's resign and become key,
  Win32's `WM_ACTIVATE`s, GTK's focus events before an idle source)
  never shows inactive for a frame. A change reaches native UI through
  `surface.Conn.Active` and `ActiveChanged`, and pages, and those of the
  window's web views, as the `mygo:window-active` event (true or
  false); pages start active, and one whose DOM is ready in an inactive
  window is told so. The bridge keeps it for mygo-runtime's
  `windowActive()`, telling listeners of changes only, and sets the
  `data-mygo-window-inactive` attribute of `<html>` while it is false.
  The system draws the materials itself: macOS's `NSVisualEffectView` of
  a window follows AppKit's active state
  (`NSVisualEffectStateFollowsWindowActiveState`), the main window's
  look, which agrees for flyouts (a panel never becomes main) and differs
  while a sheet or one of AppKit's own panels (Fonts, Colors) is key: the
  material stays active there while the content is not. Panels',
  flyouts' and those of windows `SetActiveAppearance` keeps active stay
  `NSVisualEffectStateActive`, as they did for all windows before;
  `NSGlassEffectView` has no such state and follows AppKit's. DWM draws
  Mica and Acrylic as a solid fill in a window it frames inactive: a
  window whose focusable flyout Windows activates, as a window of its
  own, keeps its frame active (`keepsActive`: `WM_NCACTIVATE` passed to
  `DefWindowProc` as active, and its drawn caption), until the
  activation leaves the flyout for another app's window
  (`ownersInactive`, `flyoutLeft`). GTK styles a window `:backdrop` by
  GDK's focus, which a focusable flyout takes from its parent: the
  window's own title bar then shows it, while its content stays active.
  In browsers, a window is active while its canvas has the focus.
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
  no shortcut took (`surface.Conn.Escape`), neither one of the last
  frame (`routeKeys`) nor one the frame handling the key builds, which
  takes it in the next frame (`settleKeys`): a key pressed as the view
  changes, before a frame shows the change, goes where the view now
  sends it, so an Escape typed ahead of the frame that builds a
  flyout's shortcut for it never closes the flyout. The bridge reports
  one no handler of the page cancelled (an `escape` message). `Flyout.Popover` shows one in an
  `NSPopover` (`internal/darwin/popover.go`), whose window AppKit makes
  as it shows, and whose transient behavior asks `popoverShouldClose:`.
  `Flyout.Tray` anchors one to a tray icon's `Bounds` instead, without a
  parent: the panel at `NSPopUpMenuWindowLevel` or an `NSPopover` shown
  from the status item's button (macOS), a topmost unowned popup
  (Windows), and on Linux, whose AppIndicator tells no bounds, an
  undecorated utility toplevel in the middle of the work area, which a
  focusable one closes as it loses the keyboard. A press on the icon
  does not dismiss its flyout (`tray.pressed` on resigning key, the
  cursor over `Shell_NotifyIconGetRect` on Windows), so its click
  toggles it, but a popover's: AppKit eats a click on a transient
  popover's positioning view, the status item's button, whatever
  `popoverShouldClose:` answers, so the button sends no action, and the
  click closes the popover (AppKit asks on the release on macOS 26, on
  the press elsewhere: one kept open there stayed open for good, #1050).
  A global monitor of mouse downs
  closes macOS's focusable panels on clicks outside the app, which a
  panel taking the keyboard of an app that was not activated never
  hears as the key window changing. A window modal to a flyout
  (`WindowOptions.Parent`, `Modal`) is centered over it by the core
  (`overFlyout`), in its display's work area, and keeps it open: the
  flyout's dismissals skip a flyout a window is modal to (`modalChild`
  on macOS and Linux, `hasModal` on Windows, where `EnableWindow`
  disables the owner as for any modal window). macOS attaches the window
  as an AppKit child window at the flyout's level rather than a sheet,
  and a local monitor eats the flyout's mouse and key events while it
  shows (`watchModalInput`); `popoverShouldClose:` keeps a popover.
  Windows makes it topmost over a topmost flyout. On X11 the popup lets
  its grab go while the window shows and grabs again as it hides
  (`enterFlyout`, `leaveFlyout`); a tray's flyout, a toplevel, does not
  close as it loses the keyboard to it. The design follows GPUI's
  `WindowKind::AnchoredPopup` and winit's `WindowType::Popup`, which
  model xdg_positioner too.
- Panels (`WindowOptions.Panel`), the windows of launchers: the core makes
  them always on top, out of the taskbar, unable to minimize, maximize or
  go full screen, and closes them on an Escape the content leaves, as
  flyouts. macOS makes a `MyGoPanel`, an `NSPanel` with the nonactivating
  style from `initWithContentRect:` on (the window server learns that a
  window never activates its app from the style it is created with; a
  style set later needs the private `_setPreventsActivation:`, as Electron
  and tauri-nspanel found), never the main window, on every Space as a
  full-screen auxiliary, outside Mission Control and the window cycle, at
  the floating level (above it, input methods' candidate windows would
  show behind the panel); `Show` and `Focus` make it key without
  `activateIgnoringOtherApps:`, and its `cancelOperation:` does nothing
  (NSPanel's closes it past `windowShouldClose:`). The key window of a
  nonactivating panel gets the keys and the main menu its key
  equivalents, though the app is not frontmost (`NSApp.isActive` reports
  it active then). Windows makes a topmost tool window owned by the hidden
  application window (a tool window alone may get a button on other
  displays' taskbars, as PowerToys found), its corners rounded as
  Windows 11 rounds other windows'; one the app maximizes fills the work
  area (`maximizeInWorkArea`), where Windows would give a window without
  a maximize box the whole monitor, over the taskbar; the key press of a `RegisterHotKey`
  shortcut lets the process take the foreground, and hiding the panel
  gives it back. Linux makes a sticky utility window, kept above, or on
  Wayland a surface of the overlay layer (`internal/linux/layer.go`):
  `libgtk-layer-shell.so.0` is loaded with the first panel, if it is
  there (0.9 patches GTK 3's private structures and intercepts nothing
  of libwayland, so it may load after GTK), and used where
  `gtk_layer_is_supported` says the compositor speaks wlr-layer-shell, as
  Ulauncher does. `gtk_layer_init_for_window` must come before GTK
  realizes the window. A position is the top and left margins from the
  edges of the output there (`gdk_display_get_monitor_at_point`), and
  without one the compositor centers the surface; gtk-layer-shell sizes
  it by its size request, not its default size. It asks for the keyboard
  exclusively while it shows: compositors give none to an on-demand
  surface as it maps, and Sway takes it back from one that goes from
  exclusive to on demand. Neither Linux nor Windows shows the application
  menu in a panel.
- `window.open()` and `target=_blank` go through `SetWindowOpenHandler`. By
  default http(s) URLs open in the default browser. Allowing one creates a
  window around the configuration or related view WebKit provides, with its
  own content manager so scripts and messages never leak between windows.
  On Windows the request's deferral holds the popup until the new window's
  web view is set up (its scripts, handlers and filters), which then becomes
  the popup (`put_NewWindow`) before WebView2 navigates it, so the page keeps
  its opener. On Linux the new window loads the URL as an independent page.
  `UserInitiated` says whether a user's gesture made a request, and so
  does `NavigateEvent` for navigations: on
  macOS WebKit's private `-[WKNavigationAction _isUserInitiated]` (looked
  up with `respondsToSelector:`; without it, a link or a form's
  `navigationType`), which sees a click whose handler calls
  `window.open()` or sets `location.href`; on Linux
  `webkit_navigation_action_is_user_gesture`, though WebKitGTK refuses a
  `window.open()` without one itself before asking
  (`javascript-can-open-windows-automatically` stays off), so the handler
  sees those only on macOS and Windows; on Windows the event arguments'
  `get_IsUserInitiated`, WebView2 having no pop-up blocker. Only main-frame
  navigations reach `OnWillNavigate`: on Linux the action's target frame
  says so (`webkit_navigation_action_get_target_frame_info`, 2.56+, bound
  when present); before it WebKitGTK tells no navigation's frame, and
  `mainFrameNavigation` takes all for the main frame's but `about:srcdoc`,
  which only an iframe shows.

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
  `webkit_web_view_execute_editing_command` on Linux; on Windows a page
  keeps their keys, which WebView2 performs, and the menu items run
  `document.execCommand` through DevTools, except Paste, which runs
  Chromium's paste command with the clipboard's formats, and Paste and
  Match Style, which inserts text), or sends them to a window showing
  native UI as a `SurfaceCommand`, and reports everything
  else through `WindowHandler.MenuItemClicked` for a window's menu, or
  `AppHandler.MenuItemClicked` for menus without one; the core toggles
  checkbox/radio state, performs window and view roles and calls `Click`
  with the menu's window, even while a submenu has focus instead,
- enables the edit roles' items of native UI as the content answers a
  `SurfaceCommandEnabled` (package ui's `commandEnabled`, which follows
  `editEvent`): on macOS the menus holding them validate their items
  (`autoenablesItems`), the surface answering `validateMenuItem:` and the
  menu target keeping the app's state, which a disabled role's item
  targets; Windows and Linux ask as a menu opens (`WM_INITMENUPOPUP`, a
  submenu's `map`, a popup before it shows), and again before a role's
  shortcut runs.

Item icons (`MenuItem.Icon`, made by `NewMenuIcon`) travel as a
`*platform.MenuIcon`, which the core shares between the icons of the same
PNG (interned by content, held weakly): backends tell a change by the
pointer and decode an image once (`MenuIcon.Native`), freeing what they
made with `runtime.AddCleanup` or with the menus that show it. Their code
is linked by `NewMenuIcon` (`LinkMenuIcons`), not by the backends' menus.
Backends that keep a snapshot to build menus from later (Windows' trays,
the application menu of new windows) keep it current with the updates they
get (`Menu.Apply`).

macOS gets a default menu bar (App, File, Edit, View, Window), which is what
makes Cmd+C/V/Q work; other platforms get none unless the app sets one.

A page's own context menu (`context_menu.go`, `Page.OnContextMenu`) goes the
other way: while a listener is set (`WindowHandler.HandlesContextMenu`), the
backend describes the menu the web view is about to show as a
`platform.ContextMenu` whose items carry a `Native` number and a `Name`,
and asks `WindowHandler.ContextMenu`. The core makes a `Menu` of template
items of them (`Menu.transient`, which refreshes no installed menu), fires
the listeners, and answers `Default` when the snapshot still shows the
web view's items as they were, none when prevented, else the snapshot,
each native item at most once. A native item the app gave a `Click`
leaves the snapshot as an item of the app's. What the backend does not tell
of the target comes from the bridge's `context-menu` record of the page's
`contextmenu` event, which arrives just before the backend's request on the
same connection, and is taken only when its point, times the page's zoom,
is the menu's (a frame's click never reaches the page's document); a new
page drops it. macOS and Linux change the native menu in place; Windows
draws a changed menu itself (see [Platform
differences](#platform-differences)). Its code is linked by `OnContextMenu`.

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
with file URLs prepared before publication and all representations eager
after flush. GTK uses application-owned selection
get/clear callbacks and owner-change generation checks; clipboard-only apps
bind selection primitives without requiring a surface. Windows shares the
drag adapter's IDataObject, vtables, native conversions and reference-counted
objects with OleSetClipboard/OleFlushClipboard; foreign reads hold a Win32
clipboard lock and retry while another owner has it. Callback signatures are
allocated once and routed by native object or user data. Native reads copy
bytes so application data outlives native handles. See
[Clipboard and drag data](data-transfer.md) for API and format mappings.

Native UI's `Context.WriteClipboardThen` writes its text with `WriteData`
and posts the error to the main thread for the next frame. A browser's
clipboard answers later: the web backend's is a `platform.LaterClipboard`,
whose `WriteTextThen` keeps the text for the page's own pastes and
answers once `navigator.clipboard.writeText` settles (it rejects without
transient user activation, and outside secure contexts there is no
`navigator.clipboard` to write).

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

A player that preserves pitch (`SetPreservesPitch`) gets a stretcher
(`audio/stretch.go`) between its source and its resampler, which then
resamples at the rate alone. It is WSOLA: hops of about 16 ms, each a
Hann crossfade from the natural continuation of the last segment into a
new one, which the speed puts a hop times the speed further on and a
search within about 10 ms moves to where the waveforms match best
(normalized cross-correlation in mono, every frame of about 16 kHz, then
each frame around the best; the natural continuation itself when in reach,
so at 1× it passes the source through). Its buffers are allocated once,
the first time; it reads ahead what a step needs, skipping what fast speeds
leave out, reads silence past the source's end and fades out once the
speed puts the next segment there, and tags each
frame it gives with the source frame the speed puts it at, which the voice
keeps beside its four frames of history for `Position`. Turned off, it
passes the source through from the natural continuation, and a voice keeps
it, so that voices that never preserved pitch pay one nil check.

`Position` is where a voice is heard, not mixed. The mixer counts the
frames it mixed and, of the latest, those the device holds: as many as it
was written, up to its latency, from when it opened (`setHeld`). Each
voice marks the output's frame and its own source frame where each mix
starts and ends, and forgets the marks the output played; `Position`
interpolates them at the frame heard, which carries speed changes, a
stream waiting for its decoder, and a pause playing out. A seek or a
stop drops them, so the position is the target at once and stands there
until the output plays what it mixed since.

The output (`audio/output.go`) is a goroutine locked to one thread that
opens the device when a voice starts, mixes 10 to 21 ms at a time and
writes it with a call that blocks while the device holds enough, and
closes the device after five idle seconds. `audio.SetLatency` chooses how
much the device holds ahead, 40 ms or 250 ms; the output opens it again
when that changes. Each platform gives one `openDevice`, which takes it:

- macOS: an Audio Queue of buffers of 512 frames, as many as hold that (4
  or 24), started once all hold audio; its callback, created once, hands
  buffers back over a channel. A theme change can keep every thread of an
  app waiting for up to 0.2 s, which 4 buffers do not ride out.
- Linux: `pa_simple_write` of PulseAudio's simple API (`libpulse-simple`,
  which PipeWire serves too) with that as its target, else ALSA's
  `snd_pcm_writei` on `default`, both through purego.
- Windows: WASAPI in shared mode, event driven, with a buffer of that,
  `AUTOCONVERTPCM` and the default-quality resampler; it compares the
  default endpoint's ID every 500 ms and fails the write when it changed,
  which reopens the device.
- Browsers: an AudioWorklet from a Blob URL that queues chunks of 1024
  frames (three, or twelve) and posts a request whenever it used one; the
  output waits for a request before mixing the next, and asks more or
  answers fewer when the latency changes. One AudioContext serves the
  page, with the `latencyHint` it began with; closing suspends it, and the
  first pointer or key input resumes it.

Where `openDevice` fails, a null device takes the frames on the wall
clock and fails after two seconds, so the output tries the device again;
`audio.Err` reports the failure. Formats register a magic prefix and a
constructor as `image` does: WAV is built in, `audio/vorbis` and
`audio/mp3` wrap pure Go decoders.

## Watching files (`watch.go`)

`WatchFiles` keeps what it saw of its directory (`watch_scan.go`): for
every file and directory, by path, a `fileWatchEntry` of 40 bytes without
pointers (the identity `os.SameFile` compares, size, modification time and
mode) rather than its `os.FileInfo`, whose `syscall.Stat_t` the collector
would scan: 220 to 260 bytes an entry, with the path and its name in its
directory's set. It is read through an `os.Root` so that a path can't lead
out of it, and symlinks are not followed.
Each directory is opened as an `os.Root` of its own from the one that has
it, and what is in it is looked at by name relative to it, one call each,
as watchman and @parcel/watcher crawl with `openat` and `fstatat`: a
root opens every directory of a deeper path in turn. On Unix, reading a
directory opened in a root stats its entries already
(`fileWatchEntryInfo`). Looking at everything again lists each directory
from the one that has it. A directory replaced by a symlink, which a root
follows only within itself, is a replacement, wherever it leads.
The backend's watcher (`platform.FileWatch`) is given targets, the entries
the core enrolls, and reports notices: a target, or a child of a directory
target by name, changed. A notice says where to look, not what happened:
after the debounce, the core lists again the directories that notices
named and stats the files, and the events are the differences between what
it sees and what it saw (`fileWatchDiff`). An entry whose identity
(`os.SameFile`) moved is a rename, unless a hard link makes it ambiguous,
and what moved with a directory renamed above it says nothing more. A
file replaced at its path by another file, as an atomic save replaces it,
is a write, not a removal and a creation: apps that follow a document
would take it for deleted. The
entries are paired by looking their identity up (`fileWatchIDs`: device
and inode, or on Windows the volume and file index `os.SameFile` reads),
so a directory of n files renamed costs n lookups, not n² comparisons;
an entry whose identity can't be read is compared with every other one. When
the system lost changes (an inotify queue overflow, Windows'
`ERROR_NOTIFY_ENUM_DIR`, FSEvents' `MustScanSubDirs` or a dropped event),
the core lists every directory again rather than stopping. Enrolling is
checked against the scan: the backend opens a target
and compares its identity with what the scan saw, then the scan stats it
again, so that an entry replaced in between (an editor's atomic save) is
looked at again rather than watched in place of the other.

A watched file is watched through its directory, enrolled as `..` with
the file as `.`: the scan's `os.Root` is the directory, and notices of the
directory naming other entries are dropped. A file replaced at its path is
reported as written. `Ignore` is asked from the directory listing's
type, before an entry is enrolled, so an ignored directory is never opened;
the directories it ignored are kept per listed directory, so that the
notices of a `Subtree` target about what is in them are dropped.

`HintsOnly` swaps the scan for `fileWatchHints` (`watch_hints.go`), behind
the same `fileWatchLooker` the watch's goroutine drives: no snapshot, only
the directories it enrolls. A notice naming an entry hints at the
directory holding it; one naming an enrolled directory, or the directory
itself, has the next update check that it is still what was enrolled and,
if not (moved, removed, replaced), drop the targets under it; a name that
may be a new directory is enrolled with what is beneath it. An overflow
hints at `.` and enrolls the tree again. Hints under one another collapse
into the shallowest. On Linux the directories are listed with `getdents`
types (`watch_hints_linux.go`), as notify walks them, so files are never
stat'ed; elsewhere a recursive watch is the root's `Subtree` target alone.
It is opt-in because it gives up what the snapshot is for: exact events,
renames, writes told from metadata, and which file.

An entry the scan can't look at, a directory without permission or a
file Windows holds (`pagefile.sys`, a sharing violation), is left as it
was seen, if at all, and what is in it unwatched, rather than ending the
watch; only the watched directory itself must be readable.

`Next` returns a batch, the events of one update. An update doesn't run
while a batch waits for `Next` (`taken` tells the goroutine when one was
taken): notices still mark what to look at, a set that falls back to
looking at everything past `MaxEntries`, so a slow reader loses nothing
and holds one batch. A directory gone ends the watch with a `FileRemove`
of `.` and `io.EOF`, as `Close` and the watch's context do.

The watch's goroutine is the only user of its scan and of its backend's
watcher, whose methods `internal/watchdriver` runs on a goroutine of its
own: it adds and removes targets between waits for the system, which
`Wake` interrupts, and calls the core's `notice` with what the system
reports; `notice` hands it to the watch's goroutine over a channel, and a
full channel makes the watch look at everything again. Neither uses the
main thread, so watches work before the app runs and don't wait on it.

- macOS: an FSEventStream per directory target, which covers the whole
  tree beneath it, so nothing is opened per file, and a file target is
  covered by the stream of its directory. A target below a stream already
  watching its tree needs no stream of its own, which keeps a watch of a
  directory that isn't recursive at one stream. The stream names what
  changed (`kFSEventStreamCreateFlagFileEvents`) and hears of the root
  itself (`kFSEventStreamCreateFlagWatchRoot`), delivering through one
  callback created at startup, on one serial dispatch queue of the
  process, never the main thread. `Subtree` targets report paths at any
  depth; one without it drops what changes deeper than its entries. The
  stream's path is resolved with `realpath`, as FSEvents names what is
  on disk, which a path given on a case-insensitive volume may not. The
  notices are hints: `MustScanSubDirs` or a full queue makes the core look
  at everything again, as an overflow does elsewhere, and the stream's
  10 ms latency is under the core's debounce, which does the coalescing an
  app sees; with `kFSEventStreamCreateFlagNoDefer` the first event after a
  quiet period comes at once, and those that follow it within the latency
  are batched. The stream starts `SinceNow`, but fseventsd can report a
  change made just before it started; the core's snapshot is taken as
  targets are enrolled, so nothing comes of it.
- Linux: one inotify instance for the process, as fsnotify, with a watch
  per directory, which names the child that changed: watches cost watch
  descriptors, not instances, of which many systems allow 128 per user
  for all apps. A goroutine reads it through the runtime's poller and
  queues each watch's notices, which its driver's `Wait` returns. The
  same directory added twice, by one watch or several, shares a watch
  descriptor, and a removed descriptor's events still queued before its
  `IN_IGNORED` must not be taken for a new one's, so the backend keeps
  generations of targets, each its watch and ID, per descriptor; the
  last target of a descriptor removes it.
- Windows: `ReadDirectoryChangesW` on overlapped handles, completing on a
  completion port per watch. Windows refuses to rename a directory while a
  handle is open on something beneath it, so a recursive watch is one
  `Subtree` target on its directory, which reports paths; other backends
  return `ErrUnsupported` for `Subtree` and get every directory. The
  directory's own handle says nothing when it is renamed, so the core also
  watches its parent and looks at the directory when the parent names it.
  The handles can't be given to the os package, which would tie them to the
  Go runtime's completion port. A handle closed still completes its I/O,
  which writes to its buffer, so a session lives until its completion.
  A drive's root has no parent to watch, and can't be renamed. The
  directories that `Lstat` takes for directories are watched, which leaves
  out junctions and mount points but not the placeholders of cloud files
  (OneDrive), reparse points that are no name surrogates.

Backends report a write (`FileWatchWrite`) only for content: FSEvents'
`ItemModified` on a file (its `ItemCreated`, which APFS also sets on an
`O_TRUNC` write, stays dirty and lets the core's comparison report the
change), inotify's `IN_MODIFY` and `IN_CLOSE_WRITE`,
and Windows without `FILE_NOTIFY_CHANGE_ATTRIBUTES`. Attribute changes are
dirty, and the core compares only size and modification time, so that
indexers and antivirus touching metadata report nothing.

## Documents (`document*.go`)

`NewDocuments` follows NSDocument and NSDocumentController for the
bookkeeping and SwiftUI's 2026 `Document` for the shape (a snapshot on the
main thread, the I/O on goroutines), with the recovery copies of VS Code
and Zed. Everything is in the core; backends only show a window's
document (`Window.SetRepresentedFile`, `SetDocumentEdited`) and replace
files (`Backend.ReplaceFile`). The menus reach documents only through the
functions the first kind registers (`linkedDocuments`), so that apps
without documents do not link them through the File menu's roles and Open
Recent (about 290 KB).

- *State.* `Document[T]` embeds `docBase`, the state that doesn't depend on
  the content's type, under one mutex: a change counter (`changes`), the
  change the file has (`savedAt`) and the recovery copy (`backedUp`), the
  edited flag, and the file's stamp (modification time and size) as last
  read or written. `docOps` is what the generic half gives the rest:
  `capture` takes the content and its change (on the main thread with a
  `Snapshot`), `markSaved` makes a written snapshot the saved one (with
  `Equal`, the document stays edited only if the content differs), so
  edits made during a write stay unsaved, as NSDocument's change count
  tokens and Zed's `did_save` keep them.
- *Writes* of a document's file and of its recovery copy hold its `io`
  mutex, which checks of the disk take too, so a watch never mistakes the
  document's own write for another app's. `writeFileSafely` writes a
  sibling, flushes it and replaces the file through the backend
  (`NSFileManager`, `ReplaceFileW`, rename with mode and owner on Linux),
  in place for hard links and folders the user can't create files in.
  Save and autosave stat the file first: a stamp other than the one known
  is a conflict.
- *Pacing.* One timer per document fires after the edits stop: the
  autosave delay for a document saved in place, a second for a recovery
  copy, shortened so that a document edited without a pause is written at
  least every 30 or 5 seconds (`dirtySince`).
- *Recovery copies* live in `PathUserData/Recovery/<kind>/<session>`, a
  session per running instance with an index of the copies, locked with
  `flock` or `LockFileEx` while the instance runs; at launch an instance
  takes the sessions whose lock it gets, reopens their documents edited
  (in conflict when the file's stamp changed), writes them into its own
  session, then removes only the old copies it restored. A copy it skips or
  cannot read, and one whose new copy cannot be written, stays for the next
  launch.
- *Watches.* One `WatchFiles` per file of open documents, shared by
  reference count, which costs the same whatever else its folder holds: a
  watch covers the folder, not each file in it. A batch
  makes the document look at the disk (`checkDisk`), as focusing its
  window does, and so does the watch once it started. A removal looks in
  the folder for the file's identity, to follow a rename.
- *Closing and quitting* never nest an event loop: `OnClose` of a
  document's window prevents a close that must save or ask, does it on a
  goroutine (dialogs are sheets on macOS), then destroys the window. The
  quit's `OnBeforeQuit` listener prevents a quit with edited documents,
  saves them in place, writes their recovery copies or asks, then calls
  `App.Quit` again (`docQuit.resolved` lets that one through), and
  windows closing during a quit no longer ask.
- *Launch.* `whenLaunched` runs once the launch arguments were delivered:
  the kinds restore their recovery copies, the files queued by
  `OnOpenFile` open, and then, with no document open, an untitled one.
- *Menus.* The document roles go to `documentRole` before the window
  roles; `refreshDocumentMenus` enables them for the focused window's
  document when focus or a document's state changes, and
  `updateRecentMenus` rebuilds the Open Recent submenus from
  `recent-documents.json` (`recent_docs.go`), which `App.AddRecentDocument`
  and `ClearRecentDocuments` write, for documents and other apps alike,
  before telling the system; in an app without documents an item goes to
  `OnOpenFile`.

## Native UI (`ui`)

The public API keeps one stable `*Context` per window. Child builders share
it and temporarily change its parent. `Element` is a 16-byte checked value:
a direct owner record, arena slot and 32-bit generation. The owner detaches
from the engine on close and retires before generation wrap. Old elements
cannot alias recycled nodes. A closed window keeps nothing of what it
showed (#1281): `mygo.Window` keeps only that it shows Content, handing the
Content to its surface as it attaches, and the closed engine drops its view,
the last frame's elements, states and scene, and the host its last scene,
renderers and placements, in case anything still holds them. Development builds and `Tester` diagnose stale
use; production methods return empty results or ignore it. `Handle` stores
persistent control identity separately for each window. Focus queries read
identity without depending on construction order; focus requests wait for a
hidden control. `Services` offers persistent clipboard, URL and redraw access.

Constructors build and style controls eagerly. `Context.Key` supplies an ID
before state initialization. A control takes its input as the public
constructor that built it returns, as Dear ImGui's widgets do, so what it
binds, a local copy too, holds the edit when the next statement reads it
(`value_input.go`). The generated facade brackets each constructor with
`beginBuild` and `endBuild`; a control's queued input records how many
constructors were under way, and `endBuild` finalizes the input queued
deeper than the constructors still under way: a container's (a radio
group's arrow keys) waits aside until the call that built its children
returns, and each entry is visited once. Input queued outside public
constructors, by the engine's own controls, is finished at once, and what
remains applies after construction. Input follows the options of the build
the user acted on: a constructor resets its fluent options (`ReadOnly`,
`OnPaste`, a slider's `Step`) for this build after keeping the last build's,
which edits apply by (`editor.withShown`, `sliderSettings.shown`), and the
retained `flagDisabled` is last build's. `Changed`, `Submitted`, click and
shortcut queries apply finished input before answering.
`ComboboxParts.Chosen` applies input and reports a choice in the same
pass; it is not carried into a later rebuild. `OnChange` and `OnSubmit` run after construction and bound
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
  themselves while a frame after a pause draws at once, in the same pass
  of the run loop, from the backend's queue (`drawSoon`), as does the
  first frame of a window out of sight, which the link does not tick
  for (`platform.FirstFrameSurface`): a window made hidden draws it
  while hidden, which makes it ready to show, and a resize's
  when AppKit displays the view, before macOS 14 an `NSTimer` at the
  display's rate, `gtk_widget_queue_draw` or a tick callback of GTK's
  frame clock,
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
  of `otherMouseDown:` on macOS, and `swipeWithEvent:`'s swipes, left
  going back as in Safari, which touchpads swiping between pages send and
  mice's makers' software sends for their buttons (Logi Options+), buttons 8 and 9 and `XF86Back` and
  `XF86Forward` on Linux, and `WM_XBUTTONUP` (taken, so that
  DefWindowProc sends no `WM_APPCOMMAND` as well), `VK_BROWSER_BACK` and
  `VK_BROWSER_FORWARD`, and `WM_APPCOMMAND`'s browser commands on Windows.
  A key held down repeats as `KeyPressed` with `Repeat`, which the
  shortcut that takes it tells (`shortcutReg.repeat`,
  `Context.ShortcutRepeated`): `isARepeat` on macOS (and the current
  event's, for the Edit menu's commands that reach shortcuts), bit 30 of
  `WM_KEYDOWN`'s lParam on Windows, `KeyboardEvent.repeat` in browsers.
  GdkEventKey flags none, so the Linux surface notes the hardware
  keycodes held (`keysDown`, before the input method sees the key): a
  press of one held already, with no release between, is a repeat, as X11
  with detectable auto-repeat (which GDK turns on) and GTK's own repeat on
  Wayland send them.
  A key typed while the input method composes is the input method's
  alone, as Enter choosing a candidate: on macOS `keyDown:` sends no
  `KeyPressed` while there is marked text, as GTK's input method filters
  such keys on Linux and IMM32 makes them `VK_PROCESSKEY` on Windows.
  A composition an input method gives up, as with Escape, goes: the
  Microsoft Japanese IME sends `WM_IME_COMPOSITION` with neither
  `GCS_COMPSTR` nor `GCS_RESULTSTR` then, which empties the marked text
  (`InputComposition.Cancel`) before `WM_IME_ENDCOMPOSITION` unmarks it.
  A composition the content does not take (`SurfaceEvent` returns false
  for `TextComposition`) macOS gives up (`discardMarkedText`, once
  `interpretKeyEvents:` returns), as the terminal leaves the dead key ´
  of Option-E that it sent as Alt, so that the next key is not composed
  with it.
  Input methods name what text and compositions replace in the text
  around the caret (`Replace`, `From`, `To`): NSTextInputClient's
  replacement ranges, GtkIMContext's `delete-surrounding`, IMM32's
  reconversion (`IMR_CONFIRMRECONVERTSTRING`). Files dragged over the
  surface (NSDraggingDestination, a GTK drag destination, an OLE
  `IDropTarget`) are `FileDragOver` events, whose answer the drag source
  shows, and a drop a `FileDrop`; files the content does not take go to
  `OnFileDrop`. On Linux the area takes the touchscreen's touches
  (`GDK_TOUCH_MASK`, in its `event` signal, before GTK's `touch-event`
  handler makes the first finger the mouse), which `touch.go` turns into
  what the mouse and a touchpad do: a tap clicks where it landed, held
  still half a second it right-clicks, as Windows's press and hold; a
  finger moving past 8 DIPs scrolls (`Precise`) the content under it
  along with it, and two fingers zoom (`PointerZoom`) between them and
  scroll as their midpoint moves. Button and motion events from a
  touchscreen are dropped once the area has had touches.
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
  An element ignoring input (`IgnoreInput`: a field of the node, as
  the flags are all taken, and of its state, which the elements inside
  it inherit at commit) adds no hits nor stop of Tab and drops the focus,
  but stays in the tree, its presses counted as any other's; it offers no
  focus, nor the increments that go through the keys. One hidden from
  assistive technology (`AccessHidden`, a field of the node and of its
  inherited state, as `IgnoreInput`; inert ones count too) paints and
  takes input, but `accessElement` skips it with what is inside it, as an
  invisible one, its text names no ancestor (`innerText`), and its
  state drops what `AccessAction` sends it.
  One only unavailable (`AccessDisabled`, ARIA's aria-disabled, a
  field of the node and of its inherited state too) keeps
  `flagDisabled` off, and so its hits, its stop of Tab and its look,
  but its node is `AccessDisabled`, which the backends already map
  (`AXEnabled`, ATK's enabled and sensitive states, UI Automation's
  `IsEnabled`), still `AccessFocusable`, with no actions but the focus
  and scrolling into view, and its state drops the others; UI
  Automation's `SetFocus` reaches a disabled element that offers the
  focus, as Chromium's does one aria-disabled.
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
  `ActiveDescendant`, `Busy`, `Current`, `AutoComplete`,
  `PositionInSet`). `AccessSelected` is `Selected` without its accent
  background and text color: it sets the same state (`checked`, which
  is `AccessChecked`, read as selected on tabs, options, rows, cells and
  tree items), so every bridge, a table's or a grid's choice of cells
  and rows included, reads it as it reads `Selected`'s. `PositionInSet`
  (`accSet`, aria-posinset and aria-setsize, each 0 for not given) wins
  over the `setPos` and `setSize` the bases give their items, which
  count those built, but not over a List's rows, whose place among all
  its rows also places them in their table on macOS and Windows. `Busy`
  (`AccessBusy`, ARIA's aria-busy) is WebKit's `AXElementBusy`, with `AXElementBusyChanged`, on
  macOS, ATK's busy state, and `busy=true` in UI Automation's
  AriaProperties; `Current` (`AccessNode.Current`, aria-current's
  kinds) is WebKit's `AXARIACurrent`, ATK's `current` attribute with the
  active state, as Chromium's, and `current=page` in AriaProperties. Both
  bring a generic element into the tree as a group, as ARIA's global
  states do. `AutoComplete` (`AccessNode.AutoComplete`,
  aria-autocomplete's inline, list and both; none writes nothing) is
  WebKit's `AXAutocompleteValue`, answered through
  `accessibilityAttributeValue:`, ATK's `autocomplete` object attribute
  with the supports-autocompletion state, as Core-AAM maps it (and as
  IAccessible2's object attribute and state), and `autocomplete=list`
  in AriaProperties, which UI Automation has for ARIA's properties
  without one of its own: Core-AAM maps nothing in UI Automation nor in
  AX API, and Chromium nothing in UI Automation; `Combobox`,
  `Autocomplete` and a `TokenField` with suggestions set it to list.
  `RoleDescription` (`AccessNode.RoleDescription`,
  aria-roledescription) is AppKit's `accessibilityRoleDescription`,
  which elements override, ATK's `roledescription` attribute and UIA's
  LocalizedControlType; `Controls` (`AccessNode.Controls`, the IDs of
  the elements, aria-controls) is `AXLinkedUIElements`, ATK's
  `controller-for` relation and `controlled-by` back, from the nodes'
  `ref_relation_set`, and UIA's ControllerFor; `DescribedBy`
  (`AccessNode.DescribedBy`, aria-describedby) adds the text of the
  elements to `Description` (their Label, else the texts inside them,
  hidden or not, joined by spaces, after the element's own), and is
  ATK's `described-by` relation with `description-for` back, and UIA's
  DescribedBy beside FullDescription; AX API has no relation for it,
  only the text in `AXHelp`. `ErrorMessage` (`AccessNode.ErrorMessage`,
  aria-errormessage) is given only while the element is invalid
  (`Invalid` sets `AccessInvalid` alone, or `Error`), puts the text
  first in `Description`, as `Error` does its message, and is WebKit's
  `AXErrorMessageElements`, answered as `AXInvalid` is, ATK's
  `error-message` relation with `error-for` back, and UIA's
  ControllerFor, as Core-AAM maps it; UIA also raises
  IsDataValidForForm and FullDescription changing, as a field turns
  invalid as it blurs. `Orientation` (aria-orientation) sets
  `AccessVertical` or `AccessHorizontal` on any element: `AXOrientation`,
  ATK's vertical or horizontal state and UIA's Orientation, a slider and
  a separator across unless told, others without one. `Search` on a
  text input sets `AccessSearch`: `AXSearchField`, ATK's
  `xml-roles:searchbox` and `text-input-type:search`, UIA's Edit with
  "search box" and the searchbox AriaRole. `RoleDeletion` and
  `RoleInsertion` (ARIA's deletion and insertion) are `AXGroup` with
  `AXDeleteStyleGroup` and `AXInsertStyleGroup`, ATK's content deletion
  and content insertion with `xml-roles`, and UIA's Text named
  "deletion" and "insertion"; `HasPopup`
  (`AccessNode.HasPopup`, aria-haspopup) is `AXPopupValue`, answered as
  `AXInvalid` is, with `AXShowMenu` pressing the element, ATK's
  has-popup state and `haspopup` attribute, and UIA's ExpandCollapse
  with `haspopup=` in AriaProperties; `Sort` sets a header's
  `AccessSortAscending`, `AccessSortDescending` or `AccessSortOther`
  (ATK's `sort:other`, UIA's ItemStatus and `sort=` in AriaProperties,
  AppKit's unknown direction). A cell (`RoleCell` or a header) that
  `Checked` or `Selected` marks is `AccessSelectable`, chosen while
  `AccessChecked`, as ARIA's gridcells with aria-selected, and the
  table around it `AccessSelectable` too: `AXSelected` and the table's
  `AXSelectedCells` (with `AXSelectedCellsChanged`), ATK's selectable
  and selected states rather than checked, UIA's SelectionItem in the
  table's Selection. `RoleNavigation`, a landmark (`Breadcrumbs`), is
  `AXGroup` with the subrole `AXLandmarkNavigation`, ATK's landmark with
  `xml-roles:navigation`, and UI Automation's Group with the navigation
  LandmarkType and AriaRole. Native UI in browsers has no accessibility
  tree yet (`UpdateAccessibility` is empty in `internal/web`), so there
  is no ARIA to map them to. The other landmarks, live regions and rules
  map as WAI-ARIA's Core Accessibility API Mappings do: `RoleRegion` is
  one only with a name (a group without), `AXLandmarkRegion`, ATK's
  landmark with `xml-roles:region`, and UIA's custom LandmarkType;
  `RoleSearch` is `AXGroup` with `AXLandmarkSearch`, ATK's landmark
  with `xml-roles:search`, and UIA's Group with the search LandmarkType;
  `RoleArticle`, no landmark, is `AXGroup` with `AXDocumentArticle`,
  ATK's article, and UIA's Group named "article";
  `RoleLog`, `RoleTimer` and `RoleAlert` are `AXGroup` with
  `AXApplicationLog`, `AXApplicationTimer` and `AXApplicationAlert`,
  ATK's log, timer and notification, and UIA's Group named "log",
  "timer" and "alert"; `RoleSeparator` is `AXSplitter`, ATK's separator
  and UIA's Separator, vertical when taller than wide (`AccessVertical`,
  with `AXOrientation`, ATK's vertical state and UIA's Orientation);
  `RoleRowHeader` is `AXCell`, ATK's row header and UIA's HeaderItem,
  named by its content but keeping what is inside it, as a cell does
  (ARIA's rowheader is not children-presentational).
  Live regions (`AccessRole.Live`) say how they are read: `AXARIALive`,
  `AXARIAAtomic` and `AXARIARelevant`, answered as `AXInvalid` is; ATK's
  `live`, `atomic` and `relevant` attributes, with their `container-`
  forms on what is inside them; UIA's LiveSetting. Screen readers read
  the platforms' live regions in web content alone (WebKit's and
  Chromium's `AXLiveRegionChanged`, Orca's web scripts), so the engine
  reads them out itself (`ui/access_live.go`): it compares each frame's
  tree with the last, announces what a log that showed gained, joined
  for each log, and puts the text an alert shows that no alert showed
  in the last tree into `AccessTree.Alerts`, which interrupt:
  `AXAnnouncementRequested` with a high priority, ATK's `notification`
  signal with `ATK_LIVE_ASSERTIVE` (ATK 2.50, else an announcement), and
  an assertive live region beside the announcements' on Windows. Alerts
  are told apart by their text, as an element's ID shifts when others
  appear before it; the tree assistive technology first gets, and a log
  first showing, read nothing out. Menus drawn in the window are AppKit's `AXMenu`,
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
  The keys that press or focus an element (`KeyShortcuts`, ARIA's
  aria-keyshortcuts) are formatted as `ui` writes a menu's shortcuts
  (`shortcutText`) as the view builds, so that an app not giving any
  links no formatting, and reach the backends as one string
  (`AccessNode.KeyShortcuts`): WebKit's `AXKeyShortcutsValue`, answered
  through `accessibilityAttributeValue:`, ATK's `keyshortcuts` object
  attribute and UI Automation's `AcceleratorKey`, as Chromium's.
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
  `AXVisibleRows`, `AXSelectedRows` and `AXSelectedRowsChanged` (a
  list without a `SetSize` or `RowCount` of its own counting as many rows as its
  items' `SetSize` says, if more than it holds), and
  `AXScrollToVisible`, an action of AppKit's older API, which elements
  then answer for all their actions; on Linux, a list choosing rows is a
  list box with `AtkSelection` and `selection-changed`, rows carry the
  `posinset` and `setsize` object attributes (as any item's), and `AtkComponent`'s
  `scroll_to` (ATK 2.30) scrolls; on Windows, rows have `PositionInSet`,
  `SizeOfSet`, `IsOffscreen`, `SelectionItem` (raising `ElementSelected`)
  in their list's `Selection`, and `ScrollItem`. A list choosing several
  rows is `AccessMultiselectable` (ATK's `multiselectable`, UIA's
  `CanSelectMultiple`, rows then raising `ElementAddedToSelection` and
  `ElementRemovedFromSelection` unless chosen alone), and its focus is on
  the row last chosen. `RoleListBox` and `RoleOption`, a list box of an
  app's own (ARIA's `listbox` and `option`), are `RoleList` and
  `RoleListItem` with `AccessSelectable`, as a `ComboboxBase`'s popup and
  its options (`choosesItems`, `flagChoosable`) and a `SelectBase`'s
  are, so the bridges show them as the lists above, selected with
  `Checked`; unlike `flagChoosable`'s rows, they take the focus only
  when `Focusable`. A `SelectBase`'s options are selected while
  highlighted, the trigger's `ActiveDescendant` the one highlighted, as a
  native select's popup: assistive technology focusing one
  (`focusHighlights`) notes it on its state (`focusAsked`), which the
  select highlights as it builds it, rather than clicking it, which
  would choose it. Grids and tree grids (`RoleGrid`, `RoleTreeGrid`)
  are tables to the backends (`AccessRole.Tabular`), whose rows a `List`
  shown as one makes rows rather than list items; a row of an app's own
  saying whether it is chosen (`Checked`, `Selected`) is
  `AccessSelectable`, as is the table or grid holding it, which holds
  its cells' choice too. The counts and indexes an app gives a grid built only where it shows (`RowCount`,
  `ColCount`, `RowIndex`, `ColIndex`, ARIA's aria-rowcount and the like)
  go to WebKit's `AXARIARowCount` and the like, and the rows' `AXIndex`
  and the table's `AXRowCount` where `PosInSet` and `SetSize` say
  nothing; to ATK's `rowcount`, `colcount`, `rowindex` and `colindex`
  attributes beside `xml-roles` `grid` or `treegrid`, as Chromium's; and
  to UI Automation's `Grid`, `GridItem`, `Table` and `TableItem`
  patterns of tables and their cells (`uia_grid.go`), which count from 0
  as the app gave them, else as rows say which of all they are after the
  header rows above them, else by what is built, found as a client asks;
  their `RowHeader`s are the `Table` pattern's row headers. A cell's
  `ColumnSpan` and `RowSpan` (`node.cellSpan`, from the layout's
  `gridCell`, whether or not its row is a grid) are `AccessNode.ColSpan`
  and `RowSpan` (aria-colspan, aria-rowspan): the lengths of its
  `AXColumnIndexRange` and `AXRowIndexRange` (`placeCells`, which puts
  the next cell after the columns it covers), ATK's `colspan` and
  `rowspan` attributes, and `GridItem`'s `ColumnSpan` and `RowSpan`,
  `Grid.GetItem` finding the cell at each place it covers.
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
  fractional and content-sized tracks, with the items' percentages of
  their grid areas, as auto while the tracks are sized), laying out again
  only what changed since the last frame (see below); commits the boxes to the elements'
  states with the hit list in paint order, the focus order and the labels;
  paints a `scene.Scene`; and presents it. Input between frames goes to the
  states of the last frame's elements. A release that comes with moves no
  pass read waits in the element's state (`releaseLater`) for the pass
  that reads them, which applies it as it ends (`forgetInput`): the view
  sees the element still pressed where the pointer let go, and released
  in the pass after. An element's identity hashes its
  parent's with its position or `Key`, so focus, scroll offsets, editors and
  animations survive rebuilding. `prune` drops the states the frame did
  not build (but those of pages Routers keep) and recycles them; a state
  with `Local` values that are `io.Closer`s (`state.closes`, which
  `Local` fills as it makes one, so that other states cost nothing) hands
  them to `engine.dropped`, which `endFrame` closes on the main thread
  once the frame ended, last made first. Those of an element whose copy,
  or an ancestor's, leaves with an `Exit` wait in `engine.exits` until
  the transition's record is gone or no longer holds that copy (the
  exit ended, or the element came back), and the engine's `close` closes
  every state's as the window closes. `Context` is stable for the window; `Element` values expire between build
  passes and validate their owner, slot and generation before accessing storage. `ListState` resolves
  its focus owner only for the current context, frame and pass, exposing
  focus and shortcuts without retaining an element in app state. Scroll
  offsets move in the layout too
  (`ui/scroll.go`): elements that asked to `ScrollIntoView`, and the focus,
  come into view before the boxes are placed, lined up as its options say
  (or glide there as a wheel's notches do, by what they went each frame,
  `glideScroll`), and placing keeps each offset within its content; a
  `ScrollState` mirrors an offset both ways, and a frame that moved one
  builds another for what read the old one. A container that snaps
  (`ScrollSnap`, `ui/snap.go`) finds the places its elements mark as it
  is placed, from their boxes relative to it, and
  keeps the offset at the one it rests at; a notch or a key heads for the
  next place at once, a touchpad's scrolling once it has been still for a
  moment (not every platform tells when the fingers lift or a fling's
  momentum ends), along a spring that the frames advance (`glideSnaps`).
  Only `ScrollSnap` installs what snapping takes (`placeSnapper`, the
  `snapper` interface of the container's state), so apps that do not snap
  leave it out. Frames
  happen only when asked: input, `Invalidate`, `After`, or `AnimationFrame`
  while something moves.
  The view builds the whole tree each frame, but layout reuses what did not
  change (`ui/layoutcache.go`), as Blink reuses the layout result of a box
  not marked as needing layout in the same constraint space and Flutter
  skips a render object whose constraints are those of its last layout.
  Nothing marks a rebuilt tree, so before laying out `sameLayout` compares
  each element with what its state kept of it from the last frame: its
  `layoutStyle` (the fields of `node` that lay it out, embedded in it), the
  flags and inherited text styles the layout reads, its grid's tracks, its
  spans, its image's size, and its children, the same elements in the same
  order, each the same in turn. `layoutBox` then reuses the last layout of
  an element that is the same when it gets a box of the same size: its
  text's layout and its children's boxes, laying out again any child laid
  out at another size since; what was measured of it (its content widths
  and heights) is reused too, so that a container laid out again measures
  its unchanged children at once. The state of a text element also keeps
  the widths of its text and its last two layouts (`heldText`), found
  again by their parameters, as Taffy keeps a node's measurements: an
  element laid out anew whose text is the same, as the rows of a List
  each frame, asks nothing of the text system's cache, whose key costs
  more to hash than to compare, and one laid out at another width asks
  for it from the last (`text.System.LayoutFrom`). A grid measures the
  widths of only the items its columns read (`contributes`): none for Fr
  columns with room known, as CSS's minmax(0, 1fr), unless they are
  `MinContent`; a minimum (`Track.Min`) measures nothing. Transforms and scroll
  offsets of other containers apply after layout and change nothing
  there. Elements that lay
  out by more than they are built with are never reused, nor anything
  around them: a List (rows built as it lays out), a Form, a Table's
  fitted columns and a toolbar (which change elements inside them as they
  lay out, so neither is anything inside those), an input, an element
  attached to another. A field added to `node` must say whether layout
  reads it (`TestLayoutInputs`); `TestLayoutCacheMatches` lays random trees
  out with and without the cache (`engine.noLayoutCache`) as they change
  at random, frame after frame.
  A frame asked for only by drawings that move (`Painter.AnimationFrame`,
  `Painter.After`, as spinners and progress bars of unknown length do)
  paints the elements of the last frame again at its own time
  (`repaintFrame`), without building or laying out; elements out of view
  are not painted, so they ask for none. Nor are the others, mostly
  (`ui/repaint.go`): the frame painting the window keeps, for each Draw or
  DrawOver callback that asked for frames, the operations it painted and
  the painter as it was (`paintPart`), and a frame that only paints again
  runs those callbacks alone and splices what they paint into the last
  frame's scene in place of what they painted (`repaintParts`). It paints
  the window whole when anything else asked (a pose moving, a drawing
  painting into a layer or a canvas, as `engine.untracked` notes), while
  the inspector shows or a drag moves, and once the parts have added more
  glyphs, stops and words than `partsGrowth` past what the whole frame
  held. `TestRepaintPartsMatchWhole` compares the frames with those
  painted whole (`engine.noParts`). Such a frame is asked for with
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
  the last frame built armed stay; the host arms them (`afterTimer`,
  `repaintTimer`), and `Painter.After`'s posts to the main thread.
  A drawing asking for fewer frames (`Painter.AnimationFrameRate`) has
  those frames come every so many of the display's refreshes, the fewest
  that give it as many frames a second at least (`engine.pace`), unless
  anything else in the window asks for each: `Painter.AnimationFrame`, a
  pose moving, the view animating. The refreshes between draw nothing
  (`paceSkips`) unless something else asked for a frame since, and the
  surface is told of the rate (`platform.FrameRateSurface`): macOS sets
  its display link's `preferredFrameRateRange`, which has a ProMotion
  display refresh as often while nothing else on it asks for more. A
  surface whose frames asked for come at once rather than at the next
  refresh (`platform.ImmediateFrameSurface`: Windows, where presenting
  paces frames) would ask again and again until the frame is due: the
  repaint timer asks for it then instead, and the frame it brings keeps
  the pace's times, as the timer fires a little late. With the display's
  rate unknown (browsers), frames come at each refresh.
  While nothing of the window shows (`platform.OccludableSurface`: a macOS
  window hidden, minimized or covered by other windows, whose display link
  still ticks, at the display's rate for half a minute, then at about 40
  Hz; a minimized window on Linux, as GDK's `window-state-event` tells
  on X11, and on Windows, as `WM_SIZE` does), what moves asks for no frame, and a frame asked for by a change of
  the state waits (`holdFrame`; both `held`), as browsers draw nothing for
  windows out of sight and as Apple asks of Mac apps: the view function,
  `Draw` and the work `After` drives wait until some of the window shows.
  `SurfaceShown`, sent as some of the window shows again
  (`windowDidChangeOcclusionState:`), draws what changed, and has a held
  animation go on from where the time puts it; a video (`VideoPlayer`)
  goes on from where it stood instead, counting its time from then
  (`engine.shown`), as one the view stopped building does. Its frames
  decode on a goroutine, two ahead, into frames handed back for it to
  decode into again; the view takes the last frame due, passing over the
  others, has the bitmap show it (`Bitmap.Update`), and asks for the next
  with `After`, or waits for the goroutine to wake the window (`waiting`)
  when it has no frame due yet. The window shows its last
  frame until then, about a refresh: captures of it meanwhile, as sharing
  its window in a video call, see that frame. A frame still builds after
  input (`input`, as keys typed into a key window another app's covers),
  as assistive technology turns on, and for `CapturePage`, which builds
  the frame waiting; and none waits before the window's first frame
  reached its surface (`windowHost.occluded`), which a window made hidden
  waits for to become ready to show. `Context.WindowHidden` reads
  `occluded`; once a view asked (`hiddenAsked`), `SurfaceHidden`, which
  the surfaces send as none of the window shows any more, builds one
  frame out of sight, as input does, for the view to stop its work, and
  `SurfaceShown` one whether a frame was held or not.
  The window gaining or losing the keyboard focus draws a frame only when
  the last frame read it (`windowFocus`, as a focus ring, a caret, a
  selection's color or a focused text input's IME state do): a window
  hiding or showing again draws nothing it would draw the same. So does
  the window becoming active or inactive (`windowActive`, which the
  widgets read through `context.emphasized` and `context.accent` where
  they would fill with the accent, and `Context.WindowActive`), told by
  `surface.Conn.ActiveChanged`. Each
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
  layout space. A `scene.Affine`'s divisor is 1 at its origin, or -1
  (`Behind`) when the origin is behind the viewer though what is drawn is
  in front: a turn in perspective around an element far from the
  window's origin puts the window's origin past the turned plane's
  horizon (#1406). Products and inverses keep the divisor's sign, and the
  shaders take it from the instance (`Transform2.z`). Elements and
  `Painter.Transform` flatten a 3D transform around its own origin and
  move it there in 2D. A transform that puts all of its plane behind the
  viewer (`TranslationZ` as far as `Perspective`'s distance, not turned)
  flattens to a singular map, which shows nothing; `BackfaceHidden` tests the normal through the whole 4×4
  inverse, perspective included (`facesAway`, as Chromium's
  `IsBackFaceVisible`). `TranslatePercent` stays apart from the transform
  (`translatePct`): `transformAt` resolves it from the box as laid out,
  after the transform and before the pose, as a pose's percents
  (`showPose`) and a `Motion`'s (`tbox.moved`) are. Painting stores the composed transform on scene
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
  moving within them, once they hold content that a transform does more
  than move (`scene.Reshaped`), or a clip reaching into their rounded
  corners (`scene.ClipCorners`, which the scissor rectangles of outer
  clips would not cut); hard clips in the drawing's coordinates,
  and clips around content only moved, evaluate on the GPU. Backdrop
  effects map local samples into the frame. With a GPU, every frame draws
  there, including resting changes to transformed content.
  Metal retains wide colors under transforms. See
  [Transforms](ui/transforms.md#rendering) for rendering details.
- **Masked clips** (`ui/mask.go`, `scene.ClipShapeOf`). A clip
  (`OpPushClip`) may carry a gradient paint, whose alpha masks it
  (`MaskLinearGradient` and the others, around the element's inset
  rectangle from `ClipInset`), or an image, whose alpha masks it over its
  rectangle (`Painter.ClipPath`: the path's coverage, rasterized by
  `internal/vec` once for each shape and kept by the engine while paints
  use it). The GPU builders draw such clips into the cached clip-mask
  textures, as transformed or rounded clips do (`raster.ClipMask`), and
  never evaluate them in the shaders, so no shader changed for them. The
  CPU renderer multiplies the clip's coverage by the mask
  (`raster.clipMasking`). Each op under a mask is masked on its own,
  rather than through a group layer as `Opacity` fades (below). A clip moved
  by a translation moves its gradient with it (`scene.MovedClipOp`), and
  damage compares the version of its image.
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
  `DragEnded`, and their button (`state.pressButton`, `PressedButton`):
  `Pressed`, `Dragged` and the controls that follow the pointer (sliders,
  `Draggable`, value drags) take the primary button's alone
  (`pressedPrimary`), a right or middle press only clicking or opening a
  menu. Keyframe sequences keep one sequence per
  key with an explicit replay token. All ask for frames only while moving,
  and end at their targets under reduced motion, but for those whose
  `NonMotion` (of `Animation`, `KeyframeOptions`, `TimelineOptions`) says
  they move nothing, which play on. Poses (`ui/pose.go`,
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
  `Motion`'s scale, rotation and blur) is a motion `channel`, so a spring keeps
  its velocity through changes and a delay holds them. A blur goes with
  the opacity, as it moves nothing, and adds to the element's `Blur` as
  Gaussians compose, which `paintEffect` draws on every renderer. Colors move as
  elements paint, after their `styleFn`, mixed premultiplied in sRGB, or
  in Oklab (`mixLab`) with the `Oklab` of their `Animation`. A `Match` moves an element
  appearing where another of the same identity showed the frame before
  (`ui/match.go`, `matchRec`, in window coordinates) after `place`
  (`matches.move`), laying it out anew at each box on the way, or scaling
  it (`ScaleContent`), across parents and scroll offsets, from the colors
  the other showed (`matches.claim` notes them before `animateExits`
  forgets it). With `Fly`, it blends the other's corners, opacity, text
  color and transform into its own (decomposed as CSS does, in 3D into a
  quaternion slerped the short way, whose sign holds at half a turn), and
  the transform of the
  elements around the other into that of its own (`worldAround`, shown
  with a transform of its own, `under`). The element it came from goes
  with it if it flies too, fading out as it fades in (Motion's crossfade
  curves): itself while still built, as in a page going away, else a copy
  that `matches.claim` takes, in place of its `Exit`, from the arena of
  the frame before, which frames with an element flying keep as those
  with an `Exit` do, with the text style and the theme of the elements
  around it; copies of its ancestors going away hide theirs. Both are
  `lift`ed, over the rest of their layer: the window's content, which the
  root paints them over before the overlays, or the overlay they are in,
  which its layer paints them over (`matches.paint`), out of their
  ancestors' clips and opacity, in the theme of the `Themed` around them.
  `commit` lists their hits there too (`engine.liftHits`, put over each
  layer as the next starts), where `contains` stops at them the clips
  around them, so that they take the pointer where they paint. Each is
  placed where its content sits on the pixels as at rest, and shown where
  it is with a translation (`matches.pin`), which elements flying inside
  it take into account, so that its text moves as one, smoothly, and ends
  as text at rest; a flight ends with its look, its box within a quarter
  of a point of its own. One gone with an ancestor going as its `Exit`
  says flies on as its copy in the ancestor's (`matches.flyOn`), from
  where it showed to where that copy has it, its colors going on as its
  transition moved them, fading as that copy fades; other copies of
  elements flying (`ghostOf`) no longer fly. A text flying with
  `ContentTransition`'s `Morph` takes the other's text layout into its
  content transition instead, whose `morph` (`ui/morph.go`) pairs the
  glyphs it comes from (`morphGlyph`, the old layout's, colored by the
  spans its rich text last painted with, which `morphRest` keeps) with
  the clusters of its layout (their longest common sequence, then the
  nearest of the same characters left) and moves, scales and recolors the
  glyphs of each pair, fading the others, the old text's going with the
  element's box, and the lines and highlights of both texts' styles and
  spans. It places each glyph as text at rest where the box goes would
  (`text.System.Origin`), drawn at the quarter of an octave nearest the
  size it shows, and moves it from there with a transform, so that the
  glyphs move smoothly and together, rather than each a pixel at a time,
  and end as text at rest. A text sent back to the one it morphs from
  before it got there morphs back from as far from the end as it was from
  the start, which shows the same; one that another text takes over
  morphs from its glyphs as they showed (`snapMorph`, from the same walk
  of them, `morphPairs.each`, as painting them). `ContentTransition`
  links `morph` only for a `Morph`, so that apps without one link none of
  it. A frame without an
  element on its way notes only where each `Match` showed and the element
  that showed it (`matchRec.node`), whose look and the transform of the
  elements around it an element flying from it reads in the next frame,
  from the arena kept, rather than each frame noting them. The engine has
  its `matcher` only once a `Transition` has a `Match` or `ScaleContent`,
  so that apps without them link none of it. An element appears with
  its `Enter` only in a parent the frame before had; the overlay layer,
  built only while it holds something, counts as the root there. One no longer built,
  whose parent still is, goes with its `Exit` as an inert copy, laid out
  and painted under its siblings of its z-index without taking room; the frames after
  one in the overlay layer, or its copy, build the layer, empty but for
  the copies, which lets the pointer and the focus through
  (`engine.overlayExits`). Frames that built
  elements with an `Exit` leave their arena to the next frame, which
  builds in the other (`Context.spare`), so that the next one copies those
  gone (`ghostOf`), with states of their own and nothing referring to the
  app's state. The parent's `ChildExit` in the frame it goes
  (`context.childExits`) takes the place of its `Exit`. An element
  appearing with `AfterExits` waits unseen (`transition.held`,
  `flagInvisible`) while copies leave under its parent (`ghosts`, which
  `animateExits` sets after the elements built have moved), and comes in
  with its `Enter` in the frame they are gone. Siblings are not laid out around a size moving; they move
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
  The Styles tab edits (`inspector_edit.go`): a view builds its elements
  anew each frame, so edits are kept by element ID and applied to the
  frame's elements after the view and before layout (`applyEdits`), through
  `setStyle` and `unsetStyle`, which `ownStyles` must read back; a property
  they don't know stays read-only (`styleKnown`). `describe` then overlays
  what the elements can't show: values as typed, declarations turned off or
  not understood (`overlayEdits`). Rules are keyed by their element, as the
  bar of changes comes above them. Closing the inspector drops the edits.
  `changesAsGo` (`inspector_gocode.go`) writes them as calls of package ui,
  whose methods a test looks up on `Element`, from what `setStyle` parsed.
  `:hov` forces states by element (`inspector.forced`): `Hovered`, `Pressed`,
  `Focused`, `FocusVisible`, `FocusWithin` and `ringShown` ask `isForced`,
  which the stub of `mygo_noinspector` answers with false.
  The tree's `flex` and `grid` badges show layouts over the content
  (`inspector_layout.go`). A grid's tracks exist only as it lays out, so
  `gridLayout` gives them to `noteGridTracks`, behind the constant
  `inspectorBuilt` since an empty stub would still grow its frame and code,
  with the layout cache off while a badge is on for it to run.
  Laying a grid out again after `place` would move its baseline items, which
  are in window coordinates by then.
  `main.go:42` opens in the developer's editor (`inspector_editor.go`):
  `MYGO_EDITOR`, `VISUAL` or `EDITOR` names the command, started in a session
  of its own so that it outlasts `mygo dev` restarting the app, and failures
  go to the Issues tab. Browsers and phones keep `os/exec` out
  (`inspector_editor_none.go`).
  Production builds of `mygo build` leave the inspector out with the build
  tag `mygo_noinspector` (`inspector_off.go` stands in for it, never
  open), added to the tags of `GOFLAGS`, unless `-debug` or
  `MYGO_INSPECTOR=1`.
- **Automation.** `plugins/automation` drives the app from another
  process over a Unix domain socket (`MYGO_AUTOMATION`), gated with the
  inspector. Core gives it one hook: the plugin's `init` sets
  `surface.Automate` only when the variable is set, and `ui`'s
  `AttachContent` (`windowHost.automate`, empty under
  `mygo_noinspector`) hands it each window's `surface.Conn` and a
  `surface.Automated`, which reads the last frame's access tree with
  the elements' keys (`node.key`, which `rekey` sets too) and tells
  whether a frame was asked for and has not come (`windowHost.frameAsked`).
  Only the plugin calls `Automated`'s methods through the interface, so
  apps that don't link it don't link them. Actions and input go to
  `Conn.Event` on the main thread, as the backend's events do; menu items
  are chosen through the backends' test hooks (`TestPerformMenuItem`,
  `TestActivateMenuItem`); captures and scripts through `Window`'s and
  `Page`'s methods. `mygo-automation` (`plugins/automation/cmd/mygo-automation`, its code
  in `internal/automationcli` so that `internal/e2e` runs it in process)
  is a client of the socket on Go's standard library alone: it prints
  trees as lines with refs (`@e7`) that a session's file maps to windows
  and element IDs, gives a number once per session so that a stale ref
  never names another element, and after `--settle` (`idle`, then
  `tree`) diffs the elements by ID against what the session saw last.
  Its guide, `SKILL.md`, is embedded in the command (`skills`), so that
  it matches the version installed.
- **Input taken as it comes.** An element with `HandleInput` gets its
  input on the main thread as the backend reports it, before the frame
  (`ui/handler.go`): keys (with their releases, which ui otherwise
  ignores), text, compositions and edit commands while it has the focus,
  unless a `Shortcut` around it claims the key, and the pointer pressed on
  it, moving over it or scrolling over it; once it takes a press, the
  moves and the release come to it, as to a pressed element, and the
  release comes as the window loses the focus or once a frame builds the
  element no more (not built again, or built over by an element without
  `HandleInput`, as when the press moved it into a split), so that the
  next press starts anew. Those releases, and a press the system takes
  (`platform.PointerCancel`: a browser's `pointercancel`, a pen's or a
  finger's `WM_POINTERUP` with `POINTER_MESSAGE_FLAG_CANCELED`, or
  `WM_CAPTURECHANGED` while a button is down, on Windows, which
  `WM_CANCELMODE` brings too, GDK's `GDK_GRAB_BROKEN` of the pointer on
  Linux, a context menu's loop that takes the release on macOS), are
  `InputEvent.Canceled`: the handler still gets its release, so that
  what holds a press lets go, but the engine commits nothing of it
  (`pointerUp`'s `canceled`): no click or right-click, no drop (the
  value's drag is given up, as with Escape), no `DragEnded`, no context
  menu waiting for the release, and `PressCanceled` reports it, which
  sends `Draggable` back to where the press took it. A cancel coming
  while nothing is pressed does nothing, so a backend sends it whenever
  it loses a press, but not as a data drag of its own takes the pointer
  (OLE's loop, `gtk_drag_begin`), whose end ends the press. The Linux
  touchscreen's `GDK_TOUCH_CANCEL` taps nothing (`touch.go`), as its
  taps press only once the finger lifts. The pointer's events carry their device and
  pressure (`SurfaceEvent.Pointer` and `Pressure`), which the engine
  keeps with the buttons they leave down (`engine.pointerDevice`): a
  device that tells no pressure presses 0.5 while a button is down and 0
  otherwise, as Pointer Events have it, a move that changes the
  pressure alone is a move, and the element pressed keeps the device and
  pressure of its press's last move for `PressedPointer`
  (`state.pressPointer`). Its
  `TextCaret` turns on the input method at that caret, with no text around
  it; `InputMethod(false)` (`TextInputState.NoInputMethod`) keeps text on
  but input methods off, as winit's `set_ime_allowed(false)`: macOS skips
  `interpretKeyEvents:` and inserts the event's `characters`, Windows
  disassociates the IMM context and keeps `WM_CHAR`, and GTK takes the
  focus from the IM context and sends the keyval's character; browsers
  cannot. A composition fn leaves (returns false for `InputCompose`), the
  input method gives up on macOS. `ui.Shape` lays out text without the
  cache of layouts, for widgets that keep their glyphs, and
  `Painter.Glyphs` draws them where they placed them.
  `ObserveInput` has an element see, without taking it, what comes to
  the focus while the focus is it or inside it (keys up and down, text,
  compositions, edit commands), and the presses and releases over it:
  `observeFocus` and `observePointer` (`ui/handler.go`) walk the focus or
  hit chain, outermost first, before anything handles the event, and do
  nothing while the last commit found no element observing
  (`engine.observers`).
  A text input's `OnTextKey` gets the keys that type no text as they
  come too, after `HandleInput`, but first applies the input queued for
  the editor (`widgetTextInput.prepare`, as the platform's queries do),
  so that fn reads the text typed before the key; a key it takes is
  taken as `HandleInput`'s are. Shortcuts around the focus claim keys
  first unless the editor wants them (`editor.wants`), as Enter.
  `HandleTextInput` connects a custom element's application-owned
  `TextInputClient` to the same platform services. A stable, focus-checked
  adapter supplies absolute UTF-16 text ranges, selection, marked text,
  mutation callbacks, range geometry and hit testing. It is invalidated on
  replacement/disposal, and focus changes unmark the old client and reset
  native composition. Revealing the element, as the focus does, scrolls
  to the client's caret (`clientCaret`), not the element's box, which a
  code editor as high as its lines makes as tall as the file. GTK and
  IMM32 use bounded surrounding context while
  macOS can query arbitrary document ranges directly. `ShapeText` and
  `ShapeRichText` retain per-paragraph `TextLayout` geometry, its lines
  included (`Lines`), as a paginating reader needs; an application
  owns its buffer, selections, rendering, editing and undo policy.
- **Sticky elements** (`ui/sticky.go`). `Sticky` sets `layoutStyle.sticky`,
  which keeps the insets of an element in the flow from moving it as
  relative positioning (`relative`); `place`, which reaches an element
  after its parent and the scroll containers around it, then moves it and
  what is inside it by `stickyShift`, as CSS Positioned Layout's sticky
  positioning does: per axis, the padding box of the nearest container
  scrolling that way (or following one, `ScrollWith`), or the window's,
  less the insets, is where its margin box stays, within its parent's
  content box (the content it scrolls, for a scroll container), the
  start's inset winning (`stick`). Unlike CSS, which makes a container
  scrolling one way scroll both, each axis looks for its own container,
  as one here scrolls only the ways it says. Being in window coordinates
  once placed, the box is where hits, ink, accessibility bounds and the
  focus ring read it; `laidOutOrigin`, which overlays attach by before
  `place`, and `reveal` add the same shift, so that a popover follows it
  and the focus coming to it where it shows scrolls nothing. It paints in
  its parent's second pass, among the absolute children, and commit moves
  its hits there (`stickyHits`), while it commits in the flow, so that
  Tab and assistive technology keep its place among its siblings.
- **Paint order** (`ui/zindex.go`). `ZIndex` sets `node.zIndex` and, on
  the parent, `zOrder`, which `add` sets too for a child added with one
  (a copy leaving); a parent without it paints and commits its children
  as before, in two passes, with no sorting nor allocation. One with it
  paints and commits them through `zPaint` and `zCommit`, which `ZIndex`
  sets, so that an app never calling it links none of this. It paints
  them in `paintOrder`: the two passes' order sorted stably by
  z-index, so that equal values keep it, the copies leaving, which keep
  the z-index their elements had, first among those of theirs,
  and, of the root, the overlay layer and what follows it last
  (`zPinned`), so that overlays stay above the content and keep their
  order (the overlay layer's own children are never sorted). Commit
  (`commitByZ`) commits them in the usual order, which the focus order
  and the labels follow, noting each child's hits and web views, and
  then moves them, child by child, to the same order (sorting pointers to
  the notes, whose code all slices of pointers share), the web views'
  indexes into the hits with them, so that the pointer and what covers a
  web view agree with the paint. The order is among siblings alone: a
  child with a z-index moves with its subtree, inside its parent's clips,
  and makes no stacking context that would mix it with its parent's
  siblings. The engine's `zScratch` holds the buffers, as a stack for the
  paint's nesting; the scene's order changing makes a `Cache` layer draw
  again, as its `scene.Differ` sees.
- **Right to left** (`ui/direction.go`). Each element has a direction
  (`node.dir`): built, it takes its parent's, and the root the window's,
  which `surface.Conn.RightToLeft` gives (package `mygo` from the app's
  language), unless `MYGO_DIRECTION` or the inspector's toggle
  (`engine.dirOverride`) override it; `Element.Direction` gives an element
  its own (`dirSet`), which those already inside take (`setDir`), and
  `AttachTo` and `Overlay` give what they build the direction of where it
  was asked for. The coordinate system is not mirrored, as Godot's: the
  layout places right to left. Flexbox turns a row's main axis
  (`flipMain`, which `Reverse` turns back) and a column's cross axis
  (`flipCross`), as `row-reverse` and `wrap-reverse` do; grids place items
  from the right of the content, which is as wide as the columns where its
  width is not known; absolute elements go to the right without insets,
  and `Right` wins over `Left` there and for relative positioning, as CSS
  says. Logical edges (`PaddingStart`, `MarginEnd`, `Start`, ...) write the
  physical side of the element's direction as they are set, noting which
  (`node.logical`), so that a later direction swaps them; logical anchors
  resolve as they are placed (`Anchor.physical`). A container scrolling
  sideways keeps its offset from the start (`scrollLeft` turns it into the
  physical one for placing, revealing and the bars), content narrower
  than it moves to its right, its wheel and keys negate their horizontal
  delta (`scrollBy`), and its vertical bar goes on the left
  (`scrollBars`). Text gets `text.Params.RTL`, the direction of
  paragraphs without a strong character, and lines start at the right
  (`text.Right`) whatever they read, while left to right each paragraph
  starts where it reads from, as before. Widgets read the direction as
  they build (`node.rtl`, `arrows`, `state.rtl` for input between frames)
  to draw and step what is directional, and context menus carry
  `platform.Menu.RightToLeft` to the backends (NSMenu's
  `userInterfaceLayoutDirection`, and the surface view's while the menu
  shows, as AppKit pops a menu up from its view's top right corner then,
  `TPM_LAYOUTRTL | TPM_RIGHTALIGN`,
  `gtk_widget_set_direction` and a north-east anchor, the web's `dir`).
  Accessibility needs nothing more: bounds are window coordinates.
- **Themes of a subtree** (`Themed`, `ui/context.go`). An element given a
  theme (`node.themed`) builds its children with it, as `Toolbar` flips
  `c.buttons` for its controls: `Children` sets `c.theme` and restores
  it, or the window's (`c.window`, the root's, which `SetTheme` sets) when
  the scope began at the window's theme and `SetTheme` was called inside.
  Its own text style holds the theme's color, size and family, as the
  root's does the window's, so that text inside resolves through the
  ancestors unchanged. What reads the theme after building finds it
  there too: `Painter.Theme` is that of the innermost themed element
  painting (`Painter.theme`, which canvases inherit), and the commit
  keeps a scroll container's bar width in its state (`barWidth`) for the
  input that hits its bars. Overlays opened in a scope are children of
  the overlay layer, above the root: `Overlay` gives those built while
  `c.theme` is not the window's that theme as their own. A copy leaving
  with an exit transition copies its theme.
- **Preferences.** `platform.Theme.Preferences` reads the settings of the
  desktop that controls follow: on macOS, `controlAccentColor` and
  `NSWorkspace`'s accessibility display options, with their notifications;
  on Windows, DWM's `AccentColor`, `SPI_GETCLIENTAREAANIMATION`,
  `SPI_GETHIGHCONTRAST` and Accessibility's `TextScaleFactor`, with
  `WM_SETTINGCHANGE`; on Linux, GTK's `gtk-enable-animations` and the
  settings portal's accent, contrast and GNOME's text scaling factor
  (`ReadAll`, `SettingChanged`). What the pointing devices can do
  (`Pointer`, `AnyPointer`, `NoHover`, `NoAnyHover`, Media Queries 4's
  interaction features), whose zero is a mouse's, as macOS's always is,
  comes from the media queries in browsers; on Linux from the sources of
  the default seat's pointing devices (`gdk_seat_get_slaves`), followed
  through its `device-added` and `device-removed`; on Windows from
  `SM_MOUSEPRESENT` and `SM_DIGITIZER`, the touch screen primary in
  tablet mode (`SM_CONVERTIBLESLATEMODE`, whose `ConvertibleSlateMode`
  setting changes), as Chromium tells them, primary the finest. A change
  goes through `Handler.ThemeChanged`, as the appearance's does; package `ui` reads them
  once until the next, the default theme follows them (`Theme.follow`), and
  `Animate` follows reduced motion. Package `mygo` reads them the same way:
  `Theme.Preferences()` keeps them until the next change, with reduced
  transparency too, which package `ui` does not follow, and without the
  pointing devices, which pages' own media queries answer. Pages get them as
  `ThemePreferences`' JSON (its `MarshalJSON`, the accent a CSS color or
  `null`), which is mygo-runtime's `ThemePreferences`: `tsgen` imports that
  type for it rather than declaring one (`runtimeTypes`). A user script
  after the bridge delivers the `mygo:preferences` event at document start;
  the bridge keeps the latest for `preferences()` and sets the
  `--mygo-accent` and `--mygo-text-scale` CSS variables, the accent left
  unset where the desktop has none. When they change, the core sends the
  event to every page and replaces that script (`Page.SetUserScripts`:
  WebKit's content controllers remove all their scripts, WebView2 by the
  ids it reports), so a document loaded later paints with the new values
  from the start.
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
  by fractions of a DIP. Following its end with a `FollowAnimation`, a
  list notes where the top of its last row was placed; as rows grow or
  come, that row goes up, and the rows show as far below their place,
  which a motion channel takes back to nothing (`glideToEnd`), counted
  as at the end. Rows are placed relative to the list, the offset
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
  GTK's pop-up buttons do, and for Enter, Space and Down. A toolbar's
  overflow menu asks for an overflowed menu button's menu after the pass
  built the button (`askMenuLater`), so the next pass builds it.
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
  their insets as relative positioning; others fade in alone. With
  `TransitionMatch`, the page going away is built too, inert, fading out
  under the page coming, absolutely positioned and fading in, while the
  elements of a `Match` move between them: those of the page going away
  that flew with the page's show no more (`inert`, in `match.go`). The first
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
- **Indexed text storage.** `internal/rope` is a persistent AVL tree of
  bounded, owned UTF-8 chunks, summarized by byte, rune, UTF-16 and newline
  counts, and `rope.Text` the value reading lines, offsets and UTF-16
  ranges from it, and lines a batch at a time; `ui/text_buffer.go` and the
  code editor's buffers keep their texts in it. Edits copy affected chunks
  and tree paths; snapshots share other chunks, and export can stream
  them. `TextInputBuffer`/`TextAreaBuffer` in
  `textbuffer_input.go` bind those roots to the existing editing client.
  Native mutations publish a new root immediately, using a version check
  to preserve concurrent program edits. External root changes refresh the
  widget and clear stale history. The indexed buffer adapter derives line
  starts from the text tree rather than moving every later paragraph's
  absolute offsets. Paragraph layout and height caches remain with the
  widget; newline-count changes update those indexes. Full text is produced
  for explicit export/value queries, never as a binding update per edit.
- **Code editor and diff viewer** (`ui/editor`, `internal/diff`). Package
  `editor` is built on ui's public API, as `ui/chart` is, with four hooks
  for it: `Painter.TextLine` paints a shaped line with a color per
  cluster, `Painter.Visible` gives the clip to cull rows, `KeepFocus`
  keeps the focus in the editor while its completions and find bar are
  pressed, and `Access` describes its text to assistive technology only
  while it reads the window. Its design follows Zed's:
  - *Buffers* (`buffer.go`) keep a `rope.Text` and a log of the changes
    of each version, in bytes and lines, old and new. Editors, the
    highlighter and the diff follow a buffer by the versions they last
    saw (`sync`): selections, folds, diagnostics and caches move through
    the changes since, with a bias at insertions, so that one buffer is
    shown by several editors and both sides of a diff. Undo keeps the
    texts before and after each step, grouped while typing of one kind
    by one editor goes on within 300 ms, and each step the selections of
    the editor that made it.
  - *Display* (`display.go`, `view.go`). Every line is a row, several
    when it wraps, none when folded or collapsed by a diff; blocks are
    rows between lines (a diff's deleted lines, spacers, separators,
    annotations). A line's top is arithmetic on sparse sums: the extra
    rows of wrapped lines in a Fenwick tree, and the heights of blocks
    and hidden lines with their prefix sums; lines are found at a height
    by bisection. Only the lines in view are laid out and shaped, kept
    for two frames as Zed's line layout cache is. Wraps are measured with
    cached advances of characters (monospaced ASCII is arithmetic) near
    the view first, a frame's budget at a time, and estimated elsewhere.
  - *Highlighting* (`syntax.go`). A `Highlighter` follows the buffer's
    edits and gives tokens per line; the default tokenizes lines from the
    state the line above ended in (`lexer.go`, `lexers.go`), keeping the
    states, and after an edit goes on below it only until a line starts
    as it did before. A patch's texts start each hunk in state 0, at
    lines the buffer moves with its edits (`setRestarts`). Optional
    interfaces give a parser's structure to Select Larger, folding,
    brackets and indentation, so that tree-sitter can plug in as a
    `Language.Highlighter` without the editor changing.
    Brackets are matched by scanning lines a batch at a time
    (`rope.Text.EachLine`), skipping strings and comments by their tokens,
    and only lines holding brackets are tokenized for it.
  - *Snippets* (`snippet.go`). `ParseSnippet` reads the flat tab-stop and
    placeholder subset of TextMate/LSP into an immutable template;
    unsupported constructs return errors. Inserting a template, directly
    or as a `Completion.Snippet`, makes each occurrence of the active
    stop an ordinary selection. Existing typing, input methods and undo
    handle them together. A session follows only its next local buffer
    change, through `apply`/`sync`; external edits end it. Fields move in
    template order, preserving ownership at adjacent boundaries, and
    linked deletions stay separate before the editor merges overlapping
    deletions. Tab ends the typing group and selects the next numbered
    stop, 0 last; undo, Escape or leaving the fields ends the session.
  - *Diffs* (`internal/diff`, `diffmodel.go`). Lines are interned and
    compared with git's histogram algorithm, falling back to Myers' with
    xdiff's cost limit, then slid by git's indent heuristic; changed
    lines are paired by similarity and their words diffed, words one
    character apart joined (Pierre's word-alt). An edit re-diffs only the
    region of the hunks it touched, extended to the unchanged lines
    around. The split view is two editors sharing their scroll, their
    lines aligned by spacer blocks, unchanged lines too where the sides
    wrap at unlike widths; the unified view one editor of the new text,
    with blocks painting the deleted lines from a view of the old one.
    The split view's sides share its width by `SplitPercent`, across a
    divider that moves as `ui.Split`'s does. The view lays both editors
    out in its `Draw`, before they paint, to align them, so the build
    gives the sides factors of growth from nothing (`diffSplit`), whose
    widths the `Draw` works out with the layout's own arithmetic, to the
    bit: a width off by a rounding would wrap a line otherwise and build
    the view again, frame after frame. The sides' minimum is kept
    against the last frame's width, and a resize builds the view again
    to keep it at the new one. Unchanged runs collapse into hidden spans under separator
    blocks. The split view's editors scroll as it is drawn, the unified
    view's as it is built; a frame whose layout or build moved the scroll
    builds the view again (`rebuildIfMoved`, `DiffView`), as one laying
    an editor out at another height does (`builtViewH`), so that what
    read `ScrollY` or `ViewportHeight` before, as an app's minimap,
    follows.
  - *Parts* (`parts.go`, `diffparts.go`). The bars and popups around the
    text (find, go to line, completions, hovers) and a diff's separators,
    actions on changes and gutter button are elements the editor places
    with the behavior they need, as ui's bases are: `shield` hands presses
    on to what is inside them rather than to the editor's handler, and
    `KeepFocus` keeps the typing in the editor. The app's render function
    fills each, else the exported default, given a handle to the part's
    state and actions (`FindBar`, `DiffSeparator`, ...), whose values
    the editor reads back after the build, as a query typed. Separators
    are built by the diff view across both sides, over the room their
    blocks leave.
- **Markdown** (`ui/markdown`), on ui's public API as `ui/editor` is, whose
  tokenizers highlight its code. The parser, `ui/markdown/parser`,
  imports nothing of ui, so that `ui/richtext` reads Markdown without the
  view; the view reaches it through its exported methods only
  (`Source`, `Footnote`, `Block.Revision` and `Healed`). It follows cmark-gfm's two
  phases over flat blocks and inlines with byte offsets: a block parser
  with a stack of open blocks, line by line, closing those a line does not
  match before adding a block, so that blocks go in document order; and
  inlines parsed per block when first asked, with cmark's guards against
  quadratic inputs (the bottoms of openers, backtick runs, comments and
  math remembered unclosed). Appending text truncates the blocks from the
  outermost block open after the last complete line (`restart`), which
  CommonMark makes sound: no rule looks ahead, and those rewriting an
  earlier block touch only open ones; the partial last line never decides
  what is closed. Definitions made or unmade parse anew the inlines of the
  blocks that looked their labels up, and the inlines of blocks parsed
  again are reused when their text is the same. An edit parses from the
  top-level block before it until a top-level block starts with nothing
  open (`quiet`) where one did, and splices the old ones after, shifted.
  Streaming completes the tail's inlines from the openers the parser left
  (`heal.go`) and holds back lines that may still change. The view keeps
  the document in its root's local state and the spans of each paragraph
  by block, links as inline `Link` elements, a code block's tokens by line
  and its text in chunks of 32 lines; it reveals text at a pace, fading it
  by span colors and element opacity, which only paint, and `Revealed`
  reads from the root's state whether the frame showed all of the source
  with no fade left, a query rather than a callback, as the frames
  that change it are those the reveal already asks for. A `Selectable`
  root's `CopyData` and `SelectedTexts` map paragraphs' runes back to
  source offsets, to copy text with block separators, Markdown, and the
  HTML of that Markdown parsed again, which `ui/richtext` pastes with its
  formatting, as it reads Markdown with the same parser
  (`ParseMarkdown`). Links open in the system only when `ui.SafeToOpen`
  allows them, in both.
- **Rich text** (`ui/richtext`). Package `richtext` edits attributed text
  on ui's public API, as `ui/editor` does: a focusable box with a text
  input client, `HandleInput`, `Draw`, `Access` and its own scrolling, and
  four hooks of ui for it: `ShapeRichTextWith` aligns and indents a
  paragraph's lines, `Element.SetChanged` reports its edits as `Changed`,
  `Services.ReadClipboardData` and `WriteClipboardData` carry its formats,
  and `InputSelect` takes a selection assistive technology made.
  - *The text* (`text.go`, `seq.go`). A `Text` is a value: its characters
    in an `internal/rope` tree, and beside it a persistent sequence of its
    paragraphs (an AVL tree of chunks of 32, `seq`), each with an
    identity, a style and its runs: attributes over its runes and, last,
    its newline's, which an empty paragraph types in, as Word's paragraph
    mark. Runs are kept maximal. An edit splits and joins both trees,
    copying the chunks and paths it touches; paragraphs joined or split
    take the style of the one their first rune comes from, as AppKit's
    `fixParagraphStyleAttributeInRange:`, but for the app's `ID`, which
    follows the text and never comes from what is inserted. Lists are
    flat, an indent per item, as Quill and Automerge keep them, which
    leaves room for a CRDT.
  - *The document* (`document.go`, `commands.go`). A `Document` holds a
    text, the selection, the typing attributes, and the steps of undo as
    the texts before and after each, which share all they did not change,
    with the region each changed and the selections. A log of changed
    regions by version lets the editor lay out again only the paragraphs a
    change of the app touched. Typing derives the typing attributes as
    NSTextView does (the character before the caret, the one after at a
    paragraph's start, links, code and tags only inside), measured by a
    Swift probe of NSTextView.
  - *The editor* (`view.go`, `edit.go`, `input.go`, `ime.go`, `paint.go`).
    It lays out a paragraph at a time with `ShapeRichTextWith`, keeping
    each layout for the runs, style and composition it was made of, and
    places them by heights in two Fenwick trees from an anchor paragraph,
    as a text area does. Every edit goes through the document, as a step
    of undo or part of the action's; typing joins typing's step.
    Attachments are inline objects of the text system; a first line's
    indent is the layout's (`TextShape.Indent`), on the side the
    paragraph starts at, the right of one reading right to left.
    `Document.VisibleParagraphs` lays out what is in view as it is read,
    as painting does, and the paint that finds the paragraphs elsewhere
    than read asks for a frame (`Services.Invalidate`), as
    `ListState.Visible` builds another.
  - *Formats out of the process.* The clipboard carries the editor's JSON,
    HTML (`golang.org/x/net/html` reads what is pasted) and plain text;
    browsers carry each text format through the clipboard events. The
    runs' formats go to assistive technology as `AccessNode.TextRuns`:
    `AXAttributedStringForRange` on macOS, ATK's run attributes on Linux,
    and UI Automation's Text pattern on Windows (`uia_text.go`), whose
    ranges are COM objects of their own.
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
  final text so a rebuild preserves the selection. A paragraph whose text
  only grew at its end keeps the selection, as a reply streaming in. A
  container's `CopyData` makes what Copy writes from the parts
  `SelectedTexts` returns: each paragraph's ID, text and runes selected.
- **Tables** (`ui/table.go`, `ui/editable.go`). A table's rows are a
  `List`'s that scrolls both ways: the list lays its rows out at least as
  wide as the columns ask (`rowMinW`), and the header, outside the list,
  follows its horizontal offset as it is placed (`followX`). The order
  and the widths the user gives the columns live in the `ListState`
  (`TableLayout`), by column ID. Dragging a header moves its column once
  the pointer went a few DIPs, past a neighbor whose middle it passes; a
  double click on a header's edge fits the column as the table lays
  out, from the intrinsic widths of the cells the frame built
  (`tableFit`). Sorting is the app's: the table sets `ListState.Sort`,
  or `SortBy` for several columns (a new slice each time, which
  `sortClicked` makes), and shows it, to assistive technology as well
  (`AccessSortAscending` and `AccessSortDescending`: AppKit's
  `AXSortDirection`, ATK's `sort` attribute, UIA's ItemStatus). An
  `EditableText` in a row reads the row being built (`Context.row`): the
  list, which keeps the focus, asks for the chosen row's text to be
  edited on Return (macOS) or F2, which it takes only once a row has such
  a text; a click on the text of a row chosen before the click edits it
  once the double-click time has passed.
- **Outlines** (`ui/outline.go`). An outline flattens the open part of
  its tree into rows every frame (`OutlineState.flatten`), which a `List`
  or a `Table` builds as they show, keyed by their items. Rows are tree
  items, with how deep they are (`AccessNode.Level`) and whether they
  have children (`AccessExpandable`), open or not: on macOS, the rows of
  an `AXOutline` with `AXDisclosureLevel` and `AXDisclosing`, which keeps
  its rows as a table does; on Linux, the `level` attribute and the
  expandable state; on Windows, Level, and ExpandCollapse with LeafNode
  for items without children. `OutlineState.Prefix` builds the start of
  the rows in place of the indentation and the arrow (`prefix`), with
  `OutlineRow.Arrow` the arrow's click and Option-click.
- **Trees** (`ui/tree.go`). `TreeBase` builds every item that shows, in
  the order the app builds them, each `Group` a column keyed by its item
  so the items inside keep their IDs as items above open and close; the
  `Tree` widget is `TreeBase` with its look. The tree is a `FocusGroup`,
  one stop of Tab, with the items taking the focus themselves. An item's
  keys open and close it as it is built; those moving the focus, and the
  first letters typed (`flagTypeSelect`, carried over to the item they
  move to), apply once the view is built (`treeBuild.finish`, from
  `context.onBuilt`), against this pass's items in order, which also
  says to assistive technology which of how many each is, as an
  outline's rows.
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
  with it. A toast keeps the time it showed (`spent`) and since when it
  runs (`at`): its time stops while the pointer rests on the toasts or
  the keyboard focus is in them with the window's (`toastsPaused`), and
  runs on while the window is in the background: a toast whose time is
  over by then closes on the next frame, before the toasts build.
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
  leave reach, the last registered (the overlay on top) first. An element
  takes one key a frame (`routeKeys`): from a second key for it, the keys
  wait for the next frame, which the first changed, as a Down and an
  Enter typed quickly into a menu, the Enter choosing the item the Down
  highlighted. Each element
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
  moves along the target into the window (`alongTarget`); one that scrolls
  vertically is first cut to the room the window has (`roomAlong`), as a
  select's long list of options. Popovers have no
  backdrop: the engine notes what each press went down on (`downs`, for a
  pass), and `PressedOutside` walks from the last through parents, and from
  a popover to its anchor, which keeps presses in popovers of elements
  inside the panel, and on its anchor, inside it, as the press goes on to
  what is under it; a press on the anchor after one elsewhere in the same
  pass, as on a web view's page, leaves open the popover it opens, and an
  element the frame first builds was pressed outside of by none, as the
  press opening it is still in `downs` when a quick click opens it. A select's popup keeps a backdrop taking the presses outside,
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
  pixels, clipped as its hits are, with the hole's corners and, under a
  transform that only moves and scales, the size the page lays out at,
  `WebViewPlacement.Width` and `Height`, which `Frame` shows scaled; under
  another transform, the bounds, upright), and painting it makes a hole
  (`OpHole`) through what the window painted, rounded as the border's
  inner edge and cut by the clips around it, so that what paints after
  the element paints over the page. The native web view is under the
  surface, which shows it through its transparent frames; what shows over
  it blends with it as over any pixels. After presenting a frame, the
  engine gives the surface each web view's frame and clip and the boxes
  of the hits after its own within its clip, as they show (`Covers`): the
  pointer is the page's where it shows, inside its corners (`Radii`), and
  nothing painted over it takes the pointer (`WebViewPlacement.At`), as a
  dialog's backdrop does over all of it. Backends cut the page to the
  corners and show it scaled, keeping its layout, so that it shows as its
  hole does over a transparent window too. A
  press there goes to the page and comes to the content as `WebViewPress`,
  which notes it for `PressedOutside` and moves the focus to the web
  view's element, unshown; the engine sets no cursor while the pointer is over a web view
  (`cursorUnknown`), whose page sets it. Web views the frame does not build
  hide and keep their pages. Glyphs over a hole are not on an opaque
  background (`Painter.holes`), so that subpixel ones do not color the
  page. Below full opacity a hole takes only that much of what is under
  it away (`OpHole`'s `Opacity`), so the page blends over it. The element
  is a Tab stop: focusing it gives the page the keyboard
  (`WebView.TabInto`, from its end going back), and Tab past the page's
  ends comes back as `WebViewTabOut`, from which the focus moves on. A
  key the page leaves unhandled comes as `WebViewKey`: the engine moves
  the focus to the web view's element, if it was elsewhere, and takes the
  key as from the keyboard (`webViewKey`), so that the shortcuts around
  the element and the window's see it; Tab never comes. The
  access tree has a node for each page, inside its element
  (`AccessNode.WebView`), which the backends fill with the page's own
  accessibility, as SwiftUI's `WebView` is in place. Backends:
  - macOS: the WKWebView is in a flipped clip view (`MyGoClipView`, its
    layer masking to its bounds) placed at the clip, under the surface in
    the content view, in paint order, in a pane (another) at its frame,
    whose bounds are the page's size: AppKit draws it scaled and finds the
    pointer in it as in any view of scaled bounds, and the pane's layer is
    cut to the corners (`cornerRadius`, continuous, and `maskedCorners`;
    a `CAShapeLayer` mask for corners of different radii), in the layer's
    units and orientation, which `convertPointToLayer:` tells, whatever
    AppKit does to scale it. The Metal layer is not opaque, and
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
    `accessibilityParent` is that node; the clip view lists none. With
    `ScrollToContainer`, `MyGoWebView`'s `scrollWheel:` gives the event to
    the surface's conversion (`scrollWheel`) instead of WebKit. WebKit
    sends a key the page did not handle again, and the WKWebView on up
    the responder chain: `MyGoClipView`'s `keyDown:` sends it to the
    surface as `WebViewKey` (`unhandledKey`), Tab aside. An Escape (or
    Cmd+.) comes as `cancelOperation:` first: WebKit sends a command its
    web process does not know up the chain and resends the key only if
    no responder took the command, and the `NSPanel` of a flyout takes
    it, so `MyGoClipView` takes `cancelOperation:` itself and sends
    `WebViewKey` with Escape (`unhandledCommand`).
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
    surface's alone. With `ScrollToContainer`, a handler of the view's
    `scroll-event`, which runs before WebKit's class handler, converts the
    event as the area's does (`scrollEvent`, moved by the view's
    allocation), sends it to the content and stops the signal. A scaled
    page's view is as large as it shows, and WebKit zooms it by the scale
    over the page's own zoom (`pageScale`), so that it lays out at the same
    width in CSS pixels. GTK runs each handler of `draw` with a clip of its
    own, and GDK draws WebKit's frames with OpenGL within a clip's
    rectangles alone: the view's handler of `draw` clips to the page's
    corners on whole pixels and calls WebKit's class handler itself
    (`GtkWidgetClass.draw`), and the region shaping the area keeps the
    pixels the corners cut off. WebKitGTK dispatches a key the page did not
    handle again once its web process says so, and its class handler of
    `key-press-event` then passes it on: a handler connected after it
    (`cbWebViewKey`) gets those keys alone and sends them as `WebViewKey`,
    but Tab, modifier keys on their own and the shortcuts of menu items with
    a role, which go on to the menu (`cbMenuKey`).
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
    surface's pixels, cut to its corners by a rectangle clip
    (`IDCompositionRectangleClip`, its setters taking floats through
    purego) and, scaled, at the page's size, which the visual's transform
    scales to its frame; the mouse goes to the page through the inverse.
    The surface gets the mouse everywhere and sends it on
    (`SendMouseInput`) where a web view takes the pointer, or to the one a
    press went to until the buttons are let go (with the capture), with a
    leave as the pointer goes; it sets the cursor the page asks for
    (`CursorChanged`), and a press moves the focus into the page.
    `TabInto` is `MoveFocus` (next or previous), and `MoveFocusRequested`
    becomes `WebViewTabOut`. The surface's drop target sends drags over a
    page to its composition controller (`ICoreWebView2CompositionController3`'s
    `DragEnter` and the others). With `ScrollToContainer`, `forwardMouse`
    keeps the wheel over that web view (`WM_MOUSEWHEEL`, `WM_MOUSEHWHEEL`)
    for the surface, which scrolls the content as over its own elements;
    a touchpad's pinch, the wheel by less than a notch with Ctrl held,
    still goes to the page, as pinches do on the other platforms.
    WebView2 tells of no key a page leaves unhandled: `keys.js`, built
    from `packages/bridge/src/keys-world.ts`, runs in an isolated world of
    each document of the web view, frames in its process included
    (`Page.addScriptToEvaluateOnNewDocument` with a `worldName`), decides
    after the page's handlers ran with the rules of `keys.ts`, and calls
    a binding of that world alone (`Runtime.addBinding` with an
    `executionContextName`, after `Runtime.enable`), which WebView2
    raises as `Runtime.bindingCalled` (`GetDevToolsProtocolEventReceiver`);
    `bindingCalled` sends `WebViewKey`. The page's scripts reach neither
    the world nor the binding, bridge or not, and their events are not
    trusted. For UI
    Automation the root also answers `IRawElementProviderHwndOverride`:
    for the window WebView2 shows a page with, it returns the page's node,
    which answers with WebView2's provider of the page
    (`get_AutomationProvider`), as WinUI 3 does. That provider has no
    children: the page's elements are under a top-level window of
    WebView2's (`Chrome_WidgetWin_1`), which
    `GetAutomationProviderForWindow` does not map to it, so UI Automation
    finds them there (seen with a runtime of October 2026).
- **Context menus** (`ui/menu.go`) open in two frames. A right-click or the
  menu key marks the element, from the states of the last frame, and the
  next frame runs its `ContextMenu` function to collect a `platform.Menu`;
  after the frame, package mygo shows it with `Menu.popup`, posted to the
  main loop, since native menus wait in a loop of their own. A choice
  comes back as the item's place and label, and its ID when the view gave
  it one (`MenuItem.ID`, `menuRun.keys`), which the next frame's run of
  the function matches to report `Chosen`, by the ID alone when there is
  one; text inputs' menus queue their editing commands instead. `PopoverMenu` opens the same way and draws
  the menu in the window instead (`ui/popovermenu.go`): the frame that
  would show it notes it (`menuState.drawn`, whose `menuShown` holds where
  it opened, from the element's corner, and the submenus open with their
  highlights), and while it shows, each frame runs the function to
  collect the `platform.Menu` and builds the menu from it with the parts
  of `MenuBase` (`menuParts`), which hold the keyboard and focus model of
  every drawn menu: themed, and headless for `MenuBase` and
  `ContextMenuBase`, whose `menuShown` is a local of their trigger and
  whose open state is the app's `*bool`. `Popup` builds an overlay of a
  point attached to the element where the menu opened, so that the menu
  flips about that point at the window's edges, and a panel (`RoleMenu`)
  for the menu and each submenu open, each attached to the item opening
  it (the submenus that `Submenu` registers, built after their menu's
  panel, as its siblings). A panel keeps the focus for its items, which
  the keys and the pointer highlight (`ActiveDescendant`), by their
  element IDs; the keys are read once the panel's items are built, which
  they move among, and Enter clicks the item highlighted for the next
  pass (`clickLater`), which reports it as `Clicked`; an item clicked
  closes the menu once its panel is built (`finish`), unless the view
  marked it `KeepOpen` after making it. A `Group`'s items are a range of
  its panel's, which the keys take as one stop (`menuStops`: the item
  checked, else the first), Left and Right moving within it. The menu
  closes for presses outside it but in the frame it opened, whose press
  opened it.
  An item chosen runs the function again at once, as the choice from a
  menu of the system's does in the next frame, or does its editing
  command; the overlay going gives the focus back to the element that
  had it. A menu whose element a pass does not build closes
  (`resolveMenu`). `PopoverMenuButton` draws a menu button's menu so
  (`menuShown.below`): opened below the button itself (`menuState.below`,
  which `openMenuButton` sets), its panel attaches to the button's
  bottom, flipping above it, and the button shows expanded; opened from a
  toolbar's overflow menu, it goes at the overflow button's corner as a
  context menu goes at its point. Opened by the keys or assistive
  technology (`menuState.key`), it starts on its first item, and a press
  on the button while it shows closes it (`buildMenu`, and `newMenu` for
  the bases).
  A window whose view sets `PopoverMenus` (`ui/menupresentation.go`,
  `engine.menuPresentation`) draws every menu so: `ContextMenu` and `Menu`
  take `PopoverMenu`'s and `PopoverMenuButton`'s path, and so do the
  widgets built on them. The drawn path is reached only through function
  values `SetMenuPresentation` sets (`menusInWindow`), so apps that never
  draw their menus do not link it.
  A text input's or selectable text's own editing menu has no function of
  the view's to build it. `resolveMenu` notes it as a drawn menu
  (`drawnMenu.edit`) and asks for another pass. After that pass's view,
  before the overlay layer joins the root, the frame finds the text's
  element in the content or the overlays (`node.find`) and draws the menu
  of its `EditItems` there, in the element's theme. One in a part that
  takes no input (`flagInert`), as a page going away, is not drawn, which
  closes it. The inspector's fields keep the system's: its panel is built
  after the overlays, over them.
  The `Tester` reads a drawn menu's items as last built (`drawnMenu.items`)
  and chooses one by its ID (`drawnMenu.choose`), which the next build
  takes as a click, so its menu helpers work on either.
  What a drawn item shows beyond its `platform.MenuItem` (`Danger`,
  `Trailing`, `Content`) stays on the `MenuItem` of the run collecting
  it (`menuRun.items`), which `drawnItem` reads, so that the system's
  menus and their backends never see it. An item with an ID keys its row
  (`menuItemKey`), whose highlight then follows it as items around it
  come and go. Once a pass built the menu, the items its levels highlight
  (`drawnMenu.highlighted`, by ID or else place and label) go to the
  next run of the function, whose `MenuItem.Highlighted` reports them;
  when they changed, the pass asks for another, so the function sees the
  highlight it built. `Menu.MaxWidth` (`menuRun.widths`, by
  `platform.Menu`) caps a panel's width, its rows' labels shrinking to an
  ellipsis.
- **Scenes** are flat lists of operations in device pixels: rounded
  rectangles with borders of a width per side, solid or dashed, filled
  with a color, a linear gradient mixed in sRGB or Oklab, a gradient of
  stops, linear, conic or radial (`Scene.Stops`, which ops point into, at
  most 32 each), or stripes, the border painted with the op's gradient where it
  says so (`BorderPaint`);
  shadows (blurred rounded rectangles, cut by the box casting them, as
  CSS's box-shadow is, or inner ones, inside the box where the hole they
  leave, blurred, does not cover it, as box-shadow: inset is); runs of
  glyphs, whose masks may take a gradient (paths drawn with one); images,
  in color or gray, blurred or not (an image's `Blur`: averaged over
  squares and blurred along rows and columns as a backdrop is, but
  transparent beyond its source rectangle, which `ui`'s `Element.Blur`
  draws its layer with); effects (`OpEffect`); holes (`OpHole`), rounded
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
- **Shaders** (`ui/shader`) are effects an app writes, in Go's syntax:
  `internal/shader` (Ebitengine's Kage compiler, forked, with uint and
  loops that need no constant bounds) compiles one to the IR of
  `internal/shaderir`, whose backends write it as a function for the
  renderer's code to call (`Fragment` of `msl`, `hlsl` and `glsl`; only
  the platform's renderer's backend is linked), and whose interpreter
  (`interp`) is the CPU twin. A shader's effect (`Effect.Shader`) tells
  where the words of its uniforms go in the buffer its code reads: Metal's
  struct (buffer 2), HLSL's cbuffer (b2) or a std140 uniform block
  (binding point 0); the op carries the words and up to four images,
  bitmaps or canvases (`EffectOp.Uniforms`, `Images`), bound to textures 5
  to 8 (t6 to t9 in Direct3D). GLSL and HLSL read them as globals, so a
  shader goes between the effects' head and tail; Metal has no globals,
  so its shaders are whole libraries with a tail of their own
  (`ShaderSource`, `shader_ps`), which `ui/shader` builds, so that apps
  drawing none do not carry it. `ui` itself only has `Painter.ShaderEffect`
  and `Painter.ShaderFill`: the compiler, its package variables and
  go/parser stay out of apps that draw no shader. A shader filling text,
  paths and icons (`ShaderFill`) turns the glyph ops its function painted,
  with grayscale masks, into effect ops whose `EffectOp.Glyphs` point at
  their mask glyphs (glyphs of color stay an op of their own): the
  builder makes one instance per glyph, as `OpGlyphs` does, with 1 in
  `Params[1]` and the corner of the box the shader fills in `Grad`, and
  the shaders' tails cover it with the mask atlas and `textCoverage`
  (above `#ifndef EFFECT`, as effects share it), the shader's color in
  place of the paint; `raster.maskedEffect` is the CPU twin. An
  element's effect (`Element.Effect`, a `ui.Effect`) renders the
  element's subtree into a canvas kept in its state, at the window's
  scale, in the element's own space (`paintEffect`: positions shift, and
  transforms in the subtree lose the element's own, `unworld`), then
  paints the effect with the element's transform. The canvas is made in
  steps of 64 pixels (`effectLayerStep`), the content in its top left
  corner, and kept while the content fits and leaves less than two steps
  over, resized in place otherwise (`Image.Resize`): an element whose
  edges cross pixels as it moves, or whose text grows, keeps one canvas
  and one texture, rather than one a size. The effect gets the content's
  part (`Canvas.region`), so shaders read beyond it as transparent, as
  beyond a canvas of its size; what the subtree paints is clipped to that
  part, and backdrops end at its edge (`Scene.Drawn`, `Scene.BackdropOf`,
  which the painter, the builders and the CPU renderer use alike), as at a
  canvas's. The canvas is transient (`Image.Transient`), unless the
  element is kept (`Element.Cache`, `ui/cache.go`): then it paints the
  subtree into the other of two scenes and compares it with the last
  through a `scene.Differ` of its own (`layerCache`), and only a scene
  that differs, or is of another scale or part, bumps the canvas's
  version and goes in the frame's `Canvases`; one that does not stays
  the canvas's scene all the same, as the atlases hold its glyphs this
  frame. The window's `Differ` then finds the image op the same, so
  damage over the element redraws one image. A kept element alone (no
  effect, no blur) renders, rather than its box, what it paints around
  it (`keptLayer.area`: each one's box and shadows grown by 4, in its
  space, clipped by those that clip). Its code is reached only through
  the interface `Cache` stores (`layerKeeper`), so apps that never call
  it, nor fade a group (below), do not link it. `Opacity` below 1 (a transition's fade and a pose's
  `Fade` too) fades an element with children as one group, as CSS's
  group opacity (`ui/group.go`): the subtree renders into the same
  transient canvas, the area `keptLayer.area` gives within the clip,
  drawn as one image at the opacity (`paintGroup`), so every renderer
  draws it as it draws an effect's, and kept as `Cache` keeps its layer
  while what it holds is the same (`keptLayer.draw`). A layer costs a
  render pass, so a group paints flat first, its opacity multiplied into
  each op as before, and `checkFlat` looks at the ops it painted
  (`overlapping`: their bounds within the clips pairwise, at most 32 of
  them, so that what culling skips counts for nothing): when two
  overlap, it renders the layer, takes the flat ops back (`takeBack`:
  the scene's slices, its canvases and the parts drawn,
  `engine.drawn`) and paints the layer in their place. The state's
  `grouped` then has the next frames go to the layer at once, back to
  flat when it no longer overlaps, so the frames before change what a
  group costs, never what shows. Holes (web views) and effects reading
  their backdrop keep a group flat, as no layer shows what is behind
  it. The code is linked by `Opacity`, `Transition` and `AnimatePose`
  (`linkGroups`). A blurred or kept element fades its canvas at its
  opacity rather than each op in it, and paints nothing faded out.
  `ui/shader` gives an effect's entry point (`ColorEffect`,
  `DistortionEffect`, `LayerEffect`) a `Fragment` that calls it, and the bridge reads the content as image 3
  (`LayerImage`) from where the box's corner lies in it
  (`__layerInset`). A shader reading its backdrop (`backdropAt`,
  `backdropSample`) is an effect with `Backdrop`: `Painter.EffectBackdrop`
  computes the backdrop as the builder will (`scene.BackdropOf` of the
  op's bounds), the bridge maps positions to its texels
  (`__backdropRow0`, `__backdropRow1`, `__backdropArea`), and the
  renderers bind their backdrop texture as its image 2
  (`gpu.BackdropImage`); the CPU twin reads `scene.BackdropImage`. Direct3D's compiler takes from 0.1 s to more than 1 s
  for a shader's effect (the renderer's head and tail, with its coverage,
  are most of a small one's; optimization levels barely change it), so
  `ui/shader` starts it as it compiles one (`d3d11.PrecompileEffect`), on
  goroutines, as many at once as the CPUs but one. The renderers share
  the code (`compileEffectCode`), waiting for what is left, and it stays
  in `%LOCALAPPDATA%\MyGo\ShaderCache`, a file named by the SHA-256 of
  what it compiled from, one directory for the user as Windows' own
  D3DSCache is, the least recently used removed beyond 32 MB. Effects
  compiled ahead of time, and apps without shaders, do not link it. On
  macOS, `metal.PrecompileShader` does the same with the system's default
  device: Metal's library (about 85 ms the first time) and the pipeline
  for BGRA8 windows (about 60 ms), which renderers on that device retain
  (`sharedShader`) in place of making them; Metal's own disk cache keeps
  them for later launches, and wide-gamut windows make their pipeline
  from the shared library.
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
  every other process. A text wrapped again at another width, as in a window
  being resized, keeps its `CTTypesetter` while frames use it, and gives it up
  8 frames after the last, or as windows go idle. Core Text opens a break
  iterator for each line of text with a language attribute, so
  `languageShapes` leaves the attribute out where it changes nothing. A
  number, new in each frame on a dashboard, is put together from the glyphs
  and kerning of its characters, measured once for each font and checked
  against Core Text from time to time (`numbers_darwin.go`). Latin text
  kept across widths is shaped into one line that its lines take their glyphs
  from, breaking at the spaces before plain words and asking Core Text for
  every other break, checked in the same way (`wrap_darwin.go`). Korean text
  breaks between words, as CSS's `word-break: keep-all` (`keepWordsWhole`
  takes the breaks between two letters or digits out of Unicode's): Core Text
  moves the break it suggests back to the last one left in the line; Pango
  is told not to break within the words (`pango_attr_allow_breaks_new`), and
  lays out again from a line a word longer than it overflows, without that
  word kept; DirectWrite's lines, when one ends within a word, are broken
  anew from its cluster metrics and shaped one by one; the canvas engine
  leaves those breaks out of its own. Every engine gives a line's glyphs in visual order,
  from the left, through right-to-left runs too (DirectWrite's runs, whose
  glyphs come in logical order, are reversed), so that what walks them,
  and `ui.Shape`'s callers, rely on one order. The package assembles their
  lines into layouts, with its own whitespace trimming, ellipsis,
  alignment, carets and selection, so those behave the same everywhere,
  and caches layouts by their parameters.
  A text that wraps none of its lines lays out alike at any width they
  fit in, but for where its alignment puts its lines: `LayoutFrom` lays
  it out at another width as the layout it is given (`ui` gives an
  element's last), or as the text's layout without a width, once made,
  sharing its lines, or copying those its alignment moves (`realign`), so
  that a text whose box grows or shrinks, as a transition or the window
  resizes it, is shaped once, and one measured, then laid out at the
  width it gets, too. The engines rasterize glyphs too (`IDWriteGlyphRunAnalysis`,
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
    skipping the ink of descenders (but a link's, on hover:
    `SolidUnderline`), a strikethrough through the middle of
    the x-height, from the text's start to its width rounded up to a
    point; Direct2D's, an underline at the font's offset rounded to whole
    pixels from the snapped baseline, a strikethrough from the baseline as
    laid out, with edges where its eight samples across and down a pixel
    put them; GTK's, at Pango's metrics in whole pixels, the underline
    along a run's ink and advances, the strikethrough along its ink.
    `LineStyle` overrides the font: a thickness (`DecorationThickness`)
    around the line's middle, and an underline's offset
    (`UnderlineOffset`, CSS's `text-underline-offset`), its top that many
    DIPs from the baseline the engine draws on, in whole pixels, for every
    engine, the browsers' canvas included; Core Text skips the ink where
    the offset puts the underline.
  - *Joined glyphs.* Renderers blend each glyph in turn, as Core Graphics
    and Direct2D do, but cairo adds the coverage of a run's glyphs before
    blending it, which differs where antialiased edges of neighbors share
    a pixel. There, on Linux, package ui has the text system join them
    (`System.GlyphRun`): their coverage added and clamped, cached as one
    bitmap.
  - *Strokes.* `TextStroke` draws each glyph's stroke as a mask of its
    own (`System.StrokeGlyph`), made in Go from the glyph's coverage for
    every engine: the coverage dilated by half the width less the
    coverage eroded by it, each edge pixel's coverage placing the edge
    within it, so the stroke is centered on the outline, round at its
    corners and as smooth as the glyph; color glyphs stroke their alpha.
    Cached with the glyphs, under a key with the width, and drawn by
    every renderer as any mask glyph, all of a text's before its glyphs
    (`paint-order: stroke fill`). `TextStroke` links it, so that apps that
    stroke no text do not.
  - *Shadows.* `TextShadow` and `Span.Shadow` draw each glyph's shadow
    as a mask of its own (`System.ShadowGlyph`), the glyph's coverage
    (the alpha of a color glyph) blurred in Go by a separable Gaussian
    of sigma half the blur, as CSS has it, padded by three sigmas and
    capped at 64 pixels. Cached with the glyphs, under a key with the
    sigma, and drawn as a mask glyph at the offset, all of a text's
    before its strokes and glyphs. `NewTextShadow`, which
    `Element.TextShadow` and a `Span`'s `Shadow` take, links it: a
    `TextShadow` has no fields to set otherwise, so a shadow that paints
    is one that linked it.
  - *Tabs.* `Style.TabSize` sets tab stops every that many spaces of the
    style's font, letter spacing included, from the start of each line,
    which each engine takes natively: Core Text's paragraph style (no
    tab stops but a `DefaultTabInterval`), DirectWrite's
    `SetIncrementalTabStop`, a Pango tab array of one stop, which it
    repeats, and the canvas engine's own lines, which otherwise measure a
    tab as the canvas does, as a space. The first `TabSize` links it
    (`System.UseTabSize`), as `Justify` does `UseJustify`.
  Paths drawn with
  `Painter` are masks in the coverage atlas, rasterized by `internal/vec`,
  cached by their shape relative to the pixel grid in quarters of a pixel,
  to which their points move first: a path looks the same whichever path
  of its key drew its mask first. A path too large for one mask, over
  4095 pixels a side, draws as tiles of it (`pathTiles`), each a mask of
  the whole path clipped to the tile: only the tiles that show and that
  its edges reach, and, inside a fill, tiles covered whole, which share
  one mask: those its edges wind around by its rule (`Path.FillRule`),
  which also picks the mask's (`vec.Rasterizer.MaskEvenOdd` for
  `EvenOdd`) and keys it, as it keys a path's clip. A path larger than
  a tile (511 pixels a side) draws as tiles too when the atlas has
  lately had no room for a path's mask (`pathCrowdedPaints`): such a mask takes a shelf of the atlas of its
  own, mostly empty for a long thin stroke, and a few of them, as wires
  redrawn every frame, leave no room for the frame's other paths, which
  would not show. A path whose mask finds no room draws as tiles at once,
  and the runtime's repaint (`System.MakeRoom`) draws the frame's large
  paths as tiles, which pack eight to a row of the largest atlas: the
  room made lets go of the masks of those the paint drew before
  (`System.Unkeep`), which it would keep as the frame drew them, and
  which would leave the tiles no room. On the
  pixel grid tiles show the pixels of one mask; under a transform off
  it, a faint line where they meet. `Path.Contains` counts the winding
  of the path flattened at four times its scale; an element's `HitTest`,
  kept in its state, is one more test of `engine.contains`, after its
  box and its own clip. A stroke's mask is the outline of
  its center line, each edge drawn once (`strokeInto`): one side,
  turning at corners and ends as the path's joins and caps say
  (`Path.Join`, `Path.Cap`), and back along the other. Its dashes (`Path.Dash`) are cut from the center line as the mask
  is drawn, from where each polyline starts in their pattern, so that a
  trimmed path's dashes stay where they are, and its key holds that place,
  in quarters of a pixel within one pattern, with the caps, joins and
  dashes. As in Skia, a dash through the start of a closed subpath is one
  dash, and the point between a dash and a gap is in the gap. Square caps
  reach past half the stroke's width, and miter joins as far as the points
  of its corners do within the miter limit (`flatPath.reach`, from the
  points the mask draws), and the mask and its tiles reach as far.
  Opaque `Painter.StrokePolyline` on a GPU is the exception: each segment
  is a rotated fill with round ends, drawn as other fills are, without a
  mask; translucent ones, and the CPU renderer, take the mask's way, so
  that overlapping segments do not show darker. A translucent line of one
  segment overlaps nothing, and is a rotated fill too: the stripes of a
  diff's fillers are hundreds of these, whose one mask would be as wide
  and as high as the filler in view is high, rasterized again as that
  changes, and drawn that large for each stripe.
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
  `Params` stays a key of the layout cache. A span's letter spacing of 0
  is the text's unless `SpacingSet` (ui's `Span.NoLetterSpacing`, or an
  inline element's `LetterSpacing(0)`) makes it its own, which each engine
  sets over the span's range as any other. Laying out reuses what shaping
  throws away: engines take the glyphs, runs and lines `shape` returns from
  a scratch each layout resets (`shapeScratch`), as the layout copies them
  into its lines, which share one allocation; Core Text keeps its buffers,
  the attributes of each font, and calls the functions of each layout
  through `SyscallN` rather than reflection. Colors, underlines and
  strikethroughs of spans only paint, by the rune each glyph starts.
  A span with an `ObjectWidth` makes its object replacement characters
  inline objects: boxes standing on the baseline, which lines break
  around and which draw no glyph (`Glyph.Object`), as a rich text's
  images, or a text's math with `Element.DrawObjects`, which paints them
  after the glyphs. The line takes their height, and their descent
  (`ObjectDescent`, encoded only for objects), in the shared layout;
  each engine only gives them their width: Core Text through run delegates, whose
  callbacks return floats Go cannot, so they are leaves in assembly
  (`rundelegate_darwin_*.s`); DirectWrite through `IDWriteInlineObject`s,
  their places found by hit testing, as its glyph runs' are; Pango through
  shape attributes; the canvas engine as clusters of their width.
  A span's `BaselineShift` moves its glyphs and objects off the baseline
  in the shared layout too, as the room the line takes for them and the
  baseline its decorations are drawn from (`decoRange.shift`).
  The engines break a paragraph at one width, which is all DirectWrite's
  layouts take, so what else paragraphs need is the shared layout's
  (`paragraph.go`), alike on every engine. A first line's indent
  (`Params.Indent`) breaks that line narrower, then the rest anew from
  where it ends. Justification (`Params.Justify`) widens the spaces and
  the room after Han characters and kana of the lines the width wrapped,
  but the last of each paragraph, moving each glyph by the room the
  clusters left of it gain, so that carets and selection follow; it is
  linked in once a `TextShape` asks for it (`System.UseJustify`), as
  apps that do not justify text need none of it. The engines all break
  lines at soft hyphens (U+00AD) and draw them as they please: the layout
  hides those within lines (a glyph of no font and no advance, which
  keeps the caret's place) and draws a hyphen, the engine's glyph for
  U+002D in the soft hyphen's style, at the end of a line broken at one.
  Only Pango leaves room for it as it breaks, so a line whose hyphen does
  not fit is broken again, narrower by the hyphen, where Unicode's line
  breaking allows (else the hyphen overflows, as a long word does), and
  the rest anew (`breakLines`). The padding of a span (`Span.PaddingX`),
  which engines know nothing of, is the shared layout's too: `padLine`
  moves the glyphs of each line's piece of the span in, and those after
  it on, by the padding at both ends of each piece, as CSS's
  `box-decoration-break: clone`, and widens the line; a line that does
  not fit with the padding of its pieces is broken again, narrower by it,
  and the rest anew, and an ellipsis leaves room for it. A paragraph
  without an indent, a soft hyphen or a padded span is shaped once.
  `Params.Balance` (`Element.Balance`) breaks a paragraph that wraps at
  narrower widths, bisecting to a quarter of a DIP, for the narrowest
  that keeps its count of lines (`System.balance`); `Params.Pretty`
  (`Element.Pretty`) breaks again only its last two lines, when the last
  is a word alone, bisecting for the widest width at which the line
  before gives it a word and they stay two, at a space
  (`System.prettyLines`), which `Element.Pretty` and a `TextShape` that
  asks for it link (`System.UsePretty`). The layout cache keys both with
  the other params, so a build lays it out once.
  Text elements built inside a text (`ui/inline.go`) are inline: the
  paragraph takes their text when its `Children` returns, and their
  styles as spans over its own when it lays out. At commit, each gets the
  boxes of its runes in the paragraph's layout (`Selection`), one hit
  each, so the pointer, the focus order, tooltips and menus work as for
  any element; they paint only their focus ring (their `Opacity`,
  `Invisible` and dimming fade the colors of their spans, `Span.faded`,
  and a color faded to nothing stays the span's own, `colorSet`, so the
  paragraph skips their glyphs but keeps their advance), and assistive
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
  drew, grows the atlas when that is much of it (what the frame left
  out counting once, however many times it draws it) and forgets the rest, and
  paints the frame again: no frame shows with glyphs missing. The mask
  atlas starts at 256×256, and the color atlas at one transparent texel
  until a color or subpixel glyph draws. Growth accounts for the width
  and height of missing bitmaps as well as their area, so long thin
  paths fit on the repaint too. The first glyph can grow an empty atlas
  immediately, without invalidating anything already painted.
- **Renderers.** `internal/gpu` turns a scene into instanced quads, in
  batches that share a scissor rectangle and an image, for a
  shader that computes the signed distance to rounded rectangles, Evan
  Wallace's blurred rounded box, gradients, stripes, the dashes of borders,
  atlas coverage and the innermost rounded clip; outer clips are scissor
  rectangles, and a clip reaching into an outer one's rounded corners cuts
  with that one's mask as well. The builder omits operations wholly hidden by the certainly
  opaque core of a later fill, keeping at most eight occluders and treating
  backdrop-reading effects as barriers. Its planner shares clipping rules
  with drawing, grows shadow support before transforming it, and skips
  scenes without potential opaque fills. Large shadows with a caster and
  border-only fills draw as scissored strips that leave out where their
  output is zero: inside the caster, between a border's edges, and outside
  a border's outer edge. With round corners they draw in bands of rows
  (`gpu/bands.go`), each drawing only the columns between where the outer
  edge may reach and where the inner one certainly covers, so a circle a
  pixel wide shades about its length rather than its box; the bands across
  the corners are as many as pay for their draw calls (`bandCost`). With
  continuous corners they draw around one rectangle. These scissors never
  change a clip mask's sampling coordinates. Both transformations are shared by all GPU backends.
  Large fills, shadows and images with continuous corners (negative radii,
  which only Metal gets) draw as nine-patches (`gpu/corners.go`): their
  corners' boxes as they are, and the cross between them as a twin with
  square corners, in batches whose pipeline lacks the curves' code. Away
  from the boxes the shader computes the same value either way: a fill's
  pixel area, and a blurred shadow whose samples miss the curved rows or
  whose blurred sides round to 0 and 1 (9 sigma from the curve). Each
  piece is a draw call of its own (its scissor rectangle), so a shape is
  cut only where the twin's pixels, within the shape's quad, save more than
  the pieces' calls cost (`pieceCost`): small shapes, near their corners
  everywhere, stay whole and join their neighbors' batches.
  Metal opts into three further builder capabilities: marking fills whose
  clip coverage is one, splitting opaque cores from blended edges, and
  batching by fragment class, the last of which Direct3D 11 takes too. OpenGL
  keeps the original instance flags and class-independent batching. Metal binds the instance buffer
  once a pass and draws each batch with its first instance as the base
  instance, which `[[instance_id]]` includes; a batch binds its pipeline,
  image, clip mask and scissor rectangle only when they changed, with
  selectors looked up once, and the scissor goes by pointer
  (`setScissorRects:count:`) rather than through a reflective purego
  function: messages from Go cost about half a microsecond each. An effect draws in a batch of its own, with a pipeline of
  its own, which the renderer makes the first time the effect draws; one
  reading its backdrop draws after a backdrop step (`Builder.Backdrops`):
  the renderer ends its pass, reads the backdrop's area of what it drew
  (Metal from the drawable, which is not `framebufferOnly`; Direct3D 11
  and OpenGL from a copy of the area, `CopySubresourceRegion` and
  `glCopyTexSubImage2D`, whose rows go up), averages and blurs it with two
  small pipelines (`down` and `blur`, scissored to the texels they
  compute, reading them with `Load`, `read` or `texelFetch`) into textures
  as large as the frame's largest backdrop, and goes on with the pass, the
  backdrop bound for the effect. An image drawn blurred goes through the
  same two pipelines before the frame's pass, which the builder asks for
  as it meets it (`Builder.BlurredImage`), from the image's texture into
  one of the frame's own, in BGRA8 whatever the frame's format, reading
  what lies beyond the image as transparent (`gpu.PassClear`); OpenGL
  reads canvases flipped and uploaded images not (`gpu.PassUnflipped`).
  The stops of gradients travel in a texture of 32-bit floats the
  builder fills each frame (`Builder.Stops`, two texels a stop: its
  color, then its place and its color in Oklab), bound to texture 9 (t10
  in Direct3D), from which the shaders' `stopsAt` mixes them as
  `internal/raster` does. Each renderer draws scenes offscreen for
  the tests of packages defining effects (`RenderOffscreen`,
  `NewOffscreen`).
- **Canvases** (`scene.NewCanvas`, `ui.Canvas`) are images drawn from a
  scene of their own. `Canvas.Draw` paints into its scene with a painter
  of scale 1 sharing the window's atlases, and lists it in the frame's
  `Scene.Canvases`, inner canvases first as they finish. Before a frame,
  each renderer draws the canvases it lists or shows whose version its
  texture does not hold (`Scene.EachCanvas`, those a canvas shows before
  it), each with a builder of its own, into a texture in the frame's
  format before effects: Metal into a BGRA8 render target in the frame's
  command buffer, Direct3D 11 into a BGRA render target view (the dynamic
  instance buffer, mapped with discard, serves all), OpenGL into a
  framebuffer object, whose rows go up, so that image instances showing
  it flip their texture coordinates (the samplers take source rectangles
  either way round). A canvas's texture lives as long as the canvas
  (`weak.Pointer`), not drawn again when shown; a transient canvas's,
  drawn anew in every frame that shows it, as an effect's, only while
  frames draw it: the renderers' sweep every 120 frames frees it, rather
  than wait for the canvas to be collected, which the little of Go's heap
  it takes may put off for minutes while its texture takes megabytes. The CPU renderer draws
  canvases into their `Pix` (`PixVersion`).
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
    on transformed coverage sampling. Batches of a class (`gpu.Class`) draw
    with shaders of their own (`classes.hlsl`, after `shader.hlsl`, whose
    functions they share, which leaves the sources effects are compiled from
    as they are): instances of a class are not transformed, so they leave out
    the general shader's coordinates, pass on 6 to 10 values rather than 15,
    skip the clip where the vertex shader finds it holds the whole quad, and
    write one color, blended by its alpha, but for subpixel glyphs, which need
    dual-source blending. A plain fill takes 83 instruction slots rather than
    the general shader's 3,627. A set of classes draws with the first of
    `classShaders` holding them, made as first drawn: each class, the sets that
    small batches join most, and all of them, which still costs less than the
    general shader, which only transformed instances and effects need. It draws
    into a flip-model swap
    chain on the surface's window, or, over a material, one for
    DirectComposition shown on it, with WARP when no hardware device
    works. At most one frame waits ahead of the screen, not DXGI's three,
    and `Present` does not block for
    it: the swap chain's frame latency object (`latency.go`) tells room
    for a frame, and a frame drawn before there is presents once there is,
    as a goroutine waiting on the object has the main thread present it
    (`SetPost`); frames paced by the display (`WM_PAINT`, but a resize's
    and `FrameNow`'s) are not built meanwhile (`Renderer.Ready`). Every
    present takes one of the object's signals, so its count stays the
    frames the screen can take. Frames that make the refresh without building
    ahead, as the last ones' cost tells (`frameCost`: the main thread's time and
    the GPU's timestamps, taken after the frame's clear, which waits for the
    window manager to give the buffer back; a frame no `Ready` started, as a
    resize's, counts none, its main thread's time not known), build once it composed the last
    (`DwmFlush`) and there is room, and present with sync interval 0, to show at
    its next composition. Heavier frames build ahead and present with sync
    interval 1, which keeps them at the display's pace a refresh later. A frame of a new size is
    presented at once (sync interval 0), not after the frame queued
    before it, and waits until DWM shows it (`DwmFlush`), so a live
    resize shows no stretched frames: the window takes its next size
    with the frame of that size. DWM may still show the window at its new
    size a refresh before that frame (#222), so the window's swap chain
    shows its frames unscaled (`DXGI_SCALING_NONE`) with the content's
    background beyond them (`SetBackgroundColor`, from
    `scene.Scene.Background`), and the window paints what it grew by
    around its surface with that background before fitting the surface
    to it (`platform.BackgroundSurface`, `fillUncovered`): what shows
    meanwhile is the background, not black. As the window shrinks, the
    surface takes the new size and presents its frame before DWM takes
    the window's, so DWM may show the window at its last size for a
    refresh, and around the surface the window's own pixels where the
    surface was, which the swap chain covered: the window paints them with
    the background too, under the surface, at each size (`fillUnder`),
    unless its frames are drawn in memory onto them. Until a frame of its
    content shows in it, the window is cloaked (`DWMWA_CLOAK`, from the
    surface's creation): shown, it would show white while the renderer
    starts and the first frame draws, a few hundred milliseconds (#905).
    Windows sends it `WM_PAINT` all the same, and uncloaks it after the
    one whose frame reached the surface (`platform.FirstFrameSurface`),
    or a second after it was shown without one (a timer, which only an
    idle main thread lets fire); a refresh of DWM between the present and
    its frame showing shows the background under the surface
    (`fillUnder`). A window made hidden gets no `WM_PAINT`: its first frame is
    drawn while it is hidden (`drawHidden`), which makes it ready to
    show, and it uncloaks after its first `WM_PAINT` once shown; a
    present that DXGI reports occluded leaves the next frame to draw
    whole. Resizing the buffers takes DWM longer
    than a frame for a window about as large as a 4K screen, so while
    the window changes size the swap chain has three buffers a quarter
    larger than the window, until a frame a second after the last change,
    which a timer asks for, gives it two of the window's size again. The
    unscaled swap chain shows those buffers whole, from the top left
    corner, and the window clips them: showing only the window's size of
    them (`SetSourceSize`) would stretch that part over the buffers' size,
    magnifying the frame (#892), as it does for a swap chain for
    composition. A frame drawn whole fills what is beyond it in the
    buffers with the content's background (`clearWhole`), which shows
    where the window grows within them. A swap chain that stretches, where
    the unscaled one is refused, shows the window's size of the buffers.
    The swap chain flips in sequence (`FLIP_SEQUENTIAL`), so that a frame
    draws only what changed since the last, as Metal's (below): DXGI keeps
    what each buffer holds, and as a frame is presented with the
    rectangles it drew as its dirty rectangles (`Present1`), copies into
    it what the last frame changed outside them, so there is no buffer
    age to track. The frame clears the rectangles
    (`ID3D11DeviceContext1::ClearView`, which Direct3D 11.1 has, WARP too)
    and draws the batches scissored to each; a frame that changed nothing
    is not presented. With WARP, on a runner
    without a GPU, a caret blinking in a window of 1280×760 pixels took a
    tenth of the CPU's time it took drawn whole;
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
    about half the main thread's time it would. A resize's frame that
    AppKit asks for (`updateLayer`) after one presented with the
    transaction, outside a live resize, draws at the display link's next
    tick, as for a window resized from its view, or right after the frame
    that took back the drawables given back when idle: drawn in AppKit's
    display of the view, presented with the transaction too, it did not
    show, and the window kept the last frame stretched to its new size
    (#1044). The main thread never
    waits for the display: after each frame presented as soon as drawn, a
    goroutine on a thread of its own (`present.go`) takes the next frame's
    drawable, which `nextDrawable` holds back while the display holds the
    others, most of a refresh at times. A frame that comes while it does
    is built at once, its canvases and instances ready, and waits
    (`pendingFrame`); once the goroutine has the drawable, it has the
    main thread (`SetPost`, through `surface.Conn.Post`) draw the frame
    into it and present it. Frames of the display link's ticks
    (`platform.SurfaceEvent.Paced`) are not built while one waits
    (`Renderer.Ready`, `windowHost.waitDisplay`): the next draws once it
    was presented (`platform.PacedSurface.FrameNow`), at the time its tick
    asked for it, so that what moves keeps its pace; the link keeps
    ticking meanwhile. Other frames, as a resize's, draw the waiting one
    first. Frames presented with the transaction, which change the layer,
    take their drawable on the main thread, as the first after a pause
    does, after any being taken. Building a frame before its drawable
    comes, as blocking for it did, keeps the GPU as busy: building only
    once the drawable came drew a sixth fewer frames where the GPU took
    most of each refresh. Instances use two frame slots, for
    the main target and each canvas, and a slot waits only for the command
    buffer that last used it. Updating an atlas or image texture in place
    still waits for every frame in flight. Rendering a canvas into its
    texture stays ordered on the command queue. The layer keeps three
    drawables, as Core Animation makes while a window animates: with two,
    on a ProMotion display, frames waited for one most of each refresh,
    and a fifth of the frames went undrawn. The third costs a frame of memory while a window animates.
    Waiting for a drawable remains separate from waiting for an instance
    buffer, so offscreen encode timings do not establish presentation latency.
    Opaque fill cores use an unblended pipeline in BGRA8; float16 keeps
    blending to preserve its interpolation rounding. Fragment functions are
    specialized by batch class and continuous corners, with general shaders
    as an identical fallback. Runs of small batches of different classes
    (each covering at most 128×128 pixels, as a box's shadow then its
    fill) join into one batch whose function holds just their classes'
    paths (`Batch.Joined`): a draw call costs the CPU more than those
    pixels cost the joined function. A worker compiles and warms these pipelines,
    cached per device, outside the frame path. Renderer release stops and
    joins that worker before releasing its shader library. Float16 shadows
    stay general, and fills with fractional opacity avoid the plain class,
    because specialization otherwise changes rounding. Function constant
    indices 0 through 7 belong to the renderer; effects start their own at 8.
    Two seconds after the last frame, as the
    driver frees its own memory of frames, a timer shrinks the drawables,
    which frees all but the one shown, until the next frame makes them
    again, and releases GPU instance buffers, atlas/image textures and
    backdrop textures: an idle window keeps one frame of memory. Its
    drawables are not `framebufferOnly`, as effects sample the drawable
    for their backdrops.
    A frame draws only what changed since its drawable was drawn. Core
    Animation hands the layer's drawables out in turn, and each holds the
    frame drawn into it last: the renderer keeps, by the drawable's
    texture (retained, so that no other takes its address), what changed
    in the frames since (`gpu.Ages`, as EGL's buffer age says), found by
    comparing each scene with the last (`scene.Differ`, as the CPU
    renderer does), grows it by the backdrops of the effects it meets,
    and, while that is less than half the frame, loads the drawable
    rather than clearing it, copies a texel cleared to the frame's color
    over the area (with the `down` pass, so that the GPU converts the
    color as it does clearing), and draws the batches scissored to it,
    leaving out those outside. The area is up to eight rectangles apart
    from each other (`gpu.Part`, from the Differ's, which `gpu.Ages`
    keeps for each drawable, those that meet or nearly do merged, and
    past eight the two whose union grows least): a spinner in one corner
    and a progress bar in the other redraw themselves, not all between,
    each batch drawing once in each rectangle it meets, so that none is
    blended twice. A backdrop an effect reads is within one rectangle,
    drawn before the effect reads it. Apple's tile-based GPUs then load and store
    only the tiles the pass draws in: a caret blinking in a window of
    3200×2000 pixels takes 0.4 ms of the GPU a frame rather than 2.4,
    and a spinner 0.4 rather than 2.9, on an M5. Copying the last frame
    forward instead (a blit) costs about what drawing it whole does, 0.34
    ms against 0.43 at 3456×2000. A frame whose scene is the last one's
    draws nothing and takes no drawable. Apple does not document that
    drawables keep what was drawn: a drawable not seen yet, the drawables
    of a new size or format, and those after the timer shrank them draw
    whole, and `MYGO_PARTIAL=0` draws every frame whole. The pixels are
    those of the frame drawn whole, byte for byte: an op that the builder
    puts under another pipeline than in the last frame (a batch
    specialized for other classes, a pipeline made meanwhile) draws a
    unit in the last place apart here and there, so its bounds count as
    changed (`pipelineDamage`), as do those of a clip that starts or stops
    holding an op drawn under a transform, which it then cuts with a mask
    (`scene.Differ`); `TestPartialFramesMatchWhole` draws random scenes
    changing frame after frame into three targets in turn and compares
    each frame with it drawn whole. Colors
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
    differs, `load_linux.go` or `load_js.go`. A frame draws only what
    changed since the last (`Renderer.Partial`, with `scene.Differ` and
    `gpu.Pipelines`, as Metal's), into a framebuffer that keeps what was
    drawn, so there is no buffer age to track: it clears the rectangles of
    the area with the scissor test on, which `glClear` keeps to, and
    draws the batches scissored to each. GTK keeps the GtkGLArea's framebuffer from render to
    render, but makes its buffers anew, holding nothing defined, whenever
    it allocates the area, at the same size too: a `SurfaceResize` has the
    next frame draw whole (`Invalidate`). Once the area has a context, GTK
    composites the whole window with OpenGL and tells EGL what each paint
    changed (`eglSwapBuffersWithDamageEXT`) in the window's coordinates
    unscaled, where EGL takes the buffer's pixels from the bottom left: at
    a scale of 2, Mesa told a compositor that a change at the top of the
    window was in its lower half, at half its size. A compositor that
    redraws only what it is told changed (Mesa's llvmpipe tells it whole
    frames changed, a GPU's buffers what GTK said) then showed title
    buttons the pointer hovered flickering or gone, and a shadow missing on
    some sides (#1071). At a scale of 2 or more on Wayland, the toplevel's
    `draw` signal adds what GTK repaints (cairo's clip) as
    `wl_surface.damage`, in the surface's coordinates, to the commit of the
    swap that follows (`damageGLPaint`); `TestGLDamageWayland` checks it
    under `weston --scale=2`. A canvas's WebGL 2 context keeps
    its drawing buffer (`preserveDrawingBuffer`), which browsers then copy
    as they composite each frame drawn; in Chrome on an M5 a caret blinking
    still took a quarter of the GPU's time it took drawn whole, and a frame
    that changed nothing leaves the canvas alone. On Mesa's llvmpipe a
    caret blink in a window of 1280×760 pixels took a sixth of the time.

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
  redraws only what differs from the last scene (`raster.Renderer`, with
  `scene.Differ`, which compares the operations of two scenes from both
  ends), with the effects reading their backdrops that meets and those
  backdrops, in rectangles apart from each other (`AddBackdrops`): it is
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
  clip. GTK neither paints a window that does not show nor runs its
  frame clock: until its first frame reached the surface
  (`platform.FirstFrameSurface`), `RequestFrame` of a window made hidden
  also adds an idle callback, which realizes the window, giving the area
  the size it will show at, and draws that frame in memory, showing it
  nowhere (`drawHidden`), which makes the window ready to show. A
  GtkGLArea has no context until it shows: the frame is followed by
  `SurfaceRenew`, so that the first in its `render` signal makes the GPU
  renderer, which the hidden one could not.

  GTK's frame clock waits, after each frame, the refresh interval of that
  frame's timings, and 1/60 s while none has one. GDK fills it in only
  from the compositor: on Wayland in the frame callback of a paint it
  commits itself, which it never asks for when it composites the window
  with OpenGL (`eglSwapBuffers` commits, with a swap interval of 0), and
  on X11 from a compositor sending `_NET_WM_FRAME_TIMINGS`. A GtkGLArea
  drew 60 frames a second on a 165 Hz display under Niri (#1219). The
  surface writes the interval of the monitor showing the window
  (`gdk_monitor_get_refresh_rate`, asked again each frame, so it follows
  the window to another monitor) into the timings of each frame it draws
  (`paceFrameClock`), unless they have one. `GdkFrameTimings` is private,
  and GTK 3.24.24 moved its `refresh_interval`: the offset is the one
  `gdk_frame_timings_get_refresh_interval` reads the write back from. A
  monitor of unknown rate, as Xvfb's, keeps 60 Hz, and a compositor's
  interval arriving later replaces the surface's. The clock still runs
  only while frames are asked for. On Wayland these frames follow a timer
  at the display's rate, not its refreshes.

  On Windows, `RequestFrame` invalidates the surface's window, whose
  `WM_PAINT` comes as soon as the message queue is empty, and frames drawn
  in memory show at once, through GDI or DirectComposition: nothing held
  them to the display, and a drawing asking for every frame
  (`Painter.AnimationFrame`) drew 5,000 a second on a 60 Hz display. After
  each, a goroutine waits for DWM's next composition (`DwmFlush`, through
  `platform.RefreshSurface`), for half a refresh at least, as it may
  return at once, and two at most, and paced frames (`WM_PAINT`, but a
  resize's and `FrameNow`'s) wait for it as they do for the swap chain's
  room (`refreshWaiter`, `windowHost.waitDisplay`): the next draws once it
  returns, through `FrameNow`, and the main thread handles input
  meanwhile.

  With a GPU renderer, every frame draws on the GPU, small changes too, so
  a gesture never switches renderers mid-way: on the CPU, a frame
  redrawing glass or a blur that content scrolls under takes 30 to 70 ms
  on a large window, where Metal takes 2 to 3. Every GPU renderer draws
  only what changed (above).

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
  (`ui/headless.go`) with the CPU renderer, at its own time
  (`engine.clock`), which stands still until `Tester.Advance` moves it.
  Its host runs no timer; it notes their deadlines, and `Advance` runs
  the frames due. `Tester.Access` and `AccessByRole`
  (`ui/tester_access.go`) build the last frame's accessibility tree on
  demand, as `accessTree` does for a window, and give its nodes as
  `AccessInfo`: the role an element was given (`node.role`), so that
  roles sharing a node's role stay apart, or the one following the
  node's in the order of `Role`; a role added to both needs nothing
  there, a state a field. The fake backend's surface lets the core's
  tests drive content windows through `package mygo`.
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
  Before that, with `i18n` in the configuration, it checks the catalogs
  and writes their functions (see Catalogs of messages), as `dev` and
  `build` do; `i18n check` only checks them. `dev -lang` passes the
  language to the app as `MYGO_LOCALE` (in browsers, through `go.env`).
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
  anything else fails. A name starting with a dot is skipped when it is a
  directory or a file manager's dropping (`.DS_Store`, `Thumbs.db`,
  `desktop.ini`), and at the top level of `resources/` whatever it is; a
  dotfile inside a resource directory ships (`junkName`, `hiddenName`),
  since packages read such names. So is what `resourceFilter` excludes
  from `resources/` (but not from listed paths): Go source files, which
  only builds read, and the paths there that match `resourcesExclude`
  (`path.Match` segments, `**` for any number of them), with what a
  matching directory holds; the merger's
  listings and `resource.walk`, which copies and fingerprints, apply it,
  while `mygo dev`'s watcher still sees those files, which a package may
  embed. The entries of these directories and listed paths are followed
  when they are links, links inside them are copied as links,
  permissions are kept. Installed paths must be unique ignoring case
  (listed resources never merge), and top-level names must not replace
  the packaging's own files
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
  architectures of a universal app combined. The CLI and the plugins
  download into that cache with `internal/download`: a 5xx, 408 or 429
  response, or a connection that fails or breaks off, is tried again
  after 1, 2 and 4 s (GitHub's release downloads answer 502 and 504 now
  and then), and a lock on `.lock` in the file's directory (`flock`,
  `LockFileEx`) has processes that need the file at once download it
  once: the others wait, then find it. Without it each renamed its own
  copy onto the file, which Windows refuses ("Access is denied") while
  another process has it open or loaded.
  `resources/icon.png` is the default icon. Each platform's output
  directory is assembled in a staging directory that replaces it whole, so
  removed resources do not linger; development builds on Linux and Windows
  remove what the previous build placed and this one lacks.
- Packages for the other platforms. Windows gets "<name> Setup
  <version>.exe", made with NSIS (`nsis.go`): a per-user install in
  `%LOCALAPPDATA%\Programs\<name>` (`windows.installDirName` names the
  folder otherwise), where the updater can write and where the installers
  of later versions install too (`InstallDirRegKey` reads the uninstall
  entry's `InstallLocation`), a Start menu shortcut, a desktop shortcut
  that the finish page's second check box (MUI's "show readme" one)
  creates, and an uninstaller registered
  under `HKCU\…\Uninstall\<identifier>`; `/S /D=<dir>` installs
  silently, without the desktop shortcut. The uninstaller runs from a
  copy of itself that nothing waits for, so it removes the app's folder
  last: the folder is gone once the uninstall is done. It tries deleting
  each shortcut again for up to 5 s, as Explorer opens a new shortcut a
  few seconds after it appears, not sharing it for deletion, and `Delete`
  fails meanwhile. `windows.installerInclude` names a script of the
  app's own that the generated one `!include`s after MUI2.nsh and the
  languages (`nsisHooks`); its `customInit`, `customInstall` and
  `customUnInstall` macros, each inserted only under `!ifmacrodef`, run
  in `.onInit`, at the end of the install section and in the uninstall
  section before `RMDir /r "$INSTDIR"`, as electron-builder's do; without
  it the script is unchanged.
  `makensis` comes from an installation of NSIS or, on Windows, where NSIS
  is rarely installed, from the official zip of the release that
  `Release` in `internal/nsis` pins, which the CLI downloads once, checks
  against its SHA-256 and unpacks into `<user cache>/mygo`, as Tauri
  does. It tries the copies of
  the zip in `Release.URLs` in order, until one answers with the right
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
  of the app: the app in `/opt/<name>`, a `/usr/bin` link for each name
  of `linux.command`, the desktop entry (categories,
  comment, URL schemes; `Exec` is the app's path) and hicolor icons,
  depending on GTK 3 and WebKitGTK 4.1. It also gets a `PKGBUILD`
  (`pkgbuild.go`), from which `makepkg -si` packages the archive for
  Arch Linux (`source` is its file name, with its SHA-256): its
  `package()` installs what the Debian package holds, with the same
  modes, the desktop entry written in it with `printf` as `deb.go`
  renders it, and the hicolor icons, which the build resizes as for the
  Debian package and writes next to it as more sources
  (`<name>-256x256.png`, `<name>-512x512.png`); pacman's hooks register
  them, the MIME package and the desktop entry, as the triggers of
  Debian do. `pkgver` is the version without the hyphen of a
  pre-release (`1.2.0beta.1`; vercmp sorts `1.2.0_beta.1` after
  `1.2.0`, as a remainder starting with a separator wins), `depends`
  maps the Debian dependencies through `archPackages`, warning about
  those it lacks, and `options` keeps the executable unstripped. Every
  value is quoted for the shell, and `-upload` leaves it out (`uploads`
  takes only known kinds). Packages hold the same files as
  the update archive; apps installed by a package manager do not update
  themselves (`Updater.Enabled` checks that the app can write where it is
  installed). Every Linux build also gets `install.sh`
  (`installscript.go`), a POSIX sh script, the same for every
  architecture, that installs the archive for the user in
  `~/.local/<name>.app` (which updates can replace), links
  `~/.local/bin/<command>` for each name of `linux.command` that no other
  program has (and removes a `~/.local/bin/<name>` link of its own, which
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
- On a macOS host, macOS targets also get "<slug>-<version>-<arch>.dmg",
  or "<slug>-<version>.dmg" for darwin/universal
  (`dmg.go`): the CLI puts a copy of the app (`ditto --noclone`, since a
  clone of a sparse file takes fewer blocks than its copy on HFS+ will),
  the `/Applications` link, the volume icon and a `.DS_Store` written in
  pure Go (`dsstore.go`, byte-identical to dmgbuild's `ds_store` package)
  that lays out the Finder window in a folder, `hdiutil` creates a writable
  HFS+ image of it, as large as its files need (APFS where HFS+ fails, as
  on macOS 26.7.1), the CLI sets the volume's
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

Regenerate checked UI wrappers with `go generate ./ui`, then what
`mygo migrate-ui` knows of their results (`internal/uimigrate/uiapi.go`,
which `TestUIAPI` checks) with `go test ./internal/uimigrate -update`.
Native library and shader generation commands are listed in [AGENTS.md](../AGENTS.md).

| suite | command | covers |
|---|---|---|
| core | `go test .` | lifecycle, quit, documents, IPC, channels, events, Eval, protocol, frontend URLs and serving, menus, trust, single instance, dev ready signal (fake backend); `go test -run '^$' -bench .` measures the Go side of IPC and custom schemes |
| generator | `go test ./internal/tsgen` | TS output, json/v2 rules, source lookup; type-checks the output with `tsc` when `bun install` was run |
| CLI | `go test ./cmd/mygo` | config, Info.plist, icons, universal binaries, template, dev launch/ready/stop (the test binary plays the app), watcher and `go list` inputs, the console's output (replayed as a terminal shows it), resources (platform directories, universal pairs, exclusions, staging, conflicts, dev placement; builds for every OS), frontend embedding (compiles an app with the overlay), `.DS_Store` against a dmgbuild golden file, a real DMG (`hdiutil`); builds and tools are skipped with `-short` |
| runtime | `bun run test` | the injected runtime, `mygo-runtime` and the plugins' packages (against a fake Go side on the real runtime, `plugins/fake-go.ts`) |
| plugins | `go test ./plugins/...` | the fetch plugin against `httptest` servers, the WebSocket client against a test server (ordering, fragments, pings, closing handshakes); the terminal's binding of libghostty-vt (layouts, rendering, encoders, selections), its pseudo-terminals, and its view through `Tester` with real shells: typing, keys as programs ask, input methods, mouse reports, selecting and copying, pasting, scrollback, exits (the library is downloaded, or named by `MYGO_GHOSTTY_VT`; `-short` skips them); the automation plugin's protocol (requests, errors, elements as JSON, input events) against a fake app, and its socket (mode, stale and taken paths); `mygo-automation` (`go test ./internal/automationcli ./plugins/automation/cmd/...`): lines, refs and their staleness, diffs, sessions, and its commands against a fake socket; `skills` printing `SKILL.md` as it is |
| native UI | `go test ./ui ./internal/text ./internal/scene ./internal/raster ./internal/svg ./internal/gpu/...` | the GPU renderers against the CPU renderer (Direct3D on Windows, Metal on macOS, OpenGL on Linux); views through `Tester`: input, focus, editing, lists, overlays, frames that fill the glyph atlas; text layout and caret geometry; atlas zones and repacking; the CPU renderer against its formulas; SVG parsing and drawing, with `FuzzParse`; `go test -run '^$' -bench . ./ui` times a frame |
| GUI | `MYGO_E2E=1 go test ./internal/e2e` | the real backend: an app driven from another process through `plugins/automation`, and with `mygo-automation`'s commands, IPC, channels, protocol, Eval, geometry, capture, menus, window.open, native UI (frames, clicks, input methods replacing typed text, file drops, assistive technology reading and acting; on macOS typing, skipped while an input method is selected, and composing; on Linux with `MYGO_GPU=1`, what OpenGL drew in the GtkGLArea, from the first frame and once a window drawing in memory asks for the GPU); on Windows too (a GitHub Actions `windows-latest` runner has WebView2) |

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

### Waiting in tests

A test never decides on how long something took. CI runners are slow
and loaded at random, so a test that sleeps then checks, or gives up
after a few seconds, fails now and then although the code is right, and
a longer timeout only makes that rarer. So:

- Wait for the condition or the event itself, with `internal/wait`:
  `wait.Until(t, "the window closed", w.IsDestroyed)`, `wait.For(t,
  "the reply", ch)`, `wait.Context(t)` for calls that take a context.
  Its guard, a minute, only turns a hang into a failure that says what
  never happened. It is not tuned: a wait that reaches it found a race
  that lost the event, or a condition that can't become true, and the
  fix goes there.
- Wait for what the product says it did, not for a side effect that
  comes later: the backup's state rather than its file, an identity the
  watch saw rather than a stat after it, the input queue drained rather
  than bytes counted.
- Native UI tests move the `ui.Tester`'s clock with `Advance`; it stands
  still otherwise, so animations, delays and double clicks don't depend
  on the machine.
- Performance is measured by counts (steps, allocations) in tests, by
  time only in benchmarks, which `bench.yml` tracks.
- Checks that nothing happens (`select` on `time.After` with nothing to
  fail) may wait a short while: a slow machine can only make them miss
  a bug, never fail right code.

### What tests leave behind

The GUI tests of a package share a process, and on Linux an X server,
which can outlive the process when one is reused between runs. A test
that leaves something in them can fail another much later, or in every
other run as a lock key flips, which looks like a race in a test that did
nothing wrong. `newWindow` checks after each test that used it (`watchLeftBehind`), and fails
that test by name if it left a window open or, on an X server, a key held
or Caps Lock or Num Lock changed, then puts them back so that the tests
after it are not affected. A test that changes them undoes it in a cleanup.

### Benchmarks

`.github/workflows/bench.yml` measures every push to main on GitHub's
macOS, Linux and Windows runners, and the website shows the results at
[/measures](https://mygo.egoist.dev/measures). `scripts/bench.ts`
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
`website/src/lib/measures.ts`): a commit whose value, and the median of
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

The Metal `BenchmarkRenderFillRate` benchmark covers nested opaque layers,
large border rings and stacked shadows at 3456×2000. It reports command
buffer GPU timestamps with the existing benchmark harness; it waits for
individual offscreen frames and does not measure display scheduling or
input latency, which `internal/inputlatency` measures (below). `MYGO_FRAME_STATS` includes instance and batch counts,
culled operations, opaque cores and Metal specialization fallbacks.
`MYGO_GPU_REQUIRED=1` makes the Metal optimization identity tests fail if
no renderer can be created. They compare raw BGRA8 and float16 output with
both disabled optimizations and the shader from upstream f2eca9f7, including
canvases and pixelated sprites.

`BenchmarkStressFrame` in `ui` times the frames of views that change all
they show every frame, in a window of 900×640 at scale 2: 600 rounded
boxes casting shadows moving and changing color, 300 labels whose numbers
change, a column of 1000 rows in a clipped box moved by a translation,
a `List` of 100,000 rows scrolling 3 rows a frame, and a dashboard of 12
cards with shadows, texts and progress bars. Its host (`stressHost`)
hands the scene on rather than draw it on the CPU, so `ns/op` is the
frame built, laid out and painted. `BenchmarkStressFrameGPU` draws the
same frames with Metal (`metal.Offscreen`), its `GPU-ms` the GPU's time,
which the page shows as the GPU metric, and its `encode-ms` the CPU's
time encoding each frame (building its instances, uploading what changed
in its atlases, encoding its passes), the Encode metric, as the Metal
renderer's benchmarks report too (`BenchmarkRenderPanels`: a page of
cards taking hundreds of batches).

Benchmarks are described on the page by their doc comments, which the
site reads when it is built: start them with the benchmark's name.

### Input latency

`go run ./internal/inputlatency` measures, on macOS, how long input takes
to show on screen in native UI: from when an event arrives to when the
frame showing it reaches the display, as its drawable's presented handler
tells (`presentedTime`). Each run is a process with a window shown without
activating the app, which types into a text area, scrolls a list with the
trackpad or drags a box, with events at any time within a refresh, or
animates with no input (`anim`, and `heavy`, which keeps the GPU busy
most of each refresh), and prints what the latency is made of: waiting
for the main thread, handling the event, then to the frame's present
(the next tick, building, a drawable), and to the screen; and of the
frames, how long the main thread waited for drawables, and what share of
the refreshes showed a new frame. The window server settles
in a state that lasts a run and changes from run to run, so compare the
medians of several runs (`-runs`, six by default), alternating builds.

On an M5 with a 120 Hz external display (eight runs each, Oct 2026), a
key typed in a text area shows in 26.6 ms on average (99th percentile
31.9), a trackpad scroll of a list in 27.2 ms (31.4) and a drag in 27.6 ms
(31.7); on the built-in display at 60 Hz, 33, 46 and 43 ms. Of a scroll's
27 ms, 4.2 wait for the display link's next tick (half a refresh), 0.4
build and paint the frame, no time waits for a drawable, and 22.7 go from
the drawable to the screen: the window server composites the frames it
has at a refresh, and the display shows that composite two refreshes after
the refresh the frame was presented in, the third when it was presented
too late in the refresh before. A bare CAMetalLayer window takes as long. A
typed key's frame starts at once (a frame after a pause draws in the pass
of the run loop that asked for it), but handling the key takes 3.4 ms:
the text input tells the input method where the text it changed is
(`textInputAdapter.changed`), which lays its paragraph out, here 1,800
characters, which the frame then reuses.

What was measured and left as it is:

- *Drawing late in the refresh*, a deadline before the next vsync rather
  than at the tick, for frames to show later input: the window server's
  composite comes from half a millisecond before the vsync to a
  millisecond and a half after it, from run to run, and the GPU must have
  finished the frame by then. Drawing 3 ms before the vsync saved 2 ms of
  29 when frames presented at the tick missed the composite, was 11 ms
  slower in a run where they made it, and drew a fifth fewer frames.
- *Skipping a refresh when frames show a refresh late*, as the presented
  handlers tell: in a bare
  CAMetalLayer window a skip brought most runs from 25.9 to 17.5 ms, but
  MyGo's frames, which take longer from the tick to their present, stayed
  late after skips.
- *CAMetalDisplayLink* with `preferredFrameLatency` 1 or 2 took a refresh
  more than drawing at a CADisplayLink's tick (33 against 17 to 25 ms at
  120 Hz), besides keeping the layer from giving drawables outside its
  callbacks (synchronous resize frames).
- *Two drawables* or *presenting with the transaction* settled in the same
  states from run to run, with no gain that held, and with two drawables
  frames waited for them.

On Windows, frames that make the refresh without building ahead present
with sync interval 0 right after the window manager composed the last,
and heavier ones wait ahead of the screen (Direct3D 11 above). Pointer
moves posted to a window of 900×640 pixels with 300 boxes moving show in
25 ms on average (a Radeon 780M at 60 Hz, from the post to the desktop
image's `LastPresentTime`), and took 43 ms when every frame was built ahead: half a refresh waits for the next frame on average, half
a millisecond builds it, and a refresh or so goes from its present to
the screen. Linux's frames follow GTK's frame clock, which is not measured:
the containers have no display that refreshes.

### Comparison with other frameworks

`benchmarks/` holds the same small apps written with MyGo's native UI, GPUI,
GPUI Fast and egui, and `go run ./benchmarks/run`, which measures them from outside the
process: see `benchmarks/README.md`. It runs by hand on a real Mac, and
`benchmarks/results.json`, which it writes, is what the website's
[/benchmarks](https://mygo.egoist.dev/benchmarks) page shows.
`.github/workflows/comparison.yml` only checks that the apps build and run.
An app of a scene does what the others do, each the usual way of its
framework: a scene changes in all of them, or in none.

### Binary size

Go's linker keeps what `main` can reach: a function called from anywhere
reachable, every method of `platform.Backend` and of the other interfaces
a backend is used through, and the data those touch (a table read on
first use is linked whole). So a feature costs every app as soon as
something every app runs calls it, even on a path that never runs. The
rules, which took counter-native back to 8.9 MB after it had grown from
10.8 MB to 12.2 MB:

- *Opt-in features hang off their constructor.* What every app reaches
  (`App.Run`, `NewWindow`, the default menus and their roles, `init`
  functions, locale resolution) calls an opt-in feature only through
  functions the feature registers when the app first uses it: the File
  menu's document roles and Open Recent call `linkedDocuments`, which the
  first `NewDocuments` fills; `App.Run` writes the TypeScript client
  through what `Bind` and `NewEvent` register; messages format dates
  through what `i18n.NewCatalog` registers. Generic constructors are only
  linked when instantiated, so they are a good place to register.
- *No reflective method lookups on shared paths.* A reachable call of
  `reflect.Type.Method`, `reflect.Value.Method` or `MethodByName` with a
  name the compiler cannot see makes the linker keep every exported method
  of every type used through an interface: 2.5 MB of counter-native. Only
  `Bind`, `BindAs` and plugins with a service do it; MyGo's own services
  use `bindFuncs`. `text/template` and `html/template` do it too.
- *Backends stay thin.* Every method of `platform.Backend` and its
  controllers is linked into every app of that platform, used or not, so
  they translate calls and keep policy, parsing and data in the core or
  the feature's package.
- *Heavy packages and data stay where they are needed*: in a subpackage
  or a plugin (`ui/editor`, `ui/markdown`, `plugins/watch`) rather than in
  `mygo` or `ui`, and out of `init`. Importing net/http alone adds 2.3 MB.
  Data read on first use is linked whole, so split a table that a shared
  path reads in part, as CLDR's numbers and dates are; embed what the
  running app reads, not what tools read (`internal/l10n` embeds its
  messages without their descriptions).

To check a change, build both apps as the benchmarks do, before and
after, on each platform it touches:

```sh
go build -trimpath -tags mygo_noinspector -ldflags "-s -w -X github.com/egoist/mygo.production=1" -o /tmp/counter-native ./examples/counter-native
GOOS=windows go build -trimpath -tags mygo_noinspector -ldflags "-s -w -H=windowsgui -X github.com/egoist/mygo.production=1" -o /tmp/hello.exe ./examples/hello
```

When an app grows, `go tool nm -size -sort size` on a build without `-s
-w` shows which symbols grew, and `go build -ldflags=-dumpdep ... 2>
deps.txt` prints every edge `a -> b` of the linker's walk: follow the
first edge to a symbol back to `main.main` to see why it is there, and
search for `<ReflectMethod>`, which marks a reflective lookup that keeps
every method.

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
Before the tests, each job downloads what they would download into the
user's cache otherwise (`go run ./internal/prefetch`): the plugins'
native libraries for its platform, and NSIS on Windows. It keeps them in
the Actions cache, under a key of the files that pin them (the plugins'
`mygo-plugin.json` and `internal/nsis/nsis.go`), so that the tests do
not depend on the network.

The NSIS that `mygo build` downloads (`Release` in `internal/nsis/nsis.go`)
has a copy on a release of this repository, which a new version of NSIS
needs before a MyGo release that pins it. Download the official zip,
check that its SHA-256 is the one `Release` pins, and publish it,
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
6. **Keep apps that don't use it the same size** (see "Binary size"):
   build `examples/counter-native` and `examples/hello` before and after.
7. **If the page runtime changes**, edit `packages/bridge` or
   `packages/runtime`, run `bun run test`, `bun run typecheck` and
   `bun run build`, and commit `internal/bridge/bridge.js`. If the generated client changes, update
   `internal/tsgen/generate.go` and its tests.
8. **Document** the behavior in the Go doc comments and platform
   differences in the README.

<!-- repository-only:end -->

## Platform differences

| feature | macOS | Linux | Windows |
|---|---|---|---|
| watching files | FSEvents, one stream per tree, no descriptor per file | inotify, per directory | `ReadDirectoryChangesW`, one for a recursive tree, and the parent to see the directory renamed |
| menu bar | application menu bar, default menu installed | per-window GTK menu bar, none by default | per-window Win32 menu bar, none by default |
| menu item icons | `NSMenuItem` image, 16 pt, template images tinted | a `GtkBox` of a `GtkImage` (a cairo surface at the widget's scale) and the label; none on the bar's own items; tray menus' items get `icon-data` over DBusMenu (`dbusmenu_gtk_parse_get_cached_item`) | Windows 11: drawn by MyGo in a column after the checks' (premultiplied BGRA, `internal/menuicon`); Windows 10: `hbmpItem`, a 32-bit premultiplied DIB section freed with its menu; none on the bar's own items |
| menu item subtitles | `NSMenuItem` `subtitle` from macOS 14; before, after the title (`Label — Subtitle`) | after the label | after the label, with its ampersands escaped |
| auto-hide menu bar | ignored | the bar widget hides; `can-activate-accel` keeps its shortcuts; Alt alone or F10 show it and open its first menu until it deactivates | the menu is attached only for the `SC_KEYMENU` menu loop that Alt alone or F10 start; shortcuts come from the webview |
| tray | NSStatusItem, click events (a Control-click a right-click) | AppIndicator: a menu, and activation (a double click) for an icon with an `OnClick` listener | notification area icon, click events |
| global shortcuts | Carbon hot keys | X11: `XGrabKey` on the root window (with Caps/Num Lock variants), key presses from a GDK filter. Wayland: the XDG `GlobalShortcuts` portal (see [Linux](#linux-internallinux)) | `RegisterHotKey` |
| notifications | UserNotifications, packaged apps only; explicit alert authorization; body/action delegate attached at launch; `Group` is `threadIdentifier` | org.freedesktop.Notifications over D-Bus, including action IDs; no `Group` | registered WinRT toast, stable AUMID/CLSID, COM warm/cold activation; no balloon fallback |
| notification removal | delivered and pending requests; removed submissions are invalidated by generation | `CloseNotification` on the bus, for those of this run | deterministic tags plus sender history removal, including earlier runs; uninstall checks cleanup before unregistering |
| a web view scaled and rounded with native UI (`WebViewPlacement`) | the pane's bounds scale it, its layer's corners cut it | WebKitGTK zooms the page to its view's size, which `resize`s it at the same width; `draw` clipped to the corners on whole pixels | the visual's transform scales it, a rectangle clip cuts it |
| WebViewAvailable (any goroutine, before `Run` too) | `WKWebView` is there, which macOS always has | WebKitGTK 4.1 (or 4.0), JavaScriptCore and libsoup load, as `webKit` loads them for a window showing a page | `internal/webview2` finds a runtime, or a `WebView2Loader.dll` next to the executable exports what `createEnvironment` calls; once an environment was made, whether that worked. A runtime whose files are there but whose browser is not passes, and fails later, making the web view |
| secrets | Security.framework generic passwords in the login keychain (`SecItem*`), service and account as keytar | libsecret's `org.freedesktop.Secret.Generic` schema (attributes `service`, `account`), loaded on first use, libsecret 0.19 or later; no libsecret, no session bus (`g_bus_get_sync`) or no Secret Service on it (`G_DBUS_ERROR_SERVICE_UNKNOWN`, `NAME_HAS_NO_OWNER`, `NO_SERVER`) is `ErrUnsupported` | Credential Manager generic credentials (`CredWriteW`), target `service/account` |
| vibrancy | all materials, behind pages and native UI; Liquid Glass (`NSGlassEffectView`) from macOS 26, the popover and HUD materials before, with a `tintColor` and the native UI's `Glass{Native}` placed over web views; frameless windows cut to `CornerRadius`; `SetActiveAppearance` through AppKit's private `_setForceMainAppearance:` | ignored | Windows 11 22H2 Mica, Acrylic, Tabbed, behind pages and native UI, in windows created with a material, which have no menu bar; Liquid Glass as Acrylic; frameless windows given a `CornerRadius` rounded by DWM |
| active windows | the core's state, from key windows; materials follow AppKit's (`NSVisualEffectStateFollowsWindowActiveState`), but panels', flyouts' and kept ones' (`NSVisualEffectStateActive`) | the core's state, from GDK's focus; GTK's own title bar goes `:backdrop` as a focusable flyout takes the keyboard | the core's state, from `WM_ACTIVATE`; DWM's Mica and Acrylic follow the frame, kept active while the window's focusable flyout is (`WM_NCACTIVATE`) |
| accent color and accessibility preferences | `NSColor.controlAccentColor` (10.14+), `accessibilityDisplayShouldReduceMotion` (10.12+), `accessibilityDisplayShouldReduceTransparency` and `accessibilityDisplayShouldIncreaseContrast` (10.10+), with `NSSystemColorsDidChangeNotification` and `NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification`; no text scale | GTK's `gtk-enable-animations` and theme name, and the settings portal's accent, contrast and GNOME's `text-scaling-factor` (`ReadAll`, `SettingChanged`); no reduced transparency; the pointing devices from the default seat's, as they come and go | DWM's `AccentColor`, `SPI_GETCLIENTAREAANIMATION`, Personalize's `EnableTransparency`, `SPI_GETHIGHCONTRAST` and Accessibility's `TextScaleFactor`, with `WM_SETTINGCHANGE`; the pointing devices from `SM_MOUSEPRESENT`, `SM_DIGITIZER` and tablet mode, not followed as they are plugged in |
| preferred languages | the global `AppleLanguages` (`CFPreferencesCopyValue`), not the app's | `LANGUAGE`, then `LC_ALL`, `LC_MESSAGES`, `LANG`; read when asked, no changes reported | `GetUserPreferredUILanguages`, `WM_SETTINGCHANGE` "intl" |
| regional formats | `autoupdatingCurrentLocale`, its calendar's first weekday, the hour of the skeleton "j"; `NSCurrentLocaleDidChangeNotification` and System Settings' distributed notifications | glibc's `nl_langinfo_l` of `LC_TIME` (`_NL_TIME_FIRST_WEEKDAY`, `T_FMT`), else by region | `GetLocaleInfoEx` of the user's default: `LOCALE_IFIRSTDAYOFWEEK`, `LOCALE_STIMEFORMAT` |
| the app's language kept | `AppleLanguages` of the app's defaults | the file `locale` in PathUserData | the file `locale` in PathUserData, and the process's preferred UI languages |
| web views' language | from the app's `AppleLanguages`, next launch | `webkit_web_context_set_preferred_languages`, live | the environment's `Language`, as it is created |
| `Theme.SetSource` | `NSApp`'s appearance (Aqua, DarkAqua, or none) | `gtk-application-prefer-dark-theme`; light also puts a theme named for its dark variant (Yaru-dark) in place of its light one (Yaru), which GTK and WebKitGTK otherwise take for dark, and the other sources reset the theme. Without a source, or with system, the setting follows the settings portal's `color-scheme` (GNOME's and KDE's dark style, which GTK 3 does not read) from `Init` on: dark where the portal prefers dark, else the desktop's own GTK settings | the title bars, menus and native UI, and pages through the DevTools protocol's emulated `prefers-color-scheme` |
| `Theme.SetGTKCSS` | nothing | a `GtkCssProvider` for the default screen at `GTK_STYLE_PROVIDER_PRIORITY_USER`, replaced by the next sheet that parses | nothing |
| a press the system cancels in native UI (`InputEvent.Canceled`, `PressCanceled`) | a context menu shown while a button is down, whose loop takes the release | `GDK_GRAB_BROKEN` of the pointer, as a menu or another window takes the grab; a touch's `GDK_TOUCH_CANCEL` taps nothing | `WM_CAPTURECHANGED` (and so `WM_CANCELMODE`) while a button is down; `WM_POINTERUP` with `POINTER_MESSAGE_FLAG_CANCELED` |
| pointer device and pressure of native UI (`InputEvent.Pointer`, `Pressure`) | a mouse event's `subtype`: a tablet's pen (`NSEventSubtypeTabletPoint`, with `pressure`, or `TabletProximity`) or a touch; `tabletPoint:` events, a pen pressing harder or softer between moves, are moves where the pointer is | the event's source device (`gdk_event_get_source_device`): a pen or eraser, or a touchscreen, its pressure axis (`gdk_event_get_axis`); the area's own touches are fingers | `WM_POINTER` messages, left to `DefWindowProc` to make mouse messages of, give the pressure (`GetPointerPenInfo`, `GetPointerTouchInfo`); the mouse messages' `GetMessageExtraInfo` (`MI_WP_SIGNATURE`) tells a pen's or a finger's |
| cursors of native UI without a system cursor | `CursorWait`, `CursorHelp`, `CursorZoomIn`, `CursorZoomOut`, `CursorAllScroll` and `CursorCell` from AppKit's private class methods (`_waitCursor`, `_helpCursor`, …, looked up with `respondsToSelector:`), `CursorProgress` its `busyButClickableCursor`, else the nearest public cursor or the arrow | the CSS names from the cursor theme, then GDK's X cursors, then the nearest (`cursorFallbacks`) | `CursorZoomIn`, `CursorZoomOut` and `CursorCell` drawn (`CreateCursor`, `cursorMasks`), the others `IDC_APPSTARTING`, `IDC_WAIT`, `IDC_HELP`, `IDC_SIZEALL`, `IDC_NO` |
| traffic lights, Dock | yes | ignored | ignored |
| hidden title bar | AppKit's traffic lights over a full-size content view | GTK's title buttons in header bars over the page, per `gtk-decoration-layout`; none where the Wayland compositor decorates windows | caption buttons drawn in a layered child window, through DirectComposition over a material, light or dark per window (`SetWindowButtonsTheme`); snap layouts; a top edge that resizes |
| progress bar | Dock tile content view (NSBoxes: NSProgressIndicator does not draw there), app-wide | Unity launcher API over D-Bus (`com.canonical.Unity.LauncherEntry`), app-wide | `ITaskbarList3`, per window |
| file icons | `NSWorkspace iconForFile:` drawn into an sRGB `NSBitmapImageRep` at the size | GIO's `standard::icon`, or a desktop entry's `GAppInfo` icon, through `gtk_icon_theme_lookup_by_gicon` | `IShellItemImageFactory::GetImage` (`SIIGBF_ICONONLY`), a DIB with straight alpha |
| icons of file types (`TypeIcon`) | `UTType typeWithFilenameExtension:` → `NSWorkspace iconForContentType:` (`iconForFileType:` before macOS 11), drawn as file icons | `g_content_type_guess` of a name with the extension → `g_content_type_get_icon`, through the icon theme | the image factory of a simple ID list (`SHSimpleIDListFromPath`), which needs no file, as file icons |
| installed apps | `.app` bundles of the Applications folders, `NSFileManager displayNameAtPath:` and `NSBundle`, off the main thread; launched with `openURL:` | `g_app_info_get_all` filtered by `g_app_info_should_show`; launched with `g_app_info_launch` and GDK's launch context | `FOLDERID_AppsFolder` enumerated in an STA of its own; launched with `ShellExecute` of `shell:AppsFolder\<ID>` |
| badge count | Dock tile label | Unity launcher API count | not shown |
| skip taskbar | ignored | skip-taskbar hint | `ITaskbarList::DeleteTab` (the window style is untouched) |
| FlashFrame | informational Dock bounce | urgency hint | `FlashWindowEx` until focused |
| visible on all workspaces | `NSWindowCollectionBehaviorCanJoinAllSpaces` | `gtk_window_stick` | ignored |
| window icon | ignored | `gtk_window_set_icon` | `WM_SETICON` at the window's DPI |
| URL schemes | Info.plist (`urlSchemes`); `RegisterURLScheme` makes the app the default handler | desktop entry + `mimeapps.list` | `HKCU\Software\Classes` |
| downloads | `shouldPerformDownload`, non-displayable or attachment responses → `WKDownload` delegate | `download-started` / `decide-destination` on the web context; response policy for attachments | `DownloadStarting` (`ICoreWebView2_4`), replacing WebView2's download UI |
| Cookies | the session's `WKWebsiteDataStore`'s `WKHTTPCookieStore`, URLs matched in Go (`platform.CookieMatchesURL`), `Set` through `NSHTTPCookie`'s parser with a stand-in name and value; `Clear` with `removeDataOfTypes:` | the session's web context's `WebKitCookieManager` (2.42+; `get_cookies` for a URL), kept in `cookies.sqlite` of the user data directory, or of a persistent session's directory (`set_persistent_storage`, as WebKitGTK keeps them in memory otherwise); libsoup 3 or 2.4 to match WebKitGTK 4.1 or 4.0; `Clear` with `webkit_website_data_manager_clear` | `ICoreWebView2CookieManager` (`ICoreWebView2_2`) of a web view of the session's profile showing a page, a window's or one in native UI, or while there is none of a hidden web view of the profile in the app window, closed once one shows a page, once an ephemeral session's store closes, or once no call needed it for 5 s and closing it loses no cookie (a `GetCookies` finds no session cookie, nor any of an InPrivate profile, which go with WebView2's processes); a window's first navigation waits for the calls under way; `ClearBrowsingData` goes the same way; changes are polled with `GetCookies` until they show, failing after 2 s; `GetCookies(uri)`, `DeleteAllCookies`; `put_Expires` called with `comCallFloat` |
| content rules | the default `WKContentRuleListStore`, which keeps compiled lists under an identifier of the id and a hash of the rules (looked up first, the id's others removed), on each window's `WKUserContentController`; the store's reason for a failure is in `NSHelpAnchorErrorKey` | a `WebKitUserContentFilterStore` in `content-rules` of the user data directory (2.24+), on each window's `WebKitUserContentManager` | matched in Go (`internal/contentrules`, indexed by the domains of `url-filter`s and four bytes of their literals) from `WebResourceRequested` with a `*` filter, blocked pages answered with an empty 403 and other blocked requests with a redirect to `data:`, which Chromium fails as a network error; WebSocket handshakes never reach it; `css-display-none` rules as a style sheet a document-creation script adopts, kept when the page sets `adoptedStyleSheets`; parsed on a goroutine, nothing kept |
| `FitContent` | the bridge, as elsewhere; WebKit's private `_setSizeToContentAutoSizeMaximumSize:` is not used; native UI measures itself (`ui/fit.go`) | the bridge; native UI measures itself | the bridge; native UI measures itself |
| `Page.ThemeColor` | the bridge, as elsewhere; WKWebView's `themeColor` (macOS 12+) is not used | the bridge | the bridge |
| `Page.ProcessID` | the web content process, `-[WKWebView _webProcessIdentifier]` (private, looked up with `respondsToSelector:`) | 0: WebKitGTK has no API for it | the browser process the app's web views share, `ICoreWebView2::get_BrowserProcessId` |
| `Page.GPUProcessID` | WebKit's GPU process, which plays the page's sound, `-[WKWebView _gpuProcessIdentifier]` (private, looked up with `respondsToSelector:`); 0 until WebKit starts it | 0: WebKitGTK has no API for it | the GPU process of the environment, the `COREWEBVIEW2_PROCESS_KIND_GPU` entry of `ICoreWebView2Environment8`'s `GetProcessInfos`, queried for each call; 0 on runtimes without it |
| a page's sound (`IsAudible`, `SetAudioMuted`) | private `-[WKWebView _isPlayingAudio]`, key-value observed, and `_mediaMutedState` / `_setPageMuted:` with the `_WKMediaAudioMuted` bit, the capture bits kept; looked up with `respondsToSelector:`, not audible and not muted where WebKit doesn't answer | `webkit_web_view_is_playing_audio` and `notify::is-playing-audio`; `webkit_web_view_set_is_muted` / `get_is_muted` (2.30+, bound when present) | `ICoreWebView2_8`'s `IsDocumentPlayingAudio`, `IsDocumentPlayingAudioChanged` and `IsMuted`, queried for each call; a mute set before the web view is made is put in `setUp`, before its first navigation |
| `BackgroundThrottling: ThrottlingOff` | `inactiveSchedulingPolicy` `WKInactiveSchedulingPolicyNone` (macOS 14+) and the private `_hiddenPageDOMTimerThrottlingEnabled`, `_hiddenPageCSSAnimationSuspensionEnabled`, `_pageVisibilityBasedProcessSuppressionEnabled` off, on a copy of the preferences when `window.open()` handed over the opener's; the private `-[WKWebView _setWindowOcclusionDetectionEnabled:NO]`; all looked up with `respondsToSelector:` | the web view's settings' `HiddenPageDOMTimerThrottling` and `HiddenPageCSSAnimationSuspension` features off (`webkit_settings_set_feature_enabled`, 2.42+, bound on first use); hidden and minimized pages still stop drawing | `--disable-background-timer-throttling --disable-renderer-backgrounding --disable-backgrounding-occluded-windows` in the environment options' `AdditionalBrowserArguments` when the page that starts the environment asks (later ones are logged); an embedded web view's controller still goes invisible with its host, which takes its page off the desktop |
| a page's capture (`CaptureState`, `SetCaptureState`) | `WKWebView`'s `cameraCaptureState` and `microphoneCaptureState` (macOS 12+) and private `_displayCaptureState` (macOS 13+), key-value observed, and their setters `set…CaptureState:completionHandler:` (nil handler), looked up with `respondsToSelector:`; `SetAudioMuted`'s `_setPageMuted:` has one bit for the camera and the microphone, so with only one of them muted the other is unmuted again after it | `webkit_web_view_get_camera_capture_state` and its microphone and display siblings with their `notify::` signals and setters (2.34+, bound when present); `webkit_web_view_set_is_muted` drops the capture mutes, which `SetAudioMuted` sets again | none: `CaptureNone`, setting does nothing |
| `Autoplay: AutoplayAllow` | `WKWebViewConfiguration`'s `mediaTypesRequiringUserActionForPlayback` `WKAudiovisualMediaTypeNone`; a `window.open()` window that doesn't ask gets the default of a new configuration back on the opener's copy | `webkit_policy_decision_use_with_policies` on every main-frame navigation, with `webkit_website_policies_new_with_policies` `autoplay` `WEBKIT_AUTOPLAY_ALLOW` (2.30+, bound when present; its variadic name/value pairs bound for one pair, as `g_object_set`): WebKitGTK decides autoplay from the navigation's policies, `WEBKIT_AUTOPLAY_ALLOW_WITHOUT_SOUND` by default, which `media-playback-requires-user-gesture` (off by default) does not lift | `--autoplay-policy=no-user-gesture-required` in the environment options' `AdditionalBrowserArguments` when the page that starts the environment asks (later ones are logged); then every page plays without a gesture |
| picture in picture (`PageOptions.PictureInPicture`) | private `-[WKPreferences _setAllowsPictureInPictureMediaPlayback:]`; `-[WKWebView _canTogglePictureInPicture]`, `_togglePictureInPicture` and `_isPictureInPictureActive`, looked up with `respondsToSelector:`; a main-frame navigation leaving with a video out turns `_setUsesPageCache:` off until one leaves without, as WebKit restores such a page believing its video still out | none: `TogglePictureInPicture` does nothing, `IsPictureInPictureActive` is false | Chromium's, on whatever the option; `TogglePictureInPicture` runs `requestPictureInPicture()` (or `exitPictureInPicture()`) through the DevTools protocol's `Runtime.evaluate` with `userGesture`, `IsPictureInPictureActive` evaluates `document.pictureInPictureElement` |
| a page's session state (`SessionState`) | `WKWebView`'s `interactionState` (macOS 12+, looked up with `respondsToSelector:`), NSData of WebKit's `_sessionStateData`; restored when the back/forward list then has a current item; WebKit puts a page that scrolled before its window first showed on screen back at the top as the window does, so a page restored before then gets a script in the default client world, as it loads, that puts it back at the private `_scrollPosition` of the current item when it next scrolls to the top before any input | `webkit_web_view_get_session_state` / `_session_state_serialize`, `_session_state_new` / `restore_session_state`, then `go_to_back_forward_list_item` of the current item (2.12+, bound on first use) | none: the core keeps the URL alone and loads it |
| ClearBrowsingData, `Session.ClearData` | the session's `WKWebsiteDataStore`, `allWebsiteDataTypes` or the `WKWebsiteDataType`s of the kinds the system's WebKit exports | the session's web context's website data manager, `WEBKIT_WEBSITE_DATA_ALL` or the kinds' types | the WebView2 profile's `ClearBrowsingDataAll`, or `ClearBrowsingData` of the kinds' `COREWEBVIEW2_BROWSING_DATA_KINDS` (`ICoreWebView2Profile2`, through the web view cookie calls use) |
| sessions (`PageOptions.Session`) | a `WKWebsiteDataStore` set on each web view's configuration: `dataStoreForIdentifier:` of a UUID made of the name (macOS 14+; before, `nonPersistentDataStore` kept until the app quits), `nonPersistentDataStore` for an ephemeral one, released when its last page closes; `window.open` hands the opener's configuration over | a `WebKitWebContext` per session, made with `webkit_web_context_new_with_website_data_manager` of a manager with base data and cache directories in `Sessions/<UUID>` of the user data and cache directories, or `webkit_website_data_manager_new_ephemeral`; web views made with `g_object_new` and `web-context`, which keeps a reference to it; custom schemes, languages and downloads set up on each context; unreferenced when an ephemeral one's last page closes | a WebView2 profile per session (`ICoreWebView2Environment10`'s controller options: `put_ProfileName` `mygo-<UUID>`, or `mygo-private-<n>` with `put_IsInPrivateModeEnabled` for an ephemeral one, a new name for each of its stores); windows, web views in native UI and the cookie calls' hidden web views made with those options |
| HTTP sign-in (`SetAuthHandler`) | `webView:didReceiveAuthenticationChallenge:completionHandler:`, `NSURLCredential` for the session; none: no prompt, the 401 page; empty answer: `RejectProtectionSpace` | `WebKitWebView::authenticate`, `webkit_credential_new` for the session; none: WebKitGTK's dialog; empty answer: authenticates without credentials | `ICoreWebView2_10`'s `BasicAuthenticationRequested`, scheme and realm parsed from its challenge, no proxy flag; none: WebView2's dialog; empty answer: `Cancel` |
| certificate errors (`SetCertificateErrorHandler`) | the server trust challenge of every connection, checked with `SecTrustEvaluateAsyncWithError` while a handler is set, allowed with `credentialForTrust:` for that connection | `load-failed-with-tls-errors`, the main frame only; allowed with `webkit_web_context_allow_tls_certificate_for_host` (app-wide) and loaded again | `ICoreWebView2_14`'s `ServerCertificateErrorDetected`, every request, after `NavigationCompleted` failed; allowed with `ALWAYS_ALLOW` (app-wide) |
| a page's context menu (`OnContextMenu`) | `-[NSView willOpenMenu:withEvent:]` of `MyGoWebView`, on a menu with an item of `forwardContextMenuAction:` or a `WKMenuItemIdentifier*`: items named by their identifier, else WebKit's tag or AppKit's action, emptied and filled again in place; the target from the bridge alone, matched by the event's point; the app's items are `buildItem`'s, sent to `mygoContextItemClicked:` and the menu's window | `WebKitWebView::context-menu`: items named by their stock action, labels from their `GtkAction`, state from their `GAction`; the hit test tells all but the selection; renamed stock items are new ones of the same action; the app's items wrap a `GSimpleAction` (`webkit_context_menu_item_new_from_gaction`) | `ICoreWebView2_11`'s `ContextMenuRequested`, with the whole target (`Complete`); a changed menu is `Handled`, built by `buildMenu` and tracked after a deferral, and a WebView2 item chosen is its `SelectedCommandId`; a runtime without `_11` never asks |
| permissions | `WKUIDelegate` media capture (camera, microphone) | `permission-request` (camera, microphone, geolocation, notifications) | `PermissionRequested` (the same four; WebView2 asks about others) |
| user agent | `customUserAgent`; `UserAgentSuffix` is the configuration's `applicationNameForUserAgent`, which a `window.open()` window does not inherit | `WebKitSettings:user-agent`; `UserAgentSuffix` appends application details, split at the first "/" | `ICoreWebView2Settings2`'s user agent, and a suffix appended to the settings' own user agent; a runtime without `Settings2` ignores both |
| swipes and pinches (`SwipeNavigation`, `PinchZoom`) | `allowsBackForwardNavigationGestures` and `allowsMagnification` of the `WKWebView`, both off unless asked for | `WebKitSettings:enable-back-forward-navigation-gestures`; WebKitGTK has no setting for pinches, so the web view's `GtkGestureZoom`, which it keeps in its data as `wk-view-zoom-gesture` (2.38+), is given `GTK_PHASE_NONE` unless `PinchZoom`: it then sees neither a touchpad's pinch nor a touchscreen's | `ICoreWebView2Settings6`'s `IsSwipeNavigationEnabled` and `ICoreWebView2Settings5`'s `IsPinchZoomEnabled`, both on in WebView2 and put to the options' values in `setUp`, before the first navigation |
| the browser's keys (reload, print, find, zoom) | none | none | WebView2's (F5, Ctrl+R, Ctrl+P, Ctrl+F, Ctrl+Plus, ...) only with the inspector (`ICoreWebView2Settings3`'s `AreBrowserAcceleratorKeysEnabled`); `AcceleratorKeyPressed` still gives the menus every key |
| file associations | `CFBundleDocumentTypes`; files arrive with `application:openURLs:` | desktop entry `MimeType` (`%U`), a shared-mime-info package in the .deb for types the app defines (`x-`) | ProgIDs and `OpenWithProgids` written by the installer |
| document windows | `representedURL` (proxy icon, Command-click folder menu) and `setDocumentEdited:` (the dot), then the traffic lights are laid out again | the core puts `*` before the title | the core puts `*` before the title |
| replacing a document's file | `-[NSFileManager replaceItemAtURL:withItemAtURL:…]` (permissions, dates, extended attributes, ACLs, identity) | rename after `chmod` and `lchown` to the target's | `ReplaceFileW`, retried on sharing violations; when it stops halfway, having removed the target, the replacement is renamed into place |
| recent documents | `noteNewRecentDocumentURL:` / `clearRecentDocuments:` of the shared `NSDocumentController` (the Dock menu) | `gtk_recent_manager_add_item`, removing the items of `g_get_application_name()` | `SHAddToRecentDocs` (`SHARD_PATHW`), `IApplicationDestinations::RemoveAllDestinations`; the process keeps the AppUserModelID the shell derives from its path, which the installer's shortcuts have too |
| jump list | ignored | ignored | `ICustomDestinationList`, committed whole from what `App.JumpList` keeps: the Recent category, `AddUserTasks` with an `IShellLinkW` per task (`PKEY_Title`, description, working directory, arguments quoted with `syscall.EscapeArg`), `CommitList` |
| Dock menu | `applicationDockMenu:` | ignored | ignored |
| PrintToPDF | `printOperationWithPrintInfo:` save job (`NSJobSavingURL`), fit to width; the core rewrites the radicals the text names to the ideographs the fonts draw (`pdf_text.go`) | `WebKitPrintOperation` to GTK's "Print to File" | DevTools `Page.printToPDF` |
| power events | NSWorkspace sleep/wake, `com.apple.screenIsLocked` distributed notifications | logind `PrepareForSleep` (system bus), screen saver `ActiveChanged` (GNOME, freedesktop) | `WM_POWERBROADCAST`, `WM_WTSSESSION_CHANGE` |
| Now Playing and media keys | `MPNowPlayingInfoCenter` (its `nowPlayingInfo` and `playbackState`; artwork as an `MPMediaItemArtwork` decoding the bytes it holds when asked) and the targets of `MPRemoteCommandCenter`'s commands, enabled as the info offers them; MediaPlayer.framework loads with the first `Set` | an MPRIS player (`org.mpris.MediaPlayer2` and `.Player` at `/org/mpris/MediaPlayer2`, exported with `g_dbus_connection_register_object`), whose bus name `org.mpris.MediaPlayer2.<app>.instance<pid>` is owned only while it shows something; `PropertiesChanged` on each change, `Seeked` when the position jumps; the artwork goes to a file in `$XDG_RUNTIME_DIR` for `mpris:artUrl` | the `SystemMediaTransportControls` of the application window (`ISystemMediaTransportControlsInterop::GetForWindow`), with agile event handlers, a `RandomAccessStreamReference` over a memory stream for the thumbnail, and the timeline properties for position and seeking |
| KeepAwake | `NSProcessInfo` activity (shows in `pmset -g assertions`) | XDG portal `Inhibit`, else `org.freedesktop.ScreenSaver.Inhibit` | `PowerCreateRequest` |
| IsOnBattery, IdleTime | IOKit power sources, `CGEventSourceSecondsSinceLastEventType` | `/sys/class/power_supply`; Mutter idle monitor or `GetSessionIdleTime` | `GetSystemPowerStatus`, `GetLastInputInfo` |
| network status | `nw_path_monitor` on a queue of its own, the first `Status` waiting (up to a second) for its first path; online when the path is satisfied, metered when it is expensive or constrained | `GNetworkMonitor`, `network-available` (a default route) and `network-metered` | `GetNetworkConnectivityHint`, `NotifyNetworkConnectivityHintChange`; online from local access up (not none or hidden), metered for a fixed or variable cost, roaming, over or near the data limit |
| window position | honored | ignored by Wayland compositors | honored |
| resize borders without a title bar | the window's own | the outer 5 px of the page | invisible, outside the window; along the top of a frameless window or a hidden title bar, a child window |
| content protection, click-through | yes | ignored | yes |
| panels | `NSPanel` with the nonactivating style, floating, on all Spaces as a full-screen auxiliary | sticky utility window, kept above, out of the taskbar and the pager; a gtk-layer-shell overlay surface on Wayland where it can be | topmost `WS_EX_TOOLWINDOW` owned by the hidden application window, rounded |
| flyouts | borderless nonactivating `NSPanel` child window, or an `NSPopover`; placed in the visible frame | `GTK_WINDOW_POPUP` (override-redirect, placed in the work area, on X11; `xdg_popup` with `gdk_window_move_to_rect` on Wayland); focusable ones grab the seat | owned `WS_POPUP` + `WS_EX_TOOLWINDOW`, `WS_EX_NOACTIVATE` unless focusable, `CS_DROPSHADOW` for a shadow, with corners rounded by DWM on Windows 11 |
| custom scheme origin | `<scheme>://<host>` | `<scheme>://<host>` | `http://<scheme>.localhost`, or `http://<host>.<scheme>.localhost` for another host (the page's `location`) |
| `LoadHTML` documents | a reload loads the base URL (`about:blank` without one), so the core cancels it (`Navigation.Reload`) and loads the HTML again; not in the history | as macOS | as macOS, but in the history: going back to one with a base URL loads that URL |
| window.open | keeps the opener | independent window | keeps the opener: the new window's web view, set up, becomes the popup (`put_NewWindow`, under a deferral) |
| keys held (`Context.KeyDown`) | `keyDown:` and `keyUp:`; a local monitor of key ups hands the surface those let go of while Command is held, which `sendEvent:` keeps from the key window; resigning first responder or key window lets go of all | key presses and releases, and a release the input method filters (`HeldKeys.Lift`): IBus and Fcitx hand back those they leave, which `HeldKeys` keeps from the content | `WM_KEYUP` and `WM_SYSKEYUP`; a `VK_PROCESSKEY` release lets go of the key at its scancode; `WM_ENTERMENULOOP` lets go of all (`HeldKeys.LiftAll`), whose releases the menu's loop takes; `WM_KILLFOCUS` all |
| native UI surface | layer-backed NSView, frames from `CADisplayLink` (a timer at the display's rate before macOS 14), input methods through NSTextInputClient | GtkGLArea (GtkDrawingArea without a GPU), GtkIMMulticontext | `MyGoSurface` child window, IMM32 |
| native UI data drags | NSDraggingSession, NSPasteboardItemDataProvider, NSDraggingDestination | GTK drag contexts, MIME selections, text/uri-list | OLE IDataObject, IDropSource, IDropTarget, Shell drag images |
| native UI accessibility | `NSAccessibilityElement` subclasses | ATK objects (GObject types registered through purego), bridged to AT-SPI by GTK | UI Automation fragments (COM objects; assembly thunks for the methods taking doubles) |
| native UI rendering | Metal, into a CAMetalLayer presenting as soon as drawn, or with the Core Animation transaction for frames that resize it or show web views | OpenGL 3.3 or ES 3.0 in the GtkGLArea's render signal; on the CPU, painted with cairo, where OpenGL runs on the CPU | Direct3D 11 (WARP without a GPU), flip-model swap chain presenting dirty rectangles |
| web views in native UI | WKWebView in a clip view under the surface; `hitTest:` lets the pointer through | WebKitWebView in a `GtkLayout` under the area in a `GtkOverlay`; an input shape lets the pointer through | WebView2 composition controller in a DirectComposition target under the frames'; the surface sends it the mouse |
| audio output | Audio Queue on the default output | PulseAudio simple API (PipeWire too), else ALSA `default` | WASAPI shared mode, reopened when the default output changes |
| gamepads | GameController.framework, extended gamepad profile | evdev, the kernel's gamepad layout, Xbox's X and Y swapped back | XInput, four slots, Guide by `XInputGetStateEx` |
