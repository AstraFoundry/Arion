//go:build windows

package d3d11

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// Canvases draw into textures as the CPU renderer draws them into pixels,
// once each: the second frame draws none anew, and shows them the same.
func TestCanvases(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := gputest.CanvasScene()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	for frame := range 2 {
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		pix, stride := r.readBack(t)
		gputest.Compare(t, "d3d11-canvas", pix, stride, s)
		if err := r.present(1); err != nil {
			t.Fatal(err)
		}
		if frame == 0 {
			s.Canvases = nil
		}
	}
	if len(r.canvases) != 3 {
		t.Errorf("%d canvas textures, want 3", len(r.canvases))
	}
}

// A transient canvas's texture stays while frames draw it, and goes once
// they stop, though the canvas is still about; another canvas's stays
// until it is collected.
func TestTransientCanvas(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r, err := New(hiddenWindow(t, 64, 64))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	layer, kept := scene.NewCanvas(64, 64), scene.NewCanvas(64, 64)
	layer.Transient = true
	s := &scene.Scene{Width: 64, Height: 64, Canvases: []*scene.Image{layer, kept}}
	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	r.sweep()
	if r.canvases[layer.ID()] == nil || r.canvases[kept.ID()] == nil {
		t.Fatal("a sweep after the frame that drew them freed a canvas's texture")
	}
	r.frame++ // a frame that draws neither
	r.sweep()
	if r.canvases[layer.ID()] != nil || r.canvases[kept.ID()] == nil {
		t.Errorf("transient canvas kept %v, other kept %v", r.canvases[layer.ID()] != nil, r.canvases[kept.ID()] != nil)
	}
	runtime.KeepAlive(layer)
	runtime.KeepAlive(kept)
}

// Frames that present sweep the textures of earlier frames' transient
// canvases even when every frame between them changes nothing: those
// count as frames too, and sweeping only at multiples of 120 frames, an
// animation that drew every other frame presented none of them, and kept
// a texture for each of its frames' canvases.
func TestSweepBetweenUnchangedFrames(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r, err := New(hiddenWindow(t, 64, 64))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	for range 3 * sweepEvery {
		c := scene.NewCanvas(16, 16)
		c.Transient = true
		s := &scene.Scene{Width: 64, Height: 64, Canvases: []*scene.Image{c}}
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
		r.frame++ // a frame that changes nothing, and presents nothing
	}
	if n := len(r.canvases); n > sweepEvery {
		t.Errorf("%d canvas textures after %d frames drawn, want at most %d", n, 3*sweepEvery, sweepEvery)
	}
}
