// Chart-streaming draws three rolling waveforms of 15,000 samples each,
// 1,000 new samples a second, reduced to a few points per pixel column.
package main

import (
	"flag"
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

const samples = 15000

type waveform struct {
	data             [3][samples]float32
	next             int64
	xs               []float32
	ys               [3][]float32
	bucketX, bucketY [3][]float32
	paths            [3]ui.Path
	points           [3][][2]float32
	start            time.Time
	mode             string
}

func value(i int64, s int) float32 {
	x := float64(i) / 1000
	// Deterministic noise and isolated spikes, which MinMax keeps.
	n := uint64(i)*6364136223846793005 + uint64(s+1)*1442695040888963407
	v := .22*math.Sin(x*float64(s+1)*7) + .08*(float64(n>>48)/65535-.5)
	if i%997 == int64(s*31) {
		v += .22
	}
	return float32(.2 + float64(s)*.3 + v*.55)
}

func newWaveform(mode string) *waveform {
	s := &waveform{mode: mode, next: samples, xs: make([]float32, samples)}
	for k := range 3 {
		s.ys[k] = make([]float32, samples)
		for i := range samples {
			s.data[k][i] = value(int64(i), k)
		}
	}
	return s
}

// update fills the ring with the samples due by now, and lays the latest
// 15,000 out as plot fractions.
func (s *waveform) update(now time.Time) {
	if s.start.IsZero() {
		s.start = now
	}
	elapsed := now.Sub(s.start)
	target := int64(samples) + elapsed.Milliseconds()
	// After a long frame, only the samples still visible are made.
	for i := max(s.next, target-samples); i < target; i++ {
		for k := range 3 {
			s.data[k][i%samples] = value(i, k)
		}
	}
	s.next = target
	phase := float32(elapsed%time.Millisecond) / float32(time.Millisecond)
	for i := range samples {
		s.xs[i] = (float32(i) - phase) / float32(samples-1)
		for k := range 3 {
			s.ys[k][i] = s.data[k][(target-samples+int64(i))%samples]
		}
	}
}

var colors = [3]ui.Color{ui.RGB(40, 140, 240), ui.RGB(240, 100, 70), ui.RGB(50, 190, 110)}

func (s *waveform) view(c *ui.Context) {
	s.update(c.Now())
	c.AnimationFrame()
	t := c.Theme()
	x := chart.NewLinear(-15, 0)
	y := chart.NewLinear(0, 1)
	chart.Plot(c, chart.Axes{Left: y.Ticks(5), Bottom: x.Ticks(5)}, func(area ui.Element) {
		area.Label("Three live waveforms").Description("15 seconds at 1000 samples per second per series").Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range y.Ticks(5) {
				p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
			}
			p.Clip(r, 0, func() {
				for k := range 3 {
					if s.mode == "raw" {
						p.StrokePath(chart.Line(r, s.xs, s.ys[k], chart.CurveLinear), 2, colors[k])
						continue
					}
					// Keep the extremes of each pixel column, into slices
					// reused every frame.
					s.bucketX[k], s.bucketY[k] = chart.MinMax(s.bucketX[k], s.bucketY[k], s.xs, s.ys[k], int(math.Ceil(float64(r.W*p.Scale()))))
					xs, ys := s.bucketX[k], s.bucketY[k]
					if s.mode == "path" {
						p.StrokePath(chart.LineInto(&s.paths[k], r, xs, ys, chart.CurveLinear), 2, colors[k])
						continue
					}
					s.points[k] = s.points[k][:0]
					for i, x := range xs {
						s.points[k] = append(s.points[k], [2]float32{chart.X(r, x), chart.Y(r, ys[i])})
					}
					p.StrokePolyline(s.points[k], 2, colors[k])
				}
			})
		})
	}).Fill().Gap(8).Padding(8)
}

func main() {
	mode := flag.String("mode", "polyline", "how to draw: polyline (MinMax, StrokePolyline), path (MinMax, StrokePath) or raw (every sample, StrokePath)")
	flag.Parse()
	if *mode != "polyline" && *mode != "path" && *mode != "raw" {
		log.Fatal("unknown mode ", *mode)
	}
	s := newWaveform(*mode)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Streaming chart", Width: 1000, Height: 600, Content: ui.View(s.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
