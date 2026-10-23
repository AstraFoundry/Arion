package application

import (
	"context"
	"errors"
	"slices"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// RewritePolicy configures the optional LLM stage of one session.
type RewritePolicy struct {
	Enabled bool
	// Scene is the formatting scene (a domain.Scene). With FollowApp it is
	// the fallback until the session's target is captured.
	Scene string
	// FollowApp decides Scene from the focused target's field, site and
	// application when the session starts (domain.DecideScene).
	FollowApp bool
	// SceneOverrides are the user's choices by place; they beat the
	// built-in rules when FollowApp decides the scene.
	SceneOverrides domain.SceneOverrides
	// Context selects the context sources the rewrite may read.
	Context ContextPolicy
	// Terms are fixed strings shown to the model as data.
	Terms []string
	// Vocabulary is the approved memory; only its entries relevant to the
	// text reach the model, at most domain.MaxRecallTerms (ADR 014).
	Vocabulary *domain.Vocabulary
	// Punctuation is the polished-text punctuation mode. Empty is smart.
	// Live preview and the inserted final text use the same mode.
	Punctuation domain.Punctuation
	// Timeout bounds the final rewrite after the user stops.
	Timeout time.Duration
	// FinalOnly skips live previews: only the text after stop is polished.
	// Cloud polishing uses it so one request is sent per dictation (ADR 022).
	FinalOnly bool
}

// Rewrite outcomes that are not validation verdicts.
const (
	RewriteTimeout     = "timeout"
	RewriteUnavailable = "unavailable"
	// RewritePreview marks a draft whose stable part is a validated preview.
	RewritePreview = "preview"
)

const (
	defaultRewriteTimeout = 10 * time.Second
	liveUnitQueue         = 64
	// cutSearch is how far back from an endpoint report the pause is sought.
	cutSearch = 2500 * time.Millisecond
)

// unit is one closed recognizer segment with its audio, in order.
type unit struct {
	text    string
	samples []float32
}

// liveState runs the edit-while-speaking pipeline of one session: each
// closed segment is re-decoded by the finalizer, and the stable text is
// rewritten as a preview while the speaker continues. Previews never commit;
// the final rewrite after stop is the only inserted proposal.
type liveState struct {
	policy RewritePolicy
	units  chan unit
	done   chan struct{}
	// rewriteCtx ends previews: cancelled at stop, before the final request.
	rewriteCtx  context.Context
	stopPreview context.CancelFunc

	// Owned by the pump goroutine.
	closed   int
	bound    int
	overflow bool
	closedAt bool

	// Guarded by Coordinator.mu.
	refined     []string
	pending     []string
	tail        string
	stable      string
	polished    string
	polishedFor string
	previews    int
	completed   *completedPreview
	// cancelPreview stops the running preview when newer text arrives.
	cancelPreview context.CancelFunc
}

// completedPreview retains the raw proposal, before display-only ending alignment.
// Live and final inference use the same rewrite policy; Live changes only how
// the coordinator displays the completed proposal while capture continues.
type completedPreview struct {
	request ports.RewriteRequest
	result  ports.RewriteResult
}

func snapshotRewriteRequest(request ports.RewriteRequest) ports.RewriteRequest {
	request.Terms = slices.Clone(request.Terms)
	request.Context.History = slices.Clone(request.Context.History)
	return request
}

func sameRewriteRequest(a, b ports.RewriteRequest) bool {
	return a.Identity == b.Identity && a.SourceRevision == b.SourceRevision &&
		a.Text == b.Text && a.PreviousText == b.PreviousText && a.Scene == b.Scene && a.Punctuation == b.Punctuation && a.Prefill == b.Prefill &&
		slices.Equal(a.Terms, b.Terms) && a.Context.Screen == b.Context.Screen &&
		a.Context.Before == b.Context.Before && a.Context.After == b.Context.After &&
		slices.Equal(a.Context.History, b.Context.History)
}

// previousStableSource carries only recognized, rule-corrected text from the
// current session. Identical source adds no spelling alternatives and keeps a
// completed preview eligible for exact reuse.
func (c *Coordinator) previousStableSource(r *run, source string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current != r || c.generation != r.identity.Generation || r.ctx.Err() != nil || r.live == nil || r.live.stable == source {
		return ""
	}
	return r.live.stable
}

// completedRewrite never reuses another session's result or a proposal whose
// effective source, scene, approved terms or context changed after preview.
func (c *Coordinator) completedRewrite(r *run, request ports.RewriteRequest) (ports.RewriteResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current != r || c.generation != r.identity.Generation || r.ctx.Err() != nil || r.live == nil || r.live.completed == nil {
		return ports.RewriteResult{}, false
	}
	completed := r.live.completed
	if !sameRewriteRequest(completed.request, request) {
		return ports.RewriteResult{}, false
	}
	return completed.result, true
}

func (c *Coordinator) newLive(r *run) *liveState {
	policy := c.opts.Rewrite()
	if !policy.Enabled && !c.opts.SegmentFinalize {
		return nil
	}
	if policy.Timeout <= 0 {
		policy.Timeout = defaultRewriteTimeout
	}
	ctx, cancel := context.WithCancel(r.ctx)
	return &liveState{policy: policy, units: make(chan unit, liveUnitQueue), done: make(chan struct{}), rewriteCtx: ctx, stopPreview: cancel}
}

// queueClosed hands newly closed segments and their audio to the live
// worker and reports whether any closed. It runs on the pump and never
// blocks: on overflow the remaining audio is finalized as one unit after stop.
func (c *Coordinator) queueClosed(r *run, draft contracts.Draft) bool {
	l := r.live
	if l.overflow || len(draft.Segments) <= l.closed {
		return false
	}
	// The endpoint is reported after the decoder has seen trailing silence,
	// by which time the next phrase may have started; cut inside the pause.
	cut := quietCut(r.samples, l.bound, len(r.samples), r.sampleRate)
	u := unit{text: domain.JoinText(draft.Segments[l.closed:]...), samples: append([]float32(nil), r.samples[l.bound:cut]...)}
	// pending is recorded before the send so the worker always finds it.
	c.mu.Lock()
	l.pending = append(l.pending, u.text)
	c.mu.Unlock()
	select {
	case l.units <- u:
		l.closed, l.bound = len(draft.Segments), cut
		// Latest wins: a preview of older text would only delay refinement.
		c.mu.Lock()
		if l.cancelPreview != nil {
			l.cancelPreview()
		}
		c.mu.Unlock()
	default:
		c.mu.Lock()
		l.pending = l.pending[:len(l.pending)-1]
		c.mu.Unlock()
		l.overflow = true
	}
	if c.deps.Diagnostics != nil {
		c.deps.Diagnostics.Observe(ports.Diagnostic{Identity: r.identity, Stage: ports.DiagnosticSegmentQueue, QueueDepth: len(l.units), QueueOverflow: l.overflow})
	}
	return true
}

// closeUnits ends the unit stream; only the pump sends units.
func (l *liveState) closeUnits() {
	if !l.closedAt {
		l.closedAt = true
		close(l.units)
	}
}

// shutdown stops previews and waits for the worker after a cancel/failure.
func (l *liveState) shutdown() {
	l.stopPreview()
	l.closeUnits()
	<-l.done
}

// acceptLiveDraft records the open segment and publishes the display text.
func (c *Coordinator) acceptLiveDraft(r *run, draft contracts.Draft) {
	tail := draft.Tail
	if len(draft.Segments) == 0 && tail == "" {
		tail = draft.Text
	}
	c.mu.Lock()
	r.live.tail = tail
	c.mu.Unlock()
	c.publishLive(r)
}

// display joins the best available closed text with the open segment:
// the validated preview of the stable source, else the refined text, then
// segments still waiting for refinement, then the open tail.
func (l *liveState) display() (string, bool) {
	closed, previewed := l.stable, false
	if l.polished != "" && l.polishedFor == l.stable {
		closed, previewed = l.polished, true
	}
	parts := append([]string{closed}, l.pending...)
	return domain.JoinText(append(parts, l.tail)...), previewed
}

func (c *Coordinator) publishLive(r *run) {
	c.mu.Lock()
	if c.current != r {
		c.mu.Unlock()
		return
	}
	text, previewed := r.live.display()
	if text == r.draft {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	rewrite := ""
	if previewed {
		rewrite = RewritePreview
	}
	c.acceptDisplay(r, text, rewrite)
}

// liveWorker refines closed segments in order and previews the latest
// stable text. Inference never runs on the capture or pump goroutines.
func (c *Coordinator) liveWorker(r *run) {
	l := r.live
	defer close(l.done)
	for u := range l.units {
		text, err := c.refine(r, u)
		if err != nil {
			if r.ctx.Err() == nil {
				c.failErr(r, err, domain.CodeModelUnsupported)
			}
			continue
		}
		c.mu.Lock()
		l.refined = append(l.refined, text)
		l.pending = l.pending[1:]
		l.stable = c.correct(domain.JoinText(l.refined...))
		stable := l.stable
		c.mu.Unlock()
		c.publishLive(r)
		// Latest wins: only preview once every queued segment is refined.
		if len(l.units) > 0 || !l.policy.Enabled || l.policy.FinalOnly || l.rewriteCtx.Err() != nil {
			continue
		}
		c.preview(r, stable)
	}
}

// refine re-decodes one segment's audio when per-segment finalization is
// on, or keeps the streaming text when it is off, no finalizer is
// configured or nothing was recognized in the segment.
func (c *Coordinator) refine(r *run, u unit) (string, error) {
	if !c.opts.SegmentFinalize || c.deps.Finalizer == nil || len(u.samples) == 0 || u.text == "" {
		return u.text, nil
	}
	started := time.Now()
	text, err := c.deps.Finalizer.Finalize(r.ctx, ports.FinalizeRequest{Identity: r.identity, SampleRate: r.sampleRate, Samples: u.samples, Draft: u.text})
	c.diagnostic(r, ports.DiagnosticFinalize, started, err)
	return text, err
}

func (c *Coordinator) correct(raw string) string {
	if !c.opts.Correction {
		return raw
	}
	return domain.Correct(raw, c.opts.Policy()).Text
}

// preview requests a live rewrite of the stable source. Failures and
// rejections leave the refined source on screen; they never end the
// session, because the final rewrite decides what is inserted.
func (c *Coordinator) preview(r *run, stable string) {
	l := r.live
	if stable == "" || c.deps.Rewriter == nil {
		return
	}
	ctx, cancel := context.WithCancel(l.rewriteCtx)
	defer cancel()
	c.mu.Lock()
	l.cancelPreview = cancel
	l.completed = nil
	c.mu.Unlock()
	// Previews use context once it is read and never wait for it.
	rc := r.contextFor(0)
	started := time.Now()
	request := snapshotRewriteRequest(ports.RewriteRequest{Identity: r.identity, Text: stable,
		Scene: string(domain.ModelScene(domain.Scene(l.policy.Scene), stable)), Punctuation: string(l.policy.Punctuation), Terms: l.policy.terms(stable), Context: rc, Live: true})
	completedRequest := snapshotRewriteRequest(request)
	result, err := c.deps.Rewriter.Rewrite(ctx, request)
	c.diagnostic(r, ports.DiagnosticPreview, started, err)
	c.mu.Lock()
	l.cancelPreview = nil
	c.mu.Unlock()
	if err != nil || ctx.Err() != nil {
		return
	}
	text := punctuate(l.policy.Punctuation, result.Text, true, stable)
	verdict := domain.ValidateRewriteDelivery(domain.RewriteCheck{Source: stable, Candidate: text, Truncated: result.Truncated, Terms: completedRequest.Terms, Context: completedRequest.Context})
	if c.deps.Diagnostics != nil {
		c.deps.Diagnostics.Observe(ports.Diagnostic{Identity: r.identity, Stage: ports.DiagnosticValidation, Verdict: verdict})
	}
	if verdict != domain.RewriteAccepted {
		return
	}
	c.mu.Lock()
	if c.current == r && c.generation == r.identity.Generation && r.ctx.Err() == nil && ctx.Err() == nil && l.stable == stable && len(l.pending) == 0 {
		l.polished, l.polishedFor = text, stable
		l.previews++
		result.Cleanup.Spans = slices.Clone(result.Cleanup.Spans)
		l.completed = &completedPreview{request: completedRequest, result: result}
	}
	c.mu.Unlock()
	c.publishLive(r)
}

// finalizeLive flushes the recognizer, stops previews, refines what was not
// refined during speech and delivers one final rewrite.
func (c *Coordinator) finalizeLive(r *run) {
	l := r.live
	draft, err := r.recognizer.Finish(r.ctx)
	if err != nil {
		l.shutdown()
		c.failErr(r, err, domain.CodeModelUnsupported)
		return
	}
	l.stopPreview()
	l.closeUnits()
	<-l.done
	if r.ctx.Err() != nil {
		return
	}
	rest := unit{samples: r.samples[l.bound:]}
	if l.closed < len(draft.Segments) {
		rest.text = domain.JoinText(append(append([]string(nil), draft.Segments[l.closed:]...), draft.Tail)...)
	} else {
		rest.text = draft.Tail
		if len(draft.Segments) == 0 && rest.text == "" {
			rest.text = draft.Text
		}
	}
	var raw string
	if !c.opts.SegmentFinalize && c.deps.Finalizer != nil && len(r.samples) > 0 {
		// Previews used streaming segments; the finalizer decodes the whole
		// utterance once, with full context.
		c.mu.Lock()
		streamed := domain.JoinText(append(append([]string(nil), l.refined...), rest.text)...)
		c.mu.Unlock()
		started := time.Now()
		raw, err = c.deps.Finalizer.Finalize(r.ctx, ports.FinalizeRequest{Identity: r.identity, SampleRate: r.sampleRate, Samples: r.samples, Draft: streamed})
		c.diagnostic(r, ports.DiagnosticFinalize, started, err)
	} else {
		var last string
		last, err = c.refine(r, rest)
		c.mu.Lock()
		raw = domain.JoinText(append(append([]string(nil), l.refined...), last)...)
		c.mu.Unlock()
	}
	if err != nil {
		c.failErr(r, err, domain.CodeModelUnsupported)
		return
	}
	c.deliver(r, raw, l.policy)
}

// rewriteFinal runs the final rewrite of the corrected source and returns
// the text to review and its outcome. ok is false when the session ended
// meanwhile. Organization failures retain completed local cleanup and
// their explicit failure outcome; the recognized source stays separate.
func (c *Coordinator) rewriteFinal(r *run, source string, policy RewritePolicy) (text, outcome string, ok bool) {
	if !policy.Enabled {
		return source, "", true
	}
	cleaned := domain.CleanSpeech(source).Text
	if c.deps.Rewriter == nil {
		return cleaned, RewriteUnavailable, true
	}
	ctx, cancel := context.WithTimeout(r.ctx, policy.Timeout)
	defer cancel()
	rc := r.contextFor(contextWait)
	previous := c.previousStableSource(r, source)
	request := snapshotRewriteRequest(ports.RewriteRequest{Identity: r.identity, Text: source, PreviousText: previous,
		Scene: string(domain.ModelScene(domain.Scene(policy.Scene), source)), Punctuation: string(policy.Punctuation), Terms: policy.termsForFinal(source, previous), Context: rc})
	result, reused := c.completedRewrite(r, request)
	var err error
	if !reused {
		c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.RewriteStarted}, State: domain.Finalizing})
		started := time.Now()
		result, err = c.deps.Rewriter.Rewrite(ctx, request)
		c.diagnostic(r, ports.DiagnosticRewrite, started, err)
	}
	switch {
	case r.ctx.Err() != nil:
		return "", "", false
	case err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
		return cleaned, RewriteTimeout, true
	case err != nil:
		return cleaned, RewriteUnavailable, true
	}
	text = punctuate(policy.Punctuation, result.Text, false, source)
	verdict := domain.ValidateRewriteDelivery(domain.RewriteCheck{Source: source, Candidate: text, Truncated: result.Truncated, Terms: request.Terms, Context: request.Context})
	if verdict != domain.RewriteAccepted {
		return cleaned, string(verdict), true
	}
	return text, string(verdict), true
}

// punctuate applies the punctuation mode to a model proposal. Smart and
// full keep the model's punctuation; a live preview of those modes still
// aligns its ending with the open source. Omit and spaces post-edit both
// the live preview and the inserted text, so the mode is what is shown.
// Search removes a sentence-final mark later, in FinishScene.
func punctuate(mode domain.Punctuation, candidate string, live bool, source string) string {
	switch mode {
	case domain.PunctuationOmit, domain.PunctuationSpaces:
		return domain.ApplyPunctuation(mode, candidate)
	default:
		if live {
			return domain.AlignLiveEnding(source, candidate)
		}
		return candidate
	}
}

// LiveStats reports how many validated previews a session displayed.
func (c *Coordinator) LiveStats() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.current == nil || c.current.live == nil {
		return 0
	}
	return c.current.live.previews
}

// quietCut returns the middle of the longest pause in the last
// cutSearch of samples[from:to], or to when no pause is found. A pause is a
// run of 20 ms windows whose energy is under 1% of the loudest window.
func quietCut(samples []float32, from, to, rate int) int {
	window := rate / 50
	if window <= 0 {
		return to
	}
	start := max(from, to-int(cutSearch.Seconds()*float64(rate)))
	n := (to - start) / window
	if n < 2 {
		return to
	}
	energy := make([]float64, n)
	peak := 0.0
	for i := range energy {
		var sum float64
		for _, v := range samples[start+i*window : start+(i+1)*window] {
			sum += float64(v) * float64(v)
		}
		energy[i] = sum / float64(window)
		peak = max(peak, energy[i])
	}
	threshold := peak/100 + 1e-9
	bestStart, bestLen, runStart := -1, 0, -1
	for i := 0; i <= n; i++ {
		if i < n && energy[i] <= threshold {
			if runStart < 0 {
				runStart = i
			}
			continue
		}
		if runStart >= 0 && i-runStart > bestLen {
			bestStart, bestLen = runStart, i-runStart
		}
		runStart = -1
	}
	if bestLen < 5 { // shorter than 100 ms is not a pause
		return to
	}
	return start + (bestStart*window + bestLen*window/2)
}

// terms are the fixed terms and the memory entries relevant to text.
func (p RewritePolicy) terms(text string) []string {
	recalled := p.Vocabulary.Terms(text, domain.MaxRecallTerms)
	if len(recalled) == 0 {
		return p.Terms
	}
	return append(append([]string(nil), p.Terms...), recalled...)
}

// termsForFinal retains only approved vocabulary relevant to either recognized
// source. Earlier model output and unapproved nouns never become memory terms.
func (p RewritePolicy) termsForFinal(source, previous string) []string {
	recalled := p.Vocabulary.Terms(source, domain.MaxRecallTerms)
	if previous != "" {
		for _, term := range p.Vocabulary.Terms(previous, domain.MaxRecallTerms) {
			if len(recalled) == domain.MaxRecallTerms {
				break
			}
			if !slices.Contains(recalled, term) {
				recalled = append(recalled, term)
			}
		}
	}
	if len(recalled) == 0 {
		return p.Terms
	}
	return append(slices.Clone(p.Terms), recalled...)
}
