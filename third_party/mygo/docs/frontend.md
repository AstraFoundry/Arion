# The frontend

Windows that show web pages show the pages of the app's frontend: HTML,
CSS and JavaScript built with any tools. New projects use TypeScript and
[Vite](https://vite.dev), but MyGo only needs a URL to load, during
development, and a directory of built files, for builds. Windows of
[native UI](ui/README.md) need no frontend, and an app that only has those has
none.

## How pages load

Windows load pages of the frontend with URLs without a scheme, such as `/`
or `/settings?tab=general`:

```go
mygo.NewWindow(mygo.WindowOptions{URL: "/"})
win.Page().LoadURL("/settings")
```

They resolve against:

- during `mygo dev`, the dev server at `devUrl` in mygo.config.ts, which
  `devCommand` starts, e.g. `http://localhost:5173/`, so the dev server's
  hot reload works in the app;
- in builds, `mygo://localhost/`, which serves the files of `frontendDist`
  that `mygo build` embeds into the executable after running
  `buildCommand`.

```ts
export default defineConfig({
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
});
```

Without `devUrl`, `mygo dev` serves `frontendDist` from disk instead. Paths
without a file extension that match no file serve `index.html`, so
client-side routers work.

The frontend is a secure context, and its origin is `mygo://localhost` on
macOS and Linux and `http://mygo.localhost` on Windows, where WebView2
serves custom schemes under `http://<scheme>.localhost`. Storage such as
`localStorage` and IndexedDB belongs to an origin, so data a page stores
under `mygo dev` (the dev server's origin) is not what the built app sees.
Keep data you care about in Go, for example in a file in
`App.Path(mygo.PathUserData)`, and hand it to pages through bound methods.

Windows can also load other content into their page:

- `win.Page().LoadURL("https://example.com")`, a web page;
- `win.Page().LoadHTML(html, baseURL)`, an HTML string, which reloads load
  again;
- `win.Page().LoadFile("docs/index.html")`, a local file, resolved against the
  working directory, then the executable's directory (and the Resources
  directory of a macOS app).

Pages that are not the app's own cannot call Go methods: see
[who may call](bindings.md#who-may-call).

### Without the CLI

`mygo build` embeds the frontend by calling `mygo.SetFrontend`, and a
program built with `go build` can do the same with `embed`:

```go
//go:embed all:dist
var dist embed.FS

func main() {
	web, _ := fs.Sub(dist, "dist")
	mygo.SetFrontend(web)
	// ...
}
```

## Custom protocols

`mygo.Protocol.Handle` serves a URL scheme from an `http.Handler`, for
content your app produces or reads at run time: pages, images, files of
the user. Handlers run on their own goroutines and stream their responses:
writes wait while the webview is behind, so a large file is never held in
memory whole. Pages load such URLs like web URLs, with `fetch`, `<img>`,
ES modules and relative URLs:

```go
// Thumbnails generated into the cache directory.
cache, _ := mygo.App.Path(mygo.PathCache)
mygo.Protocol.Handle("thumbs", mygo.FileServer(os.DirFS(cache)))

// Anything an http.Handler does.
mygo.Protocol.HandleFunc("api", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path})
})
```

```html
<img src="thumbs://localhost/photo-42.png" />
```

Register schemes before creating the windows that use them.
`mygo.FileServer` serves an `fs.FS` with content types and range requests,
falling back to `index.html` like the frontend. Handling the `mygo` scheme
replaces the frontend with your handler. Pages of custom schemes are the
app's own: they may call Go methods, see [trust](#trust).

Audio and video are the exception: WebKitGTK's media player does not play
from a custom scheme, so on Linux a `<video>` or `<audio>` of one fails with
`MEDIA_ERR_SRC_NOT_SUPPORTED` (`error.code` 4), whatever the handler
answers. Serve media from a loopback HTTP server instead, the same on every
platform, and give the page its URL:

```go
ln, err := net.Listen("tcp", "127.0.0.1:0") // a free port
if err != nil {
	log.Fatal(err)
}
go http.Serve(ln, mygo.FileServer(os.DirFS(videos)))
mediaURL := "http://" + ln.Addr().String() + "/" // <video src="{mediaURL}clip.mp4">
```

Every program of the machine can reach such a server: serve it only what
they may read, or put a secret of the app's in its paths.

Each scheme is its own origin, so a page fetching another scheme makes a
cross-origin request: the frontend reading `api://localhost/`, or, during
`mygo dev`, the dev server reading `thumbs://localhost/`. Responses to the
app's own pages, those that may call Go methods, carry the CORS headers
that let them read the response, and MyGo answers their preflights.
Other sites get none, unless the window trusts them
(`PageOptions.TrustedOrigins`), and handlers may set their own.
On Windows, fetch a scheme at `http://<scheme>.localhost/`, where
WebView2 serves it, and another host at `http://<host>.<scheme>.localhost/`
(`myapp://abc/` is `http://abc.myapp.localhost/`). Such hosts are DNS
names (letters, digits, hyphens and dots) and are reported in lower case;
URLs with another host or a port aren't served there.

### Trust

A window trusts the app's own pages: its frontend, pages of custom schemes,
`file:` and `about:` pages (`LoadHTML` without a base URL shows
`about:blank`), origins in `PageOptions.TrustedOrigins`, and in development
builds pages served from localhost. Trust follows the page's URL, in a
window or a [web view](ui/web-view.md), and changes as it navigates. A
trusted page may:

- call every method bound with `mygo.Bind`, and the plugins the app uses;
- call `window.mygo.window` to set the title of its window, minimize,
  maximize, full-screen or close it;
- get the camera, the microphone, the location and notifications without a
  permission handler;
- read the responses of every custom scheme, which carry CORS headers for
  it;
- hear the paths of the files dropped on it, and of its window's document.

So never serve third-party or user-supplied HTML from a custom scheme, nor
load it with `LoadHTML` without a base URL, in a page that trusts it: a
plugin's UI or a preview of a user's HTML would get all of the above.
Show it in an untrusted page instead, or from an origin the window does
not trust, such as an `https` site or a `data:` URL: it calls no Go
method, and gets no permission unless a
[permission handler](windows.md#permissions) grants it.

#### Untrusted pages

`PageOptions.Untrusted` makes a page someone else's whatever its URL, for
a window or a [web view](ui/web-view.md) showing a plugin's UI from a
custom scheme. It trusts no origin, `TrustedOrigins` included, and goes
further than a page of an untrusted origin: what MyGo lets every page do
by default, it lets an untrusted page do only when a handler of the
app's decides.

- Its calls, to bound methods and to `window.mygo.window`, are rejected,
  and its drag regions do nothing. MyGo drops any message it posts above
  4 KiB before decoding it, and answers the rest on the main thread,
  without a goroutine per message.
- `Broadcast` does not reach it, nor `Emit` to the window its web view is
  in. Send to it with `EmitPage`.
- It gets no permission without `SetPermissionHandler`, opens no window
  nor the browser without `SetWindowOpenHandler`, downloads nothing
  without `OnWillDownload` listeners, and goes to no other origin than
  its page's without `OnWillNavigate` listeners.

It still reads its own scheme, as any page does, and its requests reach
the handler, which tells which page asks with `mygo.CallerPage(r.Context())`.
That's how a plugin's UI talks to the app, and `EmitPage` how the app
answers:

```go
mygo.Protocol.HandleFunc("plugin", func(w http.ResponseWriter, r *http.Request) {
	p := mygo.CallerPage(r.Context())
	plugin := pluginOf[p] // the plugin whose web view made the request
	if plugin == nil || r.URL.Host != plugin.ID {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		msg, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
		...
		PluginReply.EmitPage(p, reply)
		return
	}
	http.FileServerFS(plugin.Files).ServeHTTP(w, r)
})

view, _ := win.NewWebView(mygo.WebViewOptions{
	URL:  "plugin://" + plugin.ID + "/",
	Page: mygo.PageOptions{Untrusted: true},
})
pluginOf[view.Page()] = plugin
```

```js
// In the plugin's page.
await fetch("/message", { method: "POST", body: JSON.stringify(msg) });
window.mygo.on("plugin-reply", (reply) => { ... });
```

A page can make requests to other hosts and schemes too, without reading
their responses: check the page, as above, before a handler acts on a
request.

Some things reach every page, trusted or not:

- events: `Broadcast` sends to every page of every window, and `Emit` to
  every web view of a window of native UI. Send to one page with
  `EmitPage`.
- messages to Go: an untrusted page's calls are rejected, but the app
  receives and decodes them first, whatever their size, unless the page is
  [untrusted](#untrusted-pages).
- `window.open` and `target="_blank"` links open `http(s)` URLs in the
  default browser, unless `SetWindowOpenHandler` decides
  ([links and new windows](#links-and-new-windows)).
- downloads go to the Downloads directory, unless `OnWillDownload` decides
  ([downloads](windows.md#downloads)).
- the page may navigate anywhere, unless `OnWillNavigate` cancels it
  ([pages](windows.md#pages)). A page that navigates to a trusted URL is
  trusted.

## The mygo-runtime package

MyGo injects its runtime into every page as `window.mygo`. The
`mygo-runtime` npm package, which new projects depend on, gives it types
and imports; the generated client is built on it.

```ts
import { currentWindow, isMyGo, onFileDrop, runtime } from "mygo-runtime";

if (isMyGo() && runtime().platform === "darwin") {
  document.documentElement.classList.add("mac");
}

await currentWindow.toggleMaximize();
```

| Export | What it does |
|---|---|
| `call(method, ...args)`, `on(event, listener)`, `once(event, listener)`, `event<T>(name)` | untyped calls and events, see [bindings](bindings.md#without-the-generated-client) |
| `Channel` | a stream of values from a Go method, see [channels](bindings.md#channels) |
| `isCallError(err)` | whether a rejection came from a Go error |
| `isMyGo()` | whether the page runs in a MyGo window, not in a browser tab of the dev server |
| `runtime()` | the runtime: `platform` (`"darwin"`, `"linux"` or `"win32"`), `windowId` (the Go window's `ID()`) and `version`; throws outside MyGo |
| `currentWindow` | the page's window: `minimize`, `maximize`, `unmaximize`, `toggleMaximize`, `isMaximized`, `toggleFullScreen`, `close` and `setTitle` |
| `onFileDrop(listener)` | files dropped on the window, with their paths |
| `preferences()` | the accent color and the accessibility settings of the desktop; throws outside MyGo |
| `onPreferences(listener)` | the changes of `preferences()` |
| `windowActive()`, `onWindowActive(listener)` | whether the page's window is drawn as active, the one the user's input goes to, and its changes, see [inactive windows](#inactive-windows) |
| `locale()`, `direction()`, `onLocaleChange(listener)` | the language the app shows (`App.Locale`), such as `"zh-Hans"`, its direction, `"ltr"` or `"rtl"`, and its changes; the page has them from document start, as `mygo.locale` and `mygo.dir` |

`mygo-runtime/i18n` formats messages of ICU MessageFormat, as package
`i18n` does in Go. The module `mygo generate` writes for the app's
[catalogs](i18n.md#in-pages), `locales.ts`, is built on it, and holds the
messages of every language, which the frontend's build bundles: a page
says its first words in the app's language without waiting for a request.

Opened in a regular browser, which is handy for working on the layout,
pages have no runtime: calls reject and `isMyGo()` is false.

## Custom title bars

A frameless window has no title bar or borders, and the page draws its own.
Mark the elements that drag the window with the CSS property
`--app-region: drag`:

```go
mygo.NewWindow(mygo.WindowOptions{URL: "/", Frameless: true})
```

```css
.titlebar {
  --app-region: drag;
  height: 40px;
}
.titlebar .search {
  --app-region: no-drag;
}
```

```ts
import { currentWindow } from "mygo-runtime";

minimizeButton.onclick = () => currentWindow.minimize();
maximizeButton.onclick = () => currentWindow.toggleMaximize();
closeButton.onclick = () => currentWindow.close();
```

The property is inherited: `no-drag` excludes parts of a region again, and
buttons, links, inputs and other interactive elements inside a region stay
clickable. Double-clicking a region does what double-clicking a title bar
does.

Frameless windows still resize from their edges. On Linux under Wayland,
GTK decorates frameless windows and windows with a hidden title bar itself,
as GNOME's apps, without a title bar: it draws a shadow and a thin border
around the page, which resize the window from just outside it, and drops
them while the window is maximized or in full screen. Elsewhere on Linux,
where GTK gives windows without decorations no borders (X11, or a window
with `DisableShadow`, `Transparent` or `Panel`), the outermost 5 px of the
page act as the borders: they show resize cursors, and the page gets no
clicks or pointer moves there.

On Linux a frameless window also keeps the menu bar that `App.SetMenu` or
`Window.SetMenu` gives it, above the page and so above a custom title bar.
Give such a window an empty menu of its own,
`win.SetMenu(mygo.NewMenu(nil))`, and show menus from the page instead
(`Menu.PopupAt`, from a bound method the page calls).

### Hidden title bars

A window can keep its window controls and only hide the title bar, so the
page extends under them and draws the rest of the title bar: set
`TitleBarStyle` to `mygo.TitleBarHidden`. The controls stay at the top of
the window:

- On macOS, the traffic lights. `mygo.TitleBarHiddenInset` leaves more
  room around them, and `TrafficLightPosition` moves them: the position of
  the close button's top-left corner in the window.
- On Windows, minimize, maximize or restore, and close at the top-right
  corner, drawn as Windows 11 draws its own. They fill the height of the
  title bar, `TitleBarHeight` (32 by default), and hovering maximize opens
  the snap layouts. The window resizes from its top edge.
- On Linux, GTK's own title buttons, where the desktop's button layout puts
  them, centered in `TitleBarHeight` (by default the height of the
  desktop's header bars). The layout decides which buttons show, possibly
  none, as tiling window managers are often set up. A Wayland compositor
  that decorates windows itself, as KWin, Hyprland and Sway do, has none:
  its own title bar holds the buttons, and a tiling one shows none. The
  outer pixels of the page resize the window, as a frameless one's. The
  window's menu bar hides, as with `AutoHideMenuBar`: Alt and F10 show it
  above the page while its menus are open, and its shortcuts work all
  along. `Theme.SetGTKCSS` restyles the buttons, as GTK 4's round ones,
  for example ([GTK's style](windows.md#gtks-style-on-linux)).

```go
mygo.NewWindow(mygo.WindowOptions{
	URL:            "/",
	TitleBarStyle:  mygo.TitleBarHidden,
	TitleBarHeight: 40,
})
```

From its first paint, the page learns the room the controls take from
CSS variables on `:root`:

| variable | |
|---|---|
| `--mygo-titlebar-height` | the height of the title bar |
| `--mygo-titlebar-inset-left`, `--mygo-titlebar-inset-right` | the room the controls take from the left and from the right edge |

They are `0px` in full screen, where the controls hide (on macOS they come
back with the menu bar). The page's title bar keeps clear of the controls
and, marked with `--app-region: drag`, moves the window:

```css
.titlebar {
  --app-region: drag;
  height: var(--mygo-titlebar-height, 40px);
  padding-left: var(--mygo-titlebar-inset-left, 0px);
  padding-right: var(--mygo-titlebar-inset-right, 0px);
}
```

`Transparent: true` lets a page with a transparent background show the
desktop through, and `Vibrancy` puts a blurred material behind it on macOS
and Windows 11 22H2 and later:

```go
mygo.NewWindow(mygo.WindowOptions{
	URL:           "/",
	TitleBarStyle: mygo.TitleBarHidden,
	Transparent:   true,
	Vibrancy:      mygo.VibrancySidebar, // Mica on Windows 11
})
```

See `examples/frameless` for a complete custom title bar. Native UI shows
materials the same way ([Vibrancy](ui/windows.md#vibrancy)), as
`examples/vibrancy` does in a translucent sidebar under an inset title bar.

## Dropped files

Files dragged from Finder or Explorer and dropped on the page reach the
page with their paths, which DOM drop events do not tell:

```ts
import { onFileDrop } from "mygo-runtime";

onFileDrop(({ paths, x, y }) => {
  const target = document.elementFromPoint(x, y);
  // ...
});
```

and Go, with `win.OnFileDrop`. The page's own drag and drop keeps working:
its drop listeners still get the `File` objects, and dropping files where
the page does not handle them no longer replaces the page with the file.

## Dark mode

Pages follow the system appearance through the `prefers-color-scheme` media
query. Declare `color-scheme: light dark` in CSS so that form controls and
scrollbars follow too, and give windows a `BackgroundColor` matching the
page in both appearances, which the window shows until the page paints:

```css
:root {
  color-scheme: light dark;
}
```

```go
mygo.NewWindow(mygo.WindowOptions{URL: "/", BackgroundColor: "light-dark(#f5f5f7, #1e1e1e)"})
```

`mygo.Theme.SetSource(mygo.ThemeDark)` forces an appearance, which pages see
through the same media query.

## Accent color and accessibility

`prefers-reduced-motion` and `prefers-contrast` follow the desktop on their
own. The accent color the desktop uses and the factor it asks text to grow
by are what the media queries do not tell a page: they are CSS variables on
`:root` from the first paint.

| variable | |
|---|---|
| `--mygo-accent` | the accent as `#rrggbb`; unset where the desktop has none, so `var(--mygo-accent, AccentColor)` falls back to the desktop's own color |
| `--mygo-text-scale` | the factor text grows by; `1` is the usual size, so `calc(1rem * var(--mygo-text-scale, 1))` |

```css
:root {
  --accent: var(--mygo-accent, AccentColor);
  font-size: calc(1rem * var(--mygo-text-scale, 1));
}
```

`preferences()` of `mygo-runtime` returns the settings themselves, current
from document start on, and `onPreferences` tells their changes, which only
the desktop makes; a component reads the first and subscribes to the
second, as with `matchMedia`:

```ts
import { onPreferences, preferences, type ThemePreferences } from "mygo-runtime";

const apply = ({ reduceTransparency }: ThemePreferences) =>
  document.documentElement.classList.toggle("opaque", reduceTransparency);
apply(preferences());
const off = onPreferences(apply);
```

| field | |
|---|---|
| `accent` | the accent as `"#rrggbb"`, or `null` where the desktop has none |
| `reduceMotion` | the desktop asks for less motion, as `prefers-reduced-motion` |
| `reduceTransparency` | the desktop asks for opaque surfaces: macOS's Reduce Transparency, Windows's transparency effects turned off; false on Linux, which has no such setting |
| `highContrast` | the desktop asks for more contrast, as `prefers-contrast: more` |
| `textScale` | how many times larger than usual text should be; `1` is usual |

A page that navigates starts with the values of the moment: the script
that tells them at document start is replaced when they change.
`mygo.Theme.Preferences()` reads them on the Go side, see
[native](native.md#accent-color-and-accessibility).

## Inactive windows

Native apps draw the window the user's input goes to apart from the
others: macOS fades an inactive window's sidebar and the accent of its
selections to gray, Windows draws its Mica as a solid fill, GTK styles it
`:backdrop`. A page knows which its window is from `<html>`, which has
the `data-mygo-window-inactive` attribute while the window is not active:

```css
.sidebar .selected {
  background: var(--accent);
  color: white;
}
:root[data-mygo-window-inactive] .sidebar .selected {
  background: #0000001a;
  color: inherit;
}
```

A window is active while it, or one of its flyouts, has the keyboard: a
flyout of the window leaves it active, where DOM focus would leave the
page. Panels, flyouts without a
parent and windows `SetActiveAppearance` keeps active always are. A page
starts active and is told otherwise as its DOM is ready; `windowActive()`
of `mygo-runtime` reads the state and `onWindowActive` tells its changes.
The material behind a transparent page follows on its own (macOS's in
windows other than panels and flyouts, Windows's Mica). See
[native UI](ui/windows.md#active-and-inactive-windows) for windows that
show it.

## Pages that size their window

The page of a flyout or a window with `FitContent` sizes it, without a
`ResizeObserver` or a bound method of its own: MyGo's runtime lays `html`
out at the width of its content, within the window's limits, and the
window takes the root's size as it changes (see
[windows](windows.md#size-a-flyout-to-its-content)). Let `html` and
`body` take their content's size: a `height: 100%` or `100vh` on them
keeps the window at the size it has, and a `width` on `html` replaces the
content's. Text wider than the maximum wraps, and content taller than it
scrolls.

## Links and new windows

Links with `target="_blank"` and `window.open()` call the window's open
handler. Without one, `http(s)` URLs open in the default browser and
anything else is denied; `win.Page().SetWindowOpenHandler` decides otherwise:

```go
win.Page().SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
	if strings.HasPrefix(req.URL, "https://docs.example.com/") {
		return &mygo.WindowOptions{Width: 900, Height: 700} // a window of the app
	}
	go mygo.Shell.OpenExternal(req.URL) // the default browser
	return nil
})
```

`req.UserInitiated` tells the windows a user opened, with a click on the link
or a click or key press whose handler calls `window.open()`, from the ones a
script opened on its own, as a pop-up from a timer or as the page loads.
Refuse those, as ads in an embedded page:

```go
win.Page().SetWindowOpenHandler(func(req mygo.WindowOpenRequest) *mygo.WindowOptions {
	if !req.UserInitiated {
		return nil // a pop-up
	}
	go mygo.Shell.OpenExternal(req.URL)
	return nil
})
```

On Linux, WebKitGTK blocks a `window.open()` without a user's gesture itself,
so the handler is asked only about the ones a gesture made. A script the app
runs itself with `Page.Eval` counts as a user's gesture on macOS and Linux,
where WebKit runs it as one.

`win.Page().OnWillNavigate` can cancel the page's own navigations, e.g. to keep the
app's window on the app: see [Windows](windows.md#pages).

## Preload scripts

`PageOptions.PreloadScript` runs JavaScript in every page of the window
before the page's own scripts, once `window.mygo` exists.

It runs in the top frame only. `PreloadAllFrames` runs it in the page's
iframes too, of every origin, as each starts, which a browser needs to see
what a player embedded from another site loads. `window.mygo` isn't in
iframes: to tell the page what an iframe found, post it to `window.top`.

## The web inspector

Development builds have the web inspector: right-click a page and choose
Inspect Element (Inspect on Windows), or call `win.Page().OpenDevTools()`.
Production builds do not, unless built with `mygo build -debug` or a window's
page sets `DevTools: mygo.DevToolsEnabled` in `WindowOptions.Page`.

On Windows, an app takes `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` and
`WEBVIEW2_USER_DATA_FOLDER` as other WebView2 apps do. With
`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9222`,
another process can drive its pages over the DevTools protocol, as a test
runner, Playwright's `connectOverCDP` or an agent does. This works in every
build, as with Microsoft's loader: whoever starts the app sets its
environment.

On Windows, WebView2's own keys come with the inspector: F5 and Ctrl+R reload
the page (a reload [`OnWillNavigate`](windows.md#pages) is not asked about),
Ctrl+P prints, Ctrl+F finds and Ctrl+Plus zooms only where the inspector is
on. Without it, as in production builds, a page has none of them, as on macOS
and Linux, and only the app's menu items take these keys, such as the View
menu's (`RoleViewMenu`).
