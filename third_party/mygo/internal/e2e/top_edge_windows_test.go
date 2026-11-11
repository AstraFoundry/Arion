//go:build windows

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// Hit-test codes.
const (
	htClient   = 1
	htTop      = 12
	htTopLeft  = 13
	htTopRight = 14
)

// A resizable window without a caption resizes from its top edge, within
// the thickness of the system's resize border, above its page or native
// UI, as from its other edges; below, presses reach the content, whose
// drag regions move the window. One that cannot resize, or is maximized,
// leaves the whole top to the content.
func TestCaptionlessTopEdge(t *testing.T) {
	page := `<body style="margin:0"><div style="--app-region:drag;height:48px">drag</div></body>`
	for name, c := range map[string]struct {
		opts      mygo.WindowOptions
		rightEdge bool // the edge reaches the top-right corner
	}{
		"frameless":                {mygo.WindowOptions{Frameless: true}, true},
		"frameless with acrylic":   {mygo.WindowOptions{Frameless: true, Transparent: true, Vibrancy: mygo.VibrancyAcrylic}, true},
		"frameless native UI":      {mygo.WindowOptions{Frameless: true, Content: ui.View(func(c *ui.Context) { ui.Text(c, "native") })}, true},
		"hidden title bar":         {mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden}, false},
		"hidden title, no buttons": {mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden, HideWindowButtons: true}, true},
	} {
		t.Run(name, func(t *testing.T) {
			opts := c.opts
			opts.X, opts.Y, opts.Width, opts.Height = 120, 120, 420, 300
			w := newWindow(t, opts)
			if opts.Content == nil {
				w.Page().LoadHTML(page, "")
				waitFor(t, w, "document.readyState === 'complete'")
			}
			hit := func(x, y int32) (h uintptr) {
				mygo.RunOnMain(func() { h = win.TestHitTest(w.NativeHandle(), x, y) })
				return h
			}
			eventually(t, "the top edge to resize the window", func() bool { return hit(210, 1) == htTop })
			if h := hit(1, 1); h != htTopLeft {
				t.Errorf("top-left corner: hit test = %d, want HTTOPLEFT", h)
			}
			if h := hit(-2, 1); c.rightEdge && h != htTopRight {
				t.Errorf("top-right corner: hit test = %d, want HTTOPRIGHT", h)
			}
			// Below the resize border, 8 DIPs: the content's, at 300% too.
			if h := hit(210, 40); h != htClient {
				t.Errorf("below the top edge: hit test = %d, want HTCLIENT (the content's)", h)
			}
			w.SetResizable(false)
			if h := hit(210, 1); h != htClient {
				t.Errorf("not resizable: hit test at the top = %d, want HTCLIENT", h)
			}
			w.SetResizable(true)
			w.Maximize()
			eventually(t, "the window maximized", w.IsMaximized)
			if h := hit(210, 1); h != htClient {
				t.Errorf("maximized: hit test at the top = %d, want HTCLIENT", h)
			}
		})
	}
}
