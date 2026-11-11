package ui_test

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// accessCell builds a cell of a table showing its name.
func accessCell(c *ui.Context, name string) ui.Element {
	return ui.Box(c).Role(ui.RoleCell).Label(name).Padding(4).Children(func() { ui.Text(c, name) })
}

// A spreadsheet of 10,000 rows built only where it shows tells assistive
// technology its size and which of all its rows and cells are, and that
// several rows are chosen, which say so with Selected.
func TestAccessGrid(t *testing.T) {
	chosen := map[int]bool{4120: true, 4121: true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Role(ui.RoleGrid).Label("Sheet").RowCount(10001).ColumnCount(3).Multiselectable(true).Children(func() {
			ui.Row(c).Role(ui.RoleRow).RowIndex(1).Children(func() {
				for _, col := range []string{"A", "B", "C"} {
					ui.Box(c).Role(ui.RoleColumnHeader).Label(col)
				}
			})
			for r := 4119; r < 4122; r++ {
				ui.Row(c).Role(ui.RoleRow).RowIndex(r + 2).Selected(chosen[r]).Children(func() {
					accessCell(c, fmt.Sprintf("A%d", r+1))
					// Column B is left out, as scrolled away.
					accessCell(c, fmt.Sprintf("C%d", r+1)).ColumnIndex(3)
				})
			}
		})
	}, 400, 400)

	grid := accessOf(t, tt, ui.RoleGrid, "Sheet")
	if grid.RowCount != 10001 || grid.ColumnCount != 3 || !grid.Multiselectable || !grid.Selectable {
		t.Errorf("the grid: %+v", grid)
	}
	for r, want := range map[int]bool{4120: false, 4121: true, 4122: true} {
		row := accessOf(t, tt, ui.RoleRow, fmt.Sprintf("A%d C%d", r, r))
		if row.RowIndex != r+1 || !row.Selectable || row.Selected != want || row.Checked {
			t.Errorf("row %d: %+v", r, row)
		}
	}
	if head := accessOf(t, tt, ui.RoleRow, "A B C"); head.RowIndex != 1 || head.Selectable {
		t.Errorf("the header row: %+v", head)
	}
	if c := accessOf(t, tt, ui.RoleCell, "C4121"); c.ColumnIndex != 3 || c.RowIndex != 0 {
		t.Errorf("C4121: %+v", c)
	}
	if c := accessOf(t, tt, ui.RoleCell, "A4121"); c.ColumnIndex != 0 {
		t.Errorf("A4121: %+v", c)
	}
	// Choosing one alone.
	chosen = map[int]bool{4121: true}
	tt.Frame()
	if row := accessOf(t, tt, ui.RoleRow, "A4121 C4121"); row.Selected {
		t.Errorf("row 4121 is still chosen: %+v", row)
	}
}

// A hand-built data grid's cells covering several columns or rows tell
// assistive technology so (aria-colspan, aria-rowspan), whether their
// row lays them out in a grid or not; a span to the last column counts
// the row's grid's columns, and other elements spanning say nothing.
func TestAccessGridSpans(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Role(ui.RoleGrid).Label("Budget").Children(func() {
			ui.Row(c).Role(ui.RoleRow).Children(func() {
				ui.Box(c).Role(ui.RoleRowHeader).Label("Design").RowSpan(2)
				accessCell(c, "First half").ColumnSpan(2)
				accessCell(c, "9").ColumnSpan(1)
			})
			ui.Grid(c).Role(ui.RoleRow).Columns(4).Children(func() {
				accessCell(c, "Q3").ColumnStart(2)
				accessCell(c, "Second half").ColumnSpan(-1)
				ui.Box(c).Label("Note").ColumnSpan(2)
			})
		})
	}, 400, 400)

	for _, c := range []struct {
		role       ui.Role
		name       string
		rows, cols int
	}{
		{ui.RoleRowHeader, "Design", 2, 0},
		{ui.RoleCell, "First half", 0, 2},
		{ui.RoleCell, "9", 0, 0},
		{ui.RoleCell, "Q3", 0, 0},
		{ui.RoleCell, "Second half", 0, 4},
		{ui.RoleGroup, "Note", 0, 0},
	} {
		if a := accessOf(t, tt, c.role, c.name); a.RowSpan != c.rows || a.ColumnSpan != c.cols {
			t.Errorf("%s spans %d rows and %d columns, want %d and %d", c.name, a.RowSpan, a.ColumnSpan, c.rows, c.cols)
		}
	}
}

// A List shown as a grid has rows, which say which of all they are, and
// chooses several as its Selection does; a tree grid holds its cells, and
// one that chooses one row at a time is not multiselectable.
func TestAccessGridList(t *testing.T) {
	var sel ui.Selection[int]
	cursor := -1
	s := ui.ListState{Selected: &cursor, Selection: &sel}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.List(c, &s, 10000, func(i int) {
				ui.Row(c).Children(func() {
					accessCell(c, fmt.Sprintf("Name %d", i))
					accessCell(c, fmt.Sprintf("Size %d", i))
				})
			}).Role(ui.RoleGrid).Label("Files").ColumnCount(2).Height(200)
			ui.Column(c).Role(ui.RoleTreeGrid).Label("Outline").Multiselectable(false).Children(func() {
				ui.Row(c).Role(ui.RoleRow).Level(1).Expanded(true).Children(func() {
					accessCell(c, "src")
				})
			})
		})
	}, 400, 400)
	tt.Click("Name 1")
	tt.ClickWith(ui.Cmd, "Name 3")

	grid := accessOf(t, tt, ui.RoleGrid, "Files")
	if grid.SetSize != 10000 || grid.ColumnCount != 2 || !grid.Multiselectable || !grid.Selectable {
		t.Errorf("the grid: %+v", grid)
	}
	if _, ok := tt.AccessByRole(ui.RoleListItem, ""); ok {
		t.Error("the grid's rows are list items")
	}
	for i, want := range []bool{false, true, false, true} {
		row := accessOf(t, tt, ui.RoleRow, fmt.Sprintf("Name %d Size %d", i, i))
		if row.PosInSet != i+1 || row.SetSize != 10000 || !row.Selectable || row.Selected != want {
			t.Errorf("row %d: %+v", i, row)
		}
	}
	tree := accessOf(t, tt, ui.RoleTreeGrid, "Outline")
	if tree.Multiselectable || tree.Selectable {
		t.Errorf("the tree grid: %+v", tree)
	}
	if row := accessOf(t, tt, ui.RoleRow, "src"); row.Level != 1 || !row.Expanded {
		t.Errorf("the tree grid's row: %+v", row)
	}
	accessOf(t, tt, ui.RoleCell, "src")
}
