package provision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"testing"

	"arion.local/core/domain"
	"arion.local/models"
)

func host(t *testing.T, server *httptest.Server) string {
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}

// An unreachable host fails over to its mirror without the user's
// involvement; later files go straight to the source that worked.
func TestMirrorFallbackIsSilentAndRemembered(t *testing.T) {
	files := map[string][]byte{"a.onnx": []byte("weights-a"), "b.json": []byte(`{"b":1}`)}
	var mirrorHits atomic.Int32
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mirrorHits.Add(1)
		_, _ = w.Write(files[filepath.Base(r.URL.Path)])
	}))
	t.Cleanup(mirror.Close)
	blocked := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	primary := host(t, blocked)
	blocked.Close() // connections to the primary are refused
	r := models.Rewriter{ID: "mirrored", Files: map[string]models.RemoteFile{}}
	for name, body := range files {
		r.Files[name] = models.RemoteFile{URL: "http://" + primary + "/repo/resolve/rev/" + name, SHA256: digest(body), SizeBytes: int64(len(body))}
	}
	store := NewStore(t.TempDir())
	store.Mirrors = map[string][]string{primary: {host(t, mirror)}}
	if _, err := store.EnsureRewriter(context.Background(), r, noProgress); err != nil {
		t.Fatalf("mirror not used: %v", err)
	}
	if mirrorHits.Load() != 2 {
		t.Fatalf("mirror served %d files", mirrorHits.Load())
	}
	if got := store.sources(r.Files["a.onnx"].URL)[0]; got != "http://"+host(t, mirror)+"/repo/resolve/rev/a.onnx" {
		t.Fatalf("working mirror not preferred: %s", got)
	}
}

// A mirror cannot substitute different bytes: digests still decide.
func TestMirrorBytesStillVerified(t *testing.T) {
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("tampered!")) }))
	t.Cleanup(mirror.Close)
	blocked := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	primary := host(t, blocked)
	blocked.Close()
	body := []byte("weights-a")
	r := models.Rewriter{ID: "mirrored", Files: map[string]models.RemoteFile{
		"a.onnx": {URL: "http://" + primary + "/a.onnx", SHA256: digest(body), SizeBytes: int64(len(body))}}}
	store := NewStore(t.TempDir())
	store.Mirrors = map[string][]string{primary: {host(t, mirror)}}
	if _, err := store.EnsureRewriter(context.Background(), r, noProgress); domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("tampered mirror bytes accepted: %v", err)
	}
}

func TestDefaultMirrorsCoverHuggingFace(t *testing.T) {
	store := NewStore(t.TempDir())
	got := store.sources("https://huggingface.co/org/repo/resolve/rev/model.onnx")
	if len(got) != 2 || got[1] != "https://hf-mirror.com/org/repo/resolve/rev/model.onnx" {
		t.Fatalf("sources = %v", got)
	}
	if got := store.sources("https://github.com/k2-fsa/x.tar.bz2"); len(got) != 1 {
		t.Fatalf("unmirrored host gained sources: %v", got)
	}
}
