package bootstrap

import (
	"testing"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/desktop/infrastructure/rewriting"
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
