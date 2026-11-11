# Testing

`ui.NewTester` runs a view without a window, as fast as a unit test: it
renders frames in memory, finds elements by their text, and clicks, types,
scrolls and presses keys.

```go
func TestCounter(t *testing.T) {
	app := &counter{}
	tt := ui.NewTester(app.view, 320, 240)
	if err := tt.Click("Increment"); err != nil {
		t.Fatal(err)
	}
	if app.n != 1 || !tt.HasText("1") {
		t.Errorf("count %d, texts %q", app.n, tt.Texts())
	}
}
```

Elements are found by their text, or by their `Label` for those without,
as an icon button: `tt.Click("Close")`, `tt.Find("Volume")`,
`tt.Focused("Name")`.

`Click`, `ClickWith` and `RightClick` check the first matching element
before sending input. They return an error if it has no visible area or
the center of that area would reach another control or nothing. They do
not scroll it into view or search for another uncovered point. On an
element without a context menu of its own, as a list's row, `RightClick`
opens the menu of the content the element holds at that point, as a
right-click there does; a control inside it that takes clicks or input
of its own is another control. `Find`
still returns a matching clipped box, which may have zero width or height;
`ClickAt` sends input at the given coordinates without a named target.

Two elements given the same `Key` under one parent share one state, so
that a click or the focus meant for one goes to the other: a tester
panics where the second key was given, as apps only log it.

## What a tester does

- **Pointer.** `tt.Click`, `tt.ClickAt`, `tt.ClickWith(ui.Shift, "Row 3")`,
  `tt.Press`, `tt.Move` and `tt.Release`, `tt.PressButton` and
  `tt.ReleaseButton` for another button, as `tt.PressButton(1, x, y)` for
  a right-button drag, `tt.CancelPress`, which ends the press as the
  system cancels one, with no click, `tt.Scroll`, which scrolls at
  once as a touchpad does, and `tt.Wheel`, a notch of a mouse wheel, which
  glides there as the clock advances (`tt.Advance`).
  `tt.UsePointer(ui.PointerPen, 0.6)` has the presses and moves after it
  come from a pen pressing that hard, or a finger (`ui.PointerTouch`).
- **Keyboard.** `tt.Key(ui.Cmd, ui.KeyS)` presses a key, `tt.KeyRepeat` presses
  it again as its auto-repeat does while it is held, `tt.TypeKey` a key
  with the text it types, and `tt.Type` types text into the focused
  element. `tt.Compose` shows the composition of an input method,
  `tt.Command` performs an edit command of the menus, such as `"copy"`, and
  `tt.TextCaret` returns where input methods would compose.
- **Menus.** `tt.RightClick` opens a context menu, which `tt.Menu` lists and
  `tt.ChooseMenuItem("Move to", "Archive")` chooses from, and `tt.CloseMenu`
  closes: the system's, or one [drawn in the
  window](context-menu.md#drawn-in-the-window), so a test passes with
  either.
- **The window.** `tt.SetSize`, `tt.SetScale`, `tt.SetZoom` (the content's
  [zoom](windows.md#zoom), as `Context.SetZoom`; the tester's coordinates,
  of `Find`, `ClickAt`, `Move` and `TextCaret`, stay the window's), `tt.SetDark`,
  `tt.SetPreferences` and `tt.SetDirection` (lay it out
  [right to left](right-to-left.md)) change the window and the desktop's
  settings; `ui.NewTesterWith(view, w, h, ui.TesterOptions{Preferences: p,
  Dark: true})` starts with the desktop's, which the first frame follows,
  as Media Queries 5 has the user's preferences apply from the first time
  styles resolve: a view mounted under reduced motion starts no animation
  in its first frames, which `NewTester` settles before a `SetPreferences`
  can come;
  `tt.SetFocused` takes the keyboard from the window and gives it back,
  and `tt.SetWindowActive` draws it as inactive and active again, as
  another window taking the user's input does
  ([inactive windows](windows.md#active-and-inactive-windows)), and
  `tt.SetWindowHidden` hides all of it from sight and shows it again
  ([hidden windows](windows.md#hidden-windows)).
- **What it shows.** `tt.Texts`, `tt.HasText`, `tt.Find` (the box of a
  text), `tt.Focused`, `tt.Cursor`, `tt.Clipboard`, `tt.OpenedURLs` and
  `tt.Announcements`, what the view asked screen readers to read out;
  `tt.FailOpenURL` makes the links it opens fail, as those no app opens,
  and `tt.FailClipboard` its writes of text to the clipboard, as a
  browser refuses them (`WriteClipboardThen`).
- **What assistive technology sees.** `tt.Access` and `tt.AccessByRole`
  read an element's role, name, value and states
  ([below](#what-assistive-technology-sees)).
- **Frames.** `tt.Frame` draws a frame, for what changed outside the view,
  and `tt.Image()` is the last frame, for snapshots; `ui.Render` draws a
  view once at a given scale. Inputs and `tt.Frame` run the frames the
  view asks for until it asks for no more, or 20 of them:
  `tt.FramePending` tells whether it still asks, as a view that animates
  or never settles does.
- **Time.** A tester has a clock of its own, `tt.Now()`, which stands
  still until `tt.Advance(d)` moves it on by `d`, running the frames due
  meanwhile at their times: one each refresh while something moves, and
  those of `c.After` and `Painter.After` when they come due. What depends
  on time follows it: animations, `Context.After`, `Context.Now`, the
  clicks that make a double click, the typing that undoes in one step,
  the caret's blinking, tooltips and toasts. So a test reads the same on
  a busy machine as on a fast one: never sleep to let something happen,
  advance the clock. `Advance` runs only the frames the view asks for:
  after changing what the view shows from outside it, call `tt.Frame`
  first. The undo steps of an `editor.Buffer` and a `richtext.Document`
  use the system's clock.

  ```go
  tt := ui.NewTester(app.view, 320, 240)
  tt.Click("More")
  tt.Advance(100 * time.Millisecond) // half way through a 200 ms animation
  tt.Advance(time.Second)            // and done
  if tt.FramePending() {
  	t.Error("the panel keeps asking for frames")
  }
  ```

## What assistive technology sees

`tt.AccessByRole(role, name)` returns what screen readers get of the
first element of the last frame with that role and accessible name (any
name for `""`), as Testing Library's `getByRole(role, { name })` and
Playwright's find it, and `tt.Access(name)` the first with that name
whatever its role. The `ui.AccessInfo` they return has the role, the
name, the value, the description and the `RoleDescription`, and the
states, as [accessibility](accessibility.md#states) lists them:
`Checked` and `Mixed` (`aria-checked`, `aria-pressed`), `Selected`
(`aria-selected`, the tab, tree item, row or grid cell chosen, with
`Selected` or `AccessSelected`),
`Expanded`, `HasPopup`, `Busy`, `Current`,
`AutoComplete` (`aria-autocomplete`), `Sort` (a column header's
`ui.SortDirection`), `Disabled`, `Invalid`, `Focused` (the element or
option the focus is on, `ActiveDescendant` included), `Controls` (the
names of the elements it controls, which `Access` finds), `DescribedBy`
and `ErrorMessage` (the names of the elements describing it and holding
its error), `Vertical` and `Horizontal` (`aria-orientation`), `Level`,
`PosInSet` and `SetSize` (`aria-posinset`, `aria-setsize`, as
`PositionInSet` gives them), `Multiselectable`, `RowCount`,
`ColumnCount`, `RowIndex` and `ColumnIndex` (of a
[grid](accessibility.md#grids)), `RowSpan` and `ColumnSpan` (of its
cells), `Min`, `Max` and `Now` of a range, and its `Bounds`. So a test
checks
that a state reaches assistive technology, not only that the view set
it:

```go
tt := ui.NewTester(app.view, 320, 240)
reply, ok := tt.AccessByRole(ui.RoleStatus, "Reply")
if !ok || !reply.Busy {
	t.Errorf("Reply while streaming: %+v, found %v", reply, ok)
}
if docs, _ := tt.AccessByRole(ui.RoleLink, "Docs"); docs.Current != ui.CurrentPage {
	t.Errorf("Docs is not the current page: %+v", docs)
}
```

The role is the one the element was given, by its widget or `Role`, or
the one found from what it does: a text, a button, a group. The tree is
built as screen readers would get it, whether or not one reads the
window, and reading it changes nothing: it builds no frame.

## Testing the built app

A tester runs a view inside `go test`. To test the app as built, with its
real windows, from another process, as an end-to-end test runner or an
agent does, import the [automation plugin](../plugins/automation.md) into
a test build: it reads the elements of every window as a tester's
`Find` does, by their labels or by their `Key`, acts on them, sends
input, and waits for frames as `FramePending` tells.
