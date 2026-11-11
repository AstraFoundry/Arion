//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessBusyAndCurrent reads, as Orca does through ATK, a region
// loading, busy until a press loads it, and a breadcrumb trail: a landmark
// whose xml-roles is navigation, its last crumb the current page, active.
func TestAccessBusyAndCurrent(t *testing.T) {
	var frames atomic.Int32
	loading := true
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			chosen := 0
			ui.Breadcrumbs(c, []string{"Home", "Settings"}, &chosen).Label("Path")
			if ui.Button(c, "Load").Clicked() {
				loading = false
			}
			ui.Column(c).Label("Messages").Busy(loading).Children(func() {
				ui.Text(c, "Inbox")
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Busy and current", Width: 400, Height: 300, Content: ui.View(view)})
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
	eventually(t, "the navigation landmark", func() bool {
		n, ok := find("Path")
		return ok && n.Role == "landmark" && n.Attributes["xml-roles"] == "navigation"
	})
	eventually(t, "the current page", func() bool {
		n, ok := find("Settings")
		return ok && n.Active && n.Attributes["current"] == "page"
	})
	if n, _ := find("Home"); n.Active || n.Attributes["current"] != "" {
		t.Errorf("the link home is current: %+v", n)
	}
	eventually(t, "the busy region", func() bool {
		n, ok := find("Messages")
		return ok && n.Busy
	})
	if !accessPerform(w, "Load", "press", "") {
		t.Fatal("cannot press the button")
	}
	eventually(t, "the region loaded", func() bool {
		n, ok := find("Messages")
		return ok && !n.Busy
	})
}
