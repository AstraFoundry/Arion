//go:build !js

package e2e

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestPanel shows a panel above other windows with the keyboard, unable
// to minimize or maximize, and closes it on an Escape its page leaves, as
// the user closing it, which an OnClose listener turns into hiding it. On
// macOS it floats on every Space, full-screen ones included, out of the
// window cycle, and takes the keyboard without activating the app.
func TestPanel(t *testing.T) {
	other := newWindow(t, mygo.WindowOptions{Title: "Other", Width: 300, Height: 200})
	other.Focus()
	eventually(t, "the other window focused", other.IsFocused)

	panel := newWindow(t, mygo.WindowOptions{Title: "Panel", Width: 400, Height: 120, Panel: true, Hidden: true})
	if !panel.IsPanel() || other.IsPanel() {
		t.Fatalf("IsPanel: panel %v, other window %v", panel.IsPanel(), other.IsPanel())
	}
	if !panel.IsAlwaysOnTop() || panel.IsMinimizable() || panel.IsMaximizable() {
		t.Errorf("panel: always on top %v, minimizable %v, maximizable %v", panel.IsAlwaysOnTop(), panel.IsMinimizable(), panel.IsMaximizable())
	}
	const canJoinAllSpaces, ignoresCycle, auxiliary = 1 << 0, 1 << 6, 1 << 8 // NSWindowCollectionBehavior…
	if b, ok := collectionBehavior(panel); ok && b&(canJoinAllSpaces|ignoresCycle|auxiliary) != canJoinAllSpaces|ignoresCycle|auxiliary {
		t.Errorf("collection behavior %#x, want CanJoinAllSpaces, IgnoresCycle and FullScreenAuxiliary", b)
	}
	if tool, topmost, owned, ok := panelStyles(panel); ok && !(tool && topmost && owned) {
		t.Errorf("panel: tool window without a window menu %v, topmost %v, owned by the application window %v", tool, topmost, owned)
	}
	panel.Page().LoadHTML(`<input id=i autofocus>`, "")
	waitFor(t, panel, "document.getElementById('i')")

	// Shown while another app is active, it takes the keyboard and leaves
	// that app active (macOS).
	deactivated := deactivateApp()
	if deactivated {
		eventually(t, "the app deactivated", func() bool { active, _ := appActive(); return !active })
	}
	panel.Show()
	eventually(t, "the panel focused", panel.IsFocused)
	waitFor(t, panel, "document.hasFocus() && document.activeElement.id === 'i'")
	if deactivated {
		time.Sleep(300 * time.Millisecond)
		if active, _ := appActive(); active {
			t.Error("showing the panel activated the app")
		}
	}

	var closes atomic.Int32
	off := panel.OnClose(func(e *mygo.CloseEvent) {
		closes.Add(1)
		e.PreventDefault()
		panel.Hide()
	})
	if !pressEscape(panel) {
		t.Skip("keys not available on this platform")
	}
	eventually(t, "the panel hidden on Escape", func() bool { return !panel.IsVisible() })
	if n := closes.Load(); n != 1 || panel.IsDestroyed() {
		t.Fatalf("after Escape: %d close requests, destroyed %v", n, panel.IsDestroyed())
	}

	// An Escape the page handles keeps it.
	panel.Show()
	eventually(t, "the panel focused again", panel.IsFocused)
	panel.Page().Eval(`window.keep = true; addEventListener('keydown', e => { if (e.key === 'Escape' && window.keep) { e.preventDefault(); window.escaped = true } })`)
	pressEscape(panel)
	waitFor(t, panel, "window.escaped === true")
	time.Sleep(300 * time.Millisecond)
	if !panel.IsVisible() || closes.Load() != 1 {
		t.Fatalf("an Escape the page handled: visible %v, %d close requests", panel.IsVisible(), closes.Load())
	}

	// Without the listener, an Escape closes it.
	off()
	panel.Page().Eval(`window.keep = false`)
	pressEscape(panel)
	eventually(t, "the panel closed on Escape", panel.IsDestroyed)
}

// TestPanelNativeUI closes a panel of native UI on an Escape its content
// leaves.
func TestPanelNativeUI(t *testing.T) {
	panel := newWindow(t, mygo.WindowOptions{Width: 300, Height: 100, Panel: true, Content: ui.View(func(c *ui.Context) {
		ui.Text(c, "Panel")
	})})
	eventually(t, "the panel focused", panel.IsFocused)
	time.Sleep(200 * time.Millisecond)
	if !pressEscape(panel) {
		t.Skip("keys not available on this platform")
	}
	eventually(t, "the panel closed on Escape", panel.IsDestroyed)
}
