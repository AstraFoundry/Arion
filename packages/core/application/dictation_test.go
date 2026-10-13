package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type fakeAudio struct {
	starts atomic.Int32
	sink   ports.FrameSink
	mu     sync.Mutex
	err    error
}

type fakeStream struct{ stopped atomic.Bool }

func (s *fakeStream) Stop() error { s.stopped.Store(true); return nil }

func (a *fakeAudio) Start(_ context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	a.starts.Add(1)
	if a.err != nil {
		return nil, a.err
	}
	a.mu.Lock()
	a.sink = sink
	a.mu.Unlock()
	return &fakeStream{}, nil
}

func (a *fakeAudio) push(n int, amplitude float32) {
	a.mu.Lock()
	sink := a.sink
	a.mu.Unlock()
	for i := 0; i < n; i++ {
		samples := make([]float32, 320)
		for j := range samples {
			samples[j] = amplitude
		}
		sink(contracts.Frame{Sequence: uint64(i), SampleRate: 16000, Channels: 1, Samples: samples})
	}
}

// fakeRecognizer emits one more word per accepted frame; Finish can block.
type fakeRecognizer struct {
	words   []string
	n       int
	finish  chan struct{}
	delay   time.Duration
	closed  atomic.Bool
	silence bool
}

func (r *fakeRecognizer) Accept(context.Context, contracts.Frame) error {
	time.Sleep(r.delay)
	r.n++
	return nil
}
func (r *fakeRecognizer) text() string {
	if r.silence {
		return ""
	}
	return strings.Join(r.words[:min(r.n, len(r.words))], " ")
}
func (r *fakeRecognizer) Partial(context.Context) (contracts.Draft, error) {
	return contracts.Draft{Text: r.text()}, nil
}
func (r *fakeRecognizer) Finish(ctx context.Context) (contracts.Draft, error) {
	if r.finish != nil {
		<-r.finish
	}
	return contracts.Draft{Text: r.text(), Final: true}, nil
}
func (r *fakeRecognizer) Close() error { r.closed.Store(true); return nil }

type fakeFactory struct{ r *fakeRecognizer }

func (f fakeFactory) NewRecognizer(context.Context, contracts.Identity) (ports.Recognizer, error) {
	if f.r == nil {
		return nil, domain.Fail(domain.CodeModelMissing, "not provisioned")
	}
	return f.r, nil
}

type fakeTarget struct {
	snapshot contracts.TargetSnapshot
	err      error
	commits  atomic.Int32
	outcome  contracts.CommitOutcome
	texts    []string
	mu       sync.Mutex
}

func (t *fakeTarget) Capture(context.Context, contracts.Identity) (contracts.TargetSnapshot, error) {
	return t.snapshot, t.err
}
func (t *fakeTarget) Capabilities(context.Context, contracts.TargetToken) (contracts.Capabilities, error) {
	return contracts.Capabilities{SingleInsertion: true}, nil
}
func (t *fakeTarget) ReadContext(context.Context, contracts.Identity, contracts.TargetToken) (contracts.ContextSnapshot, error) {
	return contracts.ContextSnapshot{}, nil
}
func (t *fakeTarget) Commit(_ context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	t.commits.Add(1)
	t.mu.Lock()
	t.texts = append(t.texts, request.Text)
	t.mu.Unlock()
	return contracts.CommitResult{Identity: request.Identity, Outcome: t.outcome}, nil
}

type recorder struct {
	mu     sync.Mutex
	events []contracts.Event
}

func (r *recorder) listen(e contracts.Event) {
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
}
func (r *recorder) wait(t *testing.T, match func(contracts.Event) bool) contracts.Event {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for _, e := range r.events {
			if match(e) {
				r.mu.Unlock()
				return e
			}
		}
		r.mu.Unlock()
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("event not observed; got %d events", len(r.events))
	return contracts.Event{}
}
func (r *recorder) count(match func(contracts.Event) bool) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if match(e) {
			n++
		}
	}
	return n
}

func stateIs(s domain.SessionState) func(contracts.Event) bool {
	return func(e contracts.Event) bool { return e.Kind == contracts.SessionStateChanged && e.State == s }
}
func kindIs(k contracts.EventKind) func(contracts.Event) bool {
	return func(e contracts.Event) bool { return e.Kind == k }
}

var ids atomic.Int32

func newID() string { return fmt.Sprintf("s%d", ids.Add(1)) }

func setup(t *testing.T, rec *fakeRecognizer, target *fakeTarget, opts Options) (*Coordinator, *fakeAudio, *recorder) {
	t.Helper()
	audio := &fakeAudio{}
	deps := Dependencies{Audio: audio, Recognizers: fakeFactory{rec}, NewID: newID}
	if target != nil {
		deps.Targets, deps.Input = target, target
	}
	c, err := NewCoordinator(deps, opts)
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	return c, audio, events
}

// DSK-A02: toggle, revised preview, stop, exactly one corrected commit.
func TestDictationCommitsOnce(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"你", "好", "ARION"}}
	target := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}}
	c, audio, events := setup(t, rec, target, Options{AutoCommit: true, Correction: true,
		Policy: func() domain.CorrectionPolicy { return domain.CorrectionPolicy{LatinCase: domain.LatinCaseUpper} }})
	c.Toggle()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(3, 0.2)
	events.wait(t, func(e contracts.Event) bool { return e.Kind == contracts.DraftReplaced && e.Text == "你 好 ARION" })
	c.Toggle()
	done := events.wait(t, kindIs(contracts.CommitResultReady))
	if done.Outcome != contracts.Applied || done.State != domain.Completed || done.Text != "你好 arion" {
		t.Fatalf("unexpected commit event %+v", done)
	}
	c.CommitReviewed()
	if target.commits.Load() != 1 {
		t.Fatalf("committed %d times", target.commits.Load())
	}
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Raw != "你 好 ARION" {
		t.Fatalf("raw ASR text not preserved: %q", ready.Raw)
	}
	var last uint64
	var lastRevision uint64
	events.mu.Lock()
	for _, e := range events.events {
		if e.Sequence <= last {
			t.Fatal("event sequence not increasing")
		}
		last = e.Sequence
		if e.Kind == contracts.DraftReplaced || e.Kind == contracts.CorrectionReady {
			if e.Revision <= lastRevision {
				t.Fatal("revision not increasing")
			}
			lastRevision = e.Revision
		}
	}
	events.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	if !rec.closed.Load() {
		t.Fatal("recognizer not released")
	}
}

// DSK-A04 / SPH-A02: cancel during finalization; late results never commit.
func TestCancelDuringFinalizeDropsLateResult(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"late"}, finish: make(chan struct{})}
	target := &fakeTarget{outcome: contracts.Applied}
	c, audio, events := setup(t, rec, target, Options{AutoCommit: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	events.wait(t, stateIs(domain.Finalizing))
	time.Sleep(10 * time.Millisecond)
	c.Cancel()
	close(rec.finish)
	time.Sleep(50 * time.Millisecond)
	if target.commits.Load() != 0 || events.count(kindIs(contracts.CorrectionReady)) != 0 {
		t.Fatal("cancelled session produced review or commit")
	}
	if c.Snapshot().State != domain.Cancelled {
		t.Fatalf("state %s", c.Snapshot().State)
	}
}

// SPH-A07: silence produces no commit.
func TestSilenceDoesNotCommit(t *testing.T) {
	rec := &fakeRecognizer{silence: true}
	target := &fakeTarget{outcome: contracts.Applied}
	c, audio, events := setup(t, rec, target, Options{AutoCommit: true, Correction: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(5, 0)
	c.Stop()
	failed := events.wait(t, kindIs(contracts.ErrorOccurred))
	if failed.Error != domain.CodeNoSpeech || target.commits.Load() != 0 {
		t.Fatalf("silence: %+v commits=%d", failed, target.commits.Load())
	}
}

// SPH-A04: a full bounded queue reports an explicit discontinuity.
func TestQueueOverflowFailsExplicitly(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"a", "b"}, delay: 20 * time.Millisecond}
	c, audio, events := setup(t, rec, nil, Options{QueueFrames: 2})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(50, 0.1)
	failed := events.wait(t, kindIs(contracts.ErrorOccurred))
	if failed.Error != domain.CodeAudioDiscontinuity {
		t.Fatalf("got %s", failed.Error)
	}
}

// DSK-A07 / DSK-A01: sensitive or untrusted targets never open the microphone.
func TestTargetGuardsBlockCapture(t *testing.T) {
	for _, tc := range []struct {
		target *fakeTarget
		code   domain.ErrorCode
	}{
		{&fakeTarget{snapshot: contracts.TargetSnapshot{Sensitive: true}}, domain.CodeSecureFieldBlocked},
		{&fakeTarget{err: domain.Fail(domain.CodePermissionDenied, "accessibility")}, domain.CodePermissionDenied},
	} {
		c, audio, events := setup(t, &fakeRecognizer{}, tc.target, Options{AutoCommit: true})
		c.Start()
		failed := events.wait(t, kindIs(contracts.ErrorOccurred))
		if failed.Error != tc.code || audio.starts.Load() != 0 {
			t.Fatalf("want %s without capture, got %s starts=%d", tc.code, failed.Error, audio.starts.Load())
		}
	}
}

// DSK-09: a missing model is an explicit error, not a silent fallback.
func TestMissingModelIsExplicit(t *testing.T) {
	c, audio, events := setup(t, nil, nil, Options{})
	c.Start()
	failed := events.wait(t, kindIs(contracts.ErrorOccurred))
	if failed.Error != domain.CodeModelMissing || audio.starts.Load() != 0 {
		t.Fatalf("got %s", failed.Error)
	}
}

// CTR-A06: an unknown insertion outcome is final and is not retried.
func TestUnknownOutcomeNotRetried(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"hello"}}
	target := &fakeTarget{outcome: contracts.Unknown}
	c, audio, events := setup(t, rec, target, Options{AutoCommit: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.1)
	c.Stop()
	events.wait(t, kindIs(contracts.CommitResultReady))
	c.CommitReviewed()
	c.Toggle() // starts a new session; must not resend old text
	time.Sleep(20 * time.Millisecond)
	if target.commits.Load() != 1 {
		t.Fatalf("unknown outcome retried: %d", target.commits.Load())
	}
}

// DSK-A06: switching modes closes the active generation.
func TestModeSwitchCancels(t *testing.T) {
	c, _, events := setup(t, &fakeRecognizer{words: []string{"x"}}, nil, Options{})
	c.Start()
	first := events.wait(t, stateIs(domain.Capturing))
	c.SetMode(domain.SystemIMEMode)
	events.wait(t, stateIs(domain.Cancelled))
	c.Start()
	second := events.wait(t, func(e contracts.Event) bool {
		return e.Kind == contracts.SessionStateChanged && e.State == domain.Capturing && e.SessionID != first.SessionID
	})
	if second.Generation <= first.Generation {
		t.Fatal("new session did not get a newer generation")
	}
}

type fakeComposition struct {
	mu        sync.Mutex
	revisions []uint64
	texts     []string
	cancels   int
}

func (f *fakeComposition) Update(_ context.Context, _ contracts.TargetToken, draft contracts.Draft) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revisions = append(f.revisions, draft.Revision)
	f.texts = append(f.texts, draft.Text)
	return nil
}

func (f *fakeComposition) Cancel(context.Context, contracts.Identity, contracts.TargetToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancels++
	return nil
}

// IME-A02 / IME-A03: system_ime sessions mirror drafts into the owned
// composition with increasing revisions, commit through the IME adapter and
// clear the composition when cancelled.
func TestSystemIMEComposition(t *testing.T) {
	rec := &fakeRecognizer{words: []string{"voice", "draft"}}
	shortcutTarget := &fakeTarget{outcome: contracts.Applied}
	imeTarget := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "ime"}}
	composition := &fakeComposition{}
	audio := &fakeAudio{}
	c, err := NewCoordinator(Dependencies{Audio: audio, Recognizers: fakeFactory{rec}, NewID: newID,
		Targets: shortcutTarget, Input: shortcutTarget, IMETargets: imeTarget, IMEInput: imeTarget, Composition: composition},
		Options{Mode: domain.SystemIMEMode, AutoCommit: true})
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	events.wait(t, func(e contracts.Event) bool { return e.Kind == contracts.DraftReplaced && e.Text == "voice draft" })
	c.Stop()
	events.wait(t, kindIs(contracts.CommitResultReady))
	if imeTarget.commits.Load() != 1 || shortcutTarget.commits.Load() != 0 {
		t.Fatalf("commit routed wrong: ime=%d shortcut=%d", imeTarget.commits.Load(), shortcutTarget.commits.Load())
	}
	composition.mu.Lock()
	for i := 1; i < len(composition.revisions); i++ {
		if composition.revisions[i] <= composition.revisions[i-1] {
			t.Fatal("composition revisions not increasing")
		}
	}
	if len(composition.texts) == 0 || composition.texts[len(composition.texts)-1] != "voice draft" {
		t.Fatalf("drafts not mirrored: %v", composition.texts)
	}
	composition.mu.Unlock()

	// A cancelled IME session clears its owned range.
	c.Start()
	events.wait(t, func(e contracts.Event) bool {
		return e.Kind == contracts.SessionStateChanged && e.State == domain.Capturing && e.Generation > 1
	})
	c.Cancel()
	time.Sleep(20 * time.Millisecond)
	composition.mu.Lock()
	defer composition.mu.Unlock()
	if composition.cancels != 1 {
		t.Fatalf("composition cancels = %d", composition.cancels)
	}
}

// DSK-09-style explicit outcome: system_ime without an IME adapter fails.
func TestSystemIMEWithoutAdapterFails(t *testing.T) {
	target := &fakeTarget{outcome: contracts.Applied}
	c, audio, events := setup(t, &fakeRecognizer{words: []string{"x"}}, target, Options{Mode: domain.SystemIMEMode, AutoCommit: true})
	c.Start()
	failed := events.wait(t, kindIs(contracts.ErrorOccurred))
	if failed.Error != domain.CodeTargetUnsupported || audio.starts.Load() != 0 {
		t.Fatalf("got %s starts=%d", failed.Error, audio.starts.Load())
	}
}
