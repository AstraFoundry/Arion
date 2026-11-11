//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessLiveRegionAttributes reads, as Orca does through ATK, the
// object attributes of landmarks and live regions: their ARIA role in
// xml-roles, how a live region is read, and, on what is inside it, how
// the region around it is.
func TestAccessLiveRegionAttributes(t *testing.T) {
	var frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Live regions", Width: 400, Height: 400, Content: ui.View(landmarksAndLiveRegions(&frames))})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
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
	for _, c := range []struct {
		label string
		attrs map[string]string
	}{
		{"Activity", map[string]string{"xml-roles": "region"}},
		{"Site search", map[string]string{"xml-roles": "search"}},
		{"Messages", map[string]string{"xml-roles": "log", "live": "polite", "container-live": "polite", "container-live-role": "log", "atomic": "false"}},
		{"Hi", map[string]string{"container-live": "polite", "container-live-role": "log", "container-relevant": "additions text"}},
		{"Time left", map[string]string{"xml-roles": "timer"}},
		{"Error", map[string]string{"xml-roles": "alert", "live": "assertive", "atomic": "true"}},
		{"Name is required", map[string]string{"container-live": "assertive", "container-atomic": "true"}},
	} {
		eventually(t, c.label+"'s attributes", func() bool {
			n, ok := find(c.label)
			if !ok {
				return false
			}
			for k, v := range c.attrs {
				if n.Attributes[k] != v {
					return false
				}
			}
			return true
		})
	}
	if n, _ := find("Time left"); n.Attributes["live"] != "" || n.Attributes["container-live"] != "" {
		t.Errorf("the timer is a live region: %v", n.Attributes)
	}
}
