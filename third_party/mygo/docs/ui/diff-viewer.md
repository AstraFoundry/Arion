# Diff viewer

<!-- preview diff-viewer 340 -->

`editor.DiffView` shows the changes between two texts, as Pierre's diffs
and code review tools do: side by side or one over the other, with the
words changed within lines highlighted, the unchanged lines between
changes collapsed, and the new text editable where it shows. It is a
[code editor](code-editor.md) on each side, with the editor's highlighting,
selections and commands, laid out and painted only where in view.

```go
import "github.com/egoist/mygo/ui/editor"

lang := editor.LanguageFor("main.go")
before, after := editor.NewBuffer(original), editor.NewBuffer(edited)
before.SetLanguage(lang)
after.SetLanguage(lang)
app.diff = &editor.DiffState{Old: before, New: after, Editable: true}

// In the view:
editor.DiffView(c, app.diff).Grow(1).FillWidth()
```

A `DiffState` is the viewer's state, kept across frames: the buffers it
compares, its options, and what it keeps of its own, as the lines expanded
and its scroll position. Zero options are the defaults, those of Pierre's
diffs.

The diff follows the buffers: an edit in either shows at once, the viewer
diffing again only the lines around it. `New` can be the buffer of an
editor too, as a file's, so that the diff shows its changes as they are
typed.

## Layout

- `Layout` shows the old text on the left and the new on the right, their
  lines aligned (`DiffSplit`), the two scrolling together; or the new
  text, with the lines deleted above where they were (`DiffUnified`).
- `SplitPercent` is the percentage of a split view's width the old side
  takes, half when 0. The user drags the line between the sides to change
  it, or moves it with the arrows once Tab gave it the focus, as a
  [split view](split.md)'s divider; each side keeps its gutter and 12
  columns of text, and a view too narrow for both shows halves, keeping
  the share for when it is wide again. Keep it in the app's settings to
  show the sides as the user left them:

  ```go
  app.diff.SplitPercent = app.settings.DiffSplit // 0 the first time: halves
  editor.DiffView(c, app.diff).Grow(1).FillWidth()
  app.settings.DiffSplit = app.diff.SplitPercent
  ```

  The sides keep scrolling together and their lines aligned: sideways
  each scrolls as far as its own lines run past it, and with `Wrap` the
  side wrapping a line in fewer rows leaves the room of the others under
  it.
- `Inline` highlights what changed within changed lines: words, joining
  those one character apart so that a phrase reads as one change
  (`DiffInlineWordAlt`, the default), each word, from the first word to
  the last, characters, or nothing.
- `Indicators` marks changed lines with bars at the gutter's edge, striped
  for deletions (`DiffBars`), with + and − before them (`DiffClassic`), or
  by their background alone. `NoBackground` leaves out the tints of
  changed lines, and `NoLineNumbers` the numbers.
- `ContextLines` unchanged lines show around each change, 3 by default.
  The others collapse into separators of "N unmodified lines", whose
  buttons show `ExpansionLines` more at a click (100 by default), and all
  of them with Shift; `ExpandUnchanged`, `ExpandAll` and `CollapseAll`
  show or collapse them all.
- `Wrap` wraps long lines, rows of each side keeping their partners
  aligned; `FitContent` makes the viewer as high as its lines, as the
  files of a review in a scroll view.
- `KeepScrollbars` and `NoScrollbars` are those of the code editor: the
  scroll bar stays with the pointer away, so that a diff filling the
  window shows how far down it is, or goes, with its room.
- `Font`, `LineHeight`, `TabSize` and `Theme` are those of the code
  editor. Changes are tinted with the theme's `Added`, `Deleted` and
  `Modified` colors, mixed with the background in a perceptual color
  space so that they read alike in light and dark.

Lines are compared as git's histogram diff compares them, with Myers' for
what it cannot split, and changes slide to where git's indent heuristic
puts them, so that a function added shows as that function rather than
its last lines and the next one's first. Changed lines are paired with
those most like them for highlighting words, rather than in order.

## Editing and review

With `Editable`, the new side edits as the code editor does: typing,
several carets, undo, finding and every command, the diff following each
keystroke. Cmd+F8 and Shift+Cmd+F8 go to the next and previous changes
(Alt+F5 and Shift+Alt+F5 elsewhere), and `GoToHunk` goes to one.

- `HunkActions` shows Revert and Keep on the change under the pointer, or
  the caret's: Revert puts the old lines back in `New`, Keep makes the new
  lines `Old`'s, as an agent's changes are reviewed one at a time.
  `Revert` and `Accept` do the same from the app; `Hunks` lists the
  changes, and `Stats` counts the lines added and deleted.
- `LineSelection` lets the numbers of lines be clicked, and dragged or
  Shift-clicked, to select lines, which `OnLineSelection` reports and
  `SelectedLines` and `SetSelectedLines` read and set, as a link to lines
  of a review does.
- `OnGutterButton` shows a + on the gutter of the line under the pointer,
  as to comment on it, and is called with the line when it is clicked.
- `Annotations` leave room under lines for what `RenderAnnotation` builds
  there, as the comments of a review, each as high as it asks; the other
  side of a split view leaves the room empty, so that the lines stay
  aligned.

```go
app.diff.HunkActions = true
app.diff.OnGutterButton = func(r editor.DiffLineRange) {
	app.comments = append(app.comments, comment{side: r.Side, line: r.End})
}
app.diff.Annotations = app.diff.Annotations[:0]
for i, cm := range app.comments {
	app.diff.Annotations = append(app.diff.Annotations, editor.DiffAnnotation{Side: cm.side, Line: cm.line, Height: 72, Key: i})
}
app.diff.RenderAnnotation = func(c *ui.Context, a editor.DiffAnnotation) {
	app.commentView(c, a.Key.(int))
}
```

`ActiveSide` is the side that has the keyboard, or had it last: the new
side until the old one of a split view has it. `Cursor` and `CursorPos`
return the offset and the position of its newest caret, as the code
editor's do, and `CurrentHunk` the change the caret is in, with `inside`
true, or else the last before it, -1 before the first, as "2 of 5"
counts them; `GoToHunk` moves that caret, so Next Change can step from
it. `Run` runs a command in its editor as its key would, and `Find` opens
its find bar on a query, as an app's menu items do where the menu takes
the keys first:

```go
if ui.Button(c, "Find").Clicked() {
	app.diff.Run(editor.Find)
}
i, inside := app.diff.CurrentHunk()
n := len(app.diff.Hunks())
label := fmt.Sprintf("%d changes", n) // before the first
if i >= 0 {
	label = fmt.Sprintf("%d of %d", i+1, n)
}
ui.Text(c, label)
if ui.Button(c, "Next change").Clicked() {
	app.diff.GoToHunk(i + 1)
}
if inside && ui.Button(c, "Revert this change").Clicked() {
	app.diff.Revert(i)
}
```

`ScrollY` is how far the diff is scrolled, in DIPs, and `ViewportHeight`
how high the part of it in view is, as the last frame laid them out,
whether the wheel, the keys, the scroll bar or `GoToHunk` scrolled it;
`SetScrollY` scrolls it. `LineTop(side, line)` is where a line of either
side shows, below the diff's top, collapsed lines taking no room and, in
a unified view, deleted lines above their change; `LineTop` of a side's
count of lines is the height of the whole diff. With them an app draws a
minimap beside the diff: its changes, and a box where the view is, which
a click or a drag scrolls to:

```go
d := app.diff
var scale float64 // DIPs of the minimap per DIP of the diff, as last drawn
ui.Row(c).Fill().Children(func() {
	editor.DiffView(c, d).Grow(1).FillHeight()
	mini := ui.Box(c).Width(18).FillHeight()
	mini.Draw(func(p *ui.Painter, r ui.Rect) {
		scale = float64(r.H) / max(d.LineTop(editor.DiffNew, d.New.LineCount()), 1)
		at := func(y float64) float32 { return r.Y + float32(y*scale) }
		for _, h := range d.Hunks() {
			top := at(d.LineTop(editor.DiffNew, h.NewStart))
			bottom := at(d.LineTop(editor.DiffNew, max(h.NewEnd, h.NewStart+1)))
			p.Fill(ui.Rect{X: r.X, Y: top, W: r.W, H: max(bottom-top, 2)}, app.changeColor, 0)
		}
		view := ui.Rect{X: r.X, Y: at(d.ScrollY()), W: r.W, H: float32(d.ViewportHeight() * scale)}
		p.Stroke(view, c.Theme().Border, 0, 1)
	})
	mini.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerMove && ev.Button == 0 {
			// The view centered where the minimap is pressed, or dragged.
			d.SetScrollY(float64(ev.Y)/max(scale, 1e-9) - d.ViewportHeight()/2)
			return true
		}
		return ev.Kind == ui.InputPointerUp
	})
})
```

## Styling

`Theme` colors the diff as it does the [code editor](code-editor.md#themes),
changed lines included. The parts around the lines are the app's to
build, as Pierre's diffs render them: `RenderSeparator`,
`RenderHunkActions` and `RenderGutterButton` are given the element the
viewer places and what the part shows and does, and the viewer's own
looks are `DefaultSeparator`, `DefaultHunkActions` and
`DefaultGutterButton`, which they can call. `RenderFindBar` and
`RenderGoToLine` build the bars of the sides' editors, as the code
editor's do.

```go
app.diff.RenderSeparator = func(c *ui.Context, sep ui.Element, s *editor.DiffSeparator) {
	sep.Padding(4).Children(func() {
		if s.Lines == 0 { // a patch's hunk
			ui.Text(c, s.Header).Font("monospace")
			return
		}
		if ui.Button(c, fmt.Sprintf("Show %d hidden lines", s.Lines)).Clicked() {
			s.ExpandAll()
		}
	})
}
app.diff.RenderHunkActions = func(c *ui.Context, bar ui.Element, a *editor.DiffHunkActions) {
	bar.Gap(4).Children(func() {
		if ui.Button(c, "Reject").Clicked() {
			a.Revert()
		}
		if ui.PrimaryButton(c, "Accept").Clicked() {
			a.Keep()
		}
	})
}
```

- A `DiffSeparator` spans the viewer, both sides of a split one, less the
  room of the scroll bar's track while the viewer scrolls (none with
  `FitContent`), 32 DIPs high. It collapses `Lines` unchanged lines,
  which `ExpandUp`, `ExpandDown` and `ExpandAll` show, `Step` lines at a
  time; `First` and `Last` mark those before the first change and after
  the last, which expand one way. A patch's hunk has a separator of no
  lines, with its `Header`.
- `DiffHunkActions` sit at the top right of the change under the pointer,
  or the caret's: `Index` and `Hunk` say which, and `Revert` and `Keep`
  act.
- The gutter button is a button on the gutter of the line under the
  pointer, which calls `OnGutterButton` when clicked; its render function
  gives it a look and a label.

## Patches

`editor.ParsePatch` reads the files a unified diff changes, as `git diff`
and `diff -u` print them, with renames, modes and new and deleted files;
`FilePatch.Apply` applies one to the old file's text. `SetPatch` shows a
file's patch without the files, as a pull request's diff arrives: its
hunks' lines, numbered as in the files, each hunk under its header.
Each hunk is highlighted on its own, from the start of its first line,
as `git diff` colors it, so a comment or a string that the hunk before
ends in doesn't go on into it. A language's own `Highlighter`
([Highlighters](code-editor.md#highlighters)) sees the hunks' lines as
one text.

```go
files, err := editor.ParsePatch(output) // of git diff
if err != nil {
	return err
}
for _, f := range files {
	d := &editor.DiffState{}
	d.SetPatch(f)
	app.files = append(app.files, d)
}
```

## Accessibility

Assistive technology sees each side as a text field, "Old text" and "New
text", the old one read only, with its text and selection, and the line
between the sides of a split view as a splitter named "Divider", whose
value is the old side's width, which the arrows move. A separator is
a group named by the lines it collapses, with buttons expanding them up,
down or all; Revert and Keep are buttons, "Revert change" and "Keep
change". These names, and what the viewer writes ("4 unmodified lines"),
are in the window's language
([MyGo's own strings](../i18n.md#mygos-own-strings)).
