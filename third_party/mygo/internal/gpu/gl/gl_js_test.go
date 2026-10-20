//go:build js

package gl

import (
	"bytes"
	"slices"
	"syscall/js"
	"testing"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
)

// canvas makes the WebGL 2 context of a w×h canvas current, or skips t
// where there is no page (Node) or no WebGL 2. read returns its pixels as
// the CPU renderer draws them.
func canvas(t *testing.T, w, h int) (c *Context, read func() []byte) {
	doc := js.Global().Get("document")
	if doc.IsUndefined() {
		t.Skip("no page to draw in")
	}
	el := doc.Call("createElement", "canvas")
	el.Set("width", w)
	el.Set("height", h)
	c, err := NewContext(el)
	if err != nil {
		t.Skip(err)
	}
	c.MakeCurrent()
	t.Log(versionString(), "dual-source blending:", c.dual)
	return c, func() []byte {
		pix, err := ReadFramebuffer(w, h)
		if err != nil {
			t.Fatal(err)
		}
		return pix
	}
}

func TestWebGLDrawsAsTheCPURenderer(t *testing.T) {
	// Browsers without WEBGL_blend_func_extended blend with one color,
	// which cannot cover the channels of subpixel glyphs apart: the text
	// of browsers has none (package text's canvas engine).
	for _, dual := range []bool{true, false} {
		name := "dual-source blending"
		if !dual {
			name = "one color"
		}
		t.Run(name, func(t *testing.T) {
			s := gputest.Scene()
			if !dual {
				withoutSubpixelGlyphs(s)
			}
			c, read := canvas(t, s.Width, s.Height)
			if dual && !c.dual {
				t.Skip("no WEBGL_blend_func_extended")
			}
			c.dual = dual
			r, err := New()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Release()
			if !r.es || r.dual != dual {
				t.Fatalf("the renderer takes WebGL 2 for OpenGL ES %v, with dual-source blending %v", r.es, r.dual)
			}
			// Twice: the second frame updates what the first uploaded.
			for range 2 {
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "webgl", read(), s.Width*4, s)
			}
			for name, s := range map[string]*scene.Scene{"webgl-affine": gputest.TransformedScene(), "webgl-perspective": gputest.PerspectiveScene()} {
				if !dual {
					withoutSubpixelGlyphs(s)
				}
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, name, read(), s.Width*4, s)
			}
		})
	}
}

func TestWebGLPresentsFramesDrawnInMemory(t *testing.T) {
	s := gputest.Scene()
	_, read := canvas(t, s.Width, s.Height)
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
}

// A renewed context, as after the browser restored a lost one, has the
// renderer make its objects again.
func TestWebGLRenewedContext(t *testing.T) {
	s := gputest.Scene()
	c, read := canvas(t, s.Width, s.Height)
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()
	if err := r.Render(s); err != nil {
		t.Fatal(err)
	}
	c.Renew()
	if err := r.Render(s); err != nil {
		t.Fatal(err)
	}
	gputest.Compare(t, "webgl-renewed", read(), s.Width*4, s)
}

// withoutSubpixelGlyphs leaves the subpixel glyphs out of s.
func withoutSubpixelGlyphs(s *scene.Scene) {
	ops := s.Ops[:0]
	for _, op := range s.Ops {
		if op.Kind == scene.OpGlyphs {
			gs := s.Glyphs[op.Start:op.End]
			if !slices.ContainsFunc(gs, func(g scene.Glyph) bool { return !g.Subpixel }) {
				continue
			}
			for i := range gs {
				if gs[i].Subpixel {
					gs[i].Color = scene.Color{}
				}
			}
		}
		ops = append(ops, op)
	}
	s.Ops = ops
}
