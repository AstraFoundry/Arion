//go:build darwin

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
)

// A menu item shows its subtitle on a line of its own from macOS 14, else
// after its title, and loses it as it changes.
func TestMenuItemSubtitle(t *testing.T) {
	prev := mygo.App.Menu()
	defer mygo.App.SetMenu(prev)
	menu := mygo.NewMenu([]*mygo.MenuItem{{Label: "App", Submenu: []*mygo.MenuItem{
		{ID: "code", Label: "123456", Subtitle: "GitHub"},
	}}})
	mygo.App.SetMenu(menu)
	look := func(path ...string) (title, subtitle string, err error) {
		mygo.RunOnMain(func() { title, subtitle, err = darwin.TestMenuItemSubtitle(path...) })
		return title, subtitle, err
	}
	var title, subtitle string
	eventually(t, "the item in the menu", func() bool {
		var err error
		if title, subtitle, err = look("App", "123456"); err != nil {
			title, subtitle, err = look("App", "123456 — GitHub")
		}
		return err == nil
	})
	if !(title == "123456" && subtitle == "GitHub" || title == "123456 — GitHub" && subtitle == "") {
		t.Errorf("title %q, subtitle %q", title, subtitle)
	}
	menu.ItemByID("code").SetSubtitle("")
	eventually(t, "the item without its subtitle", func() bool {
		title, subtitle, err := look("App", "123456")
		return err == nil && title == "123456" && subtitle == ""
	})
}
