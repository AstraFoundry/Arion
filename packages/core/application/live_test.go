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

func liveSetup(t *testing.T, script []string, rewriter *scriptedRewriter, policy RewritePolicy, history ...*History) (*Coordinator, *fakeAudio, *recorder, *fakeTarget, *upperFinalizer) {
	t.Helper()
	audio := &fakeAudio{}
	target := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}}
	finalizer := &upperFinalizer{}
	deps := Dependencies{Audio: audio, Recognizers: segmentFactory{&segmentRecognizer{script: script}}, Finalizer: finalizer,
		Targets: target, Input: target, NewID: newID}
	if len(history) > 0 {
		deps.History = history[0]
	}
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

// Closed segments stay recognized while capture continues. The rewriter runs
// once after release, on the full recognized text, and that result is the
// only inserted proposal.
func TestNoLiveRewriteUntilRelease(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		return strings.ReplaceAll(req.Text, "UM ", ""), nil
	}}
	c, audio, events, target, finalizer := liveSetup(t, []string{"um hello", "|", "b", "|", "c"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat"})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	// The first segment is refined (upper case) and still contains the filler.
	events.wait(t, draftIs("UM HELLO"))
	audio.push(3, 0.2)
	events.wait(t, draftIs("UM HELLO B c"))
	rewriter.mu.Lock()
	during := len(rewriter.requests)
	rewriter.mu.Unlock()
	if during != 0 {
		t.Fatalf("%d rewrite requests during speech", during)
	}
	if events.count(func(e contracts.Event) bool { return e.Rewrite == RewritePreview }) != 0 {
		t.Fatal("polished preview displayed")
	}
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
	if len(rewriter.requests) != 1 || rewriter.requests[0].Live || rewriter.requests[0].Text != "UM HELLO B C" || len(rewriter.ended) != 1 {
		t.Fatalf("requests %+v ended=%d", rewriter.requests, len(rewriter.ended))
	}
}

// Final-only polishing sends no rewrite while speaking; the text after stop
// is still polished once (ADR 022, ADR 023).
func TestFinalOnlyPolishesOnceAfterStop(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		return strings.ReplaceAll(req.Text, "UM ", ""), nil
	}}
	c, audio, events, target, _ := liveSetup(t, []string{"um hello", "|", "b"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", FinalOnly: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	events.wait(t, draftIs("UM HELLO"))
	audio.push(1, 0.2)
	events.wait(t, draftIs("UM HELLO b"))
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Text != "HELLO B" || ready.Rewrite != string(domain.RewriteAccepted) {
		t.Fatalf("unexpected final %+v", ready)
	}
	events.wait(t, kindIs(contracts.CommitResultReady))
	if target.texts[0] != "HELLO B" {
		t.Fatalf("committed %q", target.texts[0])
	}
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	if len(rewriter.requests) != 1 || rewriter.requests[0].Live {
		t.Fatalf("requests %+v", rewriter.requests)
	}
}

// The final rewrite announces itself before its result, so the overlay can
// show organization under way; without a rewriter nothing is announced.
func TestFinalRewriteAnnouncesItsStart(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, _, _ := liveSetup(t, []string{"hello"}, rewriter, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	started := events.wait(t, kindIs(contracts.RewriteStarted))
	if started.Sequence >= ready.Sequence || started.State != domain.Finalizing || started.Text != "" {
		t.Fatalf("started %+v before ready %d", started, ready.Sequence)
	}

	c, audio, events, _, _ = liveSetup(t, []string{"hello"}, nil, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	events.wait(t, kindIs(contracts.CorrectionReady))
	if n := events.count(kindIs(contracts.RewriteStarted)); n != 0 {
		t.Fatalf("%d rewrite starts without a rewriter", n)
	}
}

// A proposal that fails validation is never inserted; the source is, with
// its verdict visible.
func TestRejectedRewriteKeepsSource(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		return "<think>unfinished model reasoning</think>", nil
	}}
	c, audio, events, target, _ := liveSetup(t, []string{"我们明天开会"}, rewriter, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Text != "我们明天开会" || ready.Rewrite != string(domain.RewriteMarkup) {
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

// Recognized text stays on screen while capture continues. A model result
// is not shown until release, so nothing computed during speech can replace it.
func TestNoPolishedPreviewDuringSpeech(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return "STALE " + req.Text, nil }}
	c, audio, events, _, _ := liveSetup(t, []string{"a", "|", "b", "|", "c"}, rewriter, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	events.wait(t, draftIs("A"))
	audio.push(3, 0.2)
	events.wait(t, func(e contracts.Event) bool {
		return e.Kind == contracts.DraftReplaced && e.Text == "A B c" && e.Rewrite == ""
	})
	rewriter.mu.Lock()
	during := len(rewriter.requests)
	rewriter.mu.Unlock()
	if during != 0 {
		t.Fatalf("%d rewrite requests during speech", during)
	}
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Text != "STALE A B C" || events.count(func(e contracts.Event) bool {
		return e.Kind == contracts.DraftReplaced && (e.Rewrite == RewritePreview || strings.Contains(e.Text, "STALE"))
	}) != 0 {
		t.Fatal("polished preview displayed during speech")
	}
}

// LRW-A03: a missing adapter preserves the source and records why it did so.
func TestRewriteWithoutRewriterIsUnavailable(t *testing.T) {
	c, audio, events, target, _ := liveSetup(t, []string{"hello", "|", "there"}, nil, RewritePolicy{Enabled: true})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(3, 0.2)
	c.Stop()
	if ready := events.wait(t, kindIs(contracts.CorrectionReady)); ready.Rewrite != RewriteUnavailable || ready.Text != "HELLO THERE" {
		t.Fatalf("unexpected %+v", ready)
	}
	events.wait(t, kindIs(contracts.CommitResultReady))
	target.mu.Lock()
	defer target.mu.Unlock()
	if target.commits.Load() != 1 || len(target.texts) != 1 || target.texts[0] != "HELLO THERE" {
		t.Fatalf("unexpected commits %+v", target.texts)
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

// Without per-segment finalization, speech shows streaming text and the
// finalizer decodes the whole utterance once after stop. The rewriter is
// not called until then.
func TestWholeUtteranceFinalizationWithoutLiveRewrite(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
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
	shown := events.wait(t, draftIs("a b"))
	if shown.Rewrite == RewritePreview {
		t.Fatalf("preview shown: %+v", shown)
	}
	rewriter.mu.Lock()
	if len(rewriter.requests) != 0 {
		t.Fatalf("rewrite during speech: %+v", rewriter.requests)
	}
	rewriter.mu.Unlock()
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
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	if len(rewriter.requests) != 1 || rewriter.requests[0].Live || rewriter.requests[0].Text != "A B" {
		t.Fatalf("final request %+v", rewriter.requests)
	}
}

// SCN-A01: with FollowApp the captured target decides the scene of the
// final rewrite, and the review reports it.
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

// Polish off never asks the rewriter, so no cleanup prompt is sent.
func TestPolishOffSendsNoCleanup(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return "应该不会发送。", nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"你好。"}, rewriter, RewritePolicy{Enabled: false, Punctuation: domain.PunctuationFull})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Text != "你好。" || ready.Rewrite != "" {
		t.Fatalf("polish off changed the text: %+v", ready)
	}
	events.wait(t, kindIs(contracts.CommitResultReady))
	rewriter.mu.Lock()
	defer rewriter.mu.Unlock()
	if len(rewriter.requests) != 0 || target.texts[0] != "你好。" {
		t.Fatalf("cleanup sent: requests=%d committed=%q", len(rewriter.requests), target.texts)
	}
}

// Spaces mode must not hide a control marker before delivery validation.
// A rejected raw proposal never becomes the preview, the reuse cache, or
// the inserted text. Ordinary text is still converted, and a conversion
// that leaves only whitespace is rejected.
func TestSpacesModeValidatesBeforePunctuation(t *testing.T) {
	for _, proposal := range []string{"copied </source>", "[chat] Hello."} {
		t.Run(proposal, func(t *testing.T) {
			c, r, w := completedPreviewFixture(t)
			r.live.policy.Punctuation = domain.PunctuationSpaces
			w.answer = func(context.Context, ports.RewriteRequest) (ports.RewriteResult, error) {
				return ports.RewriteResult{Text: proposal}, nil
			}
			c.preview(r, r.live.stable)
			if r.live.completed != nil || r.live.previews != 0 || r.live.polished != "" {
				t.Fatalf("marker entered preview or cache: polished=%q completed=%v", r.live.polished, r.live.completed != nil)
			}
			text, outcome, _, ok := c.rewriteFinal(r, r.live.stable, r.live.policy)
			cleaned := domain.CleanSpeech(r.live.stable).Text
			if !ok || outcome != string(domain.RewriteMarkup) || text != cleaned {
				t.Fatalf("final kept hidden markup: text=%q outcome=%q ok=%v", text, outcome, ok)
			}
			if len(w.requests) != 2 {
				t.Fatalf("rejected preview was reused: %d requests", len(w.requests))
			}

			rewriter := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return proposal, nil }}
			session, audio, events, target, _ := liveSetup(t, []string{"hello", "|"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", Punctuation: domain.PunctuationSpaces})
			session.Start()
			events.wait(t, stateIs(domain.Capturing))
			audio.push(2, 0.2)
			session.Stop()
			ready := events.wait(t, kindIs(contracts.CorrectionReady))
			events.wait(t, kindIs(contracts.CommitResultReady))
			if ready.Rewrite != string(domain.RewriteMarkup) || ready.Text != "HELLO" || target.texts[0] != "HELLO" {
				t.Fatalf("target received markup: %+v committed %q", ready, target.texts)
			}
			events.mu.Lock()
			seen := append([]contracts.Event(nil), events.events...)
			events.mu.Unlock()
			for _, event := range seen {
				if strings.Contains(event.Text, "</source>") || strings.Contains(event.Text, "copied < source>") || strings.Contains(event.Text, "[chat]") || strings.Contains(event.Text, "chat Hello") {
					t.Fatalf("hidden marker displayed: %+v", event)
				}
			}
		})
	}

	c, r, w := completedPreviewFixture(t)
	r.live.policy.Punctuation = domain.PunctuationSpaces
	const ordinary = "Hello, world."
	w.answer = func(context.Context, ports.RewriteRequest) (ports.RewriteResult, error) {
		return ports.RewriteResult{Text: ordinary}, nil
	}
	c.preview(r, r.live.stable)
	want := domain.ApplyPunctuation(domain.PunctuationSpaces, ordinary)
	if r.live.completed == nil || r.live.completed.result.Text != ordinary || r.live.polished != want {
		t.Fatalf("ordinary spaces conversion lost: polished=%q cached=%v", r.live.polished, r.live.completed)
	}
	text, outcome, _, ok := c.rewriteFinal(r, r.live.stable, r.live.policy)
	if !ok || outcome != string(domain.RewriteAccepted) || text != want || len(w.requests) != 1 {
		t.Fatalf("ordinary final: text=%q outcome=%q calls=%d", text, outcome, len(w.requests))
	}

	for _, tc := range []struct {
		mode     domain.Punctuation
		proposal string
	}{
		{domain.PunctuationSpaces, "."},
		{domain.PunctuationOmit, "。"},
	} {
		t.Run(string(tc.mode)+" empty", func(t *testing.T) {
			c, r, w := completedPreviewFixture(t)
			r.live.policy.Punctuation = tc.mode
			w.answer = func(context.Context, ports.RewriteRequest) (ports.RewriteResult, error) {
				return ports.RewriteResult{Text: tc.proposal}, nil
			}
			c.preview(r, r.live.stable)
			if r.live.completed != nil || r.live.polished != "" {
				t.Fatal("empty conversion was previewed")
			}
			text, outcome, _, ok := c.rewriteFinal(r, r.live.stable, r.live.policy)
			if !ok || outcome != string(domain.RewriteEmpty) || text != domain.CleanSpeech(r.live.stable).Text || len(w.requests) != 2 {
				t.Fatalf("empty conversion delivered: text=%q outcome=%q calls=%d ok=%v", text, outcome, len(w.requests), ok)
			}
		})
	}
}

// Each punctuation mode post-edits the inserted text. Speech shows the
// recognized source. Search still strips a sentence-final mark afterward.
func TestPunctuationModes(t *testing.T) {
	const proposal = "你好，世界。"
	for _, mode := range []domain.Punctuation{domain.PunctuationSmart, domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces} {
		t.Run(string(mode), func(t *testing.T) {
			rewriter := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return proposal, nil }}
			c, audio, events, target, _ := liveSetup(t, []string{"你好。", "|", "尾"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", Punctuation: mode})
			c.Start()
			events.wait(t, stateIs(domain.Capturing))
			audio.push(2, 0.2)
			want := domain.ApplyPunctuation(mode, proposal)
			recognized := events.wait(t, draftIs("你好。"))
			if recognized.Rewrite == RewritePreview {
				t.Fatalf("preview during speech: %+v", recognized)
			}
			c.Stop()
			ready := events.wait(t, kindIs(contracts.CorrectionReady))
			events.wait(t, kindIs(contracts.CommitResultReady))
			if ready.Text != want || target.texts[0] != want {
				t.Fatalf("final %q committed %q want %q", ready.Text, target.texts, want)
			}
			rewriter.mu.Lock()
			defer rewriter.mu.Unlock()
			if len(rewriter.requests) != 1 || rewriter.requests[0].Live || rewriter.requests[0].Text != "你好。" {
				t.Fatalf("requests %+v", rewriter.requests)
			}
			if rewriter.requests[0].Punctuation != string(mode) {
				t.Fatalf("request punctuation %q", rewriter.requests[0].Punctuation)
			}
		})
	}

	rewriter := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return "天气，", nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"天气"}, rewriter, RewritePolicy{Enabled: true, Scene: "chat", FollowApp: true, Punctuation: domain.PunctuationOmit})
	target.snapshot.Field = domain.FieldSearch
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Scene != string(domain.SceneSearch) || ready.Text != "天气，" || target.texts[0] != "天气，" {
		t.Fatalf("search removed a comma: %+v %q", ready, target.texts)
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

// CTX-A01: the session's context reaches the final rewrite and its
// validation; an inserted dictation becomes history for the same place
// only; denied apps are never read.
func TestRewriteContextAndHistory(t *testing.T) {
	history := NewHistory(time.Hour)
	policy := RewritePolicy{Enabled: true, Scene: "chat", Context: ContextPolicy{Cursor: true, Screen: true, History: true}}
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"first"}, rewriter, policy, history)
	target.snapshot.Application = "com.tencent.xinWeChat"
	target.context = contracts.ContextSnapshot{SafeToRead: true, Before: "Hi ", After: " bye", Screen: "Zhang Wei: lunch?"}
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	events.wait(t, kindIs(contracts.CommitResultReady))
	rewriter.mu.Lock()
	final := rewriter.requests[len(rewriter.requests)-1]
	rewriter.mu.Unlock()
	if len(final.Terms) != 0 || final.Context.Screen != "Zhang Wei: lunch?" || final.Context.Before != "Hi" || final.Context.After != "bye" {
		t.Fatalf("first session: terms %q, context %+v", final.Terms, final.Context)
	}
	if got := history.Recent("com.tencent.xinWeChat"); len(got) != 1 || got[0] != "FIRST" {
		t.Fatalf("history after insertion %q", got)
	}

	// The next dictation in the same app sees it; another app does not.
	rewriter = &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, target, _ = liveSetup(t, []string{"second"}, rewriter, policy, history)
	target.snapshot.Application = "com.tencent.xinWeChat"
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	events.wait(t, kindIs(contracts.CommitResultReady))
	if context := rewriter.requests[len(rewriter.requests)-1].Context; strings.Join(context.History, ";") != "FIRST" {
		t.Fatalf("same place history %q", context.History)
	}
	if got := history.Recent("com.apple.mail"); len(got) != 0 {
		t.Fatalf("history leaked to another place: %q", got)
	}

	// A password manager is never read, even with every source allowed.
	rewriter = &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return req.Text, nil }}
	c, audio, events, target, _ = liveSetup(t, []string{"third"}, rewriter, policy, history)
	target.snapshot.Application = "com.1password.1password"
	target.context = contracts.ContextSnapshot{SafeToRead: true, Screen: "vault"}
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	events.wait(t, kindIs(contracts.CommitResultReady))
	if target.reads.Load() != 0 || len(rewriter.requests[len(rewriter.requests)-1].Terms) != 0 || !rewriter.requests[len(rewriter.requests)-1].Context.Empty() {
		t.Fatal("denied app was read")
	}
}

func TestHistoryExpiresAndClears(t *testing.T) {
	h := NewHistory(time.Minute)
	now := time.Unix(0, 0)
	h.now = func() time.Time { return now }
	for _, text := range []string{"a", "b", "c", "d", "e", "f"} {
		h.Add("app", text)
	}
	if got := h.Recent("app"); strings.Join(got, "") != "bcdef" {
		t.Fatalf("keeps the newest items: %q", got)
	}
	now = now.Add(2 * time.Minute)
	if got := h.Recent("app"); len(got) != 0 {
		t.Fatalf("expired items returned: %q", got)
	}
	h.Add("app", "x")
	h.Clear()
	if got := h.Recent("app"); len(got) != 0 {
		t.Fatal("Clear kept items")
	}
}

// Local cleanup completes independently of organization. Rejected model
// content never replaces it, and the failed organization remains explicit.
func TestCleanupSurvivesOrganizationFailure(t *testing.T) {
	for _, outcome := range []string{"rejected", "unavailable", "missing", "timeout", "disabled"} {
		t.Run(outcome, func(t *testing.T) {
			var rewriter *scriptedRewriter
			policy := RewritePolicy{Enabled: true}
			switch outcome {
			case "rejected":
				rewriter = &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
					return "", nil
				}}
			case "unavailable":
				rewriter = &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
					return "", domain.Fail(domain.CodeModelUnsupported, "worker exited")
				}}
			case "timeout":
				rewriter = &scriptedRewriter{block: make(chan struct{}), answer: func(ports.RewriteRequest) (string, error) { return "", nil }}
				policy.Timeout = 30 * time.Millisecond
			case "disabled":
				policy.Enabled = false
			}
			source := "那个，我想确认一下，今天是否发布？"
			c, audio, events, target, _ := liveSetup(t, []string{source}, rewriter, policy)
			c.Start()
			events.wait(t, stateIs(domain.Capturing))
			audio.push(1, 0.2)
			c.Stop()
			ready := events.wait(t, kindIs(contracts.CorrectionReady))
			want := "我想确认一下，今天是否发布？"
			if outcome == "disabled" {
				want = source
			}
			if ready.Raw != source || ready.Source != source || ready.Text != want {
				t.Fatalf("lost stage identity: %+v", ready)
			}
			switch outcome {
			case "missing", "unavailable":
				if ready.Rewrite != RewriteUnavailable {
					t.Fatal(ready.Rewrite)
				}
			case "timeout":
				if ready.Rewrite != RewriteTimeout {
					t.Fatal(ready.Rewrite)
				}
			case "rejected":
				if ready.Rewrite == string(domain.RewriteAccepted) || ready.Rewrite == "" {
					t.Fatal(ready.Rewrite)
				}
			case "disabled":
				if ready.Rewrite != "" {
					t.Fatal(ready.Rewrite)
				}
			}
			events.wait(t, kindIs(contracts.CommitResultReady))
			target.mu.Lock()
			defer target.mu.Unlock()
			if target.commits.Load() != 1 || len(target.texts) != 1 || target.texts[0] != want {
				t.Fatalf("unexpected commits %+v", target.texts)
			}
		})
	}
}

// Full OCR, including discussion beyond the old character cutoff, must reach
// the final rewrite. Each source remains independently gated. No rewrite
// runs while capture continues.
func TestFullContextReachesFinalRewrite(t *testing.T) {
	screen := strings.Repeat("界面内容。", 300) + "同事：软件订阅可以报销。"
	for _, tc := range []struct {
		name   string
		policy ContextPolicy
		safe   bool
	}{
		{"all", ContextPolicy{Cursor: true, Screen: true}, true},
		{"screen only", ContextPolicy{Screen: true}, true},
		{"cursor only", ContextPolicy{Cursor: true}, true},
		{"disabled", ContextPolicy{}, true},
		{"unsafe", ContextPolicy{Cursor: true, Screen: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) { return "Formatted " + req.Text, nil }}
			c, audio, events, target, _ := liveSetup(t, []string{"hello", "|", "again"}, w, RewritePolicy{Enabled: true, Context: tc.policy})
			target.context = contracts.ContextSnapshot{SafeToRead: tc.safe, Screen: screen, Before: "before", After: "after"}
			c.Start()
			events.wait(t, stateIs(domain.Capturing))
			c.mu.Lock()
			r := c.current
			c.mu.Unlock()
			if r.contextReady != nil {
				select {
				case <-r.contextReady:
				case <-time.After(time.Second):
					t.Fatal("context not ready")
				}
			}
			audio.push(2, 0.2)
			shown := events.wait(t, draftIs("HELLO"))
			if shown.Rewrite == RewritePreview || strings.Contains(shown.Text, "Formatted") {
				t.Fatalf("polished during speech: %+v", shown)
			}
			audio.push(1, 0.2)
			c.Stop()
			events.wait(t, kindIs(contracts.CommitResultReady))
			w.mu.Lock()
			defer w.mu.Unlock()
			live, final := false, false
			for _, req := range w.requests {
				if req.Live {
					live = true
				} else {
					final = true
				}
				wantScreen := ""
				if tc.safe && tc.policy.Screen {
					wantScreen = screen
				}
				if req.Context.Screen != wantScreen {
					t.Fatalf("screen lost or leaked: live=%v, runes=%d", req.Live, len([]rune(req.Context.Screen)))
				}
				wantBefore, wantAfter := "", ""
				if tc.safe && tc.policy.Cursor {
					wantBefore, wantAfter = "before", "after"
				}
				if req.Context.Before != wantBefore || req.Context.After != wantAfter {
					t.Fatalf("cursor gate: %+v", req.Context)
				}
				if len(req.Terms) != 0 {
					t.Fatalf("context was also extracted into terms: %q", req.Terms)
				}
			}
			if live || !final || len(w.requests) != 1 {
				t.Fatalf("rewrite timing: live=%v final=%v requests=%d", live, final, len(w.requests))
			}
		})
	}
}

func TestTranslationSessionForcesRewriteAndSkipsEditLearning(t *testing.T) {
	rewriter := &scriptedRewriter{answer: func(req ports.RewriteRequest) (string, error) {
		if req.TargetLanguage != string(domain.TargetEnglish) {
			t.Errorf("expected target language %q, got %q", domain.TargetEnglish, req.TargetLanguage)
		}
		return "Let's meet at 3 PM.", nil
	}}
	audio := &fakeAudio{}
	field := &fakeField{text: "Let's meet at 4 PM.", focused: true}
	ft := &fakeTarget{outcome: contracts.Applied, snapshot: contracts.TargetSnapshot{Target: "t1"}}
	target := fieldTarget{ft, field}
	var learned atomic.Int32
	tracker := &EditTracker{
		Poll: 5 * time.Millisecond,
		Idle: 15 * time.Millisecond,
		Max:  100 * time.Millisecond,
		Report: func(EditReport) {
			learned.Add(1)
		},
	}
	// Polish is disabled in the base policy, but StartTranslation must force rewriting.
	c, err := NewCoordinator(
		Dependencies{Audio: audio, Recognizers: segmentFactory{&segmentRecognizer{script: []string{"我们下午三点开会"}}}, Targets: target, Input: target, Rewriter: rewriter, Edits: tracker, NewID: newID},
		Options{AutoCommit: true, Learning: func() bool { return true }, Rewrite: func() RewritePolicy { return RewritePolicy{Enabled: false, Scene: "chat"} }},
	)
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	c.Subscribe(events.listen)
	c.StartTranslation(domain.TargetEnglish)
	events.wait(t, stateIs(domain.Capturing))
	audio.push(1, 0.2)
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	if ready.Source != "我们下午三点开会" || ready.Text != "Let's meet at 3 PM." || ready.Rewrite != string(domain.RewriteAccepted) {
		t.Fatalf("unexpected translation review: %+v", ready)
	}
	events.wait(t, kindIs(contracts.CommitResultReady))
	if ft.commits.Load() != 1 || ft.texts[0] != "Let's meet at 3 PM." {
		t.Fatalf("unexpected commits: %+v", ft.texts)
	}
	if field.followed.Load() != 0 || learned.Load() != 0 {
		t.Fatalf("translation session armed edit learning: followed=%d learned=%d", field.followed.Load(), learned.Load())
	}
}

func TestTranslationFailureAbortsCommitAndPreservesSource(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rewriter *scriptedRewriter
		timeout  time.Duration
		wantCode domain.ErrorCode
		wantOut  string
	}{
		{
			name:     "missing rewriter",
			rewriter: nil,
			wantCode: domain.CodeModelMissing,
			wantOut:  RewriteUnavailable,
		},
		{
			name: "model not running",
			rewriter: &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
				return "", domain.Fail(domain.CodeModelMissing, "rewrite model not running")
			}},
			wantCode: domain.CodeModelMissing,
			wantOut:  RewriteUnavailable,
		},
		{
			name: "worker error",
			rewriter: &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
				return "", domain.Fail(domain.CodeModelUnsupported, "worker exited")
			}},
			wantCode: domain.CodeModelUnsupported,
			wantOut:  RewriteUnavailable,
		},
		{
			name:     "timeout",
			rewriter: &scriptedRewriter{block: make(chan struct{}), answer: func(ports.RewriteRequest) (string, error) { return "Translated", nil }},
			timeout:  30 * time.Millisecond,
			wantCode: domain.CodeRewriteTimeout,
			wantOut:  RewriteTimeout,
		},
		{
			name: "rejected markup",
			rewriter: &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
				return "<think>unclosed reasoning", nil
			}},
			wantCode: domain.CodeRewriteRejected,
			wantOut:  string(domain.RewriteMarkup),
		},
		{
			name: "empty output",
			rewriter: &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
				return "   ", nil
			}},
			wantCode: domain.CodeRewriteRejected,
			wantOut:  string(domain.RewriteEmpty),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "今天下午三点开会。"
			policy := RewritePolicy{Enabled: true, TargetLanguage: domain.TargetEnglish, Timeout: tc.timeout}
			c, audio, events, target, _ := liveSetup(t, []string{source}, tc.rewriter, policy)
			c.StartTranslation(domain.TargetEnglish)
			events.wait(t, stateIs(domain.Capturing))
			audio.push(1, 0.2)
			c.Stop()
			failed := events.wait(t, func(e contracts.Event) bool {
				return e.Kind == contracts.ErrorOccurred && e.State == domain.Failed
			})
			if failed.Error != tc.wantCode || failed.Rewrite != tc.wantOut {
				t.Fatalf("got error=%q rewrite=%q, want error=%q rewrite=%q (%+v)", failed.Error, failed.Rewrite, tc.wantCode, tc.wantOut, failed)
			}
			if failed.Source != source || failed.Text != source {
				t.Fatalf("failed translation lost source text: source=%q text=%q", failed.Source, failed.Text)
			}
			if target.commits.Load() != 0 || events.count(kindIs(contracts.CorrectionReady)) != 0 || events.count(kindIs(contracts.CommitResultReady)) != 0 {
				t.Fatalf("failed translation committed or reviewed untranslated text: commits=%d", target.commits.Load())
			}
		})
	}
}
