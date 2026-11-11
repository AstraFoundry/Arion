//go:build windows

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
)

// WebView2's own keys, as F5 and Ctrl+R, which reload the page without
// asking OnWillNavigate, come with the inspector: a page without one, as
// in production builds, has them off, as WKWebView and WebKitGTK do.
func TestBrowserAcceleratorKeys(t *testing.T) {
	for _, tc := range []struct {
		devTools mygo.DevTools
		want     bool
	}{{mygo.DevToolsDisabled, false}, {mygo.DevToolsEnabled, true}} {
		w := newWindow(t, mygo.WindowOptions{Title: "Keys", Width: 300, Height: 200, Page: mygo.PageOptions{DevTools: tc.devTools}})
		w.Page().LoadHTML("<p>keys</p>", "")
		waitFor(t, w, "document.readyState == 'complete'")
		var on, ok bool
		mygo.RunOnMain(func() { on, ok = win.TestBrowserAcceleratorKeys(w.NativeHandle()) })
		if !ok {
			t.Skip("the WebView2 Runtime has no ICoreWebView2Settings3")
		}
		if on != tc.want {
			t.Errorf("DevTools %v: the browser's keys are on: %v, want %v", tc.devTools, on, tc.want)
		}
	}
}
