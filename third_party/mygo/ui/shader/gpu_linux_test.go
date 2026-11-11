//go:build amd64 || arm64

package shader

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gl"
	"github.com/egoist/mygo/internal/gpu/gputest"
)

// TestOpenGL checks that OpenGL 3.3 and OpenGL ES 3.0 draw a shader, its
// uniforms and its images, a bitmap and a canvas, as the CPU does.
func TestOpenGL(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := MustCompile(testSource)
	for _, es := range []bool{false, true} {
		sc := testScene(s)
		o, err := gl.NewOffscreen(es, sc.Width, sc.Height)
		if err != nil {
			t.Skip(err)
		}
		t.Log(o.Info())
		pix, err := o.Render(sc)
		o.Release()
		if err != nil {
			t.Fatal(err)
		}
		name := "ui-shader-gl"
		if es {
			name += "-es"
		}
		gputest.Compare(t, name, pix, sc.Width*4, sc)
	}
}

// TestOpenGLBools checks that OpenGL reads bool uniforms as the CPU does.
func TestOpenGLBools(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	sc := boolScene()
	o, err := gl.NewOffscreen(false, sc.Width, sc.Height)
	if err != nil {
		t.Skip(err)
	}
	defer o.Release()
	pix, err := o.Render(sc)
	if err != nil {
		t.Fatal(err)
	}
	gputest.Compare(t, "ui-shader-gl-bools", pix, sc.Width*4, sc)
}

// TestOpenGLDfdy checks that dfdy grows downward on OpenGL, in a canvas
// and in the frame, whose rows go up.
func TestOpenGLDfdy(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, es := range []bool{false, true} {
		sc := dfdyScene()
		o, err := gl.NewOffscreen(es, sc.Width, sc.Height)
		if err != nil {
			t.Skip(err)
		}
		pix, err := o.Render(sc)
		o.Release()
		if err != nil {
			t.Fatal(err)
		}
		checkDfdy(t, pix, sc.Width*4)
	}
}

// TestOpenGLMasked checks that OpenGL fills mask glyphs with a shader, as
// text and paths, as the CPU does.
func TestOpenGLMasked(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, es := range []bool{false, true} {
		sc := maskedScene(MustCompile(maskedSource))
		o, err := gl.NewOffscreen(es, sc.Width, sc.Height)
		if err != nil {
			t.Skip(err)
		}
		pix, err := o.Render(sc)
		o.Release()
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "ui-shader-gl-masked", pix, sc.Width*4, sc)
	}
}

// TestOpenGLEffect checks that OpenGL draws an effect's shader from its
// content, as the CPU does.
func TestOpenGLEffect(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, es := range []bool{false, true} {
		for name, sc := range effectScenes(MustCompile(effectSource)) {
			o, err := gl.NewOffscreen(es, sc.Width, sc.Height)
			if err != nil {
				t.Skip(err)
			}
			pix, err := o.Render(sc)
			o.Release()
			if err != nil {
				t.Fatal(err)
			}
			gputest.Compare(t, "ui-shader-gl-effect-"+name, pix, sc.Width*4, sc)
		}
	}
}

// TestOpenGLBackdrop checks that OpenGL gives a shader its backdrop, as
// the CPU does.
func TestOpenGLBackdrop(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, es := range []bool{false, true} {
		for name, sc := range backdropCases() {
			o, err := gl.NewOffscreen(es, sc.Width, sc.Height)
			if err != nil {
				t.Skip(err)
			}
			pix, err := o.Render(sc)
			o.Release()
			if err != nil {
				t.Fatal(err)
			}
			gputest.Compare(t, "ui-shader-gl-backdrop-"+name, pix, sc.Width*4, sc)
		}
	}
}
