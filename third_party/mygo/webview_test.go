//go:build !js

package mygo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

type pageService struct{}

func (pageService) Where(ctx context.Context) string {
	p := CallerPage(ctx)
	id := 0
	if v := p.WebView(); v != nil {
		id = v.w.id
	}
	return fmt.Sprintf("%d %d", CallerWindow(ctx).ID(), id)
}

// webViewWindow creates a window of native UI and a web view in it, which
// the view shows while show is true, and returns them with the fake web
// view, its page loaded.
func webViewWindow(t *testing.T, show *bool) (*Window, *WebView, *fake.Surface, *fake.WebView) {
	t.Helper()
	return webViewWindowOpts(t, show, WebViewOptions{})
}

// webViewWindowOpts is webViewWindow for a web view made with opts.
func webViewWindowOpts(t *testing.T, show *bool, opts WebViewOptions) (*Window, *WebView, *fake.Surface, *fake.WebView) {
	t.Helper()
	var v *WebView
	w, _, s := contentWindow(t, func(c *ui.Context) {
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Text(c, "Toolbar").Height(40)
			if v != nil && *show {
				ui.WebView(c, v).Grow(1)
			}
		})
	})
	v, err := w.NewWebView(opts)
	if err != nil {
		t.Fatal(err)
	}
	views := s.WebViews()
	if len(views) != 1 {
		t.Fatalf("%d web views", len(views))
	}
	fv := views[0]
	onMain(func() { fv.H.NavigationCommitted("about:blank") })
	page(fv.Window, `{"t":"dom-ready"}`)
	w.Invalidate()
	onMain(func() {})
	onMain(func() { s.Frame() })
	return w, v, s, fv
}

func TestWebViewThemePreferences(t *testing.T) {
	onMain(func() { fb.SetPreferences(platform.Preferences{TextScale: 1.5}) })
	defer onMain(func() { fb.SetPreferences(platform.Preferences{}) })
	show := true
	_, _, _, fv := webViewWindow(t, &show)
	if len(fv.Opts.UserScripts) != 2 || !strings.Contains(fv.Opts.UserScripts[1].Source, `"textScale":1.5`) {
		t.Fatal("web view does not install initial preferences")
	}
	onMain(func() { fb.SetPreferences(platform.Preferences{TextScale: 2}) })
	received(t, fv.Window, func(m map[string]any) bool {
		p, _ := m["p"].(map[string]any)
		return m["n"] == "mygo:preferences" && p["textScale"] == float64(2)
	})
	// The web view's next documents start with the new ones.
	src := onMainValue(func() string { return fv.Opts.UserScripts[1].Source })
	if !strings.Contains(src, `"textScale":2`) {
		t.Errorf("web view's preferences script after a change = %s", src)
	}
}

// TestWebViewTabFocus passes the page's Tab behavior to a web view: every
// control by default, Safari's text fields only when asked for.
func TestWebViewTabFocus(t *testing.T) {
	w, _, s := contentWindow(t, func(c *ui.Context) {})
	if _, err := w.NewWebView(WebViewOptions{Page: PageOptions{SafariTabFocus: true}}); err != nil {
		t.Fatal(err)
	}
	if views := s.WebViews(); len(views) != 1 || views[0].Opts.TabFocusesLinks {
		t.Errorf("web views %d: SafariTabFocus not applied", len(views))
	}
	if _, err := w.NewWebView(WebViewOptions{}); err != nil {
		t.Fatal(err)
	}
	if views := s.WebViews(); len(views) != 2 || !views[1].Opts.TabFocusesLinks {
		t.Error("Tab reaches every control by default")
	}
}

// TestWebViewScrollToContainer passes ScrollToContainer to the backend,
// off unless asked for.
func TestWebViewScrollToContainer(t *testing.T) {
	w, _, s := contentWindow(t, func(c *ui.Context) {})
	for _, on := range []bool{true, false} {
		if _, err := w.NewWebView(WebViewOptions{ScrollToContainer: on}); err != nil {
			t.Fatal(err)
		}
	}
	if views := s.WebViews(); len(views) != 2 || !views[0].Opts.ScrollToContainer || views[1].Opts.ScrollToContainer {
		t.Error("ScrollToContainer did not reach the backend as asked")
	}
}

// TestWebViewPictureInPicture passes PictureInPicture to a web view, whose
// page moves its video into picture in picture.
func TestWebViewPictureInPicture(t *testing.T) {
	w, _, s := contentWindow(t, func(c *ui.Context) {})
	v, err := w.NewWebView(WebViewOptions{Page: PageOptions{PictureInPicture: true}})
	if err != nil {
		t.Fatal(err)
	}
	if views := s.WebViews(); len(views) != 1 || !views[0].Opts.PictureInPicture {
		t.Fatalf("web views %d: PictureInPicture not applied", len(views))
	}
	v.Page().TogglePictureInPicture()
	if !v.Page().IsPictureInPictureActive() {
		t.Error("TogglePictureInPicture did not reach the web view")
	}
}

// TestPageGestures passes SwipeNavigation and PinchZoom to the backend for
// a window's page and a web view's, both off unless asked for.
func TestPageGestures(t *testing.T) {
	gestures := PageOptions{SwipeNavigation: true, PinchZoom: true}
	_, fw := testWindow(t, WindowOptions{})
	if o := fw.Opts; o.SwipeNavigation || o.PinchZoom {
		t.Errorf("a window's page swipes %v and pinches %v by default", o.SwipeNavigation, o.PinchZoom)
	}
	_, fw = testWindow(t, WindowOptions{Page: gestures})
	if o := fw.Opts; !o.SwipeNavigation || !o.PinchZoom {
		t.Errorf("a window's page asked to swipes %v and pinches %v", o.SwipeNavigation, o.PinchZoom)
	}
	_, fw = testWindow(t, WindowOptions{Page: PageOptions{PinchZoom: true}})
	if o := fw.Opts; o.SwipeNavigation || !o.PinchZoom {
		t.Errorf("a window's page asked to pinch swipes %v and pinches %v", o.SwipeNavigation, o.PinchZoom)
	}

	w, _, s := contentWindow(t, func(c *ui.Context) {})
	if _, err := w.NewWebView(WebViewOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.NewWebView(WebViewOptions{Page: gestures}); err != nil {
		t.Fatal(err)
	}
	views := s.WebViews()
	if len(views) != 2 {
		t.Fatalf("%d web views", len(views))
	}
	if o := views[0].Opts; o.SwipeNavigation || o.PinchZoom {
		t.Errorf("a web view swipes %v and pinches %v by default", o.SwipeNavigation, o.PinchZoom)
	}
	if o := views[1].Opts; !o.SwipeNavigation || !o.PinchZoom {
		t.Errorf("a web view asked to swipes %v and pinches %v", o.SwipeNavigation, o.PinchZoom)
	}
}

func TestWebViewNeedsContent(t *testing.T) {
	w, _ := testWindow(t, WindowOptions{})
	if _, err := w.NewWebView(WebViewOptions{}); !errors.Is(err, errNoContent) {
		t.Errorf("a web view in a window showing a page: %v", err)
	}
	c, _, _ := contentWindow(t, func(c *ui.Context) {})
	if _, err := c.NewWebView(WebViewOptions{BackgroundColor: "nope"}); err == nil {
		t.Error("a bad background color made a web view")
	}
}

// The view places the web view, which hides when not built and keeps its
// page; Destroy and closing the window close it.
func TestWebViewPlaced(t *testing.T) {
	show := true
	w, v, s, fv := webViewWindow(t, &show)
	if v.Window() != w || v.Page().Window() != w || v.Page().WebView() != v || w.Page() != nil {
		t.Error("the web view's window or page")
	}
	if !fv.Shown || fv.Placement.Frame.Y != 40 || fv.Placement.Frame.H != 160 || fv.Placement.Frame.W != 300 {
		t.Fatalf("placed %+v, shown %v", fv.Placement, fv.Shown)
	}
	if s.WebViewAt(150, 100) != fv || s.WebViewAt(150, 20) != nil {
		t.Error("the pointer goes to the web view where it is not, or not where it is")
	}
	show = false
	w.Invalidate()
	onMain(func() {})
	onMain(func() { s.Frame() })
	if fv.Shown || fv.IsClosed() {
		t.Errorf("a web view not built: shown %v, closed %v", fv.Shown, fv.IsClosed())
	}
	show = true
	w.Invalidate()
	onMain(func() {})
	onMain(func() { s.Frame() })
	if !fv.Shown {
		t.Error("the web view built again does not show")
	}
	v.Destroy()
	if !v.IsDestroyed() || !fv.IsClosed() || len(s.WebViews()) != 0 {
		t.Error("Destroy left the web view")
	}
	w.Invalidate()
	onMain(func() {})
	onMain(func() { s.Frame() })
	if fv.Shown {
		t.Error("a destroyed web view shows")
	}

	// Closing the window closes its web views.
	w2, v2, _, fv2 := webViewWindow(t, &show)
	w2.Destroy()
	if !v2.IsDestroyed() || !fv2.IsClosed() {
		t.Error("closing the window left its web view")
	}
}

// The web view's page calls Go as a window's does, with its window and its
// page as callers, and hears events emitted to its window.
func TestWebViewPage(t *testing.T) {
	bindForTest(t, "Pages", pageService{})
	ev := newEventForTest[string](t, "webview-test")
	show := true
	w, v, _, fv := webViewWindow(t, &show)
	if src := fv.Opts.UserScripts[0].Source; !strings.Contains(src, fmt.Sprintf(`"windowId":%d`, w.ID())) {
		t.Errorf("the bridge does not have the window's id: %s", src)
	}
	m := call(t, fv.Window, 1, "Pages.Where")
	if m["ok"] != true || m["v"] != fmt.Sprintf("%d %d", w.ID(), v.w.id) {
		t.Errorf("Pages.Where: %v", m)
	}
	if err := ev.Emit(w, "to the window"); err != nil {
		t.Fatal(err)
	}
	received(t, fv.Window, func(m map[string]any) bool { return m["t"] == "event" && m["p"] == "to the window" })
	if err := ev.EmitPage(v.Page(), "to the page"); err != nil {
		t.Fatal(err)
	}
	received(t, fv.Window, func(m map[string]any) bool { return m["t"] == "event" && m["p"] == "to the page" })
	if err := ev.Broadcast("to all"); err != nil {
		t.Fatal(err)
	}
	received(t, fv.Window, func(m map[string]any) bool { return m["t"] == "event" && m["p"] == "to all" })
	// window.close() in the page leaves the window open.
	onMain(func() { fv.H.ClosedByPage() })
	if w.IsDestroyed() {
		t.Error("the page of a web view closed its window")
	}
	// The page's methods reach the web view.
	if err := v.Page().LoadURL("https://example.com/"); err != nil {
		t.Fatal(err)
	}
	if got := v.Page().URL(); got != "https://example.com/" {
		t.Errorf("URL %q", got)
	}
	fv.SetProcessID(31)
	if got := v.Page().ProcessID(); got != 31 {
		t.Errorf("ProcessID %d, want 31", got)
	}
	fv.SetGPUProcessID(32)
	if got := v.Page().GPUProcessID(); got != 32 {
		t.Errorf("GPUProcessID %d, want 32", got)
	}
	v.Page().SetAudioMuted(true)
	if !fv.IsAudioMuted() || !v.Page().IsAudioMuted() {
		t.Errorf("SetAudioMuted: the web view's %v, the page's %v", fv.IsAudioMuted(), v.Page().IsAudioMuted())
	}
	var audible []bool
	v.Page().OnAudibleChanged(func(a bool) { audible = append(audible, a) })
	onMain(func() { fv.SetAudible(true) })
	if !v.Page().IsAudible() || !slices.Equal(audible, []bool{true}) {
		t.Errorf("a web view playing: IsAudible %v, changes %v", v.Page().IsAudible(), audible)
	}
	var muted []CaptureState
	v.Page().OnCaptureStateChanged(func(_ CaptureKind, s CaptureState) { muted = append(muted, s) })
	onMain(func() { fv.SetCapture(platform.CaptureMicrophone, platform.CaptureActive) })
	v.Page().SetCaptureState(CaptureMicrophone, CaptureMuted)
	if got := v.Page().CaptureState(CaptureMicrophone); got != CaptureMuted || !slices.Equal(muted, []CaptureState{CaptureActive, CaptureMuted}) {
		t.Errorf("a web view's microphone muted: CaptureState %d, changes %v", got, muted)
	}
}

// SetBackgroundColor fills the web view, not its window, and one in
// light-dark follows the appearance.
func TestWebViewSetBackgroundColor(t *testing.T) {
	show := true
	_, v, _, fv := webViewWindow(t, &show)
	fw := fb.Windows()[len(fb.Windows())-1]
	background := func() (web, win platform.Color) {
		onMain(func() { web, win = fv.Background, fw.Background })
		return
	}
	if err := v.SetBackgroundColor("#102030"); err != nil {
		t.Fatal(err)
	}
	if web, win := background(); web != (platform.Color{R: 0x10, G: 0x20, B: 0x30, A: 0xff}) || win != (platform.Color{}) {
		t.Errorf("background %v, window's %v", web, win)
	}
	if err := v.SetBackgroundColor("light-dark(#fff, #000)"); err != nil {
		t.Fatal(err)
	}
	Theme.SetSource(ThemeDark)
	defer Theme.SetSource(ThemeSystem)
	if web, _ := background(); web != (platform.Color{A: 0xff}) {
		t.Errorf("dark background %v", web)
	}
	if err := v.SetBackgroundColor("nope"); err == nil {
		t.Error("SetBackgroundColor accepted an invalid color")
	}
	v.Destroy()
	if err := v.SetBackgroundColor("#fff"); err != nil {
		t.Errorf("SetBackgroundColor of a destroyed web view: %v", err)
	}
}

// A press on the web view takes the focus from the content, which hears
// of it as a press outside its popovers.
func TestWebViewPress(t *testing.T) {
	var v *WebView
	open, typed := false, ""
	w, _, s := contentWindow(t, func(c *ui.Context) {
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			menu := ui.Button(c, "Menu").Height(30)
			if menu.Clicked() {
				open = true
			}
			ui.Popover(c, menu, &open, func() { ui.Text(c, "Item") })
			ui.TextInput(c, &typed).Label("Name")
			if v != nil {
				ui.WebView(c, v).Grow(1)
			}
		})
	})
	v, err := w.NewWebView(WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fv := s.WebViews()[0]
	frame := func() {
		w.Invalidate()
		onMain(func() {})
		onMain(func() { s.Frame() })
	}
	frame()
	onMain(func() {
		s.Press(30, 15) // the menu button
		s.Frame()
	})
	if !open {
		t.Fatal("the popover did not open")
	}
	var pressed *fake.WebView
	onMain(func() {
		pressed = s.Press(150, 180)
		s.Frame()
	})
	if pressed != fv || !fv.Focused || open {
		t.Errorf("a press on the page: web view %v, focused %v, popover open %v", pressed, fv.Focused, open)
	}
}

// A rounded web view in a panel scaling as one surface, as a peek zooming
// in: the backend gets the page's corners and the size it lays out at,
// which the scale leaves as it is, and the page takes no press where its
// corners cut it off.
func TestWebViewRoundedScaled(t *testing.T) {
	var v *WebView
	scale := float32(1)
	w, _, s := contentWindow(t, func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			ui.Box(c).Absolute().Left(30).Top(20).Size(240, 160).Scale(scale, scale).Children(func() {
				if v != nil {
					ui.WebView(c, v).Grow(1).Radius(20)
				}
			})
		})
	})
	v, err := w.NewWebView(WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fv := s.WebViews()[0]
	for _, k := range []float32{1, 0.9, 0.75, 0.6} {
		scale = k
		w.Invalidate()
		onMain(func() {})
		onMain(func() { s.Frame() })
		p := fv.Placement
		if !fv.Shown || p.Width != 240 || p.Height != 160 || p.Radii != [4]float64{20, 20, 20, 20} {
			t.Fatalf("scaled %v: placed %+v: want the page 240 by 160, its corners 20", k, p)
		}
		if sx, sy := p.Scale(); math.Abs(sx-float64(k)) > 1e-3 || math.Abs(sy-float64(k)) > 1e-3 {
			t.Errorf("scaled %v: the backend scales the page by %v, %v", k, sx, sy)
		}
		f := p.Frame
		if s.WebViewAt(150, 100) != fv || s.WebViewAt(f.X+2, f.Y+2) != nil || s.WebViewAt(f.X+f.W-2, f.Y+f.H-2) != nil {
			t.Errorf("scaled %v: the page takes the pointer outside its corners, or not inside them", k)
		}
		var pressed *fake.WebView
		onMain(func() { pressed = s.Press(f.X+2, f.Y+f.H-2) })
		if pressed != nil {
			t.Errorf("scaled %v: a press in a corner cut off went to the page", k)
		}
	}
}

// TestWebViewUnhandledKeys passes the keys a web view's page leaves
// unhandled to the native UI, as if the web view's element had the focus:
// the shortcuts of the elements around it and of the window see them,
// those of other elements do not, and a native text input that had the
// focus types none of them.
func TestWebViewUnhandledKeys(t *testing.T) {
	var v *WebView
	var got []string
	name := "ab"
	w, _, s := contentWindow(t, func(c *ui.Context) {
		note := func(ok bool, what string) {
			if ok {
				got = append(got, what)
			}
		}
		note(c.Shortcut(ui.Cmd, ui.KeyK), "window Cmd+K")
		note(c.Shortcut(0, ui.KeyEscape), "window Escape")
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			list := ui.Column(c).Width(100).Children(func() {
				ui.TextInput(c, &name).Label("Name")
			})
			note(list.Shortcut(0, ui.KeyJ), "list J")
			reader := ui.Column(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
				if v != nil {
					ui.WebView(c, v).Grow(1)
				}
			})
			note(reader.Shortcut(0, ui.KeyK), "reader K")
		})
	})
	v, err := w.NewWebView(WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fv := s.WebViews()[0]
	frame := func() {
		w.Invalidate()
		onMain(func() {})
		onMain(func() { s.Frame() })
	}
	frame()
	keys := func(press func()) []string {
		t.Helper()
		got = nil
		onMain(press)
		frame()
		return got
	}
	if pressed := onMainValue(func() *fake.WebView { return s.Press(200, 100) }); pressed != fv {
		t.Fatal("the press did not reach the page")
	}
	if g := keys(func() { fv.UnhandledKey(0, platform.KeyK) }); !slices.Equal(g, []string{"reader K"}) {
		t.Errorf("K the page left: %q, want the shortcut of the element around the web view", g)
	}
	if g := keys(func() { fv.UnhandledKey(platform.Modifiers(ui.Cmd), platform.KeyK) }); !slices.Equal(g, []string{"window Cmd+K"}) {
		t.Errorf("Cmd+K the page left: %q, want the window's shortcut", g)
	}
	if g := keys(func() { fv.UnhandledKey(0, platform.KeyJ) }); len(g) != 0 {
		t.Errorf("J the page left: %q, want no shortcut of an element beside the web view", g)
	}
	if g := keys(func() { fv.UnhandledKey(0, platform.KeyTab) }); len(g) != 0 {
		t.Errorf("Tab the page left: %q, want it the page's", g)
	}

	// The page took the keyboard without a press, as WebView.Focus gives
	// it, while a native text input had the focus: the input takes none
	// of the page's keys.
	onMain(func() {
		s.Press(50, 10)
		s.Frame()
	})
	v.Focus()
	if g := keys(func() { fv.UnhandledKey(0, platform.KeyBackspace) }); len(g) != 0 || name != "ab" {
		t.Errorf("Backspace the page left: shortcuts %q, the native input holds %q", g, name)
	}
	if g := keys(func() { fv.UnhandledKey(0, platform.KeyEscape) }); !slices.Equal(g, []string{"window Escape"}) {
		t.Errorf("Escape the page left: %q, want the window's shortcut", g)
	}
	// A web view the view no longer shows gives the content no key.
	v.Destroy()
	frame()
	if g := keys(func() { fv.UnhandledKey(platform.Modifiers(ui.Cmd), platform.KeyK) }); len(g) != 0 {
		t.Errorf("a key of a destroyed web view's page: %q", g)
	}
}
