# Split view

<!-- preview split 270 -->

`ui.Split` lays out two panes side by side, with a divider between them that
the user drags to resize them, or moves with the arrows once it has the
keyboard focus: a `*float32` is the width of the first, which the divider
keeps 40 DIPs from either edge. When the window leaves less room than
that, the first pane gives way so the second keeps its 40 DIPs, and the
first takes its size back when there is room again; the `*float32` keeps
the size the user chose. `Changed` reports a move.

```go
ui.Split(c, &app.sidebarWidth, func() { app.files(c) }, func() { app.editor(c) }).Fill()
```

`ui.SplitVertical` puts the first above the second, the `*float32` its
height:

```go
ui.SplitVertical(c, &app.consoleHeight, func() { app.editor(c) }, func() { app.console(c) }).Grow(1)
```

`ui.SplitPercent` and `ui.SplitVerticalPercent` take the percentage of
the room the first pane takes instead, from 0 to 100: both panes then grow
and shrink with the window, in proportion, as a results panel under an
editor does.

```go
ui.SplitVerticalPercent(c, &app.editorPercent, func() { app.editor(c) }, func() { app.results(c) }).Grow(1)
```

Keep the size in the app's state, and save it to restore the layout when
the app starts again. The pointer shows a resize cursor over the divider.

`SplitReset` makes a double click on the divider put the first pane back at
the size the app chooses, in the split's own unit: DIPs for `Split` and
`SplitVertical`, a percentage for the percent variants. A split without it
keeps its size on a double click. The size is kept even when the window is
too small to show it, so the pane takes it back when there is room again,
as a press on the divider does.

```go
ui.Split(c, &app.sidebarWidth, func() { app.files(c) }, func() { app.editor(c) }).SplitReset(220).Fill()
```

## Accessibility

Assistive technology sees the divider as a splitter named "Divider",
which takes the focus and moves with the arrows.
