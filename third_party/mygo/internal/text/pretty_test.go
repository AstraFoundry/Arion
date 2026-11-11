package text

import "testing"

// TestPretty gives a paragraph's last line a word from the line before
// rather than one alone, as narrow as that leaves it, keeping the lines
// before as they are; leaves a last line of words, a paragraph whose line
// before has no word to spare, and one MaxLines cuts as they wrap.
func TestPretty(t *testing.T) {
	s := breakSystem(0)
	s.UsePretty()
	p := Params{Text: "aaa bbb ccc ddd eee fff ggg hhh i", Style: Style{Size: 10}, Width: 160}
	if l := s.Layout(p); len(l.Lines) != 3 || lineText(l, 2) != "i" {
		t.Fatalf("not pretty, %d lines, the last %q", len(l.Lines), lineText(l, len(l.Lines)-1))
	}
	p.Pretty = true
	l := s.Layout(p)
	if len(l.Lines) != 3 || lineText(l, 0) != "aaa bbb ccc ddd " || lineText(l, 1) != "eee fff ggg " || lineText(l, 2) != "hhh i" {
		t.Fatalf("pretty: %q", texts(l))
	}
	// Laid out once, and kept.
	made := s.LayoutsMade()
	if s.Layout(p) != l || s.LayoutsMade() != made {
		t.Error("the pretty layout was made again")
	}
	// Each paragraph on its own, after a newline too.
	p.Text = "aaa bbb ccc ddd e\naaa bbb ccc ddd eee ff"
	if l := s.Layout(p); len(l.Lines) != 4 || lineText(l, 0) != "aaa bbb ccc " || lineText(l, 1) != "ddd e" || lineText(l, 3) != "eee ff" {
		t.Errorf("two paragraphs: %q", texts(l))
	}
	// A two-line paragraph, the first indented.
	p = Params{Text: "aaa bbb ccc ddd e", Style: Style{Size: 10}, Width: 160, Indent: 10, Pretty: true}
	if l := s.Layout(p); len(l.Lines) != 2 || lineText(l, 1) != "ddd e" {
		t.Errorf("indented: %q", texts(l))
	}
	for _, tc := range []struct {
		name string
		p    Params
		last string
	}{
		{"a last line of words", Params{Text: "aaa bbb ccc ddd ee ff", Width: 160}, "ee ff"},
		{"no word to spare", Params{Text: "aaaaaaaaaaaaaaaa b", Width: 160}, "b"},
		{"one line", Params{Text: "aaa b", Width: 160}, "aaa b"},
	} {
		tc.p.Style, tc.p.Pretty = Style{Size: 10}, true
		if l := s.Layout(tc.p); lineText(l, len(l.Lines)-1) != tc.last {
			t.Errorf("%s: %q", tc.name, texts(l))
		}
	}
	p = Params{Text: "aaa bbb ccc ddd e", Style: Style{Size: 10}, Width: 160, MaxLines: 1, Pretty: true}
	if l := s.Layout(p); len(l.Lines) != 1 || !l.Truncated {
		t.Errorf("cut to a line: %q", texts(l))
	}
}

func texts(l *Layout) []string {
	var out []string
	for i := range l.Lines {
		out = append(out, lineText(l, i))
	}
	return out
}
