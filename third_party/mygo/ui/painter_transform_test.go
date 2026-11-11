package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

// A card turned over in perspective around its center, near the foot of
// a tall window, shows, and a clip it draws in turns with it at its own
// size, not the window's (#1406): the window's origin is then behind the
// viewer, past the turned plane's horizon.
func TestPainterTransformTurnsFarFromTheOrigin(t *testing.T) {
	red := RGB(255, 0, 0)
	card := Rect{X: 24, Y: 612, W: 76, H: 76}
	for _, deg := range []float32{-85, -75, 40} {
		tt := coreNewTester(func(c *context) {
			coreBox(c).Size(300, 800).Draw(func(p *Painter, r Rect) {
				p.Transform(RotationX(deg).Then(Perspective(600)), 62, 650, func() {
					p.Clip(card, 8, func() { p.Fill(Rect{X: 0, Y: 0, W: 300, H: 800}, red, 0) })
				})
			})
		}, 300, 800)
		if c := tt.Image().RGBAAt(62, 650); c != (color.RGBA{255, 0, 0, 255}) {
			t.Errorf("%v°: the card's center shows as %v", deg, c)
		}
		if c := tt.Image().RGBAAt(62, 600); c == (color.RGBA{255, 0, 0, 255}) {
			t.Errorf("%v°: the card shows above itself", deg)
		}
		clips := 0
		for _, op := range tt.h.last.Ops {
			if op.Kind == scene.OpPushClip && op.Transform.Perspective() {
				clips++
				if b := op.Transform.Bounds(op.Rect); b.W > 2*card.W || b.H > card.H+1 {
					t.Errorf("%v°: the clip covers %+v", deg, b)
				}
			}
		}
		if clips != 1 {
			t.Errorf("%v°: %d clips turn with the card", deg, clips)
		}
	}
}

func TestPainterTransform(t *testing.T) {
	red, blue := RGB(220, 20, 30), RGB(30, 60, 220)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(200, 100).Draw(func(p *Painter, r Rect) {
			// Moved right 50: a red square at 50..70.
			p.Transform(Translation(50, 0), 0, 0, func() {
				p.Fill(Rect{X: r.X, Y: r.Y, W: 20, H: 20}, red, 0)
			})
			// A quarter turn around its center, nested in twice as large
			// around the corner (100, 0).
			p.Transform(Scaling(2, 2), r.X+100, r.Y, func() {
				p.Transform(Rotation(90), r.X+110, r.Y+10, func() {
					p.Fill(Rect{X: r.X + 100, Y: r.Y + 5, W: 20, H: 10}, blue, 0)
				})
			})
		})
	}, 200, 100)
	img := tt.Image()
	at := func(x, y int) color.RGBA { return img.RGBAAt(x, y) }
	if c := at(60, 10); c != (color.RGBA{220, 20, 30, 255}) {
		t.Errorf("translated square at 60 = %v", c)
	}
	if c := at(10, 10); c.R == 220 && c.G == 20 {
		t.Error("the square is still where it was drawn")
	}
	// The 20×10 bar turned upright around (110, 10) covers 105..115 by
	// 0..20, then twice as large around (100, 0): 110..130 by 0..40.
	if c := at(120, 30); c != (color.RGBA{30, 60, 220, 255}) {
		t.Errorf("scaled and turned bar at (120, 30) = %v", c)
	}
	if c := at(140, 10); c.B == 220 {
		t.Errorf("the bar reaches (140, 10): %v", c)
	}
}

// An image the transform moves out of view is culled; one it moves into
// view is drawn.
func TestPainterTransformCulls(t *testing.T) {
	bm := NewBitmap(stripedImage(4, 4, 1, 0))
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 100).Draw(func(p *Painter, r Rect) {
			p.Transform(Translation(-1000, 0), 0, 0, func() {
				p.ImageRegion(bm, Rect{}, Rect{X: r.X, Y: r.Y, W: 10, H: 10}, ImageOptions{})
			})
			p.Transform(Translation(1000, 0), 0, 0, func() {
				p.ImageRegion(bm, Rect{}, Rect{X: r.X - 1000, Y: r.Y, W: 10, H: 10}, ImageOptions{})
			})
		})
	}, 100, 100)
	images := 0
	for _, op := range tt.h.last.Ops {
		if op.Image != nil {
			images++
		}
	}
	if images != 1 {
		t.Errorf("%d images drawn, want the one in view", images)
	}
}

// Under a transform, Visible is in the coordinates drawn in: an element
// zoomed to twice its size, 400 square in a window 200 square, sees its
// top left quarter; one zoomed to half its size sees all of itself, and
// past it what the window shows; a painter's own transform likewise.
func TestPainterVisibleTransformed(t *testing.T) {
	var big, small, drawn Rect
	coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(0).Top(0).Size(200, 200).TransformOrigin(0, 0).Scale(2, 2).Draw(func(p *Painter, r Rect) {
			big = p.Visible()
		})
		coreBox(c).Absolute().Left(0).Top(0).Size(200, 200).TransformOrigin(0, 0).Scale(0.5, 0.5).Draw(func(p *Painter, r Rect) {
			small = p.Visible()
			p.Transform(Scaling(4, 4), 0, 0, func() { drawn = p.Visible() })
		})
	}, 200, 200)
	if want := (Rect{0, 0, 100, 100}); big != want {
		t.Errorf("zoomed in, visible %+v, want %+v", big, want)
	}
	if want := (Rect{0, 0, 400, 400}); small != want {
		t.Errorf("zoomed out, visible %+v, want %+v", small, want)
	}
	if want := (Rect{0, 0, 100, 100}); drawn != want {
		t.Errorf("zoomed out then in by the painter, visible %+v, want %+v", drawn, want)
	}
}
