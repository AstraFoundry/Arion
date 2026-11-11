//go:build windows

package e2e

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	win "github.com/egoist/mygo/internal/windows"
)

// Alt alone, or held with Control or Shift, keeps a popup menu MyGo shows
// open, as it keeps open the menus of Windows 11 itself, WinUI's flyouts:
// the keyboard still reaches the menu afterwards to choose an item, which it
// couldn't were the menu gone.
func TestPageMenuKeepsOpenOnAlt(t *testing.T) {
	if !pageMenusHooked {
		t.Skip("right-clicking a page is not hooked here")
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Alt menu", Width: 400, Height: 300})
	w.Page().LoadHTML(contextMenuPage, "")
	waitFor(t, w, "document.readyState === 'complete' && window.mygo")

	chosen := make(chan struct{}, 1)
	off := w.Page().OnContextMenu(func(e *mygo.ContextMenuEvent) {
		// The web view's items gone, one of MyGo's own left: the keyboard's
		// first Down and Return reach it.
		for _, it := range e.Menu.Items() {
			if it.ID != "" {
				e.Menu.Remove(it.ID)
			}
		}
		e.Menu.Append(&mygo.MenuItem{Label: "Choose", Click: func(*mygo.MenuItem, *mygo.Window) {
			chosen <- struct{}{}
		}})
	})
	defer off()

	for _, c := range []struct {
		name string
		keys []byte
	}{
		{"Alt", []byte{0x12}},                    // VK_MENU
		{"Control then Alt", []byte{0x11, 0x12}}, // VK_CONTROL, which Windows takes for AltGr
		{"Shift then Alt", []byte{0x10, 0x12}},   // VK_SHIFT
	} {
		// The keyboard types in the window in front.
		front := false
		mygo.RunOnMain(func() { front = win.TestForeground(w.NativeHandle()) })
		if !front {
			t.Skip("the window cannot come to the front, where the keyboard types")
		}
		if !rightClickPage(w, 20, 20) {
			t.Fatalf("%s: right-clicking the page failed", c.name)
		}
		wait.Until(t, c.name+": the menu shown", func() bool { _, shown := pageMenu(); return shown })
		keys := c.keys
		mygo.RunOnMain(func() { win.TestPressKeys(keys...) })
		// The menu's own keys, after theirs: they reach it only if it
		// stayed open, as Windows 11's menus do.
		mygo.RunOnMain(func() { win.TestPressKeys(0x28, 0x0D) }) // VK_DOWN, VK_RETURN
		select {
		case <-chosen:
		case <-time.After(wait.Guard):
			if _, shown := pageMenu(); shown {
				t.Fatalf("%s: the menu stayed, and every key missed it", c.name)
			}
			t.Fatalf("%s: the menu left: the keyboard never chose its item", c.name)
		}
		wait.Until(t, c.name+": the menu closed", func() bool { _, shown := pageMenu(); return !shown })
	}
}

// Alt alone leaves the menus of a window's own menu bar, as Win32 has them
// leave: MyGo keeps it open only in the popups it shows itself, not in the
// dropdowns Windows opens from the bar.
func TestMenuBarClosesOnAlt(t *testing.T) {
	prev := mygo.App.Menu()
	defer mygo.App.SetMenu(prev)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Label: "App", Submenu: []*mygo.MenuItem{{Label: "Item"}}}}))
	w := newWindow(t, mygo.WindowOptions{Title: "Alt in the bar", Width: 400, Height: 300})
	eventually(t, "the menu bar on the window", func() bool { shown, _ := menuBarShown(w); return shown })
	if !openMenus(w, 0) {
		t.Skip("the window cannot come to the front, where the keyboard types")
	}
	// A Down in the bar opens its first menu, a menu of Windows'.
	mygo.RunOnMain(func() { win.TestPressKeys(0x28) }) // VK_DOWN
	eventually(t, "the first menu of the bar open", func() bool { menus, _ := popupMenus(); return len(menus) > 0 })
	mygo.RunOnMain(func() { win.TestPressKeys(0x12) }) // VK_MENU
	eventually(t, "Alt to close the menu of the bar", func() bool { menus, _ := popupMenus(); return len(menus) == 0 })
	closeMenus()
}
