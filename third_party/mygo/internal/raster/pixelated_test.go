package raster

import (
	"math"
	"testing"
)

// Pixelated sampling lands on a pixel's center, but within one pixel of the
// frame from its edge, where it blends with the next across that pixel.
func TestPixelated(t *testing.T) {
	for _, scale := range []float32{2, 2.5, 3, 4.75} {
		d := 1 / scale
		blended := 0
		for x := range 40 {
			u := (float32(x) + 0.5) * d // the image's coordinate of pixel x
			p := pixelated(u, d)
			// Within rounding of a pixel's center.
			frac := p - 0.5 - float32(math.Round(float64(p-0.5)))
			if math.Abs(float64(frac)) < 1e-4 {
				// The center of the pixel under u.
				if c := float32(math.Floor(float64(u))) + 0.5; math.Abs(float64(p-c)) > 1e-4 {
					t.Errorf("scale %v, pixel %d: %v, want %v", scale, x, p, c)
				}
				continue
			}
			blended++
			// Blending: u is within half a frame pixel of an edge.
			edge := float32(math.Round(float64(u)))
			if math.Abs(float64(u-edge)) > float64(d)/2+1e-4 {
				t.Errorf("scale %v, pixel %d blends %v away from an edge", scale, x, u-edge)
			}
		}
		// Whole scales put no pixel of the frame across an edge.
		if whole := scale == float32(math.Trunc(float64(scale))); whole != (blended == 0) {
			t.Errorf("scale %v blends %d pixels", scale, blended)
		}
	}
	// Shown smaller, it is bilinear.
	if p := pixelated(3.3, 2); p != 3.3 {
		t.Errorf("at half the size: %v", p)
	}
}
