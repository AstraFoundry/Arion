package shader

import (
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/gpu/metal"
)

// TestMetal checks that Metal draws a shader, its uniforms and its
// images, a bitmap and a canvas, as the CPU does.
func TestMetal(t *testing.T) {
	s := MustCompile(testSource)
	sc := testScene(s)
	pix, err := metal.RenderOffscreen(sc)
	if err != nil {
		t.Skip("no Metal:", err)
	}
	gputest.Compare(t, "ui-shader-metal", pix, sc.Width*4, sc)
}

// TestMetalBools checks that Metal reads bools at the end of its struct of
// uniforms, a byte each.
func TestMetalBools(t *testing.T) {
	sc := boolScene()
	pix, err := metal.RenderOffscreen(sc)
	if err != nil {
		t.Skip("no Metal:", err)
	}
	gputest.Compare(t, "ui-shader-metal-bools", pix, sc.Width*4, sc)
}

// TestMetalDfdy checks that dfdy grows downward on Metal.
func TestMetalDfdy(t *testing.T) {
	pix, err := metal.RenderOffscreen(dfdyScene())
	if err != nil {
		t.Skip("no Metal:", err)
	}
	checkDfdy(t, pix, 40*4)
}

// TestMetalMasked checks that Metal fills mask glyphs with a shader, as
// text and paths, as the CPU does.
func TestMetalMasked(t *testing.T) {
	sc := maskedScene(MustCompile(maskedSource))
	pix, err := metal.RenderOffscreen(sc)
	if err != nil {
		t.Skip("no Metal:", err)
	}
	gputest.Compare(t, "ui-shader-metal-masked", pix, sc.Width*4, sc)
}

// TestMetalEffect checks that Metal draws an effect's shader from its
// content, as the CPU does.
func TestMetalEffect(t *testing.T) {
	for name, sc := range effectScenes(MustCompile(effectSource)) {
		pix, err := metal.RenderOffscreen(sc)
		if err != nil {
			t.Skip("no Metal:", err)
		}
		gputest.Compare(t, "ui-shader-metal-effect-"+name, pix, sc.Width*4, sc)
	}
}

// TestMetalBackdropEndsAtDrawn checks that Metal ends a backdrop at the
// edge of what a frame draws (Scene.Drawn), as the CPU does.
func TestMetalBackdropEndsAtDrawn(t *testing.T) {
	for name, scs := range edgeBackdrops() {
		sc := scs[1]
		pix, err := metal.RenderOffscreen(sc)
		if err != nil {
			t.Skip("no Metal:", err)
		}
		gputest.Compare(t, "ui-shader-metal-backdrop-drawn-"+name, pix, sc.Width*4, sc)
	}
}

// TestMetalBackdrop checks that Metal gives a shader its backdrop, as the
// CPU does.
func TestMetalBackdrop(t *testing.T) {
	for name, sc := range backdropCases() {
		pix, err := metal.RenderOffscreen(sc)
		if err != nil {
			t.Skip("no Metal:", err)
		}
		gputest.Compare(t, "ui-shader-metal-backdrop-"+name, pix, sc.Width*4, sc)
	}
}
