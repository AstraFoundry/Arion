package chart

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkLine constructs three linear paths of 15,000 samples, without rendering.
func BenchmarkLine(b *testing.B) {
	xs, ys := make([]float32, 15000), make([]float32, 15000)
	for i := range xs {
		xs[i] = float32(i) / 14999
		ys[i] = float32(math.Sin(float64(i)))
	}
	r := ui.Rect{W: 1280, H: 720}
	b.ReportAllocs()
	for b.Loop() {
		for range 3 {
			Line(r, xs, ys, CurveLinear)
		}
	}
}

// BenchmarkStreamingFrame includes build, layout, paint and CPU fallback at 1x/2x.
// The desktop example measures GPU submission and actual frame scheduling instead.
func BenchmarkStreamingFrame(b *testing.B) {
	for _, scale := range []float32{1, 2} {
		b.Run(fmt.Sprintf("%gx", scale), func(b *testing.B) {
			xs, ys := make([]float32, 15000), make([]float32, 15000)
			var dx, dy []float32
			frame := 0
			tt := ui.NewTester(func(c *ui.Context) {
				for i := range xs {
					xs[i] = float32(i) / 14999
					ys[i] = .5 + .3*float32(math.Sin(float64(i+frame)/20))
				}
				ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
					dx, dy = MinMax(dx, dy, xs, ys, int(r.W*p.Scale()))
					for range 3 {
						p.StrokePath(Line(r, dx, dy, CurveLinear), 2, ui.RGB(30, 100, 220))
					}
				})
			}, 1280, 720)
			tt.SetScale(scale)
			b.ReportAllocs()
			for b.Loop() {
				frame++
				tt.Frame()
			}
		})
	}
}
