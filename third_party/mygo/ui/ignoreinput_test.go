package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A row showing its Delete button while it is hovered keeps it built at
// rest, ignoring input: the pointer reaches the row through it, but
// assistive technology finds the button and presses it.
func TestIgnoreInput(t *testing.T) {
	var rowClicks, deletes, innerClicks int
	var rowHovered, delHovered bool
	var rowID uint64
	view := func(c *context) {
		row := coreRow(c).Size(200, 40).Cursor(CursorPointer).Label("row")
		rowHovered, rowID = row.Hovered(), row.id
		if row.Clicked() {
			rowClicks++
		}
		row.Children(func() {
			coreBox(c).Grow(1)
			del := coreButton(c, "Delete").Cursor(CursorText).IgnoreInput(!rowHovered)
			if !rowHovered {
				del.Opacity(0)
			}
			delHovered = del.Hovered()
			if del.Clicked() {
				deletes++
			}
		})
		// Inside an element ignoring input, a clickable one ignores it too.
		coreBox(c).Size(200, 40).IgnoreInput(true).Children(func() {
			if coreBox(c).Size(40, 40).Label("inner").Clicked() {
				innerClicks++
			}
		})
	}
	tt := coreNewTester(view, 300, 200)
	del, ok := tt.Find("Delete")
	if !ok {
		t.Fatal("the button ignoring input is not found")
	}
	x, y := del.X+del.W/2, del.Y+del.H/2

	// The pointer goes through it to the row: its hover, cursor and press.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x), Y: float64(y)})
	if chain := tt.rt.hitChain(x, y); len(chain) == 0 || chain[0] != rowID {
		t.Errorf("the pointer over the button hits %v, not the row", chain)
	}
	if tt.Cursor() != CursorPointer {
		t.Errorf("cursor %v over the button ignoring input, want the row's", tt.Cursor())
	}
	tt.settle()
	if !rowHovered {
		t.Fatal("the row does not hover under the button")
	}
	// Now that the row hovers, the button takes the pointer itself.
	tt.Move(x, y)
	if !delHovered || tt.Cursor() != CursorText {
		t.Errorf("the button shown takes no pointer: hovered %v, cursor %v", delHovered, tt.Cursor())
	}
	tt.ClickAt(x, y)
	if deletes != 1 || rowClicks != 0 {
		t.Errorf("clicking the button shown: %d deletes, %d row clicks", deletes, rowClicks)
	}
	// Back at rest, a press on it is the row's.
	tt.Move(250, 100)
	tt.settle()
	if rowHovered {
		t.Fatal("the row hovers with the pointer away")
	}
	tt.Press(x, y)
	tt.Release(x, y)
	if deletes != 1 || rowClicks != 1 {
		t.Errorf("pressing the button at rest: %d deletes, %d row clicks", deletes, rowClicks)
	}
	inner, _ := tt.Find("inner")
	tt.ClickAt(inner.X+inner.W/2, inner.Y+inner.H/2)
	if innerClicks != 0 {
		t.Error("an element inside one ignoring input took a click")
	}

	// Assistive technology finds it, as a button it can press but not
	// focus, and pressing it runs its handler.
	tt.Move(250, 150)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	n := accessNode(t, tt.h.access, platform.RoleButton, "Delete")
	if n.Actions&platform.ActionPress == 0 || n.Actions&platform.ActionFocus != 0 || n.States&(platform.AccessFocusable|platform.AccessOffscreen) != 0 {
		t.Errorf("the button ignoring input to assistive technology: %+v", n)
	}
	if b := n.Bounds; b.X > float64(del.X) || b.Y > float64(del.Y) || b.X+b.W < float64(del.X+del.W) || b.Y+b.H < float64(del.Y+del.H) {
		t.Errorf("bounds %+v, not around its text at %+v", b, del)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
	if deletes != 2 || rowClicks != 1 {
		t.Errorf("pressing it from assistive technology: %d deletes, %d row clicks", deletes, rowClicks)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessFocus})
	if tt.rt.focused == n.ID || tt.h.access.Focus == n.ID {
		t.Error("assistive technology focused the button ignoring input")
	}
	accessNode(t, tt.h.access, platform.RoleGroup, "inner")
}

// Tab passes an element ignoring input by, and neither a click nor Focus
// gives it the focus, which it loses as it starts ignoring input.
func TestIgnoreInputTakesNoFocus(t *testing.T) {
	ignore, focusB := true, false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreButton(c, "A")
			b := coreButton(c, "B").IgnoreInput(ignore)
			if focusB {
				b.Focus()
			}
			coreBox(c).IgnoreInput(ignore).Children(func() {
				coreButton(c, "inside")
			})
			coreButton(c, "C")
		})
	}, 200, 300)
	tt.Key(0, KeyTab)
	if !tt.Focused("A") {
		t.Fatal("Tab does not focus A")
	}
	tt.Key(0, KeyTab)
	if !tt.Focused("C") {
		t.Errorf("Tab from A goes to %d, not C past B and the button inside", tt.rt.focused)
	}
	tt.Key(Shift, KeyTab)
	if !tt.Focused("A") {
		t.Error("Shift+Tab from C does not pass B by")
	}
	// A click on it is a click on what is under it, which takes no focus.
	tt.Click("B")
	if tt.Focused("B") {
		t.Error("clicking B focused it")
	}
	focusB = true
	tt.Frame()
	if tt.Focused("B") {
		t.Error("Focus gave the focus to B ignoring input")
	}
	// Taking input again, it takes the focus; then loses it as it ignores
	// input once more.
	ignore = false
	tt.Frame()
	tt.Frame()
	if !tt.Focused("B") {
		t.Fatal("Focus does not focus B taking input")
	}
	focusB, ignore = false, true
	tt.Frame()
	if tt.rt.focused != 0 {
		t.Errorf("B kept the focus as it ignores input: %d", tt.rt.focused)
	}
}

// The wheel over an element ignoring input scrolls what is under it, even
// a scroll container it is not inside, and of what goes through the keys,
// assistive technology still sets the value of a text input ignoring
// input, without focusing it.
func TestIgnoreInputScrollsAndSetsValue(t *testing.T) {
	var s ScrollState
	name := "Ada"
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Children(func() {
			coreScroll(c).Fill().TrackScroll(&s).Children(func() {
				coreBox(c).Height(1000).Shrink(0)
			})
			coreScroll(c).Absolute().Left(0).Top(0).Right(0).Height(100).IgnoreInput(true).Children(func() {
				coreTextInput(c, &name).Label("name")
				coreBox(c).Height(1000).Shrink(0)
			})
		})
	}, 200, 200)
	tt.Move(100, 50)
	tt.Scroll(100, 50, 0, 120)
	if s.Y != 120 {
		t.Errorf("the wheel over the element ignoring input scrolled what is under it to %v", s.Y)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	f := accessNode(t, tt.h.access, platform.RoleTextField, "")
	if f.Actions&platform.ActionSetValue == 0 || f.Actions&platform.ActionFocus != 0 {
		t.Errorf("the text input ignoring input: %+v", f)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: f.ID, Action: platform.AccessSetValue, Text: "Grace"})
	if name != "Grace" || tt.rt.focused != 0 {
		t.Errorf("setting its value: %q, focus %d", name, tt.rt.focused)
	}
}
