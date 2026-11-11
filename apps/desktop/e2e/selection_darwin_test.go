//go:build arion_e2e && darwin

package e2e

import (
	"os"
	"testing"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/macos"
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
	if !request("clipboardCheck", map[string]any{"text": selected}).ClipboardOwned {
		t.Fatal("copy did not leave the selection on the clipboard")
	}

	request("focus", map[string]any{"target": "secure"})
	eventually(t, "secure field focused", func() bool { return request("state", nil).Focused == "secure" })
	text, secure, err = macos.ReadSelectedText()
	if err != nil {
		t.Fatal(err)
	}
	assertSecureSelectionRefused(t, application.SelectionReading{Text: text, Secure: secure})
}
