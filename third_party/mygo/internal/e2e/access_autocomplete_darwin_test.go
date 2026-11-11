//go:build darwin

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestAccessAutoComplete reads, as VoiceOver does, what the suggestions
// of combo boxes complete (aria-autocomplete) in AXAutocompleteValue, as
// WebKit's elements give it; a text field without suggestions names no
// such attribute.
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
	for label, want := range map[string]string{"City": "list", "Word": "both"} {
		eventually(t, label+"'s AXAutocompleteValue", func() bool {
			v, _, named, ok := axAttribute(w, label, "AXAutocompleteValue")
			return ok && named && v == want
		})
	}
	if v, _, named, ok := axAttribute(w, "Note", "AXAutocompleteValue"); !ok || named {
		t.Errorf("Note's AXAutocompleteValue %q, named %v, found %v", v, named, ok)
	}
}
