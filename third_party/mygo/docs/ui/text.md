# Text

`ui.Text` shows text that wraps at the width it gets, and `ui.Textf` formats
it:

```go
ui.Text(c, "Settings").FontSize(20).Bold()
ui.Textf(c, "%d files, %s", n, size).TextColor(c.Theme().TextMuted)
```

## Style

`FontSize`, `FontWeight`, `Bold`, `Italic`, `Font`, `LineHeight`,
`FixedLineHeight` (in DIPs, whatever the font size), `TextColor`,
`TextAlign`, `Underline`, `WavyUnderline`, `DottedUnderline` (as links
mark hover), `Strikethrough`,
`DecorationColor` and `DecorationThickness` (of the underline and
strikethrough), `UnderlineOffset` (how far below the baseline the top of
the underline is, as CSS's `text-underline-offset`), `TextBackground` (a
highlight behind the lines), `LetterSpacing` and `FontFeatures` style it,
set on the text or on any element above it, whose texts inherit them. `FontFeatures` turns on
OpenType features of the font by tag, or sets them with `tag=value`:
`FontFeatures("tnum")` gives digits of one width for numbers that change,
`FontFeatures("liga=0")` turns ligatures off.

A link underlined a little lower than the font puts it, clear of the
descenders, as `underline-offset-4` does on the web:

```go
ui.Link(c, "Read the docs", url).Underline().DecorationThickness(1).UnderlineOffset(4)
```

`TextStroke(width, color)` strokes the outlines of the glyphs under their
fill, as CSS's `-webkit-text-stroke` with `paint-order: stroke fill`: half
the stroke shows around each glyph, a halo that keeps a label readable
over whatever is behind it. All the strokes of a text paint under all its
glyphs, so a stroke never covers the glyph beside it; the text of an
inline element strokes with its own. A chart's labels over its bars,
outlined in the background's color:

```go
ui.Text(c, "42%").TextColor(t.Text).TextStroke(3, t.Background)
```

`TextShadow(x, y, blur, color)` casts a shadow of the text, as CSS's
`text-shadow`: the shapes of its glyphs, offset by `x` and `y` DIPs and
blurred by `blur`, under its glyphs and their strokes, so a caption over a
photo stands out from it. The text of the elements inside casts it too,
or their own; a transparent color casts none. A `Span`'s `Shadow` casts
one for its text, made by `NewTextShadow`. Underlines and strikethroughs cast no shadow, and only
one shadow paints, where CSS takes a list:

```go
ui.Text(c, "Golden hour").TextColor(ui.RGB(255, 255, 255)).TextShadow(0, 1, 4, ui.RGBA(0, 0, 0, 0.6))
ui.RichText(c, ui.Span{Text: "Sale", Shadow: ui.NewTextShadow(0, 2, 3, t.Accent)})
```

`TabSize(n)` puts tab stops every `n` spaces of the font from the start of
each line, as CSS's `tab-size`, so a code block's tabs indent 2 or 4
columns; without it, tabs stop where the system's text stack puts them,
which differs between systems:

```go
ui.Text(c, source).Font("monospace").TabSize(4)
```

Text in Arabic, Hebrew and the other scripts written so reads right to
left, mixed with text reading left to right as Unicode's bidirectional
algorithm says: a paragraph reads in the direction of its first strongly
directional character. `TextAlign`'s `Start` is the left, or the right of
a paragraph reading right to left, and of any text of an element laid out
[right to left](right-to-left.md), where paragraphs without a strongly
directional character, as numbers, read right to left too.

`SingleLine` keeps text on one line, cut with an ellipsis, `MaxLines`
limits it to a few, and `Ellipsis(" →")` ends what they cut with another
mark; `NoWrap` keeps its lines whole, breaking them only at newlines.
`Balance` wraps text into lines of about the same width, as CSS's
`text-wrap: balance`: a heading that takes two lines splits near its
middle instead of leaving one word alone on the second. It keeps the
number of lines that wrapping at the full width takes, and finds the
narrowest width that still fits in that many, once when the text is laid
out. It works on `Text` and `RichText`, and is meant for short text:
headings, captions, labels on cards.

```go
ui.Text(c, "Everything you need to ship a desktop app").FontSize(28).Bold().Balance()
```

`Pretty` keeps the last line of each paragraph from holding a single
word, as CSS's `text-wrap: pretty` does at least: the line before gives it
a word, wrapped as wide as that leaves the two, and the lines before stay
as they are, so body text ends on more than one word at no cost to the
rest of the paragraph. A line before with no word to spare, as one long
word, stays as it is. `TextShape.Pretty` does the same for
`ShapeRichTextWith`.

`Selectable` lets the user select a text with the pointer, Shift and the
arrows, and copy it, as an error message or an identifier to paste
elsewhere.

Put `Selectable` on a container to share one selection across its `Text`
and `RichText` descendants:

```go
ui.Column(c).Selectable().Gap(12).Children(func() {
	ui.Text(c, "First paragraph.")
	ui.Text(c, "Second paragraph.")
})
```

A drag can cross paragraphs in either direction. Shift-click and Shift
with the arrows extend it, and dragging near a scroll container's edge
scrolls it. Cmd+C (Ctrl+C on Linux and Windows), the Edit menu's Copy,
and the context menu copy the selected text, joined with newlines in the
order the paragraphs were built. Select All selects the whole container,
including built paragraphs outside its viewport. Inline text and links
inside a paragraph are copied as part of that paragraph.

`CopyData` on the container sets what Copy writes from the parts of the
paragraphs selected, as a document joins them with blank lines as text
and writes HTML too to keep their formatting, and
`SelectedTexts` returns those parts, for a menu item copying them
otherwise. A paragraph whose text grows at its end keeps the selection.

Nested `Selectable` containers have separate selections. Buttons and
text inputs keep their own interaction; their labels do not join the
container's selection. Use `Unselectable` on a text or a container to
exclude its subtree; inline elements follow their paragraph's selection.
Changing the selected text or removing an endpoint clears the shared
selection. A virtualized list only includes the paragraphs it has built,
so use a regular scroll container when selecting an entire document.

The highlight is the theme's `Selection`; `SelectionColor` sets another
for an element and the text inside it, as text on a colored bubble needs
one that shows on it. A right-click on selectable text shows Copy and
Select All; a `ContextMenu` on the text or its `Selectable` container
replaces that menu, and `Menu.EditItems` puts Copy and Select All in it.

The caret of a text input is the theme's `Accent`; `CaretColor` sets another
for an input and the inputs inside an element, as CSS's `caret-color` does.

## Rich text

`ui.RichText` mixes styles in one paragraph: each `ui.Span` sets what it
changes (font, size, weight, italics, color, underlines, strikethrough,
their color, thickness and offset, a background, letter spacing, features) over
the style of the text, and the spans wrap together. A `Radius` rounds a
span's background, which then hugs its glyphs rather than filling the
line, as code within text shows:

```go
ui.RichText(c,
	ui.Span{Text: "Saved "},
	ui.Span{Text: "report.pdf", Weight: 600},
	ui.Span{Text: " to "},
	ui.Span{Text: "Documents", Color: t.Accent, Underline: true},
)

ui.RichText(c,
	ui.Span{Text: "Run "},
	ui.Span{Text: "go test", Font: "monospace", Background: t.Surface, Radius: 4},
)

ui.RichText(c,
	ui.Span{Text: "Did you mean "},
	ui.Span{Text: "recieve", WavyUnderline: true, DecorationColor: t.Danger},
	ui.Span{Text: "? "},
	ui.Span{Text: "match", Background: ui.RGBA(250, 204, 21, 0.4)},
)
```

`PaddingX` and `PaddingY` pad a span's background, which then hugs its
glyphs too, as CSS pads an inline box: a key cap or a mention chip within
a sentence. As on the web, `PaddingX` takes room in the line, moving the
span's glyphs in and the words after it on, and lines break to fit it;
`PaddingY` only paints, over the lines above and below, which keep their
height. A span that wraps pads its piece of each line at both ends, with
its own corners, as `box-decoration-break: clone` does:

```go
ui.RichText(c,
	ui.Span{Text: "Press "},
	ui.Span{Text: "Ctrl K", Background: t.Surface, Radius: 4, PaddingX: 6, PaddingY: 2},
	ui.Span{Text: " to search"},
)
```

A span's `LetterSpacing` left zero takes the text's; `NoLetterSpacing`
makes the zero its own, as CSS's `letter-spacing: 0` on an inline box,
for a number or code that keeps its normal spacing within tracked
capitals. An element within a sentence does the same with
`LetterSpacing(0)`:

```go
ui.RichText(c,
	ui.Span{Text: "BUILD "},
	ui.Span{Text: "4.2.1", NoLetterSpacing: true},
).LetterSpacing(2)
```

A span with an `ObjectWidth` makes its object replacement characters
(U+FFFC) inline objects: boxes `ObjectWidth` wide, `ObjectHeight` above
the baseline and `ObjectDescent` below it, which wrap with the words and
which the line makes room for, as a formula's subscripts need.
`DrawObjects` draws them, after the glyphs, in the text's clip: it gets
each one's rune in the paragraph and its box.

```go
ui.RichText(c,
	ui.Span{Text: "where "},
	ui.Span{Text: "\uFFFC", ObjectWidth: f.W, ObjectHeight: f.Ascent, ObjectDescent: f.Descent},
	ui.Span{Text: " is the mean"},
).DrawObjects(func(p *ui.Painter, at int, r ui.Rect) { f.Draw(p, r) })
```

A span's `BaselineShift` raises its text that many DIPs above the
baseline, or lowers it when negative, as superscripts and subscripts are,
whatever the font has for them; the line makes room for it, and the
span's underline and strikethrough go with it:

```go
ui.RichText(c,
	ui.Span{Text: "H"},
	ui.Span{Text: "2", Size: 10, BaselineShift: -3},
	ui.Span{Text: "O boils at 100 °C"},
	ui.Span{Text: "1", Size: 10, BaselineShift: 6},
)
```

A soft hyphen (U+00AD) in a text is a place a line may break at: it
shows as a hyphen where a line breaks there, and as nothing elsewhere, as
in browsers. When the hyphen itself does not fit and the line has no
other place to break but within a word, the hyphen overflows the width.

Text in Korean, the app's language or one `Lang("ko")` sets, breaks its
lines between words, at spaces and after punctuation, as CSS's
`word-break: keep-all` has it, rather than between any two syllables; a
word longer than a line still breaks within it (see
[Languages](../i18n.md#native-ui)).

## Elements within a sentence

Text elements built in a text's `Children` continue its paragraph, as
HTML's inline elements do: each styles its own text over the paragraph's,
and keeps what elements do (`Clicked`, `Hovered`, the keyboard focus, a
`Tooltip`, a `ContextMenu`), with its words, on every line they take, as
its area. A [link](link.md) inside is a link within the sentence, which Tab
reaches and assistive technology reads as a link:

```go
ui.RichText(c).Children(func() {
	ui.Text(c, "Read ")
	ui.Link(c, "the guide", "https://example.com/guide")
	ui.Text(c, " or ")
	if ui.Text(c, "show an example").TextColor(t.Accent).Clicked() {
		app.example = true
	}
	ui.Text(c, ".")
})
```

Only text elements (`Text`, `Link`, `RichText`) go inside a text; their
sizes, padding, borders and corners do not apply, and a `Background`
highlights their text.

`Opacity` fades an element's words within the sentence, with the colors
of the elements inside it, a transparent `TextColor` paints none of them,
and `Invisible` hides them as CSS's `visibility: hidden` does: their room
in the line stays, so a word streamed into a reply can fade in, or a
placeholder hide, without moving the text around it:

```go
ui.Text(c, "").Children(func() {
	for i, word := range reply.words {
		ui.Text(c, word).Opacity(reply.fadeIn(i)) // 0 to 1 as each arrives
	}
})
```

## Fonts

Text is laid out and drawn by the system's own text engine (DirectWrite on
Windows, Core Text on macOS, Pango on Linux) in the system's font (Segoe UI,
SF, the desktop's interface font, as GTK apps have it), falling back to the
system's fonts for other scripts and emoji as native apps do, with
right-to-left text in its order. `Font("monospace")` picks the system's
monospaced font, and `ui.RegisterFont` adds your own:

```go
//go:embed Inter.ttf
var inter []byte

func init() {
	if err := ui.RegisterFont(inter, "Inter"); err != nil {
		log.Fatal(err)
	}
}
```

Then `Font("Inter")` uses it, or set it for every element in the theme's
`Font`. The fonts registered under a family make it up, apart from other
fonts of their own family: on the desktop its weight and style are matched
among them, whichever was registered first. A font's own family name still
finds it, unless the app registers other fonts under that name. So two
faces that declare the same weight, as a font's Light and Semilight, are
told apart by registering one under a family of its own.

A list of families, as `Font("Inter, Noto Sans JP")`, draws with the first
the system or the app has, and what it lacks with the next that has it,
before the system's own choice.

`ui.FontFamilies()` lists the families text can use, as a font setting
offers them: the app's registered ones first, under the names it gave them
(or their own when it gave none), then the system's, each once and sorted
without regard to case. Each `ui.FontFamily` tells whether it is
`Monospaced`, for a terminal's or an editor's font setting, and whether
the app `Registered` it:

```go
for _, f := range ui.FontFamilies() {
	if f.Monospaced {
		fonts = append(fonts, f.Name)
	}
}
```

The first call asks the system, which takes a while with many fonts
installed (tens of milliseconds with hundreds), without holding up the
windows' text; later calls reuse the system's list until `RegisterFont`,
so fonts installed while the app runs show after a restart. Browsers do
not tell what fonts they have: there the list holds the registered
families alone, and a family counts as monospaced once its font has
loaded.

Text that a widget lays out itself, as in the cells of a terminal, is
shaped once with `ui.Shape`: see [drawing](drawing.md#shaping-text).
