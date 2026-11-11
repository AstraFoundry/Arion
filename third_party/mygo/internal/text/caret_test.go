package text

import (
	"slices"
	"strings"
	"testing"
)

// Fixed advances keep geometry regressions independent of installed fonts.
func mixedLayout() *Layout {
	runes := []rune("abc אבג def")
	l := &Layout{Runes: runes, Lines: []Line{{Start: 0, End: len(runes), Width: 110, Height: 20}}}
	for i := range runes {
		x, rtl := float32(i*10), i >= 4 && i < 7
		if rtl {
			x = float32((10 - i) * 10)
		}
		l.Lines[0].Glyphs = append(l.Lines[0].Glyphs, Glyph{X: x, Advance: 10, Cluster: i, Runes: 1, RTL: rtl})
	}
	return l
}

func TestVisualCaretAffinityAndNavigation(t *testing.T) {
	l := mixedLayout()
	a, _, _ := l.CaretAt(CaretPosition{4, Upstream})
	b, _, _ := l.CaretAt(CaretPosition{4, Downstream})
	if a != 40 || b != 70 {
		t.Fatalf("bidi edges %v %v", a, b)
	}
	p := CaretPosition{3, Downstream}
	for _, want := range []CaretPosition{{4, Upstream}, {6, Downstream}, {5, Downstream}, {4, Downstream}, {8, Upstream}} {
		p = l.MoveCaret(p, 1)
		if p != want {
			t.Fatalf("right: got %+v want %+v", p, want)
		}
	}
	for _, want := range []CaretPosition{{7, Downstream}, {5, Upstream}, {6, Upstream}, {7, Upstream}, {3, Downstream}} {
		p = l.MoveCaret(p, -1)
		if p != want {
			t.Fatalf("left: got %+v want %+v", p, want)
		}
	}
}

func TestVisualSelectionKeepsBidiGaps(t *testing.T) {
	l := mixedLayout()
	ranges := l.SelectionRanges(CaretPosition{6, Downstream}, CaretPosition{8, Upstream})
	if !slices.Equal(ranges, []Range{{4, 6}, {7, 8}}) {
		t.Fatalf("selection %v", ranges)
	}
	var rects []Rect
	for _, r := range ranges {
		rects = append(rects, l.SelectionVisual(r.Start, r.End, false)...)
	}
	if len(rects) != 2 || rects[0].X != 50 || rects[1].X != 70 {
		t.Fatalf("bidi rectangles %v", rects)
	}
}

func TestVisualCaretFractionalRunEdges(t *testing.T) {
	l := mixedLayout()
	l.Lines[0].Glyphs[4].Advance += 0.00001
	p := l.MoveCaret(CaretPosition{Index: 5}, 1)
	if p != (CaretPosition{Index: 4, Affinity: Downstream}) {
		t.Fatalf("rounding selected the wrong run edge: %+v", p)
	}
}

func TestVisualCaretWrapAndGraphemes(t *testing.T) {
	l := &Layout{Runes: []rune("é😀x"), Lines: []Line{
		{Start: 0, End: 3, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 0, Runes: 2}, {X: 10, Advance: 10, Cluster: 2, Runes: 1}}},
		{Start: 3, End: 4, Y: 20, Width: 10, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 3, Runes: 1}}},
	}}
	p := l.MoveCaret(CaretPosition{}, 1)
	if p.Index != 2 {
		t.Fatalf("split grapheme %+v", p)
	}
	_, up, _ := l.CaretAt(CaretPosition{3, Upstream})
	_, down, _ := l.CaretAt(CaretPosition{3, Downstream})
	if up != 0 || down != 20 {
		t.Fatalf("wrap affinity %v %v", up, down)
	}
	p = l.PositionAt(20, 10)
	if p != (CaretPosition{3, Upstream}) {
		t.Fatalf("wrap hit %+v", p)
	}
	p = l.MoveCaret(p, 1)
	if p.Index != 3 || p.Affinity != Downstream {
		t.Fatalf("wrap movement %+v", p)
	}
}

// A trimmed wrap space shares the last visible glyph's edge. Choosing the
// later offset on a tie would copy whitespace the pointer never crossed.
func TestVisualCaretHitBeforeTrimmedWrapSpace(t *testing.T) {
	l := &Layout{Runes: []rune("ab cd"), Lines: []Line{
		{Start: 0, End: 3, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 0, Runes: 1}, {X: 10, Advance: 10, Cluster: 1, Runes: 1}}},
		{Start: 3, End: 5, Y: 20, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 3, Runes: 1}, {X: 10, Advance: 10, Cluster: 4, Runes: 1}}},
	}}
	x, y, h := l.Caret(2)
	if p := l.PositionAt(x, y+h/2); p != (CaretPosition{Index: 2, Affinity: Upstream}) {
		t.Fatalf("hit before trimmed space selected %+v", p)
	}
}

// Compare the line filter with the original glyph-by-glyph scan. Fixed geometry
// covers every boundary without depending on platform fonts or shaping engines.
func TestSelectionVisualLineSkipMatchesFullScan(t *testing.T) {
	wrapped := &Layout{Runes: []rune("aaaa bbbb\ncccc dddd")}
	for _, span := range [][2]int{{0, 5}, {5, 9}, {10, 19}} {
		line := Line{Start: span[0], End: span[1], Width: float32((span[1] - span[0]) * 10), Height: 20, Y: float32(len(wrapped.Lines) * 20)}
		for i := span[0]; i < span[1]; i++ {
			line.Glyphs = append(line.Glyphs, Glyph{X: float32((i - span[0]) * 10), Advance: 10, Cluster: i, Runes: 1})
		}
		wrapped.Lines = append(wrapped.Lines, line)
	}
	// The initial and final empty lines and the consecutive newlines all have
	// selectable newline boxes, even though they contain no glyphs.
	emptyLines := &Layout{Runes: []rune("\nx\n\n"), Lines: []Line{
		{Start: 0, End: 0, Height: 20},
		{Start: 1, End: 2, Y: 20, Width: 10, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 1, Runes: 1}}},
		{Start: 3, End: 3, Y: 40, Height: 20, RTL: true},
		{Start: 4, End: 4, Y: 60, Height: 20},
	}}
	// A bidi line between two wrapped lines also has a cluster with multiple
	// glyphs and runes, plus an ellipsis (zero runes) on the final line.
	bidi := mixedLayout()
	bidi.Runes = append([]rune("fi"), append(bidi.Runes, 'x')...)
	bidi.Lines[0].Start, bidi.Lines[0].End = 2, len(bidi.Runes)-1
	bidi.Lines[0].Y = 20
	for i := range bidi.Lines[0].Glyphs {
		bidi.Lines[0].Glyphs[i].Cluster += 2
	}
	bidi.Lines = append([]Line{{Start: 0, End: 2, Width: 20, Height: 20, Glyphs: []Glyph{
		{Advance: 12, Cluster: 0, Runes: 2}, {X: 10, Advance: 10, Cluster: 0, Runes: 2},
	}}}, bidi.Lines...)
	bidi.Lines = append(bidi.Lines, Line{Start: len(bidi.Runes) - 1, End: len(bidi.Runes), Y: 40, Width: 20, Height: 20, Glyphs: []Glyph{
		{Advance: 10, Cluster: len(bidi.Runes) - 1, Runes: 1}, {X: 10, Advance: 10, Cluster: len(bidi.Runes), Runes: 0},
	}})
	for _, tc := range []struct {
		name   string
		layout *Layout
	}{
		{"wrapped", wrapped}, {"empty_lines", emptyLines}, {"bidi_ligature", bidi},
		{"empty", &Layout{Lines: []Line{{Height: 20}}}},
		{"no_lines", &Layout{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for start := -1; start <= len(tc.layout.Runes)+1; start++ {
				for end := -1; end <= len(tc.layout.Runes)+1; end++ {
					for _, on := range []bool{false, true} {
						got := tc.layout.SelectionVisual(start, end, on)
						want := selectionVisualFullScan(tc.layout, start, end, on)
						if !slices.Equal(got, want) {
							t.Fatalf("range %d..%d on=%v: got %v, full scan %v", start, end, on, got, want)
						}
					}
				}
			}
		})
	}
}

// selectionVisualFullScan preserves the pre-filter implementation as an oracle:
// changes to which lines are visited must preserve its rectangles exactly.
func selectionVisualFullScan(l *Layout, start, end int, on bool) []Rect {
	if start > end {
		start, end = end, start
	}
	if on {
		end = len(l.Runes) + 1
	}
	var out []Rect
	for li := range l.Lines {
		line := &l.Lines[li]
		var pieces []Rect
		for i := 0; i < len(line.Glyphs); {
			first := line.Glyphs[i]
			x0, x1 := first.X, first.X+first.Advance
			i++
			for i < len(line.Glyphs) && line.Glyphs[i].Cluster == first.Cluster && line.Glyphs[i].Runes == first.Runes {
				x0, x1 = min(x0, line.Glyphs[i].X), max(x1, line.Glyphs[i].X+line.Glyphs[i].Advance)
				i++
			}
			a, z := max(start, first.Cluster), min(end, first.Cluster+first.Runes)
			if a >= z || first.Runes <= 0 {
				continue
			}
			width := x1 - x0
			left, right := float32(a-first.Cluster)/float32(first.Runes), float32(z-first.Cluster)/float32(first.Runes)
			if first.RTL {
				left, right = 1-right, 1-left
			}
			pieces = append(pieces, Rect{x0 + left*width, line.Y, (right - left) * width, line.Height})
		}
		if start <= line.End && end > line.End && (li < len(l.Lines)-1 && l.Lines[li+1].Start > line.End || li == len(l.Lines)-1 && on) {
			x, width := line.X+line.Width, l.Params.Style.FontSize()/3
			if line.RTL {
				x = line.X - width
			}
			pieces = append(pieces, Rect{x, line.Y, width, line.Height})
		}
		slices.SortFunc(pieces, func(a, b Rect) int {
			if a.X < b.X {
				return -1
			}
			if a.X > b.X {
				return 1
			}
			return 0
		})
		for _, r := range pieces {
			if r.W <= 0 {
				continue
			}
			if n := len(out); n > 0 && out[n-1].Y == r.Y && out[n-1].X+out[n-1].W >= r.X-0.01 {
				out[n-1].W = max(out[n-1].X+out[n-1].W, r.X+r.W) - out[n-1].X
			} else {
				out = append(out, r)
			}
		}
	}
	return out
}

// TestVisualStopsMatchesFullScan compares visualStops with the original
// pairwise scan as an oracle, on fixtures for every stop source: glyph
// edges, fractionally split clusters, bidi runs, offsets only the boundary
// sweep visits (trimmed wrap spaces, empty lines) and glyphs outside the
// line's range. Fixed geometry covers every boundary without depending on
// platform fonts or shaping engines.
func TestVisualStopsMatchesFullScan(t *testing.T) {
	var long strings.Builder
	for i := 0; i < 4096; i++ {
		long.WriteRune('a')
		if i%7 == 3 {
			long.WriteRune('\u0301')
		}
	}
	// Duplicate clusters append two stops at one offset, a ligature shares
	// one cluster between glyphs, and the ellipsis has no runes at all.
	ligature := mixedLayout()
	ligature.Runes = append([]rune("fi"), append(ligature.Runes, 'x')...)
	ligature.Lines[0].Start, ligature.Lines[0].End = 2, len(ligature.Runes)-1
	ligature.Lines[0].Y = 20
	for i := range ligature.Lines[0].Glyphs {
		ligature.Lines[0].Glyphs[i].Cluster += 2
	}
	ligature.Lines = append([]Line{{Start: 0, End: 2, Width: 20, Height: 20, Glyphs: []Glyph{
		{Advance: 12, Cluster: 0, Runes: 2}, {X: 10, Advance: 10, Cluster: 0, Runes: 2},
	}}}, ligature.Lines...)
	ligature.Lines = append(ligature.Lines, Line{Start: len(ligature.Runes) - 1, End: len(ligature.Runes), Y: 40, Width: 20, Height: 20, Glyphs: []Glyph{
		{Advance: 10, Cluster: len(ligature.Runes) - 1, Runes: 1}, {X: 10, Advance: 10, Cluster: len(ligature.Runes), Runes: 0},
	}})
	for _, tc := range []struct {
		name   string
		layout *Layout
	}{
		{"ascii", graphemeLayout("hello world")},
		{"crlf", graphemeLayout("a\r\nb")},
		{"emoji_zwj", graphemeLayout("\U0001F468\u200D\U0001F469\u200D\U0001F466x")},
		{"flags", graphemeLayout("🇯🇵🇯🇵")},
		{"cjk", graphemeLayout("你好世界")},
		{"combining", graphemeLayout("a\u0301e\u0327x")},
		{"trailing_spaces", graphemeLayout("ab  ")},
		{"rtl_bidi", mixedLayout()},
		{"long_mixed", graphemeLayout(long.String())},
		{"wrapped_trimmed_space", &Layout{Runes: []rune("ab cd"), Lines: []Line{
			{Start: 0, End: 3, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 0, Runes: 1}, {X: 10, Advance: 10, Cluster: 1, Runes: 1}}},
			{Start: 3, End: 5, Y: 20, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 3, Runes: 1}, {X: 10, Advance: 10, Cluster: 4, Runes: 1}}},
		}}},
		{"empty_lines", &Layout{Runes: []rune("\nx\n\n"), Lines: []Line{
			{Start: 0, End: 0, Height: 20},
			{Start: 1, End: 2, Y: 20, Width: 10, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 1, Runes: 1}}},
			{Start: 3, End: 3, Y: 40, Height: 20, RTL: true},
			{Start: 4, End: 4, Y: 60, Height: 20},
		}}},
		{"ligature", ligature},
		{"glyphs_outside_line", &Layout{Runes: []rune("abcd"), Lines: []Line{
			{Start: 0, End: 2, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 3, Runes: 1}}},
			{Start: 2, End: 4, Y: 20, Width: 20, Height: 20, Glyphs: []Glyph{{Advance: 10, Cluster: 0, Runes: 1}}},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for li := range tc.layout.Lines {
				got := tc.layout.visualStops(li)
				want := visualStopsFullScan(tc.layout, li)
				if !slices.Equal(got, want) {
					t.Fatalf("line %d: got %v, full scan %v", li, got, want)
				}
			}
		})
	}
}

// graphemeLayout lays out s as one line of fixed-advance glyphs, one per
// grapheme at the offsets the segmenter reports, so multi-rune graphemes
// split their edge positions fractionally.
func graphemeLayout(s string) *Layout {
	runes := []rune(s)
	var b Boundaries
	b.Reset(runes)
	offsets := b.GraphemeOffsets()
	l := &Layout{Runes: runes, Lines: []Line{{Start: 0, End: len(runes), Width: float32(len(runes)) * 10, Height: 20}}}
	for k := 0; k+1 < len(offsets); k++ {
		l.Lines[0].Glyphs = append(l.Lines[0].Glyphs, Glyph{X: float32(offsets[k]) * 10, Advance: 10, Cluster: offsets[k], Runes: offsets[k+1] - offsets[k]})
	}
	return l
}

// visualStopsFullScan preserves the pre-mask implementation as an oracle:
// it rebuilds a line's stops with the original pairwise ContainsFunc scan,
// without touching the cache.
func visualStopsFullScan(l *Layout, li int) []caretStop {
	line := &l.Lines[li]
	if l.boundaries == nil {
		var b Boundaries
		b.Reset(l.Runes)
		l.boundaries = b.GraphemeOffsets()
	}
	var stops []caretStop
	line.upstream = slices.Clone(line.lineCarets())
	for i := 0; i < len(line.Glyphs); {
		g := line.Glyphs[i]
		x0, x1 := g.X, g.X+g.Advance
		i++
		for i < len(line.Glyphs) && line.Glyphs[i].Cluster == g.Cluster && line.Glyphs[i].Runes == g.Runes {
			x0 = min(x0, line.Glyphs[i].X)
			x1 = max(x1, line.Glyphs[i].X+line.Glyphs[i].Advance)
			i++
		}
		if g.Runes <= 0 {
			continue
		}
		for k := 0; k <= g.Runes; k++ {
			at := g.Cluster + k
			if _, ok := slices.BinarySearch(l.boundaries, at); !ok {
				continue
			}
			x := x0 + (x1-x0)*float32(k)/float32(g.Runes)
			if g.RTL {
				x = x1 - (x1-x0)*float32(k)/float32(g.Runes)
			}
			affinity, arrival := Downstream, 0
			if k == 0 {
				arrival = -1
			}
			if k == g.Runes {
				affinity, arrival = Upstream, 1
			}
			if g.RTL {
				arrival = -arrival
			}
			if affinity == Upstream && at >= line.Start && at <= line.End {
				line.upstream[at-line.Start] = x
			}
			stops = append(stops, caretStop{CaretPosition{at, affinity}, x, arrival})
		}
	}
	// Keep trimmed spaces and empty lines navigable too.
	carets := line.lineCarets()
	first, _ := slices.BinarySearch(l.boundaries, line.Start)
	for k := first; k < len(l.boundaries) && l.boundaries[k] <= line.End; k++ {
		at := l.boundaries[k]
		if !slices.ContainsFunc(stops, func(s caretStop) bool { return s.position.Index == at }) {
			stops = append(stops, caretStop{CaretPosition{at, Downstream}, carets[at-line.Start], 0})
		}
	}
	if len(stops) == 0 {
		stops = append(stops, caretStop{CaretPosition{line.Start, Downstream}, line.X, 0})
	}
	slices.SortStableFunc(stops, func(a, b caretStop) int {
		if a.x < b.x {
			return -1
		}
		if a.x > b.x {
			return 1
		}
		return 0
	})
	return stops
}
