# Table

<!-- preview table 310 -->

`ui.Table` puts a [list](list.md)'s rows in columns, under a header of
`ui.TableColumn`s, each with a title, a width (or 0 to share the room the
others leave) and an alignment; `cell` builds the content of a row's
column, usually a `Text`. It builds only the rows in view, as a list does,
and rows are as high as their tallest cell, so text that wraps makes its
row taller.

```go
cols := []ui.TableColumn{{Title: "Name"}, {Title: "Size", Width: 90, Align: ui.End}}
app.files.Selected = &app.file
if ui.Table(c, &app.files, cols, len(files), func(row, col int) {
	switch col {
	case 0:
		ui.Text(c, files[row].Name).SingleLine()
	case 1:
		ui.Text(c, files[row].Size())
	}
}).Grow(1).Submitted() {
	app.open(files[app.file])
}
```

## Choosing rows

The table takes the same `ListState` as a list, or `nil`, and takes the
keyboard focus for it: with `Selected`, a click or the keys choose a row,
with `Selection` several, and `Submitted` reports a double click or Enter;
`Key`, `Label`, `Reorder`, `DragRows` and the rest work as for a list. A row `Header`
names spans every column, which `cell` builds as column 0, and stays at the
top while its section's rows scroll under it.

Chosen and hovered rows are square from edge to edge, as in Finder's list
view; `RoundRows` rounds their corners at the theme's `Radius`, as a list's
rows are, for an app whose other lists are rounded.

## Choosing cells

With `Selection`, `CellSelect` makes the table choose a block of cells, as a
spreadsheet does: the rows chosen, times the columns from `Cells.Anchor` to
`Cells.Cursor` in the order shown (`Cells.All` for every column). It is one
block: separate ranges of cells, as Excel's Cmd-drag adds, are not chosen.

- **Clicks.** A click on a cell chooses its row and its column; a
  Shift-click extends the rows and the columns to it; a Cmd-click (Ctrl-click
  on Linux and Windows) adds its row or takes it out and keeps the columns.
  A press on a cell outside the block chooses as the pointer goes down; a
  press inside the block waits for the release.
- **Dragging** over cells chooses those it goes over, and scrolls the table
  near its edges, down and sideways.
- **Keys.** Left and Right move `Cells.Cursor`, Shift with them extends the
  columns; the keys that move the rows leave one column unless with Shift;
  Cmd+A chooses every row and every column. Right to left, Left goes to the
  column on the left of the screen. In an `OutlineTable`, Left and Right
  still open and close the rows, and the columns are chosen with the pointer.
- **Right-click.** On a cell outside the block, it chooses that cell before
  the context menu opens; inside, the block stays. `MenuKeepsSelection`
  leaves the block as it is.
- **Row headers.** A `RowHeader` column, as a spreadsheet's row numbers,
  chooses whole rows: a click or a drag on its cells chooses rows with every
  column, and a click on its header every row. It does not sort.

The cells of the block show as a chosen row does, in the accent color with
the accent's text color, as do the row headers of whole rows chosen, and the
cell at `Cells.Cursor` in the row the keys move from has a border while the
table has the focus. A cell holding the focus, as an editor in it, shows as
it is. Chosen rows are not tinted whole. `SelectedColumns` returns the
block's column IDs in the order shown, and `SelectColumns` chooses every row
and a range of columns, for the app's own commands. `CellSelected(row, col)`
tells a cell builder whether its cell is chosen, for text of its own color
to show on the accent. A cell that reads its
row's double click with `RowDoubleClicked` leaves the click to the row, so
the block follows it.

`CellsStyle` paints the block in colors of its own rather than the accent,
as a spreadsheet's two levels and stripes: `Cell` fills the block's cells,
`CellText` colors their text, `Row` fills the rest of the rows chosen, and
`Stripe` fills every other row. A zero color paints none. The cursor's
border takes `CellText` in the block and `Cell` out of it, or the accent
and its text where those are zero.

```go
t := c.Theme()
app.grid.CellSelect = true
app.grid.CellsStyle = ui.CellsStyle{
	Cell:     t.Accent,
	CellText: t.AccentText,
	Row:      t.Accent.Alpha(0.12),
	Stripe:   t.Text.Alpha(0.04),
}
```

```go
cols := []ui.TableColumn{{Title: "#", RowHeader: true, Width: 48, Fixed: true}, {Title: "Name"}, {Title: "Size", Width: 90}}
app.grid.Selection, app.grid.CellSelect = &app.chosen, true
ui.Table(c, &app.grid, cols, len(rows), func(row, col int) {
	text, color := rows[row][col], c.Theme().Text
	if text == "" {
		text, color = "empty", c.Theme().TextMuted
	}
	if app.grid.CellSelected(row, col) {
		color = c.Theme().AccentText // muted does not show on the accent
	}
	ui.Text(c, text).TextColor(color).SingleLine()
}).Grow(1)
```

## Arranging columns

The user arranges the columns, as in Finder:

- **Resizing.** Dragging the right edge of a column's header resizes the
  column, between its `MinWidth` and `MaxWidth`, and a double click there
  fits it to its cells. The columns before it that share the room left keep
  their widths, for the edge to follow the pointer.
- **Moving.** Dragging a header moves its column among the others.
- **Scrolling sideways.** Columns wider than the table scroll sideways, the
  header with them.
- **Fixed columns.** A `Fixed` column stays where it is, as wide as it is.

`ListState.Columns` keeps the order and the widths, by the columns' `ID`s
(their titles by default): set it from saved settings to restore them, and
save it as it changes.

## Sorting

With `ListState.Sort`, a click on the header of a `Sortable` column sorts
the rows by it, ascending, and a second click reverses the order: the header
shows an arrow, assistive technology reads the order, and the table's
`Changed` reports it. The table shows the rows in the order `cell` gets
them, so sort them as it says, keyed by `Key` for the choice to follow its
rows:

```go
cols := []ui.TableColumn{{Title: "Name", Sortable: true}, {Title: "Size", Width: 90, Align: ui.End, Sortable: true}}
app.files.Sort = &app.sort
sorted := sortFiles(app.files, app.sort) // by app.sort.Column, Descending or not
app.files.Key = func(i int) any { return sorted[i].ID }
```

To sort by several columns, set `ListState.SortBy`. A click still sorts by
one column, and reverses it when it was first; a Cmd-click (Ctrl-click on
Linux and Windows) or a Shift-click, as in web data grids, adds the column
after those sorted by, ascending; another makes it descending, and a third
takes it out. The headers show their direction and, when there are several,
their place. `SortBy` gets a new slice as it changes, so one the app kept
stays as it was. With `Sort` set too, the table keeps it the first of
`SortBy`, the zero `SortOrder` when it sorts by none, so code that reads the
column sorting first goes on reading `Sort`:

```go
app.files.SortBy = &app.sortBy // []ui.SortOrder, the first first
sorted := sortFiles(app.files, app.sortBy...)
```

## Header menu

`ListState.HeaderMenu` gives the header a context menu, built as
[ContextMenu](context-menu.md)'s is. `col` is the index of the column
clicked in the columns given to `Table`, whatever order the user moved them
into, or -1 for the part of the header past the columns, as AppKit's
`clickedColumn`. A menu built without items does not open:

```go
app.files.HeaderMenu = func(m *ui.Menu, col int) {
	if col >= 0 && m.Item("Hide Column").Chosen() {
		app.hide(cols[col].ID)
	}
	if m.Item("Show All Columns").Chosen() {
		app.showAll()
	}
}
```

## Header cells

`ListState.HeaderCell` is called as the table builds each column's header
cell, with the cell and the column's index, as `HeaderMenu`'s: what it
builds goes in the cell after the title and the sort arrow, and it may
give the cell a tooltip or a card of its own:

```go
app.files.HeaderCell = func(cell ui.Element, col int) {
	cell.Tooltip(app.columnTypes[col])
}
```

## Cell menu

`ListState.CellMenu` gives the cells a context menu over the whole cell,
padding included, built as [ContextMenu](context-menu.md)'s is. `row` and
`col` are as the cell builder receives them. A menu built without items does
not open, and a context menu of what a cell holds takes the right-click over
it instead:

```go
app.grid.CellMenu = func(m *ui.Menu, row, col int) {
	if m.Item("Copy").Chosen() {
		app.copy(row, col)
	}
}
```

## Renaming in place

An [editable text](editable-text.md) in a cell is renamed in place, as
Finder's file names: Return on macOS and F2 elsewhere edit the chosen row's
text, as does a click on it once the row is chosen.

```go
ui.Table(c, &app.files, cols, len(files), func(row, col int) {
	if col == 0 && ui.EditableText(c, &files[row].Name).Changed() {
		app.rename(files[row])
	}
})
```

## Accessibility

Assistive technology sees a table, named by its `Label`, of rows named by
the text inside them, each saying which of all the rows it is, with column
headers that say the order the rows are sorted in, and the cells of a
`RowHeader` column naming their rows as row headers. With `CellSelect`,
the cells of the block say they are chosen.

For a tree of rows in columns, see [Outline](outline.md#in-columns).
