package ui

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/surface"
)

// testWebView stands for a *mygo.WebView in the tests of ui.
type testWebView struct{ name string }

func (*testWebView) SurfaceWebView(*surface.Conn) platform.WebView { return nil }

// placed returns where the last frame showed v, if it did.
func placed(tt *Tester, v NativeWebView) (placedWebView, bool) {
	for _, p := range tt.h.webViews {
		if p.v == v {
			return p, true
		}
	}
	return placedWebView{}, false
}

// A web view shows in its element's content box, under what paints after
// it, which takes the pointer there: a popover over it, its own children.
func TestWebViewPlacement(t *testing.T) {
	page := &testWebView{"page"}
	open := false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().AlignItems(Stretch).Children(func() {
			menu := coreButton(c, "Menu")
			if menu.Clicked() {
				open = !open
			}
			corePopover(c, menu, &open, func() { coreText(c, "Item").Padding(4, 8) })
			coreWebView(c, page).Grow(1).Padding(10).Border(1, RGB(0, 0, 0)).Children(func() {
				coreText(c, "Badge").Absolute().Top(20).Right(20)
			})
		})
	}, 400, 300)
	p, ok := placed(tt, page)
	if !ok {
		t.Fatal("the web view does not show")
	}
	if p.frame != p.clip || p.frame.X != 11 || p.frame.W != 378 || p.frame.Y+p.frame.H != 289 {
		t.Errorf("the web view shows at %+v, clipped to %+v: want its content box", p.frame, p.clip)
	}
	badge, _ := tt.Find("Badge")
	if len(p.covers) != 1 || !p.covers[0].Contains(float64(badge.X+1), float64(badge.Y+1)) {
		t.Errorf("covers %+v: want the badge's box", p.covers)
	}
	if tt.WebViewAt(200, 200) != page || tt.WebViewAt(badge.X+1, badge.Y+1) != nil || tt.WebViewAt(5, 200) != nil {
		t.Error("the pointer goes to the web view where it is not, or not where it is")
	}

	// The popover's panel takes the pointer over the page.
	if err := tt.Click("Menu"); err != nil {
		t.Fatal(err)
	}
	item, ok := tt.Find("Item")
	if !ok {
		t.Fatal("the popover did not open")
	}
	if tt.WebViewAt(item.X+1, item.Y+1) != nil {
		t.Error("the page takes the pointer under the popover")
	}
	// A press on the page closes it, as presses outside a popover do.
	tt.ClickAt(200, 250)
	if open || tt.HasText("Item") {
		t.Error("a press on the page left the popover open")
	}
}

// The element paints a hole through what is under it, rounded as its
// corners inside its border, which what paints after it covers.
func TestWebViewHole(t *testing.T) {
	page := &testWebView{"page"}
	tt := coreNewTester(func(c *context) {
		c.Root().Background(RGB(255, 255, 255))
		coreBox(c).Fill().Padding(20).Children(func() {
			coreWebView(c, page).Grow(1).Radius(12).Border(2, RGB(0, 0, 0)).Children(func() {
				coreBox(c).Absolute().Top(10).Left(10).Size(20, 20).Background(RGB(255, 0, 0))
			})
		})
	}, 200, 140)
	var holes []scene.Op
	for _, op := range tt.rt.scene.Ops {
		if op.Kind == scene.OpHole {
			holes = append(holes, op)
		}
	}
	if len(holes) != 1 || holes[0].Rect != (scene.Rect{X: 22, Y: 22, W: 156, H: 96}) || holes[0].Radii[0] != 10 {
		t.Fatalf("holes %+v: want one in the content box, rounded by 10", holes)
	}
	img := tt.Image()
	at := func(x, y int) [4]uint8 {
		o := img.PixOffset(x, y)
		return [4]uint8(img.Pix[o : o+4])
	}
	if c := at(100, 70); c[3] != 0 {
		t.Errorf("the hole is %v: want transparent", c)
	}
	if c := at(22, 22); c[3] != 255 {
		t.Errorf("the hole's rounded corner is %v: want the background", c)
	}
	if c := at(40, 40); c != [4]uint8{255, 0, 0, 255} {
		t.Errorf("the box over the page is %v: want red", c)
	}
	if c := at(10, 10); c != [4]uint8{255, 255, 255, 255} {
		t.Errorf("the window around the web view is %v: want white", c)
	}
}

// A translucent web view blends over what is painted under it: its hole
// takes the opacity away from the pixels under it.
func TestWebViewOpacity(t *testing.T) {
	page := &testWebView{"page"}
	tt := coreNewTester(func(c *context) {
		c.Root().Background(RGB(255, 255, 255))
		coreBox(c).Fill().Padding(20).Opacity(0.5).Children(func() {
			coreWebView(c, page).Grow(1)
		})
	}, 200, 140)
	o := tt.Image().PixOffset(100, 70)
	if c := tt.Image().Pix[o : o+4]; c[3] < 126 || c[3] > 129 {
		t.Errorf("the hole of a web view at half opacity is %v: want half the white under it", c)
	}
	if places := tt.rt.webPlaces; len(places) != 1 {
		t.Errorf("placed %d web views: want the translucent one", len(places))
	}
}

// Tab stops at a web view to give its page the keyboard, from its end
// going back, and moves on from it as Tab leaves the page.
func TestWebViewTab(t *testing.T) {
	page := &testWebView{"page"}
	var before, after string
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Children(func() {
			coreTextInput(c, &before).Label("Before")
			coreWebView(c, page).Height(100)
			coreTextInput(c, &after).Label("After")
		})
	}, 200, 200)
	label := func() string {
		if s := tt.rt.states[tt.rt.focused]; s != nil && s.webView != nil {
			return "web view"
		}
		for _, l := range []string{"Before", "After"} {
			if tt.Focused(l) {
				return l
			}
		}
		return "nothing"
	}
	tt.Key(0, KeyTab)
	if got := label(); got != "Before" {
		t.Fatalf("Tab focused %q: want Before", got)
	}
	tt.Key(0, KeyTab)
	if label() != "web view" || tt.h.tabbedInto != page || tt.h.tabbedBack {
		t.Fatalf("Tab after Before focused %q and gave %v the keyboard (back %v): want the page", label(), tt.h.tabbedInto, tt.h.tabbedBack)
	}
	tt.rt.webViewTabOut(page, false)
	tt.Frame()
	if got := label(); got != "After" {
		t.Errorf("Tab out of the page focused %q: want After", got)
	}
	tt.h.tabbedInto = nil
	tt.Key(Shift, KeyTab)
	if label() != "web view" || tt.h.tabbedInto != page || !tt.h.tabbedBack {
		t.Fatalf("Shift+Tab from After focused %q and gave %v the keyboard (back %v): want the page, from its end", label(), tt.h.tabbedInto, tt.h.tabbedBack)
	}
	tt.rt.webViewTabOut(page, true)
	tt.Frame()
	if got := label(); got != "Before" {
		t.Errorf("Shift+Tab out of the page focused %q: want Before", got)
	}
	// Clicked into, the page tabs out to what follows its element.
	tt.rt.focused = 0
	tt.rt.webViewTabOut(page, false)
	tt.Frame()
	if got := label(); got != "After" {
		t.Errorf("Tab out of a page clicked into focused %q: want After", got)
	}
}

// A web view in a scroll container shows the part of it in view, and none
// once out of view or not built; a press on it takes the focus from the
// content, and the page sets the cursor over it.
func TestWebViewClipped(t *testing.T) {
	page := &testWebView{"page"}
	shown := true
	var text string
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().AlignItems(Stretch).Children(func() {
			coreTextInput(c, &text).Label("Name")
			coreScroll(c).Grow(1).Children(func() {
				coreBox(c).Height(100)
				if shown {
					coreWebView(c, page).Height(300)
				}
				coreBox(c).Height(400)
			})
		})
	}, 300, 300)
	p, ok := placed(tt, page)
	if !ok {
		t.Fatal("the web view does not show")
	}
	if p.frame.H != 300 || p.clip.Y+p.clip.H != 300 || p.clip.Y != p.frame.Y {
		t.Errorf("frame %+v clip %+v: want it clipped by the scroll container at the window's bottom", p.frame, p.clip)
	}
	// A press on the page takes the focus from the input.
	if err := tt.Click("Name"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("Name") {
		t.Fatal("the input did not take the focus")
	}
	tt.ClickAt(150, 200)
	if tt.Focused("Name") {
		t.Error("the input kept the focus as the page was pressed")
	}
	tt.Move(150, 250)
	if tt.rt.cursor != cursorUnknown {
		t.Errorf("the content set the cursor %v over the page", tt.rt.cursor)
	}
	tt.Scroll(150, 250, 0, 1000)
	if _, ok := placed(tt, page); ok {
		t.Error("the web view shows once scrolled out of view")
	}
	tt.Scroll(150, 250, 0, -1000)
	shown = false
	tt.Frame()
	if _, ok := placed(tt, page); ok {
		t.Error("the web view shows when not built")
	}
}

// Text over a web view is not on an opaque background, whatever opaque
// background is around the web view: subpixel glyphs would color the page.
func TestWebViewNotOpaque(t *testing.T) {
	opaque := map[string]bool{}
	probe := func(c *context, name string) *node {
		return coreBox(c).Size(20, 20).Draw(func(p *Painter, r Rect) { opaque[name] = p.opaque })
	}
	tt := coreNewTester(func(c *context) {
		c.Root().Background(RGB(255, 255, 255))
		coreColumn(c).Fill().AlignItems(Stretch).Children(func() {
			probe(c, "above")
			coreWebView(c, &testWebView{}).Grow(1).Padding(30).Background(RGB(255, 255, 255)).Children(func() {
				probe(c, "padding").Absolute().Top(5).Left(5)
				probe(c, "over")
				coreColumn(c).Background(RGB(255, 255, 255)).Children(func() { probe(c, "card") })
			})
		})
	}, 200, 200)
	tt.Frame()
	want := map[string]bool{"above": true, "padding": true, "over": false, "card": true}
	for name, w := range want {
		if opaque[name] != w {
			t.Errorf("%s: opaque %v, want %v", name, opaque[name], w)
		}
	}
}

// A press on a popover's anchor after a press elsewhere, as on a web
// view's page, before the next frame opens the popover: the last press
// decides.
func TestPopoverLastPressDecides(t *testing.T) {
	open, behind := false, 0
	tt := coreNewTester(func(c *context) {
		coreRow(c).Padding(10).Gap(100).AlignItems(Start).Children(func() {
			more := coreButtonBase(c).Size(60, 20).Label("more")
			if more.Clicked() {
				open = true
			}
			if coreButtonBase(c).Size(60, 20).Label("behind").Clicked() {
				behind++
			}
			corePopoverBase(c, more, &open, func(*node) { coreButtonBase(c).Size(80, 20).Label("item") })
		})
	}, 400, 300)
	press := func(r Rect) {
		x, y := float64(r.X+r.W/2), float64(r.Y+r.H/2)
		tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: x, Y: y})
		tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: x, Y: y})
	}
	more, _ := tt.Find("more")
	elsewhere, _ := tt.Find("behind")
	press(elsewhere)
	press(more)
	tt.Frame()
	if !open || behind != 1 {
		t.Fatalf("a press elsewhere then on the anchor: open %v, behind clicked %d times", open, behind)
	}
	// Then a press elsewhere closes it.
	press(more)
	press(elsewhere)
	tt.Frame()
	if open {
		t.Error("a press on the anchor then elsewhere left the popover open")
	}
}

// A web view in a panel scaling as one surface, as a peek zooming in,
// keeps its page's layout and shows it scaled: the placement carries the
// page's size, unchanged throughout the animation, and the corners of its
// hole, inside which alone the page takes the pointer; what the panel
// paints over the page covers it where it shows, scaled.
func TestWebViewScaledRounded(t *testing.T) {
	page := &testWebView{"page"}
	small := false
	tt := coreNewTester(func(c *context) {
		target := float32(1)
		if small {
			target = 0.5
		}
		scale := c.Root().AnimateTo("scale", target, Animation{Duration: 400 * time.Millisecond})
		coreBox(c).Absolute().Left(40).Top(40).Size(320, 200).Scale(scale, scale).Children(func() {
			coreWebView(c, page).Grow(1).Radius(24).Children(func() {
				coreBox(c).Absolute().Top(10).Right(10).Size(20, 20) // a button over the page
			})
		})
	}, 400, 280)
	check := func(scale float64) placedWebView {
		t.Helper()
		p, ok := placed(tt, page)
		if !ok {
			t.Fatalf("scaled %v: the web view does not show", scale)
		}
		if p.size != [2]float64{320, 200} {
			t.Errorf("scaled %v: the page lays out at %v: want 320 by 200, as unscaled", scale, p.size)
		}
		if p.radii != [4]float64{24, 24, 24, 24} {
			t.Errorf("scaled %v: the page's corners %v: want the hole's, 24", scale, p.radii)
		}
		near := func(a, b float64) bool { return math.Abs(a-b) < 1e-3 }
		want := platform.RectF{X: 200 - 160*scale, Y: 140 - 100*scale, W: 320 * scale, H: 200 * scale}
		if f := p.frame; !near(f.X, want.X) || !near(f.Y, want.Y) || !near(f.W, want.W) || !near(f.H, want.H) {
			t.Errorf("scaled %v: the page shows at %+v: want %+v", scale, f, want)
		}
		pl := p.placement(nil)
		if sx, sy := pl.Scale(); !near(sx, scale) || !near(sy, scale) {
			t.Errorf("scaled %v: the placement scales the page by %v, %v", scale, sx, sy)
		}
		return p
	}
	at := func(x, y float64) bool { return tt.WebViewAt(float32(x), float32(y)) == page }

	p := check(1)
	if !at(200, 140) || at(p.frame.X+3, p.frame.Y+3) || at(p.frame.X+p.frame.W-3, p.frame.Y+p.frame.H-3) || !at(p.frame.X+3, 140) {
		t.Error("unscaled, the page takes the pointer outside its rounded corners, or not inside them")
	}

	small = true
	tt.Frame()
	for range 4 {
		tt.Advance(100 * time.Millisecond)
		p, _ := placed(tt, page)
		check(p.frame.W / 320)
	}
	p = check(0.5)
	if !at(200, 140) || at(p.frame.X+2, p.frame.Y+2) || at(p.frame.X+p.frame.W-2, p.frame.Y+2) || !at(p.frame.X+2, 140) {
		t.Error("scaled, the page takes the pointer outside its rounded corners, or not inside them")
	}
	// The button: 20 by 20, 10 from the top-right corner, scaled by half
	// around the panel's center.
	if at(270, 100) || !at(250, 100) {
		t.Error("the button over the scaled page does not cover it where it shows")
	}
	// A press in a corner the page does not show in is the content's.
	tt.ClickAt(float32(p.frame.X+1), float32(p.frame.Y+1))
	if tt.WebViewAt(float32(p.frame.X+1), float32(p.frame.Y+1)) != nil {
		t.Error("a corner cut off takes the pointer to the page")
	}
}

// A key the page of a web view leaves unhandled acts as if the web view's
// element had the focus, which a press on the page gave it: Page Down
// scrolls the container around it, as a reader showing its document in
// native UI needs, and the shortcuts around it see the key.
func TestWebViewKey(t *testing.T) {
	page := &testWebView{"page"}
	var scroll ScrollState
	var text string
	nexts := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().AlignItems(Stretch).Children(func() {
			coreTextInput(c, &text).Label("Name")
			reader := coreScroll(c).Grow(1).TrackScroll(&scroll).Children(func() {
				coreWebView(c, page).Height(1000)
			})
			if reader.Shortcut(0, KeyJ) {
				nexts++
			}
		})
	}, 300, 300)
	tt.ClickAt(150, 200)
	if s := tt.rt.states[tt.rt.focused]; s == nil || s.webView != page || tt.rt.focusVisible {
		t.Fatal("a press on the page did not give the web view's element the focus, unshown")
	}
	tt.rt.webViewKey(page, 0, KeyPageDown, KeyUnknown, false)
	tt.Frame()
	if scroll.Y <= 0 {
		t.Errorf("Page Down the page left scrolled the container to %v", scroll.Y)
	}
	tt.rt.webViewKey(page, 0, KeyJ, KeyUnknown, false)
	tt.Frame()
	if nexts != 1 {
		t.Errorf("J the page left reached the container's shortcut %d times, want 1", nexts)
	}
	// A key of a web view no element shows goes nowhere.
	tt.rt.webViewKey(&testWebView{"other"}, 0, KeyJ, KeyUnknown, false)
	tt.Frame()
	if nexts != 1 {
		t.Error("a key of another web view reached the shortcut")
	}
}
