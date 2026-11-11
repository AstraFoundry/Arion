//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessAutoComplete reads, as Orca does through ATK, what the
// suggestions of combo boxes complete (aria-autocomplete): the
// supports-autocompletion state and the autocomplete object attribute, as
// Core-AAM maps it, and neither on a text field without suggestions.
func TestAccessAutoComplete(t *testing.T) {
	var frames atomic.Int32
	var city, word, note string
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Padding(20).Gap(10).Children(func() {
			ui.Autocomplete(c, &city, []string{"Paris", "Prague"}).Label("City")
			ui.TextInputBase(c, &word).Role(ui.RoleComboBox).Label("Word").AutoComplete(ui.AutoCompleteBoth)
			ui.TextInput(c, &note).Label("Note")
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Autocomplete", Width: 400, Height: 300, Content: ui.View(view)})
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
	for label, want := range map[string]string{"City": "list", "Word": "both"} {
		eventually(t, label+"'s autocomplete", func() bool {
			n, ok := find(label)
			return ok && n.Role == "combo box" && n.SupportsAutocompletion && n.Attributes["autocomplete"] == want
		})
	}
	n, ok := find("Note")
	if _, has := n.Attributes["autocomplete"]; !ok || has || n.SupportsAutocompletion {
		t.Errorf("Note: %+v, found %v", n, ok)
	}
}
