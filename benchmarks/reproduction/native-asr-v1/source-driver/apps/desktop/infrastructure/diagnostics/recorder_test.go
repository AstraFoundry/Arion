package diagnostics

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func openTest(t *testing.T) *Recorder {
	t.Helper()
	r, err := Open(filepath.Join(t.TempDir(), "logs"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	return r
}

func unzip(t *testing.T, path string) map[string][]byte {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	out := map[string][]byte{}
	for _, f := range z.File {
		if runtime.GOOS != "windows" && f.Mode().Perm() != 0600 {
			t.Errorf("mode %s: %v", f.Name, f.Mode())
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = b
	}
	return out
}

// PRV-A02: deliberately sensitive event fields and error messages never enter
// logs or the export; unrelated profile/native files are excluded entirely.
func TestExportExcludesSensitiveCanaries(t *testing.T) {
	r := openTest(t)
	secret := "PRIVATE_TRANSCRIPT_PROMPT_PATH_TOKEN"
	for _, name := range []string{"preferences.json", "vocabulary.bin", "usage.json", "native-stderr.log"} {
		if err := os.WriteFile(filepath.Join(r.dir, name), []byte(secret), 0600); err != nil {
			t.Fatal(err)
		}
	}
	id := contracts.Identity{SessionID: "0123456789abcdef", Generation: 2}
	e := contracts.Event{Envelope: contracts.Envelope{Identity: id, Kind: contracts.SessionStateChanged, Sequence: 1}, State: domain.Preparing,
		Text: secret, Raw: secret, Source: secret, Detail: secret, Scene: secret}
	r.Session(e)
	e.Kind, e.Sequence, e.State = contracts.DraftReplaced, 2, domain.Capturing
	for i := 0; i < 100; i++ {
		r.Session(e)
	}
	e.Kind, e.Sequence, e.Rewrite = contracts.CorrectionReady, 3, "negation_dropped"
	r.Session(e)
	e.Rewrite = "uncertainty_dropped"
	r.Session(e)
	e.Kind, e.Sequence, e.Outcome = contracts.CommitResultReady, 4, contracts.Unknown
	r.Session(e)
	r.Record(Entry{Event: "preferences_save", Code: Failure(errors.New("/Users/private/" + secret))})
	r.Record(Entry{Event: secret})
	r.Record(Entry{Event: "session_error", Session: secret})
	r.Record(Entry{Event: "session_error", Code: domain.ErrorCode(secret)})
	r.Record(Entry{Event: "session_error", Outcome: secret})
	path := filepath.Join(t.TempDir(), "diagnostics.zip")
	if err := r.Export(path, "0.1.0"); err != nil {
		t.Fatal(err)
	}
	files := unzip(t, path)
	if len(files) != 2 {
		t.Fatalf("unexpected files: %v", files)
	}
	joined := bytes.Join([][]byte{files["manifest.json"], files["events.0.jsonl"]}, nil)
	if bytes.Contains(joined, []byte(secret)) || bytes.Contains(joined, []byte("/Users/")) {
		t.Fatal("sensitive canary exported")
	}
	log := string(files["events.0.jsonl"])
	for _, event := range []string{"session_state", "first_draft", "correction_ready", "commit_result", "IO_FAILED", "events_dropped", "negation_dropped", "uncertainty_dropped", "unknown"} {
		if !strings.Contains(log, event) {
			t.Errorf("missing %s: %s", event, log)
		}
	}
	if strings.Count(log, "first_draft") != 1 {
		t.Fatal("draft stream was not sampled")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("export permissions %v", info.Mode())
		}
		info, err = os.Stat(r.dir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0700 {
			t.Errorf("log directory permissions %v", info.Mode())
		}
	}
	var manifest Manifest
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != 1 || manifest.OS != runtime.GOOS || manifest.Version != "0.1.0" {
		t.Fatalf("manifest %+v", manifest)
	}
}

func TestRotationAndRetention(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "events.4.jsonl")
	if err := os.WriteFile(stale, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-Retention - time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	r, err := open(dir, 400, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired file remains: %v", err)
	}
	for i := 0; i < 50; i++ {
		r.Record(Entry{Event: "session_state", Session: "0123456789abcdef", State: domain.Capturing, Sequence: uint64(i + 1)})
		// Barriers keep this test about retention rather than queue pressure.
		if _, err := r.snapshotFiles(); err != nil {
			t.Fatal(err)
		}
	}
	files, err := r.snapshotFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != MaxFiles {
		t.Fatalf("files %d", len(files))
	}
	for name, data := range files {
		if len(data) > 400 {
			t.Errorf("unbounded %s: %d", name, len(data))
		}
	}
	if !bytes.Contains(files["events.0.jsonl"], []byte(`"sequence":50`)) {
		t.Fatal("newest event lost")
	}
	if bytes.Contains(bytes.Join(values(files), nil), []byte(`"sequence":1,`)) {
		t.Fatal("old events not rotated")
	}
}

func values(files map[string][]byte) [][]byte {
	out := [][]byte{}
	for _, b := range files {
		out = append(out, b)
	}
	return out
}

func TestExportRejectsTamperedLogWithoutPublishing(t *testing.T) {
	for _, payload := range []string{
		`{"at":"2026-01-01T00:00:00Z","event":"app_started","text":"private"}`,
		`{"at":"2026-01-01T00:00:00Z","event":"private"}`,
		`not-json`,
	} {
		t.Run(payload, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "events.0.jsonl"), []byte(payload+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			r, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "out.zip")
			if err := r.Export(path, "0.1.0"); err == nil {
				t.Fatal("tampered source accepted")
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("partial export published")
			}
			// A corrupt source is an explicit persistent sink failure.
			if err := r.Close(); err == nil {
				t.Fatal("failure not surfaced")
			}
		})
	}
}

func TestExportDoesNotFollowLogSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need Windows privileges")
	}
	dir := t.TempDir()
	private := filepath.Join(t.TempDir(), "private")
	os.WriteFile(private, []byte("PRIVATE"), 0600)
	if err := os.Symlink(private, filepath.Join(dir, "events.0.jsonl")); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestConcurrentExportAndClose(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "logs"))
	if err != nil {
		t.Fatal(err)
	}
	var writers sync.WaitGroup
	for i := 0; i < 8; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for j := 0; j < 300; j++ {
				r.Record(Entry{Event: "app_started"})
			}
		}()
	}
	exported := make(chan error, 1)
	go func() { exported <- r.Export(filepath.Join(t.TempDir(), "out.zip"), "0.1.0") }()
	writers.Wait()
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-exported; err != nil && !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	r.Record(Entry{Event: "app_started"})
	if _, err := r.snapshotFiles(); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed snapshot %v", err)
	}
}

func TestExportInvalidDestinationPreservesExistingFile(t *testing.T) {
	r := openTest(t)
	path := filepath.Join(t.TempDir(), "keep.zip")
	os.WriteFile(path, []byte("keep"), 0600)
	if err := r.Export(path, "/Users/private/version"); err == nil {
		t.Fatal("private version accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("destination changed: %q %v", data, err)
	}
	if err := r.Export(filepath.Join(r.dir, "out.zip"), "0.1.0"); err == nil {
		t.Fatal("export inside logs accepted")
	}
	if err := r.Export(filepath.Join(t.TempDir(), "missing", "out.zip"), "0.1.0"); err == nil {
		t.Fatal("unwritable destination accepted")
	}
}

func TestDailyRotationAndExpiredEntries(t *testing.T) {
	var clock atomic.Int64
	initial := time.Now().UTC()
	clock.Store(initial.UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	r, err := open(t.TempDir(), MaxFileBytes, now)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.Record(Entry{Event: "app_started"})
	if _, err := r.snapshotFiles(); err != nil {
		t.Fatal(err)
	}
	clock.Store(initial.Add(25 * time.Hour).UnixNano())
	r.Record(Entry{Event: "app_stopped"})
	files, err := r.snapshotFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("daily rotation did not occur: %d", len(files))
	}
	clock.Store(initial.Add(Retention + 48*time.Hour).UnixNano())
	r.Record(Entry{Event: "app_started"})
	files, err = r.snapshotFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || bytes.Contains(files["events.0.jsonl"], []byte("app_stopped")) {
		t.Fatalf("expired logs retained: %v", files)
	}
}

func TestExportPreservesHistoricalModelIDAndUnversionedBuild(t *testing.T) {
	dir := t.TempDir()
	historical := Entry{At: time.Now().UTC(), Event: "model_status", ModelID: "retired-model-v1", Outcome: "ready"}
	data, err := json.Marshal(historical)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "events.0.jsonl"), append(data, '\n'), 0600)
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	// New events cannot claim a model absent from this build's catalog.
	r.Record(Entry{Event: "model_status", ModelID: "private-user-text", Outcome: "ready"})
	path := filepath.Join(t.TempDir(), "logs.zip")
	if err := r.Export(path, ""); err != nil {
		t.Fatal(err)
	}
	files := unzip(t, path)
	if !bytes.Contains(files["events.0.jsonl"], []byte("retired-model-v1")) {
		t.Fatal("historical model identity lost")
	}
	if bytes.Contains(files["events.0.jsonl"], []byte("private-user-text")) {
		t.Fatal("unregistered model recorded")
	}
	if bytes.Contains(files["manifest.json"], []byte(`"version"`)) {
		t.Fatal("invented version for unversioned build")
	}
}

func TestSinkFailurePreservesDestination(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Close the idle sink to inject a real OS write failure on every platform.
	if err := r.file.Close(); err != nil {
		t.Fatal(err)
	}

	r.Record(Entry{Event: "app_started"})
	path := filepath.Join(t.TempDir(), "logs.zip")
	os.WriteFile(path, []byte("keep"), 0600)
	if err := r.Export(path, "0.1.0"); err == nil {
		t.Fatal("sink failure hidden")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("previous destination lost: %q %v", data, err)
	}
	if err := r.Close(); err == nil {
		t.Fatal("sink failure not returned")
	}
}

func TestFullExportQueueDoesNotBlockProducers(t *testing.T) {
	// Stall the worker completely. A pending export must release the producer
	// lock while waiting for queue space, so capture can continue dropping logs.
	r := &Recorder{runID: "0123456789abcdef", commands: make(chan command, 1), space: make(chan struct{}, 1), done: make(chan struct{}), now: time.Now}
	r.Record(Entry{Event: "app_started"})
	result := make(chan error, 1)
	go func() { _, err := r.snapshotFiles(); result <- err }()
	finished := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			r.Record(Entry{Event: "app_started"})
		}
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("export blocked diagnostic producers")
	}
	if r.dropped.Load() != 1000 {
		t.Fatalf("unreported pressure: %d", r.dropped.Load())
	}
	r.mu.Lock()
	r.closed = true
	close(r.done)
	r.mu.Unlock()
	select {
	case err := <-result:
		if !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("export did not close")
	}
}

func TestInputMethodSetupDiagnosticsUseClosedStages(t *testing.T) {
	for _, event := range []string{"ime_setup_started", "ime_setup_completed", "ime_setup_cancelled", "ime_setup_failed"} {
		for _, phase := range []string{"discovery", "enable", "selection", "connection"} {
			entry := Entry{At: time.Now(), Event: event, Phase: phase}
			if !entry.valid() {
				t.Fatalf("setup stage rejected: %+v", entry)
			}
		}
	}
	entry := Entry{At: time.Now(), Event: "ime_source_state", Outcome: "discovering"}
	if !entry.valid() {
		t.Fatal("discovery state lost")
	}
	entry.Phase = "private filesystem path"
	if entry.valid() {
		t.Fatal("arbitrary stage accepted")
	}
}
