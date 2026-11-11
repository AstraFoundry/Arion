//go:build linux && (amd64 || arm64)

package gl

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/raster"
)

// apis are the APIs the renderer draws with: OpenGL 3.3 and OpenGL ES 3.0.
var apis = []struct {
	name string
	es   bool
}{{"OpenGL", false}, {"OpenGL ES", true}}

// offscreen makes a context current on the calling thread, without a
// window, with a framebuffer of w×h pixels (see Offscreen), or skips t.
// read returns its pixels as the CPU renderer draws them.
func offscreen(t *testing.T, es bool, w, h int) (read func() []byte) {
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	o, err := NewOffscreen(es, w, h)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(o.Release)
	t.Log(o.Info())
	return func() []byte {
		pix, err := ReadFramebuffer(w, h)
		if err != nil {
			t.Fatal(err)
		}
		return pix
	}
}

func TestDrawsAsTheCPURenderer(t *testing.T) {
	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			s := gputest.Scene()
			read := offscreen(t, api.es, s.Width, s.Height)
			r, err := New()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Release()
			if r.es != api.es {
				t.Fatalf("the renderer takes the context for OpenGL ES: %v", r.es)
			}
			// Twice: the second frame updates what the first uploaded.
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl", read(), s.Width*4, s)
			}

			s = gputest.TransformedScene()
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl-affine", read(), s.Width*4, s)
			}
			s = gputest.PerspectiveScene()
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl-perspective", read(), s.Width*4, s)
			}
			s = gputest.BehindOriginScene()
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				pix := read()
				gputest.Compare(t, "gl-behind-origin", pix, s.Width*4, s)
				if p := pix[370*s.Width*4+160*4:][:4]; p[1] > 100 {
					t.Errorf("the card turned far from its origin does not show: %v at its center", p)
				}
			}
			for i, s := range gputest.TranslatedScenes() {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl-translated"+[]string{"", "-between-pixels"}[i], read(), s.Width*4, s)
			}
			s = gputest.MaskedClipScene()
			// Twice: the second frame finds the clips' masks it made.
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl-masked-clips", read(), s.Width*4, s)
			}
			s = gputest.CornerClipScene()
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "gl-corner-clips", read(), s.Width*4, s)
			}
		})
	}
}

// TestBorderedCardDrawsAsTheCPURenderer draws the card of #531 (a large
// opaque card with a border of 2 pixels at scale 2, and a small button like
// it) with round corners, as OpenGL draws continuous ones, and requires the
// CPU renderer's pixels along their straight edges, exactly.
func TestBorderedCardDrawsAsTheCPURenderer(t *testing.T) {
	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			s := gputest.BorderedCardScene(false)
			read := offscreen(t, api.es, s.Width, s.Height)
			r, err := New()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Release()
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.CompareStraightEdges(t, "gl-bordered-card", read(), s.Width*4, s)
			}
		})
	}
}

func TestPresentsFramesDrawnInMemory(t *testing.T) {
	for _, api := range apis {
		t.Run(api.name, func(t *testing.T) {
			s := gputest.Scene()
			read := offscreen(t, api.es, s.Width, s.Height)
			want := raster.NewImage(s.Width, s.Height)
			raster.Render(want, s)
			var p Presenter
			// Twice: the second frame reuses the texture.
			for range 2 {
				if err := p.Present(want.Pix, want.Stride, want.W, want.H); err != nil {
					t.Fatal(err)
				}
				if got := read(); !bytes.Equal(got, want.Pix[:len(got)]) {
					t.Fatal("the frame presented differs from the frame drawn")
				}
			}
		})
	}
}
