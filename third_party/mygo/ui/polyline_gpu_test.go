//go:build linux && (amd64 || arm64)

package ui

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gl"
	"github.com/egoist/mygo/internal/gpu/gputest"
)

func TestStrokePolylineGPU(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	points := [][2]float32{{-10, 20}, {30, 70}, {60, 20}, {80, 60}, {100, 10}, {110, 25}, {170, 60}, {210, 20}}
	for _, es := range []bool{false, true} {
		for _, scale := range []float32{1, 2} {
			for _, tt := range []*Tester{
				polylineTester(points, true, RGB(20, 60, 230), scale),
				polylineTester(points[:2], true, RGBA(20, 60, 230, .4), scale),
			} {
				s := &tt.rt.scene
				o, err := gl.NewOffscreen(es, s.Width, s.Height)
				if err != nil {
					t.Skip(err)
				}
				t.Log(o.Info())
				for range 2 {
					pix, err := o.Render(s)
					if err != nil {
						o.Release()
						t.Fatal(err)
					}
					gputest.Compare(t, "polyline", pix, s.Width*4, s)
				}
				o.Release()
			}
		}
	}
}
