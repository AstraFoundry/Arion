package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// listBoxDump describes the nodes labeled labels, in that order, as the
// repro of #1172 dumps them: the platform's role, and whether each is
// selectable, selected (checked) and focusable for assistive technology.
func listBoxDump(tree *platform.AccessTree, labels ...string) string {
	roles := map[platform.AccessRole]string{
		platform.RoleList: "list", platform.RoleListItem: "listitem", platform.RolePopup: "popup",
		platform.RoleGroup: "group", platform.RoleComboBox: "combobox", platform.RolePopUpButton: "popupbutton",
		platform.RoleTextField: "textfield",
	}
	var b strings.Builder
	for _, l := range labels {
		n, ok := platform.AccessNode{}, false
		for _, m := range tree.Nodes {
			if m.Label == l {
				n, ok = m, true
			}
		}
		if !ok {
			fmt.Fprintf(&b, "%-14q missing\n", l)
			continue
		}
		fmt.Fprintf(&b, "%-14q %s", l, roles[n.Role])
		for _, s := range []struct {
			on   bool
			name string
		}{
			{n.States&platform.AccessSelectable != 0, "selectable"},
			{n.States&platform.AccessChecked != 0, "selected"},
			{n.States&platform.AccessFocusable != 0, "focusable"},
			{n.ID == tree.Focus, "focus"},
		} {
			if s.on {
				b.WriteString(" " + s.name)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// TestListBoxSemantics dumps the access tree of a view holding a
// ComboboxBase, a SelectBase and a list of options built by hand
// (RoleListBox, RoleOption) that a text area points at with
// ActiveDescendant, as a suggestion list at the caret: each is a list
// box of options, its selected option the one highlighted (#1172).
func TestListBoxSemantics(t *testing.T) {
	font, size, message := "", "b", ""
	own, ownClicks := 0, 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(8).Children(func() {
			cb := coreComboboxBase(c, &font)
			cb.Input.Label("font").Width(200)
			cb.Popup(func(panel *node) {
				panel.Label("combo-panel")
				for _, v := range []string{"ca", "cb"} {
					cb.Item(v).Label("opt-" + v).Children(func() { coreText(c, v) })
				}
			})
			sel := coreSelectBase(c, &size)
			sel.Trigger.Label("size").Padding(4).Children(func() { coreText(c, "Size "+size) })
			sel.Popup(func(panel *node) {
				panel.Label("select-panel")
				for _, v := range []string{"a", "b", "c"} {
					sel.Item(v).Label("opt-" + v).Padding(4).Children(func() { coreText(c, v) })
				}
			})
			area := coreTextAreaBase(c, &message).Label("message").Width(200).Height(40)
			coreColumn(c).Role(RoleListBox).Label("own-list").Children(func() {
				for i, v := range []string{"x", "y"} {
					opt := coreRow(c).Role(RoleOption).Label("own-" + v).Checked(i == own)
					if opt.Clicked() {
						own, ownClicks = i, ownClicks+1
					}
					opt.Children(func() { coreText(c, v) })
					if i == own {
						area.ActiveDescendant(opt)
					}
				}
			})
		})
	}, 400, 600)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})

	// The list built by hand: a list box, whose options say which one is
	// selected, which the text area with the focus has it for. They take
	// no focus of their own: the text area keeps it.
	tt.Click("message")
	want := `"message"      textfield focusable
"own-list"     list selectable
"own-x"        listitem selectable selected focus
"own-y"        listitem selectable
`
	if got := listBoxDump(tt.h.access, "message", "own-list", "own-x", "own-y"); got != want {
		t.Errorf("the list built by hand:\n%s\nwant:\n%s", got, want)
	}
	if n := accessNode(t, tt.h.access, platform.RoleListItem, "own-y"); n.Actions&platform.ActionFocus != 0 || n.Actions&platform.ActionPress == 0 {
		t.Errorf("an option built by hand: %+v", n)
	}
	// Pressing one, as Windows' SelectionItem.Select and ATK's
	// add_selection do, clicks it.
	y := accessNode(t, tt.h.access, platform.RoleListItem, "own-y")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: y.ID, Action: platform.AccessPress})
	if own != 1 || ownClicks != 1 || tt.h.access.Focus != y.ID {
		t.Errorf("pressing own-y: chose %d with %d clicks, focus on %d", own, ownClicks, tt.h.access.Focus)
	}

	// ComboboxBase, unchanged.
	tt.Click("font")
	tt.Key(0, KeyDown)
	want = `"combo-panel"  list selectable
"opt-ca"       listitem selectable selected focusable focus
"opt-cb"       listitem selectable focusable
`
	if got := listBoxDump(tt.h.access, "combo-panel", "opt-ca", "opt-cb"); got != want {
		t.Errorf("the combobox:\n%s\nwant:\n%s", got, want)
	}

	tt.Key(0, KeyEscape) // out of the select's way

	// SelectBase: a list box of options, the one highlighted selected,
	// which the trigger keeping the focus has it for, and which says
	// where it is. The trigger's value is the choice.
	tt.Click("size")
	want = `"size"         popupbutton focusable
"select-panel" list selectable
"opt-a"        listitem selectable focusable
"opt-b"        listitem selectable selected focusable focus
"opt-c"        listitem selectable focusable
`
	if got := listBoxDump(tt.h.access, "size", "select-panel", "opt-a", "opt-b", "opt-c"); got != want {
		t.Errorf("the select, open:\n%s\nwant:\n%s", got, want)
	}
	if n := accessNode(t, tt.h.access, platform.RoleListItem, "opt-b"); n.PosInSet != 2 || n.SetSize != 3 {
		t.Errorf("opt-b is %d of %d", n.PosInSet, n.SetSize)
	}
	// Moving the highlight moves the selected option, and chooses nothing.
	tt.Key(0, KeyUp)
	want = `"size"         popupbutton focusable
"opt-a"        listitem selectable selected focusable focus
"opt-b"        listitem selectable focusable
`
	if got := listBoxDump(tt.h.access, "size", "opt-a", "opt-b"); got != want || size != "b" {
		t.Errorf("Up chose %q:\n%s\nwant:\n%s", size, got, want)
	}
	if n := accessNode(t, tt.h.access, platform.RolePopUpButton, "size"); n.Value != "b" || n.States&platform.AccessExpanded == 0 {
		t.Errorf("the trigger: %+v", n)
	}
	// Assistive technology focusing an option highlights it, as the
	// arrows do: the popup stays open, the choice as it was, and Enter
	// chooses it.
	optC := accessNode(t, tt.h.access, platform.RoleListItem, "opt-c")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: optC.ID, Action: platform.AccessFocus})
	want = `"opt-a"        listitem selectable focusable
"opt-c"        listitem selectable selected focusable focus
`
	if got := listBoxDump(tt.h.access, "opt-a", "opt-c"); got != want || size != "b" || !tt.HasText("Size b") {
		t.Errorf("focusing opt-c chose %q:\n%s\nwant:\n%s", size, got, want)
	}
	tt.Key(0, KeyEnter)
	if _, open := byID(tt.h.access, optC.ID); size != "c" || open {
		t.Errorf("Enter after focusing opt-c chose %q, open %v", size, open)
	}
	// Pressing one chooses it, as a click does.
	tt.Click("size")
	optA := accessNode(t, tt.h.access, platform.RoleListItem, "opt-a")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: optA.ID, Action: platform.AccessPress})
	if size != "a" {
		t.Errorf("pressing opt-a chose %q", size)
	}
	if _, ok := byID(tt.h.access, optA.ID); ok {
		t.Error("pressing opt-a leaves the popup open")
	}
}
