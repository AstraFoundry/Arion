package ui

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

var fonts = []string{"Arial", "Helvetica", "Times", "Courier", "Optima"}

func TestComboboxChosenRebuildsDependentUI(t *testing.T) {
	for _, keyboard := range []bool{false, true} {
		name := "pointer"
		if keyboard {
			name = "keyboard"
		}
		t.Run(name, func(t *testing.T) {
			text, picked, choices := "", "none", 0
			tt := NewTester(func(c *Context) {
				Text(c, "Picked: "+picked)
				p := ComboboxBase(c.Key("combo"), &text)
				p.Input.Label("Combo").Width(180)
				p.Popup(func(panel Element) {
					panel.Children(func() {
						for _, value := range []string{"Alpha", "Beta"} {
							p.Item(value).Children(func() { Text(c, value) })
						}
					})
				})
				if value, ok := p.Chosen(); ok {
					picked = value
					choices++
				}
			}, 300, 300)
			if err := tt.Click("Combo"); err != nil {
				t.Fatal(err)
			}
			if keyboard {
				tt.Key(0, KeyDown)
				tt.Key(0, KeyDown)
				tt.Key(0, KeyEnter)
			} else if err := tt.Click("Beta"); err != nil {
				t.Fatal(err)
			}
			if picked != "Beta" || choices != 1 || !tt.HasText("Picked: Beta") {
				t.Fatalf("picked %q, choices %d, texts %q", picked, choices, tt.Texts())
			}
			tt.Frame()
			if choices != 1 {
				t.Fatal("another frame repeated the choice")
			}
		})
	}
}

func TestCombobox(t *testing.T) {
	font, changes := "Arial", 0
	tt := coreNewTester(func(c *context) {
		coreRow(c).Gap(8).Children(func() {
			if coreCombobox(c, &font, fonts).Label("Font").Width(200).Changed() {
				changes++
			}
			coreButton(c, "Next")
		})
	}, 400, 400)
	// A click shows all the options.
	tt.Click("Font")
	if !tt.HasText("Helvetica") || !tt.HasText("Optima") {
		t.Fatalf("a click shows %q", tt.Texts())
	}
	// Typing filters them, those starting with the text first.
	tt.Key(Cmd, KeyA)
	tt.Type("ti")
	if !tt.HasText("Helvetica") || !tt.HasText("Times") || !tt.HasText("Optima") || tt.HasText("Courier") {
		t.Fatalf("ti shows %q", tt.Texts())
	}
	if texts := tt.Texts(); slices.Index(texts, "Times") > slices.Index(texts, "Optima") {
		t.Errorf("Times, starting with ti, does not come first: %q", texts)
	}
	tt.Type("me")
	if tt.HasText("Optima") || !tt.HasText("Times") {
		t.Fatalf("time shows %q", tt.Texts())
	}
	tt.Key(0, KeyEnter)
	if font != "Times" || changes != 1 || tt.HasText("Optima") {
		t.Fatalf("Enter chose %q (%d changes), the popup shows %v", font, changes, tt.HasText("Optima"))
	}
	// Down opens them all, and moves; a click chooses.
	tt.Key(0, KeyDown)
	if !tt.HasText("Courier") {
		t.Fatal("Down did not open the options")
	}
	tt.Click("Courier")
	if font != "Courier" || !tt.Focused("Font") {
		t.Fatalf("a click chose %q; focused %v", font, tt.Focused("Font"))
	}
	// Escape closes; what was typed goes back to the choice as the focus
	// leaves.
	tt.Key(Cmd, KeyA)
	tt.Type("zzz")
	tt.Key(0, KeyEscape)
	if tt.HasText("Arial") {
		t.Error("Escape left the options")
	}
	tt.Key(0, KeyTab)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleComboBox, "Font"); n.Value != "Courier" || font != "Courier" {
		t.Errorf("after the focus left, it shows %q, chose %q", n.Value, font)
	}
}

// A click opening a combobox's options selects its choice, as the chevron
// does, so that typing replaces it and filters, instead of appending to it
// and matching nothing; clicks while they show place the caret.
func TestComboboxClickSelectsChoice(t *testing.T) {
	font, changes := "Arial", 0
	tt := coreNewTester(func(c *context) {
		if coreCombobox(c, &font, fonts).Label("Font").Width(200).Changed() {
			changes++
		}
	}, 400, 400)
	tt.Click("Font")
	if !tt.HasText("Helvetica") {
		t.Fatalf("a click shows %q", tt.Texts())
	}
	tt.Type("ti")
	if !tt.HasText("Times") || !tt.HasText("Helvetica") || !tt.HasText("Optima") || tt.HasText("Courier") {
		t.Fatalf("typing after a click filters %q", tt.Texts())
	}
	tt.Key(0, KeyEnter)
	if font != "Times" || changes != 1 {
		t.Fatalf("Enter chose %q (%d changes)", font, changes)
	}
	// The chevron, at the field's end, opens them the same way.
	r, _ := tt.Find("Font")
	tt.ClickAt(r.X+r.W-12, r.Y+r.H/2)
	if !tt.HasText("Optima") {
		t.Fatalf("the chevron shows %q", tt.Texts())
	}
	tt.Type("co")
	if tt.HasText("Optima") || !tt.HasText("Courier") {
		t.Fatalf("typing after the chevron filters %q", tt.Texts())
	}
	tt.Key(0, KeyEnter)
	if font != "Courier" || changes != 2 {
		t.Fatalf("Enter chose %q (%d changes)", font, changes)
	}
	// A click while they show places the caret: typing appends. (Away
	// from the first, which a double-click would select a word.)
	tt.Click("Font")
	tt.ClickAt(r.X+r.W/2+20, r.Y+r.H/2)
	tt.Type("x")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleComboBox, "Font"); n.Value == "x" {
		t.Errorf("a second click left the choice selected: %q", n.Value)
	}
}

func TestComboboxAccessibility(t *testing.T) {
	font := "Arial"
	tt := coreNewTester(func(c *context) {
		coreCombobox(c, &font, fonts).Label("Font").Width(200)
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Click("Font")
	tt.Key(0, KeyDown) // the first
	tt.Key(0, KeyDown)
	tree := tt.h.access
	cb := accessNode(t, tree, platform.RoleComboBox, "Font")
	if cb.States&platform.AccessExpanded == 0 || cb.Value != "Arial" {
		t.Errorf("the combobox: %+v", cb)
	}
	list := accessNode(t, tree, platform.RoleList, "")
	if list.States&platform.AccessSelectable == 0 {
		t.Errorf("the popup: %+v", list)
	}
	// The option the arrows are on has the focus, and says where it is.
	f, ok := byID(tree, tree.Focus)
	if !ok || f.Role != platform.RoleListItem || f.Label != "Helvetica" || f.PosInSet != 2 || f.SetSize != 5 || f.States&platform.AccessChecked == 0 {
		t.Errorf("the focus is on %+v", f)
	}
}

func TestAutocomplete(t *testing.T) {
	city, submitted := "", 0
	cities := []string{"Paris", "Parma", "Lyon", "Comparis"}
	tt := coreNewTester(func(c *context) {
		if coreAutocomplete(c, &city, cities).Label("City").Width(200).Submitted() {
			submitted++
		}
	}, 400, 400)
	tt.Click("City")
	if tt.HasText("Paris") {
		t.Fatal("suggestions before anything was typed")
	}
	tt.Type("par")
	if !tt.HasText("Paris") || !tt.HasText("Parma") || !tt.HasText("Comparis") || tt.HasText("Lyon") {
		t.Fatalf("par suggests %q", tt.Texts())
	}
	// Enter on no suggestion is Submitted, with what was typed.
	tt.Key(0, KeyEnter)
	if city != "par" || submitted != 1 {
		t.Fatalf("Enter: %q, %d submitted", city, submitted)
	}
	tt.Type("i")
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if city != "Paris" || tt.HasText("Comparis") {
		t.Errorf("Down, Enter took %q", city)
	}
}

func TestSearchField(t *testing.T) {
	query, changes, dialog := "", 0, true
	tt := coreNewTester(func(c *context) {
		coreModal(c, &dialog, func() {
			if coreSearchField(c, &query).Label("Search mail").Width(240).AutoFocus().Changed() {
				changes++
			}
		})
	}, 500, 400)
	tt.Click("Search mail")
	tt.Type("invoice")
	if query != "invoice" || changes == 0 {
		t.Fatalf("typed %q, %d changes", query, changes)
	}
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	if query != "" || !tt.Focused("Search mail") {
		t.Fatalf("Clear left %q", query)
	}
	tt.Type("x")
	tt.Key(0, KeyEscape)
	if query != "" || !dialog {
		t.Fatalf("Escape with text: %q, dialog open %v", query, dialog)
	}
	tt.Key(0, KeyEscape)
	if dialog {
		t.Error("Escape in an empty search field did not reach the dialog")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	dialog = true
	tt.Frame()
	n := accessNode(t, tt.h.access, platform.RoleTextField, "Search mail")
	if n.States&platform.AccessSearch == 0 {
		t.Errorf("the search field reads %+v", n)
	}
}

func TestTokenField(t *testing.T) {
	tags := []string{"go"}
	langs := []string{"typescript", "python", "rust"}
	tt := coreNewTester(func(c *context) {
		coreTokenField(c, &tags, langs).Label("Tags").Width(360)
	}, 500, 400)
	tt.Click("Tags")
	tt.Type("zig,")
	tt.Type("lua")
	tt.Key(0, KeyEnter)
	if !slices.Equal(tags, []string{"go", "zig", "lua"}) {
		t.Fatalf("typed tokens: %q", tags)
	}
	tt.Key(0, KeyBackspace)
	if !slices.Equal(tags, []string{"go", "zig"}) {
		t.Fatalf("Backspace: %q", tags)
	}
	tt.Type("ty")
	if !tt.HasText("typescript") {
		t.Fatalf("ty suggests %q", tt.Texts())
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if !slices.Equal(tags, []string{"go", "zig", "typescript"}) {
		t.Fatalf("a suggestion: %q", tags)
	}
	if err := tt.Click("Remove go"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tags, []string{"zig", "typescript"}) {
		t.Errorf("Remove go left %q", tags)
	}
	if !tt.Focused("Tags") {
		t.Error("the input lost the focus")
	}
}

// TestComboboxScrollsPastHighlight scrolls a long popup with the wheel past
// the option the pointer highlighted, which stays where the wheel left it,
// and Down still brings the option it highlights into view.
func TestComboboxScrollsPastHighlight(t *testing.T) {
	var many []string
	for i := range 60 {
		many = append(many, fmt.Sprintf("Option %02d", i))
	}
	choice := ""
	tt := coreNewTester(func(c *context) {
		coreCombobox(c, &choice, many).Label("Pick").Width(200)
	}, 400, 600)
	tt.Click("Pick")
	first, ok := tt.Find("Option 00")
	if !ok {
		t.Fatalf("a click shows %q", tt.Texts())
	}
	// The pointer highlights the first, then the wheel scrolls.
	x, y := first.X+first.W/2, first.Y+first.H/2
	tt.Move(x, y)
	for range 30 {
		tt.Scroll(x, y, 0, 120)
	}
	if r, _ := tt.Find("Option 59"); r.H == 0 || r.Y > first.Y+400 {
		t.Fatalf("scrolled to the end, the last option is at %v; the first showed at %v", r, first)
	}
	// Down highlights the next, which comes into view.
	tt.Key(0, KeyDown)
	if r, _ := tt.Find("Option 01"); r.Y < first.Y-first.H || r.Y > first.Y+200 {
		t.Errorf("Down highlights Option 01 out of view, at %v; the first showed at %v", r, first)
	}
}

// TestSelectBaseScrollsToHighlight moves the highlight of a select whose
// popup scrolls with the keys: the option it highlights comes into view.
func TestSelectBaseScrollsToHighlight(t *testing.T) {
	var many []string
	for i := range 40 {
		many = append(many, fmt.Sprintf("Size %02d", i))
	}
	choice := many[0]
	tt := coreNewTester(func(c *context) {
		sel := coreSelectBase(c, &choice)
		sel.Trigger.Label("Size").Children(func() { coreText(c, choice) })
		sel.Popup(func(panel *node) {
			coreScroll(c).MaxHeight(120).Children(func() {
				for _, s := range many {
					sel.Item(s).Height(20).Children(func() { coreText(c, s) })
				}
			})
		})
	}, 400, 600)
	tt.Click("Size")
	top, ok := tt.Find("Size 01")
	if !ok {
		t.Fatalf("a click shows %q", tt.Texts())
	}
	tt.Key(0, KeyEnd)
	if r, _ := tt.Find("Size 39"); r.Y < top.Y-20 || r.Y+r.H > top.Y-20+120 {
		t.Errorf("End highlights Size 39 out of view, at %v; Size 01 showed at %v", r, top)
	}
}
