//go:build linux && (amd64 || arm64)

package e2e

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

func decorations(w *mygo.Window) (d linux.Decorations) {
	mygo.RunOnMain(func() { d = linux.TestDecorations(w.NativeHandle()) })
	return d
}

// GTK decorates frameless windows and windows with a hidden title bar
// itself on Wayland, with an empty title bar: its shadow and border go
// around the page, which keeps the size asked for, and GTK drops them while
// the window is maximized or in full screen (#597). Windows without a
// shadow, transparent ones and panels stay undecorated, as every window
// without a title bar on X11.
func TestClientSideDecorations(t *testing.T) {
	content := func() *ui.Content { return ui.View(func(c *ui.Context) { ui.Text(c, "decorations") }) }
	for _, tc := range []struct {
		name string
		opts mygo.WindowOptions
		csd  bool
	}{
		{"frameless", mygo.WindowOptions{Frameless: true}, true},
		{"hidden title bar", mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden}, true},
		{"hidden window buttons", mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden, HideWindowButtons: true}, true},
		{"fixed frameless", mygo.WindowOptions{Frameless: true, DisableResize: true}, true},
		{"without shadow", mygo.WindowOptions{Frameless: true, DisableShadow: true}, false},
		{"transparent", mygo.WindowOptions{Frameless: true, Transparent: true}, false},
		{"panel", mygo.WindowOptions{Frameless: true, Panel: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			opts.Width, opts.Height, opts.UseContentSize = 400, 300, true
			opts.Content = content()
			w := newWindow(t, opts)
			d := decorations(w)
			want := tc.csd && !d.OnX11
			if d.CSD != want {
				t.Fatalf("GTK decorates the window: %v, want %v (X11: %v)", d.CSD, want, d.OnX11)
			}
			size := func(width, height int) func(linux.Decorations) bool {
				return func(d linux.Decorations) bool { return d.Content == [2]int{width, height} }
			}
			shadowed := func(d linux.Decorations) bool {
				return d.Room[0] > 0 && d.Room[1] > 0 && d.Room[2] > 0 && d.Room[3] > 0
			}
			bare := func(d linux.Decorations) bool { return d.Room == [4]int{} }
			// is waits for the window's decorations and its content to be
			// laid out as asked.
			is := func(what string, conds ...func(linux.Decorations) bool) {
				t.Helper()
				var d linux.Decorations
				if !wait.Poll(func() bool {
					d = decorations(w)
					for _, c := range conds {
						if !c(d) {
							return false
						}
					}
					return true
				}) {
					t.Fatalf("%s never happened: %+v", what, d)
				}
			}
			if !want {
				is("an undecorated window of 400x300", bare, size(400, 300))
				if tc.opts.DisableShadow && w.HasShadow() {
					t.Error("HasShadow = true, want false")
				}
				return
			}
			is("decorations around content of 400x300", shadowed, size(400, 300))
			if width, height := w.ContentSize(); width != 400 || height != 300 {
				t.Errorf("content size = %dx%d, want 400x300", width, height)
			}
			if b := w.Bounds(); b.Width != 400 || b.Height != 300 {
				t.Errorf("bounds = %+v, want 400x300 without the shadow", b)
			}
			if !w.HasShadow() {
				t.Error("HasShadow = false, want true")
			}

			if tc.opts.DisableResize {
				return
			}
			// Maximized and in full screen, GTK drops the decorations.
			w.Maximize()
			eventually(t, "a maximized window", w.IsMaximized)
			is("a maximized window without decorations", bare)
			w.Unmaximize()
			eventually(t, "an unmaximized window", func() bool { return !w.IsMaximized() })
			is("the decorations back around content of 400x300", shadowed, size(400, 300))

			w.SetFullScreen(true)
			eventually(t, "a window in full screen", w.IsFullScreen)
			display := mygo.Screen.DisplayNearestPoint(mygo.Point{X: w.Bounds().X, Y: w.Bounds().Y}).Bounds
			is(fmt.Sprintf("full screen content of %dx%d", display.Width, display.Height), bare, size(display.Width, display.Height))
			w.SetFullScreen(false)
			eventually(t, "a window out of full screen", func() bool { return !w.IsFullScreen() })
			is("the decorations back around content of 400x300", shadowed, size(400, 300))
		})
	}
}
