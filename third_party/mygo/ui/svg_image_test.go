package ui

import (
	"image/color"
	"testing"
)

// TestSVGImage draws an SVG to pixels, fitted, with currentColor.
func TestSVGImage(t *testing.T) {
	s := MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="currentColor"/></svg>`))
	img := s.Image(20, 10, color.NRGBA{G: 255, A: 255})
	if b := img.Bounds(); b.Dx() != 20 || b.Dy() != 10 {
		t.Fatalf("bounds %v, want 20 by 10", b)
	}
	// Fitted in the middle, as preserveAspectRatio's default says.
	if c := img.RGBAAt(10, 5); c != (color.RGBA{G: 255, A: 255}) {
		t.Errorf("the middle is %v, want green", c)
	}
	if c := img.RGBAAt(1, 5); c.A != 0 {
		t.Errorf("beside the square is %v, want nothing", c)
	}
	if c := s.Image(4, 4, nil).RGBAAt(2, 2); c != (color.RGBA{A: 255}) {
		t.Errorf("with no current color, %v, want black", c)
	}
	var none *SVG
	if w, h := none.Size(); w != 0 || h != 0 {
		t.Errorf("a nil SVG's size is %v by %v", w, h)
	}
	if b := none.Image(4, 4, nil).Bounds(); b.Dx() != 4 {
		t.Errorf("a nil SVG's image is %v", b)
	}
	if b := s.Image(-1, 3, nil).Bounds(); !b.Empty() {
		t.Errorf("an image of no pixels is %v", b)
	}
}
