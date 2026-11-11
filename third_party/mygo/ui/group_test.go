package ui

import (
	"image"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// near reports whether the pixel of img at x, y is c, within a unit or two
// of rounding.
func nearPixel(t *testing.T, img *image.RGBA, x, y int, c [3]int) {
	t.Helper()
	p := img.RGBAAt(x, y)
	got := [3]int{int(p.R), int(p.G), int(p.B)}
	for i := range got {
		if d := got[i] - c[i]; d < -2 || d > 2 {
			t.Fatalf("pixel at %d, %d is %v, want %v", x, y, got, c)
		}
	}
}

// layered reports whether the scene of the frame draws a layer.
func layered(tt *Tester) bool {
	for _, op := range tt.rt.scene.Ops {
		if op.Kind == scene.OpImage && op.Image != nil && op.Image.Scene != nil {
			return true
		}
	}
	return false
}

// TestOpacityGroup is the issue's view: two boxes overlapping inside a
// column at half opacity show as one, as CSS's group opacity has them,
// where they overlap as where they do not.
func TestOpacityGroup(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	img := coreRender(func(c *context) {
		coreColumn(c).Background(white).Children(func() {
			coreColumn(c).Padding(20).Background(white).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(40, 40).Background(red)
				coreBox(c).Absolute().Left(20).Top(0).Size(40, 40).Background(red)
			}).Opacity(0.5)
		})
	}, 100, 80, 1)
	nearPixel(t, img, 25, 20, [3]int{255, 127, 127}) // where they overlap
	nearPixel(t, img, 5, 20, [3]int{255, 127, 127})
	nearPixel(t, img, 55, 20, [3]int{255, 127, 127})
	nearPixel(t, img, 80, 60, [3]int{255, 255, 255})
}

// TestOpacityGroupFlat checks that what fades without overlapping paints
// flat, each thing faded on its own, as it shows the same: after the first
// frame, which tells, no layer.
func TestOpacityGroupFlat(t *testing.T) {
	red := RGB(255, 0, 0)
	moved := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 80).Children(func() {
			coreBox(c).Absolute().Left(0).Top(0).Size(40, 40).Background(red)
			x := float32(50)
			if moved {
				x = 20
			}
			coreBox(c).Absolute().Left(x).Top(0).Size(40, 40).Background(red)
		}).Opacity(0.5)
	}, 100, 80)
	tt.Frame()
	if layered(tt) {
		t.Fatal("boxes side by side painted into a layer")
	}
	nearPixel(t, tt.Image(), 25, 20, [3]int{255, 127, 127})
	// Moved over the other, the box overlaps it: painted flat first, as
	// the frame before was, the group is painted again into its layer.
	moved = true
	tt.Frame()
	if !layered(tt) {
		t.Fatal("overlapping boxes painted flat")
	}
	nearPixel(t, tt.Image(), 25, 20, [3]int{255, 127, 127})
	if tt.FramePending() {
		t.Fatal("a group kept asking for frames")
	}
	moved = false
	tt.Frame()
	tt.Frame()
	if layered(tt) {
		t.Fatal("boxes apart again stayed in a layer")
	}
}

// TestOpacityGroupOpaque checks that an element at full opacity, or one
// without children, paints no layer.
func TestOpacityGroupOpaque(t *testing.T) {
	red := RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreBox(c).Size(40, 40).Background(red).Border(4, RGB(0, 0, 255)).Opacity(0.5)
			coreColumn(c).Background(red).Children(func() {
				coreBox(c).Size(40, 40).Background(RGB(0, 0, 255))
			}).Opacity(1)
		})
	}, 100, 100)
	if layered(tt) {
		t.Fatal("painted a layer")
	}
}

// TestOpacityGroupCache checks that a kept layer (Cache) fades as one too:
// its opacity goes on the bitmap, not into it.
func TestOpacityGroupCache(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	img := coreRender(func(c *context) {
		coreColumn(c).Background(white).Size(100, 80).Children(func() {
			coreBox(c).Size(60, 40).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(40, 40).Background(red)
				coreBox(c).Absolute().Left(20).Top(0).Size(40, 40).Background(red)
			}).Cache().Opacity(0.5)
		})
	}, 100, 80, 1)
	nearPixel(t, img, 25, 20, [3]int{255, 127, 127})
	nearPixel(t, img, 5, 20, [3]int{255, 127, 127})
}

// TestOverlapping checks the bounds that tell a group to paint flat.
func TestOverlapping(t *testing.T) {
	s := &scene.Scene{}
	all := scene.Rect{W: 100, H: 100}
	fill := func(x float32) scene.Op { return scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: x, W: 10, H: 10}} }
	if over, flat := overlapping(s, []scene.Op{fill(0), fill(10), fill(20)}, all); over || flat {
		t.Error("boxes side by side overlap")
	}
	if over, _ := overlapping(s, []scene.Op{fill(0), {Kind: scene.OpPushClip, Rect: scene.Rect{W: 100, H: 100}}, fill(5), {Kind: scene.OpPopClip}}, all); !over {
		t.Error("boxes overlapping do not")
	}
	if _, flat := overlapping(s, []scene.Op{fill(0), {Kind: scene.OpHole, Rect: scene.Rect{X: 50, W: 10, H: 10}}}, all); !flat {
		t.Error("a web view's hole goes into a layer")
	}
	many := make([]scene.Op, groupOps+1)
	for i := range many {
		many[i] = fill(float32(i) * 10)
	}
	if over, _ := overlapping(s, many, scene.Rect{W: 1e4, H: 10}); !over {
		t.Errorf("more than %d ops are not taken as overlapping", groupOps)
	}
	// Out of the clips, as painting skips what is out of them, what
	// overlaps counts for nothing.
	clip := scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{W: 8, H: 10}}
	if over, _ := overlapping(s, []scene.Op{clip, fill(0), fill(9), {Kind: scene.OpPopClip}}, all); over {
		t.Error("boxes overlapping out of their clip overlap")
	}
	if over, _ := overlapping(s, []scene.Op{fill(0), fill(5)}, scene.Rect{W: 4, H: 10}); over {
		t.Error("boxes overlapping out of the clip overlap")
	}
}

// TestOpacityGroupDraws checks a group whose Draw callback paints over
// the element's background, painted flat first and then taken back: the
// callback paints into the layer, once more, and the parts it asked to
// paint again are the layer's, not the scene's.
func TestOpacityGroupDraws(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Background(white).Size(100, 80).Children(func() {
			coreBox(c).Size(60, 40).Background(red).Children(func() {
				coreBox(c).Size(40, 40).Draw(func(p *Painter, r Rect) {
					p.Fill(r, red, 0)
					p.AnimationFrame()
				})
			}).Opacity(0.5)
		})
	}, 100, 80)
	tt.Frame()
	if !layered(tt) {
		t.Fatal("a drawing over its background painted flat")
	}
	if len(tt.rt.drawn) != 0 {
		t.Fatalf("%d parts of the scene drawn in a layer", len(tt.rt.drawn))
	}
	nearPixel(t, tt.Image(), 20, 20, [3]int{255, 127, 127})
	nearPixel(t, tt.Image(), 50, 20, [3]int{255, 127, 127})
}

// TestOpacityGroupTurned checks a turned group: its layer is drawn turned,
// at its opacity as one.
func TestOpacityGroupTurned(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	img := coreRender(func(c *context) {
		coreColumn(c).Background(white).Size(200, 200).Children(func() {
			coreBox(c).Absolute().Left(50).Top(50).Size(100, 100).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(100, 100).Background(red)
				coreBox(c).Absolute().Left(25).Top(25).Size(50, 50).Background(red)
			}).Rotate(45).Opacity(0.5)
		})
	}, 200, 200, 1)
	nearPixel(t, img, 100, 100, [3]int{255, 127, 127})
	nearPixel(t, img, 100, 60, [3]int{255, 127, 127})
	nearPixel(t, img, 5, 5, [3]int{255, 255, 255})
}

// TestOpacityGroupFadedOut checks that a group painting into its layer
// shows nothing once faded out, as an Exit or a pose's Fade ends. It drew
// its layer at an opacity of 0, which an op takes for 1: the copy of a
// popup closing, or a page fading away, came back whole at the end.
func TestOpacityGroupFadedOut(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	opacity := float32(0.5)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Background(white).Size(100, 80).Children(func() {
			coreBox(c).Size(60, 40).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(40, 40).Background(red)
				coreBox(c).Absolute().Left(20).Top(0).Size(40, 40).Background(red)
			}).Opacity(opacity)
		})
	}, 100, 80)
	tt.Frame()
	if !layered(tt) {
		t.Fatal("overlapping boxes painted flat")
	}
	opacity = 0
	tt.Frame()
	nearPixel(t, tt.Image(), 25, 20, [3]int{255, 255, 255})
	nearPixel(t, tt.Image(), 5, 20, [3]int{255, 255, 255})
}

// painted reports whether some pixel of img within r is not white.
func painted(img *image.RGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if p := img.RGBAAt(x, y); p.R < 250 || p.G < 250 || p.B < 250 {
				return true
			}
		}
	}
	return false
}

// TestOpacityGroupHoldsBlur checks that a group's layer holds the blur of
// what is inside it, beyond the boxes: it was cut at their edges, which
// showed a square around a word's glow faded with it.
func TestOpacityGroupHoldsBlur(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(200, 100).Background(white).Children(func() {
			coreBox(c).Absolute().Left(60).Top(30).Size(80, 40).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(40, 40).Background(red)
				coreBox(c).Absolute().Left(20).Top(0).Size(40, 40).Background(red)
				coreBox(c).Absolute().Left(40).Top(0).Size(40, 40).Background(red).Blur(6)
			}).Opacity(0.5)
		})
	}, 200, 100)
	for range 2 {
		tt.Frame()
		if !layered(tt) {
			t.Fatal("overlapping boxes painted flat")
		}
		// 8 DIPs right of the boxes, within the blur's 18.
		if !painted(tt.Image(), image.Rect(146, 40, 150, 60)) {
			t.Fatal("the blur was cut at the edge of the boxes")
		}
	}
}

// TestOpacityGroupHoldsTextShadow checks that a group's layer holds a text's
// shadow cast beyond the boxes.
func TestOpacityGroupHoldsTextShadow(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(200, 100).Background(white).Children(func() {
			coreBox(c).Absolute().Left(20).Top(30).Size(40, 40).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(40, 40).Background(red)
				coreText(c, "Hi").Absolute().Left(0).Top(0).FontSize(20).TextShadow(80, 0, 0, RGB(0, 0, 255))
			}).Opacity(0.5)
		})
	}, 200, 100)
	tt.Frame()
	if !layered(tt) {
		t.Fatal("the text over the box painted flat")
	}
	if !painted(tt.Image(), image.Rect(100, 30, 140, 70)) {
		t.Fatal("the text's shadow beyond the boxes was cut")
	}
}

// TestCacheHoldsBlur checks that a kept layer (Cache) holds the blur of what
// is inside it, beyond the boxes.
func TestCacheHoldsBlur(t *testing.T) {
	white, red := RGB(255, 255, 255), RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(200, 100).Background(white).Children(func() {
			coreBox(c).Absolute().Left(60).Top(30).Size(40, 40).Children(func() {
				coreBox(c).Size(40, 40).Background(red).Blur(6)
			}).Cache()
		})
	}, 200, 100)
	for range 2 {
		tt.Frame()
		if !painted(tt.Image(), image.Rect(106, 40, 110, 60)) {
			t.Fatal("the blur was cut at the edge of the boxes")
		}
	}
}
