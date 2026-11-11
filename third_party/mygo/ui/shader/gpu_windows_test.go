package shader

import (
	"testing"

	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/gpu/gputest"
)

// TestDirect3D checks that Direct3D 11 draws a shader, its uniforms and
// its images, a bitmap and a canvas, as the CPU does.
func TestDirect3D(t *testing.T) {
	s := MustCompile(testSource)
	if _, err := d3d11.CompileEffect(s.fx.HLSL.Source); err != nil {
		t.Fatal(err)
	}
	sc := testScene(s)
	pix, stride, err := d3d11.RenderOffscreen(sc)
	if err != nil {
		t.Skip("no Direct3D:", err)
	}
	gputest.Compare(t, "ui-shader-d3d11", pix, stride, sc)
}

// TestDirect3DBools checks that Direct3D reads bool uniforms as the CPU
// does.
func TestDirect3DBools(t *testing.T) {
	sc := boolScene()
	pix, stride, err := d3d11.RenderOffscreen(sc)
	if err != nil {
		t.Skip("no Direct3D:", err)
	}
	gputest.Compare(t, "ui-shader-d3d11-bools", pix, stride, sc)
}

// TestDirect3DDfdy checks that dfdy grows downward on Direct3D 11.
func TestDirect3DDfdy(t *testing.T) {
	pix, stride, err := d3d11.RenderOffscreen(dfdyScene())
	if err != nil {
		t.Skip("no Direct3D:", err)
	}
	checkDfdy(t, pix, stride)
}

// TestDirect3DMasked checks that Direct3D fills mask glyphs with a shader,
// as text and paths, as the CPU does.
func TestDirect3DMasked(t *testing.T) {
	sc := maskedScene(MustCompile(maskedSource))
	pix, stride, err := d3d11.RenderOffscreen(sc)
	if err != nil {
		t.Skip("no Direct3D:", err)
	}
	gputest.Compare(t, "ui-shader-d3d11-masked", pix, stride, sc)
}

// TestDirect3DEffect checks that Direct3D draws an effect's shader from
// its content, as the CPU does.
func TestDirect3DEffect(t *testing.T) {
	for name, sc := range effectScenes(MustCompile(effectSource)) {
		pix, stride, err := d3d11.RenderOffscreen(sc)
		if err != nil {
			t.Skip("no Direct3D:", err)
		}
		gputest.Compare(t, "ui-shader-d3d11-effect-"+name, pix, stride, sc)
	}
}

// TestDirect3DBackdrop checks that Direct3D gives a shader its backdrop,
// as the CPU does.
func TestDirect3DBackdrop(t *testing.T) {
	for name, sc := range backdropCases() {
		pix, stride, err := d3d11.RenderOffscreen(sc)
		if err != nil {
			t.Skip("no Direct3D:", err)
		}
		gputest.Compare(t, "ui-shader-d3d11-backdrop-"+name, pix, stride, sc)
	}
}
