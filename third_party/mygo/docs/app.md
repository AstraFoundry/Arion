# The application

`mygo.App` is the application: its lifecycle, its identity, and how it
meets the rest of the system.

## The main function

```go
func main() {
	mygo.Bind(&Notes{})                // services the frontend calls
	mygo.App.WhenReady(func() {        // once the app has launched
		mygo.NewWindow(mygo.WindowOptions{URL: "/"})
	})
	if err := mygo.App.Run(); err != nil { // the event loop, until the app quits
		log.Fatal(err)
	}
}
```

`App.Run` runs the native event loop and returns when the app quits. It
must be called from the main goroutine, because the UI toolkits of every
platform must run on the thread that started the process; MyGo keeps the
main goroutine on it. Windows exist only once the app is ready: create them
in `WhenReady`, or from any goroutine after that.

Before `Run`, `main` sets the app up: it binds services, handles protocols,
adds listeners, and makes settings such as `App.SetName`, `App.SetMenu`,
`App.Dock.SetMenu`, `Theme.SetSource` and `Theme.SetGTKCSS`, which take
effect when the app starts. It may also read what needs no running app, such as `App.Path`,
`App.Locale` or `Power.IsOnBattery`, and register the app with the system
(`App.SetOpenAtLogin`, `App.RegisterURLScheme`). Everything else needs the
running app: when `main` calls the clipboard, displays, dialogs,
notifications, global shortcuts, trays, Dock badges, `Theme.IsDark`,
`Theme.Preferences` and the like before `Run`, they panic with a message
naming the call, on every platform. Use them from `WhenReady` on; other
goroutines may call them earlier, and wait for the app to start.

## Threads

Every function and method of MyGo is safe to call from any goroutine: calls
made off the main thread are sent to it and wait for the result. What runs
where:

- Event listeners, such as `OnClose` or `OnActivate`, menu item clicks and
  global shortcuts run on the main thread, which also draws the UI: keep
  them short, and start goroutines for slow work.
- Bound methods run on goroutines of their own, one per call: they may
  block.
- `mygo.RunOnMain(fn)` runs `fn` on the main thread, for native APIs called
  through `Window.NativeHandle`.

## Lifecycle

| Event | When |
|---|---|
| `OnReady` | the app has launched; `WhenReady` also runs its function right away when the app is already ready |
| `OnActivate(func(hasVisibleWindows bool))` | the app was activated again, e.g. by a click on its Dock icon (macOS) |
| `OnDidBecomeActive`, `OnDidResignActive` | the app became, or stopped being, the active app |
| `OnWindowCreated(func(w *mygo.Window))` | a window was created |
| `OnWindowAllClosed` | the last window closed, unless the app is quitting |
| `OnBeforeQuit(func(e *mygo.QuitEvent))` | a quit started, before the windows close; `e.PreventDefault()` cancels it |
| `OnWillQuit(func(e *mygo.QuitEvent))` | every window has closed and the app is about to quit; it can still be canceled |
| `OnQuit` | the event loop stopped, right before `Run` returns |

Each returns a function that removes the listener.

### Quitting

When its last window closes, an app quits, unless `OnWindowAllClosed` has a
listener. On macOS apps usually keep running with their Dock icon, and open
a window again when it is clicked:

```go
mygo.App.OnWindowAllClosed(func() {
	if runtime.GOOS != "darwin" {
		mygo.App.Quit()
	}
})
mygo.App.OnActivate(func(hasVisibleWindows bool) {
	if !hasVisibleWindows {
		openMainWindow()
	}
})
```

- `App.Quit()` closes every window, which `OnClose` listeners can refuse,
  then quits; `OnBeforeQuit` and `OnWillQuit` listeners can cancel it too.
  Cmd+Q, the Quit menu item and Ctrl+C in the terminal (SIGINT and SIGTERM)
  quit the same way; a second signal exits at once.
- `App.Exit(code)` exits right away, without events.
- `App.Relaunch()` quits, then starts the app again with the same
  arguments, for example after a setting that needs a restart.

## A single instance

Most desktop apps run once: starting them again brings the running
instance to the front. `RequestSingleInstanceLock` returns false in a
second instance, after passing its command line to the first:

```go
func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return // the running instance takes over
	}
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if w := mainWindow(); w != nil {
			w.Restore()
			w.Focus()
		}
	})
	// ...
}
```

## Deep links

Apps open URLs of their own schemes, such as `myapp://open?note=42`. List
the schemes in mygo.config.ts, which registers them with the system when the
app is installed:

```ts
export default defineConfig({
  urlSchemes: ["myapp"],
});
```

and handle the URLs with `OnOpenURL`, registered before `Run` so it also
gets the URL the app was started with:

```go
mygo.App.OnOpenURL(func(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	openNote(u.Query().Get("note"))
})
```

On Windows and Linux a URL starts a new instance of the app: request the
[single instance lock](#a-single-instance), and the URLs of later instances
reach `OnOpenURL` in the first one. Apps that are not installed with the
package `mygo build` makes, a portable executable for example, register
their schemes at run time with `App.RegisterURLScheme("myapp")`, which is
cheap to call at every launch; `UnregisterURLScheme` and
`IsURLSchemeRegistered` go with it. On macOS schemes must be listed in
the configuration.

## File associations

Apps open files of their types from the file manager. Declare the types in
mygo.config.ts:

```ts
export default defineConfig({
  fileAssociations: [
    {
      ext: ["md", "markdown"],
      name: "Markdown Document",
      mimeType: "text/markdown",
    },
  ],
});
```

and open them with `OnOpenFile`, registered before `Run` so it also gets
the files the app was started with:

```go
mygo.App.OnOpenFile(func(path string) {
	openDocument(path)
})
```

It also gets files dropped on the app's Dock icon (macOS) and files passed
on the command line. As with deep links, the single instance lock makes the
first instance get the files of later ones on Windows and Linux.

Apps whose windows each show a file get this and the rest of what
document-based apps do from [documents](documents.md).

## Recent documents

Record the documents the user opens, so that the app and the system offer
them again:

```go
if err := mygo.App.AddRecentDocument(path); err != nil {
	log.Println(err)
}
```

`App.RecentDocuments()` returns them, newest first, ten at most, also
those of earlier runs, without the files that no longer exist; the Open
Recent menu (`RoleOpenRecent`, see [menus](menus.md)) lists them. macOS
lists them in the app's Dock menu too, Linux in the user's recently used
files (`~/.local/share/recently-used.xbel`, the Recent of file choosers and
file managers), and Windows in the Recent category of the app's
[jump list](#jump-lists), for the file types the app is registered to open
(see [file associations](#file-associations)). Choosing one of them, in the
Open Recent menu, the Dock menu or the jump list, opens the file with the
app, which gets it through `OnOpenFile`. [Documents](documents.md) record
the files they open and save, and open those chosen themselves; other
apps, such as those that open folders, record what they open.

`ClearRecentDocuments` removes them all, for example when the user asks to
clear the app's history, as the Open Recent menu's Clear Menu does. On
Linux the user's other recent files stay.

## Start at login

```go
if err := mygo.App.SetOpenAtLogin(true); err != nil {
	log.Println(err)
}
enabled := mygo.App.OpenAtLogin()
```

Offer it as a setting. The app becomes a login item of the user on macOS
13 and later (which needs the packaged app), a launch agent on macOS 12, an
XDG autostart entry on Linux and a Run entry of the user on Windows, which
the system's settings list and can turn off. `WasOpenedAtLogin` tells a
launch at login apart, to start in the background:

```go
mygo.NewWindow(mygo.WindowOptions{URL: "/", Hidden: mygo.App.WasOpenedAtLogin()})
```

On Windows the Run entry starts the app with the user's normal
(unelevated) token, even when the user is an administrator: the app
cannot create network adapters, change the system's proxy or write to
protected folders. An app that needs administrator rights registers a
logon task in Task Scheduler with the run level `HighestAvailable`
instead (`schtasks /Create /SC ONLOGON /RL HIGHEST`, or the Task
Scheduler API), which starts it elevated without asking, and does not
call `SetOpenAtLogin`. Creating the task needs administrator rights
itself, so the app asks for them once, as the user turns the setting on.
`WasOpenedAtLogin` does not tell such a launch apart: give the task's
command an argument of the app's own for that.

## Directories

`App.Path` returns well-known directories:

| Name | Directory |
|---|---|
| `PathUserData` | the app's data: `~/Library/Application Support/<Name>`, `~/.config/<Name>` (`$XDG_CONFIG_HOME`) or `%AppData%\<Name>`; created on first use |
| `PathCache` | the app's cache: `~/Library/Caches/<Name>`, `~/.cache/<Name>` or `%LocalAppData%\<Name>`; created on first use |
| `PathLogs` | where to write logs: `~/Library/Logs/<Name>`, else `logs` in the user data directory; created on first use |
| `PathResources` | the files the app ships with, see [resources](distribution.md#resources) |
| `PathAppData` | the per-user configuration directory, without the app's name |
| `PathHome`, `PathTemp`, `PathExe` | the home directory, the temporary directory, the executable |
| `PathDesktop`, `PathDocuments`, `PathDownloads`, `PathMusic`, `PathPictures`, `PathVideos` | the user's folders |

```go
dir, err := mygo.App.Path(mygo.PathUserData)
db := filepath.Join(dir, "notes.db")
```

`App.SetPath` overrides a directory, for example to keep test data apart.

## Name and version

A packaged app knows its name and version from its configuration, which
`mygo build` links into it: `App.Name()` and `App.Version()`. A program run with
`go run` has the executable's name and no version, unless it calls
`App.SetName` and `App.SetVersion`, before `Run`. The name sets the user
data, cache and log directories and the single instance lock. Development
builds of `mygo dev` are named `<Name> Dev`, so that they keep their data
apart from the installed app's; calling `SetName` ties them together
again.

`mygo.IsDev()` reports whether the app is a development build: `go run`,
`go build` and `mygo dev` builds are, `mygo build` builds are not, and
`MYGO_ENV=production` makes any build behave like one. `App.IsPackaged()`
reports whether it runs from an app bundle. `App.Locale()` returns the
language the app shows, such as `"zh-Hans"`, of those it is written in,
and `App.PreferredLanguages()` the languages the user reads, as the system
lists them; see [the language the app shows](i18n.md#the-language-the-app-shows).
Before 0.5, `App.Locale()` returned the user's first preferred language
whatever the app is written in.

## The Dock and the taskbar

- `App.SetBadgeCount(n)` shows a count on the app's icon: the Dock (macOS)
  or the launcher entry (Linux docks that support it); zero clears it.
- `App.Dock` controls the Dock icon on macOS: `SetBadge` shows any text,
  `Bounce` asks for attention, `SetIcon` replaces the icon, `SetMenu` adds
  items to its menu (see [the Dock menu](menus.md#the-dock-menu)), and
  `Hide` and `Show` remove and restore it.
- `App.SetActivationPolicy(mygo.ActivationPolicyAccessory)` removes the
  Dock icon of an app that lives in the menu bar, see
  [tray icons](menus.md#tray-icons); its windows can still show.
  `App.ActivationPolicy` returns the policy.
- `App.Focus` brings the app to the front, and `App.IsActive` reports
  whether it is there: the active app, or the one with the focused window
  on Windows and Linux. On macOS, `App.Hide` and
  `App.Show` hide and show all its windows, and `App.ShowAboutPanel` shows
  the standard about panel.

### Jump lists

On Windows, right-clicking the app's taskbar button shows its jump list:
the documents the user opened recently (see
[recent documents](#recent-documents)) and tasks, programs to run with
arguments, often the app itself, such as a text editor's "New note".
`App.JumpList.SetTasks` sets the tasks; Windows keeps them after the app
quits, so setting them once at startup is enough:

```go
mygo.App.OnOpenURL(func(rawURL string) {
	if rawURL == "myapp://new-note" {
		newNote()
	}
})
// ... once the app is ready:
if err := mygo.App.JumpList.SetTasks([]mygo.JumpListTask{
	{Title: "New note", Description: "Write a new note", Args: []string{"myapp://new-note"}},
	{Title: "Notepad", Path: `C:\Windows\notepad.exe`},
}); err != nil {
	log.Println(err)
}
```

A task that runs the app starts it again with `Args`: when the app is not
running, the new instance gets them; when it is, with
[a single instance](#a-single-instance) the running one does. A URL of
one of the app's [schemes](#deep-links) reaches `OnOpenURL` either way,
and a file of its types `OnOpenFile`, so tasks are best made of them.
Other arguments go to the new instance's `os.Args`, and to the running
one's `OnSecondInstance`: handle them in both places. Each argument is
passed as it is, and MyGo quotes them for the command line.

`Path` is the app's own executable unless set, `WorkingDir` the
directory it runs in, `Description` the tooltip, and the icon is the
program's own unless `IconPath` (with `IconIndex`) names another. No
tasks removes them. Windows replaces the whole jump list whenever the app
changes a part of it, so `App.JumpList` keeps the parts and commits them
together. Its methods do nothing on macOS and Linux.

## Drawing web pages on Linux

WebKitGTK draws pages on the GPU with its DMA-BUF renderer where it can,
and that renderer shows blank windows with NVIDIA's driver, in virtual
machines and in containers. So MyGo turns it off by default: before GTK
starts, `App.Run` sets `WEBKIT_DISABLE_DMABUF_RENDERER=1` unless the
environment already has the variable. Pages then draw on the CPU, which
costs frame rate where the renderer would have worked: on a Wayland desktop
with Intel's GPU, a long page (a chat transcript of 4,600 elements)
scrolls at 11–19 frames per second with a core busy, against 56 with the
renderer and half the CPU.

An app that knows where it runs can leave the renderer on, before
`App.Run`:

```go
mygo.App.SetDMABUFRenderer(mygo.DMABUFRendererAuto)
```

| Setting | The renderer |
|---|---|
| `DMABUFRendererOff` (the default) | off |
| `DMABUFRendererAuto` | on where the host is known to draw with it, off elsewhere |
| `DMABUFRendererOn` | left to WebKitGTK, which uses it wherever it can |

`DMABUFRendererAuto` leaves it on when GTK will open a Wayland display
(`WAYLAND_DISPLAY` names a socket that exists, and `GDK_BACKEND` does not
put another backend first) and the system has a GPU with a Mesa driver
(a render node of a driver with 3D): not NVIDIA's driver
(`/sys/module/nvidia`, `/proc/driver/nvidia/version`), not software OpenGL
(`LIBGL_ALWAYS_SOFTWARE`, `GALLIUM_DRIVER=llvmpipe`), not in a container
(`/.dockerenv`, `/run/.containerenv`), WSL or a virtual machine (the
firmware's `product_name` or `sys_vendor` names VirtualBox, VMware, QEMU,
KVM, Hyper-V and others). Use `DMABUFRendererOn` only where you know the
machines, as in a kiosk.

`WEBKIT_DISABLE_DMABUF_RENDERER`, set by the user, always wins, whatever
the app chose: `0` turns the renderer on, `1` off. Other systems ignore the
setting.

## Browsing data

`App.ClearBrowsingData()` deletes what the app's pages stored: cookies,
local and session storage, IndexedDB, service workers and caches, for
example when the user signs out. Open pages keep what they hold in memory
until they reload. It clears the default session; `Session.ClearData`
clears another, or some kinds of data only.

### Cookies

`mygo.Cookies` reads and changes the cookies of the app's pages, `HttpOnly`
ones included: the default cookie store, which all windows share and
which the system webview keeps from one run to the next. Set one, for
example, to sign the pages in with a token the backend got, before the
first page loads, or read the session a page's sign-in set:

```go
err := mygo.Cookies.Set(mygo.Cookie{
	Name:     "session",
	Value:    token,
	Domain:   "example.com",
	Secure:   true,
	HTTPOnly: true,
	Expires:  time.Now().Add(30 * 24 * time.Hour),
})

// The cookies a request to the URL sends.
sent, err := mygo.Cookies.List(mygo.CookieFilter{URL: "https://example.com/account"})
// Those named session of example.com and its subdomains.
sessions, err := mygo.Cookies.List(mygo.CookieFilter{Name: "session", Domain: "example.com"})
// Every cookie, sorted by domain, path and name.
all, err := mygo.Cookies.List(mygo.CookieFilter{})

err = mygo.Cookies.Delete(mygo.Cookie{Name: "session", Domain: "example.com"})
err = mygo.Cookies.Clear() // every cookie, but not the pages' storage
```

A cookie's name, domain and path identify it: `Set` replaces the cookie of
the same name, domain and path, and `Delete` deletes it, whatever its other
fields, so it deletes a cookie `List` returned. A domain without a leading
dot is a host's, sent to that host only; with one, `.example.com`, the
cookie goes to its subdomains too. Domains are ASCII (punycode for others)
and an empty path is `/`. An empty `SameSite` is `Lax`, and `None` needs
`Secure`. Names and values take what browsers take: anything but control
characters, `;`, and spaces at either end, and names no `=`. A zero
`Expires` makes a cookie that lasts until the app quits, and one in the
past deletes the cookie.

A filter's fields narrow the list: `URL`, an http or https URL, keeps the
cookies a request to it sends, matched by the engine where it can (by
domain, path and `Secure` as RFC 6265 says on macOS); `Name` keeps those
of a name; `Domain` those of a domain and its subdomains, with or without
a leading dot; `Path` those of exactly a path.

The methods are safe from any goroutine once the app is running, and on
the main thread the app keeps handling events while they wait. Cookies
may carry the user's credentials: rather than binding the module to the
pages, bind a backend method that does what the page needs.

On Windows, WebView2 gives cookies only through a web view: while no
window or web view in native UI shows a page, the methods, and
`App.ClearBrowsingData`, make a hidden one, which also makes the WebView2
browser process start, and a window's first page waits for the calls
under way when its web view is made. The hidden web view closes once a
page shows, or, in an app showing none, a few seconds after the last
call, and WebView2's processes exit with it; it stays while the
session has cookies that WebView2 keeps only while they run: those with
a zero `Expires`, and any of an ephemeral session. WebView2 applies changes after its calls return, so the methods
wait until the change shows, and fail when it does not within two
seconds, as when the engine refuses a cookie. On Linux they need
WebKitGTK 2.42 or newer, and `Set` libsoup 2.70 or newer; WebKitGTK keeps
the cookies in `cookies.sqlite` of the app's user data directory
(`~/.config/<Name>`).

### Sessions

Every page shows in the default session unless `PageOptions.Session` names
another: a store of cookies, local and session storage, IndexedDB, caches
and service workers of its own, which the pages of other sessions don't
see, as Electron's partitions. Two accounts of a service side by side, a
private window, a preview that must not see the user's sign-ins, or a
third-party page kept away from the app's own storage each take one:

```go
work := mygo.NewSession("work")        // kept on disk from one run to the next
private := mygo.NewEphemeralSession()  // in memory only

win := mygo.NewWindow(mygo.WindowOptions{
	URL:  "https://mail.example.com",
	Page: mygo.PageOptions{Session: work},
})

list, err := work.Cookies.List(mygo.CookieFilter{})
err = work.Cookies.Set(mygo.Cookie{Name: "session", Value: token, Domain: "example.com"})
err = work.ClearData(mygo.SiteDataAll)                             // signing out
err = work.ClearData(mygo.SiteDataCookies | mygo.SiteDataCaches)  // some kinds
```

`NewSession` returns the same session for the same name, whose data stays
on disk as the default session's does. `NewEphemeralSession` returns a new
session every time, which writes nothing to disk: its data goes when the
last page using it closes, and its next page starts with none. A cookie set
before its first page opens goes to that page. `mygo.Cookies` and
`App.ClearBrowsingData` are the default session's, which
`mygo.DefaultSession()` returns too. A window a page opens with
`window.open` or a `target="_blank"` link is in its opener's session,
whatever the `Session` of the options the open handler returns. Custom
protocols (`mygo.Protocol`) serve the pages of every session, and a web
view in [native UI](ui/web-view.md) takes a session in its `PageOptions`
too.

`ClearData` takes `SiteDataCookies`, `SiteDataLocalStorage` (with session
storage), `SiteDataIndexedDB` (with WebSQL), `SiteDataCaches` (the HTTP
cache and Cache Storage), `SiteDataServiceWorkers`, or `SiteDataAll`: all
the data the engine keeps for sites, those kinds and the others it has.

| | persistent session | ephemeral session |
|---|---|---|
| macOS | `WKWebsiteDataStore dataStoreForIdentifier:` with a UUID made of the name, from macOS 14; before, an in-memory store that lasts until the app quits | `nonPersistentDataStore` |
| Linux | a web context of its own, with its data in `Sessions/<UUID>` of the user data directory and its caches in that of the cache directory; WebKitGTK 2.16 or newer | a web context with an ephemeral website data manager |
| Windows | a WebView2 profile, `mygo-<UUID>`, in the app's WebView2 user data; WebView2 Runtime 1.0.1185 or newer | an InPrivate profile, `mygo-private-<n>`, which WebView2 drops with its last web view |

On Windows, the cookies of a session come from a web view of its profile,
as the default session's do: before one of its windows shows a page, the
calls make a hidden one.
