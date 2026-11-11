package ui

import (
	"testing"
	"time"
)

// layerSpy is an effect that draws nothing, and notes the bitmaps it
// draws from.
type layerSpy struct {
	ids   map[uint64]bool
	sizes [][2]int
	parts [][2]int
}

func (s *layerSpy) EffectMargin() Point { return Point{X: 3, Y: 3} }

func (s *layerSpy) PaintEffect(p *Painter, box Rect, content *Bitmap, inset Point) {
	s.ids[content.img.ID()] = true
	s.sizes = append(s.sizes, [2]int{content.img.W, content.img.H})
	w, h := content.Size()
	s.parts = append(s.parts, [2]int{w, h})
}

// TestEffectLayerKept moves an element with an effect across pixels, as
// an animation takes it, growing it a little: it keeps the bitmap it
// renders into, its texture with it, rather than a bitmap a size, and the
// effect reads as much of it as the element and the margin take.
func TestEffectLayerKept(t *testing.T) {
	fx := &layerSpy{ids: map[uint64]bool{}}
	moved := false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Width(400).Children(func() {
			w := float32(200)
			if moved {
				coreBox(c).Height(37)
				w = 230
			}
			coreBox(c).Key("fx").Width(w).Height(30).Background(RGB(255, 0, 0)).
				Transition(ElementTransition{Duration: 400 * time.Millisecond, Ease: Linear}).Effect(fx)
		})
	}, 400, 200)
	tt.Frame()
	moved = true
	for range 30 {
		tt.Advance(16 * time.Millisecond)
		tt.Frame()
	}
	if len(fx.ids) != 1 {
		t.Errorf("%d bitmaps, want 1", len(fx.ids))
	}
	for i, s := range fx.sizes {
		if s[0]%effectLayerStep != 0 || s[1]%effectLayerStep != 0 {
			t.Fatalf("bitmap of %v, not in steps of %d", s, effectLayerStep)
		}
		if p := fx.parts[i]; p[0] > s[0] || p[1] > s[1] || p[0] < 200+6 || p[1] < 30+6 {
			t.Fatalf("part of %v of a bitmap of %v", p, s)
		}
	}
	if last := fx.parts[len(fx.parts)-1]; last != [2]int{230 + 6, 30 + 6} {
		t.Errorf("last part %v, want the box and its margin", last)
	}
}

// TestEffectLayerResized grows an element with an effect past its
// bitmap, then shrinks it to well under: the bitmap is resized in place,
// keeping its texture's identity, larger in steps, then smaller again.
func TestEffectLayerResized(t *testing.T) {
	fx := &layerSpy{ids: map[uint64]bool{}}
	w := float32(200)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Width(w).Height(30).Background(RGB(255, 0, 0)).Effect(fx)
	}, 600, 100)
	for _, step := range []struct {
		w           float32
		layer, part [2]int
	}{
		{200, [2]int{256, 64}, [2]int{206, 36}},
		{400, [2]int{448, 64}, [2]int{406, 36}}, // past the bitmap: made larger
		{330, [2]int{448, 64}, [2]int{336, 36}}, // less than two steps over: kept
		{100, [2]int{128, 64}, [2]int{106, 36}}, // two steps over or more: made smaller
	} {
		w = step.w
		tt.Frame()
		if s, p := fx.sizes[len(fx.sizes)-1], fx.parts[len(fx.parts)-1]; s != step.layer || p != step.part {
			t.Errorf("width %v: bitmap of %v, part %v, want %v and %v", step.w, s, p, step.layer, step.part)
		}
	}
	if len(fx.ids) != 1 {
		t.Errorf("%d bitmaps, want 1 resized", len(fx.ids))
	}
}

// TestEffectLayerTransient: the bitmap of an element's effect is drawn
// anew whenever it shows, so renderers may free its texture once frames
// stop drawing it.
func TestEffectLayerTransient(t *testing.T) {
	var st *state
	tt := coreNewTester(func(c *context) {
		e := coreBox(c).Width(50).Height(20).Background(RGB(0, 0, 255)).Effect(&layerSpy{ids: map[uint64]bool{}})
		st = e.st
	}, 100, 100)
	tt.Frame()
	if st.effectLayer == nil || !st.effectLayer.img.Transient {
		t.Errorf("layer %v, not transient", st.effectLayer)
	}
}

// imageEffect draws its content as an image, through ImageRegion or
// Image, over the box and its margin.
type imageEffect struct{ region bool }

func (imageEffect) EffectMargin() Point { return Point{X: 3, Y: 3} }

func (fx imageEffect) PaintEffect(p *Painter, box Rect, content *Bitmap, inset Point) {
	s := p.Scale()
	w, h := content.Size()
	area := Rect{X: box.X - inset.X/s, Y: box.Y - inset.Y/s, W: float32(w) / s, H: float32(h) / s}
	if fx.region {
		p.ImageRegion(content, Rect{}, area, ImageOptions{})
	} else {
		p.Image(content, area, FillBox)
	}
}

// TestEffectDrawsItsPart: an effect drawing its content as an image draws
// the content's part of the layer, at its size, rather than the whole
// layer squeezed into it.
func TestEffectDrawsItsPart(t *testing.T) {
	for _, region := range []bool{true, false} {
		img := coreNewTester(func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Padding(10).Children(func() {
				coreBox(c).Width(70).Height(20).Background(RGB(255, 0, 0)).Effect(imageEffect{region})
			})
		}, 200, 100).Image()
		for _, x := range []int{12, 45, 77} { // the box spans 10 to 80
			if p := img.RGBAAt(x, 20); p.R != 255 || p.G > 10 {
				t.Errorf("region %v: at x %d %v, want red", region, x, p)
			}
		}
		if p := img.RGBAAt(85, 20); p.G < 245 {
			t.Errorf("region %v: beyond the box %v, want white", region, p)
		}
	}
}

// wholeLayer draws the whole of its content's layer, beyond the content's
// part, as it lies.
type wholeLayer struct{}

func (wholeLayer) EffectMargin() Point { return Point{X: 3, Y: 3} }

func (wholeLayer) PaintEffect(p *Painter, box Rect, content *Bitmap, inset Point) {
	s := p.Scale()
	all := &Bitmap{img: content.img, w: content.img.W, h: content.img.H}
	p.ImageRegion(all, Rect{}, Rect{X: box.X - inset.X/s, Y: box.Y - inset.Y/s, W: float32(all.w) / s, H: float32(all.h) / s}, ImageOptions{})
}

// TestEffectLayerClipsOverflow: what an element paints beyond its box and
// the effect's margin is not drawn into its layer, though the layer has
// room for it, as a layer of the content's size has none.
func TestEffectLayerClipsOverflow(t *testing.T) {
	img := coreNewTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Padding(10).Children(func() {
			coreBox(c).Width(70).Height(20).Effect(wholeLayer{}).Children(func() {
				coreBox(c).Absolute().Left(60).Top(0).Width(60).Height(20).Background(RGB(255, 0, 0))
			})
		})
	}, 300, 100).Image()
	if p := img.RGBAAt(80, 20); p.R != 255 || p.G > 10 { // the margin, from 80 to 83
		t.Errorf("in the margin %v, want red", p)
	}
	if p := img.RGBAAt(100, 20); p.G < 245 { // the layer's padding
		t.Errorf("beyond the margin %v, want white", p)
	}
}
