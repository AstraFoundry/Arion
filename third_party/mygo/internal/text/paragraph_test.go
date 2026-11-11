package text

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// breakEngine breaks lines greedily after spaces and soft hyphens, as the
// system's engines do, leaving no room for the hyphen of a line broken at
// a soft hyphen, as all but Pango do. Every rune is 10 DIPs wide but soft
// hyphens, which are shy wide; a space ending a line takes no room, and
// a word wider than a line breaks between its runes.
type breakEngine struct {
	stubEngine
	shy float32
	// omit leaves soft hyphens without a glyph, as an engine may.
	omit bool
	// shaped counts the runes shape shaped.
	shaped int
}

func (e *breakEngine) shape(text []rune, style Style, spans []Span, width float32, rtl, wholeWords bool) []shapedLine {
	e.shaped += len(text)
	f := e.font(style)
	adv := func(r rune) float32 {
		if r == softHyphen {
			return e.shy
		}
		return 10
	}
	var lines []shapedLine
	for start := 0; start < len(text); {
		end := len(text)
		if width > 0 {
			w, opp, i := float32(0), -1, start
			for ; i < len(text); i++ {
				if text[i] != ' ' && w+adv(text[i]) > width+1e-3 {
					break
				}
				w += adv(text[i])
				if text[i] == ' ' || text[i] == softHyphen {
					opp = i + 1
				}
			}
			if i < len(text) {
				end = opp
				if end <= start {
					end = max(i, start+1)
				}
			}
		}
		run := shapedRun{font: f, start: start, end: end}
		x := float32(0)
		for i := start; i < end; i++ {
			if e.omit && text[i] == softHyphen {
				continue
			}
			run.glyphs = append(run.glyphs, Glyph{Font: f, ID: uint32(text[i]), X: x, Advance: adv(text[i]), Cluster: i, RTL: rtl})
			x += adv(text[i])
		}
		lines = append(lines, shapedLine{start: start, end: end, runs: []shapedRun{run}})
		start = end
	}
	return lines
}

func breakSystem(shy float32) *System {
	s := newSystem()
	s.eng = &breakEngine{shy: shy}
	s.UseJustify()
	return s
}

// lineText returns the runes of a line of a layout.
func lineText(l *Layout, i int) string { return string(l.Runes[l.Lines[i].Start:l.Lines[i].End]) }

// glyphsOf returns the glyphs of rune r in a line.
func glyphsOf(line Line, r int) []Glyph {
	var out []Glyph
	for _, g := range line.Glyphs {
		if g.Cluster == r {
			out = append(out, g)
		}
	}
	return out
}

// TestSoftHyphens breaks lines at soft hyphens, showing a hyphen there
// and nothing elsewhere, and breaks a line again where its hyphen would
// not fit, which all engines but Pango leave no room for.
func TestSoftHyphens(t *testing.T) {
	for _, shy := range []float32{0, 5} {
		s := breakSystem(shy)
		// Whole, the soft hyphens take no room.
		whole := s.Layout(Params{Text: "ab\u00ADcd", Style: Style{Size: 10}})
		if whole.Width != 40 || len(whole.Lines) != 1 {
			t.Fatalf("shy %v: unbroken, %d lines %v wide, want one of 40", shy, len(whole.Lines), whole.Width)
		}
		if g := glyphsOf(whole.Lines[0], 2); len(g) != 1 || g[0].Font != nil || g[0].Advance != 0 || g[0].X != 20 {
			t.Errorf("shy %v: an unbroken soft hyphen is %+v", shy, g)
		}
		if g := glyphsOf(whole.Lines[0], 3); len(g) != 1 || g[0].X != 20 {
			t.Errorf("shy %v: the rune after it is %+v", shy, g)
		}

		// "aaaa-" fits 50 with its hyphen.
		l := s.Layout(Params{Text: "aaaa\u00ADbbbb cc", Style: Style{Size: 10}, Width: 50})
		if len(l.Lines) != 3 || lineText(l, 0) != "aaaa\u00AD" || lineText(l, 1) != "bbbb " {
			t.Fatalf("shy %v: lines %q", shy, []string{lineText(l, 0), lineText(l, 1)})
		}
		hy := glyphsOf(l.Lines[0], 4)
		if len(hy) != 1 || hy[0].ID != '-' || hy[0].Advance != 10 || hy[0].X != 40 || l.Lines[0].Width != 50 {
			t.Errorf("shy %v: the hyphen is %+v, the line %v wide", shy, hy, l.Lines[0].Width)
		}

		// At 45, "aa\u00ADaa-" does not fit: the line breaks at the first.
		l = s.Layout(Params{Text: "aa\u00ADaa\u00ADbbbb", Style: Style{Size: 10}, Width: 45})
		var got []string
		for i := range l.Lines {
			got = append(got, lineText(l, i))
			if l.Lines[i].Width > 45 {
				t.Errorf("shy %v: line %d is %v wide", shy, i, l.Lines[i].Width)
			}
		}
		if want := []string{"aa\u00AD", "aa\u00AD", "bbbb"}; !slices.Equal(got, want) {
			t.Errorf("shy %v: lines %q, want %q", shy, got, want)
		}
		for i := range 2 {
			if hy := glyphsOf(l.Lines[i], l.Lines[i].End-1); len(hy) != 1 || hy[0].ID != '-' {
				t.Errorf("shy %v: line %d ends with %+v", shy, i, hy)
			}
		}
		// A line whose part of a word alone fits beside the hyphen keeps
		// the word whole, its hyphen overflowing.
		l = s.Layout(Params{Text: "aaaa\u00ADbb", Style: Style{Size: 10}, Width: 45})
		if len(l.Lines) != 2 || lineText(l, 0) != "aaaa\u00AD" || l.Lines[0].Width != 50 {
			t.Errorf("shy %v: %d lines, the first %q %v wide", shy, len(l.Lines), lineText(l, 0), l.Lines[0].Width)
		}
		// Lines whose hyphens overflow shape their own runes again, not
		// the rest of the paragraph.
		e := s.eng.(*breakEngine)
		e.shaped = 0
		l = s.Layout(Params{Text: strings.Repeat("a\u00AD", 20) + "a", Style: Style{Size: 10}, Width: 15})
		if len(l.Lines) != 21 || e.shaped > 3*41 {
			t.Errorf("shy %v: %d lines, shaping %d runes", shy, len(l.Lines), e.shaped)
		}
		// An engine that gives soft hyphens no glyph gets the hyphen too.
		om := breakSystem(shy)
		om.eng.(*breakEngine).omit = true
		l = om.Layout(Params{Text: "aaaa\u00ADbbbb cc", Style: Style{Size: 10}, Width: 50})
		if hy := glyphsOf(l.Lines[0], 4); len(hy) != 1 || hy[0].ID != '-' || hy[0].X != 40 || l.Lines[0].Width != 50 {
			t.Errorf("shy %v: without a glyph, the hyphen is %+v, the line %v wide", shy, hy, l.Lines[0].Width)
		}
		if a := glyphsOf(l.Lines[0], 3); len(a) != 1 || a[0].Runes != 1 {
			t.Errorf("shy %v: the rune before the hyphen is %+v", shy, a)
		}
		// Carets step over the hidden soft hyphen as over nothing.
		l = s.Layout(Params{Text: "ab\u00ADcd", Style: Style{Size: 10}})
		if x2, _, _ := l.Caret(2); x2 != 20 {
			t.Errorf("shy %v: caret before the soft hyphen at %v", shy, x2)
		}
		if x3, _, _ := l.Caret(3); x3 != 20 {
			t.Errorf("shy %v: caret after the soft hyphen at %v", shy, x3)
		}
	}
}

// TestSoftHyphensOfEngine checks the system's engine: soft hyphens show
// as hyphens where lines break at them, as nothing elsewhere, and lines
// fit their width with them.
func TestSoftHyphensOfEngine(t *testing.T) {
	s := Shared()
	const text = "extra\u00ADordinary\u00ADthings hap\u00ADpen"
	plain := s.Layout(Params{Text: strings.ReplaceAll(text, "\u00AD", ""), Style: Style{Size: 16}})
	whole := s.Layout(Params{Text: text, Style: Style{Size: 16}})
	if math.Abs(float64(whole.Width-plain.Width)) > 1.5 {
		t.Errorf("unbroken, the text is %v wide, %v without its soft hyphens", whole.Width, plain.Width)
	}
	hyphens := 0
	for w := float32(60); w <= 260; w += 5 {
		l := s.Layout(Params{Text: text, Style: Style{Size: 16}, Width: w})
		for i, line := range l.Lines {
			// A line broken at a soft hyphen with no other place to break
			// may overflow, as a word too long for a line does: at 60,
			// DirectWrite fits "ordinary" but not "ordinary-", and Core
			// Text neither.
			word := i < len(l.Lines)-1 && line.End > line.Start && l.Runes[line.End-1] == softHyphen &&
				!strings.ContainsAny(string(l.Runes[line.Start:line.End-1]), " \u00AD")
			if line.Width > w+0.5 && !word {
				t.Errorf("at %v, line %d (%q) is %v wide", w, i, lineText(l, i), line.Width)
			}
			for r := line.Start; r < line.End; r++ {
				if l.Runes[r] != softHyphen {
					continue
				}
				broken := r == line.End-1 && i < len(l.Lines)-1
				for _, g := range glyphsOf(line, r) {
					if shows := g.Font != nil && g.Advance > 0; shows != broken {
						t.Errorf("at %v, the soft hyphen ending line %d (%q) shows: %v", w, i, lineText(l, i), shows)
					}
				}
				if broken {
					hyphens++
				}
			}
		}
	}
	if hyphens == 0 {
		t.Error("no line broke at a soft hyphen")
	}
}

// TestIndent moves the first line in by Params.Indent and wraps it that
// much narrower, on the side its paragraph starts.
func TestIndent(t *testing.T) {
	s := breakSystem(0)
	l := s.Layout(Params{Text: "aaa bbb ccc ddd", Style: Style{Size: 10}, Width: 80, Indent: 25})
	var got []string
	for i := range l.Lines {
		got = append(got, lineText(l, i))
	}
	if want := []string{"aaa ", "bbb ccc ", "ddd"}; !slices.Equal(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
	if l.Lines[0].X != 25 || l.Lines[1].X != 0 || l.Lines[2].X != 0 {
		t.Errorf("lines at %v, %v and %v", l.Lines[0].X, l.Lines[1].X, l.Lines[2].X)
	}
	if x, _, _ := l.Caret(0); x != 25 {
		t.Errorf("the caret at the start is at %v", x)
	}
	// Unwrapped, the indent takes room.
	if l := s.Layout(Params{Text: "aaa", Style: Style{Size: 10}, Indent: 25}); l.Width != 55 || l.Lines[0].X != 25 {
		t.Errorf("one line %v wide at %v", l.Width, l.Lines[0].X)
	}
	// Centered, the first line goes to the middle of the room beside it.
	if l := s.Layout(Params{Text: "aaa", Style: Style{Size: 10}, Width: 100, Indent: 20, Align: Center}); l.Lines[0].X != 45 {
		t.Errorf("centered at %v, want 45", l.Lines[0].X)
	}
	// Cut short, the first line ends with its ellipsis beside the indent.
	if l := s.Layout(Params{Text: "aaaa bbbb cccc", Style: Style{Size: 10}, Width: 60, Indent: 20, MaxLines: 1}); !l.Truncated || l.Lines[0].Width > 40 || l.Lines[0].X != 20 {
		t.Errorf("cut short, the line %v wide at %v", l.Lines[0].Width, l.Lines[0].X)
	}
	// Only the first paragraph's first line.
	if l := s.Layout(Params{Text: "a\nb", Style: Style{Size: 10}, Indent: 20}); l.Lines[0].X != 20 || l.Lines[1].X != 0 {
		t.Errorf("paragraphs at %v and %v", l.Lines[0].X, l.Lines[1].X)
	}

	// The system's engine, either way.
	sys := Shared()
	long := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 3)
	in := sys.Layout(Params{Text: long, Style: Style{Size: 16}, Width: 200, Indent: 40})
	if first := in.Lines[0]; first.X != 40 || first.Width > 160.5 || in.Lines[1].X != 0 {
		t.Errorf("the first line %v wide at %v, the second at %v", first.Width, first.X, in.Lines[1].X)
	}
	rtl := sys.Layout(Params{Text: strings.Repeat("שלום עולם ", 6), Style: Style{Size: 16}, Width: 200, Indent: 40})
	if first := rtl.Lines[0]; !first.RTL || first.X+first.Width > 160.01 || first.X+first.Width < 159.99 || rtl.Lines[1].X+rtl.Lines[1].Width < 199.99 {
		t.Errorf("right to left, the first line ends at %v, the second at %v", first.X+first.Width, rtl.Lines[1].X+rtl.Lines[1].Width)
	}
}

// TestJustify stretches the lines but the last of each paragraph to the
// width by their spaces.
func TestJustify(t *testing.T) {
	s := breakSystem(0)
	p := Params{Text: "aaa bbb ccc ddd\naa bb", Style: Style{Size: 10}, Width: 100, Justify: true}
	l := s.Layout(p)
	if len(l.Lines) != 3 || lineText(l, 0) != "aaa bbb " {
		t.Fatalf("%d lines, the first %q", len(l.Lines), lineText(l, 0))
	}
	first := l.Lines[0]
	if first.X != 0 || first.Width != 100 {
		t.Errorf("the first line %v wide at %v", first.Width, first.X)
	}
	if b := glyphsOf(first, 4); len(b) != 1 || b[0].X != 70 {
		t.Errorf("the second word at %+v", b)
	}
	if sp := glyphsOf(first, 3); len(sp) != 1 || sp[0].Advance != 40 {
		t.Errorf("the space %+v", sp)
	}
	if x, _, _ := l.Caret(4); x != 70 {
		t.Errorf("the caret before the second word is at %v", x)
	}
	for i := 1; i < 3; i++ {
		if line := l.Lines[i]; line.X != 0 || line.Width != 70 && line.Width != 50 {
			t.Errorf("the last line %d of a paragraph %v wide at %v", i, line.Width, line.X)
		}
	}
	// Centered, the last lines are.
	p.Align = Center
	if l := s.Layout(p); l.Lines[0].X != 0 || l.Lines[1].X != 15 {
		t.Errorf("centered, the lines at %v and %v", l.Lines[0].X, l.Lines[1].X)
	}
	// Spaces kept hang past the width.
	p.Align, p.KeepSpaces = Start, true
	if l := s.Layout(p); l.Lines[0].Width != 110 || glyphsOf(l.Lines[0], 4)[0].X != 70 || l.Width != 110 {
		t.Errorf("keeping spaces, the first line is %v wide, the text %v", l.Lines[0].Width, l.Width)
	}
	// The text is as wide as its lines stretched.
	if l := s.Layout(Params{Text: "aaa bbb cccc", Style: Style{Size: 10}, Width: 100, Justify: true}); l.Width != 100 {
		t.Errorf("justified, the text is %v wide", l.Width)
	}
	// Indented, the first line fills the room beside its indent.
	p.KeepSpaces, p.Indent = false, 10
	if l := s.Layout(p); l.Lines[0].X != 10 || l.Lines[0].Width != 90 {
		t.Errorf("indented, the first line %v wide at %v", l.Lines[0].Width, l.Lines[0].X)
	}
	// A line without spaces stays as it is.
	if l := s.Layout(Params{Text: "aaaaaaaaaaaa", Style: Style{Size: 10}, Width: 75, Justify: true}); l.Lines[0].Width != 70 {
		t.Errorf("a word broken is %v wide", l.Lines[0].Width)
	}

	// Not linked in, Justify does nothing.
	plain := newSystem()
	plain.eng = &breakEngine{}
	if l := plain.Layout(Params{Text: "aaa bbb ccc", Style: Style{Size: 10}, Width: 100, Justify: true}); l.Lines[0].Width != 70 {
		t.Errorf("without UseJustify, the first line is %v wide", l.Lines[0].Width)
	}

	// The system's engine.
	sys := Shared()
	sys.UseJustify()
	long := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 4)
	lj := sys.Layout(Params{Text: long, Style: Style{Size: 16}, Width: 230, Justify: true})
	for i, line := range lj.Lines[:len(lj.Lines)-1] {
		if right := line.X + line.Width; line.X != 0 || right < 229.99 || right > 230.01 {
			t.Errorf("line %d from %v to %v", i, line.X, right)
		}
	}
	prev := float32(-1)
	for i := 0; i <= len(lj.Runes); i++ {
		x, y, _ := lj.Caret(i)
		if li := lj.LineAt(i); i > 0 && lj.Lines[li].Start == i {
			prev = -1
		}
		if x < prev {
			t.Fatalf("caret %d at %v, %v, after %v", i, x, y, prev)
		}
		prev = x
	}
}

// TestJustifyCJK widens the room after Han characters and kana, but the
// last, in lines without spaces.
func TestJustifyCJK(t *testing.T) {
	runes := []rune("日本語です")
	line := Line{Start: 0, End: len(runes), Width: 50}
	for i := range runes {
		line.Glyphs = append(line.Glyphs, Glyph{X: float32(i) * 10, Advance: 10, Cluster: i, Runes: 1})
	}
	left, ok := justifyLine(&line, runes, 90)
	if !ok || left != 0 || line.Width != 90 {
		t.Fatalf("justified %v, %v wide, from %v", ok, line.Width, left)
	}
	for i, g := range line.Glyphs {
		if g.X != float32(i)*20 {
			t.Errorf("glyph %d at %v, want %v", i, g.X, i*20)
		}
	}
	if line.Glyphs[4].Advance != 10 {
		t.Errorf("the last glyph widened to %v", line.Glyphs[4].Advance)
	}
}

// TestBaselineShift raises and lowers spans, which the line makes room
// for, and their decorations with them.
func TestBaselineShift(t *testing.T) {
	s := Shared()
	plain := s.Layout(Params{Text: "x2 H2O", Style: Style{Size: 16}})
	spans := []Span{{End: 1}, {End: 2, Size: 10, BaselineShift: 7}, {End: 4}, {End: 5, Size: 10, BaselineShift: -4}}
	l := s.Layout(Params{Text: "x2 H2O", Style: Style{Size: 16}, Spans: EncodeSpans(spans)})
	line := l.Lines[0]
	sup, sub, x := glyphsOf(line, 1)[0], glyphsOf(line, 4)[0], glyphsOf(line, 0)[0]
	if x.Y != line.Baseline || sup.Y != line.Baseline-7 || sub.Y != line.Baseline+4 {
		t.Errorf("baselines %v, %v and %v of the line's %v", x.Y, sup.Y, sub.Y, line.Baseline)
	}
	if line.Ascent < sup.Font.Ascent+7 || line.Descent < sub.Font.Descent+4 {
		t.Errorf("ascent %v and descent %v for a superscript of ascent %v and a subscript of descent %v", line.Ascent, line.Descent, sup.Font.Ascent, sub.Font.Descent)
	}
	if l.Height <= plain.Height {
		t.Errorf("shifted, the text is %v high, %v without", l.Height, plain.Height)
	}
	// Decorations follow the span.
	under := func(l *Layout, i int) float32 {
		st := s.Decorate(l, 0, i, i+1, 0, 0, 1, Underline, LineStyle{})
		if len(st) == 0 {
			t.Fatalf("no underline of glyph %d", i)
		}
		return st[0].Top
	}
	flat := s.Layout(Params{Text: "x2 H2O", Style: Style{Size: 16}, Spans: EncodeSpans([]Span{{End: 1}, {End: 2, Size: 10}, {End: 4}, {End: 5, Size: 10}})})
	k := slices.IndexFunc(line.Glyphs, func(g Glyph) bool { return g.Cluster == 1 })
	kf := slices.IndexFunc(flat.Lines[0].Glyphs, func(g Glyph) bool { return g.Cluster == 1 })
	if d := (under(flat, kf) - flat.Lines[0].Baseline) - (under(l, k) - line.Baseline); d < 6 || d > 8 {
		t.Errorf("the superscript's underline is %v above where it would be", d)
	}
	// Objects shift too.
	obj := s.Layout(Params{Text: "a\uFFFC", Style: Style{Size: 16}, Spans: EncodeSpans([]Span{{End: 1}, {End: 2, ObjectWidth: 5, ObjectHeight: 30, BaselineShift: 10}})})
	if obj.Lines[0].Ascent < 40 {
		t.Errorf("a raised object leaves an ascent of %v", obj.Lines[0].Ascent)
	}
}

// TestBalance wraps a paragraph in as many lines as it takes, at the
// narrowest width that keeps them as many, aligned within the width: no
// word alone on the last line.
func TestBalance(t *testing.T) {
	s := breakSystem(0)
	p := Params{Text: "aaa bbb ccc ddd e", Style: Style{Size: 10}, Width: 160}
	if l := s.Layout(p); len(l.Lines) != 2 || lineText(l, 1) != "e" {
		t.Fatalf("unbalanced, %d lines, the last %q", len(l.Lines), lineText(l, len(l.Lines)-1))
	}
	p.Balance = true
	l := s.Layout(p)
	if len(l.Lines) != 2 || lineText(l, 0) != "aaa bbb " || lineText(l, 1) != "ccc ddd e" {
		t.Fatalf("balanced, %d lines: %q, %q", len(l.Lines), lineText(l, 0), lineText(l, len(l.Lines)-1))
	}
	if l.Lines[0].Width != 70 || l.Lines[1].Width != 90 || l.Width != 90 {
		t.Errorf("balanced, the lines are %v and %v wide, the text %v", l.Lines[0].Width, l.Lines[1].Width, l.Width)
	}
	// Laid out once, and kept: a frame laying it out again finds it.
	made := s.LayoutsMade()
	if s.Layout(p) != l || s.LayoutsMade() != made {
		t.Error("the balanced layout was made again")
	}
	// Aligned within the width, each paragraph balanced on its own.
	p.Text, p.Align = "aaa bbb ccc ddd e\naa bb", Center
	l = s.Layout(p)
	if len(l.Lines) != 3 || l.Lines[0].X != 45 || l.Lines[1].X != 35 || lineText(l, 2) != "aa bb" || l.Lines[2].X != 55 {
		t.Errorf("centered, %d lines at %v, %v, %v", len(l.Lines), l.Lines[0].X, l.Lines[1].X, l.Lines[len(l.Lines)-1].X)
	}
	// A line it fits on, it keeps.
	if l := s.Layout(Params{Text: "aaa bbb", Style: Style{Size: 10}, Width: 160, Balance: true}); len(l.Lines) != 1 || l.Width != 70 {
		t.Errorf("one line: %d lines, %v wide", len(l.Lines), l.Width)
	}
	// Three lines of about the same length, rather than two full ones
	// and a short one.
	p = Params{Text: "aaa bbb ccc ddd eee fff g", Style: Style{Size: 10}, Width: 120, Balance: true}
	l = s.Layout(p)
	if len(l.Lines) != 3 || lineText(l, 0) != "aaa bbb " || lineText(l, 1) != "ccc ddd " || lineText(l, 2) != "eee fff g" {
		t.Errorf("three lines: %q", []string{lineText(l, 0), lineText(l, 1), lineText(l, len(l.Lines)-1)})
	}
	// MaxLines cutting the paragraph, it wraps as without Balance.
	p = Params{Text: "aaa bbb ccc ddd e", Style: Style{Size: 10}, Width: 160, MaxLines: 1, Balance: true}
	if l := s.Layout(p); len(l.Lines) != 1 || !l.Truncated || !strings.HasPrefix(lineText(l, 0), "aaa bbb ccc") {
		t.Errorf("cut to a line: %d lines, %q", len(l.Lines), lineText(l, 0))
	}
	// Without a width, nothing wraps.
	if l := s.Layout(Params{Text: "aaa bbb ccc ddd e", Style: Style{Size: 10}, Balance: true}); len(l.Lines) != 1 {
		t.Errorf("without a width, %d lines", len(l.Lines))
	}
}
