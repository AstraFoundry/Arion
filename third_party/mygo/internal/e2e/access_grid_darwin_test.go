//go:build darwin

package e2e

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/ui"
)

// TestAccessGrid reads, as VoiceOver does, a spreadsheet of 10,000 rows
// built only where it shows: a table whose size is in AXARIARowCount and
// AXARIAColumnCount, and AXRowCount, whose rows say which of all they are
// (AXIndex) and, with its cells, their places in AXARIARowIndex and
// AXARIAColumnIndex, as WebKit's grids do; a cell without a column of its
// own names no AXARIAColumnIndex, and one covering two columns says so in
// the length of its AXColumnIndexRange (aria-colspan), the cell after it
// in the column after those.
func TestAccessGrid(t *testing.T) {
	var frames atomic.Int32
	cell := func(c *ui.Context, name string) ui.Element {
		return ui.Box(c).Role(ui.RoleCell).Label(name).Padding(2).Children(func() { ui.Text(c, name) })
	}
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleGrid).Label("Sheet").RowCount(10001).ColumnCount(3).Multiselectable(true).Children(func() {
				for r := 4120; r < 4122; r++ {
					ui.Row(c).Role(ui.RoleRow).RowIndex(r + 1).Selected(r == 4121).Children(func() {
						cell(c, fmt.Sprintf("A%d", r))
						cell(c, fmt.Sprintf("C%d", r)).ColumnIndex(3)
					})
				}
				ui.Row(c).Role(ui.RoleRow).RowIndex(10001).Children(func() {
					cell(c, "Total").ColumnSpan(2)
					cell(c, "Sum")
				})
			})
			ui.Column(c).Role(ui.RoleTreeGrid).Label("Outline").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Level(1).Children(func() { cell(c, "src") })
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Grid", Width: 400, Height: 300, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	// find reads the element named label through the newer API, which
	// VoiceOver reads its role, rows and places through.
	find := func(label string) (n darwin.TestAccessNode, ok bool) {
		mygo.RunOnMain(func() {
			for _, node := range darwin.TestAccessibility(w.NativeHandle()) {
				if node.Label == label && !ok {
					n, ok = node, true
				}
			}
		})
		return n, ok
	}
	eventually(t, "the grid", func() bool {
		n, ok := find("Sheet")
		return ok && n.Role == "AXTable" && n.RowCount == 10001
	})
	if n, _ := find("Outline"); n.Role != "AXTable" {
		t.Errorf("the tree grid: %+v", n)
	}
	if n, _ := find("A4121 C4121"); n.Role != "AXRow" || n.Index != 4121 {
		t.Errorf("row 4,122: %+v", n)
	}
	// The older API, which says the role as the newer one does, and the
	// counts and places as WebKit's grids do.
	for _, c := range []struct{ label, attr, want string }{
		{"Sheet", "AXRole", "AXTable"},
		{"Sheet", "AXARIARowCount", "10001"},
		{"Sheet", "AXARIAColumnCount", "3"},
		{"A4121 C4121", "AXARIARowIndex", "4122"},
		{"C4121", "AXARIAColumnIndex", "3"},
		{"C4121", "AXARIARowIndex", "4122"},
		{"A4121", "AXARIARowIndex", "4122"},
	} {
		eventually(t, c.label+"'s "+c.attr, func() bool {
			v, _, named, ok := axAttribute(w, c.label, c.attr)
			return ok && (named || c.attr == "AXRole") && v == c.want
		})
	}
	if v, _, named, ok := axAttribute(w, "A4121", "AXARIAColumnIndex"); !ok || named {
		t.Errorf("A4121's AXARIAColumnIndex %q, named %v, found %v", v, named, ok)
	}
	// The ranges, which read as descriptions holding want.
	for _, c := range []struct{ label, attr, want string }{
		{"Total", "AXColumnIndexRange", "{0, 2}"},
		{"Total", "AXRowIndexRange", "{10000, 1}"},
		{"Sum", "AXColumnIndexRange", "{2, 1}"},
	} {
		eventually(t, c.label+"'s "+c.attr, func() bool {
			v, _, named, ok := axAttribute(w, c.label, c.attr)
			return ok && named && strings.Contains(v, c.want)
		})
	}
}
