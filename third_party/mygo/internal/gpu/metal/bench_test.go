//go:build darwin

package metal

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// BenchmarkRenderSprites draws frames of a 2D game at 1920×1080
// (gputest's SpriteScene: 1,872 sprites of one sheet) into one texture,
// waiting for each: ns/op is the time of a frame, GPU-ms what the GPU
// spent on it, by its command buffer's timestamps, and encode-ms what the
// CPU spent encoding it.
func BenchmarkRenderSprites(b *testing.B) {
	benchRender(b, gputest.SpriteScene(1920, 1080, 3, 3, 300))
}

// BenchmarkRenderSpritesZoomedOut draws the game zoomed out: about 7,000
// sprites.
func BenchmarkRenderSpritesZoomedOut(b *testing.B) {
	benchRender(b, gputest.SpriteScene(1920, 1080, 1.5, 3, 1000))
}

// BenchmarkRenderEmpty draws frames of nothing at 1920×1080: what the
// others spend besides their sprites.
func BenchmarkRenderEmpty(b *testing.B) {
	benchRender(b, &scene.Scene{Width: 1920, Height: 1080, Clear: scene.Color{A: 255}})
}

func benchRender(b *testing.B, s *scene.Scene) { benchRenderWith(b, s, nil) }

// benchRenderWith is benchRender with the renderer set up by setup, if not
// nil, and the batches of the frame reported.
func benchRenderWith(b *testing.B, s *scene.Scene, setup func(*Renderer)) {
	o, err := NewOffscreen(s.Width, s.Height)
	if err != nil {
		b.Skip("no Metal:", err)
	}
	defer o.Release()
	if setup != nil {
		setup(o.r)
	}
	var gpu, encoded time.Duration
	for b.Loop() {
		d, err := o.Render(s)
		if err != nil {
			b.Fatal(err)
		}
		gpu += d
		encoded += o.Encoded()
	}
	b.ReportMetric(gpu.Seconds()*1000/float64(b.N), "GPU-ms")
	b.ReportMetric(encoded.Seconds()*1000/float64(b.N), "encode-ms")
	if setup != nil {
		b.ReportMetric(float64(len(o.r.b.Batches)), "batches")
	}
}

// BenchmarkRenderBoxes draws 600 rounded boxes casting shadows at
// 1280×800, and BenchmarkRenderTranslatedBoxes the same boxes under
// translations, as an animation moves them.
func BenchmarkRenderBoxes(b *testing.B) { benchRender(b, gputest.BoxesScene(1280, 800, 600, false)) }

func BenchmarkRenderTranslatedBoxes(b *testing.B) {
	benchRender(b, gputest.BoxesScene(1280, 800, 600, true))
}

// BenchmarkRenderCards draws a dashboard of twelve cards with continuous
// corners at 2560×1600 (gputest.CardsScene), each casting two shadows,
// which the builder draws as nine-patches (gpu.Builder.SplitCorners).
func BenchmarkRenderCards(b *testing.B) { benchRender(b, gputest.CardsScene(2560, 1600)) }

// BenchmarkRenderPanels draws a page of 96 cards at 2560×1600
// (gputest.PanelsScene): clips and images change from card to card, so a
// frame takes hundreds of batches, and encode-ms is mostly their cost.
func BenchmarkRenderPanels(b *testing.B) {
	benchRenderWith(b, gputest.PanelsScene(2560, 1600, 96), func(*Renderer) {})
}
