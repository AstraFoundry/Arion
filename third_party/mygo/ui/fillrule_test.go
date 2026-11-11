package ui

import (
	"image/color"
	"testing"
)

// TestFillRule fills and clips to rings of two squares and of two
// circles, each drawn the same way round: NonZero fills the hole, EvenOdd
// leaves it, in one mask and in tiles too, where a tile inside the hole
// that no edge reaches is covered by NonZero alone.
func TestFillRule(t *testing.T) {
	blue := RGB(20, 60, 230)
	isBlue := func(c color.RGBA) bool { return c.B > 180 && c.R < 120 }
	isWhite := func(c color.RGBA) bool { return c.R > 240 && c.G > 240 && c.B > 240 }
	view := func(rule FillRule) func(c *context) {
		return func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
				ring := MustParsePath("M0 0 H100 V100 H0 Z M25 25 H75 V75 H25 Z")
				ring.FillRule(rule)
				p.FillPath(ring, blue)
				var circles Path
				circles.Circle(200, 50, 45).Circle(200, 50, 25).FillRule(rule)
				p.FillPathGradient(&circles, LinearGradient{From: blue, To: blue})
				var clip Path
				clip.Circle(320, 50, 45).Circle(320, 50, 25).FillRule(rule)
				p.ClipPath(&clip, func() { p.Fill(Rect{270, 0, 100, 100}, blue, 0) })
			})
		}
	}
	for _, tiled := range []bool{false, true} {
		for _, rule := range []FillRule{NonZero, EvenOdd} {
			func() {
				if tiled {
					defer func(most, tile int) { pathMaskMax, pathTile = most, tile }(pathMaskMax, pathTile)
					pathMaskMax, pathTile = 0, 17
				}
				img := coreRender(view(rule), 380, 100, 1)
				for _, x := range []int{50, 200, 320} {
					hole := img.RGBAAt(x, 50)
					if rule == NonZero && !isBlue(hole) || rule == EvenOdd && !isWhite(hole) {
						t.Errorf("tiled %v, rule %d: the hole at x=%d shows %v", tiled, rule, x, hole)
					}
				}
				for _, x := range []int{12, 162, 282} {
					if band := img.RGBAAt(x, 50); !isBlue(band) {
						t.Errorf("tiled %v, rule %d: the band at x=%d shows %v", tiled, rule, x, band)
					}
				}
			}()
		}
	}
}

// TestPathContains tests points of a ring of two circles by both rules,
// of a path's curves and of a trimmed one, without allocating.
func TestPathContains(t *testing.T) {
	var ring Path
	ring.Circle(50, 50, 40).Circle(50, 50, 20)
	for _, tc := range []struct {
		x, y             float32
		nonZero, evenOdd bool
	}{
		{50, 50, true, false},  // the hole
		{50, 20, true, true},   // the band
		{88, 50, true, true},   // inside the curve near its edge
		{91, 50, false, false}, // just outside it
		{80, 80, false, false}, // outside the corner of the circle
		{-5, 50, false, false},
	} {
		ring.FillRule(NonZero)
		if got := ring.Contains(tc.x, tc.y); got != tc.nonZero {
			t.Errorf("NonZero: Contains(%v, %v) = %v", tc.x, tc.y, got)
		}
		ring.FillRule(EvenOdd)
		if got := ring.Contains(tc.x, tc.y); got != tc.evenOdd {
			t.Errorf("EvenOdd: Contains(%v, %v) = %v", tc.x, tc.y, got)
		}
	}
	// A subpath drawn the other way round leaves a hole by either rule.
	hole := MustParsePath("M0 0 H100 V100 H0 Z M25 25 V75 H75 V25 Z")
	if hole.Contains(50, 50) || !hole.Contains(10, 50) {
		t.Error("a subpath wound the other way round does not leave a hole")
	}
	// Open subpaths close, as filling closes them.
	var tri Path
	tri.MoveTo(0, 0).LineTo(100, 0).LineTo(0, 100)
	if !tri.Contains(20, 20) || tri.Contains(80, 80) {
		t.Error("an open triangle does not contain what it fills")
	}
	// Trim keeps what it fills: the first half of a square's outline is
	// the triangle under its diagonal.
	var sq Path
	sq.MoveTo(0, 0).LineTo(100, 0).LineTo(100, 100).LineTo(0, 100).Close().Trim(0, 0.5)
	if !sq.Contains(80, 20) || sq.Contains(20, 80) {
		t.Error("a trimmed path does not contain what it fills")
	}
	if (&Path{}).Contains(0, 0) {
		t.Error("an empty path contains a point")
	}
	ring.Contains(50, 50)
	if n := testing.AllocsPerRun(100, func() { ring.Contains(50, 20) }); n != 0 {
		t.Errorf("Contains allocates %v times", n)
	}
}

// TestHitTest makes a ring take the pointer on its band alone: a press
// in its hole goes to the element under it, as does the hover, and the
// element's children take the pointer as they would.
func TestHitTest(t *testing.T) {
	var ringClicks, underClicks, innerClicks int
	var ringHovered bool
	ring := func(r Rect) *Path {
		var p Path
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		return p.Circle(cx, cy, r.W/2).Circle(cx, cy, r.W/2-20).FillRule(EvenOdd)
	}
	view := func(c *context) {
		coreBox(c).Size(300, 200).Children(func() {
			if coreBox(c).Absolute().Left(0).Top(0).Size(300, 200).Clicked() {
				underClicks++
			}
			e := coreBox(c).Absolute().Left(50).Top(50).Size(100, 100).
				HitTest(func(x, y float32, r Rect) bool { return ring(r).Contains(x, y) })
			ringHovered = e.Hovered()
			if e.Clicked() {
				ringClicks++
			}
			e.Children(func() {
				if coreBox(c).Absolute().Left(95).Top(95).Size(10, 10).Clicked() {
					innerClicks++
				}
			})
		})
	}
	tt := coreNewTester(view, 300, 200)
	tt.Move(100, 60) // on the band
	tt.settle()
	if !ringHovered {
		t.Error("the ring does not hover with the pointer on its band")
	}
	tt.ClickAt(100, 60)
	if ringClicks != 1 || underClicks != 0 {
		t.Errorf("a click on the band: %d ring clicks, %d under it", ringClicks, underClicks)
	}
	tt.Move(100, 100) // in the hole
	tt.settle()
	if ringHovered {
		t.Error("the ring hovers with the pointer in its hole")
	}
	tt.ClickAt(100, 100)
	if ringClicks != 1 || underClicks != 1 {
		t.Errorf("a click in the hole: %d ring clicks, %d under it", ringClicks, underClicks)
	}
	// Outside the ring's circle, in the corner of its box.
	tt.ClickAt(55, 55)
	if ringClicks != 1 || underClicks != 2 {
		t.Errorf("a click in the corner: %d ring clicks, %d under it", ringClicks, underClicks)
	}
	// A child out of the band takes the pointer.
	tt.ClickAt(150, 150)
	if innerClicks != 1 || underClicks != 2 {
		t.Errorf("a click on a child: %d of its clicks, %d under it", innerClicks, underClicks)
	}
}
