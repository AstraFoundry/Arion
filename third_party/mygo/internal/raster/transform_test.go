package raster

import (
	"bytes"
	"github.com/egoist/mygo/internal/scene"
	"testing"
)

func TestTransformedSceneDamageAndNestedClips(t *testing.T) {
	s := &scene.Scene{Width: 150, Height: 100}
	m := scene.Translation(70, 20).Mul(scene.Rotation(90))
	s.Ops = []scene.Op{
		{Kind: scene.OpPushClip, Rect: scene.Rect{W: 60, H: 40}, Transform: m},
		{Kind: scene.OpPushClip, Rect: scene.Rect{X: 5, Y: 5, W: 30, H: 20}, Transform: m, Radii: [4]float32{4, 4, 4, 4}},
		{Kind: scene.OpFill, Rect: scene.Rect{W: 100, H: 100}, Color: scene.Color{R: 255, A: 255}, Transform: m},
		{Kind: scene.OpPopClip}, {Kind: scene.OpPopClip},
	}
	var r Renderer
	defer r.Release()
	r.Render(s)
	p := r.Image.Pix[35*r.Image.Stride+4*55:][:4]
	if p[2] != 255 || p[3] != 255 {
		t.Fatalf("transformed clip interior: %v", p)
	}
	p = r.Image.Pix[30*r.Image.Stride+4*35:][:4]
	if p[3] != 0 {
		t.Fatal("paint escaped rotated clips")
	}
	if damage := r.Render(s); len(damage) != 0 {
		t.Fatalf("unchanged scene damaged %v", damage)
	}
	for i := range s.Ops {
		s.Ops[i].Transform = scene.Translation(20, 0).Mul(s.Ops[i].Transform)
	}
	r.Render(s)
	want := NewImage(s.Width, s.Height)
	Render(want, s)
	if !bytes.Equal(r.Image.Pix, want.Pix) {
		t.Fatal("incremental transformed drawing differs from full drawing")
	}
}

func TestTransformedThinBorderGradientAndImage(t *testing.T) {
	s := &scene.Scene{Width: 100, Height: 100}
	s.Ops = []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 10, H: 10}, Color: scene.Color{R: 255, A: 255}, Color2: scene.Color{B: 255, A: 255}, Paint: scene.PaintLinear, Gradient: [4]float32{0, 0, 10, 0}, Border: scene.Uniform(1), BorderColor: scene.Color{G: 255, A: 255}, Transform: scene.Translation(20, 20).Mul(scene.Scaling(3, 2))}}
	m := NewImage(100, 100)
	Render(m, s)
	at := func(x, y int) []byte { return m.Pix[y*m.Stride+4*x:][:4] }
	if at(21, 30)[1] != 255 {
		t.Fatalf("scaled border: %v", at(21, 30))
	}
	if p := at(35, 30); p[0] < 100 || p[2] < 100 || p[1] != 0 {
		t.Fatalf("gradient not sampled locally: %v", p)
	}
	img := scene.NewImageRGBA(2, 1, []byte{255, 0, 0, 255, 0, 0, 255, 255})
	s.Ops = []scene.Op{{Kind: scene.OpImage, Rect: scene.Rect{W: 20, H: 10}, Src: scene.Rect{W: 2, H: 1}, Image: img, Transform: scene.Translation(50, 20).Mul(scene.Rotation(90))}}
	Render(m, s)
	if at(45, 21)[2] < 240 || at(45, 38)[0] < 240 {
		t.Fatalf("rotated image colors: %v %v", at(45, 21), at(45, 38))
	}
}
