package main

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

var (
	//go:embed palette.kage
	paletteSource string
	//go:embed orb.kage
	orbSource string
	//go:embed ring.kage
	ringSource string
	//go:embed shimmer.kage
	shimmerSource string
)

// The shaders, those that glow with palette.kage's colors after it.
var (
	orb     = shader.MustCompile("package main\n" + paletteSource + orbSource)
	ring    = shader.MustCompile("package main\n" + paletteSource + ringSource)
	shimmer = shader.MustCompile("package main\n" + shimmerSource)
)

// The shaders' uniforms, looked up once, and their values, set each frame.
var (
	orbPhase, orbEnergy                         = orb.Uniform("Phase"), orb.Uniform("Energy")
	ringTime, ringStrength, ringPad, ringRadius = ring.Uniform("Time"), ring.Uniform("Strength"), ring.Uniform("Pad"), ring.Uniform("Radius")
	shimmerTime, shimmerBase, shimmerShine      = shimmer.Uniform("Time"), shimmer.Uniform("Base"), shimmer.Uniform("Shine")
)

// shaders holds a window's values of the shaders' uniforms.
type shaders struct {
	orb, still, ring, shimmer *shader.Values
}

func newShaders() shaders {
	return shaders{orb.NewValues(), orb.NewValues(), ring.NewValues(), shimmer.NewValues()}
}

// drawOrb draws the orb in r, or, where the window draws without a GPU, a
// plain disc of its colors.
func (a *app) drawOrb(p *ui.Painter, r ui.Rect) {
	if !p.Accelerated() {
		s := min(r.W, r.H) / 1.25
		p.Fill(ui.Rect{X: r.X + (r.W-s)/2, Y: r.Y + (r.H-s)/2, W: s, H: s}, ui.Hex("#AA6EEE"), s/2)
		return
	}
	orb.Draw(p, r, shader.Options{Values: a.sh.orb.Float(orbPhase, float32(a.orbPhase)).Float(orbEnergy, float32(a.energy))})
}
