# Desktop APIs

MyGo wraps the parts of the operating system desktop apps use most, for
apps whose windows show web pages, native UI or both. Like the rest of
MyGo, each can be called from any goroutine.

## Dialogs

`mygo.Dialog` shows the system's file and message dialogs. Its methods
block until the user answers: call them from bound methods, which run on
goroutines of their own, or start a goroutine for them in event listeners
and menu items, which should return quickly. `Parent` attaches a dialog to a
window, as a sheet on macOS.

```go
// Pick files.
paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
	Parent:   win,
	Title:    "Import",
	Filters:  []mygo.FileFilter{{Name: "Images", Extensions: []string{"png", "jpg"}}},
	Multiple: true,
}) // nil when canceled

// Choose where to save.
path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
	Parent:      win,
	DefaultPath: "notes.json",
}) // "" when canceled

// Ask a question.
res, err := mygo.Dialog.Message(mygo.MessageOptions{
	Parent:             win,
	Type:               mygo.MessageWarning,
	Message:            "Delete 3 notes?",
	Detail:             "You cannot undo this.",
	Buttons:            []string{"Cancel", "Delete"},
	DestructiveButtons: []int{1},
	CheckboxLabel:      "Don't ask again",
})
if res.Button == 1 { /* Delete */ }

// Report an error.
mygo.Dialog.Error("Import failed", err.Error())
```

`OpenDialogOptions` also pick directories (`Directory`), show hidden files
and, on macOS, a message and a New Folder button.

A filter takes files by type too, as HTML's `accept`: `MIMETypes` such as
`"application/pdf"`, or all of a kind, `"image/*"`, which the system's
picker matches as it knows them, rather than a list of every extension:

```go
mygo.FileFilter{Name: "Pictures", MIMETypes: []string{"image/*"}}
```

macOS 11 and later take them as types of files (`UTType`), the kinds
`image/*`, `audio/*`, `video/*`, `text/*` and `font/*` as theirs
(`public.image` and the others), which is all earlier versions take; GTK
matches them, wildcards too; Windows filters by the extensions its
registry gives the type, or every type of the kind and every file its
`PerceivedType` says is of it.

In message dialogs Enter activates `DefaultButton` and Escape
`CancelButton`, also when they are one button, as Cancel above. On macOS
Cmd+. cancels too, and a `DefaultButton` of -1 leaves Enter to no button.
`CancelButton` defaults to the first button labeled Cancel or No, else to
the first button. For Delete / Keep, set `CancelButton: 1` to choose Keep;
otherwise Escape chooses Delete. `DestructiveButtons` sets apart buttons
that delete or can't be undone.

## Notifications

`NotificationPermission()` queries current OS authorization without prompting:
`unknown`, `denied`, `allowed`, or `unsupported`, plus an API error separately.
It re-queries settings rather than caching consent. From an explicit enable
interaction, call `RequestNotificationPermission()`. macOS requests alert
presentation only; Windows reads the sender's current notification setting and
never rewrites the user's OS preferences. `Show` never requests authorization.
A successful submission says the OS accepted the request, not that a banner was
visible: Focus, sharing and presentation settings remain authoritative.

```go
status, err := mygo.RequestNotificationPermission() // explicit enable action
if err != nil || status != mygo.NotificationAuthorizationAllowed {
    return
}
n := mygo.NewNotification(mygo.NotificationOptions{
    ID: "stable/1.2.3/available",
    Title: notificationTitle, Body: notificationBody, Silent: true,
    Actions: []mygo.NotificationAction{
        {ID: "view", Title: viewLabel, Foreground: true},
        {ID: "later", Title: laterLabel},
    },
})
if err := n.Show(); err != nil {
    log.Println(err)
}
```

macOS requires a packaged app with a stable bundle identifier. Its delegate is
attached before launch completes and receives the response's notification ID
and action ID. `Foreground` asks macOS to activate the app for that action;
background actions can update application policy without opening a window.
Linux uses the desktop's D-Bus notification service and forwards action IDs.

On Windows, native WinRT toast requires installed per-user registration. Set
`windows.notifications: true` in `mygo.json`, and call the installer's entry
point first in `main`, before bootstrap or the single-instance lock:

```go
if handled, err := mygo.HandleNotificationRegistration(); handled {
    if err != nil { log.Fatal(err) }
    return
}
```

The NSIS installer invokes this entry point to create a Start menu shortcut
with `System.AppUserModel.ID` and `System.AppUserModel.ToastActivatorCLSID` and
an HKCU `LocalServer32` command quoting the executable path. A stable activator
CLSID derives from the app identifier. Startup registers the COM class factory
and uses the same AUMID for `Windows.UI.Notifications`. Activation accepts only
a bounded versioned ID/action envelope, never commands or URLs. An existing
registered process receives warm activation; a cold process queues activation
until application listeners can handle it. A bounded singleton relay handles
the startup race where another process owns the lock before registering COM.
Uninstall clears the owned sender's history before removing its shortcut and
COM registration. Only install registration failure aborts setup. Uninstall
still runs the helper, then continues, so shortcut, uninstall-key, and file
removal proceed. Portable executables without registration return an explicit
capability error. There is no balloon fallback and no Windows App SDK runtime
deployment dependency.

Register `App.OnNotificationAction` before `Run` to receive both body activation
(action `""`) and custom IDs, including notifications from an earlier process.
Use either this application handler or `OnNotificationClick` for navigation;
body activation also emits `OnNotificationClick`, while custom actions do not.
Validate application-specific channel/version/action values in the handler.
Notification IDs are nonempty UTF-8 of at most 256 bytes without control
characters. Up to four actions use unique lowercase ASCII token IDs of at most
64 bytes, with localized labels. No text is interpreted as an install command.

`n.Close()` or `RemoveNotification(id)` removes a stable ID, including macOS and
Windows notifications from earlier runs. `ClearNotifications()` removes this
sender's history. macOS also removes pending requests; generations invalidate a
submission removed while its asynchronous permission query was in flight.
Windows uses a deterministic 16-character tag and a framework group so removal
does not depend on an in-memory object. `Group` additionally controls macOS's
thread grouping. Windows history-removal failures are logged; the installer
checks cleanup errors directly before unregistering the sender.

The isolated [notification probe](../examples/notification-probe/) has its own
sender identity. Arion's `node scripts/check-notifications.mjs` runs portable
wire/API/installer regressions and Windows cross-builds. `--native` builds and
runs the actual packaged permission/query/cleanup lifecycle on the host;
`--interactive` additionally requests the probe's consent when needed and waits
for actual warm view/later and cold body/later actions from an OS UI driver.
`--prepare` leaves a test-owned JSONL-controlled probe for a separate driver.
Never reset a user's permission database to manufacture fresh-consent cases.
Cross-builds and API submissions are not banner/cold-activation acceptance.

`Image` shows an image with the notification, the bytes of a PNG or JPEG,
such as the cover of the track that started playing. macOS attaches it,
as a thumbnail that grows as the notification expands; Linux gives the
desktop's notification service its pixels, scaled down to fit 256 pixels;
Windows shows it as the balloon's large icon, in place of the app's. An
image the system cannot decode leaves the notification without one.
`NewNotification` copies it, so the app can reuse its buffer; MyGo does
not fetch remote images:

```go
// One notification for what plays, replaced as the track changes.
mygo.NewNotification(mygo.NotificationOptions{
	ID:     "now-playing",
	Title:  track.Title,
	Body:   track.Artist + " — " + track.Album,
	Image:  track.Cover,
	Silent: true,
}).Show()
```

### Sender avatars on macOS

An incoming direct message can supply a sender to display a prominent
avatar with the app's small icon on macOS 12 and later:

```go
mygo.NewNotification(mygo.NotificationOptions{
	ID: "message:7", Group: "conversation:3",
	Title: "Ada", Subtitle: "Lunch", Body: "See you at noon",
	Sender: &mygo.NotificationSender{
		ID: "user:ada", Name: "Ada", Avatar: avatarPNG,
	},
}).Show()
```

`Sender.ID` identifies the person in the app; it is not inferred to be an
email address or phone number. `Name` defaults to `Title`. `Avatar` holds
encoded image bytes, such as PNG, JPEG or SVG; macOS rasterizes them as PNG
before donation. The app fetches them and decides
whether the message's sender and content may be shared with the system.
`NewNotification` copies the sender and image, so the app can reuse its
buffers. `Group` is the conversation identifier; without one, it uses the
notification's ID. This API represents a direct incoming message, not a
group chat or outgoing message. macOS controls the communication layout;
the subtitle is included above the message in the body.

Enable **Communication Notifications** for the app's Apple App ID and add
`INSendMessageIntent` to `macos.infoPlist.NSUserActivityTypes`. Sign with
`com.apple.developer.usernotifications.communication` and embed a matching
provisioning profile, including for Developer ID distribution: set
`macos.provisioningProfile`, which `mygo build` copies to
`Contents/embedded.provisionprofile` before signing (see
[Provisioning profiles](distribution.md#provisioning-profiles)). The
profile must authorize the app's identifier, certificate and entitlements.
Do not add the restricted entitlement to an ad-hoc build: macOS can refuse
to launch it.

MyGo checks the signed entitlement before donating the incoming message to
SiriKit and updating its notification content. Without the capability, with
an incomplete sender or invalid avatar, or when donation/content conversion
fails, it sends the ordinary notification. Windows, Linux and unsupported
platforms retain their existing behavior. IDs, sound, grouping, click
handlers, `Close` and `ClearNotifications` use the same notification lifecycle.

## Clipboard

```go
mygo.Clipboard.WriteText("hello")
text := mygo.Clipboard.ReadText()

mygo.Clipboard.WriteHTML("<b>hello</b>")
mygo.Clipboard.WriteImage(png) // a PNG
img := mygo.Clipboard.ReadImage()

formats := mygo.Clipboard.AvailableFormats()
mygo.Clipboard.Clear()
```

Pages can use the web's `navigator.clipboard` too.

`Clipboard.Write(transfer.Data)` and `Clipboard.Read(formats...)` share the
native drag/drop model: ordered items, alternative representations, file
lists, custom MIME formats, and lazy providers. `Formats` discovers portable
names without requesting bytes; `ReadFormat`, `ReadFiles`, `WriteFiles`, and
`Flush` provide explicit negotiation, file transfer, and persistence. See
[Clipboard and drag data](data-transfer.md) for provider lifetimes and native
format mappings. The [clipboard example](../examples/clipboard) demonstrates
these operations.

## The shell

```go
mygo.Shell.OpenExternal("https://example.com") // the default browser
mygo.Shell.OpenPath("/Users/ada/report.pdf")   // the default app for the file
mygo.Shell.ShowItemInFolder(path)              // Finder, Explorer, the file manager
mygo.Shell.TrashItem(path)                     // to the trash
mygo.Shell.Beep()
```

`TrashItem` never deletes for good: when the item cannot go to the trash
it fails and leaves the item where it is. On Windows that is an item too
big for the recycle bin, or on a drive without one, as network shares and
many removable drives are, which Windows would otherwise delete without
asking.

`FileIcon` returns the icon the file manager shows for a file, a folder
or an application, as a PNG of the size asked for in pixels: pass the size
in DIPs times the display's scale factor.

```go
png, err := mygo.Shell.FileIcon("/Applications/Safari.app", 64)
```

On macOS it is Finder's icon (`NSWorkspace`), drawn from the
representation that fits the size best; a symbolic link has the icon of
what it points to, without Finder's arrow. On Windows it is Explorer's
(`IShellItemImageFactory`, icons only, no thumbnails), and the path may
also name a shell item, such as an app of the Start menu by
`shell:AppsFolder\<AppUserModelID>`. On Linux it is the icon of the file's
content type from the icon theme, and for a desktop entry (`.desktop`), the
icon of its application, or where the theme has none, that of desktop
entries.

`TypeIcon` returns the icon of a file type, by its extension, with or
without the dot, without a file of that type on disk: the icon of an
attachment not yet downloaded, say. It is drawn as `FileIcon` draws its
icon, and fails in browsers.

```go
png, err := mygo.Shell.TypeIcon("pdf", 64)
```

`InstalledApps` lists the applications installed on the computer, sorted
by name, as a launcher shows them, and `LaunchApp` starts one, or brings
it to the front:

```go
go func() {
	apps, err := mygo.Shell.InstalledApps() // reads the disk: off the main thread
	if err != nil {
		return
	}
	for _, app := range apps {
		fmt.Println(app.Name, app.ID)
	}
	icon, _ := mygo.Shell.FileIcon(apps[0].Path, 64)
	_ = mygo.Shell.LaunchApp(apps[0])
}()
```

Each app has the `Name` the system shows, in the user's language, an
`ID` and the `Path` whose icon `FileIcon` draws:

| | the apps | `ID` | `Path` |
|---|---|---|---|
| macOS | the `.app` bundles of `/Applications`, `/System/Applications`, `~/Applications` and CoreServices' `Applications`, and of their folders (Utilities), plus Finder; an app behind symbolic links once, at the link, as `/Applications/Safari.app` | the bundle identifier, or the path for a bundle without one | the bundle |
| Windows | the Start menu's All apps, desktop and packaged apps alike (the shell's Applications folder) | the AppUserModelID, or for a desktop app without one, its path under a known folder | `shell:AppsFolder\<ID>` |
| Linux | the desktop entries menus show (not `NoDisplay`, `Hidden`, or meant for another desktop) | the desktop file ID, such as `org.gnome.Nautilus.desktop` | the desktop entry |

`InstalledApps` does not block the main thread when called from another
goroutine, and returns in tens of milliseconds. Drawing icons takes a few
milliseconds each, on the main thread: draw those that show, and keep
them.

## Displays

`mygo.Screen` describes the connected displays, in device-independent
pixels with the origin at the top-left corner of the primary display:

```go
primary := mygo.Screen.PrimaryDisplay()
fmt.Println(primary.Bounds, primary.WorkArea, primary.ScaleFactor)

for _, d := range mygo.Screen.Displays() { /* ... */ }

cursor := mygo.Screen.CursorScreenPoint()
d := mygo.Screen.DisplayNearestPoint(cursor)

mygo.Screen.OnDisplaysChanged(func() { /* rearrange windows */ })
```

`WorkArea` leaves out the menu bar, the Dock and taskbars.

On Windows, displays can have different scales. Each display's
coordinates start at its top-left corner in pixels divided by the
smallest scale of the displays, and go on in its own scale, so that
displays never overlap: one with a larger scale than another's leaves a
gap after it. Where every display has the same scale, coordinates are
the pixels divided by that scale. A window's bounds are in the scale of
the display it is the most on.
`DisplayMatching(rect)` returns the display a rectangle overlaps most, the
one a window is on.

## Dark mode

`mygo.Theme` is the light or dark appearance, which pages follow through
the `prefers-color-scheme` media query:

```go
dark := mygo.Theme.IsDark()
mygo.Theme.OnUpdated(func() { log.Println("dark:", mygo.Theme.IsDark()) })

mygo.Theme.SetSource(mygo.ThemeDark) // or ThemeLight, or ThemeSystem to follow the system
```

`SetSource` may be called before `App.Run`, for example with an appearance
the user saved in the app's preferences: the app then starts in it.

## Accent color and accessibility

`mygo.Theme.Preferences()` returns the accent color of the desktop's own
controls and the accessibility settings they follow, all read at once:

```go
p := mygo.Theme.Preferences()
p.Accent             // a color.NRGBA (image/color), A 0 where the desktop has none
p.ReduceMotion       // the desktop asks for less motion
p.ReduceTransparency // the desktop asks for opaque surfaces, e.g. no vibrancy
p.HighContrast       // the desktop asks for more contrast
p.TextScale          // how many times larger text should be; 1 is the usual size
```

The accent is 8-bit sRGB with straight alpha; `ui.RGB(p.Accent.R, p.Accent.G, p.Accent.B)`
makes it a `ui.Color` for drawing of your own.

`ReduceMotion` is macOS's Reduce Motion, Windows's animation effects and
GNOME's animations turned off; `ReduceTransparency` macOS's Reduce
Transparency and Windows's transparency effects turned off, which Linux
desktops do not have, so it is false there; `HighContrast` macOS's
Increase Contrast, Windows's contrast themes and the higher contrast of
the settings portal (or a HighContrast GTK theme) on Linux. macOS has no
text scale, so `TextScale` is 1 there. The accent is missing on Linux
desktops whose settings portal has none, and in browsers.

`Theme.OnUpdated` fires when the desktop changes any of them, as
`SetSource` does. Reading them can call out to the desktop (the settings
portal on Linux), so MyGo reads them once and keeps them until the next
change. Package `ui`'s default theme already follows the accent, the
contrast and the text scale, and `Animate` reduced motion. Like the other
calls that need the running app, `Preferences` panics in `main` before
`App.Run`, and another goroutine calling it waits for the app to start.

Pages get them from document start on, as `preferences()` of
`mygo-runtime` and the `--mygo-accent` and `--mygo-text-scale` CSS
variables, see [the frontend guide](frontend.md#accent-color-and-accessibility);
`prefers-reduced-motion` and `prefers-contrast` already follow the desktop.
A `ThemePreferences` sent through an event or returned by a bound method
is that same JSON, and the generated client types it with mygo-runtime's
`ThemePreferences`.

## Power

```go
// Keep the computer awake during a long task, and the display on with true.
release := mygo.Power.KeepAwake("Exporting the video", false)
defer release()

mygo.Power.OnSuspend(func() { pauseSync() })
mygo.Power.OnResume(func() { resumeSync() })
mygo.Power.OnLockScreen(func() { hideSecrets() })
mygo.Power.OnUnlockScreen(func() { /* ... */ })

onBattery := mygo.Power.IsOnBattery()
idle := mygo.Power.IdleTime() // since the last keyboard or mouse input
```

The reason given to `KeepAwake` shows where the system lists what keeps it
awake. On Linux, `OnLockScreen` is called when the screen saver starts,
which usually locks the screen, and `IdleTime` is 0 on desktops that do not
report it.

## Now Playing and media keys

`mygo.NowPlaying` shows what the app plays where the system shows media:
Now Playing in Control Center and the menu bar on macOS, the media
controls of the volume flyout and the lock screen on Windows, and on Linux
the MPRIS clients, as GNOME's and KDE's media controls and `playerctl`.
While it shows something, the media keys of keyboards and headsets, and
those controls, send the app commands:

```go
mygo.NowPlaying.Set(mygo.NowPlayingInfo{
	Title: "Spring", Artist: "Vivaldi", Album: "The Four Seasons",
	Artwork:  cover, // PNG or JPEG
	Duration: 3*time.Minute + 20*time.Second,
	Commands: mygo.MediaPlay | mygo.MediaPause | mygo.MediaNext | mygo.MediaPrevious | mygo.MediaSeek,
})
mygo.NowPlaying.OnCommand(func(cmd mygo.MediaCommand, pos time.Duration) {
	switch cmd {
	case mygo.MediaPause:
		player.Pause()
		mygo.NowPlaying.SetPlayback(mygo.PlaybackPaused, pos) // where playback is
	case mygo.MediaSeek:
		player.Seek(pos)
		mygo.NowPlaying.SetPlayback(mygo.PlaybackPlaying, pos)
	case mygo.MediaNext:
		playNext() // and Set the next track's info
	}
})
mygo.NowPlaying.Clear() // the app no longer shows in the controls
```

The system advances the position while the state is `PlaybackPlaying`, so
the app sets it again only when playback jumps. The play/pause key sends
`MediaPause` while the state is playing and `MediaPlay` otherwise, and
only the commands the info offers reach `OnCommand` (`Commands` 0 offers
play, pause and stop). `pos` is where `MediaSeek` moves to, and for the
other commands where playback is as they come. An app playing a player of package `audio` can let
`Control` play, pause, stop and seek it, and show its state:

```go
release := mygo.NowPlaying.Control(music) // an *audio.Player
defer release()
```

`Control` hears the player's end; after the app plays, pauses or seeks it
itself, `NowPlaying.Sync` shows it. In browsers, `NowPlaying` is the
page's Media Session.

## Network

`mygo.Network` tells whether the computer is online and whether the
connection is metered — a phone hotspot or a plan that counts the data,
where an app should hold off on large downloads:

```go
status := mygo.Network.Status()
if status.Online && !status.Metered {
	syncNow()
}

mygo.Network.OnChanged(func(s mygo.NetworkStatus) {
	log.Printf("online: %v, metered: %v", s.Online, s.Metered)
})
```

`Online` means the system has a connection to a network beyond the
computer, as the browsers' `navigator.onLine` does, which does not prove
that a server answers: a network behind a captive portal, or one whose
way to the internet is down, counts. macOS and Linux count a route out
(a satisfied path, GLib's `network-available`), Windows a connectivity
level of local access or more. `Metered` is an expensive path (cellular, a
phone's hotspot) or Low Data Mode on macOS, a metered connection on Linux,
and on Windows a connection whose cost is fixed or variable, or that
roams, is over its data limit or approaches it. In browsers `Online` is
`navigator.onLine`, and `Metered` is false.

`Status` is safe from any goroutine; on macOS the first call waits for
the system's first answer, which comes within milliseconds. Changes are
watched from the first `OnChanged` on, whose listener runs on the main
thread, only when the status changed; it returns a function that removes
the listener.

Pages already have `navigator.onLine`. To show the native status in the
UI, broadcast a typed event from the listener, so the generated client
receives it:

```go
// Connectivity is pushed to every window when the connection changes, as
// {online, metered}.
var Connectivity = mygo.NewEvent[mygo.NetworkStatus]("connectivity")

func init() {
	mygo.Network.OnChanged(func(s mygo.NetworkStatus) { Connectivity.Broadcast(s) })
}
```

### Proxies

Web pages reach the network through the webview, which follows the
proxy the user set in the system's settings, with its bypass list and
setup script (PAC file): WebView2 as Chromium does, from Windows' proxy
settings; WKWebView from macOS's; WebKitGTK through GIO's default proxy
resolver, which reads GNOME's settings, or KDE's through libproxy.

Go's `http.DefaultClient` reads only `HTTP_PROXY`, `HTTPS_PROXY` and
`NO_PROXY`. Package `netproxy` makes Go's requests follow the system's
settings as the pages do, and MyGo's own requests use it: those of the
[updater](updates.md) and, unless their options name a client, of the
[fetch](plugins/fetch.md) and [websocket](plugins/websocket.md) plugins.
Your own clients opt in with its transport:

```go
import "github.com/egoist/mygo/netproxy"

client := &http.Client{Transport: netproxy.Transport(), Timeout: time.Minute}
```

or, for a transport of your own, `t.Proxy = netproxy.Proxy`. For each
request:

1. Loopback hosts (`localhost`, `*.localhost`, `127.0.0.0/8`, `::1`) are
   reached directly.
2. When `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY` or `NO_PROXY` (or its
   lowercase name) is set, the environment decides, as
   `http.ProxyFromEnvironment` does, so that a proxy set for one run of
   the app wins.
3. Otherwise the system's settings for the URL decide: on Windows the
   user's, read by WinHTTP, with automatic detection (WPAD) and setup
   scripts; on macOS those of System Settings > Network > Proxies, read
   by CFNetwork, with setup scripts; on Linux GIO's default proxy
   resolver, as WebKitGTK. Without an answer, requests go direct.

The system's answer for a scheme and host is kept for 5 seconds, so
turning the proxy on or off, or changing it, applies to the requests that
follow while the app runs. Requests under way, such as a download or a
stream, keep their connection. Go's transport speaks to HTTP and HTTPS
proxies and SOCKS5, not SOCKS4; it sends a proxy's password only when the
system's answer holds it, as GNOME's settings can. In browsers, the
browser makes Go's requests, through its own proxy.

## Secrets

`mygo.Secrets` keeps small secrets, such as access tokens and API keys, in
the system's credential store, which encrypts them for the user, unlike a
file in the user data directory: the login Keychain on macOS, the Secret
Service on Linux (GNOME Keyring, KWallet, KeePassXC…, through libsecret)
and the Credential Manager on Windows.

```go
if err := mygo.Secrets.Set("github-token", []byte(token)); err != nil {
	log.Println(err)
}

token, err := mygo.Secrets.Get("github-token")
if errors.Is(err, mygo.ErrSecretNotFound) {
	// Ask the user to sign in.
}

mygo.Secrets.Delete("github-token") // on sign out; nil when there is none
```

A secret may be at most `mygo.MaxSecretSize` (2560) bytes, the most the
Credential Manager keeps, and a key at most 256 bytes. To keep more,
encrypt it into a file in the user data directory with a key kept in
`Secrets`. The methods are safe from any goroutine, also before `Run`, and
may block while the system asks the user something, e.g. to unlock the
keyring; on the main thread the app keeps handling events meanwhile. To
use them from the page, bind a backend method that calls them, so that
only what the page needs crosses into it.

A store keeps a secret under a service and an account: `Secrets` takes the
key for the account and the app's identifier (`identifier` of
`mygo.config.ts`) for the service. The development app of `mygo dev` is
`<identifier>.dev`, so it keeps its secrets apart from the installed app's;
an app run without an identifier, as with `go run`, uses its name.
`Secrets.Service` chooses another service, to share secrets with another
program of the app, such as its command line tool, or to read those an
Electron app stored with keytar, which keeps them as `Secrets` does under
the service the app passed it:

```go
// keytar.getPassword("MyApp", "refresh-token") in the Electron app
token, err := mygo.Secrets.Service("MyApp").Get("refresh-token")
```

Secrets protect against other users and stolen disks, not against other
apps of the same user: macOS asks the user before another app reads them,
and before a rebuilt app does when it is not signed with the same
identity, while Linux and Windows let them read.

Where there is no credential store, the methods fail with an error for
which `errors.Is(err, errors.ErrUnsupported)` holds: in browsers, and on
Linux without libsecret (the `libsecret-1-0` package on Debian and Ubuntu),
a session bus or a Secret Service on it, as on servers and minimal
desktops. An app can then fall back, e.g. ask for the secret each time:

```go
token, err := mygo.Secrets.Get("github-token")
switch {
case errors.Is(err, errors.ErrUnsupported):
	// No credential store: ask the user, keep it in memory.
case errors.Is(err, mygo.ErrSecretNotFound):
	// Ask the user to sign in.
}
```

## Watching files

`mygo.WatchFiles` watches a file or a directory, and with `Recursive` the
directories in it, for changes. It returns once everything is watched, and
`Next` waits for the next changes and returns them in a batch; use it on a
goroutine of its own:

```go
w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{
	Recursive: true,
	Ignore: func(p string, isDir bool) bool {
		return isDir && (path.Base(p) == "node_modules" || path.Base(p) == ".git")
	},
})
if err != nil {
	return err
}
go func() {
	defer w.Close()
	for {
		events, err := w.Next(ctx)
		if err != nil {
			return // io.EOF once closed, or once dir went away
		}
		for _, e := range events {
			log.Println(e.Op, e.Path)
		}
	}
}()
```

An event's `Path` is relative to what is watched (`w.Root()`, its path
with symlinks resolved) and separated by slashes, `.` for itself, and `Op`
is `mygo.FileCreate`, `FileWrite`, `FileRemove` or `FileRename`, with
`OldPath` where a rename moved from. What is already there is not
reported. Changes wait for 50 ms of quiet (`Debounce`, right away when
negative) before they are reported, and never more than 500 ms
(`MaxDelay`). A watch compares what it sees with what it saw, so events say
what is different, not every step that made it so: treat them as hints to
read the files again. A write is a file whose size or modification time
changed, or that the system says was written, or a file replaced at its
path by another, as editors and atomic saves replace it (a temporary file
renamed over it): permissions and other metadata, which indexers and
antivirus touch, are not reported, nor are directories written, whose
changes are their entries'.

A watch doesn't look again until `Next` has taken the last batch: when the
app is busy, what changed meanwhile comes in the next batch, so nothing is
lost and nothing piles up. What the watch can't look at, such as a
directory without permission, is left out rather than ending it.
`Ignore` leaves out files and directories by
their path: a directory it ignores is never opened, and doesn't count in
`MaxEntries`.

A file is watched through its directory, as [fsnotify](https://github.com/fsnotify/fsnotify)
recommends: a file replaced, as editors save, is a write of `.`, one
removed a removal, and one that appears again a creation. When the
directory itself, or the directory of the file, is removed, renamed or
replaced, the last batch has a `FileRemove` of `.`, and `Next` then returns
`io.EOF`.

`Close`, or canceling the context given to `WatchFiles`, stops the watch:
`Next` returns `io.EOF`, and `Done` is closed once it has released its
handles. The app stops its watches when it quits. A watch stops with
`mygo.ErrWatchLimit` when its directory has more than `MaxEntries` files
and directories: 65536 on macOS, where one FSEvents stream covers the
whole tree, and 8192 elsewhere, where a backend may take a handle per
entry. The [watch plugin](plugins/watch.md) gives
pages watches of the directories and files an app chooses, and says how
each platform watches.

`HintsOnly` trades all of that precision for cost. The watch keeps no
snapshot and stats no file: it reports only `mygo.FileHint` events, each
naming a directory in or under which the system said something changed,
and `.` after the system lost changes. Which file changed, how, and whether
it was renamed are never known. It suits a dev reloader, which rebuilds on
any change:

```go
w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{
	Recursive: true,
	HintsOnly: true, // only "something under this directory changed"
	Ignore: func(p string, isDir bool) bool {
		return isDir && path.Base(p) == "node_modules"
	},
})
```

Never use it where the file matters: following a document, syncing, or
anything that would act on a path it can't know. With it, `Ignore` is asked
about directories only (a change to a file it would ignore still hints at
the file's directory), and `MaxEntries` counts the directories enrolled. On
macOS and Windows a recursive hints-only watch enrolls only its directory,
which the system watches whole; on Linux it enrolls every directory,
reading their entries' types without stat'ing them. Symlinks are still not
followed, nothing outside the directory is watched, and the watched
directory going away still ends the watch with a `FileRemove` of `.`.

## Global shortcuts

Global shortcuts work while the app is in the background, for example to
bring it to the front:

```go
err := mygo.GlobalShortcut.Register("CmdOrCtrl+Shift+Space", func() {
	win.Show()
	win.Focus()
})
```

The function runs on the main thread. Shortcuts use the syntax of
[accelerators](menus.md#accelerators); `Unregister`, `UnregisterAll` and
`IsRegistered` manage them. `Register` fails when the shortcut is taken,
by the app or, where the system tells, by another app.

On Linux under X11 the app grabs the keys itself. Under Wayland, where apps
cannot, the desktop binds shortcuts for the app through the XDG desktop
portal, which KDE Plasma 6 and GNOME 48 and later provide: the desktop may
ask the user to confirm new shortcuts and lets them pick other keys, and it
needs the app installed, since it identifies apps by their desktop entry.
The shortcut works shortly after `Register` returns. Windows the function
shows or focuses take the focus with the key press's activation token where
the desktop provides one, as GNOME does.
