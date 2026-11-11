package ui

import "testing"

// combo builds a field with a list attached to it, as a combobox does,
// and a button outside it; left counts the passes the combo's FocusLeft
// reported.
type combo struct {
	open, focusList, focusOther bool
	left                        int
}

func (m *combo) view(c *Context) {
	var field Element
	Row(c).AlignItems(Start).Gap(10).Children(func() {
		box := Column(c).Children(func() {
			field = ButtonBase(c).Size(60, 20).Label("field")
		})
		if box.FocusLeft() {
			m.left++
		}
		ButtonBase(c).Size(60, 20).Label("other")
		if m.focusOther {
			ButtonBase(c).Size(60, 20).Label("late").AutoFocus()
		}
	})
	if m.open {
		Overlay(c, func() {
			Column(c).AttachTo(field, AnchorBottomLeft, AnchorTopLeft).Children(func() {
				opt := ButtonBase(c).Size(60, 20).Label("option")
				if m.focusList {
					opt.AutoFocus()
				}
			})
		})
	}
}

func TestFocusLeft(t *testing.T) {
	m := &combo{open: true}
	tt := NewTester(m.view, 300, 200)
	if err := tt.Click("field"); err != nil {
		t.Fatal(err)
	}
	if m.left != 0 {
		t.Fatal("the focus moving into the element left it")
	}
	if err := tt.Click("option"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("option") || m.left != 0 {
		t.Fatalf("the focus on the attached list %v, left %d times; want it inside", tt.Focused("option"), m.left)
	}
	if err := tt.Click("field"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("other"); err != nil {
		t.Fatal(err)
	}
	if m.left != 1 {
		t.Fatalf("the focus moving outside left %d times, want once", m.left)
	}
	tt.Frame()
	if m.left != 1 {
		t.Fatal("FocusLeft told it again in the next frame")
	}
	// Back in, then the window loses the keyboard: the focus goes nowhere.
	if err := tt.Click("field"); err != nil {
		t.Fatal(err)
	}
	tt.SetFocused(false)
	tt.Frame()
	if m.left != 2 {
		t.Fatalf("the window losing the keyboard left %d times, want 2 in all", m.left)
	}
	tt.SetFocused(true)
	tt.Frame()
	if m.left != 2 {
		t.Fatal("the focus coming back left")
	}
}

// TestFocusLeftToAnElementBuiltAfter moves the focus, in the frame, to an
// element built after the one asking: the next pass tells.
func TestFocusLeftToAnElementBuiltAfter(t *testing.T) {
	m := &combo{}
	tt := NewTester(m.view, 300, 200)
	if err := tt.Click("field"); err != nil {
		t.Fatal(err)
	}
	m.focusOther = true
	tt.Frame()
	if !tt.Focused("late") || m.left != 1 {
		t.Fatalf("the focus on late %v, left %d times; want once", tt.Focused("late"), m.left)
	}
}

// TestFocusLeftIntoAListOpening opens the list as the focus moves into it,
// in one frame, before the list was ever placed: not out of the combo.
func TestFocusLeftIntoAListOpening(t *testing.T) {
	m := &combo{}
	tt := NewTester(m.view, 300, 200)
	if err := tt.Click("field"); err != nil {
		t.Fatal(err)
	}
	m.open, m.focusList = true, true
	tt.Frame()
	tt.Frame()
	if !tt.Focused("option") || m.left != 0 {
		t.Fatalf("the focus on the new list %v, left %d times; want it inside", tt.Focused("option"), m.left)
	}
	if tt.FramePending() {
		t.Fatal("frames go on")
	}
	// The list closing with the focus in it, which goes away with it,
	// moves the focus out of no element still built.
	m.open = false
	tt.Frame()
	tt.Frame()
	if m.left != 0 {
		t.Fatalf("the focus going away with the list left the combo %d times", m.left)
	}
}

func TestFocusTakesText(t *testing.T) {
	var takes []bool
	text, ro := "", "read"
	tt := NewTester(func(c *Context) {
		takes = append(takes[:0], c.FocusTakesText())
		Column(c).Children(func() {
			TextInput(c, &text).Label("name")
			TextInput(c, &ro).Label("fixed").ReadOnly(true)
			Button(c, "go")
			Box(c).Size(40, 20).Label("term").Focusable().TextCaret(Rect{W: 1, H: 10}).
				HandleInput(func(InputEvent) bool { return false })
		})
	}, 300, 200)
	if takes[0] {
		t.Fatal("nothing focused takes text")
	}
	for _, c := range []struct {
		click string
		want  bool
	}{{"name", true}, {"go", false}, {"fixed", false}, {"term", true}} {
		if err := tt.Click(c.click); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if !tt.Focused(c.click) {
			t.Fatalf("%s has no focus", c.click)
		}
		if takes[0] != c.want {
			t.Errorf("%s focused: FocusTakesText %v, want %v", c.click, takes[0], c.want)
		}
		if active, _ := tt.TextInputActive(); active != c.want {
			t.Errorf("%s focused: the window takes text %v, FocusTakesText says %v", c.click, active, c.want)
		}
	}
	if err := tt.Click("name"); err != nil {
		t.Fatal(err)
	}
	tt.SetFocused(false)
	tt.Frame()
	if takes[0] {
		t.Fatal("a window without the keyboard takes text")
	}
}
