//go:build darwin

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestAccessKeyShortcuts reads, as VoiceOver does, the keys that press a
// toolbar's buttons (aria-keyshortcuts) in AXKeyShortcutsValue, as
// WebKit's elements give them, written as the menus write them; a button
// without names no such attribute.
func TestAccessKeyShortcuts(t *testing.T) {
	var frames atomic.Int32
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Row(c).Role(ui.RoleToolbar).Label("Tools").Padding(20).Gap(10).Children(func() {
			ui.Button(c, "Search").KeyShortcuts(ui.Cmd, ui.KeyK)
			ui.Button(c, "Redo").KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyZ).KeyShortcuts(ui.Cmd, ui.KeyY)
			ui.Button(c, "Share")
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Key shortcuts", Width: 400, Height: 200, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	for label, want := range map[string]string{"Search": "⌘K", "Redo": "⇧⌘Z ⌘Y"} {
		eventually(t, label+"'s AXKeyShortcutsValue", func() bool {
			v, _, named, ok := axAttribute(w, label, "AXKeyShortcutsValue")
			return ok && named && v == want
		})
	}
	if v, _, named, ok := axAttribute(w, "Share", "AXKeyShortcutsValue"); !ok || named {
		t.Errorf("Share's AXKeyShortcutsValue %q, named %v, found %v", v, named, ok)
	}
}
