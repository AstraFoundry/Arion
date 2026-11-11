//go:build linux && (amd64 || arm64)

package e2e

import (
	"slices"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessDescribedByErrorOrientationSearch reads, as Orca does through
// ATK, a search field described by a hint (its description, described-by
// and description-for back), marked invalid without a message and then
// with the element holding it (error-message and error-for back, only
// while invalid), a toolbar and a separator told their orientation, and
// a price deleted and inserted.
func TestAccessDescribedByErrorOrientationSearch(t *testing.T) {
	var frames atomic.Int32
	var bad, message atomic.Bool
	view := func(c *ui.Context) {
		frames.Add(1)
		var query string
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			hint := ui.Text(c, "Titles only")
			in := ui.TextInput(c, &query).Label("Find").Search().DescribedBy(hint).Invalid(bad.Load())
			if message.Load() {
				in.ErrorMessage(ui.Text(c, "Type a word"))
			}
			ui.Column(c).Role(ui.RoleToolbar).Label("Tools").Orientation(ui.Vertical).Children(func() {
				ui.Button(c, "Bold")
			})
			ui.Box(c).Size(2, 40).Role(ui.RoleSeparator).Label("Rule").Orientation(ui.Horizontal)
			ui.Row(c).Gap(6).Children(func() {
				ui.Text(c, "$40").Strikethrough().Role(ui.RoleDeletion)
				ui.Text(c, "$30").Role(ui.RoleInsertion)
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Relations", Width: 400, Height: 400, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	read := func(label string) (n linux.TestAccessNode, states []string, relations map[string][]string, ok bool) {
		mygo.RunOnMain(func() {
			nodes, _ := linux.TestAccessibility(w.NativeHandle())
			for _, node := range nodes {
				if node.Label == label {
					n = node
				}
			}
			states, relations, ok = linux.TestAccessStatesAndRelations(w.NativeHandle(), label)
		})
		return n, states, relations, ok
	}
	eventually(t, "the search field described by its hint", func() bool {
		n, states, rel, ok := read("Find")
		return ok && n.Role == "entry" && n.Description == "Titles only" && n.Attributes["xml-roles"] == "searchbox" &&
			n.Attributes["text-input-type"] == "search" && slices.Equal(rel["described-by"], []string{"Titles only"}) &&
			!slices.Contains(states, "invalid-entry")
	})
	eventually(t, "the hint describing the field", func() bool {
		_, _, rel, ok := read("Titles only")
		return ok && slices.Equal(rel["description-for"], []string{"Find"})
	})
	eventually(t, "the toolbar running up and down", func() bool {
		_, states, _, ok := read("Tools")
		return ok && slices.Contains(states, "vertical") && !slices.Contains(states, "horizontal")
	})
	eventually(t, "the rule told it runs across", func() bool {
		_, states, _, ok := read("Rule")
		return ok && slices.Contains(states, "horizontal") && !slices.Contains(states, "vertical")
	})
	for label, role := range map[string]string{"$40": "content deletion", "$30": "content insertion"} {
		eventually(t, label+" as "+role, func() bool {
			n, _, _, ok := read(label)
			return ok && n.Role == role
		})
	}
	// Invalid without a message: the state alone, the description as it
	// was.
	bad.Store(true)
	w.Invalidate()
	eventually(t, "the field invalid without a message", func() bool {
		n, states, rel, ok := read("Find")
		return ok && slices.Contains(states, "invalid-entry") && n.Description == "Titles only" && rel["error-message"] == nil
	})
	// Then with the element holding it, read first.
	message.Store(true)
	w.Invalidate()
	eventually(t, "the field pointing at its error", func() bool {
		n, _, rel, ok := read("Find")
		return ok && n.Description == "Type a word\nTitles only" && slices.Equal(rel["error-message"], []string{"Type a word"})
	})
	eventually(t, "the error naming its field", func() bool {
		_, _, rel, ok := read("Type a word")
		return ok && slices.Equal(rel["error-for"], []string{"Find"})
	})
	// Valid again, the message points at nothing.
	bad.Store(false)
	w.Invalidate()
	eventually(t, "the valid field", func() bool {
		n, states, rel, ok := read("Find")
		return ok && !slices.Contains(states, "invalid-entry") && n.Description == "Titles only" && rel["error-message"] == nil
	})
}
