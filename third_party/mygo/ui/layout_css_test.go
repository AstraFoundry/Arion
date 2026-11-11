package ui

import (
	"math"
	"testing"
)

// The flexbox tests below lay out what Chrome lays out alike: the sizes
// they want are Chrome's.

// layoutIn lays out what build makes in a 600×500 window.
func layoutIn(t *testing.T, build func(c *context)) {
	t.Helper()
	coreNewTester(func(c *context) {
		coreColumn(c).Size(600, 500).Children(func() { build(c) })
	}, 600, 500)
}

func wantSize(t *testing.T, name string, e *node, w, h float32) {
	t.Helper()
	if math.Abs(float64(e.w-w)) > 0.05 || math.Abs(float64(e.h-h)) > 0.05 {
		t.Errorf("%s is %v×%v, want %v×%v", name, e.w, e.h, w, h)
	}
}

// A box is never smaller than its padding, whatever its size says.
func TestSizeNoSmallerThanPadding(t *testing.T) {
	var inColumn, inRow *node
	layoutIn(t, func(c *context) {
		inColumn = coreBox(c).Width(8).Padding(0, 3, 0, 11)
		coreRow(c).Children(func() {
			inRow = coreBox(c).Width(8).Padding(0, 3, 0, 11)
		})
	})
	wantSize(t, "the box in a column", inColumn, 14, 0)
	wantSize(t, "the box in a row", inRow, 14, 0)
}

// Items shrink by their flex base sizes less their padding.
func TestShrinkByInnerBaseSize(t *testing.T) {
	var a, b *node
	layoutIn(t, func(c *context) {
		coreRow(c).AlignItems(Stretch).Width(300).Children(func() {
			a = coreBox(c).Basis(200)
			b = coreBox(c).Basis(200).Padding(0, 0, 0, 100)
		})
	})
	wantSize(t, "the first", a, 133.33, 0)
	wantSize(t, "the second", b, 166.67, 0)
}

// A flex base size smaller than the padding is the padding's.
func TestBaseSizeNoSmallerThanPadding(t *testing.T) {
	var a, b *node
	layoutIn(t, func(c *context) {
		a = coreBox(c).Grow(1).Height(9).Padding(14, 0, 0, 0)
		b = coreBox(c).Grow(2)
	})
	wantSize(t, "the padded", a, 600, 176)
	wantSize(t, "the other", b, 600, 324)
}

// An item's minimum does not change the flex base size its free space is
// shared from.
func TestBaseSizeBeforeMinimum(t *testing.T) {
	var a, b *node
	layoutIn(t, func(c *context) {
		// Growing from their content, as Grow with no basis would.
		a = coreBox(c)
		a.grow = 1
		b = coreBox(c).MinHeight(166)
		b.grow = 1
	})
	wantSize(t, "the first", a, 600, 250)
	wantSize(t, "the second", b, 600, 250)
}

// An item whose minimum wins over its maximum is frozen at it, and its
// siblings share the rest.
func TestMinimumOverMaximumFreezes(t *testing.T) {
	var a, b *node
	layoutIn(t, func(c *context) {
		coreColumn(c).Children(func() {
			a = coreRow(c).Height(163).MinHeight(42).MaxHeight(14)
			b = coreBox(c).Children(func() {
				coreBox(c).Margin(8, 0, 0, 0)
			})
			b.grow = 1
		})
	})
	wantSize(t, "the clamped", a, 600, 42)
	wantSize(t, "the growing", b, 600, 8)
}

// Items of a column that wraps stretch to their line, as wide as the
// widest there, not to the column.
func TestWrappingColumnStretchesToLine(t *testing.T) {
	var a, b *node
	coreNewTester(func(c *context) {
		coreColumn(c).Size(600, 500).Wrap().Children(func() {
			a = coreRow(c).AlignItems(Stretch).Height(10)
			b = coreRow(c).Size(50, 10)
		})
	}, 600, 500)
	wantSize(t, "the stretched", a, 50, 10)
	wantSize(t, "the widest", b, 50, 10)
}

// A row as wide as its content counts an item that cannot shrink at least
// as wide as its basis, and one that cannot grow at most as wide, but no
// narrower than its content's minimum.
func TestContentWidthOfRowByBasis(t *testing.T) {
	var fixed, capped *node
	layoutIn(t, func(c *context) {
		coreRow(c).AlignItems(Start).Children(func() {
			fixed = coreRow(c).Children(func() {
				coreBox(c).Shrink(0).Basis(47)
			})
			capped = coreRow(c).Children(func() {
				coreBox(c).Basis(56).Children(func() {
					coreBox(c).Width(117)
				})
			})
		})
	})
	wantSize(t, "the row of an item that cannot shrink", fixed, 47, 0)
	wantSize(t, "the row of an item that cannot grow", capped, 117, 0)
}

// A text of a no-break space between words keeps its advance, as
// `&nbsp;` does on the web; a space alone in its text is removed, as a
// collapsible space starting a line is in CSS.
func TestNoBreakSpaceText(t *testing.T) {
	var a, nbsp, b, space *node
	layoutIn(t, func(c *context) {
		coreRow(c).Children(func() {
			a = coreText(c, "a")
			nbsp = coreText(c, "\u00a0")
			b = coreText(c, "b")
			space = coreText(c, " ")
		})
	})
	if nbsp.w <= 0 || math.Abs(float64(b.x-a.w-nbsp.w)) > 0.05 {
		t.Errorf("the no-break space is %v wide, b at %v after a %v wide", nbsp.w, b.x, a.w)
	}
	if space.w != 0 {
		t.Errorf("the space is %v wide", space.w)
	}
}
