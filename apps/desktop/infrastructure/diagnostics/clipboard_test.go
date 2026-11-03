package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func clipboardDiagnosticFixture() ClipboardTargetDiagnostic {
	return ClipboardTargetDiagnostic{Focus: "unsupported", Window: "available",
		Verification: "missing_focus", Trusted: true}
}

func TestClipboardTargetDiagnosticRejectsContentAndUnboundPayload(t *testing.T) {
	for _, change := range []func(*ClipboardTargetDiagnostic){
		func(d *ClipboardTargetDiagnostic) { d.Focus = "PRIVATE_EDITOR_TEXT" },
		func(d *ClipboardTargetDiagnostic) { d.Window = "PRIVATE_WINDOW_TITLE" },
		func(d *ClipboardTargetDiagnostic) { d.SystemFocus = "PRIVATE_APP_IDENTITY" },
		func(d *ClipboardTargetDiagnostic) { d.Verification = "PRIVATE_ERROR_DETAIL" },
	} {
		d := clipboardDiagnosticFixture()
		change(&d)
		if (Entry{At: time.Now(), Event: "clipboard_target", ClipboardTarget: &d}).valid() {
			t.Fatal("accepted arbitrary content in target diagnostic")
		}
	}
	d := clipboardDiagnosticFixture()
	if (Entry{At: time.Now(), Event: "app_started", ClipboardTarget: &d}).valid() ||
		(Entry{At: time.Now(), Event: "clipboard_target"}).valid() {
		t.Fatal("accepted an unbound clipboard-target payload")
	}
}

func TestClipboardTargetDiagnosticExportOwnsValues(t *testing.T) {
	r := openTest(t)
	d := clipboardDiagnosticFixture()
	r.Record(Entry{Event: "clipboard_target", ClipboardTarget: &d})
	d.Focus = "PRIVATE_CONTENT_CANARY"
	path := filepath.Join(t.TempDir(), "logs.zip")
	if err := r.Export(path, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	data := unzip(t, path)["events.0.jsonl"]
	if strings.Contains(string(data), "PRIVATE_CONTENT_CANARY") {
		t.Fatal("caller mutation escaped validation")
	}
	if strings.Contains(string(data), `"systemFocus"`) || strings.Contains(string(data), `"systemOwnerMatches"`) {
		t.Fatal("new observations emitted retired query fields")
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatal(err)
	}
	if e.ClipboardTarget == nil || *e.ClipboardTarget != clipboardDiagnosticFixture() {
		t.Fatal("lost bounded capture/query outcomes")
	}
}

func TestClipboardTargetDiagnosticAcceptsLegacyScopeFields(t *testing.T) {
	d := clipboardDiagnosticFixture()
	d.SystemFocus, d.SystemOwnerMatches = "available", true
	if !(Entry{At: time.Now(), Event: "clipboard_target", ClipboardTarget: &d}).valid() {
		t.Fatal("older development observations are no longer exportable")
	}
}

func TestClipboardTargetDiagnosticExportRejectsUnknownFields(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "logs"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() }) // Corrupt input deliberately leaves a sticky error.
	d := clipboardDiagnosticFixture()
	r.Record(Entry{Event: "clipboard_target", ClipboardTarget: &d})
	if _, err := r.snapshotFiles(); err != nil {
		t.Fatal(err)
	}
	path := r.path(0)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"clipboardTarget":{`, `"clipboardTarget":{"title":"PRIVATE_CANARY",`, 1))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.Export(filepath.Join(t.TempDir(), "logs.zip"), "0.1.0"); err == nil {
		t.Fatal("export accepted unrecognized target data")
	}
	if err := r.Close(); err == nil {
		t.Fatal("corrupt input did not retain its diagnostic failure")
	}
}
