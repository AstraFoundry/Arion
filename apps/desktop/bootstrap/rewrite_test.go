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
	if a.rewritePolicy().Enabled {
		t.Fatal("polishing enabled without the user's choice")
	}
	a.prefs.Polish = true
	if a.rewriter.Ready() {
		t.Fatal("test worker unexpectedly ready")
	}
	if !a.rewritePolicy().Enabled {
		t.Fatal("worker readiness silently disabled the user's choice")
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
