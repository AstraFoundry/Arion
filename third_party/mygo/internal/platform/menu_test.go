package platform

import "testing"

func TestFitIcon(t *testing.T) {
	for _, c := range []struct{ w, h, box, ww, wh int }{
		{16, 16, 16, 16, 16},
		{64, 64, 32, 32, 32},
		{60, 40, 16, 16, 11}, // a 3:2 flag
		{40, 60, 16, 11, 16},
		{4, 3, 24, 24, 18},
		{200, 1, 16, 16, 1}, // never thinner than a pixel
		{0, 10, 16, 0, 0},
	} {
		if w, h := FitIcon(c.w, c.h, c.box); w != c.ww || h != c.wh {
			t.Errorf("FitIcon(%d, %d, %d) = %d, %d, want %d, %d", c.w, c.h, c.box, w, h, c.ww, c.wh)
		}
	}
}

func TestMenuIconNativeOnce(t *testing.T) {
	ic := &MenuIcon{PNG: []byte{1}}
	n := 0
	make := func() any { n++; return nil }
	ic.Native(make)
	ic.Native(make)
	if n != 1 {
		t.Errorf("made %d times, want once, a failure kept too", n)
	}
}

// Apply keeps a snapshot a backend builds menus from later current.
func TestMenuApply(t *testing.T) {
	icon := &MenuIcon{PNG: []byte{1}}
	m := &Menu{Items: []*MenuItem{
		{ID: 1, Label: "A"},
		{ID: 2, Label: "Sub", Type: MenuItemSubmenu, Submenu: &Menu{Items: []*MenuItem{
			{ID: 3, Label: "B", Type: MenuItemRadio},
		}}},
	}}
	old := m.Items[1].Submenu.Items[0]
	m.Apply(&MenuItem{ID: 3, Label: "B", Type: MenuItemRadio, Checked: true, Icon: icon})
	got := m.Items[1].Submenu.Items[0]
	if !got.Checked || got.Icon != icon {
		t.Errorf("item 3 = %+v, want checked with the icon", got)
	}
	if old.Checked {
		t.Error("Apply changed the item it replaced, which a menu being built may hold")
	}
	if m.Items[0].Label != "A" {
		t.Error("Apply changed another item")
	}
	(*Menu)(nil).Apply(got) // no menu, nothing to do
}
