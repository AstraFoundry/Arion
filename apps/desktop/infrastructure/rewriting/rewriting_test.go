package rewriting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

func TestTimeoutIsScopedToWindowsCPU(t *testing.T) {
	if Timeout("windows-amd64") != 30*time.Second || Timeout("darwin-arm64") != 0 || Timeout("unsupported") != 0 {
		t.Fatal("local CPU timeout escaped the supported Windows platform")
	}
}

func TestLocateRequiresBothPlatformExecutables(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, runtime.GOOS+"-"+runtime.GOARCH, "llm")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, workerName()), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if Locate(root) != "" {
		t.Fatal("reported readiness without native runtime")
	}
	if err := os.WriteFile(filepath.Join(dir, runtimeName()), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if Locate(root) != dir {
		t.Fatal("did not locate the platform runtime pair")
	}
}

func TestWindowsCatalogCapabilityRemainsOffUntilProvisioned(t *testing.T) {
	catalog, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := New(catalog, provision.NewStore(t.TempDir()), t.TempDir(), 4, "windows-amd64")
	if s.Status().State != Off || s.Ready() {
		t.Fatal("Windows support must not fabricate ready state")
	}
	if err := s.Enable(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.Status().State != Off || s.Ready() {
		t.Fatal("missing model became ready")
	}
}

func downloadFixture(t *testing.T, handler http.HandlerFunc) (*Service, *provision.Store) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	store := provision.NewStore(t.TempDir())
	model := models.Rewriter{ID: DefaultModel, Platforms: []string{"test"}, Files: map[string]models.RemoteFile{
		"model.gguf": {URL: server.URL + "/model.gguf", SHA256: strings.Repeat("a", 64), SizeBytes: 32},
	}}
	s := New(models.Catalog{Rewriters: []models.Rewriter{model}}, store, t.TempDir(), 1, "test")
	t.Cleanup(s.Disable)
	return s, store
}

// LRW-07: selecting local polishing and restarting never authorize a download.
func TestEnableDoesNotDownload(t *testing.T) {
	var requests atomic.Int32
	s, store := downloadFixture(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected download", http.StatusForbidden)
	})
	for range 2 {
		s.Refresh()
		if err := s.Enable(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if status := s.Status(); status.State != Off || status.Installed || status.Failure != "" {
		t.Fatalf("missing model should await confirmation: %+v", status)
	}
	if requests.Load() != 0 {
		t.Fatal("enabling a missing model made a network request")
	}
	if _, err := os.Stat(filepath.Join(store.Root, ".downloads")); !os.IsNotExist(err) {
		t.Fatalf("enabling created download state: %v", err)
	}
	// Corruption is an explicit failure, rather than an automatic re-download.
	dir := filepath.Join(store.Root, DefaultModel)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".verified.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Enable(context.Background()); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("corruption was not reported: %v", err)
	}
	if requests.Load() != 0 || s.Status().State != Failed {
		t.Fatal("corruption triggered an unconfirmed repair download")
	}
}

func TestExplicitDownloadCanBeCancelledWithoutAutomaticResume(t *testing.T) {
	started := make(chan struct{})
	var requests atomic.Int32
	s, _ := downloadFixture(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.Download(ctx) }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("explicit download did not start")
	}
	s.Disable()
	select {
	case err := <-done:
		if domain.CodeOf(err) != domain.CodeCancelled {
			t.Fatalf("download did not cancel: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("download did not stop")
	}
	if err := s.Enable(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || s.Status().State != Off {
		t.Fatalf("cancelled download resumed without confirmation: %d %+v", requests.Load(), s.Status())
	}
}
