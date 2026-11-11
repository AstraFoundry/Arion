package raster

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

var (
	red   = scene.Color{R: 255, A: 255}
	green = scene.Color{G: 255, A: 255}
	blue  = scene.Color{B: 255, A: 255}
)

// at returns the RGBA of pixel (x, y) of a BGRA image.
func at(img *Image, x, y int) [4]byte {
	p := img.Pix[y*img.Stride+4*x:][:4]
	return [4]byte{p[2], p[1], p[0], p[3]}
}

func nearRGBA(a, b [4]byte, tol int) bool {
	for i := range a {
		if d := int(a[i]) - int(b[i]); d > tol || d < -tol {
			return false
		}
	}
	return true
}

// A conic gradient starts at its angle, clockwise from upwards, and goes
// once round its center: its stops at their fractions of the turn.
func TestConic(t *testing.T) {
	s := &scene.Scene{Width: 100, Height: 100}
	s.Stops = []scene.Stop{{Color: red, At: 0}, {Color: green, At: 0.25}, {Color: blue, At: 0.5}, {Color: red, At: 1}}
	s.Ops = []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 100, H: 100}, Paint: scene.PaintConic,
		Gradient: [4]float32{50, 50, 0, 0}, Stops: [2]int32{0, 4}}}
	img := NewImage(100, 100)
	Render(img, s)
	for _, c := range []struct {
		x, y int
		want [4]byte
	}{
		{90, 50, [4]byte{0, 255, 0, 255}},   // right: a quarter turn
		{50, 90, [4]byte{0, 0, 255, 255}},   // down: half
		{10, 50, [4]byte{128, 0, 128, 255}}, // left: halfway from blue to red
		{50, 10, [4]byte{255, 0, 0, 255}},   // up, just right of the start
		{90, 10, [4]byte{128, 128, 0, 255}}, // up-right: an eighth, halfway to green
	} {
		if got := at(img, c.x, c.y); !nearRGBA(got, c.want, 6) {
			t.Errorf("(%d, %d) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
	// Turned a quarter, green is up.
	s.Ops[0].Gradient[2] = -math.Pi / 2
	Render(img, s)
	if got := at(img, 50, 10); !nearRGBA(got, [4]byte{0, 255, 0, 255}, 6) {
		t.Errorf("turned a quarter back, up = %v, want green", got)
	}
}

// A radial gradient spreads from its center, its stops at their fractions
// of its radius: its first color at the center, its last past the radius,
// and between them the sRGB mix, or the Oklab one, which is lighter.
func TestRadial(t *testing.T) {
	s := &scene.Scene{Width: 100, Height: 100}
	s.Stops = []scene.Stop{{Color: red, At: 0}, {Color: blue, At: 1}}
	s.Ops = []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 100, H: 100}, Paint: scene.PaintRadial,
		Gradient: [4]float32{50, 50, 40, 0}, Stops: [2]int32{0, 2}}}
	img := NewImage(100, 100)
	Render(img, s)
	for _, c := range []struct {
		x, y int
		want [4]byte
	}{
		{50, 50, [4]byte{251, 0, 4, 255}},   // the center
		{70, 50, [4]byte{125, 0, 130, 255}}, // halfway out, to the right
		{50, 29, [4]byte{125, 0, 130, 255}}, // and up: a circle
		{95, 50, [4]byte{0, 0, 255, 255}},   // past the radius
		{2, 2, [4]byte{0, 0, 255, 255}},     // the corner
	} {
		if got := at(img, c.x, c.y); !nearRGBA(got, c.want, 4) {
			t.Errorf("(%d, %d) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
	// Off center, with the radius as far as the right edge.
	s.Ops[0].Gradient = [4]float32{20, 50, 80, 0}
	Render(img, s)
	if got := at(img, 20, 50); !nearRGBA(got, [4]byte{253, 0, 2, 255}, 4) {
		t.Errorf("the center moved left %v, want red", got)
	}
	if got := at(img, 59, 50); !nearRGBA(got, [4]byte{128, 0, 127, 255}, 4) {
		t.Errorf("halfway to the right edge %v, want the mix", got)
	}
	mid := at(img, 59, 50)
	s.Ops[0].Paint = scene.PaintRadialOklab
	Render(img, s)
	ok := at(img, 59, 50)
	if int(ok[0])+int(ok[1])+int(ok[2]) <= int(mid[0])+int(mid[1])+int(mid[2]) {
		t.Errorf("the Oklab mix %v is no lighter than the sRGB one %v", ok, mid)
	}
	if got := at(img, 20, 50); got[0] < 245 || got[1] > 20 || got[2] > 20 || got[3] != 255 {
		t.Errorf("the Oklab center %v, want red", got)
	}
}

// A linear gradient with stops shows its first color before its first
// stop and its last after its last, and mixes in sRGB, or in Oklab, whose
// middle between red and blue is lighter.
func TestLinearStops(t *testing.T) {
	s := &scene.Scene{Width: 100, Height: 10}
	s.Stops = []scene.Stop{{Color: red, At: 0.2}, {Color: blue, At: 0.8}}
	s.Ops = []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 100, H: 10}, Paint: scene.PaintLinearStops,
		Gradient: [4]float32{0, 0, 100, 0}, Stops: [2]int32{0, 2}}}
	img := NewImage(100, 10)
	Render(img, s)
	if got := at(img, 5, 5); got != [4]byte{255, 0, 0, 255} {
		t.Errorf("before the first stop %v, want red", got)
	}
	if got := at(img, 95, 5); got != [4]byte{0, 0, 255, 255} {
		t.Errorf("after the last stop %v, want blue", got)
	}
	mid := at(img, 50, 5)
	if !nearRGBA(mid, [4]byte{126, 0, 129, 255}, 3) {
		t.Errorf("between the stops %v, want the sRGB mix", mid)
	}
	s.Ops[0].Paint = scene.PaintLinearStopsOklab
	Render(img, s)
	ok := at(img, 50, 5)
	if int(ok[0])+int(ok[1])+int(ok[2]) <= int(mid[0])+int(mid[1])+int(mid[2]) {
		t.Errorf("the Oklab mix %v is no lighter than the sRGB one %v", ok, mid)
	}
}

// A fill whose border takes its gradient (BorderPaint) paints the border
// with it, times BorderColor, and its inside with plain Color.
func TestBorderPaint(t *testing.T) {
	s := &scene.Scene{Width: 100, Height: 100}
	s.Stops = []scene.Stop{{Color: red, At: 0}, {Color: blue, At: 1}}
	s.Ops = []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 100, H: 100}, Paint: scene.PaintLinearStops,
		Gradient: [4]float32{0, 0, 100, 0}, Stops: [2]int32{0, 2}, Color: green,
		Border: scene.Uniform(10), BorderColor: scene.Color{R: 255, G: 255, B: 255, A: 255}, BorderPaint: true}}
	img := NewImage(100, 100)
	Render(img, s)
	if got := at(img, 2, 50); !nearRGBA(got, [4]byte{248, 0, 6, 255}, 3) {
		t.Errorf("the border on the left %v, want red", got)
	}
	if got := at(img, 97, 50); !nearRGBA(got, [4]byte{6, 0, 248, 255}, 3) {
		t.Errorf("the border on the right %v, want blue", got)
	}
	if got := at(img, 50, 50); got != [4]byte{0, 255, 0, 255} {
		t.Errorf("the inside %v, want green", got)
	}
	// Half transparent white halves it.
	s.Ops[0].BorderColor.A = 128
	s.Ops[0].Color = scene.Color{}
	Render(img, s)
	if got := at(img, 50, 2); got[3] < 120 || got[3] > 135 {
		t.Errorf("the border at half %v, want alpha about 128", got)
	}
}

// A blurred image spreads its pixels as a Gaussian of the blur, keeping
// their total, transparent beyond its Src; a blur large enough to be
// averaged first (BackdropOf) spreads as wide.
func TestImageBlur(t *testing.T) {
	for _, sigma := range []float32{3, 12} {
		const n = 160
		src := &scene.Image{W: n, H: n, Pix: make([]byte, 4*n*n)}
		// A white square in the middle.
		for y := 70; y < 90; y++ {
			for x := 70; x < 90; x++ {
				copy(src.Pix[4*(y*n+x):], []byte{255, 255, 255, 255})
			}
		}
		s := &scene.Scene{Width: n, Height: n}
		s.Ops = []scene.Op{{Kind: scene.OpImage, Rect: scene.Rect{W: n, H: n}, Image: src, Src: scene.Rect{W: n, H: n}, Blur: sigma}}
		img := NewImage(n, n)
		Render(img, s)
		total := 0.0
		for y := range n {
			for x := range n {
				total += float64(at(img, x, y)[3]) / 255
			}
		}
		if total < 400*0.95 || total > 400*1.05 {
			t.Errorf("blur %v: alpha in all %.1f, want about 400, the square's", sigma, total)
		}
		// Across the middle row, the coverage of a square of 20 blurred.
		for _, x := range []int{60, 70, 80} {
			want := 0.5 * (math.Erf((90-float64(x)-0.5)/(float64(sigma)*math.Sqrt2)) - math.Erf((70-float64(x)-0.5)/(float64(sigma)*math.Sqrt2)))
			wantY := 0.5 * (math.Erf(10/(float64(sigma)*math.Sqrt2)) - math.Erf(-10/(float64(sigma)*math.Sqrt2)))
			got := float64(at(img, x, 80)[3]) / 255
			if math.Abs(got-want*wantY) > 0.06 {
				t.Errorf("blur %v: alpha at (%d, 80) %.3f, want %.3f", sigma, x, got, want*wantY)
			}
		}
		if got := at(img, 2, 2); got[3] != 0 {
			t.Errorf("blur %v: the corner %v, want transparent", sigma, got)
		}
	}
}
