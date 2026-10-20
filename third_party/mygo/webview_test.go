//go:build !js

package mygo

import (
	"context"
	"errors"
	"fmt"
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
	var v *WebView
	w, _, s := contentWindow(t, func(c *ui.Context) {
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Text(c, "Toolbar").Height(40)
			if v != nil && *show {
				ui.WebView(c, v).Grow(1)
			}
		})
	})
	v, err := w.NewWebView(WebViewOptions{})
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
