//go:build !mygo_noinspector

package ui

import "testing"

// The inspector's own fields keep the system's editing menu: its panel
// keeps its own look, over the window's overlays.
func TestMenuPresentationInspectorFields(t *testing.T) {
	tt, _ := inspected(drawingMenus(func(c *context) {
		coreColumn(c).Key("box").Width(100).Children(func() { coreText(c, "Inside") })
	}))
	openInspector(tt)
	chooseKeyed(t, tt, "box")
	clickValue(t, tt, "width")
	tt.Key(Shift, KeyF10)
	if tt.h.menu == nil || tt.rt.menu.drawn != nil {
		t.Fatalf("the inspector's field: system menu %v, drawn %v", tt.h.menu != nil, tt.rt.menu.drawn != nil)
	}
}
