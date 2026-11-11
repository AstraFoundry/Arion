# Tree

<!-- preview tree 280 -->

`ui.Tree` creates a tree whose items `ui.TreeItem` builds, each inside the
item it belongs to. An item's children show while its `*bool` is true; a
click on its arrow opens or closes it, and `Clicked` reports a click
elsewhere on it, or Enter, for choosing it, which `Selected` shows.

```go
item := func(path, label string, open *bool, children func()) {
	if ui.TreeItem(c, label, open, children).Selected(app.chosen == path).Clicked() {
		app.chosen = path
	}
}
ui.Tree(c, func() {
	item("src", "src", &app.srcOpen, func() {
		item("src/main.go", "main.go", nil, nil)
	})
	item("go.mod", "go.mod", nil, nil)
})
```

Nil children, or a nil open, makes a leaf.

## Keyboard

The tree is one stop of Tab, which goes to the item that had the focus
last, or the one `Selected`. While an item has the keyboard focus, Up and
Down move it to the items around, Home and End to the first and the last,
Right opens the item or moves to its first child, and Left closes it or
moves to its parent; right to left, Left opens and Right closes. Typing
the first letters of an item moves to it, and the same letter again to the
next item starting with it.

A tree builds every item it shows. For trees of thousands of items, or
with columns, use an [outline](outline.md), which builds only the rows in
view.

## Without a look

`ui.TreeBase` is a tree without a look, for rows of your own, as a JSON
viewer's: a `Root` column holding the items, each made by `Item`, whose
`Row` you style and fill, and whose `Group` builds the items inside it, in
a column below the row, while its `*bool` is true (a nil one makes a leaf).
The tree has the keys and the focus of `Tree`'s, and `Depth` says how deep
an item is, to indent its row by. A key given with `Context.Key` before
`Item` keeps the focus on an item as items above it open and close.

```go
var node func(n *Node)
tree := ui.TreeBase(c)
tree.Root.Children(func() {
	node = func(n *Node) {
		c.Key(n.path)
		item := tree.Item(n.open) // nil for a leaf
		row := item.Row.Gap(4).Padding(3, 8).PaddingStart(8 + 16*float32(item.Depth())).Radius(6)
		if app.chosen == n.path {
			row.Background(t.Surface)
		}
		if row.Clicked() {
			app.chosen = n.path
		}
		row.Children(func() {
			if n.open != nil {
				arrow := ui.Icon(c, chevron).Size(16, 16)
				if *n.open {
					arrow.Rotate(90)
				}
				if arrow.Clicked() {
					*n.open = !*n.open
				}
			}
			ui.Text(c, n.key).FontWeight(500)
			ui.Text(c, n.value).TextColor(t.TextMuted)
		})
		item.Group(func() {
			for _, kid := range n.children {
				node(kid)
			}
		}) // nil while closed; style it, as with a line along it
	}
	for _, n := range app.roots {
		node(n)
	}
})
```

`Tree` is `TreeBase` with the theme's look. `TreeBase` builds every item
that shows; for trees of thousands of items, use an
[outline](outline.md#rows-of-your-own) with a `Prefix` of your own. See
[custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees a tree of items, each with its level, open or
closed when it has children, and which of the items shown it is, as an
outline's rows; it opens and closes them as the arrows do.
