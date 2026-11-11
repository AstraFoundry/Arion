package ui

import "testing"

// TestShapedLineHeight spaces the lines of shaped text as Font.LineHeight
// says, and the caret follows them, as Element.LineHeight does for Text
// (issue #807).
func TestShapedLineHeight(t *testing.T) {
	plain := Font{Size: 20}
	tall := Font{Size: 20, LineHeight: 2}
	_, h := ShapeRichText([]Span{{Text: "a\nb"}}, tall, 0).Size()
	if h < 79 || h > 81 {
		t.Errorf("two lines at 2x of 20 DIPs are %v tall, want 80", h)
	}
	if _, ph := ShapeText("a\nb", plain, 0).Size(); ph >= h {
		t.Errorf("the font's own lines (%v) are not shorter than 2x (%v)", ph, h)
	}
	l := ShapeText("a\nb", tall, 0)
	if y := l.Caret(2).Y - l.Caret(0).Y; y < 39 || y > 41 {
		t.Errorf("the second line's caret is %v below the first, want 40", y)
	}
	if plain.Metrics() != tall.Metrics() {
		t.Errorf("Metrics follow LineHeight: %+v, want the font's own %+v", tall.Metrics(), plain.Metrics())
	}
}

// TestTextLayoutLines gives the lines of a layout in UTF-16 offsets, as
// a paginating reader asks where a page ends.
func TestTextLayoutLines(t *testing.T) {
	l := ShapeText("😀 one two three four five six seven\nend", Font{Size: 16}, 120)
	lines := l.Lines()
	if len(lines) < 3 {
		t.Fatalf("%d lines", len(lines))
	}
	if lines[0].Start != 0 || lines[len(lines)-1].End != 40 || lines[len(lines)-1].Start != 37 {
		t.Errorf("the lines from %d, the last %d to %d", lines[0].Start, lines[len(lines)-1].Start, lines[len(lines)-1].End)
	}
	_, h := l.Size()
	for i, line := range lines {
		if i > 0 && lines[i-1].End != line.Start && lines[i-1].End+1 != line.Start {
			t.Errorf("line %d starts at %d, after one ending at %d", i, line.Start, lines[i-1].End)
		}
		if line.Baseline <= line.Rect.Y || line.Baseline >= line.Rect.Y+line.Rect.H || line.Ascent <= 0 || line.Descent <= 0 {
			t.Errorf("line %d: %+v", i, line)
		}
		// The caret at the line's start is on it.
		if c := l.Caret(line.Start); c.Y != line.Rect.Y {
			t.Errorf("line %d at %v, its first caret at %v", i, line.Rect.Y, c.Y)
		}
	}
	if last := lines[len(lines)-1]; last.Rect.Y+last.Rect.H != h {
		t.Errorf("the last line ends at %v of %v", last.Rect.Y+last.Rect.H, h)
	}
}

// TestTextShapeParagraph lays out a paragraph as a book does: indented,
// justified, its words whole.
func TestTextShapeParagraph(t *testing.T) {
	spans := []Span{{Text: "Call me Ishmael. Some years ago, never mind how long precisely, having little or no money in my purse."}}
	l := ShapeRichTextWith(spans, Font{Size: 16}, TextShape{Width: 240, Justify: true, Indent: 32})
	lines := l.Lines()
	if len(lines) < 3 || lines[0].Rect.X != 32 {
		t.Fatalf("%d lines, the first at %v", len(lines), lines[0].Rect.X)
	}
	for i, line := range lines[:len(lines)-1] {
		// Spaces ending a line hang past the width, as the layout keeps them.
		if right := line.Rect.X + line.Rect.W; right < 239.99 {
			t.Errorf("line %d ends at %v", i, right)
		}
	}
	if last := lines[len(lines)-1]; last.Rect.X != 0 || last.Rect.W >= 239 {
		t.Errorf("the last line %v wide at %v", last.Rect.W, last.Rect.X)
	}
	long := []Span{{Text: "Supercalifragilistic word"}}
	if l := ShapeRichTextWith(long, Font{Size: 16}, TextShape{Width: 40, NoBreakWords: true}); len(l.Lines()) != 2 {
		t.Errorf("whole words in %d lines", len(l.Lines()))
	}
}

// TestBaselineShiftObjects raises an object with its span.
func TestBaselineShiftObjects(t *testing.T) {
	spans := []Span{{Text: "a"}, {Text: "\uFFFC", ObjectWidth: 8, ObjectHeight: 8, BaselineShift: 6}}
	l := ShapeRichText(spans, Font{Size: 16}, 0)
	objs, line := l.Objects(), l.Lines()[0]
	if len(objs) != 1 || objs[0].Rect.Y != line.Baseline-14 || objs[0].Rect.H != 8 {
		t.Errorf("objects %+v on a baseline at %v", objs, line.Baseline)
	}
}
