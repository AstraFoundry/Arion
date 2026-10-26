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
	Parent:        win,
	Type:          mygo.MessageWarning,
	Message:       "Delete 3 notes?",
	Detail:        "You cannot undo this.",
	Buttons:       []string{"Delete", "Cancel"},
	CheckboxLabel: "Don't ask again",
})
if res.Button == 0 { /* Delete */ }

// Report an error.
mygo.Dialog.Error("Import failed", err.Error())
```

`OpenDialogOptions` also pick directories (`Directory`), show hidden files
and, on macOS, a message and a New Folder button. In message dialogs Enter
activates `DefaultButton` and Escape `CancelButton`, by default the first
button labeled Cancel or No.

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
COM registration. Registration/removal failures abort setup visibly. Portable
executables without registration return an explicit capability error. There is
no balloon fallback and no Windows App SDK runtime deployment dependency.

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
