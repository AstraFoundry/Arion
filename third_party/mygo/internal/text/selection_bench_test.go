package text

import "testing"

// BenchmarkSelectionVisual finds the hit boxes of four six-rune inline spans
// per wrapped line, as committing a paragraph with many links does. SmallInline
// uses four lines, LongInline 128, and VeryLongInline 512, each with 64 glyphs
// per line. WholeRange selects all 512 lines as a control. Fixed advances keep
// the fixtures independent of installed fonts; shaping and setup are not timed.
// One inline operation processes every span in the paragraph.
func BenchmarkSelectionVisual(b *testing.B) {
	for _, tc := range []struct {
		name  string
		lines int
		whole bool
	}{
		{"SmallInline", 4, false},
		{"LongInline", 128, false},
		{"VeryLongInline", 512, false},
		{"WholeRange", 512, true},
	} {
		b.Run(tc.name, func(b *testing.B) {
			layout := selectionBenchmarkLayout(tc.lines)
			var ranges []Range
			if tc.whole {
				ranges = []Range{{0, len(layout.Runes)}}
			} else {
				for _, line := range layout.Lines {
					for offset := 4; offset < 64; offset += 16 {
						ranges = append(ranges, Range{line.Start + offset, line.Start + offset + 6})
					}
				}
			}
			want := len(ranges)
			if tc.whole {
				want = tc.lines
			}
			b.ReportAllocs()
			for b.Loop() {
				count := 0
				for _, r := range ranges {
					count += len(layout.SelectionVisual(r.Start, r.End, false))
				}
				if count != want {
					b.Fatalf("got %d rectangles, want %d", count, want)
				}
			}
		})
	}
}

func selectionBenchmarkLayout(lines int) *Layout {
	const glyphsPerLine = 64
	l := &Layout{
		Runes:  make([]rune, lines*glyphsPerLine),
		Lines:  make([]Line, lines),
		Width:  glyphsPerLine * 8,
		Height: float32(lines * 20),
	}
	for i := range l.Runes {
		l.Runes[i] = 'a'
	}
	for i := range l.Lines {
		start := i * glyphsPerLine
		line := &l.Lines[i]
		*line = Line{
			Start: start, End: start + glyphsPerLine,
			Y: float32(i * 20), Width: l.Width, Height: 20,
			Glyphs: make([]Glyph, glyphsPerLine),
		}
		for j := range line.Glyphs {
			line.Glyphs[j] = Glyph{X: float32(j * 8), Advance: 8, Cluster: start + j, Runes: 1}
		}
	}
	return l
}
