package ui

import (
	"strings"
	"testing"
)

// TestTextLines checks that TrackLines reports where the text area wraps
// its lines and where they are, as its caret has them.
func TestTextLines(t *testing.T) {
	long := strings.Repeat("word ", 80)
	s := "one\n" + long + "\nthree\n" + strings.Repeat("short\n", 500)
	var lines TextLines
	tt := coreNewTester(func(c *context) { coreTextArea(c, &s).TrackLines(&lines).Fill() }, 400, 300)
	tt.Frame()
	st := textAreaState(tt)
	ed, a := st.editor, st.editor.area
	if n := lines.Count(); n != 504 {
		t.Fatalf("Count = %d, want 504", n)
	}
	if rows := lines.Rows(0); len(rows) != 1 || rows[0] != 0 {
		t.Errorf("Rows(0) = %v, want [0]", rows)
	}
	rows := lines.Rows(1)
	if len(rows) < 3 || rows[0] != 0 {
		t.Fatalf("Rows(1) = %v, want several rows from 0", rows)
	}
	start := ed.buf.start(1)
	for k, r := range rows {
		_, y, _ := a.caretAt(ed, start+r, 0)
		// A row's first rune has its caret on that row.
		if want := lines.Top(1) + float32(k)*a.line.Height; float32(y) != want {
			t.Errorf("row %d starts at rune %d, at y %v; want %v", k, r, y, want)
		}
	}
	if top, want := lines.Top(2), lines.Top(1)+float32(len(rows))*a.line.Height; top != want {
		t.Errorf("Top(2) = %v, want %v under the wrapped line", top, want)
	}
	if at := lines.At(lines.Top(1) + a.line.Height*1.5); at != 1 {
		t.Errorf("At inside the wrapped line = %d, want 1", at)
	}
	if at := lines.At(-10); at != 0 {
		t.Errorf("At above = %d, want 0", at)
	}
	if at := lines.At(1e9); at != 503 {
		t.Errorf("At below = %d, want 503", at)
	}
	if h := lines.Height(); h < lines.Top(503) {
		t.Errorf("Height %v is above the last line, at %v", h, lines.Top(503))
	}

	// An edit: the lines follow the text.
	tt.Press(20, 10)
	tt.Release(20, 10)
	tt.Key(Ctrl, KeyHome)
	tt.Key(0, KeyEnter)
	tt.Key(0, KeyEnter)
	if n := lines.Count(); n != 506 {
		t.Fatalf("after the edit, Count = %d, want 506", n)
	}
	if rows := lines.Rows(3); len(rows) < 3 {
		t.Errorf("after the edit, the long line has rows %v", rows)
	}
}

// TestTextLinesBeforeLayout checks that lines not laid out yet are
// placed a line apart.
func TestTextLinesBeforeLayout(t *testing.T) {
	var lines TextLines
	if lines.Count() != 0 || lines.Top(3) <= 0 || lines.At(100) != 0 || len(lines.Rows(0)) != 1 {
		t.Errorf("zero TextLines: Count %d, Top(3) %v, At %d, Rows %v", lines.Count(), lines.Top(3), lines.At(100), lines.Rows(0))
	}
}
