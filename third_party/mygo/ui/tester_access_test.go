package ui_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// accessOf returns what assistive technology sees of the element of role
// r named name, failing the test without one.
func accessOf(t *testing.T, tt *ui.Tester, r ui.Role, name string) ui.AccessInfo {
	t.Helper()
	a, ok := tt.AccessByRole(r, name)
	if !ok {
		t.Fatalf("no element of role %d named %q", r, name)
	}
	return a
}

// TestTesterAccess checks what Tester.Access and AccessByRole read of the
// last frame's accessibility tree: roles given and found from what
// elements do, names, values and states.
func TestTesterAccess(t *testing.T) {
	var (
		streaming = true
		agree     bool
		tab       = 1
		volume    = 30.0
		name      = "Ada"
		open      bool
	)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Box(c).Role(ui.RoleStatus).Label("Reply").Busy(streaming)
			ui.Checkbox(c, &agree, "I agree")
			ui.CheckboxBase(c, &agree).Label("Select all").Mixed()
			ui.Tabs(c, &tab, "One", "Two")
			ui.Slider(c, &volume, 0, 100).Label("Volume")
			ui.TextInput(c, &name).Label("Name").Placeholder("Your name")
			ui.Text(c, "Settings").Role(ui.RoleHeading).Level(2)
			ui.Row(c).Role(ui.RoleNavigation).Label("Main").Children(func() {
				ui.Link(c, "Home", "/")
				ui.Link(c, "Docs", "/docs").Current(ui.CurrentPage)
			})
			ui.Row(c).Role(ui.RoleSearch).Label("Site search").Children(func() {
				ui.Button(c, "Go")
			})
			ui.Column(c).Role(ui.RoleArticle).Label("Post by Ann").Children(func() {
				ui.Text(c, "Shipped the beta")
			})
			ui.Collapsible(c, "More", &open, func() { ui.Text(c, "Hidden") })
			ui.Button(c, "Save").Disabled(true)
			ui.Box(c).Size(24, 24).Label("Error").Description("Disk full")
		})
	}, 400, 600)

	reply := accessOf(t, tt, ui.RoleStatus, "Reply")
	if !reply.Busy || reply.Role != ui.RoleStatus {
		t.Errorf("Reply: %+v", reply)
	}
	if a, ok := tt.Access("Reply"); !ok || !reflect.DeepEqual(a, reply) {
		t.Errorf("Access(Reply) = %+v, %v; AccessByRole found %+v", a, ok, reply)
	}
	streaming = false
	tt.Frame()
	if accessOf(t, tt, ui.RoleStatus, "Reply").Busy {
		t.Error("Reply is still busy")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Reply"); ok {
		t.Error("found Reply as a button")
	}
	if _, ok := tt.Access("Nothing"); ok {
		t.Error("found an element named Nothing")
	}

	if box := accessOf(t, tt, ui.RoleCheckBox, "I agree"); box.Checked || box.Mixed || !box.Focusable {
		t.Errorf("I agree: %+v", box)
	}
	if err := tt.Click("I agree"); err != nil {
		t.Fatal(err)
	}
	if box := accessOf(t, tt, ui.RoleCheckBox, "I agree"); !box.Checked || !box.Focused {
		t.Errorf("I agree, clicked: %+v", box)
	}
	if all := accessOf(t, tt, ui.RoleCheckBox, "Select all"); !all.Mixed || all.Checked {
		t.Errorf("Select all: %+v", all)
	}

	// A tab chosen is selected, as aria-selected says, not checked.
	if two := accessOf(t, tt, ui.RoleTab, "Two"); !two.Selected || two.Checked {
		t.Errorf("tab Two: %+v", two)
	}
	if one := accessOf(t, tt, ui.RoleTab, "One"); one.Selected {
		t.Errorf("tab One: %+v", one)
	}

	if v := accessOf(t, tt, ui.RoleSlider, "Volume"); v.Min != 0 || v.Max != 100 || v.Now != 30 {
		t.Errorf("Volume: %+v", v)
	}
	field := accessOf(t, tt, ui.RoleTextField, "Name")
	if field.Value != "Ada" || field.Placeholder != "Your name" || field.Bounds.W <= 0 {
		t.Errorf("Name: %+v", field)
	}
	if r, _ := tt.Find("Name"); r != field.Bounds {
		t.Errorf("Name at %v, Find says %v", field.Bounds, r)
	}
	if h := accessOf(t, tt, ui.RoleHeading, "Settings"); h.Level != 2 {
		t.Errorf("Settings: %+v", h)
	}
	accessOf(t, tt, ui.RoleNavigation, "Main")
	if a, ok := tt.Access("Site search"); !ok || a.Role != ui.RoleSearch {
		t.Errorf("Site search: %+v, %v", a, ok)
	}
	accessOf(t, tt, ui.RoleArticle, "Post by Ann")
	if docs := accessOf(t, tt, ui.RoleLink, "Docs"); docs.Current != ui.CurrentPage {
		t.Errorf("Docs: %+v", docs)
	}
	if home := accessOf(t, tt, ui.RoleLink, "Home"); home.Current != ui.CurrentNone {
		t.Errorf("Home: %+v", home)
	}
	if first := accessOf(t, tt, ui.RoleLink, ""); first.Name != "Home" {
		t.Errorf("the first link is %q", first.Name)
	}
	if more := accessOf(t, tt, ui.RoleDisclosure, "More"); more.Expanded {
		t.Errorf("More: %+v", more)
	}
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if more := accessOf(t, tt, ui.RoleDisclosure, "More"); !more.Expanded {
		t.Errorf("More, opened: %+v", more)
	}
	if save := accessOf(t, tt, ui.RoleButton, "Save"); !save.Disabled || save.Focusable {
		t.Errorf("Save: %+v", save)
	}
	// A box with a Label and nothing else is a group.
	if e := accessOf(t, tt, ui.RoleGroup, "Error"); e.Description != "Disk full" {
		t.Errorf("Error: %+v", e)
	}
}

// TestTesterAccessReadOnly checks that reading the tree changes nothing:
// it builds no frame, and asks for none.
func TestTesterAccessReadOnly(t *testing.T) {
	frames := 0
	tt := ui.NewTester(func(c *ui.Context) {
		frames++
		ui.Button(c, "Save")
	}, 200, 100)
	before := frames
	if _, ok := tt.AccessByRole(ui.RoleButton, "Save"); !ok {
		t.Fatal("no button Save")
	}
	tt.Access("Save")
	if frames != before || tt.FramePending() {
		t.Errorf("reading the tree built %d frames, and asks for more: %v", frames-before, tt.FramePending())
	}
}

// TestTesterAccessRows checks a list's rows: where they are in the list,
// and the one chosen.
func TestTesterAccessRows(t *testing.T) {
	fruit := []string{"Apples", "Pears", "Plums"}
	selected := 0
	list := ui.ListState{Selected: &selected}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.List(c, &list, len(fruit), func(i int) {
			ui.Text(c, fruit[i]).Padding(6, 12)
		}).Grow(1).Label("Fruit")
	}, 300, 300)
	if err := tt.Click("Pears"); err != nil {
		t.Fatal(err)
	}
	if list := accessOf(t, tt, ui.RoleList, "Fruit"); list.SetSize != 3 || !list.Selectable {
		t.Errorf("Fruit: %+v", list)
	}
	pears := accessOf(t, tt, ui.RoleListItem, "Pears")
	if pears.PosInSet != 2 || pears.SetSize != 3 || !pears.Selected || pears.Checked {
		t.Errorf("Pears: %+v", pears)
	}
	if plums := accessOf(t, tt, ui.RoleListItem, "Plums"); plums.Selected {
		t.Errorf("Plums: %+v", plums)
	}
}

// TestTesterAccessStates checks the states built on the others: a role
// description, a popup, what an element controls, a column's sort and a
// grid's cell chosen without painting it.
func TestTesterAccessStates(t *testing.T) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.Column(c).Role(ui.RoleGroup).Label("Highlights").RoleDescription("carousel")
			ui.Button(c, "Fruit").HasPopup(ui.PopupListBox)
			toggle := ui.Button(c, "Details").Expanded(open)
			if open {
				toggle.Controls(ui.Column(c).Role(ui.RoleRegion).Label("More details").Children(func() {
					ui.Text(c, "Hidden")
				}))
			}
			ui.Column(c).Role(ui.RoleTable).Label("November").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Children(func() {
					ui.Text(c, "Day").Role(ui.RoleColumnHeader).Sort(ui.SortDescending)
					ui.Text(c, "Note").Role(ui.RoleColumnHeader).Sort(ui.SortOther)
				})
				ui.Row(c).Role(ui.RoleRow).Children(func() {
					ui.Text(c, "1").Role(ui.RoleCell).Checked(false)
					ui.Text(c, "2").Role(ui.RoleCell).Checked(true)
				})
			})
		})
	}, 400, 400)

	if h := accessOf(t, tt, ui.RoleGroup, "Highlights"); h.RoleDescription != "carousel" {
		t.Errorf("Highlights: %+v", h)
	}
	if f := accessOf(t, tt, ui.RoleButton, "Fruit"); f.HasPopup != ui.PopupListBox || f.RoleDescription != "" {
		t.Errorf("Fruit: %+v", f)
	}
	if d := accessOf(t, tt, ui.RoleButton, "Details"); !slices.Equal(d.Controls, []string{"More details"}) {
		t.Errorf("Details controls %q", d.Controls)
	}
	if day := accessOf(t, tt, ui.RoleColumnHeader, "Day"); day.Sort != ui.SortDescending {
		t.Errorf("Day: %+v", day)
	}
	if note := accessOf(t, tt, ui.RoleColumnHeader, "Note"); note.Sort != ui.SortOther {
		t.Errorf("Note: %+v", note)
	}
	if two := accessOf(t, tt, ui.RoleCell, "2"); !two.Selected || !two.Selectable || two.Checked {
		t.Errorf("cell 2: %+v", two)
	}
	if one := accessOf(t, tt, ui.RoleCell, "1"); one.Selected || !one.Selectable {
		t.Errorf("cell 1: %+v", one)
	}
	if nov := accessOf(t, tt, ui.RoleTable, "November"); !nov.Selectable {
		t.Errorf("November: %+v", nov)
	}

	open = false
	tt.Frame()
	if d := accessOf(t, tt, ui.RoleButton, "Details"); d.Controls != nil {
		t.Errorf("Details, closed, controls %q", d.Controls)
	}
}

// TestTesterAccessRelations checks, through the public API, what a field
// is described by, its error read while invalid, an orientation given, a
// search field of the app's own, and content deleted and inserted.
func TestTesterAccessRelations(t *testing.T) {
	card, query := "", ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			hint := ui.Text(c, "16 digits")
			msg := ui.Text(c, "Required")
			ui.TextInput(c, &card).Label("Card").DescribedBy(hint).Invalid(card == "").ErrorMessage(msg)
			ui.TextInput(c, &query).Label("Search mail").Search()
			ui.Column(c).Role(ui.RoleToolbar).Label("Tools").Orientation(ui.Vertical)
			ui.Row(c).Children(func() {
				ui.Text(c, "$40").Strikethrough().Role(ui.RoleDeletion)
				ui.Text(c, "$30").Role(ui.RoleInsertion)
			})
		})
	}, 400, 400)
	f := accessOf(t, tt, ui.RoleTextField, "Card")
	if !f.Invalid || f.Description != "Required\n16 digits" || !slices.Equal(f.DescribedBy, []string{"16 digits"}) || !slices.Equal(f.ErrorMessage, []string{"Required"}) {
		t.Errorf("Card: %+v", f)
	}
	if s := accessOf(t, tt, ui.RoleTextField, "Search mail"); !s.Search {
		t.Errorf("Search mail: %+v", s)
	}
	if b := accessOf(t, tt, ui.RoleToolbar, "Tools"); !b.Vertical || b.Horizontal {
		t.Errorf("Tools: %+v", b)
	}
	accessOf(t, tt, ui.RoleDeletion, "$40")
	accessOf(t, tt, ui.RoleInsertion, "$30")
}
