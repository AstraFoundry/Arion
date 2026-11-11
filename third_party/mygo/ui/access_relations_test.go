package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestDescribedBy checks aria-describedby (#1181): a hint the view draws
// describes a field after its own description, a description with a
// link reads as one text, one hidden from assistive technology still
// counts, and a Field's or a SearchField's description goes to its
// input.
func TestDescribedBy(t *testing.T) {
	card, email, query, name := "", "", "", ""
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			hint := coreText(c, "16 digits")
			coreTextInput(c, &card).Label("Card").Description("No spaces").DescribedBy(hint).DescribedBy(hint)
			help := coreRow(c).Children(func() {
				coreText(c, "We never share it.")
				coreLink(c, "Privacy", "https://example.com/privacy")
			})
			hidden := coreText(c, "Work address preferred").AccessHidden(true)
			coreTextInput(c, &email).Label("Email").DescribedBy(help).DescribedBy(hidden)
			coreField(c, "Name", func() { coreTextInput(c, &name) }).DescribedBy(coreText(c, "As on your card"))
			coreSearchField(c, &query).Label("Find").DescribedBy(coreText(c, "Titles only"))
		})
	}, 400, 500)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	hint := accessNode(t, tree, platform.RoleText, "16 digits")
	n := accessNode(t, tree, platform.RoleTextField, "Card")
	if n.Description != "No spaces\n16 digits" || !slices.Equal(n.DescribedBy, []uint64{hint.ID}) {
		t.Errorf("the card's description %q, described by %v, want the hint %d", n.Description, n.DescribedBy, hint.ID)
	}
	if n := accessNode(t, tree, platform.RoleTextField, "Email"); n.Description != "We never share it. Privacy Work address preferred" || len(n.DescribedBy) != 2 {
		t.Errorf("the email's description %q, described by %v", n.Description, n.DescribedBy)
	}
	if n := accessNode(t, tree, platform.RoleTextField, "Name"); n.Description != "As on your card" {
		t.Errorf("the field's input's description %q", n.Description)
	}
	if n := accessNode(t, tree, platform.RoleTextField, "Find"); n.Description != "Titles only" || n.States&platform.AccessSearch == 0 {
		t.Errorf("the search field's input: %+v", n)
	}
	info, ok := tt.Access("Card")
	if !ok || !slices.Equal(info.DescribedBy, []string{"16 digits"}) {
		t.Errorf("Tester.Access: %+v", info)
	}
}

// TestInvalidErrorMessage checks aria-invalid without a message and
// aria-errormessage (#1181): Invalid marks a control without touching
// its description, Invalid(false) leaves Error's mark, and an element
// holding the error is read first, and pointed at, only while the
// control is invalid.
func TestInvalidErrorMessage(t *testing.T) {
	card, email, city := "", "", ""
	touched, bad := false, false
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			coreTextInput(c, &card).Label("Card").Description("16 digits").Invalid(touched && card == "")
			msg := coreText(c, "Enter an email")
			coreTextInput(c, &email).Label("Email").Description("For receipts").Invalid(bad).ErrorMessage(msg)
			coreField(c, "City", func() { coreTextInput(c, &city) }).Error("Required").Invalid(false)
		})
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	if n := accessNode(t, tree, platform.RoleTextField, "Card"); n.States&platform.AccessInvalid != 0 {
		t.Errorf("the card is invalid before it was touched")
	}
	n := accessNode(t, tree, platform.RoleTextField, "Email")
	if n.States&platform.AccessInvalid != 0 || n.Description != "For receipts" || n.ErrorMessage != nil {
		t.Errorf("the valid email: %+v", n)
	}
	if n := accessNode(t, tree, platform.RoleTextField, "City"); n.States&platform.AccessInvalid == 0 || n.Description != "Required" {
		t.Errorf("Invalid(false) took Error's mark back: %+v", n)
	}
	touched, bad = true, true
	tt.Frame()
	tree = tt.h.access
	if n := accessNode(t, tree, platform.RoleTextField, "Card"); n.States&platform.AccessInvalid == 0 || n.Description != "16 digits" {
		t.Errorf("the empty card touched: %+v", n)
	}
	msg := accessNode(t, tree, platform.RoleText, "Enter an email")
	n = accessNode(t, tree, platform.RoleTextField, "Email")
	if n.States&platform.AccessInvalid == 0 || n.Description != "Enter an email\nFor receipts" || !slices.Equal(n.ErrorMessage, []uint64{msg.ID}) {
		t.Errorf("the invalid email: %+v", n)
	}
	info, ok := tt.Access("Email")
	if !ok || !info.Invalid || !slices.Equal(info.ErrorMessage, []string{"Enter an email"}) {
		t.Errorf("Tester.Access: %+v", info)
	}
	if info, _ := tt.Access("Card"); !info.Invalid || info.ErrorMessage != nil {
		t.Errorf("Tester.Access of the card: %+v", info)
	}
}

// TestOrientation checks aria-orientation on elements of your own
// (#1181): a slider, a toolbar and a separator say which way they run,
// in place of what they would say, and a value of neither way takes it
// back.
func TestOrientation(t *testing.T) {
	var v float64
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			coreBox(c).Size(8, 100).Focusable().Role(RoleSlider).Label("Volume").Range(0, 10, 3).Orientation(Vertical)
			coreBox(c).Size(100, 8).Focusable().Role(RoleSlider).Label("Pan").Range(0, 10, 3)
			coreColumn(c).Role(RoleToolbar).Label("Tools").Orientation(Vertical)
			coreBox(c).Size(2, 40).Role(RoleSeparator).Label("Rule").Orientation(Horizontal)
			coreBox(c).Size(2, 40).Role(RoleSeparator).Label("Tall").Orientation(Horizontal | Vertical)
			coreSliderBase(c, &v, 0, 1).Size(8, 100).Label("Up").Vertical()
			coreRow(c).Role(RoleGroup).Label("Plain")
		})
	}, 400, 500)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	ways := platform.AccessVertical | platform.AccessHorizontal
	for _, c := range []struct {
		role  platform.AccessRole
		label string
		want  platform.AccessStates
	}{
		{platform.RoleSlider, "Volume", platform.AccessVertical},
		{platform.RoleSlider, "Pan", 0},
		{platform.RoleToolbar, "Tools", platform.AccessVertical},
		{platform.RoleSeparator, "Rule", platform.AccessHorizontal}, // taller than wide, but told
		{platform.RoleSeparator, "Tall", platform.AccessVertical},   // from its shape
		{platform.RoleSlider, "Up", platform.AccessVertical},
		{platform.RoleGroup, "Plain", 0},
	} {
		if n := accessNode(t, tree, c.role, c.label); n.States&ways != c.want {
			t.Errorf("%s runs %b, want %b", c.label, n.States&ways, c.want)
		}
	}
	if info, _ := tt.Access("Rule"); !info.Horizontal || info.Vertical {
		t.Errorf("Tester.Access: %+v", info)
	}
}

// TestDeletionInsertion checks ARIA's deletion and insertion roles
// (#1181): a price struck out and its new one, and the lines of a diff,
// whose texts show inside them.
func TestDeletionInsertion(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			coreRow(c).Gap(6).Children(func() {
				coreText(c, "$40").Strikethrough().Role(RoleDeletion)
				coreText(c, "$30").Role(RoleInsertion)
			})
			coreColumn(c).Role(RoleDeletion).Children(func() { coreText(c, "- old line") })
			coreColumn(c).Role(RoleInsertion).Children(func() { coreText(c, "+ new line") })
		})
	}, 400, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	accessNode(t, tree, platform.RoleDeletion, "$40")
	accessNode(t, tree, platform.RoleInsertion, "$30")
	for role, line := range map[platform.AccessRole]string{platform.RoleDeletion: "- old line", platform.RoleInsertion: "+ new line"} {
		text := accessNode(t, tree, platform.RoleText, line)
		if p := text.Parent; p < 0 || tree.Nodes[p].Role != role || tree.Nodes[p].Label != "" {
			t.Errorf("%q is not inside a %d of its own", line, role)
		}
	}
	if info, ok := tt.AccessByRole(RoleDeletion, "$40"); !ok || info.Role != RoleDeletion {
		t.Errorf("Tester.AccessByRole: %+v", info)
	}
	if info, ok := tt.AccessByRole(RoleInsertion, "$30"); !ok || info.Role != RoleInsertion {
		t.Errorf("Tester.AccessByRole: %+v", info)
	}
}

// TestSearch checks that a text input of an app's own asks to be a
// search field (searchbox), as SearchField's input is, while it calls
// Search.
func TestSearch(t *testing.T) {
	query, note := "", ""
	search := true
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			in := coreTextInput(c, &query).Label("Search mail")
			if search {
				in.Search()
			}
			coreTextInput(c, &note).Label("Note")
			coreBox(c).Label("Box").Search() // no text input: nothing
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleTextField, "Search mail"); n.States&platform.AccessSearch == 0 {
		t.Error("the input calling Search is no search field")
	}
	if n := accessNode(t, tt.h.access, platform.RoleTextField, "Note"); n.States&platform.AccessSearch != 0 {
		t.Error("another input is a search field")
	}
	if n := accessNode(t, tt.h.access, platform.RoleGroup, "Box"); n.States&platform.AccessSearch != 0 {
		t.Error("a box is a search field")
	}
	if info, _ := tt.Access("Search mail"); !info.Search {
		t.Errorf("Tester.Access: %+v", info)
	}
	search = false
	tt.Frame()
	if n := accessNode(t, tt.h.access, platform.RoleTextField, "Search mail"); n.States&platform.AccessSearch != 0 {
		t.Error("the input that stopped calling Search is still a search field")
	}
}
