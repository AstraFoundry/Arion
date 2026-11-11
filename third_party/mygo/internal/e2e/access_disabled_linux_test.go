//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessDisabled reads, as Orca does through ATK, a button that is
// only unavailable (AccessDisabled, aria-disabled): not enabled, yet
// focusable, and focused when assistive technology asks, where a button
// Disabled is neither.
func TestAccessDisabled(t *testing.T) {
	var frames atomic.Int32
	var focused atomic.Bool
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Padding(20).Gap(10).Children(func() {
			del := ui.Button(c, "Delete project").AccessDisabled(true)
			focused.Store(del.Focused())
			ui.Button(c, "Close").Disabled(true)
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Unavailable", Width: 400, Height: 200, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	states := func(label string) (enabled, focusable, ok bool) {
		mygo.RunOnMain(func() { enabled, focusable, ok = linux.TestAccessibilityEnabled(w.NativeHandle(), label) })
		return enabled, focusable, ok
	}
	eventually(t, "the unavailable button", func() bool {
		enabled, focusable, ok := states("Delete project")
		return ok && !enabled && focusable
	})
	if enabled, focusable, ok := states("Close"); !ok || enabled || focusable {
		t.Errorf("Close: enabled %v, focusable %v, found %v", enabled, focusable, ok)
	}
	if accessPerform(w, "Delete project", "press", "") {
		t.Error("assistive technology pressed the unavailable button")
	}
	if accessPerform(w, "Close", "focus", "") {
		t.Error("assistive technology focused the disabled button")
	}
	if !accessPerform(w, "Delete project", "focus", "") {
		t.Fatal("cannot focus the unavailable button")
	}
	eventually(t, "the unavailable button focused", focused.Load)
}
