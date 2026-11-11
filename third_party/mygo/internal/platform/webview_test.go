package platform

import "testing"

// A placement's page takes the pointer in its clip, inside its rounded
// corners, as it shows scaled, and where nothing covers it.
func TestWebViewPlacementAt(t *testing.T) {
	// A page 200 by 100 shown at half its size, from 10, 20, its corners
	// rounded by 20 of its DIPs: 10 of the surface's.
	p := WebViewPlacement{
		Frame: RectF{X: 10, Y: 20, W: 100, H: 50}, Clip: RectF{X: 10, Y: 20, W: 100, H: 50},
		Width: 200, Height: 100, Radii: [4]float64{20, 20, 20, 0},
		Covers: []RectF{{X: 50, Y: 40, W: 10, H: 10}},
	}
	if sx, sy := p.Scale(); sx != 0.5 || sy != 0.5 {
		t.Errorf("scale %v, %v: want a half", sx, sy)
	}
	if x, y := p.ToPage(60, 45); x != 100 || y != 50 {
		t.Errorf("the surface's 60, 45 is the page's %v, %v: want its center", x, y)
	}
	for _, c := range []struct {
		x, y float64
		want bool
	}{
		{60, 30, true},   // inside
		{11, 21, false},  // cut off by the top-left corner
		{109, 21, false}, // the top-right
		{109, 69, false}, // the bottom-right
		{11, 69, true},   // the bottom-left is square
		{13, 23, true},   // inside the top-left corner's curve
		{11, 30, true},   // below the top-left corner
		{55, 45, false},  // covered
		{5, 30, false},   // outside the clip
	} {
		if got := p.At(c.x, c.y); got != c.want {
			t.Errorf("At(%v, %v) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
	// Without a size, the page is the frame's.
	q := WebViewPlacement{Frame: RectF{W: 40, H: 30}, Clip: RectF{W: 40, H: 30}}
	if w, h := q.Size(); w != 40 || h != 30 || !q.At(1, 1) || q.Rounded() {
		t.Errorf("a placement without a size or corners: size %v by %v", w, h)
	}
}
