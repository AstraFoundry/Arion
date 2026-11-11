//go:build !mygo_noinspector

package ui

import "testing"

// TestInspectorDirection lays the window out right to left with the
// inspector's toggle, and back, the panel staying left to right.
func TestInspectorDirection(t *testing.T) {
	var dir Direction
	tt, _ := inspected(func(c *context) {
		dir = c.Direction()
		coreRow(c).Fill().Children(func() { coreText(c, "Content") })
	})
	openInspector(tt)
	if err := tt.Click("RTL"); err != nil {
		t.Fatal(err)
	}
	if dir != RightToLeft {
		t.Fatal("the toggle did not lay the window out right to left")
	}
	panel := tt.rt.states[tt.rt.insp.panel]
	if panel == nil || panel.rtl {
		t.Error("the panel is not left to right")
	}
	tt.Click("RTL")
	if dir != LeftToRight {
		t.Error("the toggle did not lay the window out left to right again")
	}
}
