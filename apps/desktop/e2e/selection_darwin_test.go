//go:build arion_e2e && darwin

package e2e

import (
	"os"
	"testing"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/macos"
	arion "arion.local/desktop/ui/shared/runtime"
)

func TestSelectionActions(t *testing.T) {
	if os.Getenv("ARION_E2E_SELECTION") != "1" {
		t.Skip("set ARION_E2E_SELECTION=1 to read an owned editor selection")
	}
	if (macos.Permissions{}).Accessibility() != macos.AuthGranted {
		t.Fatal("selection E2E requires existing Accessibility access for the test runner")
	}
	request := startEditorFixture(t)
	request("clipboardSave", nil)
	t.Cleanup(func() { request("clipboardRestore", nil) })
	const selected = "ArionSelectionTerm"
	request("set", map[string]any{"text": "prefix " + selected + " suffix", "location": 7, "length": len(selected)})
	request("focus", map[string]any{"target": "A"})
	eventually(t, "owned editor focused", func() bool { return request("state", nil).Focused == "A" })
	text, secure, err := macos.ReadSelectedText()
	if err != nil {
		t.Fatal(err)
	}
	assertSelectionCopyHotwordURLAndSwitch(t, selected, application.SelectionReading{Text: text, Secure: secure})
	exerciseSelectionPopover(t, selected)
	if !request("clipboardCheck", map[string]any{"text": selected}).ClipboardOwned {
		t.Fatal("copy did not leave the selection on the clipboard")
	}
	request("set", map[string]any{"text": "prefix " + selected + " suffix", "location": 7, "length": 0})
	assertNoSelectionPopover(t)

	request("focus", map[string]any{"target": "secure"})
	eventually(t, "secure field focused", func() bool { return request("state", nil).Focused == "secure" })
	text, secure, err = macos.ReadSelectedText()
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
	request("focus", map[string]any{"target": "A"})
	request("set", map[string]any{"text": "prefix " + selected + " suffix", "location": 7, "length": len(selected)})
	assertNoSelectionPopover(t)
}
