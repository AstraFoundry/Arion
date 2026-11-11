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

// TestAccessControlsPopupSortSelected reads, as Orca does through ATK, a
// carousel's role description, a disclosure button controlling the region
// it opens (controller-for, and controlled-by back), a button opening a
// list box (has-popup), a column sorted in another order, and a grid's
// day chosen without painting it (selected, not checked), with Checked
// or AccessSelected.
func TestAccessControlsPopupSortSelected(t *testing.T) {
	var frames atomic.Int32
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleGroup).Label("Highlights").RoleDescription("carousel")
			toggle := ui.Button(c, "Details").Expanded(true)
			details := ui.Column(c).Role(ui.RoleRegion).Label("More details").Children(func() {
				ui.Text(c, "Shipped today")
			})
			toggle.Controls(details)
			ui.Button(c, "Fruit").HasPopup(ui.PopupListBox)
			ui.Column(c).Role(ui.RoleTable).Label("November").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Gap(8).Children(func() {
					ui.Text(c, "Day").Role(ui.RoleColumnHeader).Sort(ui.SortOther)
				})
				ui.Row(c).Role(ui.RoleRow).Gap(8).Children(func() {
					ui.Text(c, "1").Role(ui.RoleCell).Checked(false)
					ui.Text(c, "2").Role(ui.RoleCell).Checked(true)
					ui.Text(c, "3").Role(ui.RoleCell).AccessSelected(true)
					ui.Text(c, "4").Role(ui.RoleCell).AccessSelected(false)
				})
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "States", Width: 400, Height: 400, Content: ui.View(view)})
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
	eventually(t, "the carousel", func() bool {
		n, _, _, ok := read("Highlights")
		return ok && n.Attributes["roledescription"] == "carousel"
	})
	eventually(t, "the button controlling the region", func() bool {
		_, _, rel, ok := read("Details")
		return ok && slices.Equal(rel["controller-for"], []string{"More details"})
	})
	eventually(t, "the region controlled by the button", func() bool {
		_, _, rel, ok := read("More details")
		return ok && slices.Equal(rel["controlled-by"], []string{"Details"})
	})
	eventually(t, "the button opening a list box", func() bool {
		n, states, _, ok := read("Fruit")
		return ok && n.Attributes["haspopup"] == "listbox" && slices.Contains(states, "has-popup")
	})
	eventually(t, "the column sorted otherwise", func() bool {
		n, _, _, ok := read("Day")
		return ok && n.Attributes["sort"] == "other"
	})
	eventually(t, "the day chosen", func() bool {
		_, states, _, ok := read("2")
		return ok && slices.Contains(states, "selectable") && slices.Contains(states, "selected") && !slices.Contains(states, "checked")
	})
	for _, day := range []string{"1", "4"} {
		if _, states, _, _ := read(day); !slices.Contains(states, "selectable") || slices.Contains(states, "selected") {
			t.Errorf("the day %s not chosen: %v", day, states)
		}
	}
	if _, states, _, _ := read("3"); !slices.Contains(states, "selectable") || !slices.Contains(states, "selected") || slices.Contains(states, "checked") {
		t.Errorf("the day 3 chosen with AccessSelected: %v", states)
	}
}
