//go:build linux && (amd64 || arm64)

package gl

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// BenchmarkRenderSprites draws frames of a 2D game at 1280×720 (gputest's
// SpriteScene, tiles 2 times as large: about 1,800 sprites of one sheet)
// and reads each back, which waits for it. Where no GPU runs it, Mesa's
// llvmpipe draws on the CPU, so the time is mostly its own.
func BenchmarkRenderSprites(b *testing.B) {
	benchRender(b, gputest.SpriteScene(1280, 720, 2, 3, 300))
}

// BenchmarkRenderEmpty draws frames of nothing at 1280×720: what the
// sprites' frames spend besides them.
func BenchmarkRenderEmpty(b *testing.B) {
	benchRender(b, &scene.Scene{Width: 1280, Height: 720, Clear: scene.Color{A: 255}})
}

func benchRender(b *testing.B, s *scene.Scene) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	o, err := NewOffscreen(false, s.Width, s.Height)
	if err != nil {
		b.Skip(err)
	}
	defer o.Release()
	for b.Loop() {
		if _, err := o.Render(s); err != nil {
			b.Fatal(err)
		}
	}
}
