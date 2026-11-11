//go:build amd64 || arm64

package interp_test

import (
	"regexp"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gl"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/shaderir/glsl"
	"github.com/egoist/mygo/internal/shaderir/interp"
)

// TestOpenGL draws a shader with OpenGL 3.3 and OpenGL ES 3.0, its GLSL
// function wrapped in an effect, and checks that the interpreter computes
// the same pixels.
func TestOpenGL(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	p, prog := compile(t, noiseSource)
	words := noiseUniforms(p)
	code := constUniforms(glsl.Fragment(p, "shaderFragment", glsl.Options{FlipY: "true"}),
		regexp.MustCompile(`(?m)^uniform ((?:\w+) (U\d+)(?:\[\d+\])?);$`), p, words, false)
	fx := &scene.Effect{
		Name: "shader",
		GLSL: code + `
vec4 effect(vec2 p, Effect e) {
	return shaderFragment(vec4(p, 0.0, 1.0), p - e.rect.xy);
}`,
		Pixels: func() scene.EffectPixels { return &shaderPixels{prog: prog, in: &interp.Inputs{Uniforms: words}} },
	}
	for _, es := range []bool{false, true} {
		s := effectScene(fx)
		o, err := gl.NewOffscreen(es, s.Width, s.Height)
		if err != nil {
			t.Skip(err)
		}
		t.Log(o.Info())
		pix, err := o.Render(s)
		o.Release()
		if err != nil {
			t.Fatal(err)
		}
		name := "shader-gl"
		if es {
			name += "-es"
		}
		gputest.Compare(t, name, pix, s.Width*4, s)
	}
}
