//go:build arion_e2e && darwin

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/macos"
	"arion.local/desktop/infrastructure/wordseg"
)

// MEM-A09 / MEM-A11 on a real AppKit field: the production target adapter
// inserts, keeps following the field, and the tracker learns the word the
// user retyped; a secure field is never followed. Needs Accessibility for
// the test runner, granted by a person, so it runs only with
// ARION_E2E_LEARNING=1.
func TestLearnFromEditInOwnedEditor(t *testing.T) {
	if os.Getenv("ARION_E2E_LEARNING") != "1" {
		t.Skip("set ARION_E2E_LEARNING=1 after granting Accessibility to the test runner")
	}
	if (macos.Permissions{}).Accessibility() != macos.AuthGranted {
		t.Fatal("ARION_E2E_LEARNING=1 requires Accessibility access for the test runner")
	}
	request := startEditorFixture(t)
	request("reset", nil)
	eventually(t, "editor A focused", func() bool {
		state := request("focus", map[string]any{"target": "A"})
		return state.Active && state.Focused == "A"
	})
	targets := macos.NewTargets()
	id := contracts.Identity{SessionID: "learn", Generation: 1}
	target, err := targets.Capture(context.Background(), id)
	if err != nil || target.Sensitive {
		t.Fatalf("capture: %+v %v", target, err)
	}
	if route := targets.RouteOf(target.Target); route != macos.RouteClipboard {
		t.Fatalf("native editor route %q, want clipboard", route)
	}
	if !targets.Follow(target.Target) {
		t.Fatal("a readable editor was not followed")
	}
	const inserted = "ask jason about the release"
	result, err := targets.Commit(context.Background(), contracts.CommitRequest{Identity: id, Target: target.Target, Text: inserted})
	if err != nil || result.Outcome == contracts.NotApplied {
		t.Fatalf("commit: %+v %v", result, err)
	}
	reports := make(chan application.EditReport, 1)
	tracker := &application.EditTracker{Segmenter: wordseg.Segmenter{}, Poll: 100 * time.Millisecond, Idle: time.Second, Report: func(r application.EditReport) { reports <- r }}
	tracker.Track(targets, target.Target, id, inserted)
	time.Sleep(400 * time.Millisecond)
	if state := request("replace", map[string]any{"target": "A", "from": "jason", "to": "Jayson"}); state.A.Text != "ask Jayson about the release" {
		t.Fatalf("fixture text %q", state.A.Text)
	}
	select {
	case r := <-reports:
		if r.Settlement != application.SettleIdle || r.Outcome.Verdict != domain.EditLearned ||
			len(r.Outcome.Pairs) != 1 || r.Outcome.Pairs[0] != (domain.Replacement{From: "jason", To: "Jayson"}) {
			t.Fatalf("report %+v", r)
		}
		t.Logf("settled after %d reads, %d failures", r.Reads, r.Failures)
	case <-time.After(10 * time.Second):
		t.Fatal("no edit report")
	}
	if _, err := targets.ReadField(context.Background(), target.Target); err == nil {
		t.Fatal("field still readable after the tracker released it")
	}

	eventually(t, "secure field focused", func() bool { return request("focus", map[string]any{"target": "secure"}).Focused == "secure" })
	secure, err := targets.Capture(context.Background(), contracts.Identity{SessionID: "learn", Generation: 2})
	if err == nil && targets.Follow(secure.Target) {
		t.Fatal("a secure field was followed")
	}
}
