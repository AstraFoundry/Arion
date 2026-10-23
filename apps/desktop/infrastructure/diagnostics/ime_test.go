package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/desktop/infrastructure/imebridge"
)

func diagnosticInputMethod() InputMethodDiagnostic {
	return InputMethodDiagnostic{Operation: "query", Step: "observed", ProcessID: 123, Current: "other",
		Mode:   InputSourceDiagnostic{ID: imebridge.SourceID, Found: true, Enabled: true, Selectable: true, IdentityMatches: true},
		Parent: InputSourceDiagnostic{ID: imebridge.BundleID, Found: true, Enabled: true, IdentityMatches: true}}
}

func TestInputMethodDiagnosticRejectsForeignIdentityAndContent(t *testing.T) {
	changes := []func(*InputMethodDiagnostic){
		func(d *InputMethodDiagnostic) { d.RequestedID = "com.private.editor" },
		func(d *InputMethodDiagnostic) { d.Mode.ID = "private input" },
		func(d *InputMethodDiagnostic) { d.Parent.ID = "/Users/private" },
		func(d *InputMethodDiagnostic) { d.Current = "com.private.inputmethod" },
		func(d *InputMethodDiagnostic) { d.Operation = "private contents" },
		func(d *InputMethodDiagnostic) { d.Step = "private contents" },
		func(d *InputMethodDiagnostic) { d.HostBuild = "private contents" },
		func(d *InputMethodDiagnostic) { d.BuildCode = "private error" },
	}
	for i, change := range changes {
		d := diagnosticInputMethod()
		change(&d)
		if (Entry{At: time.Now(), Event: "ime_native", InputMethod: &d}).valid() {
			t.Fatalf("accepted sensitive field mutation %d", i)
		}
	}
	d := diagnosticInputMethod()
	if (Entry{At: time.Now(), Event: "app_started", InputMethod: &d}).valid() ||
		(Entry{At: time.Now(), Event: "ime_native"}).valid() {
		t.Fatal("accepted an unbound input-method payload")
	}
}

func TestInputMethodDiagnosticExportOwnsValuesAndPreservesStatus(t *testing.T) {
	r := openTest(t)
	d := diagnosticInputMethod()
	d.Operation, d.Step, d.RequestedID = "select", "after", imebridge.SourceID
	status := int32(-50)
	d.OSStatus = &status
	d.HostBuild = strings.Repeat("ab", 32)
	r.Record(Entry{Event: "ime_native", InputMethod: &d})
	// Mutation after ingestion must not alter the worker's validated snapshot.
	d.Mode.ID, status = "PRIVATE_CANARY", 98765
	path := filepath.Join(t.TempDir(), "logs.zip")
	if err := r.Export(path, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	data := unzip(t, path)["events.0.jsonl"]
	if strings.Contains(string(data), "PRIVATE_CANARY") {
		t.Fatal("caller mutation escaped validation")
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatal(err)
	}
	if e.InputMethod == nil || e.InputMethod.OSStatus == nil || *e.InputMethod.OSStatus != -50 ||
		e.InputMethod.Mode.ID != imebridge.SourceID || e.InputMethod.HostBuild != strings.Repeat("ab", 32) {
		t.Fatalf("lost diagnostic fields: %s", data)
	}
}

func TestInputMethodDiagnosticExportRejectsNestedUnknownFields(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "logs"))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	d := diagnosticInputMethod()
	r.Record(Entry{Event: "ime_native", InputMethod: &d})
	if _, err := r.snapshotFiles(); err != nil {
		t.Fatal(err)
	}
	path := r.path(0)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"inputMethod":{`, `"inputMethod":{"rawId":"PRIVATE_CANARY",`, 1))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.Export(filepath.Join(t.TempDir(), "logs.zip"), "0.1.0"); err == nil {
		t.Fatal("export accepted an unknown nested field")
	}
	if err := r.Close(); err == nil {
		t.Fatal("corrupt diagnostic source did not retain its failure")
	}
}
