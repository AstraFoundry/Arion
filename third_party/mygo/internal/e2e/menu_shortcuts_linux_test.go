//go:build linux

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The shortcuts of the app's menu items work in a window of native UI:
// before the content, which gets the rest. The roles' keys, as Edit ›
// Copy's Ctrl+C, go to the content first, as a terminal's Ctrl+C.
func TestNativeMenuShortcutsLinux(t *testing.T) {
	var settings, split, copied, typed int
	prev := mygo.App.Menu()
	defer mygo.App.SetMenu(prev)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "App", Submenu: []*mygo.MenuItem{
			{Label: "Settings", Accelerator: "Ctrl+,", Click: func(*mygo.MenuItem, *mygo.Window) { settings++ }},
			{Label: "Split", Accelerator: "Alt+Shift+=", Click: func(*mygo.MenuItem, *mygo.Window) { split++ }},
		}},
		{Role: mygo.RoleEditMenu},
	}))
	w := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, Content: ui.View(func(c *ui.Context) {
		c.OnShortcut(ui.Ctrl, ui.KeyC, func() { copied++ })
		c.OnShortcut(ui.Ctrl, ui.KeyComma, func() { typed++ })
		ui.Text(c, "shortcuts")
	})})
	// A click gives the window the keyboard, which needs no window manager.
	if !click(w, 50, 50) {
		t.Skip("click automation unavailable")
	}
	counts := func() (s, p, c, ty int) {
		mygo.RunOnMain(func() { s, p, c, ty = settings, split, copied, typed })
		return
	}
	if !pressKeys("Control_L", "comma") {
		t.Skip("no X server to press keys with")
	}
	eventually(t, "Ctrl+, to run Settings", func() bool { s, _, _, _ := counts(); return s == 1 })
	pressKeys("Alt_L", "Shift_L", "equal")
	eventually(t, "Alt+Shift+= to run Split", func() bool { _, p, _, _ := counts(); return p == 1 })
	pressKeys("Control_L", "c")
	eventually(t, "the content's Ctrl+C", func() bool { _, _, c, _ := counts(); return c == 1 })
	if _, _, _, ty := counts(); ty != 0 {
		t.Errorf("Ctrl+, reached the content %d times", ty)
	}
}

// An edit role's item, made insensitive when its menu opened with nothing
// of the native UI to act on, takes its shortcut again in a page of the
// window: GTK drops the shortcut of an insensitive item.
func TestWebViewEditShortcutLinux(t *testing.T) {
	prev := mygo.App.Menu()
	defer mygo.App.SetMenu(prev)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Label: "Edit", Submenu: []*mygo.MenuItem{
		{Role: mygo.RoleSelectAll, Accelerator: "Ctrl+Shift+Y"},
	}}}))
	var view *mygo.WebView
	var frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Text(c, "native")
			if view != nil {
				ui.WebView(c, view).Grow(1)
			}
		})
	})})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(`<textarea id=t>MyGo</textarea><script>window.ready = true</script>`, "")
	waitForPage(t, v.Page(), "window.ready")
	if !click(w, 20, 8) {
		t.Skip("click automation unavailable")
	}
	if on, err := menuItemEnabled(w, "Edit", "Select All"); err != nil || on {
		t.Fatalf("nothing focused: Edit > Select All enabled = %v (%v), want false", on, err)
	}
	if !pressKeys("Tab") {
		t.Skip("no X server to press keys with")
	}
	eventually(t, "Tab into the page", func() bool {
		id, _ := mygo.EvalAs[string](v.Page(), "document.hasFocus() && document.activeElement ? document.activeElement.id : ''")
		return id == "t"
	})
	if _, err := v.Page().Eval("document.getElementById('t').setSelectionRange(0, 0)"); err != nil {
		t.Fatal(err)
	}
	pressKeys("Control_L", "Shift_L", "y")
	eventually(t, "Ctrl+Shift+Y to select the page's text", func() bool {
		n, _ := mygo.EvalAs[int](v.Page(), "document.getElementById('t').selectionEnd - document.getElementById('t').selectionStart")
		return n == 4
	})
}
