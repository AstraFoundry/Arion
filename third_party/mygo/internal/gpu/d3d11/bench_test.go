//go:build windows

package d3d11

import (
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// BenchmarkRenderSprites draws frames of a 2D game at 1920×1080
// (gputest's SpriteScene: 1,872 sprites of one sheet) and reads each
// back, which waits for it; with WARP where no GPU runs it, as on CI.
func BenchmarkRenderSprites(b *testing.B) {
	benchRender(b, gputest.SpriteScene(1920, 1080, 3, 3, 300))
}

// BenchmarkRenderEmpty draws frames of nothing at 1920×1080: what the
// sprites' frames spend besides them.
func BenchmarkRenderEmpty(b *testing.B) {
	benchRender(b, &scene.Scene{Width: 1920, Height: 1080, Clear: scene.Color{A: 255}})
}

func benchRender(b *testing.B, s *scene.Scene) {
	hwnd, err := newHiddenWindow(s.Width, s.Height)
	if err != nil {
		b.Skip(err)
	}
	defer destroyWindow(hwnd)
	r, err := New(hwnd)
	if err != nil {
		b.Skip(err)
	}
	defer r.Release()
	for b.Loop() {
		if err := r.draw(s); err != nil {
			b.Fatal(err)
		}
		if _, _, err := r.read(); err != nil {
			b.Fatal(err)
		}
	}
}
