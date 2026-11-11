package main

import (
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// markdownSample is what the Markdown page shows, and streams.
const markdownSample = "## Streaming Markdown\n\n" +
	"MyGo parses **CommonMark** and *GitHub Flavored Markdown* as it streams in: only the block being written is parsed again, and the end shows what its half-written syntax will become, so `**bol` is already **bold**.\n\n" +
	"### What it shows\n\n" +
	"1. Headings, paragraphs, *emphasis*, **strong**, ~~strikethrough~~ and `code`\n" +
	"2. Lists, ordered and not:\n" +
	"   - nested items\n" +
	"   - and [links](https://github.com/egoist/mygo) that open in the browser\n" +
	"3. Task lists:\n" +
	"   - [x] parse incrementally\n" +
	"   - [ ] take over the world\n\n" +
	"> Text can be selected across blocks, and copied as text or, from the context menu, as Markdown.\n\n" +
	"```go\n" +
	"// Stream shows a reply as it comes, and View once it is done.\n" +
	"func message(c *ui.Context, m *Message) {\n" +
	"\tif m.Done {\n" +
	"\t\tmarkdown.View(c, m.Text, &app.md)\n" +
	"\t} else {\n" +
	"\t\tmarkdown.Stream(c, m.Text, &app.md)\n" +
	"\t}\n" +
	"}\n" +
	"```\n\n" +
	"| Renderer | Streaming | Selection |\n" +
	"|---|:-:|---|\n" +
	"| Re-parse everything | O(n²) | per block |\n" +
	"| **MyGo** | O(the open block) | across blocks |\n\n" +
	"Math such as $$e^{i\\pi} + 1 = 0$$ shows as its TeX unless the app renders it, and footnotes[^1] are listed at the end.\n\n" +
	"---\n\n" +
	"[^1]: Like this one.\n"

// markdownDemo is the state of the Markdown page.
type markdownDemo struct {
	opts   markdown.Options
	text   string
	done   bool
	unit   int
	wrap   bool
	stream int // the stream going on, so that a new one stops it
	shown  bool
}

var (
	revealNames = []string{"Word", "Character", "Sentence", "Line", "Paragraph", "At once"}
	revealUnits = []markdown.RevealUnit{markdown.ByWord, markdown.ByCharacter, markdown.BySentence, markdown.ByLine, markdown.ByParagraph, markdown.AtOnce}
)

func (g *gallery) markdownPage(c *ui.Context) {
	m := &g.md
	if !m.shown {
		m.text, m.done, m.shown = markdownSample, true, true
	}
	t := c.Theme()
	ui.Text(c, "markdown.View shows Markdown, and markdown.Stream a reply as it streams in, at an even pace, fading in.").TextColor(t.TextMuted)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Wrap().Children(func() {
		if ui.PrimaryButton(c, "Stream it").Clicked() {
			g.streamMarkdown()
		}
		ui.Segmented(c, &m.unit, revealNames...)
		ui.Checkbox(c, &m.wrap, "Wrap code")
	})
	m.opts.Reveal = &markdown.Reveal{Unit: revealUnits[m.unit]}
	m.opts.WrapCode = m.wrap
	box := ui.Column(c).Padding(20, 24).Radius(t.Radius+4).Border(1, t.Border).Background(t.Background).MaxWidth(760)
	box.Children(func() {
		if m.done {
			markdown.View(c, m.text, &m.opts)
		} else {
			markdown.Stream(c, m.text, &m.opts)
		}
	})
}

// streamMarkdown streams the sample in, a few bytes at a time, at the
// uneven pace of a language model's tokens.
func (g *gallery) streamMarkdown() {
	m := &g.md
	m.stream++
	id := m.stream
	m.text, m.done = "", false
	if g.win == nil {
		m.text, m.done = markdownSample, true
		return
	}
	go func() {
		src := markdownSample
		for i := 0; i < len(src); {
			// Tokens of 1 to 8 bytes, in bursts.
			n := min(len(src)-i, 1+rand.IntN(8))
			for n < len(src)-i && src[i+n]&0xC0 == 0x80 {
				n++
			}
			i += n
			text := src[:i]
			time.Sleep(time.Duration(10+rand.IntN(40)) * time.Millisecond)
			g.win.Update(func() {
				if m.stream == id {
					m.text = text
				}
			})
		}
		g.win.Update(func() {
			if m.stream == id {
				m.done = true
			}
		})
	}()
}
