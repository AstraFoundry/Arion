// Package focusclass decides whether the focused control can accept dictated
// text. Platform adapters collect accessibility facts; this package applies
// one rule to those facts so macOS and Windows stay aligned and testable
// without a live editor.
//
// A text role or a text selection by itself is not an insertion point.
// Read-only transcripts stay non-editable. Secure fields stay blocked.
// Chromium and WebKit contenteditable composers are included when they
// expose an editable-text ancestor or a caret whose text range is writable,
// including when the focused node is a group or static text inside the
// composer rather than a native text field.
package focusclass

import "arion.local/core/domain"

// FieldKind maps an accessibility role to the scene field type. It does not
// decide whether text can be inserted.
func FieldKind(role, subrole string) domain.FieldKind {
	switch {
	case subrole == "AXSearchField":
		return domain.FieldSearch
	case role == "AXTextField" || role == "AXTextArea" || role == "AXComboBox":
		return domain.FieldText
	default:
		return domain.FieldUnknown
	}
}

// MacObservation is the focused macOS accessibility element, plus whether
// the platform explicitly reported that nothing is focused.
type MacObservation struct {
	Role                  string
	Subrole               string
	ValueSettable         bool
	SelectedTextSettable  bool
	SelectedRangeSettable bool
	EditableAncestor      bool
	Secure                bool
	HasSelection          bool
	Focused               bool
	ExplicitlyUnfocused   bool
}

// MacDecision is the insertion classification for one observation.
// NoInput is a positive observation that no editable field is focused.
// An uncertain or unread observation leaves both flags false.
type MacDecision struct {
	Editable bool
	NoInput  bool
}

// DecideMac classifies a macOS focus observation.
//
// Text fields, text areas, combo boxes and search fields are editable when
// their value or selected text is writable. A contenteditable root is often
// an AXTextArea whose selection range is writable while AXValue and
// AXSelectedText are not; that range counts only together with the
// editable-ancestor attribute, because a read-only transcript can move its
// selection too. When focus lands on a group, static text or other text
// container inside the composer, the editable ancestor is enough. Buttons,
// links and other controls keep their roles and are not insertion points
// even if they sit inside an editor.
func DecideMac(o MacObservation) MacDecision {
	if o.Secure || o.Subrole == "AXSecureTextField" {
		return MacDecision{}
	}
	if !o.Focused {
		return MacDecision{NoInput: o.ExplicitlyUnfocused}
	}
	if macEditable(o) {
		return MacDecision{Editable: true}
	}
	// A text-entry role that is not writable is a read-only field. Copying
	// still happens, but that is not a positive "no input" observation.
	if o.Role != "" && !o.HasSelection && FieldKind(o.Role, o.Subrole) == domain.FieldUnknown {
		return MacDecision{NoInput: true}
	}
	return MacDecision{}
}

func macEditable(o MacObservation) bool {
	entry := textEntry(o.Role, o.Subrole)
	if entry && (o.ValueSettable || o.SelectedTextSettable) {
		return true
	}
	if entry && o.SelectedRangeSettable && o.EditableAncestor {
		return true
	}
	return o.EditableAncestor && richTextContainer(o.Role)
}

func textEntry(role, subrole string) bool {
	return FieldKind(role, subrole) != domain.FieldUnknown
}

func richTextContainer(role string) bool {
	switch role {
	case "AXGroup", "AXStaticText", "AXUnknown", "AXWebArea", "AXScrollArea",
		"AXHeading", "AXListItem", "AXCell", "AXRow":
		return true
	default:
		return false
	}
}

// UI Automation control types used by editable-focus classification.
// Values are Microsoft's CONTROLTYPEID constants.
const (
	UIAButton    int32 = 50000
	UIAComboBox  int32 = 50003
	UIAEdit      int32 = 50004
	UIAHyperlink int32 = 50005
	UIAListItem  int32 = 50007
	UIAText      int32 = 50020
	UIACustom    int32 = 50025
	UIAGroup     int32 = 50026
	UIADataItem  int32 = 50029
	UIADocument  int32 = 50030
	UIAPane      int32 = 50033
	UIATable     int32 = 50036
)

// WindowsCaretHost reports control types whose keyboard focus can sit on a
// composer that does not use a Win32 Edit window. Buttons and links are
// excluded so a caret elsewhere in the window is not treated as theirs.
func WindowsCaretHost(controlType int32) bool {
	switch controlType {
	case UIAEdit, UIADocument, UIAComboBox, UIAGroup, UIACustom, UIAText,
		UIAPane, UIAListItem, UIADataItem, UIATable:
		return true
	default:
		return false
	}
}

// WinObservation is the focused UI Automation element.
// Uncertain flags mean the provider failed, not that the pattern is absent.
type WinObservation struct {
	ControlType       int32
	Password          bool
	ValuePattern      bool
	ValueReadOnly     bool
	TextPattern       bool
	TextReadOnly      bool
	TextReadOnlyKnown bool
	TextEditPattern   bool
	CaretEditable     bool
	CaretUncertain    bool
}

// WinDecision is the insertion classification for one Windows observation.
// Known is false when the provider did not answer; that must not be stored
// as a positive no-input observation.
type WinDecision struct {
	Editable bool
	Secure   bool
	Known    bool
}

// DecideWindows classifies a UI Automation focus observation.
//
// A writable Value pattern, a Text pattern that is not read-only, or
// TextEditPattern establishes an editor. Chromium contenteditable composers
// are often a Group with no Value pattern, under a Document whose own text
// pattern is read-only. A caret selection whose range is not read-only is
// an editor in that case. A standard Edit or Document that exposes none of
// these patterns stays unknown. Other controls that expose none of them are
// a positive non-editable focus.
func DecideWindows(o WinObservation) WinDecision {
	if o.Password {
		return WinDecision{Secure: true, Known: true}
	}
	if o.ValuePattern {
		return WinDecision{Editable: !o.ValueReadOnly, Known: true}
	}
	if o.TextEditPattern {
		return WinDecision{Editable: true, Known: true}
	}
	if o.TextPattern && o.TextReadOnlyKnown && !o.TextReadOnly {
		return WinDecision{Editable: true, Known: true}
	}
	if o.CaretEditable {
		return WinDecision{Editable: true, Known: true}
	}
	// A text pattern that answered read-only stays a positive non-editable
	// observation when the extra caret query fails.
	if o.TextPattern && o.TextReadOnlyKnown {
		return WinDecision{Known: true}
	}
	if o.CaretUncertain || o.TextPattern {
		return WinDecision{}
	}
	if (o.ControlType == UIAEdit || o.ControlType == UIADocument) && !o.TextPattern && !o.TextEditPattern {
		return WinDecision{}
	}
	return WinDecision{Known: true}
}
