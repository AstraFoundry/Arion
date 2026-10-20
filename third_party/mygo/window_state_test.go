//go:build !js

package mygo

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
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
		for deadline := time.Now().Add(2 * time.Second); ; time.Sleep(5 * time.Millisecond) {
			got := onMainValue(func() WindowState { return windowStates.byKey[key] })
			if got == want {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s captured as %+v, want %+v", key, got, want)
			}
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
	case <-time.After(time.Second):
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
