package ui

import (
	"math"
	"testing"
)

// The boxes the baseline tests below want without text are Chrome's.

func wantBox(t *testing.T, name string, e *node, want Rect) {
	t.Helper()
	got := Rect{e.x, e.y, e.w, e.h}
	if p := e.parent; p != nil {
		got.X, got.Y = got.X-p.x, got.Y-p.y
	}
	for i, v := range [4]float32{got.X - want.X, got.Y - want.Y, got.W - want.W, got.H - want.H} {
		if math.Abs(float64(v)) > 0.05 {
			t.Errorf("%s is at %v, want %v (%d)", name, got, want, i)
			return
		}
	}
}

// Text of different sizes in a row sits on one baseline, and the row is as
// high as the highest ascent and the deepest descent.
func TestBaselineText(t *testing.T) {
	var row, price, unit *node
	coreNewTester(func(c *context) {
		row = coreRow(c).AlignItems(Baseline).Gap(4).Children(func() {
			price = coreText(c, "$42").FontSize(40)
			unit = coreText(c, "/month").FontSize(13).Padding(3, 0, 0, 0)
		})
	}, 400, 300)
	pl, ul := price.tl.Lines[0], unit.tl.Lines[0]
	pb := price.y + price.contentY() + pl.Baseline
	ub := unit.y + unit.contentY() + ul.Baseline
	if math.Abs(float64(pb-ub)) > 0.01 {
		t.Errorf("the baselines are at %v and %v", pb, ub)
	}
	if price.y != row.y {
		t.Errorf("the price is %v below the row's top, want 0", price.y-row.y)
	}
	if want := max(price.h, unit.y-row.y+unit.h); math.Abs(float64(row.h-want)) > 0.01 {
		t.Errorf("the row is %v high, want %v", row.h, want)
	}
}

// AlignSelf(Baseline) lines up only the elements that ask, and a box
// without text puts its bottom on their baseline.
func TestBaselineSelf(t *testing.T) {
	var big, small, dot, top *node
	coreNewTester(func(c *context) {
		coreRow(c).AlignItems(Start).Children(func() {
			big = coreText(c, "Title").FontSize(28).AlignSelf(Baseline)
			small = coreText(c, "note").FontSize(11).AlignSelf(Baseline)
			dot = coreBox(c).Size(6, 6).AlignSelf(Baseline)
			top = coreText(c, "top").FontSize(11)
		})
	}, 400, 300)
	bb := big.y + big.contentY() + big.tl.Lines[0].Baseline
	sb := small.y + small.contentY() + small.tl.Lines[0].Baseline
	if math.Abs(float64(bb-sb)) > 0.01 || math.Abs(float64(dot.y+dot.h-bb)) > 0.01 {
		t.Errorf("the baselines are at %v, %v and the dot's bottom at %v", bb, sb, dot.y+dot.h)
	}
	if top.y != big.y {
		t.Errorf("the text aligned to the start is at %v, want the row's top %v", top.y, big.y)
	}
}

// In a grid, the items of a row aligned by their baselines line them up,
// and the row is as high as that takes; elements without text line up
// their bottoms.
func TestBaselineGrid(t *testing.T) {
	var grid, a, b, c3, d, fixed, e, f *node
	coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			grid = coreGrid(c).ColumnTracks(Fixed(100), Fixed(100)).AlignItems(Baseline).Children(func() {
				a = coreBox(c).Height(20).Margin(5, 0, 0, 0)
				b = coreBox(c).Height(50)
				c3 = coreBox(c).Height(10)
				d = coreBox(c).Height(30).Margin(0, 0, 4, 0).AlignSelf(Start)
			})
			fixed = coreGrid(c).ColumnTracks(Fixed(100), Fixed(100)).RowTracks(Fixed(200)).AlignItems(Baseline).Children(func() {
				e = coreBox(c).Height(20)
				f = coreBox(c).Height(50).Margin(3, 0, 0, 0)
			})
		})
	}, 400, 600)
	wantBox(t, "the grid", grid, Rect{0, 0, 200, 84})
	wantBox(t, "the first", a, Rect{0, 30, 100, 20})
	wantBox(t, "the second", b, Rect{100, 0, 100, 50})
	wantBox(t, "the third", c3, Rect{0, 50, 100, 10})
	wantBox(t, "the fourth", d, Rect{100, 50, 100, 30})
	wantBox(t, "the grid of a fixed row", fixed, Rect{0, 84, 200, 200})
	wantBox(t, "its first", e, Rect{0, 33, 100, 20})
	wantBox(t, "its second", f, Rect{100, 3, 100, 50})
}

// A row's baseline is that of its first child aligned by its baseline,
// and a column's that of its first child.
func TestBaselineNested(t *testing.T) {
	var row, a, col, inner *node
	coreNewTester(func(c *context) {
		row = coreRow(c).Width(300).AlignItems(Baseline).Children(func() {
			a = coreBox(c).Height(40)
			col = coreColumn(c).Padding(7, 0, 0, 0).Children(func() {
				coreBox(c).Height(10)
				coreBox(c).Height(20)
			})
			inner = coreRow(c).AlignItems(Baseline).Margin(2, 0, 0, 0).Children(func() {
				coreBox(c).Height(5)
				coreBox(c).Height(15).Margin(0, 0, 6, 0)
			})
		})
	}, 400, 300)
	wantBox(t, "the row", row, Rect{0, 0, 300, 60})
	wantBox(t, "the box", a, Rect{0, 0, 0, 40})
	wantBox(t, "the column", col, Rect{0, 23, 0, 37})
	wantBox(t, "the inner row", inner, Rect{0, 25, 0, 21})
}

// Across a column, Baseline lines up the left edges of the children, right
// to left too, as Chrome does.
func TestBaselineColumn(t *testing.T) {
	var a, b *node
	coreNewTester(func(c *context) {
		coreColumn(c).Size(300, 100).Direction(RightToLeft).AlignItems(Baseline).Children(func() {
			a = coreBox(c).Size(50, 10)
			b = coreBox(c).Size(100, 10).Margin(0, 10, 0, 0)
		})
	}, 400, 300)
	wantBox(t, "the first", a, Rect{190, 0, 50, 10})
	wantBox(t, "the second", b, Rect{190, 10, 100, 10})
}

// An invisible child keeps its room and gives its box a baseline, as
// CSS's visibility: hidden does, while a hidden one, as display: none,
// gives none.
func TestBaselineInvisible(t *testing.T) {
	var price, sizer, shown *node
	coreNewTester(func(c *context) {
		coreRow(c).AlignItems(Baseline).Gap(4).Children(func() {
			price = coreText(c, "$42").FontSize(40)
			coreBox(c).Children(func() {
				sizer = coreText(c, "88:88").FontSize(13).Invisible()
				coreText(c, "12:34").FontSize(13).Absolute()
			})
			coreBox(c).Children(func() {
				coreText(c, "gone").FontSize(30).Hidden(true)
				shown = coreText(c, "here").FontSize(11)
			})
		})
	}, 400, 300)
	want := price.y + price.contentY() + price.tl.Lines[0].Baseline
	for name, e := range map[string]*node{"invisible": sizer, "shown": shown} {
		if b := e.y + e.contentY() + e.tl.Lines[0].Baseline; math.Abs(float64(b-want)) > 0.01 {
			t.Errorf("the %s text's baseline is at %v, want %v", name, b, want)
		}
	}
}
