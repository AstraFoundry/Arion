package ui

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A chart in a scroll container zooms with a mouse wheel and pinches, and
// pans with a touchpad's scrolling, with where the pointer is over it:
// the container scrolls with neither.
func TestChartZoomsAndPans(t *testing.T) {
	var page ScrollState
	scale, panX := float32(1), float32(0)
	var at [2]float32
	tt := NewTester(func(c *Context) {
		Scroll(c).Fill().TrackScroll(&page).Children(func() {
			Box(c).Height(100).Shrink(0)
			Box(c).Size(200, 100).Shrink(0).HandleInput(func(ev InputEvent) bool {
				switch {
				case ev.Kind == InputZoom:
					scale *= ev.Scale
				case ev.Kind == InputScroll && !ev.Precise:
					if ev.DY < 0 {
						scale *= 1.25
					} else if ev.DY > 0 {
						scale *= 0.8
					}
				case ev.Kind == InputScroll:
					panX += ev.DX
				default:
					return false
				}
				at = [2]float32{ev.X, ev.Y}
				return true
			})
			Box(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	defer tt.rt.close()
	tt.Frame()

	tt.Wheel(50, 130, 0, -40)
	tt.Advance(wheelAnimation.Duration)
	if scale != 1.25 || page.Y != 0 {
		t.Errorf("a notch up zoomed to %v and scrolled the page to %v, want 1.25 and 0", scale, page.Y)
	}
	if at != [2]float32{50, 30} {
		t.Errorf("the wheel was at %v of the chart, want [50 30]", at)
	}
	tt.Pinch(150, 160, 2)
	if scale != 2.5 || at != [2]float32{150, 60} {
		t.Errorf("a pinch zoomed to %v at %v, want 2.5 at [150 60]", scale, at)
	}
	tt.Scroll(60, 150, 30, 5)
	if panX != 30 || page.Y != 0 {
		t.Errorf("the touchpad panned %v and scrolled the page to %v, want 30 and 0", panX, page.Y)
	}

	// Over the rest of the page, the touchpad scrolls the page, and a
	// pinch zooms nothing.
	tt.Scroll(50, 50, 0, 40)
	tt.Pinch(50, 20, 3)
	tt.Frame()
	if page.Y != 40 || scale != 2.5 {
		t.Errorf("over the page: scrolled to %v, zoomed to %v; want 40 and 2.5", page.Y, scale)
	}
}

// An element that leaves scrolling to the scroll container still gets
// the pinches it takes, and a pinch that zooms by 1 comes to no element.
func TestZoomLeftToContainer(t *testing.T) {
	var page ScrollState
	var zooms []float32
	tt := NewTester(func(c *Context) {
		Scroll(c).Fill().TrackScroll(&page).Children(func() {
			Box(c).Size(200, 100).Shrink(0).HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputZoom {
					zooms = append(zooms, ev.Scale)
					return true
				}
				return false
			})
			Box(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	defer tt.rt.close()
	tt.Frame()
	tt.Scroll(50, 50, 0, 30)
	tt.Pinch(50, 20, 0.5)
	tt.Pinch(50, 20, 1) // no zoom
	tt.Frame()
	if page.Y != 30 || len(zooms) != 1 || zooms[0] != 0.5 {
		t.Errorf("page scrolled to %v, zooms %v; want 30 and [0.5]", page.Y, zooms)
	}
}

// zoomText is a line of text, laid out in a box of fixed size.
func zoomText(c *context) {
	coreColumn(c).Absolute().Top(0).Left(0).Width(160).Height(30).Padding(6).Children(func() {
		coreText(c, "Hamburgefonstiv 0123").FontSize(13)
	})
}

// sharpness sums the squared differences of horizontally neighboring
// pixels' luminance: blurrier text has softer edges, and less of it.
func sharpness(m *image.RGBA) float64 {
	b := m.Bounds()
	lum := func(x, y int) float64 {
		i := m.PixOffset(x, y)
		return 0.2126*float64(m.Pix[i]) + 0.7152*float64(m.Pix[i+1]) + 0.0722*float64(m.Pix[i+2])
	}
	var s float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X + 1; x < b.Max.X; x++ {
			d := lum(x, y) - lum(x-1, y)
			s += d * d
		}
	}
	return s
}

// TestZoomDrawsAtTheScreensResolutionTimesTheZoom checks that text zoomed
// 2 times draws as text at twice the scale does, pixel for pixel, and
// sharper than the same text scaled up by a transform.
func TestZoomDrawsAtTheScreensResolutionTimesTheZoom(t *testing.T) {
	native := coreNewTester(zoomText, 160, 30)
	native.SetScale(2)
	zoomed := coreNewTester(zoomText, 320, 60)
	zoomed.SetZoom(2)
	a, b := native.Image(), zoomed.Image()
	if a.Bounds() != b.Bounds() || a.Bounds().Dx() != 320 {
		t.Fatalf("frames of %v and %v, want 320×60", a.Bounds(), b.Bounds())
	}
	differ, maxDiff := 0, 0
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		if d < 0 {
			d = -d
		}
		if d > 0 {
			differ++
		}
		maxDiff = max(maxDiff, d)
	}
	if differ > 0 {
		t.Errorf("zoomed 2 times, %d of %d components differ from scale 2's, by up to %d", differ, len(a.Pix), maxDiff)
	}

	scaled := coreNewTester(func(c *context) {
		coreColumn(c).Absolute().Top(0).Left(0).Width(160).Height(30).TransformOrigin(0, 0).Scale(2, 2).Children(func() { zoomText(c) })
	}, 320, 60)
	z, s := sharpness(b), sharpness(scaled.Image())
	t.Logf("sharpness zoomed %.0f, scaled by a transform %.0f", z, s)
	if z < s*1.2 {
		t.Errorf("zoomed text's sharpness %.0f, scaled up by a transform %.0f: want it clearly sharper", z, s)
	}
	// Unzoomed, nothing changes.
	once := coreNewTester(zoomText, 160, 30)
	once.SetZoom(1)
	plain := coreNewTester(zoomText, 160, 30)
	if !identicalImage(once.Image(), plain.Image()) {
		t.Error("zoom 1 draws otherwise than no zoom")
	}
}

// identicalImage reports whether a and b have the same size and pixels.
func identicalImage(a, b *image.RGBA) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

// TestZoomHitTesting checks that the pointer hits, in the window, the
// elements as the zoom shows them, and that the view sees the content's
// size and title bar.
func TestZoomHitTesting(t *testing.T) {
	var (
		clicked    string
		w, h       float32
		bar        TitleBar
		zoom       float32
		scrolled   float32
		scrollArea ScrollState
	)
	view := func(c *context) {
		w, h = c.Size()
		bar, zoom = c.TitleBar(), c.Zoom()
		coreRow(c).Children(func() {
			if coreBox(c).Size(50, 50).Label("A").Clicked() {
				clicked = "A"
			}
			coreBox(c).Size(10, 50)
			if coreBox(c).Size(50, 50).Label("B").Clicked() {
				clicked = "B"
			}
		})
		coreScroll(c).TrackScroll(&scrollArea).Absolute().Top(60).Left(0).Size(100, 40).Children(func() {
			coreBox(c).Size(100, 400)
		})
		scrolled = scrollArea.Y
	}
	tt := coreNewTester(view, 400, 200)
	tt.SetTitleBar(TitleBar{Height: 28, Left: 72, Right: 10})
	tt.SetZoom(2)
	if w != 200 || h != 100 || zoom != 2 {
		t.Errorf("zoomed 2 times, the view sees %v×%v at zoom %v, want 200×100 at 2", w, h, zoom)
	}
	if bar != (TitleBar{Height: 14, Left: 36, Right: 5}) {
		t.Errorf("TitleBar = %+v, want the window's halved", bar)
	}
	if r, _ := tt.Find("B"); r != (Rect{120, 0, 100, 100}) {
		t.Errorf("Find(B) = %v, want the window's box {120 0 100 100}", r)
	}
	// (170, 50) is past both boxes in the content, and on B zoomed.
	tt.ClickAt(170, 50)
	if clicked != "B" {
		t.Fatalf("a click at (170, 50) hit %q, want B", clicked)
	}
	tt.ClickAt(90, 90)
	if clicked != "A" {
		t.Fatalf("a click at (90, 90) hit %q, want A", clicked)
	}
	if err := tt.Click("B"); err != nil || clicked != "B" {
		t.Fatalf("Click(B): %v, hit %q", err, clicked)
	}
	// A touchpad scrolls the content as far as the fingers move in the
	// window: 40 DIPs of it are 20 of the content.
	tt.send(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 100, Y: 160, DY: 40, Precise: true})
	if scrolled != 20 {
		t.Errorf("40 DIPs of a touchpad scrolled the content %v, want 20", scrolled)
	}
	tt.SetZoom(1)
	if w != 400 || h != 200 {
		t.Errorf("back at 1, the view sees %v×%v", w, h)
	}
	clicked = ""
	tt.ClickAt(170, 50)
	if clicked != "" {
		t.Errorf("unzoomed, a click at (170, 50) hit %q", clicked)
	}
}

// TestZoomSetByTheView checks that a view zooming its content builds the
// frame at the new size at once.
func TestZoomSetByTheView(t *testing.T) {
	z := float32(1)
	var sizes [][2]float32
	tt := coreNewTester(func(c *context) {
		c.SetZoom(z)
		w, h := c.Size()
		sizes = append(sizes, [2]float32{w, h})
		coreBox(c).Size(10, 10)
	}, 300, 200)
	sizes = nil
	z = 1.5
	tt.Frame()
	if last := sizes[len(sizes)-1]; last != [2]float32{200, 200 / 1.5} {
		t.Errorf("zoomed 1.5 times, the frame's last pass saw %v", last)
	}
	if b := tt.Image().Bounds(); b.Dx() != 300 || b.Dy() != 200 {
		t.Errorf("the zoomed frame is %v, want the window's 300×200 pixels", b)
	}
	z = 0
	tt.Frame()
	if c := tt.rt.zoomFactor(); c != 1 {
		t.Errorf("SetZoom(0) left zoom %v, want 1", c)
	}
}

// TestZoomTellsTheWindowsCoordinates checks what goes to the window in its
// DIPs: input methods' caret, assistive technology's boxes and context
// menus' places.
func TestZoomTellsTheWindowsCoordinates(t *testing.T) {
	value := "Ada"
	page := &testWebView{"page"}
	view := func(c *context) {
		coreColumn(c).Padding(10).Gap(10).Children(func() {
			coreTextInput(c, &value).Width(120).Label("Name").AutoFocus()
			coreRow(c).Padding(8).Children(func() { coreText(c, "Report") }).ContextMenu(func(m *Menu) { m.Item("Open") })
			coreBox(c).Size(30, 20).MaximizeButton()
			coreWebView(c, page).Size(100, 40).Children(func() {
				coreText(c, "Badge").Absolute().Top(4).Left(4)
			})
		})
	}
	var maxButton [3]Rect
	var webView [3]placedWebView
	caret := func(z float32) (Rect, platform.RectF, [2]float32) {
		tt := coreNewTester(view, 400, 400)
		tt.SetZoom(z)
		tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
		r, ok := tt.TextCaret()
		if !ok {
			t.Fatalf("zoom %v: no caret", z)
		}
		field := accessNode(t, tt.h.access, platform.RoleTextField, "Name")
		maxButton[int(z)] = tt.h.maxButton
		webView[int(z)], _ = placed(tt, page)
		if err := tt.RightClick("Report"); err != nil {
			t.Fatal(err)
		}
		return r, field.Bounds, tt.h.menuAt
	}
	c1, f1, m1 := caret(1)
	c2, f2, m2 := caret(2)
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	if !near(float64(c2.X), 2*float64(c1.X)) || !near(float64(c2.Y), 2*float64(c1.Y)) || !near(float64(c2.H), 2*float64(c1.H)) {
		t.Errorf("caret %v zoomed 2 times, %v unzoomed", c2, c1)
	}
	if !near(f2.X, 2*f1.X) || !near(f2.Y, 2*f1.Y) || !near(f2.W, 2*f1.W) || !near(f2.H, 2*f1.H) {
		t.Errorf("text field's accessibility box %+v zoomed 2 times, %+v unzoomed", f2, f1)
	}
	if !near(float64(m2[0]), 2*float64(m1[0])) || !near(float64(m2[1]), 2*float64(m1[1])) {
		t.Errorf("context menu at %v zoomed 2 times, %v unzoomed", m2, m1)
	}
	if b1, b2 := maxButton[1], maxButton[2]; b1.W == 0 || b2 != (Rect{2 * b1.X, 2 * b1.Y, 2 * b1.W, 2 * b1.H}) {
		t.Errorf("maximize button at %v zoomed 2 times, %v unzoomed", b2, b1)
	}
	w1, w2 := webView[1], webView[2]
	// Lines snap to pixels in the zoomed layout as in the unzoomed one, so
	// with a platform's own fonts (Core Text) what lies below text can land
	// up to a DIP of the window from twice its unzoomed place.
	doubled := func(z, r platform.RectF) bool {
		within := func(a, b float64) bool { return math.Abs(a-2*b) <= 1 }
		return within(z.X, r.X) && within(z.Y, r.Y) && within(z.W, r.W) && within(z.H, r.H)
	}
	if w1.frame.W == 0 || !doubled(w2.frame, w1.frame) || !doubled(w2.clip, w1.clip) ||
		len(w1.covers) != 1 || len(w2.covers) != 1 || !doubled(w2.covers[0], w1.covers[0]) {
		t.Errorf("web view placed at %+v zoomed 2 times, %+v unzoomed", w2, w1)
	}
}

// Scrolling over an element whose handler leaves it goes on to the
// handler around it, as it would without the inner one: an editor scrolls
// under the boxes it shows over its text, and a scroll container around
// both scrolls only when neither takes it.
func TestScrollPassesToOuterHandler(t *testing.T) {
	var page ScrollState
	var outer float32
	take := true
	tt := NewTester(func(c *Context) {
		Scroll(c).Fill().TrackScroll(&page).Children(func() {
			Box(c).Size(200, 150).Shrink(0).HandleInput(func(ev InputEvent) bool {
				if ev.Kind != InputScroll || !take {
					return false
				}
				outer += ev.DY
				return true
			}).Children(func() {
				Box(c).Size(100, 50).HandleInput(func(InputEvent) bool { return false })
			})
			Box(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	defer tt.rt.close()
	tt.Frame()

	tt.Scroll(20, 20, 0, 30)
	tt.Frame()
	if outer != 30 || page.Y != 0 {
		t.Fatalf("over the inner box: outer handler scrolled %v, page %v; want 30, 0", outer, page.Y)
	}
	take = false
	tt.Scroll(20, 20, 0, 30)
	tt.Frame()
	if outer != 30 || page.Y != 30 {
		t.Fatalf("neither handler taking it: outer %v, page %v; want 30, 30", outer, page.Y)
	}
}

// A handler around a scroll container sees the wheel before the container
// scrolls, as a canvas that zooms with Ctrl and the wheel does, and the
// container scrolls when the handler leaves it.
func TestScrollHandlerAroundContainer(t *testing.T) {
	var inner ScrollState
	var zoomed float32
	tt := NewTester(func(c *Context) {
		Box(c).Fill().HandleInput(func(ev InputEvent) bool {
			if ev.Kind != InputScroll || ev.Mods&Ctrl == 0 {
				return false
			}
			zoomed += ev.DY
			return true
		}).Children(func() {
			Scroll(c).Size(200, 200).TrackScroll(&inner).Children(func() {
				Box(c).Height(1000).Shrink(0)
			})
		})
	}, 200, 200)
	defer tt.rt.close()
	tt.Frame()

	tt.send(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 20, Y: 20, DY: 30, Precise: true, Mods: platform.Modifiers(Ctrl)})
	tt.Frame()
	if zoomed != 30 || inner.Y != 0 {
		t.Fatalf("Ctrl and the wheel: handler took %v, container scrolled %v; want 30, 0", zoomed, inner.Y)
	}
	tt.Scroll(20, 20, 0, 30)
	tt.Frame()
	if zoomed != 30 || inner.Y != 30 {
		t.Fatalf("the wheel alone: handler took %v, container scrolled %v; want 30, 30", zoomed, inner.Y)
	}
}
