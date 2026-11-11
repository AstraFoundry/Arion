package text

import "testing"

// BenchmarkVisualStopsLongLine computes the visual caret stops of one very
// long line, as opening a minified or generated file does. Fixed advances
// keep the fixture independent of installed fonts; shaping and setup are not
// timed, and the stops cache is dropped every iteration, as an edit does.
func BenchmarkVisualStopsLongLine(b *testing.B) {
	const runes = 200_000
	l := &Layout{Runes: make([]rune, runes), Lines: []Line{{Start: 0, End: runes, Width: runes * 8, Height: 20}}}
	for i := range l.Runes {
		l.Runes[i] = 'a'
	}
	line := &l.Lines[0]
	line.Glyphs = make([]Glyph, runes)
	for i := range line.Glyphs {
		line.Glyphs[i] = Glyph{X: float32(i) * 8, Advance: 8, Cluster: i, Runes: 1}
	}
	b.ReportAllocs()
	for b.Loop() {
		line.stops = nil
		stops := l.visualStops(0)
		if len(stops) != 2*runes {
			b.Fatalf("got %d stops, want %d", len(stops), 2*runes)
		}
	}
}
