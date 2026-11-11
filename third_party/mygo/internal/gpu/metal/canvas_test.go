//go:build darwin

package metal

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// Canvases draw into textures as the CPU renderer draws them into pixels,
// once each: the second frame draws none anew, and shows them the same.
func TestCanvases(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Skip("no Metal:", err)
	}
	defer r.Release()
	s := gputest.CanvasScene()
	for frame := range 2 {
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-canvas", pix, s.Width*4, s)
		if frame == 0 {
			s.Canvases = nil // drawn: only shown from now on
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
	r, err := newRenderer()
	if err != nil {
		t.Skip("no Metal:", err)
	}
	defer r.Release()
	layer, kept := scene.NewCanvas(64, 64), scene.NewCanvas(64, 64)
	layer.Transient = true
	s := &scene.Scene{Width: 64, Height: 64}
	s.Canvases = []*scene.Image{layer, kept}
	if _, err := r.renderOffscreen(s); err != nil {
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
