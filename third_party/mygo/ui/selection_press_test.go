package ui

import (
	"fmt"
	"testing"
)

func TestPressedSelectionRebuildsBeforePaint(t *testing.T) {
	for _, virtual := range []bool{false, true} {
		t.Run(fmt.Sprintf("virtual=%v", virtual), func(t *testing.T) {
			selected, draft, builds := 0, "", 0
			var state ListState
			tt := coreNewTester(func(c *context) {
				builds++
				coreColumn(c).Fill().Children(func() {
					coreTextf(c, "Selected %d", selected)
					list := coreColumn(c).Grow(1).Focusable().Label("Items")
					focused := list.FocusWithin()
					list.Children(func() {
						row := func(i int) {
							e := coreRow(c).Key(i).Height(32).Label(fmt.Sprintf("Row %d", i))
							if i == selected && focused {
								e.Background(c.theme.Accent)
							}
							if e.Pressed() || e.Clicked() {
								selected = i
							}
							e.Children(func() { coreTextf(c, "Row %d", i) })
						}
						if virtual {
							coreList(c, &state, 2, row).Grow(1)
						} else {
							for i := range 2 {
								row(i)
							}
						}
					})
					coreTextInput(c, &draft).Label("Editor")
				})
			}, 240, 200)
			if err := tt.Click("Editor"); err != nil {
				t.Fatal(err)
			}
			r, ok := tt.Find("Row 1")
			if !ok {
				t.Fatal("row is missing")
			}
			builds = 0
			tt.Press(r.X+r.W/2, r.Y+r.H/2)
			if selected != 1 || !tt.HasText("Selected 1") {
				t.Fatalf("pointer down chose %d, but the painted frame still shows %q", selected, tt.Texts())
			}
			if builds != 2 {
				t.Fatalf("pointer down built %d passes, want 2", builds)
			}
			builds = 0
			tt.Frame()
			if builds != 1 {
				t.Fatalf("holding the pointer built %d passes, want 1", builds)
			}
		})
	}
}

func TestButtonClickStillWaitsForRelease(t *testing.T) {
	clicks := 0
	tt := coreNewTester(func(c *context) {
		b := coreButton(c, "Button").Label("Button control")
		if b.Pressed() {
			b.Background(c.theme.Accent)
		}
		if b.Clicked() {
			clicks++
		}
	}, 240, 100)
	r, _ := tt.Find("Button control")
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	if clicks != 0 {
		t.Fatal("button activated on pointer down")
	}
	tt.Release(r.X+r.W/2, r.Y+r.H/2)
	if clicks != 1 {
		t.Fatalf("button click count %d", clicks)
	}
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	tt.Move(r.X+r.W+10, r.Y+r.H/2)
	tt.Release(r.X+r.W+10, r.Y+r.H/2)
	if clicks != 1 {
		t.Fatal("releasing outside the button activated it")
	}
}

// PressedInside tells a press on an element or inside it without the
// element taking the pointer: a table's row still chooses on a click on
// a cell asking it, and a press elsewhere is not one.
func TestPressedInsideLeavesThePress(t *testing.T) {
	sel := -1
	s := ListState{Selected: &sel}
	pressed := map[int]int{}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Padding(10).Children(func() {
			coreTable(c, &s, []TableColumn{{Title: "Name"}}, 3, func(row, col int) {
				cell := coreBox(c).Children(func() { coreTextf(c, "row %d", row) })
				if cell.PressedInside() {
					pressed[row]++
				}
			}).Grow(1)
			coreText(c, "elsewhere")
		})
	}, 400, 300)
	r, _ := tt.Find("row 1")
	tt.ClickAt(r.X+5, r.Y+r.H/2) // on the cell's text, inside it
	tt.Frame()
	if pressed[1] != 1 || sel != 1 {
		t.Fatalf("pressed %v, chosen %d", pressed, sel)
	}
	tt.Click("elsewhere")
	tt.Frame()
	if len(pressed) != 1 || pressed[1] != 1 {
		t.Fatalf("a press elsewhere counted: %v", pressed)
	}
}
