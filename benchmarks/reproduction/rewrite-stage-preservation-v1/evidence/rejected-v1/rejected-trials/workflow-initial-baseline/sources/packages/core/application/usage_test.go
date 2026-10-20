package application

import (
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func usageEvent(session string, kind contracts.EventKind, state domain.SessionState, atMs float64) contracts.Event {
	return contracts.Event{Envelope: contracts.Envelope{Identity: contracts.Identity{SessionID: session}, Kind: kind}, State: state, AtMs: atMs}
}

func TestUsageMeterCountsCommittedSessions(t *testing.T) {
	var meter UsageMeter
	feed := func(events ...contracts.Event) (domain.Usage, bool) {
		var usage domain.Usage
		var done bool
		for _, e := range events {
			if u, ok := meter.Observe(e); ok {
				usage, done = u, true
			}
		}
		return usage, done
	}
	ready := usageEvent("a", contracts.CorrectionReady, domain.Reviewing, 2600)
	ready.Rewrite, ready.Scene, ready.Source, ready.Text = string(domain.RewriteAccepted), "email", "hi there", "Hi there,"
	commit := usageEvent("a", contracts.CommitResultReady, domain.Completed, 2700)
	commit.Outcome, commit.Text = contracts.Applied, "Hi there, 好的"
	usage, ok := feed(usageEvent("a", contracts.SessionStateChanged, domain.Preparing, 0),
		usageEvent("a", contracts.SessionStateChanged, domain.Capturing, 120),
		usageEvent("b", contracts.SessionStateChanged, domain.Finalizing, 900),
		usageEvent("a", contracts.SessionStateChanged, domain.Finalizing, 2120),
		ready, commit, commit)
	if !ok || usage != (domain.Usage{SpeechMs: 2000, Words: 4, Scene: domain.SceneEmail}) {
		t.Fatalf("committed: %+v %v", usage, ok)
	}

	// A rejected rewrite commits the recognized text: no scene.
	rejected := usageEvent("c", contracts.CorrectionReady, domain.Reviewing, 1000)
	rejected.Rewrite, rejected.Scene, rejected.Source, rejected.Text = string(domain.RewriteLength), "email", "ok", "ok"
	commit = usageEvent("c", contracts.CommitResultReady, domain.Completed, 1100)
	commit.Outcome, commit.Text = contracts.Unknown, "ok"
	usage, ok = feed(usageEvent("c", contracts.SessionStateChanged, domain.Preparing, 0), rejected, commit)
	if !ok || usage.Scene != "" || usage.Words != 1 {
		t.Fatalf("rejected rewrite: %+v %v", usage, ok)
	}

	failed := usageEvent("d", contracts.CommitResultReady, domain.Failed, 900)
	failed.Outcome, failed.Text = contracts.NotApplied, "lost"
	if _, ok := feed(usageEvent("d", contracts.SessionStateChanged, domain.Preparing, 0), failed); ok {
		t.Fatal("a failed commit counted")
	}
}
