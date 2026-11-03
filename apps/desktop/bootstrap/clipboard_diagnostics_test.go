package bootstrap

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/diagnostics"
)

type clipboardDiagnosticTarget struct {
	clipboardDeliveryTarget
	observations int
}

func (t *clipboardDiagnosticTarget) ClipboardDiagnostic() diagnostics.ClipboardTargetDiagnostic {
	t.observations++
	return diagnostics.ClipboardTargetDiagnostic{Focus: "unsupported", Window: "available",
		Verification: "missing_focus", Trusted: true}
}

func TestClipboardTargetDiagnosticsOmitHistoryAndPreserveAction(t *testing.T) {
	a := clipboardFixture(t)
	target := &clipboardDiagnosticTarget{}
	a.recordClipboardTarget(target)
	if target.observations != 0 {
		t.Fatal("missing diagnostic storage triggered a native observation")
	}
	dir := t.TempDir()
	r, err := diagnostics.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.diagnostics = r
	payload := domain.ClipTextPayload("PRIVATE_CLIPBOARD_CANARY")
	a.state.Clipboard.Items = []domain.Clip{{ID: payload.ID(), Payload: payload}}
	a.state.Clipboard.CopyOnly = true
	a.recordClipboardTarget(target)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "events.0.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var entry diagnostics.Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(payload.Text)) || entry.ClipboardTarget == nil || entry.ClipboardTarget.Verification != "missing_focus" || target.observations != 1 {
		t.Fatal("clipboard diagnostic did not preserve bounded outcomes")
	}
	if !a.state.Clipboard.CopyOnly || target.checked != 0 || target.pasted != 0 {
		t.Fatal("observation changed the delivery mode or attempted delivery")
	}
}
