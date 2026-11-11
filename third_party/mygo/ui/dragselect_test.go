package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func dragSelectList(t *testing.T, n int) (*Tester, *ListState, *Selection[int]) {
	t.Helper()
	s := &ListState{DragSelect: true}
	sel := &Selection[int]{}
	s.Selection = sel
	tt := NewTester(func(c *Context) {
		List(c, s, n, func(i int) { Textf(c, "Row %d", i) }).Grow(1)
	}, 300, 300)
	tt.Frame()
	return tt, s, sel
}

func rowCenter(t *testing.T, tt *Tester, text string) (float32, float32) {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q", text)
	}
	return center(r)
}

// Pressing a row and dragging over others chooses the rows between, and
// the release leaves them chosen, though it comes back to the first row.
func TestDragSelectRows(t *testing.T) {
	tt, _, sel := dragSelectList(t, 20)
	x, y := rowCenter(t, tt, "Row 1")
	_, y4 := rowCenter(t, tt, "Row 4")
	tt.Press(x, y)
	tt.Move(x, y+3)
	tt.Move(x, y4)
	if got := chosenRows(sel); !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Fatalf("dragged Row 1 to Row 4: chose %v", got)
	}
	_, y2 := rowCenter(t, tt, "Row 2")
	tt.Move(x, y2)
	tt.Release(x, y2)
	if got := chosenRows(sel); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("back to Row 2 and released: chose %v", got)
	}
	// A drag back onto the row pressed does not end as a click on it.
	tt.Press(x, y)
	tt.Move(x, y4)
	tt.Move(x, y)
	tt.Release(x, y)
	if got := chosenRows(sel); !slices.Equal(got, []int{1}) {
		t.Fatalf("dragged away and back to Row 1: chose %v", got)
	}
	// Cmd adds the rows dragged over to those chosen.
	tt.Click("Row 9")
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y), Mods: platform.Modifiers(Cmd)})
	tt.Move(x, y+3)
	_, y3 := rowCenter(t, tt, "Row 3")
	tt.Move(x, y3)
	tt.Release(x, y3)
	if got := chosenRows(sel); !slices.Equal(got, []int{1, 2, 3, 9}) {
		t.Fatalf("Cmd-drag Row 1 to Row 3 with Row 9 chosen: chose %v", got)
	}
}

// A drag whose moves and release all come between two frames, as when
// frames are slow, chooses the rows up to where the pointer let go, though
// it moved on since, before the keys that came after it and unless a press
// came after it; so does a drag over a table's cells.
func TestDragSelectBetweenFrames(t *testing.T) {
	tt, _, sel := dragSelectList(t, 20)
	x, y := rowCenter(t, tt, "Row 1")
	_, y7 := rowCenter(t, tt, "Row 7")
	_, y9 := rowCenter(t, tt, "Row 9")
	event := func(kind platform.SurfaceEventKind, y float32) {
		tt.rt.event(platform.SurfaceEvent{Kind: kind, X: float64(x), Y: float64(y)})
	}
	drag := func() {
		event(platform.PointerDown, y)
		event(platform.PointerMove, y7)
		event(platform.PointerUp, y7)
	}
	drag()
	event(platform.PointerMove, y9)
	tt.Frame()
	if got := chosenRows(sel); !slices.Equal(got, []int{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("dragged Row 1 to Row 7 and released within a frame: chose %v", got)
	}
	drag()
	tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(KeyDown)})
	tt.Frame()
	if got := chosenRows(sel); !slices.Equal(got, []int{8}) {
		t.Fatalf("then dragged Row 1 to Row 7 and pressed Down within a frame: chose %v", got)
	}
	drag()
	event(platform.PointerDown, y9)
	event(platform.PointerUp, y9)
	tt.Frame()
	if got := chosenRows(sel); !slices.Equal(got, []int{9}) {
		t.Fatalf("then dragged Row 1 to Row 7 and clicked Row 9 within a frame: chose %v", got)
	}

	tt2, s, sel2 := cellTable(t, 8, 3, 0)
	cx, cy := rowCenter(t, tt2, "r1c1")
	cx2, cy2 := rowCenter(t, tt2, "r4c2")
	cx3, _ := rowCenter(t, tt2, "r4c3")
	tt2.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(cx), Y: float64(cy)})
	tt2.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(cx2), Y: float64(cy2)})
	tt2.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(cx2), Y: float64(cy2)})
	tt2.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(cx3), Y: float64(cy2)})
	tt2.Frame()
	wantBlock(t, s, sel2, "dragged r1c1 to r4c2 and released within a frame", []int{1, 2, 3, 4}, ColumnRange{Cursor: "B", Anchor: "A"}, []string{"A", "B"})
}

// A drag adding to the choice (Cmd) chooses the rows from the one pressed
// to the one under the pointer besides those chosen before, on every
// platform: going back takes out the rows it leaves.
func TestDragSelectAddingShrinksBack(t *testing.T) {
	tt, _, sel := dragSelectList(t, 20)
	tt.Click("Row 9")
	x, y := rowCenter(t, tt, "Row 1")
	_, y5 := rowCenter(t, tt, "Row 5")
	_, y2 := rowCenter(t, tt, "Row 2")
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y), Mods: platform.Modifiers(Cmd)})
	tt.Move(x, y+3)
	tt.Move(x, y5)
	if got := chosenRows(sel); !slices.Equal(got, []int{1, 2, 3, 4, 5, 9}) {
		t.Fatalf("Cmd-drag Row 1 to Row 5 with Row 9 chosen: chose %v", got)
	}
	tt.Move(x, y2)
	tt.Release(x, y2)
	if got := chosenRows(sel); !slices.Equal(got, []int{1, 2, 9}) {
		t.Fatalf("Cmd-drag to Row 5 and back to Row 2: chose %v, want [1 2 9]", got)
	}
}

// Held below the list, the drag scrolls it and goes on choosing, to the
// end, though the row pressed scrolls out of view.
func TestDragSelectScrollsAtEdge(t *testing.T) {
	tt, s, sel := dragSelectList(t, 200)
	x, y := rowCenter(t, tt, "Row 1")
	tt.Press(x, y)
	tt.Move(x, y+3)
	tt.Move(x, 299)
	for range 400 {
		tt.Frame()
	}
	got := chosenRows(sel)
	if !s.dragSel.on || len(got) != 199 || got[0] != 1 || got[len(got)-1] != 199 {
		t.Fatalf("held at the bottom edge: drag on %v, chose %d rows (%v…%v)", s.dragSel.on, len(got), got[:min(2, len(got))], got[max(0, len(got)-2):])
	}
	tt.Release(x, 299)
}

// In a list shorter than its two edges, a drag held below its middle
// scrolls it down and one held above scrolls it up.
func TestDragSelectShortListScrollsTowardThePointer(t *testing.T) {
	s := &ListState{DragSelect: true}
	s.Selection = &Selection[int]{}
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			List(c, s, 30, func(i int) { Textf(c, "Row %d", i) }).Height(50)
		})
	}, 300, 300)
	tt.Frame()
	st := s.frame.e.st
	for _, held := range []struct {
		y    float32
		down bool
	}{{30, true}, {20, false}} {
		scrollBy(st, 0, 100-float32(st.scrollY))
		tt.Frame()
		x, from := st.x+20, st.scrollY
		tt.Press(x, st.y+25)
		tt.Move(x, st.y+25+4)
		tt.Move(x, st.y+held.y)
		tt.Frame()
		if moved := st.scrollY - from; held.down && moved <= 0 || !held.down && moved >= 0 {
			t.Errorf("held %v down a list 50 tall: scrolled by %v", held.y, moved)
		}
		tt.Release(x, st.y+held.y)
	}
}

func TestDragSelectWithReorderPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("DragSelect with Reorder did not panic")
		}
	}()
	s := &ListState{DragSelect: true, Reorder: func([]int, int) {}}
	s.Selection = &Selection[int]{}
	NewTester(func(c *Context) { List(c, s, 3, func(i int) { Textf(c, "Row %d", i) }) }, 100, 100).Frame()
}

// In a list with DragSelect, and a table choosing cells, a press on a row
// or a cell not chosen chooses it as the pointer goes down, as ImGui and
// spreadsheets do; a press on one chosen waits for the release, which
// keeps a drag of the choice. Other lists choose on the release.
func TestPressChoosesInDragAndCellLists(t *testing.T) {
	tt, _, sel := dragSelectList(t, 20)
	x, y := rowCenter(t, tt, "Row 3")
	tt.Press(x, y)
	if got := chosenRows(sel); !slices.Equal(got, []int{3}) {
		t.Fatalf("DragSelect: press on Row 3 chose %v before the release", got)
	}
	tt.Release(x, y)
	if got := chosenRows(sel); !slices.Equal(got, []int{3}) {
		t.Fatalf("DragSelect: the release after the press chose %v", got)
	}
	x5, y5 := rowCenter(t, tt, "Row 5")
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x5), Y: float64(y5), Mods: platform.Modifiers(Cmd)})
	if got := chosenRows(sel); !slices.Equal(got, []int{3, 5}) {
		t.Fatalf("DragSelect: Cmd-press on Row 5 chose %v", got)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x5), Y: float64(y5), Mods: platform.Modifiers(Cmd)})
	if got := chosenRows(sel); !slices.Equal(got, []int{3, 5}) {
		t.Fatalf("DragSelect: the release of the Cmd-click took Row 5 out again: %v", got)
	}
	tt.Press(x, y)
	if got := chosenRows(sel); !slices.Equal(got, []int{3, 5}) {
		t.Fatalf("DragSelect: a press on chosen Row 3 changed the choice before the release: %v", got)
	}
	tt.Release(x, y)
	if got := chosenRows(sel); !slices.Equal(got, []int{3}) {
		t.Fatalf("DragSelect: the click on chosen Row 3 chose %v", got)
	}

	tt2, s, sel2 := cellTable(t, 8, 3, 0)
	cx, cy := rowCenter(t, tt2, "r4c2")
	tt2.Press(cx, cy)
	wantBlock(t, s, sel2, "CellSelect: press on r4c2", []int{4}, ColumnRange{Cursor: "B", Anchor: "B"}, []string{"B"})
	tt2.Release(cx, cy)

	var plainSel Selection[int]
	plain := ListState{Selection: &plainSel}
	tt3 := NewTester(func(c *Context) {
		List(c, &plain, 6, func(i int) { Textf(c, "Row %d", i) }).Grow(1)
	}, 300, 300)
	tt3.Frame()
	px, py := rowCenter(t, tt3, "Row 2")
	tt3.Press(px, py)
	if got := chosenRows(&plainSel); len(got) != 0 {
		t.Fatalf("plain list: press on Row 2 chose %v before the release", got)
	}
	tt3.Release(px, py)
	if got := chosenRows(&plainSel); !slices.Equal(got, []int{2}) {
		t.Fatalf("plain list: click on Row 2 chose %v", got)
	}
}

// A press that chose a row but did not end as a click on it, as one
// dragged away, does not swallow the next click on the row; nor does the
// release of one row and the press of another in one frame cross them.
func TestPressChoiceDoesNotSwallowClicks(t *testing.T) {
	tt, _, sel := dragSelectList(t, 20)
	x, y := rowCenter(t, tt, "Row 3")
	_, y5 := rowCenter(t, tt, "Row 5")
	tt.Press(x, y)
	tt.Move(x, y+6)
	tt.Move(x, y5)
	tt.Release(x, y5)
	if got := chosenRows(sel); !slices.Equal(got, []int{3, 4, 5}) {
		t.Fatalf("dragged Row 3 to Row 5: chose %v", got)
	}
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y)})
	tt.Frame()
	if got := chosenRows(sel); !slices.Equal(got, []int{3}) {
		t.Fatalf("then a click on Row 3 within one frame: chose %v", got)
	}

	_, y1 := rowCenter(t, tt, "Row 1")
	_, y7 := rowCenter(t, tt, "Row 7")
	tt.Press(x, y7)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y7)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y1)})
	tt.Frame()
	tt.Release(x, y1)
	if got := chosenRows(sel); !slices.Equal(got, []int{1}) {
		t.Fatalf("Row 7 released and Row 1 pressed in one frame, then released: chose %v", got)
	}
}

// A Shift-press extends the choice, and a drag from it goes on extending
// from where the choice began rather than starting again from the row.
func TestShiftPressDragExtends(t *testing.T) {
	tt, _, sel := dragSelectList(t, 20)
	tt.Click("Row 2")
	x, y := rowCenter(t, tt, "Row 5")
	_, y6 := rowCenter(t, tt, "Row 6")
	tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: float64(x), Y: float64(y), Mods: platform.Modifiers(Shift)})
	tt.Move(x, y+6)
	tt.Move(x, y6)
	if got := chosenRows(sel); !slices.Equal(got, []int{2, 3, 4, 5, 6}) {
		t.Fatalf("click Row 2, Shift-press Row 5, drag to Row 6: chose %v", got)
	}
	tt.Release(x, y6)
	if got := chosenRows(sel); !slices.Equal(got, []int{2, 3, 4, 5, 6}) {
		t.Fatalf("and released: chose %v", got)
	}
}

// A drag chooses the rows under the pointer as the list shows them, moved
// by a transform of an element around it.
func TestDragSelectUnderTransform(t *testing.T) {
	var sel Selection[int]
	s := &ListState{Selection: &sel, DragSelect: true}
	tt := NewTester(func(c *Context) {
		Box(c).Grow(1).Translate(0, -40).Children(func() {
			List(c, s, 20, func(i int) { Textf(c, "Row %d", i) }).Grow(1)
		})
	}, 300, 400)
	tt.Frame()
	x, y := rowCenter(t, tt, "Row 2")
	_, y5 := rowCenter(t, tt, "Row 5")
	tt.Press(x, y)
	tt.Move(x, y+6)
	tt.Move(x, y5)
	tt.Release(x, y5)
	if got := chosenRows(&sel); !slices.Equal(got, []int{2, 3, 4, 5}) {
		t.Fatalf("dragged Row 2 to Row 5 under Translate(0, -40): chose %v", got)
	}
}

// DragSelecting reports a drag choosing rows while the pointer is held,
// for work to put off until it ends, and not from the release on.
func TestDragSelecting(t *testing.T) {
	tt, s, _ := dragSelectList(t, 20)
	if s.DragSelecting() {
		t.Fatal("drag selecting before any press")
	}
	x, y := rowCenter(t, tt, "Row 1")
	_, y4 := rowCenter(t, tt, "Row 4")
	tt.Press(x, y)
	if s.DragSelecting() {
		t.Fatal("drag selecting on a press that has not moved")
	}
	tt.Move(x, y+6)
	tt.Move(x, y4)
	if !s.DragSelecting() {
		t.Fatal("not drag selecting while dragging Row 1 to Row 4")
	}
	tt.Release(x, y4)
	if s.DragSelecting() {
		t.Fatal("drag selecting after the release")
	}
}
