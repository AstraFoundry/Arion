package ui

import "testing"

// The toolbar's overflow button is chrome: when every control fits, the
// button is hidden and must not widen the toolbar. Regression test for
// https://github.com/egoist/mygo/issues/491
func TestToolbarEndsAtLastControl(t *testing.T) {
	tt := NewTester(func(c *Context) {
		Row(c).Width(400).Children(func() {
			Text(c, "Changes")
			Spacer(c)
			Toolbar(c, func() {
				Button(c, "One")
				Button(c, "Two").Label("Last")
			}).Padding(0).Label("Tools")
		})
	}, 400, 100)
	bar, _ := tt.Find("Tools")
	last, _ := tt.Find("Last")
	if end := last.X + last.W; end != bar.X+bar.W {
		t.Fatalf("last button ends at %v, toolbar ends at %v", end, bar.X+bar.W)
	}
}
