# Layout

Elements lay out their children with flexbox, as in CSS, or in a
[grid](grid.md), in device-independent pixels (DIPs):

```go
ui.Row(c).Gap(8).Padding(12).Children(func() {
	ui.Icon(c, folder)
	ui.Text(c, "Documents").Grow(1)
	ui.Text(c, "12 items").TextColor(c.Theme().TextMuted)
})
```

## Containers

- `ui.Column` stacks its children from top to bottom and stretches them to
  its width; `ui.Box` is a column too. `ui.Row` places them from left to
  right and centers them vertically. `Reverse` lays them out the other
  way, from the right or the bottom, as CSS's `row-reverse` and
  `column-reverse`.
- `ui.Spacer` takes the free space of its row or column, pushing its
  siblings apart, and `ui.Divider` draws a thin line across its row or
  column. `Dividers(width, color)` draws a line between each two children
  of a row, a column or a [list](list.md), in the middle of the gap
  between them, which it does not widen: give the element a `Gap` at
  least as wide.
- `ui.Grid` lays its children out in columns and rows: see
  [Grid](grid.md). Containers that scroll are [scroll views](scroll.md),
  and [lists](list.md) build only the rows in view.

## Sizes

- `Width`, `Height` and `Size` set sizes; `WidthPercent` and `HeightPercent`
  take a share of the parent; `MinWidth`, `MaxWidth`, `MinHeight` and
  `MaxHeight` bound them, and their `Percent` forms by a share of the
  parent. An element with both takes the larger minimum and the smaller
  maximum, as CSS's `max-width: min(78%, 34rem)` does a chat bubble's:
  `MaxWidthPercent(78).MaxWidth(544)`. `Fill`, `FillWidth` and
  `FillHeight` take the parent's whole content box. In a [grid](grid.md#sizing-children), a child's percentages
  are of its cell.
- `Grow(1)` gives an element the free space along its parent's direction,
  like `flex: 1`: a list that fills the rest of a window, or a `Spacer` that
  pushes the elements after it to the end. `Shrink`, `Basis` and
  `BasisPercent` work as in CSS.
- `AspectRatio` keeps an element's proportions.

## Spacing

`Gap` spaces the children, `GapX` and `GapY` horizontally and vertically
apart; `Padding` and `Margin` take one, two or four values, as in CSS, and
`PaddingX`, `PaddingY`, `MarginX` and `MarginY` two sides. A margin of
`ui.Auto` takes the free space on its side, as in CSS: `Margin(0, ui.Auto)`
centers an element in a column, and `Margin(0, 0, 0, ui.Auto)` sends a
row's child, and those after it, to the right.

`PaddingStart` and `PaddingEnd`, `MarginStart` and `MarginEnd`, and
`BorderStartWidth` and `BorderEndWidth` set the side lines start on and
the other one: the left and the right, or the right and the left of an
element laid out [right to left](right-to-left.md), as CSS's logical
properties do. `MarginStart(ui.Auto)` sends a row's child to its end in
either direction.

## Alignment

`Justify` places the children along the direction (`Start`, `Center`,
`End`, `SpaceBetween`, `SpaceAround`, `SpaceEvenly`), `AlignItems` across
it (`Start`, `Center`, `End`, `Stretch`, `Baseline`), `AlignSelf` one child; `Center`
centers them both ways. `Start` is the left of a row and the top of a
column, and the right of an element laid out
[right to left](right-to-left.md), which lays a row out from there. `Wrap` wraps a row onto more lines, which
`WrapReverse` stacks upward, and `AlignContent` places those lines (as
`Justify` places children, or `Stretch`).

`Baseline` lines up the first lines of text of a row's children, as a
large price beside its small unit, and the row is as high as that takes.
The baseline of a box is that of its first child (in a row, of its first
child aligned by its baseline), and a child without text lines up its
bottom, as in CSS:

```go
ui.Row(c).AlignItems(ui.Baseline).Gap(4).Children(func() {
	ui.Text(c, "$42").FontSize(40)
	ui.Text(c, "/month").TextColor(t.TextMuted)
})
```

## Positioning

`Absolute` takes an element out of the flow, placed with `Top`, `Right`,
`Bottom` and `Left` in its parent, in DIPs or with `TopPercent` and the
like; automatic margins center it between the sides it is placed from. As
in CSS, they are measured from just inside the parent's border, whatever
its padding: in a box with a border 1 DIP wide, `Top(0)` is 1 DIP below
the box's top edge. `Start` and `End` (and `StartPercent`, `EndPercent`)
place it from the side lines start on and from the other, which are the
right and the left [right to left](right-to-left.md). On an element in the
flow, `Top` and the others move
it from where the layout put it, without moving its siblings, as CSS's
relative positioning does: a badge raised a little above its text.
`Sticky` makes them hold an element in the flow inside the scroll
container around it instead, as CSS's `position: sticky` (see [Scroll
view](scroll.md#sticky-elements)).

```go
ui.Box(c).Size(40, 40).Children(func() {
	ui.Icon(c, bell).FontSize(24)
	ui.Badge(c, "3").Absolute().Top(-4).Right(-6)
})
```

`Attach(at, self)` takes an element out of the flow too, and puts a point
of it (`self`) on a point of its parent's box (`at`): a corner, the middle
of a side or the center, as `ui.AnchorTopRight` or `ui.AnchorCenter`, or
one on the side lines start or end on, as `ui.AnchorTopEnd`, which is on
the left right to left. The
element keeps its own size, and `Top`, `Right`, `Bottom` and `Left` move
it from there:

```go
ui.Box(c).Children(func() {
	ui.Avatar(c, "Ada Lovelace", nil)
	// Centered on the avatar's top right corner, whatever its size.
	ui.Badge(c, "3").Attach(ui.AnchorTopRight, ui.AnchorCenter)
})
ui.Box(c).Fill().Children(func() {
	ui.PrimaryButton(c, "New").Attach(ui.AnchorBottomRight, ui.AnchorBottomRight).Right(16).Bottom(16)
})
```

## Paint order

Siblings paint in the order they are built, those in the flow first and
the absolute and sticky ones over them, and a later one takes the pointer
where they overlap. `ZIndex` changes that order without moving the element
in the tree, as CSS's `z-index`: siblings with a higher value paint over
those with a lower one and take the pointer first, those with the same
value keep the order above, and the default is 0. A negative value puts an
element under its siblings, over its parent's background. The layout, the
order `Tab` moves the focus in and what assistive technology reads still
follow the tree, which is why reordering the children is no substitute.

```go
// Overlapping avatars, the first on top.
ui.Row(c).Children(func() {
	for i, u := range users {
		a := ui.Avatar(c, u.Name, u.Photo).ZIndex(len(users) - i)
		if i > 0 {
			a.Margin(0, 0, 0, -8)
		}
	}
})

// A cell growing over its neighbours while the pointer is over it.
cell := ui.Box(c).Size(80, 80)
if cell.Hovered() {
	cell.Scale(1.2, 1.2).ZIndex(1)
}
```

The order is among siblings alone: the element moves with all that is
inside it, still clipped by its parent, and is never mixed with the
elements of other parents, as if each z-index other than 0 made a CSS
stacking context. Raising a card inside a column does not lift it over the
next column; raise the column. The window's content stays under its
[overlays](overlays.md), and overlays keep the order they open in. The copy
of an element leaving with its [`Exit`](transitions.md) keeps the z-index
the element had, under its siblings of the same one: a toast raised over the
page fades out over it.

## Clipping and visibility

- `Clip` cuts its children to its rounded box, inside its border, which it
  draws over them; `ClipX` and `ClipY` clip only on the sides or above and
  below.
- `Invisible` hides an element and its children, which keep their room
  but draw nothing and take neither the pointer nor the focus, as CSS's
  `visibility: hidden`.
- `Hidden(true)` hides them and takes their room away too, as CSS's
  `display: none`: the siblings close up, with no gap left for it, and
  assistive technology does not read it. Unlike not building the element,
  it keeps the state of what is inside it, a scroll offset, text being
  edited, an animation, for when `Hidden(false)` shows it again. The focus
  leaves it as it hides, and what is attached to it (`AttachTo`) hides with
  it.

  ```go
  ui.Column(c).Hidden(!app.showDetails).Children(func() {
  	ui.TextArea(c, &app.notes)
  })
  ```
- `Debug` outlines an element and everything inside it, with their padding
  and margins, to see why the layout is what it is. The
  [inspector](inspector.md) shows the boxes of every element, and why.
- Elements move, resize and recolor smoothly with a
  [transition](transitions.md).
