package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/transfer"
	"image/color"
	"math"
	"testing"
	"time"
)

func closeFloat(t *testing.T, got, want float32) {
	t.Helper()
	if math.Abs(float64(got-want)) > 0.01 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTransformRenderingAndHitCoordinates(t *testing.T) {
	clicks := 0
	var box *node
	var events []InputEvent
	tt := coreNewTester(func(c *context) {
		box = coreBox(c).Key("target").Absolute().Left(60).Top(50).Size(80, 30).Background(RGB(220, 20, 30)).Rotate(45)
		if box.Clicked() {
			clicks++
		}
		box.HandleInput(func(ev InputEvent) bool { events = append(events, ev); return false })
	}, 220, 180)
	x, y := box.LocalToWindow(15, 10)
	if c := tt.Image().RGBAAt(int(x), int(y)); c.R != 220 || c.G != 20 {
		t.Fatalf("transformed interior: %v", c)
	}
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Fatalf("transformed press: %d clicks", clicks)
	}
	if len(events) < 2 {
		t.Fatal("no pointer events")
	}
	closeFloat(t, events[0].X, 15)
	closeFloat(t, events[0].Y, 10)
	r := box.Bounds()
	// A point inside the bounding rectangle but outside the rotated shape.
	tt.ClickAt(r.X+1, r.Y+1)
	if clicks != 1 {
		t.Fatal("bounding rectangle took a click outside the rotated box")
	}
	lx, ly, ok := box.WindowToLocal(x, y)
	if !ok {
		t.Fatal("no inverse")
	}
	closeFloat(t, lx, 15)
	closeFloat(t, ly, 10)
	if box.LayoutBounds() != (Rect{60, 50, 80, 30}) {
		t.Fatalf("transform changed layout: %v", box.LayoutBounds())
	}
}

func TestNestedTransformsComposeAndReflect(t *testing.T) {
	var child *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(40).Top(30).Size(80, 80).TransformOrigin(0, 0).Scale(2, 2).Translate(10, 5).Children(func() {
			child = coreBox(c).Absolute().Left(10).Top(15).Size(20, 10).TransformOrigin(0, 0).Scale(-1, 1).Rotate(90).Background(RGB(10, 120, 220))
			if child.Clicked() {
				clicks++
			}
		})
	}, 250, 220)
	// Child (5,4) reflects to (-5,4), rotates to (-4,-5), then
	// parent's scale and translation move (46,40) to (62,55).
	x, y := child.LocalToWindow(5, 4)
	closeFloat(t, x, 62)
	closeFloat(t, y, 55)
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Fatal("nested reflected transform did not hit")
	}
	if c := tt.Image().RGBAAt(int(x), int(y)); c.B != 220 {
		t.Fatalf("nested transform drew %v", c)
	}
}

func TestTransformedNestedClips(t *testing.T) {
	var parent, child *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		parent = coreBox(c).Absolute().Left(50).Top(50).Size(80, 60).Rotate(30).Radius(12).Clip()
		parent.Children(func() {
			coreBox(c).Size(50, 50).Clip().Scale(1.2, 0.8).Children(func() {
				child = coreBox(c).Size(100, 80).Shrink(0).Background(RGB(230, 0, 0))
				if child.Clicked() {
					clicks++
				}
			})
		})
	}, 220, 200)
	x, y := child.LocalToWindow(20, 20)
	ttt := tt.Image().RGBAAt(int(x), int(y))
	if ttt.R != 230 || ttt.G != 0 {
		t.Fatalf("clipped interior: %v", ttt)
	}
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Fatal("clipped interior did not hit")
	}
	x, y = child.LocalToWindow(75, 20)
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Fatal("hit escaped nested clip")
	}
	if c := tt.Image().RGBAAt(int(x), int(y)); c.R == 230 && c.G == 0 {
		t.Fatal("drawing escaped nested clip")
	}
}

func TestTransformCaretAccessibilityAndAnchor(t *testing.T) {
	value := "hello"
	open := true
	var input, panel *node
	tt := coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(80).Top(50).Size(130, 60).Scale(1.5, 1.2).Rotate(20).Children(func() {
			input = coreTextInput(c, &value).Width(100).AutoFocus().Label("transformed input")
		})
		panel = corePopover(c, input, &open, func() { coreText(c, "anchored") })
	}, 500, 300)
	s := input.st
	r := transformRect(s.world, s.editor.caretRect(s))
	caret, active := tt.TextCaret()
	if !active {
		t.Fatal("transformed input has no caret")
	}
	closeFloat(t, caret.X, r.X)
	closeFloat(t, caret.Y, r.Y)
	closeFloat(t, caret.W, r.W)
	closeFloat(t, caret.H, r.H)
	drawnCaret := false
	for _, op := range tt.h.last.Ops {
		if op.Kind == scene.OpFill && op.Rect.W == 1 && op.Color == tt.rt.c.theme.Accent.scene() {
			drawnCaret = true
			if op.Transform != s.world.Pixels(tt.h.scale) {
				t.Fatal("painted caret did not follow the input's transform")
			}
		}
	}
	if !drawnCaret {
		t.Fatal("no painted caret")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	n := accessNode(t, tt.h.access, platform.RoleTextField, "transformed input")
	b := input.Bounds()
	closeFloat(t, float32(n.Bounds.X), b.X)
	closeFloat(t, float32(n.Bounds.Y), b.Y)
	closeFloat(t, float32(n.Bounds.W), b.W)
	if panel.Bounds().Y < b.Y+b.H {
		t.Fatalf("overlay %v overlaps transformed anchor %v", panel.Bounds(), b)
	}
	if panel.world.Set {
		t.Fatal("overlay inherited anchor's rotation")
	}
}

func TestTransformedTextInputClientGeometry(t *testing.T) {
	client := &primitiveClient{text: "hello", selection: TextInputSelection{Range: TextInputRange{Start: 3, End: 3}}}
	var e *node
	xScale := float32(1.5)
	tt := coreNewTester(func(c *context) {
		e = coreBox(c).Absolute().Left(80).Top(60).Size(100, 40).TransformOrigin(0, 0).Scale(xScale, 1.2).Rotate(30).HandleTextInput(client).AutoFocus()
	}, 400, 250)
	defer tt.rt.close()
	native := tt.h.ime.Client
	for _, r := range []TextInputRange{{Start: 3, End: 3}, {Start: 1, End: 4}} {
		local, _, _ := client.BoundsForRange(r)
		want := transformRect(e.world, Rect{X: e.x + local.X, Y: e.y + local.Y, W: local.W, H: local.H})
		got, actual, ok := native.BoundsForRange(r)
		if !ok || actual != r {
			t.Fatal("native range geometry unavailable", actual, ok)
		}
		closeFloat(t, float32(got.X), want.X)
		closeFloat(t, float32(got.Y), want.Y)
		closeFloat(t, float32(got.W), want.W)
		closeFloat(t, float32(got.H), want.H)
	}
	x, y := e.LocalToWindow(18, 15)
	if index, ok := native.IndexForPoint(float64(x), float64(y)); !ok || index != 4 {
		t.Fatal("transformed native hit test", index, ok)
	}
	xScale = 0
	tt.Frame()
	if _, ok := native.IndexForPoint(float64(x), float64(y)); ok {
		t.Fatal("collapsed native client accepted a hit")
	}
}

func TestTransformedContainerTextSelection(t *testing.T) {
	var first, second *node
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Absolute().Left(100).Top(70).Width(160).Rotate(30).Selectable().Children(func() {
			first = coreText(c, "Alpha beta")
			second = coreText(c, "Gamma delta")
		})
	}, 400, 250)
	defer tt.rt.close()
	point := func(e *node, at int) (float32, float32) {
		ed := e.st.editor
		x, y, h := ed.layout.Caret(at)
		return e.LocalToWindow(ed.originX+x, ed.originY+y+h/2)
	}
	x, y := point(first, 2)
	tt.Press(x, y)
	x, y = point(second, 4)
	tt.Move(x, y)
	tt.Release(x, y)
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "pha beta\nGamm" {
		t.Fatalf("transformed selection copied %q", got)
	}
}

func TestTransformedNativeDragPreviewHotspot(t *testing.T) {
	var e *node
	tt := coreNewTester(func(c *context) {
		e = coreBox(c).Absolute().Left(80).Top(60).Size(80, 40).Scale(1.5, 1.2).Rotate(30).Background(RGB(230, 20, 30)).DragData(transfer.TextData("drag"))
	}, 400, 250)
	defer tt.rt.close()
	x, y := e.LocalToWindow(15, 15)
	tt.Press(x, y)
	tt.Move(x+10, y+10)
	o := tt.h.dragOptions
	if o.Preview == nil {
		t.Fatal("transformed source has no native preview")
	}
	b := e.Bounds()
	if o.Hotspot.X != int(x-b.X) || o.Hotspot.Y != int(y-b.Y) {
		t.Fatal("native preview hotspot lost the press", o.Hotspot, b)
	}
	got := color.RGBAModel.Convert(o.Preview.At(o.Hotspot.X, o.Hotspot.Y)).(color.RGBA)
	if got.R != 230 || got.G != 20 || got.B != 30 {
		t.Fatal("native preview does not show the transformed source at the press", got)
	}
}

func TestTransformedCustomCaretAndCapture(t *testing.T) {
	var e *node
	var got InputEvent
	tt := coreNewTester(func(c *context) {
		e = coreBox(c).Absolute().Left(60).Top(70).Size(50, 40).Scale(2, 1).Rotate(-25).Focusable().AutoFocus().TextCaret(Rect{10, 5, 1, 16})
		e.HandleInput(func(ev InputEvent) bool { got = ev; return true })
	}, 300, 220)
	caret, active := tt.TextCaret()
	if !active {
		t.Fatal("no custom caret")
	}
	want := transformRect(e.world, Rect{e.x + 10, e.y + 5, 1, 16})
	closeFloat(t, caret.X, want.X)
	closeFloat(t, caret.W, want.W)
	x, y := e.LocalToWindow(10, 10)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y)})
	x, y = e.LocalToWindow(70, 50)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x), Y: float64(y)})
	closeFloat(t, got.X, 70)
	closeFloat(t, got.Y, 50)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y)})
	if got.Kind != InputPointerUp {
		t.Fatal("captured transformed element lost release")
	}
}

func TestSingularTransformIsInvisibleAndDoesNotHit(t *testing.T) {
	var e *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		e = coreBox(c).Size(100, 50).Scale(0, 1).Background(RGB(255, 0, 0))
		if e.Clicked() {
			clicks++
		}
	}, 200, 100)
	tt.ClickAt(50, 25)
	if clicks != 0 {
		t.Fatal("collapsed transform took a click")
	}
	if _, _, ok := e.WindowToLocal(50, 25); ok {
		t.Fatal("singular transform has an inverse")
	}
	if tt.Image().RGBAAt(50, 25) == (color.RGBA{255, 0, 0, 255}) {
		t.Fatal("collapsed transform drew")
	}
}

func TestTransformUpdatesDamageAndTransitionGeometry(t *testing.T) {
	x := float32(0)
	angle := float32(0)
	var e *node
	tt, now := clockTester(func(c *context) {
		e = coreBox(c).Absolute().Left(x).Top(40).Size(50, 30).Rotate(angle).Background(RGB(240, 0, 0)).Transition(ElementTransition{Duration: time.Second, Ease: Linear})
	}, 250, 160)
	x, angle = 100, 45
	tt.Frame()
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	closeFloat(t, e.LayoutBounds().X, 50)
	px, py := e.LocalToWindow(25, 15)
	if got := tt.Image().RGBAAt(int(px), int(py)); got.R != 240 {
		t.Fatalf("transition and transform drew %v", got)
	}
	if got := tt.Image().RGBAAt(10, 50); got.R == 240 && got.G == 0 {
		t.Fatal("damage left the previous position painted")
	}
}

func TestInlineTransformMovesGlyphsAndInteraction(t *testing.T) {
	var link *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreText(c, "Before ").Absolute().Left(30).Top(30).FontSize(22).Children(func() {
			link = coreText(c, "LINK").TextColor(RGB(230, 0, 0)).TextBackground(RGB(255, 240, 0)).Underline().Translate(50, 30).Rotate(8)
			if link.Clicked() {
				clicks++
			}
		})
	}, 300, 180)
	x, y := link.LocalToWindow(link.st.w/2, link.st.h/2)
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Fatal("transformed inline text did not hit")
	}
	r := link.Bounds()
	img := tt.Image()
	colored := false
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			p := img.RGBAAt(x, y)
			if p.R > 180 && p.G < 80 {
				colored = true
			}
		}
	}
	if !colored {
		t.Fatal("inline glyphs did not follow their transform")
	}
}

func TestTransformedViewStaysIdle(t *testing.T) {
	tt := coreNewTester(func(c *context) { coreBox(c).Size(80, 40).Rotate(10).Translate(50, 30).Background(RGB(20, 80, 120)) }, 200, 120)
	if tt.rt.animating || tt.rt.repainting || tt.h.requested.Load() || tt.rt.timer != nil || tt.rt.repaintTimer != nil {
		t.Fatal("static transformed content scheduled work")
	}
}

func TestTransformsIn3D(t *testing.T) {
	turn := float32(40)
	var card, back *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(400, 300).Children(func() {
			card = coreBox(c).Key("card").Absolute().Left(100).Top(100).Size(200, 100).Background(RGB(255, 0, 0)).
				RotateY(turn).Perspective(400).BackfaceHidden()
			back = coreBox(c).Key("back").Absolute().Left(100).Top(100).Size(200, 100).Background(RGB(0, 0, 255)).
				RotateY(turn + 180).Perspective(400).BackfaceHidden()
			if card.Clicked() {
				clicks++
			}
		})
	}, 400, 300)
	b := card.Bounds()
	// Turned away on its right, the card is narrower, and its right edge
	// shorter than its left.
	if b.W >= 200 || b.W < 100 || b.H <= 100 {
		t.Fatalf("turned, the card shows within %v", b)
	}
	lx, ly, ok := card.WindowToLocal(card.LocalToWindow(150, 60))
	if !ok || math.Abs(float64(lx-150)) > 1e-2 || math.Abs(float64(ly-60)) > 1e-2 {
		t.Fatalf("there and back, %v, %v", lx, ly)
	}
	_, y0 := card.LocalToWindow(0, 0)
	_, y1 := card.LocalToWindow(200, 0)
	if y1 <= y0 {
		t.Errorf("the far edge is not shorter: tops at %v and %v", y0, y1)
	}
	// The front shows and takes the pointer; the back, turned away, neither.
	x, y := card.LocalToWindow(190, 50)
	if p := tt.Image().RGBAAt(int(x), int(y)); p.R < 200 || p.B > 50 {
		t.Errorf("the front shows as %v", p)
	}
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Errorf("a click on the far side of the front: %d clicks", clicks)
	}
	// Past 90 degrees, the back shows instead.
	turn = 140
	tt.Frame()
	x, y = back.LocalToWindow(100, 50)
	if p := tt.Image().RGBAAt(int(x), int(y)); p.B < 200 || p.R > 50 {
		t.Errorf("turned over, the back shows as %v", p)
	}
	tt.ClickAt(x, y)
	if clicks != 1 {
		t.Errorf("turned away, the front took a click")
	}
}

func TestSkewAndPerspectiveTransforms(t *testing.T) {
	m := Skewing(45, 0).flat()
	if x, y := m.Point(0, 10); math.Abs(float64(x-10)) > 1e-4 || y != 10 {
		t.Errorf("skewed 45 degrees, (0, 10) goes to %v, %v", x, y)
	}
	// Turned about the vertical axis without perspective, it only looks
	// narrower.
	if f := RotationY(60).flat(); f.Perspective() || math.Abs(float64(f.A-0.5)) > 1e-6 {
		t.Errorf("turned flat: %+v", f)
	}
	// A turn there and back is nothing.
	if f := RotationX(30).Then(Perspective(500)).Then(Translation(5, 5)).flat(); !f.Perspective() {
		t.Error("in perspective, the map is affine")
	}
	id := RotationY(25).Then(RotationY(-25)).flat()
	if id.Set && (math.Abs(float64(id.A-1)) > 1e-5 || math.Abs(float64(id.G)) > 1e-6) {
		t.Errorf("turned there and back: %+v", id)
	}
	if !RotationY(120).facesAway() || RotationY(60).facesAway() || Scaling(-1, 1).facesAway() || RotationX(-100).Then(Perspective(300)).facesAway() == false {
		t.Error("the faces turned away are not those past 90 degrees")
	}
	for _, f := range []func(){func() { Perspective(0) }, func() { Skewing(float32(math.NaN()), 0) }} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid transform accepted")
				}
			}()
			f()
		}()
	}
}
