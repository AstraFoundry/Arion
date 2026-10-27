package application

import (
	"fmt"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

func TestFinalOnlyContextSourcesFollowSwitches(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			sources := ContextPolicy{Cursor: mask&1 != 0, Screen: mask&2 != 0, History: mask&4 != 0}
			history := NewHistory(time.Hour)
			history.Add("com.apple.mail", "HISTORY_CANARY")
			rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
			c, audio, events, target, _ := liveSetup(t, []string{"hello"}, rewriter, RewritePolicy{Enabled: true, Scene: "email", Context: sources}, history)
			target.snapshot.Application = "com.apple.mail"
			target.context = contracts.ContextSnapshot{SafeToRead: true, Before: "BEFORE_CANARY", After: "AFTER_CANARY", Screen: "SCREEN_CANARY"}
			c.Start()
			events.wait(t, stateIs(domain.Capturing))
			audio.push(1, 0.2)
			c.Stop()
			events.wait(t, kindIs(contracts.CommitResultReady))
			rewriter.mu.Lock()
			defer rewriter.mu.Unlock()
			if len(rewriter.requests) != 1 {
				t.Fatalf("final-only requests: %d", len(rewriter.requests))
			}
			got := rewriter.requests[0].Context
			if (got.Before != "") != sources.Cursor || (got.After != "") != sources.Cursor || (got.Screen != "") != sources.Screen || (len(got.History) > 0) != sources.History {
				t.Fatalf("sources %+v yielded %+v", sources, got)
			}
			if !sources.Cursor && !sources.Screen && target.reads.Load() != 0 {
				t.Fatal("read target with both sources disabled")
			}
		})
	}
}
