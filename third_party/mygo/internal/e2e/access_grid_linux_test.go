//go:build linux && (amd64 || arm64)

package e2e

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessGrid reads, as Orca does through ATK, a spreadsheet of 10,000
// rows built only where it shows: a table whose xml-roles is grid, with
// its size in the rowcount and colcount attributes, choosing several rows
// (multiselectable, AtkSelection), whose rows and cells say their places
// (rowindex, colindex), a cell covering two columns its span (colspan);
// a List shown as a grid, whose rows are table rows; and a tree grid, a
// tree table.
func TestAccessGrid(t *testing.T) {
	var frames atomic.Int32
	cell := func(c *ui.Context, name string) ui.Element {
		return ui.Box(c).Role(ui.RoleCell).Label(name).Padding(2).Children(func() { ui.Text(c, name) })
	}
	cursor := -1
	files := ui.ListState{Selected: &cursor}
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleGrid).Label("Sheet").RowCount(10001).ColumnCount(3).Multiselectable(true).Children(func() {
				ui.Row(c).Role(ui.RoleRow).RowIndex(1).Children(func() {
					for _, col := range []string{"A", "B", "C"} {
						ui.Box(c).Role(ui.RoleColumnHeader).Label(col).Padding(2).Children(func() { ui.Text(c, col) })
					}
				})
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
			ui.List(c, &files, 500, func(i int) {
				ui.Row(c).Children(func() { cell(c, fmt.Sprintf("File %d", i)) })
			}).Role(ui.RoleGrid).Label("Files").Height(100)
			ui.Column(c).Role(ui.RoleTreeGrid).Label("Outline").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Level(1).Children(func() { cell(c, "src") })
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Grid", Width: 400, Height: 500, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	// find returns the first element named label, of the role named role
	// if given.
	find := func(label string, role ...string) (n linux.TestAccessNode, ok bool) {
		mygo.RunOnMain(func() {
			nodes, _ := linux.TestAccessibility(w.NativeHandle())
			for _, node := range nodes {
				if node.Label == label && (len(role) == 0 || node.Role == role[0]) && !ok {
					n, ok = node, true
				}
			}
		})
		return n, ok
	}
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	eventually(t, "the grid", func() bool {
		n, ok := find("Sheet")
		return ok && n.Role == "table" && n.Attributes["xml-roles"] == "grid" && n.Attributes["rowcount"] == "10001" &&
			n.Attributes["colcount"] == "3" && n.Multiselectable && n.Selection
	})
	row, _ := find("A4121 C4121")
	if row.Role != "table row" || row.Attributes["rowindex"] != "4122" || !row.Selectable || !row.Selected {
		t.Errorf("the row chosen: %+v", row)
	}
	if row, _ := find("A4120 C4120"); row.Attributes["rowindex"] != "4121" || !row.Selectable || row.Selected {
		t.Errorf("the row not chosen: %+v", row)
	}
	if c, _ := find("C4121"); c.Role != "table cell" || c.Attributes["colindex"] != "3" || c.Attributes["rowindex"] != "4122" {
		t.Errorf("C4121: %+v", c)
	}
	if c, ok := find("A4121"); !ok || c.Attributes["colindex"] != "" {
		t.Errorf("A4121 has a column index of its own: %+v", c)
	}
	if c, _ := find("Total"); c.Role != "table cell" || c.Attributes["colspan"] != "2" || c.Attributes["rowspan"] != "" || c.Attributes["rowindex"] != "10001" {
		t.Errorf("a cell covering two columns: %+v", c)
	}
	if c, ok := find("Sum"); !ok || c.Attributes["colspan"] != "" {
		t.Errorf("a cell covering one column: %+v", c)
	}
	if h, _ := find("B"); h.Role != "column header" && h.Role != "table column header" || h.Attributes["rowindex"] != "1" {
		t.Errorf("the header B: %+v", h)
	}
	if g, _ := find("Files"); g.Role != "table" || g.Attributes["xml-roles"] != "grid" || g.Multiselectable || g.Attributes["setsize"] != "500" {
		t.Errorf("the list shown as a grid: %+v", g)
	}
	if r, ok := find("File 1", "table row"); !ok || r.Attributes["posinset"] != "2" || r.Attributes["setsize"] != "500" {
		t.Errorf("a row of the list shown as a grid: %+v", r)
	}
	if g, _ := find("Outline"); g.Role != "tree table" || g.Attributes["xml-roles"] != "treegrid" {
		t.Errorf("the tree grid: %+v", g)
	}
}
