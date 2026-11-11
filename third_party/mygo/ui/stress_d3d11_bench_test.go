//go:build windows

package ui

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/scene"
)

// stressManyBoxes is stressBoxes with n boxes, which go over each other past
// 600.
func stressManyBoxes(n int) func() (func(c *Context), func()) {
	return func() (func(c *Context), func()) {
		frame := 0
		view := func(c *Context) {
			Box(c).Fill().Background(RGB(246, 247, 249)).Children(func() {
				for i := range n {
					t := float32(frame + i)
					x := 8 + float32(i%30)*29 + float32(frame%40)*0.25
					y := 8 + float32(i/30%20)*31 + float32((frame+i)%20)*0.5
					Box(c).Absolute().Left(x).Top(y).Size(24, 24).Radius(6).
						Shadow(0, 2, 8, 0, RGBA(0, 0, 0, 0.27)).
						Background(RGB(uint8(40+int(t)*7%180), uint8(80+i*13%150), uint8(120+frame*3%120)))
				}
			})
		}
		return view, func() { frame++ }
	}
}

// stressPaced draws a frame a 60th of a second, as a window does: a GPU that
// runs them one after the other clocks higher, and shows costs less of what a
// frame costs in a window.
var stressPaced = os.Getenv("MYGO_STRESS_PACED") == "1"

// BenchmarkStressFrameGPU runs the frames of the stress scenes, each drawn
// with Direct3D 11 into a window that never shows and waited for: GPU-ms is
// what the GPU spent drawing it, by its timestamps.
func BenchmarkStressFrameGPU(b *testing.B) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	type stress struct {
		name string
		view func() (func(c *Context), func())
	}
	scenes := []stress{{"boxes4", stressManyBoxes(2400)}, {"boxes16", stressManyBoxes(9600)}}
	for _, sc := range stressScenes {
		scenes = append(scenes, stress{sc.name, sc.view})
	}
	for _, sc := range scenes {
		b.Run(sc.name, func(b *testing.B) {
			o, err := d3d11.NewOffscreen(stressW*stressScale, stressH*stressScale)
			if err != nil {
				b.Skip("no Direct3D 11:", err)
			}
			defer o.Release()
			var gpu time.Duration
			f := newStressFrames(sc.view, func(s *scene.Scene) {
				d, e := o.Render(s)
				if e != nil {
					err = e
				}
				gpu += d
			})
			gpu = 0
			start := time.Now()
			for b.Loop() {
				if stressPaced {
					time.Sleep(time.Until(start.Add(time.Second / 60)))
					start = time.Now()
				}
				f.next()
			}
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(gpu.Seconds()*1000/float64(b.N), "GPU-ms")
		})
	}
}
