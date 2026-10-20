package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type reuseRewriter struct {
	requests []ports.RewriteRequest
	answer   func(context.Context, ports.RewriteRequest) (ports.RewriteResult, error)
}

func (w *reuseRewriter) Rewrite(ctx context.Context, request ports.RewriteRequest) (ports.RewriteResult, error) {
	w.requests = append(w.requests, request)
	if w.answer != nil {
		return w.answer(ctx, request)
	}
	return ports.RewriteResult{Text: request.Text + "."}, nil
}
func (*reuseRewriter) End(contracts.Identity) {}

func completedPreviewFixture(t *testing.T) (*Coordinator, *run, *reuseRewriter) {
	t.Helper()
	session, err := domain.NewSession("preview-reuse", 1, domain.ShortcutMode)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []domain.SessionState{domain.Preparing, domain.Capturing} {
		if err := session.Advance(state); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	previewCtx, stop := context.WithCancel(ctx)
	t.Cleanup(stop)
	ready := make(chan struct{})
	close(ready)
	policy := RewritePolicy{Enabled: true, Scene: "chat", Timeout: time.Second, Terms: []string{"Chatterfly"}}
	r := &run{ctx: ctx, cancel: cancel, identity: contracts.Identity{SessionID: session.ID(), Generation: 1}, session: session,
		started: time.Now(), contextReady: ready, rewriteContext: domain.RewriteContext{Screen: "Chatterfly", Before: "before", After: "after", History: []string{"earlier"}},
		live: &liveState{policy: policy, stable: "Check Chatterfly", rewriteCtx: previewCtx, stopPreview: stop}}
	w := &reuseRewriter{}
	c := &Coordinator{deps: Dependencies{Rewriter: w}, current: r, generation: 1}
	return c, r, w
}

func TestCompletedPreviewReusedWithRawFinalEnding(t *testing.T) {
	c, r, w := completedPreviewFixture(t)
	c.preview(r, r.live.stable)
	if r.live.polished != "Check Chatterfly" {
		t.Fatalf("unexpected live ending: %q", r.live.polished)
	}
	// Stopping preview scheduling does not invalidate an already completed result.
	r.live.stopPreview()
	text, outcome, ok := c.rewriteFinal(r, r.live.stable, r.live.policy)
	if !ok || outcome != string(domain.RewriteAccepted) || text != "Check Chatterfly." {
		t.Fatalf("raw final proposal lost: %q %q %v", text, outcome, ok)
	}
	if len(w.requests) != 1 || !w.requests[0].Live {
		t.Fatalf("repeated inference: %+v", w.requests)
	}
}

func TestFinalRewriteRerunsWhenEffectiveInputsChange(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(*run)
		change  func(*run, *RewritePolicy, *string)
	}{
		{name: "whole-audio finalizer changes source", change: func(_ *run, _ *RewritePolicy, source *string) { *source = "Check chatter fly" }},
		{name: "screen changes", change: func(r *run, _ *RewritePolicy, _ *string) { r.rewriteContext.Screen = "Another project" }},
		{name: "cursor before changes", change: func(r *run, _ *RewritePolicy, _ *string) { r.rewriteContext.Before = "new before" }},
		{name: "cursor after changes", change: func(r *run, _ *RewritePolicy, _ *string) { r.rewriteContext.After = "new after" }},
		{name: "history changes in place", change: func(r *run, _ *RewritePolicy, _ *string) { r.rewriteContext.History[0] = "new history" }},
		{name: "approved terms change in place", change: func(_ *run, p *RewritePolicy, _ *string) { p.Terms[0] = "Another approved name" }},
		{name: "recalled vocabulary changes", change: func(_ *run, p *RewritePolicy, _ *string) {
			p.Vocabulary = domain.NewVocabulary([]domain.Replacement{{From: "Chapter fly", To: "Chatterfly"}})
		}},
		{name: "scene changes", change: func(_ *run, p *RewritePolicy, _ *string) { p.Scene = "email" }},
		{name: "OCR becomes ready after preview", prepare: func(r *run) { r.contextReady = make(chan struct{}) }, change: func(r *run, _ *RewritePolicy, _ *string) { close(r.contextReady) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, r, w := completedPreviewFixture(t)
			if tc.prepare != nil {
				tc.prepare(r)
			}
			c.preview(r, r.live.stable)
			if r.live.completed == nil {
				t.Fatal("preview did not complete")
			}
			source, policy := r.live.stable, r.live.policy
			tc.change(r, &policy, &source)
			text, outcome, ok := c.rewriteFinal(r, source, policy)
			if !ok || outcome != string(domain.RewriteAccepted) || text != source+"." {
				t.Fatalf("unexpected final: %q %q %v", text, outcome, ok)
			}
			if len(w.requests) != 2 || w.requests[1].Live || w.requests[1].Text != source {
				t.Fatalf("changed inputs did not rerun: %+v", w.requests)
			}
		})
	}
}

func TestIncompleteOrStalePreviewCannotBeReused(t *testing.T) {
	for _, name := range []string{"error", "rejected", "truncated", "cancelled request returns success", "cancelled session returns success", "superseded session", "superseded generation", "new closed segment", "source changes"} {
		t.Run(name, func(t *testing.T) {
			c, r, w := completedPreviewFixture(t)
			w.answer = func(_ context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
				result := ports.RewriteResult{Text: req.Text + "."}
				switch name {
				case "error":
					return result, errors.New("worker failed")
				case "rejected":
					result.Text = ""
				case "truncated":
					result.Truncated = true
				case "cancelled request returns success":
					r.live.cancelPreview()
				case "cancelled session returns success":
					c.Cancel()
				case "superseded session":
					c.current = &run{}
				case "superseded generation":
					c.generation++
				case "new closed segment":
					r.live.pending = []string{"more words"}
				case "source changes":
					r.live.stable = "new stable source"
				}
				return result, nil
			}
			c.preview(r, "Check Chatterfly")
			if r.live.completed != nil || r.live.previews != 0 {
				t.Fatalf("invalid preview accepted: %+v", r.live.completed)
			}
		})
	}
}

func TestFailedLatestPreviewDoesNotResurrectEarlierResult(t *testing.T) {
	c, r, w := completedPreviewFixture(t)
	c.preview(r, r.live.stable)
	w.answer = func(_ context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
		if req.Live {
			return ports.RewriteResult{}, errors.New("latest request failed")
		}
		return ports.RewriteResult{Text: "Fresh final."}, nil
	}
	c.preview(r, r.live.stable)
	text, _, ok := c.rewriteFinal(r, r.live.stable, r.live.policy)
	if !ok || text != "Fresh final." || len(w.requests) != 3 {
		t.Fatalf("earlier result resurrected: %q, calls=%d", text, len(w.requests))
	}
}

func TestCancelledSessionCannotDeliverCompletedPreview(t *testing.T) {
	c, r, _ := completedPreviewFixture(t)
	c.preview(r, r.live.stable)
	c.Cancel()
	if _, _, ok := c.rewriteFinal(r, r.live.stable, r.live.policy); ok {
		t.Fatal("cancelled session delivered cached result")
	}
}

func TestCompletedPreviewReuseCommitsOnce(t *testing.T) {
	w := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return "Hello.", nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"hello", "|"}, w, RewritePolicy{Enabled: true, Scene: "chat"})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	events.wait(t, draftIs("Hello"))
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	events.wait(t, kindIs(contracts.CommitResultReady))
	c.CommitReviewed()
	if ready.Source != "HELLO" || ready.Text != "Hello." || target.commits.Load() != 1 {
		t.Fatalf("unexpected delivery: %+v commits=%d", ready, target.commits.Load())
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.requests) != 1 || !w.requests[0].Live || len(w.ended) != 1 {
		t.Fatalf("unexpected worker lifecycle: %+v ended=%d", w.requests, len(w.ended))
	}
}

func TestFinalRewriteCarriesRecognizedPreviousSource(t *testing.T) {
	c, r, w := completedPreviewFixture(t)
	w.answer = func(_ context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
		if req.Live {
			return ports.RewriteResult{Text: "Generated alternative."}, nil
		}
		return ports.RewriteResult{Text: req.Text}, nil
	}
	c.preview(r, r.live.stable)
	source := "Check chatter fly"
	text, _, ok := c.rewriteFinal(r, source, r.live.policy)
	if !ok || text != source || len(w.requests) != 2 {
		t.Fatalf("unexpected final: %q, calls=%d", text, len(w.requests))
	}
	if w.requests[0].PreviousText != "" {
		t.Fatal("live request included previous-source evidence")
	}
	final := w.requests[1]
	if final.Text != source || final.PreviousText != "Check Chatterfly" || final.PreviousText == r.live.polished {
		t.Fatalf("recognized source confused with generated output: %+v", final)
	}
}

func TestEarlierSourceEvidenceStaysWithinCurrentGeneration(t *testing.T) {
	for _, name := range []string{"cancelled", "new session", "new generation", "no live state"} {
		t.Run(name, func(t *testing.T) {
			c, r, _ := completedPreviewFixture(t)
			switch name {
			case "cancelled":
				c.Cancel()
			case "new session":
				c.current = &run{}
			case "new generation":
				c.generation++
			case "no live state":
				r.live = nil
			}
			if source := c.previousStableSource(r, "Check chatter fly"); source != "" {
				t.Fatalf("stale source leaked: %q", source)
			}
		})
	}
}

func TestFinalRetainsApprovedMemoryFromRecognizedEarlierSource(t *testing.T) {
	c, r, w := completedPreviewFixture(t)
	r.rewriteContext = domain.RewriteContext{}
	r.live.policy.Terms = nil
	r.live.policy.Vocabulary = domain.NewVocabulary([]domain.Replacement{{From: "Chapter fly", To: "Chatterfly"}})
	w.answer = func(_ context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
		if req.Live {
			return ports.RewriteResult{Text: "Generated OtherBrand."}, nil
		}
		return ports.RewriteResult{Text: req.Text}, nil
	}
	c.preview(r, r.live.stable)
	source := "Check chad fly"
	if _, _, ok := c.rewriteFinal(r, source, r.live.policy); !ok {
		t.Fatal("final request cancelled")
	}
	final := w.requests[len(w.requests)-1]
	if final.PreviousText != "Check Chatterfly" || len(final.Terms) != 1 || final.Terms[0] != "Chapter fly → Chatterfly" || !final.Context.Empty() {
		t.Fatalf("approved memory was lost or model output promoted: %+v", final)
	}
}

func TestFinalVocabularyRecallDeduplicatesAndBoundsBothSources(t *testing.T) {
	replacements := []domain.Replacement{
		{From: "alpha", To: "AlphaBrand"}, {From: "beta", To: "BetaBrand"},
		{From: "gamma", To: "GammaBrand"}, {From: "delta", To: "DeltaBrand"},
		{From: "epsilon", To: "EpsilonBrand"}, {From: "zeta", To: "ZetaBrand"},
		{From: "eta", To: "EtaBrand"}, {From: "theta", To: "ThetaBrand"},
		{From: "iota", To: "IotaBrand"}, {From: "kappa", To: "KappaBrand"},
	}
	p := RewritePolicy{Terms: []string{"fixed approved term"}, Vocabulary: domain.NewVocabulary(replacements)}
	terms := p.termsForFinal("AlphaBrand", "AlphaBrand BetaBrand GammaBrand DeltaBrand EpsilonBrand ZetaBrand EtaBrand ThetaBrand IotaBrand KappaBrand")
	if len(terms) != domain.MaxRecallTerms+1 || terms[0] != p.Terms[0] || terms[1] != "alpha → AlphaBrand" {
		t.Fatalf("unexpected recall order or budget: %q", terms)
	}
	seen := map[string]bool{}
	for _, term := range terms[1:] {
		if seen[term] {
			t.Fatalf("duplicate approved term: %q", terms)
		}
		seen[term] = true
	}
	if untouched := p.termsForFinal("unrelated", "invented model noun"); len(untouched) != 1 || untouched[0] != p.Terms[0] {
		t.Fatalf("unapproved noun became memory: %q", untouched)
	}
}
