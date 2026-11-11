//go:build arion_e2e && windows

package e2e

import (
	"testing"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/windows"
	arion "arion.local/desktop/ui/shared/runtime"
)

func TestWindowsSelectionActions(t *testing.T) {
	editor := startWindowsEditor(t)
	const selected = "ArionSelectionTerm"
	editor.setText("prefix " + selected + " suffix")
	editor.selectRange(7, 7+uintptr(len(selected)))
	editor.focus(t)
	text, secure, err := windows.ReadSelectedText()
	if err != nil {
		t.Fatal(err)
	}
	assertSelectionCopyHotwordURLAndSwitch(t, selected, application.SelectionReading{Text: text, Secure: secure})
	exerciseSelectionPopover(t, selected)
	editor.selectRange(0, 0)
	assertNoSelectionPopover(t)

	editor.focusControl(t, editor.password)
	text, secure, err = windows.ReadSelectedText()
	if err != nil {
		t.Fatal(err)
	}
	assertSecureSelectionRefused(t, application.SelectionReading{Text: text, Secure: secure})
	before, err := app.ReopenedVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	assertNoSelectionPopover(t)
	var actionErr error
	app.Update(func(*arion.State) { actionErr = app.Store().Actions.AddSelectionVocabulary() })
	if actionErr == nil {
		t.Fatal("secure field saved a hotword")
	}
	after, err := app.ReopenedVocabulary()
	if err != nil || len(after) != len(before) {
		t.Fatalf("secure field changed vocabulary: %v %v", before, after)
	}
	setSelectionEnabled(t, false)
	editor.focus(t)
	editor.selectRange(7, 7+uintptr(len(selected)))
	assertNoSelectionPopover(t)
}
