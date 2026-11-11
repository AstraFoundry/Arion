# Launchers

A launcher, as Spotlight, Raycast, Alfred, PowerToys Run or Ulauncher, is
summoned from anywhere with a keyboard shortcut, takes what the user types
over the app they are in, and goes away as fast as it came. MyGo has the
parts: a [panel](windows.md#panels) to show it in, a
[global shortcut](native.md#global-shortcuts) to summon it, the
[installed apps](native.md#the-shell) and their icons to search, and an app
without a Dock icon that lives in the [tray](menus.md#tray-icons).
[`examples/launcher`](https://github.com/egoist/mygo/tree/main/examples/launcher)
puts them together in native UI:

```sh
go run ./examples/launcher
```

## An app in the background

A launcher has no Dock icon and no entry in the app switcher, runs with
no window showing, and runs once:

```go
func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return // the launcher that runs shows itself
	}
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnSecondInstance(func([]string, string) { show() })
	mygo.App.WhenReady(ready)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

A packaged app says it from the start, so that macOS never shows it in the
Dock, with `LSUIElement` in [`mygo.json`](configuration.md#mygojson):

```json
{
  "macos": { "infoPlist": { "LSUIElement": true } }
}
```

A tray icon gives the user a way in other than the shortcut, and to quit;
[`App.SetOpenAtLogin`](app.md#start-at-login) starts the launcher with the
session, and `App.WasOpenedAtLogin` tells it to stay hidden then.

## The panel

A panel floats above every app, full-screen ones too, on whichever Space
or virtual desktop the user is on, and on macOS takes the keyboard without
activating the launcher: the app the user was in keeps its menu bar, and
has the keyboard back once the panel hides. Create it hidden, once, and
hide it rather than close it, so that it shows again at once, as it was:

```go
panel := mygo.NewWindow(mygo.WindowOptions{
	Panel:         true,
	Hidden:        true,
	Frameless:     true,
	Transparent:   true,
	Vibrancy:      mygo.VibrancyGlass, // Liquid Glass on macOS 26, Acrylic on Windows 11
	CornerRadius:  16,
	Width:         680,
	Height:        420,
	DisableResize: true,
	Content:       ui.View(app.view), // or URL: "/launcher"
})
panel.OnClose(func(e *mygo.CloseEvent) { // Escape, Cmd-W, Alt-F4
	e.PreventDefault()
	panel.Hide()
})
panel.OnBlur(panel.Hide) // the user went elsewhere
```

An Escape that the page or native UI leaves closes a panel, which the
`OnClose` listener turns into hiding it. Handle Escape yourself where it
does something first, as clearing the query does in the example.

Show it on the display the user is looking at, which the pointer tells,
a third of the way down as Spotlight does, and give its search field the
keyboard:

```go
func show() {
	d := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	w, h := panel.Size()
	panel.SetPosition(d.WorkArea.X+(d.WorkArea.Width-w)/2, d.WorkArea.Y+(d.WorkArea.Height-h)/3)
	app.query = ""
	app.input.Focus()
	panel.Show()
}
```

## The shortcut

```go
err := mygo.GlobalShortcut.Register("Alt+Shift+Space", func() {
	if panel.IsVisible() {
		panel.Hide()
	} else {
		show()
	}
})
```

Pick one other launchers leave free: Raycast and PowerToys Run take
Alt+Space, Spotlight Cmd+Space, and macOS gives a shortcut to every app
that registers it, so both would open. Let the user choose another in the
launcher's settings: `Unregister` the old one and `Register` the new one,
and tell them when `Register` fails because it is taken. On Windows the
key press lets the panel take the foreground; on Wayland the desktop binds
the shortcut, see [global shortcuts](native.md#global-shortcuts).

## Finding apps

`Shell.InstalledApps` lists the applications installed on the computer,
sorted by name, with the names the system shows in the user's language.
It reads the disk, in tens of milliseconds: list them on a goroutine as
the panel shows, and hand them to the view with `Window.Update` (or to the
page through a bound service):

```go
go func() {
	apps, err := mygo.Shell.InstalledApps()
	if err != nil {
		log.Print(err)
	}
	panel.Update(func() {
		app.apps = apps
		app.filter()
	})
}()
```

`Shell.FileIcon(app.Path, size)` draws an app's icon as a PNG, in pixels:
the size in DIPs times the scale factor of the display. It takes a few
milliseconds: draw the icons of the rows that show, and keep them by path.
`Shell.LaunchApp(app)` starts the chosen app, or brings it to the front;
hide the panel first, so that the app the user was in has the keyboard
until the launched one takes it.

## Platforms

| | macOS | Windows | Linux |
|---|---|---|---|
| no Dock icon, no taskbar button | `LSUIElement`, `ActivationPolicyAccessory` | a panel has none | a panel has none |
| the panel | a nonactivating `NSPanel`, over full-screen apps of regular and accessory apps alike | a topmost tool window | a utility window kept above on X11; a layer surface where Wayland compositors have one (with `libgtk-layer-shell.so.0`) |
| the shortcut | Carbon hot keys | `RegisterHotKey` | X11 grabs, the XDG desktop portal under Wayland |
| the apps | the Applications folders | the Start menu's All apps | desktop entries |

On Wayland, a panel that is a layer surface keeps the keyboard while it
shows, as launchers there do, so it is blurred only as it hides: close it
on Escape.
