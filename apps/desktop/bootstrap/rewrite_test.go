package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/models"
)

// RWT-A06: loading or missing workers must not silently disable polishing.
func TestRewritePolicyPreservesChoiceBeforeWorkerReady(t *testing.T) {
	a := &App{prefs: application.DefaultPreferences("darwin"), rewriter: &rewriting.Service{}}
	a.prefs.Punctuation = "omit"
	if policy := a.rewritePolicy(); policy.Enabled || policy.Punctuation != "" {
		t.Fatal("polish off still configures a cleanup prompt")
	}
	a.prefs.Polish = true
	if a.rewriter.Ready() {
		t.Fatal("test worker unexpectedly ready")
	}
	policy := a.rewritePolicy()
	if !policy.Enabled || policy.Punctuation != "omit" {
		t.Fatalf("worker readiness changed the choice: enabled=%v punctuation=%q", policy.Enabled, policy.Punctuation)
	}
}

func TestSupersededPolishDownloadDoesNotStart(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected download", http.StatusForbidden)
	}))
	defer server.Close()
	model := models.Rewriter{ID: rewriting.DefaultModel, Platforms: []string{"test"}, Files: map[string]models.RemoteFile{
		"model.gguf": {URL: server.URL + "/model.gguf", SHA256: strings.Repeat("a", 64), SizeBytes: 32},
	}}
	service := rewriting.New(models.Catalog{Rewriters: []models.Rewriter{model}}, provision.NewStore(t.TempDir()), t.TempDir(), 1, "test")
	defer service.Disable()
	a := &App{prefs: application.DefaultPreferences("darwin"), rewriter: service}
	a.prefs.Polish = true
	// Hold the first operation so the UI can cancel a queued confirmation,
	// switch away and then select local again before any operation starts.
	barrier := make(chan struct{})
	a.polishDone = barrier
	a.startPolish(true)
	a.snapshot.Lock()
	a.prefs.Polish = false
	a.snapshot.Unlock()
	a.startPolish(false)
	a.snapshot.Lock()
	a.prefs.Polish = true
	a.snapshot.Unlock()
	a.startPolish(false)
	close(barrier)
	select {
	case <-a.polishDone:
	case <-time.After(5 * time.Second):
		t.Fatal("local polishing operations did not finish")
	}
	if requests.Load() != 0 || service.Status().State != rewriting.Off {
		t.Fatal("a superseded confirmation authorized a download")
	}
}

func TestOriginalRemainsCopyableAfterCleanup(t *testing.T) {
	for _, outcome := range []string{"accepted", "content_dropped", "unsupported_content", "timeout", "unavailable"} {
		e := contracts.Event{Source: "那个，我想问一下。", Text: "我想问一下。", Rewrite: outcome}
		if got := dictationOriginal(e); got != e.Source {
			t.Fatalf("%s lost original", outcome)
		}
		e.Text = e.Source
		if got := dictationOriginal(e); got != "" {
			t.Fatalf("%s offered identical original", outcome)
		}
	}
}
