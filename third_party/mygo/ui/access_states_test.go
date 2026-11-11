package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestRoleDescriptionControlsPopupSort checks aria-roledescription,
// aria-controls, aria-haspopup and aria-sort on elements of your own: a
// carousel and its slide, a disclosure button and the region it opens, a
// button opening a list box, and the headers of a sortable table.
func TestRoleDescriptionControlsPopupSort(t *testing.T) {
	open := true
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreColumn(c).Role(RoleGroup).Label("Highlights").RoleDescription("carousel").Children(func() {
				coreColumn(c).Role(RoleGroup).Label("1 of 3").RoleDescription("slide")
			})
			toggle := coreButton(c, "Details").Expanded(open)
			if open {
				details := coreColumn(c).Role(RoleRegion).Label("More details").Children(func() {
					coreText(c, "Shipped today")
				})
				toggle.Controls(details).Controls(details)
			}
			coreButton(c, "Fruit").HasPopup(PopupListBox)
			coreColumn(c).Role(RoleTable).Label("Files").Children(func() {
				coreRow(c).Role(RoleRow).Children(func() {
					coreText(c, "Name").Role(RoleColumnHeader).Sort(SortAscending)
					coreText(c, "Size").Role(RoleColumnHeader).Sort(SortOther)
					coreText(c, "Kind").Role(RoleColumnHeader)
				})
			})
		})
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	if n := accessNode(t, tree, platform.RoleGroup, "Highlights"); n.RoleDescription != "carousel" {
		t.Errorf("the carousel's role description %q", n.RoleDescription)
	}
	if n := accessNode(t, tree, platform.RoleGroup, "1 of 3"); n.RoleDescription != "slide" {
		t.Errorf("the slide's role description %q", n.RoleDescription)
	}
	details := accessNode(t, tree, platform.RoleRegion, "More details")
	if n := accessNode(t, tree, platform.RoleButton, "Details"); !slices.Equal(n.Controls, []uint64{details.ID}) {
		t.Errorf("the button controls %v, want the region %d", n.Controls, details.ID)
	}
	if n := accessNode(t, tree, platform.RoleButton, "Fruit"); n.HasPopup != platform.PopupListBox || n.HasPopup.String() != "listbox" {
		t.Errorf("the button opens %v", n.HasPopup)
	}
	sort := platform.AccessSortAscending | platform.AccessSortDescending | platform.AccessSortOther
	for label, want := range map[string]platform.AccessStates{"Name": platform.AccessSortAscending, "Size": platform.AccessSortOther, "Kind": 0} {
		if n := accessNode(t, tree, platform.RoleColumnHeader, label); n.States&sort != want {
			t.Errorf("%s sorts %b, want %b", label, n.States&sort, want)
		}
	}
	// Closed, the button controls nothing.
	open = false
	tt.Frame()
	if n := accessNode(t, tt.h.access, platform.RoleButton, "Details"); len(n.Controls) != 0 {
		t.Errorf("the closed button controls %v", n.Controls)
	}
}

// TestCellSelected checks aria-selected on cells: Checked chooses a cell
// of a grid without painting it, as a calendar's day, and the table holds
// the choice; Selected paints it too; cells saying nothing are not
// choosable.
func TestCellSelected(t *testing.T) {
	picked := 2
	var cells [4]*node
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Role(RoleTable).Label("November").Children(func() {
			coreRow(c).Role(RoleRow).Children(func() {
				for d := range 3 {
					cells[d] = coreText(c, string(rune('1'+d))).Role(RoleCell).Checked(d == picked)
				}
				cells[3] = coreText(c, "4").Role(RoleCell).Selected(picked == 3)
			})
		})
		coreColumn(c).Role(RoleTable).Label("Plain").Children(func() {
			coreRow(c).Role(RoleRow).Children(func() {
				coreText(c, "x").Role(RoleCell)
			})
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	for d, label := range []string{"1", "2", "3", "4"} {
		n := accessNode(t, tree, platform.RoleCell, label)
		if n.States&platform.AccessSelectable == 0 {
			t.Errorf("cell %s is not choosable", label)
		}
		if got := n.States&platform.AccessChecked != 0; got != (d == picked) {
			t.Errorf("cell %s chosen %v", label, got)
		}
	}
	if cells[2].bg != (Color{}) {
		t.Error("Checked painted the cell")
	}
	if n := accessNode(t, tree, platform.RoleTable, "November"); n.States&platform.AccessSelectable == 0 {
		t.Error("the grid does not hold its cells' choice")
	}
	if n := accessNode(t, tree, platform.RoleCell, "x"); n.States&platform.AccessSelectable != 0 {
		t.Error("a cell saying nothing is choosable")
	}
	if n := accessNode(t, tree, platform.RoleTable, "Plain"); n.States&platform.AccessSelectable != 0 {
		t.Error("a table of plain cells chooses them")
	}
	// The word VoiceOver is given to read after a chosen cell's name.
	if tree.Selected != "selected" {
		t.Errorf("the tree's word for a chosen cell is %q", tree.Selected)
	}
	picked = 3
	tt.Frame()
	if n := accessNode(t, tt.h.access, platform.RoleCell, "4"); n.States&platform.AccessChecked == 0 || cells[3].bg == (Color{}) {
		t.Error("Selected neither chose nor painted the cell")
	}
}
