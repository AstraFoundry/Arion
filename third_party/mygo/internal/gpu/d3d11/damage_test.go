//go:build windows

package d3d11

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// changingScenes calls frame with the steps of random scenes changing a
// little at a time, over an opaque color and a translucent one, each with
// a renderer drawing in part and one drawing whole, both new: each scene
// has an atlas of its own.
func changingScenes(t *testing.T, seeds uint64, frame func(r, whole *Renderer, s *scene.Scene, step int, what string)) {
	for seed := range seeds {
		m := gputest.NewChanging(seed)
		if seed%3 == 2 {
			m.Clear = scene.Color{R: 20, G: 40, B: 200, A: 77}
		}
		s := m.Scene()
		r, err := New(hiddenWindow(t, s.Width, s.Height))
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		if r.ctx1 == 0 {
			r.Release()
			t.Skip("no Direct3D 11.1")
		}
		whole, err := New(hiddenWindow(t, s.Width, s.Height))
		if err != nil {
			t.Fatal(err)
		}
		whole.ctx1 = 0 // never in part
		// No subtests: the windows take their messages on this thread.
		for step := range 30 {
			what := fmt.Sprintf("seed %d", seed)
			if step > 0 {
				what += ", " + m.Change(s)
			}
			frame(r, whole, s, step, what)
		}
		r.Release()
		whole.Release()
	}
}

// TestPartialFramesMatchWhole draws random scenes changing a little at a
// time into a back buffer that keeps what was drawn, as DXGI completes
// one presented in part with the last frame, each frame only where it
// changed, and requires the bytes of each frame drawn whole.
func TestPartialFramesMatchWhole(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	seeds := uint64(20)
	if testing.Short() {
		seeds = 5
	}
	frames, partial := 0, 0
	changingScenes(t, seeds, func(r, whole *Renderer, s *scene.Scene, step int, what string) {
		drawn, err := r.drawFrame(s)
		if err != nil {
			t.Fatal(err)
		}
		r.held = drawn || r.held // as presenting it does
		frames++
		part := "whole"
		switch r.redrew {
		case redrewPart:
			partial++
			part = r.part.String()
		case redrewNothing:
			part = "nothing"
		}
		got, _ := r.readBack(t)
		if err := whole.draw(s); err != nil {
			t.Fatal(err)
		}
		want, _ := whole.readBack(t)
		if !bytes.Equal(got, want) {
			t.Fatalf("step %d (%s), redrew %s: %d bytes differ from the frame drawn whole", step, what, part, differing(got, want))
		}
	})
	if partial < frames/3 {
		t.Errorf("%d frames of %d drawn in part", partial, frames)
	}
}

// TestPresentsInPart presents random scenes changing a little at a time,
// each drawn only where it changed, with that as the dirty rectangle, in a
// window that shows: DXGI copies the rest from the last frame into the
// buffer presented. Once the swap chain flipped again, that buffer, back,
// must hold the frame drawn whole.
func TestPresentsInPart(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	seeds := uint64(6)
	if testing.Short() {
		seeds = 2
	}
	// presented holds the frames presented, drawn whole.
	var presented [][]byte
	partial := 0
	changingScenes(t, seeds, func(r, whole *Renderer, s *scene.Scene, step int, what string) {
		if step == 0 {
			const swShowNoActivate = 4
			user32.NewProc("ShowWindow").Call(r.hwnd, swShowNoActivate)
			presented = presented[:0]
		}
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
		switch r.redrew {
		case redrewNothing:
			return // nothing presented: the swap chain did not flip
		case redrewPart:
			partial++
		}
		if err := whole.draw(s); err != nil {
			t.Fatal(err)
		}
		want, _ := whole.readBack(t)
		presented = append(presented, want)
		// The back buffer now holds the frame presented before, as DXGI
		// completed it.
		if n := len(presented); n >= 2 {
			if got, _ := r.readBack(t); !bytes.Equal(got, presented[n-2]) {
				t.Fatalf("step %d (%s): the frame before, presented, differs in %d bytes from it drawn whole", step, what, differing(got, presented[n-2]))
			}
		}
	})
	if partial == 0 {
		t.Error("no frame was presented in part")
	}
}

func differing(a, b []byte) (n int) {
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}
