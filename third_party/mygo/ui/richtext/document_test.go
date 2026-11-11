package richtext

import "testing"

// TestDocumentLinkEndsWithLine checks that typing at the end of a line
// linked whole, its newline too, doesn't continue the link.
func TestDocumentLinkEndsWithLine(t *testing.T) {
	d := NewDocument(NewText("abc\nxyz"))
	d.SetSelection(0, 4)
	d.SetLink("u")
	d.SetSelection(3, 3)
	if l := d.TypingAttributes().Link; l != "" {
		t.Fatalf("typing link %q", l)
	}
}

// TestDocumentFormattingFromNewline checks the formats of a selection
// that starts at a newline: those of the runes after it.
func TestDocumentFormattingFromNewline(t *testing.T) {
	d := NewDocument(TextOf(Paragraph{Runs: []Run{{Text: "plain"}}}, Paragraph{Runs: []Run{{Text: "BOLD", Attributes: Attributes{Weight: 700}}}}))
	d.SetSelection(5, 10)
	if f := d.Formatting(); f.Bold != FormatOn {
		t.Fatalf("bold %v", f.Bold)
	}
	d.ToggleBold()
	if d.Text().AttributesAt(7).Bold() {
		t.Fatal("⌘B didn't take bold off")
	}
}

// TestDocumentInsertLeavesCaret checks that inserting over a selection
// leaves the caret after what was inserted, as typing and pasting do.
func TestDocumentInsertLeavesCaret(t *testing.T) {
	d := NewDocument(NewText("hello world"))
	d.SetSelection(6, 11)
	d.InsertText("there")
	if a, c := d.Selection(); a != 11 || c != 11 {
		t.Fatalf("InsertText: selection %d, %d", a, c)
	}
	d.SetSelection(0, 1)
	d.Insert(NewText("A\nB"))
	if a, c := d.Selection(); a != 3 || c != 3 {
		t.Fatalf("Insert: selection %d, %d", a, c)
	}
}

// TestDocumentFormatToEnd checks that formatting to the end of the text
// formats what typing gets in a last empty paragraph.
func TestDocumentFormatToEnd(t *testing.T) {
	d := NewDocument(NewText("abc\n"))
	d.SetSelection(0, 4)
	d.ToggleBold()
	d.SetSelection(4, 4)
	if !d.TypingAttributes().Bold() {
		t.Fatal("typing at the end isn't bold")
	}
}

// TestDocumentNoOpSteps checks that commands changing nothing make no
// step of undo, and that a full history keeps its steps through them.
func TestDocumentNoOpSteps(t *testing.T) {
	d := NewDocument(NewText("abc"))
	d.SetSelection(0, 3)
	v := d.Version()
	d.SetLink("")
	d.ToggleChecked(0)
	if d.Version() != v || d.CanUndo() {
		t.Fatalf("version %d → %d, can undo %v", v, d.Version(), d.CanUndo())
	}
	for range 200 {
		d.InsertText("x")
	}
	n := len(d.undo)
	d.Edit(func(*Edit) {})
	if len(d.undo) != n {
		t.Fatalf("undo steps %d → %d", n, len(d.undo))
	}
}
