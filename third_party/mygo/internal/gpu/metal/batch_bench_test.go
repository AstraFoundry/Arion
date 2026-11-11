//go:build darwin

package metal

import (
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// BenchmarkRenderBatches draws scenes whose ops interleave the classes of
// the shader (gpu.Class) and scenes of long runs of one class, with the
// renderer's batches ("joined"), with the batches of each class apart
// however small ("apart", gpu.Builder.NoJoin) and with every batch in the
// general class ("general"): ns/op is the time of a frame, GPU-ms what the GPU spent on
// it, and batches how many draw calls it took.
func BenchmarkRenderBatches(b *testing.B) {
	cases := []struct {
		name string
		s    func() *scene.Scene
	}{
		{"boxes", func() *scene.Scene { return gputest.BoxesScene(1280, 800, 600, false) }},
		{"translated boxes", func() *scene.Scene { return gputest.BoxesScene(1280, 800, 600, true) }},
		{"rows", func() *scene.Scene { return gputest.RowsScene(1280, 800, 40) }},
		{"fills", func() *scene.Scene { return gputest.FillScene(3456, 2000) }},
		{"opaque", func() *scene.Scene { return gputest.OpaqueScene(3456, 2000) }},
	}
	for _, c := range cases {
		for _, mode := range []string{"joined", "apart", "general"} {
			b.Run(c.name+"/"+mode, func(b *testing.B) {
				benchRenderWith(b, c.s(), func(r *Renderer) { r.b.NoJoin, r.b.NoClasses = mode == "apart", mode == "general" })
			})
		}
	}
}
