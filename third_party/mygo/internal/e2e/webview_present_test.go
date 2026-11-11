//go:build !js

package e2e

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// screenPixel returns the color the display shows at (x, y), in DIPs, of
// a window of native UI, web views under it included, where a platform
// can read it.
func screenPixel(w *mygo.Window, x, y float64) (r, g, b uint8, ok bool) {
	if r, g, b, ok := screenColor(w, x, y); ok {
		return r, g, b, true
	}
	data, ok := surfaceOnScreen(w)
	if !ok || data == nil {
		return 0, 0, 0, false
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, 0, false
	}
	s := deviceScale(w)
	c := img.At(int(x*s), int(y*s))
	if !(image.Point{int(x * s), int(y * s)}).In(img.Bounds()) {
		return 0, 0, 0, false
	}
	cr, cg, cb, _ := c.RGBA()
	return uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), true
}

func reddish(r, g, b uint8) bool { return r > 180 && g < 80 && b < 80 }

func greenish(r, g, b uint8) bool { return g > 150 && r < 80 && b < 80 }

// TestContentWindowWebViewRoundedScaled shows a red page in a rounded web
// view over a window of native UI with nothing painted around it, as a
// peek panel over a live window does, in a panel that then scales by
// half: the page shows inside its corners only, keeps its layout as it
// shows scaled, and takes clicks where it shows, at its own coordinates,
// but in the corners cut off.
func TestContentWindowWebViewRoundedScaled(t *testing.T) {
	var view *mygo.WebView
	var scale atomic.Value // float32
	scale.Store(float32(1))
	var frame atomic.Pointer[ui.Rect]
	w := newWindow(t, mygo.WindowOptions{Title: "Web view rounded", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		// A patch of native UI, where the display is read to tell it can
		// be.
		ui.Box(c).Absolute().Left(4).Top(4).Size(40, 40).Background(ui.RGB(0, 200, 0))
		k := scale.Load().(float32)
		ui.Box(c).Absolute().Left(60).Top(50).Size(280, 200).Scale(k, k).Children(func() {
			if view != nil {
				e := ui.WebView(c, view).Grow(1).Radius(40)
				e.Draw(func(*ui.Painter, ui.Rect) {
					// Where it shows, scaled.
					x0, y0 := e.LocalToWindow(0, 0)
					x1, y1 := e.LocalToWindow(280, 200)
					frame.Store(&ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0})
				})
			}
		})
	})})
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	// The page notes its sizes as it resizes, with what the test was
	// doing then.
	v.Page().LoadHTML(`<!doctype html><body style="margin:0;height:100vh;background:#f00"
	onmousedown="window.downs.push([event.clientX, event.clientY])">
<script>
window.downs = []; window.resizes = []; window.phase = "loading"
onresize = () => window.resizes.push(window.phase + " " + innerWidth + "x" + innerHeight + "@" + devicePixelRatio)
requestAnimationFrame(() => requestAnimationFrame(() => { window.ready = true }))
</script></body>`, "")
	waitForPage(t, v.Page(), "window.ready")
	w.Focus()
	at := func(want float32) ui.Rect {
		t.Helper()
		eventually(t, "the web view placed", func() bool {
			w.Invalidate()
			f := frame.Load()
			return f != nil && math.Abs(float64(f.W-280*want)) < 0.5
		})
		return *frame.Load()
	}
	// The display is read as it shows the native UI around the page, which
	// goes the same way to the screen: where it cannot be read (macOS
	// here; the Windows ARM64 runners, whose GDI never sees what
	// DirectComposition composes, as in TestContentWindowTransparent),
	// the patch never reads green, and what the page shows is not
	// compared.
	var patch string
	readable := false
	for deadline := time.Now().Add(3 * time.Second); !readable && time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		r, g, b, ok := screenPixel(w, 24, 24)
		patch = fmt.Sprintf("#%02x%02x%02x (read %v)", r, g, b, ok)
		readable = ok && greenish(r, g, b)
	}
	if !readable {
		t.Logf("the display shows %s where the window paints native UI green: not reading what the page shows", patch)
	}
	shows := func(what string, f ui.Rect) {
		t.Helper()
		if !readable {
			return
		}
		points := [][2]float32{{f.X + f.W/2, f.Y + f.H/2}, {f.X + 3, f.Y + 3}, {f.X + f.W - 4, f.Y + 3}, {f.X + f.W - 4, f.Y + f.H - 4}, {f.X + 3, f.Y + f.H - 4}}
		var seen string
		if !wait.Poll(func() bool {
			seen = ""
			good := true
			for i, p := range points {
				r, g, b, _ := screenPixel(w, float64(p[0]), float64(p[1]))
				seen += fmt.Sprintf(" (%.0f, %.0f) #%02x%02x%02x", p[0], p[1], r, g, b)
				good = good && reddish(r, g, b) == (i == 0)
			}
			return good
		}) {
			t.Fatalf("%s: the display shows%s: want the page red in the middle, not in its corners", what, seen)
		}
	}
	downs := func() [][2]float64 {
		d, _ := mygo.EvalAs[[][2]float64](v.Page(), "window.downs")
		return d
	}
	// The page's frames up to now drawn, with the resizes they bring.
	drawn := func() {
		t.Helper()
		if _, err := v.Page().Eval("window.drawn = false; requestAnimationFrame(() => requestAnimationFrame(() => { window.drawn = true }))"); err != nil {
			t.Fatal(err)
		}
		waitForPage(t, v.Page(), "window.drawn")
	}
	resizes := func() []string {
		r, _ := mygo.EvalAs[[]string](v.Page(), "window.resizes")
		return r
	}

	f := at(1)
	shows("unscaled", f)
	if !click(w, float64(f.X+f.W/2), float64(f.Y+f.H/2)) {
		t.Skip("click automation not available on this platform")
	}
	eventually(t, "the click on the page", func() bool { return len(downs()) == 1 })
	drawn()
	before := len(resizes())
	if _, err := v.Page().Eval(`window.phase = "scaled"`); err != nil {
		t.Fatal(err)
	}

	scale.Store(float32(0.5))
	f = at(0.5)
	shows("scaled by half", f)
	drawn()
	if n, _ := mygo.EvalAs[float64](v.Page(), "innerWidth"); n != 280 {
		t.Errorf("scaled by half, the page is %v wide: want 280, as unscaled", n)
	}
	// WebKitGTK zooms the page to show it scaled, which lays it out again
	// at the same width, telling it of a resize.
	if r := resizes(); len(r) > before && runtime.GOOS != "linux" {
		t.Errorf("scaled by half, the page resized: %q: want it only shown smaller", r[before:])
	}
	// A click a quarter of the way in lands a quarter of the way into the
	// page, at its own coordinates.
	click(w, float64(f.X+f.W/4), float64(f.Y+f.H/4))
	eventually(t, "the click on the scaled page", func() bool { return len(downs()) == 2 })
	if d := downs()[1]; math.Abs(d[0]-70) > 2 || math.Abs(d[1]-50) > 2 {
		t.Errorf("the click a quarter of the way into the scaled page reached it at %v: want 70, 50", d)
	}
	// In a corner cut off, the click is the content's.
	click(w, float64(f.X+2), float64(f.Y+2))
	click(w, float64(f.X+f.W/2), float64(f.Y+f.H/2))
	eventually(t, "the click after the one in the corner", func() bool { return len(downs()) >= 3 })
	if n := len(downs()); n != 3 {
		t.Errorf("the page got %d clicks: want none in its corner cut off", n-2)
	}
}
