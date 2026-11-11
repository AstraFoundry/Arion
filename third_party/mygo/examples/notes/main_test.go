package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The view runs without a window: a new note takes a title and a list
// typed with Markdown, and its title shows in the sidebar.
func TestNotes(t *testing.T) {
	a := &app{}
	a.next = 1
	a.notes = []*note{{ID: 1, Text: welcome()}}
	tt := ui.NewTester(a.view, 900, 640)
	if err := tt.Click("New"); err != nil {
		t.Fatal(err)
	}
	tt.Press(600, 300)
	tt.Release(600, 300)
	for _, r := range "Shopping\n- milk" {
		if r == '\n' {
			tt.Key(0, ui.KeyEnter)
			continue
		}
		tt.Type(string(r))
	}
	n := a.current()
	if got := n.title(); got != "Shopping" {
		t.Errorf("title %q", got)
	}
	if md := n.document().Text().Markdown(); !strings.Contains(md, "# Shopping") || !strings.Contains(md, "- milk") {
		t.Errorf("Markdown %q", md)
	}
	if !tt.HasText("Shopping") {
		t.Errorf("the sidebar shows %q", tt.Texts())
	}
}

// The paragraph style menu makes the paragraph of the caret a heading.
func TestParagraphStyle(t *testing.T) {
	a := &app{}
	a.next = 1
	a.notes = []*note{{ID: 1, Text: welcome()}}
	tt := ui.NewTester(a.view, 900, 640)
	tt.Press(600, 300)
	tt.Release(600, 300)
	if err := tt.Click("Paragraph style"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Subheading"); err != nil {
		t.Fatal(err)
	}
	if got := a.current().document().Formatting().Paragraph.Heading; got != 3 {
		t.Errorf("heading %d after choosing Subheading, want 3", got)
	}
}
