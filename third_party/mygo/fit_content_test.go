//go:build !js

package mygo

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

func TestFitContentDIPs(t *testing.T) {
	f := &fitContent{min: platform.Size{Width: 100, Height: 40}, max: platform.Size{Width: 300, Height: 200}}
	for _, c := range []struct {
		w, h, zoom float64
		want       platform.Size
	}{
		{120, 80, 1, platform.Size{Width: 120, Height: 80}},
		{120.5, 80.25, 1, platform.Size{Width: 121, Height: 81}}, // fractions take a pixel
		{120.004, 80, 1, platform.Size{Width: 120, Height: 80}},  // but not rounding errors
		{60.25, 45, 2, platform.Size{Width: 121, Height: 90}},    // CSS pixels of a zoomed page
		{150, 100, 0, platform.Size{Width: 150, Height: 100}},    // no zoom is 1
		{10, 10, 1, platform.Size{Width: 100, Height: 40}},       // the minimum
		{500, 400, 1, platform.Size{Width: 300, Height: 200}},    // the maximum
		{0, 0, 1, platform.Size{Width: 100, Height: 40}},
	} {
		if got, ok := f.fit(contentSize{Width: c.w, Height: c.h}, c.zoom, 0); !ok || got != c.want {
			t.Errorf("fit(%v, %v, zoom %v) = %v, %v; want %v", c.w, c.h, c.zoom, got, ok, c.want)
		}
	}
	unbounded := &fitContent{}
	if got, _ := unbounded.fit(contentSize{Height: 1e300}, 1, 0); got != (platform.Size{Width: 1, Height: maxFitDIPs}) {
		t.Errorf("unbounded fit of an empty, huge page = %v", got)
	}
	for _, bad := range [][2]float64{{-1, 10}, {10, -0.5}} {
		if got, ok := f.fit(contentSize{Width: bad[0], Height: bad[1]}, 1, 0); ok {
			t.Errorf("fit(%v) = %v, want no size", bad, got)
		}
	}
}

// The room of the scrollbars counts where the content needs them: a
// vertical scrollbar where the content is taller than the window may be,
// not while the window is only shorter than the content for a moment.
func TestFitContentScrollbars(t *testing.T) {
	f := &fitContent{max: platform.Size{Width: 300, Height: 200}}
	for _, c := range []struct {
		what      string
		c         contentSize
		maxHeight int
		want      platform.Size
	}{
		{"a vertical scrollbar for a moment", contentSize{Width: 120, Height: 150, Scrollbar: 17}, 0, platform.Size{Width: 120, Height: 150}},
		{"taller than the maximum", contentSize{Width: 120, Height: 500, Scrollbar: 17}, 0, platform.Size{Width: 137, Height: 200}},
		{"taller than the work area", contentSize{Width: 120, Height: 150, Scrollbar: 17}, 100, platform.Size{Width: 137, Height: 100}},
		{"overlay scrollbars", contentSize{Width: 120, Height: 500}, 0, platform.Size{Width: 120, Height: 200}},
		{"wider than the maximum", contentSize{Width: 300, Height: 60, HorizontalScrollbar: 17}, 0, platform.Size{Width: 300, Height: 77}},
		{"zoomed", contentSize{Width: 150, Height: 60, HorizontalScrollbar: 8.5}, 0, platform.Size{Width: 300, Height: 137}},
		{"no room", contentSize{Width: 120, Height: 500, Scrollbar: -1, HorizontalScrollbar: math.NaN()}, 0, platform.Size{Width: 120, Height: 200}},
	} {
		zoom := 1.0
		if c.what == "zoomed" {
			zoom = 2
		}
		if got, ok := f.fit(c.c, zoom, c.maxHeight); !ok || got != c.want {
			t.Errorf("%s: fit(%+v) = %v, %v; want %v", c.what, c.c, got, ok, c.want)
		}
	}
}

// fitLimitsEvent matches the page event of FitContent's limits.
func fitLimitsEvent(minW, minH, maxW, maxH float64) func(m map[string]any) bool {
	return func(m map[string]any) bool {
		p, _ := m["p"].(map[string]any)
		return m["n"] == "mygo:fit-content" && p["minWidth"] == minW && p["minHeight"] == minH && p["maxWidth"] == maxW && p["maxHeight"] == maxH
	}
}

// A flyout with FitContent takes the size its page tells, within its
// limits, and its page hears the limits in its CSS pixels.
func TestFlyoutFitContent(t *testing.T) {
	parent, _ := testWindow(t, WindowOptions{X: 100, Y: 100, Width: 400, Height: 300})
	fl, fw := testFlyout(t, FlyoutOptions{
		Parent:     parent,
		Anchor:     Rectangle{Y: 200},
		Width:      150,
		Height:     120,
		FitContent: true,
		MinWidth:   100,
		MinHeight:  40,
		MaxWidth:   300,
		MaxHeight:  200,
	})
	if s := fw.Opts.UserScripts[0].Source; !strings.Contains(s, `"fit":{"minWidth":100,"minHeight":40,"maxWidth":300,"maxHeight":200}`) {
		t.Errorf("the bridge's configuration has no limits: %s", s[strings.LastIndex(s, "})("):])
	}
	if o := fw.Opts; o.MinSize != (platform.Size{}) || o.MaxSize != (platform.Size{}) || o.Width != 150 || o.Height != 120 {
		t.Errorf("flyout window options: size %dx%d, min %v, max %v", o.Width, o.Height, o.MinSize, o.MaxSize)
	}
	size := func(what string, w, h int) {
		t.Helper()
		if b := fl.Bounds(); b.Width != w || b.Height != h {
			t.Errorf("%s: flyout %dx%d, want %dx%d", what, b.Width, b.Height, w, h)
		}
	}
	size("before the page told its size", 150, 120)
	page(fw, `{"t":"content-size","w":120.5,"h":80}`)
	size("the content's size", 121, 80)
	page(fw, `{"t":"content-size","w":500,"h":400}`)
	size("too large", 300, 200)
	page(fw, `{"t":"content-size","w":20,"h":10}`)
	size("too small", 100, 40)
	page(fw, `{"t":"content-size","w":-1,"h":10}`)
	size("no size", 100, 40)

	// SetSize holds until the content's size changes.
	fl.SetSize(160, 90)
	page(fw, `{"t":"content-size","w":20,"h":10}`)
	size("the same content after SetSize", 160, 90)
	page(fw, `{"t":"content-size","w":200,"h":150}`)
	size("other content after SetSize", 200, 150)

	// Zoom changes the CSS pixels of the limits and of the content.
	page(fw, `{"t":"dom-ready"}`)
	fl.Page().SetZoomFactor(2)
	received(t, fw, fitLimitsEvent(50, 20, 150, 100))
	if s := fw.Opts.UserScripts[0].Source; !strings.Contains(s, `"fit":{"minWidth":50,"minHeight":20,"maxWidth":150,"maxHeight":100}`) {
		t.Errorf("the next pages' limits after a zoom: %s", s[strings.LastIndex(s, "})("):])
	}
	size("the content zoomed", 300, 200) // 200x150 CSS pixels, bounded
	page(fw, `{"t":"content-size","w":60,"h":30}`)
	size("zoomed content", 120, 60)

	// A flyout without FitContent does not fit; native UI fits without a
	// bridge (TestFlyoutFitNativeContent).
	native, nfw := testFlyout(t, FlyoutOptions{Parent: parent, FitContent: true, Content: ui.View(func(c *ui.Context) {})})
	plain, pfw := testFlyout(t, FlyoutOptions{Parent: parent})
	onMain(func() {
		if native.fit == nil || plain.fit != nil {
			t.Errorf("fit of native UI %v, of a flyout without FitContent %v", native.fit, plain.fit)
		}
	})
	if strings.Contains(pfw.Opts.UserScripts[0].Source, `"fit"`) || len(nfw.Opts.UserScripts) > 0 && strings.Contains(nfw.Opts.UserScripts[0].Source, `"fit"`) {
		t.Error("limits in the bridge of a flyout that does not fit, or of native UI")
	}
	page(pfw, `{"t":"content-size","w":50,"h":50}`)
	if b := plain.Bounds(); b.Width != 200 || b.Height != 200 {
		t.Errorf("a flyout without FitContent took its content's size: %+v", b)
	}
}

// A flyout of native UI with FitContent takes the size its content
// measures, within its limits and the work area, and follows it as it
// changes, placed against its anchor at each size: two frames, the first
// laid out at the size the flyout has, the next at the size it took.
func TestFlyoutFitNativeContent(t *testing.T) {
	// The fake display's work area is 1440x875 at 0,25: the anchor is 100
	// DIPs above its bottom.
	parent, _ := testWindow(t, WindowOptions{X: 100, Y: 600, Width: 400, Height: 300})
	rows, width := 2, float32(100)
	fl, fw := testFlyout(t, FlyoutOptions{
		Parent:     parent,
		Anchor:     Rectangle{Y: 200},
		Width:      150,
		Height:     120,
		FitContent: true,
		MinWidth:   100,
		MinHeight:  40,
		MaxWidth:   300,
		MaxHeight:  250,
		Content: ui.View(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(10).Gap(5).Children(func() {
				for i := 0; i < rows; i++ {
					ui.Box(c).Size(width, 20)
				}
			})
		}),
	})
	s := fw.FakeSurface()
	resizes := 0
	// settle draws the frames the flyout asks for, as the display would,
	// and runs the resize each posts; a surface tells its content it
	// resized, which asks for the next frame.
	settle := func() {
		t.Helper()
		for i := 0; ; i++ {
			if i == 10 {
				t.Fatal("the flyout keeps resizing")
			}
			before := fl.Bounds()
			drew := false
			onMain(func() { drew = s.Frame() })
			onMain(func() {})
			if b := fl.Bounds(); b.Width == before.Width && b.Height == before.Height {
				if !drew {
					return
				}
				continue
			}
			resizes++
			onMain(func() { s.Send(platform.SurfaceEvent{Kind: platform.SurfaceResize}) })
		}
	}
	bounds := func(what string, x, y, w, h int) {
		t.Helper()
		if b := fl.Bounds(); b != (Rectangle{X: x, Y: y, Width: w, Height: h}) {
			t.Errorf("%s: flyout %+v, want %dx%d at %d,%d", what, b, w, h, x, y)
		}
	}
	update := func(fn func()) {
		t.Helper()
		fl.Update(fn)
		onMain(func() {})
		settle()
	}
	bounds("before its first frame, above the anchor", 100, 800-120, 150, 120)
	settle()
	bounds("its content's size, below the anchor", 100, 800, 120, 65)
	if resizes != 1 {
		t.Errorf("%d resizes for the first frame", resizes)
	}

	// It follows the content, flipping above the anchor where the work
	// area has no room below.
	update(func() { rows = 12 })
	bounds("taller, above the anchor", 100, 800-250, 120, 250) // 12 rows: 315 DIPs, the most 250
	update(func() { rows, width = 1, 20 })
	bounds("smaller, below again", 100, 800, 100, 40)
	update(func() { width = 500 })
	bounds("wider than the most", 100, 800, 300, 40)

	// The same content resizes nothing, and a size SetSize gives holds
	// until the content's changes.
	resizes = 0
	fl.Invalidate()
	onMain(func() {})
	settle()
	if resizes != 0 {
		t.Errorf("%d resizes for the same content", resizes)
	}
	fl.SetSize(220, 90)
	fl.Invalidate()
	onMain(func() {})
	settle()
	bounds("the same content after SetSize", 100, 800, 220, 90)
	update(func() { width = 180 })
	bounds("other content after SetSize", 100, 800, 200, 40)

	// Without limits, the work area bounds it.
	free, ffw := testFlyout(t, FlyoutOptions{
		Parent:     parent,
		FitContent: true,
		Content:    ui.View(func(c *ui.Context) { ui.Box(c).Size(2000, 2000) }),
	})
	s = ffw.FakeSurface()
	fl = free
	settle()
	if b := free.Bounds(); b.Width != 1440 || b.Height != 875 {
		t.Errorf("huge content: flyout %+v, want the work area's 1440x875", b)
	}
}

// A hidden flyout of native UI with FitContent draws no frames, as a
// hidden window does: content that changed meanwhile lays out as the
// flyout shows, which shows at the size it takes, not at the size it hid
// at with a resize after its first frame.
func TestFlyoutFitNativeContentHidden(t *testing.T) {
	parent, _ := testWindow(t, WindowOptions{X: 100, Y: 100, Width: 400, Height: 300})
	rows, width, builds := 2, float32(100), 0
	fl, fw := testFlyout(t, FlyoutOptions{
		Parent:     parent,
		Anchor:     Rectangle{Y: 200},
		Width:      150,
		Height:     120,
		FitContent: true,
		MinWidth:   100,
		MinHeight:  40,
		Hidden:     true,
		Content: ui.View(func(c *ui.Context) {
			builds++
			ui.Column(c).Fill().Padding(10).Gap(5).Children(func() {
				for i := 0; i < rows; i++ {
					ui.Box(c).Size(width, 20)
				}
			})
		}),
	})
	s := fw.FakeSurface()
	resizes := 0
	// settle draws the frames the flyout asks for, as the display would
	// while it shows, and runs the resizes they post.
	settle := func() {
		t.Helper()
		for i := 0; ; i++ {
			if i == 10 {
				t.Fatal("the flyout keeps resizing")
			}
			before := fl.Bounds()
			drew := false
			onMain(func() { drew = s.Frame() })
			onMain(func() {})
			if b := fl.Bounds(); b.Width == before.Width && b.Height == before.Height {
				if !drew {
					return
				}
				continue
			}
			resizes++
			onMain(func() { s.Send(platform.SurfaceEvent{Kind: platform.SurfaceResize}) })
		}
	}
	// The size the platform has the flyout at as it shows it.
	var shown platform.Size
	fl.OnShow(func() {
		b := fw.Bounds()
		shown = platform.Size{Width: b.Width, Height: b.Height}
	})
	show := func(what string, showFn func(), w, h int) {
		t.Helper()
		resizes = 0
		showFn()
		if want := (platform.Size{Width: w, Height: h}); shown != want {
			t.Errorf("%s: shown at %v, want %v", what, shown, want)
		}
		settle()
		if b := fl.Bounds(); b.Width != w || b.Height != h || resizes != 0 {
			t.Errorf("%s: %+v after %d resizes once shown, want %dx%d after none", what, b, resizes, w, h)
		}
	}

	// Its first frame draws while hidden, which makes it ready to show.
	settle()
	if b := fl.Bounds(); b.Width != 120 || b.Height != 65 {
		t.Fatalf("hidden, after its first frame: %+v, want 120x65", b)
	}
	// Changed while hidden, before its first Show: 4 rows.
	fl.Update(func() { rows = 4 })
	onMain(func() {})
	show("changed before its first Show", fl.Show, 120, 115)

	// Shown, hidden, changed and shown again.
	fl.Hide()
	fl.Update(func() { rows, width = 1, 20 })
	onMain(func() {})
	show("changed while hidden", fl.ShowInactive, 100, 40)

	// Changed on the main thread, with Invalidate, and shown at once.
	fl.Hide()
	show("invalidated and shown at once", func() {
		onMain(func() {
			rows = 3
			fl.Invalidate()
			fl.Show()
		})
	}, 100, 90)

	// Unchanged while hidden: nothing builds before it shows.
	fl.Hide()
	show("unchanged", func() {
		n := builds
		fl.Show()
		if builds != n {
			t.Errorf("%d builds as a flyout shows again unchanged", builds-n)
		}
	}, 100, 90)
}

// A window with FitContent sizes its page area to its content, in the
// dimension it changes in, within its limits and the work area.
func TestWindowFitContent(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{X: 100, Y: 100, Width: 400, Height: 300, FitContent: true, MinHeight: 100, MaxWidth: 600})
	if s := fw.Opts.UserScripts[0].Source; !strings.Contains(s, `"fit":{"minWidth":0,"minHeight":100,"maxWidth":600,"maxHeight":0}`) {
		t.Errorf("the bridge's configuration has no limits: %s", s[strings.LastIndex(s, "})("):])
	}
	size := func(what string, width, height int) {
		t.Helper()
		if gw, gh := w.ContentSize(); gw != width || gh != height {
			t.Errorf("%s: page area %dx%d, want %dx%d", what, gw, gh, width, height)
		}
	}
	page(fw, `{"t":"content-size","w":250,"h":150}`)
	size("the content's size", 250, 150)
	if x, y := w.Position(); x != 100 || y != 100 {
		t.Errorf("the window moved to %d,%d", x, y)
	}

	// The user resizes it: that holds until the content changes, in the
	// dimension it changes in.
	w.SetContentSize(320, 200)
	page(fw, `{"t":"content-size","w":250,"h":150}`)
	size("the same content after a resize", 320, 200)
	page(fw, `{"t":"content-size","w":250,"h":180}`)
	size("taller content", 320, 180)
	page(fw, `{"t":"content-size","w":260,"h":180}`)
	size("wider content", 260, 180)

	// The user resizes it, and the content grows before the web view takes
	// the size: the page shows a vertical scrollbar for a moment, which
	// changes none of the content's width.
	w.SetContentSize(400, 200)
	page(fw, `{"t":"content-size","w":260,"h":220,"sw":17}`)
	size("taller content right after a resize", 400, 220)
	page(fw, `{"t":"content-size","w":260,"h":220}`)
	size("the scrollbar gone", 400, 220)

	// Its limits, and the fake display's work area (1440x875).
	page(fw, `{"t":"content-size","w":900,"h":5000}`)
	size("huge content", 600, 875)
	page(fw, `{"t":"content-size","w":200,"h":10}`)
	size("short content", 200, 100)

	// Not while maximized.
	w.Maximize()
	page(fw, `{"t":"content-size","w":300,"h":300}`)
	size("content of a maximized window", 200, 100)

	// Without FitContent, the content's size changes nothing.
	plain, pfw := testWindow(t, WindowOptions{Width: 400, Height: 300})
	if strings.Contains(pfw.Opts.UserScripts[0].Source, `"fit"`) {
		t.Error("limits in the bridge of a window that does not fit")
	}
	page(pfw, `{"t":"content-size","w":250,"h":150}`)
	if gw, gh := plain.ContentSize(); gw != 400 || gh != 300 {
		t.Errorf("a window without FitContent took its content's size: %dx%d", gw, gh)
	}
}
