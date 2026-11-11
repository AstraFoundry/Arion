//go:build linux && (amd64 || arm64)

package e2e

import (
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// TestAccessListBox reads, as Orca does through ATK, a list box built by
// hand (RoleListBox, RoleOption) and a SelectBase's popup: list boxes of
// selectable list items, the option chosen or highlighted selected.
// Focusing an option of the select highlights it without choosing it,
// and pressing one chooses it (#1172). Options of a list box built only
// where it shows say which of all they are (posinset, setsize), and
// options and tabs showing their choice their own way are selected
// (AccessSelected, #1181).
func TestAccessListBox(t *testing.T) {
	var frames atomic.Int32
	size, own := "b", 0
	result := func(i int) string { return "Result " + strconv.Itoa(i) }
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleListBox).Label("People").Children(func() {
				for i, name := range []string{"Ada", "Grace"} {
					opt := ui.Row(c).Role(ui.RoleOption).Label(name).Checked(i == own)
					if opt.Clicked() {
						own = i
					}
					opt.Children(func() { ui.Text(c, name) })
				}
			})
			ui.Column(c).Role(ui.RoleListBox).Label("Results").Children(func() {
				for i := 11; i <= 13; i++ {
					ui.Row(c).Role(ui.RoleOption).Label(result(i)).PositionInSet(i, 50).AccessSelected(i == 12).
						Children(func() { ui.Text(c, result(i)) })
				}
			})
			ui.Row(c).Role(ui.RoleTabList).Label("Views").Gap(8).Children(func() {
				for i, name := range []string{"List", "Board"} {
					ui.ButtonBase(c).Role(ui.RoleTab).Label(name).AccessSelected(i == 1).
						Children(func() { ui.Text(c, name) })
				}
			})
			sel := ui.SelectBase(c, &size)
			sel.Trigger.Label("Size").Padding(4).Children(func() { ui.Text(c, "Size "+size) })
			sel.Popup(func(panel ui.Element) {
				panel.Label("Sizes")
				for _, v := range []string{"a", "b", "c"} {
					sel.Item(v).Label("size " + v).Padding(4).Children(func() { ui.Text(c, v) })
				}
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "List boxes", Width: 400, Height: 500, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	// find returns the first node labeled label: an option, not its text.
	find := func(label string) (n linux.TestAccessNode, ok bool) {
		mygo.RunOnMain(func() {
			nodes, _ := linux.TestAccessibility(w.NativeHandle())
			for _, node := range nodes {
				if node.Label == label && !ok {
					n, ok = node, true
				}
			}
		})
		return n, ok
	}
	option := func(label string, selected bool) func() bool {
		return func() bool {
			n, ok := find(label)
			return ok && n.Role == "list item" && n.Selectable && n.Selected == selected
		}
	}
	chosen := func() (s string, o int) {
		mygo.RunOnMain(func() { s, o = size, own })
		return s, o
	}

	eventually(t, "the list box built by hand", func() bool {
		n, ok := find("People")
		return ok && n.Role == "list box"
	})
	eventually(t, "Result 12 of 50 selected", func() bool {
		n, ok := find("Result 12")
		return ok && n.Role == "list item" && n.Selectable && n.Selected && n.Attributes["posinset"] == "12" && n.Attributes["setsize"] == "50"
	})
	if n, _ := find("Result 11"); n.Selected || n.Attributes["posinset"] != "11" || n.Attributes["setsize"] != "50" {
		t.Errorf("Result 11: %+v", n)
	}
	if n, _ := find("Board"); n.Role != "page tab" || !n.Selected {
		t.Errorf("the tab chosen: %+v", n)
	}
	if n, _ := find("List"); n.Role != "page tab" || n.Selected {
		t.Errorf("the tab not chosen: %+v", n)
	}
	eventually(t, "Ada selected", option("Ada", true))
	eventually(t, "Grace not selected", option("Grace", false))
	if !accessPerform(w, "Grace", "press", "") {
		t.Fatal("cannot press Grace")
	}
	eventually(t, "Grace chosen", func() bool { _, o := chosen(); return o == 1 })
	eventually(t, "Grace selected", option("Grace", true))

	if !accessPerform(w, "Size", "press", "") {
		t.Fatal("cannot press the select")
	}
	eventually(t, "the select's list box", func() bool {
		n, ok := find("Sizes")
		return ok && n.Role == "list box"
	})
	eventually(t, "the choice selected", option("size b", true))
	eventually(t, "size a not selected", option("size a", false))
	// Focusing an option highlights it, and chooses nothing.
	if !accessPerform(w, "size c", "focus", "") {
		t.Fatal("cannot focus size c")
	}
	eventually(t, "size c selected", option("size c", true))
	if s, _ := chosen(); s != "b" {
		t.Errorf("focusing size c chose %q", s)
	}
	if _, open := find("size a"); !open {
		t.Error("focusing size c closed the popup")
	}
	if !accessPerform(w, "size a", "press", "") {
		t.Fatal("cannot press size a")
	}
	eventually(t, "size a chosen", func() bool { s, _ := chosen(); return s == "a" })
	eventually(t, "the popup closed", func() bool { _, open := find("size a"); return !open })
}
