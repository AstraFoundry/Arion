# Outline

<!-- preview outline 320 -->

`ui.Outline` shows a tree of items, as AppKit's outline view and Finder's
list view: items at the top, and below each item open its children,
indented, with an arrow that opens and closes it. The app gives the items at
the top and `children`, which returns an item's, nil for one without any;
the outline builds only the rows in view, as a [list](list.md) does, so
trees of any size are as fast as small ones.

```go
type fileTree struct {
	children map[string][]string // the paths in each directory
	files    ui.OutlineState[string]
	row      int
}

func (app *fileTree) view(c *ui.Context) {
	app.files.List.Selected = &app.row
	ui.Outline(c, &app.files, []string{"/"}, func(dir string) []string {
		return app.children[dir] // nil for a file
	}, func(path string) {
		ui.Text(c, filepath.Base(path))
	}).Grow(1)
	chosen := app.files.Item(app.row)
	// …
}
```

## Outline state

Its state, an `OutlineState`, holds the rows' `ListState`, which chooses
rows as a list's does (`Selected`, `Selection`), and the items `Open`, a
`Selection` of the items showing their children. The rows' keys are the
items, so their choice and state follow them as items open and close above
them. `Rows`, `Item` and `Depth` read the rows shown. `Indent` is the width
each level adds before a row's arrow, the theme's `Space(4)` when zero; it
must not be negative. `Prefix` builds the start of the rows instead
([rows of your own](#rows-of-your-own)).

## Opening and closing

- A click on an item's arrow opens or closes it; Option-click (Alt-click)
  opens or closes all inside it too.
- While the outline has the focus, Right opens the item chosen or goes to
  its first child, and Left closes it or goes to its parent; with Option on
  macOS and Shift elsewhere, as in GTK, they open and close all inside.
- Closing an item whose child was chosen chooses the item.

## Rows of your own

`Prefix`, a function of the `OutlineState`, builds the start of each row in
place of the indentation and the arrow, for rows of a look of your own, as
a JSON viewer's or a file tree's with lines along its levels: its
`OutlineRow` says how deep the row's item is (`Depth`), whether it has
children (`Branch`) and whether they show (`Open`), and `Arrow` makes the
box that opens and closes it, as the outline's arrow does, which you size
and draw. The rows then have neither padding nor a gap, and center what
they hold; the keys, the choice and assistive technology are the
outline's.

```go
app.files.Prefix = func(r ui.OutlineRow) {
	ui.Box(c).Width(16 * float32(r.Depth)).Shrink(0)
	arrow := r.Arrow().Size(20, 28).Center()
	if r.Branch {
		arrow.Children(func() {
			icon := ui.Icon(c, chevron)
			if r.Open {
				icon.Rotate(90)
			}
		})
	}
}
```

## In columns

`ui.OutlineTable` is the same in a [table](table.md)'s columns, the first of
which shows the arrows, indented as the items are deep: the columns are a
table's, which the user sorts, resizes and moves.

```go
ui.OutlineTable(c, &app.files, cols, roots, children, func(path string, col int) {
	switch col {
	case 0:
		ui.Text(c, filepath.Base(path)).SingleLine()
	case 1:
		ui.Text(c, app.size(path))
	}
})
```

## Accessibility

Assistive technology sees a tree, or a table with rows that open and close,
whose rows say their level, whether they open, and whether they are open,
and which of all the rows they are; it opens and closes them as the arrows
do.
