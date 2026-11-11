package ui

import "testing"

// TestPreserveSpaces checks that spaces keep their advance on a text that
// asks for it (CSS Text 3, white-space: break-spaces), and are dropped at
// the end of a line otherwise (white-space: normal).
func TestPreserveSpaces(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreRow(c).Children(func() {
			coreText(c, "  ").Label("gap").PreserveSpaces()
			coreText(c, "a").Label("a1")
		})
		coreRow(c).Children(func() {
			coreText(c, "  ").Label("plain gap")
			coreText(c, "a").Label("a2")
		})
		coreRow(c).Children(func() {
			coreText(c, "a  ").Label("kept").PreserveSpaces()
		})
		coreRow(c).Children(func() {
			coreText(c, "a  ").Label("dropped")
		})
	}, 300, 200)
	rect := func(l string) Rect {
		r, ok := tt.Find(l)
		if !ok {
			t.Fatalf("no element %q", l)
		}
		return r
	}
	if g, a := rect("gap"), rect("a1"); g.W <= 0 || a.X <= g.X {
		t.Errorf("a text of spaces is %v wide, and the next text starts at %v", g.W, a.X-g.X)
	}
	if g := rect("plain gap"); g.W != 0 {
		t.Errorf("without it, a text of spaces is %v wide", g.W)
	}
	if k, d := rect("kept"), rect("dropped"); k.W <= d.W {
		t.Errorf("trailing spaces: %v wide kept, %v dropped", k.W, d.W)
	}
}
