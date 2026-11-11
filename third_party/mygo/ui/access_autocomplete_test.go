package ui_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAccessAutoComplete reads what the suggestions of fields complete
// (aria-autocomplete) as assistive technology gets it: what a combobox of
// one's own tells, AutoCompleteList for the widgets whose options match
// what was typed, and none for a field without suggestions or one taken
// back.
func TestAccessAutoComplete(t *testing.T) {
	var query, city, font string
	var tags, labels []string
	kind := ui.AutoCompleteList
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			ui.TextInputBase(c, &query).Role(ui.RoleComboBox).Label("Search").Expanded(query != "").AutoComplete(kind)
			ui.Autocomplete(c, &city, []string{"Paris", "Prague"}).Label("City")
			ui.Combobox(c, &font, []string{"Helvetica", "Courier"}).Label("Font")
			ui.TokenField(c, &tags, []string{"go", "zig"}).Label("Tags")
			ui.TokenField(c, &labels, nil).Label("Labels")
			ui.TextInput(c, &query).Label("Note")
		})
	}, 400, 400)

	for _, want := range []struct {
		role ui.Role
		name string
		kind ui.AutoComplete
	}{
		{ui.RoleComboBox, "Search", ui.AutoCompleteList},
		{ui.RoleComboBox, "City", ui.AutoCompleteList},
		{ui.RoleComboBox, "Font", ui.AutoCompleteList},
		{ui.RoleComboBox, "Tags", ui.AutoCompleteList},
		{ui.RoleTextField, "Labels", ui.AutoCompleteNone},
		{ui.RoleTextField, "Note", ui.AutoCompleteNone},
	} {
		if a := accessOf(t, tt, want.role, want.name); a.AutoComplete != want.kind {
			t.Errorf("%s completes %d, want %d", want.name, a.AutoComplete, want.kind)
		}
	}
	// An element tells what it is given in the frame.
	for _, k := range []ui.AutoComplete{ui.AutoCompleteInline, ui.AutoCompleteBoth, ui.AutoCompleteNone} {
		kind = k
		tt.Frame()
		if a := accessOf(t, tt, ui.RoleComboBox, "Search"); a.AutoComplete != k {
			t.Errorf("Search completes %d, want %d", a.AutoComplete, k)
		}
	}
}
