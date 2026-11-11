//go:build !js

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// throttlingPage counts the ticks of a 10 ms timer and the animation
// frames, stamped with Date.now(), whose clock the pages and the test
// share.
const throttlingPage = `<!doctype html><html><body style="margin:0;height:100vh;background:#fff">
<script>
window.ticks = []; window.frames = [];
setInterval(() => ticks.push(Date.now()), 10);
(function frame() { frames.push(Date.now()); requestAnimationFrame(frame); })();
window.between = (list, from, to) => list.filter((t) => t >= from && t < to).length;
window.ready = true;
</script></body></html>`

// TestBackgroundThrottling measures the rates over throttleSpan, after
// letting the window's change settle for throttleSettle.
const (
	throttleSettle = 500 * time.Millisecond
	throttleSpan   = 2 * time.Second
)

// throttlingPageIn shows throttlingPage with throttling in a new window,
// or in a web view in a new window of native UI, and returns the window
// and the page.
func throttlingPageIn(t *testing.T, webView bool, throttling mygo.BackgroundThrottling, x int) (*mygo.Window, *mygo.Page) {
	t.Helper()
	opts := mygo.WindowOptions{Title: fmt.Sprintf("Throttling %d", throttling), X: x, Y: 80, Width: 300, Height: 200}
	if !webView {
		opts.Page.BackgroundThrottling = throttling
		w := newWindow(t, opts)
		w.Page().LoadHTML(throttlingPage, "")
		waitFor(t, w, "window.ready && frames.length > 2")
		return w, w.Page()
	}
	var view atomic.Pointer[mygo.WebView]
	opts.Content = ui.View(func(c *ui.Context) {
		if v := view.Load(); v != nil {
			ui.WebView(c, v).Fill()
		}
	})
	w := newWindow(t, opts)
	v, err := w.NewWebView(mygo.WebViewOptions{Page: mygo.PageOptions{BackgroundThrottling: throttling}})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view.Store(v) })
	v.Page().LoadHTML(throttlingPage, "")
	waitForPage(t, v.Page(), "window.ready && frames.length > 2")
	return w, v.Page()
}

// TestBackgroundThrottling hides windows (then, where the platform can,
// minimizes them, and covers them with another window) and counts how
// often the timers of their pages ran meanwhile: with ThrottlingOff they
// keep close to their rate, 100 ticks a second, where a throttled page
// gets one a second or none. A page that keeps the default is only
// logged, as the animation frames are: whether and how much a web view
// throttles varies.
func TestBackgroundThrottling(t *testing.T) {
	if runtime.GOOS == "windows" && os.Getenv("MYGO_E2E_UNTHROTTLED") == "" {
		// WebView2's browser takes the arguments of the first page, the
		// warm-up's in this process: the test runs in one whose warm-up
		// asks for them, with a user data folder of its own, as a browser
		// running with other arguments refuses it this process's.
		ctx, cancel := context.WithTimeout(t.Context(), 4*wait.Guard)
		defer cancel()
		data, err := os.MkdirTemp("", "mygo-unthrottled")
		if err != nil {
			t.Fatal(err)
		}
		// The browser may hold the folder a moment after the test ends.
		defer os.RemoveAll(data)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBackgroundThrottling$", "-test.v")
		cmd.Env = append(os.Environ(), "MYGO_E2E_UNTHROTTLED=1", "LOCALAPPDATA="+data)
		out, err := cmd.CombinedOutput()
		t.Logf("%s", out)
		// The child's windows had the foreground, which the tests after
		// expect this process's to get.
		takeForeground(newWindow(t, mygo.WindowOptions{Title: "front", Width: 120, Height: 80}))
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	type state struct {
		name string
		// do changes the windows, and returns what changes them back.
		do func(t *testing.T, ws []*mygo.Window) (undo func())
	}
	states := []state{{"hidden", func(t *testing.T, ws []*mygo.Window) func() {
		for _, w := range ws {
			w.Hide()
		}
		for _, w := range ws {
			eventually(t, "the window hidden", func() bool { return !w.IsVisible() })
		}
		return func() {
			for _, w := range ws {
				w.Show()
			}
		}
	}}}
	if runtime.GOOS != "linux" { // Xvfb has no window manager to minimize or stack windows
		states = append(states, state{"minimized", func(t *testing.T, ws []*mygo.Window) func() {
			for _, w := range ws {
				w.Minimize()
			}
			for _, w := range ws {
				eventually(t, "the window minimized", w.IsMinimized)
			}
			return func() {
				for _, w := range ws {
					w.Restore()
				}
			}
		}}, state{"covered", func(t *testing.T, ws []*mygo.Window) func() {
			d := mygo.Screen.PrimaryDisplay()
			cover := mygo.NewWindow(mygo.WindowOptions{Title: "Cover", Frameless: true,
				X: d.Bounds.X, Y: d.Bounds.Y, Width: d.Bounds.Width, Height: d.Bounds.Height, AlwaysOnTop: true,
				Content: ui.View(func(c *ui.Context) { ui.Text(c, "Cover") })})
			cover.Focus()
			eventually(t, "the cover focused", cover.IsFocused)
			return cover.Destroy
		}})
	}
	for _, s := range states {
		for _, webView := range []bool{false, true} {
			name := s.name + "/window"
			if webView {
				name = s.name + "/web view"
			}
			t.Run(name, func(t *testing.T) {
				offWin, off := throttlingPageIn(t, webView, mygo.ThrottlingOff, 80)
				onWin, on := throttlingPageIn(t, webView, mygo.ThrottlingOn, 420)
				ok, settings := throttlingOff(offWin, webView)
				if settings == 0 && runtime.GOOS == "linux" {
					t.Skip("WebKitGTK older than 2.42 has no features to turn off")
				}
				if settings > 0 && !ok {
					t.Errorf("ThrottlingOff: the page's settings throttle it (of %d)", settings)
				}
				if ok, settings := throttlingOff(onWin, webView); settings > 0 && ok && runtime.GOOS != "windows" {
					// On Windows, the browser's arguments are every page's.
					t.Errorf("ThrottlingOn: the page's settings keep it running (of %d)", settings)
				}
				undo := sync.OnceFunc(s.do(t, []*mygo.Window{offWin, onWin}))
				t.Cleanup(undo) // a cover must not stay over the next tests
				from := time.Now().Add(throttleSettle)
				to := from.Add(throttleSpan)
				// Wait in the time of the unthrottled page, which goes on.
				waitForPage(t, off, fmt.Sprintf("Date.now() >= %d", to.UnixMilli()))
				undo()
				count := func(p *mygo.Page, list string) int {
					n, err := mygo.EvalAs[int](p, fmt.Sprintf("between(%s, %d, %d)", list, from.UnixMilli(), to.UnixMilli()))
					if err != nil {
						t.Fatal(err)
					}
					return n
				}
				waitForPage(t, on, "true") // back, it answers again
				offTicks := count(off, "ticks")
				t.Logf("%s for %v: ThrottlingOff %d ticks, %d frames; ThrottlingOn %d ticks, %d frames",
					name, throttleSpan, offTicks, count(off, "frames"), count(on, "ticks"), count(on, "frames"))
				// 10 a second: a tenth of the timer's rate, ten times a
				// throttled page's.
				if want := int(throttleSpan / (100 * time.Millisecond)); offTicks < want {
					t.Errorf("%s with ThrottlingOff: %d ticks of a 10 ms timer in %v, want %d or more", name, offTicks, throttleSpan, want)
				}
			})
		}
	}
}
