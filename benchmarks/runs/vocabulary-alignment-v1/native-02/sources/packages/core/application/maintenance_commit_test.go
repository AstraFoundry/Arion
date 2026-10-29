package application

import (
	"context"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// Cancellation invalidates the result but does not prove an adapter's pending
// OS insertion has returned. Maintenance must wait for that ownership too.
func TestMaintenanceWaitsForCancelledReviewedCommit(t *testing.T) {
	entered, releaseCommit := make(chan struct{}), make(chan struct{})
	target := &fakeTarget{snapshot: contracts.TargetSnapshot{Target: "owned-editor"}}
	target.commitHook = func(ctx context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
		close(entered)
		<-releaseCommit // Model an OS call that cannot return immediately on cancel.
		return contracts.CommitResult{Identity: request.Identity, Outcome: contracts.Unknown}, nil
	}
	recognizer := &fakeRecognizer{words: []string{"reviewed text"}}
	c, audio, events := setup(t, recognizer, target, Options{})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	// Review-only sessions capture their target only when auto-commit is enabled;
	// supply the same captured capability a review workflow owns before commit.
	c.mu.Lock()
	c.current.target = target.snapshot
	c.current.hasTarget = true
	c.mu.Unlock()
	audio.push(1, 0.2)
	events.wait(t, kindIs(contracts.DraftReplaced))
	c.Stop()
	events.wait(t, kindIs(contracts.CorrectionReady))
	// The recognition worker must have ended, so it cannot accidentally hide
	// the reviewed-commit ownership bug under its own teardown count.
	deadline := time.Now().Add(time.Second)
	for c.workers.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if c.workers.Load() != 0 {
		t.Fatal("recognition worker did not finish before review")
	}
	done := make(chan struct{})
	go func() { c.CommitReviewed(); close(done) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("reviewed commit did not reach adapter")
	}
	c.Cancel()
	if c.Snapshot().Active {
		t.Fatal("cancel did not invalidate reviewed session")
	}
	if release, ok := c.TryMaintenance(); ok {
		release()
		close(releaseCommit)
		<-done
		t.Fatal("maintenance acquired while cancelled insertion was still pending")
	}
	close(releaseCommit)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reviewed commit did not finish")
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if release, ok := c.TryMaintenance(); ok {
			release()
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("finished insertion did not release maintenance ownership")
}
