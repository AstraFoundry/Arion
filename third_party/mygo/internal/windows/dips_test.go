package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func display(x, y, width, height, dpi int) screenDisplay {
	r := platform.Rect{X: x, Y: y, Width: width, Height: height}
	work := r
	work.Height -= 40 * dpi / 96 // a taskbar
	return screenDisplay{bounds: r, work: work, dpi: dpi}
}

// layouts are arrangements of displays as Windows reports them, in
// pixels, the primary one at 0, 0.
var layouts = map[string][]screenDisplay{
	"200% and 175% to its right (#1039)": {display(0, 0, 3840, 2160, 192), display(3840, 0, 2560, 1600, 168)},
	"200% and 175% to its right, lower":  {display(0, 0, 3840, 2160, 192), display(3840, 647, 2560, 1600, 168)},
	"100% and 200% to its right":         {display(0, 0, 2560, 1440, 96), display(2560, 0, 3840, 2160, 192)},
	"200% and 100% to its left":          {display(0, 0, 3840, 2160, 192), display(-2560, 0, 2560, 1440, 96)},
	"150% and 100% above it":             {display(0, 0, 2880, 1800, 144), display(200, -1440, 2560, 1440, 96)},
	"125%, 150% left, 250% right":        {display(0, 0, 2560, 1600, 120), display(-1920, 300, 1920, 1080, 144), display(2560, -400, 3840, 2400, 240)},
	"150% twice":                         {display(0, 0, 2880, 1800, 144), display(2880, 0, 2880, 1800, 144)},
}

// TestDIPSpaceDisplaysApart checks that no two displays overlap in DIPs,
// and that each one's work area is in its bounds.
func TestDIPSpaceDisplaysApart(t *testing.T) {
	for name, displays := range layouts {
		s := newDIPSpace(displays, 96)
		for i := range displays {
			a, work := s.displayDIPs(i)
			if w, h := overlap(work, a); w != work.Width || h != work.Height {
				t.Errorf("%s: display %d's work area %+v is not in its bounds %+v", name, i, work, a)
			}
			for j := i + 1; j < len(displays); j++ {
				b, _ := s.displayDIPs(j)
				if w, h := overlap(a, b); w > 0 && h > 0 {
					t.Errorf("%s: displays %d %+v and %d %+v overlap in DIPs", name, i, a, j, b)
				}
			}
		}
	}
}

// TestDIPSpaceDisplays checks where displays are in DIPs: the primary one
// in its scale from 0, 0, the others from their top-left pixel scaled by
// the smallest scale, and all of them as one where they have one scale.
func TestDIPSpaceDisplays(t *testing.T) {
	for _, tt := range []struct {
		layout string
		want   []platform.Rect
	}{
		{"200% and 175% to its right (#1039)", []platform.Rect{{Width: 1920, Height: 1080}, {X: 2194, Width: 1463, Height: 914}}},
		{"200% and 175% to its right, lower", []platform.Rect{{Width: 1920, Height: 1080}, {X: 2194, Y: 370, Width: 1463, Height: 914}}},
		{"100% and 200% to its right", []platform.Rect{{Width: 2560, Height: 1440}, {X: 2560, Width: 1920, Height: 1080}}},
		{"200% and 100% to its left", []platform.Rect{{Width: 1920, Height: 1080}, {X: -2560, Width: 2560, Height: 1440}}},
		{"150% twice", []platform.Rect{{Width: 1920, Height: 1200}, {X: 1920, Width: 1920, Height: 1200}}},
	} {
		s := newDIPSpace(layouts[tt.layout], 96)
		for i, want := range tt.want {
			if got, _ := s.displayDIPs(i); got != want {
				t.Errorf("%s: display %d at %+v, want %+v", tt.layout, i, got, want)
			}
		}
	}
}

// TestDIPSpacePoints checks that a point names one display both ways: a
// pixel of a display is in its DIPs, a DIP of a display in its pixels,
// and DIPs come back from their pixels.
func TestDIPSpacePoints(t *testing.T) {
	for name, displays := range layouts {
		s := newDIPSpace(displays, 96)
		for i, d := range displays {
			b := d.bounds
			bd, _ := s.displayDIPs(i)
			for _, f := range [][2]int{{0, 0}, {1, 1}, {3, 7}, {1, 2}, {4, 4}, {7, 3}, {8, 8}} {
				px := platform.Point{X: b.X + (b.Width-1)*f[0]/8, Y: b.Y + (b.Height-1)*f[1]/8}
				if p := s.pointToDIP(px.X, px.Y); !inside(p, bd) {
					t.Errorf("%s: pixel %+v of display %d is at DIP %+v, outside its %+v", name, px, i, p, bd)
				}
				dip := platform.Point{X: bd.X + (bd.Width-1)*f[0]/8, Y: bd.Y + (bd.Height-1)*f[1]/8}
				px = s.pointToPixels(dip.X, dip.Y)
				if !inside(px, b) {
					t.Errorf("%s: DIP %+v of display %d is at pixel %+v, outside its %+v", name, dip, i, px, b)
				}
				if back := s.pointToDIP(px.X, px.Y); back != dip {
					t.Errorf("%s: DIP %+v is pixel %+v, which is DIP %+v", name, dip, px, back)
				}
			}
		}
	}
}

// TestDIPSpaceIssue1039 places a window on a display of another scale
// than the primary's, where its DIPs say, and reads them back.
func TestDIPSpaceIssue1039(t *testing.T) {
	s := newDIPSpace(layouts["200% and 175% to its right (#1039)"], 96)
	_, work := s.displayDIPs(1)
	if work.X != 2194 {
		t.Fatalf("the secondary display's work area starts at DIP %d, want 2194", work.X)
	}
	r := platform.Rect{X: work.X, Y: work.Y, Width: 900, Height: 600}
	px, i := s.rectToPixels(r)
	if want := (platform.Rect{X: 3840, Width: 1575, Height: 1050}); px != want || i != 1 {
		t.Fatalf("rectToPixels(%+v) = %+v on display %d, want %+v on display 1", r, px, i, want)
	}
	if back := s.rectToDIP(px); back != r {
		t.Fatalf("rectToDIP(%+v) = %+v, want %+v", px, back, r)
	}
	// From the primary display, as SetBounds moves a window there.
	r = platform.Rect{X: 2194 + 300, Y: 200, Width: 800, Height: 500}
	if px, _ := s.rectToPixels(r); px != (platform.Rect{X: 3840 + 525, Y: 350, Width: 1400, Height: 875}) {
		t.Fatalf("rectToPixels(%+v) = %+v", r, px)
	}
	if p := s.pointToPixels(2194, 0); p != (platform.Point{X: 3840}) {
		t.Fatalf("pointToPixels(2194, 0) = %+v, want 3840, 0", p)
	}
}

// TestDIPSpaceRects checks that windows on a display, across its edges
// but mostly on it, or partly off the screen, keep their rectangle in
// DIPs through pixels, in the scale of that display.
func TestDIPSpaceRects(t *testing.T) {
	for name, displays := range layouts {
		s := newDIPSpace(displays, 96)
		for i := range displays {
			bd, _ := s.displayDIPs(i)
			for _, r := range []platform.Rect{
				{X: bd.X + 40, Y: bd.Y + 30, Width: 800, Height: 600},
				{X: bd.X + bd.Width - 700, Y: bd.Y + 100, Width: 800, Height: 400}, // over the right edge
				{X: bd.X - 100, Y: bd.Y - 50, Width: 640, Height: 480},             // over the top-left corner
				{X: bd.X + 10, Y: bd.Y + bd.Height - 150, Width: 300, Height: 200}, // over the bottom edge
			} {
				px, on := s.rectToPixels(r)
				if on != i {
					t.Errorf("%s: rectToPixels(%+v) took the scale of display %d, want %d", name, r, on, i)
				}
				if back := s.rectToDIP(px); back != r {
					t.Errorf("%s: rectToPixels(%+v) = %+v, which rectToDIP gives back as %+v", name, r, px, back)
				}
			}
		}
	}
}

// TestDIPSpaceBoundsSettled checks that a window anywhere, set to the
// bounds it has, as SetSize and SetPosition do, moves by no more than
// rounding its pixels to DIPs takes, and then stays: across displays of
// different scales, a rectangle in DIPs can be the most on another
// display than its pixels are, and is placed by the scale of the display
// it ends up the most on.
func TestDIPSpaceBoundsSettled(t *testing.T) {
	for name, displays := range layouts {
		s := newDIPSpace(displays, 96)
		for i, d := range displays {
			b := d.bounds
			for _, px := range []platform.Rect{
				{X: b.X + 81, Y: b.Y + 47, Width: 1501, Height: 1001},
				{X: b.X + b.Width - 701, Y: b.Y + 99, Width: 1503, Height: 801}, // across the right edge
				{X: b.X - 1601, Y: b.Y + 33, Width: 2999, Height: 603},          // across the left edge
				{X: b.X - 99, Y: b.Y - 999, Width: 1279, Height: 1401},          // across the top
				{X: b.X + b.Width + 555, Y: b.Y + 77, Width: 999, Height: 701},  // off the screen
			} {
				p1, _ := s.rectToPixels(s.rectToDIP(px))
				p2, _ := s.rectToPixels(s.rectToDIP(p1))
				if p2 != p1 {
					t.Errorf("%s: display %d: pixels %+v moved to %+v, then to %+v", name, i, px, p1, p2)
				}
				if s.pixelDisplay(p1) == s.pixelDisplay(px) {
					slack := (s.dpiOf(s.pixelDisplay(px)) + 191) / 192
					if abs(p1.X-px.X) > slack || abs(p1.Y-px.Y) > slack || abs(p1.Width-px.Width) > slack || abs(p1.Height-px.Height) > slack {
						t.Errorf("%s: display %d: pixels %+v moved to %+v", name, i, px, p1)
					}
				}
			}
		}
	}
}

func abs(v int) int { return max(v, -v) }

// TestDIPSpaceWindowAcross gives a window across displays the scale of
// the one it is the most on, as Windows gives it its DPI.
func TestDIPSpaceWindowAcross(t *testing.T) {
	s := newDIPSpace(layouts["100% and 200% to its right"], 96)
	// 560 pixels on the 100% display, 940 on the 200% one.
	px := platform.Rect{X: 2000, Y: 100, Width: 1500, Height: 800}
	if got, want := s.rectToDIP(px), (platform.Rect{X: 2280, Y: 50, Width: 750, Height: 400}); got != want {
		t.Fatalf("rectToDIP(%+v) = %+v, want %+v", px, got, want)
	}
	if got, _ := s.rectToPixels(platform.Rect{X: 2280, Y: 50, Width: 750, Height: 400}); got != px {
		t.Fatalf("rectToPixels gives %+v, want %+v", got, px)
	}
}

// TestDIPSpaceOffScreen converts points and rectangles beyond every
// display, and in the gaps between displays in DIPs, through the
// nearest display.
func TestDIPSpaceOffScreen(t *testing.T) {
	s := newDIPSpace(layouts["200% and 175% to its right (#1039)"], 96)
	// Between the primary display's DIPs (to 1920) and the secondary's
	// (from 2194): nearer the primary.
	if p := s.pointToPixels(2000, 100); p != (platform.Point{X: 4000, Y: 200}) {
		t.Errorf("pointToPixels(2000, 100) = %+v, want 4000, 200", p)
	}
	// Nearer the secondary.
	if p := s.pointToPixels(2190, 100); p != (platform.Point{X: 3833, Y: 175}) {
		t.Errorf("pointToPixels(2190, 100) = %+v, want 3833, 175", p)
	}
	// Below the secondary display, and left of the primary.
	if p := s.pointToDIP(5000, 2000); p != (platform.Point{X: 2194 + 662, Y: 1142}) {
		t.Errorf("pointToDIP(5000, 2000) = %+v", p)
	}
	if p := s.pointToDIP(-301, -3); p != (platform.Point{X: -151, Y: -2}) {
		t.Errorf("pointToDIP(-301, -3) = %+v, want -151, -2", p)
	}
}

// TestDIPSpaceNoDisplays converts at the system's DPI without displays.
func TestDIPSpaceNoDisplays(t *testing.T) {
	s := newDIPSpace(nil, 144)
	if got := s.rectToDIP(platform.Rect{X: 150, Y: -150, Width: 300, Height: 30}); got != (platform.Rect{X: 100, Y: -100, Width: 200, Height: 20}) {
		t.Errorf("rectToDIP = %+v", got)
	}
	if got, i := s.rectToPixels(platform.Rect{X: 100, Y: 100, Width: 200, Height: 20}); got != (platform.Rect{X: 150, Y: 150, Width: 300, Height: 30}) || i != -1 {
		t.Errorf("rectToPixels = %+v, %d", got, i)
	}
}

func TestScaled(t *testing.T) {
	for _, tt := range []struct{ v, num, den, want int }{
		{3840, 96, 168, 2194}, {647, 96, 168, 370}, {1, 96, 192, 1}, {3, 96, 192, 2},
		{-1, 96, 192, 0}, {-3, 96, 192, -1}, {-301, 96, 192, -150}, {-2560, 96, 96, -2560},
		{900, 168, 96, 1575}, {-7, 168, 96, -12},
	} {
		if got := scaled(tt.v, tt.num, tt.den); got != tt.want {
			t.Errorf("scaled(%d, %d, %d) = %d, want %d", tt.v, tt.num, tt.den, got, tt.want)
		}
	}
}

func inside(p platform.Point, r platform.Rect) bool {
	return p.X >= r.X && p.X < r.X+r.Width && p.Y >= r.Y && p.Y < r.Y+r.Height
}

func overlap(a, b platform.Rect) (int, int) {
	return max(0, min(a.X+a.Width, b.X+b.Width)-max(a.X, b.X)),
		max(0, min(a.Y+a.Height, b.Y+b.Height)-max(a.Y, b.Y))
}
