# Markdown

<!-- preview markdown 440 -->

Package `ui/markdown` shows Markdown in native UI: CommonMark, GitHub's
tables, task lists, strikethrough and autolinks, footnotes and math. It
is built for the replies of language models, which stream in token by
token: text appended is parsed alone, half-written syntax shows as what
it will become, and text comes in smoothly, fading in.

```go
import "github.com/egoist/mygo/ui/markdown"

// Kept in the app's state, shared by every view.
app.md = markdown.Options{}

// In the view:
markdown.View(c, app.readme, &app.md)
```

`View` shows a finished document. Give it the source each frame, as a
`Text` its string: it parses again only what changed since the last
frame, and keeps what it made of the rest, the text layouts of its
paragraphs included.

## Streaming replies

`Stream` shows Markdown still being written. Append the tokens of a
reply to its string as they come, and show it with `View` once it is
complete, in the same place, so that the view carries on:

```go
func (app *chat) message(c *ui.Context, m *Message) {
	if m.Done {
		markdown.View(c, m.Text, &app.md)
	} else {
		markdown.Stream(c, m.Text, &app.md)
	}
}
```

Appending text parses only the outermost block still open before it,
or nothing but the new lines when no block is: a token costs the same at
the end of a long reply as at its start, and blocks already shown are
neither parsed nor laid out again.

The end of a reply shows what its syntax will most likely become, as
the chat apps show it, without changing its text: `**bol` is bold, an
unclosed code span is code, `$$x_i` (and `$x_i` with `ExtDollarMath`)
is math, whose `_` and `*` are TeX rather than emphasis,
`[the docs](https://ex` is a link whose destination is still coming (not
clickable yet), and an open code fence shows as code. Lines that could still turn into something else wait for
what follows them: a line of a few markers, as `-` or `==`, which would
make the paragraph above a heading, and a table's header, which waits
for its delimiter row so that it does not flash as a paragraph of pipes.
Blocks already finished are never changed, and a reply shown with
`View` once done is exactly as CommonMark reads it.

### Pace and fade

Tokens arrive in bursts. `Stream` shows text at an even pace instead,
a word at a time, catching up with what came within a quarter of a
second, and each word fades in. `Options.Reveal` changes how:

```go
app.md.Reveal = &markdown.Reveal{
	Unit:    markdown.ByLine,         // a line at a time, once complete
	CatchUp: 400 * time.Millisecond,  // text waits at most this long
	Fade:    500 * time.Millisecond,  // and fades in over this
}
```

`ByWord` (the default), `ByCharacter`, `BySentence`, `ByLine` and
`ByParagraph` show a unit at a time, the last three only once it is
complete: a sentence ends at a period, a question or exclamation mark
followed by a space, an ideographic full stop or the end of a line.
`AtOnce` shows text as it comes. A negative `Fade` turns fading off, as does the
desktop's Reduce Motion. Text fades by its color and the rest of blocks
(list markers, code blocks, a table's rows) by their opacity, which
paint only: nothing is laid out again for it. What is still to reveal
when the message switches to `View` keeps coming in.

### Once a reply has shown

The last bytes of a reply arrive before its last words show: the pace
is still catching up, and they still fade in. `Revealed`, given the
element `View` or `Stream` returned, reports whether it shows all of its
text, done fading in, so that a chat shows a reply's time and actions
then, not ahead of words still coming in:

```go
var md ui.Element
if m.Done {
	md = markdown.View(c, m.Text, &app.md)
} else {
	md = markdown.Stream(c, m.Text, &app.md)
}
if m.Done && markdown.Revealed(md) {
	ui.Text(c, m.Time).TextColor(c.Theme().TextMuted)
}
```

It tells what the frame shows: read it after the view, and build what it
decides below it. The view asks for frames while it reveals and fades,
so the time shows on its own once the fade ends. Text that shows at once
(`AtOnce`, a `View` of text no `Stream` left to reveal) is revealed in
its first frame, and without a fade (a negative `Fade`, Reduce Motion)
once its last word shows. More text makes it false again. Of a `Stream`
it tells only that what came so far shows, so check that the reply is
complete too, as `m.Done` above.

## A chat

Show each message as a row of a [list](list.md) that follows its end:
the list keeps the end in view as the reply grows, gliding up to it with
`FollowAnimation`, until the user scrolls up, and follows again once
they scroll back to the end. `AtEnd` and `ScrollToEnd` make a button
back to the latest.

```go
app.list.Key = func(i int) any { return app.messages[i].ID }
app.list.FollowEnd = true
app.list.FollowAnimation = &ui.Smooth
ui.List(c, &app.list, len(app.messages), func(i int) {
	app.message(c, app.messages[i])
}).Grow(1).Gap(16).Justify(ui.End)
```

The [chat example](../../examples/markdown-chat) streams canned replies
in this way.

## Selecting and copying

A view's text is one selection across its blocks, as in a browser. Copy
puts the selected text on the clipboard, with blank lines between
paragraphs, a list's items on lines of their own and a table's cells
separated by tabs; as HTML, so that it pastes with its formatting into a
[rich text editor](rich-text-editor.md), mail or a word processor; and as
Markdown (`transfer.Markdown`). The context menu's Copy as Markdown, and Shift+Cmd+C
(Shift+Ctrl+C on Linux and Windows), copy the source of the selection,
completing the emphasis, code and links it cuts through, or the whole
document when nothing is selected. Text keeps its selection as a reply
streams in.

`ShareSelection` leaves the selection to a `Selectable` container around
the view, as one across a chat's messages.

## Links and images

Links to `http`, `https` and `mailto` addresses (`ui.SafeToOpen`, as the
rich text editor opens them) open as a [link](link.md)'s do, and show their address in a tooltip; a path
without a scheme goes there in the router around the view; `#a-heading`
scrolls to the heading of that anchor (`Slug`); and links of other
schemes, as `javascript:`, do nothing. `OnLink` handles every click
instead:

```go
app.md.OnLink = func(url string) { app.open(url) }
```

`LinkMenu` gives each link a [context menu](context-menu.md) of its own,
given its destination, as a chat whose replies link files offers Copy
Path and Reveal in Finder for the file right-clicked. The link's menu
replaces the one around the view, as its `Selectable` container's, and
`EditItems` puts the text's Copy and Select All back in it; links to
`#a-heading`, footnote references and a link whose destination is still
coming keep the menu around them:

```go
app.md.LinkMenu = func(m *ui.Menu, url string) {
	if m.Item("Copy Path").Chosen() {
		app.copyPath(url)
	}
	if m.Item("Reveal in Finder").Chosen() {
		app.reveal(url)
	}
	m.Separator()
	m.EditItems()
}
```

`Anchor` opens a document at a heading, given its slug with or without
the `#`: a docs reader that opens `guide.md#install` from a search hit,
or that comes back where its user was. The view scrolls to it as the
document shows, as another document replaces it and as `Anchor`
changes; `AnchorVersion` asks for the same heading again. An `OnLink`
that handles `#a-heading` links sets it:

```go
app.md.OnLink = func(url string) {
	if strings.HasPrefix(url, "#") {
		app.md.Anchor = url
		app.md.AnchorVersion++
		return
	}
	app.open(url)
}
```

`LinkIcon` puts an icon at the start of a link's text, from its
destination and the size of the text around it: a box as `InlineMath`
makes, which clicks as the link and stays on the line of its first
word. It copies as nothing, as text, HTML and Markdown. When it returns
false, the link has none; a link still streaming in has none yet. Links
take their color from `Theme.Link`:

```go
app.md.LinkIcon = func(dest string, size float32) (markdown.InlineObject, bool) {
	if !strings.HasPrefix(dest, "http") {
		return markdown.InlineObject{}, false
	}
	return markdown.InlineObject{
		Width: size * 1.2, Height: size * 0.86, Depth: size * 0.14,
		Draw: func(p *ui.Painter, r ui.Rect) {
			p.Icon(globe, ui.Rect{X: r.X, Y: r.Y, W: size, H: size}, app.md.Theme.Link)
		},
	}, true
}
```

Images show their description until `LoadImage` loads them, on a
goroutine of their own; without it, they stay descriptions, as a view
loads nothing from the network by itself. `FileImages(dir)` loads the
files of a directory. A paragraph of images alone shows them as large
as they are, at most as wide as the view.

```go
app.md.LoadImage = markdown.FileImages(filepath.Dir(app.path))
```

## Code

Fenced code is highlighted in its language with the tokenizers and the
colors of the [code editor](code-editor.md), a line at a time from the
state of the line before, so that a block streaming in highlights only
its new lines; its lines are laid out in chunks, so that it lays out
only its last chunk again as it grows. A block has its language and a
Copy button above its code, which scrolls sideways (`WrapCode` wraps it
instead). The button waits for a block being written to close.

`CodeBlock` builds code blocks in place of `DefaultCodeBlock`, and
`Code` the blocks of the languages it lists, as diagrams. Each gets the
block's language, its code and whether its fence is still open, and can
use the parts of the default:

```go
app.md.Code = map[string]func(*ui.Context, *markdown.CodeBlockView){
	"mermaid": func(c *ui.Context, b *markdown.CodeBlockView) {
		if b.Open { // still streaming: show the code until it closes
			markdown.DefaultCodeBlock(c, b)
			return
		}
		ui.Image(c, app.diagram(b.Code()))
	},
}
app.md.CodeBlock = func(c *ui.Context, b *markdown.CodeBlockView) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, b.Language).Bold()
		b.Body(c)       // the highlighted, selectable code
		b.CopyButton(c) // the copy button
	})
}
```

## Math and HTML

Math between `$$` and `$$`, `\[` and `\]`, or `\(` and `\)` shows as its
TeX in the code font, as language models write it; `$…$` too with the
`ExtDollarMath` extension, except where the dollars read as money.
`Math` builds blocks of display math instead, as with a renderer of
your own; a block still streaming is `Open` to `Block`, which can show
its TeX until it closes.

`InlineMath` makes math within text an object of the paragraph, from
its TeX and the size of the text around it: a box `Width` wide,
`Height` above the baseline and `Depth` below it, which wraps with the
words and which `Draw` paints. Copied, as text or Markdown, it is its
source, as `$x_i$`. When it returns false, the math shows as TeX. It is
asked whenever its paragraph changes, as each frame of one streaming in,
so keep what it makes:

```go
app.md.InlineMath = func(tex string, size float32) (markdown.InlineObject, bool) {
	f, err := app.formula(tex, size) // laid out once, kept by TeX and size
	if err != nil {
		return markdown.InlineObject{}, false
	}
	return markdown.InlineObject{Width: f.W, Height: f.Ascent, Depth: f.Descent, Draw: f.Draw}, true
}
```

Raw HTML shows as text, so that a reply cannot hide anything. Within
text, the tags of formatting that models write (`<b>`, `<i>`, `<u>`,
`<s>`, `<code>`, `<kbd>`, `<mark>`, `<sub>`, `<sup>` and `<br>`) apply;
`HTML` builds HTML blocks, as for `<details>` or `<think>`.

## Styling

A view looks as the window does, in its fonts and colors. `Theme` sets
what changes, the rest taking `DefaultTheme`'s values for the window's
theme, so that a theme of one field changes one thing:

```go
app.md.Theme = &markdown.Theme{
	FontSize:     15,
	Link:         ui.Hex("#7c3aed"),
	CodeFont:     "JetBrains Mono",
	BlockSpacing: 1,
	Headings: [6]markdown.HeadingStyle{
		{Size: 1.8, Weight: 800, Rule: true},
		{Size: 1.4, Rule: true},
	},
}
```

A theme sets the font, the code font and their line heights; the colors
of text, secondary text, links, strong text and code; the size of code;
the spacing between blocks and list items; each heading level's size,
weight, color, spacing and rule; list indentation; block quotes' bar and
text; code blocks' background, border, corners, padding and token colors
(an `editor.Theme`); tables' lines, header, stripes, padding and widest
column; marked text; and check boxes. Lengths are in ems, times the font
size, so that a theme scales with it. `LinkWeight` is the least weight of
links' text, for links that read as links where color alone doesn't, in a
gray print theme or to a colorblind reader.

`Block` replaces how any block shows: it is asked first for each block,
and builds it when it returns true, from the document's blocks and
inlines (see [the parser](#the-parser)):

```go
app.md.Block = func(c *ui.Context, d *parser.Document, i int) bool {
	b := d.Block(i)
	if b.Kind != parser.ThematicBreak {
		return false
	}
	ui.Text(c, "⁂").TextAlign(ui.Center).AlignSelf(ui.Stretch)
	return true
}
```

## Accessibility

Headings are headings of their level, lists and their items lists and
list items, links links, tables tables of rows, column headers and
cells, task items check boxes, checked or not, images images named by
their descriptions, and code blocks groups named by their language,
with a Copy code button.

## The parser

The parser is a package of its own, `ui/markdown/parser`, which
imports nothing of ui, so that what only reads Markdown (a rich text
editor importing it, a server rendering HTML) leaves the view out.
`Parse` reads a string, `Append` adds text and
`Edit` changes a range, each parsing again only what the change touches.
An edit parses from the top-level block before it until a block starts,
with nothing open, where one started before, from which it keeps the
blocks it had: typing in a long document's preview parses a few blocks.
A link defined later than where it is used makes the blocks using it
parse their inlines again, and only those.

```go
d := parser.Parse(src)
d.Append(token)
for i := range d.Children(0) { // the top-level blocks
	b := d.Block(i)
	if b.Kind == parser.CodeBlock {
		fmt.Println(b.Language(), d.Lines(i))
	}
}
html := d.HTML()
```

Blocks and inlines are flat slices in document order, each with its
byte range in the source (`Start` and `End`) and the index after its
children (`Next`). `Inlines` parses a block's inlines the first time it
is asked. `New(ext)` reads other extensions than `Default`, and
`SetStreaming` completes the end as `Stream` does.

It passes all 652 examples of the CommonMark 0.31.2 spec and those of
GitHub's extensions, takes linear time on the inputs that are slow in
other parsers, as cmark does, and depends on nothing outside the
standard library.

One rule is relaxed for Chinese, Japanese and Korean, which put no space
around emphasis: next to a CJK character, full-width punctuation or a
curly quote, `*` and `~~` open and close even where punctuation would
stop them in CommonMark, so `**结论。**后文` is bold. `_` and a single `~`
keep CommonMark's rule. No example of the spec has CJK text there, so
all of them still pass.

## Performance

Measured on an Apple M5 with `CGO_ENABLED=0 GOMAXPROCS=1`, on a 50 KB
reply of headings, nested lists, code, a table and links:

| | Time | Allocated |
|---|---:|---:|
| Parsing it whole, inlines included | 0.3 ms | 0.8 MB, 1,300 allocations |
| Appending a token of 4 bytes, the tail's inlines parsed | 2 µs | 600 B, 2 allocations |
| A frame showing it whole, nothing changing | 1.3 ms | 16 KB, 350 allocations |
| A frame with a token of 16 bytes appended | 0.8 ms | 36 KB, 350 allocations |

Frames build every block's elements, as views do, while the parse, the
spans of paragraphs and their text layouts are kept: a document much
longer than a chat's reply is best shown a section at a time.

<!-- repository-only:start -->

## Run the benchmarks and tests

```sh
CGO_ENABLED=0 GOMAXPROCS=1 go test ./ui/markdown -run '^$' -bench . -benchmem
go test ./ui/markdown -run 'TestCommonMark|TestGFM|TestAppend|TestEdits|TestPathological' -v
```

<!-- repository-only:end -->
