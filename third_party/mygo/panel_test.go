//go:build !js

package mygo

import "testing"

// TestPanelOptions gives a panel's backend a window above others, out of
// the taskbar, that cannot minimize, maximize or go full screen.
func TestPanelOptions(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Panel: true})
	o := fw.Opts
	if !w.IsPanel() || !o.Panel || !o.AlwaysOnTop || !o.SkipTaskbar || o.Minimizable || o.Maximizable || o.Fullscreenable {
		t.Errorf("panel: IsPanel %v, options %+v", w.IsPanel(), o)
	}
	if !o.Focusable || !o.Resizable || !o.Closable {
		t.Errorf("a panel lost abilities it keeps: %+v", o)
	}
	w, fw = testWindow(t, WindowOptions{})
	if w.IsPanel() || fw.Opts.Panel {
		t.Error("a window without Panel is a panel")
	}
}

// TestPanelEscape closes a panel on an Escape its page leaves, as the
// user closing it, which OnClose listeners may prevent.
func TestPanelEscape(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{Panel: true})
	keep, closes := true, 0
	w.OnClose(func(e *CloseEvent) {
		closes++
		if keep {
			e.PreventDefault()
		}
	})
	escape := func() {
		page(fw, `{"t":"escape"}`)
		onMain(func() {}) // the close posted after the message
	}
	escape()
	if closes != 1 || w.IsDestroyed() {
		t.Fatalf("OnClose heard %d closes, destroyed %v", closes, w.IsDestroyed())
	}
	keep = false
	escape()
	if !w.IsDestroyed() {
		t.Error("Escape did not close the panel")
	}
}
