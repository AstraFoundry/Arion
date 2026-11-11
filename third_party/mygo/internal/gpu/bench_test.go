package gpu

import (
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// BenchmarkBuildSprites turns the scene of a 2D game's frame at 1920×1080
// into instances: tiles 3 times as large under 300 sprites, all of one
// sheet, which make one draw call.
func BenchmarkBuildSprites(b *testing.B) { benchBuild(b, 3, 3, 300) }

// BenchmarkBuildSpritesZoomedOut is the same game zoomed out to tiles 1.5
// times as large, under 1,000 sprites: about 7,000 instances.
func BenchmarkBuildSpritesZoomedOut(b *testing.B) { benchBuild(b, 1.5, 3, 1000) }

func benchBuild(b *testing.B, scale float32, layers, sprites int) {
	s := gputest.SpriteScene(1920, 1080, scale, layers, sprites)
	var bld Builder
	image := func(*scene.Image) uintptr { return 1 }
	for b.Loop() {
		bld.Build(s, image)
	}
	b.ReportMetric(float64(len(bld.Instances)), "instances")
	b.ReportMetric(float64(len(bld.Batches)), "batches")
}

// BenchmarkBuildTranslatedBoxes turns 600 rounded boxes casting shadows,
// each under a translation, into instances, the translations folded into
// them (BenchmarkBuildTranslatedBoxesUnfolded: under their transforms).
func BenchmarkBuildTranslatedBoxes(b *testing.B) { benchBuildBoxes(b, true, false) }

func BenchmarkBuildTranslatedBoxesUnfolded(b *testing.B) { benchBuildBoxes(b, true, true) }

// BenchmarkBuildBoxes turns the same boxes placed without transforms into
// instances.
func BenchmarkBuildBoxes(b *testing.B) { benchBuildBoxes(b, false, false) }

func benchBuildBoxes(b *testing.B, translated, noFold bool) {
	s := gputest.BoxesScene(1280, 800, 600, translated)
	bld := Builder{Classes: true, ClipSkip: true, NoFold: noFold}
	image := func(*scene.Image) uintptr { return 1 }
	for b.Loop() {
		bld.Build(s, image)
	}
	b.ReportMetric(float64(len(bld.Instances)), "instances")
	b.ReportMetric(float64(len(bld.Batches)), "batches")
}

// BenchmarkDiffRows compares a frame of a list of 200 rows at 3456×2000
// with the last (scene.Differ), as Metal's renderer does before drawing
// what changed: unchanged, and with one row's badge changing color, and
// BenchmarkBuildRows builds it, for comparison.
func BenchmarkDiffRows(b *testing.B) {
	for _, change := range []bool{false, true} {
		name := "unchanged"
		if change {
			name = "one_row"
		}
		b.Run(name, func(b *testing.B) {
			s := gputest.RowsScene(3456, 2000, 200)
			d := scene.Differ{Reach: 2}
			d.Diff(s)
			d.Remember(s)
			for i := 0; b.Loop(); i++ {
				if change {
					s.Ops[len(s.Ops)/2].Color.R = byte(i)
				}
				d.Diff(s)
				d.Remember(s)
			}
			b.ReportMetric(float64(len(s.Ops)), "ops")
			b.ReportMetric(float64(len(s.Glyphs)), "glyphs")
		})
	}
}

func BenchmarkBuildRows(b *testing.B) {
	s := gputest.RowsScene(3456, 2000, 200)
	bld := Builder{Classes: true, ClipSkip: true, Opaque: true}
	image := func(*scene.Image) uintptr { return 1 }
	for b.Loop() {
		bld.Build(s, image)
	}
	b.ReportMetric(float64(len(bld.Instances)), "instances")
}
