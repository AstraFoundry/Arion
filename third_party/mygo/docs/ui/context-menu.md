# Context menu

<!-- preview context-menu 180 -->

`ContextMenu` gives an element a menu of the system's, which opens where
the element is right-clicked (Control-clicked on macOS), and below it when
the menu key or Shift+F10 is pressed while it or one inside it has the
focus. The function builds the items when the menu opens, and runs again in
the frame after one was chosen, where its `Chosen` reports it:

```go
row.ContextMenu(func(m *ui.Menu) {
	if m.Item("Rename").Shortcut(0, ui.KeyF2).Chosen() {
		app.renaming = i
	}
	if m.Item("Pinned").Checked(note.pinned).Chosen() {
		note.pinned = !note.pinned
	}
	m.Submenu("Move to", func(m *ui.Menu) {
		for _, f := range app.folders {
			if m.Item(f.name).Chosen() {
				moveTo, moved = f, i
			}
		}
	})
	m.Separator()
	if m.Item("Delete").Disabled(note.locked).Chosen() {
		deleted = i
	}
})
```

The function runs as the element builds, in the loop building the rows:
a choice that takes the row out of the notes, as moving or deleting it,
notes it, for the change once the loop is done (see
[views](views.md#events-and-actions)).

## Items

- `Item(label)` adds an item; `Chosen` reports whether it was chosen.
- `Checked(on)` shows a check mark by it, as for a setting that choosing the
  item toggles, and `Radio(on)` shows it as one of a group choosing one
  setting, marked as the one chosen when on is true.
- `Disabled(d)` grays it out: it cannot be chosen.
- `Tooltip(s)` shows `s` as the pointer rests on it, to say what it does.
  The system's menus show it on macOS and Linux, and `PopoverMenu` on
  every platform, also as the keys rest on it.
- `Icon(icon)` shows an image beside its label, made once with
  [`mygo.NewMenuIcon`](../menus.md#icons) of a PNG image, or
  `mygo.NewMenuIconSVG` of an SVG. The system's menus show it as they show
  the icons of `mygo.MenuItem`, browsers show none, and `PopoverMenu`
  draws it on every platform.
- `IconColor(color)` draws its template icon in `color` instead of the
  text's, as a red trash can by Delete; on the highlight it takes the
  highlight's text color, as the label does. Only drawn menus
  (`PopoverMenu`, `PopoverMenuButton`, and every menu of a window that
  [draws them](#every-menu-of-a-window)) show it: the system's tint
  template icons in the color of their text.
- `Shortcut(mods, key)` shows a key by it, as the shortcut that does the
  same. It only shows the key: handle it with `Shortcut` on the context or
  an element.
- `Subtitle(s)` shows a second, smaller line under its label, as
  [`mygo.MenuItem`'s](../menus.md) does: under the label on macOS 14 and
  in browsers, after it elsewhere (`Label — Subtitle`), and under it in
  `PopoverMenu` on every platform. Assistive technology reads it as the
  item's description.
- `ID(id)` gives it an identity of its own, which `Chosen` matches instead
  of its place and label: a label that changes while the menu shows, as
  `Sync 3 Files`, or items before it that come and go, keep the choice on
  it.
- `Separator()` adds a line between groups of items, and
  `Submenu(label, build)` an item opening a submenu, which `build` fills.

## Which element

The innermost element with a menu gets the click, as a row's button with a
menu of its own inside a row with another. [Text inputs](text-input.md)
have the editing commands of their platform's text fields, and selectable
[text](text.md) has Copy and Select All. `ContextMenu` on the input, the
text, or the `Selectable` container replaces that menu, and `EditItems`
adds those commands to a menu of one's own, acting on the text that was
clicked, as a chat message's text offers Reply above Copy:

```go
message.Selectable().ContextMenu(func(m *ui.Menu) {
	if m.Item("Reply").Chosen() {
		app.reply(msg)
	}
	m.Separator()
	m.EditItems()
})
```

An element within text, as a link in a paragraph, opens its menu on that
text: `EditItems` in the link's `ContextMenu` adds the paragraph's Copy
and Select All, which act on the selection.

The same menu opens below a button with `Element.Menu`: see
[menu button](menu-button.md). In tests, `tt.RightClick` opens a context
menu, which `tt.Menu` lists and `tt.ChooseMenuItem` chooses from.

## Drawn in the window

`PopoverMenu` takes the same function, and MyGo draws the menu in the
window with the theme's look, as a [popover](popover.md), where the
system would show its own: beside the popovers and buttons of an app's own
design, the system's menu is the one that looks out of place.

```go
card.PopoverMenu(func(m *ui.Menu) {
	if m.Item("Play Next").Chosen() {
		app.playNext(album)
	}
	if m.Item("Pinned").Checked(album.pinned).Chosen() {
		album.pinned = !album.pinned
	}
	m.Submenu("Sort by", func(m *ui.Menu) {
		for _, by := range []string{"Title", "Artist", "Year"} {
			if m.Item(by).Radio(app.sort == by).Chosen() {
				app.sort = by
			}
		}
	})
	m.Separator()
	if m.Item("Remove").Shortcut(0, ui.KeyDelete).Disabled(album.locked).Chosen() {
		app.remove(album)
	}
})
```

It opens as the system's would, where the element is right-clicked, or
below it for the menu key or Shift+F10, after that point, or before it
where the window has no room, and follows the element as the window
scrolls. It takes the focus as it opens and gives it back as it closes to
the element that had it, and has the keys of the system's menus:

- Up and Down move among the items, past separators and those disabled,
  and round; Home and End go to the first and the last, and typing the
  first letters of an item goes to it.
- Right opens the submenu of the item, and Left closes it, the other way
  round [right to left](right-to-left.md); the pointer opens it as it
  moves onto the item.
- Enter and Space choose the item, as a click does; Escape closes the
  submenu on top, then the menu, and Tab closes the menu.
- A press outside the menu closes it, and goes on to what is under the
  pointer.

Its items show their `Shortcut` after their label, with a check mark or
a dot by those `Checked` or `Radio` marks. Assistive technology sees a
menu of menu items, check box items and radio items, checked or not, as
it sees the system's.

A drawn menu's items are elements: in tests, `tt.RightClick` opens it,
`tt.HasText` finds its items and `tt.Click` chooses one. `tt.Menu`,
`tt.ChooseMenuItem` and `tt.CloseMenu` work on it as on the system's, so a
test passes with either.

`PopoverMenuButton` draws the menu of a [menu
button](menu-button.md#drawn-in-the-window) so, below the button, and takes
the same function: a row's right-click and its ⋯ button can share one
menu, drawn alike.

### Richer items

A drawn menu's items take more than the system's do, without a menu of
one's own ([without a look](#without-a-look)); the system's menus show
those items with their label alone:

```go
more.PopoverMenuButton(func(m *ui.Menu) {
	m.MaxWidth(320)
	for _, f := range app.files {
		if m.Item(f.Name).ID(f.Path).Trailing(f.Size).Chosen() {
			app.open(f)
		}
	}
	m.Separator()
	item := m.Item("Download").ID("download")
	item.Content(func(x ui.MenuItemContent) {
		ui.Text(x.Context, "Downloading "+app.download.Name)
		ui.Progress(x.Context, app.download.Progress)
	})
	m.Separator()
	if m.Item("Delete…").Danger().Chosen() {
		app.confirmDelete()
	}
})
```

- `Danger()` shows the item in the theme's `Danger` color, as one that
  deletes or discards. Highlighted, it keeps the theme's highlight, its
  label in `Danger` where that is legible on the highlight, else in
  `AccentText`. Assistive technology hears no color: let the label say
  what the item does.
- `Trailing(s)` shows a value at the item's end, before its shortcut, as a
  file's size; assistive technology reads it in the item's description.
- `Content(fn)` builds what the item shows in place of its label and
  subtitle, between its mark and icon and its trailing text and shortcut.
  `fn` gets the context, whether the item is highlighted or disabled, and
  the colors of its text (`MenuItemContent`); text in it takes the item's
  color. The item keeps its label, which assistive technology reads as
  its name and typing goes to. The content stays part of the item: a
  press on it chooses the item. A control in it, as a text input
  filtering the items after it, takes its own presses and keeps the focus
  while the pointer moves over its item, with the keys it handles (the
  text typed, Space, Enter, Home and End); Up and Down move the highlight
  and give the menu the focus back, and Escape closes the menu.
- `Highlighted()` reports, as the function builds, whether the item is
  the one the pointer or the keys are on, or leads to it through a
  submenu, as the menu's last pass built it: the function builds again as
  the highlight moves, so that it can show what goes with the item, as a
  preview of the theme highlighted. Call `ID` first, and give the items
  IDs when the items built change with the highlight: a drawn item's ID
  also keys its row, whose highlight then stays on it.
- `Menu.MaxWidth(w)` keeps the menu at most `w` wide, its submenus too
  unless they set their own: labels, subtitles and trailing texts too long
  for it end with an ellipsis, while shortcuts, icons and marks keep their
  room. Assistive technology, `tt.Menu` and typing see each label whole,
  and a `Tooltip` can show it.

### Every menu of a window

`Context.SetMenuPresentation(ui.PopoverMenus)` draws every popup menu of
the window so, not only those given with `PopoverMenu`: the menus of
`ContextMenu` and `Menu`, the ones MyGo's widgets build with them (a
[table](table.md)'s header and cell menus, a toolbar's overflow menu, the
code editor's), and the editing menu of text inputs and selectable text.
An app with a look of its own then shows no menu of the desktop's beside
its own, and its menu functions do not change. It stays until set again,
so the view can follow a setting of the app's:

```go
func (app *App) View(c *ui.Context) {
	if app.settings.systemMenus {
		c.SetMenuPresentation(ui.SystemMenus) // the default
	} else {
		c.SetMenuPresentation(ui.PopoverMenus)
	}
	// ...
}
```

`PopoverMenu` and `PopoverMenuButton` draw their menus in the window
whatever it is. Only the menus MyGo builds follow it: the menu bar, the
tray's menu and the Dock's stay the system's, and a [web
view](web-view.md)'s are the web view's own.

## Without a look

`ui.ContextMenuBase` gives an element a context menu without a look,
shown while a `*bool` is true: it opens as `PopoverMenu`'s does, where
the element is right-clicked or below it for the menu key, and has its
keys, and its `Popup` builds the menu as
[`MenuBase`](menu-button.md#without-a-look)'s does, with items of your
own layout:

```go
m := ui.ContextMenuBase(c, row, &app.rowMenu)
m.Popup(func(panel ui.Element) {
	panel.Padding(4).Radius(10).Background(t.Background).Border(1, t.Border)
	for _, label := range app.labels {
		item := m.CheckItem(note.has(label)).Gap(8).Padding(6, 10).Radius(6)
		if item.Highlighted() {
			item.Background(t.SurfaceHover)
		}
		item.Children(func() {
			ui.Box(c).Size(10, 10).Radius(5).Background(label.Color)
			ui.Text(c, label.Name)
		})
		if item.Clicked() {
			note.toggle(label)
		}
	}
})
```
