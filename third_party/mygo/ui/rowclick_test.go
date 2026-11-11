package ui

import (
	"slices"
	"testing"
)

// A cell that reads its row's double click with RowDoubleClicked leaves
// the click to the row, which chooses as a click, Shift-click and
// Cmd-click do.
func TestCellSeesRowDoubleClick(t *testing.T) {
	var s ListState
	var sel Selection[int]
	s.Selection = &sel
	cols := []TableColumn{{Title: "A"}, {Title: "B"}}
	doubleCol, doubles := -1, 0
	tt := NewTester(func(c *Context) {
		Table(c, &s, cols, 6, func(row, col int) {
			cell := Box(c).FillWidth().Children(func() { Textf(c, "r%dc%d", row, col) })
			if cell.RowDoubleClicked() {
				doubleCol, doubles = col, doubles+1
			}
		}).Grow(1)
	}, 400, 300)
	tt.Frame()
	tt.Click("r1c1")
	if got := chosenRows(&sel); !slices.Equal(got, []int{1}) {
		t.Fatalf("click r1c1: chose %v", got)
	}
	tt.ClickWith(Shift, "r3c0")
	if got := chosenRows(&sel); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("Shift-click r3c0: chose %v", got)
	}
	tt.ClickWith(Cmd, "r5c1")
	if got := chosenRows(&sel); !slices.Equal(got, []int{1, 2, 3, 5}) {
		t.Fatalf("Cmd-click r5c1: chose %v", got)
	}
	// The keys extend from the row clicked last.
	tt.Click("r0c0")
	tt.Key(Shift, KeyDown)
	if got := chosenRows(&sel); !slices.Equal(got, []int{0, 1}) {
		t.Fatalf("click r0c0, Shift+Down: chose %v", got)
	}
	r, _ := tt.Find("r4c1")
	tt.ClickAt(r.X+2, r.Y+2)
	tt.ClickAt(r.X+2, r.Y+2)
	tt.Frame()
	if doubles != 1 || doubleCol != 1 || !slices.Equal(chosenRows(&sel), []int{4}) {
		t.Fatalf("double click r4c1: the cell saw %d, in column %d; chose %v", doubles, doubleCol, chosenRows(&sel))
	}
}

// A row whose double click opens a text input keeps the input focused: the
// row's click does not take the focus back to the list.
func TestRowDoubleClickEditorKeepsFocus(t *testing.T) {
	var s ListState
	sel := -1
	s.Selected = &sel
	editing, text := -1, "x"
	var focused bool
	tt := NewTester(func(c *Context) {
		List(c, &s, 4, func(i int) {
			if i == editing {
				in := TextInput(c, &text).AutoFocus()
				focused = in.Focused()
				return
			}
			if Box(c).FillWidth().Children(func() { Textf(c, "Row %d", i) }).RowDoubleClicked() {
				editing = i
			}
		}).Grow(1)
	}, 300, 300)
	tt.Frame()
	r, _ := tt.Find("Row 2")
	tt.ClickAt(r.X+2, r.Y+2)
	tt.ClickAt(r.X+2, r.Y+2)
	tt.Frame()
	tt.Frame()
	if editing != 2 || !focused {
		t.Fatalf("double-clicked Row 2: editing %d, input focused %v", editing, focused)
	}
}
