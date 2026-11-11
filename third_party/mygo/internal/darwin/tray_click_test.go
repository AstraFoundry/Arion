//go:build darwin

package darwin

import "testing"

// TestSecondaryClick tells a status item's right-clicks from its clicks: the
// right button's, and the primary button's with Control held, whatever else is.
func TestSecondaryClick(t *testing.T) {
	const (
		leftUp, rightUp                 = 2, 4
		shift, control, option, command = 1 << 17, 1 << 18, 1 << 19, 1 << 20
	)
	for _, c := range []struct {
		typ, flags uint
		want       bool
	}{
		{leftUp, 0, false},
		{leftUp, shift, false},
		{leftUp, option, false},
		{leftUp, command, false},
		{leftUp, control, true},
		{leftUp, control | shift, true},
		{rightUp, 0, true},
		{rightUp, control, true},
	} {
		if got := secondaryClick(c.typ, c.flags); got != c.want {
			t.Errorf("type %d, flags %#x: %v, want %v", c.typ, c.flags, got, c.want)
		}
	}
}

// TestClickedCopy finds the copy of a menu bar item that a click hit from the
// pointer's place on screen and the click's place in the item's window: here a
// copy on an external display, far from the item's window.
func TestClickedCopy(t *testing.T) {
	got := clickedCopy(NSPoint{X: 2251, Y: 1422}, NSPoint{X: 30, Y: 12}, NSSize{Width: 54, Height: 24})
	want := NSRect{Origin: NSPoint{X: 2221, Y: 1410}, Size: NSSize{Width: 54, Height: 24}}
	if got != want {
		t.Errorf("clickedCopy = %+v, want %+v", got, want)
	}
}

// TestItemFrame keeps the copy clicked while the item's window keeps the frame
// it had at the click, and goes back to the window's frame without a click or
// once the window moved.
func TestItemFrame(t *testing.T) {
	window := NSRect{Origin: NSPoint{X: 159, Y: 1050}, Size: NSSize{Width: 54, Height: 24}}
	clicked := NSRect{Origin: NSPoint{X: 2221, Y: 1410}, Size: NSSize{Width: 54, Height: 24}}
	if got := itemFrame(window, NSRect{}, NSRect{}); got != window {
		t.Errorf("no click: %+v, want the window's %+v", got, window)
	}
	if got := itemFrame(window, clicked, window); got != clicked {
		t.Errorf("clicked: %+v, want the copy's %+v", got, clicked)
	}
	moved := window
	moved.Origin.X -= 40
	if got := itemFrame(moved, clicked, window); got != moved {
		t.Errorf("window moved: %+v, want its new frame %+v", got, moved)
	}
}
