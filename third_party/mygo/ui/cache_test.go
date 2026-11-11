package ui

import (
	"fmt"
	"image"
	"testing"
	"time"
)

// cacheCard is a card with a shadow, text, a gradient and a child beyond
// its box, with a drawing moving next to it, kept by Cache when cached; tilt
// rotates it, wide widens it, and accent colors its text.
type cacheCard struct {
	cached     bool
	tilt, wide float32
	accent     Color
	st         *state
	moves      int
}

func (v *cacheCard) view(c *context) {
	coreColumn(c).Fill().Background(RGB(255, 255, 255)).Padding(20).Children(func() {
		// Text over the card's opaque background, smoothed as over the
		// window's (ClearType's subpixels on Windows), in its bitmap too.
		card := coreBox(c).Width(120+v.wide).Height(50).Radius(12).Background(RGB(30, 60, 200)).
			Shadow(0, 4, 10, 2, RGBA(0, 0, 0, 90)).Padding(8).Children(func() {
			coreText(c, "Daily mix").TextColor(v.accent)
			coreBox(c).Absolute().Right(0).Top(0).Size(30, 50).Radius(12).Gradient(RGB(30, 60, 200), RGB(200, 40, 120), 30)
			// Beyond the card, which does not clip it.
			coreBox(c).Absolute().Left(150+v.wide).Top(30).Size(30, 14).Background(RGB(20, 160, 60))
		})
		if v.tilt != 0 {
			card.Rotate(v.tilt)
		}
		if v.cached {
			card.Cache()
		}
		v.st = card.st
		coreBox(c).Size(60, 12).Draw(func(p *Painter, r Rect) {
			v.moves++
			p.AnimationFrame()
			p.Fill(Rect{r.X + float32(p.Now().UnixMilli()%50), r.Y, 8, r.H}, RGB(0, 0, 0), 2)
		})
	})
}

// sameImage reports where a and b differ by more than a level or two of a
// channel, as compositing a layer rounds twice: "" for nowhere.
func sameImage(a, b *image.RGBA) string {
	n, at := 0, ""
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		if d < -2 || d > 2 {
			if n++; at == "" {
				px := i / 4
				at = fmt.Sprintf("(%d,%d) %d≠%d", px%a.Rect.Dx(), px/a.Rect.Dx(), a.Pix[i], b.Pix[i])
			}
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d bytes differ, first at %s", n, at)
}

// TestCacheKeepsLayer: a card kept by Cache shows as it does painted as
// it is, beyond its box too, and its bitmap is drawn once while a drawing
// next to it moves, whether frames paint the window whole or only the
// drawing; then once again as its text changes color, its size changes,
// or the window's scale does, but not as it turns.
func TestCacheKeepsLayer(t *testing.T) {
	for _, parts := range []bool{true, false} {
		t.Run(fmt.Sprint("parts=", parts), func(t *testing.T) {
			cached := &cacheCard{cached: true, accent: RGB(255, 255, 255)}
			plain := &cacheCard{accent: RGB(255, 255, 255)}
			ct, pt := newRepaintTester(cached.view), newRepaintTester(plain.view)
			ct.rt.noParts, pt.rt.noParts = !parts, !parts
			if cached.st.effectLayer == nil || cached.st.effectLayer.img.Transient {
				t.Fatalf("layer %v, want one kept", cached.st.effectLayer)
			}
			layer := cached.st.effectLayer.img
			// check draws frames as the drawing moves, from the frame in
			// which the layer had version v: drawn again in the first, as
			// redrawn says, then not.
			check := func(what string, v uint64, redrawn, resampled bool) {
				t.Helper()
				for i := range 4 {
					if i > 0 {
						ct.frame(7 * time.Millisecond)
						pt.frame(7 * time.Millisecond)
					}
					if d := sameImage(ct.Image(), pt.Image()); d != "" && !resampled {
						t.Fatalf("%s, frame %d: cached differs from painted as it is: %s", what, i, d)
					}
					if i == 0 && redrawn != (layer.Version() != v) {
						t.Errorf("%s: layer drawn again %v, want %v", what, layer.Version() != v, redrawn)
					}
					if i > 0 && layer.Version() != v {
						t.Errorf("%s, frame %d: layer drawn again while nothing changed", what, i)
					}
					v = layer.Version()
				}
			}
			check("moving next to it", layer.Version(), false, false)
			for _, step := range []struct {
				what    string
				change  func(v *cacheCard)
				redrawn bool
			}{
				{"text colored", func(v *cacheCard) { v.accent = RGB(255, 220, 0) }, true},
				{"widened", func(v *cacheCard) { v.wide = 40 }, true},
				{"turned", func(v *cacheCard) { v.tilt = 10 }, false},
			} {
				v := layer.Version()
				step.change(cached)
				step.change(plain)
				ct.Frame()
				pt.Frame()
				// Turned, the bitmap is resampled, where painting turns
				// the shapes themselves.
				check(step.what, v, step.redrawn, step.what == "turned")
			}
			cached.tilt, plain.tilt = 0, 0
			ct.Frame()
			pt.Frame()
			v := layer.Version()
			ct.SetScale(2)
			pt.SetScale(2)
			if layer.Version() == v {
				t.Errorf("at another scale: layer not drawn again")
			}
			if d := sameImage(ct.Image(), pt.Image()); d != "" {
				t.Errorf("at scale 2: cached differs from painted as it is: %s", d)
			}
		})
	}
}

// TestCacheAreaClipped: the bitmap of an element that clips is its box
// and its shadows' reach, not what lies inside beyond it.
func TestCacheAreaClipped(t *testing.T) {
	var st *state
	coreNewTester(func(c *context) {
		e := coreBox(c).Size(100, 40).ClipX().ClipY().Cache().Children(func() {
			coreBox(c).Size(400, 300).Background(RGB(255, 0, 0))
		})
		st = e.st
	}, 600, 400)
	w, h := st.effectLayer.Size()
	if w != 128 || h != 64 {
		t.Errorf("layer of %d×%d, want 128×64 for 100×40 and the focus ring's reach", w, h)
	}
}

// TestCacheKeepsBlur: a blurred element kept by Cache renders into its
// bitmap once, while the frames after it paint the window again.
func TestCacheKeepsBlur(t *testing.T) {
	var st *state
	tt := newRepaintTester(func(c *context) {
		coreRow(c).Children(func() {
			st = coreBox(c).Size(60, 30).Background(RGB(200, 0, 0)).Blur(4).Cache().st
			coreBox(c).Size(20, 20).Draw(func(p *Painter, r Rect) {
				p.AnimationFrame()
				p.Fill(Rect{r.X, r.Y, float32(p.Now().UnixMilli() % 20), r.H}, RGB(0, 0, 0), 0)
			})
		})
	})
	tt.rt.noParts = true
	layer := st.effectLayer.img
	v := layer.Version()
	for range 5 {
		tt.frame(9 * time.Millisecond)
	}
	if layer.Transient || layer.Version() != v {
		t.Errorf("transient %v, drawn %d times again, want kept", layer.Transient, layer.Version()-v)
	}
}
