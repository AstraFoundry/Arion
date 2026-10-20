package gputest

import (
	"math"

	"github.com/egoist/mygo/internal/scene"
)

// TransformedScene exercises all primitives and effects under affine
// composition, with nested transformed clips and an inline glyph transform.
func TransformedScene() *scene.Scene {
	s := Scene()
	s.Clear = scene.Color{R: 100, G: 120, B: 160, A: 80}
	m := scene.Translation(25, 20).Mul(scene.Rotation(4)).Mul(scene.Scaling(0.8, 0.9))
	for i := range s.Ops {
		s.Ops[i].Transform = m
	}
	if len(s.Glyphs) > 0 {
		s.Glyphs[0].Transform = scene.Rotation(15)
	}
	return s
}

// PerspectiveScene exercises all primitives and effects seen in
// perspective: the scene turned 20 degrees about its vertical center line,
// seen from 900 pixels, as RotationY and Perspective make it, with a
// skewed glyph and a nested clip turned the other way.
func PerspectiveScene() *scene.Scene {
	s := Scene()
	s.Clear = scene.Color{R: 100, G: 120, B: 160, A: 80}
	turn := func(degrees, d, cx, cy float32) scene.Affine {
		sin, cos := math.Sincos(float64(degrees) * math.Pi / 180)
		m := scene.Projective(float32(cos), 0, 0, 1, 0, 0, float32(sin)/d, 0)
		return scene.Translation(cx, cy).Mul(m).Mul(scene.Translation(-cx, -cy))
	}
	m := turn(20, 900, float32(s.Width)/2, float32(s.Height)/2)
	for i := range s.Ops {
		s.Ops[i].Transform = m
	}
	if len(s.Glyphs) > 0 {
		s.Glyphs[0].Transform = scene.Matrix(1, 0, 0.3, 1, 0, 0)
	}
	return s
}
