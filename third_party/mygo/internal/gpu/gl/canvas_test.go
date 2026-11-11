//go:build linux && (amd64 || arm64)

package gl

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// Canvases draw into textures as the CPU renderer draws them into pixels,
// shown the right way up although GL's rows go up, and once each.
func TestCanvases(t *testing.T) {
	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			s := gputest.CanvasScene()
			read := offscreen(t, api.es, s.Width, s.Height)
			r, err := New()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Release()
			for frame := range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl-canvas", read(), s.Width*4, s)
				if frame == 0 {
					s.Canvases = nil
				}
			}
			if len(r.canvases) != 3 {
				t.Errorf("%d canvas textures, want 3", len(r.canvases))
			}
		})
	}
}

// A transient canvas's texture stays while frames draw it, and goes once
// they stop, though the canvas is still about; another canvas's stays
// until it is collected.
func TestTransientCanvas(t *testing.T) {
	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			offscreen(t, api.es, 64, 64)
			r, err := New()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Release()
			layer, kept := scene.NewCanvas(64, 64), scene.NewCanvas(64, 64)
			layer.Transient = true
			s := &scene.Scene{Width: 64, Height: 64, Canvases: []*scene.Image{layer, kept}}
			if err := r.Render(s); err != nil {
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
		})
	}
}
