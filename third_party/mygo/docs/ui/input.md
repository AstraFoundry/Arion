# Input

Elements report what the user did to them since the last frame, as you
build them: ask, and handle it where the element is built.

A `ui.Element` value is valid for one build pass. Its zero value is absent,
and queries on that value return false or zero. Use `Valid` to check optional
controls. Stale use is diagnosed in development and `Tester`; production
methods return empty results or ignore the operation.

Store a `ui.Handle` for focus or shortcuts across builds. Each window has
its own binding for the handle; use that window's Context when querying it.

```go
card := ui.Column(c).Padding(12).Radius(8).Focusable()
if card.Hovered() {
	card.Background(t.SurfaceHover)
}
if card.DoubleClicked() {
	app.open(item)
}
```

## The pointer

`Hovered`, `Pressed`, `Clicked`, `DoubleClicked`, `RightClicked`, `Dragged`
(how far the pointer moved since the last frame while pressing the
element) and `PointerPosition`. `ClickModifiers` returns the modifier keys
held for the last click, as Shift for a Shift-click. `c.Modifiers()` returns the modifier
keys held now, as one goes down or up on its own too, and the view draws
again as they change: a list showing each row's Cmd+1–9 while Cmd is held
reads it. `PassThrough` lets the
pointer through the element to what is below; `HitTest` lets it through
where a function of the pointer's position says, as a ring's hole or the
gaps between a pie's segments (see
[Drawing](drawing.md#fill-rules-and-hits));  `IgnoreInput` lets the
pointer through it and those inside it, and keeps the keyboard focus away
from them; with
[`AccessHidden`](accessibility.md#out-of-assistive-technologys-way), which
hides them from assistive technology, they are inert.

Elements that take the pointer give it to the innermost under it: a button
in a clickable row takes its own clicks.

`PressedInside` reports a press on an element or inside it without the
element taking the pointer, where `Pressed` and `Clicked` make it take
it: a table's cell notices an Alt-click while its row still chooses:

```go
cell := ui.Box(c).Children(func() { ui.Text(c, value) })
if cell.PressedInside() && c.Modifiers()&ui.Alt != 0 {
	app.inspect(row, col)
}
```

`Pressed` stays true while the pointer is held inside the element. A
release that comes with moves no build has read leaves the element
pressed for the build that reads them, with the pointer where it let go
(`PointerPosition`), and released in the build after. Observing
the start of a press rebuilds the view before painting, so a row chosen by
`if row.Pressed() || row.Clicked()` shows its new selection and focus colors
in that frame. Guard the choice against selecting the same item again.
`Clicked` reports activation on release inside the element, or through the
keyboard or assistive technology; buttons use that activation.

`Pressed`, `Clicked`, `Dragged`, `DragVelocity` and `DragEnded` follow the
primary button alone, as CSS's `:active` and the web's drag gestures do
(`event.button === 0`): a right or middle drag moves no card, slider or
split, and only the right-click (`RightClicked`) or [context
menu](context-menu.md) it opens comes of it. `PressedButton` returns the
button pressing the element, whichever it is (0 the primary, 1 the
secondary, 2 the middle), while the press lasts; with `PointerPosition`,
it builds a gesture of another button, as a canvas panning with the
middle button:

```go
x, y, _ := canvas.PointerPosition()
if b, ok := canvas.PressedButton(); ok && b == 2 {
	if app.panning {
		app.pan.X += x - app.last.X
		app.pan.Y += y - app.last.Y
	}
	app.panning, app.last = true, ui.Point{X: x, Y: y}
} else {
	app.panning = false
}
```

Every press comes from the primary pointer: the mouse, or the first finger
on a touchscreen; a second finger presses nothing, in browsers too
(`isPrimary`). [`HandleInput`](#every-key-as-it-comes) hears every
button go down and up, with `InputEvent.Button`.

`PressedPointer` tells what presses the element, and how hard, as
Pointer Events' `pointerType` and `pressure`: `PointerMouse` (a mouse or
a touchpad), `PointerPen` (a pen on a tablet or a screen) or
`PointerTouch` (a finger on a touchscreen), and a pressure from 0 to 1
as of the press's last move. A device that tells no pressure, as a
mouse, presses 0.5 while a button is down. `HandleInput` gets the same
on every `InputPointerDown`, `InputPointerMove` and `InputPointerUp`
(`InputEvent.Pointer` and `Pressure`), every move as it comes rather
than once a frame, and a pen pressing harder or softer where it stands
comes as a move too. A signature pad widens its stroke with a pen's
pressure, and goes by the pointer's speed for a mouse:

```go
ui.Box(c).Grow(1).Draw(app.paint).HandleInput(func(ev ui.InputEvent) bool {
	switch {
	case ev.Kind == ui.InputPointerDown, ev.Kind == ui.InputPointerMove && ev.Button == 0:
		width := app.widthBySpeed(ev.X, ev.Y, ev.Time) // a mouse's, or a finger's
		if ev.Pointer == ui.PointerPen {
			width = 1 + 5*ev.Pressure
		}
		app.stroke = append(app.stroke, stroke{ev.X, ev.Y, width})
		return true
	}
	return ev.Kind == ui.InputPointerUp
})
```

The systems tell the device of each press and move: AppKit's tablet
events on macOS (a mouse event's subtype and `pressure`, and
`tabletPoint:` between moves), the `WM_POINTER` messages of pens and
fingers on Windows, which Windows turns into the mouse messages it
marks as theirs, GDK's source device and pressure axis on Linux, and
`PointerEvent` in browsers. A `Tester` presses as a pen or a finger
after `UsePointer(ui.PointerPen, 0.6)`.

### A press the system cancels

The system can end a press without the pointer letting go, as Pointer
Events' `pointercancel`: a browser turning a touch into the page's
scrolling, a menu or another window taking the mouse while a button is
down, a pen or a finger Windows gives up on. The window losing the
keyboard ends a press the same way. Such a press commits nothing a release
would: it clicks nothing (`Clicked`, `RightClicked`), drops no value it
drags (`Drop`), ends no drag (`DragEnded`), opens no context menu, and a
`Draggable` element goes back to where the press took it. `PressCanceled`
reports it, in the frame after, for a gesture to undo what it began:

```go
dx, _, held := handle.Dragged()
switch {
case held:
	app.preview += dx
case handle.PressCanceled():
	app.preview = 0 // as it was
case app.preview != 0:
	app.width, app.preview = app.width+app.preview, 0
}
```

`HandleInput` still gets the press's `InputPointerUp`, so that what it
holds lets go, with `InputEvent.Canceled` set: roll back rather than
commit.

```go
ui.Box(c).Size(120, 40).HandleInput(func(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputPointerDown:
		app.armed = true
	case ui.InputPointerUp:
		if app.armed && !ev.Canceled {
			app.fire()
		}
		app.armed = false
	}
	return true
})
```

A `Tester` cancels a press with `CancelPress`.

While the pointer presses an element, the elements it was over as the
press began, as the row around a button, stay `Hovered` as long as it is
over them, as in CSS, and the others hover no more: dragging over other
elements does not light them up. A button that shows while its row is
hovered stays as it is pressed, to take its click:

```go
row := ui.Row(c).Padding(8)
hovered := row.Hovered()
row.Children(func() {
	ui.Text(c, item.Title).Grow(1)
	if hovered && ui.Button(c, "Remove").Clicked() {
		removed = item.ID
	}
})
```

Built only while the row is hovered, the button is not there for
assistive technology. `IgnoreInput` keeps it built, transparent and out of
the pointer's and the keyboard's way at rest, still read and pressed by
assistive technology (see [Accessibility](accessibility.md#out-of-the-pointers-way)):

```go
del := ui.Button(c, "Remove").IgnoreInput(!hovered)
if !hovered {
	del.Opacity(0)
}
```

## The keyboard focus

`Focusable` elements take the focus when clicked, and Tab and Shift+Tab
move it between them, with a focus ring when it moves by keyboard.
`AutoFocus` gives an element the focus when it appears, such as the first
field of a dialog, and `Focus` keeps it there while you call it; `Focused`,
`FocusVisible` and `FocusWithin` report it. Enter and Space press a focused
button, and Space toggles a focused check box, switch, toggle or radio
button. A focused link, or an element with `Role(ui.RoleLink)`, takes Enter
alone, as in a web page: Space scrolls the container around it.

`TabStop(false)` makes an element focusable but out of Tab's order, as
HTML's `tabindex="-1"`: it takes the focus from a click, from `Focus` and
`AutoFocus`, and from assistive technology, while Tab and Shift+Tab pass
over it, on to the controls inside it and after it. A card or a panel
takes the focus so to hold its keys without coming before its own
buttons; `TabStop(true)` puts it back in the order. In a [focus
group](#focus-groups), Tab enters at a member it stops at, while the
arrows still reach the others.

```go
card := ui.Column(c).Padding(12).TabStop(false).Children(func() {
	ui.Text(c, item.Title)
	ui.Button(c, "Open") // Tab goes here, past the card
})
if card.Shortcut(0, ui.KeyDelete) {
	app.remove(item)
}
```

`FocusLeft` reports that the focus left an element since the last frame:
it was on the element or inside it, and is now outside it, or nowhere, as
the window lost the keyboard. That is the web's `focusout` whose
`relatedTarget` the element does not contain. The overlays attached to the
element and to those inside it (`AttachTo`), as a combobox's list or a
button's popover, are inside it, as for `PressedOutside`, so a popup field
closes as the focus leaves it but not as it moves into its own list:

```go
var field ui.Element
combo := ui.Column(c).Children(func() {
	field = ui.TextInput(c, &app.query)
})
if combo.FocusLeft() {
	app.open = false
}
if app.open {
	ui.Overlay(c, func() {
		ui.Column(c).AttachTo(field, ui.AnchorBottomLeft, ui.AnchorTopLeft).Children(app.options)
	})
}
```

Where the focus went is what the elements report as they are built
(`Focused`, `FocusWithin`), and a [handle](#persistent-focus)'s
`FocusWithin(c)` before its control is built.

`c.FocusTakesText()` reports whether the focus takes text, typed or
composed: a text input or text area, or an editor of rich text or code,
that is not read only, or an element that takes text itself
(`HandleTextInput`, `TextCaret`). It is the web's guard of single-key
shortcuts (`isContentEditable`, an `input` or a `textarea`): a command
palette's letters step aside while a field types them.

```go
if !c.FocusTakesText() && c.Shortcut(0, ui.KeyK) {
	app.palette = true
}
```

## Persistent focus

Bind an app-owned handle to a control on each build:

```go
ui.TextInput(c.Key("search"), &app.query).Bind(&app.search)
```

Declare `search ui.Handle` in your app state. `app.search.Focus()` requests
focus and waits while the control is hidden. `CancelFocus()` cancels the
request. `Focused(c)` and `FocusWithin(c)` read the control's identity before
or after it is built. `c.Resolve(app.search)` returns only this pass's element.
`app.search.Bounds(c)` reads its box from the committed frame, including
between builds in an input callback. Closing a window cancels its focus
request.

A handle can bind in several windows. Queries take that window's Context;
use `Focus(c)` or `CancelFocus(c)` to select a window explicitly. The
no-argument focus form requires at most one open binding.

## Focus bound to data

```go
type pane int
const (none pane = iota; files; diff)

ui.Column(c.Key("files")).FocusBind(&app.pane, files).Children(func() { app.filesView(c) })
ui.Column(c.Key("diff")).FocusBind(&app.pane, diff).Children(func() { app.diffView(c) })

// In an action:
app.pane = diff

// Actual focus in this window:
focused := ui.FocusedValue(c, &app.pane)
```

`FocusBind` takes a pointer to a comparable field and a matching value.
Each value names one control in the window; reserve zero for no focus.
The field holds desired focus. A request waits if its control is hidden,
so use `FocusedValue` to read actual focus while it waits. Assigning zero
clears focus. User focus changes update the field when no request is pending.
Use separate fields for separate windows.

## Focus groups

The controls of a toolbar, a radio group, a segmented control or a tab list
are one stop of Tab, as they are natively: Tab moves the focus into the
group, to the control that had it last (or the radio button or tab chosen),
and on out of it, while the arrows move it among them, Home and End to the
first and last. `FocusGroup` does it for a container of your own, with the
arrows `ui.Horizontal`, `ui.Vertical` or both; a slider or a text input
inside keeps the arrows it takes.

```go
ui.Row(c).Gap(4).FocusGroup(ui.Horizontal).Children(func() {
	for _, tool := range tools {
		if ui.Button(c, tool.Name).Clicked() {
			app.tool = tool
		}
	}
})
```

## Shortcuts

`c.Shortcut(ui.Cmd, ui.KeyS)` reports a key pressed with exactly those
modifiers anywhere in the window, and `Element.Shortcut` only while the
element or one inside it has the focus, which comes first. A focused button
keeps Enter and Space, a link Enter, and a toggle Space, so
`c.Shortcut(0, ui.KeyEnter)` presses a dialog's default button wherever
else the focus is.

```go
if c.Shortcut(ui.Cmd, ui.KeyS) {
	app.save()
}
```

A button doing what a shortcut does tells screen readers its key with
[`KeyShortcuts`](accessibility.md#keyboard-shortcuts), as
`ui.Button(c, "Save").KeyShortcuts(ui.Cmd, ui.KeyS)`.

A key held down repeats, and each repeat comes as a press again, so an
action that steps runs while the key is held. `c.ShortcutRepeated()` tells
whether the key that the last shortcut to report one took came as such a
repeat, as the web's `KeyboardEvent.repeat`, whichever shortcut that was
(the window's, an element's, a list's or an overlay's), and an
`OnShortcut` action can ask it as it runs. A toggle takes the press alone:

```go
if c.Shortcut(0, ui.KeyUp) {
	app.count++ // and again while Up is held
}
if c.Shortcut(0, ui.KeyT) && !c.ShortcutRepeated() {
	app.panel = !app.panel // once a press
}
```

A repeat the action leaves is still taken: it goes to no other shortcut.
GTK flags no repeat, so on Linux a press of a key held already, with no
release between, is one, as X11 (with detectable auto-repeat, which GDK
turns on) and Wayland send them.

`ui.Cmd` is Command on macOS and Ctrl elsewhere. `ui.KeyBack` and
`ui.KeyForward` are the back and forward buttons of a mouse and the keys of
keyboards that have them, which a [router](navigation.md) takes. Shortcuts
of [menus](../menus.md) still work, and the Edit menu's roles (cut, copy,
paste, select all, undo, redo) act on the focused text input. Where the
focus takes no such command, as a list with nothing editable, the role
goes to the shortcuts of its key, so `c.Shortcut(ui.Cmd, ui.KeyV)` sees
Paste on macOS too, where the menu takes Command+V before the window. The
Edit menu enables a role's item only while the focus, or such a shortcut,
would take it.

Shortcuts work on layouts that type no Latin letters too: with Cmd, Ctrl,
Alt or Super held, a key typing a letter of no US key, as ы on a Russian
layout, stands for the key at its place on a US keyboard, so Cmd+ы is
`ui.KeyS`. Typed without those modifiers, it stays text.

A focused text input takes the editing keys of the platform first: on
macOS, Option and Command with the arrows and Backspace, and Control with
A, E, B, F, N, P, D, H and K, as in other Mac apps; elsewhere, it leaves Alt
and the arrows, which go back and forward, and the function keys.

A shortcut action can be declared before binding its control:

```go
app.filesView.OnShortcut(c, ui.Cmd, ui.KeyK, app.openSelected)
ui.List(c.Key("files"), &app.list, len(app.files)).Bind(&app.filesView).
    Rows(func(row ui.ListRow) {
        ui.Text(row.Context, app.files[row.Index].Name)
    })
```

`Handle.OnShortcut` runs after construction only when the window built an
enabled control for that handle. Hidden controls take no command.
`c.OnShortcut` declares an action in the current parent scope; the root
Context handles keys left by the focused control and active overlays.
Use `Element.OnShortcut` for an action inside an element's focus subtree.
`c.OnWindowShortcut` declares the window's action wherever `c` is, as
`c.Shortcut` reports it: a component built deep in the view, as a hint
that shows its key and runs it, hears it while the focus is anywhere else,
or nowhere. Built inside a dialog, it runs while the dialog shows; built
behind one, it waits.

```go
func shortcutHint(c *ui.Context, key ui.Key, label string, run func()) {
	c.OnWindowShortcut(ui.Cmd, key, run)
	ui.Text(c, label) // "⌘K"
}
```

## Keys held

`c.KeyDown(ui.KeyK)` reports whether a key is held down in the window, and
`c.KeysDown()` lists the keys held, in the order they went down, whichever
element has the focus and whether or not it took them: they only watch,
so the focused field still types. A view that reads them draws again as
they change. A cheat sheet of shortcuts lights the key caps held while its
search field has the focus, and fades the rows that use none of them:

```go
ui.SearchField(c, &app.query).Label("Search shortcuts").AutoFocus()
held, mods := c.KeysDown(), c.Modifiers()
for _, s := range app.shortcuts {
	row := shortcutRow(c, s, held, mods) // lights the caps held
	if (len(held) > 0 || mods != 0) && !s.uses(held, mods) {
		row.Opacity(0.4)
	}
}
```

Keys count as shortcuts take them, so Cmd+ы on a Russian layout holds
`ui.KeyS`; the modifier keys are `c.Modifiers()`'s. A key whose press runs
a menu item's shortcut goes to the menu, not the window. The window lets
go of every key as it loses the keyboard to another window or app, where
their releases go, and of each key whose release the system keeps from
it: on macOS, and in a Mac's browsers, one let go of while Command is
held; on Windows, one held as a menu opens, as with Alt and a letter or
Alt and Space, whose release the menu takes; and one whose release an
input method takes, as it begins composing while the key is held. A
`Tester` holds keys with `KeyDown` and `KeyUp`, and `SetFocused(false)`
lets go of them.

## Input methods

Text inputs take text composed with input methods, which see the text
around the caret: macOS's press and hold replaces the letter it accents,
Japanese input methods convert typed text again, and others predict from
what comes before.

## Dropped files

`DroppedFiles` returns the paths of the files dropped on the element from
Finder, Explorer or a file manager, and `FileDragOver` reports files dragged
over it, to show it would take them. The window takes files only over such
elements, or anywhere when it has `OnFileDrop` listeners, which get the
files no element takes, with where they were dropped:

```go
zone := ui.Column(c).Size(260, 80).Border(1, t.Border)
if files := zone.DroppedFiles(); files != nil {
	app.files = files
}
if zone.FileDragOver() {
	zone.Border(2, t.Accent)
}
```

Values dragged within or between windows, and serialized data exchanged
with other applications, are [drag and drop](drag-and-drop.md).

## Scrolling and pinching

Scroll containers scroll with the wheel and the touchpad on their own. An
element that does something else with them, as a chart zooming and
panning its candles, gets them with `HandleInput`: `ui.InputScroll`
scrolls by `DX` and `DY` DIPs, positive down and to the right of the
content, with `X` and `Y` the pointer relative to the element; `Precise`
tells a touchpad, which scrolls by pixels, from a mouse wheel's notches.
`ui.InputZoom` is a touchpad's pinch, at `X` and `Y`, zooming by `Scale`:
2 shows the content twice as large as it was. macOS, Linux on Wayland and
browsers report pinches; Windows sends them as scrolling with Ctrl held,
so there a touchpad's scrolling with Ctrl comes as `ui.InputZoom` too,
while a mouse wheel's notches stay `ui.InputScroll` with `ui.Ctrl` in
`Mods`. On Linux a touchscreen scrolls and pinches as a touchpad does: a
finger dragging scrolls the content under it along with it (`Precise`),
two fingers pinch, between them, and scroll as they move together, a tap
clicks and a finger held still for half a second right-clicks. What the
element takes no scroll container around it scrolls with; what it
leaves, by returning false, they do.

```go
ui.Box(c).Fill().HandleInput(func(ev ui.InputEvent) bool {
	switch {
	case ev.Kind == ui.InputZoom:
		app.chart.zoom(ev.X, ev.Scale)
	case ev.Kind == ui.InputScroll && !ev.Precise:
		// A notch, which scrolls further on some platforms than on
		// others, zooms by a quarter.
		if ev.DY < 0 {
			app.chart.zoom(ev.X, 1.25)
		} else if ev.DY > 0 {
			app.chart.zoom(ev.X, 0.8)
		}
	case ev.Kind == ui.InputScroll:
		app.chart.pan(ev.DX)
	default:
		return false
	}
	return true
})
```

A `Tester` scrolls with `Scroll` (a touchpad), `Wheel` (a notch) and
`Pinch`.

## Every key, as it comes

Widgets that take every key themselves, as the
[terminal](../plugins/terminal.md) does, get their input as it comes with
`HandleInput`, before the next frame: keys pressed and released, text typed
and composed while they have the focus, the edit commands of the menus,
and the pointer pressed on them, moving over them or scrolling over them.
`InputEvent.PhysicalKey` is the key by its place on a US keyboard,
whatever the layout types, for keys that go by position, as a game's WASD.
The function reports whether it took the event; one it leaves goes on to
shortcuts, Tab, context menus and scroll containers as usual, and keys that
the window or an element around the focus handles with `Shortcut` go there
first. `TextCaret` turns on the system's input methods for such an element
while it has the focus, composing at the caret it gives; `InputMethod(false)`
turns them off again while keys still type text, as for Vim's normal mode,
except in browsers, which cannot. `c.ReadClipboard`,
`c.WriteClipboard` and `c.OpenURL` copy, paste and open links for them;
`c.OpenURLThen` opens a link too, and its function gets what came of it a
moment later, as the system opens it, with an error when no app could,
and the view builds a frame anew. `c.WriteClipboardThen` copies text the
same way, as `navigator.clipboard.writeText` resolves or rejects: its
function gets an error when the text did not get there, as when
another app holds the clipboard, or a browser refuses a write the user
did not act for, so that a copy button can show "Copied" or "Failed"
(a `Tester` refuses them after `FailClipboard`). `Services.ReadClipboardData` and
`WriteClipboardData` read and write several formats at once, as a rich
text editor copies its own, HTML and plain text, and
`ClipboardFormats` lists the clipboard's without reading them, as a
terminal decides what a paste reads. A text field of the app's
own gets `InputSelect` when a screen reader moves its selection: the runes
`Start` to `End` of the text its `Access` describes.

```go
ui.Box(c).Fill().Focusable().HandleInput(func(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown:
		return app.key(ev.Mods, ev.Key)
	case ui.InputText:
		app.insert(ev.Text)
		return true
	}
	return false
}).TextCaret(app.caretRect())

c.OpenURLThen(url, func(err error) {
	if err != nil {
		c.Toast("No app opens " + url)
	}
})

if ui.Button(c, "Copy").Clicked() {
	c.WriteClipboardThen(link, func(err error) {
		app.copied, app.copyFailed = err == nil, err != nil
	})
}
```

### Every key of the keyboard

`InputEvent.Code` is the key by its position on any keyboard, a `ui.KeyCode`:
it tells the sides of the modifiers (`ui.CodeShiftLeft`, `ui.CodeAltRight`,
`ui.CodeSuperLeft`) and names the keys `ui.Key` has no name for, as
`ui.CodeNumpad5` or `ui.CodeIntlBackslash`, the 102nd key beside the left
Shift. Its values are USB HID usage IDs of the keyboard page and its names
those of `KeyboardEvent.code`, with Super for Meta, so one table turns them
into a guest's scancodes.

`HandleInput` and `ObserveInput` get every key the system tells the window
of, as a web page's listeners do. That includes modifiers pressed and released
on their own, a lone tap of Super among them, Caps Lock, Num Lock, Scroll Lock,
Print Screen and Pause, and the keys whose press the layout types no `Key` for,
as a key held with AltGr. Those come as `ui.InputKeyDown` and `ui.InputKeyUp`
with `Key` being `ui.KeyUnknown`, and only `Code` names them. `Mods` includes
the modifier a press brings and leaves out the one a release takes. A handler
that picks its keys with a `switch` never notices them; one that reacts to any
key should check that `Key` is not `ui.KeyUnknown`.

```go
ui.Box(c).Grow(1).Focusable().AutoFocus().Bind(&app.screen).HandleInput(func(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown:
		return app.guest.press(ev.Code)
	case ui.InputKeyUp:
		return app.guest.release(ev.Code)
	}
	return false
})
if !app.screen.Focused(c) {
	app.guest.releaseAll() // what is let go of elsewhere never comes up here
}
```

What the function takes goes no further, as Alt, which otherwise opens the
menu bar on Windows and Linux, so return false for keys you don't use;
shortcuts keep working either way. `Tester.CodeDown` and `Tester.CodeUp` press
and let go of such keys in tests.

What a system tells has limits:

- Keys the desktop takes for itself never come, as Super where a launcher
  opens with it, or combinations a window manager binds.
- On macOS Caps Lock comes as a press and a release together as its lock
  changes, not as the key moves. Fn has no code.
- On Windows AltGr comes as `ui.CodeControlLeft` then `ui.CodeAltRight`, as
  Windows makes up the left Control, and `Mods` holds Ctrl and Alt. Print
  Screen comes only as a release, as Windows keeps its press. With both Shifts
  held, only the last one's release comes. The Windows key's release still
  opens the Start menu.
- In browsers the page gets what the browser leaves it: `Code` is the
  event's `code`.
- `WebView`s take the keys while they have the focus.

## Input inside a region

`ObserveInput` has a container see the input that comes to the elements
inside it, as an ancestor listening to the `keydown` events that bubble up
to it, without taking it: the keys pressed and released (with `Repeat`),
the text typed and composed and the menus' edit commands while the focus
is the container or inside it, and the pointer pressed and released over
it. A demo that stops playing at the first key pressed in it, or a panel
that closes its hint once one of its fields is typed in, use it:

```go
panel := ui.Column(c).Gap(8).ObserveInput(func(ev ui.InputEvent) {
	if ev.Kind == ui.InputKeyDown || ev.Kind == ui.InputText {
		app.hint = false
	}
})
panel.Children(func() {
	if app.hint {
		ui.Text(c, "Type a name and an email")
	}
	ui.TextInput(c, &app.name).Label("Name")
	ui.TextInput(c, &app.email).Label("Email")
})
```

The function runs before the focused element gets the input, as UI Events'
capture phase, the outermost container first, and a frame follows. The
input then goes on as it would: to the focused element's `HandleInput`,
the text input, the shortcuts and Tab, so the function sees the keys a
`Shortcut` takes too. It does not see the keys the system or the menus
take before the window does (an edit command comes instead), nor the input
of popovers, menus and dialogs the container opens, which are not inside
it. A key that moved the focus, as Tab, comes up where the focus went.

## See also

- [Context menus](context-menu.md) and [menu buttons](menu-button.md).
- [Tooltips](tooltip.md).
- Custom title bars: [windows with native UI](windows.md).
