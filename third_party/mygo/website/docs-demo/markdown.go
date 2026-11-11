package main

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

const demoMarkdown = "### A reply, streamed\n\n" +
	"The parser reads **CommonMark** and *GitHub's* extensions as text comes. Only the block being written is parsed again, so `**bol` is already **bold**.\n\n" +
	"- [x] tables, task lists and ~~strikethrough~~\n- [ ] [links](https://github.com/egoist/mygo) and footnotes\n\n" +
	"```go\nmarkdown.Stream(c, reply, &app.md)\n```\n\n" +
	"| Strategy | Per token |\n|---|--:|\n| Re-parse all | O(n) |\n| **MyGo** | O(open block) |\n\n" +
	"Text comes in at an even pace and fades in. Pick how much shows at a time above, and stream it again.\n"

var (
	demoRevealNames = []string{"Word", "Sentence", "Line", "Paragraph"}
	demoRevealUnits = []markdown.RevealUnit{markdown.ByWord, markdown.BySentence, markdown.ByLine, markdown.ByParagraph}
)

func init() {
	var opts markdown.Options
	var started time.Time
	var scroll ui.ScrollState
	streaming, unit := false, 0
	demo("markdown", func(c *ui.Context) {
		text := demoMarkdown
		if streaming {
			// About 200 bytes a second, as a language model streams.
			n := min(len(demoMarkdown), int(c.Now().Sub(started).Seconds()*200))
			for n < len(demoMarkdown) && demoMarkdown[n]&0xC0 == 0x80 {
				n++
			}
			text = demoMarkdown[:n]
			if n == len(demoMarkdown) {
				streaming = false
			} else {
				c.AnimationFrame()
			}
			// The end stays in view while it streams, unless scrolled up.
			if scroll.Y >= scroll.MaxY-1 {
				scroll.Y = math.MaxFloat32
			}
		}
		opts.Reveal = &markdown.Reveal{Unit: demoRevealUnits[unit]}
		ui.Column(c).AlignSelf(ui.Stretch).Grow(1).MinHeight(0).Gap(12).Children(func() {
			// The controls stay at the top as the reply grows below them.
			ui.Row(c).Gap(12).AlignItems(ui.Center).Shrink(0).Children(func() {
				if ui.PrimaryButton(c, "Stream it").Clicked() {
					streaming, started = true, c.Now()
					scroll.Y = 0
				}
				ui.Segmented(c, &unit, demoRevealNames...).Label("Stream by")
			})
			ui.Scroll(c).TrackScroll(&scroll).Grow(1).MinHeight(0).Children(func() {
				if streaming {
					markdown.Stream(c, text, &opts)
				} else {
					markdown.View(c, text, &opts)
				}
			})
		})
	})
}
