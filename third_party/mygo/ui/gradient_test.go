package ui

import (
	"image"
	"math"
	"testing"
)

var (
	gRed   = RGB(255, 0, 0)
	gGreen = RGB(0, 255, 0)
	gBlue  = RGB(0, 0, 255)
)

// rgba returns pixel (x, y) of img.
func rgba(img *image.RGBA, x, y int) [4]uint8 {
	p := img.Pix[img.PixOffset(x, y):][:4]
	return [4]uint8{p[0], p[1], p[2], p[3]}
}

func nearRGBA8(a, b [4]uint8, tol int) bool {
	for i := range a {
		if d := int(a[i]) - int(b[i]); d > tol || d < -tol {
			return false
		}
	}
	return true
}

// A conic gradient fills the element around its middle, from its angle
// clockwise, as CSS's conic-gradient().
func TestConicGradient(t *testing.T) {
	stops := []GradientStop{{gRed, 0}, {gGreen, 0.25}, {gBlue, 0.5}, {gRed, 1}}
	for _, c := range []struct {
		g    ConicGradient
		x, y int
		want [4]uint8
	}{
		{ConicGradient{Stops: stops}, 90, 50, [4]uint8{0, 255, 0, 255}},            // right: a quarter
		{ConicGradient{Stops: stops}, 50, 90, [4]uint8{0, 0, 255, 255}},            // down: half
		{ConicGradient{Stops: stops, Angle: 90}, 50, 90, [4]uint8{0, 255, 0, 255}}, // from the right
		{ConicGradient{Stops: stops, Center: Point{0.1, 0.5}}, 90, 52, [4]uint8{0, 255, 0, 255}},
	} {
		g := c.g
		tt := NewTester(func(ctx *Context) {
			Box(ctx).Size(100, 100).ConicGradient(g)
		}, 100, 100)
		if got := rgba(tt.Image(), c.x, c.y); !nearRGBA8(got, c.want, 8) {
			t.Errorf("%+v at (%d, %d) = %v, want %v", g, c.x, c.y, got, c.want)
		}
	}
}

// A radial gradient spreads its stops from its center, the middle unless
// set, to its radius, in DIPs, or to the farthest corner without one, as
// CSS's radial-gradient(): its first color at the center, its last past
// the radius.
func TestRadialGradient(t *testing.T) {
	stops := []GradientStop{{gRed, 0}, {gBlue, 1}}
	mid := [4]uint8{128, 0, 127, 255}
	for _, c := range []struct {
		g    RadialGradient
		x, y int
		want [4]uint8
	}{
		{RadialGradient{Stops: stops, Radius: 40}, 50, 50, [4]uint8{255, 0, 0, 255}}, // the center
		{RadialGradient{Stops: stops, Radius: 40}, 70, 50, mid},                      // halfway out
		{RadialGradient{Stops: stops, Radius: 40}, 50, 30, mid},                      // as far up
		{RadialGradient{Stops: stops, Radius: 40}, 95, 50, [4]uint8{0, 0, 255, 255}}, // past the radius
		{RadialGradient{Stops: stops, Radius: 40}, 5, 95, [4]uint8{0, 0, 255, 255}},
		// The farthest corner, 50√2 away.
		{RadialGradient{Stops: stops}, 85, 50, mid},
		{RadialGradient{Stops: stops}, 99, 99, [4]uint8{3, 0, 252, 255}},
		// Off center: the farthest corners are on the right, √(80²+50²) away.
		{RadialGradient{Stops: stops, Center: Point{0.2, 0.5}}, 20, 50, [4]uint8{255, 0, 0, 255}},
		{RadialGradient{Stops: stops, Center: Point{0.2, 0.5}}, 67, 50, mid},
		{RadialGradient{Stops: stops, Center: Point{0.2, 0.5}, Radius: 10}, 50, 50, [4]uint8{0, 0, 255, 255}},
	} {
		g := c.g
		tt := NewTester(func(ctx *Context) {
			Box(ctx).Size(100, 100).RadialGradient(g)
		}, 100, 100)
		if got := rgba(tt.Image(), c.x, c.y); !nearRGBA8(got, c.want, 6) {
			t.Errorf("%+v at (%d, %d) = %v, want %v", g, c.x, c.y, got, c.want)
		}
	}
	// Mixed in Oklab, the middle is lighter than in sRGB.
	tt := NewTester(func(ctx *Context) {
		Box(ctx).Size(100, 100).RadialGradient(RadialGradient{Stops: stops, Radius: 40, Oklab: true})
	}, 100, 100)
	got := rgba(tt.Image(), 70, 50)
	if int(got[0])+int(got[1])+int(got[2]) <= int(mid[0])+int(mid[1])+int(mid[2])+20 || got[3] != 255 {
		t.Errorf("the Oklab middle %v, want lighter than the sRGB one %v", got, mid)
	}
	// The radius is in DIPs: twice as many pixels at twice the scale.
	img := Render(func(ctx *Context) {
		Box(ctx).Size(100, 100).RadialGradient(RadialGradient{Stops: stops, Radius: 40})
	}, 100, 100, 2)
	if got := rgba(img, 140, 100); !nearRGBA8(got, mid, 6) {
		t.Errorf("halfway out at scale 2 %v, want %v", got, mid)
	}
	if got := rgba(img, 190, 100); !nearRGBA8(got, [4]uint8{0, 0, 255, 255}, 6) {
		t.Errorf("past the radius at scale 2 %v, want blue", got)
	}
}

// A painter fills a rounded rectangle with a radial gradient, as an
// element's.
func TestFillRadialGradient(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).Draw(func(p *Painter, r Rect) {
			p.FillRadialGradient(Rect{X: 10, Y: 10, W: 80, H: 80}, RadialGradient{
				Stops: []GradientStop{{gRed, 0.5}, {gBlue, 0.5}}, Center: Point{0.25, 0.25}, Radius: 20,
			}, 40)
		})
	}, 100, 100)
	img := tt.Image()
	if got := rgba(img, 32, 30); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("inside half the radius %v, want red", got)
	}
	if got := rgba(img, 30, 60); got != [4]uint8{0, 0, 255, 255} {
		t.Errorf("past half the radius %v, want blue", got)
	}
	if got := rgba(img, 12, 12); got != [4]uint8{255, 255, 255, 255} {
		t.Errorf("outside the rounded corner %v, want the background", got)
	}
}

// A linear gradient with stops shows each where it lies along the line.
func TestLinearGradientStops(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 20).LinearGradient(LinearGradient{Angle: 90, Stops: []GradientStop{{gRed, 0.25}, {gGreen, 0.5}, {gBlue, 0.75}}})
	}, 100, 20)
	img := tt.Image()
	for _, c := range []struct {
		x    int
		want [4]uint8
	}{{5, [4]uint8{255, 0, 0, 255}}, {50, [4]uint8{0, 255, 0, 255}}, {95, [4]uint8{0, 0, 255, 255}}} {
		if got := rgba(img, c.x, 10); !nearRGBA8(got, c.want, 8) {
			t.Errorf("at %d = %v, want %v", c.x, got, c.want)
		}
	}
}

// A border painted with a conic gradient takes its colors around the
// element, over the background, which keeps its own.
func TestBorderConicGradient(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).Background(RGB(255, 255, 255)).Border(8, Color{}).
			BorderConicGradient(ConicGradient{Stops: []GradientStop{{gRed, 0}, {gBlue, 0.5}, {gRed, 1}}})
	}, 100, 100)
	img := tt.Image()
	if got := rgba(img, 50, 96); !nearRGBA8(got, [4]uint8{0, 0, 255, 255}, 10) {
		t.Errorf("the border below %v, want blue", got)
	}
	if got := rgba(img, 50, 3); got[0] < 200 || got[2] > 60 {
		t.Errorf("the border above %v, want red", got)
	}
	if got := rgba(img, 50, 50); got != [4]uint8{255, 255, 255, 255} {
		t.Errorf("the inside %v, want the background", got)
	}
	// A linear gradient of two colors too, from left to right, leaving
	// the inside as what is under it, as outside the element.
	tt = NewTester(func(c *Context) {
		Box(c).Size(100, 100).Border(8, Color{}).BorderLinearGradient(LinearGradient{From: gRed, To: gBlue, Angle: 90})
	}, 120, 120)
	img = tt.Image()
	if l, r := rgba(img, 3, 50), rgba(img, 96, 50); l[0] < 200 || r[2] < 200 {
		t.Errorf("the border's sides %v and %v, want red then blue", l, r)
	}
	if in, out := rgba(img, 50, 50), rgba(img, 110, 110); in != out {
		t.Errorf("the inside %v, want %v, what is under it", in, out)
	}
}

// A blurred element spreads around as a Gaussian of its blur, keeping all
// it draws, and still takes clicks where it is.
func TestBlur(t *testing.T) {
	clicked := false
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Center().Children(func() {
			b := Box(c).Size(40, 40).Background(RGB(0, 0, 0)).Blur(6).Label("Blurred")
			if b.Clicked() {
				clicked = true
			}
		})
	}, 120, 120)
	img := tt.Image()
	// Over white: the darkness the square leaves, all of it, about 40×40.
	total := 0.0
	for y := range 120 {
		for x := range 120 {
			total += 1 - float64(rgba(img, x, y)[0])/255
		}
	}
	if math.Abs(total-1600) > 80 {
		t.Errorf("the blurred square darkens %.0f pixels' worth, want about 1600", total)
	}
	// Its edge at half, its middle nearly black, and the blur reaching out.
	if v := rgba(img, 40, 60)[0]; v < 100 || v > 160 {
		t.Errorf("at the square's edge %d, want about half", v)
	}
	if v := rgba(img, 60, 60)[0]; v > 40 {
		t.Errorf("in the middle %d, want nearly black", v)
	}
	if v := rgba(img, 32, 60)[0]; v == 255 || v < 180 {
		t.Errorf("8 DIPs outside %d, want a little darkened", v)
	}
	if err := tt.Click("Blurred"); err != nil {
		t.Fatal(err)
	}
	if !clicked {
		t.Error("the blurred element took no click")
	}
}

// Paths fill and stroke with a conic gradient around their bounds.
func TestPathConicGradient(t *testing.T) {
	g := ConicGradient{Stops: []GradientStop{{gRed, 0}, {gBlue, 0.5}, {gRed, 1}}}
	tt := NewTester(func(c *Context) {
		Box(c).Size(100, 100).Draw(func(p *Painter, r Rect) {
			var disc Path
			disc.Circle(50, 50, 40)
			p.FillPathConicGradient(&disc, g)
		})
	}, 100, 100)
	img := tt.Image()
	if got := rgba(img, 50, 85); !nearRGBA8(got, [4]uint8{0, 0, 255, 255}, 12) {
		t.Errorf("below the middle %v, want blue", got)
	}
	if got := rgba(img, 50, 15); got[0] < 200 {
		t.Errorf("above the middle %v, want red", got)
	}
}
