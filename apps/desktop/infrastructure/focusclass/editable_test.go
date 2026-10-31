package focusclass

import "testing"

func TestMacEditableFocus(t *testing.T) {
	cases := []struct {
		name string
		in   MacObservation
		want MacDecision
	}{
		{
			name: "native text field",
			in:   MacObservation{Role: "AXTextField", ValueSettable: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "search field",
			in:   MacObservation{Role: "AXTextField", Subrole: "AXSearchField", SelectedTextSettable: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "text area with writable selected text",
			in:   MacObservation{Role: "AXTextArea", SelectedTextSettable: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "read-only transcript can move its selection",
			in:   MacObservation{Role: "AXTextArea", SelectedRangeSettable: true, HasSelection: true, Focused: true},
			want: MacDecision{},
		},
		{
			name: "contenteditable root exposes a writable range and an editable ancestor",
			in:   MacObservation{Role: "AXTextArea", SelectedRangeSettable: true, EditableAncestor: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "contenteditable group",
			in:   MacObservation{Role: "AXGroup", EditableAncestor: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "static text inside a composer",
			in:   MacObservation{Role: "AXStaticText", EditableAncestor: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "web area that is itself the editor",
			in:   MacObservation{Role: "AXWebArea", EditableAncestor: true, Focused: true},
			want: MacDecision{Editable: true},
		},
		{
			name: "button inside a composer",
			in:   MacObservation{Role: "AXButton", EditableAncestor: true, Focused: true},
			want: MacDecision{NoInput: true},
		},
		{
			name: "link inside a composer",
			in:   MacObservation{Role: "AXLink", EditableAncestor: true, Focused: true},
			want: MacDecision{NoInput: true},
		},
		{
			name: "layout group",
			in:   MacObservation{Role: "AXGroup", Focused: true},
			want: MacDecision{NoInput: true},
		},
		{
			name: "web area without an editor",
			in:   MacObservation{Role: "AXWebArea", Focused: true},
			want: MacDecision{NoInput: true},
		},
		{
			name: "secure text field",
			in:   MacObservation{Role: "AXTextField", Subrole: "AXSecureTextField", ValueSettable: true, Focused: true},
			want: MacDecision{},
		},
		{
			name: "secure input session",
			in:   MacObservation{Role: "AXTextArea", ValueSettable: true, Secure: true, Focused: true},
			want: MacDecision{},
		},
		{
			name: "explicitly nothing focused",
			in:   MacObservation{ExplicitlyUnfocused: true},
			want: MacDecision{NoInput: true},
		},
		{
			name: "provider returned no element and no absence",
			in:   MacObservation{},
			want: MacDecision{},
		},
		{
			name: "non-text role with a selection is not a positive empty editor",
			in:   MacObservation{Role: "AXGroup", HasSelection: true, Focused: true},
			want: MacDecision{},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := DecideMac(test.in); got != test.want {
				t.Fatalf("DecideMac() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestWindowsEditableFocus(t *testing.T) {
	cases := []struct {
		name string
		in   WinObservation
		want WinDecision
	}{
		{
			name: "writable value",
			in:   WinObservation{ControlType: UIAEdit, ValuePattern: true},
			want: WinDecision{Editable: true, Known: true},
		},
		{
			name: "read-only value",
			in:   WinObservation{ControlType: UIAEdit, ValuePattern: true, ValueReadOnly: true},
			want: WinDecision{Known: true},
		},
		{
			name: "writable text pattern",
			in:   WinObservation{ControlType: UIADocument, TextPattern: true, TextReadOnlyKnown: true},
			want: WinDecision{Editable: true, Known: true},
		},
		{
			name: "read-only document",
			in:   WinObservation{ControlType: UIADocument, TextPattern: true, TextReadOnly: true, TextReadOnlyKnown: true},
			want: WinDecision{Known: true},
		},
		{
			name: "group composer under a read-only document",
			in:   WinObservation{ControlType: UIAGroup, CaretEditable: true},
			want: WinDecision{Editable: true, Known: true},
		},
		{
			name: "document whose pattern is read-only but whose caret is not",
			in:   WinObservation{ControlType: UIADocument, TextPattern: true, TextReadOnly: true, TextReadOnlyKnown: true, CaretEditable: true},
			want: WinDecision{Editable: true, Known: true},
		},
		{
			name: "text edit pattern",
			in:   WinObservation{ControlType: UIACustom, TextEditPattern: true},
			want: WinDecision{Editable: true, Known: true},
		},
		{
			name: "password",
			in:   WinObservation{ControlType: UIAEdit, Password: true, ValuePattern: true},
			want: WinDecision{Secure: true, Known: true},
		},
		{
			name: "edit without a pattern stays unknown",
			in:   WinObservation{ControlType: UIAEdit},
			want: WinDecision{},
		},
		{
			name: "document without a pattern stays unknown",
			in:   WinObservation{ControlType: UIADocument},
			want: WinDecision{},
		},
		{
			name: "button",
			in:   WinObservation{ControlType: UIAButton},
			want: WinDecision{Known: true},
		},
		{
			name: "broken text attribute",
			in:   WinObservation{ControlType: UIAEdit, TextPattern: true},
			want: WinDecision{},
		},
		{
			name: "broken caret query",
			in:   WinObservation{ControlType: UIAGroup, CaretUncertain: true},
			want: WinDecision{},
		},
		{
			name: "read-only text pattern survives a failed caret query",
			in:   WinObservation{ControlType: UIADocument, TextPattern: true, TextReadOnly: true, TextReadOnlyKnown: true, CaretUncertain: true},
			want: WinDecision{Known: true},
		},
		{
			name: "read-only value is not rescued by a caret",
			in:   WinObservation{ControlType: UIAEdit, ValuePattern: true, ValueReadOnly: true, CaretEditable: true},
			want: WinDecision{Known: true},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := DecideWindows(test.in); got != test.want {
				t.Fatalf("DecideWindows() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestWindowsCaretHostSkipsControls(t *testing.T) {
	if !WindowsCaretHost(UIAGroup) || !WindowsCaretHost(UIADocument) || !WindowsCaretHost(UIAEdit) {
		t.Fatal("text hosts were rejected")
	}
	if WindowsCaretHost(UIAButton) || WindowsCaretHost(UIAHyperlink) {
		t.Fatal("controls were accepted as caret hosts")
	}
}
