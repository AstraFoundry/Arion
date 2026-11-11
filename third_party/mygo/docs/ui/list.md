# List

<!-- preview list 290 -->

`ui.List` shows rows of a collection of any length: it builds only the rows
in view, and a few beyond, so a list of millions of rows is as fast as one
of ten. Rows take the height of their content, which may differ from row to
row: the list measures rows as they show and estimates the others from
them.

```go
ui.List(c, nil, len(app.files), func(i int) {
	ui.Text(c, app.files[i].Name).Padding(6, 12)
}).Grow(1)
```

Give a list a size, or grow it in its parent. A list without a size, or
with only a `MaxHeight`, is as high as its rows.

## Layout

`Gap` spaces the rows, `Padding` pads the content and scrolls with it, and
`Justify(ui.End)` puts rows that do not fill the list at its bottom, as a
chat's first messages. What else you build in a list shows while it has no
rows, as a message that it is empty:

```go
ui.List(c, nil, len(results), func(i int) {
	ui.Text(c, results[i].Title).Padding(6, 12)
}).Grow(1).Children(func() {
	if len(results) == 0 {
		ui.Text(c, "No results").TextColor(c.Theme().TextMuted).Padding(12)
	}
})
```

## Rows wider than the list

`RowWidth(w)` makes the rows at least `w` DIPs wide, as the columns of a
table ask: where they are wider than the list, it scrolls sideways too, and
where they are not, they grow to its width. A header above the rows goes
with them through `ScrollWith`; clip it, so that what scrolls out of it
does not show. `ui.Table` does this for its columns.

```go
head := ui.Row(c).Clip().Children(app.titles)
list := ui.List(c, &app.files, len(app.files)).Grow(1).RowWidth(app.columnsWidth())
list.Rows(app.row)
head.ScrollWith(list)
```

## List state

A list keeps its place by a row rather than by an offset, so the rows in
view stay where they are while the rows around them are measured, grow,
come and go, and its offsets count millions of rows to a fraction of a DIP.
Give it a `ui.ListState` in your state, in place of `nil`, to say how it
behaves and to move it:

- `Key` returns an identity for the item of a row, such as its ID: the
  state of a row (focus, text being edited, …) follows its item, and the
  list keeps its place, and its choice, when rows are added or removed
  above them, as when older messages load.
- `KeepOffset` keeps a list with a `Key` at its offset as its rows move,
  rather than with the item at its top, for rows sorted again as they
  change, as a process monitor's: the rows' state and the choice still
  follow their items.
- `FollowEnd` starts the list at its end, and keeps the end in view as rows
  come or grow, until the user scrolls away from it, as a chat or a log
  does; scrolling back to the end follows it again. `FollowAnimation`
  makes it glide there, as a spring (`&ui.Smooth`) or a curve, rather
  than show the end at once: what is added comes up from below, as the
  chat apps scroll, and the user scrolling stops it. Each layout reads
  `FollowEnd` as it is then, and `HoldPlace` keeps the list where it is
  for the next layout alone: call it in the frame a click opens a fold in
  the last message, and the line clicked stays under the pointer rather
  than following the end as the row grows.
- `Header` names the rows heading sections: the header of the section at
  the top stays pinned there while its rows scroll under it, until the next
  header pushes it away. Give headers a background.
- `Label` returns the text of a row: typing its first letters while the
  list has the focus chooses it, as in Finder or Explorer, and assistive
  technology reads it as the row's name.

Give each list a `ListState` of its own, and set its fields before
building the list.

## Choosing rows

- `Selected` lets a click, or Up, Down, Home and End while the list has the
  keyboard focus (and Page Up and Page Down on Linux and Windows), choose a
  row, which shows in the accent color; the list's `Changed` reports a new
  choice, and `Submitted` a double click or Enter.
- `Selection` lets the user choose several rows, held in a
  `ui.Selection[K]` by the keys of their items (`Key`), or by their indices
  without one: a click chooses one, Cmd-click (Ctrl-click on Linux and
  Windows) adds a row or takes it out, and Shift-click or Shift with the
  keys chooses the rows from the one last chosen; Cmd+A chooses all. On
  Linux and Windows, Ctrl with the keys moves without choosing, and
  Ctrl+Space adds the row there or takes it out. `Selected`, set as well, is
  the row last chosen.

Files to choose several of, by their paths:

```go
type fileList struct {
	files  []File
	chosen ui.Selection[string]
	list   ui.ListState
}

func (app *fileList) view(c *ui.Context) {
	app.list.Key = func(i int) any { return app.files[i].Path }
	app.list.Label = func(i int) string { return app.files[i].Name }
	app.list.Selection = &app.chosen
	ui.List(c, &app.list, len(app.files), func(i int) {
		ui.Text(c, app.files[i].Name).Padding(6, 12)
	}).Grow(1)
	if ui.Button(c, fmt.Sprintf("Delete %d", app.chosen.Len())).Clicked() {
		for path := range app.chosen.All() {
			app.delete(path)
		}
		app.chosen.Clear()
	}
}
```

The keys of items gone from the list stay in the selection until you take
them out. `Element.ClickModifiers` gives the modifier keys of any click, for
widgets of your own that do the same.

In a list with `Selection`, a right-click on a row outside the choice
chooses it before its context menu opens, as Finder and Qt do, so the menu
acts on the row it was opened on; on a chosen row it leaves the choice as it
is. `MenuKeepsSelection` turns this off: the menu opens on the row clicked,
whose builder knows its row, and the choice stays. A list with `Selected`
alone never chooses on a right-click.

```go
app.files.Selection = &app.chosen
app.files.MenuKeepsSelection = true // the menu acts on the row clicked alone
```

`DragSelect`, with `Selection`, chooses rows as the pointer drags over them
from the row it pressed, as a Shift-click on the row under the pointer
would (with Cmd, or Ctrl on Linux and Windows, held as the press began,
besides the rows chosen), scrolling the list as the pointer nears or passes
its top or bottom. A press on a row not chosen chooses it as the pointer goes
down, as a spreadsheet does, rather than at the release; a press on a row
chosen waits for the release, so a drag of the choice keeps it.
`DragSelecting` reports a drag choosing rows while the pointer holds it, for
work that follows the choice, as a sum of the rows chosen, to wait until it
ends. A list cannot both `DragSelect` and `Reorder` (or `DragRows`).

`ExtendTo(row)` chooses the rows from the one Shift extends from to `row`,
as a Shift-click on it does, for keys of your own, as one that extends the
choice to the last row:

```go
if app.extendToEnd {
	app.list.ExtendTo(len(app.files) - 1) // after the list is built
}
```

What a row holds can tell which part of it a double click landed on without
taking the click from the row, which goes on choosing: `RowDoubleClicked`
reports a double click on the row with the pointer over the element.
`DoubleClicked` on the element would take the click, and the row would not
choose.

```go
ui.List(c, &app.list, len(app.files), func(i int) {
	name := ui.Text(c, app.files[i].Name).Padding(6, 12)
	if name.RowDoubleClicked() {
		app.rename(i)
	}
})
```

## Scrolling

`ScrollTo(row, align)` shows a row at the `ui.Start`, `ui.Center` or
`ui.End` of the list, `ScrollIntoView(row)` scrolls as little as shows it,
and `ScrollToEnd` scrolls to the end, rows not built yet included: the list
scrolls as the frame is laid out, to where the row is once measured.
`Visible` returns the first and last rows in view and `AtEnd` whether the
end shows, to load more as the list nears the end of what was loaded.

A chat:

```go
app.chat.Key = func(i int) any { return app.messages[i].ID }
app.chat.FollowEnd = true
app.chat.FollowAnimation = &ui.Smooth
if first, _ := app.chat.Visible(); first < 5 && app.more {
	app.loadOlder() // above the messages in view, which stay put
}
if !app.chat.AtEnd() && ui.Button(c, "Jump to latest").Clicked() {
	app.chat.ScrollToEnd()
}
ui.List(c, &app.chat, len(app.messages), func(i int) {
	message(c, app.messages[i])
}).Grow(1).Justify(ui.End)
```

## Reordering

`Reorder` lets the user drag rows to another place in the list: the rows
chosen, when the row dragged is one of them, else that row, which the list
passes in order, and the row they go before, the number of rows for the
end. Move them there: the list shows where they would go as they are
dragged, and scrolls near its edges.

```go
app.list.Reorder = func(rows []int, to int) {
	app.items = move(app.items, rows, to)
}
```

Rows that are buttons, which open what they show as they are clicked,
reorder too: a press on a row's button drags the row once the pointer
moves, and clicks the button if it does not.

## Dragging rows out

`DragRows` lets the user drag rows out of the list, to drop them elsewhere:
the rows chosen, when the row dragged is one of them, else that row, drag
the value `DragRows` returns, which `ui.Drop` and `ui.DragOver` take as
they take any value dragged. The list itself takes nothing and shows no
line between its rows, so a target inside it, as a folder in a list of
files, or one around it, gets the value wherever the pointer is:

```go
app.files.DragRows = func(rows []int) any {
	return app.pathsOf(rows) // []string
}
// In the row of a folder:
if paths, ok := ui.Drop[[]string](row); ok {
	app.move(paths, folder)
}
```

Return nil for rows that do not drag. A list cannot both `Reorder` and
`DragRows`.

See [drag and drop](drag-and-drop.md).

## The focus

A list keeps its control identity in `ListState.Handle`. Focus queries work
before or after constructing the list, and a focus request waits while the
list is hidden:

```go
app.list.Handle.OnShortcut(c, ui.Cmd, ui.KeyK, app.openCurrentFile)
ui.List(c.Key("files"), &app.list, len(app.files)).Grow(1).
    Rows(func(row ui.ListRow) {
        text := ui.Text(row.Context, app.files[row.Index].Name).Padding(6, 12)
        if row.Selected() && row.ListFocused() {
            text.TextColor(c.Theme().Accent)
        }
    })
```

Call `app.list.Handle.Focus()` on the UI thread to request focus, or
`Focus(c)` to select a window explicitly. `Focused(c)` reports focus on the
list and `FocusWithin(c)` includes its descendants. A handle's shortcut
action runs after construction only if its enabled control is present.

`ListRow` supplies the current Context, row index, selection and list focus.
`ListState.Selected` or `Selection` enables selection. `ItemKey` gives each
item a stable key. Configure them before calling Rows.

For custom row builders using `List(c, state, n, func(i int))`, the
ListState polling helpers resolve this pass's list after construction or
inside its row callback. Persistent focus and command handling use Handle.
A Table shares this identity through its ListState, and an Outline through
its `List` state.

The row holding keyboard focus stays built when it scrolls out of view,
so editing continues and Tab can scroll the next focused row into view.

## Accessibility

Assistive technology sees a list, named by its `Label`, of list items named
by the text inside them or the state's `Label`, each saying which of all
the rows it is, "5 of 10,000", although the list builds only those in view.
A list choosing its rows gives assistive technology the focus on the row
chosen: Up and Down are read as they move the choice, and focusing a row
chooses it. A screen reader moving out of view asks the list to scroll
there, which builds the rows it reaches. A list of rows of cells, as a
spreadsheet's, is a [grid](accessibility.md#grids) with
`Role(ui.RoleGrid)`: its rows are then rows of a table.

For rows in columns, see [Table](table.md); for a tree, [Outline](outline.md).
