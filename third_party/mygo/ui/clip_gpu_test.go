//go:build linux && (amd64 || arm64)

package ui

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gl"
	"github.com/egoist/mygo/internal/gpu/gputest"
)

// TestNestedClipKeepsRoundedCornersGPU draws, with OpenGL, a rounded card
// that clips, holding a column that clips too and fills its bottom (#1418),
// at scale 1 and 2: the card's corners cut the column, as the CPU renderer
// draws them.
func TestNestedClipKeepsRoundedCornersGPU(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, scale := range []float32{1, 2} {
		tt := NewTester(func(c *Context) {
			Column(c).Fill().Padding(20).Children(func() {
				Column(c).Width(320).Height(120).Radius(10).Background(RGB(255, 255, 255)).Border(1, RGB(180, 180, 180)).Clip().Children(func() {
					Box(c).Height(40).FillWidth().Shrink(0)
					Column(c).Grow(1).FillWidth().Clip().Children(func() {
						Box(c).Fill().Background(RGB(0, 122, 255))
					})
				})
			})
		}, 400, 220)
		tt.SetScale(scale)
		s := &tt.rt.scene
		o, err := gl.NewOffscreen(false, s.Width, s.Height)
		if err != nil {
			t.Skip(err)
		}
		t.Log(o.Info())
		pix, err := o.Render(s)
		o.Release()
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "nested-clip", pix, s.Width*4, s)
	}
}
