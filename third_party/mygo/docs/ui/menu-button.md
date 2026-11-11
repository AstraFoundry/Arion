# Menu button

<!-- preview menu-button 120 -->

`ui.MenuButton` creates a button showing a label and an arrow, which opens
a menu of the system's below it, as the system's pop-up buttons do: as the
pointer goes down on it, or for Enter, Space, Up or Down while it has the
keyboard focus. The function builds the items when the menu opens, and runs
again in the frame after one was chosen, where its `Chosen` reports it:

```go
ui.MenuButton(c, "Sort by", func(m *ui.Menu) {
	for _, by := range []string{"Name", "Date", "Size"} {
		if m.Item(by).Checked(app.sort == by).Chosen() {
			app.sort = by
		}
	}
})
```

## Items

Menus take the same items as [context menus](context-menu.md#items):
`Item`, with `Checked`, `Radio`, `Disabled` and `Shortcut`, `Separator`
and `Submenu`.

## Any button

`Element.Menu` gives the same menu to any button, as one showing only an
icon:

```go
ui.Button(c, "").Label("More").Children(func() { ui.Icon(c, dots) }).Menu(func(m *ui.Menu) {
	if m.Item("Duplicate").Chosen() {
		app.duplicate()
	}
	if m.Item("Delete").Chosen() {
		app.delete()
	}
})
```

In a [toolbar](toolbar.md), a menu button takes no face of its own until
hovered.

## Drawn in the window

`ui.PopoverMenuButton` takes the same function, and MyGo draws the menu in
the window with the theme's look, as
[`PopoverMenu`](context-menu.md#drawn-in-the-window) draws a context menu,
where the system would show its own: beside the popovers and buttons of
an app's own design, the system's menu is the one that looks out of place,
and a right-click's drawn menu and a ⋯ button's system menu, side by
side, look like two menus. `Element.PopoverMenuButton` gives such a menu to
any button, as `Element.Menu` does, and takes the same function as
`PopoverMenu`, so one function serves both. To draw every menu of a
window so, `Menu`'s included, see [every menu of a
window](context-menu.md#every-menu-of-a-window):

```go
menu := func(m *ui.Menu) {
	if m.Item("Rename").Shortcut(0, ui.KeyF2).Chosen() {
		app.renaming = i
	}
	if m.Item("Pinned").Checked(note.pinned).Chosen() {
		note.pinned = !note.pinned
	}
	m.Separator()
	if m.Item("Delete").Disabled(note.locked).Chosen() {
		app.delete(i)
	}
}
row.PopoverMenu(menu)
row.Children(func() {
	ui.Text(c, note.title).Grow(1)
	ui.Button(c, "").Label("More").Children(func() { ui.Icon(c, dots) }).PopoverMenuButton(menu)
})
```

It opens as the system's would, as the pointer goes down on the button,
and for Enter, Space, Up or Down while the button has the focus: below the
button, lined up with its start, or above it where the window has no room.
Pressing the button again closes it. It has the keys, the focus and the
look of `PopoverMenu`'s: it takes the focus as it opens, on its first
item when the keyboard opened it, and gives it back to the button as it
closes; Up, Down, Home, End and typing move among its items, Right and
Left open and close submenus, Enter and Space choose, and Escape, Tab and
presses outside close it. A choice reports `Chosen` as from
`PopoverMenu`.

`ui.PopoverMenuButtonOptions`, after the function, place the menu
elsewhere and let the app open and close it. `At` and `Self` attach the
menu as [`AttachTo`](overlays.md#your-own-overlays) does, its point
`Self` on the button's point `At`: `AnchorBottomEnd` and `AnchorTopEnd`
line it up with the
button's end, for a ⋯ button at the window's end, `AnchorTopStart` and
`AnchorBottomStart` put it above, and `AnchorTopEnd` and `AnchorTopStart`
after it. It still goes to the other side where the window has no room.
`Open`, a `*bool`, is whether the menu shows, as `MenuBase`'s: the button
sets it as the user opens and closes the menu, and the app sets it to open
the menu, or to close it, as when what it acts on goes away in the
background:

```go
more.PopoverMenuButton(menu, ui.PopoverMenuButtonOptions{
	At: ui.AnchorBottomEnd, Self: ui.AnchorTopEnd, Open: &app.moreOpen,
})

// Later, on an event from elsewhere, before the window updates:
app.moreOpen = false
```

A menu the app opens takes the focus, with no item highlighted, and gives
it back to the element that had it as it closes. A window shows one drawn
menu at a time: another opening closes it, setting `Open` to false.

Assistive technology sees the menu button, expanded while its menu
shows, and the menu with its menu items, check box items and radio items,
as it sees the system's. In a toolbar's overflow menu, choosing the button
opens its drawn menu below the overflow button.

A drawn menu's items are elements: in tests, `tt.Click` on the button
opens it, `tt.HasText` finds its items and `tt.Click` chooses one.
`tt.Menu`, `tt.ChooseMenuItem` and `tt.CloseMenu` work on it as on the
system's, so a test passes with either.

## Without a look

`ui.MenuBase` is a menu button without a look, for a menu whose items
have a layout of your own, as an avatar and a name, or a swatch: its
`Trigger` is the button, which takes children, and the menu shows while
a `*bool` is true. `Popup` builds the menu, as
[`PopoverBase`](popover.md#without-a-look) builds a panel: style the
panel, and build in it the items, each a row taking children, that
`Item`, `CheckItem`, `RadioItem` and `SubmenuItem` make, and the
submenus with `Submenu`. An item is `Highlighted` while the pointer or the
keys are on it, and its `Clicked` reports its choice, which closes the
menu:

```go
m := ui.MenuBase(c, &app.accountMenu)
m.Trigger.Gap(6).Padding(4, 8).Radius(999).Children(func() {
	ui.Avatar(c, app.user.Name, app.user.Photo).Size(24, 24)
	ui.Icon(c, chevron)
})
m.Popup(func(panel ui.Element) {
	panel.Margin(4, 0, 0, 0).Padding(4).Radius(10).Background(t.Background).Border(1, t.Border)
	for _, acct := range app.accounts {
		item := m.RadioItem(acct == app.user).Gap(8).Padding(6, 10).Radius(6)
		if item.Highlighted() {
			item.Background(t.SurfaceHover)
		}
		item.Children(func() {
			ui.Avatar(c, acct.Name, acct.Photo).Size(20, 20)
			ui.Text(c, acct.Name)
		})
		if item.Clicked() {
			app.switchTo(acct)
		}
	}
	more := m.SubmenuItem().Padding(6, 10).Children(func() { ui.Text(c, "Settings") })
	m.Submenu(more, func(panel ui.Element) {
		panel.Padding(4).Radius(10).Background(t.Background).Border(1, t.Border)
		if m.Item().Padding(6, 10).Children(func() { ui.Text(c, "Profile") }).Clicked() {
			app.openProfile()
		}
	})
})
```

It has the keys and the focus of `PopoverMenuButton`'s, which is built on
the same parts with the theme's look: it opens as the pointer goes down
on the button, and for Enter, Space, Up or Down while the button has the
focus, on its first item then; Up, Down, Home, End and typing the first
letters of an item move the highlight, past items `Disabled`; Right and
Left open and close submenus, the other way round right to left; Enter
and Space choose, Escape closes the submenu on top, and Tab and presses
outside close the menu, which gives the focus back to the button. Setting
the `*bool` opens and closes it too. Assistive technology sees the menu
button, expanded while its menu shows, a menu of menu items, check box
items and radio items, checked as `CheckItem` and `RadioItem` say, and
each submenu named by its item. Separators and headings are elements of
your own between the items. See [custom widgets](custom-widgets.md), and
[`ContextMenuBase`](context-menu.md#without-a-look) for a context menu.

An item that `KeepOpen` leaves the menu open as it is chosen, by the
pointer, Enter or Space: its `Clicked` reports the choice, and it stays
highlighted, as Radix's menus do when `onSelect` prevents the default.
`Group` makes a row of items that the keys take as one: Up and Down stop
at it once, on its item checked, else its first, and Left and Right move
among its items, round, the other way round right to left. A theme
switcher in an account menu is such a row of radio items:

```go
m.Group(func(row ui.Element) {
	row.Gap(2).Padding(4).Label("Theme")
	for _, th := range []string{"Light", "Dark", "System"} {
		item := m.RadioItem(app.theme == th).KeepOpen().Grow(1).Padding(4, 8).Radius(6)
		if item.Highlighted() {
			item.Background(t.SurfaceHover)
		}
		item.Children(func() { ui.Text(c, th) })
		if item.Clicked() {
			app.theme = th
		}
	}
})
```

Assistive technology sees a group, named by its `Label`, holding radio
items counted on their own (the items outside groups are counted
together).

## Accessibility

Assistive technology sees a menu button: `AXMenuButton` on macOS, a push
button with a menu on Linux, and a button that expands on Windows.
