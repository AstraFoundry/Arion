package ui

import "testing"

// A child growing in a column whose height its content sets keeps the
// height of its content, as with CSS's flex: 1.
func TestGrowInColumnOfContentHeight(t *testing.T) {
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreRow(c).Children(func() {
				coreColumn(c).Grow(1).Children(func() {
					coreColumn(c).Grow(1).Padding(10).Children(func() {
						coreText(c, "Label")
					})
				})
			})
			coreText(c, "After")
		})
	}
	tt := coreNewTester(view, 400, 300)
	label, _ := tt.Find("Label")
	after, _ := tt.Find("After")
	if after.Y < label.Y+label.H+10 {
		t.Errorf("the text after the row at %v, over the row's content at %v", after.Y, label)
	}
}
