//go:build !js

package mygo

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestWebViewAvailable: the call reports what the backend says, so an app
// can ask before it creates its first window showing a page, as NewWindow
// panics where the platform has no webview.
func TestWebViewAvailable(t *testing.T) {
	if !WebViewAvailable() {
		t.Error("WebViewAvailable = false with a webview the fake has")
	}
	onMain(func() { fb.NoWebView = true })
	t.Cleanup(func() { onMain(func() { fb.NoWebView = false }) })
	if WebViewAvailable() {
		t.Error("WebViewAvailable = true with NoWebView set")
	}
}

// TestWebViewAvailableChoosesWindowKind: what the call is for, the guard an
// app writes, opens the window the machine can show, and both kinds work
// here: with no webview it opens one showing native UI, which needs none.
func TestWebViewAvailableChoosesWindowKind(t *testing.T) {
	for _, noWebView := range []bool{false, true} {
		name := "webview"
		if noWebView {
			name = "no webview"
		}
		t.Run(name, func(t *testing.T) {
			onMain(func() { fb.NoWebView = noWebView })
			t.Cleanup(func() { onMain(func() { fb.NoWebView = false }) })

			var w *Window
			if WebViewAvailable() {
				w = NewWindow(WindowOptions{URL: "https://example.com"})
			} else {
				w = NewWindow(WindowOptions{Content: ui.View(func(c *ui.Context) { ui.Box(c) })})
			}
			t.Cleanup(w.Destroy)

			// The window is of the kind the guard chose: a page, or native
			// UI, which has none and needs no webview to be built.
			if noWebView {
				if w.Page() != nil {
					t.Error("a window of native UI has a page")
				}
				if _, err := w.NewWebView(WebViewOptions{}); err != nil {
					t.Errorf("NewWebView in a window of native UI: %v", err)
				}
				return
			}
			if w.Page() == nil {
				t.Error("a window showing a page has none")
			}
		})
	}
}
