package ui

import (
	"image/color"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/text"
)

func isWhite(c color.RGBA) bool { return c.R > 250 && c.G > 250 && c.B > 250 }
func isRed(c color.RGBA) bool   { return c.R > 200 && c.G < 40 && c.B < 40 }

// ClipInset clips the element, its background and its children, to its
// box inset, in DIPs or percent; a negative inset reaches past the box,
// so that a shadow shows along one axis and is cut along the other.
func TestClipInset(t *testing.T) {
	red, white := RGB(255, 0, 0), RGB(255, 255, 255)
	draw := func(clip func(e *node)) func(c *context) {
		return func(c *context) {
			coreBox(c).Fill().Background(white).Padding(20).Children(func() {
				e := coreBox(c).Size(100, 60).Background(red).Shadow(0, 0, 0, 10, RGB(0, 0, 255))
				clip(e)
				e.Children(func() { coreBox(c).Size(200, 10).Background(RGB(0, 200, 0)) })
			})
		}
	}
	img := coreRender(draw(func(e *node) { e.ClipInsetPercent(0, 50, 0, 0, 0) }), 160, 100, 1)
	if c := img.RGBAAt(40, 50); !isRed(c) {
		t.Errorf("the half shown: %v", c)
	}
	if c := img.RGBAAt(100, 50); !isWhite(c) {
		t.Errorf("the half hidden: %v", c)
	}
	if c := img.RGBAAt(100, 25); !isWhite(c) {
		t.Errorf("the child in the half hidden: %v", c)
	}
	if c := img.RGBAAt(40, 25); c.G < 180 || c.R > 40 {
		t.Errorf("the child in the half shown: %v", c)
	}
	img = coreRender(draw(func(e *node) { e.ClipInset(-10, 0, -10, 0, 0) }), 160, 100, 2)
	if c := img.RGBAAt(2*60, 2*15); c.B < 200 || c.R > 40 {
		t.Errorf("the shadow above the box, inside the clip: %v", c)
	}
	if c := img.RGBAAt(2*15, 2*50); !isWhite(c) {
		t.Errorf("the shadow left of the box, outside the clip: %v", c)
	}
	img = coreRender(draw(func(e *node) { e.ClipInset(0, 0, 0, 0, 30) }), 160, 100, 1)
	if c := img.RGBAAt(22, 22); !isWhite(c) {
		t.Errorf("outside the clip's rounded corner: %v", c)
	}
}

// An inset an element animates wipes it in: none of it shows, then part
// of it, then all of it.
func TestClipInsetAnimates(t *testing.T) {
	open := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Padding(20).Children(func() {
			e := coreBox(c).Size(100, 60).Background(RGB(255, 0, 0))
			target := float32(100)
			if open {
				target = 0
			}
			e.ClipInsetPercent(0, e.Animate("wipe", target, 300*time.Millisecond), 0, 0, 0)
		})
	}, 160, 100)
	if c := tt.Image().RGBAAt(30, 50); !isWhite(c) {
		t.Fatalf("hidden: %v", c)
	}
	open = true
	tt.Frame()
	tt.Advance(150 * time.Millisecond)
	img := tt.Image()
	if c := img.RGBAAt(22, 50); !isRed(c) {
		t.Errorf("halfway, the left: %v", c)
	}
	if c := img.RGBAAt(118, 50); !isWhite(c) {
		t.Errorf("halfway, the right: %v", c)
	}
	tt.Advance(time.Second)
	if c := tt.Image().RGBAAt(118, 50); !isRed(c) {
		t.Errorf("shown: %v", c)
	}
}

// What ClipInset hides takes no clicks.
func TestClipInsetHidesFromThePointer(t *testing.T) {
	clicks := 0
	tt := coreNewTester(func(c *context) {
		b := coreBox(c).Size(100, 60).Background(RGB(255, 0, 0)).ClipInsetPercent(0, 50, 0, 0, 0)
		if b.Clicked() {
			clicks++
		}
	}, 160, 100)
	tt.ClickAt(80, 30)
	if clicks != 0 {
		t.Errorf("a click where the clip hides the element clicked it")
	}
	tt.ClickAt(20, 30)
	if clicks != 1 {
		t.Errorf("a click where the element shows: %d clicks", clicks)
	}
}

// A mask's gradient shows the element as much as its alpha: all at one
// end, none at the other, part between; its own colors do not matter.
func TestMaskGradients(t *testing.T) {
	opaque, clear := RGB(10, 200, 30), RGBA(10, 200, 30, 0)
	for _, c := range []struct {
		name string
		mask func(e *node)
		x    int // where the element shows whole, at y 50, and is gone at 220-x
	}{
		{"linear", func(e *node) { e.MaskLinearGradient(LinearGradient{From: opaque, To: clear, Angle: 90}) }, 22},
		{"linear stops", func(e *node) {
			e.MaskLinearGradient(LinearGradient{Angle: 270, Stops: []GradientStop{{clear, 0}, {opaque, 1}}})
		}, 22},
		{"radial", func(e *node) {
			e.MaskRadialGradient(RadialGradient{Center: Point{0.1, 0.5}, Radius: 150, Stops: []GradientStop{{opaque, 0}, {clear, 1}}})
		}, 30},
	} {
		img := coreRender(func(cx *context) {
			coreBox(cx).Fill().Background(RGB(255, 255, 255)).Padding(20).Children(func() {
				e := coreBox(cx).Size(180, 60).Background(RGB(255, 0, 0))
				c.mask(e)
			})
		}, 220, 100, 1)
		if p := img.RGBAAt(c.x, 50); !isRed(p) {
			t.Errorf("%s: where it is opaque, %v", c.name, p)
		}
		if p := img.RGBAAt(220-c.x, 50); c.name != "radial" && !isWhite(p) {
			t.Errorf("%s: where it is transparent, %v", c.name, p)
		}
		if p := img.RGBAAt(110, 50); p.R < 250 || p.G < 60 || p.G > 200 {
			t.Errorf("%s: halfway, %v", c.name, p)
		}
		if p := img.RGBAAt(110, 10); !isWhite(p) {
			t.Errorf("%s: outside the box, %v", c.name, p)
		}
	}
}

// Masks composites layers of masks, each across its box, as CSS's
// mask-composite: a ring of the padding, cut out of the border box by the
// content box, three ways; the content box alone; and half the box with
// the content box.
func TestMaskLayers(t *testing.T) {
	black := RGB(0, 0, 0)
	solid := LinearGradient{From: black, To: black}
	left := LinearGradient{Angle: 90, Stops: []GradientStop{{black, 0}, {black, 0.5}, {Transparent, 0.5}, {Transparent, 1}}}
	ring := "ring"
	for _, c := range []struct {
		name   string
		layers []Mask
		// What shows: in the padding on the left and the right, and in the
		// content box.
		left, right, content bool
	}{
		{"exclude", []Mask{{Image: solid, Box: MaskContentBox, Composite: MaskExclude}, {Image: solid}}, true, true, false},
		{"subtract", []Mask{{Image: solid, Composite: MaskSubtract}, {Image: solid, Box: MaskContentBox}}, true, true, false},
		{"intersect", []Mask{{Image: solid, Box: MaskContentBox, Composite: MaskIntersect}, {Image: solid}}, false, false, true},
		{"add", []Mask{{Image: left}, {Image: solid, Box: MaskContentBox}}, true, false, true},
		{"padding box", []Mask{{Image: solid, Box: MaskPaddingBox, Composite: MaskExclude}, {Image: solid}}, false, false, false},
	} {
		for _, scale := range []float32{1, 2} {
			img := coreRender(func(cx *context) {
				coreBox(cx).Fill().Background(RGB(255, 255, 255)).Padding(20).Children(func() {
					coreBox(cx).Size(100, 60).Padding(10).Border(2, RGB(255, 0, 0)).
						Background(RGB(255, 0, 0)).Label(ring).Masks(c.layers...)
				})
			}, 140, 100, scale)
			at := func(x, y int) color.RGBA { return img.RGBAAt(int(float32(x)*scale), int(float32(y)*scale)) }
			for _, p := range []struct {
				what  string
				x     int
				shows bool
			}{{"the padding on the left", 27, c.left}, {"the padding on the right", 113, c.right}, {"the content box", 70, c.content}} {
				if px := at(p.x, 50); p.shows && !isRed(px) || !p.shows && !isWhite(px) {
					t.Errorf("%s at %v: %s is %v", c.name, scale, p.what, px)
				}
			}
			if px := at(10, 50); !isWhite(px) {
				t.Errorf("%s at %v: outside the box, %v", c.name, scale, px)
			}
			if px := at(21, 50); c.name == "padding box" && !isRed(px) {
				t.Errorf("%s at %v: the border, outside the padding box, is %v", c.name, scale, px)
			}
		}
	}
}

// The boxes of layers are rounded as CSS rounds them inside a border and
// padding: a rounded box's ring is as wide at its corners.
func TestMaskLayersAreRounded(t *testing.T) {
	black := RGB(0, 0, 0)
	solid := LinearGradient{From: black, To: black}
	img := coreRender(func(cx *context) {
		coreBox(cx).Fill().Background(RGB(255, 255, 255)).Padding(20).Children(func() {
			coreBox(cx).Size(100, 60).Radius(20).Padding(10).Border(2, RGB(255, 0, 0)).Background(RGB(255, 0, 0)).
				Masks(Mask{Image: solid, Box: MaskContentBox, Composite: MaskExclude}, Mask{Image: solid})
		})
	}, 140, 100, 1)
	// The content box, 32 to 108 across, has corners 8 round.
	if px := img.RGBAAt(33, 33); !isRed(px) {
		t.Errorf("in the ring at the content box's rounded corner: %v", px)
	}
	if px := img.RGBAAt(42, 42); !isWhite(px) {
		t.Errorf("in the content box, past its corner: %v", px)
	}
	if px := img.RGBAAt(21, 21); !isWhite(px) {
		t.Errorf("outside the box's rounded corner: %v", px)
	}
}

// Masks of one gradient across the box mask as MaskLinearGradient and the
// others do, which replace the layers, and the mask of layers is made
// once for each size.
func TestMaskLayersAreKept(t *testing.T) {
	black := RGB(0, 0, 0)
	solid := LinearGradient{From: black, To: black}
	layers := []Mask{{Image: solid, Box: MaskContentBox, Composite: MaskExclude}, {Image: solid}}
	var e *node
	one := false
	tt := coreNewTester(func(c *context) {
		e = coreBox(c).Size(100, 60).Padding(10).Background(RGB(255, 0, 0)).Masks(layers...)
		if one {
			e.MaskConicGradient(ConicGradient{Stops: []GradientStop{{black, 0}, {black, 1}}})
		}
	}, 140, 100)
	if len(tt.rt.pathClips) != 1 {
		t.Fatalf("%d masks for one element", len(tt.rt.pathClips))
	}
	var img *scene.Image
	for _, c := range tt.rt.pathClips {
		img = c.img
	}
	tt.Frame()
	for _, c := range tt.rt.pathClips {
		if c.img != img {
			t.Error("the mask was made again")
		}
	}
	tt.SetScale(2)
	tt.Frame()
	wide := false
	for _, c := range tt.rt.pathClips {
		wide = wide || c.img.W == 200 && c.used == tt.rt.paints
	}
	if !wide {
		t.Error("at twice the scale, no mask twice as wide")
	}
	one = true
	tt.Frame()
	if k := e.clipping; k.mask != maskConic || k.layers != nil {
		t.Errorf("MaskConicGradient left %v and %d layers", k.mask, len(k.layers))
	}
	layers = layers[1:]
	one = false
	tt.Frame()
	if k := e.clipping; k.mask != maskLinear || k.layers != nil {
		t.Errorf("one layer across the box makes %v and %d layers", k.mask, len(k.layers))
	}
}

// Painter.ClipPath cuts what is drawn to a path's inside, smoothed at its
// edges, under a transform too, and clips nest.
func TestClipPath(t *testing.T) {
	triangle := MustParsePath("M10 10 L110 10 L10 110 Z")
	for _, scale := range []float32{1, 2} {
		img := coreRender(func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(p *Painter, r Rect) {
				p.ClipPath(triangle, func() {
					p.Fill(Rect{0, 0, 120, 120}, RGB(255, 0, 0), 0)
				})
				p.Transform(Translation(120, 0), 0, 0, func() {
					p.ClipPath(triangle, func() {
						p.Clip(Rect{0, 0, 40, 120}, 0, func() {
							p.Fill(Rect{0, 0, 120, 120}, RGB(255, 0, 0), 0)
						})
					})
				})
			})
		}, 240, 120, scale)
		s := int(scale)
		at := func(x, y int) color.RGBA { return img.RGBAAt(x*s, y*s) }
		if c := at(30, 30); !isRed(c) {
			t.Errorf("%vx: inside the path: %v", scale, c)
		}
		if c := at(90, 90); !isWhite(c) {
			t.Errorf("%vx: outside the path: %v", scale, c)
		}
		if c := at(5, 50); !isWhite(c) {
			t.Errorf("%vx: left of the path: %v", scale, c)
		}
		if c := at(150, 30); !isRed(c) {
			t.Errorf("%vx: moved, inside both clips: %v", scale, c)
		}
		if c := at(170, 20); !isWhite(c) {
			t.Errorf("%vx: moved, inside the path, outside the clip in it: %v", scale, c)
		}
		// On the slanted edge, smoothed.
		edge := img.RGBAAt(60*s, 60*s-1)
		if isRed(edge) || isWhite(edge) {
			t.Errorf("%vx: on the edge: %v, want part of it", scale, edge)
		}
	}
}

// A path's clip mask is made once for its shape: the frames after find it.
func TestClipPathMaskIsKept(t *testing.T) {
	var tri Path
	tri.MoveTo(10, 10).LineTo(110, 10).LineTo(10, 110).Close()
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Draw(func(p *Painter, r Rect) {
			p.ClipPath(&tri, func() { p.Fill(r, RGB(255, 0, 0), 0) })
		})
	}, 120, 120)
	img := func() any {
		for _, c := range tt.rt.pathClips {
			return c.img
		}
		return nil
	}
	first := img()
	if first == nil || len(tt.rt.pathClips) != 1 {
		t.Fatalf("%d masks kept", len(tt.rt.pathClips))
	}
	tt.Frame()
	tt.Frame()
	if img() != first || len(tt.rt.pathClips) != 1 {
		t.Error("the mask was made again")
	}
}

// ParsePath reads SVG path data: absolute and relative commands, implicit
// lines after a move, smooth curves, arcs, and numbers run together.
func TestParsePath(t *testing.T) {
	for _, c := range []struct {
		d    string
		want Rect
	}{
		{"M10 20 L30 40 H50 V60 Z", Rect{10, 20, 40, 40}},
		{"m10 20 l20 20 h20 v20 z", Rect{10, 20, 40, 40}},
		{"M0 0 10 10 20 0", Rect{0, 0, 20, 10}},
		{"M0,0L-5-5", Rect{-5, -5, 5, 5}},
		{"M0 0 C0 10 10 10 10 0 S20-10 20 0", Rect{0, -10, 20, 20}},
		// The cubic curves' control points, two thirds of the way to the
		// quadratic ones.
		{"M0 0 Q5 10 10 0 T20 0", Rect{0, -20.0 / 3, 20, 40.0 / 3}},
		{"M0 0 A10 10 0 0 1 20 0", Rect{0, -10, 20, 10}},
		{"M.5.5l1e1 0", Rect{0.5, 0.5, 10, 0}},
	} {
		p, err := ParsePath(c.d)
		if err != nil {
			t.Errorf("%q: %v", c.d, err)
			continue
		}
		b := p.Bounds()
		for i, v := range [4]float32{b.X - c.want.X, b.Y - c.want.Y, b.W - c.want.W, b.H - c.want.H} {
			if math.Abs(float64(v)) > 0.2 {
				t.Errorf("%q: bounds %v, want %v (%d)", c.d, b, c.want, i)
				break
			}
		}
	}
	if p, err := ParsePath(""); err != nil || len(p.cmds) != 0 {
		t.Errorf("empty data: %d commands, %v", len(p.cmds), err)
	}
	// In error, the path up to the error, as SVG draws it.
	for _, d := range []string{"M0 0 L10 10 L20", "M0 0 L10 10 x", "L10 10", "M0 0 L10 10 Z 5"} {
		p, err := ParsePath(d)
		if err == nil || !strings.HasPrefix(err.Error(), "svg: path data: ") {
			t.Errorf("%q: error %v", d, err)
			continue
		}
		if want := map[bool]int{true: 0, false: 2}[d[0] == 'L']; len(p.cmds) < want {
			t.Errorf("%q: %d commands before the error", d, len(p.cmds))
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustParsePath took data in error")
		}
	}()
	MustParsePath("M0 0 L")
}

// A path parsed fills, strokes and trims as one built.
func TestParsedPathDraws(t *testing.T) {
	built := (&Path{}).MoveTo(10, 10).LineTo(90, 10).CubeTo(90, 60, 40, 90, 10, 90).Close()
	parsed := MustParsePath("M10 10H90C90 60 40 90 10 90Z")
	render := func(p *Path) []byte {
		return coreRender(func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Draw(func(pt *Painter, r Rect) {
				pt.FillPath(p, RGB(0, 0, 255))
				pt.StrokePath(p.Trim(0, 0.5), 3, RGB(255, 0, 0))
			})
		}, 100, 100, 1).Pix
	}
	if a, b := render(built), render(parsed); string(a) != string(b) {
		t.Error("the parsed path draws apart from the same path built")
	}
}

// Balance wraps a text in as many lines as it would take, of about the
// same length: the last is not a word alone.
func TestBalancedText(t *testing.T) {
	const s = "Balanced headings wrap their words evenly over lines x"
	var plain, balanced *node
	coreNewTester(func(c *context) {
		coreColumn(c).Width(240).AlignItems(Start).Children(func() {
			plain = coreText(c, s).FontSize(20)
			balanced = coreText(c, s).FontSize(20).Balance()
		})
	}, 400, 400)
	pl, bl := plain.tl.Lines, balanced.tl.Lines
	if len(pl) < 2 || len(bl) != len(pl) {
		t.Fatalf("%d lines plain, %d balanced", len(pl), len(bl))
	}
	spread := func(lines []text.Line) float32 {
		lo, hi := float32(math.MaxFloat32), float32(0)
		for _, l := range lines {
			lo, hi = min(lo, l.Width), max(hi, l.Width)
		}
		return hi - lo
	}
	if spread(bl) >= spread(pl) {
		t.Errorf("balanced lines spread %v, plain ones %v", spread(bl), spread(pl))
	}
	if !balanced.tl.Params.Balance {
		t.Error("the layout is not balanced")
	}
}
