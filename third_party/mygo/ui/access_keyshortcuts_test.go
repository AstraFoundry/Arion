package ui_test

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAccessKeyShortcuts reads the keys that press elements
// (aria-keyshortcuts) as assistive technology gets them: as the
// platform's menus write them, several separated by spaces, each once,
// and none for a key with no name or an element not given one.
func TestAccessKeyShortcuts(t *testing.T) {
	palette, shortcuts := false, true
	tt := ui.NewTester(func(c *ui.Context) {
		if c.Shortcut(ui.Cmd, ui.KeyK) {
			palette = !palette
		}
		ui.Row(c).Role(ui.RoleToolbar).Label("Tools").Children(func() {
			search := ui.Button(c, "Search")
			if shortcuts {
				search.KeyShortcuts(ui.Cmd, ui.KeyK)
			}
			if search.Clicked() {
				palette = !palette
			}
			ui.Button(c, "Redo").KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyZ).KeyShortcuts(ui.Cmd, ui.KeyY).KeyShortcuts(ui.Cmd, ui.KeyY)
			ui.Button(c, "Help").KeyShortcuts(ui.Super|ui.Alt, ui.KeyF1).KeyShortcuts(0, ui.KeyBack)
			ui.Button(c, "Close").Disabled(true).KeyShortcuts(0, ui.KeyEscape)
			ui.Button(c, "Share")
		})
	}, 600, 200)

	want := map[string]string{
		"Search": "Ctrl+K",
		"Redo":   "Ctrl+Shift+Z Ctrl+Y",
		"Help":   "Super+Alt+F1",
		"Close":  "Escape",
		"Share":  "",
	}
	switch runtime.GOOS {
	case "darwin":
		want["Search"], want["Redo"], want["Help"], want["Close"] = "⌘K", "⇧⌘Z ⌘Y", "⌥⌘F1", "⎋"
	case "windows":
		want["Help"] = "Win+Alt+F1"
	}
	for name, keys := range want {
		if a := accessOf(t, tt, ui.RoleButton, name); a.KeyShortcuts != keys {
			t.Errorf("%s's key shortcuts %q, want %q", name, a.KeyShortcuts, keys)
		}
	}
	// It only tells the key, which the view handles.
	tt.Key(ui.Cmd, ui.KeyK)
	if !palette {
		t.Error("Cmd+K did not open the palette")
	}
	// An element tells the keys it is given in the frame.
	shortcuts = false
	tt.Frame()
	if a := accessOf(t, tt, ui.RoleButton, "Search"); a.KeyShortcuts != "" {
		t.Errorf("Search keeps the key shortcuts %q", a.KeyShortcuts)
	}
}
