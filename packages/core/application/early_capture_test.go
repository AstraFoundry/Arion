package application

import (
	"context"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// gatedTarget holds the target lookup until released, like a slow
// accessibility query.
type gatedTarget struct {
	*fakeTarget
	entered chan struct{}
	release chan struct{}
}

func (g *gatedTarget) Capture(ctx context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	close(g.entered)
	select {
	case <-g.release:
	case <-ctx.Done():
		return contracts.TargetSnapshot{}, ctx.Err()
	}
	return g.fakeTarget.Capture(ctx, id)
}

func gatedSession(t *testing.T, rec *fakeRecognizer) (*Coordinator, *fakeAudio, *recorder, *gatedTarget) {
	t.Helper()
	target := &gatedTarget{fakeTarget: &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}},
		entered: make(chan struct{}), release: make(chan struct{})}
	audio := &fakeAudio{}
	c, err := NewCoordinator(Dependencies{Audio: audio, Recognizers: fakeFactory{rec}, NewID: newID, Targets: target, Input: target}, Options{AutoCommit: true})
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	return c, audio, events, target
}

// SPH-14: speech during setup is recognized once setup finishes.
func TestSpeechDuringPreparationIsRecognized(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"early", "words", "survive", "setup"}}
	c, audio, events, target := gatedSession(t, rec)
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	<-target.entered
	c.mu.Lock()
	phase, recognizerReady := c.current.audio, c.current.recognizer != nil
	c.mu.Unlock()
	if phase != audioQueued || recognizerReady {
		t.Fatalf("early audio phase %v recognizer ready %v", phase, recognizerReady)
	}
	audio.push(3, .2) // the target lookup is still running
	if rec.n != 0 {
		t.Fatalf("recognizer received %d frames before setup finished", rec.n)
	}
	close(target.release)
	events.wait(t, func(e contracts.Event) bool {
		return e.Kind == contracts.DraftReplaced && e.Text == "early words survive"
	})
	c.Stop()
	done := events.wait(t, kindIs(contracts.CommitResultReady))
	if rec.n != 3 || done.Outcome != contracts.Applied {
		t.Fatalf("frames=%d outcome=%v", rec.n, done.Outcome)
	}
}

func TestSystemIMEWaitsForCompositionBeforeStartingAudio(t *testing.T) {
	ime := &gatedTarget{
		fakeTarget: &fakeTarget{snapshot: contracts.TargetSnapshot{Target: "ime"}},
		entered:    make(chan struct{}), release: make(chan struct{}),
	}
	audio := &fakeAudio{}
	c, err := NewCoordinator(Dependencies{
		Audio: audio, Recognizers: fakeFactory{&fakeRecognizer{}}, NewID: newID,
		Targets: ime, Input: ime, IMETargets: ime, IMEInput: ime,
	}, Options{AutoCommit: true, Mode: domain.SystemIMEMode})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Cancel)
	events := &recorder{}
	c.Subscribe(events.listen)
	c.Start()
	select {
	case <-ime.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("IME composition acquisition did not start")
	}
	if n := audio.starts.Load(); n != 0 {
		t.Fatalf("microphone started %d times before composition confirmation", n)
	}
	close(ime.release)
	events.wait(t, stateIs(domain.Capturing))
	if n := audio.starts.Load(); n != 1 {
		t.Fatalf("microphone started %d times after composition confirmation, want 1", n)
	}
}

// Stopping while setup still runs finishes the audio already queued.
func TestStopDuringPreparationFinishesQueuedAudio(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"quick", "tap"}}
	c, audio, events, target := gatedSession(t, rec)
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	<-target.entered
	audio.push(2, .2)
	c.Stop()
	c.mu.Lock()
	phase := c.current.audio
	c.mu.Unlock()
	if phase != audioQueued {
		t.Fatalf("stop abandoned queued audio: %v", phase)
	}
	close(target.release)
	done := events.wait(t, kindIs(contracts.CommitResultReady))
	if rec.n != 2 || done.Outcome != contracts.Applied || done.Text == "" {
		t.Fatalf("frames=%d event=%+v", rec.n, done)
	}
}

// Cancelling during setup drops the queued audio without recognizing it.
func TestCancelDuringPreparationDropsQueuedAudio(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"secret"}}
	c, audio, events, target := gatedSession(t, rec)
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	<-target.entered
	audio.push(3, .2)
	c.Cancel()
	c.mu.Lock()
	phase := c.current.audio
	c.mu.Unlock()
	if phase != audioAbandoned {
		t.Fatalf("cancel left queued audio in %v", phase)
	}
	events.wait(t, stateIs(domain.Cancelled))
	audio.released(t)
	if rec.n != 0 {
		t.Fatalf("%d queued frames were recognized after cancel", rec.n)
	}
}
