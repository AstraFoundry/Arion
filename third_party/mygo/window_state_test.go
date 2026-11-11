//go:build !js

package mygo

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

func TestWindowState(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, windowStateFile)
	App.SetPath(PathUserData, dir)
	delay := windowStateDelay
	windowStateDelay = 20 * time.Millisecond
	forget := func() { windowStates.loaded, windowStates.byKey, windowStates.dirty = false, nil, false }
	onMain(forget)
	// A new process: the states are written as when the app quits, and read
	// again.
	relaunch := func() { onMain(func() { saveWindowStates(); forget() }) }
	// Runs once the test's windows closed and wrote their states in dir.
	t.Cleanup(func() {
		onMain(forget)
		App.SetPath(PathUserData, "")
		windowStateDelay = delay
	})
	saved := func() map[string]WindowState {
		t.Helper()
		var m map[string]WindowState
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	captured := func(key string, want WindowState) {
		t.Helper()
		var got WindowState
		if !wait.Poll(func() bool {
			got = onMainValue(func() WindowState { return windowStates.byKey[key] })
			return got == want
		}) {
			t.Fatalf("%s captured as %+v, want %+v", key, got, want)
		}
	}

	// Nothing saved: the options apply. The window is remembered once it
	// settled, and written when it closes.
	w, fw := testWindow(t, WindowOptions{StateKey: "main", X: 100, Y: 120, Width: 700, Height: 500})
	if o := fw.Opts; o.X != 100 || o.Y != 120 || o.Width != 700 || o.Maximized {
		t.Errorf("first window options = %+v", o)
	}
	onMain(func() { fw.SetBounds(platform.Rect{X: 200, Y: 150, Width: 800, Height: 600}) })
	captured("main", WindowState{X: 200, Y: 150, Width: 800, Height: 600})
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("states written before a window closed: %v", err)
	}
	// Maximized, it keeps its normal bounds.
	onMain(func() {
		fw.Maximize()
		fw.SetBounds(platform.Rect{Y: 25, Width: 1440, Height: 875})
	})
	w.Destroy()
	if got, want := saved()["main"], (WindowState{X: 200, Y: 150, Width: 800, Height: 600, Maximized: true}); got != want {
		t.Errorf("saved %+v, want %+v", got, want)
	}

	// The next launch gets it back.
	relaunch()
	_, fw = testWindow(t, WindowOptions{StateKey: "main", Width: 400, Height: 300, UseContentSize: true})
	if o := fw.Opts; o.X != 200 || o.Y != 150 || o.Width != 800 || o.Height != 600 || !o.Maximized || o.Center || o.UseContentSize {
		t.Errorf("restored options = %+v", o)
	}

	// Off every display: centered, with its size as far as it fits.
	onMain(func() { windowStates.byKey["gone"] = WindowState{X: 5000, Y: 100, Width: 2000, Height: 700} })
	_, fw = testWindow(t, WindowOptions{StateKey: "gone"})
	if o := fw.Opts; !o.Center || o.Width != 1440 || o.Height != 700 {
		t.Errorf("off screen options = %+v", o)
	}

	// Created maximized, the requested bounds are its normal ones.
	testWindow(t, WindowOptions{StateKey: "max", Maximized: true, Width: 1000, Height: 700})
	if got, want := onMainValue(func() WindowState { return windowStates.byKey["max"] }), (WindowState{X: 220, Y: 112, Width: 1000, Height: 700, Maximized: true}); got != want {
		t.Errorf("maximized at creation: %+v, want %+v", got, want)
	}

	// A damaged file is ignored.
	relaunch()
	if err := os.WriteFile(file, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, fw = testWindow(t, WindowOptions{StateKey: "main", X: 10, Y: 30})
	if o := fw.Opts; o.X != 10 || o.Y != 30 || o.Maximized {
		t.Errorf("options with a damaged file = %+v", o)
	}

	// A save that fails, as when Windows cannot replace a file that another
	// program has open, keeps the states for the next one.
	if err := os.Mkdir(file+".tmp", 0o755); err != nil { // no temporary file can be written
		t.Fatal(err)
	}
	want := WindowState{X: 1, Y: 2, Width: 300, Height: 200}
	kept := onMainValue(func() bool {
		windowStates.byKey["failed"] = want
		windowStates.dirty = true
		saveWindowStates()
		return windowStates.dirty
	})
	if !kept {
		t.Error("a failed save drops the states")
	}
	if err := os.Remove(file + ".tmp"); err != nil {
		t.Fatal(err)
	}
	onMain(saveWindowStates)
	if got := saved()["failed"]; got != want {
		t.Errorf("saved %+v after a failed save, want %+v", got, want)
	}
}

// memoryStates is a WindowStateStore in memory, such as an app's database.
// Main thread only.
type memoryStates struct {
	loads  int
	states map[string]WindowState
	err    error
}

func (m *memoryStates) LoadWindowStates() (map[string]WindowState, error) {
	m.loads++
	return maps.Clone(m.states), nil
}

func (m *memoryStates) SaveWindowStates(states map[string]WindowState) error {
	if m.err != nil {
		return m.err
	}
	m.states = states
	return nil
}

func TestWindowStateStore(t *testing.T) {
	dir := t.TempDir()
	App.SetPath(PathUserData, dir)
	forget := func() { windowStates.loaded, windowStates.byKey, windowStates.dirty = false, nil, false }
	onMain(forget)
	store := &memoryStates{states: map[string]WindowState{"main": {X: 30, Y: 40, Width: 640, Height: 480, Maximized: true}}}
	App.SetWindowStateStore(store)
	// Runs once the test's windows closed and saved their states.
	t.Cleanup(func() {
		onMain(forget)
		App.SetWindowStateStore(nil)
		App.SetPath(PathUserData, "")
		if _, err := os.Stat(filepath.Join(dir, windowStateFile)); !os.IsNotExist(err) {
			t.Errorf("%s written with a store: %v", windowStateFile, err)
		}
	})

	// The store's states apply, loaded once.
	_, fw := testWindow(t, WindowOptions{StateKey: "main", Width: 400, Height: 300})
	if o := fw.Opts; o.X != 30 || o.Y != 40 || o.Width != 640 || o.Height != 480 || !o.Maximized || o.Center {
		t.Errorf("restored options = %+v", o)
	}
	w, _ := testWindow(t, WindowOptions{StateKey: "other", X: 10, Y: 20, Width: 500, Height: 400})
	if loads := onMainValue(func() int { return store.loads }); loads != 1 {
		t.Errorf("states loaded %d times, want once", loads)
	}

	// A closing window saves all of them.
	w.Destroy()
	saved := onMainValue(func() map[string]WindowState { return store.states })
	if got, want := saved["other"], (WindowState{X: 10, Y: 20, Width: 500, Height: 400}); got != want {
		t.Errorf("saved %+v, want %+v", got, want)
	}
	if _, ok := saved["main"]; !ok {
		t.Errorf("saved %v without main", saved)
	}
	// The store may keep them.
	onMain(func() { windowStates.byKey["later"] = WindowState{Width: 1, Height: 1} })
	if _, ok := saved["later"]; ok {
		t.Error("the saved states change with the windows'")
	}

	// A save that fails keeps the states for the next one.
	want := WindowState{X: 1, Y: 2, Width: 300, Height: 200}
	kept := onMainValue(func() bool {
		store.err = errors.New("database is locked")
		windowStates.byKey["failed"] = want
		windowStates.dirty = true
		saveWindowStates()
		store.err = nil
		return windowStates.dirty
	})
	if !kept {
		t.Error("a failed save drops the states")
	}
	onMain(saveWindowStates)
	if got := onMainValue(func() WindowState { return store.states["failed"] }); got != want {
		t.Errorf("saved %+v after a failed save, want %+v", got, want)
	}
}

// pixelScreen, pixelBackend and pixelWindow stand for a backend that
// places windows in pixels, as Windows does.
type pixelScreen struct {
	platform.Screen
	displays []platform.Display
}

func (s pixelScreen) Displays() []platform.Display { return s.displays }

type pixelBackend struct {
	platform.Backend
	displays []platform.Display
	accept   bool
	pixels   platform.Rect // what RestoreWindowState was given
}

func (b *pixelBackend) Screen() platform.Screen { return pixelScreen{displays: b.displays} }

func (b *pixelBackend) RestoreWindowState(o *platform.WindowOptions, pixels platform.Rect) bool {
	b.pixels = pixels
	if !b.accept || pixels.Width <= 0 {
		return false
	}
	o.StateBounds, o.Center = &pixels, false
	return true
}

type pixelWindow struct {
	platform.Window
	bounds, pixels                   platform.Rect
	maximized, fullScreen, minimized bool
}

func (w *pixelWindow) Bounds() platform.Rect            { return w.bounds }
func (w *pixelWindow) WindowStateBounds() platform.Rect { return w.pixels }
func (w *pixelWindow) IsMaximized() bool                { return w.maximized }
func (w *pixelWindow) IsFullScreen() bool               { return w.fullScreen }
func (w *pixelWindow) IsMinimized() bool                { return w.minimized }

// TestWindowStatePixels restores a state's pixels, read back from JSON,
// through a backend that places windows in pixels, whatever its DIPs.
func TestWindowStatePixels(t *testing.T) {
	data, err := json.Marshal(WindowState{X: 2286, Y: 57, Width: 900, Height: 600, Maximized: true, PixelX: 4000, PixelY: 100, PixelWidth: 1575, PixelHeight: 1050})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"x":2286,"y":57,"width":900,"height":600,"maximized":true,"pixelX":4000,"pixelY":100,"pixelWidth":1575,"pixelHeight":1050}`; string(data) != want {
		t.Errorf("JSON = %s, want %s", data, want)
	}
	var st WindowState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	b := &pixelBackend{accept: true, displays: []platform.Display{{Primary: true, WorkArea: platform.Rect{Width: 1920, Height: 1040}, ScaleFactor: 2}}}
	p := &platform.WindowOptions{Width: 1180, Height: 780, Center: true, UseContentSize: true}
	restoreWindowStateFrom(st, p, b)
	want := platform.Rect{X: 4000, Y: 100, Width: 1575, Height: 1050}
	if b.pixels != want || p.StateBounds == nil || *p.StateBounds != want {
		t.Fatalf("pixels = %+v, StateBounds = %+v, want %+v", b.pixels, p.StateBounds, want)
	}
	if p.Center || p.UseContentSize || !p.Maximized || p.Width != 900 || p.Height != 600 {
		t.Errorf("restored options = %+v", *p)
	}
}

// TestWindowStateWithoutPixels takes the DIPs of a state without pixels
// only where every display has the same scale, and centers the window
// whose pixels show on no display.
func TestWindowStateWithoutPixels(t *testing.T) {
	primary := platform.Display{Primary: true, WorkArea: platform.Rect{Width: 1920, Height: 1040}, ScaleFactor: 2}
	tests := []struct {
		name     string
		pixels   platform.Rect
		displays []platform.Display
		want     platform.WindowOptions
	}{
		{
			name:     "same scale",
			displays: []platform.Display{{Primary: true, WorkArea: platform.Rect{Width: 3840, Height: 2080}, ScaleFactor: 2}},
			want:     platform.WindowOptions{X: 100, Y: 100, Width: 2200, Height: 1200, FullScreen: true},
		},
		{
			name:     "different scales",
			displays: []platform.Display{primary, {WorkArea: platform.Rect{X: 1920, Width: 1280, Height: 720}, ScaleFactor: 1.25}},
			want:     platform.WindowOptions{Center: true, Width: 1920, Height: 1040, FullScreen: true},
		},
		{
			name:     "disconnected display",
			pixels:   platform.Rect{X: 8000, Y: 100, Width: 2200, Height: 1200},
			displays: []platform.Display{primary},
			want:     platform.WindowOptions{Center: true, Width: 1920, Height: 1040, FullScreen: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := WindowState{X: 100, Y: 100, Width: 2200, Height: 1200, FullScreen: true,
				PixelX: tt.pixels.X, PixelY: tt.pixels.Y, PixelWidth: tt.pixels.Width, PixelHeight: tt.pixels.Height}
			p := &platform.WindowOptions{}
			restoreWindowStateFrom(st, p, &pixelBackend{displays: tt.displays})
			if !reflect.DeepEqual(*p, tt.want) {
				t.Errorf("restored options = %+v, want %+v", *p, tt.want)
			}
		})
	}
	if !sameScale(nil) {
		t.Error("sameScale(nil) = false")
	}
}

// TestWindowStatePixelCapture captures a window's DIPs and pixels
// together, and keeps both while the window is maximized, full screen or
// minimized, or the system gives no pixels.
func TestWindowStatePixelCapture(t *testing.T) {
	n := &pixelWindow{
		bounds: platform.Rect{X: 2286, Y: 57, Width: 900, Height: 600},
		pixels: platform.Rect{X: 4001, Y: 101, Width: 1575, Height: 1050},
	}
	saved := captureWindowState(WindowState{}, n)
	want := WindowState{X: 2286, Y: 57, Width: 900, Height: 600, PixelX: 4001, PixelY: 101, PixelWidth: 1575, PixelHeight: 1050}
	if saved != want {
		t.Fatalf("captured %+v, want %+v", saved, want)
	}
	n.bounds, n.pixels = platform.Rect{Width: 1920, Height: 1040}, platform.Rect{Width: 3840, Height: 2080}
	for _, state := range []string{"maximized", "full screen", "minimized"} {
		n.maximized, n.fullScreen, n.minimized = state == "maximized", state == "full screen", state == "minimized"
		st := captureWindowState(saved, n)
		st.Maximized, st.FullScreen = false, false
		if st != saved {
			t.Errorf("%s: captured %+v, want the normal bounds %+v", state, st, saved)
		}
	}
	n.maximized, n.fullScreen, n.minimized = false, false, false
	n.pixels = platform.Rect{}
	if st := captureWindowState(saved, n); st != saved {
		t.Errorf("without pixels: captured %+v, want %+v", st, saved)
	}
}

func TestFileDrop(t *testing.T) {
	w, fw := readyWindow(t, WindowOptions{})
	drops := make(chan *FileDropEvent, 1)
	w.OnFileDrop(func(e *FileDropEvent) { drops <- e })

	fw.Dropped = []string{"/tmp/a.txt", "/tmp/b"}
	page(fw, `{"t":"drop","x":10.4,"y":20}`)
	select {
	case e := <-drops:
		if len(e.Paths) != 2 || e.Paths[0] != "/tmp/a.txt" || e.X != 10 || e.Y != 20 {
			t.Errorf("drop = %+v", e)
		}
	case <-time.After(wait.Guard):
		t.Fatal("OnFileDrop was not called")
	}
	// about:blank is the app's own page: it gets the paths too.
	m := received(t, fw, func(m map[string]any) bool { return m["t"] == "event" && m["n"] == "mygo:file-drop" })
	if p := m["p"].(map[string]any); p["x"] != float64(10) || len(p["paths"].([]any)) != 2 {
		t.Errorf("page event = %v", m)
	}

	// A drop without files (text, or a page posting on its own) is ignored.
	page(fw, `{"t":"drop","x":1,"y":2}`)
	select {
	case e := <-drops:
		t.Errorf("unexpected drop %+v", e)
	case <-time.After(50 * time.Millisecond):
	}

	defer func() {
		if recover() == nil {
			t.Error("NewEvent accepted a reserved name")
		}
	}()
	NewEvent[int]("mygo:file-drop")
}

func TestWindowExtras(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{SkipTaskbar: true, AutoHideMenuBar: true})
	if !fw.Opts.SkipTaskbar {
		t.Error("SkipTaskbar option not passed")
	}
	if !fw.Opts.AutoHideMenu {
		t.Error("AutoHideMenuBar option not passed")
	}
	for _, c := range []struct {
		p     ProgressBar
		state string
		value float64
	}{
		{ProgressBar{Value: 0.25}, "normal", 0.25},
		{ProgressBar{State: ProgressNormal}, "normal", 0},
		{ProgressBar{State: ProgressError, Value: 2}, "error", 1},
		{ProgressBar{State: ProgressIndeterminate}, "indeterminate", 0},
		{ProgressBar{Value: -1}, "", 0},
		{ProgressBar{}, "", 0},
	} {
		w.SetProgressBar(c.p)
		if st, v := onMainValue(func() string { return fw.Progress }), onMainValue(func() float64 { return fw.ProgressValue }); st != c.state || v != c.value {
			t.Errorf("SetProgressBar(%+v) = %q %v, want %q %v", c.p, st, v, c.state, c.value)
		}
	}
	w.FlashFrame(true)
	w.SetSkipTaskbar(false)
	w.SetVisibleOnAllWorkspaces(true)
	w.SetAutoHideMenuBar(true)
	if !onMainValue(func() bool { return fw.Flashing && !fw.SkipsTaskbar && fw.OnAllWorkspaces && fw.AutoHidesMenu }) {
		t.Error("window extras not applied")
	}
	if err := w.SetIcon([]byte("not a png")); err == nil {
		t.Error("SetIcon accepted a bad image")
	}
}

func TestPrintToPDFOptions(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	pdf, err := w.Page().PrintToPDF(PDFOptions{})
	if err != nil || string(pdf) != "%PDF-1.4 fake" {
		t.Fatalf("PrintToPDF = %q, %v", pdf, err)
	}
	want := platform.PDFOptions{PageWidth: 8.5, PageHeight: 11, MarginTop: 0.4, MarginRight: 0.4, MarginBottom: 0.4, MarginLeft: 0.4}
	if got := onMainValue(func() platform.PDFOptions { return fw.PDF }); got != want {
		t.Errorf("defaults = %+v, want %+v", got, want)
	}
	w.Page().PrintToPDF(PDFOptions{PageSize: PageA4, Landscape: true, Margins: &Margins{}, Background: true})
	want = platform.PDFOptions{Landscape: true, PageWidth: 8.27, PageHeight: 11.69, Background: true}
	if got := onMainValue(func() platform.PDFOptions { return fw.PDF }); got != want {
		t.Errorf("options = %+v, want %+v", got, want)
	}
}
