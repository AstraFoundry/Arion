//go:build darwin

package metal

import (
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// TestZoomedFrames renders the frames of content zoomed 2 times on a
// screen of scale 2: the layer keeps the screen's scale, the frames
// changing a little draw in part, and a frame changing the zoom alone
// changes the layer not.
func TestZoomedFrames(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	frame := func(i int, scale, zoom float32) {
		t.Helper()
		s := &scene.Scene{}
		s.Reset(128, 64, scene.Color{A: 255})
		s.Scale, s.Zoom = scale, zoom
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(4 * i), W: 8, H: 8}, Color: scene.Color{R: 255, A: 255}})
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
		r.pre.wait()
		r.waitLast()
	}
	partial := 0
	for i := range 8 {
		frame(i, 4, 2)
		if r.scale != 2 {
			t.Fatalf("frame %d: the layer's scale is %v, want the screen's, 2", i, r.scale)
		}
		if r.redrew == redrewPart {
			partial++
		}
	}
	if partial == 0 {
		t.Error("no zoomed frame drew in part")
	}
	frame(8, 2, 0)
	if r.scale != 2 {
		t.Errorf("unzoomed, the layer's scale is %v, want 2", r.scale)
	}
}
