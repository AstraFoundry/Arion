# Select

<!-- preview select 230 -->

`ui.Select` creates a drop-down choosing one of a list of strings into a
`*string`: a click on it, or Enter, Space, Up or Down while it has the
keyboard focus, opens its options below it, where the arrows, Home and End
move among them and Enter, Space or a click chooses one; Escape closes
them. `Changed` reports a new
choice.

```go
ui.Select(c, &app.plan, []string{"Free", "Pro", "Team"}).Label("Plan")
```

The popup shows above the select where there is no room below it, and gives
the focus back to the select as it closes. With more options than the window
has room for, it is as tall as the room and its options scroll.

For a long list that typing filters, use a [combobox](combobox.md); for a
few options shown at once, [radio buttons](radio.md).

## Without a look

`ui.SelectBase` is a select without a look, choosing a value of any
comparable type: a `Trigger` opening a `Popup` of `Item`s, which the arrows
highlight (`Highlighted`) and Enter chooses. `Highlight` moves the
highlight to an option, as a select of yours may as the user types its
first letters:

```go
sel := ui.SelectBase(c, &app.size)
sel.Trigger.Gap(6).Padding(6, 10).Radius(8).Border(1, t.Border).Children(func() {
	ui.Text(c, app.size)
	ui.Icon(c, chevron)
})
sel.Popup(func(panel ui.Element) {
	panel.Margin(4, 0, 0, 0).Padding(4).Radius(10).Background(t.Background).Border(1, t.Border)
	for _, size := range sizes {
		item := sel.Item(size).Padding(6, 10).Radius(6)
		if item.Highlighted() {
			item.Background(t.Accent).TextColor(t.AccentText)
		}
		item.Children(func() { ui.Text(c, size) })
	}
})
```

`SetOpen` opens and closes the popup, as typing on the closed trigger may
open it, highlighting the option typed, or Tab moving on close it; the
popup opens with the highlight on the choice, which `Highlight` then
moves, once the options are built if they are not yet:

```go
if !sel.Open() && sel.Trigger.Shortcut(0, ui.KeyL) {
	sel.SetOpen(true)
	sel.Highlight("L")
}
if !sel.Trigger.Focused() {
	sel.SetOpen(false) // Tab moved on
}
```

`ui.SelectBaseOpen` takes the open flag from you instead, as
[`PopoverBase`](popover.md) does: the popup shows while `*open` is true,
the select sets it as it opens and closes, and you may set it too. Owning
it, you can keep the popup a little longer as it closes, as a list that
grows out of its trigger and shrinks back into it, which takes no input
while it shrinks:

```go
// The select closed the list since the last pass: it stays until it
// has shrunk back into the trigger.
if !app.open && app.listShown {
	app.open, app.closing = true, true
}
sel := ui.SelectBaseOpen(c, &app.size, &app.open)
sel.Trigger.Height(32).Padding(0, 10).Children(func() { ui.Text(c, app.size) })
var target float32
if app.open && !app.closing {
	target = 1
}
grow := sel.Trigger.Animate("grow", target, 200*time.Millisecond)
if app.closing && grow == 0 {
	app.open, app.closing = false, false
}
panel := sel.Popup(func(panel ui.Element) {
	// From the trigger's height to the options', over the trigger.
	full := float32(len(sizes)) * 32
	panel.AttachTo(sel.Trigger, ui.AnchorTopStart, ui.AnchorTopStart).Height(32 + grow*(full-32)).ClipY().
		IgnoreInput(app.closing).Radius(8).Background(t.Background).Border(1, t.Border)
	for _, size := range sizes {
		sel.Item(size).Height(32).Padding(0, 10).Children(func() { ui.Text(c, size) })
	}
})
app.listShown = panel.Valid()
```

See [custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees a pop-up button whose value is the choice, named
by its `Label`, and, while it is open, a list box of its options
(`ui.RoleListBox`, `ui.RoleOption`), each saying which of how many it is.
As in a native select's popup, the option the arrows or the pointer are on
is the one selected, which the focus follows while the select keeps it;
assistive technology focusing an option highlights it, as the arrows do,
and pressing one chooses it, as a click does.
