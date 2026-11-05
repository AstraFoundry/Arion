package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCloudModelDiagnosticValidatesBoundedFields(t *testing.T) {
	for _, tc := range []struct {
		failure string
		status  int
		valid   bool
	}{
		{"MODELS", 403, true}, {"MODELS", 429, true}, {"MODELS", 503, true},
		{"REJECTED", 401, true}, {"NETWORK", 0, true}, {"MODELS", 0, true},
		{"PRIVATE_KEY_OR_RESPONSE", 403, false}, {"MODELS", -1, false},
		{"MODELS", 99, false}, {"MODELS", 600, false},
	} {
		d := CloudModelsDiagnostic{Failure: tc.failure, HTTPStatus: tc.status}
		entry := Entry{At: time.Now(), Event: "cloud_models_failed", CloudModels: &d}
		if entry.valid() != tc.valid {
			t.Fatalf("unexpected validation for %+v", d)
		}
		entry.Event = "app_started"
		if entry.valid() {
			t.Fatal("accepted cloud diagnostic on another event")
		}
	}
	if (Entry{At: time.Now(), Event: "cloud_models_failed"}).valid() {
		t.Fatal("accepted failure without a diagnostic")
	}
}

func TestCloudModelDiagnosticExportOwnsValues(t *testing.T) {
	r := openTest(t)
	d := CloudModelsDiagnostic{Failure: "MODELS", HTTPStatus: 403}
	r.Record(Entry{Event: "cloud_models_failed", CloudModels: &d})
	d.Failure = "PRIVATE_RESPONSE_CANARY"
	path := filepath.Join(t.TempDir(), "logs.zip")
	if err := r.Export(path, "0.1.2"); err != nil {
		t.Fatal(err)
	}
	data := unzip(t, path)["events.0.jsonl"]
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "PRIVATE_RESPONSE_CANARY") || entry.CloudModels == nil ||
		*entry.CloudModels != (CloudModelsDiagnostic{Failure: "MODELS", HTTPStatus: 403}) {
		t.Fatal("export lost safe fields or retained caller mutation")
	}
}

func TestCloudModelDiagnosticExportRejectsResponseContent(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "logs"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() }) // Corrupt input deliberately leaves a sticky failure.
	r.Record(Entry{Event: "cloud_models_failed", CloudModels: &CloudModelsDiagnostic{Failure: "MODELS", HTTPStatus: 403}})
	if _, err := r.snapshotFiles(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(r.path(0))
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"cloudModels":{`, `"cloudModels":{"body":"PRIVATE_RESPONSE_CANARY",`, 1))
	if err := os.WriteFile(r.path(0), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.Export(filepath.Join(t.TempDir(), "logs.zip"), "0.1.2"); err == nil {
		t.Fatal("export accepted a provider response body")
	}
}
