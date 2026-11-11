//go:build arion_e2e && windows

package e2e

import (
	"testing"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/windows"
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

	editor.focusControl(t, editor.password)
	text, secure, err = windows.ReadSelectedText()
	if err != nil {
		t.Fatal(err)
	}
	assertSecureSelectionRefused(t, application.SelectionReading{Text: text, Secure: secure})
}
