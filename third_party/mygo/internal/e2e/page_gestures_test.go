//go:build !js

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const gesturesPage = `<!doctype html><html><body style="margin:0;height:100vh">
<p>Pinch me</p><script>window.ready = true</script></body></html>`

// TestPageGestures checks that each platform's webview swipes between
// pages and zooms with a pinch only when PageOptions ask for it: in the
// pages of windows and of web views in native UI, as the webview's own
// settings read back.
func TestPageGestures(t *testing.T) {
	both := mygo.PageOptions{SwipeNavigation: true, PinchZoom: true}
	plain := newWindow(t, mygo.WindowOptions{Title: "No gestures", Width: 300, Height: 200})
	on := newWindow(t, mygo.WindowOptions{Title: "Gestures", Width: 300, Height: 200, Page: both})
	swipeOnly := newWindow(t, mygo.WindowOptions{Title: "Swipes", Width: 300, Height: 200, Page: mygo.PageOptions{SwipeNavigation: true}})

	var views atomic.Pointer[[2]*mygo.WebView]
	var frames atomic.Int32
	host := newWindow(t, mygo.WindowOptions{Title: "Web views' gestures", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			if v := views.Load(); v != nil {
				ui.WebView(c, v[0]).Grow(1)
				ui.WebView(c, v[1]).Grow(1)
			}
		})
	})})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	plainView, err := host.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	onView, err := host.NewWebView(mygo.WebViewOptions{Page: both})
	if err != nil {
		t.Fatal(err)
	}
	host.Update(func() { views.Store(&[2]*mygo.WebView{plainView, onView}) })

	// The pages load, so that the web views are made: WebView2 makes its
	// own after the window.
	for _, p := range []*mygo.Page{plain.Page(), on.Page(), swipeOnly.Page(), plainView.Page(), onView.Page()} {
		p.LoadHTML(gesturesPage, "")
		waitForPage(t, p, "window.ready")
	}

	for _, tc := range []struct {
		name         string
		w            *mygo.Window
		webView      int
		swipe, pinch bool
	}{
		{"a window's page by default", plain, -1, false, false},
		{"a window's page with both", on, -1, true, true},
		{"a window's page with SwipeNavigation", swipeOnly, -1, true, false},
		{"a web view by default", host, 0, false, false},
		{"a web view with both", host, 1, true, true},
	} {
		swipe, pinch, ok := pageGestures(tc.w, tc.webView)
		if !ok {
			t.Skip("the webview's gesture settings can't be read here")
		}
		if swipe != tc.swipe || pinch != tc.pinch {
			t.Errorf("%s swipes %v and pinches %v, want %v and %v", tc.name, swipe, pinch, tc.swipe, tc.pinch)
		}
	}
}

// TestPinchZoomPage pinches the touchpad over a page that asked for
// PinchZoom and over one that did not, each in turn, until the first
// zooms: the same pinches leave the second as it is, as WebKitGTK zooms
// on a pinch unless MyGo stops it.
func TestPinchZoomPage(t *testing.T) {
	zoomed := newWindow(t, mygo.WindowOptions{Title: "Pinch zoom", Width: 300, Height: 200, Page: mygo.PageOptions{PinchZoom: true}})
	plain := newWindow(t, mygo.WindowOptions{Title: "No pinch zoom", Width: 300, Height: 200})
	for _, w := range []*mygo.Window{zoomed, plain} {
		w.Page().LoadHTML(gesturesPage, "")
		waitFor(t, w, "window.ready")
	}
	// The page answers its scripts after the events before them.
	scale := func(w *mygo.Window) float64 {
		s, _ := mygo.EvalAs[float64](w.Page(), "window.visualViewport ? visualViewport.scale : 1")
		return s
	}
	pinches := 0
	// WebKit asks the page where it is before it zooms: a pinch over a
	// page that has not answered yet does nothing, and a later one zooms.
	eventually(t, "a pinch zooming the page", func() bool {
		if !pinchPage(zoomed, -1, 150, 100, 1.5, 2, 2) || !pinchPage(plain, -1, 150, 100, 1.5, 2, 2) {
			t.Skip("pinch automation unavailable")
		}
		pinches++
		if s := scale(plain); s != 1 {
			t.Fatalf("%d pinches zoomed a page without PinchZoom to %v", pinches, s)
		}
		return scale(zoomed) > 1
	})
	if s := scale(plain); s != 1 {
		t.Errorf("%d pinches zoomed a page without PinchZoom to %v", pinches, s)
	}
}
