package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// testNode is an item of a tree a TreeBase test lays out itself.
type testNode struct {
	name string
	open *bool
	kids []*testNode
}

// TestTreeBase builds a tree of its own rows on TreeBase: the keys of a
// tree move the focus and open and close its items, the tree is one stop
// of Tab, and assistive technology sees its items' levels, states and
// places.
func TestTreeBase(t *testing.T) {
	srcOpen, cmdOpen := false, false
	roots := []*testNode{
		{name: "src", open: &srcOpen, kids: []*testNode{
			{name: "main.go"},
			{name: "cmd", open: &cmdOpen, kids: []*testNode{{name: "tool.go"}}},
			{name: "mux.go"},
		}},
		{name: "go.mod"},
		{name: "go.sum"},
	}
	depths := map[string]int{}
	groups := map[string]bool{}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreButton(c, "Before")
			tree := coreTreeBase(c)
			tree.Root.Gap(2).Children(func() {
				var build func(n *testNode)
				build = func(n *testNode) {
					c.nextKey, c.keySet = n.name, true
					item := tree.Item(n.open)
					depths[n.name] = item.Depth()
					item.Row.Height(20).PaddingStart(16 * float32(item.Depth())).Children(func() {
						arrow := coreBox(c).Size(16, 16).Label(n.name + " arrow")
						if n.open != nil && arrow.Clicked() {
							*n.open = !*n.open
						}
						coreText(c, n.name)
					})
					groups[n.name] = item.Group(func() {
						for _, k := range n.kids {
							build(k)
						}
					}) != nil
				}
				for _, n := range roots {
					build(n)
				}
			})
			coreButton(c, "After")
		})
	}, 400, 400)
	if tt.HasText("main.go") || groups["src"] {
		t.Fatalf("a closed item shows its children: %q", tt.Texts())
	}
	// One stop of Tab, which goes to the item that had the focus last.
	tt.Click("Before")
	tt.Key(0, KeyTab)
	if !tt.Focused("src") {
		t.Fatal("Tab from Before did not go to the first item")
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyTab)
	if !tt.Focused("After") {
		t.Fatal("Tab from an item did not leave the tree")
	}
	tt.Key(Shift, KeyTab)
	if !tt.Focused("go.mod") {
		t.Fatal("Shift-Tab did not go back to the item that had the focus")
	}
	// The focus follows a keyed item as an item above it opens.
	srcOpen = true
	tt.Frame()
	if !groups["src"] || !tt.HasText("mux.go") {
		t.Fatalf("src opened shows %q", tt.Texts())
	}
	if !tt.Focused("go.mod") {
		t.Fatal("go.mod lost the focus as src opened above it")
	}
	// A click on an arrow of the app's own closes it.
	tt.Click("src arrow")
	if srcOpen || tt.HasText("mux.go") {
		t.Fatal("a click on src's arrow left it open")
	}
	tt.Click("src arrow")
	if depths["src"] != 0 || depths["cmd"] != 1 || depths["go.mod"] != 0 {
		t.Errorf("depths %v", depths)
	}
	steps := []struct {
		key   Key
		focus string
	}{
		{KeyHome, "src"},
		{KeyUp, "src"},
		{KeyDown, "main.go"},
		{KeyDown, "cmd"},
		{KeyRight, "cmd"}, // opens it
		{KeyRight, "tool.go"},
		{KeyRight, "tool.go"},
		{KeyLeft, "cmd"},
		{KeyLeft, "cmd"}, // closes it
		{KeyDown, "mux.go"},
		{KeyLeft, "src"},
		{KeyEnd, "go.sum"},
		{KeyDown, "go.sum"},
	}
	for _, step := range steps {
		tt.Key(0, step.key)
		if !tt.Focused(step.focus) {
			t.Fatalf("%v did not focus %s", step.key, step.focus)
		}
	}
	if cmdOpen || tt.HasText("tool.go") {
		t.Error("Left on the open cmd left it open")
	}
	// Typing the first letters: the same letter again goes on to the next
	// item, more letters to the first starting with them, round.
	for _, step := range []struct {
		key   Key
		focus string
	}{{KeyM, "main.go"}, {KeyM, "mux.go"}, {KeyM, "main.go"}} {
		tt.Key(0, step.key)
		if !tt.Focused(step.focus) {
			t.Fatalf("typing %v did not focus %s", step.key, step.focus)
		}
	}
	tt.Advance(2 * typePause)
	tt.Key(0, KeyG)
	tt.Key(0, KeyO)
	if !tt.Focused("go.mod") {
		t.Fatal("typing go did not focus go.mod")
	}
	tt.Advance(2 * typePause)
	tt.Key(0, KeyM)
	tt.Key(0, KeyU)
	if !tt.Focused("mux.go") {
		t.Fatal("typing mu did not focus mux.go")
	}
	// Assistive technology: a tree of items with their levels, open or
	// not, and which of the items shown each is.
	tt.Key(0, KeyHome)
	tt.Key(0, KeyLeft) // closes src
	tt.Key(0, KeyRight)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	accessNode(t, tree, platform.RoleTree, "")
	for _, want := range []struct {
		name  string
		level int
		state platform.AccessStates
		pos   int
	}{
		{"src", 1, platform.AccessExpandable | platform.AccessExpanded, 1},
		{"main.go", 2, 0, 2},
		{"cmd", 2, platform.AccessExpandable, 3},
		{"go.sum", 1, 0, 6},
	} {
		n := accessNode(t, tree, platform.RoleTreeItem, want.name)
		if got := n.States & (platform.AccessExpandable | platform.AccessExpanded); n.Level != want.level || got != want.state {
			t.Errorf("%s: level %d, states %b", want.name, n.Level, got)
		}
		if n.PosInSet != want.pos || n.SetSize != 6 {
			t.Errorf("%s: item %d of %d, want %d of 6", want.name, n.PosInSet, n.SetSize, want.pos)
		}
	}
	// It opens and closes them as the keys do.
	n := accessNode(t, tree, platform.RoleTreeItem, "cmd")
	if n.Actions&platform.ActionExpand == 0 {
		t.Fatalf("cmd cannot expand: %+v", n)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessExpand})
	if !cmdOpen || !tt.HasText("tool.go") {
		t.Error("expanding cmd for assistive technology")
	}
}

// TestTreeKeys gives Tree the keys of TreeBase: Home, End and typing the
// first letters move the focus, and the tree is one stop of Tab.
func TestTreeKeys(t *testing.T) {
	open := true
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreTree(c, func() {
				coreTreeItem(c, "lib", &open, func() {
					coreTreeItem(c, "util.go", nil, nil)
					coreTreeItem(c, "zip.go", nil, nil)
				})
				coreTreeItem(c, "README.md", nil, nil)
			})
			coreButton(c, "After")
		})
	}, 400, 300)
	tt.Click("util.go")
	for _, step := range []struct {
		key   Key
		focus string
	}{{KeyEnd, "README.md"}, {KeyHome, "lib"}, {KeyZ, "zip.go"}, {KeyR, "README.md"}, {KeyTab, "After"}} {
		tt.Advance(2 * typePause) // apart, not one word
		tt.Key(0, step.key)
		if !tt.Focused(step.focus) {
			t.Fatalf("%v did not focus %s", step.key, step.focus)
		}
	}
}

// TestOutlinePrefix leaves the start of an outline's rows to the app:
// its arrow opens and closes the item, at the depth it says, and the
// outline adds no indentation or arrow of its own.
func TestOutlinePrefix(t *testing.T) {
	var s OutlineState[string]
	tt := coreNewTester(func(c *context) {
		s.Prefix = func(r OutlineRow) {
			coreBox(c).Width(12 * float32(r.Depth)).Shrink(0)
			a := r.Arrow().Size(16, 16)
			if r.Branch {
				a.Label(map[bool]string{false: "open ", true: "close "}[r.Open] + fmt.Sprint(r.Depth))
			}
		}
		coreOutline(c, &s, []string{"src", "docs", "README.md"}, children, func(path string) {
			coreText(c, baseName(path))
		}).Grow(1)
	}, 400, 400)
	src, _ := tt.Find("src")
	arrow, _ := tt.Find("open 0")
	if src.X != arrow.X+16 {
		t.Errorf("src's text at %v, its arrow at %v: want the text right after the 16-wide arrow", src.X, arrow.X)
	}
	if err := tt.Click("open 0"); err != nil {
		t.Fatal(err)
	}
	if !s.Open.Has("src") || !tt.HasText("main.go") {
		t.Fatalf("a click on the Prefix's arrow: open %v, texts %q", s.Open.Has("src"), tt.Texts())
	}
	if ui, _ := tt.Find("ui"); ui.X != src.X+12 {
		t.Errorf("src/ui's text at %v, src's at %v: want its prefix 12 wider", ui.X, src.X)
	}
	if err := tt.Click("close 0"); err != nil {
		t.Fatal(err)
	}
	if s.Open.Has("src") || tt.HasText("main.go") {
		t.Fatal("a second click on the arrow left src open")
	}
	// Assistive technology opens and closes it as before.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	n := accessNode(t, tt.h.access, platform.RoleTreeItem, "docs")
	if n.Level != 1 || n.States&platform.AccessExpandable == 0 {
		t.Errorf("docs: %+v", n)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: n.ID, Action: platform.AccessExpand})
	if !s.Open.Has("docs") {
		t.Error("expanding docs for assistive technology")
	}
}
