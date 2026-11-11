package ui

import "testing"

// BenchmarkFrameAllChosenMillionRows measures a frame of a million-row
// table choosing blocks of cells with every row chosen, as scrolling builds
// them.
func BenchmarkFrameAllChosenMillionRows(b *testing.B) {
	const rows = 1_000_000
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 60}, {Title: "A"}, {Title: "B"}}
	sel := &Selection[int]{}
	s := &ListState{CellSelect: true, Selection: sel}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, rows, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r0c1")
	tt.Key(Cmd, KeyA)
	tt.Frame()
	for b.Loop() {
		tt.Scroll(200, 150, 0, 120)
	}
	if sel.Len() != rows {
		b.Fatalf("chose %d rows", sel.Len())
	}
}
