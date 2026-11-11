# Right to left

Arabic, Hebrew, Persian and Urdu read from right to left, and so do the
interfaces of apps in those languages: a row starts at the right, a
sidebar is on the right, a back arrow points right. Native UI lays a window
out so when the app's language reads right to left, as Flutter's
`Directionality`, SwiftUI's `layoutDirection`, CSS's `direction` and GTK's
text direction do. The coordinate system is not mirrored, as in Godot:
X still grows to the right in `Rect`, in what `Draw` paints, and in the
pointer's position. Only the layout is.

## The direction

`ui.LeftToRight` and `ui.RightToLeft` are the two directions
(`ui.Direction`). A window takes the direction of the app's language,
which package `mygo` gives it, and every element that of the element it is
built in. `Direction` gives an element and what is built inside it a
direction of their own, as a phone number, a code sample or a media
player's controls that stay left to right in a right-to-left window:

```go
ui.Row(c).Direction(ui.LeftToRight).Gap(8).Children(func() {
	ui.Button(c, "⏮")
	ui.Button(c, "⏯")
	ui.Button(c, "⏭")
})
```

Give it right after creating the element, before its children, as you
would `Key`: widgets read the direction as they are built. The popups
attached to an element (popovers, comboboxes' options, tooltips) and what
an `Overlay` builds take the direction of where they were asked for.

`Context.Direction` returns the direction of what is built now, for
widgets of your own that draw what points along the line:

```go
forward := "M5 2 11 8 5 14"
if c.Direction() == ui.RightToLeft {
	forward = "M11 2 5 8 11 14"
}
```

## What mirrors

Right to left:

- A `Row` lays its children out from the right, and a `Column` aligns them
  across from the right: `Justify`, `AlignItems` and `AlignSelf` take
  `Start` and `End` from the direction. `Reverse` lays a row out from the
  left. A grid places its columns from the right, and `JustifyItems` and
  `JustifySelf` start at the right of their cells.
- `TextAlign`'s `Start` is the right of the element, whatever its text
  reads, as in CSS, AppKit and GTK, and `End` the left. A paragraph reads
  in the direction of its first strongly directional character, as Unicode's
  bidirectional algorithm finds it, and one without any, as a number or an
  empty line, right to left. Left to right, nothing changes: a paragraph
  in Arabic still starts at the right, as Apple's guidelines align a
  paragraph by its language.
- The logical edges are on the right where they start: `PaddingStart`,
  `PaddingEnd`, `MarginStart`, `MarginEnd`, `BorderStartWidth`,
  `BorderEndWidth`, the `Start` and `End` of positioned elements (and
  `StartPercent`, `EndPercent`), and the anchors `AnchorTopStart`,
  `AnchorStart`, `AnchorBottomStart`, `AnchorTopEnd`, `AnchorEnd` and
  `AnchorBottomEnd`, as CSS's logical properties and Flutter's
  `EdgeInsetsDirectional` and `AlignmentDirectional`. An absolute element
  placed from neither side goes to the right, and where both `Left` and
  `Right` are given and its width leaves no room for both, `Right` wins.
- Scrolling sideways starts at the right, and `ScrollState.X` counts from
  there (0 shows the start, `MaxX` the end), as Flutter's offsets do; the
  wheel and the trackpad move the content as they do left to right, and
  Shift with the wheel goes toward the end. The vertical scroll bar is on
  the left.
- The controls mirror what points along the line: a switch turns on to
  the left; sliders, range sliders, progress bars and meters fill from the
  right, and their arrows move the knob the way they point; tabs,
  segmented controls, toolbars, ratings, grid views and calendars go to
  their next item with Left; disclosure arrows, breadcrumbs' chevrons, a
  toolbar's overflow and the back button point left, or right for back;
  trees, outlines and disclosures indent from the right and open with
  Left; a split's first pane is on the right; a table's columns go from
  the right and resize from their left edges; pages pushed on a router
  slide in from the left, and Alt+Right goes back.
- Popovers, a select's and a combobox's options, and the menus of menu
  buttons open at the start of what opens them, and context menus are laid
  out right to left, opening to the left of the pointer.
- In a single-line text input, the text starts at the right, its
  placeholder too, and the caret of an empty one is there. The arrows move
  the caret the way they point, through text of either direction, and
  Option (Control elsewhere) and Command with the arrows go by words and to
  line ends the way they point in a right-to-left paragraph, as AppKit and
  GTK do.

## What does not

- What is physical stays physical, as in CSS: `Left`, `Right`,
  `Padding(top, right, bottom, left)`, `BorderWidth`, `AnchorTopLeft`,
  `Radius`'s corners, gradients' angles, transforms.
- Text, images, icons and what `Draw` paints are not mirrored: a check
  mark, a magnifying glass, a spinner and a chart read as they do left to
  right. Draw what must point the other way after reading
  `Context.Direction`.
- Code reads left to right: the code editor and the diff viewer
  (`ui/editor`), Markdown's code blocks and the terminal keep their own
  direction, as do time inputs, as times are written so.
- The rich text editor (`ui/richtext`) aligns its paragraphs as their text
  reads, as a document does.
- The inspector reads left to right.

## Trying it

`MYGO_DIRECTION=rtl` lays every window of native UI out right to left
(and `ltr` left to right), whatever the app's language, to try an
interface before it is translated, as GTK's `GTK_DEBUG=invert-text-dir`
and Qt's `-reverse` do:

```sh
MYGO_DIRECTION=rtl mygo dev
```

The [inspector](inspector.md)'s RTL toggle does the same for its window
while it runs, and a `Tester` lays its window out right to left after
`SetDirection(ui.RightToLeft)`, as a test of the layout would:

```go
tt := ui.NewTester(app.view, 400, 300)
tt.SetDirection(ui.RightToLeft)
back, _ := tt.Find("Back")
if back.X < 200 {
	t.Error("the back button is not at the right")
}
```

The [gallery](../../examples/gallery) shows every control right to left
with the toggle in its toolbar.
