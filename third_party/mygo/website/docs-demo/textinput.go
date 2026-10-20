package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

// textinputFruits are what the search field filters.
var textinputFruits = []string{"Apple", "Apricot", "Banana", "Blueberry", "Cherry", "Grape", "Lemon", "Mango", "Orange", "Peach", "Pear", "Plum"}

// textinputFonts are the combobox's options.
var textinputFonts = []string{"Arial", "Courier New", "Georgia", "Helvetica", "Times New Roman", "Trebuchet MS", "Verdana"}

// textinputCities are the autocomplete's suggestions.
var textinputCities = []string{"Amsterdam", "Berlin", "Lisbon", "London", "Madrid", "Paris", "Prague", "Rome", "Vienna"}

// textinputTags are the token field's suggestions.
var textinputTags = []string{"design", "docs", "go", "native", "performance", "release", "typescript"}

// textinputText is the text the find bar finds in.
const textinputText = "The quick brown fox jumps over the lazy dog. The dog sleeps in the sun; the fox runs on into the woods."

func init() {
	name, password, notes, greeted := "", "", "", ""
	demo("text-input", func(c *ui.Context) {
		if ui.TextInput(c, &name).Placeholder("Name").Label("Name").Width(320).Submitted() {
			greeted = name
		}
		ui.TextInput(c, &password).Placeholder("Password").Label("Password").Password().Width(320)
		ui.TextArea(c, &notes).Placeholder("Notes, on as many lines as you like").Label("Notes").Lines(2, 5).Width(320)
		if greeted != "" {
			status(c, fmt.Sprintf("Hello, %s! Notes: %d characters", greeted, len([]rune(notes))))
		} else {
			status(c, fmt.Sprintf("Enter in the name submits it. Notes: %d characters", len([]rune(notes))))
		}
	})

	document := ui.NewTextBuffer("func main() {\n\tfmt.Println(\"Hello, world\")\n}\n")
	demo("text-input-client", func(c *ui.Context) {
		ui.TextAreaBuffer(c, document).Font("monospace").Label("Document").Width(400).Height(120)
		status(c, fmt.Sprintf("%d lines, %d characters, version %d", document.LineCount(), document.Len(), document.Version()))
	})

	copies, opacity := 2.0, 0.5
	demo("number-input", func(c *ui.Context) {
		ui.Form(c, func() {
			ui.Field(c, "Copies", func() { ui.NumberInput(c, &copies, 1, 99, 1).Label("Copies").Width(140) })
			ui.Field(c, "Opacity", func() { ui.NumberInput(c, &opacity, 0, 1, 0.05).Label("Opacity").Width(140) })
		})
		status(c, fmt.Sprintf("%d copies at %d%% opacity", int(copies), int(math.Round(opacity*100))))
	})

	query := ""
	demo("search-field", func(c *ui.Context) {
		ui.SearchField(c, &query).Label("Search fruits").Width(320)
		var found []string
		for _, f := range textinputFruits {
			if strings.Contains(strings.ToLower(f), strings.ToLower(query)) {
				found = append(found, f)
			}
		}
		if len(found) > 0 {
			ui.Row(c).Wrap().Gap(6).Justify(ui.Center).Width(320).Children(func() {
				for _, f := range found {
					ui.Badge(c.Key(f), f)
				}
			})
		}
		status(c, fmt.Sprintf("%d of %d fruits", len(found), len(textinputFruits)))
	})

	font := "Georgia"
	demo("combobox", func(c *ui.Context) {
		ui.Combobox(c, &font, textinputFonts).Label("Font").Width(240)
		ui.Text(c, "The quick brown fox").Font(font).FontSize(20)
		ui.Spacer(c) // room below for the options
	})

	city := ""
	demo("autocomplete", func(c *ui.Context) {
		if ui.Autocomplete(c, &city, textinputCities).Label("City").Placeholder("City").Width(240).Submitted() {
			city = strings.TrimSpace(city)
		}
		if city == "" {
			status(c, "Type a city, or any place")
		} else {
			status(c, "Shipping to "+city)
		}
		ui.Spacer(c) // room below for the suggestions
	})

	tags := []string{"go", "native"}
	demo("token-field", func(c *ui.Context) {
		ui.TokenField(c, &tags, textinputTags).Label("Tags").Width(360)
		if len(tags) == 0 {
			status(c, "No tags: type one and press Enter")
		} else {
			status(c, fmt.Sprintf("%d tags: %s", len(tags), strings.Join(tags, ", ")))
		}
		ui.Spacer(c) // room below for the suggestions
	})

	file, renamed := "Quarterly report.pdf", 0
	demo("editable-text", func(c *ui.Context) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconFile)
			if ui.EditableText(c, &file).Changed() {
				renamed++
			}
		})
		if renamed == 0 {
			status(c, "Double-click the name to rename it")
		} else {
			status(c, fmt.Sprintf("Renamed %d times", renamed))
		}
	})

	finding, find, match := true, "fox", 0
	demo("find-bar", func(c *ui.Context) {
		t := c.Theme()
		// The app finds: the matches of the query, by their byte offsets.
		var matches []int
		if find != "" {
			lower, q := strings.ToLower(textinputText), strings.ToLower(find)
			for i := 0; ; {
				j := strings.Index(lower[i:], q)
				if j < 0 {
					break
				}
				matches = append(matches, i+j)
				i += j + len(q)
			}
		}
		if c.Shortcut(ui.Cmd, ui.KeyF) {
			finding = true
		}
		ui.Column(c).Width(480).BorderWidth(1).BorderColor(t.Border).Radius(8).Clip().Children(func() {
			ui.FindBar(c, &finding, &find, len(matches), &match)
			ui.Column(c).Padding(12).Gap(10).Children(func() {
				// The text, its matches highlighted, the current one in the
				// accent color.
				var spans []ui.Span
				at := 0
				for i, m := range matches {
					spans = append(spans, ui.Span{Text: textinputText[at:m]})
					s := ui.Span{Text: textinputText[m : m+len(find)], Background: t.Selection}
					if finding && i == match {
						s.Background, s.Color = t.Accent, t.AccentText
					} else if !finding {
						s.Background = ui.Color{}
					}
					spans = append(spans, s)
					at = m + len(find)
				}
				spans = append(spans, ui.Span{Text: textinputText[at:]})
				ui.RichText(c, spans...)
				if !finding && ui.Button(c, "Find…").AlignSelf(ui.Start).Clicked() {
					finding = true
				}
			})
		})
	})
}
