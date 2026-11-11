//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessKeyShortcuts reads, as Orca does through ATK, the keys that
// press a toolbar's buttons (aria-keyshortcuts) in their keyshortcuts
// object attribute, and none on a button without.
func TestAccessKeyShortcuts(t *testing.T) {
	var frames atomic.Int32
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Row(c).Role(ui.RoleToolbar).Label("Tools").Padding(20).Gap(10).Children(func() {
			ui.Button(c, "Search").KeyShortcuts(ui.Ctrl, ui.KeyK)
			ui.Button(c, "Redo").KeyShortcuts(ui.Ctrl|ui.Shift, ui.KeyZ).KeyShortcuts(ui.Ctrl, ui.KeyY)
			ui.Button(c, "Share")
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Key shortcuts", Width: 400, Height: 200, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	find := func(label string) (n linux.TestAccessNode, ok bool) {
		mygo.RunOnMain(func() {
			nodes, _ := linux.TestAccessibility(w.NativeHandle())
			for _, node := range nodes {
				if node.Label == label {
					n, ok = node, true
				}
			}
		})
		return n, ok
	}
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	for label, want := range map[string]string{"Search": "Ctrl+K", "Redo": "Ctrl+Shift+Z Ctrl+Y"} {
		eventually(t, label+"'s key shortcuts", func() bool {
			n, ok := find(label)
			return ok && n.Role == "button" && n.Attributes["keyshortcuts"] == want
		})
	}
	n, ok := find("Share")
	if _, has := n.Attributes["keyshortcuts"]; !ok || has {
		t.Errorf("Share: %+v, found %v", n, ok)
	}
}
