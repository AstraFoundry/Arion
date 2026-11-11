package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// TestSelectBaseSetOpen opens a select from its closed trigger as the
// user types an option's first letter, the highlight on that option, and
// closes it as Tab moves on (#1173).
func TestSelectBaseSetOpen(t *testing.T) {
	size := "M"
	sizes := []string{"S", "M", "L", "XL"}
	var highlighted string
	tt := coreNewTester(func(c *context) {
		sel := coreSelectBase(c, &size)
		sel.Trigger.Label("size").Padding(4).Children(func() { coreText(c, "Size: "+size) })
		if !sel.Open() && sel.Trigger.Shortcut(0, KeyL) {
			sel.SetOpen(true)
			sel.Highlight("L")
		}
		if !sel.Trigger.Focused() {
			sel.SetOpen(false) // Tab moved on
		}
		highlighted = ""
		sel.Popup(func(panel *node) {
			for _, s := range sizes {
				item := sel.Item(s).Padding(4).Label("item " + s)
				if item.Highlighted() {
					highlighted = s
				}
				item.Children(func() { coreText(c, s) })
			}
		})
		coreButton(c, "after")
	}, 300, 300)
	tt.Key(0, KeyTab) // the trigger has the focus
	// Typed before the popup ever showed: the option is highlighted once
	// it is built.
	tt.Key(0, KeyL)
	if !tt.HasText("XL") || highlighted != "L" || size != "M" {
		t.Fatalf("L opens the popup %v, highlighting %q, choosing %q", tt.HasText("XL"), highlighted, size)
	}
	tt.Key(0, KeyEnter)
	if size != "L" || tt.HasText("XL") {
		t.Errorf("Enter chose %q, the popup open %v", size, tt.HasText("XL"))
	}
	// Opened again, the highlight starts on the choice.
	tt.Key(0, KeyDown)
	if !tt.HasText("XL") || highlighted != "L" {
		t.Fatalf("Down opens the popup %v, highlighting %q", tt.HasText("XL"), highlighted)
	}
	tt.Key(0, KeyDown)
	if highlighted != "XL" {
		t.Errorf("Down highlights %q", highlighted)
	}
	// Tab moves on, which closes it, choosing nothing.
	tt.Key(0, KeyTab)
	if tt.HasText("XL") || size != "L" {
		t.Errorf("Tab leaves the popup open %v, choosing %q", tt.HasText("XL"), size)
	}
	if f := tt.rt.states[tt.rt.focused]; f == nil || f.role == RolePopUpButton {
		t.Error("Tab left the focus on the trigger")
	}
}

// TestSelectBaseOpenCallerFlag checks that a select whose open flag the
// caller owns shows its popup while the flag is set, sets it as it opens
// and closes itself, and starts the highlight on the choice however it
// opens (#1173).
func TestSelectBaseOpenCallerFlag(t *testing.T) {
	size, open := "M", false
	sizes := []string{"S", "M", "L", "XL"}
	var highlighted string
	var trigger platform.AccessNode
	tt := coreNewTester(func(c *context) {
		sel := coreSelectBaseOpen(c, &size, &open)
		sel.Trigger.Label("size").Padding(4).Children(func() { coreText(c, "Size: "+size) })
		highlighted = ""
		sel.Popup(func(panel *node) {
			for _, s := range sizes {
				item := sel.Item(s).Padding(4).Label("item " + s)
				if item.Highlighted() {
					highlighted = s
				}
				item.Children(func() { coreText(c, s) })
			}
		})
	}, 300, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	expanded := func() bool {
		trigger = accessNode(t, tt.h.access, platform.RolePopUpButton, "size")
		return trigger.States&platform.AccessExpanded != 0
	}
	// The caller opens it, as between frames.
	open = true
	tt.Frame()
	if !tt.HasText("XL") || highlighted != "M" || !expanded() {
		t.Fatalf("the caller's flag opens the popup %v, highlighting %q, expanded %v", tt.HasText("XL"), highlighted, expanded())
	}
	// A click on an option chooses it and clears the flag.
	tt.Click("item L")
	if size != "L" || open || tt.HasText("XL") || expanded() {
		t.Errorf("a click chose %q, the flag %v, the popup open %v", size, open, tt.HasText("XL"))
	}
	// The trigger sets it, and Escape clears it.
	tt.Click("size")
	if !open || highlighted != "L" {
		t.Errorf("a click on the trigger: the flag %v, highlighting %q", open, highlighted)
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEscape)
	if open || tt.HasText("XL") || size != "L" {
		t.Errorf("Escape: the flag %v, the popup open %v, choosing %q", open, tt.HasText("XL"), size)
	}
	// Opened by the caller again, the highlight is back on the choice,
	// and the caller closes it.
	open = true
	tt.Frame()
	if highlighted != "L" {
		t.Errorf("opened again, the popup highlights %q", highlighted)
	}
	open = false
	tt.Frame()
	if tt.HasText("XL") || expanded() {
		t.Error("the caller's flag does not close the popup")
	}
}

// TestSelectBaseOpenShrinks builds a list that grows out of its trigger
// and shrinks back into it as it closes, as Arc's morph-select, on a flag
// the caller owns: the list stays, taking no input, until it has shrunk
// (#1173).
func TestSelectBaseOpenShrinks(t *testing.T) {
	size := "M"
	sizes := []string{"S", "M", "L", "XL"}
	var open, closing, listShown bool
	var grow float32
	var panelH float32
	tt := coreNewTester(func(c *context) {
		if !open && listShown {
			open, closing = true, true
		}
		sel := coreSelectBaseOpen(c, &size, &open)
		sel.Trigger.Label("size").Height(24).Padding(4).Children(func() { coreText(c, "Size: "+size) })
		var target float32
		if open && !closing {
			target = 1
		}
		grow = sel.Trigger.Animate("grow", target, 200*time.Millisecond)
		if closing && grow == 0 {
			open, closing = false, false
		}
		panelH = 0
		panel := sel.Popup(func(panel *node) {
			panel.IgnoreInput(closing).ClipY().Height(24 + grow*(4*24-24))
			for _, s := range sizes {
				sel.Item(s).Height(24).Label("item " + s).Children(func() { coreText(c, s) })
			}
		})
		if panel != nil {
			panelH = panel.Bounds().H
		}
		listShown = panel != nil
	}, 300, 300)
	tt.Click("size")
	tt.Advance(time.Second)
	if !open || grow != 1 || panelH != 4*24 {
		t.Fatalf("open: the flag %v, grown %v, the list %v high", open, grow, panelH)
	}
	tt.Click("item L")
	if size != "L" || !closing || !tt.HasText("XL") {
		t.Fatalf("chose %q: closing %v, the list shows %v", size, closing, tt.HasText("XL"))
	}
	tt.Advance(100 * time.Millisecond)
	if !closing || grow <= 0 || grow >= 1 || panelH <= 24 || panelH >= 4*24 {
		t.Errorf("halfway: closing %v, grown %v, the list %v high", closing, grow, panelH)
	}
	// Shrinking, it takes no clicks.
	tt.ClickAt(150, 24+12)
	if size != "L" {
		t.Errorf("a click on the shrinking list chose %q", size)
	}
	tt.Advance(time.Second)
	if open || closing || tt.HasText("XL") {
		t.Errorf("shrunk: the flag %v, closing %v, the list shows %v", open, closing, tt.HasText("XL"))
	}
}
