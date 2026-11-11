//go:build darwin

package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/gpu/metal"
	"github.com/egoist/mygo/internal/scene"
)

// BenchmarkStressFrameGPU runs the frames of the stress scenes, each
// drawn with Metal into a texture and waited for: ns/op is a frame's
// whole time, CPU and GPU one after the other, and GPU-ms what the GPU
// spent drawing it, by its command buffer's timestamps, and encode-ms what
// the CPU spent encoding it for the GPU.
func BenchmarkStressFrameGPU(b *testing.B) {
	for _, sc := range stressScenes {
		b.Run(sc.name, func(b *testing.B) {
			o, err := metal.NewOffscreen(stressW*stressScale, stressH*stressScale)
			if err != nil {
				b.Skip("no Metal:", err)
			}
			defer o.Release()
			var gpu, encoded time.Duration
			f := newStressFrames(sc.view, func(s *scene.Scene) {
				d, e := o.Render(s)
				if e != nil {
					err = e
				}
				gpu += d
				encoded += o.Encoded()
			})
			gpu, encoded = 0, 0
			for b.Loop() {
				f.next()
			}
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(gpu.Seconds()*1000/float64(b.N), "GPU-ms")
			b.ReportMetric(encoded.Seconds()*1000/float64(b.N), "encode-ms")
		})
	}
}
