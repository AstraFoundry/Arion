package shader

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/shader"
	"github.com/egoist/mygo/ui"
)

// A color outside sRGB sets its uniform in extended sRGB for a wide
// gamut, beside the nearest sRGB color; one inside sets it once.
func TestWideColors(t *testing.T) {
	s := MustCompile("package main\nvar Tint vec4\nfunc Fragment() vec4 { return Tint }\n")
	tint := s.Uniform("Tint")
	vivid := ui.Oklch(0.7, 0.3, 30) // beyond sRGB's reds
	var narrow, wide ui.ShaderInput
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			narrow = s.input(p, r, r, Options{Values: s.NewValues().Color(tint, ui.RGB(255, 0, 0))}, ui.Point{})
			wide = s.input(p, r, r, Options{Uniforms: map[string]any{"Tint": vivid}}, ui.Point{})
		})
	}
	ui.NewTester(view, 10, 10).Image()
	if narrow.WideUniforms != nil {
		t.Errorf("an sRGB color has wide words")
	}
	if wide.WideUniforms == nil {
		t.Fatalf("a color outside sRGB has no wide words")
	}
	w := tint.u.word
	srgbRed := math.Float32frombits(wide.Uniforms[w])
	wideRed, wideGreen := math.Float32frombits(wide.WideUniforms[w]), math.Float32frombits(wide.WideUniforms[w+1])
	if srgbRed > 1 || wideRed <= 1 && wideGreen >= 0 {
		t.Errorf("red %v in sRGB, %v %v in extended sRGB: want the latter outside 0 to 1", srgbRed, wideRed, wideGreen)
	}
	for i := range wide.Uniforms {
		if (i < w || i >= w+4) && wide.Uniforms[i] != wide.WideUniforms[i] {
			t.Errorf("word %d differs for a wide gamut", i)
		}
	}
}

// The renderers bind the backdrop where the bridge reads it.
func TestBackdropImage(t *testing.T) {
	if gpu.BackdropImage != shader.BackdropImage {
		t.Fatalf("renderers bind the backdrop as image %d, shaders read image %d", gpu.BackdropImage, shader.BackdropImage)
	}
}
