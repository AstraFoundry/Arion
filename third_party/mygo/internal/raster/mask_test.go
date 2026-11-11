package raster

import (
	"bytes"
	"image"
	"math/rand/v2"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// A clip masked by a gradient lets through, at each pixel, the alpha of
// the gradient there: all of it where the gradient is opaque, none where
// it is transparent, half halfway, of everything it clips; outside its
// rectangle, nothing.
func TestClipMaskedByGradient(t *testing.T) {
	white, red := scene.Color{R: 255, G: 255, B: 255, A: 255}, scene.Color{R: 255, A: 255}
	black := scene.Color{A: 255}
	for _, c := range []struct {
		name string
		clip scene.Op
	}{
		{"linear", scene.Op{Paint: scene.PaintLinear, Color: black, Gradient: [4]float32{20, 0, 120, 0}}},
		{"linear stops", scene.Op{Paint: scene.PaintLinearStops, Gradient: [4]float32{20, 0, 120, 0}, Stops: [2]int32{0, 2}}},
		{"linear stops, Oklab", scene.Op{Paint: scene.PaintLinearStopsOklab, Gradient: [4]float32{20, 0, 120, 0}, Stops: [2]int32{0, 2}}},
	} {
		s := &scene.Scene{Width: 140, Height: 40, Clear: white,
			Stops: []scene.Stop{{Color: black, At: 0}, {Color: scene.Color{}, At: 1}}}
		clip := c.clip
		clip.Kind, clip.Rect = scene.OpPushClip, scene.Rect{X: 10, Y: 10, W: 120, H: 20}
		s.Ops = []scene.Op{clip,
			{Kind: scene.OpFill, Rect: scene.Rect{W: 140, H: 40}, Color: red},
			{Kind: scene.OpPopClip}}
		dst := NewImage(s.Width, s.Height)
		Render(dst, s)
		writePNG(t, "gradient-mask", dst)
		for _, w := range []struct {
			x, y int
			g    byte // how much green and blue are left of the white
		}{
			{5, 20, 255}, {12, 20, 0}, {19, 20, 0}, {69, 20, 127}, {119, 20, 254}, {125, 20, 255}, {69, 5, 255}, {69, 35, 255},
		} {
			if p := pixel(dst, w.x, w.y); p[0] != 255 || !near(p[1], w.g) || p[1] != p[2] {
				t.Errorf("%s: pixel (%d, %d) is %v, want green and blue %d", c.name, w.x, w.y, p, w.g)
			}
		}
	}
}

// A clip masked by an image, as a path's coverage, lets through the
// image's alpha, bilinearly sampled where a transform maps it, and nothing
// outside the image.
func TestClipMaskedByImage(t *testing.T) {
	const n = 8
	pix := make([]byte, 4*n*n)
	for y := range n {
		for x := range n {
			if x >= 2 && x < 6 && y >= 2 && y < 6 {
				pix[4*(y*n+x)+3] = 255
			}
		}
	}
	img := scene.NewImageRGBA(n, n, pix)
	s := &scene.Scene{Width: 40, Height: 40, Clear: scene.Color{}}
	s.Ops = []scene.Op{
		{Kind: scene.OpPushClip, Rect: scene.Rect{X: 10, Y: 10, W: n, H: n}, Image: img, Src: scene.Rect{W: n, H: n}},
		{Kind: scene.OpFill, Rect: scene.Rect{W: 40, H: 40}, Color: scene.Color{B: 255, A: 255}},
		{Kind: scene.OpPopClip},
	}
	dst := NewImage(s.Width, s.Height)
	Render(dst, s)
	for y := range 40 {
		for x := range 40 {
			want := byte(0)
			if x >= 12 && x < 16 && y >= 12 && y < 16 {
				want = 255
			}
			if p := pixel(dst, x, y); p[3] != want || p[2] != want {
				t.Fatalf("pixel (%d, %d) is %v, want blue of alpha %d", x, y, p, want)
			}
		}
	}
	// Twice as large: the image's pixels sampled between them.
	s.Ops[0].Rect = scene.Rect{X: 10, Y: 10, W: 2 * n, H: 2 * n}
	Render(dst, s)
	if p := pixel(dst, 19, 19); p[3] != 255 {
		t.Errorf("inside the image's square, scaled: %v", p)
	}
	if p := pixel(dst, 12, 19); p[3] != 0 {
		t.Errorf("outside the image's square, scaled: %v", p)
	}
	if p := pixel(dst, 13, 19); p[3] == 0 || p[3] == 255 {
		t.Errorf("on the edge of the image's square, scaled: %v, want part of it", p)
	}
}

// A masked clip only moved by its transform draws as the clip moved by
// hand, its gradient's points with it, as renderers fold it
// (scene.TranslationFolder) or not.
func TestMovedMaskedClip(t *testing.T) {
	scene.FoldTranslations()
	black := scene.Color{A: 255}
	build := func(dx, dy float32, m scene.Affine) *scene.Scene {
		s := &scene.Scene{Width: 80, Height: 60, Clear: scene.Color{R: 255, G: 255, B: 255, A: 255}}
		s.Ops = []scene.Op{
			{Kind: scene.OpPushClip, Rect: scene.Rect{X: 10 + dx, Y: 10 + dy, W: 50, H: 30}, Radii: scene.Uniform(6), Paint: scene.PaintLinear,
				Color: black, Gradient: [4]float32{10 + dx, 10 + dy, 60 + dx, 40 + dy}, Transform: m},
			{Kind: scene.OpFill, Rect: scene.Rect{W: 80, H: 60}, Color: scene.Color{G: 128, A: 255}},
			{Kind: scene.OpPopClip},
		}
		return s
	}
	want := NewImage(80, 60)
	Render(want, build(7, 5, scene.Affine{}))
	got := NewImage(80, 60)
	Render(got, build(0, 0, scene.Translation(7, 5)))
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Error("a masked clip translated draws apart from the same clip moved")
	}
}

// Where a clip lets a part through, a fill's border goes over the fill
// before both are blended, as the shaders do: the fill does not show
// through an opaque border.
func TestBorderOverFillUnderMask(t *testing.T) {
	s := &scene.Scene{Width: 40, Height: 40, Clear: scene.Color{R: 255, G: 255, B: 255, A: 255}}
	s.Ops = []scene.Op{
		{Kind: scene.OpPushClip, Rect: scene.Rect{W: 40, H: 40}, Paint: scene.PaintLinear,
			Color: scene.Color{A: 128}, Color2: scene.Color{A: 128}, Gradient: [4]float32{0, 0, 40, 0}},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 5, Y: 5, W: 30, H: 30}, Color: scene.Color{R: 255, A: 255},
			Border: scene.Uniform(4), BorderColor: scene.Color{B: 255, A: 255}},
		{Kind: scene.OpPopClip},
	}
	dst := NewImage(40, 40)
	Render(dst, s)
	// Half blue over white, without red.
	if p := pixel(dst, 6, 20); p[0] != p[1] || !near(p[1], 127) || p[2] != 255 {
		t.Errorf("on the border: %v, want half blue over white", p)
	}
}

// ClipMask, which the GPU's renderers sample, lets through what the CPU
// renderer's clips do, masks of gradients and images too, exactly.
func TestClipMaskMatchesMaskedClips(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 1100))
	f := func(lo, hi float32) float32 { return lo + rng.Float32()*(hi-lo) }
	const n = 16
	pix := make([]byte, 4*n*n)
	for i := range n * n {
		pix[4*i+3] = byte(rng.IntN(256))
	}
	img := scene.NewImageRGBA(n, n, pix)
	stops := []scene.Stop{{Color: scene.Color{A: 255}, At: 0.2}, {Color: scene.Color{A: 40}, At: 0.5}, {Color: scene.Color{}, At: 0.9}}
	for i := range 200 {
		op := scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{X: f(-10, 60), Y: f(-10, 50), W: f(0, 80), H: f(0, 70)}}
		switch rng.IntN(4) {
		case 0:
			op.Paint, op.Color, op.Color2 = scene.PaintLinear, scene.Color{A: 255}, scene.Color{A: byte(rng.IntN(256))}
			op.Gradient = [4]float32{f(0, 100), f(0, 80), f(0, 100), f(0, 80)}
		case 1:
			op.Paint, op.Stops, op.Gradient = scene.PaintRadial, [2]int32{0, 3}, [4]float32{f(0, 100), f(0, 80), f(1, 60), 0}
		case 2:
			op.Paint, op.Stops, op.Gradient = scene.PaintConicOklab, [2]int32{0, 3}, [4]float32{f(0, 100), f(0, 80), f(0, 6), 0}
		case 3:
			op.Image, op.Src = img, scene.Rect{W: n, H: n}
		}
		switch rng.IntN(3) {
		case 0:
			op.Transform = scene.Translation(f(-20, 20), f(-20, 20)).Mul(scene.Rotation(f(-30, 30)))
		case 1:
			op.Transform = scene.Translation(f(-5, 5), f(-5, 5))
		}
		s := &scene.Scene{Width: 100, Height: 80, Stops: stops, Ops: []scene.Op{op,
			{Kind: scene.OpFill, Rect: scene.Rect{W: 100, H: 80}, Color: scene.Color{A: 255}}, {Kind: scene.OpPopClip}}}
		dst := NewImage(s.Width, s.Height)
		Render(dst, s)
		shape := scene.ClipShapeOf(&op, stops)
		mask := ClipMask(nil, []scene.ClipShape{shape}, image.Rect(0, 0, s.Width, s.Height))
		for k := 3; k < len(mask.Pix); k += 4 {
			if d := int(mask.Pix[k]) - int(dst.Pix[k]); d < -1 || d > 1 {
				t.Fatalf("case %d, %+v: pixel (%d, %d) is %d in the mask, %d drawn", i, op, k/4%s.Width, k/4/s.Width, mask.Pix[k], dst.Pix[k])
			}
		}
	}
}
