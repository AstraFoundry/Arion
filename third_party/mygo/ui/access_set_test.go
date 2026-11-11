package ui_test

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Items of a list box built only where it shows, and of a tree laid out
// flat in one column, tell assistive technology which of how many they
// are (aria-posinset, aria-setsize), which no tree of elements says; a
// Tree's items take the place the app gives them over their own, and a
// List's rows keep theirs.
func TestAccessPositionInSet(t *testing.T) {
	cursor := -1
	files := ui.ListState{Selected: &cursor}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Column(c).Role(ui.RoleListBox).Label("Results").Children(func() {
				for i := 10; i < 13; i++ {
					ui.Row(c).Role(ui.RoleOption).PositionInSet(i+1, 50).Children(func() {
						ui.Text(c, fmt.Sprintf("Result %d", i+1))
					})
				}
			})
			ui.Column(c).Role(ui.RoleTree).Label("Flat").Children(func() {
				ui.Box(c).Role(ui.RoleTreeItem).Label("src").Level(1).PositionInSet(1, 2)
				ui.Box(c).Role(ui.RoleTreeItem).Label("main.go").Level(2).PositionInSet(1, 1)
				ui.Box(c).Role(ui.RoleTreeItem).Label("go.mod").Level(1).PositionInSet(2, 2)
				// Not given, or given in part.
				ui.Box(c).Role(ui.RoleTreeItem).Label("none").PositionInSet(0, -3)
				ui.Box(c).Role(ui.RoleTreeItem).Label("size only").PositionInSet(-1, 7)
			})
			ui.Tree(c, func() {
				ui.TreeItem(c, "a", nil, nil)
				ui.TreeItem(c, "b", nil, nil).PositionInSet(40, 0)
				ui.TreeItem(c, "c", nil, nil).PositionInSet(0, 100)
			})
			ui.List(c, &files, 30, func(i int) {
				ui.Text(c, fmt.Sprintf("File %d", i)).PositionInSet(1, 1)
			}).Height(100)
		})
	}, 400, 800)

	for i := 10; i < 13; i++ {
		name := fmt.Sprintf("Result %d", i+1)
		if o := accessOf(t, tt, ui.RoleOption, name); o.PosInSet != i+1 || o.SetSize != 50 {
			t.Errorf("%s is %d of %d, want %d of 50", name, o.PosInSet, o.SetSize, i+1)
		}
	}
	for _, c := range []struct {
		name      string
		pos, size int
	}{
		{"src", 1, 2}, {"main.go", 1, 1}, {"go.mod", 2, 2}, {"none", 0, 0}, {"size only", 0, 7},
		// The Tree's own place, but where the app gives one.
		{"a", 1, 3}, {"b", 40, 3}, {"c", 3, 100},
	} {
		if it := accessOf(t, tt, ui.RoleTreeItem, c.name); it.PosInSet != c.pos || it.SetSize != c.size {
			t.Errorf("%s is %d of %d, want %d of %d", c.name, it.PosInSet, it.SetSize, c.pos, c.size)
		}
	}
	if r := accessOf(t, tt, ui.RoleListItem, "File 2"); r.PosInSet != 3 || r.SetSize != 30 {
		t.Errorf("a List's row is %d of %d, want 3 of 30, the List's", r.PosInSet, r.SetSize)
	}
}

// AccessSelected tells assistive technology that a tab, an option, a row,
// a cell or an item of a tree of the app's own is selected, as Selected
// does, without painting it in the accent color: a row or a cell saying
// so is chosen among others, in the table or grid holding the choice.
func TestAccessSelected(t *testing.T) {
	tab, pick := 1, 2
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Children(func() {
			ui.Row(c).Role(ui.RoleTabList).Label("Views").Children(func() {
				for i, name := range []string{"List", "Board", "Calendar"} {
					v := ui.ButtonBase(c).Role(ui.RoleTab).Label(name).Padding(4).AccessSelected(i == tab)
					if v.Clicked() {
						tab = i
					}
					v.Children(func() { ui.Text(c, name) })
				}
			})
			ui.Column(c).Role(ui.RoleListBox).Label("Fruit").Children(func() {
				for i, name := range []string{"Apple", "Pear", "Plum"} {
					ui.Row(c).Role(ui.RoleOption).Label(name).Height(24).AccessSelected(i == pick).Children(func() {
						ui.Text(c, name)
						if i == pick {
							ui.Text(c, "✓")
						}
					})
				}
			})
			ui.Column(c).Role(ui.RoleGrid).Label("Sheet").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Label("Mine").Height(24).AccessSelected(true).Children(func() {
					ui.Box(c).Role(ui.RoleCell).Label("A1").AccessSelected(false)
					ui.Box(c).Role(ui.RoleCell).Label("B1").AccessSelected(true)
				})
				ui.Row(c).Role(ui.RoleRow).Label("Painted").Height(24).Selected(true)
				ui.Row(c).Role(ui.RoleRow).Label("Neither").Height(24)
			})
			ui.Column(c).Role(ui.RoleTree).Label("Files").Children(func() {
				ui.Box(c).Role(ui.RoleTreeItem).Label("src").AccessSelected(true)
			})
		})
	}, 400, 600)

	for i, name := range []string{"List", "Board", "Calendar"} {
		if v := accessOf(t, tt, ui.RoleTab, name); v.Selected != (i == tab) || v.Checked {
			t.Errorf("tab %s: %+v", name, v)
		}
	}
	for i, name := range []string{"Apple", "Pear", "Plum"} {
		if o := accessOf(t, tt, ui.RoleOption, name); o.Selected != (i == pick) || !o.Selectable || o.Checked {
			t.Errorf("option %s: %+v", name, o)
		}
	}
	if g := accessOf(t, tt, ui.RoleGrid, "Sheet"); !g.Selectable {
		t.Errorf("the grid holds no choice: %+v", g)
	}
	for _, c := range []struct {
		role               ui.Role
		name               string
		selectable, chosen bool
	}{
		{ui.RoleRow, "Mine", true, true},
		{ui.RoleRow, "Painted", true, true},
		{ui.RoleRow, "Neither", false, false},
		{ui.RoleCell, "A1", true, false},
		{ui.RoleCell, "B1", true, true},
		{ui.RoleTreeItem, "src", false, true},
	} {
		if a := accessOf(t, tt, c.role, c.name); a.Selectable != c.selectable || a.Selected != c.chosen || a.Checked {
			t.Errorf("%s: %+v", c.name, a)
		}
	}

	// Selected paints its row in the accent color, AccessSelected not.
	img := tt.Image()
	scale := float32(img.Bounds().Dx()) / 400
	at := func(name string) [4]uint8 {
		a := accessOf(t, tt, ui.RoleRow, name)
		x, y := int((a.Bounds.X+a.Bounds.W-2)*scale), int((a.Bounds.Y+a.Bounds.H/2)*scale)
		p := img.RGBAAt(x, y)
		return [4]uint8{p.R, p.G, p.B, p.A}
	}
	if mine, painted, neither := at("Mine"), at("Painted"), at("Neither"); mine != neither || mine == painted {
		t.Errorf("rows painted %v (AccessSelected), %v (Selected), %v (neither)", mine, painted, neither)
	}

	// Choosing another tab moves the state.
	if err := tt.Click("Calendar"); err != nil {
		t.Fatal(err)
	}
	if v := accessOf(t, tt, ui.RoleTab, "Calendar"); !v.Selected {
		t.Errorf("Calendar after its click: %+v", v)
	}
	if v := accessOf(t, tt, ui.RoleTab, "Board"); v.Selected {
		t.Errorf("Board after Calendar's click: %+v", v)
	}
}
