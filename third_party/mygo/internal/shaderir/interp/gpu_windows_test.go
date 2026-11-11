package interp_test

import (
	"regexp"
	"testing"

	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/shaderir/hlsl"
	"github.com/egoist/mygo/internal/shaderir/interp"
)

// TestDirect3D draws a shader with Direct3D 11, its HLSL function wrapped
// in an effect, and checks that the interpreter computes the same pixels.
func TestDirect3D(t *testing.T) {
	p, prog := compile(t, noiseSource)
	words := noiseUniforms(p)
	// The uniforms' constants in place of the cbuffer's members.
	code := hlsl.Fragment(p, "shaderFragment", 2, 6)
	code = regexp.MustCompile(`(?s)cbuffer Uniforms : register\(b\d+\) \{\n(.*?)\n\}`).ReplaceAllString(code, "$1")
	code = constUniforms(code, regexp.MustCompile(`(?m)^\t((?:\w+) (U\d+)(?:\[\d+\])?) : packoffset\(\w+(?:\.\w)?\);$`), p, words, true)
	src := code + `
float4 effect(float2 p, Effect e) {
	return shaderFragment(float4(p, 0, 1), p - e.rect.xy);
}`
	if _, err := d3d11.CompileEffect(src); err != nil {
		t.Fatal(err)
	}
	fx := &scene.Effect{
		Name:   "shader",
		HLSL:   scene.EffectCode{Source: src},
		Pixels: func() scene.EffectPixels { return &shaderPixels{prog: prog, in: &interp.Inputs{Uniforms: words}} },
	}
	s := effectScene(fx)
	pix, stride, err := d3d11.RenderOffscreen(s)
	if err != nil {
		t.Skip("no Direct3D:", err)
	}
	gputest.Compare(t, "shader-d3d11", pix, stride, s)
}
