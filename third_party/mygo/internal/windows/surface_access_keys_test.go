//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// The keys that press an element (aria-keyshortcuts) are its
// AcceleratorKey, as Chromium gives them; an element without has none.
func TestUIAAcceleratorKey(t *testing.T) {
	tree := newTestTree(t, false)
	tree.update(&platform.AccessTree{Nodes: []platform.AccessNode{
		{ID: 1, Parent: -1, Role: platform.RoleToolbar, Label: "Tools"},
		{ID: 2, Parent: 0, Role: platform.RoleButton, Label: "Search", KeyShortcuts: "Ctrl+K"},
		{ID: 3, Parent: 0, Role: platform.RoleButton, Label: "Redo", KeyShortcuts: "Ctrl+Shift+Z Ctrl+Y"},
		{ID: 4, Parent: 0, Role: platform.RoleButton, Label: "Share"},
	}})
	for id, want := range map[uint64]string{2: "Ctrl+K", 3: "Ctrl+Shift+Z Ctrl+Y"} {
		var v variant
		if tree.nodes[id].property(uiaAcceleratorKeyProperty, &v); v.VT != vtBSTR || bstrString(uintptr(v.Val)) != want {
			t.Errorf("node %d's AcceleratorKey %+v, want %q", id, v, want)
		}
	}
	var v variant
	if tree.nodes[4].property(uiaAcceleratorKeyProperty, &v); v.VT != vtEmpty {
		t.Errorf("Share's AcceleratorKey %+v, want none", v)
	}
}
