# Windows

A window is a native window that shows a web page, loaded from its `URL`,
or [native UI](ui/README.md) that MyGo draws, from its `Content`. Create windows
with `mygo.NewWindow` once the app is ready:

```go
mygo.App.WhenReady(func() {
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "Editor",
		URL:       "/",
		Width:     1200,
		Height:    800,
		MinWidth:  600,
		MinHeight: 400,
		StateKey:  "editor",
	})
	win.OnClosed(func() { log.Println("closed") })
})
```

A window of native UI takes `Content` in place of `URL`:

```go
mygo.NewWindow(mygo.WindowOptions{Title: "Inspector", Content: ui.View(app.view)})
```

Everything in this guide applies to windows of both kinds, apart from
[pages](#pages), which only windows showing web pages have, and the
options about them. Like every MyGo API, windows can be created and
changed from any goroutine; calls hop to the main thread, which drives the
UI.

## Options

The zero `WindowOptions` is a visible, resizable window of 800×600 centered
on the screen. Sizes and positions are in device-independent pixels.

| Option | |
|---|---|
| `Title` | the window title; defaults to the app name, and a page's `<title>` replaces it |
| `URL` | loaded once the window exists; `/` and other URLs without a scheme are pages of the [frontend](frontend.md#how-pages-load) |
| `Content` | shows [native UI](ui/README.md) that MyGo draws, instead of a page |
| `Width`, `Height` | the size of the window, or of what it shows with `UseContentSize` |
| `X`, `Y` | the top-left corner; the window is centered when both are 0 |
| `MinWidth`, `MinHeight`, `MaxWidth`, `MaxHeight` | limits of resizing; 0 means none |
| `FitContent` | sizes the page area to its page's content, and follows it, within the limits above, as a [flyout's](#size-a-flyout-to-its-content) |
| `StateKey` | remembers the window's bounds and state under this key, see [below](#remember-where-windows-were) |
| `Hidden` | creates the window without showing it |
| `Maximized`, `FullScreen` | creates it maximized or in full screen |
| `Frameless` | no title bar and borders: the page draws them, see [custom title bars](frontend.md#custom-title-bars), or the [native UI](ui/windows.md) |
| `TitleBarStyle` | hides the title bar but keeps the window buttons over the page, see [custom title bars](frontend.md#custom-title-bars), or over [native UI](ui/windows.md) |
| `TrafficLightPosition` | moves the window buttons of a hidden title bar (macOS) |
| `TitleBarHeight` | the height of the title bar the page draws under the window buttons (Linux, Windows) |
| `HideWindowButtons` | hides the window buttons of a hidden title bar, which keeps its resizing top edge, shadow and corners: the page draws its own and calls `Minimize`, `Maximize`, `Unmaximize` and `Close` |
| `Transparent`, `Vibrancy` | a transparent window, and the material behind a transparent page or [native UI](ui/windows.md) (macOS, Windows 11 22H2): `VibrancyGlass` and `VibrancyGlassClear` are macOS 26's Liquid Glass. On Windows, a transparent window of native UI shows what is behind it where it draws nothing, and a window with a material has no menu bar: Alt and F10 open its menus in a popup |
| `VibrancyTint` | tints Liquid Glass, in CSS syntax such as `"rgba(0, 122, 255, 0.3)"` or `"light-dark(...)"`; other materials and platforms ignore it (macOS 26). `SetVibrancyTint` changes it later |
| `CornerRadius` | rounds a frameless window's corners: its content and material are cut to them on macOS, and Windows 11 gives it its own rounded corners |
| `BackgroundColor` | fills the window until the page paints, in CSS syntax such as `"#1e1e1e"`, or `"light-dark(#f5f5f7, #1e1e1e)"` to follow the appearance |
| `Opacity` | between 0 and 1 |
| `DisableResize`, `DisableMove`, `DisableMinimize`, `DisableMaximize`, `DisableClose`, `DisableFullScreen`, `DisableShadow` | take abilities away. On Linux under Wayland, `DisableShadow` keeps a frameless window or a hidden title bar without the shadow and border GTK draws around them |
| `AlwaysOnTop` | keeps the window above others |
| `SkipTaskbar` | leaves the window out of the taskbar (Linux, Windows) |
| `AutoHideMenuBar` | shows the menu bar only while the keyboard is in it, from Alt or F10 (Linux, Windows), see [the menu bar](menus.md#the-menu-bar) |
| `Parent`, `Modal` | a child window, modal to its parent, a [flyout](#a-dialog-over-a-flyout) too; for one placed next to a part of it, see [flyouts](#flyouts) |
| `Panel` | the window of a launcher or a quick entry, floating above all apps, see [panels](#panels) |
| `Page` | the options of the window's [page](#pages): its preload script, trusted origins, web inspector, zoom, user agent and its suffix, swipes and pinches |

## Show windows without flashing

A window showing a web page shows its `BackgroundColor`, white by
default, until the page paints, so make it the page's background. When the page follows dark mode,
give both colors with `light-dark(<light>, <dark>)`: the window switches
with the appearance. To show windows only once their page is ready, create
them hidden and show them from `OnReadyToShow`:

```go
win := mygo.NewWindow(mygo.WindowOptions{URL: "/", Hidden: true, BackgroundColor: "#1e1e1e"})
win.OnReadyToShow(win.Show)
```

Windows of native UI need neither: they have no page to load, and show
with their first frame (on Windows, the window stays out of sight until
it drew it). One created `Hidden` is ready to show once it drew its first
frame, which it draws while hidden, so `OnReadyToShow(win.Show)` shows it
as it does a page's.

The view of native UI first runs once `NewWindow` returned, so it can use
the window `NewWindow` returns:

```go
app.win = mygo.NewWindow(mygo.WindowOptions{Content: ui.View(func(c *ui.Context) {
	ui.Text(c, app.win.Title()) // never nil
})})
```

Called on the main thread, as in `App.WhenReady` or an event handler,
`NewWindow` leaves showing the window to the main loop's next turn, when
the window draws its first frame, or to the first method that acts on it,
such as `IsVisible` or `Maximize`, which finds it shown. `OnShow` added
right after `NewWindow` hears that show, and `Hide` keeps it from
happening. Called from another goroutine, `NewWindow` shows the window and
returns while the main thread goes on drawing it: guard the variable
there, or set it with `Window.Update`.

## Remember where windows were

With a `StateKey`, a window remembers its position, size, and maximized and
full screen state in `window-state.json` in the user data directory, which
is written when the window closes and when the app quits. The next window
created with the same key, at the next launch for example, gets them back,
as long as it would show on a connected display. They override `X`, `Y`,
`Width`, `Height`, `UseContentSize`, `Maximized` and `FullScreen`, which
stay the defaults of the first launch. Give each kind of window its own key:
`"main"`, `"preferences"`.

On Windows, a state also keeps the window's bounds in the screen's pixels
(`PixelX`, `PixelY`, `PixelWidth`, `PixelHeight`): where displays have
different scales, its position in DIPs changes with the scales, and
earlier versions' cannot tell the displays apart.

An app that keeps its data elsewhere, such as in its own database, keeps
the states there with a `WindowStateStore`, set before the first window
with a `StateKey`, and no `window-state.json` is written. It keeps every
field of a `WindowState`, as JSON does: on Windows, a window whose state
lost its pixels is centered when the displays have different scales.

```go
type windowStates struct{ db *sql.DB }

func (s windowStates) LoadWindowStates() (map[string]mygo.WindowState, error) {
	var data []byte
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = 'windows'`).Scan(&data)
	if err != nil {
		return nil, err
	}
	var states map[string]mygo.WindowState
	return states, json.Unmarshal(data, &states)
}

func (s windowStates) SaveWindowStates(states map[string]mygo.WindowState) error {
	data, err := json.Marshal(states)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES ('windows', ?)`, data)
	return err
}

mygo.App.SetWindowStateStore(windowStates{db})
```

MyGo loads the states once, when the first window with a `StateKey` is
created, and saves them all when such a window closes and when the app
quits, on the main thread. Windows open with their options when there are
no states or loading them fails; a save that fails is tried again the next
time.

### Bring a page back

A window's position comes back with its `StateKey`; its page comes back
with its session state: `Page.SessionState` saves the back/forward list,
with the scroll positions and form fields the engine saved, and
`PageOptions.SessionState` brings it back in a new window or web view, in
place of its `URL`, Back included. An app keeps the state where it keeps
its data, to bring the page back at the next launch, or to close a page it
isn't showing and bring it back later, as a browser's sleeping tabs:

```go
win.OnClose(func(e *mygo.CloseEvent) {
	if state, err := win.Page().SessionState(); err == nil {
		os.WriteFile(sessionFile, state, 0o600)
	}
})

// At the next launch:
state, _ := os.ReadFile(sessionFile)
win := mygo.NewWindow(mygo.WindowOptions{StateKey: "main", URL: "/", Page: mygo.PageOptions{SessionState: state}})
```

The state is opaque bytes in a format MyGo versions: the page's URL and the
engine's own state. A state the platform can't read, saved on another
platform or by a web engine that encodes it otherwise, brings back the page
it was at alone, and one that is no state at all (a missing file) loads
`URL`. It may hold what the user typed into forms: keep it as the user's
data. The HTML that `LoadHTML` loaded is not in it.

WebKit saves the scroll positions and form fields as it goes: the scroll
position of the page it shows a moment after the user stops scrolling
(WebKitGTK 2.38 and Safari 16's WebKit or later), its form fields only as
it goes out of view (its window hidden, for one), and the other entries'
scroll positions as the page leaves them. A state saved since has them, so a page saved
while it shows comes back at its scroll position, but with the form fields
it had when it last went out of view.

| Platform | Session state |
|---|---|
| macOS | `WKWebView`'s `interactionState`, macOS 12 or later; the page's URL alone before |
| Linux | `webkit_web_view_get_session_state`, serialized, and `webkit_web_view_restore_session_state` |
| Windows | the page's URL alone: WebView2 has no API for the back/forward list |

## Size, position and state

```go
win.SetSize(1024, 768)
win.SetContentSize(1024, 700) // the page area
win.SetPosition(100, 100)
win.SetBounds(mygo.Rectangle{X: 100, Y: 100, Width: 1024, Height: 768})
win.Center()

win.Maximize()          // Unmaximize, ToggleMaximize, IsMaximized
win.Minimize()          // Restore, IsMinimized
win.SetFullScreen(true) // ToggleFullScreen, IsFullScreen
win.Hide()              // Show, ShowInactive (without focusing), IsVisible
win.Focus()             // Blur, IsFocused
```

Sizes and positions are in the window's DIPs, whatever the zoom of its
page (`Page.SetZoomFactor`) or of its [native UI](ui/windows.md#zoom):
at a zoom of 1.25, a page area of 1000 DIPs shows 800 of the page's CSS
pixels, or of the view's DIPs. `Bounds`, `ContentBounds`, `Size`,
`ContentSize` and `Position` read them back;
[`mygo.Screen`](native.md#displays) tells where the displays are.
`Bounds` are the whole window's, title bar and borders included (on Linux
X11, the window manager's frame), and `ContentBounds` those of the area the
page or native UI shows in, below a menu bar. On Linux under Wayland the
compositor decides where windows go and ignores the positions apps set,
and its decorations are not part of `Bounds`.

Other properties:

- `SetTitle`, `SetResizable`, `SetMovable`, `SetMinimizable`,
  `SetMaximizable`, `SetClosable`, `SetMinimumSize`, `SetMaximumSize`,
  `SetAlwaysOnTop`, `SetOpacity`, `SetHasShadow`, `SetBackgroundColor` and
  `SetVibrancy` change what the options set. On Windows, `SetVibrancy`
  changes the material of a window created with one, and shows none in
  other windows. On Linux, a window keeps the shadow it was created with
  (`DisableShadow`).
- `SetIcon(png)` gives the window its own icon in its title bar and taskbar
  button (Linux, Windows).
- `SetSkipTaskbar` hides the window from the taskbar (Linux, Windows).
- `SetAutoHideMenuBar` hides the menu bar until Alt or F10, or shows it
  for good again (Linux, Windows).
- `SetVisibleOnAllWorkspaces` shows it on every Space or virtual desktop
  (macOS, Linux); on macOS also over other apps in full screen, as an
  overlay does, even with `DisableFullScreen`, if the app is an accessory
  (`ActivationPolicyAccessory`, `App.Dock.Hide`): macOS shows no windows of
  regular apps there, but [panels](#panels).
- `SetContentProtection(true)` keeps it out of screenshots and screen
  recordings.
- `SetIgnoreMouseEvents(true)` lets clicks through to what is behind it.
- `SetActiveAppearance(true)` keeps the window drawn as active while
  another app is in front (macOS), so that the clear Liquid Glass
  (`VibrancyGlassClear`) of a panel floating over it stays clear. It uses a
  private AppKit call, which a later macOS may drop: apps for the Mac App
  Store should leave it off. Its material stays active then as well, and
  its pages and native UI too, on every platform (see
  [inactive windows](#active-and-inactive-windows)).
- `SetWindowButtonsTheme(mygo.ThemeDark)` draws the window buttons of a
  hidden title bar for a dark background in a light app, as over a dark
  view at the top of the page, and `ThemeLight` for a light one;
  `ThemeSystem` follows the app again. The page and the menus keep the
  app's appearance (Windows; the buttons of macOS and Linux keep it).
- `SetProgressBar` shows the progress of a task on the taskbar button
  (Windows), the Dock icon (macOS) or the launcher entry (Linux docks that
  support it):

  ```go
  win.SetProgressBar(mygo.ProgressBar{Value: 0.4})
  win.SetProgressBar(mygo.ProgressBar{State: mygo.ProgressIndeterminate})
  win.SetProgressBar(mygo.ProgressBar{}) // done
  ```

- `FlashFrame(true)` asks for attention: the taskbar button flashes
  (Windows), the window is marked urgent (Linux) or the Dock icon bounces
  (macOS).

## Active and inactive windows

A window is drawn as active while the user's input goes to it: while it,
or one of its flyouts, has the keyboard. Native apps draw the others apart
so that users tell at a glance which window has it, and MyGo's do too:

- The material (`Vibrancy`) of a window fades to an opaque gray on macOS
  while it is inactive, as AppKit's sidebars do, and Windows 11 draws
  its Mica or Acrylic as a solid fill. Panels and flyouts keep theirs
  active, floating over whatever app is in front, as does a window
  `SetActiveAppearance(true)` keeps active.
- Pages get the `data-mygo-window-inactive` attribute on `<html>` while
  their window is inactive, for CSS, and `windowActive()` of
  `mygo-runtime` (see [inactive windows](frontend.md#inactive-windows)).
- Native UI reads `c.WindowActive()`, and its built-in widgets fade their
  accent to gray (see [native UI](ui/windows.md#active-and-inactive-windows)).

A window's focusable flyout taking the keyboard leaves the window active:
on Windows its title bar and Mica stay active too. A window that never had
the keyboard counts as active, so that the first frame of a window about
to take it is drawn active; one shown with `ShowInactive` is not. Panels,
and flyouts without a parent, are always active.

## Child and modal windows

A window with a `Parent` stays above it and closes with it. It can go to
any display, and stays where it is as its parent moves, on every platform.
On macOS it is ordered above its parent whenever the parent comes in
front, stays above a parent that is `AlwaysOnTop`, goes with it into
full screen, and hides while the parent is minimized or hidden. With `Modal`,
it also blocks its parent until it closes, like a dialog:

```go
prefs := mygo.NewWindow(mygo.WindowOptions{
	URL:    "/preferences",
	Parent: win,
	Modal:  true,
	Width:  500,
	Height: 400,
})
```

On macOS a modal window is a sheet of its parent. Its parent may be a
flyout too: see [a dialog over a flyout](#a-dialog-over-a-flyout).

## Flyouts

A flyout is a window without a frame, a background or a taskbar button,
owned by its parent and placed next to a rectangle of it, its anchor: the
list of a combo box, a hover card, a popover with a form. Unlike an
[overlay](ui/overlays.md) of native UI or a popover of the page, it may
extend beyond the parent's edges, and the work area of the display decides
where it goes, not the parent's content. Its page or [native UI](ui/README.md)
draws all of it, its background included:

```go
fl := mygo.NewFlyout(mygo.FlyoutOptions{
	Parent:    win,
	Anchor:    mygo.Rectangle{X: 20, Y: 40, Width: 120, Height: 28}, // in win's content area
	Placement: mygo.PlacementBottomStart,
	Gap:       4,
	Width:     240,
	Height:    300,
	Content:   ui.View(app.suggestions),
})
```

The anchor is in DIPs relative to the parent's content area, which is
what `ui.Element.Bounds` gives for the element that opens the flyout; a
rectangle of zero size anchors it to a point. `Placement` names the side
of the anchor the flyout goes to and how it lines up with it along that
side, as `"bottom-start"`, `"right"` or `"top-end"`. Where the work area
has no room for it there, the flyout flips to the other side, or the other
end, if that has room, then slides back into the work area, then shrinks
to fit it, as GTK places menus and Wayland compositors place popups. It
stays above its parent, follows it as it moves and closes with it.
`SetAnchor` places it against another rectangle, and `SetSize` resizes it
where it goes; `IsFlyout` tells a flyout from other windows.

With `Focusable`, a flyout takes the keyboard as it shows and closes as a
menu does when the user presses outside it, another window takes the
keyboard or the app is deactivated. `OnClose` listeners hear it and may
keep it open, except on Wayland, whose compositor dismisses it. On Linux
and Windows the press only closes the flyout, as it closes their menus;
on macOS it goes on to what is under the pointer, but on the anchor, as
AppKit's popovers. So a click on the button that opened it closes it, and
does not open it again. Menus, lists to choose from and popovers are
focusable, and handle their own keys: the arrows, Enter.

Without it, a flyout never takes the keyboard, which stays in the parent,
and it stays until it is closed: a tooltip, a hover card, or the
suggestions of a text field in the parent, whose view drives them and
closes them as it sees fit, as when the field loses the keyboard. (On
Wayland, only a popup that takes the keyboard hears of presses outside
it.)

Either closes, as the user closing it, on an Escape its content leaves:
one no element or shortcut of native UI takes, or no handler of the page
cancels with `preventDefault()`, as the web's popovers.

`Shadow` gives it the system's window shadow on macOS, where it follows
the shape of what the flyout draws and outlines it with a hairline (so
draw no border of your own there), and on Windows, where Windows 11 rounds
the flyout's corners as it rounds its menus and outlines it (so draw
neither a border nor rounded corners there).

On macOS, `Popover` gives it the look of the system's popovers instead: it
shows in an `NSPopover`, which draws its material, rounded corners and an
arrow pointing at the middle of the anchor, and animates. AppKit places it
on the side `Placement` names, or the opposite one; the rest of the
placement, and `Gap`, do not apply. As AppKit's popovers do, it has the
keyboard while it shows, and with `Focusable` it closes as the user clicks
elsewhere. Its content draws no background over the material: in native
UI, `c.Vibrancy()` reports it, so a view draws its own panel elsewhere:

```go
func (app *App) about(c *ui.Context) {
	c.Root().Background(ui.Transparent)
	box := ui.Column(c).Fill() // over the popover's material
	if !c.Vibrancy() {
		box = ui.Column(c).Fill().Background(c.Theme().Surface).Radius(10)
	}
	box.Padding(12).Children(func() { ui.Text(c, "MyGo") })
}
```

A flyout can hang from a tray icon instead of a window, as the panel of
a menu bar app: give it the `Tray` in place of a `Parent`. It goes next to
the icon, on the side `Placement` names, and floats over the windows of
other apps. Below the macOS menu bar, `PlacementBottom` centers it under
the icon (with `Popover`, the arrow of the system's popover points at
it); on Windows, the same placement flips above an icon of a taskbar at
the bottom of the screen. On macOS, which shows a menu bar item on each
display, it goes under the copy of the icon the user clicked. A press on
the icon does not dismiss a focusable flyout, so the icon's click can
toggle it; with `Popover`, the click closes it, as AppKit's popovers
close on their anchor, and does not reach `OnClick`. Either way, a click
on the icon closes the open flyout with the code below, and does not
open it again. Keep the app running as the flyout, its only window,
closes:

```go
mygo.App.OnWindowAllClosed(func() {})
tray.OnClick(func() {
	if panel != nil {
		panel.Close()
		return
	}
	panel = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      tray,
		Placement: mygo.PlacementBottom,
		Width:     280,
		Height:    230,
		Focusable: true,
		Popover:   true,
		Content:   ui.View(app.panel),
	})
	panel.OnClosed(func() { panel = nil })
})
```

Linux's tray icons tell neither clicks nor where they are: open the flyout
from a menu item of the icon there, and it shows in the middle of the
primary display's work area. `examples/menubar-flyout` is such an app.

### A dialog over a flyout

A [dialog](ui/dialog.md) of native UI draws in its window, so a flyout's is
cut to the flyout, and a tray's flyout may have no other window that shows.
Ask from a window modal to the flyout instead: give it the flyout as its
`Parent`, with `Modal`. Without `X` and `Y`, it is centered over the
flyout, moved into the work area of its display, and it stays above the
flyout, over other apps' windows too for a tray's. The flyout takes no
input while it shows, and stays open wherever the keyboard goes: a
focusable flyout gets the keyboard back as the window closes, and from
then on closes as it loses it again. Closing the flyout, as a click on
the tray icon may, closes the window.

```go
if ui.Button(c, "Quit…").Clicked() {
	mygo.NewWindow(mygo.WindowOptions{
		Title:          "Quit?",
		Parent:         panel,
		Modal:          true,
		Width:          360,
		Height:         140,
		UseContentSize: true,
		DisableResize:  true,
		Content:        ui.View(app.confirmQuit),
	})
}
```

On macOS the window is no sheet of the flyout, which a sheet would hang
from, but a window attached to it, at its level. On Wayland, where a
flyout with a `Parent` is a popup, the compositor decides whether the
popup stays as the window takes the keyboard; a tray's flyout is a
window there, and stays.

### Size a flyout to its content

With `FitContent`, a flyout takes the size of its page's content and
follows it as it changes, as browsers size the popups of their
extensions: a menu bar popover with a list, a hover card, a launcher's
results. `Width` and `Height` are its size only until the page tells its
own, which it does before its DOM is ready, so a flyout created `Hidden`
and shown from `OnReadyToShow` shows at its size. `MinWidth`, `MinHeight`,
`MaxWidth` and `MaxHeight` bound it, and the work area, as any flyout:

```go
fl := mygo.NewFlyout(mygo.FlyoutOptions{
	Tray:       tray,
	Placement:  mygo.PlacementBottom,
	FitContent: true,
	MaxWidth:   400,
	MaxHeight:  600,
	URL:        "/panel",
})
```

MyGo's runtime in the page lays its root element (`html`) out at the
width of its content (`width: max-content`), within the limits, so that
its text wraps at `MaxWidth`, and watches its size with a
`ResizeObserver`: its height is its content's, the body's margins
included, beyond `MaxHeight` the page scrolls, and the room its
scrollbar takes then is part of the size. A page that gives `html` a size of its own,
as `height: 100%`, keeps the flyout at that size: let the root and the
body take their content's. The size is the page's CSS pixels times its
zoom, in DIPs, and a size that changes nothing resizes nothing.

A flyout of native UI (`Content`) fits the same way, a tray panel whose
list of nodes, form or error line changes as the app runs, with no size
to guess and no `SetSize` to call:

```go
fl := mygo.NewFlyout(mygo.FlyoutOptions{
	Tray:       tray,
	Placement:  mygo.PlacementBottom,
	FitContent: true,
	MinWidth:   260,
	MaxHeight:  480,
	Content: ui.View(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
			ui.Text(c, "Nodes")
			ui.Scroll(c).Grow(1).Children(func() { app.nodes(c) })
			if app.err != nil {
				ui.Text(c, app.err.Error())
			}
		})
	}),
})
```

Each frame measures its root's content as it lays it out, as CSS sizes
`width: fit-content; height: auto`: its widest line of children
(max-content), within `MinWidth` and `MaxWidth` and the work area, and
its height at that width, within `MinHeight` and `MaxHeight`, beyond
which a `Scroll` in it scrolls: `Fill` and `Grow` take the flyout's size
as it lays out, and measure as their content. Elements placed `Absolute`, as menus and
toasts, take none of it. A frame that measures another size resizes the
flyout, placed again against its anchor (flipping where the work area
has no room), and the next frame lays the content out at that size: the
first frame after a change shows it at the size the flyout had. A
hidden flyout draws no frames: content changed while it was hidden
(with `Update` or `Invalidate`) lays out as `Show` or `ShowInactive`
shows it, and the flyout shows at the size that content takes. A frame
that measures what the last did resizes nothing, so a size `SetSize`
gives holds until the content's changes. A view that builds its content
after the window's size (`Context.Size`) settles at the size it took
rather than resize back and forth.

`WindowOptions.FitContent` does the same for the page area of a window,
a [panel](#panels) for one, within its `MinWidth`, `MinHeight`,
`MaxWidth` and `MaxHeight` and the work area of its display. It follows
the content only when the content's size changes, and only in the
dimension that changed, so a size the user gives a resizable window holds
until then, and not while the window is maximized or in full screen.

How each platform shows them:

| | macOS | Linux | Windows |
|---|---|---|---|
| window | a borderless, nonactivating `NSPanel`, a child window of its parent while it shows, or at the level of menus for a tray | a `GTK_WINDOW_POPUP`: override-redirect on X11, an `xdg_popup` of its parent on Wayland; for a tray, an undecorated utility window kept above | an owned `WS_POPUP`, `WS_EX_TOOLWINDOW`, without a redirection bitmap; topmost and unowned for a tray |
| placement | in the screen's visible frame | in the monitor's work area on X11; the compositor's positioner (`gdk_window_move_to_rect`) on Wayland | in the monitor's work area (`MonitorFromRect`) |
| keyboard | the key window when focusable; the parent stays the main window | a grab of the seat and the app's input when focusable, as GTK's menus | activated when focusable, else `WS_EX_NOACTIVATE` |

On Wayland, a popup shows only over the topmost one: showing a flyout
closes the others that show, but its own ancestors, as the user would.
`examples/flyout` opens a list of colors, a note editor and an About box,
a popover on macOS.

## Panels

A panel is the window of a launcher, a command palette or a quick entry,
as Spotlight's and Raycast's are: summoned from anywhere by a
[global shortcut](native.md#global-shortcuts), it floats above the windows
of every app, on whichever Space or virtual desktop the user is on, over
apps in full screen too, and stays out of the taskbar, the Dock's windows
and the app switcher. It cannot be minimized, maximized or go full screen,
and has no menu bar on Linux and Windows. On macOS, showing it gives it the
keyboard without activating the app: the app the user was in stays the
active one, as its menu bar shows, and has the keyboard back as the panel
hides. Clicks in the panel leave it active too.

```go
mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory) // no Dock icon (macOS)
mygo.App.OnWindowAllClosed(func() {})                     // keep running with no window

mygo.App.WhenReady(func() {
	panel := mygo.NewWindow(mygo.WindowOptions{
		Panel:       true,
		Hidden:      true,
		Frameless:   true,
		Transparent: true,
		Vibrancy:    mygo.VibrancyHUD,
		Width:       680,
		Height:      420,
		URL:         "/launcher",
	})
	// Hide rather than close, so that it shows at once the next time:
	// on Escape, Cmd-W or Alt-F4, and as the user goes elsewhere.
	panel.OnClose(func(e *mygo.CloseEvent) {
		e.PreventDefault()
		panel.Hide()
	})
	panel.OnBlur(panel.Hide)

	mygo.GlobalShortcut.Register("Alt+Shift+Space", func() {
		if panel.IsVisible() {
			panel.Hide()
			return
		}
		// On the display with the pointer, a third of the way down.
		d := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
		width, height := panel.Size()
		panel.SetPosition(d.WorkArea.X+(d.WorkArea.Width-width)/2, d.WorkArea.Y+(d.WorkArea.Height-height)/3)
		panel.Show()
	})
})
```

A panel whose results grow and shrink, as a launcher's, takes its page's
size with `FitContent`, as [a flyout](#size-a-flyout-to-its-content)
does: give it `MaxHeight` rather than a `Height` to pick.

An Escape the page does not handle (`preventDefault()`) or native UI
leaves closes a panel, as the user closing it, which `OnClose` listeners
may turn into hiding it, as above. A launcher is usually an accessory app,
without a Dock icon, from the start: give it `LSUIElement` in
`macos.infoPlist` of [`mygo.config.ts`](configuration.md), or set its
activation policy before `App.Run`, as above. A regular app is activated
as it launches, as any app the user opens: the first window it shows,
a panel too, brings it to the front.

How each platform shows them:

| | macOS | Linux | Windows |
|---|---|---|---|
| window | an `NSPanel` with the nonactivating style, never the main window | a utility window (`_NET_WM_WINDOW_TYPE_UTILITY`), kept above, out of the taskbar and the pager; a surface of the overlay layer under Wayland (gtk-layer-shell) | a topmost tool window (`WS_EX_TOOLWINDOW`), owned by a hidden window, so it never has a taskbar button, with Windows 11's rounded corners |
| workspaces | every Space, as a full-screen auxiliary (`CanJoinAllSpaces`, `FullScreenAuxiliary`), out of Mission Control and the window cycle | every workspace (sticky) | the virtual desktop it shows on |
| keyboard | the key window, the app staying inactive | the active window; on a layer surface, exclusively while it shows | the foreground window, which the key press of a global shortcut lets it take |

Under Wayland, where apps neither place nor raise their windows, a panel
is a surface of the layer shell where the compositor speaks the protocol
(Sway, Hyprland, KDE Plasma, COSMIC…) and `libgtk-layer-shell.so.0` is
installed, as launchers there are: it shows on the overlay layer, above
windows in full screen too, out of the taskbar, centered on the output
the user is on, or where `X` and `Y` or `SetPosition` put it. It keeps the
keyboard while it shows, as other launchers there do, so that it gets the
keys from the start: it is blurred only as it hides, so close it on
Escape, which the user presses there. Elsewhere, as on GNOME, which has no
layer shell, it is a window as others: the compositor decides where it
shows and what is above it.

## Events

Listeners run on the main thread: keep them short, and start goroutines for
slow work. Each `On…` method returns a function that removes the listener.

| Event | When |
|---|---|
| `OnClose(func(e *mygo.CloseEvent))` | the window is about to close, by the user or `Close`; `e.PreventDefault()` keeps it open |
| `OnClosed` | the window closed |
| `OnFocus`, `OnBlur` | it gained or lost the keyboard focus |
| `OnShow`, `OnHide` | it was shown or hidden |
| `OnResize`, `OnMove` | it was resized or moved |
| `OnMaximize`, `OnUnmaximize`, `OnMinimize`, `OnRestore` | its state changed |
| `OnEnterFullScreen`, `OnLeaveFullScreen` | it entered or left full screen |
| `OnReadyToShow` | its first page is ready to be displayed, or its native UI drew its first frame |
| `OnFileDrop` | files were dropped on its page, see [dropped files](frontend.md#dropped-files), or on native UI that did not take them, see [dropped files](ui/input.md#dropped-files) |

Ask before closing a window with unsaved changes:

```go
win.OnClose(func(e *mygo.CloseEvent) {
	if !doc.Dirty() {
		return
	}
	e.PreventDefault()
	go func() {
		res, _ := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:  win,
			Type:    mygo.MessageQuestion,
			Message: "Save changes before closing?",
			Buttons: []string{"Save", "Don't Save", "Cancel"},
		})
		switch res.Button {
		case 0:
			doc.Save()
			win.Destroy()
		case 1:
			win.Destroy()
		}
	}()
})
```

`Close` closes a window as the user would, asking `OnClose` listeners;
`Destroy` closes it at once.

## Pages

A window shows one page at a time and has a history, like a browser tab.
`win.Page()` returns it, or nil for a window that shows
[native UI](ui/README.md):

```go
page := win.Page()
page.LoadURL("/settings") // LoadHTML, LoadFile
page.Reload()             // ReloadIgnoringCache, Stop
page.GoBack()             // GoForward, CanGoBack, CanGoForward
page.URL()                // the current page
page.SetZoomFactor(1.25)  // ZoomFactor
```

`WindowOptions.Page` sets up the page:

```go
mygo.NewWindow(mygo.WindowOptions{
	URL:  "/",
	Page: mygo.PageOptions{ZoomFactor: 1.25, DevTools: mygo.DevToolsEnabled},
})
```

| Option | |
|---|---|
| `PreloadScript` | JavaScript run before every page's own scripts |
| `PreloadAllFrames` | runs `PreloadScript` in the page's iframes too, of every origin, where `window.mygo` isn't |
| `TrustedOrigins` | other origins whose pages may call Go, see [who may call](bindings.md#who-may-call) |
| `DevTools` | `DevToolsAuto` (the inspector in development builds), `DevToolsEnabled` or `DevToolsDisabled`, which also turns WebView2's own keys, as F5 reloading, on or off (Windows); for a window of native UI, its [inspector](ui/inspector.md) |
| `ZoomFactor`, `UserAgent` | the page's zoom (1 is 100%) and user agent |
| `SafariTabFocus` | leaves Tab to text fields and elements with a tabindex, as Safari does unless the system's keyboard navigation is on; by default Tab reaches links, buttons, checkboxes and selects too (macOS) |
| `PictureInPicture` | lets the page's videos go into picture in picture, see [picture in picture](#picture-in-picture) (macOS; WebView2 lets them already) |
| `SwipeNavigation` | a horizontal swipe goes back and forward through the page's history, as in browsers, see [gestures](#swipes-and-pinches) |
| `PinchZoom` | a pinch magnifies the page, as in browsers, see [gestures](#swipes-and-pinches) |
| `UserAgentSuffix` | appended to the user agent instead of replacing it, which `UserAgent` does, such as `"MyApp/1.2"`; it appends to `UserAgent` too, and `SetUserAgent` keeps it, and leading and trailing spaces are trimmed |
| `ContentRules` | rules that block what the page loads and hide parts of it, see [blocking content](#blocking-content) |
| `BackgroundThrottling` | `ThrottlingOn` (the web view's own) or `ThrottlingOff`, which keeps the page running while its window is hidden, minimized or covered, see [background pages](#background-pages) |
| `Autoplay` | `AutoplayUserGesture` (the web view's own policy) or `AutoplayAllow`, which lets the page play audible media before the user clicked or typed in it, in a hidden window too, see [autoplay](#autoplay) |
| `Session` | the store of the page's cookies, storage and caches, apart from other sessions', see [sessions](app.md#sessions); nil is the default session |

`UserAgentSuffix` is for the sites that want to see the app, and for the ones
that serve a bare web view older pages because its user agent names no browser:
`"Version/17.0 Safari/605.1.15"` makes the page look like Safari.

Page events, on `win.Page()`:

| Event | When |
|---|---|
| `OnWillNavigate(func(e *mygo.NavigateEvent))` | the page is about to navigate, for example after a click on a link; `e.PreventDefault()` cancels it. `e.UserInitiated` says whether a user's gesture started it, a click on a link or a click whose handler sets `location.href`, or the page's script did on its own. Not called for `LoadURL` and its siblings, nor for the page's iframes, except on Linux before WebKitGTK 2.56, which tells no navigation's frame: there it is called for an iframe's too, but a `srcdoc` one's. |
| `OnDidNavigate(func(url string))` | a navigation committed |
| `OnDOMReady` | the page's DOM is ready (`DOMContentLoaded`) |
| `OnDidFinishLoad` | the page finished loading |
| `OnDidFailLoad(func(err *mygo.LoadError))` | the page failed to load |
| `OnPageTitleUpdated(func(e *mygo.TitleEvent))` | the page's `<title>` changed; preventing the event keeps the window title as it is |
| `OnThemeColorChanged(func(c color.NRGBA, ok bool))` | the page's [theme color](#the-theme-color) changed; `ok` is false when it has none any more |
| `OnRenderProcessGone(func(reason string))` | the web content process crashed or was killed; `Reload` recovers |
| `OnAudibleChanged(func(audible bool))` | the page started or stopped [making sound](#sound) |
| `OnCaptureStateChanged(func(kind mygo.CaptureKind, s mygo.CaptureState))` | the page started or stopped [capturing](#camera-and-microphone) the camera, the microphone or the display, or the capture was muted or unmuted |

Keep a window on the app's own pages and open other links in the browser:

```go
win.Page().OnWillNavigate(func(e *mygo.NavigateEvent) {
	if u, err := url.Parse(e.URL); err == nil && (u.Scheme == "http" || u.Scheme == "https") && e.UserInitiated {
		e.PreventDefault()
		go mygo.Shell.OpenExternal(e.URL)
	}
})
```

During development the app's pages come from the dev server, an `http` URL:
check the host, or `mygo.IsDev()`, when your rule depends on the scheme.

Links that open new windows go through the window's open handler, see
[links and new windows](frontend.md#links-and-new-windows).

### Swipes and pinches

A page neither goes back on a swipe nor zooms on a pinch, as desktop apps
don't: `SwipeNavigation` and `PinchZoom` turn them on, for a window's page
and a [web view's](ui/web-view.md) alike.

```go
mygo.NewWindow(mygo.WindowOptions{
	URL:  "https://example.com/",
	Page: mygo.PageOptions{SwipeNavigation: true, PinchZoom: true},
})
```

A pinch magnifies what the page shows without laying it out again, as
browsers do, and leaves `ZoomFactor`, which lays the page out at another
size, as it is.

| | `SwipeNavigation` | `PinchZoom` |
|---|---|---|
| macOS | a trackpad's or a Magic Mouse's swipe between pages, as System Settings sets it up (`allowsBackForwardNavigationGestures`) | a trackpad's pinch and smart zoom (`allowsMagnification`) |
| Linux | two fingers across a touchpad, one across a touchscreen (`enable-back-forward-navigation-gestures`) | a touchscreen's pinch, and a touchpad's, which GTK reports on Wayland only. WebKitGTK zooms with no setting to stop it: MyGo turns its zoom gesture off from WebKitGTK 2.38, and older versions always zoom |
| Windows | a swipe across a touchscreen (`IsSwipeNavigationEnabled`) | a touchscreen's pinch (`IsPinchZoomEnabled`); a WebView2 Runtime without `ICoreWebView2Settings5` always zooms |

WebKitGTK and WebView2 have pinches, and WebView2 swipes, on by default:
MyGo turns them off unless the page asks for them.

### The theme color

`ThemeColor` returns the page's theme color, its `<meta name="theme-color">`,
so that the app can tint its title bar, toolbar or window background to
match the page, as Safari does; `OnThemeColorChanged` tells when it
changes:

```go
page := win.Page()
page.OnThemeColorChanged(func(c color.NRGBA, ok bool) {
	if !ok {
		c = defaultToolbarColor // the page has none
	}
	toolbar.SetBackground(ui.RGB(c.R, c.G, c.B))
})
```

A page may have one color for light appearance and another for dark, with
the metas' `media`:

```html
<meta name="theme-color" content="#ffffff" media="(prefers-color-scheme: light)">
<meta name="theme-color" content="#111111" media="(prefers-color-scheme: dark)">
```

The color is the content of the first meta named `theme-color` in the
page's `<head>` whose `media` is absent or matches, and whose content is a
color, as browsers pick it. The page's engine computes it, so any CSS color
works (`dodgerblue`, `rgb(30 144 255 / 50%)`, `oklch(…)`), and it comes as
8-bit sRGB with straight alpha. The event fires only when the color
changes: when the page adds, removes or changes its metas, when the app's
appearance ([`Theme`](native.md#dark-mode)) changes and another meta's
`media` matches, and when another page has another color. A page that just started loading
keeps the color of the one before until its DOM is ready, so going between
pages of the same color fires nothing.

MyGo's runtime in the page watches the metas, the same way on every
platform, in pages of any origin, trusted or not. A page the runtime does
not run in, as an image or a PDF the web view shows, has no theme color.

### Sound

`IsAudible` tells whether a page is making sound, from any of its frames,
a video of another site included, and `OnAudibleChanged` when that
changes, for a speaker on a tab or a window, or a media app's state.
`SetAudioMuted` mutes the whole page, as a browser mutes a tab, without the
page seeing its elements muted; it lasts as the page navigates:

```go
page := win.Page()
page.OnAudibleChanged(func(audible bool) {
	log.Println("playing:", audible) // show a speaker on the window's tab
})

// A menu item that mutes or unmutes the window it was chosen for.
mute := &mygo.MenuItem{Label: "Mute or Unmute Window", Click: func(_ *mygo.MenuItem, win *mygo.Window) {
	if win == nil || win.Page() == nil {
		return // no window, or one showing native UI
	}
	win.Page().SetAudioMuted(!win.Page().IsAudioMuted())
}}
```

A page muted right after `NewWindow` starts muted, even where its web view
is not made yet. A muted page can still be audible, as a tab shows a
crossed-out speaker; the engine decides when a paused or silent source stops
counting, which can take a moment after the sound stops.

| Platform | Making sound | Muting |
|---|---|---|
| macOS | WebKit's private `_isPlayingAudio`, observed; false if WebKit stops answering it | WebKit's private `_setPageMuted:`, as Safari mutes a tab; does nothing if WebKit stops answering it |
| Windows | `IsDocumentPlayingAudio` of WebView2 (`ICoreWebView2_8`); false on runtimes older than it | `IsMuted` of the same; does nothing on older runtimes |
| Linux | `webkit_web_view_is_playing_audio` | `webkit_web_view_set_is_muted`, WebKitGTK 2.30 or later; does nothing on older versions |

Where a platform can't mute, `IsAudioMuted` is false. Both are false for a
window showing native UI and after the window closed. A [web view's](ui/web-view.md)
page has them too.

### Background pages

Web views slow down a page nobody can see, to save power: once its window is
hidden, minimized or behind others, its timers fire once a second at most,
`requestAnimationFrame` waits, and macOS may suspend it. A page that plays,
syncs or computes in the background keeps running with `ThrottlingOff`, as
Electron's `backgroundThrottling: false`:

```go
mygo.NewWindow(mygo.WindowOptions{
	URL:  "/player",
	Page: mygo.PageOptions{BackgroundThrottling: mygo.ThrottlingOff},
})
```

Its timers keep their rate and its process the priority of a visible page's,
in windows and in [web views](ui/web-view.md) alike. A covered window's page keeps
its animation frames too on macOS and Windows, but the page of a hidden or
minimized window still goes hidden (`document.visibilityState`) on macOS and
Linux, and so does a web view of a hidden window on Windows, which draws
nothing then: run what must go on with timers.

| | |
|---|---|
| macOS | `WKPreferences.inactiveSchedulingPolicy` none (macOS 14+), WebKit's private hidden-page timer throttling, CSS animation suspension and process suppression off, and window occlusion detection off, so that a covered window counts as visible |
| Linux | WebKitGTK's `HiddenPageDOMTimerThrottling` and `HiddenPageCSSAnimationSuspension` features off (WebKitGTK 2.42+; nothing before) |
| Windows | its browser runs with `--disable-background-timer-throttling --disable-renderer-backgrounding --disable-backgrounding-occluded-windows` |

On Windows those arguments are the WebView2 environment's, which all of an
app's pages share and which the first window or web view that shows a page
creates: give the option to that one. Then no page of the app has its timers
throttled; a page that asks later is too late, and the log says so. Two
processes of the app share the browser too, so they must agree: WebView2
refuses web views to a second one whose first page asks otherwise than the
running one's did. A web
view in a hidden window's [native UI](ui/web-view.md) still goes hidden, as
it must for its window to leave the screen.

### Autoplay

WebKitGTK and WebView2 let a page start audible media only once the user
clicked or typed in it: before, `play()` of an unmuted `<audio>` or `<video>`
rejects with `NotAllowedError`, and muted media that plays pauses again when
it is unmuted. The page of a hidden window never gets such a gesture, so
there it can't play sound at all. An app whose interface is [native UI](native.md), and
which plays its media in a page of a hidden window, lets that page play with
`AutoplayAllow`:

```go
mygo.NewWindow(mygo.WindowOptions{
	URL:    "/player",
	Hidden: true,
	Page:   mygo.PageOptions{Autoplay: mygo.AutoplayAllow},
})
```

Then `play()` resolves and `autoplay` elements start with their sound, in
windows and in [web views](ui/web-view.md) alike. `AutoplayUserGesture`, the
default, leaves the page to its web view's own policy.

| | |
|---|---|
| macOS | `WKWebViewConfiguration.mediaTypesRequiringUserActionForPlayback` none |
| Linux | the website policies of the page's main-frame navigations allow autoplay (`WEBKIT_AUTOPLAY_ALLOW`, WebKitGTK 2.30+; nothing before) |
| Windows | its browser runs with `--autoplay-policy=no-user-gesture-required` |

On Windows, as with [`ThrottlingOff`](#background-pages), that argument is
the WebView2 environment's, which all of an app's pages share and which the
first window or web view that shows a page creates: give the option to that
one. Then every page of the app may play without a gesture, those that keep
`AutoplayUserGesture` too; a page that asks later is too late, and the log
says so. In an app whose first window is native UI, the window of the page
that plays is often the first to show a page, and its option is enough. One
whose first window shows a page gives that window the option too, or creates
the page that plays first. Two processes of the app must agree, as with
`ThrottlingOff`.

### Camera and microphone

`CaptureState` tells whether a page captures the camera, the microphone or
the display (`getUserMedia()`, `getDisplayMedia()`), from any of its frames,
and whether the capture is muted: `CaptureNone`, `CaptureActive` or
`CaptureMuted`. `OnCaptureStateChanged` tells when that changes, for a
recording indicator in the window or the menu bar, as Safari shows.
`SetCaptureState` mutes, unmutes or stops a capture without the page's
help, as the mute button of a call app: a muted capture keeps the page's
tracks, which get silence or black frames, and a stopped one ends them.

```go
page := win.Page()
page.OnCaptureStateChanged(func(kind mygo.CaptureKind, s mygo.CaptureState) {
	if kind == mygo.CaptureMicrophone {
		log.Println("microphone live:", s == mygo.CaptureActive) // show a recording indicator
	}
})

// A menu item that mutes or unmutes the microphone of the window it was chosen for.
mute := &mygo.MenuItem{Label: "Mute or Unmute Microphone", Accelerator: "CmdOrCtrl+Shift+M", Click: func(_ *mygo.MenuItem, win *mygo.Window) {
	if win == nil || win.Page() == nil {
		return
	}
	p := win.Page()
	switch p.CaptureState(mygo.CaptureMicrophone) {
	case mygo.CaptureActive:
		p.SetCaptureState(mygo.CaptureMicrophone, mygo.CaptureMuted)
	case mygo.CaptureMuted:
		p.SetCaptureState(mygo.CaptureMicrophone, mygo.CaptureActive)
	}
}}
```

The page asks for the devices first, which [permissions](#permissions)
decide. A change shows in `CaptureState` and the event once the page took
it, a moment after `SetCaptureState`, which does nothing while the page
captures no such device. The page can't unmute a capture itself: asking for
the device again goes to the permission handler, as a new capture does.

| Platform | Capture state | Muting and stopping |
|---|---|---|
| macOS | `cameraCaptureState` and `microphoneCaptureState` of `WKWebView` (macOS 12+), and WebKit's private `_displayCaptureState` (macOS 13+), observed; `CaptureNone` if WebKit doesn't answer them | `setCameraCaptureState:completionHandler:`, `setMicrophoneCaptureState:completionHandler:` and the private `_setDisplayCaptureState:completionHandler:`; while only one of the camera and the microphone is muted, muting or unmuting the page's sound (`SetAudioMuted`) mutes the other for a moment, as WebKit mutes them together |
| Linux | `webkit_web_view_get_camera_capture_state` and its microphone and display siblings, WebKitGTK 2.34 or later, and their `notify` signals | their setters; muting or unmuting the page's sound (`SetAudioMuted`) unmutes every capture in WebKitGTK, which MyGo mutes again at once, so a muted capture may be live for a moment |
| Windows | always `CaptureNone`: WebView2 tells no capture | does nothing |

`CaptureState` is `CaptureNone` for a window showing native UI and after
the window closed. A [web view's](ui/web-view.md) page has them too.

### Picture in picture

`PageOptions.PictureInPicture` lets a page's videos float out into picture
in picture, a small window over every app's, as they do in Safari: the
button of a video's controls, the item of its context menu, and
`requestPictureInPicture()` in the page's script. WKWebView keeps it to
Safari, so a page on macOS needs the option; WebView2 lets videos go
whatever it says. `TogglePictureInPicture` moves a video out, or the one
out back into the page, without a click in the page, and
`IsPictureInPictureActive` tells whether one is out, whoever put it there:

```go
win := mygo.NewWindow(mygo.WindowOptions{
	URL:  "/player",
	Page: mygo.PageOptions{PictureInPicture: true},
})

// A menu item that floats the video of the window it was chosen for.
pip := &mygo.MenuItem{Label: "Picture in Picture", Accelerator: "CmdOrCtrl+Shift+P", Click: func(_ *mygo.MenuItem, win *mygo.Window) {
	if win != nil && win.Page() != nil {
		win.Page().TogglePictureInPicture()
	}
}}
```

| Platform | |
|---|---|
| macOS | WebKit's private `_allowsPictureInPictureMediaPlayback` preference, and `_togglePictureInPicture`, `_canTogglePictureInPicture` and `_isPictureInPictureActive` for the video WebKit picks for its media controls; all do nothing, or false, if WebKit stops answering them. A page that navigates with its video out is left out of the back/forward cache (private `_setUsesPageCache:`), from which WebKit would bring it back believing its video still out |
| Windows | Chromium's, on in every page. `TogglePictureInPicture` calls `requestPictureInPicture()` of the video playing, else of the largest, in the page's document, not its frames, as a user's gesture, and `IsPictureInPictureActive` asks the page for `document.pictureInPictureElement`, which a video in a frame leaves null |
| Linux | none: WebKitGTK has no picture in picture, so `TogglePictureInPicture` does nothing and `IsPictureInPictureActive` is false |

Both do nothing, or false, for a window showing native UI and after the
window closed. A [web view's](ui/web-view.md) page has them too.

### Find in page

`FindInPage` highlights the matches of a text and reports how many there
are, like the find bar of a browser; call it as the user types, and with
`FindNext` to go to the next match:

```go
res, err := win.Page().FindInPage("mygo", mygo.FindOptions{FindNext: true})
fmt.Printf("%d of %d\n", res.Active, res.Matches)
win.Page().StopFindInPage()
```

### Printing, PDF and screenshots

```go
win.Page().Print() // the print dialog

pdf, err := win.Page().PrintToPDF(mygo.PDFOptions{
	PageSize:   mygo.PageA4,
	Landscape:  false,
	Margins:    &mygo.Margins{Top: 0.5, Right: 0.5, Bottom: 0.5, Left: 0.5}, // inches
	Background: true, // print background colors and images
})

png, err := win.CapturePage() // what the window shows, as a PNG
```

`PrintToPDF` lays the page out for printing, so its print style sheets
apply. The zero `PDFOptions` prints Letter pages in portrait with margins of
0.4 inch and no backgrounds, like a browser's print dialog.

The PDF's text copies and searches as the page's. On macOS, the PDFs that
WebKit prints name radicals for many Chinese characters, such as ⼈
(U+2F08) for 人, which the system's fonts draw with the same glyph.
`PrintToPDF` asks the installed fonts which characters their glyphs are
and appends corrected text maps to the PDF, which leaves its pages as they
were. Text in a font that is not installed, such as a web font, keeps the
radicals.

### Downloads

A page downloads a file when it follows a link with the `download`
attribute, or gets a response it cannot show, such as an attachment. By
default the file goes to the Downloads directory, under the name the page
or the server suggests. `OnWillDownload` chooses another path or cancels
the download, and `OnDownloadDone` reports how it ended:

```go
win.Page().OnWillDownload(func(e *mygo.DownloadEvent) {
	if strings.HasSuffix(e.SuggestedName, ".exe") {
		e.PreventDefault()
		return
	}
	dir, _ := mygo.App.Path(mygo.PathDocuments)
	e.Path = filepath.Join(dir, e.SuggestedName)
})
win.Page().OnDownloadDone(func(d *mygo.Download) {
	if d.Err == nil {
		mygo.Shell.ShowItemInFolder(d.Path)
	}
})
```

### Permissions

Pages ask for the camera, the microphone, the location or notifications.
By default the app's own pages, those of custom schemes included (see
[trust](frontend.md#trust)), get what they ask for and other pages get
nothing. `Page.SetPermissionHandler` decides instead; it runs on the main
thread:

```go
win.Page().SetPermissionHandler(func(req mygo.PermissionRequest) bool {
	return req.Origin == "https://meet.example.com" &&
		!slices.Contains(req.Permissions, mygo.PermissionGeolocation)
})
```

The operating system may ask the user too. macOS does, once per app, for
the camera and the microphone, and ends apps that use them without
explaining why in `Info.plist`: add usage descriptions to `macos.infoPlist`
in mygo.config.ts:

```ts
export default defineConfig({
  macos: {
    infoPlist: {
      NSCameraUsageDescription: "Scan documents with the camera.",
      NSMicrophoneUsageDescription: "Record voice notes.",
    },
  },
});
```

### Sign-in and certificates

A page behind HTTP authentication (Basic, Digest, NTLM), of its server or
of a proxy, asks for a user name and a password. `Page.SetAuthHandler`
answers, on the main thread, with the credentials, or with an empty
`AuthResponse` to cancel, which shows the server's refusal:

```go
win.Page().SetAuthHandler(func(req mygo.AuthRequest) mygo.AuthResponse {
	if req.Host == "router.lan" && !req.IsProxy {
		return mygo.AuthResponse{Username: "admin", Password: password}
	}
	return mygo.AuthResponse{}
})
```

The web view keeps the credentials a server accepted for the rest of the
session. Without a handler, WebKitGTK and WebView2 show their own sign-in
dialog, and macOS none. To ask the user in the app's own way, cancel,
prompt, then `Reload` once the handler has the credentials.

A server whose certificate does not verify, such as a dev server with a
self-signed certificate, fails to load, as in a browser.
`Page.SetCertificateErrorHandler` can load it anyway: allow only the
certificate you expect, which `CertificateError.Certificate` holds
(DER-encoded), never a host whatever its certificate:

```go
win.Page().SetCertificateErrorHandler(func(e mygo.CertificateError) bool {
	sum := sha256.Sum256(e.Certificate)
	return e.Host == "dev.example" && sum == devCertificateSHA256
})
```

On Linux only the page itself is asked about, not what it loads, and on
Linux and Windows a certificate allowed for a host stays allowed in every
window until the app quits. Client certificates are left to the system.

### Blocking content

Content rules block what pages load and hide parts of them, as ad and
tracker blockers do. They are written in WebKit's
[content blocker format](https://developer.apple.com/documentation/safariservices/creating-a-content-blocker):
a JSON array of rules, each a trigger that matches loads by a regular
expression of their URL, and an action. `mygo.CompileContentRules`
compiles them, and a page applies them through `PageOptions.ContentRules`
or `Page.SetContentRules`:

```go
rules, err := mygo.CompileContentRules("ads", `[
	{"trigger": {"url-filter": "^[^:]+://+([^:/]+\\.)?ads\\.example\\.com[:/]"}, "action": {"type": "block"}},
	{"trigger": {"url-filter": ".*", "if-domain": ["*news.example"]},
	 "action": {"type": "css-display-none", "selector": ".sponsored"}}]`)
if err != nil {
	return err
}
win := mygo.NewWindow(mygo.WindowOptions{URL: "https://news.example/",
	Page: mygo.PageOptions{ContentRules: []*mygo.ContentRules{rules}}})
```

Triggers take `url-filter`, `url-filter-is-case-sensitive`, one of
`if-domain`, `unless-domain`, `if-top-url` and `unless-top-url`,
`resource-type` and `load-type` (`first-party` or `third-party`).
Actions are `block`, `block-cookies`, `make-https`, `css-display-none` with
a `selector`, and `ignore-previous-rules`, which undoes what the rules
before it did to a load. Regular expressions are WebKit's: no alternatives
(`|`), counted repetitions or classes such as `\d`, and ASCII only.
Domains are lowercase, in punycode. A rule whose selector the engine cannot
parse is skipped; any other mistake fails the whole list.

`CompileContentRules` takes a while for long lists, so call it off the
main thread. The engine keeps what it compiled under the id: the same
rules load right away on later launches, and other rules compiled under
the id replace the earlier ones. WebKit limits the rules of a list (to
150,000, or 50,000 in older versions), so split a long list into several
and apply them together; a page applies its lists in order, each list's
`ignore-previous-rules` undoing only its own rules. `SetContentRules`
replaces a page's lists, and with none removes them: the loads from then
on go through them, and the page shows what it loaded until it reloads.

macOS and Linux (WebKitGTK 2.24 or newer) apply the rules with WebKit's
content blockers. WebView2 has none, so on Windows MyGo sees every request
of a page applying rules and fails the blocked ones: a blocked `fetch`
rejects and an `XMLHttpRequest` errs, as in WebKit, while a blocked page
shows WebView2's error page for a 403 response. It redirects what
`make-https` matches, and hides elements with a style sheet each document
adopts. There:

- WebSockets are not blocked: WebView2 does not show MyGo their requests.
- Popups (`resource-type` `popup`) are not matched.
- `block-cookies` removes the cookies of the request, not those of the
  response.
- An element whose own `style` says `display: ... !important` stays: the
  style sheet is the page's, where WebKit's is the user's, which wins.

### The web inspector

`OpenDevTools`, `CloseDevTools`, `ToggleDevTools` and `IsDevToolsOpened`
of a page control its inspector, when it has one, see
[the web inspector](frontend.md#the-web-inspector).

### The page's process

`ProcessID` returns the ID of the operating system's process that runs a
page, for an app's own task manager, memory reports or telling which window
a busy process belongs to; 0 when it is unknown:

```go
pid := win.Page().ProcessID()
```

| Platform | The process |
|---|---|
| macOS | the page's web content process (`com.apple.WebKit.WebContent`), from WebKit's private `_webProcessIdentifier`; 0 if WebKit stops answering it. A page may move to another process when it navigates to another site or after its process crashed, so ask again rather than keep the ID |
| Windows | WebView2's browser process (`msedgewebview2.exe`), which the app's web views share and whose children render the pages |
| Linux | none: WebKitGTK has no API for the process of a page, so it is always 0 |

It is 0 too before the web view has started a process, after the window
closed, and for a window showing native UI.

`GPUProcessID` returns the ID of the process that does the page's graphics
and media. On macOS that is where the page's sound comes out, so an app
that shows what a page plays (a music player's visualizer, say) can capture
it with a Core Audio process tap (`CATapDescription`, macOS 14.2 and
later), which needs the process's ID:

```go
pid := win.Page().GPUProcessID()
```

| Platform | The process |
|---|---|
| macOS | WebKit's GPU process (`com.apple.WebKit.GPU`), which plays the page's audio and video, from WebKit's private `_gpuProcessIdentifier`; 0 if WebKit doesn't answer it or hasn't started the process yet, which it does once a page first needs it, to play media or draw with the GPU |
| Windows | WebView2's GPU process (`msedgewebview2.exe --type=gpu-process`), a child of `ProcessID`'s browser process, from `ICoreWebView2Environment8`'s `GetProcessInfos`; 0 on runtimes without it. WebView2 plays sound from its audio service, a utility process, not this one: capture the tree of `ProcessID`'s process for it (process loopback) |
| Linux | none: WebKitGTK has no API for its GPU process, so it is always 0 |
| Browsers | none: always 0 |

It is 0 too before the web view has started a process, after the window
closed, and for a window showing native UI. The app's web views may share
the process, and it gets a new ID each time the app runs and whenever the
web view starts it again, after it crashed or exited, so ask again rather
than keep the ID.

## Finding windows

- `mygo.Windows()` lists the open windows, `mygo.FocusedWindow()` returns
  the focused one and `mygo.WindowByID(id)` the one with an `ID()`, which
  pages know as `runtime().windowId`.
- `mygo.CallerWindow(ctx)` returns the window whose page called a bound
  method, and `mygo.CallerPage(ctx)` the page: the window's, or that of a
  web view in a window of native UI ([Web view](ui/web-view.md)). Given
  the `r.Context()` of a request to a custom scheme, they return the
  window and the page that made it.
- `mygo.App.OnWindowCreated` is called for every new window.

## GTK's style on Linux

On Linux, GTK draws part of a window's frame itself, in the desktop's
GTK 3 theme: the title bar and decorations of a window the desktop leaves
to decorate itself (GNOME on Wayland), the shadow and border GTK puts
around a frameless window or a hidden title bar on Wayland, and the title
buttons of a hidden title bar; its menus and dialogs too.
`mygo.Theme.SetGTKCSS` styles them with a GTK 3 style sheet of the app's,
for a frame that matches the rest of the desktop, such as the flat header
bars and round title buttons of GTK 4 and libadwaita apps:

```go
const gtk4Frame = `
decoration { border-radius: 12px; box-shadow: 0 3px 9px 1px rgba(0, 0, 0, .28); }
.maximized decoration, .fullscreen decoration, .tiled decoration { border-radius: 0; }
.csd.popup decoration, tooltip.csd decoration { border-radius: 5px; }
headerbar { min-height: 46px; padding: 0 6px; background-color: #f9fafb;
            background-image: none; border-width: 0; box-shadow: none; }
button.titlebutton { min-width: 24px; min-height: 24px; margin: 0 3px; padding: 0;
                     border: none; border-radius: 9999px; background-color: transparent; }
`

if err := mygo.Theme.SetGTKCSS(gtk4Frame); err != nil {
	log.Print(err) // the sheet does not parse
}
```

- GTK takes the sheet at the user's priority
  (`GTK_STYLE_PROVIDER_PRIORITY_USER`): its rules win over the theme's,
  MyGo's own and those of the user's `~/.config/gtk-3.0/gtk.css`.
- Another call replaces the sheet, as a switch of the app's theme does, and
  `""` removes it. A sheet that does not parse returns GTK's error, with
  where it is, and the sheet set before stays.
- It may be called from any goroutine, before `App.Run` too: other
  goroutines wait for Run to load the sheet, and main's sheet waits for
  Run, which logs its error, as no call is left to return it.
- Pages and native UI never see it: MyGo draws them, and pages have their
  own CSS. On macOS, Windows and in browsers it does nothing and returns
  nil.

GTK 3 rounds only the top corners of a window it decorates, whatever the
`border-radius` of `decoration`: it clips the decorations to the radius but
never the window's content, so the page or the native UI covers the
bottom corners, which stay square. A window that needs four rounded
corners draws them itself: a `Frameless`, `Transparent` window, which GTK
leaves undecorated, without its shadow, whose page or native UI rounds its
own background, under a compositing window manager or on Wayland.

## Native access

`win.NativeHandle()` returns the native window, an `NSWindow*` on macOS, a
`GtkWindow*` on Linux and an `HWND` on Windows, for APIs MyGo does not
cover. Call native APIs on the main thread with `mygo.RunOnMain`.
