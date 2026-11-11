# Grid

<!-- preview grid 260 -->

`ui.Grid` lays its children out in columns and rows, as CSS grid does: they
fill its cells row by row, and it adds rows as they need them. `Columns(3)`
makes three columns of equal width; `ColumnTracks` sets them one by one, as
`ui.Fixed(220)` DIPs, `ui.Fr(1)`, a share of the room the others leave, or
`ui.FitContent()`, as wide as their content:

```go
ui.Grid(c).Columns(3).Gap(12).Children(func() {
	for _, p := range app.photos {
		ui.Image(c, p).AspectRatio(1).Fit(ui.Cover).Radius(8)
	}
})

ui.Grid(c).ColumnTracks(ui.FitContent(), ui.Fr(1)).GapX(16).GapY(8).Children(func() {
	ui.Text(c, "Name").Bold().ColumnSpan(-1) // across every column
	ui.Text(c, "Email")
	ui.TextInput(c, &app.email)
	ui.Text(c, "Bio")
	ui.TextArea(c, &app.bio).RowSpan(2)
})
```

## Minimums

An `Fr` track shares the room even where its content is wider, as CSS's
`minmax(0, 1fr)`. `Min` gives a track a least size, as CSS's
`minmax(240px, 1fr)`: a main pane beside a side column that keeps a
readable width, the grid overflowing its box (and a scroll around it
scrolling) rather than squeezing the pane. An `Fr` track whose share would
be less keeps its minimum, and the other `Fr` tracks share what is left.
`MinContent` makes an `Fr` track no smaller than its children's content,
their min-content widths and `MinWidth`s (heights, for rows), as CSS's
plain `1fr`, which is `minmax(auto, 1fr)`: a toolbar's label column that
takes the room left but never gets narrower than its label's longest word,
or its `MinWidth`.

```go
ui.Grid(c).ColumnTracks(ui.Fr(1).Min(240), ui.Fixed(320)).Children(func() {
	mainPane(c)
	sideColumn(c)
})

ui.Grid(c).ColumnTracks(ui.Fr(1).MinContent(), ui.FitContent()).GapX(8).Children(func() {
	ui.Text(c, app.title).MinWidth(120)
	ui.Button(c, "Share")
})
```

A child spanning several `MinContent` tracks widens them in proportion
to their factors, as CSS does; a track that is `Fr` without `MinContent`,
or `Fixed`, takes none of it.

## Placing children

`ColumnSpan` and `RowSpan` make a child span tracks, a negative span every
track to the last, and `ColumnStart` and `RowStart` put it in a given column
and row, counting from 1. `RowTracks` and `Rows` set the first rows, which
`Fr` makes share the grid's height. A cell of a table or a grid of your
own that spans tracks tells screen readers it covers them too
(`aria-colspan`, `aria-rowspan`): see
[Grids](accessibility.md#grids).

## Sizing children

A child's percentages, as `FillWidth`, `WidthPercent` and `HeightPercent`,
their `Percent` minimums and maximums, and `LeftPercent` and the like, are
of its cell, the tracks it spans, not of the grid, as in CSS. A
`ui.FitContent()` track sizes to such a child's content as if it had no
size set, and the child then takes its share of the track.

## Aligning

Children stretch to fill their cells unless `JustifyItems` (horizontally)
and `AlignItems` (vertically) say otherwise, or `JustifySelf` and
`AlignSelf` for one; automatic margins center them. `AlignItems(ui.Baseline)`
lines up the first lines of text of the items of each row, as in a
[row](layout.md#alignment). Tracks that fit their
content share the room left over unless `Justify` or `AlignContent` place
them instead.

A grid builds all its children. For a grid of many items, built only as
they show, use a [grid view](grid-view.md); for labels beside controls, a
[form](form.md).
