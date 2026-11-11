package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// optionsMenuApp is a view with a menu button whose menu is drawn in the
// window with options, placed where justify and align put it, another
// such button, and what their items did.
type optionsMenuApp struct {
	o, other         PopoverMenuButtonOptions
	justify, align   Align
	chosen           []string
	disabled, hidden bool
}

func (a *optionsMenuApp) view(c *context) {
	coreColumn(c).Fill().Padding(20).Gap(10).Justify(a.justify).AlignItems(a.align).Children(func() {
		coreButton(c, "Edit")
		if !a.hidden {
			corePopoverMenuButton(c, "Actions", func(m *Menu) {
				for _, s := range []string{"Open", "Rename", "Delete"} {
					if m.Item(s).Chosen() {
						a.chosen = append(a.chosen, s)
					}
				}
			}, a.o).Disabled(a.disabled)
		}
		corePopoverMenuButton(c, "Other", func(m *Menu) {
			if m.Item("Elsewhere").Chosen() {
				a.chosen = append(a.chosen, "Elsewhere")
			}
		}, a.other)
	})
}

func TestPopoverMenuButtonPlacement(t *testing.T) {
	tests := []struct {
		name           string
		at, self       Anchor
		justify, align Align
		rtl            bool
		// check says what is wrong with the menu's first item at open and
		// its last at last, for the button at b, or "".
		check func(open, last, b Rect) string
	}{
		{name: "below, at the end", at: AnchorBottomEnd, self: AnchorTopEnd, justify: Center, align: Center,
			check: func(open, last, b Rect) string {
				if open.Y < b.Y+b.H || open.X+open.W > b.X+b.W || open.X+open.W < b.X+b.W-20 {
					return "not below the button, lined up with its end"
				}
				return ""
			}},
		{name: "above, at the start", at: AnchorTopStart, self: AnchorBottomStart, justify: Center, align: Center,
			check: func(open, last, b Rect) string {
				if last.Y+last.H > b.Y-8 || open.X < b.X || open.X > b.X+20 {
					return "not above the button, apart from it, lined up with its start"
				}
				return ""
			}},
		{name: "above, at the end", at: AnchorTopEnd, self: AnchorBottomEnd, justify: Center, align: Center,
			check: func(open, last, b Rect) string {
				if last.Y+last.H > b.Y-8 || open.X+open.W > b.X+b.W || open.X+open.W < b.X+b.W-20 {
					return "not above the button, lined up with its end"
				}
				return ""
			}},
		{name: "after", at: AnchorTopEnd, self: AnchorTopStart, justify: Center, align: Start,
			check: func(open, last, b Rect) string {
				if open.X < b.X+b.W+8 || open.Y < b.Y || open.Y > b.Y+20 {
					return "not after the button, apart from it, lined up with its top"
				}
				return ""
			}},
		{name: "above with no room above flips below", at: AnchorTopEnd, self: AnchorBottomEnd, justify: Start, align: End,
			check: func(open, last, b Rect) string {
				if open.Y < b.Y+b.H || open.X+open.W > b.X+b.W || open.X+open.W < b.X+b.W-20 {
					return "not below the button, lined up with its end"
				}
				return ""
			}},
		{name: "at the end with no room before it flips to the start", at: AnchorBottomEnd, self: AnchorTopEnd, justify: Center, align: Start,
			check: func(open, last, b Rect) string {
				if open.Y < b.Y+b.H || open.X < b.X || open.X > b.X+20 {
					return "not below the button, lined up with its start"
				}
				return ""
			}},
		{name: "at the end, right to left", at: AnchorBottomEnd, self: AnchorTopEnd, justify: Center, align: Center, rtl: true,
			check: func(open, last, b Rect) string {
				if open.Y < b.Y+b.H || open.X < b.X || open.X > b.X+20 {
					return "not below the button, lined up with its left, its end"
				}
				return ""
			}},
		{name: "the default", justify: Center, align: Center,
			check: func(open, last, b Rect) string {
				if open.Y < b.Y+b.H || open.Y > b.Y+b.H+20 || open.X < b.X || open.X > b.X+20 {
					return "not below the button, lined up with its start"
				}
				return ""
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &optionsMenuApp{o: PopoverMenuButtonOptions{At: tc.at, Self: tc.self}, justify: tc.justify, align: tc.align}
			tt := coreNewTester(a.view, 500, 400)
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			if tc.rtl {
				tt.SetDirection(RightToLeft)
			}
			if err := tt.Click("Actions"); err != nil {
				t.Fatal(err)
			}
			// The button's whole box: Find gives the text in it.
			bb := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions").Bounds
			b := Rect{float32(bb.X), float32(bb.Y), float32(bb.W), float32(bb.H)}
			open, ok := tt.Find("Open")
			if !ok {
				t.Fatal("no menu")
			}
			last, _ := tt.Find("Delete")
			if msg := tc.check(open, last, b); msg != "" {
				t.Errorf("Open at %+v, Delete at %+v: %s at %+v", open, last, msg, b)
			}
			// It stays there over the frames after.
			tt.Frame()
			if again, _ := tt.Find("Open"); again != open {
				t.Errorf("Open moved from %+v to %+v", open, again)
			}
		})
	}
}

func TestPopoverMenuButtonOpenState(t *testing.T) {
	var open, other bool
	a := &optionsMenuApp{o: PopoverMenuButtonOptions{Open: &open}, other: PopoverMenuButtonOptions{Open: &other}}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Edit")
	if tt.HasText("Open") || open {
		t.Fatal("the menu shows before it is opened")
	}

	// The app opens it: it takes the focus, with no item highlighted.
	open = true
	tt.Frame()
	if !tt.HasText("Open") || !open {
		t.Fatalf("the app opened the menu, which does not show (open %v): texts %q", open, tt.Texts())
	}
	if s := tt.rt.states[tt.rt.focused]; s == nil || s.role != RoleMenu {
		t.Error("the focus is not on the menu the app opened")
	}
	if n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions"); n.States&platform.AccessExpanded == 0 {
		t.Error("the button is not expanded with its menu open")
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if len(a.chosen) != 1 || a.chosen[0] != "Open" {
		t.Errorf("chosen = %q, want Open, the first item, as Down moved to it", a.chosen)
	}
	if open || tt.HasText("Open") {
		t.Errorf("choosing an item left open %v, the menu showing %v", open, tt.HasText("Open"))
	}
	if !tt.Focused("Edit") {
		t.Error("the focus did not go back to Edit, which had it as the menu opened")
	}

	// The app closes it, as from an event in the background.
	open = true
	tt.Frame()
	open = false
	tt.Frame()
	if tt.HasText("Open") {
		t.Error("the app closed the menu, which still shows")
	}
	if !tt.Focused("Edit") {
		t.Error("the focus did not go back to Edit as the app closed the menu")
	}
	if n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions"); n.States&platform.AccessExpanded != 0 {
		t.Error("the button is expanded with its menu closed")
	}

	// The user opens and closes it: open follows.
	tt.Click("Actions")
	if !open {
		t.Error("open is false with the menu the user opened")
	}
	tt.Key(0, KeyEscape)
	if open || tt.HasText("Open") {
		t.Errorf("Escape left open %v, the menu showing %v", open, tt.HasText("Open"))
	}
	tt.Click("Actions")
	tt.Click("Actions")
	if open {
		t.Error("open is true once the button closed its menu")
	}

	// Another menu opening closes it.
	open = true
	tt.Frame()
	other = true
	tt.Frame()
	if open || !other || tt.HasText("Open") || !tt.HasText("Elsewhere") {
		t.Errorf("the other menu opening left open %v, other %v, texts %q", open, other, tt.Texts())
	}
	other = false
	tt.Frame()
	if tt.HasText("Elsewhere") {
		t.Error("the other menu still shows")
	}

	// A disabled button's does not open, and one that goes away closes.
	a.disabled = true
	tt.Frame()
	open = true
	tt.Frame()
	if open || tt.HasText("Open") {
		t.Errorf("a disabled button's menu opened: open %v", open)
	}
	a.disabled = false
	tt.Frame()
	open = true
	tt.Frame()
	if !tt.HasText("Open") {
		t.Fatal("the menu does not open again")
	}
	a.hidden = true
	tt.Frame()
	if tt.HasText("Open") {
		t.Error("the menu shows without its button")
	}
}

func TestPopoverMenuButtonOptionsCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("two options did not panic")
		}
	}()
	tt := coreNewTester(func(c *context) {
		coreButtonBase(c).Label("More").PopoverMenuButton(func(m *Menu) { m.Item("A") }, PopoverMenuButtonOptions{}, PopoverMenuButtonOptions{})
	}, 200, 200)
	tt.Frame()
}
