package raster

import (
	"bytes"
	"image"
	"math/rand/v2"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// referenceClipMask is ClipMask as one coverage test per pixel and clip,
// which ClipMask must match exactly.
func referenceClipMask(clips []scene.ClipShape, area image.Rectangle) []byte {
	w, h := area.Dx(), area.Dy()
	pix := make([]byte, 4*w*h)
	for y := range h {
		for x := range w {
			wx, wy := float32(x+area.Min.X)+0.5, float32(y+area.Min.Y)+0.5
			cov := float32(1)
			for _, c := range clips {
				s := newShape(c.Rect, scene.Corners(c.Rect, c.Radii, c.Continuous))
				inv, ok := c.Transform.Inverse()
				if !ok {
					cov = 0
					break
				}
				cov *= transformedCoverage(&s, c.Transform, inv, wx, wy)
				if cov == 0 {
					break
				}
			}
			pix[4*(y*w+x)+3] = to8(cov)
		}
	}
	return pix
}

func TestClipMaskMatchesPerPixelCoverage(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 255))
	f := func(lo, hi float32) float32 { return lo + rng.Float32()*(hi-lo) }
	shape := func() scene.ClipShape {
		r := scene.Rect{X: f(-20, 120), Y: f(-20, 100), W: f(0, 160), H: f(0, 130)}
		c := scene.ClipShape{Rect: r, Continuous: rng.IntN(2) == 0}
		if rng.IntN(4) > 0 {
			rad := f(0, 40)
			c.Radii = [4]float32{rad, rad * f(0, 1), rad, f(0, 30)}
		}
		switch rng.IntN(5) {
		case 0:
			c.Transform = scene.Translation(f(-30, 30), f(-30, 30)).Mul(scene.Rotation(f(-45, 45)))
		case 1:
			s := f(0.5, 2)
			c.Transform = scene.Scaling(s, s)
		case 2:
			c.Transform = scene.Scaling(0, 1)
		case 3:
			c.Transform = scene.Translation(f(0, 40), f(0, 40))
			c.Transform.G, c.Transform.H = f(-0.004, 0.004), f(-0.004, 0.004)
		}
		return c
	}
	area := image.Rect(-10, -5, 190, 150)
	for i := range 400 {
		clips := make([]scene.ClipShape, 1+rng.IntN(3))
		for j := range clips {
			clips[j] = shape()
		}
		want := referenceClipMask(clips, area)
		got := ClipMask(nil, clips, area)
		if !bytes.Equal(got.Pix, want) {
			for k := range want {
				if got.Pix[k] != want[k] {
					px := k / 4
					t.Fatalf("case %d, %+v: pixel (%d, %d) is %d, want %d", i, clips,
						px%area.Dx()+area.Min.X, px/area.Dx()+area.Min.Y, got.Pix[k], want[k])
				}
			}
		}
	}
}

func TestClipMaskReusesAndResizesItsImage(t *testing.T) {
	clips := []scene.ClipShape{{Rect: scene.Rect{X: 2, Y: 2, W: 40, H: 30}, Radii: [4]float32{8, 8, 8, 8}}}
	img := ClipMask(nil, clips, image.Rect(0, 0, 50, 40))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	again := ClipMask(img, clips, image.Rect(0, 0, 60, 40))
	if again != img || !bytes.Equal(again.Pix, referenceClipMask(clips, image.Rect(0, 0, 60, 40))) {
		t.Fatal("a reused mask kept stale pixels")
	}
}

func BenchmarkClipMaskRoundedCard(b *testing.B) {
	// The rounded card of #255 at 2x, a list row's cell nested in it.
	card := scene.ClipShape{Rect: scene.Rect{X: 32, Y: 136, W: 3400, H: 2000}, Radii: [4]float32{22, 22, 22, 22}}
	cell := scene.ClipShape{Rect: scene.Rect{X: 32, Y: 200, W: 1900, H: 64}}
	for _, bc := range []struct {
		name  string
		clips []scene.ClipShape
		area  image.Rectangle
	}{
		{"card", []scene.ClipShape{card}, image.Rect(32, 136, 3432, 2136)},
		{"cell", []scene.ClipShape{card, cell}, image.Rect(32, 200, 1932, 264)},
	} {
		b.Run(bc.name, func(b *testing.B) {
			img := ClipMask(nil, bc.clips, bc.area)
			for b.Loop() {
				ClipMask(img, bc.clips, bc.area)
			}
		})
	}
}
