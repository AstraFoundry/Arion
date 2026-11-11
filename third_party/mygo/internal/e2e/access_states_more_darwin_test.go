//go:build darwin

package e2e

import (
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestAccessRoleDescriptionPopupSelected reads, as VoiceOver does, a
// carousel's role description, a button opening a list box, and a
// grid's day chosen without painting it.
func TestAccessRoleDescriptionPopupSelected(t *testing.T) {
	var frames atomic.Int32
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleGroup).Label("Highlights").RoleDescription("carousel")
			ui.Button(c, "Fruit").HasPopup(ui.PopupListBox)
			ui.Column(c).Role(ui.RoleTable).Label("November").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Gap(8).Children(func() {
					ui.Text(c, "Day").Role(ui.RoleColumnHeader).Sort(ui.SortDescending)
					ui.Text(c, "Note").Role(ui.RoleColumnHeader)
				})
				ui.Row(c).Role(ui.RoleRow).Gap(8).Children(func() {
					ui.Text(c, "1").Role(ui.RoleCell).Checked(false)
					ui.Text(c, "2").Role(ui.RoleCell).Checked(true)
				})
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "States", Width: 400, Height: 300, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	// The headers' roles through the newer API, which VoiceOver reads,
	// as through the older one below.
	for label, want := range map[string]string{"Day": "AXButton/AXSortButton", "Note": "AXCell"} {
		eventually(t, label+"'s role", func() bool {
			nodes, _ := accessibility(w)
			return slices.ContainsFunc(nodes, func(n accessNode) bool {
				return n.label == label && n.role == want
			})
		})
	}
	// VoiceOver asks for the attributes an element names, and reads a
	// table's choice by its rows, as NSTableView's: the row of the chosen
	// cell is chosen, and the cells say which column they are in, under
	// which header. It reads no cell's AXSelected, so the chosen cell
	// says it after its name. The header sorting the rows is a sort
	// button, as NSTableHeaderView's.
	// The ranges and arrays read as descriptions, which hold want.
	for _, c := range []struct{ label, attr, want string }{
		{"Highlights", "AXRoleDescription", "carousel"},
		{"Fruit", "AXPopupValue", "listbox"},
		{"2, selected", "AXSelected", "1"},
		{"1", "AXSelected", "0"},
		{"1", "AXColumnIndexRange", "{0, 1}"},
		{"2, selected", "AXColumnIndexRange", "{1, 1}"},
		{"2, selected", "AXRowIndexRange", "{1, 1}"},
		{"1", "AXColumnHeaderUIElements", "MyGoAccessibilityElement"},
		{"November", "AXSelectedCells", "MyGoAccessibilityElement"},
		{"November", "AXSelectedRows", "MyGoAccessibilityElement"},
		{"November", "AXColumnHeaderUIElements", "MyGoAccessibilityElement"},
		{"Day", "AXRole", "AXButton"},
		{"Day", "AXSubrole", "AXSortButton"},
		{"Day", "AXSortDirection", "AXDescendingSortDirection"},
		{"Note", "AXRole", "AXCell"},
		{"Note", "AXSortDirection", "AXUnknownSortDirection"},
	} {
		eventually(t, c.label+"'s "+c.attr, func() bool {
			v, _, named, ok := axAttribute(w, c.label, c.attr)
			partial := strings.HasSuffix(c.attr, "Range") || strings.HasSuffix(c.attr, "s")
			return ok && (named || c.attr == "AXRole" || c.attr == "AXSubrole") && (v == c.want || partial && strings.Contains(v, c.want))
		})
	}
}
