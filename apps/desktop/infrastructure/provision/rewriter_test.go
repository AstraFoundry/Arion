package provision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"arion.local/core/domain"
	"arion.local/models"
)

func rewriterFixture(t *testing.T, files map[string][]byte, served map[string][]byte) models.Rewriter {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := served[filepath.Base(r.URL.Path)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	r := models.Rewriter{ID: "tiny-llm", Files: map[string]models.RemoteFile{}}
	for name, body := range files {
		r.Files[name] = models.RemoteFile{URL: server.URL + "/" + name, SHA256: digest(body), SizeBytes: int64(len(body))}
	}
	return r
}

func TestEnsureRewriterVerifiesAndPublishes(t *testing.T) {
	files := map[string][]byte{"model.onnx": []byte("weights-v1"), "genai_config.json": []byte(`{"model":{}}`)}
	r := rewriterFixture(t, files, files)
	store := NewStore(t.TempDir())
	if _, err := store.InstalledRewriter(r); domain.CodeOf(err) != domain.CodeModelMissing {
		t.Fatalf("expected missing before provisioning, got %v", err)
	}
	dir, err := store.EnsureRewriter(context.Background(), r, noProgress)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != string(body) {
			t.Fatalf("%s not published", name)
		}
	}
	// Tampering after installation is detected, never loaded (MOD-A01).
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("weights-v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InstalledRewriter(r); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("tampered file accepted: %v", err)
	}
}

func TestEnsureRewriterRejectsWrongBytes(t *testing.T) {
	files := map[string][]byte{"model.onnx": []byte("weights-v1")}
	r := rewriterFixture(t, files, map[string][]byte{"model.onnx": []byte("weights-v9")})
	store := NewStore(t.TempDir())
	if _, err := store.EnsureRewriter(context.Background(), r, noProgress); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("wrong bytes accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, r.ID)); !os.IsNotExist(err) {
		t.Fatal("unverified rewriter published")
	}
}

func TestEnsureRewriterOffline(t *testing.T) {
	files := map[string][]byte{"model.onnx": []byte("weights-v1")}
	r := rewriterFixture(t, files, files)
	store := NewStore(t.TempDir())
	store.Offline = true
	if _, err := store.EnsureRewriter(context.Background(), r, noProgress); domain.CodeOf(err) != domain.CodeProvisioningFailed {
		t.Fatalf("offline store downloaded: %v", err)
	}
}
