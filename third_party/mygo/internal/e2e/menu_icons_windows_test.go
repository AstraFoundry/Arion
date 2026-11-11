//go:build windows && (amd64 || arm64)

package e2e

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// Menu items show icons beside their labels, with their checks, change
// them in place, and free their bitmaps (Windows 10's) as they do and as
// their menus go.
func TestMenuIconsWindows(t *testing.T) {
	japan := menuIcon(t, 48, 32, color.NRGBA{255, 0, 0, 255}) // 3:2, as flags
	square := menuIcon(t, 64, 64, color.NRGBA{0, 0, 255, 128})
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Servers", Icon: japan, Submenu: []*mygo.MenuItem{
			{ID: "tokyo", Label: "Tokyo", Type: mygo.MenuItemRadio, Checked: true, Icon: japan},
			{ID: "ny", Label: "New York", Type: mygo.MenuItemRadio, Icon: square},
			{ID: "direct", Label: "Direct", Type: mygo.MenuItemRadio},
		}},
	})
	w := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, Content: ui.View(func(c *ui.Context) { ui.Text(c, "menus") })})
	w.SetMenu(menu)
	type state struct {
		w, h    int
		checked bool
	}
	at := func(path ...string) (s state) {
		var err error
		mygo.RunOnMain(func() { s.w, s.h, s.checked, err = win.TestMenuItemIcon(w.NativeHandle(), path...) })
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	want := func(what string, got, want state) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %+v, want %+v", what, got, want)
		}
	}
	want("Tokyo", at("Servers", "Tokyo"), state{16, 11, true})
	want("New York", at("Servers", "New York"), state{16, 16, false})
	want("Direct", at("Servers", "Direct"), state{0, 0, false})

	menu.ItemByID("direct").SetIcon(square)
	eventually(t, "Direct to get an icon", func() bool { return at("Servers", "Direct") == state{16, 16, false} })
	menu.ItemByID("tokyo").SetIcon(nil)
	eventually(t, "Tokyo to lose its icon", func() bool { return at("Servers", "Tokyo") == state{0, 0, true} })
	menu.ItemByID("ny").SetChecked(true)
	eventually(t, "New York checked with its icon", func() bool {
		return at("Servers", "New York") == state{16, 16, true} && !at("Servers", "Tokyo").checked
	})

	// Replacing icons and building the menu again leaves no bitmaps behind.
	var before int
	mygo.RunOnMain(func() { before = win.TestGDIObjects() })
	for i := range 200 {
		ic := japan
		if i%2 == 1 {
			ic = square
		}
		for _, id := range []string{"tokyo", "ny", "direct"} {
			menu.ItemByID(id).SetIcon(ic)
		}
		if i%10 == 0 {
			w.SetMenu(menu)
		}
	}
	w.SetMenu(menu)
	eventually(t, "the last icons", func() bool { return at("Servers", "Tokyo") == state{16, 16, false} })
	var after int
	mygo.RunOnMain(func() { after = win.TestGDIObjects() })
	if after > before+8 {
		t.Errorf("GDI objects went from %d to %d over 600 replaced icons", before, after)
	}
}
