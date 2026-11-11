package game

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// stepper records the steps a View runs.
type stepper struct {
	steps           int
	wDown, wPressed []bool
	spacePressed    int
	wReleased       int
	clicks          int
	focused         bool
	alpha           float64
	draws           int
}

func (s *stepper) Update(in *Input) {
	s.steps++
	s.wDown = append(s.wDown, in.Down(ui.KeyW))
	s.wPressed = append(s.wPressed, in.Pressed(ui.KeyW))
	if in.Pressed(ui.KeySpace) {
		s.spacePressed++
	}
	if in.Released(ui.KeyW) {
		s.wReleased++
	}
	if in.PointerPressed(0) {
		s.clicks++
	}
	s.focused = in.Focused()
}

func (s *stepper) Draw(p *ui.Painter, r ui.Rect, alpha float64) {
	s.alpha, s.draws = alpha, s.draws+1
}

// gameTester tests a game in a view whose frames are at the test's
// clock, now.
func gameTester(t *testing.T, g Game, now *time.Time) *ui.Tester {
	before := frameTime
	frameTime = func(*ui.Painter) time.Time { return *now }
	t.Cleanup(func() { frameTime = before })
	tt := ui.NewTester(func(c *ui.Context) { View(c, g).Size(200, 100) }, 200, 100)
	tt.Frame()
	return tt
}

func TestView(t *testing.T) {
	g := &stepper{}
	now := time.Unix(1000, 0)
	tt := gameTester(t, g, &now)
	if g.steps != 0 || g.draws == 0 {
		t.Fatalf("first frame: %d steps, %d draws", g.steps, g.draws)
	}
	// 100 ms are 6 steps, and a third of the next.
	now = now.Add(105 * time.Millisecond)
	tt.Frame()
	if g.steps != 6 || !g.focused {
		t.Errorf("%d steps in 105 ms, focused %v", g.steps, g.focused)
	}
	if math.Abs(g.alpha-0.3) > 0.01 {
		t.Errorf("alpha %v", g.alpha)
	}
	// Painting again at the same time runs no step.
	tt.Frame()
	if g.steps != 6 {
		t.Errorf("%d steps after painting twice", g.steps)
	}
	// W held: down in every step, pressed in the first only.
	g.wDown, g.wPressed = nil, nil
	tt.KeyDown(ui.KeyW)
	now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if len(g.wDown) != 3 || !g.wDown[0] || !g.wDown[2] || !g.wPressed[0] || g.wPressed[1] {
		t.Errorf("W held: down %v, pressed %v", g.wDown, g.wPressed)
	}
	// Space tapped between two steps: pressed in the next.
	tt.Key(0, ui.KeySpace)
	now = now.Add(20 * time.Millisecond)
	tt.Frame()
	if g.spacePressed != 1 {
		t.Errorf("Space pressed in %d steps", g.spacePressed)
	}
	// The window losing the keyboard lets go of W.
	tt.SetFocused(false)
	now = now.Add(20 * time.Millisecond)
	tt.Frame()
	if g.focused {
		t.Error("focused without the keyboard")
	}
	if g.wReleased != 1 || g.wDown[len(g.wDown)-1] {
		t.Errorf("W after losing the keyboard: released %d, down %v", g.wReleased, g.wDown[len(g.wDown)-1])
	}
	tt.SetFocused(true)
	// A click.
	tt.ClickAt(10, 10)
	now = now.Add(20 * time.Millisecond)
	tt.Frame()
	if g.clicks != 1 {
		t.Errorf("%d clicks", g.clicks)
	}
	// After a long pause, a quarter of a second of steps, not all.
	before := g.steps
	now = now.Add(10 * time.Second)
	tt.Frame()
	if n := g.steps - before; n != 15 {
		t.Errorf("%d steps after 10 s", n)
	}
}

// fastGame runs at its own rate.
type fastGame struct{ stepper }

func (fastGame) Rate() int { return 120 }

func TestRate(t *testing.T) {
	g := &fastGame{}
	now := time.Unix(1000, 0)
	tt := gameTester(t, g, &now)
	now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if g.steps != 12 {
		t.Errorf("%d steps in 100 ms at 120 a second", g.steps)
	}
}

var red = ui.RGB(255, 0, 0)

// paint renders what draw paints into a 400×300 view.
func paint(scale float32, draw func(p *ui.Painter, r ui.Rect)) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) { ui.Box(c).Size(400, 300).Draw(draw) }, 400, 300)
	tt.SetScale(scale)
	tt.Frame()
	return tt
}

func isRed(c color.RGBA) bool { return c.R > 250 && c.G < 5 && c.B < 5 }

func TestCamera(t *testing.T) {
	r := ui.Rect{X: 50, Y: 20, W: 400, H: 300}
	cam := Camera{X: 100, Y: 80, Zoom: 2}
	// The camera's point shows at the middle.
	if x, y := cam.ToWorld(r, 200, 150); x != 100 || y != 80 {
		t.Errorf("middle = %v, %v", x, y)
	}
	if x, y := cam.ToWorld(r, 0, 0); x != 0 || y != 5 {
		t.Errorf("corner = %v, %v", x, y)
	}
	if v := cam.Visible(r); v != (ui.Rect{X: 0, Y: 5, W: 200, H: 150}) {
		t.Errorf("visible = %v", v)
	}
	// Transform maps the world back to where ToWorld found it: the world's
	// point at the view's top-left corner paints there.
	img := paint(1, func(p *ui.Painter, r ui.Rect) {
		wx, wy := cam.ToWorld(r, 0, 0)
		p.Transform(cam.Transform(r), 0, 0, func() {
			p.Fill(ui.Rect{X: wx, Y: wy, W: 2, H: 2}, red, 0)
		})
	}).Image()
	if !isRed(img.RGBAAt(1, 1)) || isRed(img.RGBAAt(6, 6)) {
		t.Errorf("world corner: %v at 1, 1, %v at 6, 6", img.RGBAAt(1, 1), img.RGBAAt(6, 6))
	}
	turned := Camera{Rotation: 90}
	img = paint(1, func(p *ui.Painter, r ui.Rect) {
		wx, wy := turned.ToWorld(r, 300, 150)
		p.Transform(turned.Transform(r), 0, 0, func() {
			p.Fill(ui.Rect{X: wx - 2, Y: wy - 2, W: 4, H: 4}, red, 0)
		})
	}).Image()
	if !isRed(img.RGBAAt(300, 150)) || isRed(img.RGBAAt(310, 150)) {
		t.Errorf("turned: %v at 300, 150, %v at 310, 150", img.RGBAAt(300, 150), img.RGBAAt(310, 150))
	}
}

func TestIntegerFit(t *testing.T) {
	var dst, small ui.Rect
	var k, smallK int
	paint(2, func(p *ui.Painter, r ui.Rect) {
		// 1000×600 DIPs at 2×: 2000×1200 pixels, for 480×270: 4 times (1920×1080).
		dst, k = IntegerFit(p, 480, 270, ui.Rect{X: 10, Y: 20, W: 1000, H: 600})
		// Smaller than the canvas: one pixel each, overflowing.
		small, smallK = IntegerFit(p, 480, 270, ui.Rect{W: 100, H: 100})
	})
	if k != 4 || dst != (ui.Rect{X: 30, Y: 50, W: 960, H: 540}) {
		t.Errorf("fit %v, k %d", dst, k)
	}
	if smallK != 1 || small.W != 240 {
		t.Errorf("fit %v, k %d in a small view", small, smallK)
	}
}

// Through an unturned camera, the world moves by whole pixels: a tile at
// whole world units has no edge pixel partly covered.
func TestCameraSnaps(t *testing.T) {
	img := paint(2, func(p *ui.Painter, r ui.Rect) {
		Camera{X: 10.3, Y: 7.77, Zoom: 3}.Draw(p, r, func() {
			p.Fill(ui.Rect{X: 0, Y: 0, W: 16, H: 16}, red, 0)
		})
	}).Image()
	bg := img.RGBAAt(799, 599)
	reds := 0
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			c := img.RGBAAt(x, y)
			if isRed(c) {
				reds++
			} else if c != bg {
				t.Fatalf("pixel %d, %d partly covered: %v", x, y, c)
			}
		}
	}
	if reds != 96*96 {
		t.Errorf("%d red pixels", reds)
	}
}

// A modifier on its own is no key of a game's, and is left to the window, as
// Alt to the menu bar.
func TestModifierKeyAlone(t *testing.T) {
	l := &loop{in: newInput()}
	for _, ev := range []ui.InputEvent{
		{Kind: ui.InputKeyDown, Code: ui.CodeShiftLeft, Mods: ui.Shift},
		{Kind: ui.InputKeyUp, Code: ui.CodeShiftLeft},
		{Kind: ui.InputKeyUp, Code: ui.CodeAltLeft},
	} {
		if l.input(ev) {
			t.Errorf("took %+v", ev)
		}
	}
	if l.in.Down(ui.KeyUnknown) || l.in.Pressed(ui.KeyUnknown) || l.in.Released(ui.KeyUnknown) {
		t.Error("a key of no name is held")
	}
}
