package application

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// segmentRecognizer closes a segment at every "|" in its script: each
// accepted frame reveals one more script entry.
type segmentRecognizer struct {
	script []string
	n      int
	closed atomic.Bool
}

func (r *segmentRecognizer) Accept(context.Context, contracts.Frame) error { r.n++; return nil }
func (r *segmentRecognizer) draft() contracts.Draft {
	var segments []string
	current := ""
	for _, word := range r.script[:min(r.n, len(r.script))] {
		if word == "|" {
			segments = append(segments, current)
			current = ""
			continue
		}
		current += word
	}
	return contracts.Draft{Text: domain.JoinText(append(append([]string(nil), segments...), current)...), Segments: segments, Tail: current}
}
func (r *segmentRecognizer) Partial(context.Context) (contracts.Draft, error) { return r.draft(), nil }
func (r *segmentRecognizer) Finish(context.Context) (contracts.Draft, error) {
	d := r.draft()
	d.Final = true
	return d, nil
}
func (r *segmentRecognizer) Close() error { r.closed.Store(true); return nil }

type segmentFactory struct{ r *segmentRecognizer }

func (f segmentFactory) NewRecognizer(context.Context, contracts.Identity) (ports.Recognizer, error) {
	return f.r, nil
}

// upperFinalizer marks refined segments so tests can tell them apart.
type upperFinalizer struct {
	mu    sync.Mutex
	calls []int
}

func (f *upperFinalizer) Finalize(_ context.Context, request ports.FinalizeRequest) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, len(request.Samples))
	f.mu.Unlock()
	return strings.ToUpper(request.Draft), nil
}

// scriptedRewriter maps sources to proposals and records every request.
type scriptedRewriter struct {
	mu     sync.Mutex
	answer func(ports.RewriteRequest) (string, error)
	block  chan struct{}
	// liveBlock holds the first live request until released or cancelled.
	liveBlock chan struct{}
	liveSeen  bool
	requests  []ports.RewriteRequest
	ended     []contracts.Identity
}

func (w *scriptedRewriter) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
	w.mu.Lock()
	w.requests = append(w.requests, req)
	block := w.block
	holdLive := req.Live && w.liveBlock != nil && !w.liveSeen
	if req.Live {
		w.liveSeen = true
	}
	w.mu.Unlock()
	if holdLive {
		select {
		case <-w.liveBlock:
			return ports.RewriteResult{Text: "STALE"}, nil
		case <-ctx.Done():
			return ports.RewriteResult{}, domain.Fail(domain.CodeCancelled, "cancelled")
		}
	}
	if block != nil && !req.Live {
		select {
		case <-block:
		case <-ctx.Done():
			return ports.RewriteResult{}, domain.Fail(domain.CodeCancelled, "cancelled")
		}
	}
	text, err := w.answer(req)
	return ports.RewriteResult{Text: text}, err
}
func (w *scriptedRewriter) End(id contracts.Identity) {
	w.mu.Lock()
	w.ended = append(w.ended, id)
	w.mu.Unlock()
}

func liveSetup(t *testing.T, script []string, rewriter *scriptedRewriter, policy RewritePolicy) (*Coordinator, *fakeAudio, *recorder, *fakeTarget, *upperFinalizer) {
	t.Helper()
	audio := &fakeAudio{}
	target := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}}
	finalizer := &upperFinalizer{}
	deps := Dependencies{Audio: audio, Recognizers: segmentFactory{&segmentRecognizer{script: script}}, Finalizer: finalizer,
		Targets: target, Input: target, NewID: newID}
	if rewriter != nil {
		deps.Rewriter = rewriter
	}
	c, err := NewCoordinator(deps, Options{AutoCommit: true, SegmentFinalize: true, Rewrite: func() RewritePolicy { return policy }})
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	return c, audio, events, target, finalizer
}

func draftIs(text string) func(contracts.Event) bool {
	return func(e contracts.Event) bool { return e.Kind == contracts.DraftReplaced && e.Text == text }
}

// Closed segments are refined and previewed while capture continues; the
// final rewrite of the whole text is the only inserted proposal.
func TestLivePreviewThenFinalRewrite(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		return strings.ReplaceAll(req.Text, "UM ", ""), nil
	}}
	c, audio, events, target, finalizer := liveSetup(t, []string{"um hello", "|", "b", "|", "c"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat"})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	// The first segment is refined (upper case) and previewed (filler gone)
	// while the user is still speaking.
	events.wait(t, draftIs("HELLO"))
	audio.push(3, 0.2)
	events.wait(t, draftIs("HELLO B c"))
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Raw != "UM HELLO B C" || ready.Source != "UM HELLO B C" || ready.Text != "HELLO B C" || ready.Rewrite != string(domain.RewriteAccepted) {
		t.Fatalf("unexpected final %+v", ready)
	}
	events.wait(t, kindIs(contracts.CommitResultReady))
	if target.texts[0] != "HELLO B C" {
		t.Fatalf("committed %q", target.texts[0])
	}
	finalizer.mu.Lock()
	refined := len(finalizer.calls)
	finalizer.mu.Unlock()
	if refined != 3 {
		t.Fatalf("expected three per-segment refinements, got %d", refined)
	}
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	live, final := 0, 0
	for _, r := range rewriter.requests {
		if r.Live {
			live++
		} else {
			final++
		}
	}
	if live < 1 || final != 1 || len(rewriter.ended) != 1 {
		t.Fatalf("live=%d final=%d ended=%d", live, final, len(rewriter.ended))
	}
}

// A proposal that fails validation is never inserted; the source is, with
// its verdict visible.
func TestRejectedRewriteKeepsSource(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		return "我们明天在北京总部开会讨论新的预算方案", nil
	}}
	c, audio, events, target, _ := liveSetup(t, []string{"我们明天开会"}, rewriter, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Text != "我们明天开会" || ready.Rewrite != string(domain.RewriteLength) {
		t.Fatalf("unexpected final %+v", ready)
	}
	events.wait(t, kindIs(contracts.CommitResultReady))
	if target.texts[0] != "我们明天开会" {
		t.Fatalf("committed %q", target.texts[0])
	}
}

func TestRewriteFailureAndTimeoutAreExplicit(t *testing.T) {
	failing := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
		return "", domain.Fail(domain.CodeModelUnsupported, "worker exited")
	}}
	c, audio, events, _, _ := liveSetup(t, []string{"hello"}, failing, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Rewrite != RewriteUnavailable || ready.Text != "HELLO" {
		t.Fatalf("unexpected %+v", ready)
	}

	slow := &scriptedRewriter{block: make(chan struct{}), answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, _, _ = liveSetup(t, []string{"hello"}, slow, RewritePolicy{Enabled: true, Timeout: 30 * time.Millisecond})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Rewrite != RewriteTimeout || ready.Text != "HELLO" {
		t.Fatalf("unexpected %+v", ready)
	}
}

// RWT-A02: cancelling during the final rewrite discards the proposal; no
// review, no commit, and the worker session state is released.
func TestCancelDuringFinalRewrite(t *testing.T) {
	rewriter := &scriptedRewriter{block: make(chan struct{}), answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"hello"}, rewriter, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	deadline := time.Now().Add(2 * time.Second)
	for {
		rewriter.mu.Lock()
		n := len(rewriter.requests)
		rewriter.mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	c.Cancel()
	time.Sleep(50 * time.Millisecond)
	if target.commits.Load() != 0 || events.count(kindIs(contracts.CorrectionReady)) != 0 {
		t.Fatal("cancelled rewrite produced review or commit")
	}
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	if len(rewriter.ended) != 1 {
		t.Fatalf("session state released %d times", len(rewriter.ended))
	}
}

// A preview computed for an older stable text is never displayed after a
// newer segment closes (RWT-A02 for previews).
func TestStalePreviewNotDisplayed(t *testing.T) {
	gate := make(chan struct{})
	rewriter := &scriptedRewriter{liveBlock: gate, answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, _, _ := liveSetup(t, []string{"a", "|", "b", "|", "c"}, rewriter, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	events.wait(t, draftIs("A"))
	audio.push(3, 0.2)
	events.wait(t, func(e contracts.Event) bool {
		return e.Kind == contracts.DraftReplaced && strings.HasPrefix(e.Text, "A B")
	})
	close(gate)
	c.Stop()
	events.wait(t, kindIs(contracts.CorrectionReady))
	if events.count(func(e contracts.Event) bool { return strings.Contains(e.Text, "STALE") }) != 0 {
		t.Fatal("stale preview displayed")
	}
}

// Without a rewriter, enabling rewriting degrades to the plain pipeline.
func TestRewriteWithoutRewriterIsDisabled(t *testing.T) {
	c, audio, events, _, _ := liveSetup(t, []string{"hello"}, nil, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Rewrite != "" || ready.Text != "HELLO" {
		t.Fatalf("unexpected %+v", ready)
	}
}

func TestQuietCutFindsPause(t *testing.T) {
	rate := 16000
	var samples []float32
	add := func(seconds float64, amplitude float32) {
		for i := 0; i < int(seconds*float64(rate)); i++ {
			samples = append(samples, amplitude)
		}
	}
	add(1, 0.3)   // first phrase
	add(1.2, 0)   // pause
	add(0.4, 0.3) // next phrase already captured at endpoint report
	cut := quietCut(samples, 0, len(samples), rate)
	if want := int(1.6 * float64(rate)); cut < want-rate/50 || cut > want+rate/50 {
		t.Fatalf("cut at %d, want about %d", cut, want)
	}
	if got := quietCut(samples[:rate], 0, rate, rate); got != rate {
		t.Fatalf("no pause should keep the report position, got %d", got)
	}
}

// Without per-segment finalization, previews show streaming text and the
// finalizer decodes the whole utterance once after stop.
func TestWholeUtteranceFinalizationWithPreviews(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		if req.Live {
			return strings.ToUpper(req.Text), nil
		}
		return req.Text, nil
	}}
	audio := &fakeAudio{}
	target := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}}
	finalizer := &upperFinalizer{}
	c, err := NewCoordinator(Dependencies{Audio: audio, Recognizers: segmentFactory{&segmentRecognizer{script: []string{"a", "|", "b"}}},
		Finalizer: finalizer, Targets: target, Input: target, Rewriter: rewriter, NewID: newID},
		Options{AutoCommit: true, Rewrite: func() RewritePolicy { return RewritePolicy{Enabled: true} }})
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(3, 0.2)
	events.wait(t, func(e contracts.Event) bool {
		return e.Kind == contracts.DraftReplaced && e.Rewrite == RewritePreview && e.Text == "A b"
	})
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Raw != "A B" {
		t.Fatalf("whole-utterance finalization not used: %+v", ready)
	}
	finalizer.mu.Lock()
	defer finalizer.mu.Unlock()
	if len(finalizer.calls) != 1 {
		t.Fatalf("expected one whole-utterance decode, got %d", len(finalizer.calls))
	}
}

// SCN-A01: with FollowApp the captured target decides the scene of every
// preview and the final rewrite, and the review reports it.
func TestFollowAppDecidesScene(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"hello", "|", "b"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", FollowApp: true})
	target.snapshot.Application, target.snapshot.Site = "com.google.Chrome", "mail.google.com"
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(3, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Scene != string(domain.SceneEmail) {
		t.Fatalf("review scene %q", ready.Scene)
	}
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	if len(rewriter.requests) == 0 {
		t.Fatal("no rewrite requests")
	}
	for _, req := range rewriter.requests {
		if req.Scene != string(domain.SceneEmail) {
			t.Fatalf("request scene %q (live=%v)", req.Scene, req.Live)
		}
	}

	// Without FollowApp the declared scene stays, whatever the target.
	rewriter = &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, target, _ = liveSetup(t, []string{"hello"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat"})
	target.snapshot.Application = "com.apple.mail"
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Scene != string(domain.SceneChat) {
		t.Fatalf("fixed scene became %q", ready.Scene)
	}
}

// A search field asks the model for plain text and delivers a query
// without sentence-final punctuation, whether or not the rewrite passed.
func TestSearchSceneFinishesQuery(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text + "。", nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"weather"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", FollowApp: true})
	target.snapshot.Field = domain.FieldSearch
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Scene != string(domain.SceneSearch) || ready.Text != "WEATHER" {
		t.Fatalf("unexpected %+v", ready)
	}
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	if last := rewriter.requests[len(rewriter.requests)-1]; last.Scene != string(domain.SceneChat) {
		t.Fatalf("model asked for %q", last.Scene)
	}
}

// A scene the user chose for the place beats the built-in rules.
func TestSceneOverrideWins(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	overrides := domain.SceneOverrides{"com.apple.mail": domain.SceneWork}
	c, audio, events, target, _ := liveSetup(t, []string{"hello"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", FollowApp: true, SceneOverrides: overrides})
	target.snapshot.Application = "com.apple.mail"
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Scene != string(domain.SceneWork) {
		t.Fatalf("override ignored: %q", ready.Scene)
	}
}
