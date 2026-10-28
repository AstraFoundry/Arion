package application

import (
	"context"
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

func rewriteFixture(t *testing.T) (*Coordinator, *run, *reuseRewriter) {
	t.Helper()
	session, err := domain.NewSession("final-rewrite", 1, domain.ShortcutMode)
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
	ready := make(chan struct{})
	close(ready)
	policy := RewritePolicy{Enabled: true, Scene: "chat", Timeout: time.Second, Terms: []string{"Chatterfly"}}
	r := &run{ctx: ctx, cancel: cancel, identity: contracts.Identity{SessionID: session.ID(), Generation: 1}, session: session,
		started: time.Now(), contextReady: ready, rewriteContext: domain.RewriteContext{Screen: "Chatterfly", Before: "before", After: "after", History: []string{"earlier"}},
		live: &liveState{policy: policy, stable: "Check Chatterfly"}}
	w := &reuseRewriter{}
	c := &Coordinator{deps: Dependencies{Rewriter: w}, current: r, generation: 1}
	return c, r, w
}

func TestCancelledSessionDoesNotDeliverRewrite(t *testing.T) {
	c, r, _ := rewriteFixture(t)
	c.Cancel()
	if _, _, _, ok := c.rewriteFinal(r, r.live.stable, r.live.policy); ok {
		t.Fatal("cancelled session delivered a rewrite")
	}
}

// Release sends the recognized text once. The rewrite is committed once.
func TestFinalRewriteReceivesRecognizedText(t *testing.T) {
	w := &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return "Hello.", nil }}
	c, audio, events, target, _ := liveSetup(t, []string{"hello", "|"}, w, RewritePolicy{Enabled: true, Scene: "chat"})
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	audio.push(2, 0.2)
	events.wait(t, draftIs("HELLO"))
	w.mu.Lock()
	if len(w.requests) != 0 {
		t.Fatalf("rewrite during speech: %+v", w.requests)
	}
	w.mu.Unlock()
	c.Stop()
	ready := events.wait(t, kindIs(contracts.CorrectionReady))
	events.wait(t, kindIs(contracts.CommitResultReady))
	c.CommitReviewed()
	if ready.Source != "HELLO" || ready.Text != "Hello." || target.commits.Load() != 1 {
		t.Fatalf("unexpected delivery: %+v commits=%d", ready, target.commits.Load())
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.requests) != 1 || w.requests[0].Text != "HELLO" || len(w.ended) != 1 {
		t.Fatalf("unexpected worker lifecycle: %+v ended=%d", w.requests, len(w.ended))
	}
}

func TestFinalRewriteCarriesRecognizedPreviousSource(t *testing.T) {
	c, r, w := rewriteFixture(t)
	w.answer = func(_ context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
		return ports.RewriteResult{Text: req.Text}, nil
	}
	source := "Check chatter fly"
	text, _, _, ok := c.rewriteFinal(r, source, r.live.policy)
	if !ok || text != source || len(w.requests) != 1 {
		t.Fatalf("unexpected final: %q, calls=%d", text, len(w.requests))
	}
	final := w.requests[0]
	if final.Text != source || final.PreviousText != "Check Chatterfly" {
		t.Fatalf("recognized source missing: %+v", final)
	}
}

func TestEarlierSourceEvidenceStaysWithinCurrentGeneration(t *testing.T) {
	for _, name := range []string{"cancelled", "new session", "new generation", "no live state"} {
		t.Run(name, func(t *testing.T) {
			c, r, _ := rewriteFixture(t)
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
	c, r, w := rewriteFixture(t)
	r.rewriteContext = domain.RewriteContext{}
	r.live.policy.Terms = nil
	r.live.policy.Vocabulary = domain.NewVocabulary([]domain.Replacement{{From: "Chapter fly", To: "Chatterfly"}})
	w.answer = func(_ context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
		return ports.RewriteResult{Text: req.Text}, nil
	}
	source := "Check chad fly"
	if _, _, _, ok := c.rewriteFinal(r, source, r.live.policy); !ok {
		t.Fatal("final request cancelled")
	}
	final := w.requests[0]
	if final.PreviousText != "Check Chatterfly" || len(final.Terms) != 1 || final.Terms[0] != "Chatterfly" || !final.Context.Empty() {
		t.Fatalf("approved memory was lost: %+v", final)
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
	if len(terms) != domain.MaxRecallTerms+1 || terms[0] != p.Terms[0] || terms[1] != "AlphaBrand" {
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

func TestFinalVocabularyAliasesShareBudgetAcrossRecognitionSources(t *testing.T) {
	p := RewritePolicy{Vocabulary: domain.NewVocabulary([]domain.Replacement{
		{From: "cloud code", To: "Claude Code"}, {From: "clod code", To: "Claude Code"},
		{From: "阿里昂", To: "Arion"},
	})}
	got := p.termsForFinal("cloud code", "clod code 阿里昂")
	if len(got) != 2 || got[0] != "Claude Code" || got[1] != "Arion" {
		t.Fatalf("aliases consumed context budget: %q", got)
	}
}
