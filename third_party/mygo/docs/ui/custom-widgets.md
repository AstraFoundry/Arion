# Custom widgets

Every widget is built on a base, the same widget without a look: the base
handles the pointer, the keyboard and the focus, and tells assistive
technology what it is, and leaves every color, size and shape to the
elements it returns, which you style as any other. Build a design of your
own on them, as headless component libraries do on the web.

| Base | What it does |
|---|---|
| `ButtonBase` | a row that takes the focus, and reports `Clicked` for the pointer, Enter and Space |
| `CheckboxBase`, `SwitchBase`, `ToggleBase` | a row that toggles a `*bool` |
| `CheckboxStateBase` | a check box that can be mixed, setting a `*ui.CheckState` (`CheckOff`, `CheckOn`, `CheckMixed`), which a press checks, or unchecks once checked ([Checkbox](checkbox.md#without-a-look)) |
| `RadioBase` | a row that selects its value into a `*T` |
| `SliderBase` | sets a `*float64` from where the pointer is across its content box, inside its padding, or up it (`Vertical`), and with the arrows, Page Up and Down, Home and End, by its `Step` |
| `TabsBase` | the tab `List`, whose `Tab`s choose a `*int`, with the arrows moving the choice and the focus past those disabled |
| `SegmentedBase` | a `Track` of `Segment`s, a radio group choosing a `*int` |
| `CollapsibleBase` | a `Trigger` that shows and hides what `Panel` builds, growing and shrinking it with `Progress`, which goes from 0 closed to 1 open |
| `AccordionBase` | a `Root` of sections, each `Item` a `CollapsibleBase` whose triggers the arrows, Home and End move the focus between ([Accordion](accordion.md#without-a-look)) |
| `TreeBase` | a `Root` of `Item`s, each a `Row` and a `Group` of the items inside it while its `*bool` is true, with the arrows, Home, End and typing moving the focus and opening and closing them ([Tree](tree.md#without-a-look)) |
| `SelectBase` | a `Trigger` opening a `Popup` of `Item`s choosing a `*T`, which the arrows highlight (`Highlighted`) and Enter chooses; `SetOpen` opens and closes it, and `SelectBaseOpen` takes the open flag from you |
| `ComboboxBase` | an `Input` whose text filters the `Item`s of a `Popup` below it, which the arrows highlight and Enter or a click chooses (`Chosen`), or shows a row of yours when none match ([Combobox](combobox.md#without-a-look)) |
| `PopoverBase`, `DialogBase` | a panel beside an anchor, or over a backdrop covering the window, that a press outside or Escape closes |
| `MenuBase`, `ContextMenuBase` | a menu button's `Trigger`, or an element's context menu, opening a `Popup` of `Item`s, `CheckItem`s, `RadioItem`s and `SubmenuItem`s with their `Submenu`s, which the pointer and the keys highlight (`Highlighted`) and choose (`Clicked`), closing the menu unless `KeepOpen`, and `Group`s of items the keys take as one stop ([Menu button](menu-button.md#without-a-look)) |
| `TooltipBase` | a tip beside an anchor while the pointer rests on it or the keyboard focus is on it, which a press or Escape hides ([Tooltip](tooltip.md#without-a-look)) |
| `ToastViewportBase`, `ToastBase` | the window's toasts in a viewport of your own, each a status with an `ActionButton` and a `CloseButton`, their time stopping while the pointer or the focus is on them ([Toast](toast.md#without-a-look)) |
| `TextInputBase`, `TextAreaBase` | text inputs without padding, background, border or corners, `ReadOnly` or not |

Charts have bases of their own: [package `ui/chart`](chart.md) has scales,
axes, marks and a selection without a look.

Base constructors return value parts with the same build lifetime as other
elements. Give stateful bases a key with `Context.Key` before construction.

A segmented control on `SegmentedBase`, and a select on `SelectBase`:

```go
t := c.Theme()
view := ui.SegmentedBase(c.Key("period"), &app.view, 3)
view.Track.Padding(3).Radius(999).Background(t.Surface).Children(func() {
	for i, name := range []string{"Day", "Week", "Month"} {
		seg := view.Segment(i).Padding(5, 14).Radius(999)
		if i == app.view {
			seg.Background(t.Background).Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.15))
		}
		seg.Children(func() { ui.Text(c, name) })
	}
})

sel := ui.SelectBase(c.Key("size"), &app.size)
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

Bases ring the element with the keyboard focus, as every element taking
the focus is, with the theme's ring: `FocusRingWidth` wide,
`FocusRingOffset` outside the element's edge, in `Focus` (see
[the focus ring](styling.md#the-focus-ring)). `FocusRingOffset(d)` moves
an element's ring, inside it for a negative `d`, as for a header in a box
that clips what goes past it, and `FocusRingColor` colors it.
`FocusRing(false)` turns the ring off for a widget that draws its own,
with `Painter.FocusRing` while `FocusVisible`.

A disabled element shows at half its opacity, with what is inside it,
bases too. `DimDisabled(false)` leaves that to you, for a widget that
shows being disabled in a look of its own: a card that stays readable
with the reason it is unavailable, or a select's options grayed your way.
Style it while `IsDisabled`:

```go
card := ui.RadioBase(c, &app.plan, p.ID).Disabled(!p.Available).DimDisabled(false)
card.Column().Gap(4).Padding(12).Radius(8).Border(1, t.Border).Children(func() {
	name := ui.Text(c, p.Name)
	if card.IsDisabled() {
		name.TextColor(t.TextMuted)
		ui.Text(c, p.DisabledReason).FontSize(12)
	}
})
```

A widget that must stay focusable while it is unavailable, for screen
readers to reach it, tells them with
[`AccessDisabled`](accessibility.md#unavailable-still-reachable) instead
of `Disabled`, and styles itself while it is.

## Widgets of your own

For text controls with their own buffer or selection model, use
[`HandleTextInput` and retained text geometry](text-input-client.md).

Build your own widgets from elements. An element keeps state of its own
from frame to frame with `ui.Local`:

```go
// Spoiler hides text until clicked.
func Spoiler(c *ui.Context, text string) {
	t := c.Theme()
	box := ui.Box(c).Padding(2, 6).Radius(4).Focusable()
	shown := ui.Local(box, "shown", func() bool { return false })
	if box.Clicked() {
		*shown = !*shown
	}
	if *shown {
		box.Background(t.Surface)
	} else {
		box.Background(t.Text) // the color of the text, hiding it
	}
	box.Children(func() { ui.Text(c, text) })
}
```

### Stopping what a widget started

What `Local`'s `init` makes is the element's. When a frame no longer
builds the element, its state goes, and a value that is an `io.Closer`
(`T` or `*T` has `Close() error`) is closed, as the web aborts an
`AbortController` when an element is removed. A widget that starts work
in the background, a streaming request, a ticker or a subscription,
stops it in `Close`:

```go
type stream struct{ cancel context.CancelFunc }

func (s *stream) Close() error { s.cancel(); return nil }

// Feed shows the lines read from url while it is in the view.
func Feed(c *ui.Context, win *mygo.Window, app *App, url string) {
	e := ui.Column(c)
	ui.Local(e, "stream", func() *stream {
		ctx, cancel := context.WithCancel(context.Background())
		go read(ctx, url, func(line string) {
			win.Update(func() { app.lines = append(app.lines, line) })
		})
		return &stream{cancel}
	})
	e.Children(func() {
		for _, line := range app.lines {
			ui.Text(c, line)
		}
	})
}
```

- `Close` runs once, on the main thread, after the frame that no longer
  built the element; an element's last `Local` made closes first, as
  deferred calls run. The window waits for it: cancel the work and return,
  rather than wait for the work to stop.
- `Close` may change the app's state, on the main thread; ask for the
  frame that shows it with `Window.Invalidate`, or `Invalidate` of the
  `c.Services()` the widget kept.
- An element leaving with an exit
  [transition](transitions.md#coming-and-going) closes its values, and its
  children's, once the exit ended, as its copy is gone: the copy paints
  what the element had until then. Built again before that, the element
  starts anew, with `init`, and the values of the copy close.
- A [list](list.md) builds only the rows in view: a row scrolled out of
  view closes its values, and makes them anew with `init` as it comes
  back. Keep what outlives a row in the app's state, by its item.
- The pages a [router](navigation.md#pages-keep-their-state) keeps in
  its history keep their elements' values, which close as it lets the
  pages go.
- Every element's values close as its window closes, and so as the app
  quits.
- A `Close` that panics panics out of the frame, as the view would; the
  values left close after the next frame.
- A nil value is not closed, as a `*stream` that `init` left nil to start
  later, and `Close`'s error is ignored.
- A value the element must not close, as one the app shares, belongs in
  the app's state rather than in a `Local`.

A widget whose input changes what it edits reports it with `SetChanged`
on its element, as it builds: `Changed` then returns true for it, as for
the widgets' own. The [rich text editor](rich-text-editor.md) does so.

Give a widget a [role](accessibility.md#roles), a name and its
[states](accessibility.md#states) for assistive technology, take the keys
it needs with `Shortcut` on its element, or every key as it comes with
[`HandleInput`](input.md#every-key-as-it-comes), and paint what elements
do not with [`Draw`](drawing.md). Overlays of your own, as drawers, hover
cards and menus, are placed with `AttachTo` and close with
`PressedOutside` and `OverlayShortcut`; `Modal` keeps the focus in one:
see [overlays](overlays.md#your-own-overlays).
