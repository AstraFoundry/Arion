package ui_test

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestListRowWidth scrolls a List whose rows are wider than it sideways,
// a header going with them through ScrollWith, and down, the header
// staying.
func TestListRowWidth(t *testing.T) {
	var s ui.ListState
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			head := ui.Row(c).Height(20).Shrink(0).Clip().Children(func() {
				ui.Text(c, "Name").Width(500).Shrink(0)
				ui.Text(c, "Size").Width(100).Shrink(0)
			})
			list := ui.List(c, &s, 50).Grow(1).RowWidth(600)
			list.Rows(func(r ui.ListRow) {
				ui.Row(r.Context).Height(20).Children(func() {
					ui.Text(r.Context, "file").Width(500).Shrink(0)
					ui.Text(r.Context, strconv.Itoa(r.Index)+" KB").Width(100).Shrink(0)
				})
			})
			head.ScrollWith(list)
		})
	}, 300, 200)
	size, _ := tt.Find("Size")
	cell, _ := tt.Find("0 KB")
	if size.X != 500 || cell.X != 500 {
		t.Fatalf("before scrolling: the header's Size at %v, the cell under it at %v, want 500", size.X, cell.X)
	}
	tt.Scroll(150, 100, 300, 0)
	size, _ = tt.Find("Size")
	cell, _ = tt.Find("0 KB")
	if size.X != 200 || cell.X != 200 {
		t.Errorf("scrolled 300 sideways: the header's Size at %v, the cell under it at %v, want 200", size.X, cell.X)
	}
	tt.Scroll(150, 100, 0, 200)
	if after, _ := tt.Find("Size"); after.Y != size.Y || after.X != size.X {
		t.Errorf("scrolled down: the header moved from %v to %v", size, after)
	}
	if r, ok := tt.Find("10 KB"); !ok || r.Y != cell.Y || r.X != 200 {
		t.Errorf("scrolled 200 down: the eleventh row at %v, want where the first was, %v", r, cell)
	}
}

// TestListRowWidthFits grows the rows of a List to its width, where they
// ask for less, and keeps it from scrolling sideways.
func TestListRowWidthFits(t *testing.T) {
	var s ui.ListState
	var row ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		list := ui.List(c, &s, 5).Fill().RowWidth(100)
		list.Rows(func(r ui.ListRow) {
			e := ui.Row(r.Context).Height(20).Children(func() {
				ui.Text(r.Context, "row "+strconv.Itoa(r.Index))
			})
			if r.Index == 0 {
				row = e.Bounds()
			}
		})
	}, 300, 200)
	tt.Frame()
	if row.W != 300 {
		t.Errorf("a row asking for 100 in a list 300 wide is %v wide", row.W)
	}
	tt.Scroll(150, 100, 100, 0)
	if r, _ := tt.Find("row 0"); r.X != 0 {
		t.Errorf("a list whose rows fit scrolled sideways to %v", r.X)
	}
}
