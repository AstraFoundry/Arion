package ui

import (
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// tableView shows a table of files: name, kind and size.
func tableView(s *ListState, cols []TableColumn, names []string) func(c *context) {
	return func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, s, cols, len(names), func(row, col int) {
				switch col {
				case 0:
					coreText(c, names[row]).SingleLine()
				case 1:
					coreText(c, "Document")
				case 2:
					coreTextf(c, "%d KB", row+1)
				}
			}).Grow(1)
		})
	}
}

func TestTableSort(t *testing.T) {
	sort := SortOrder{Column: "Size"}
	sel := -1
	s := ListState{Sort: &sort, Selected: &sel}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End, Sortable: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "b.txt"}), 500, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleColumnHeader, "Size"); n.States&platform.AccessSortAscending == 0 {
		t.Errorf("the column sorted by: %+v", n)
	}
	tt.Click("Name")
	if sort != (SortOrder{Column: "Name"}) {
		t.Fatalf("a click on Name: %+v", sort)
	}
	tt.Click("Name")
	if sort != (SortOrder{Column: "Name", Descending: true}) {
		t.Fatalf("a second click on Name: %+v", sort)
	}
	tree := tt.h.access
	if n := accessNode(t, tree, platform.RoleColumnHeader, "Name"); n.States&platform.AccessSortDescending == 0 || n.Actions&platform.ActionPress == 0 {
		t.Errorf("Name, sorted descending: %+v", n)
	}
	if n := accessNode(t, tree, platform.RoleColumnHeader, "Size"); n.States&(platform.AccessSortAscending|platform.AccessSortDescending) != 0 {
		t.Errorf("Size, no longer sorted by: %+v", n)
	}
	// A column that does not sort.
	tt.Click("Kind")
	if sort != (SortOrder{Column: "Name", Descending: true}) || sel != -1 {
		t.Errorf("a click on Kind: %+v, chose %d", sort, sel)
	}
}

// header returns the box of a column's header.
func header(t *testing.T, tt *Tester, title string) platform.RectF {
	t.Helper()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	return accessNode(t, tt.h.access, platform.RoleColumnHeader, title).Bounds
}

func TestTableResize(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100, MinWidth: 60}, {Title: "Size", Width: 80, Fixed: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "a much longer name.txt"}), 500, 300)
	kind := header(t, tt, "Kind")
	// Dragging the right edge of Kind's header.
	edge := float32(kind.X+kind.W) - 2
	tt.Press(edge, float32(kind.Y+kind.H/2))
	tt.Move(edge+30, float32(kind.Y+kind.H/2))
	tt.Release(edge+30, float32(kind.Y+kind.H/2))
	if w, h := s.Columns.Widths["Kind"], header(t, tt, "Kind"); w != 130 || h.W != 130 || float32(h.X+h.W)-2 != edge+30 {
		t.Fatalf("dragged 30 DIPs wider: %v, the header %v", w, h)
	}
	// The column before it, sharing the room left, kept its width, for the
	// edge to follow the pointer.
	if w := s.Columns.Widths["Name"]; w != 300 {
		t.Errorf("Name took %v", w)
	}
	edge += 30
	// No narrower than its least.
	tt.Press(edge, float32(kind.Y+kind.H/2))
	tt.Move(edge-130, float32(kind.Y+kind.H/2))
	tt.Release(edge-130, float32(kind.Y+kind.H/2))
	if w := header(t, tt, "Kind").W; w != 60 {
		t.Errorf("dragged narrower than its least: %v", w)
	}
	// A double click fits it to its cells, and the cells follow.
	kind = header(t, tt, "Kind")
	edge = float32(kind.X+kind.W) - 2
	tt.ClickAt(edge, float32(kind.Y+kind.H/2))
	tt.ClickAt(edge, float32(kind.Y+kind.H/2))
	doc, _ := tt.Find("Document")
	if w := float32(header(t, tt, "Kind").W); w < doc.W+20 || w > doc.W+21 || w < 70 {
		t.Errorf("fitted to %v for a text of %v", w, doc.W)
	}
	// A fixed column has no edge to drag.
	size := header(t, tt, "Size")
	tt.Press(float32(size.X+size.W-1), float32(size.Y+size.H/2))
	tt.Move(float32(size.X+size.W+40), float32(size.Y+size.H/2))
	tt.Release(float32(size.X+size.W+40), float32(size.Y+size.H/2))
	if w := header(t, tt, "Size").W; w != 80 {
		t.Errorf("a fixed column resized to %v", w)
	}
}

func TestTableReorder(t *testing.T) {
	sort := SortOrder{}
	s := ListState{Sort: &sort}
	cols := []TableColumn{{Title: "Name", Fixed: true}, {Title: "Kind", Width: 100, Sortable: true}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	kind := header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)
	// Kind dragged past the middle of Size.
	x := float32(kind.X + 20)
	tt.Press(x, y)
	for _, d := range []float32{5, 20, 50, 60} {
		tt.Move(x+d, y)
	}
	tt.Release(x+60, y)
	if got := s.Columns.Order; len(got) != 3 || got[1] != "Size" || got[2] != "Kind" {
		t.Fatalf("the order: %q", got)
	}
	if sort.Column != "" {
		t.Errorf("dragging the header sorted by %q", sort.Column)
	}
	kind, size := header(t, tt, "Kind"), header(t, tt, "Size")
	doc, _ := tt.Find("Document")
	if !(size.X < kind.X) || float64(doc.X) < kind.X {
		t.Errorf("Size at %v, Kind at %v, its cell at %v", size.X, kind.X, doc.X)
	}
	// Name is fixed: it neither moves nor lets others past it.
	tt.Press(float32(size.X+10), y)
	for _, d := range []float32{-5, -40, -100, -200} {
		tt.Move(float32(size.X+10)+d, y)
	}
	tt.Release(float32(size.X+10)-200, y)
	if got := s.Columns.Order; got[0] != "Name" {
		t.Errorf("moved past a fixed column: %q", got)
	}
	// The release's own move moves the column too, and is no click.
	x = float32(header(t, tt, "Kind").X + 20)
	tt.Press(x, y)
	tt.Move(x-5, y)
	tt.Move(x-20, y)
	tt.Release(x-60, y)
	if got := s.Columns.Order; got[1] != "Kind" || got[2] != "Size" {
		t.Errorf("Kind let go past the middle of Size: %q", got)
	}
	if sort.Column != "" {
		t.Errorf("dragging the header sorted by %q", sort.Column)
	}
}

// A column dragged past several others stays the one dragged: its header
// keeps the press as the others move aside.
func TestTableReorderFar(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 80}, {Title: "Size", Width: 80}, {Title: "Date", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 600, 300)
	kind := header(t, tt, "Kind")
	y := float32(kind.Y + kind.H/2)
	x := float32(kind.X + 20)
	tt.Press(x, y)
	for d := float32(5); d <= 180; d += 15 {
		tt.Move(x+d, y)
	}
	tt.Release(x+180, y)
	if got := s.Columns.Order; len(got) != 4 || got[1] != "Size" || got[2] != "Date" || got[3] != "Kind" {
		t.Fatalf("the order after dragging Kind past Size and Date: %q", got)
	}
}

// The last column sharing the room left ends at the table's end: its
// header has no edge, as AppKit's. A last column with a width keeps its.
func TestTableLastColumnEdge(t *testing.T) {
	// edge reports whether the end of a column's header resizes it, its
	// pointer moving there and dragging it 30 DIPs out.
	edge := func(tt *Tester, s *ListState, title string, rtl bool) bool {
		h := header(t, tt, title)
		x, y, dx := float32(h.X+h.W)-2, float32(h.Y+h.H/2), float32(-30)
		if rtl {
			x, dx = float32(h.X)+2, 30
		}
		tt.Move(x, y)
		cursor := tt.Cursor() == CursorResizeEW
		before := header(t, tt, title).W
		tt.Press(x, y)
		tt.Move(x+dx, y)
		tt.Release(x+dx, y)
		after := header(t, tt, title).W
		if cursor != (after != before) {
			t.Errorf("%s: the resize pointer %v, but dragged from %v to %v", title, cursor, before, after)
		}
		return cursor
	}
	for _, rtl := range []bool{false, true} {
		s := ListState{}
		cols := []TableColumn{{Title: "Name", Width: 150}, {Title: "Kind", Width: 100}, {Title: "Size"}}
		tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
		if rtl {
			tt.SetDirection(RightToLeft)
		}
		if edge(tt, &s, "Size", rtl) {
			t.Errorf("rtl %v: the last column, sharing the room left, has an edge", rtl)
		}
		if _, ok := s.Columns.Widths["Size"]; ok {
			t.Errorf("rtl %v: the last column took a width: %v", rtl, s.Columns.Widths)
		}
		if !edge(tt, &s, "Kind", rtl) {
			t.Errorf("rtl %v: a column with a width before it has no edge", rtl)
		}
		// Moved before the others, it has its edge back, and the last one,
		// with a width, keeps its own.
		s.Columns.Order = []string{"Size", "Name", "Kind"}
		tt.Frame()
		if !edge(tt, &s, "Size", rtl) || !edge(tt, &s, "Kind", rtl) {
			t.Errorf("rtl %v: moved first, Size or the last column, Kind, has no edge", rtl)
		}
	}
}

func TestTableScrollsSideways(t *testing.T) {
	s := ListState{}
	cols := []TableColumn{{Title: "Name", Width: 200}, {Title: "Kind", Width: 200}, {Title: "Size", Width: 200}}
	names := make([]string, 50)
	for i := range names {
		names[i] = "file"
	}
	tt := coreNewTester(tableView(&s, cols, names), 400, 300)
	if tt.HasText("Size") {
		size := header(t, tt, "Size")
		if size.X < 390 {
			t.Fatalf("the third column of 200 shows at %v in a table 380 wide", size.X)
		}
	}
	tt.Scroll(200, 150, 300, 0)
	size := header(t, tt, "Size")
	cell, _ := tt.Find("1 KB")
	if size.X > 300 || float64(cell.X) != size.X+10 {
		t.Errorf("scrolled sideways: the header at %v, the text of its cell at %v", size.X, cell.X)
	}
	// Down still moves the rows, and the header stays.
	before := size.Y
	tt.Scroll(200, 150, 0, 300)
	if size := header(t, tt, "Size"); size.Y != before || s.first == 0 {
		t.Errorf("scrolled down: the header at %v (was %v), first row %d", size.Y, before, s.first)
	}
}

func TestEditableTextInTable(t *testing.T) {
	names := []string{"notes.txt", "photo.jpeg", "README"}
	sel, opened, renames := -1, -1, 0
	s := ListState{Selected: &sel}
	cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80}}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			if coreTable(c, &s, cols, len(names), func(row, col int) {
				if col == 0 {
					if coreEditableText(c, &names[row]).Changed() {
						renames++
					}
				} else {
					coreText(c, "1 KB")
				}
			}).Grow(1).Submitted() {
				opened = sel
			}
		})
	}, 400, 300)
	rename := func() {
		if runtime.GOOS == "darwin" {
			tt.Key(0, KeyEnter)
		} else {
			tt.Key(0, KeyF2)
		}
	}
	tt.Click("photo.jpeg")
	rename()
	if opened != -1 {
		t.Fatalf("the key editing the name opened the row")
	}
	// The name before its extension is selected: typing replaces it.
	tt.Type("beach")
	tt.Key(0, KeyEnter)
	if names[1] != "beach.jpeg" || renames != 1 {
		t.Fatalf("renamed to %q (%d)", names[1], renames)
	}
	if !tt.Focused("beach.jpeg") {
		t.Error("the table did not take the focus back")
	}
	// Escape goes back.
	tt.Key(0, KeyF2)
	tt.Type("gone")
	tt.Key(0, KeyEscape)
	if names[1] != "beach.jpeg" || !tt.HasText("beach.jpeg") {
		t.Errorf("Escape left %q", names[1])
	}
	// A click on the name of the row chosen edits it, once a double click
	// is out of the question; the focus leaving keeps what was typed.
	tt.Click("beach.jpeg")
	if tt.HasText("beach.jpeg") != true {
		t.Fatal("the click edited at once")
	}
	tt.Advance(renameDelay + 50*time.Millisecond)
	tt.Type("sea")
	tt.Click("notes.txt")
	if names[1] != "sea.jpeg" || sel != 0 {
		t.Errorf("after a click elsewhere: %q, chose %d", names[1], sel)
	}
	// A double click opens, and does not edit.
	tt.Click("notes.txt")
	tt.Click("notes.txt")
	tt.Advance(renameDelay + 50*time.Millisecond)
	tt.Type("x")
	if opened != 0 || names[0] != "notes.txt" {
		t.Errorf("a double click: opened %d, the name %q", opened, names[0])
	}
}

func TestEditableText(t *testing.T) {
	title := "Untitled.md"
	changes := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			if coreEditableText(c, &title).Changed() {
				changes++
			}
			coreButton(c, "Other")
		})
	}, 400, 300)
	r, _ := tt.Find("Untitled.md")
	tt.ClickAt(r.X+5, r.Y+5)
	tt.ClickAt(r.X+5, r.Y+5)
	tt.Type("Notes")
	tt.Key(0, KeyTab)
	if title != "Notes.md" || changes != 1 {
		t.Fatalf("after Tab: %q (%d)", title, changes)
	}
	tt.Click("Notes.md")
	tt.Key(0, KeyEnter)
	tt.Key(Cmd, KeyA)
	tt.Type("Plan")
	tt.Key(0, KeyEnter)
	if title != "Plan" || !tt.Focused("Plan") {
		t.Errorf("after Enter: %q, focused %v", title, tt.Focused("Plan"))
	}
}

func TestTableRoundRows(t *testing.T) {
	names := []string{"a.txt", "b.txt", "c.txt"}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80, Align: End}}
	// radius returns the corners of row i's box, or of the box inside it
	// that shows the pointer over it in a table that does not choose.
	radius := func(s *ListState, i int, inner bool) [4]float32 {
		t.Helper()
		for _, r := range s.frame.rows {
			if r.i == i {
				if inner {
					return r.e.first.radius
				}
				return r.e.radius
			}
		}
		t.Fatalf("row %d not built", i)
		return [4]float32{}
	}
	r := LightTheme().Radius
	square, round := [4]float32{}, [4]float32{r, r, r, r}

	sel := 1
	s := ListState{Selected: &sel}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 1, false); got != square {
		t.Errorf("a chosen row by default: %v", got)
	}
	s = ListState{Selected: &sel, RoundRows: true}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 1, false); got != round {
		t.Errorf("a chosen row with RoundRows: %v", got)
	}

	// Rows chosen together make one block.
	var chosen Selection[string]
	chosen.Add("a.txt")
	chosen.Add("b.txt")
	s = ListState{Selection: &chosen, Key: func(i int) any { return names[i] }, RoundRows: true}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 0, false); got != [4]float32{r, r, 0, 0} {
		t.Errorf("the first of two rows chosen: %v", got)
	}
	if got := radius(&s, 1, false); got != [4]float32{0, 0, r, r} {
		t.Errorf("the second of two rows chosen: %v", got)
	}

	// A table that does not choose shows the pointer over its rows.
	s = ListState{}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 0, true); got != square {
		t.Errorf("a row of a table that does not choose, by default: %v", got)
	}
	s = ListState{RoundRows: true}
	coreNewTester(tableView(&s, cols, names), 500, 300)
	if got := radius(&s, 0, true); got != round {
		t.Errorf("a row of a table that does not choose, with RoundRows: %v", got)
	}
}

// With SortBy, a Cmd-click sorts by several columns: it adds a column,
// ascending, after those sorted by, makes it descending, then takes it
// out. A click sorts by one column again.
func TestTableSortBy(t *testing.T) {
	var by []SortOrder
	s := ListState{SortBy: &by}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100, Sortable: true}, {Title: "Size", Width: 80, Sortable: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "b.txt"}), 500, 300)
	check := func(step string, want ...SortOrder) {
		t.Helper()
		if !slices.Equal(by, want) {
			t.Fatalf("%s: %+v, want %+v", step, by, want)
		}
	}
	tt.Click("Name")
	check("a click on Name", SortOrder{Column: "Name"})
	tt.ClickWith(Cmd, "Kind")
	check("a Cmd-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Kind"})
	kept := by
	tt.ClickWith(Cmd, "Size")
	check("a Cmd-click on Size", SortOrder{Column: "Name"}, SortOrder{Column: "Kind"}, SortOrder{Column: "Size"})
	if len(kept) != 2 {
		t.Fatalf("a click changed the slice the app kept: %+v", kept)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleColumnHeader, "Size"); n.States&platform.AccessSortAscending == 0 || n.Label != "Size" {
		t.Errorf("Size, sorted by after Name and Kind: %+v", n)
	}
	tt.ClickWith(Cmd, "Kind")
	check("a second Cmd-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Kind", Descending: true}, SortOrder{Column: "Size"})
	tt.ClickWith(Cmd, "Kind")
	check("a third Cmd-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Size"})
	// The first column goes the same way, the next taking its place.
	tt.ClickWith(Cmd, "Name")
	check("a Cmd-click on Name", SortOrder{Column: "Name", Descending: true}, SortOrder{Column: "Size"})
	tt.ClickWith(Cmd, "Name")
	check("a second Cmd-click on Name", SortOrder{Column: "Size"})
	// A click sorts by one column again, and on the first reverses it.
	tt.ClickWith(Cmd, "Kind")
	tt.Click("Kind")
	check("a click on Kind", SortOrder{Column: "Kind"})
	tt.ClickWith(Cmd, "Name")
	tt.Click("Kind")
	check("a click on Kind, first", SortOrder{Column: "Kind", Descending: true})
}

func TestTableHeaderMenu(t *testing.T) {
	built, chosen := -1, -1
	s := ListState{Columns: TableLayout{Order: []string{"Size", "Name", "Kind"}}}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	// Without HeaderMenu, a header has no menu.
	tt.RightClick("Kind")
	if m := tt.Menu(); m != nil {
		t.Fatalf("a menu without HeaderMenu: %q", m)
	}
	s.HeaderMenu = func(m *Menu, col int) {
		built = col
		if m.Item("Hide Column").Chosen() {
			chosen = col
		}
	}
	tt.Frame()
	// Kind shows third, and is the second of the columns.
	tt.RightClick("Kind")
	if m := tt.Menu(); len(m) != 1 || m[0] != "Hide Column" || built != 1 {
		t.Fatalf("the menu %q, built for column %d", m, built)
	}
	if err := tt.ChooseMenuItem("Hide Column"); err != nil {
		t.Fatal(err)
	}
	if chosen != 1 {
		t.Errorf("chose the item for column %d", chosen)
	}
}

// Set with SortBy, Sort follows its first column, for the app to read
// the column that sorts first; a Shift-click adds a column as a Cmd-click
// does, as web grids do.
func TestTableSortFollowsSortBy(t *testing.T) {
	var by []SortOrder
	var sort SortOrder
	s := ListState{Sort: &sort, SortBy: &by}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind", Width: 100, Sortable: true}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt", "b.txt"}), 500, 300)
	check := func(step string, first SortOrder, want ...SortOrder) {
		t.Helper()
		if !slices.Equal(by, want) || sort != first {
			t.Fatalf("%s: SortBy %+v, Sort %+v; want %+v, %+v", step, by, sort, want, first)
		}
	}
	tt.Click("Name")
	check("a click on Name", SortOrder{Column: "Name"}, SortOrder{Column: "Name"})
	tt.ClickWith(Shift, "Kind")
	check("a Shift-click on Kind", SortOrder{Column: "Name"}, SortOrder{Column: "Name"}, SortOrder{Column: "Kind"})
	tt.ClickWith(Cmd, "Name")
	tt.ClickWith(Shift, "Name")
	check("Name taken out", SortOrder{Column: "Kind"}, SortOrder{Column: "Kind"})
	tt.ClickWith(Shift, "Kind")
	tt.ClickWith(Shift, "Kind")
	check("sorted by none", SortOrder{})
	// The app setting SortBy itself.
	by = []SortOrder{{Column: "Kind", Descending: true}}
	tt.Frame()
	check("SortBy set by the app", SortOrder{Column: "Kind", Descending: true}, SortOrder{Column: "Kind", Descending: true})
}

// The header past the columns has HeaderMenu's menu for column -1, which
// does not open when it has no items.
func TestTableHeaderMenuPastTheColumns(t *testing.T) {
	built := 0
	items := true
	s := ListState{HeaderMenu: func(m *Menu, col int) {
		built = col
		if items {
			m.Item("Reset Columns")
		}
	}}
	cols := []TableColumn{{Title: "Name", Width: 120}, {Title: "Kind", Width: 100}}
	tt := coreNewTester(tableView(&s, cols, []string{"a.txt"}), 500, 300)
	kind, _ := tt.Find("Kind")
	tt.RightClickAt(450, kind.Y+kind.H/2)
	if m := tt.Menu(); len(m) != 1 || m[0] != "Reset Columns" || built != -1 {
		t.Fatalf("past the columns: the menu %q, built for column %d", m, built)
	}
	tt.CloseMenu()
	items = false
	tt.RightClickAt(450, kind.Y+kind.H/2)
	if m := tt.Menu(); m != nil {
		t.Fatalf("a menu without items opened: %q", m)
	}
	items = true
	tt.RightClick("Kind")
	if built != 1 {
		t.Fatalf("a column's header built the menu for column %d", built)
	}
}

// CellMenu gives the cells a context menu over the whole cell, its padding
// too, told the row and the column as the cell builder receives them.
func TestTableCellMenu(t *testing.T) {
	var s ListState
	cols := []TableColumn{{Title: "A", Width: 100, Fixed: true}, {Title: "B", Width: 100, Fixed: true}}
	gotRow, gotCol := -1, -1
	s.CellMenu = func(m *Menu, row, col int) {
		gotRow, gotCol = row, col
		if col == 1 {
			m.Item("Open")
		}
	}
	tt := NewTester(func(c *Context) {
		Table(c, &s, cols, 4, func(row, col int) { Textf(c, "r%dc%d", row, col) }).Grow(1)
	}, 400, 300)
	tt.Frame()
	r, _ := tt.Find("r2c1")
	tt.RightClickAt(r.X+r.W+20, r.Y+r.H/2) // past the text, in the cell
	if m := tt.Menu(); len(m) != 1 || m[0] != "Open" || gotRow != 2 || gotCol != 1 {
		t.Fatalf("right-click in r2c1 past its text: menu %q, row %d, col %d", m, gotRow, gotCol)
	}
	tt.CloseMenu()
	r, _ = tt.Find("r1c0")
	tt.RightClickAt(r.X+2, r.Y+r.H/2)
	if m := tt.Menu(); m != nil || gotRow != 1 || gotCol != 0 {
		t.Fatalf("a menu built without items opened: %q (row %d, col %d)", m, gotRow, gotCol)
	}
}

// HeaderCell gets each column's header cell as the table builds it: what
// it builds goes in the cell after the title, and col is the column's
// index in the columns given to Table.
func TestTableHeaderCell(t *testing.T) {
	hovered := -1
	s := ListState{}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}, {Title: "Size", Width: 80}}
	tt := coreNewTester(func(c *context) {
		s.HeaderCell = func(cell Element, col int) {
			coreTextf(c, "mark %d", col)
			if cell.Hovered() {
				hovered = col
			}
		}
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, &s, cols, 1, func(row, col int) { coreText(c, "a.txt") }).Grow(1)
		})
	}, 500, 300)
	if !tt.HasText("mark 0") || !tt.HasText("mark 2") {
		t.Fatalf("marks: %q", tt.Texts())
	}
	name, _ := tt.Find("Name")
	mark, _ := tt.Find("mark 0")
	if mark.X < name.X+name.W || mark.Y > name.Y+name.H {
		t.Errorf("the mark at %+v, not after the title at %+v", mark, name)
	}
	size, _ := tt.Find("Size")
	tt.Move(size.X+2, size.Y+size.H/2)
	tt.Frame()
	if hovered != 2 {
		t.Fatalf("hovered %d", hovered)
	}
}
