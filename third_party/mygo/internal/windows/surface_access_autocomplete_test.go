//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// What the suggestions of a field complete (aria-autocomplete) is in its
// AriaProperties, as IAccessible2's object attribute says it, after the
// other states there; a field completing nothing has none.
func TestUIAAutoComplete(t *testing.T) {
	tree := newTestTree(t, false)
	tree.update(&platform.AccessTree{Nodes: []platform.AccessNode{
		{ID: 1, Parent: -1, Role: platform.RoleGroup, Label: "Form"},
		{ID: 2, Parent: 0, Role: platform.RoleComboBox, Label: "City", AutoComplete: platform.AutoCompleteList},
		{ID: 3, Parent: 0, Role: platform.RoleComboBox, Label: "Search", AutoComplete: platform.AutoCompleteBoth, States: platform.AccessBusy},
		{ID: 4, Parent: 0, Role: platform.RoleTextField, Label: "Word", AutoComplete: platform.AutoCompleteInline},
		{ID: 5, Parent: 0, Role: platform.RoleTextField, Label: "Note"},
	}})
	for id, want := range map[uint64]string{2: "autocomplete=list", 3: "busy=true;autocomplete=both", 4: "autocomplete=inline"} {
		var v variant
		if tree.nodes[id].property(uiaAriaPropertiesProperty, &v); v.VT != vtBSTR || bstrString(uintptr(v.Val)) != want {
			t.Errorf("node %d's AriaProperties %+v, want %q", id, v, want)
		}
	}
	var v variant
	if tree.nodes[5].property(uiaAriaPropertiesProperty, &v); v.VT == vtBSTR && bstrString(uintptr(v.Val)) != "" {
		t.Errorf("Note's AriaProperties %q, want none", bstrString(uintptr(v.Val)))
	}
}
