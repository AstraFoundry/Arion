package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// TestMenuItemTooltip gives an item a tooltip: the system's menu takes it
// as its own items' (shown on macOS and Linux), and a drawn menu shows it
// as the pointer rests on the item.
func TestMenuItemTooltip(t *testing.T) {
	const tip = "Runs the query under the caret"
	menu := func(m *Menu) {
		m.Item("Open")
		m.Item("Run").Tooltip(tip)
	}
	t.Run("system", func(t *testing.T) {
		tt := coreNewTester(func(c *context) {
			coreRow(c).Padding(20).Children(func() { coreText(c, "Report.pdf") }).ContextMenu(menu)
		}, 400, 300)
		if err := tt.RightClick("Report.pdf"); err != nil {
			t.Fatal(err)
		}
		if m := tt.h.menu; m == nil || len(m.Items) != 2 || m.Items[1].ToolTip != tip || m.Items[0].ToolTip != "" {
			t.Fatalf("the system's menu %v", m)
		}
	})
	t.Run("drawn", func(t *testing.T) {
		tt, now := clockTester(func(c *context) {
			coreRow(c).Padding(20).Children(func() { coreText(c, "Report.pdf") }).PopoverMenu(menu)
		}, 400, 300)
		tick := ticker(tt, now)
		if err := tt.RightClick("Report.pdf"); err != nil {
			t.Fatal(err)
		}
		r, ok := tt.Find("Run")
		if !ok {
			t.Fatal("no drawn item Run")
		}
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tick(700 * time.Millisecond)
		if !tt.HasText(tip) {
			t.Fatalf("no tip once the pointer rested on the item: texts %q", tt.Texts())
		}
	})
}

// TestMenuItemTooltipByKeys shows a drawn item's tip as the keys rest on
// it, as on a focused element's (the WAI-ARIA tooltip pattern): the menu
// gives the focus to its highlighted item (ActiveDescendant). Assistive
// technology reads the tip as the item's description, and the shortcut as
// its key shortcuts.
func TestMenuItemTooltipByKeys(t *testing.T) {
	const tip = "Runs the query under the caret"
	tt := coreNewTester(func(c *context) {
		coreRow(c).Padding(20).Focusable().Label("Report").Children(func() { coreText(c, "Report.pdf") }).PopoverMenu(func(m *Menu) {
			m.Item("Open")
			m.Item("Run").Shortcut(Cmd, KeyEnter).Tooltip(tip)
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Report.pdf")
	tt.Key(0, KeyContextMenu)
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Run")
	if n := accessFocus(t, tt); n.Description != tip || n.KeyShortcuts == "" {
		t.Errorf("Run reads with the description %q and key shortcuts %q", n.Description, n.KeyShortcuts)
	}
	tt.Advance(time.Second)
	if !tt.HasText(tip) {
		t.Error("no tip once the keys rested on the item")
	}
}

// TestMenuKeysThenPointer keeps the focus a pointer's once the pointer
// takes over in a menu the keys moved in: the menu button the focus goes
// back to shows neither its focus nor its tip.
func TestMenuKeysThenPointer(t *testing.T) {
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Fill().Padding(40).AlignItems(Start).Children(func() {
			coreButton(c, "More").Tooltip("More actions").PopoverMenuButton(func(m *Menu) {
				m.Item("Copy")
				m.Item("Paste")
			})
		})
	}, 400, 300)
	tick := ticker(tt, now)
	tt.Click("More")
	tt.Key(0, KeyDown)
	r, _ := tt.Find("Paste")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	if tt.HasText("Copy") {
		t.Fatal("the menu stays after a choice")
	}
	tt.Move(380, 280)
	tick(time.Second)
	if tt.rt.focusVisible || tt.HasText("More actions") {
		t.Errorf("after a pointer's choice: focus visible %v, tip %v", tt.rt.focusVisible, tt.HasText("More actions"))
	}
}
