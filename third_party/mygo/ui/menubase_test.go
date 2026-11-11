package ui

import (
	"image/color"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// baseMenuApp is a view with a menu button on MenuBase and a row with a
// context menu on ContextMenuBase, whose items are of a look of its own,
// and what they did.
type baseMenuApp struct {
	open, rowOpen bool
	chosen        []string
	pinned        bool
	sort          string
	highlighted   []string
}

func (a *baseMenuApp) view(c *context) {
	a.highlighted = a.highlighted[:0]
	coreColumn(c).Fill().Padding(20).Gap(10).AlignItems(Start).Children(func() {
		coreButton(c, "Edit")
		m := coreMenuBase(c, &a.open)
		m.Trigger.Label("Actions").Padding(4).Children(func() { coreText(c, "Actions") })
		m.Popup(func(panel *node) { a.items(c, m, panel) })
		row := coreRow(c).Padding(8).Focusable().Label("Report").Children(func() { coreText(c, "Report.pdf") })
		rm := coreContextMenuBase(c, row, &a.rowOpen)
		rm.Popup(func(panel *node) { a.items(c, rm, panel) })
	})
}

// items builds the items of m, each a row holding its name.
func (a *baseMenuApp) items(c *context, m *menuParts, panel *node) {
	panel.Padding(4)
	item := func(e *node, name string) *node {
		e.Padding(4).Children(func() { coreText(c, name) })
		if e.Highlighted() {
			a.highlighted = append(a.highlighted, name)
		}
		return e
	}
	if item(m.Item(), "Open").Clicked() {
		a.chosen = append(a.chosen, "Open")
	}
	if item(m.CheckItem(a.pinned), "Pinned").Clicked() {
		a.pinned = !a.pinned
	}
	coreBox(c).Height(1) // a separator of its own
	sortBy := item(m.SubmenuItem(), "Sort by")
	m.Submenu(sortBy, func(panel *node) {
		panel.Padding(4)
		for _, by := range []string{"Name", "Date"} {
			if item(m.RadioItem(a.sort == by), by).Clicked() {
				a.sort = by
			}
		}
	})
	item(m.Item(), "Delete").Disabled(true)
}

func TestMenuBaseKeyboard(t *testing.T) {
	a := &baseMenuApp{sort: "Name"}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Actions")
	if !a.open || !menuShows(tt) || len(a.highlighted) != 0 {
		t.Fatalf("a click: open %v, highlighted %q", a.open, a.highlighted)
	}
	if s := tt.rt.states[tt.rt.focused]; s == nil || s.role != RoleMenu {
		t.Fatal("the focus is not on the menu as it opens")
	}
	steps := []struct {
		key  Key
		want string
	}{
		{KeyDown, "Open"}, // from none, the first
		{KeyDown, "Pinned"},
		{KeyDown, "Sort by"}, // past the separator
		{KeyDown, "Open"},    // past Delete, disabled, and round
		{KeyUp, "Sort by"},
		{KeyHome, "Open"},
		{KeyEnd, "Sort by"},
		{KeyRight, "Name"}, // into the submenu, on its first item
		{KeyDown, "Date"},
		{KeyLeft, "Sort by"}, // out of it
		{KeyEnter, "Name"},   // into it again
		{KeyEscape, "Sort by"},
	}
	for i, s := range steps {
		tt.Key(0, s.key)
		if !menuShows(tt) {
			t.Fatalf("step %d: the menu closed", i)
		}
		wantFocus(t, tt, s.want)
		if !slices.Contains(a.highlighted, s.want) {
			t.Fatalf("step %d: %q highlighted, not %q", i, a.highlighted, s.want)
		}
	}
	if tt.HasText("Date") {
		t.Error("Escape left the submenu open")
	}
	// Typing the first letters of an item goes to it.
	tt.TypeKey(0, KeyP, "p")
	wantFocus(t, tt, "Pinned")

	// Enter in the submenu chooses, closing the menu, and the focus goes
	// back to the button.
	tt.Key(0, KeyEnd)
	tt.Key(0, KeyRight)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if a.sort != "Date" || a.open || menuShows(tt) {
		t.Errorf("Enter on Date: sort %q, open %v", a.sort, a.open)
	}
	if !tt.Focused("Actions") {
		t.Error("the focus did not go back to the button")
	}
	// Enter on the button opens the menu on its first item; Space chooses.
	tt.Key(0, KeyEnter)
	wantFocus(t, tt, "Open")
	tt.Key(0, KeySpace)
	if !slices.Equal(a.chosen, []string{"Open"}) || a.open {
		t.Errorf("Space: chose %q, open %v", a.chosen, a.open)
	}
	// Down opens it too, and Escape and Tab close it.
	for _, k := range []Key{KeyEscape, KeyTab} {
		tt.Key(0, KeyDown)
		if !a.open {
			t.Fatal("Down did not open the menu")
		}
		tt.Key(0, k)
		if a.open || menuShows(tt) || !tt.Focused("Actions") {
			t.Errorf("key %d: open %v, focus back %v", k, a.open, tt.Focused("Actions"))
		}
	}
}

// Up on the button opens the menu on its last item that can be chosen
// (WAI-ARIA Authoring Practices, Menu Button: Up Arrow), past Delete,
// which is disabled; Down then moves round to the first.
func TestMenuBaseUpOpensOnLast(t *testing.T) {
	a := &baseMenuApp{sort: "Name"}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Edit")
	tt.Key(0, KeyTab)
	if !tt.Focused("Actions") {
		t.Fatal("the focus is not on the button")
	}
	tt.Key(0, KeyUp)
	if !a.open || !menuShows(tt) {
		t.Fatalf("Up did not open the menu: open %v", a.open)
	}
	wantFocus(t, tt, "Sort by")
	if !slices.Contains(a.highlighted, "Sort by") || slices.Contains(a.highlighted, "Open") {
		t.Errorf("highlighted %q, want only the last item", a.highlighted)
	}
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Open")
	tt.Key(0, KeyEscape)
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "Open")
}

func TestMenuBasePointer(t *testing.T) {
	a := &baseMenuApp{sort: "Name"}
	tt := coreNewTester(a.view, 400, 400)
	tt.Click("Actions")
	// The pointer highlights the item it moves onto, and opens a submenu.
	r, _ := tt.Find("Pinned")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if !slices.Equal(a.highlighted, []string{"Pinned"}) {
		t.Errorf("the pointer on Pinned highlights %q", a.highlighted)
	}
	r, _ = tt.Find("Sort by")
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	if !tt.HasText("Date") {
		t.Fatal("the submenu did not open under the pointer")
	}
	if d, _ := tt.Find("Date"); d.X < r.X+r.W-10 {
		t.Errorf("the submenu at %+v, not after its item at %+v", d, r)
	}
	tt.Click("Date")
	if a.sort != "Date" || a.open {
		t.Errorf("clicking Date: sort %q, open %v", a.sort, a.open)
	}
	// A disabled item takes no click, and the menu stays.
	tt.Click("Actions")
	tt.Click("Delete")
	if len(a.chosen) != 0 || !a.open {
		t.Errorf("clicking Delete, disabled: chose %q, open %v", a.chosen, a.open)
	}
	tt.Click("Pinned")
	if !a.pinned || a.open {
		t.Errorf("clicking Pinned: pinned %v, open %v", a.pinned, a.open)
	}
	// A press on the button closes it, as one outside does.
	tt.Click("Actions")
	tt.Click("Actions")
	if a.open || menuShows(tt) {
		t.Error("the button pressed again left the menu open")
	}
	tt.Click("Actions")
	tt.ClickAt(390, 390)
	if a.open {
		t.Error("a press outside left the menu open")
	}
	// The view opens it too, below the button, and closes it.
	a.open = true
	tt.Frame()
	b, _ := tt.Find("Actions")
	if o, ok := tt.Find("Open"); !ok || o.Y < b.Y+b.H {
		t.Errorf("opened by the view: Open at %+v, the button at %+v", o, b)
	}
	a.open = false
	tt.Frame()
	if menuShows(tt) {
		t.Error("closed by the view, the menu shows")
	}
}

func TestContextMenuBase(t *testing.T) {
	a := &baseMenuApp{sort: "Name"}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	r, _ := tt.Find("Report.pdf")
	x, y := r.X+40, r.Y+r.H/2
	tt.RightClickAt(x, y)
	if !a.rowOpen || tt.Menu() != nil {
		t.Fatalf("a right-click: open %v, the system's menu %q", a.rowOpen, tt.Menu())
	}
	if o, _ := tt.Find("Open"); o.X < x || o.Y < y || o.X > x+20 || o.Y > y+20 {
		t.Errorf("Open at %+v, not after the click at (%v, %v)", o, x, y)
	}
	tt.Click("Open")
	if !slices.Equal(a.chosen, []string{"Open"}) || a.rowOpen {
		t.Errorf("clicking Open: chose %q, open %v", a.chosen, a.rowOpen)
	}
	// The menu key opens it while the row has the focus, on its first item.
	tt.Click("Report.pdf")
	tt.Key(0, KeyContextMenu)
	if !a.rowOpen {
		t.Fatal("the menu key opened no menu")
	}
	wantFocus(t, tt, "Open")
	tt.Key(0, KeyEscape)
	if a.rowOpen || !tt.Focused("Report.pdf") {
		t.Errorf("Escape: open %v, focus back %v", a.rowOpen, tt.Focused("Report.pdf"))
	}
}

func TestMenuBaseAccessibility(t *testing.T) {
	a := &baseMenuApp{sort: "Date", pinned: true}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	n := accessNode(t, tt.h.access, platform.RoleMenuButton, "Actions")
	if n.Actions&platform.ActionPress == 0 || n.States&platform.AccessExpanded != 0 {
		t.Errorf("the menu button: %+v", n)
	}
	// Pressing it opens the menu, on its first item.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessPress})
	wantFocus(t, tt, "Open")
	tree := tt.h.access
	if n := accessNode(t, tree, platform.RoleMenuButton, "Actions"); n.States&platform.AccessExpanded == 0 {
		t.Error("the button is not expanded with its menu open")
	}
	menu := accessNode(t, tree, platform.RoleMenu, "")
	open := accessNode(t, tree, platform.RoleMenuItem, "Open")
	if open.PosInSet != 1 || open.SetSize != 4 || tree.Nodes[open.Parent].ID != menu.ID {
		t.Errorf("Open: %+v", open)
	}
	if pinned := accessNode(t, tree, platform.RoleMenuItemCheckBox, "Pinned"); pinned.States&platform.AccessChecked == 0 {
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
	// The submenu is named by its item.
	accessNode(t, tree, platform.RoleMenu, "Sort by")
	name := accessNode(t, tree, platform.RoleMenuItemRadio, "Name")
	date := accessNode(t, tree, platform.RoleMenuItemRadio, "Date")
	if name.States&platform.AccessChecked != 0 || date.States&platform.AccessChecked == 0 {
		t.Errorf("radio items: Name %b, Date %b", name.States, date.States)
	}
}

// themeMenuApp is a view with a menu on MenuBase holding a row of radio
// items, a theme's, that the keys take as one stop and that switch it
// without closing the menu, between items that close it, the first left
// out when noProfile is set.
type themeMenuApp struct {
	open        bool
	noProfile   bool
	theme       string
	chosen      []string
	highlighted []string
}

func (a *themeMenuApp) view(c *context) {
	a.highlighted = a.highlighted[:0]
	coreColumn(c).Fill().Padding(20).AlignItems(Start).Children(func() {
		m := coreMenuBase(c, &a.open)
		m.Trigger.Label("Account").Padding(4).Children(func() { coreText(c, "Account") })
		m.Popup(func(panel *node) {
			panel.Padding(4)
			item := func(e *node, name string) *node {
				e.Padding(4).Children(func() { coreText(c, name) })
				if e.Highlighted() {
					a.highlighted = append(a.highlighted, name)
				}
				if e.Clicked() {
					a.chosen = append(a.chosen, name)
				}
				return e
			}
			if !a.noProfile {
				item(m.Item(), "Profile")
			}
			m.Group(func(row *node) {
				row.Gap(4).Label("Theme")
				for _, th := range []string{"Light", "Dark", "System"} {
					if item(m.RadioItem(a.theme == th).KeepOpen(), th).Clicked() {
						a.theme = th
					}
				}
			})
			item(m.Item(), "Sign out")
		})
	})
}

func TestMenuBaseGroupKeys(t *testing.T) {
	a := &themeMenuApp{theme: "Dark"}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnter)
	wantFocus(t, tt, "Profile")
	steps := []struct {
		key  Key
		want string
	}{
		{KeyDown, "Dark"}, // the group, at its item checked
		{KeyDown, "Sign out"},
		{KeyUp, "Dark"},
		{KeyRight, "System"},
		{KeyRight, "Light"}, // round
		{KeyLeft, "System"},
		{KeyEnter, "System"}, // chosen, and the menu stays
		{KeyDown, "Sign out"},
		{KeyUp, "System"}, // the group's item checked now
		{KeyLeft, "Dark"},
		{KeyLeft, "Light"},
		{KeySpace, "Light"},
		{KeyHome, "Profile"},
		{KeyEnd, "Sign out"},
		{KeyDown, "Profile"},
		{KeyUp, "Sign out"},
	}
	for i, s := range steps {
		tt.Key(0, s.key)
		if !a.open {
			t.Fatalf("step %d: the menu closed", i)
		}
		wantFocus(t, tt, s.want)
		if !slices.Equal(a.highlighted, []string{s.want}) {
			t.Fatalf("step %d: %q highlighted, not %q", i, a.highlighted, s.want)
		}
	}
	if a.theme != "Light" || !slices.Equal(a.chosen, []string{"System", "Light"}) {
		t.Errorf("theme %q, chose %q", a.theme, a.chosen)
	}
	// An item that does not keep the menu open closes it.
	tt.Key(0, KeyEnter)
	if a.open || !tt.Focused("Account") || !slices.Equal(a.chosen, []string{"System", "Light", "Sign out"}) {
		t.Errorf("Enter on Sign out: open %v, chose %q", a.open, a.chosen)
	}
	// Opened by the keyboard on a group, it starts at its item checked.
	a.theme, a.chosen = "System", nil
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "System")

	// Right to left, Left moves on and Right back.
	tt.Key(0, KeyEscape)
	tt.SetDirection(RightToLeft)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	wantFocus(t, tt, "System")
	tt.Key(0, KeyLeft)
	wantFocus(t, tt, "Light")
	tt.Key(0, KeyRight)
	tt.Key(0, KeyRight)
	wantFocus(t, tt, "Dark")
	if !a.open || len(a.chosen) != 0 {
		t.Errorf("moving in the group: open %v, chose %q", a.open, a.chosen)
	}

	// A menu starting with the group opens on its item checked.
	tt.Key(0, KeyEscape)
	a.noProfile = true
	tt.Key(0, KeyEnter)
	wantFocus(t, tt, "System")
	if !slices.Equal(a.highlighted, []string{"System"}) {
		t.Errorf("opened on the group: %q highlighted", a.highlighted)
	}
}

func TestMenuBaseKeepOpenPointer(t *testing.T) {
	a := &themeMenuApp{theme: "Light"}
	tt := coreNewTester(a.view, 400, 400)
	tt.Click("Account")
	tt.Click("Dark")
	if a.theme != "Dark" || !a.open || !tt.HasText("Profile") {
		t.Fatalf("clicking Dark: theme %q, open %v", a.theme, a.open)
	}
	if !slices.Equal(a.highlighted, []string{"Dark"}) {
		t.Errorf("after the click, %q highlighted", a.highlighted)
	}
	// The keys go on from the item clicked.
	tt.Key(0, KeyRight)
	tt.Key(0, KeyEnter)
	if a.theme != "System" || !a.open {
		t.Errorf("Right and Enter after the click: theme %q, open %v", a.theme, a.open)
	}
	tt.Click("Profile")
	if a.open || tt.HasText("Profile") || !slices.Equal(a.chosen, []string{"Dark", "System", "Profile"}) {
		t.Errorf("clicking Profile: open %v, chose %q", a.open, a.chosen)
	}
}

func TestMenuBaseGroupAccessibility(t *testing.T) {
	a := &themeMenuApp{theme: "Dark", open: true}
	tt := coreNewTester(a.view, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	menu := accessNode(t, tree, platform.RoleMenu, "")
	group := accessNode(t, tree, platform.RoleGroup, "Theme")
	if tree.Nodes[group.Parent].ID != menu.ID {
		t.Errorf("the group is not in the menu: %+v", group)
	}
	for i, th := range []string{"Light", "Dark", "System"} {
		n := accessNode(t, tree, platform.RoleMenuItemRadio, th)
		if tree.Nodes[n.Parent].ID != group.ID || n.PosInSet != i+1 || n.SetSize != 3 {
			t.Errorf("%s: %+v", th, n)
		}
		if checked := n.States&platform.AccessChecked != 0; checked != (th == "Dark") {
			t.Errorf("%s: checked %v", th, checked)
		}
	}
	for i, name := range []string{"Profile", "Sign out"} {
		if n := accessNode(t, tree, platform.RoleMenuItem, name); n.PosInSet != i+1 || n.SetSize != 2 {
			t.Errorf("%s: %+v", name, n)
		}
	}
}

func TestAccordionBase(t *testing.T) {
	open := []bool{true, false, false}
	names := []string{"General", "Privacy", "Advanced"}
	tt := coreNewTester(func(c *context) {
		acc := coreAccordionBase(c)
		acc.Root.Gap(4).Children(func() {
			for i, name := range names {
				item := acc.Item(&open[i])
				item.Trigger.Label(name).Padding(4).Children(func() { coreText(c, name) })
				item.Panel(func() { coreButton(c, name+" content") })
			}
		})
	}, 300, 300)
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	if !tt.HasText("General content") || tt.HasText("Privacy content") {
		t.Fatalf("shows %q", tt.Texts())
	}
	tt.Click("Privacy")
	if !open[1] || !tt.HasText("Privacy content") || !tt.Focused("Privacy") {
		t.Fatalf("a click: open %v", open)
	}
	// The arrows, Home and End move between the headers, not into the
	// sections.
	for _, step := range []struct {
		key  Key
		want string
	}{{KeyDown, "Advanced"}, {KeyDown, "Advanced"}, {KeyHome, "General"}, {KeyUp, "General"}, {KeyDown, "Privacy"}, {KeyEnd, "Advanced"}} {
		tt.Key(0, step.key)
		if !tt.Focused(step.want) {
			t.Fatalf("%v did not move to %s", step.key, step.want)
		}
	}
	tt.Key(0, KeySpace)
	if !open[2] {
		t.Error("Space did not open Advanced")
	}
	// Tab goes into the content of a section open.
	tt.Key(Shift, KeyTab)
	if !tt.Focused("Privacy content") {
		t.Error("Shift+Tab did not go into Privacy's content")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for i, name := range names {
		if n := accessNode(t, tt.h.access, platform.RoleDisclosure, name); (n.States&platform.AccessExpanded != 0) != open[i] {
			t.Errorf("%s: %+v", name, n)
		}
	}
}

func TestCheckboxStateBase(t *testing.T) {
	rows := []bool{true, false, true}
	changes := 0
	state := func() CheckState {
		n := 0
		for _, r := range rows {
			if r {
				n++
			}
		}
		switch n {
		case len(rows):
			return CheckOn
		case 0:
			return CheckOff
		}
		return CheckMixed
	}
	var shown CheckState
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			all := state()
			if coreCheckboxStateBase(c, &all).Size(20, 20).Label("all").Changed() {
				changes++
				for i := range rows {
					rows[i] = all == CheckOn
				}
			}
			shown = all
			coreCheckboxState(c, &all, "themed")
		})
	}, 200, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	states := func(label string) uint64 {
		return uint64(accessNode(t, tt.h.access, platform.RoleCheckBox, label).States)
	}
	if s := states("all"); s&uint64(platform.AccessMixed) == 0 || s&uint64(platform.AccessChecked) != 0 {
		t.Fatalf("some rows checked: states %b, not mixed", s)
	}
	if states("themed")&uint64(platform.AccessMixed) == 0 {
		t.Error("the themed check box is not mixed")
	}
	// A click on a mixed box checks it.
	tt.Click("all")
	if shown != CheckOn || !slices.Equal(rows, []bool{true, true, true}) || changes != 1 {
		t.Fatalf("a click on mixed: %v, rows %v, %d changes", shown, rows, changes)
	}
	if s := states("all"); s&uint64(platform.AccessChecked) == 0 || s&uint64(platform.AccessMixed) != 0 {
		t.Errorf("all rows checked: states %b", s)
	}
	// Space on a checked one, which the click focused, unchecks it.
	if !tt.Focused("all") {
		t.Fatal("the click did not focus the check box")
	}
	tt.Key(0, KeySpace)
	if shown != CheckOff || slices.Contains(rows, true) || changes != 2 {
		t.Errorf("Space on checked: %v, rows %v", shown, rows)
	}
	tt.Key(0, KeySpace)
	if shown != CheckOn {
		t.Errorf("Space on unchecked: %v", shown)
	}
}

// TestCheckboxStateShowsMixed checks the themed check box draws a dash
// while mixed, and neither a dash nor a check while unchecked.
func TestCheckboxStateShowsMixed(t *testing.T) {
	state := CheckMixed
	tt := coreNewTester(func(c *context) {
		th := *c.Theme()
		th.Background = RGB(255, 255, 255)
		c.SetTheme(&th)
		coreColumn(c).Padding(20).AlignItems(Start).Children(func() {
			coreCheckboxState(c, &state, "")
		})
	}, 60, 60)
	// The middle of the box: the dash, white on the accent.
	if c := tt.Image().RGBAAt(28, 28); c != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("the middle of a mixed box is %v, not the dash", c)
	}
	if c := tt.Image().RGBAAt(22, 28); c == (color.RGBA{255, 255, 255, 255}) {
		t.Error("a mixed box is not filled")
	}
	state = CheckOff
	tt.Frame()
	if c := tt.Image().RGBAAt(22, 28); c != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("an unchecked box is filled: %v", c)
	}
}

// TestFocusRingGeometry draws the ring of a focused button as the theme
// and the element set it, and reads where it is.
func TestFocusRingGeometry(t *testing.T) {
	white := color.RGBA{255, 255, 255, 255}
	red := RGB(255, 0, 0)
	// ring returns the columns, from the button's left edge at x=20, the
	// ring paints on the button's middle row, and its color there.
	ring := func(theme func(*Theme), elem func(*node)) (cols []int, at color.RGBA) {
		tt := coreNewTester(func(c *context) {
			th := *c.Theme()
			th.Background = RGB(255, 255, 255)
			th.Focus = red
			if theme != nil {
				theme(&th)
			}
			c.SetTheme(&th)
			coreColumn(c).Padding(20).AlignItems(Start).Children(func() {
				b := coreButtonBase(c).Size(40, 20).Label("b")
				if elem != nil {
					elem(b)
				}
			})
		}, 100, 60)
		tt.Key(0, KeyTab)
		img := tt.Image()
		for x := 10; x < 30; x++ {
			if c := img.RGBAAt(x, 30); c != white {
				cols = append(cols, x-20)
				at = c
			}
		}
		return cols, at
	}
	for _, tc := range []struct {
		name  string
		theme func(*Theme)
		elem  func(*node)
		want  []int
		color color.RGBA
	}{
		{"default", nil, nil, []int{-3, -2}, color.RGBA{255, 0, 0, 255}},
		{"theme's width and offset", func(t *Theme) { t.FocusRingWidth, t.FocusRingOffset = 3, 2 }, nil, []int{-5, -4, -3}, color.RGBA{255, 0, 0, 255}},
		{"on the edge", func(t *Theme) { t.FocusRingOffset = 0 }, nil, []int{-2, -1}, color.RGBA{255, 0, 0, 255}},
		{"inside, set on the element", nil, func(e *node) { e.FocusRingOffset(-4) }, []int{2, 3}, color.RGBA{255, 0, 0, 255}},
		{"the element's color", nil, func(e *node) { e.FocusRingColor(RGB(0, 0, 255)) }, []int{-3, -2}, color.RGBA{0, 0, 255, 255}},
	} {
		cols, at := ring(tc.theme, tc.elem)
		if !slices.Equal(cols, tc.want) || at != tc.color {
			t.Errorf("%s: the ring covers columns %v in %v, want %v in %v", tc.name, cols, at, tc.want, tc.color)
		}
	}
}
