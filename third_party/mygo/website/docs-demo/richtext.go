package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

func init() {
	bold := richtext.Attributes{Weight: 700}
	doc := richtext.NewDocument(richtext.TextOf(
		richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 2}, Runs: []richtext.Run{{Text: "Trip to Lisbon"}}},
		richtext.Paragraph{Runs: []richtext.Run{{Text: "Flights land at "}, {Text: "10:40", Attributes: bold},
			{Text: ". The hotel is in "}, {Text: "Alfama", Attributes: richtext.Attributes{Italic: true}}, {Text: "."}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks, Checked: true}, Runs: []richtext.Run{{Text: "Passport"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks}, Runs: []richtext.Run{{Text: "Adapter for "}, {Text: "type F", Attributes: richtext.Attributes{Code: true}}}},
	))
	demo("rich-text-editor", func(c *ui.Context) {
		t := c.Theme()
		f := doc.Formatting()
		ui.Row(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			for _, b := range []struct {
				label string
				on    bool
				run   func()
			}{
				{"B", f.Bold == richtext.FormatOn, doc.ToggleBold},
				{"I", f.Italic == richtext.FormatOn, doc.ToggleItalic},
				{"U", f.Underline == richtext.FormatOn, doc.ToggleUnderline},
				{"H", f.Paragraph.Heading > 0, func() { doc.SetHeading(2 - min(f.Paragraph.Heading, 2)) }},
				{"•", f.Paragraph.List == richtext.Bullets, func() { doc.ToggleList(richtext.Bullets) }},
				{"☐", f.Paragraph.List == richtext.Tasks, func() { doc.ToggleList(richtext.Tasks) }},
			} {
				btn := ui.Button(c, b.label).KeepFocus().Padding(3, 9)
				if b.on {
					btn.Background(t.Accent.Alpha(0.18))
				}
				if btn.Clicked() {
					b.run()
				}
			}
		})
		richtext.Editor(c, doc, richtext.EditorOptions{Label: "Note"}).AlignSelf(ui.Stretch).Height(190)
		words := 0
		inWord := false
		for _, r := range doc.Text().PlainText() {
			if r == ' ' || r == '\n' {
				inWord = false
			} else if !inWord {
				inWord = true
				words++
			}
		}
		status(c, fmt.Sprintf("%d words · %d paragraphs", words, doc.Text().ParagraphCount()))
	})
}
