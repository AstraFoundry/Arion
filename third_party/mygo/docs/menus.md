# Menus and the tray

Menus are built from templates of `*mygo.MenuItem` and used as the menu
bar, context menus, the Dock menu and the menus of tray icons.

## Templates

```go
menu := mygo.NewMenu([]*mygo.MenuItem{
	{Role: mygo.RoleAppMenu},
	{Label: "File", Submenu: []*mygo.MenuItem{
		{Label: "New Note", Accelerator: "CmdOrCtrl+N", Click: func(item *mygo.MenuItem, win *mygo.Window) {
			newNote()
		}},
		{Label: "Export…", Accelerator: "CmdOrCtrl+E", Click: exportNotes},
		mygo.Separator(),
		{Role: mygo.RoleClose},
	}},
	{Role: mygo.RoleEditMenu},
	{Label: "View", Submenu: []*mygo.MenuItem{
		{ID: "sidebar", Label: "Show Sidebar", Type: mygo.MenuItemCheckbox, Checked: true, Click: toggleSidebar},
		mygo.Separator(),
		{Label: "List", Type: mygo.MenuItemRadio, Checked: true},
		{Label: "Grid", Type: mygo.MenuItemRadio},
		mygo.Separator(),
		{Role: mygo.RoleToggleDevTools},
		{Role: mygo.RoleToggleFullScreen},
	}},
	{Role: mygo.RoleWindowMenu},
})
```

A `MenuItem` has:

- `Label`, and an `Accelerator`, its keyboard shortcut (see
  [accelerators](#accelerators));
- `Click`, called on the main thread when the item is chosen, with the item
  and, on Linux and Windows, the window whose menu bar or context menu was
  chosen. Other menus use the focused window, or nil;
- `Type`: `MenuItemNormal`, `MenuItemCheckbox`, `MenuItemRadio`,
  `MenuItemSeparator` or `MenuItemSubmenu`; items with a `Submenu` are
  submenus, others normal items unless set. Adjacent radio items form a
  group, in which checking one unchecks the others. Checkbox and radio
  items toggle `Checked` before `Click` runs;
- `Disabled`, `Hidden` and `ToolTip`;
- `Icon`, an image shown beside the label (see [icons](#icons));
- `Subtitle`, a second, smaller line under the label. macOS 14 and browsers
  show it so; earlier macOS, Windows and Linux show it after the label, as
  `Label — Subtitle`;
- `ID`, to find the item with `Menu.ItemByID`;
- `Role`, a built-in behavior with its label and accelerator.

### Roles

Roles do what users expect of standard items, with their usual label and
shortcut, which `Label` and `Accelerator` override:

- editing: `RoleUndo`, `RoleRedo`, `RoleCut`, `RoleCopy`, `RolePaste`,
  `RoleDelete`, `RoleSelectAll`, and on macOS `RolePasteAndMatchStyle`.
  Their keys go to the content first, as in a browser: in a page on
  Windows, WebView2 cuts, copies and pastes with the page's own events and
  every format of the clipboard. Choosing Paste from the menu restores
  focus to the page and runs Chromium's paste command through DevTools,
  with the same paste event and formats. The page also sees `keydown` and
  `keyup` for an `Unidentified` key without modifiers, so its Ctrl+V
  handlers are not invoked. Paste and Match Style still inserts plain
  text. An edit role given a key the page has no use for, as Copy on
  Ctrl+Shift+C, still runs its command;
- formatting: `RoleBold`, `RoleItalic`, `RoleUnderline`,
  `RoleStrikethrough`, `RoleAlignLeft`, `RoleAlignCenter` and
  `RoleAlignRight`, which format the selection of a [rich text
  editor](ui/rich-text-editor.md) of native UI, or of an editable page
  (`document.execCommand`); Paste and Match Style pastes plain text into a
  rich text editor;
- the page: `RoleReload`, `RoleForceReload`, `RoleToggleDevTools` (in a
  window of native UI, its [inspector](ui/inspector.md)),
  `RoleResetZoom`, `RoleZoomIn`, `RoleZoomOut`;
- the window: `RoleToggleFullScreen`, `RoleMinimize`, `RoleZoom`,
  `RoleClose`;
- [documents](documents.md#the-file-menu): `RoleNew`, `RoleOpen`,
  `RoleOpenRecent`, `RoleSave`, `RoleSaveAs`, `RoleDuplicate`,
  `RoleRename`, `RoleMoveTo`, `RoleRevert`;
- the app: `RoleQuit`, and on macOS `RoleAbout`, `RoleHide`,
  `RoleHideOthers`, `RoleUnhide`, `RoleFront`, `RoleServices`,
  `RoleStartSpeaking`, `RoleStopSpeaking`;
- whole menus: `RoleFileMenu`, `RoleEditMenu`, `RoleFormatMenu` (Bold,
  Italic, Underline, Strikethrough and the alignments), `RoleViewMenu` and
  `RoleWindowMenu`, and on macOS `RoleAppMenu` (the menu named after the
  app), `RoleWindow` (the menu listing the windows) and `RoleHelp` (the
  menu with the search field).

Items with the roles of macOS are left out on other platforms, so one
template serves every platform.

The items of the editing roles, in menu bars and popups alike, are enabled
only while the focus can act on them, as in other apps: in
[native UI](ui/README.md), Cut and Copy with text selected, Undo with
something to undo, Paste in a text input, all of them for an element that
takes its input (`HandleInput`), and on macOS a role whose key a
[shortcut](ui/input.md) of the focus or the window takes.
On macOS the menus holding them ask the focus: a page's web view enables
them, and the Speech items, as WebKit does, and native UI, which does not
speak, disables the Speech items. On Linux and Windows a page's items keep
the state the app gave them. An item the app disables stays disabled.

### Changing items

Change items of a menu in use with their methods, so that the native menu
follows: `SetLabel`, `SetEnabled`, `SetVisible`, `SetChecked`,
`SetAccelerator`, `SetSubtitle` and `SetIcon`:

```go
item := menu.ItemByID("sidebar")
item.SetChecked(false)
```

`Menu.Append` and `Menu.Insert` add items.

### Icons

An item's `Icon` is an image shown beside its label, as the flag of each
server of a VPN's tray menu. `NewMenuIcon` makes one of a PNG image, in
its own colors, and returns an error for data that is not a PNG image;
make each once and give it to as many items as show it:

```go
//go:embed flags/jp.png
var japanPNG []byte

japan, err := mygo.NewMenuIcon(japanPNG, false)
if err != nil {
	log.Fatal(err)
}
servers := mygo.NewMenu([]*mygo.MenuItem{
	{ID: "tokyo", Label: "Tokyo", Type: mygo.MenuItemRadio, Checked: true, Icon: japan},
	{ID: "osaka", Label: "Osaka", Type: mygo.MenuItemRadio, Icon: japan},
	{ID: "direct", Label: "Direct", Type: mygo.MenuItemRadio},
})
```

The image fits a box of 16 by 16 points, keeping its aspect: give it at
least 32 by 32 pixels, or 48 by 48, so that it stays sharp on
high-resolution displays and at 150% and 200% scaling. A checkbox or radio
item shows its check as well. `SetIcon`
replaces an item's icon in menus in use, and nil removes it:

```go
servers.ItemByID("tokyo").SetIcon(warning)
servers.ItemByID("osaka").SetIcon(nil)
```

`NewMenuIcon`'s second argument, template, makes a black and transparent
image take the color of the menu's text, as macOS tints its template
images: white in a dark menu and on the highlight. Linux shows template
icons as they are.

`NewMenuIconSVG` makes an icon of an SVG instead, the `*ui.SVG` that
`ui.ParseSVG` parses, so that an app shows the icons it draws with
`ui.Icon` in its menus too. It draws the SVG once, with native UI's
renderer (`ui.SVG`'s `Image`), to an image of 64 pixels on its longer
side, which stays sharp up to 400% scaling, and returns an error for a nil
SVG or one without a size. As a template, only its shapes count, whatever
their colors, as for `ui.Icon`:

```go
//go:embed icons/trash.svg
var trashSVG []byte

var trash = ui.MustParseSVG(trashSVG) // ui.Icon(c, trash) in the window

trashIcon, err := mygo.NewMenuIconSVG(trash, true)
```

In the menus native UI draws, an item's
[`IconColor`](ui/context-menu.md#items) gives its template icon a color
of its own.

MyGo decodes and scales each image once, however many items show it and
however often their menus open or are made again, the icons of the same
image included, and frees what it made of it once neither the app nor a
menu holds the icon; the bitmaps of Windows 10's menus go with their
menus. Apps that make no icon don't link the code that draws them.

The context menus of native UI take the same icons, with
[`Icon`](ui/context-menu.md#items) on their items.

- **macOS** shows the icon (an `NSImage`) after the column of checks, on
  every menu, the menu bar's own items included, which show it in the
  menu bar.
- **Windows 11** menus, which MyGo draws, give icons a column of their
  own between the checks and the labels, as the system's menus do, at the
  menu's display scaling. **Windows 10** draws them as the items'
  bitmaps (32-bit, with alpha), in the column of its checks: a checked
  item shows its icon framed as checked in place of a check mark. The
  items of the menu bar itself show no icon.
- **Linux** shows icons in the items of menu bars' menus and context
  menus (GTK); the items of the menu bar itself show none. A tray icon's
  menu is drawn by the panel, which gets the icons over DBusMenu
  (`icon-data`): GNOME Shell's AppIndicator extension and KDE Plasma
  show them, other panels may not.

## The menu bar

```go
mygo.App.SetMenu(menu)
```

On macOS the menu bar belongs to the app. Apps get a default one, with the
app, File, Edit, View and Window menus, until they set their own: keep
`RoleAppMenu` and `RoleEditMenu` in yours, which make Cmd+Q and copy and
paste work.

On Linux and Windows menu bars belong to windows. `App.SetMenu` gives its
menu to every window without one of its own, `Window.SetMenu` gives a
window its own, and windows have none by default, except in apps with
[documents](documents.md), which get File and Edit menus. `App.SetMenu(nil)`
removes the menu.

On Windows, menu bars and menus follow the appearance, light or dark, as
the title bar does (see [dark mode](frontend.md#dark-mode)). On Windows 11
they look like the system's own menus, as File Explorer's: roomier items,
highlights with rounded corners, shortcuts smaller and in a fainter
color, the rounder corners of the system's flyouts and, from Windows 11
22H2, Acrylic behind popup menus, as WinUI's menus have, unless
transparency effects are off, energy saver or a contrast theme is on, or
the session is a remote one, when they have the solid background WinUI's
menus fall back to, with one faint outline.

A window with `AutoHideMenuBar` keeps its menu bar out of sight, and the
room for the page. Alt pressed alone, or F10, shows the bar with the
keyboard in it (Windows) or its first menu open (Linux), and the bar hides
again once the user picks an item or leaves it. Its shortcuts work all
along:

```go
mygo.NewWindow(mygo.WindowOptions{URL: "/", AutoHideMenuBar: true})
```

A Windows window without a title bar, frameless or with a hidden title
bar, has no room for a menu bar either: Alt and F10 open its menus in a
popup from the top-left corner, below the title bar the page draws. On
Linux a window with a hidden title bar hides its menu bar as
`AutoHideMenuBar` does, so that the window buttons stay at the top of the
window, in the page's title bar; a frameless window shows the bar above
the page.

A window in full screen leaves the whole screen to the page: its menu bar
hides until it leaves full screen. Alt and F10 still open its menus, in a
popup from the top-left corner on Windows, and in the bar, which shows
while they are open, on Linux.

## Context menus

`Menu.Popup` shows a menu at the mouse over a window and returns once it
closes; `PopupAt` shows it at a position in the page. Show one from a bound
method, which the page calls on right-click:

```go
// ShowNoteMenu shows the context menu of a note.
func (n *Notes) ShowNoteMenu(ctx context.Context, id int64) {
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Rename", Click: func(*mygo.MenuItem, *mygo.Window) { n.startRename(id) }},
		{Label: "Delete", Click: func(*mygo.MenuItem, *mygo.Window) { n.Delete(id) }},
	})
	menu.Popup(mygo.CallerWindow(ctx))
}
```

```ts
note.addEventListener("contextmenu", (e) => {
  e.preventDefault();
  Notes.showNoteMenu(id);
});
```

Without a handler of their own, pages show the webview's standard context
menu, with Copy and Paste on text and Inspect Element in development
builds. Windows showing [native UI](ui/context-menu.md) give elements context
menus with `ContextMenu`, built like the rest of their interface.

### The webview's menu

`Page.OnContextMenu` changes that standard menu as it is about to open,
as Electron's `context-menu` event does. The event tells what was
right-clicked (`LinkURL`, `LinkText`, `ImageURL`, `MediaURL`,
`SelectionText`, `Editable`, `FrameURL`), and its `Menu` holds the
webview's own items. Hide, rename, disable or remove them, and add items
of the app's, which work as in any other menu:

```go
page.OnContextMenu(func(e *mygo.ContextMenuEvent) {
	e.Menu.Remove(mygo.ContextMenuReload, mygo.ContextMenuInspect)
	if e.LinkURL != "" {
		e.Menu.Append(mygo.Separator(), &mygo.MenuItem{
			Label: "Open in Browser",
			Click: func(*mygo.MenuItem, *mygo.Window) { go mygo.Shell.OpenExternal(e.LinkURL) },
		})
	}
})
```

`e.PreventDefault()` shows no menu, and setting `e.Menu` to a menu of the
app's shows that one instead. The webview's items keep doing what the
webview does, wherever they are moved, unless the app gives one a `Click`,
which then runs in its place. A page whose own `contextmenu` handler calls
`preventDefault` opens no menu, and the listeners are not called.

The webview's items have the same IDs on every platform that has them:

| ID | item |
|---|---|
| `ContextMenuBack`, `ContextMenuForward`, `ContextMenuReload`, `ContextMenuStop` | the page's navigation |
| `ContextMenuInspect` | Inspect Element, in builds with DevTools |
| `ContextMenuUndo`, `ContextMenuRedo`, `ContextMenuCut`, `ContextMenuCopy`, `ContextMenuPaste`, `ContextMenuPasteAndMatchStyle`, `ContextMenuDelete`, `ContextMenuSelectAll` | editing |
| `ContextMenuSpellingGuess` | each word suggested for a misspelled one |
| `ContextMenuOpenLink`, `ContextMenuOpenLinkInNewWindow`, `ContextMenuDownloadLink`, `ContextMenuCopyLink` | links |
| `ContextMenuOpenImageInNewWindow`, `ContextMenuDownloadImage`, `ContextMenuCopyImage`, `ContextMenuCopyImageURL` | images |
| `ContextMenuOpenMediaInNewWindow`, `ContextMenuDownloadMedia`, `ContextMenuCopyMediaURL`, `ContextMenuMediaControls`, `ContextMenuMediaLoop`, `ContextMenuMediaFullScreen`, `ContextMenuPictureInPicture` | video and audio |
| `ContextMenuOpenFrameInNewWindow`, `ContextMenuLookUp`, `ContextMenuSearchWeb`, `ContextMenuShare`, `ContextMenuEmoji`, `ContextMenuPrint` | the rest |

The other items keep the platform's own name as their ID: WebKit's
identifier on macOS (`WKMenuItemIdentifierTranslate`), the stock action's
nick on Linux (`unicode`), WebView2's name on Windows (`createQrCode`).
Separators, and items the platform names nothing, have none.

What each platform does:

- **macOS** changes WebKit's menu in place as it opens, so Look Up,
  Translate, Services and the spelling suggestions keep working. WebKit
  tells nothing of what was clicked: MyGo's runtime reports it from the
  page's `contextmenu` event, in the page's own document, so a click in a
  frame tells nothing and `FrameURL` stays empty.
- **Linux** changes WebKitGTK's menu in place. WebKitGTK tells the link,
  the image, the media and whether the element is editable, and the page
  the selection. The app's items show no shortcut or icon, radio items show
  as check boxes, and only items of WebKitGTK's stock actions can be
  renamed. `FrameURL` stays empty.
- **Windows** gets everything from WebView2 (1.0.1185 and later; older
  runtimes never call the listeners). A menu the app changed is drawn by
  MyGo, like its other popup menus, with WebView2's items doing what they
  do; a menu left as it was is WebView2's own. WebView2 gives the spelling
  suggestions no words, so a menu MyGo draws leaves them out, and it
  leaves out the URL of a `data:` image, which the page's `contextmenu`
  event then tells.

A menu the app changed shows no separator first, last or after another,
as removing the items between two leaves.

## The Dock menu

On macOS `App.Dock.SetMenu` adds items above the standard ones of the menu
that right-clicking the Dock icon opens. Like `App.SetMenu`, it may be
called in `main` before `App.Run`:

```go
mygo.App.Dock.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
	{Label: "New Window", Click: func(*mygo.MenuItem, *mygo.Window) { openMainWindow() }},
}))
```

## Tray icons

A tray icon sits in the menu bar on macOS and in the notification area on
Windows and Linux. Create one once the app is ready:

```go
//go:embed resources/tray.png
var trayIcon []byte

mygo.App.WhenReady(func() {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon:           trayIcon,
		IconIsTemplate: true, // macOS tints it to match the menu bar
		ToolTip:        "Notes",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "Open Notes", Click: func(*mygo.MenuItem, *mygo.Window) { openMainWindow() }},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}),
	})
	if err != nil {
		log.Println("tray:", err)
	}
	_ = tray
})
```

The icon is a PNG, ideally 16×16 points (a 32×32 image for high-resolution
displays); a template icon, black on transparent, lets macOS tint it for
light and dark menu bars. A tray with a menu shows it on click. Without a
menu, clicks reach `OnClick`, and `OnRightClick` gets right-clicks, and on
macOS Control-clicks, as a Control-click is a secondary click there. Change
the tray with `SetIcon`, `SetMenu`, `SetToolTip` and, on macOS,
`SetTitle`, the text next to the icon; `Destroy` removes it.

The items of a tray's menu can show [icons](#icons), as a country's flag
beside each server, and change as the items of any menu do: the menu
shows their state the next time it opens.

A title that shows a changing number, such as a network speed, should
keep its width, or the item grows and shrinks and the icon and the items
beside it move. On macOS the title's digits all have the same width, so a
value padded to a fixed number of characters keeps it: pad with figure
spaces (U+2007), which are as wide as a digit, since an ordinary space is
narrower.

```go
// "  9.0 MB/s", " 12.5 MB/s", "128.0 MB/s": all as wide.
func speed(mbps float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%5.1f", mbps), " ", "\u2007") + " MB/s"
}

tray.SetTitle(speed(rate))
```

Only digits are tabular: letters keep their own widths, so a title that
switches units (`9 KB/s`, then `1.2 MB/s`) changes width with the unit.
Keeping that steady is the app's choice, by showing the value in one unit
as above, for one.

An app may show several icons, as a menu bar app showing its measures each
in an item of its own. On macOS a new icon goes left of those already
there, and the user moves icons by dragging them with ⌘ held: give each an
`ID`, unique among the app's icons, and it goes back where the user put it
whenever it is made again, at the next launch or after `Destroy`. On
Windows, where the user chooses which icons show on the taskbar rather
than in its overflow, an icon with an `ID` keeps that choice, whatever
order the app makes its icons in.

A template icon follows the menu bar by itself. An icon in color, as one
drawing each of its measures in its own color, draws its ink for the bar
it shows on: `Tray.Dark` reports whether that bar is dark (the macOS menu
bar, as the system's appearance or the wallpaper under it tints it, or the
Windows taskbar), and `OnAppearanceChange` tells when it turns, to set the
icon again:

```go
draw := func() { tray.SetIcon(meterIcon(tray.Dark()), false) }
tray.OnAppearanceChange(draw)
draw()
```

`Tray.PopUpMenu` opens the menu set with `SetMenu` programmatically. On
macOS, AppKit places it below the status item and highlights its button
while the menu is open. On Windows 11 it opens as the system's tray menus
do, centered on the icon a little above the taskbar (or beside it, where
the taskbar is on another edge); on Windows 10, and for an icon in the
taskbar's overflow, at the pointer. To show a menu only on right-click while keeping
primary clicks available, set the menu in `OnRightClick`, call
`PopUpMenu`, then clear it with `SetMenu(nil)` when the call returns.

A tray icon does not keep the app running: the app quits when its last
window closes ([Quitting](app.md#quitting)). To close a window to the
tray, hide it instead, which keeps the window and the app. A quit, as the
tray menu's `RoleQuit`, starts before the windows close, so `OnClose` can
tell the closes it makes from the user's:

```go
var quitting bool // OnBeforeQuit and OnClose run on the main thread
mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })

win.OnClose(func(e *mygo.CloseEvent) {
	if !quitting {
		e.PreventDefault()
		win.Hide()
	}
})
// The icon's click, without a menu, or an item of its menu, brings it back.
tray.OnClick(win.Show)
```

On Linux tray icons use AppIndicator, which needs
`libayatana-appindicator3`: they show a menu, and an icon with an `OnClick`
listener answers activation (with a release of the library that has the
`activate` signal; older ones report no clicks), which GNOME's AppIndicator extension sends on
a double click and which opens the menu on a single one after the double
click time. `NewTray` returns an error without the library.

Apps that live in the menu bar usually drop their Dock icon on macOS:

```go
mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
```

For an icon that opens a window of native UI, see
[the menu bar example](https://github.com/egoist/mygo/tree/main/examples/menubar-native).
It leaves the tray menu unset on macOS and Windows, uses `OnClick` to
toggle a frameless window positioned with `Tray.Bounds`, and hides it on
`OnBlur` or Escape. On Linux, an AppIndicator menu item opens the same
window.

## Accelerators

Accelerators are keys with modifiers joined by `+`, such as `CmdOrCtrl+N`
or `Alt+Shift+F4`, in any case.

Modifiers:

| Modifier | Key |
|---|---|
| `CmdOrCtrl` (`CommandOrControl`) | Command on macOS, Control elsewhere |
| `Cmd` (`Command`), `Super`, `Meta`, `Win` | Command on macOS, the Windows or Super key elsewhere |
| `Ctrl` (`Control`) | Control |
| `Alt` (`Option`, `AltGr`) | Alt, or Option on macOS |
| `Shift` | Shift |

Keys:

- a character: a letter, a digit or a sign such as `,` or `/` (`Plus` for
  `+`);
- `F1` to `F24`;
- `Enter` (`Return`), `Tab`, `Space`, `Backspace`, `Delete` (`Del`),
  `Insert` (`Ins`), `Escape` (`Esc`);
- `Up`, `Down`, `Left`, `Right`, `Home`, `End`, `PageUp`, `PageDown`;
- `Num0` to `Num9`, `NumDec`, `NumAdd`, `NumSub`, `NumMult`, `NumDiv` (the
  numeric keypad);
- `VolumeUp`, `VolumeDown`, `VolumeMute`, `MediaPlayPause`,
  `MediaNextTrack`, `MediaPreviousTrack`, `MediaStop`, `PrintScreen`,
  `CapsLock`, `NumLock`, `ScrollLock`.

The same syntax registers [global shortcuts](native.md#global-shortcuts).
