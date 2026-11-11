package interp_test

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/gpu/metal"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/shaderir"
	"github.com/egoist/mygo/internal/shaderir/interp"
	"github.com/egoist/mygo/internal/shaderir/msl"
)

// TestMetal draws a shader with the Metal renderer, its Metal function
// wrapped in an effect, and checks that the interpreter computes the same
// pixels.
func TestMetal(t *testing.T) {
	p, prog := compile(t, noiseSource)
	words := noiseUniforms(p)
	fx := &scene.Effect{
		Name:   "shader",
		Metal:  scene.EffectCode{Source: metalEffect(p, words)},
		Pixels: func() scene.EffectPixels { return &shaderPixels{prog: prog, in: &interp.Inputs{Uniforms: words}} },
	}
	s := effectScene(fx)
	pix, err := metal.RenderOffscreen(s)
	if err != nil {
		t.Skip("no Metal:", err)
	}
	gputest.Compare(t, "shader-metal", pix, s.Width*4, s)
}

// metalEffect returns an effect's Metal source calling p's fragment
// function, with the uniform values words laid out as Metal lays out its
// struct of uniforms, in the source.
func metalEffect(p *shaderir.Program, words []uint32) string {
	at, size := msl.UniformLanes(p)
	buf := make([]byte, (size+15)/16*16)
	for k, w := range words {
		binary.LittleEndian.PutUint32(buf[at[k]:], w)
	}
	var lanes []string
	for i := 0; i < len(buf); i += 4 {
		lanes = append(lanes, fmt.Sprintf("%du", binary.LittleEndian.Uint32(buf[i:])))
	}
	return msl.Fragment(p, "shaderFragment") + fmt.Sprintf(`
constant uint shaderUniforms[%d] = {%s};

float4 effect(float2 p, Effect e, texture2d<float> backdrop) {
	constant Uniforms& u = *(constant Uniforms*)shaderUniforms;
	return shaderFragment(u, backdrop, backdrop, backdrop, backdrop, float4(p, 0.0f, 1.0f), p - e.rect.xy);
}
`, len(lanes), strings.Join(lanes, ", "))
}
