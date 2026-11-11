//go:build darwin

package metal

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// TestGlyphProbeJoinedEdges checks that batches joining small rounded
// fills, with ordinary and continuous corners, and glyphs at fractional
// positions draw the pixels of the general shader, and of the classes'
// batches kept apart, keeping the atlas unchanged across each comparison.
// A Radeon Pro 560 dropped glyph pixels there until sampleLocal sampled
// level 0 explicitly (#649).
func TestGlyphProbeJoinedEdges(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	atlas := scene.NewAtlas(1, 128, 128)
	mask := make([]byte, 32*32)
	for y := range 32 {
		for x := range 32 {
			d := math.Hypot(float64(x)-15.5, float64(y)-15.5)
			coverage := math.Min(1, math.Max(0, 1.5-math.Abs(d-12)))
			mask[y*32+x] = byte(255 * coverage)
		}
	}
	u, v, _ := atlas.Alloc(32, 32)
	atlas.Put(u, v, 32, 32, mask, 32)
	positions := []float32{0, 0.2, 0.4, 0.5, 0.6, 0.8}
	cases, joinedFrames, largest := 0, 0, 0
	for _, dx := range positions {
		for _, dy := range positions {
			for _, continuous := range []bool{false, true} {
				s := &scene.Scene{Width: 960, Height: 640, Scale: 2,
					Clear: scene.Color{R: 41, G: 42, B: 45, A: 255}, MaskAtlas: atlas}
				s.Ops = append(s.Ops, scene.Op{Kind: scene.OpPushClip,
					Rect: scene.Rect{X: 2 + dx, Y: 2 + dy, W: 950, H: 630}, Radii: scene.Uniform(8)})
				for row := range 8 {
					for col := range 12 {
						x, y := 10+float32(col)*77+dx, 10+float32(row)*76+dy
						s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill,
							Rect: scene.Rect{X: x, Y: y, W: 52, H: 48}, Radii: scene.Uniform(8),
							Continuous: continuous, Color: scene.Color{R: 47, G: 50, B: 55, A: 255}})
						start := int32(len(s.Glyphs))
						s.Glyphs = append(s.Glyphs, scene.Glyph{X: x + 10, Y: y + 8, W: 32, H: 32,
							U: uint16(u), V: uint16(v), UW: 32, VH: 32,
							Color: scene.Color{R: 154, G: 160, B: 166, A: 255}})
						s.Ops = append(s.Ops, scene.Op{Kind: scene.OpGlyphs, Start: start, End: start + 1})
					}
				}
				s.Ops = append(s.Ops, scene.Op{Kind: scene.OpPopClip})
				r.b.NoClasses, r.b.NoJoin = true, false
				want := renderRaw(t, r, s, false)
				for _, apart := range []bool{false, true} {
					r.b.NoClasses, r.b.NoJoin = false, apart
					got := renderRaw(t, r, s, false)
					for _, b := range r.b.Batches {
						if b.Joined != 0 {
							joinedFrames++
							break
						}
					}
					different, maximum := 0, 0
					for i := range got {
						delta := int(got[i]) - int(want[i])
						if delta < 0 {
							delta = -delta
						}
						if delta != 0 {
							different++
							maximum = max(maximum, delta)
						}
					}
					largest = max(largest, maximum)
					if maximum > 2 {
						t.Errorf("dx=%g dy=%g continuous=%t apart=%t: %d bytes differ; largest channel delta=%d", dx, dy, continuous, apart, different, maximum)
					}
				}
				cases++
			}
		}
	}
	if joinedFrames == 0 {
		t.Fatal("no mixed batch was exercised")
	}
	t.Logf("%d scenes, %d frames containing joined batches, largest channel delta=%d", cases, joinedFrames, largest)
}
