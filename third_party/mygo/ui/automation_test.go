//go:build !mygo_noinspector

package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestAutomationTreeKeys reads the keys of elements as plugins/automation
// does: Element.Key's, and Context.Key's for widgets that take theirs
// before they are made.
func TestAutomationTreeKeys(t *testing.T) {
	name, agree := "Ada", false
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			Button(c, "Add").Key("add")
			c.Key("name")
			TextInput(c, &name)
			c.Key(7)
			Checkbox(c, &agree, "Agree")
			Button(c, "Plain")
		})
	}, 400, 300)
	tt.Frame()
	tree, keys := automated{&windowHost{rt: tt.rt}}.AutomationTree()
	want := map[platform.AccessRole]any{platform.RoleButton: "add", platform.RoleTextField: "name", platform.RoleCheckBox: 7}
	seen := 0
	for _, n := range tree.Nodes {
		k, ok := keys[n.ID]
		switch {
		case n.Role == platform.RoleButton && n.Label == "Plain":
			if ok {
				t.Errorf("a button without a key has %v", k)
			}
		case want[n.Role] != nil:
			seen++
			if k != want[n.Role] {
				t.Errorf("%v %q: key %v, want %v", n.Role, n.Label, k, want[n.Role])
			}
		}
	}
	if seen != len(want) {
		t.Errorf("found %d of the keyed elements in %+v", seen, tree.Nodes)
	}
}
