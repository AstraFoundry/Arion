package application

import (
	"context"
	"errors"
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
	// Timeout bounds the final rewrite after the user stops.
	Timeout time.Duration
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
	// cancelPreview stops the running preview when newer text arrives.
	cancelPreview context.CancelFunc
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
		if len(l.units) > 0 || !l.policy.Enabled || l.rewriteCtx.Err() != nil {
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
	c.mu.Unlock()
	// Previews use context once it is read and never wait for it.
	rc := r.contextFor(0)
	started := time.Now()
	result, err := c.deps.Rewriter.Rewrite(ctx, ports.RewriteRequest{Identity: r.identity, Text: stable,
		Scene: string(domain.ModelScene(domain.Scene(l.policy.Scene), stable)), Terms: withContextTerms(l.policy.terms(stable), rc, stable), Live: true})
	c.diagnostic(r, ports.DiagnosticPreview, started, err)
	c.mu.Lock()
	l.cancelPreview = nil
	c.mu.Unlock()
	if err != nil {
		return
	}
	text := domain.AlignLiveEnding(stable, result.Text)
	verdict := domain.ValidateRewrite(domain.RewriteCheck{Source: stable, Candidate: text, Truncated: result.Truncated, Terms: l.policy.terms(stable), Context: rc})
	if c.deps.Diagnostics != nil {
		c.deps.Diagnostics.Observe(ports.Diagnostic{Identity: r.identity, Stage: ports.DiagnosticValidation, Verdict: verdict})
	}
	if verdict != domain.RewriteAccepted {
		return
	}
	c.mu.Lock()
	if l.stable == stable {
		l.polished, l.polishedFor = text, stable
		l.previews++
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
	if c.deps.Rewriter == nil {
		return source, RewriteUnavailable, true
	}
	ctx, cancel := context.WithTimeout(r.ctx, policy.Timeout)
	defer cancel()
	rc := r.contextFor(contextWait)
	started := time.Now()
	result, err := c.deps.Rewriter.Rewrite(ctx, ports.RewriteRequest{Identity: r.identity, Text: source,
		Scene: string(domain.ModelScene(domain.Scene(policy.Scene), source)), Terms: withContextTerms(policy.terms(source), rc, source)})
	c.diagnostic(r, ports.DiagnosticRewrite, started, err)
	switch {
	case r.ctx.Err() != nil:
		return "", "", false
	case err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
		return source, RewriteTimeout, true
	case err != nil:
		return source, RewriteUnavailable, true
	}
	verdict := domain.ValidateRewrite(domain.RewriteCheck{Source: source, Candidate: result.Text, Truncated: result.Truncated, Terms: policy.terms(source), Context: rc})
	if verdict != domain.RewriteAccepted {
		return source, string(verdict), true
	}
	return result.Text, string(verdict), true
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
