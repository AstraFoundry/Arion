//go:build darwin

package e2e

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestAccessDescribedByErrorMessageSearch reads, as VoiceOver does, a
// search field of the app's own described by a hint (AXHelp), invalid,
// and pointing at the element holding its error (AXErrorMessageElements,
// as WebKit's aria-errormessage), which it names only while invalid.
func TestAccessDescribedByErrorMessageSearch(t *testing.T) {
	var frames atomic.Int32
	var bad atomic.Bool
	bad.Store(true)
	view := func(c *ui.Context) {
		frames.Add(1)
		var query string
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			hint := ui.Text(c, "Titles only")
			msg := ui.Text(c, "Type a word")
			ui.TextInput(c, &query).Label("Find").Search().DescribedBy(hint).Invalid(bad.Load()).ErrorMessage(msg)
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Relations", Width: 400, Height: 300, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	for _, c := range []struct{ attr, want string }{
		{"AXSubrole", "AXSearchField"},
		{"AXHelp", "Type a word\nTitles only"},
		{"AXInvalid", "true"},
		{"AXErrorMessageElements", "MyGoAccessibilityElement"},
	} {
		eventually(t, "the field's "+c.attr, func() bool {
			v, _, named, ok := axAttribute(w, "Find", c.attr)
			return ok && (named || c.attr == "AXSubrole") && strings.Contains(v, c.want)
		})
	}
	bad.Store(false)
	w.Invalidate()
	eventually(t, "the valid field", func() bool {
		v, _, named, ok := axAttribute(w, "Find", "AXErrorMessageElements")
		help, _, _, _ := axAttribute(w, "Find", "AXHelp")
		return ok && !named && v == "" && help == "Titles only"
	})
}
