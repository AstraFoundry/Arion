package macos

import (
	"testing"

	"arion.local/desktop/infrastructure/diagnostics"
)

func TestInputMethodObservationsSampleChangesAndKeepOperationResults(t *testing.T) {
	var entries []diagnostics.Entry
	SetIMEDiagnostics(func(e diagnostics.Entry) { entries = append(entries, e) })
	t.Cleanup(func() { SetIMEDiagnostics(nil) })
	d := diagnostics.InputMethodDiagnostic{Operation: "query", Step: "observed", ProcessID: 123, Current: "other",
		Mode: inputSourceDiagnostic(IMESourceID, 27), Parent: inputSourceDiagnostic(IMEBundleID, 19)}
	emitInputMethodDiagnostic(d, nil)
	emitInputMethodDiagnostic(d, nil)
	d.Mode.Selected, d.Current = true, "arion_mode"
	emitInputMethodDiagnostic(d, nil)
	if len(entries) != 2 {
		t.Fatalf("unchanged polling was logged or change lost: %d", len(entries))
	}
	if !entries[0].InputMethod.Mode.Found || !entries[0].InputMethod.Mode.IdentityMatches ||
		!entries[0].InputMethod.Mode.Selectable || entries[0].InputMethod.Parent.Selectable {
		t.Fatal("native flags were mapped incorrectly")
	}
	d.Operation, d.Step = "select", "after"
	status := int32(0)
	d.OSStatus = &status
	emitInputMethodDiagnostic(d, nil)
	emitInputMethodDiagnostic(d, nil)
	if len(entries) != 4 || entries[3].InputMethod.OSStatus == nil || *entries[3].InputMethod.OSStatus != 0 {
		t.Fatal("repeated native results were suppressed or zero status lost")
	}
	SetIMEDiagnostics(nil)
	emitInputMethodDiagnostic(d, nil)
	if len(entries) != 4 {
		t.Fatal("detached observer still received diagnostics")
	}
}
