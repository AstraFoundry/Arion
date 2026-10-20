# Text input

<!-- preview text-input 250 -->

`ui.TextInput` creates a single-line text input editing a `*string`, and
`ui.TextArea` a multi-line one. They change the string as the user types:
read it, or ask `Changed` for work that follows, and `Submitted` for Enter
in a single-line input.

```go
if ui.TextInput(c, &app.query).Placeholder("Search").Label("Search").Changed() {
	app.results = search(app.query)
}

if ui.TextInput(c, &app.name).AutoFocus().Submitted() {
	app.rename()
}

ui.TextArea(c, &app.notes).Height(160)
```

- `Placeholder` shows a text while the input is empty, on one line in a
  `TextInput`, wrapped in a text area.
- `Password` hides what it holds, and keeps it from the clipboard and input
  methods, in the frames that call it: an eye button that shows the text
  stops calling it. Only a `TextInput` takes it: as on every platform, a text
  area has no password mode.
- `AutoFocus` gives it the keyboard focus as it appears, as the first field
  of a dialog.
- `ReadOnly(true)` shows the text without letting the user change it: it
  still takes the focus, without a caret, and its text can be selected and
  copied.
- `Disabled(true)` grays it out.
- `OnPaste(fn)` sees a paste (Cmd+V or Ctrl+V, the Edit menu's or the
  context menu's Paste) before it is inserted, with the clipboard's text,
  empty when it holds none, as with an image. Returning true takes the
  paste, which then changes nothing, as `preventDefault` does on the web:

  ```go
  ui.TextArea(c, &app.draft).Lines(1, 8).OnPaste(func(text string) bool {
  	if len(text) > 10_000 {
  		app.attach(text) // a large block becomes an attachment
  		return true
  	}
  	if text == "" {
  		if png := mygo.Clipboard.ReadImage(); png != nil {
  			app.attachImage(png)
  			return true
  		}
  	}
  	return false
  })
  ```

A `TextInput` too narrow for its text keeps the caret in view while it has
the focus, and shows the start of its text without it.

`TextRanges` colors runs of the text, and sets their weight, in the frames
that call it, as a message field shows the mentions in what is typed:

```go
in := ui.TextAreaBase(c, &app.draft).Lines(1, 8)
for _, m := range mentions(app.draft) {
	in.TextRanges(ui.TextRange{Start: m.start, End: m.end, Color: m.color, Weight: 600})
}
```

The ranges are runes of the text, found in it each frame, and do not
overlap; a password shows none. While an input method composes, its text
sits unstyled at the caret and the ranges keep their style around it.

A text area is at least a few lines high and grows with its text; given a
height, it scrolls within it, with the wheel and a scroll bar, and keeps
the caret in view as it moves. `Lines(min, max)` makes it as high as its
text wraps at its width, from `min` lines up to `max`, past which it
scrolls, as a message field grows with what is typed:

```go
ui.TextArea(c, &app.draft).Lines(1, 8)
```

A text area lays out only the paragraphs in view and keeps their layouts
until they change, so that it holds texts of hundreds of thousands of
lines, as a log or a source file, and stays as quick to type in.

## Editing

Text inputs edit as the platform's text fields do: selection with the
pointer (a double click selects a word, a triple click a line), with Shift
and the arrows, and by words and lines with the platform's keys; undo and
redo; cut, copy and paste, also from the Edit menu's roles; and a context
menu of the editing commands. They take text composed with input methods,
which see the text around the caret, so that press and hold, Japanese
conversion and predictions work as in other apps. The keys typed while an
input method composes are its own, as Enter choosing a candidate or Escape
giving the composition up: they submit nothing, press no shortcut and
close no dialog. `Composing` reports whether it composes, its text not yet
in the string.

The input keeps the text being edited, its selection and its undo history
from frame to frame; setting the string from elsewhere replaces the text.

## The caret

`TextSelection` returns the selection as offsets in runes into the text,
the caret where they are equal, and `SetTextSelection` moves it, as an app
completing the word being typed puts the caret after it:

```go
input := ui.TextArea(c, &app.draft)
if start, _ := input.TextSelection(); app.completed != "" {
	app.draft, start = complete(app.draft, start, app.completed)
	input.SetTextSelection(start, start)
}
```

## Beside the lines

`TrackLines` keeps in a `ui.TextLines` where a text area lays out the lines
of its text, for an app that paints beside them: numbers by the first row
of each line, or marks on runs of a line. A line is the text between
newlines, on several rows where the area wraps it. `Top(i)` is the top of
line `i` in the area's content, below its padding and before it scrolls,
`At(y)` the line at a height, `Rows(i)` the runes where each row of line
`i` starts, and `Height` the height of the text. With the area's
[`ScrollState`](scroll.md#keeping-the-offset), a gutter numbers the lines
in view:

```go
const pad = 8 // the text area's top padding
ui.Row(c).Grow(1).Children(func() {
	ui.Box(c).Width(48).FillHeight().Draw(func(p *ui.Painter, r ui.Rect) {
		top := r.Y + pad - app.scroll.Y // the top of the area's content
		p.Clip(r, 0, func() {
			for i := app.lines.At(app.scroll.Y); i < app.lines.Count() && top+app.lines.Top(i) < r.Y+r.H; i++ {
				p.Text(r.X+8, top+app.lines.Top(i), strconv.Itoa(i+1), 12, p.Theme().TextMuted)
			}
		})
	})
	ui.TextArea(c, &app.source).Padding(pad, 12).Grow(1).FillHeight().
		TrackLines(&app.lines).TrackScroll(&app.scroll)
})
```

The heights of the lines the area has not laid out are estimates, as its
scroll bar's are, until it shows them. In `Draw` the lines are those of the
frame; while the view is built, those of the last one.

## Errors

In a [field](form.md), `Error` marks the value invalid: the input draws its
border in the theme's `Danger` color, and assistive technology reads the
message with it.

```go
ui.Field(c, "Email", func() {
	ui.TextInput(c, &app.email)
}).Error(app.emailError)
```

## Without a look

`ui.TextInputBase` and `ui.TextAreaBase` are text inputs without padding,
background, border or corners, for inputs of your own design: see
[custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees a text field, or a text area, named by its
`Label` or its field, whose value is its text, with the caret and the
selection; it edits it as typing does, unless it is read-only. A password
field hides its value.
