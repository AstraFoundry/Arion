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
  `TextInput`, wrapped in a text area. On a field around an input, as a
  `SearchField`, `TokenField` or `Combobox`, it shows in that input.
  `PlaceholderColor` paints it in another color than the theme's
  `TextMuted`, as a warning hint in a form.
- `CaretWidth` paints the caret wider than one DIP, as a terminal's block
  or a large-print theme's, or hides it with zero; it changes nothing else.
- `Password` hides what it holds, and keeps it from the clipboard and input
  methods, in the frames that call it: an eye button that shows the text
  stops calling it. Only a `TextInput` takes it: as on every platform, a text
  area has no password mode.
- `AutoFocus` gives it the keyboard focus as it appears, as the first field
  of a dialog.
- `ReadOnly(true)` shows the text without letting the user change it: it
  still takes the focus, without a caret, and its text can be selected and
  copied.
- `MaxLength(n)` keeps the user from making the text longer than n, counted
  as HTML's `maxlength` counts it, in UTF-16 code units: what is typed,
  pasted or committed by an input method is cut to what fits. A value the app
  sets is not cut, and n <= 0 means no limit.
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

- `OnTextKey(fn)` sees a key that types no text before the input takes
  it: Enter, Tab, Escape, Backspace and Delete, the arrows and the other
  keys that move the caret, and chords with Cmd or Ctrl, as AppKit's
  `doCommandBySelector` and a web page's `keydown` do. fn runs as the key
  comes, after the keys and text typed before it, and returning true takes
  the key: a text area inserts no newline, a `TextInput` does not submit,
  Tab does not move the focus. A key left goes on as without fn. Keys that
  a `Shortcut` around the focus handles go to it first, as with
  `HandleInput`, unless the input edits with them, as with Enter and the
  arrows. fn may run between frames, so it reads the text and replaces it
  through its `ui.TextKey` rather than an `Element` kept from the view
  (they panic once fn returns), as a chat composer sends with Enter, breaks
  the line with Shift+Enter, and puts the last message back with Up:

  ```go
  ui.TextArea(c, &app.draft).Lines(1, 8).OnTextKey(func(k ui.TextKey) bool {
  	switch {
  	case k.Key == ui.KeyEnter && k.Mods == 0:
  		app.send(k.Text())
  		k.SetText("")
  		return true
  	case k.Key == ui.KeyUp && k.Mods == 0 && k.Text() == "":
  		k.SetText(app.lastSent())
  		return true
  	}
  	return false // Shift+Enter breaks the line
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

The ranges are runes of the text, found in it each frame. They may overlap
and nest, as the attributes of an attributed string (`NSAttributedString`)
do: a range sets its `Color` when its alpha is not zero and its `Weight`
when it is not zero, and where ranges overlap, each of the two is the last
given range's that sets it, across the calls of the frame. A bold range
with a colored mention inside it shows the mention bold and colored:

```go
in.TextRanges(
	ui.TextRange{Start: 0, End: 40, Weight: 700},
	ui.TextRange{Start: 10, End: 16, Color: t.Accent},
)
```

A password shows none. While an input method composes, its text sits
unstyled at the caret and the ranges keep their style around it.

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
It preserves the last undo step, so an app can reformat what the user
typed without losing that step. Setting the string to `""` does not discard
the history: Undo can restore text from before the clear.

To start a new editing session, clear the string and change the input's
[key](views.md#keys). The new identity gets a new editor, without the old
selection, composition or undo history:

```go
// Keep secret and secretSession in app state.
ui.TextInput(c.Key(app.secretSession), &app.secret).Password().Label("Secret")

if ui.Button(c, "Cancel").Clicked() {
	app.secret = ""
	app.secretSession++ // the next build creates a fresh editor
}
```

For indexed storage, `ui.TextInputBuffer` and `ui.TextAreaBuffer` edit a
`*ui.TextBuffer`. Replacing its contents externally with `buffer.Set("")`
discards the control's undo and redo history and composition when it next
synchronizes with the buffer; its selection is clamped to the new text.
Neither changing a key nor replacing a buffer guarantees that old text is
erased from memory; existing `TextSnapshot` values retain their contents.

A single-line input replaces line breaks in pasted text with spaces. Use
`OnPaste` to reject a multiline value before it is changed or inserted:

```go
ui.TextInput(c, &app.secret).Password().OnPaste(func(text string) bool {
	return strings.ContainsAny(text, "\r\n") // import "strings"
})
```

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

`TextSelection` returns the selection in order. `TextAnchorCaret` returns
it from where it was started to its caret, so an app can tell a selection
made backwards from one made forwards: an editor's own keys that extend a
selection, such as Shift-Home moving to the first character of the line,
extend it from its anchor:

```go
anchor, caret := input.TextAnchorCaret()
input.SetTextSelection(anchor, lineStart(app.draft, caret))
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
