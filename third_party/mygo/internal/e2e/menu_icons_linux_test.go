//go:build linux && (amd64 || arm64)

package e2e

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// Menu items show icons beside their labels, with their checks, and change
// them in place: replaced, removed, given one.
func TestMenuIconsLinux(t *testing.T) {
	japan := menuIcon(t, 48, 32, color.NRGBA{255, 0, 0, 255}) // 3:2, as flags
	square := menuIcon(t, 64, 64, color.NRGBA{0, 0, 255, 255})
	var chosen string
	click := func(it *mygo.MenuItem, _ *mygo.Window) { chosen = it.ID }
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Servers", Icon: japan, Submenu: []*mygo.MenuItem{
			{ID: "tokyo", Label: "Tokyo", Type: mygo.MenuItemRadio, Checked: true, Icon: japan, Click: click},
			{ID: "ny", Label: "New York", Type: mygo.MenuItemRadio, Icon: square, Click: click},
			{ID: "direct", Label: "Direct", Type: mygo.MenuItemRadio, Click: click},
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
		mygo.RunOnMain(func() { s.w, s.h, s.checked, err = linux.TestMenuItemIcon(w.NativeHandle(), path...) })
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
	want("the menu bar's item", at("Servers"), state{0, 0, false})

	menu.ItemByID("direct").SetIcon(square)
	eventually(t, "Direct to get an icon", func() bool { return at("Servers", "Direct") == state{16, 16, false} })
	menu.ItemByID("tokyo").SetIcon(nil)
	eventually(t, "Tokyo to lose its icon", func() bool { return at("Servers", "Tokyo") == state{0, 0, true} })
	menu.ItemByID("tokyo").SetIcon(square)
	eventually(t, "Tokyo to get an icon again", func() bool { return at("Servers", "Tokyo") == state{16, 16, true} })
	menu.ItemByID("ny").SetIcon(japan)
	menu.ItemByID("ny").SetLabel("Osaka")
	eventually(t, "New York to become Osaka", func() bool {
		var err error
		mygo.RunOnMain(func() { _, _, _, err = linux.TestMenuItemIcon(w.NativeHandle(), "Servers", "Osaka") })
		return err == nil
	})
	want("Osaka", at("Servers", "Osaka"), state{16, 11, false})

	// The item with an icon is chosen as any other, its check following.
	if err := activateMenu(w, "Servers", "Osaka"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "Osaka to be chosen", func() bool {
		var c string
		mygo.RunOnMain(func() { c = chosen })
		return c == "ny"
	})
	eventually(t, "Osaka checked, Tokyo not", func() bool {
		return at("Servers", "Osaka").checked && !at("Servers", "Tokyo").checked
	})
}
