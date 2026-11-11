package ui

import (
	"image/color"
	"testing"
)

// A canvas of 8×4 pixels, drawn once, shows scaled up 4 times, pixelated:
// red and blue halves, each pixel a block of 4×4.
func TestCanvas(t *testing.T) {
	cv := NewCanvas(8, 4)
	cv.Bitmap().Pixelated()
	draws := 0
	drawn := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(32, 16).Draw(func(p *Painter, r Rect) {
			if !drawn {
				drawn = true
				draws++
				cv.Draw(p, func(cp *Painter) {
					cp.Fill(Rect{W: 4, H: 4}, RGB(255, 0, 0), 0)
					cp.Fill(Rect{X: 4, W: 4, H: 4}, RGB(0, 0, 255), 0)
					// A pixel of green, which pixelated scaling keeps whole.
					cp.Fill(Rect{X: 1, Y: 1, W: 1, H: 1}, RGB(0, 255, 0), 0)
				})
			}
			p.Image(cv.Bitmap(), r, FillBox)
		})
	}, 32, 16)
	img := tt.Image()
	at := func(x, y int) color.RGBA { return img.RGBAAt(x, y) }
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{
		{2, 2, color.RGBA{255, 0, 0, 255}},
		{6, 6, color.RGBA{0, 255, 0, 255}},
		{9, 9, color.RGBA{255, 0, 0, 255}},
		{20, 8, color.RGBA{0, 0, 255, 255}},
		{31, 15, color.RGBA{0, 0, 255, 255}},
	} {
		if got := at(c.x, c.y); got != c.want {
			t.Errorf("pixel %d,%d = %v, want %v", c.x, c.y, got, c.want)
		}
	}
	// Frames after keep what it holds without drawing it anew.
	tt.Frame()
	if draws != 1 || at(20, 8) != (color.RGBA{0, 0, 255, 255}) {
		t.Errorf("after another frame: %d draws, %v", draws, at(20, 8))
	}
}

// A canvas shows another, drawn in the same frame, and text.
func TestNestedCanvas(t *testing.T) {
	inner, outer := NewCanvas(4, 4), NewCanvas(8, 8)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(80, 40).Draw(func(p *Painter, r Rect) {
			outer.Draw(p, func(op *Painter) {
				inner.Draw(p, func(ip *Painter) {
					ip.Fill(Rect{W: 4, H: 4}, RGB(0, 200, 0), 0)
				})
				op.Image(inner.Bitmap(), Rect{X: 2, Y: 2, W: 4, H: 4}, FillBox)
			})
			p.Image(outer.Bitmap(), Rect{X: r.X, Y: r.Y, W: 8, H: 8}, FillBox)
			text := NewCanvas(60, 20)
			text.Draw(p, func(tp *Painter) { tp.Text(2, 2, "Hi", 14, RGB(255, 255, 255)) })
			p.Image(text.Bitmap(), Rect{X: r.X + 10, Y: r.Y, W: 60, H: 20}, FillBox)
		})
	}, 80, 40)
	img := tt.Image()
	if got := img.RGBAAt(4, 4); got != (color.RGBA{0, 200, 0, 255}) {
		t.Errorf("inner canvas in outer: %v", got)
	}
	if got := img.RGBAAt(0, 0); got.A != 0 && got != (color.RGBA{255, 255, 255, 255}) {
		// The theme's background shows where the outer canvas is clear.
		t.Logf("outside the inner canvas: %v", got)
	}
	lit := 0
	for y := 0; y < 20; y++ {
		for x := 10; x < 70; x++ {
			if c := img.RGBAAt(x, y); c.R > 200 && c.G > 200 && c.B > 200 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no text drawn in the canvas")
	}
}
