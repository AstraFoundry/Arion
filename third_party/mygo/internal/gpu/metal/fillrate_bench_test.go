//go:build darwin

package metal

import (
	"github.com/egoist/mygo/internal/scene"
	"testing"
)

// BenchmarkRenderFillRate measures GPU timestamp duration for large opaque
// layers, border-only rings and stacked shadows at Retina window size.
// Like BenchmarkRenderSprites, it waits for each offscreen frame; it does
// not measure drawable waits, presentation latency or frames in flight.
func BenchmarkRenderFillRate(b *testing.B) {
	const w, h = 3456, 2000
	fill := func(r scene.Rect, c scene.Color) scene.Op { return scene.Op{Kind: scene.OpFill, Rect: r, Color: c} }
	background := fill(scene.Rect{W: w, H: h}, scene.Color{R: 20, G: 22, B: 28, A: 255})
	var layers []scene.Op
	for i := range 6 {
		d := float32(i)
		layers = append(layers, fill(scene.Rect{X: d, Y: d, W: w - 2*d, H: h - 2*d}, scene.Color{R: byte(20 + i), G: 22, B: 28, A: 255}))
	}
	panel := scene.Rect{X: 576, Y: 48, W: 2800, H: 1856}
	radii := scene.Uniform(24)
	card := fill(panel, scene.Color{R: 30, G: 32, B: 38, A: 255})
	card.Radii = radii
	card.Continuous = true
	ring := fill(panel, scene.Color{})
	ring.Radii = radii
	ring.Continuous = true
	ring.Border = scene.Uniform(2)
	ring.BorderColor = scene.Color{A: 100}
	shadows := []scene.Op{background, fill(scene.Rect{W: w, H: h}, scene.Color{A: 100})}
	for _, blur := range []float32{6, 16, 32, 64} {
		shadows = append(shadows, scene.Op{Kind: scene.OpShadow, Rect: panel, Radii: radii, Cast: panel, CastRadii: radii, Blur: blur, Color: scene.Color{A: 60}})
	}
	shadows = append(shadows, card)
	for _, tc := range []struct {
		name string
		ops  []scene.Op
	}{
		{"nested_layers", layers}, {"border_ring", []scene.Op{background, card, ring}}, {"shadow_panel", shadows},
	} {
		b.Run(tc.name, func(b *testing.B) {
			benchRender(b, &scene.Scene{Width: w, Height: h, Clear: scene.Color{A: 255}, Ops: tc.ops})
		})
	}
}
