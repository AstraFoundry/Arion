package ui

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestPopoverMenuIconColor draws a template icon in its item's IconColor,
// but in the highlight's text color while highlighted, dimmed while
// disabled, and leaves an icon in its own colors as it is.
func TestPopoverMenuIconColor(t *testing.T) {
	tmpl := &platform.MenuIcon{PNG: menuIconPNG(t, color.NRGBA{A: 255}), Template: true}
	blue := &platform.MenuIcon{PNG: menuIconPNG(t, color.NRGBA{B: 255, A: 255})}
	red := RGB(220, 20, 20)
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreText(c, "Row").PopoverMenu(func(m *Menu) {
				m.Item("Trash").Icon(testMenuIcon{tmpl}).IconColor(red)
				m.Item("Plain").Icon(testMenuIcon{tmpl})
				m.Item("Off").Icon(testMenuIcon{tmpl}).IconColor(red).Disabled(true)
				m.Item("Own").Icon(testMenuIcon{blue}).IconColor(red)
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	text := func(s string) Rect {
		var r Rect
		for _, n := range tt.rt.labels {
			if n.text == s {
				r = n.r
			}
		}
		return r
	}
	// at returns the color at the middle of the box of an item's icon.
	at := func(s string) (int, int, int) {
		r := text(s)
		x := int(r.X) - 8 - platform.MenuIconSize/2
		cr, cg, cb, _ := tt.Image().At(x, int(r.Y+r.H/2)).RGBA()
		return int(cr >> 8), int(cg >> 8), int(cb >> 8)
	}
	isRed := func(r, g, b int) bool { return r > 180 && g < 70 && b < 70 }
	if r, g, b := at("Trash"); !isRed(r, g, b) {
		t.Errorf("the icon with IconColor is (%d, %d, %d), want red", r, g, b)
	}
	if r, g, b := at("Plain"); isRed(r, g, b) || r > 120 {
		t.Errorf("the icon without IconColor is (%d, %d, %d), want the text's dark color", r, g, b)
	}
	if r, g, b := at("Off"); isRed(r, g, b) {
		t.Errorf("the disabled item's icon is (%d, %d, %d), want it dimmed as its label", r, g, b)
	}
	if r, g, b := at("Own"); r > 70 || b < 180 {
		t.Errorf("the icon in its own colors is (%d, %d, %d), want blue", r, g, b)
	}

	// Highlighted, it takes the highlight's text color.
	tt.Key(0, KeyDown)
	th := tt.rt.c.theme
	want := th.AccentText.SRGB()
	near := func(v int, w uint8) bool { return v-int(w) <= 40 && int(w)-v <= 40 }
	if r, g, b := at("Trash"); isRed(r, g, b) || !near(r, want.R) || !near(g, want.G) || !near(b, want.B) {
		t.Errorf("the highlighted icon is (%d, %d, %d), want the highlight's text color %v", r, g, b, want)
	}
}

// TestContextMenuIconColor gives the system's menu the item's IconColor, in
// sRGB, which its backends leave out.
func TestContextMenuIconColor(t *testing.T) {
	ic := &platform.MenuIcon{Template: true}
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreText(c, "Row").ContextMenu(func(m *Menu) {
				m.Item("Trash").Icon(testMenuIcon{ic}).IconColor(RGB(220, 20, 20))
				m.Item("Plain").Icon(testMenuIcon{ic})
			})
		})
	}
	tt := coreNewTester(view, 200, 200)
	if err := tt.RightClick("Row"); err != nil {
		t.Fatal(err)
	}
	items := tt.h.menu.Items
	if c := items[0].IconColor; c == nil || *c != (platform.Color{R: 220, G: 20, B: 20, A: 255}) {
		t.Errorf("IconColor = %v, want (220, 20, 20)", c)
	}
	if items[1].IconColor != nil {
		t.Errorf("an item without IconColor has %v", items[1].IconColor)
	}
}
