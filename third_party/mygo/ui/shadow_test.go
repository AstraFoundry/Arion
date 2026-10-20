package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// TestInsetShadow checks that an inner shadow tints the box just inside
// its edge, within its rounded corners and its border, and nowhere else.
func TestInsetShadow(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Padding(20).Background(RGB(255, 255, 255)).Children(func() {
			// A box at (20, 20) to (220, 120).
			coreBox(c).Size(200, 100).Radius(24).Background(RGB(255, 255, 255)).InsetShadow(0, 0, 4, 2, RGB(255, 0, 0))
			// One at (20, 130) to (220, 230), with a border, an outer shadow
			// and two inner ones, offset.
			coreBox(c).Margin(10, 0, 0, 0).Size(200, 100).Background(RGB(255, 255, 255)).Border(4, RGB(0, 0, 255)).
				Shadow(0, 0, 0, 3, RGB(0, 255, 0)).InsetShadow(0, 6, 0, 0, RGB(255, 0, 0)).InsetShadow(0, 6, 0, 0, RGB(0, 0, 0))
		})
	}, 240, 250)
	img := tt.Image()
	white := color.RGBA{255, 255, 255, 255}
	red := func(x, y int) bool {
		c := img.RGBAAt(x, y)
		return c.R > 200 && c.G < 120 && c.B < 120
	}
	if !red(21, 70) || !red(218, 70) || !red(120, 21) || !red(120, 118) {
		t.Errorf("inside the edges: %v %v %v %v", img.RGBAAt(21, 70), img.RGBAAt(218, 70), img.RGBAAt(120, 21), img.RGBAAt(120, 118))
	}
	if c := img.RGBAAt(120, 70); c != white {
		t.Errorf("the middle of the box: %v, want white", c)
	}
	// Outside the corner's curve, and outside the box.
	for _, p := range [][2]int{{21, 21}, {218, 21}, {21, 118}, {218, 118}, {18, 70}, {222, 70}, {120, 18}, {120, 123}} {
		if c := img.RGBAAt(p[0], p[1]); c != white {
			t.Errorf("(%d, %d) = %v, want white", p[0], p[1], c)
		}
	}

	// The border stays blue and the outer shadow green; inside, the
	// later inner shadow paints over the earlier, along the top only.
	if c := img.RGBAAt(21, 180); c != (color.RGBA{0, 0, 255, 255}) {
		t.Errorf("the border: %v", c)
	}
	if c := img.RGBAAt(18, 180); c != (color.RGBA{0, 255, 0, 255}) {
		t.Errorf("the outer shadow: %v", c)
	}
	if c := img.RGBAAt(120, 136); c != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("below the top border: %v, want the last inner shadow's black", c)
	}
	for _, p := range [][2]int{{120, 142}, {120, 224}, {26, 180}} {
		if c := img.RGBAAt(p[0], p[1]); c != white {
			t.Errorf("(%d, %d) = %v, want white", p[0], p[1], c)
		}
	}

	// The ops: the outer shadow before the background, the inner ones
	// after it, in order.
	var kinds []string
	for _, op := range tt.h.last.Ops[len(tt.h.last.Ops)-4:] {
		switch {
		case op.Kind == scene.OpShadow && op.Inset:
			kinds = append(kinds, "inset")
		case op.Kind == scene.OpShadow:
			kinds = append(kinds, "shadow")
		default:
			kinds = append(kinds, "fill")
		}
	}
	if got := len(kinds); got != 4 || kinds[0] != "shadow" || kinds[1] != "fill" || kinds[2] != "inset" || kinds[3] != "inset" {
		t.Errorf("ops %v, want shadow, fill, inset, inset", kinds)
	}
}
