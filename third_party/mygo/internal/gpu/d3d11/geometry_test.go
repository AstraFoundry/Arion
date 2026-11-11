//go:build windows

package d3d11

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// The shared builder's culling and scissors must preserve every pixel,
// including nested clip masks and canvases drawn into textures, as with
// OpenGL (see the gl package's test of the same name).
func TestGeometryOptimizationsPreservePixels(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	type fixture struct {
		name string
		s    *scene.Scene
	}
	var cases []fixture
	for _, c := range gputest.CullCases() {
		cases = append(cases, fixture{"cull/" + c.Name, c.Scene})
	}
	for _, c := range gputest.RingCases() {
		cases = append(cases, fixture{"ring/" + c.Name, c.Scene})
	}
	for i, c := range gputest.ShadowCases() {
		cases = append(cases, fixture{fmt.Sprintf("shadow/%d", i), gputest.ShadowPanelScene(c)})
	}
	for _, c := range gputest.MaskedCases() {
		cases = append(cases, fixture{"masked/" + c.Name, c.Scene})
	}
	cases = append(cases,
		fixture{"transformed", gputest.TransformedScene()},
		fixture{"perspective", gputest.PerspectiveScene()},
		fixture{"translated", gputest.TranslatedScenes()[0]},
		fixture{"translated-between-pixels", gputest.TranslatedScenes()[1]},
		fixture{"masked-clips", gputest.MaskedClipScene()},
		fixture{"corner-clips", gputest.CornerClipScene()},
		fixture{"wide-colors", gputest.WideScene()},
		fixture{"canvases", gputest.CanvasScene()},
	)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := New(hiddenWindow(t, c.s.Width, c.s.Height))
			if err != nil {
				t.Skip("no Direct3D 11:", err)
			}
			defer r.Release()
			if err := r.draw(c.s); err != nil {
				t.Fatal(err)
			}
			want, _ := r.readBack(t)
			if err := r.present(1); err != nil {
				t.Fatal(err)
			}
			r.b.NoCull, r.b.NoShadowSplit, r.b.NoRingSplit = true, true, true
			for _, canvas := range r.canvases {
				canvas.b.NoCull, canvas.b.NoShadowSplit, canvas.b.NoRingSplit = true, true, true
			}
			// Draw every canvas again under the reference policy, rather
			// than comparing against its already optimized texture.
			c.s.EachCanvas(func(img *scene.Image) { img.Changed() })
			if err := r.draw(c.s); err != nil {
				t.Fatal(err)
			}
			if got, _ := r.readBack(t); !bytes.Equal(got, want) {
				t.Fatal("culling or split scissors changed pixels")
			}
		})
	}
}
