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
	if tt.rt.animating || tt.rt.repainting || tt.FramePending() || !tt.h.afterAt.IsZero() || !tt.h.repaintAt.IsZero() {
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

// TranslatePercent moves by fractions of the element's own box as the
// frame lays it out: in place in its first frame, following its size in
// the frame it changes, after its other transforms.
func TestTranslatePercent(t *testing.T) {
	w := float32(40)
	var tip *node
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(200, 200).Children(func() {
			tip = coreBox(c).Absolute().Left(100).Top(100).Size(w, 20).
				Scale(2, 2).TranslatePercent(-50, -50).TranslatePercent(0, -50).Translate(1, 0)
		})
	}, 200, 200)
	tt.Frame()
	// Scaled around its center, then moved by its own size: 20 left and
	// 20 up, whatever the scale, and 1 right.
	if b := tip.Bounds(); b.X != 100-20-20+1 || b.Y != 100-10-20 || b.W != 80 || b.H != 40 {
		t.Fatalf("in its first frame, it shows at %+v", b)
	}
	w = 80
	tt.Frame()
	if b := tip.Bounds(); b.X != 100-40-40+1 || b.W != 160 {
		t.Errorf("resized, it shows at %+v", b)
	}
	// Transform replaces it.
	var plain *node
	tt = coreNewTester(func(c *context) {
		plain = coreBox(c).Size(40, 20).TranslatePercent(-50, 0).Transform(Translation(3, 0))
	}, 200, 200)
	tt.Frame()
	if m := plain.st.world; m.X != 3 {
		t.Errorf("after Transform, it is moved by %v", m.X)
	}
}

// TranslationZ and Translation3D move content along Z, which Perspective
// after them shows nearer or farther, scaled about the origin by
// distance/(distance-z), as CSS's translateZ() and translate3d().
func TestTranslationAlongZ(t *testing.T) {
	if z := TranslationZ(0); z != (Transform{}) || z.set() {
		t.Errorf("TranslationZ(0) is %+v, not identity", z)
	}
	if m := Translation3D(3, 4, 0); m != Translation(3, 4) {
		t.Errorf("Translation3D(3, 4, 0) is %+v, not Translation(3, 4)", m)
	}
	want := identity4()
	want[3], want[7], want[11] = 5, 6, 7
	if m := Translation3D(5, 6, 7).mat4(); m != want {
		t.Errorf("Translation3D(5, 6, 7) is %v, not %v", m, want)
	}
	want = identity4()
	want[11] = -120
	if m := TranslationZ(-120).mat4(); m != want {
		t.Errorf("TranslationZ(-120) is %v, not %v", m, want)
	}
	// Without perspective, Z shows nothing: the plane only moves in x and y.
	if f := TranslationZ(50).flat(); f.Set {
		t.Errorf("TranslationZ(50) flattens to %+v, not identity", f)
	}
	if f := Translation3D(5, 6, 50).flat(); f != scene.Translation(5, 6) {
		t.Errorf("Translation3D(5, 6, 50) flattens to %+v", f)
	}
	near := func(got, want float32) bool {
		return math.Abs(float64(got-want)) <= 1e-5*max(1, math.Abs(float64(want)))
	}
	for _, c := range []struct{ x, y, z, d float32 }{{0, 0, -120, 800}, {0, 0, 200, 800}, {30, -10, -400, 600}, {12, 8, 100, 400}} {
		f := Translation3D(c.x, c.y, c.z).Then(Perspective(c.d)).flat()
		s := c.d / (c.d - c.z)
		if f.Perspective() || !near(f.A, s) || !near(f.D, s) || f.B != 0 || f.C != 0 || !near(f.X, c.x*s) || !near(f.Y, c.y*s) {
			t.Errorf("Translation3D(%v, %v, %v) seen from %v: %+v, not scaled by %v", c.x, c.y, c.z, c.d, f, s)
		}
		if x, y := f.Point(10, 20); !near(x, (10+c.x)*s) || !near(y, (20+c.y)*s) {
			t.Errorf("Translation3D(%v, %v, %v) seen from %v takes (10, 20) to %v, %v", c.x, c.y, c.z, c.d, x, y)
		}
	}
	// At or past the viewer, nothing shows.
	for _, z := range []float32{800, 900} {
		if f := TranslationZ(z).Then(Perspective(800)).flat(); f != (scene.Affine{Set: true}) {
			t.Errorf("TranslationZ(%v) seen from 800 flattens to %+v, not nothing", z, f)
		}
	}
	// Turned as well, there and back.
	f := TranslationZ(-150).Then(RotationY(30)).Then(RotationX(-20)).Then(Perspective(700)).flat()
	inv, ok := f.Inverse()
	if !ok {
		t.Fatal("no inverse")
	}
	if x, y := inv.Point(f.Point(37, -21)); !near(x, 37) || !near(y, -21) {
		t.Errorf("there and back, (37, -21) is %v, %v", x, y)
	}
	// On a carousel, items pushed out along Z and turned round face the
	// viewer by where they are seen from, as in CSS: one turned 80 degrees
	// on a ring of 300 seen from 800 shows its back, though it turns less
	// than a quarter turn.
	ring := func(deg float32) Transform { return TranslationZ(300).Then(RotationY(deg)).Then(Perspective(800)) }
	if !ring(80).facesAway() || ring(30).facesAway() || ring(0).facesAway() || RotationY(80).Then(Perspective(800)).facesAway() {
		t.Error("the items of a carousel that face away are not those whose backs the viewer sees")
	}
	if TranslationZ(-100).Then(Perspective(800)).facesAway() || TranslationZ(100).facesAway() {
		t.Error("moved along Z, the plane faces away")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("an infinite Z accepted")
			}
		}()
		TranslationZ(float32(math.Inf(1)))
	}()
}

// An element moved back along Z in perspective shows and takes the
// pointer smaller about its origin, and one moved past the viewer neither
// shows nor takes it.
func TestTranslateZShowsAndHitsInPerspective(t *testing.T) {
	z := float32(-200)
	var card *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(400, 300).Children(func() {
			card = coreBox(c).Key("card").Absolute().Left(100).Top(100).Size(200, 100).Background(RGB(255, 0, 0)).
				TranslateZ(z).Perspective(800)
			if card.Clicked() {
				clicks++
			}
		})
	}, 400, 300)
	// 800/1000 its size, about its center, (200, 150).
	if b := card.Bounds(); math.Abs(float64(b.X-120)) > 1e-3 || math.Abs(float64(b.Y-110)) > 1e-3 ||
		math.Abs(float64(b.W-160)) > 1e-3 || math.Abs(float64(b.H-80)) > 1e-3 {
		t.Fatalf("200 back, seen from 800, the card shows within %+v", b)
	}
	if card.LayoutBounds() != (Rect{100, 100, 200, 100}) {
		t.Errorf("Z changed layout: %v", card.LayoutBounds())
	}
	x, y := card.LocalToWindow(0, 0)
	closeFloat(t, x, 120)
	closeFloat(t, y, 110)
	lx, ly, ok := card.WindowToLocal(card.LocalToWindow(150, 60))
	if !ok || math.Abs(float64(lx-150)) > 1e-2 || math.Abs(float64(ly-60)) > 1e-2 {
		t.Fatalf("there and back, %v, %v", lx, ly)
	}
	img := tt.Image()
	if p := img.RGBAAt(200, 150); p.R < 200 || p.G > 50 {
		t.Errorf("its center shows as %v", p)
	}
	// Inside its layout box, outside what it shows.
	if p := img.RGBAAt(105, 105); p.G < 200 {
		t.Errorf("the card shows at its layout's corner: %v", p)
	}
	tt.ClickAt(105, 105)
	if clicks != 0 {
		t.Error("the card took a click outside what it shows")
	}
	tt.ClickAt(125, 115)
	if clicks != 1 {
		t.Errorf("a click inside what it shows: %d clicks", clicks)
	}
	// Moved as far as the viewer, it shows nothing and takes nothing.
	z = 800
	tt.Frame()
	if _, _, ok := card.WindowToLocal(200, 150); ok {
		t.Error("behind the viewer, the card has an inverse")
	}
	if p := tt.Image().RGBAAt(200, 150); p.G < 200 {
		t.Errorf("behind the viewer, the card shows as %v", p)
	}
	tt.ClickAt(200, 150)
	if clicks != 1 {
		t.Error("behind the viewer, the card took a click")
	}
}

// A turn in perspective around an origin far from the window's shows in
// front of the viewer, though the window's origin is then behind it, past
// the turned plane's horizon (#1406): from -89 to -70 degrees, a card 650
// DIPs down seen from 600 had every corner behind the viewer.
func TestTurnFarFromTheOriginStaysInFront(t *testing.T) {
	ox, oy := float32(62), float32(650)
	card := scene.Rect{X: 24, Y: 612, W: 76, H: 76}
	for _, deg := range []float32{-89, -80, -70, -60, -30, 0, 30, 60, 89} {
		tr := RotationX(deg).Then(Perspective(600))
		own := tr.flat()
		// Moved there in 3D, as an app may, and in 2D, as elements and
		// Painter.Transform do.
		for i, m := range []scene.Affine{
			Translation(-ox, -oy).Then(tr).Then(Translation(ox, oy)).flat(),
			scene.Translation(ox, oy).Mul(own).Mul(scene.Translation(-ox, -oy)),
		} {
			for _, p := range [][2]float32{{24, 612}, {100, 612}, {24, 688}, {100, 688}} {
				if w := m.W(p[0], p[1]); w <= 1e-6 {
					t.Errorf("%v° (%d): the corner (%v, %v) is behind the viewer, w %v", deg, i, p[0], p[1], w)
				}
				x, y := m.Point(p[0], p[1])
				wx, wy := own.Point(p[0]-ox, p[1]-oy)
				if math.Abs(float64(x-wx-ox)) > 0.05 || math.Abs(float64(y-wy-oy)) > 0.05 {
					t.Errorf("%v° (%d): the corner (%v, %v) shows at %v, %v, not %v, %v", deg, i, p[0], p[1], x, y, wx+ox, wy+oy)
				}
			}
			if b := m.Bounds(card); b.W > 2*card.W || b.H > card.H+1 {
				t.Errorf("%v° (%d): the card shows within %+v", deg, i, b)
			}
		}
	}
}

// A card near the foot of a clipped sidebar, turning over in perspective
// around its center, shows, takes the pointer, and clips its content to
// its own size, not the sidebar's (#1406, #1407).
func TestTurningCardFarFromTheOriginShowsAndHits(t *testing.T) {
	turn := float32(-75)
	var card *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(300, 800).Radius(12).Clip().Children(func() {
			card = coreBox(c).Key("card").Absolute().Left(24).Top(612).Size(76, 76).Radius(8).Clip().
				RotateX(turn).Perspective(600).Children(func() {
				coreBox(c).Size(76, 76).Background(RGB(255, 0, 0))
			})
			if card.Clicked() {
				clicks++
			}
		})
	}, 300, 800)
	for _, deg := range []float32{-85, -75, -65, 40} {
		turn = deg
		tt.Frame()
		if b := card.Bounds(); b.W > 2*76 || b.H > 77 || b.X < 0 || b.Y < 600 {
			t.Errorf("%v°: the card shows within %+v", deg, b)
		}
		clips := 0
		for _, op := range tt.h.last.Ops {
			if op.Kind != scene.OpPushClip || !op.Transform.Perspective() {
				continue
			}
			clips++
			if b := op.Transform.Bounds(op.Rect); b.W > 2*76 || b.H > 77 {
				t.Errorf("%v°: the card's clip covers %+v", deg, b)
			}
		}
		if clips == 0 {
			t.Errorf("%v°: the card's clip does not turn with it", deg)
		}
		// Its center stays where it is.
		if p := tt.Image().RGBAAt(62, 650); p.R < 200 || p.G > 50 {
			t.Errorf("%v°: the card's center shows as %v", deg, p)
		}
		lx, ly, ok := card.WindowToLocal(62, 650)
		if !ok || math.Abs(float64(lx-38)) > 0.05 || math.Abs(float64(ly-38)) > 0.05 {
			t.Errorf("%v°: the card's center is at %v, %v (%v) in it", deg, lx, ly, ok)
		}
		before := clicks
		tt.ClickAt(62, 650)
		if clicks != before+1 {
			t.Errorf("%v°: a click on the card's center missed it", deg)
		}
		// Above what it shows, inside its layout box, it takes nothing.
		tt.ClickAt(62, 614)
		if clicks != before+1 {
			t.Errorf("%v°: a click above what the card shows hit it", deg)
		}
	}
}
