# Rich text editor

<!-- preview rich-text-editor 300 -->

Package `ui/richtext` edits attributed text, as NSTextView does on macOS
and SwiftUI's `TextEditor` bound to an `AttributedString`. It supports:

- **Formats:** bold, italic, underline, strikethrough, inline code, fonts,
  sizes and colors.
- **Links:** opened by a click, and set from the app or with ⌘K.
- **Paragraphs:** headings, bulleted, numbered and task lists,
  quotations, blocks of code and alignment.
- **Attachments:** images inside lines.
- **Editing:** the shortcuts and the Format menu of word processors, and
  undo that brings formats back with the text.
- **Exchange:** copy and paste in rich formats, input methods that compose
  in the typing attributes, Markdown export, and screen readers that
  read the formats.

It lays out and draws only the paragraphs in view, so a long document edits
as a short one.

```go
import "github.com/egoist/mygo/ui/richtext"

// Kept by the app, as its other state.
app.doc = richtext.NewDocument(richtext.NewText("Hello"))

// In the view:
if richtext.Editor(c, app.doc).Fill().Changed() {
	app.save(app.doc.Text())
}
```

The [notes example](../../examples/notes) is a notes app with a toolbar, a
link popover and the Format menu, which saves its notes as JSON.

## Text

A `Text` is attributed text: paragraphs, each with a `ParagraphStyle`,
made of runs, each with its `Attributes`. It is a value, as a string is.
Its methods return a new text and leave the one they are called on as it
was, sharing with it all they did not change. Keeping a version, to undo,
to save later or to compare, costs a pointer, and a version can be read
from any goroutine while others are made.

```go
note := richtext.TextOf(
	richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 1}, Runs: []richtext.Run{{Text: "Groceries"}}},
	richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks}, Runs: []richtext.Run{
		{Text: "Milk, "},
		{Text: "oat", Attributes: richtext.Attributes{Weight: 700}},
	}},
)
note = note.Format(0, 9, func(a *richtext.Attributes) { a.Color = ui.RGB(37, 99, 235) })
note = note.ReplaceString(note.Len(), note.Len(), "\nBread", richtext.Attributes{})
```

Offsets count runes, newlines included, as `Element.TextSelection` does.
To read a text:

- `String` and `PlainText` give its characters;
- `Paragraphs` and `Runs` give its structure;
- `AttributesAt` and `ParagraphStyleAt` describe one rune.

To change one:

- `Replace` and `ReplaceString` change its text;
- `Format` changes the attributes of a range;
- `FormatParagraphs` changes the styles of the paragraphs a range touches.

Attributes left zero take the editor's style: its font, size and color.
`Code`, `Link` and `Tag` (the app's own mark over a run, such as a mention
or a comment) hold only what they hold: text typed at their edges does not
join them.

A paragraph's style and the attributes of its newline belong to the
paragraph, as in word processors: an empty paragraph remembers what typing
gets there. When paragraphs are joined or split, each takes the style of the
paragraph its first rune comes from, as AppKit fixes paragraph styles:

- Deleting the newline after a heading keeps the heading.
- Deleting the heading whole, its newline included, leaves the paragraph
  after it.
- Text pasted at the start of a paragraph brings its own style.

Lists are flat, as in Quill and Automerge: a paragraph's `Indent` is its
level, and items next to each other make a list, numbered at each level from
1 (in letters and Roman numerals below the first).

A paragraph's `ID` is the app's name for it, as the key of what the app keeps
about it, such as notes or a timeline. The editor does not show it, and it
follows the paragraph's text through edits:

- Splitting a paragraph keeps the ID in the first half, or in the second when
  Enter at its start moves the paragraph down. The other half has none.
- Joining two paragraphs keeps the first's, unless the first was empty.
- Paragraphs pasted or inserted have none: `Replace` never takes the IDs of
  the text it inserts, so no two paragraphs get one ID.

Undo and redo bring IDs back with the text, and JSON saves them. The
clipboard, HTML, Markdown and RTF leave them out. Give IDs as the text is
made (`TextOf`, `UnmarshalJSON`), or have the document name the paragraphs
that edits make, in the step of undo that makes them:

```go
app.doc = richtext.NewDocument(chapter)
app.doc.IdentifyParagraphs(func() string { return rand.Text() })

// On save, every paragraph has an ID:
for start, p := range app.doc.Text().Paragraphs() {
	app.store.Save(p.Style.ID, start, p)
}
```

`MarshalJSON` and `UnmarshalJSON` save a text. `Markdown` and `HTML` export
it, and `ParseMarkdown`, `ParseHTML` and `ParseRTF` read Markdown, HTML and
RTF, as a paste does.

## Documents

A `Document` is what an editor edits: a text, its selection with the
attributes typing gets there, and the history of undo. The app keeps it from
frame to frame. Its methods are safe from any goroutine, so a toolbar, a menu
or a timer formats and edits it while the editor shows it. The editor
follows at its next frame. Schedule changes from other goroutines with
`Window.Update` so that a frame comes.

Commands act on the selection as the editor's shortcuts do. On a range,
they change its runs or paragraphs as one step of undo. On a caret,
character formats change only the typing attributes, which the next typed
text gets and the next move of the caret forgets, as ⌘B does in NSTextView.

```go
f := app.doc.Formatting() // the selection's formats, for a toolbar
bold := ui.Button(c, "B")
if f.Bold == richtext.FormatOn {
	bold.Background(t.Accent.Alpha(0.18))
}
if bold.KeepFocus().Clicked() {
	app.doc.ToggleBold()
}
```

`Formatting` gives each character format as `FormatOn`, `FormatOff` or
`FormatMixed`, along with the link the selection is in and the style of its
first paragraph. These methods change the document:

| Method | What it does |
|---|---|
| `ToggleBold`, `ToggleItalic`, `ToggleUnderline`, `ToggleStrikethrough`, `ToggleCode` | turns a format on, or off when all of the selection has it |
| `Format(fn)` | changes any attribute, as a font, a size or a color |
| `SetLink(url)` | links the selection, the link at the caret, or inserts the URL; `""` takes the link away |
| `SetHeading(level)`, `ToggleList(kind)`, `ToggleQuote`, `ToggleCodeBlock`, `Indent(levels)`, `SetAlign(a)` | change the paragraphs selected |
| `FormatParagraphs(fn)` | changes any paragraph style |
| `ToggleChecked(i)` | checks the task holding rune `i` |
| `InsertText`, `Insert`, `InsertAttachment` | replace the selection |
| `Edit(fn)` | changes it any way, as one step of undo |
| `Undo`, `Redo`, `BreakUndoCoalescing` | take steps back and forth |
| `SetText` | replaces the text, as opening another note, and clears the history |
| `IdentifyParagraphs(newID)` | names the paragraphs without an `ID`, and those edits make |

`Edit` runs a function on an `Edit`, whose `Replace`, `ReplaceString`,
`Format`, `FormatParagraphs` and `Select` change the document as one step of
undo. The selection follows what changes, as AttributedString tracks
indices: a caret where text is inserted goes after it, and a range does not
grow at its end.

```go
app.doc.Edit(func(e *richtext.Edit) {
	e.ReplaceString(0, 0, "Dear Ann,\n", richtext.Attributes{})
	e.Format(0, 8, func(a *richtext.Attributes) { a.Italic = true })
})
```

## The editor

`Editor` is the editor with the look of a text area: padding, a background,
a border and corners. `EditorBase` has none of these, for an editor of the
app's own look.

`Changed` reports the edits the user made in the editor, not those the app
made through the document. `EditorOptions` set:

- `Label` and `Placeholder`;
- `Font`, the editor's font and size;
- `ReadOnly`, a text the user selects, copies, and clicks the links of;
- `NoMarkdown`, which turns the Markdown shortcuts off;
- `OnLink`, to open links itself instead of the system;
- `OnLinkCommand`, to ask for a link's URL on ⌘K, as in a popover;
- `TagStyle`, to style the runs that carry a `Tag`.
- `Theme`, how the text looks (see [Theme](#theme)).

The keys are those of word processors, with Control in place of Command off
macOS:

| Key | macOS | Windows and Linux |
|---|---|---|
| Bold, italic, underline | ⌘B, ⌘I, ⌘U | Ctrl+B, Ctrl+I, Ctrl+U |
| Strikethrough | ⇧⌘X | Ctrl+Shift+X |
| Code | ⌘E | Ctrl+Shift+E |
| Link | ⌘K | Ctrl+K |
| Numbered, bulleted, task list | ⇧⌘7, ⇧⌘8, ⇧⌘L | Ctrl+Shift+7, 8, L |
| Indent, outdent | ⌘], ⌘[ (Tab and ⇧Tab in a list) | Ctrl+], Ctrl+[ |
| Align left, center, right | ⌘{, ⌘\|, ⌘} | Ctrl+L, Ctrl+E, Ctrl+R |
| Heading 1 to 6, body | ⌥⌘1 to ⌥⌘6, ⌥⌘0 | Ctrl+Alt+1 to 6, Ctrl+Alt+0 |
| Previous, next paragraph | ⌥↑, ⌥↓ | Ctrl+↑, Ctrl+↓ |
| Start, end of the text | ⌘↑, ⌘↓ | Ctrl+Home, Ctrl+End |

Paragraphs behave as in Notes and Google Docs:

- Enter continues a list, and on an empty item leaves it.
- Enter at the end of a heading starts a body paragraph.
- Backspace at the start of a list's item, a heading, a quotation or a
  block of code takes the style away before joining the paragraph to the
  one above.
- A click on a task's check box checks it, and a click on a link opens it:
  an `http`, `https` or `mailto` address (`ui.SafeToOpen`), as Markdown
  views open them, since other schemes and paths run programs on Windows.
  `OnLink` opens links itself instead.

The Markdown shortcuts format what is typed: at the start of a paragraph,
`# ` to `###### ` make headings, `- `, `* ` and `+ ` bulleted lists, `1. `
numbered ones, `[ ] ` and `[x] ` tasks, `> ` a quotation, and ```` ``` ````
then Enter a block of code. Around words, `**bold**`, `*italic*`,
`_italic_`, `` `code` `` and `~~strikethrough~~` format what they hold as the
closing mark is typed; an underscore inside a word, as in `snake_case`,
stays one. Undo takes a shortcut back to what was typed.

The [Format menu's roles](../menus.md#roles) (`RoleFormatMenu`, or
`RoleBold`, `RoleAlignCenter`, …) format the editor with the keyboard
focus. Paste and Match Style pastes plain text in the typing attributes.

## Theme

`EditorOptions.Theme` sets how the text looks: the colors of text, muted
text, links, the caret and the selection; the font, size and colors of
code, within text and in blocks, and the corners and room of blocks of
code; each level of heading's size, weight, color and room above and
below; the first-line indent of body paragraphs and the room between
them; the indentation of lists, the colors of their markers and check
boxes; the bar and text of quotations; and the padding of `Editor`,
which replaces the window theme's. What it leaves zero takes the
value of `DefaultTheme` for the window's theme, so that a theme sets only
what it changes, and follows the window into dark mode:

```go
violet := ui.Hex("#7c3aed")
richtext.Editor(c, doc, richtext.EditorOptions{Theme: &richtext.Theme{
	Link:     violet,
	Checkbox: violet,
	Headings: [6]richtext.HeadingStyle{{Size: 2.2, Weight: 800}},
}})
```

Lengths are in ems, times the editor's font size, so that a theme scales
with `Font`, but for the padding and the quotation bar's width, in DIPs. Changing the theme lays the text out again. The colors the
editor shows go to screen readers with its runs.

Body paragraphs (not headings, items of lists or code) run together by
default, as in notes. For prose, set them apart in either of two ways:

- `FirstLineIndent` indents the first line of a body paragraph, as printed
  fiction does (about 1.2 ems). The first paragraph after a heading, a list,
  a block of code, an empty paragraph or one aligned otherwise (a centered
  scene break, "* * *") isn't indented, nor is the first of a quotation, as
  nothing needs to set it apart. Only paragraphs aligned at the start are
  indented.
- `ParagraphSpacing` puts room between two body paragraphs, as prose on
  screens does. Headings keep their own room above and below.

```go
richtext.EditorBase(c, app.doc, richtext.EditorOptions{Theme: &richtext.Theme{
	FirstLineIndent: 1.2,
}})
```

The indent is no character: the caret, selection, clicks and arrow keys
skip over it, as in a word processor.

## Paragraphs on screen

`Document.VisibleParagraphs` returns the paragraphs in view in the editor
showing the document: each one's index, the rune it starts at, and the top
and height of its lines in DIPs from the top of the editor's element,
scrolled. Read it as the view builds to line a gutter up with the
paragraphs, as a timeline, notes in the margin or paragraph numbers. Put the
gutter beside the editor with the same top, and draw at its top plus each
paragraph's:

```go
boxes := app.doc.VisibleParagraphs()
ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
	ui.Box(c).Width(40).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
		for _, b := range boxes {
			p.Fill(ui.Rect{X: r.X + 16, Y: r.Y + b.Top, W: 4, H: b.Height}, t.Accent, 2)
		}
	})
	richtext.EditorBase(c, app.doc).Grow(1)
})
```

The boxes are where the editor paints the paragraphs in that frame, since
it lays them out as they are read. When a frame moves them from where they
were read, as the editor scrolls, its width changes or it reveals the caret,
it builds another, so the gutter follows without the app watching the
scrolling. The room above and below a paragraph (a heading's margins, the
spacing between paragraphs) is left out of its box, so a mark at its top
lines up with its first line. `VisibleParagraphs` lays out only what is in
view, as painting does, so it costs as little in a long document as in a
short one. It returns nil until an editor showed the document.

## Typing at the edges of formats

Text typed takes the attributes of the character before the caret. At the
start of a paragraph it takes those of the character after it, and in place
of a selection those of the first character selected. These are NSTextView's
rules, measured on macOS 27, and those of ProseMirror, Lexical and Quill:

- Typing at the end of bold continues bold; at its start, it does not.
- Typing at the end of a link, a code span or a tag does not continue it.
  Inside one, it does.
- After the text before the caret is deleted, typing gets its attributes
  until the caret moves: delete a bold word and type, and the text is bold.
- After Enter, typing keeps the attributes typed before it.

## Attachments

An attachment is an object replacement character (U+FFFC) whose
`Attributes.Attachment` holds an `Attachment`: an image, or a drawing of the
app's, laid out as a box inside the line, standing on the baseline. It is
selected, copied and deleted as one character, and lines break around it.
A box wider than the editor is scaled down to fit.

```go
photo, _ := ui.DecodeBitmap(data)
app.doc.InsertAttachment(&richtext.Attachment{Image: photo, Name: "Beach", Width: 240})
```

Without an image or a `Draw` function, an attachment shows its `Name` in a
box, as an image not loaded yet. HTML and Markdown write an attachment's
`URL`.

## Copy and paste

Copying puts the selection on the clipboard in three formats:

- the editor's own (`richtext.Format`, JSON), which MyGo apps paste whole,
  attachments included within the app;
- HTML;
- plain text.

Pasting reads the richest format the clipboard holds: the editor's own,
then HTML, then RTF, then Markdown (`transfer.Markdown`, which Markdown
views and apps copy), then plain text. A [Markdown view](markdown.md)'s
selection pastes with its formatting.

- HTML, as browsers, mail, Word and Google Docs copy it, keeps its
  headings, lists, task lists, quotations, blocks of code, alignment,
  bold, italic, underline, strikethrough, code and links.
- RTF, as TextEdit, Pages and WordPad copy it without HTML, keeps its
  paragraphs, alignment, lists, bold, italic, underline, strikethrough and
  links. Fonts, sizes and colors are left out, as
Notes leaves them, so that what is pasted takes the editor's. Paste and
Match Style (⇧⌥⌘V) pastes plain text in the typing attributes.

## Input methods

An input method composes at the caret in the typing attributes, as AppKit
and Qt show it, underlined while it composes. The whole composition is one
step of undo with what it replaced.

## Accessibility

Screen readers read the editor as a text field, with the formats of its
runs:

- **VoiceOver** reads `AXAttributedStringForRange`: fonts, colors,
  underlines, strikethroughs and links.
- **Orca** reads ATK's run attributes.
- **Narrator** reads UI Automation's Text pattern, by characters, formats,
  words and lines, with the font, weight, italics, lines, colors and
  heading level of each run, and the mixed value where they vary.

Screen readers move the selection too.

## Large documents

The text is a rope of chunks with a persistent sequence of paragraphs beside
it, so an edit copies what it touches and the path to it, and costs
O(log n). The editor lays out only the paragraphs in view, keeps the layouts
of those near it, and estimates the heights of the others. Typing a
character in a document of 100,000 formatted paragraphs, and drawing the
frame on the CPU, takes about as long as in one of 1,000 (`BenchmarkTypeFrame`
in `ui/richtext`).

## Markdown

`Text.Markdown` writes CommonMark with GitHub's extensions:

- headings, lists, task lists, quotations and fenced blocks of code;
- `**bold**`, `*italic*`, `~~strikethrough~~`, `` `code` ``, links and
  images;
- underlines as `<u>`, and as `<strong>`, `<em>` and `<del>` the marks
  that Markdown's delimiters cannot open or close where they are, as bold
  "foo." right before "bar".

Alignment, fonts and colors have no Markdown and are left out.

`ParseMarkdown` reads it back, and what a model or another app wrote:
CommonMark with GitHub's extensions and footnotes, with the parser of
[Markdown views](markdown.md) (`ui/markdown/parser`). A list item's
paragraphs after its first are indented under it, and a table's rows are
paragraphs of their cells apart by tabs. Math stays as written, as the
editor does not show it. What `Markdown` writes reads back the same.
