package ui

import (
	"fmt"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// menuApp is a view with a row whose menu is drawn in the window, a
// button that takes the focus, and what its items did.
type menuApp struct {
	chosen []string
	pinned bool
	sort   string
	hidden bool // the row is not built
}

func (a *menuApp) view(c *context) {
	coreColumn(c).Fill().Padding(20).Gap(10).Children(func() {
		coreButton(c, "Edit")
		if a.hidden {
			return
		}
		row := coreRow(c).Padding(8).Focusable().Label("Report").Children(func() {
			coreText(c, "Report.pdf")
		})
		row.PopoverMenu(func(m *Menu) {
			if m.Item("Open").Shortcut(Cmd, KeyO).Chosen() {
				a.chosen = append(a.chosen, "Open")
			}
			if m.Item("Pinned").Checked(a.pinned).Chosen() {
				a.pinned = !a.pinned
			}
			m.Separator()
			m.Submenu("Move to", func(m *Menu) {
				for _, f := range []string{"Desktop", "Documents"} {
					if m.Item(f).Chosen() {
						a.chosen = append(a.chosen, "Move to "+f)
					}
				}
			})
			m.Submenu("Sort by", func(m *Menu) {
				for _, by := range []string{"Name", "Date"} {
					if m.Item(by).Radio(a.sort == by).Chosen() {
						a.sort = by
					}
				}
			})
			if m.Item("Delete").Disabled(true).Chosen() {
				a.chosen = append(a.chosen, "Delete")
			}
		})
	})
}

// menuShows reports whether the drawn menu shows.
func menuShows(tt *Tester) bool { return tt.HasText("Open") }

// accessFocus returns the node assistive technology reads as focused.
func accessFocus(t *testing.T, tt *Tester) platform.AccessNode {
	t.Helper()
	tree := tt.h.access
	if tree == nil {
		t.Fatal("no access tree")
	}
	n, ok := byID(tree, tree.Focus)
	if !ok {
		t.Fatalf("the focus %d is no node of the tree", tree.Focus)
	}
	return n
}

// wantFocus checks that the item labeled label of a menu has the focus,
// as assistive technology reads it.
func wantFocus(t *testing.T, tt *Tester, label string) {
	t.Helper()
	n := accessFocus(t, tt)
	if n.Label != label || n.Role < platform.RoleMenuItem || n.Role > platform.RoleMenuItemRadio {
		t.Fatalf("focus on %q (role %d), want the menu item %q", n.Label, n.Role, label)
	}
}

func TestPopoverMenuOpensAtThePointer(t *testing.T) {
	a := &menuApp{}
	tt := coreNewTester(a.view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	x, y := r.X+10, r.Y+r.H/2
	tt.RightClickAt(x, y)
	if !menuShows(tt) {
		t.Fatalf("no menu after a right-click: texts %q", tt.Texts())
	}
	if tt.h.menu != nil {
		t.Errorf("the system shows a menu too: %v", tt.h.menu.Items)
	}
	open, _ := tt.Find("Open")
	if open.X < x || open.Y < y || open.X > x+20 || open.Y > y+20 {
		t.Errorf("Open at %+v, not below and after the click at (%v, %v)", open, x, y)
	}
	if s := tt.rt.states[tt.rt.focused]; s == nil || s.role != RoleMenu {
		t.Errorf("the focus is not on the menu as it opens")
	}
	// The press opening the menu was outside it, and left it open; frames
	// after do not close it.
	tt.Frame()
	if !menuShows(tt) {
		t.Fatal("the menu closed after it opened")
	}
	for _, s := range []string{"Pinned", "Move to", "Delete"} {
		if !tt.HasText(s) {
			t.Errorf("no item %q", s)
		}
	}
	keys := "Ctrl+O"
	if runtime.GOOS == "darwin" {
		keys = "⌘O"
	}
	if !tt.HasText(keys) {
		t.Errorf("Open does not show its shortcut %q: %q", keys, tt.Texts())
	}

	// A right-click elsewhere on the row opens it there.
	tt.RightClickAt(x+250, y)
	if open, _ := tt.Find("Open"); !menuShows(tt) || open.X+open.W > x+250 {
		t.Errorf("a right-click at %v puts the menu at %+v", x+250, open)
	}
}

// TestPopoverMenuInAScrolledElement checks the menu of an element
// scrolled partly out of view opens at the pointer, as the menu attaches
// to the element's whole box (#894).
func TestPopoverMenuInAScrolledElement(t *testing.T) {
	view := func(c *context) {
		coreScroll(c).Fill().Children(func() {
			col := coreColumn(c).FillWidth().Gap(10).Children(func() {
				for i := range 60 {
					coreText(c, fmt.Sprintf("Line %d", i))
				}
			})
			col.PopoverMenu(func(m *Menu) {
				m.Item("Open")
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	tt.Scroll(200, 150, 0, 400)
	tt.Frame()
	x, y := float32(100), float32(200)
	tt.RightClickAt(x, y)
	open, ok := tt.Find("Open")
	if !ok {
		t.Fatalf("no menu: %q", tt.Texts())
	}
	if open.X < x || open.Y < y || open.X > x+20 || open.Y > y+20 {
		t.Errorf("Open at %+v, not below and after the click at (%v, %v)", open, x, y)
	}
}

func TestPopoverMenuFlipsAtTheWindowsEdges(t *testing.T) {
	a := &menuApp{}
	view := func(c *context) {
		coreColumn(c).Fill().Justify(End).Padding(4).Children(func() {
			coreRow(c).Padding(8).Children(func() { coreText(c, "Report.pdf") }).PopoverMenu(func(m *Menu) {
				m.Item("Open")
				m.Item("Rename")
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	x, y := float32(390), r.Y+r.H/2
	tt.RightClickAt(x, y)
	open, ok := tt.Find("Open")
	if !ok {
		t.Fatal("no menu")
	}
	if open.X+open.W > x || open.Y > y {
		t.Errorf("Open at %+v, not before and above the click at (%v, %v) in the window's corner", open, x, y)
	}

	tt = coreNewTester(a.view, 400, 300)
	tt.SetDirection(RightToLeft)
	r, _ = tt.Find("Report.pdf")
	x, y = r.X+r.W-10, r.Y+r.H/2
	tt.RightClickAt(x, y)
	open, _ = tt.Find("Open")
	if open.X+open.W > x || open.Y < y {
		t.Errorf("right to left, Open at %+v is not below and before the click at (%v, %v)", open, x, y)
	}
}

func TestPopoverMenuKeyboard(t *testing.T) {
	a := &menuApp{}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Report.pdf")
	if !tt.Focused("Report.pdf") {
		t.Fatal("the row did not take the focus")
	}
	tt.Key(0, KeyContextMenu)
	if !menuShows(tt) {
		t.Fatal("the menu key opened no menu")
	}
	r, _ := tt.Find("Report.pdf")
	if open, _ := tt.Find("Open"); open.Y < r.Y || open.Y > r.Y+r.H+20 {
		t.Errorf("Open at %+v, not below the row's top at %+v", open, r)
	}
	wantFocus(t, tt, "Open")
	steps := []struct {
		mods Modifiers
		key  Key
		want string
	}{
		{0, KeyDown, "Pinned"},
		{0, KeyDown, "Move to"}, // past the separator
		{0, KeyDown, "Sort by"},
		{0, KeyDown, "Open"}, // past Delete, disabled, and round
		{0, KeyUp, "Sort by"},
		{0, KeyHome, "Open"},
		{0, KeyEnd, "Sort by"},
		{0, KeyRight, "Name"}, // into the submenu
		{0, KeyDown, "Date"},
		{0, KeyLeft, "Sort by"}, // out of it
		{0, KeyUp, "Move to"},
		{0, KeyEnter, "Desktop"}, // into the submenu
		{0, KeyEscape, "Move to"},
	}
	for i, s := range steps {
		tt.Key(s.mods, s.key)
		if !menuShows(tt) {
			t.Fatalf("step %d: the menu closed", i)
		}
		wantFocus(t, tt, s.want)
	}
	if tt.HasText("Desktop") {
		t.Error("Escape left the submenu open")
	}

	// Typing the first letters of an item goes to it.
	tt.TypeKey(0, KeyP, "p")
	wantFocus(t, tt, "Pinned")
	tt.Advance(2 * time.Second)
	tt.TypeKey(0, KeyS, "s")
	wantFocus(t, tt, "Sort by")
	tt.Advance(2 * time.Second)
	tt.TypeKey(0, KeyM, "m")
	tt.TypeKey(0, KeyO, "o")
	wantFocus(t, tt, "Move to")
	tt.Advance(2 * time.Second)
	tt.TypeKey(0, KeyD, "d") // Delete is disabled
	wantFocus(t, tt, "Move to")

	// Enter in the submenu chooses, and the focus goes back to the row.
	tt.Key(0, KeyRight)
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Documents")
	tt.Key(0, KeyEnter)
	if menuShows(tt) {
		t.Error("the menu stays after a choice")
	}
	if !slices.Equal(a.chosen, []string{"Move to Documents"}) {
		t.Errorf("chose %q", a.chosen)
	}
	if !tt.Focused("Report.pdf") {
		t.Error("the focus did not go back to the row")
	}

	// Space chooses too, and Shift+F10 opens it.
	tt.Key(Shift, KeyF10)
	wantFocus(t, tt, "Open")
	tt.Key(0, KeySpace)
	if !slices.Equal(a.chosen, []string{"Move to Documents", "Open"}) || menuShows(tt) {
		t.Errorf("Space: chose %q, menu shows %v", a.chosen, menuShows(tt))
	}

	// Escape closes the menu, and Tab too.
	for _, k := range []Key{KeyEscape, KeyTab} {
		tt.Key(0, KeyContextMenu)
		tt.Key(0, k)
		if menuShows(tt) {
			t.Errorf("key %d left the menu open", k)
		}
		if !tt.Focused("Report.pdf") {
			t.Errorf("key %d: the focus did not go back to the row", k)
		}
	}
}

func TestPopoverMenuRightToLeftKeys(t *testing.T) {
	a := &menuApp{}
	tt := coreNewTester(a.view, 400, 400)
	tt.SetDirection(RightToLeft)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Report.pdf")
	tt.Key(0, KeyContextMenu)
	tt.Key(0, KeyEnd)
	tt.Key(0, KeyRight) // not into the submenu, which opens to the left
	wantFocus(t, tt, "Sort by")
	tt.Key(0, KeyLeft)
	wantFocus(t, tt, "Name")
	sort, _ := tt.Find("Sort by")
	if name, _ := tt.Find("Name"); name.X+name.W > sort.X {
		t.Errorf("the submenu at %+v is not left of its item at %+v", name, sort)
	}
	tt.Key(0, KeyRight)
	wantFocus(t, tt, "Sort by")
}

func TestPopoverMenuPointer(t *testing.T) {
	a := &menuApp{}
	tt := coreNewTester(a.view, 400, 400)
	tt.RightClick("Report.pdf")
	if err := tt.Click("Pinned"); err != nil {
		t.Fatal(err)
	}
	if !a.pinned || menuShows(tt) {
		t.Errorf("clicking Pinned: pinned %v, menu shows %v", a.pinned, menuShows(tt))
	}

	// A disabled item takes no click, and the menu stays.
	tt.RightClick("Report.pdf")
	tt.Click("Delete")
	if len(a.chosen) != 0 || !menuShows(tt) {
		t.Errorf("clicking Delete, disabled: chose %q, menu shows %v", a.chosen, menuShows(tt))
	}

	// The pointer opens a submenu as it moves onto its item.
	r, _ := tt.Find("Sort by")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if !tt.HasText("Name") {
		t.Fatal("the submenu did not open under the pointer")
	}
	if err := tt.Click("Date"); err != nil {
		t.Fatal(err)
	}
	if a.sort != "Date" || menuShows(tt) {
		t.Errorf("clicking Date: sort %q, menu shows %v", a.sort, menuShows(tt))
	}
	// Moving onto another item closes the submenu.
	tt.RightClick("Report.pdf")
	r, _ = tt.Find("Move to")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	r, _ = tt.Find("Open")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if tt.HasText("Desktop") {
		t.Error("the submenu stays as the pointer moves to another item")
	}
}

func TestPopoverMenuCloses(t *testing.T) {
	a := &menuApp{}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Edit")
	if !tt.Focused("Edit") {
		t.Fatal("the button did not take the focus")
	}
	tt.RightClick("Report.pdf")
	if tt.Focused("Edit") || !menuShows(tt) {
		t.Fatal("the menu did not take the focus")
	}
	tt.Key(0, KeyEscape)
	if menuShows(tt) {
		t.Fatal("Escape left the menu open")
	}
	if !tt.Focused("Edit") {
		t.Error("the focus did not go back to the button it came from")
	}

	// A press outside closes it, as one on the row does.
	tt.RightClick("Report.pdf")
	tt.ClickAt(390, 390)
	if menuShows(tt) {
		t.Error("a press outside left the menu open")
	}
	tt.RightClick("Report.pdf")
	r, _ := tt.Find("Report.pdf")
	tt.ClickAt(r.X+1, r.Y+1)
	if menuShows(tt) {
		t.Error("a press on the row left the menu open")
	}
	// A press in a submenu does not.
	tt.RightClick("Report.pdf")
	r, _ = tt.Find("Move to")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	r, _ = tt.Find("Desktop")
	tt.Press(r.X+1, r.Y+1)
	if !menuShows(tt) || !tt.HasText("Desktop") {
		t.Error("a press in the submenu closed the menu")
	}
	tt.Release(r.X+1, r.Y+1)

	// The menu goes with its element.
	tt.RightClick("Report.pdf")
	a.hidden = true
	tt.Frame()
	if menuShows(tt) || tt.rt.menu.drawn != nil {
		t.Error("the menu stays without its element")
	}
}

func TestPopoverMenuAccessibility(t *testing.T) {
	a := &menuApp{sort: "Date", pinned: true}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.RightClick("Report.pdf")
	tree := tt.h.access
	menu := accessNode(t, tree, platform.RoleMenu, "")
	if f := accessFocus(t, tt); f.ID != menu.ID {
		t.Errorf("focus on %q, not the menu, which the pointer opened", f.Label)
	}
	open := accessNode(t, tree, platform.RoleMenuItem, "Open")
	if open.Label != "Open" || open.KeyShortcuts == "" || open.PosInSet != 1 || open.SetSize != 5 {
		t.Errorf("Open: %+v", open)
	}
	if n := tree.Nodes[open.Parent]; n.ID != menu.ID {
		t.Errorf("Open is in %q, not the menu", n.Label)
	}
	pinned := accessNode(t, tree, platform.RoleMenuItemCheckBox, "Pinned")
	if pinned.States&platform.AccessChecked == 0 {
		t.Error("Pinned is not checked")
	}
	if del := accessNode(t, tree, platform.RoleMenuItem, "Delete"); del.States&platform.AccessDisabled == 0 {
		t.Error("Delete is not disabled")
	}
	sortBy := accessNode(t, tree, platform.RoleMenuItem, "Sort by")
	if sortBy.States&platform.AccessExpandable == 0 || sortBy.States&platform.AccessExpanded != 0 {
		t.Errorf("Sort by, closed: states %b", sortBy.States)
	}
	tt.Key(0, KeyEnd)
	tt.Key(0, KeyRight)
	tree = tt.h.access
	if n := accessNode(t, tree, platform.RoleMenuItem, "Sort by"); n.States&platform.AccessExpanded == 0 {
		t.Error("Sort by is not expanded with its submenu open")
	}
	accessNode(t, tree, platform.RoleMenu, "Sort by")
	name := accessNode(t, tree, platform.RoleMenuItemRadio, "Name")
	date := accessNode(t, tree, platform.RoleMenuItemRadio, "Date")
	if name.States&platform.AccessChecked != 0 || date.States&platform.AccessChecked == 0 {
		t.Errorf("radio items: Name %b, Date %b", name.States, date.States)
	}
	wantFocus(t, tt, "Name")
}

func TestPopoverMenuEditItems(t *testing.T) {
	value := "hello"
	cleared := false
	view := func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			in := coreTextInput(c, &value).Label("Name")
			in.PopoverMenu(func(m *Menu) {
				if m.Item("Clear").Chosen() {
					cleared = true
				}
				m.Separator()
				m.EditItems()
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	tt.RightClick("Name")
	if !tt.HasText("Select All") || !tt.HasText("Clear") {
		t.Fatalf("no editing items: %q", tt.Texts())
	}
	if err := tt.Click("Select All"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("Name")
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "hello" || cleared {
		t.Errorf("Select All, Copy: clipboard %q, cleared %v", tt.Clipboard(), cleared)
	}
}

// TestRadioMenuItems checks that Radio makes items of the system's menus
// radio items.
func TestRadioMenuItems(t *testing.T) {
	sort := "Date"
	view := func(c *context) {
		coreText(c, "Files").ContextMenu(func(m *Menu) {
			for _, by := range []string{"Name", "Date"} {
				if m.Item(by).Radio(sort == by).Chosen() {
					sort = by
				}
			}
		})
	}
	tt := coreNewTester(view, 300, 200)
	tt.RightClick("Files")
	items := tt.h.menu.Items
	if items[0].Type != platform.MenuItemRadio || items[0].Checked || !items[1].Checked {
		t.Errorf("items %+v, %+v", items[0], items[1])
	}
	if err := tt.ChooseMenuItem("Name"); err != nil || sort != "Name" {
		t.Errorf("chose Name: %v, sort %q", err, sort)
	}
}

// TestOverlayOpenedByAPress checks that the press opening an overlay is
// not outside it, as a right-click opening a menu of one's own.
func TestOverlayOpenedByAPress(t *testing.T) {
	open := false
	view := func(c *context) {
		coreColumn(c).Fill().Children(func() {
			if coreBox(c).Size(80, 30).Label("Card").RightClicked() {
				open = true
			}
		})
		if open {
			coreOverlay(c, func() {
				panel := coreText(c, "Panel").Absolute().Left(200).Top(100)
				if panel.PressedOutside() {
					open = false
				}
			})
		}
	}
	tt := coreNewTester(view, 400, 300)
	// Both in one frame, as a quick click comes.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: 10, Y: 10, Button: 1, Clicks: 1})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: 10, Y: 10, Button: 1})
	tt.settle()
	if !open || !tt.HasText("Panel") {
		t.Fatal("the press opening the overlay closed it")
	}
	tt.ClickAt(300, 250)
	if open {
		t.Error("a press outside left the overlay open")
	}
}

// TestPopoverMenuInAList checks a menu of a list's rows, which the list
// builds as it lays out.
func TestPopoverMenuInAList(t *testing.T) {
	var s ListState
	selected := 0
	s.Selected = &selected
	opened := -1
	view := func(c *context) {
		coreList(c, &s, 50, func(i int) {
			row := coreRow(c).Height(28).Children(func() { coreText(c, "Row "+string(rune('A'+i%26))) })
			row.PopoverMenu(func(m *Menu) {
				if m.Item("Open row").Chosen() {
					opened = i
				}
			})
		}).Grow(1)
	}
	tt := coreNewTester(view, 300, 300)
	tt.RightClick("Row C")
	if err := tt.Click("Open row"); err != nil {
		t.Fatal(err)
	}
	if opened != 2 || tt.HasText("Open row") {
		t.Errorf("opened row %d, menu shows %v", opened, tt.HasText("Open row"))
	}
}

// buttonMenuApp is a view with a menu button whose menu is drawn in the
// window, a button that takes the focus, and what its items did.
type buttonMenuApp struct {
	chosen   []string
	pinned   bool
	sort     string
	disabled bool
	// bottom puts the menu button at the window's bottom, and end at its
	// end.
	bottom, end bool
}

func (a *buttonMenuApp) view(c *context) {
	col := coreColumn(c).Fill().Padding(20).Gap(10).AlignItems(Start)
	if a.bottom {
		col.Justify(End)
	}
	if a.end {
		col.AlignItems(End)
	}
	col.Children(func() {
		coreButton(c, "Edit")
		corePopoverMenuButton(c, "Actions", func(m *Menu) {
			if m.Item("Open").Shortcut(Cmd, KeyO).Chosen() {
				a.chosen = append(a.chosen, "Open")
			}
			if m.Item("Pinned").Checked(a.pinned).Chosen() {
				a.pinned = !a.pinned
			}
			m.Separator()
			m.Submenu("Move to", func(m *Menu) {
				for _, f := range []string{"Desktop", "Documents"} {
					if m.Item(f).Chosen() {
						a.chosen = append(a.chosen, "Move to "+f)
					}
				}
			})
			m.Submenu("Sort by", func(m *Menu) {
				for _, by := range []string{"Name", "Date"} {
					if m.Item(by).Radio(a.sort == by).Chosen() {
						a.sort = by
					}
				}
			})
			if m.Item("Delete").Disabled(true).Chosen() {
				a.chosen = append(a.chosen, "Delete")
			}
		}).Label("Actions").Disabled(a.disabled)
	})
}

func TestPopoverMenuButtonOpensBelow(t *testing.T) {
	a := &buttonMenuApp{}
	tt := coreNewTester(a.view, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	// The menu opens as the button goes down, below it.
	b, _ := tt.Find("Actions")
	tt.Press(b.X+b.W/2, b.Y+b.H/2)
	if !menuShows(tt) {
		t.Fatalf("no menu as the button goes down: texts %q", tt.Texts())
	}
	if tt.h.menu != nil {
		t.Errorf("the system shows a menu too: %v", tt.h.menu.Items)
	}
	open, _ := tt.Find("Open")
	if open.Y < b.Y+b.H || open.Y > b.Y+b.H+20 || open.X < b.X || open.X > b.X+20 {
		t.Errorf("Open at %+v, not below the button at %+v, at its start", open, b)
	}
	if s := tt.rt.states[tt.rt.focused]; s == nil || s.role != RoleMenu {
		t.Errorf("the focus is not on the menu as it opens")
	}
	if n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions"); n.States&platform.AccessExpanded == 0 {
		t.Errorf("the button is not expanded with its menu open: states %b", n.States)
	}
	// The release, and frames after, leave it open.
	tt.Release(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
	if !menuShows(tt) {
		t.Fatal("the menu closed after it opened")
	}
	// The button pressed again closes it, giving the focus back.
	tt.Click("Actions")
	if menuShows(tt) {
		t.Error("pressing the button again left the menu open")
	}
	if !tt.Focused("Actions") {
		t.Error("the focus did not go back to the button")
	}
	if n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions"); n.States&platform.AccessExpanded != 0 {
		t.Error("the button is expanded with its menu closed")
	}
	tt.Click("Actions")
	if !menuShows(tt) {
		t.Error("the button does not open its menu once closed")
	}

	// A disabled button opens none.
	tt.Key(0, KeyEscape)
	a.disabled = true
	tt.Frame()
	tt.Click("Actions")
	if menuShows(tt) {
		t.Error("a disabled menu button opens its menu")
	}
}

func TestPopoverMenuButtonFlipsAtTheWindowsEdges(t *testing.T) {
	// No room below the button: the menu goes above it.
	a := &buttonMenuApp{bottom: true}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("Actions")
	b, _ := tt.Find("Actions")
	open, ok := tt.Find("Open")
	if !ok {
		t.Fatal("no menu")
	}
	del, _ := tt.Find("Delete")
	if del.Y+del.H > b.Y || open.Y < 0 {
		t.Errorf("Open at %+v, Delete at %+v: not above the button at %+v at the window's bottom", open, del, b)
	}

	// Right to left, it lines up with the button's right.
	a = &buttonMenuApp{}
	tt = coreNewTester(a.view, 400, 300)
	tt.SetDirection(RightToLeft)
	tt.Click("Actions")
	b, _ = tt.Find("Actions")
	open, _ = tt.Find("Open")
	if open.Y < b.Y+b.H || open.X+open.W > b.X+b.W || open.X+open.W < b.X+b.W-20 {
		t.Errorf("right to left, Open at %+v is not below the button at %+v, at its right", open, b)
	}

	// At the window's end, it moves along the button into the window.
	a = &buttonMenuApp{end: true}
	tt = coreNewTester(a.view, 400, 300)
	tt.Click("Actions")
	b, _ = tt.Find("Actions")
	open, _ = tt.Find("Open")
	if open.Y < b.Y+b.H || open.X+open.W > 400 {
		t.Errorf("at the window's end, Open at %+v is not below the button at %+v, in the window", open, b)
	}
}

func TestPopoverMenuButtonKeyboard(t *testing.T) {
	a := &buttonMenuApp{}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Edit")
	tt.Key(0, KeyTab)
	if !tt.Focused("Actions") {
		t.Fatal("the menu button did not take the focus")
	}
	// Enter, Space and Down open it, on its first item, and Escape closes
	// it, giving the focus back.
	for _, k := range []Key{KeyEnter, KeySpace, KeyDown} {
		tt.Key(0, k)
		if !menuShows(tt) {
			t.Fatalf("key %d opened no menu", k)
		}
		wantFocus(t, tt, "Open")
		tt.Key(0, KeyEscape)
		if menuShows(tt) {
			t.Fatalf("Escape left the menu open")
		}
		if !tt.Focused("Actions") {
			t.Fatalf("key %d: the focus did not go back to the button", k)
		}
	}
	tt.Key(0, KeyDown)
	b, _ := tt.Find("Actions")
	if open, _ := tt.Find("Open"); open.Y < b.Y+b.H {
		t.Errorf("Open at %+v, not below the button at %+v", open, b)
	}
	steps := []struct {
		key  Key
		want string
	}{
		{KeyDown, "Pinned"},
		{KeyDown, "Move to"}, // past the separator
		{KeyDown, "Sort by"},
		{KeyDown, "Open"}, // past Delete, disabled, and round
		{KeyUp, "Sort by"},
		{KeyHome, "Open"},
		{KeyEnd, "Sort by"},
		{KeyRight, "Name"}, // into the submenu
		{KeyDown, "Date"},
		{KeyLeft, "Sort by"}, // out of it
		{KeyUp, "Move to"},
		{KeyEnter, "Desktop"}, // into the submenu
		{KeyEscape, "Move to"},
	}
	for i, s := range steps {
		tt.Key(0, s.key)
		if !menuShows(tt) {
			t.Fatalf("step %d: the menu closed", i)
		}
		wantFocus(t, tt, s.want)
	}

	// Typing the first letters of an item goes to it.
	tt.TypeKey(0, KeyP, "p")
	wantFocus(t, tt, "Pinned")
	tt.Advance(2 * time.Second)
	tt.TypeKey(0, KeyS, "s")
	wantFocus(t, tt, "Sort by")
	tt.Advance(2 * time.Second)
	tt.TypeKey(0, KeyM, "m")
	tt.TypeKey(0, KeyO, "o")
	wantFocus(t, tt, "Move to")

	// Enter in the submenu chooses, and the focus goes back to the button.
	tt.Key(0, KeyRight)
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Documents")
	tt.Key(0, KeyEnter)
	if menuShows(tt) {
		t.Error("the menu stays after a choice")
	}
	if !slices.Equal(a.chosen, []string{"Move to Documents"}) {
		t.Errorf("chose %q", a.chosen)
	}
	if !tt.Focused("Actions") {
		t.Error("the focus did not go back to the button")
	}

	// Space chooses too, and Tab closes it.
	tt.Key(0, KeyEnter)
	tt.Key(0, KeySpace)
	if !slices.Equal(a.chosen, []string{"Move to Documents", "Open"}) || menuShows(tt) {
		t.Errorf("Space: chose %q, menu shows %v", a.chosen, menuShows(tt))
	}
	tt.Key(0, KeyEnter)
	tt.Key(0, KeyTab)
	if menuShows(tt) || !tt.Focused("Actions") {
		t.Errorf("Tab: menu shows %v, button focused %v", menuShows(tt), tt.Focused("Actions"))
	}
}

// TestPopoverMenuButtonKeysInOneFrame checks keys typed quickly, coming
// in one frame: each goes where the one before left the view, in order.
func TestPopoverMenuButtonKeysInOneFrame(t *testing.T) {
	a := &buttonMenuApp{}
	tt := coreNewTester(a.view, 400, 400)
	keys := func(ks ...Key) {
		for _, k := range ks {
			tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key(k)})
			tt.rt.event(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.Key(k)})
		}
		tt.settle()
	}
	tt.Click("Actions")
	keys(KeyDown, KeyDown, KeyEnter)
	if !a.pinned || menuShows(tt) {
		t.Errorf("Down, Down and Enter: pinned %v, menu shows %v", a.pinned, menuShows(tt))
	}

	// Enter chooses the item highlighted before the Down after it.
	tt.Click("Actions")
	keys(KeyDown)
	keys(KeyEnter, KeyDown)
	if !slices.Equal(a.chosen, []string{"Open"}) || !a.pinned || menuShows(tt) {
		t.Errorf("Enter and Down: chose %q, pinned %v, menu shows %v", a.chosen, a.pinned, menuShows(tt))
	}
}

func TestPopoverMenuButtonPointer(t *testing.T) {
	a := &buttonMenuApp{}
	tt := coreNewTester(a.view, 400, 400)
	tt.Click("Edit")
	tt.Click("Actions")
	if err := tt.Click("Pinned"); err != nil {
		t.Fatal(err)
	}
	if !a.pinned || menuShows(tt) {
		t.Errorf("clicking Pinned: pinned %v, menu shows %v", a.pinned, menuShows(tt))
	}
	if !tt.Focused("Actions") {
		t.Error("the focus did not go back to the button, which the press gave it")
	}

	// A disabled item takes no click, and the menu stays.
	tt.Click("Actions")
	tt.Click("Delete")
	if len(a.chosen) != 0 || !menuShows(tt) {
		t.Errorf("clicking Delete, disabled: chose %q, menu shows %v", a.chosen, menuShows(tt))
	}

	// The pointer opens a submenu as it moves onto its item.
	r, _ := tt.Find("Sort by")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if err := tt.Click("Date"); err != nil {
		t.Fatal(err)
	}
	if a.sort != "Date" || menuShows(tt) {
		t.Errorf("clicking Date: sort %q, menu shows %v", a.sort, menuShows(tt))
	}

	// A press outside closes it, and goes on to what is under the pointer.
	tt.Click("Actions")
	tt.Click("Edit")
	if menuShows(tt) || !tt.Focused("Edit") {
		t.Errorf("a press on Edit: menu shows %v, Edit focused %v", menuShows(tt), tt.Focused("Edit"))
	}
}

func TestPopoverMenuButtonAccessibility(t *testing.T) {
	a := &buttonMenuApp{sort: "Date", pinned: true}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions")
	if n.Actions&platform.ActionPress == 0 || n.States&platform.AccessFocusable == 0 || n.States&platform.AccessExpanded != 0 {
		t.Errorf("the menu button: %+v", n)
	}
	// Pressing it opens the menu, on its first item.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
	if !menuShows(tt) || tt.h.menu != nil {
		t.Fatalf("pressing it: drawn menu %v, system's %v", menuShows(tt), tt.h.menu != nil)
	}
	wantFocus(t, tt, "Open")
	tree := tt.h.access
	menu := accessNode(t, tree, platform.RoleMenu, "")
	open := accessNode(t, tree, platform.RoleMenuItem, "Open")
	if open.PosInSet != 1 || open.SetSize != 5 || tree.Nodes[open.Parent].ID != menu.ID {
		t.Errorf("Open: %+v", open)
	}
	if pinned := accessNode(t, tree, platform.RoleMenuItemCheckBox, "Pinned"); pinned.States&platform.AccessChecked == 0 {
		t.Error("Pinned is not checked")
	}
	if sortBy := accessNode(t, tree, platform.RoleMenuItem, "Sort by"); sortBy.States&platform.AccessExpandable == 0 {
		t.Errorf("Sort by: states %b", sortBy.States)
	}
	if n := accessNode(t, tree, platform.RoleMenuButton, "Actions"); n.States&platform.AccessExpanded == 0 {
		t.Error("the button is not expanded with its menu open")
	}
}

// TestPopoverMenuButtonInAToolbar opens the drawn menu of a menu button
// that does not fit in its toolbar, from the toolbar's overflow menu.
func TestPopoverMenuButtonInAToolbar(t *testing.T) {
	chosen := ""
	tt := coreNewTester(func(c *context) {
		coreToolbar(c, func() {
			for _, n := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
				coreButton(c, n)
			}
			corePopoverMenuButton(c, "Export", func(m *Menu) {
				for _, as := range []string{"PDF", "PNG"} {
					if m.Item("As " + as).Chosen() {
						chosen = as
					}
				}
			})
		}).Fill()
	}, 260, 300)
	if tt.HasText("Export") {
		t.Fatal("the menu button fits")
	}
	tt.Click("More")
	if err := tt.ChooseMenuItem("Export"); err != nil {
		t.Fatal(err)
	}
	more, _ := tt.Find("More")
	png, ok := tt.Find("As PNG")
	if !ok {
		t.Fatalf("no drawn menu: texts %q", tt.Texts())
	}
	if png.Y < more.Y+more.H || png.X < 0 || png.X+png.W > 260 {
		t.Errorf("As PNG at %+v, not below the overflow button at %+v", png, more)
	}
	if err := tt.Click("As PNG"); err != nil {
		t.Fatal(err)
	}
	if chosen != "PNG" || tt.HasText("As PDF") {
		t.Errorf("chose %q, menu shows %v", chosen, tt.HasText("As PDF"))
	}
}
