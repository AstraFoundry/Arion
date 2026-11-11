package ui

import (
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// cellTable shows a table that chooses blocks of cells (CellSelect), of
// n rows: a column of row headers, "#", then columns A to the letter of
// count, each width wide (sharing the room for 0); cell r1c2 is row 1's
// in column B. Each cell has a context menu.
func cellTable(t *testing.T, n, count int, width float32) (*Tester, *ListState, *Selection[int]) {
	t.Helper()
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}}
	for k := range count {
		cols = append(cols, TableColumn{Title: string(rune('A' + k)), Width: width})
	}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, n, func(row, col int) {
			Textf(c, "r%dc%d", row, col).ContextMenu(func(m *Menu) { m.Item("Copy") })
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	return tt, s, sel
}

func wantBlock(t *testing.T, s *ListState, sel *Selection[int], what string, rows []int, cells ColumnRange, cols []string) {
	t.Helper()
	if got := chosenRows(sel); !slices.Equal(got, rows) || s.Cells != cells || !slices.Equal(s.SelectedColumns(), cols) {
		t.Fatalf("%s: rows %v, cells %+v, columns %v; want %v, %+v, %v", what, got, s.Cells, s.SelectedColumns(), rows, cells, cols)
	}
}

// A click on a cell chooses its row and its column, a Shift-click the
// rows and the columns from there, a Cmd-click adds the row and keeps the
// columns, and a drag chooses the rows and the columns it goes over.
func TestCellSelectClickShiftDrag(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.Click("r1c1")
	wantBlock(t, s, sel, "click r1c1", []int{1}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
	tt.ClickWith(Shift, "r3c3")
	wantBlock(t, s, sel, "Shift-click r3c3", []int{1, 2, 3}, ColumnRange{Cursor: "C", Anchor: "A"}, []string{"A", "B", "C"})
	tt.ClickWith(Cmd, "r5c2")
	wantBlock(t, s, sel, "Cmd-click r5c2", []int{1, 2, 3, 5}, ColumnRange{Cursor: "C", Anchor: "A"}, []string{"A", "B", "C"})
	x, y := rowCenter(t, tt, "r0c2")
	x2, y2 := rowCenter(t, tt, "r2c1")
	tt.Press(x, y)
	tt.Move(x, y+3)
	tt.Move(x2, y2)
	tt.Release(x2, y2)
	wantBlock(t, s, sel, "drag r0c2 to r2c1", []int{0, 1, 2}, ColumnRange{Cursor: "A", Anchor: "B"}, []string{"A", "B"})
}

// A click chooses the column it pressed, though the pointer moves on
// before the frame that applies it.
func TestCellSelectClickColumnIsWherePressed(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	x, y := rowCenter(t, tt, "r1c1")
	x2, _ := rowCenter(t, tt, "r1c3")
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x2), Y: float64(y)})
	tt.Frame()
	wantBlock(t, s, sel, "click r1c1, pointer on r1c3 by the frame", []int{1}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
}

// The cells of the block show as chosen rows do, in the accent color
// under their text, which takes the accent's text color; a cell holding
// the focus, as an editor in it, shows as it is, without the cursor's
// border.
func TestCellSelectBlockLooksChosen(t *testing.T) {
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
	value := "edit"
	var editCell, plainCell *node
	var theme *Theme
	tt := NewTester(func(c *Context) {
		theme = c.Theme()
		Table(c, s, cols, 4, func(row, col int) {
			switch {
			case row == 1 && col == 1:
				editCell = TextInputBase(c, &value).node().parent
			case row == 1 && col == 2:
				plainCell = Textf(c, "r%dc%d", row, col).node().parent
			default:
				Textf(c, "r%dc%d", row, col)
			}
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c2")
	tt.Key(Shift, KeyLeft)
	wantBlock(t, s, sel, "r1 A..B", []int{1}, ColumnRange{Cursor: "A", Anchor: "B"}, []string{"A", "B"})
	chosen := func(n *node) bool { return n.bg == theme.Accent && n.ts.color == theme.AccentText }
	if !chosen(editCell) || !chosen(plainCell) || plainCell.paintAfterFn != nil {
		t.Fatal("the block's cells do not show as chosen, or paint over their text")
	}
	if editCell.paintAfterFn == nil {
		t.Fatal("the cursor's cell has no border")
	}
	st := editCell.st
	tt.ClickAt(st.x+st.w/2, st.y+st.h/2)
	if !editCell.FocusWithin() {
		t.Fatal("the editor did not take the focus")
	}
	if chosen(editCell) || editCell.paintAfterFn != nil || editCell.checked != 2 {
		t.Fatal("the focused editor's cell shows as chosen or has the border, or is not selected for assistive technology")
	}
	if !chosen(plainCell) {
		t.Fatal("the other cells of the block lost their color")
	}
}

// Whole rows chosen show their row header chosen too; a block of some
// columns leaves it as it is.
func TestRowHeaderShowsWholeRowsChosen(t *testing.T) {
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
	var head *node
	var theme *Theme
	tt := NewTester(func(c *Context) {
		theme = c.Theme()
		Table(c, s, cols, 4, func(row, col int) {
			n := Textf(c, "r%dc%d", row, col).node()
			if row == 2 && col == 0 {
				head = n.parent
			}
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r2c0")
	if head.bg != theme.Accent || head.ts.color != theme.AccentText || head.checked != 2 {
		t.Fatal("the row header of a whole row chosen does not show as chosen")
	}
	tt.Click("r2c1")
	if head.bg == theme.Accent || head.checked == 2 {
		t.Fatal("the row header shows as chosen with one cell of its row")
	}
}

// Left and Right move the column the keys move from, Shift with them
// extends the columns, and the keys that move the rows leave one column.
func TestCellSelectKeys(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.Click("r1c2")
	tt.Key(0, KeyRight)
	wantBlock(t, s, sel, "Right from B", []int{1}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
	tt.Key(0, KeyRight)
	wantBlock(t, s, sel, "Right from the last column", []int{1}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
	tt.Key(Shift, KeyLeft)
	wantBlock(t, s, sel, "Shift+Left", []int{1}, ColumnRange{Cursor: "B", Anchor: "C"}, []string{"B", "C"})
	tt.Key(Shift, KeyDown)
	wantBlock(t, s, sel, "Shift+Down", []int{1, 2}, ColumnRange{Cursor: "B", Anchor: "C"}, []string{"B", "C"})
	tt.Key(0, KeyDown)
	wantBlock(t, s, sel, "Down", []int{3}, ColumnRange{Cursor: "B", Anchor: "B"}, []string{"B"})
	tt.Key(0, KeyLeft)
	tt.Key(0, KeyLeft)
	wantBlock(t, s, sel, "Left twice, past the row headers", []int{3}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
	tt.Key(Cmd, KeyA)
	wantBlock(t, s, sel, "Cmd+A", []int{0, 1, 2, 3, 4, 5, 6, 7}, ColumnRange{Cursor: "A", Anchor: "A", All: true}, []string{"A", "B", "C"})

	// Right to left, Left goes to the column on the left of the screen.
	tt, s, sel = cellTable(t, 8, 3, 0)
	tt.SetDirection(RightToLeft)
	tt.Frame()
	tt.Click("r1c2")
	tt.Key(0, KeyLeft)
	wantBlock(t, s, sel, "Left from B right to left", []int{1}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
}

// A click or a drag on the row headers chooses whole rows, and a click on
// their header every row; that header does not sort.
func TestRowHeaderSelectsRowsAndAll(t *testing.T) {
	tt, s, sel := cellTable(t, 6, 3, 0)
	tt.Click("r2c0")
	wantBlock(t, s, sel, "click the row header of row 2", []int{2}, ColumnRange{Cursor: "A", Anchor: "A", All: true}, []string{"A", "B", "C"})
	tt.ClickWith(Shift, "r4c0")
	wantBlock(t, s, sel, "Shift-click the row header of row 4", []int{2, 3, 4}, ColumnRange{Cursor: "A", Anchor: "A", All: true}, []string{"A", "B", "C"})
	tt.Click("r1c2")
	x, y := rowCenter(t, tt, "r0c0")
	_, y2 := rowCenter(t, tt, "r1c0")
	tt.Press(x, y)
	tt.Move(x, y+3)
	tt.Move(x, y2)
	tt.Release(x, y2)
	wantBlock(t, s, sel, "drag the row headers of rows 0 to 1", []int{0, 1}, ColumnRange{Cursor: "B", Anchor: "B", All: true}, []string{"A", "B", "C"})
	tt.Click("r3c3")
	tt.Click("#")
	wantBlock(t, s, sel, "click the header of the row headers", []int{0, 1, 2, 3, 4, 5}, ColumnRange{Cursor: "C", Anchor: "C", All: true}, []string{"A", "B", "C"})

	// It does not sort, though Sortable.
	var sort SortOrder
	sel2 := &Selection[int]{}
	s2 := &ListState{CellSelect: true, Sort: &sort, Selection: sel2}
	cols := []TableColumn{{Title: "#", RowHeader: true, Sortable: true}, {Title: "A", Sortable: true}}
	tt = NewTester(func(c *Context) {
		Table(c, s2, cols, 3, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("#")
	if sort != (SortOrder{}) || chosenRows(sel2) == nil {
		t.Fatalf("click the header of the row headers, Sortable: sort %+v, rows %v", sort, chosenRows(sel2))
	}
}

// A right-click on a cell outside the block chooses that cell; one inside
// keeps the block.
func TestCellSelectRightClick(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.Click("r1c1")
	if err := tt.RightClick("r4c2"); err != nil {
		t.Fatal(err)
	}
	wantBlock(t, s, sel, "right-click r4c2", []int{4}, ColumnRange{Cursor: "B", Anchor: "B"}, []string{"B"})
	tt.CloseMenu()
	tt.Click("r1c1")
	tt.ClickWith(Shift, "r3c2")
	if err := tt.RightClick("r2c1"); err != nil {
		t.Fatal(err)
	}
	wantBlock(t, s, sel, "right-click r2c1 in the block", []int{1, 2, 3}, ColumnRange{Cursor: "B", Anchor: "A"}, []string{"A", "B"})
	tt.CloseMenu()
	if err := tt.RightClick("r2c3"); err != nil {
		t.Fatal(err)
	}
	wantBlock(t, s, sel, "right-click r2c3, a chosen row out of the block's columns", []int{2}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
	tt.CloseMenu()

	s.MenuKeepsSelection = true
	if err := tt.RightClick("r5c1"); err != nil {
		t.Fatal(err)
	}
	wantBlock(t, s, sel, "right-click r5c1, MenuKeepsSelection", []int{2}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
	tt.CloseMenu()
}

// The cells of the block tell assistive technology they are chosen, and
// show it: the rows are not tinted whole.
func TestCellSelectAccessibility(t *testing.T) {
	tt, s, _ := cellTable(t, 8, 3, 0)
	tt.Click("r1c1")
	tt.ClickWith(Shift, "r2c2")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	cell := func(text string) platform.AccessNode {
		t.Helper()
		for _, n := range tree.Nodes {
			if n.Label == text && n.Parent >= 0 && tree.Nodes[n.Parent].Role == platform.RoleCell {
				return tree.Nodes[n.Parent]
			}
		}
		t.Fatalf("no cell holding %q", text)
		return platform.AccessNode{}
	}
	for _, text := range []string{"r1c1", "r1c2", "r2c1", "r2c2"} {
		if cell(text).States&platform.AccessChecked == 0 {
			t.Errorf("cell %s in the block is not chosen", text)
		}
	}
	for _, text := range []string{"r1c3", "r0c1", "r3c2"} {
		if cell(text).States&platform.AccessChecked != 0 {
			t.Errorf("cell %s out of the block is chosen", text)
		}
	}
	accent := LightTheme().Accent
	for _, r := range s.frame.rows {
		if r.i == 1 && r.e.bg == accent {
			t.Errorf("chosen row 1 is tinted whole")
		}
	}
}

// Dragging cells past the table's right edge scrolls it sideways and goes
// on choosing columns.
func TestCellSelectDragScrollsSideways(t *testing.T) {
	tt, s, _ := cellTable(t, 8, 10, 100)
	head, _ := tt.Find("A")
	x, y := rowCenter(t, tt, "r1c1")
	tt.Press(x, y)
	tt.Move(x+5, y)
	tt.Move(399, y)
	for range 30 {
		tt.Frame()
	}
	cols := s.SelectedColumns()
	moved, _ := tt.Find("A")
	if s.Cells.Anchor != "A" || len(cols) < 6 || moved.X >= head.X {
		t.Fatalf("held at the right edge: cells %+v, columns %v; the header of A at %v, was %v", s.Cells, cols, moved.X, head.X)
	}
	tt.Release(399, y)

	// Right shows the column it moves to.
	tt, s, _ = cellTable(t, 8, 10, 100)
	tt.Click("r1c1")
	for range 8 {
		tt.Key(0, KeyRight)
	}
	if r, ok := tt.Find(s.Cells.Cursor); s.Cells.Cursor != "I" || !ok || r.X < 0 || r.X+r.W > 400 {
		t.Fatalf("Right 8 times from A: cursor %q, its header at %+v", s.Cells.Cursor, r)
	}

	// Right to left, Left shows the column it moves to.
	tt, s, _ = cellTable(t, 8, 10, 100)
	tt.SetDirection(RightToLeft)
	tt.Frame()
	tt.Click("r1c1")
	for range 8 {
		tt.Key(0, KeyLeft)
	}
	h, list := tt.rt.states[s.tableCols[s.colIndex(s.Cells.Cursor)].head], s.frame.e.st
	if s.Cells.Cursor != "I" || h.x < list.x || h.x+h.w > list.x+list.w {
		t.Fatalf("Left 8 times from A right to left: cursor %q, its header at %v..%v", s.Cells.Cursor, h.x, h.x+h.w)
	}
}

// Left and Right leave one cell, as in a spreadsheet: the cursor's row
// alone; with Shift the rows stay.
func TestCellSelectLeftRightCollapseRows(t *testing.T) {
	tt, s, sel := cellTable(t, 6, 3, 0)
	tt.Click("r1c1")
	tt.ClickWith(Shift, "r3c2")
	tt.Key(Shift, KeyLeft)
	wantBlock(t, s, sel, "Shift+Left", []int{1, 2, 3}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
	tt.ClickWith(Shift, "r3c2")
	tt.Key(0, KeyRight)
	wantBlock(t, s, sel, "Right", []int{3}, ColumnRange{Cursor: "C", Anchor: "C"}, []string{"C"})
}

// CellSelected is false before the first build and out of range; in a
// table not choosing cells, every cell of a chosen row is chosen, and in a
// table with no selection none is. TestCellSelectedMatchesDrawing covers
// the block.
func TestCellSelectedQuery(t *testing.T) {
	var fresh ListState
	if fresh.CellSelected(0, 0) {
		t.Fatal("a list never built reports a cell chosen")
	}
	tt, s, _ := cellTable(t, 8, 3, 0)
	tt.Click("r2c0")
	for _, rc := range [][2]int{{-1, 1}, {8, 1}, {2, 9}, {2, -1}} {
		if s.CellSelected(rc[0], rc[1]) {
			t.Errorf("row 2 chosen whole: CellSelected(%d, %d) out of range is true", rc[0], rc[1])
		}
	}

	sel := -1
	single := &ListState{Selected: &sel}
	tt = NewTester(func(c *Context) {
		Table(c, single, []TableColumn{{Title: "A"}, {Title: "B"}}, 4, func(row, col int) { Textf(c, "s%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("s2c0")
	if !single.CellSelected(2, 0) || !single.CellSelected(2, 1) || single.CellSelected(1, 0) {
		t.Error("a table not choosing cells: the cells of the chosen row are not all chosen, or another is")
	}

	none := &ListState{}
	tt = NewTester(func(c *Context) {
		Table(c, none, []TableColumn{{Title: "A"}}, 4, func(row, col int) { Textf(c, "n%d", row) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	if none.CellSelected(0, 0) {
		t.Error("a table with no selection reports a cell chosen")
	}
}

// CellSelected agrees with what the table draws, for every cell built.
func TestCellSelectedMatchesDrawing(t *testing.T) {
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}, {Title: "C"}}
	cells := map[[2]int]*node{}
	var theme *Theme
	tt := NewTester(func(c *Context) {
		theme = c.Theme()
		clear(cells)
		Table(c, s, cols, 6, func(row, col int) {
			cells[[2]int{row, col}] = Textf(c, "r%dc%d", row, col).node().parent
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	check := func(what string) {
		t.Helper()
		for rc, n := range cells {
			if drawn, got := n.bg == theme.Accent, s.CellSelected(rc[0], rc[1]); drawn != got {
				t.Errorf("%s: cell (%d, %d) drawn chosen %v, CellSelected %v", what, rc[0], rc[1], drawn, got)
			}
		}
	}
	tt.Click("r1c1")
	tt.ClickWith(Shift, "r3c2")
	check("block r1c1..r3c2")
	tt.Click("r4c0")
	check("row header of row 4")
}

// A click without a press, as assistive technology or Enter makes, chooses
// the row and leaves the columns of the block as they are.
func TestCellSelectClickWithoutPressKeepsColumns(t *testing.T) {
	tt, s, sel := cellTable(t, 8, 3, 0)
	tt.Click("r1c2")
	var row4 uint64
	for id, r := range s.rows {
		if r.row == 4 {
			row4 = id
		}
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: row4, Action: platform.AccessFocus})
	wantBlock(t, s, sel, "assistive technology focusing row 4", []int{4}, ColumnRange{Cursor: "B", Anchor: "B"}, []string{"B"})
}

// A click chooses the cell under the pointer as the table shows it, moved
// by a transform of an element around it.
func TestCellSelectUnderTransform(t *testing.T) {
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{Title: "A", Width: 100, Fixed: true}, {Title: "B", Width: 100, Fixed: true}, {Title: "C", Width: 100, Fixed: true}}
	tt := NewTester(func(c *Context) {
		Box(c).Grow(1).Translate(-60, 0).Children(func() {
			Table(c, s, cols, 6, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
		})
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c1")
	wantBlock(t, s, sel, "click r1c1 under Translate(-60, 0)", []int{1}, ColumnRange{Cursor: "B", Anchor: "B"}, []string{"B"})
}

// A drag over the row headers chooses whole rows though their column has
// no title, which makes its ID "".
func TestRowHeaderDragUntitled(t *testing.T) {
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	cols := []TableColumn{{RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 6, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	x, y := rowCenter(t, tt, "r1c0")
	_, y2 := rowCenter(t, tt, "r3c0")
	tt.Press(x, y)
	tt.Move(x, y+3)
	tt.Move(x, y2)
	tt.Release(x, y2)
	wantBlock(t, s, sel, "drag the untitled row headers of rows 1 to 3", []int{1, 2, 3}, ColumnRange{Cursor: "A", Anchor: "A", All: true}, []string{"A", "B"})
}

// In an OutlineTable choosing cells, Right and Left still open and close
// the row chosen, and leave the columns as they are.
func TestOutlineTableCellSelectKeys(t *testing.T) {
	var s OutlineState[string]
	s.List.Selection, s.List.CellSelect = &Selection[string]{}, true
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}}
	tt := NewTester(func(c *Context) {
		OutlineTable(c, &s, cols, []string{"src", "docs", "README.md"}, children, func(path string, col int) {
			Textf(c, "%s/%d", baseName(path), col).SingleLine()
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("src/0")
	want := ColumnRange{Cursor: "Name", Anchor: "Name"}
	tt.Key(0, KeyRight)
	if !s.Open.Has("src") || s.List.Cells != want {
		t.Fatalf("Right: open %v, cells %+v", s.Open.Has("src"), s.List.Cells)
	}
	tt.Key(0, KeyLeft)
	if s.Open.Has("src") || s.List.Cells != want {
		t.Fatalf("Left: open %v, cells %+v", s.Open.Has("src"), s.List.Cells)
	}
}

// CellsStyle paints the block's cells, the rest of the rows chosen, and
// every other row in colors of its own: the spreadsheet's two levels and
// stripes.
func TestCellsStyleTwoLevelAndStripes(t *testing.T) {
	s := &ListState{CellSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	var theme *Theme
	var style CellsStyle
	rowNode := map[int]*node{}
	cellNode := map[[2]int]*node{}
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
	tt := NewTester(func(c *Context) {
		theme = c.Theme()
		style = CellsStyle{
			Cell:     theme.Accent,
			CellText: theme.AccentText,
			Row:      theme.Text.Alpha(0.12),
			Stripe:   theme.Text.Alpha(0.04),
		}
		s.CellsStyle = style
		Table(c, s, cols, 4, func(row, col int) {
			cell := Textf(c, "r%dc%d", row, col).node().parent
			cellNode[[2]int{row, col}] = cell
			rowNode[row] = cell.parent.parent // the list's row, under the table's
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c1")
	tt.Frame()
	wantBlock(t, s, sel, "click r1c1", []int{1}, ColumnRange{Cursor: "A", Anchor: "A"}, []string{"A"})
	if cell := cellNode[[2]int{1, 1}]; cell.bg != style.Cell || cell.ts.color != style.CellText {
		t.Fatalf("chosen cell: bg %v text %v", cell.bg, cell.ts.color)
	}
	if other := cellNode[[2]int{1, 2}]; other.bg != Transparent {
		t.Fatalf("an unchosen cell of the row is painted over the row: %v", other.bg)
	}
	if row := rowNode[1]; row.bg != style.Row {
		t.Fatalf("chosen row: %v want %v", row.bg, style.Row)
	}
	if row := rowNode[0]; row.bg != Transparent {
		t.Fatalf("even row painted: %v", row.bg)
	}
	if row := rowNode[3]; row.bg != style.Stripe {
		t.Fatalf("striped row: %v want %v", row.bg, style.Stripe)
	}
}

// cellNodeTable shows a table choosing cells (CellSelect) whose block is
// painted with style, of 4 rows: a column of row headers, "#", then A
// and B. It returns the Tester, the state, the theme, the cells by row
// and column, column 0 being the row headers, and the list's rows.
func cellNodeTable(t *testing.T, style CellsStyle) (*Tester, *ListState, *Theme, map[[2]int]*node, map[int]*node) {
	t.Helper()
	s := &ListState{CellSelect: true, CellsStyle: style}
	s.Selection = &Selection[int]{}
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40, Fixed: true}, {Title: "A"}, {Title: "B"}}
	cells := map[[2]int]*node{}
	rows := map[int]*node{}
	var theme *Theme
	tt := NewTester(func(c *Context) {
		theme = c.Theme()
		Table(c, s, cols, 4, func(row, col int) {
			cell := Textf(c, "r%dc%d", row, col).node().parent
			cells[[2]int{row, col}] = cell
			rows[row] = cell.parent.parent // the list's row, under the table's
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	return tt, s, theme, cells, rows
}

// borderIs reports whether the top edge of n holds a pixel of color c:
// what DrawOver(cursorBorder) paints along it.
func borderIs(tt *Tester, n *node, c Color) bool {
	want := color.RGBA{c.R, c.G, c.B, c.A}
	st, img := n.st, tt.Image()
	y := int(st.y)
	for x := int(st.x) + 2; x < int(st.x+st.w)-2; x++ {
		for _, py := range []int{y, y + 1} {
			if py >= 0 && py < img.Rect.Dy() && x >= 0 && x < img.Rect.Dx() && img.RGBAAt(x, py) == want {
				return true
			}
		}
	}
	return false
}

// The zero CellsStyle draws the block as the table does without one: the
// accent under its text, the cursor's border in that text in the block
// and in the accent alone out of it, and no row painted.
func TestCellsStyleZeroDrawsAsTheAccent(t *testing.T) {
	tt, s, theme, cells, rows := cellNodeTable(t, CellsStyle{})
	tt.Click("r1c2")
	tt.Key(Shift, KeyLeft)
	if got, want := s.Cells, (ColumnRange{Cursor: "A", Anchor: "B"}); got != want {
		t.Fatalf("cells %+v, want %+v", got, want)
	}
	for _, col := range []int{1, 2} {
		if cell := cells[[2]int{1, col}]; cell.bg != theme.Accent || cell.ts.color != theme.AccentText {
			t.Fatalf("cell r1c%d: bg %v text %v; want the accent and its text", col, cell.bg, cell.ts.color)
		}
	}
	if !borderIs(tt, cells[[2]int{1, 1}], theme.AccentText) {
		t.Error("the cursor's border in the block is not the accent's text color")
	}
	for _, row := range []int{0, 1, 3} {
		if r := rows[row]; r.bg != Transparent {
			t.Errorf("row %d is painted %v without a CellsStyle", row, r.bg)
		}
	}
	// The cursor out of the block, in a row a Cmd-click took out of the
	// choice: its border is the accent alone.
	tt.ClickWith(Cmd, "r1c1")
	if s.CellSelected(1, 1) {
		t.Fatal("Cmd-clicking the cursor's cell kept its row chosen")
	}
	if !borderIs(tt, cells[[2]int{1, 1}], theme.Accent) {
		t.Error("the cursor's border out of the block is not the accent")
	}
}

// A CellsStyle's cursor border takes the block's text color in the block
// and the block's own out of it, or the accent and its text when they are
// zero.
func TestCellsStyleCursorBorder(t *testing.T) {
	green, red := RGB(0, 160, 0), RGB(200, 0, 0)
	cases := []struct {
		name  string
		style CellsStyle
	}{
		{"zero", CellsStyle{}},
		{"Cell", CellsStyle{Cell: green}},
		{"CellText", CellsStyle{CellText: red}},
		{"Cell and CellText", CellsStyle{Cell: green, CellText: red}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tt, _, theme, cells, _ := cellNodeTable(t, tc.style)
			or := func(def, c Color) Color {
				if c == Transparent {
					return def
				}
				return c
			}
			bg, text := or(theme.Accent, tc.style.Cell), or(theme.AccentText, tc.style.CellText)
			tt.Click("r1c2")
			tt.Key(Shift, KeyLeft)
			n := cells[[2]int{1, 1}]
			if n.bg != bg || n.ts.color != text {
				t.Fatalf("the cursor's cell in the block: bg %v text %v; want %v, %v", n.bg, n.ts.color, bg, text)
			}
			if !borderIs(tt, n, text) {
				t.Errorf("the cursor's border in the block is not the text color %v", text)
			}
			tt.ClickWith(Cmd, "r1c1")
			if !borderIs(tt, n, or(theme.Accent, tc.style.Cell)) {
				t.Errorf("the cursor's border out of the block is not %v", or(theme.Accent, tc.style.Cell))
			}
		})
	}
}

// A CellsStyle paints only what its colors name: Row the chosen rows,
// Stripe every other, and Cell the block, which shows in the accent when
// Cell is zero.
func TestCellsStylePartial(t *testing.T) {
	green, blue := RGB(0, 160, 0), RGB(0, 0, 200)

	tt, _, theme, cells, rows := cellNodeTable(t, CellsStyle{Row: green})
	tt.Click("r1c1")
	if n := cells[[2]int{1, 1}]; n.bg != theme.Accent || n.ts.color != theme.AccentText {
		t.Errorf("Row alone: the block's cell bg %v text %v; want the accent and its text", n.bg, n.ts.color)
	}
	if rows[1].bg != green || rows[0].bg != Transparent || rows[3].bg != Transparent {
		t.Errorf("Row alone: rows 1, 0 and 3 are %v, %v, %v; want %v, none, none", rows[1].bg, rows[0].bg, rows[3].bg, green)
	}

	tt, _, _, _, rows = cellNodeTable(t, CellsStyle{Stripe: blue})
	tt.Click("r1c1")
	if rows[1].bg != Transparent || rows[3].bg != blue || rows[0].bg != Transparent {
		t.Errorf("Stripe alone: rows 1, 3 and 0 are %v, %v, %v; want none, %v, none", rows[1].bg, rows[3].bg, rows[0].bg, blue)
	}

	tt, _, theme, cells, rows = cellNodeTable(t, CellsStyle{Cell: green})
	tt.Click("r1c1")
	if n := cells[[2]int{1, 1}]; n.bg != green || n.ts.color != theme.AccentText {
		t.Errorf("Cell alone: the block's cell bg %v text %v; want %v and the accent's text", n.bg, n.ts.color, green)
	}
	if rows[1].bg != Transparent || rows[3].bg != Transparent {
		t.Errorf("Cell alone: rows 1 and 3 are %v and %v; want none", rows[1].bg, rows[3].bg)
	}
}
