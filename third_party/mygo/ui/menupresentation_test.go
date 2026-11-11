package ui

import (
	"slices"
	"testing"
)

// drawingMenus builds view in a window that draws its menus.
func drawingMenus(view func(c *context)) func(c *context) {
	return func(c *context) {
		c.SetMenuPresentation(PopoverMenus)
		view(c)
	}
}

// rightClickField right-clicks the middle of the text input labeled label.
func rightClickField(t *testing.T, tt *Tester, label string) {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("no text input %q", label)
	}
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
}

// wantDrawn fails unless the menu showing label is drawn in the window and
// the system shows none.
func wantDrawn(t *testing.T, tt *Tester, label string) {
	t.Helper()
	if tt.h.menu != nil || !tt.HasText(label) {
		t.Fatalf("%q: the system's menu %v, drawn in the window %v", label, tt.h.menu != nil, tt.HasText(label))
	}
}

func TestMenuPresentationContextMenu(t *testing.T) {
	p := PopoverMenus
	var chosen []string
	view := func(c *context) {
		c.SetMenuPresentation(p)
		coreColumn(c).Fill().Padding(20).Children(func() {
			coreRow(c).Padding(8).Children(func() { coreText(c, "Report.pdf") }).ContextMenu(func(m *Menu) {
				if m.Item("Open").Chosen() {
					chosen = append(chosen, "Open")
				}
				m.Separator()
				m.Submenu("Move to", func(m *Menu) {
					if m.Item("Desktop").Chosen() {
						chosen = append(chosen, "Desktop")
					}
				})
			})
		})
	}
	tt := coreNewTester(view, 400, 300)
	if err := tt.RightClick("Report.pdf"); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, tt, "Open")
	if got, want := tt.Menu(), []string{"Open", "-", "Move to"}; !slices.Equal(got, want) {
		t.Fatalf("Menu() = %q, want %q", got, want)
	}
	if err := tt.ChooseMenuItem("Move to", "Desktop"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(chosen, []string{"Desktop"}) || tt.HasText("Open") || tt.Menu() != nil {
		t.Fatalf("chose %q; menu still drawn %v, Menu() %q", chosen, tt.HasText("Open"), tt.Menu())
	}
	// A click on an item chooses it, as PopoverMenu's.
	tt.RightClick("Report.pdf")
	if err := tt.Click("Open"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(chosen, []string{"Desktop", "Open"}) {
		t.Fatalf("chose %q", chosen)
	}
	// CloseMenu closes it without a choice.
	tt.RightClick("Report.pdf")
	tt.CloseMenu()
	if tt.HasText("Open") || tt.Menu() != nil {
		t.Fatalf("CloseMenu left the menu: drawn %v, Menu() %q", tt.HasText("Open"), tt.Menu())
	}
	// Back to the system's.
	p = SystemMenus
	tt.Frame()
	tt.RightClick("Report.pdf")
	if tt.h.menu == nil || tt.HasText("Open") {
		t.Fatalf("SystemMenus: system menu %v, drawn %v", tt.h.menu != nil, tt.HasText("Open"))
	}
}

func TestMenuPresentationMenuButton(t *testing.T) {
	var chosen string
	view := drawingMenus(func(c *context) {
		coreColumn(c).Fill().Padding(20).AlignItems(Start).Children(func() {
			coreMenuButton(c, "Actions", func(m *Menu) {
				if m.Item("Duplicate").Chosen() {
					chosen = "Duplicate"
				}
			})
		})
	})
	tt := coreNewTester(view, 400, 300)
	tt.Click("Actions")
	wantDrawn(t, tt, "Duplicate")
	// Pressed again, the button closes its menu.
	tt.Click("Actions")
	if tt.HasText("Duplicate") {
		t.Fatal("a second press left the menu open")
	}
	tt.Click("Actions")
	if err := tt.ChooseMenuItem("Duplicate"); err != nil || chosen != "Duplicate" {
		t.Fatalf("chose %q: %v", chosen, err)
	}
}

// The menus MyGo's widgets build with ContextMenu follow the window too.
func TestMenuPresentationTableHeaderMenu(t *testing.T) {
	chosen := -1
	s := ListState{HeaderMenu: func(m *Menu, col int) {
		if m.Item("Hide Column").Chosen() {
			chosen = col
		}
	}}
	cols := []TableColumn{{Title: "Name"}, {Title: "Kind", Width: 100}}
	tt := coreNewTester(drawingMenus(tableView(&s, cols, []string{"a.txt"})), 500, 300)
	tt.RightClick("Kind")
	wantDrawn(t, tt, "Hide Column")
	if err := tt.ChooseMenuItem("Hide Column"); err != nil || chosen != 1 {
		t.Fatalf("chose column %d: %v", chosen, err)
	}
}

func TestMenuPresentationTextInput(t *testing.T) {
	value := "hello"
	view := drawingMenus(func(c *context) {
		coreColumn(c).Fill().Padding(20).Children(func() {
			coreTextInput(c, &value).Width(200).Label("Name")
		})
	})
	tt := coreNewTester(view, 400, 300)
	rightClickField(t, tt, "Name")
	wantDrawn(t, tt, "Select All")
	if !slices.Contains(tt.Menu(), "Paste") {
		t.Fatalf("no editing menu drawn: Menu() %q", tt.Menu())
	}
	if err := tt.ChooseMenuItem("Select All"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("Select All") {
		t.Fatal("the menu stayed open after a choice")
	}
	// In the selection, a right-click keeps it.
	rightClickField(t, tt, "Name")
	if err := tt.ChooseMenuItem("Cut"); err != nil {
		t.Fatal(err)
	}
	if value != "" || tt.h.clipboard != "hello" {
		t.Fatalf("after Cut: value %q, clipboard %q", value, tt.h.clipboard)
	}
	// Shift+F10 opens it from the keyboard.
	value = "again"
	tt.Frame()
	tt.Key(Shift, KeyF10)
	wantDrawn(t, tt, "Select All")
	tt.Key(0, KeyEscape)
	if tt.HasText("Select All") {
		t.Fatal("Escape left the editing menu open")
	}
}

func TestMenuPresentationSelectableText(t *testing.T) {
	view := drawingMenus(func(c *context) {
		coreColumn(c).Fill().Padding(20).Children(func() {
			coreText(c, "Some words to copy").Selectable()
		})
	})
	tt := coreNewTester(view, 400, 300)
	if err := tt.RightClick("Some words to copy"); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, tt, "Select All")
	if got := tt.Menu(); !slices.Contains(got, "Copy") || !slices.Contains(got, "Select All") {
		t.Fatalf("Menu() = %q", got)
	}
	if err := tt.ChooseMenuItem("Select All"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("Some words to copy")
	if err := tt.ChooseMenuItem("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.h.clipboard != "Some words to copy" {
		t.Fatalf("copied %q", tt.h.clipboard)
	}
}

// A menu button in a toolbar's overflow menu opens its own menu from it, as
// with the system's menus (TestMenuButtonInAToolbarsOverflow).
func TestMenuPresentationToolbarOverflow(t *testing.T) {
	chosen := ""
	tt := coreNewTester(drawingMenus(func(c *context) {
		coreToolbar(c, func() {
			for _, n := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
				coreButton(c, n)
			}
			coreMenuButton(c, "Export", func(m *Menu) {
				for _, as := range []string{"PDF", "PNG"} {
					if m.Item("As " + as).Chosen() {
						chosen = as
					}
				}
			})
		}).Fill()
	}), 260, 300)
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, tt, "Export")
	if err := tt.ChooseMenuItem("Export"); err != nil {
		t.Fatal(err)
	}
	wantDrawn(t, tt, "As PDF")
	if got, want := tt.Menu(), []string{"As PDF", "As PNG"}; !slices.Equal(got, want) {
		t.Fatalf("choosing Export shows %q, want %q", got, want)
	}
	if err := tt.Click("As PNG"); err != nil || chosen != "PNG" {
		t.Fatalf("chose As PNG: %v, chosen %q", err, chosen)
	}
}

// A text input in an overlay, as a dialog's, draws its editing menu too.
func TestMenuPresentationTextInputInAnOverlay(t *testing.T) {
	value := "in a dialog"
	tt := coreNewTester(drawingMenus(func(c *context) {
		coreOverlay(c, func() {
			coreColumn(c).Absolute().Left(40).Top(40).Padding(10).Children(func() {
				coreTextInput(c, &value).Width(200).Label("Dialog field")
			})
		})
	}), 400, 300)
	rightClickField(t, tt, "Dialog field")
	wantDrawn(t, tt, "Select All")
}

// drawnPanel returns the panel of the drawn menu, or nil.
func drawnPanel(tt *Tester) *node {
	var found *node
	var walk func(e *node)
	walk = func(e *node) {
		if e.widget == "PopoverMenu" && found == nil {
			found = e
		}
		for ch := e.first; ch != nil; ch = ch.next {
			walk(ch)
		}
	}
	walk(tt.rt.c.root)
	return found
}

// A text input in a Themed opens its editing menu in that theme, as a
// ContextMenu there does.
func TestMenuPresentationEditMenuTakesTheThemed(t *testing.T) {
	dark := DarkTheme()
	value := "themed"
	tt := coreNewTester(drawingMenus(func(c *context) {
		c.SetTheme(LightTheme())
		coreThemed(c, dark).Fill().Padding(20).Children(func() {
			coreTextInput(c, &value).Width(200).Label("Dark field")
		})
	}), 400, 300)
	rightClickField(t, tt, "Dark field")
	wantDrawn(t, tt, "Select All")
	p := drawnPanel(tt)
	if p == nil || p.bg != dark.Background {
		t.Fatalf("the editing menu's panel %v, want the Themed's background %v", p, dark.Background)
	}
}

// Set back to SystemMenus, the window closes its drawn editing menu, as it
// does a ContextMenu's.
func TestMenuPresentationSwitchingBackClosesTheEditMenu(t *testing.T) {
	p := PopoverMenus
	value := "hello"
	tt := coreNewTester(func(c *context) {
		c.SetMenuPresentation(p)
		coreTextInput(c, &value).Width(200).Label("Name")
	}, 400, 300)
	rightClickField(t, tt, "Name")
	wantDrawn(t, tt, "Select All")
	p = SystemMenus
	tt.Frame()
	if tt.HasText("Select All") || tt.Menu() != nil {
		t.Fatalf("the drawn editing menu stayed: Menu() %q", tt.Menu())
	}
}

// The editing menu of a text input on a page going away closes, as the page
// takes no input.
func TestMenuPresentationEditMenuOfALeavingPage(t *testing.T) {
	r := NewRouter("/")
	// The page going away fades out while the other comes: it is built,
	// inert, until then.
	r.Transition = TransitionMatch
	value := "on the first page"
	tt := coreNewTester(drawingMenus(func(c *context) {
		r.coreView(c, func(rt *Route) {
			if rt.Match("/") {
				coreTextInput(c, &value).Width(200).Label("First field")
			} else {
				coreText(c, "Second page")
			}
		}).Fill()
	}), 400, 300)
	rightClickField(t, tt, "First field")
	wantDrawn(t, tt, "Select All")
	r.Push("/second")
	tt.Frame()
	if !tt.HasText("Second page") {
		t.Fatalf("no second page: %q", tt.Texts())
	}
	if tt.HasText("Select All") {
		t.Fatal("the editing menu of the page going away still shows")
	}
}
