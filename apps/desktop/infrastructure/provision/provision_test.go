package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"arion.local/core/domain"
	"arion.local/models"
)

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func fixture(t *testing.T) ([]byte, models.Artifact) {
	t.Helper()
	archive, err := os.ReadFile("testdata/model.tar.bz2")
	if err != nil {
		t.Fatal(err)
	}
	encoder, tokens := []byte("encoder-bytes-v1"), []byte("a 1\nb 2\n")
	artifact := models.Artifact{
		ID:     "tiny",
		Source: models.Source{SHA256: digest(archive), SizeBytes: int64(len(archive))},
		Files: map[string]models.File{
			"encoder": {Member: "model/encoder.onnx", SHA256: digest(encoder), SizeBytes: int64(len(encoder))},
			"tokens":  {Member: "model/tokens.txt", SHA256: digest(tokens), SizeBytes: int64(len(tokens))},
		},
	}
	return archive, artifact
}

// serve serves body, honoring byte ranges like a release CDN.
func serve(t *testing.T, body []byte, requests *atomic.Int32) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if value := r.Header.Get("Range"); value != "" {
			start, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(value, "bytes="), "-"))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(body[start:])
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

func noProgress(string, Phase, int64, int64) {}

func TestEnsureVerifiesAndPublishes(t *testing.T) {
	archive, artifact := fixture(t)
	var requests atomic.Int32
	artifact.Source.URL = serve(t, archive, &requests).URL
	store := NewStore(t.TempDir())
	dir, err := store.Ensure(context.Background(), artifact, noProgress)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "encoder.onnx")); string(data) != "encoder-bytes-v1" {
		t.Fatal("encoder not published")
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
		t.Fatal("unlisted archive member extracted")
	}
	// A verified install loads without the network.
	store.Offline = true
	if _, err := store.Ensure(context.Background(), artifact, noProgress); err != nil || requests.Load() != 1 {
		t.Fatalf("reinstall: %v requests=%d", err, requests.Load())
	}
}

// MOD-A01: a corrupt archive or file never loads.
func TestCorruptArtifactRejected(t *testing.T) {
	archive, artifact := fixture(t)
	corrupt := append([]byte(nil), archive...)
	corrupt[len(corrupt)/2] ^= 0xff
	var requests atomic.Int32
	artifact.Source.URL = serve(t, corrupt, &requests).URL
	store := NewStore(t.TempDir())
	if _, err := store.Ensure(context.Background(), artifact, noProgress); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("corrupt archive accepted: %v", err)
	}
	if _, err := store.Installed(artifact); err == nil {
		t.Fatal("corrupt artifact reported installed")
	}
}

// MOD-A01: a tampered file in a verified install is detected at load.
func TestTamperedInstallDetected(t *testing.T) {
	archive, artifact := fixture(t)
	var requests atomic.Int32
	artifact.Source.URL = serve(t, archive, &requests).URL
	store := NewStore(t.TempDir())
	dir, err := store.Ensure(context.Background(), artifact, noProgress)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "encoder.onnx"), []byte("encoder-bytes-v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Installed(artifact); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("tampered install accepted: %v", err)
	}
}

// MOD-A04: a failed replacement keeps the previous verified install.
func TestFailedReplacementKeepsPrevious(t *testing.T) {
	archive, artifact := fixture(t)
	var requests atomic.Int32
	artifact.Source.URL = serve(t, archive, &requests).URL
	store := NewStore(t.TempDir())
	if _, err := store.Ensure(context.Background(), artifact, noProgress); err != nil {
		t.Fatal(err)
	}
	broken := artifact
	broken.Files = map[string]models.File{"encoder": {Member: "model/encoder.onnx", SHA256: strings.Repeat("0", 64), SizeBytes: 16}}
	// Force a re-download by changing the archive identity the marker expects.
	if err := os.WriteFile(filepath.Join(store.Root, artifact.ID, ".verified.json"), []byte(`{"archiveSha256":"stale"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Ensure(context.Background(), broken, noProgress); err == nil {
		t.Fatal("unverifiable replacement accepted")
	}
	if data, _ := os.ReadFile(filepath.Join(store.Root, artifact.ID, "encoder.onnx")); string(data) != "encoder-bytes-v1" {
		t.Fatal("previous install was removed before a replacement verified")
	}
}

// Downloads resume and refuse to exceed the catalog size.
func TestResumeAndSizeBound(t *testing.T) {
	archive, artifact := fixture(t)
	var requests atomic.Int32
	artifact.Source.URL = serve(t, archive, &requests).URL
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(filepath.Join(store.Root, ".downloads"), 0o700); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(store.Root, ".downloads", artifact.Source.SHA256+".tar.bz2.partial")
	if err := os.WriteFile(partial, archive[:100], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Ensure(context.Background(), artifact, noProgress); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	oversized := artifact
	oversized.ID = "oversized"
	oversized.Source.SizeBytes = int64(len(archive) - 10)
	if _, err := NewStore(t.TempDir()).Ensure(context.Background(), oversized, noProgress); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("oversized archive not rejected: %v", err)
	}
	offline := NewStore(t.TempDir())
	offline.Offline = true
	if _, err := offline.Ensure(context.Background(), artifact, noProgress); domain.CodeOf(err) != domain.CodeProvisioningFailed {
		t.Fatalf("offline store downloaded: %v", err)
	}
}
