package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// node returns the node of the tree with role and a label containing
// label.
func accessNode(t *testing.T, tree *platform.AccessTree, role platform.AccessRole, label string) platform.AccessNode {
	t.Helper()
	for _, n := range tree.Nodes {
		if n.Role == role && strings.Contains(n.Label, label) {
			return n
		}
	}
	var have []string
	for _, n := range tree.Nodes {
		have = append(have, n.Label)
	}
	t.Fatalf("no node of role %d labeled %q among %q", role, label, have)
	return platform.AccessNode{}
}

func TestAccessibilityTree(t *testing.T) {
	d := &demo{name: "Ada", volume: 30}
	tt := coreNewTester(d.view, 640, 600)
	if tt.h.access != nil {
		t.Fatal("a tree before assistive technology asked")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	if tree == nil || len(tree.Nodes) == 0 {
		t.Fatal("no tree after AccessibilityOn")
	}
	inc := accessNode(t, tree, platform.RoleButton, "Increment")
	if inc.Actions&platform.ActionPress == 0 || inc.Bounds.W <= 0 {
		t.Errorf("Increment: %+v", inc)
	}
	accessNode(t, tree, platform.RoleText, "MyGo UI")
	agree := accessNode(t, tree, platform.RoleCheckBox, "I agree")
	if agree.States&platform.AccessChecked != 0 {
		t.Error("the check box is checked")
	}
	accessNode(t, tree, platform.RoleSwitch, "")
	accessNode(t, tree, platform.RoleRadio, "Alpha")
	slider := accessNode(t, tree, platform.RoleSlider, "")
	if slider.Min != 0 || slider.Max != 100 || slider.Now != 30 {
		t.Errorf("slider range %v..%v at %v", slider.Min, slider.Max, slider.Now)
	}
	field := accessNode(t, tree, platform.RoleTextField, "")
	if field.Value != "Ada" || field.Actions&platform.ActionSetValue == 0 {
		t.Errorf("text field: %+v", field)
	}
	accessNode(t, tree, platform.RolePopUpButton, "")
	list := accessNode(t, tree, platform.RoleList, "")
	row := accessNode(t, tree, platform.RoleText, "Row 3")
	for p := row.Parent; ; p = tree.Nodes[p].Parent {
		if p < 0 {
			t.Fatal("the row is not inside the list")
		}
		if tree.Nodes[p].ID == list.ID {
			break
		}
	}
	// Buttons name themselves after their text, which is not a node.
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleText && n.Label == "Increment" {
			t.Error("the button's text is a node of its own")
		}
	}

	// Actions act as the pointer and the keyboard would.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: inc.ID, Action: platform.AccessPress})
	if d.count != 1 {
		t.Errorf("pressing Increment: count %d", d.count)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: agree.ID, Action: platform.AccessPress})
	if !d.agree {
		t.Error("pressing the check box did not check it")
	}
	if n := accessNode(t, tt.h.access, platform.RoleCheckBox, "I agree"); n.States&platform.AccessChecked == 0 {
		t.Error("the tree after the frame shows the check box unchecked")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: slider.ID, Action: platform.AccessIncrement})
	if d.volume <= 30 {
		t.Errorf("incrementing the slider: %v", d.volume)
	}
	if tt.h.access.Focus != slider.ID {
		t.Error("the slider has no focus after incrementing it")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "Grace"})
	if d.name != "Grace" {
		t.Errorf("setting the text field: %q", d.name)
	}
	if f := accessNode(t, tt.h.access, platform.RoleTextField, ""); f.Value != "Grace" || f.SelStart != 5 || f.SelEnd != 5 || tt.h.access.Focus != f.ID {
		t.Errorf("text field after setting it: %+v, focus %d", f, tt.h.access.Focus)
	}
	// Inserting text replaces the selection, as typing does: what
	// assistive technology and automation type with AXSelectedText.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSelectText, From: 0, To: 5})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessInsertText, Text: "Ada"})
	if d.name != "Ada" {
		t.Errorf("inserting over the selection: %q", d.name)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessInsertText, Text: " L."})
	if f := accessNode(t, tt.h.access, platform.RoleTextField, ""); d.name != "Ada L." || f.SelStart != 6 || f.SelEnd != 6 {
		t.Errorf("inserting at the caret: %q, %+v", d.name, f)
	}
}

// byID returns the node of the tree with id.
func byID(tree *platform.AccessTree, id uint64) (platform.AccessNode, bool) {
	for _, n := range tree.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return platform.AccessNode{}, false
}

func TestAccessibilityOfLists(t *testing.T) {
	sel := -1
	s := ListState{Selected: &sel}
	tt := coreNewTester(func(c *context) {
		coreButton(c, "Before")
		coreList(c, &s, 1000, func(i int) { coreTextf(c, "Row %d", i).Height(30) }).Grow(1)
	}, 300, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	// Each row says which of all it is, and that it can be chosen.
	item := accessNode(t, tree, platform.RoleListItem, "Row 3")
	if item.PosInSet != 4 || item.SetSize != 1000 {
		t.Errorf("row 3 is %d of %d", item.PosInSet, item.SetSize)
	}
	if item.States&(platform.AccessSelectable|platform.AccessFocusable) != platform.AccessSelectable|platform.AccessFocusable ||
		item.Actions&(platform.ActionPress|platform.ActionFocus|platform.ActionScrollIntoView) == 0 {
		t.Errorf("row 3: %+v", item)
	}
	accessNode(t, tree, platform.RoleText, "Row 3") // its content shows too
	if b := accessNode(t, tree, platform.RoleButton, "Before"); b.Actions&platform.ActionScrollIntoView != 0 {
		t.Error("a button out of any scroll container scrolls into view")
	}
	// Focusing a row chooses it, the list taking the focus: assistive
	// technology follows the choice as the arrows move it.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: item.ID, Action: platform.AccessFocus})
	tree = tt.h.access
	if sel != 3 || tree.Focus != item.ID || tt.rt.focused != s.frame.e.id {
		t.Fatalf("focused row 3: chose %d, focus on %d (row %d)", sel, tree.Focus, item.ID)
	}
	if n, _ := byID(tree, item.ID); n.States&platform.AccessChecked == 0 {
		t.Error("the chosen row is not chosen")
	}
	tt.Key(0, KeyDown)
	tree = tt.h.access
	if n, ok := byID(tree, tree.Focus); !ok || n.PosInSet != 5 || n.States&platform.AccessChecked == 0 {
		t.Errorf("Down: the focus is on %+v", n)
	}
	// The last row built, out of view, scrolls into view when assistive
	// technology asks, and the rows after it come.
	last := platform.AccessNode{}
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleListItem && n.PosInSet > last.PosInSet {
			last = n
		}
	}
	if last.Bounds.Y+last.Bounds.H <= 400 {
		t.Fatalf("the last row built shows: %+v", last)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: last.ID, Action: platform.AccessScrollIntoView})
	tree = tt.h.access
	if n, ok := byID(tree, last.ID); !ok || n.Bounds.Y+n.Bounds.H > 400.01 {
		t.Errorf("scrolled into view: %+v (%v)", n, ok)
	}
	beyond := false
	for _, n := range tree.Nodes {
		beyond = beyond || n.Role == platform.RoleListItem && n.PosInSet > last.PosInSet
	}
	if !beyond {
		t.Error("no row after it came")
	}
}

func TestAccessibilityScrollsToARowNoLongerBuilt(t *testing.T) {
	// A frame that scrolled to a row built more rows around it than the
	// next builds: assistive technology may ask for one of those.
	var s ListState
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 1000, func(i int) { coreTextf(c, "Row %d", i).Height(30) }).Grow(1)
	}, 300, 400)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	s.ScrollTo(20, Center)
	tt.Frame()
	tree := tt.h.access
	last := platform.AccessNode{}
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleListItem && n.PosInSet > last.PosInSet {
			last = n
		}
	}
	// The next frame builds the rows of the place alone.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: last.ID, Action: platform.AccessScrollIntoView})
	if r, ok := rowBox(tt, &s, last.PosInSet-1); !ok || r.Y < 0 || r.Y+r.H > 400.01 {
		t.Errorf("row %d after asking to see it: %v (%v)", last.PosInSet-1, r, ok)
	}
}

func TestAccessibilityOfTables(t *testing.T) {
	sel := 2
	s := ListState{Selected: &sel}
	cols := []TableColumn{{Title: "Name"}, {Title: "Size", Width: 80}}
	tt := coreNewTester(func(c *context) {
		coreTable(c, &s, cols, 500, func(row, col int) { coreTextf(c, "Cell %d %d", row, col) }).Grow(1)
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	rows := 0
	for _, n := range tree.Nodes {
		if n.Role != platform.RoleRow {
			continue
		}
		if rows++; rows == 1 {
			if n.PosInSet != 0 {
				t.Errorf("the header is row %d", n.PosInSet)
			}
			continue
		}
		if n.SetSize != 500 || n.States&platform.AccessSelectable == 0 {
			t.Errorf("a row: %+v", n)
		}
		// A row holds its cells, not another row.
		for _, m := range tree.Nodes {
			if m.Parent >= 0 && tree.Nodes[m.Parent].ID == n.ID && m.Role != platform.RoleCell {
				t.Errorf("row %d holds a node of role %d", n.PosInSet, m.Role)
			}
		}
	}
	if rows < 5 {
		t.Errorf("%d rows", rows)
	}
	// The table has the focus for its rows.
	tt.Click("Cell 4 0")
	tree = tt.h.access
	if n, ok := byID(tree, tree.Focus); !ok || n.Role != platform.RoleRow || n.PosInSet != 5 {
		t.Errorf("clicked row 4: the focus is on %+v", n)
	}
}

// TestColumnHeaderShowsItsChildren checks that a column header keeps the
// buttons inside it in the tree, as ARIA's columnheader, whose children
// are not presentational, and is still named by its content (#1275).
func TestColumnHeaderShowsItsChildren(t *testing.T) {
	sorted := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Role(RoleColumnHeader).Children(func() {
			if coreButton(c, "Sort").Clicked() {
				sorted++
			}
			coreButton(c, "Options")
		})
	}, 300, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	head := accessNode(t, tree, platform.RoleColumnHeader, "Sort Options")
	if head.Label != "Sort Options" {
		t.Errorf("the header is named %q", head.Label)
	}
	for _, name := range []string{"Sort", "Options"} {
		b := accessNode(t, tree, platform.RoleButton, name)
		if b.Parent < 0 || tree.Nodes[b.Parent].ID != head.ID {
			t.Errorf("%s is not inside the header", name)
		}
		if b.States&platform.AccessFocusable == 0 || b.Actions&platform.ActionPress == 0 {
			t.Errorf("%s: %+v", name, b)
		}
	}
	sort := accessNode(t, tree, platform.RoleButton, "Sort")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: sort.ID, Action: platform.AccessPress})
	if sorted != 1 {
		t.Errorf("pressing Sort clicked it %d times", sorted)
	}
	// A table's headers are named by their titles alone, with nothing
	// inside them.
	s := ListState{}
	cols := []TableColumn{{Title: "Name", Sortable: true}, {Title: "Kind"}}
	tt = coreNewTester(tableView(&s, cols, []string{"a.txt"}), 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree = tt.h.access
	for _, title := range []string{"Name", "Kind"} {
		h := accessNode(t, tree, platform.RoleColumnHeader, title)
		for _, n := range tree.Nodes {
			if n.Parent >= 0 && tree.Nodes[n.Parent].ID == h.ID {
				t.Errorf("the header %s holds %+v", title, n)
			}
		}
	}
}

// TestHeadingShowsItsChildren checks that a heading exposes the button
// inside it, as an accordion header does, while keeping its name and level.
func TestHeadingShowsItsChildren(t *testing.T) {
	open := false
	label := ""
	tt := coreNewTester(func(c *context) {
		coreRow(c).Role(RoleHeading).Level(3).Label(label).Children(func() {
			if coreButton(c, "Privacy").Expanded(open).Clicked() {
				open = !open
			}
		})
	}, 300, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	heading := accessNode(t, tree, platform.RoleHeading, "Privacy")
	if heading.Label != "Privacy" || heading.Level != 3 {
		t.Errorf("heading: %+v", heading)
	}
	button := accessNode(t, tree, platform.RoleButton, "Privacy")
	if button.Parent < 0 || tree.Nodes[button.Parent].ID != heading.ID {
		t.Error("the button is not inside the heading")
	}
	if button.States&platform.AccessFocusable == 0 || button.Actions&platform.ActionPress == 0 {
		t.Errorf("button: %+v", button)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: button.ID, Action: platform.AccessPress})
	if !open {
		t.Error("pressing the button did not open the section")
	}
	// An explicit name replaces the content's name without hiding it.
	label = "Privacy settings"
	tt.Frame()
	tree = tt.h.access
	heading = accessNode(t, tree, platform.RoleHeading, label)
	if heading.Label != label || heading.Level != 3 {
		t.Errorf("named heading: %+v", heading)
	}
	button = accessNode(t, tree, platform.RoleButton, "Privacy")
	if button.States&platform.AccessExpanded == 0 {
		t.Error("the button is not expanded after opening the section")
	}
}

func TestAccessibilityOfOverlays(t *testing.T) {
	open := true
	tt := coreNewTester(func(c *context) {
		coreButton(c, "Behind")
		coreModal(c, &open, func() {
			coreText(c, "Sure?")
			coreButton(c, "OK")
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	dialog := accessNode(t, tree, platform.RoleDialog, "")
	ok := accessNode(t, tree, platform.RoleButton, "OK")
	if tree.Nodes[ok.Parent].ID != dialog.ID {
		t.Error("the dialog's button is not inside it")
	}
}

func TestRole(t *testing.T) {
	on := false
	tt := coreNewTester(func(c *context) {
		b := coreBox(c).Size(20, 20).Focusable().Role(RoleSwitch).Label("Wi-Fi")
		if b.Clicked() {
			on = !on
		}
		coreBox(c).Size(20, 20).Label("hidden").Role(RoleNone)
	}, 100, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	sw := accessNode(t, tt.h.access, platform.RoleSwitch, "Wi-Fi")
	for _, n := range tt.h.access.Nodes {
		if n.Label == "hidden" {
			t.Error("an element of RoleNone is in the tree")
		}
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: sw.ID, Action: platform.AccessPress})
	if !on {
		t.Error("pressing the custom switch did not click it")
	}
}

// TestInputMethodContext checks that input methods see the text around
// the caret and replace what they typed, as macOS's press and hold does.
func TestInputMethodContext(t *testing.T) {
	d := &demo{name: "caf"}
	tt := coreNewTester(d.view, 640, 600)
	r, _ := tt.Find("I agree")
	tt.ClickAt(r.X+20, r.Y+40+r.H/2)
	tt.Key(0, KeyEnd)
	ime := tt.h.ime
	ctx := platform.ClientTextContext(ime.Client)
	if !ime.Active || ime.Client == nil || ctx.Text != "caf" || ctx.Start != 3 || ctx.End != 3 {
		t.Fatalf("text input state %+v", ime)
	}
	// Typing e, then holding it: the input method composes over the e
	// it typed, then commits the accented letter in its place.
	tt.Type("e")
	if platform.ClientTextContext(tt.h.ime.Client).Text != "cafe" {
		t.Fatalf("after typing: %+v", tt.h.ime)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.TextComposition, Text: "e", Caret: 1, Replace: true, From: 3, To: 4})
	if d.name != "caf" {
		t.Errorf("composing over the e: %q", d.name)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.TextInput, Text: "é"})
	if d.name != "café" {
		t.Errorf("committing: %q", d.name)
	}
	// Client replacements use absolute UTF-16 document offsets, even beyond
	// the surrounding window supplied to GTK and IMM32.
	d.name = strings.Repeat("x", 2*imeContext) + "ab"
	tt.Frame()
	tt.Key(0, KeyEnd)
	ime = tt.h.ime
	ctx = platform.ClientTextContext(ime.Client)
	if !strings.HasSuffix(ctx.Text, "xab") || ime.Client.Selection().Caret() != len(d.name) {
		t.Fatalf("long text: %+v", ctx)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.TextInput, Text: "B", Replace: true, From: len(d.name) - 1, To: len(d.name)})
	if !strings.HasSuffix(d.name, "xaB") {
		t.Errorf("replacing in a long text: ...%q", d.name[len(d.name)-5:])
	}
}

// BenchmarkAccessTree describes a frame of many clickable cards of two
// texts each, as the content does every frame once assistive technology
// asked for the window.
func BenchmarkAccessTree(b *testing.B) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			for i := range 400 {
				card := coreColumn(c).Padding(8)
				card.Clicked()
				card.Children(func() {
					coreTextf(c, "Game %d", i)
					coreText(c, "Studio")
				})
			}
		})
	}, 640, 20000)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	b.ReportAllocs()
	for b.Loop() {
		tt.rt.accessTree()
	}
}

// TestRowNamedByLabels: a list's row is named by its content, an element
// inside it with a Label by the label rather than its texts, as the web
// names from content with a descendant's aria-label.
func TestRowNamedByLabels(t *testing.T) {
	var ls ListState
	tt := NewTester(func(c *Context) {
		List(c, &ls, 1, func(i int) {
			Row(c).Children(func() {
				Box(c).Label("Done").Children(func() { Text(c, "✓") })
				Text(c, "Write the report")
			})
		}).Fill()
	}, 320, 240)
	tt.rt.access = true
	tt.Frame()
	for _, n := range tt.rt.accessTree().Nodes {
		if n.Role == platform.RoleListItem {
			if n.Label != "Done Write the report" {
				t.Errorf("row named %q, want %q", n.Label, "Done Write the report")
			}
			return
		}
	}
	t.Error("no row for assistive technology")
}
