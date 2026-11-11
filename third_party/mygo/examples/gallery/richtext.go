package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// sampleNote is the text the Rich Text page starts with.
func sampleNote() richtext.Text {
	bold := richtext.Attributes{Weight: 700}
	return richtext.TextOf(
		richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 1}, Runs: []richtext.Run{{Text: "Trip to Lisbon"}}},
		richtext.Paragraph{Runs: []richtext.Run{
			{Text: "Flights land at "}, {Text: "10:40", Attributes: bold}, {Text: " on Friday. The hotel is in "},
			{Text: "Alfama", Attributes: richtext.Attributes{Italic: true}}, {Text: ", near the "},
			{Text: "castle", Attributes: richtext.Attributes{Link: "https://en.wikipedia.org/wiki/S%C3%A3o_Jorge_Castle"}}, {Text: "."},
		}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 2}, Runs: []richtext.Run{{Text: "Pack"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks, Checked: true}, Runs: []richtext.Run{{Text: "Passport"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks}, Runs: []richtext.Run{{Text: "Walking shoes"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks}, Runs: []richtext.Run{{Text: "Adapter for "}, {Text: "type F", Attributes: richtext.Attributes{Code: true}}, {Text: " sockets"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 2}, Runs: []richtext.Run{{Text: "Eat"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Numbers}, Runs: []richtext.Run{{Text: "Pastéis de nata", Attributes: bold}, {Text: " at Manteigaria"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Numbers, Indent: 1}, Runs: []richtext.Run{{Text: "Warm, with cinnamon"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Numbers}, Runs: []richtext.Run{{Text: "Grilled sardines"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{Quote: true}, Runs: []richtext.Run{{Text: "Lisbon is a city to walk, slowly, uphill."}}},
		richtext.Paragraph{Runs: []richtext.Run{{Text: "Type "}, {Text: "# ", Attributes: richtext.Attributes{Code: true}}, {Text: ", "}, {Text: "- ", Attributes: richtext.Attributes{Code: true}}, {Text: " or "}, {Text: "**bold**", Attributes: richtext.Attributes{Code: true}}, {Text: " to format as you write."}}},
	)
}

// richText shows a rich text editor with a toolbar formatting its
// selection, which shows the selection's formats.
func (g *gallery) richText(c *ui.Context) {
	t := c.Theme()
	if g.note == nil {
		g.note = richtext.NewDocument(sampleNote())
	}
	doc := g.note
	f := doc.Formatting()
	button := func(label string, on bool, run func()) {
		b := ui.Button(c, label).KeepFocus().Padding(4, 9).MinWidth(30)
		if on {
			b.Background(t.Accent.Alpha(0.18))
		}
		if b.Clicked() {
			run()
		}
	}
	ui.Row(c).Gap(4).Wrap().AlignItems(ui.Center).Children(func() {
		button("B", f.Bold == richtext.FormatOn, doc.ToggleBold)
		button("I", f.Italic == richtext.FormatOn, doc.ToggleItalic)
		button("U", f.Underline == richtext.FormatOn, doc.ToggleUnderline)
		button("S", f.Strikethrough == richtext.FormatOn, doc.ToggleStrikethrough)
		button("</>", f.Code == richtext.FormatOn, doc.ToggleCode)
		ui.Box(c).Width(1).Height(20).Background(t.Border)
		for level, name := range []string{"Body", "H1", "H2", "H3"} {
			button(name, !f.ParagraphsMixed && f.Paragraph.Heading == level, func() { doc.SetHeading(level) })
		}
		ui.Box(c).Width(1).Height(20).Background(t.Border)
		button("• List", f.Paragraph.List == richtext.Bullets, func() { doc.ToggleList(richtext.Bullets) })
		button("1. List", f.Paragraph.List == richtext.Numbers, func() { doc.ToggleList(richtext.Numbers) })
		button("☑ Tasks", f.Paragraph.List == richtext.Tasks, func() { doc.ToggleList(richtext.Tasks) })
		button("❝", f.Paragraph.Quote, doc.ToggleQuote)
		button("{ }", f.Paragraph.CodeBlock, doc.ToggleCodeBlock)
		ui.Box(c).Width(1).Height(20).Background(t.Border)
		button("Left", f.Paragraph.Align == ui.Start, func() { doc.SetAlign(ui.Start) })
		button("Center", f.Paragraph.Align == ui.Center, func() { doc.SetAlign(ui.Center) })
		button("Right", f.Paragraph.Align == ui.End, func() { doc.SetAlign(ui.End) })
		ui.Box(c).Width(1).Height(20).Background(t.Border)
		if ui.Button(c, "Undo").KeepFocus().Disabled(!doc.CanUndo()).Clicked() {
			doc.Undo()
		}
		if ui.Button(c, "Redo").KeepFocus().Disabled(!doc.CanRedo()).Clicked() {
			doc.Redo()
		}
	})
	richtext.Editor(c, doc, richtext.EditorOptions{Label: "Note"}).Height(460)
	ui.Text(c, "⌘B, ⌘I, ⌘U, ⇧⌘X, ⌘E and ⌘K format; ⇧⌘7, ⇧⌘8 and ⇧⌘L make lists, Tab and ⇧Tab indent them; ⌥⌘1–3 make headings.").FontSize(12).TextColor(t.TextMuted)
}
