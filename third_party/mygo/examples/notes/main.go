// Notes is a notes app in native UI: notes in a sidebar, each edited as
// rich text (package ui/richtext), with a toolbar of its formats, the
// Format menu, links, Markdown shortcuts and export, and the notes saved
// as JSON in the user's configuration directory.
//
//	go run ./examples/notes
//
// It runs in browsers too, where it keeps the notes in memory:
//
//	go run ./cmd/mygo dev -web examples/notes
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// note is a note: its text, saved, and the document the editor edits.
type note struct {
	ID      int           `json:"id"`
	Text    richtext.Text `json:"text"`
	Updated time.Time     `json:"updated"`
	doc     *richtext.Document
}

// document returns the note's document, made the first time.
func (n *note) document() *richtext.Document {
	if n.doc == nil {
		n.doc = richtext.NewDocument(n.Text)
	}
	return n.doc
}

// title returns the first line of the note, or "New Note".
func (n *note) title() string {
	t := n.Text
	if n.doc != nil {
		t = n.doc.Text()
	}
	s := t.PlainText()
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if s = strings.TrimSpace(s); s == "" {
		return "New Note"
	}
	return s
}

type app struct {
	notes    []*note
	selected string // the ID of the note shown
	next     int
	// linkOpen shows the popover asking for a link's URL.
	linkOpen bool
	linkURL  string
	path     string // where the notes are saved, "" in browsers
}

func (a *app) current() *note {
	for _, n := range a.notes {
		if strconv.Itoa(n.ID) == a.selected {
			return n
		}
	}
	if len(a.notes) == 0 {
		a.add()
	}
	a.selected = strconv.Itoa(a.notes[0].ID)
	return a.notes[0]
}

// add adds an empty note at the top, and shows it.
func (a *app) add() *note {
	a.next++
	n := &note{ID: a.next, Updated: time.Now(), Text: richtext.TextOf(richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 1}})}
	a.notes = append([]*note{n}, a.notes...)
	a.selected = strconv.Itoa(n.ID)
	return n
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	n := a.current()
	doc := n.document()
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		n = a.add()
		doc = n.document()
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(230).Shrink(0).PaddingY(12).Gap(8).Background(t.Surface).Children(func() {
			ui.Row(c).Padding(0, 16).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Notes").Bold().Grow(1)
				if ui.Button(c, "New").Label("New note").Clicked() {
					a.add()
				}
			})
			ui.Sidebar(c, &a.selected, func() {
				for _, n := range a.notes {
					ui.SidebarItem(c.Key(n.ID), strconv.Itoa(n.ID), nil, n.title())
				}
			}).Grow(1).Label("Notes")
		})
		ui.Column(c).Grow(1).MinWidth(0).Padding(12, 20).Gap(10).Children(func() {
			a.toolbar(c, doc)
			changed := richtext.Editor(c.Key(n.ID), doc, richtext.EditorOptions{
				Label:         "Note",
				Placeholder:   "Title",
				OnLinkCommand: func() { a.openLink(doc) },
			}).Grow(1).Changed()
			if changed {
				n.Text, n.Updated = doc.Text(), time.Now()
				a.save()
			}
			words := len(strings.FieldsFunc(doc.Text().PlainText(), func(r rune) bool { return unicode.IsSpace(r) }))
			ui.Textf(c, "%d words · edited %s", words, n.Updated.Format("Jan 2, 15:04")).FontSize(12).TextColor(t.TextMuted)
		})
	})
}

// toolbar formats the selection of doc, showing its formats.
func (a *app) toolbar(c *ui.Context, doc *richtext.Document) {
	t := c.Theme()
	f := doc.Formatting()
	button := func(label, tip string, on bool, run func()) ui.Element {
		b := ui.Button(c, label).KeepFocus().Padding(4, 9).MinWidth(30).Tooltip(tip)
		if on {
			b.Background(t.Accent.Alpha(0.18))
		}
		if b.Clicked() {
			run()
		}
		return b
	}
	divider := func() { ui.Box(c).Width(1).Height(20).Background(t.Border) }
	ui.Row(c).Gap(4).Wrap().AlignItems(ui.Center).Children(func() {
		style := map[int]string{0: "Body", 1: "Title", 2: "Heading", 3: "Subheading"}[f.Paragraph.Heading]
		if f.ParagraphsMixed {
			style = ""
		}
		if ui.Select(c, &style, []string{"Title", "Heading", "Subheading", "Body"}).KeepFocus().Width(130).Label("Paragraph style").Changed() {
			doc.SetHeading(map[string]int{"Title": 1, "Heading": 2, "Subheading": 3}[style])
		}
		divider()
		button("B", "Bold (⌘B)", f.Bold == richtext.FormatOn, doc.ToggleBold).Bold()
		button("I", "Italic (⌘I)", f.Italic == richtext.FormatOn, doc.ToggleItalic).Italic()
		button("U", "Underline (⌘U)", f.Underline == richtext.FormatOn, doc.ToggleUnderline).Underline()
		button("S", "Strikethrough (⇧⌘X)", f.Strikethrough == richtext.FormatOn, doc.ToggleStrikethrough).Strikethrough()
		button("</>", "Code (⌘E)", f.Code == richtext.FormatOn, doc.ToggleCode)
		link := button("Link", "Link (⌘K)", f.Link != "", func() { a.openLink(doc) })
		ui.Popover(c, link, &a.linkOpen, func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				input := ui.TextInput(c, &a.linkURL).Placeholder("https://").Width(260).AutoFocus().Label("URL")
				if input.Submitted() || ui.PrimaryButton(c, "Link").Clicked() {
					doc.SetLink(strings.TrimSpace(a.linkURL))
					a.linkOpen = false
				}
				if f.Link != "" && ui.Button(c, "Remove").Clicked() {
					doc.SetLink("")
					a.linkOpen = false
				}
			})
		})
		divider()
		button("•", "Bulleted list (⇧⌘8)", f.Paragraph.List == richtext.Bullets, func() { doc.ToggleList(richtext.Bullets) })
		button("1.", "Numbered list (⇧⌘7)", f.Paragraph.List == richtext.Numbers, func() { doc.ToggleList(richtext.Numbers) })
		button("☐", "Checklist (⇧⌘L)", f.Paragraph.List == richtext.Tasks, func() { doc.ToggleList(richtext.Tasks) })
		button("❝", "Quote", f.Paragraph.Quote, doc.ToggleQuote)
		button("{}", "Code block", f.Paragraph.CodeBlock, doc.ToggleCodeBlock)
		divider()
		if ui.Button(c, "Copy Markdown").KeepFocus().Clicked() {
			c.Services().WriteClipboard(doc.Text().Markdown())
		}
	})
}

// openLink asks for the URL of a link over the selection.
func (a *app) openLink(doc *richtext.Document) {
	a.linkURL = doc.Formatting().Link
	a.linkOpen = true
}

// load reads the saved notes, or makes the first.
func (a *app) load() {
	if dir, err := os.UserConfigDir(); err == nil {
		a.path = filepath.Join(dir, "MyGo Notes", "notes.json")
		if b, err := os.ReadFile(a.path); err == nil && json.Unmarshal(b, &a.notes) == nil {
			for _, n := range a.notes {
				a.next = max(a.next, n.ID)
			}
		}
	}
	if len(a.notes) == 0 {
		a.next++
		a.notes = []*note{{ID: a.next, Updated: time.Now(), Text: welcome()}}
	}
}

// save writes the notes, where the app has a place for them.
func (a *app) save() {
	if a.path == "" {
		return
	}
	for _, n := range a.notes {
		if n.doc != nil {
			n.Text = n.doc.Text()
		}
	}
	b, err := json.MarshalIndent(a.notes, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.path), 0o755); err == nil {
		os.WriteFile(a.path, b, 0o644)
	}
}

// welcome is the first note.
func welcome() richtext.Text {
	code := richtext.Attributes{Code: true}
	return richtext.TextOf(
		richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 1}, Runs: []richtext.Run{{Text: "Welcome to Notes"}}},
		richtext.Paragraph{Runs: []richtext.Run{{Text: "Write with "}, {Text: "bold", Attributes: richtext.Attributes{Weight: 700}},
			{Text: ", "}, {Text: "italic", Attributes: richtext.Attributes{Italic: true}}, {Text: ", "},
			{Text: "links", Attributes: richtext.Attributes{Link: "https://github.com/egoist/mygo"}}, {Text: " and lists, from the toolbar, the Format menu or the keyboard."}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{Heading: 2}, Runs: []richtext.Run{{Text: "Try"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks, Checked: true}, Runs: []richtext.Run{{Text: "Open this note"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks}, Runs: []richtext.Run{{Text: "Type "}, {Text: "- ", Attributes: code}, {Text: " to start a list, "}, {Text: "# ", Attributes: code}, {Text: " for a heading"}}},
		richtext.Paragraph{Style: richtext.ParagraphStyle{List: richtext.Tasks}, Runs: []richtext.Run{{Text: "Copy the note as Markdown"}}},
		richtext.Paragraph{Runs: []richtext.Run{{Text: "Attachments sit in the line, as "},
			{Text: string(richtext.ObjectReplacement), Attributes: richtext.Attributes{Attachment: fileChip("itinerary.pdf")}},
			{Text: " does."}}},
	)
}

// fileChip returns an attachment drawn as a chip with a file's name.
func fileChip(name string) *richtext.Attachment {
	font := ui.Font{Size: 12}
	w, _ := ui.ShapeText(name, font, 0).Size()
	return &richtext.Attachment{Name: name, Width: w + 28, Height: 20, Draw: func(p *ui.Painter, r ui.Rect) {
		t := p.Theme()
		p.Fill(r, t.Accent.Alpha(0.12), r.H/2)
		p.Fill(ui.Rect{X: r.X + 8, Y: r.Y + 6, W: 7, H: 8}, t.Accent, 1.5)
		l := ui.ShapeText(name, font, 0)
		_, lh := l.Size()
		p.TextLayout(l, r.X+20, r.Y+(r.H-lh)/2, t.Accent)
	}}
}

func main() {
	a := &app{}
	a.load()
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Role: mygo.RoleFileMenu},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleFormatMenu},
			{Role: mygo.RoleWindowMenu},
		}))
		mygo.NewWindow(mygo.WindowOptions{
			Title:    "Notes",
			Width:    900,
			Height:   640,
			MinWidth: 560, MinHeight: 400,
			StateKey: "notes",
			Content:  ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
