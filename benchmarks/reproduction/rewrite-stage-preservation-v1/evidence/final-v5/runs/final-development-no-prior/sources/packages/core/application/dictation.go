// Package application orchestrates dictation sessions over domain-owned ports.
package application

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// Dependencies are the adapters wired by a composition root. Targets and
// Input are required only when Options.AutoCommit or CommitReviewed is used.
// The IME adapters serve system_ime sessions: the input method owns the
// marked-text range, receives every draft and commits the reviewed text.
type Dependencies struct {
	// SelectMode chooses the route once at session start. It must return quickly
	// and must not call the coordinator. Nil uses Options.Mode for explicit hosts.
	SelectMode  func() domain.InputMode
	Diagnostics ports.Diagnostics
	Audio       ports.AudioCapture
	Recognizers ports.RecognizerFactory
	Finalizer   ports.Finalizer
	Targets     ports.TargetProvider
	Input       ports.InputTarget
	IMETargets  ports.TargetProvider
	IMEInput    ports.InputTarget
	Composition ports.CompositionTarget
	// Rewriter is the optional local LLM stage; nil leaves dictation as
	// recognized and rule-corrected text (009).
	Rewriter ports.Rewriter
	// History keeps recent dictations per place for rewrite context; nil
	// keeps none.
	History *History
	// Edits follows committed fields to learn the user's corrections; nil
	// learns nothing (ADR 014).
	Edits *EditTracker
	NewID func() string
}

// Options configure one coordinator. Policy is read when a session finalizes,
// so settings changes apply to the next review without restarting.
type Options struct {
	Mode       domain.InputMode
	AutoCommit bool
	Correction bool
	Policy     func() domain.CorrectionPolicy
	// Rewrite is read when a session starts; enabling it turns on the live
	// pipeline: per-segment finalization, live previews and a final rewrite.
	Rewrite func() RewritePolicy
	// SegmentFinalize runs the finalizer per closed segment while speaking
	// even without rewriting.
	SegmentFinalize bool
	// Learning is read at each commit; it gates edit tracking.
	Learning func() bool
	// Recalled receives the vocabulary entries that corrected a dictation,
	// off the delivery path's critical section; nil ignores them.
	Recalled    func([]domain.Replacement)
	QueueFrames int
	MaxAudio    time.Duration
}

const (
	defaultQueueFrames = 256
	defaultMaxAudio    = 5 * time.Minute
)

// Listener receives events in sequence order. It must return quickly.
type Listener func(contracts.Event)

// Coordinator owns at most one active dictation session. Every public method
// is safe for concurrent use; callbacks from stale generations are dropped.
type Coordinator struct {
	deps Dependencies
	opts Options

	mu         sync.Mutex
	generation uint64
	current    *run

	emitMu    sync.Mutex
	listeners []Listener
}

type run struct {
	session  *domain.Session
	identity contracts.Identity
	ctx      context.Context
	cancel   context.CancelFunc
	started  time.Time
	sequence atomic.Uint64

	frames    chan contracts.Frame
	overflow  atomic.Bool
	stopOnce  sync.Once
	stream    ports.AudioStream
	pumpDone  chan struct{}
	stopAsked bool

	recognizer ports.Recognizer
	target     contracts.TargetSnapshot
	hasTarget  bool
	// place names the target for history; contextReady closes once
	// rewriteContext is read (startContext).
	place          string
	contextReady   chan struct{}
	rewriteContext domain.RewriteContext
	// live is set at start when the live pipeline runs for this session.
	live *liveState

	// Owned by the pump goroutine.
	samples    []float32
	sampleRate int
	lastText   string

	// Guarded by Coordinator.mu.
	draft string
	raw   string
	final string
}

// NewCoordinator validates its configuration.
func NewCoordinator(deps Dependencies, opts Options) (*Coordinator, error) {
	if deps.Audio == nil || deps.Recognizers == nil || deps.NewID == nil {
		return nil, domain.Fail(domain.CodeContractUnsupported, "audio, recognizer and id adapters are required")
	}
	if opts.AutoCommit && (deps.Targets == nil || deps.Input == nil) {
		return nil, domain.Fail(domain.CodeContractUnsupported, "auto commit requires target adapters")
	}
	if opts.Mode == "" {
		opts.Mode = domain.ShortcutMode
	}
	if opts.QueueFrames <= 0 {
		opts.QueueFrames = defaultQueueFrames
	}
	if opts.MaxAudio <= 0 {
		opts.MaxAudio = defaultMaxAudio
	}
	if opts.Policy == nil {
		opts.Policy = func() domain.CorrectionPolicy { return domain.CorrectionPolicy{} }
	}
	if opts.Rewrite == nil {
		opts.Rewrite = func() RewritePolicy { return RewritePolicy{} }
	}
	return &Coordinator{deps: deps, opts: opts}, nil
}

// Subscribe registers a listener and returns a function removing it.
func (c *Coordinator) Subscribe(listener Listener) func() {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	c.listeners = append(c.listeners, listener)
	index := len(c.listeners) - 1
	return func() {
		c.emitMu.Lock()
		defer c.emitMu.Unlock()
		c.listeners[index] = nil
	}
}

// Snapshot is the current session state for views that attach late.
type Snapshot struct {
	Mode     domain.InputMode
	Identity contracts.Identity
	State    domain.SessionState
	Draft    string
	Final    string
	Failure  domain.ErrorCode
	Active   bool
}

func (c *Coordinator) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := c.current
	if r == nil {
		return Snapshot{State: domain.Idle}
	}
	return Snapshot{Mode: r.session.Mode(), Identity: r.identity, State: r.session.State(), Draft: r.draft, Final: r.final,
		Failure: r.session.Failure(), Active: !r.session.Terminal()}
}

// SetMode switches input mode. It closes the current generation (DSK-01/A06).
func (c *Coordinator) SetMode(mode domain.InputMode) {
	c.Cancel()
	c.mu.Lock()
	c.opts.Mode = mode
	c.mu.Unlock()
}

// Toggle starts a session when idle, stops capture while capturing and
// cancels one still preparing. Toggling during finalization reports busy.
func (c *Coordinator) Toggle() {
	c.mu.Lock()
	r := c.current
	state := domain.Idle
	if r != nil {
		state = r.session.State()
	}
	c.mu.Unlock()
	switch {
	case r == nil || domain.IsTerminal(state):
		c.Start()
	case state == domain.Capturing:
		c.Stop()
	case state == domain.Preparing:
		c.Cancel()
	default:
		c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.ErrorOccurred}, State: state, Error: domain.CodeSessionBusy})
	}
}

// Start begins a new session in a new generation. Adapter calls that may
// block run off the caller's thread; failures are delivered as events.
func (c *Coordinator) Start() {
	if c.deps.Edits != nil {
		// The next insertion changes the field: judge the last one now.
		c.deps.Edits.Settle()
	}
	var selectedMode domain.InputMode
	if c.deps.SelectMode != nil {
		selectedMode = c.deps.SelectMode()
	}
	c.mu.Lock()
	if c.current != nil && !c.current.session.Terminal() {
		r := c.current
		c.mu.Unlock()
		c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.ErrorOccurred}, State: r.session.State(), Error: domain.CodeSessionBusy})
		return
	}
	c.generation++
	mode := c.opts.Mode
	if c.deps.SelectMode != nil {
		mode = selectedMode
	}
	session, err := domain.NewSession(c.deps.NewID(), c.generation, mode)
	if err != nil {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &run{session: session, identity: contracts.Identity{SessionID: session.ID(), Generation: session.Generation()},
		ctx: ctx, cancel: cancel, started: time.Now(), frames: make(chan contracts.Frame, c.opts.QueueFrames), pumpDone: make(chan struct{})}
	r.live = c.newLive(r)
	_ = session.Advance(domain.Preparing)
	c.current = r
	needTarget := c.opts.AutoCommit
	c.mu.Unlock()
	c.emitState(r, domain.Preparing)
	go c.prepare(r, needTarget)
}

// targets returns the capture and commit adapters for the session's mode.
func (c *Coordinator) targets(r *run) (ports.TargetProvider, ports.InputTarget) {
	if r.session.Mode() == domain.SystemIMEMode {
		return c.deps.IMETargets, c.deps.IMEInput
	}
	return c.deps.Targets, c.deps.Input
}

// composing reports whether drafts are mirrored into an owned IME range.
func (c *Coordinator) composing(r *run) bool {
	return r.session.Mode() == domain.SystemIMEMode && c.deps.Composition != nil && r.hasTarget
}

func (c *Coordinator) prepare(r *run, needTarget bool) {
	if needTarget {
		provider, _ := c.targets(r)
		if provider == nil {
			c.failDetail(r, domain.CodeTargetUnsupported, "input mode unavailable")
			return
		}
		started := time.Now()
		target, err := provider.Capture(r.ctx, r.identity)
		c.diagnostic(r, ports.DiagnosticTarget, started, err)
		if err != nil {
			c.failErr(r, err, domain.CodePermissionDenied)
			return
		}
		if target.Sensitive {
			c.fail(r, domain.CodeSecureFieldBlocked)
			return
		}
		r.target, r.hasTarget = target, true
	}
	if l := r.live; l != nil && l.policy.Enabled && l.policy.FollowApp {
		// Set before the live worker starts; only it and delivery read it.
		var hints domain.SceneHints
		if r.hasTarget {
			hints = r.target.SceneHints()
		}
		l.policy.Scene = string(domain.DecideScene(hints, l.policy.SceneOverrides).Scene)
	}
	c.startContext(r)
	started := time.Now()
	recognizer, err := c.deps.Recognizers.NewRecognizer(r.ctx, r.identity)
	c.diagnostic(r, ports.DiagnosticRecognizer, started, err)
	if err != nil {
		c.failErr(r, err, domain.CodeModelMissing)
		return
	}
	r.recognizer = recognizer
	started = time.Now()
	stream, err := c.deps.Audio.Start(r.ctx, r.identity, func(frame contracts.Frame) {
		if r.overflow.Load() {
			return
		}
		select {
		case r.frames <- frame:
		default:
			// SPH-02: never conceal dropped audio.
			if !r.overflow.Swap(true) {
				go c.fail(r, domain.CodeAudioDiscontinuity)
			}
		}
	})
	c.diagnostic(r, ports.DiagnosticCapture, started, err)
	if err != nil {
		_ = recognizer.Close()
		c.failErr(r, err, domain.CodeDeviceUnavailable)
		return
	}
	c.mu.Lock()
	r.stream = stream
	if c.current != r || r.session.Terminal() {
		c.mu.Unlock()
		r.stopAudio()
		_ = recognizer.Close()
		return
	}
	_ = r.session.Advance(domain.Capturing)
	c.mu.Unlock()
	c.emitState(r, domain.Capturing)
	if r.live != nil {
		go c.liveWorker(r)
	}
	go c.pump(r)
}

func (r *run) stopAudio() {
	r.stopOnce.Do(func() {
		if r.stream != nil {
			_ = r.stream.Stop()
		}
		close(r.frames)
	})
}

// pump is the only goroutine that touches the recognizer after capture
// starts, so native resources are never freed while decoding.
func (c *Coordinator) pump(r *run) {
	defer close(r.pumpDone)
	maxSamples := 0
	for frame := range r.frames {
		if r.ctx.Err() != nil {
			continue
		}
		if frame.Discontinuity || r.overflow.Load() {
			c.fail(r, domain.CodeAudioDiscontinuity)
			continue
		}
		if r.sampleRate == 0 {
			r.sampleRate = int(frame.SampleRate)
			maxSamples = int(c.opts.MaxAudio.Seconds() * float64(frame.SampleRate) * float64(max(frame.Channels, 1)))
		}
		r.samples = append(r.samples, frame.Samples...)
		c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.AudioLevel}, Level: Level(frame.Samples)})
		if err := r.recognizer.Accept(r.ctx, frame); err != nil {
			c.failErr(r, err, domain.CodeModelUnsupported)
			continue
		}
		draft, err := r.recognizer.Partial(r.ctx)
		if err != nil {
			c.failErr(r, err, domain.CodeModelUnsupported)
			continue
		}
		if r.live != nil {
			// A segment closing moves text from the tail to the stable part
			// even when the joined text is unchanged.
			if c.queueClosed(r, draft) || draft.Text != r.lastText {
				r.lastText = draft.Text
				c.acceptLiveDraft(r, draft)
			}
		} else if draft.Text != r.lastText {
			r.lastText = draft.Text
			c.acceptDraft(r, draft.Text)
		}
		if len(r.samples) >= maxSamples {
			go c.Stop()
		}
	}
	if r.overflow.Load() {
		c.fail(r, domain.CodeAudioDiscontinuity)
	}
	c.mu.Lock()
	finalize := r.stopAsked && c.current == r && r.session.State() == domain.Finalizing
	c.mu.Unlock()
	switch {
	case finalize && r.ctx.Err() == nil && r.live != nil:
		c.finalizeLive(r)
	case finalize && r.ctx.Err() == nil:
		c.finalize(r)
	case r.live != nil:
		r.live.shutdown()
	}
	_ = r.recognizer.Close()
	if r.live != nil && c.deps.Rewriter != nil {
		// No private KV state outlives the session in the worker.
		c.deps.Rewriter.End(r.identity)
	}
	r.samples = nil // SPH-09: session audio is discarded when processing ends.
}

func (c *Coordinator) acceptDraft(r *run, text string) { c.acceptDisplay(r, text, "") }

// acceptDisplay publishes a draft; rewrite is "preview" when the display
// includes a validated live rewrite.
func (c *Coordinator) acceptDisplay(r *run, text, rewrite string) {
	c.mu.Lock()
	if c.current != r {
		c.mu.Unlock()
		return
	}
	revision := r.session.Revision() + 1
	if err := r.session.AcceptRevision(revision); err != nil {
		c.mu.Unlock()
		return
	}
	r.draft = text
	state := r.session.State()
	c.mu.Unlock()
	if c.composing(r) {
		// IME-A02: drafts update only the owned marked range; failures end
		// the session explicitly instead of continuing without composition.
		if err := c.deps.Composition.Update(r.ctx, r.target.Target, contracts.Draft{Identity: r.identity, Revision: revision, Text: text}); err != nil {
			c.failErr(r, err, domain.CodeTargetChanged)
			return
		}
	}
	c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.DraftReplaced, Revision: revision}, State: state, Text: text, Rewrite: rewrite})
}

// Stop ends capture; finalization, correction and commit continue on the
// session's own goroutine.
func (c *Coordinator) Stop() {
	c.mu.Lock()
	r := c.current
	if r == nil || r.session.State() != domain.Capturing {
		c.mu.Unlock()
		return
	}
	_ = r.session.Advance(domain.Finalizing)
	r.stopAsked = true
	c.mu.Unlock()
	c.emitState(r, domain.Finalizing)
	go r.stopAudio()
}

// Cancel invalidates the active generation. Late results cannot update the
// UI, learn or insert (DSK-A04, SPH-A02).
func (c *Coordinator) Cancel() { c.cancelMode("") }

// CancelMode invalidates only a session using the specified route. Losing an
// IME client must not cancel dictation targeting an external editor.
func (c *Coordinator) CancelMode(mode domain.InputMode) { c.cancelMode(mode) }

func (c *Coordinator) cancelMode(mode domain.InputMode) {
	c.mu.Lock()
	r := c.current
	if r == nil || r.session.Terminal() || (mode != "" && r.session.Mode() != mode) {
		c.mu.Unlock()
		return
	}
	c.generation++
	_ = r.session.Advance(domain.Cancelled)
	r.cancel()
	started := r.stream != nil
	c.mu.Unlock()
	c.releaseComposition(r)
	c.emitState(r, domain.Cancelled)
	if started {
		go r.stopAudio()
	}
}

func (c *Coordinator) fail(r *run, code domain.ErrorCode) { c.failDetail(r, code, "") }

// failErr fails with the error's code (or fallback) and its diagnostic detail.
func (c *Coordinator) failErr(r *run, err error, fallback domain.ErrorCode) {
	detail := ""
	var coded *domain.Error
	if errors.As(err, &coded) {
		detail = coded.Detail
	}
	c.failDetail(r, codeOr(err, fallback), detail)
}

func (c *Coordinator) failDetail(r *run, code domain.ErrorCode, detail string) {
	c.mu.Lock()
	if c.current != r || r.session.FailWith(code) != nil {
		c.mu.Unlock()
		return
	}
	r.cancel()
	started := r.stream != nil
	text := r.final
	if text == "" {
		text = r.draft
	}
	c.mu.Unlock()
	c.releaseComposition(r)
	c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.ErrorOccurred}, State: domain.Failed, Error: code, Detail: detail, Text: text})
	if started {
		go r.stopAudio()
	}
}

// releaseComposition clears the owned IME range of an ended session
// (IME-A03); it never touches text outside that range.
func (c *Coordinator) releaseComposition(r *run) {
	if c.composing(r) {
		_ = c.deps.Composition.Cancel(context.Background(), r.identity, r.target.Target)
	}
}

func (c *Coordinator) finalize(r *run) {
	draft, err := r.recognizer.Finish(r.ctx)
	if err != nil {
		c.failErr(r, err, domain.CodeModelUnsupported)
		return
	}
	raw := draft.Text
	if c.deps.Finalizer != nil && len(r.samples) > 0 {
		started := time.Now()
		second, err := c.deps.Finalizer.Finalize(r.ctx, ports.FinalizeRequest{Identity: r.identity, SampleRate: r.sampleRate, Samples: r.samples, Draft: raw})
		c.diagnostic(r, ports.DiagnosticFinalize, started, err)
		if err != nil {
			c.failErr(r, err, domain.CodeModelUnsupported)
			return
		}
		raw = second
	}
	c.deliver(r, raw, RewritePolicy{})
}

// deliver corrects the final recognized text, optionally rewrites it,
// reviews it and commits it when configured.
func (c *Coordinator) deliver(r *run, raw string, policy RewritePolicy) {
	source := raw
	if c.opts.Correction {
		corrected := domain.Correct(raw, c.opts.Policy())
		source = corrected.Text
		if len(corrected.Recalled) > 0 && c.opts.Recalled != nil {
			go c.opts.Recalled(corrected.Recalled)
		}
	}
	text, outcome := source, ""
	if strings.TrimSpace(source) != "" {
		var ok bool
		if text, outcome, ok = c.rewriteFinal(r, source, policy); !ok {
			return
		}
		if policy.Enabled {
			text = domain.FinishScene(domain.Scene(policy.Scene), text)
		}
	}
	c.mu.Lock()
	if c.current != r || r.ctx.Err() != nil {
		c.mu.Unlock()
		return
	}
	revision := r.session.Revision() + 1
	if err := r.session.AcceptRevision(revision); err != nil {
		c.mu.Unlock()
		return
	}
	r.raw, r.draft = raw, raw
	if strings.TrimSpace(text) == "" {
		// SPH-09 / SPH-A07: silence never commits text.
		_ = r.session.FailWith(domain.CodeNoSpeech)
		c.mu.Unlock()
		c.releaseComposition(r)
		c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.ErrorOccurred}, State: domain.Failed, Error: domain.CodeNoSpeech})
		return
	}
	_ = r.session.Advance(domain.Reviewing)
	r.final = text
	c.mu.Unlock()
	scene := ""
	if outcome != "" {
		scene = policy.Scene
	}
	c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.CorrectionReady, Revision: revision}, State: domain.Reviewing, Raw: raw, Source: source, Rewrite: outcome, Scene: scene, Text: text})
	if c.opts.AutoCommit {
		c.commit(r, revision, text)
	}
}

// CommitReviewed commits the reviewed text of the current session once, for
// review-before-insert workflows.
func (c *Coordinator) CommitReviewed() {
	c.mu.Lock()
	r := c.current
	if r == nil || r.session.State() != domain.Reviewing || !r.hasTarget {
		c.mu.Unlock()
		return
	}
	revision, text := r.session.Revision(), r.final
	c.mu.Unlock()
	c.commit(r, revision, text)
}

func (c *Coordinator) commit(r *run, revision uint64, text string) {
	c.mu.Lock()
	if c.current != r || r.session.ClaimCommit(revision) != nil {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	c.emitCurrentState(r, domain.Committing)
	_, input := c.targets(r)
	follower := c.follower(r, input)
	started := time.Now()
	result, err := input.Commit(r.ctx, contracts.CommitRequest{
		Identity: r.identity, Target: r.target.Target, SourceRevision: revision, Text: text})
	c.diagnostic(r, ports.DiagnosticCommit, started, err)
	if err != nil {
		result = contracts.CommitResult{Identity: r.identity, Outcome: contracts.NotApplied, Failure: codeOr(err, domain.CodeTargetUnsupported)}
	}
	c.mu.Lock()
	if c.current != r || r.ctx.Err() != nil || r.session.State() != domain.Committing {
		c.mu.Unlock()
		if follower != nil {
			follower.Release(r.target.Target)
		}
		return
	}
	if follower != nil {
		if result.Outcome == contracts.NotApplied {
			follower.Release(r.target.Target)
		} else {
			c.deps.Edits.Track(follower, r.target.Target, r.identity, text)
		}
	}
	var event contracts.Event
	switch result.Outcome {
	case contracts.Applied, contracts.Unknown:
		// CTR-A06: an unknown outcome is final; it is never retried.
		_ = r.session.Advance(domain.Completed)
		if result.Outcome == contracts.Applied {
			c.remember(r, text)
		}
		event = contracts.Event{Envelope: contracts.Envelope{Kind: contracts.CommitResultReady, Revision: revision}, State: domain.Completed, Outcome: result.Outcome, Error: result.Failure, Text: text}
	default:
		code := result.Failure
		if code == "" {
			code = domain.CodeTargetUnsupported
		}
		_ = r.session.FailWith(code)
		event = contracts.Event{Envelope: contracts.Envelope{Kind: contracts.CommitResultReady, Revision: revision}, State: domain.Failed, Outcome: contracts.NotApplied, Error: code, Text: text}
	}
	c.mu.Unlock()
	if result.Outcome == contracts.NotApplied {
		c.releaseComposition(r)
	}
	c.emit(r, event)
}

// follower asks the commit adapter to keep the field readable after the
// commit when learning is on. Secure fields and IME compositions are never
// followed (DSK-08); the adapter refuses fields it cannot read safely.
func (c *Coordinator) follower(r *run, input ports.InputTarget) ports.FieldReader {
	if c.deps.Edits == nil || c.opts.Learning == nil || !c.opts.Learning() || r.target.Sensitive || c.composing(r) {
		return nil
	}
	reader, ok := input.(ports.FieldReader)
	if !ok || !reader.Follow(r.target.Target) {
		return nil
	}
	return reader
}

func (c *Coordinator) emitState(r *run, state domain.SessionState) {
	c.emit(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.SessionStateChanged}, State: state})
}

// emitCurrentState serializes validation with publication, so cancellation
// cannot publish first and then be overwritten by a delayed commit state.
func (c *Coordinator) emitCurrentState(r *run, state domain.SessionState) {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	c.mu.Lock()
	current := c.current == r && r.session.State() == state && r.ctx.Err() == nil
	c.mu.Unlock()
	if current {
		c.publish(r, contracts.Event{Envelope: contracts.Envelope{Kind: contracts.SessionStateChanged}, State: state})
	}
}

func (c *Coordinator) emit(r *run, event contracts.Event) {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	c.publish(r, event)
}

// publish requires emitMu, keeping per-session sequence and listener order.
func (c *Coordinator) publish(r *run, event contracts.Event) {
	event.Version = contracts.Version
	event.Identity = r.identity
	event.Sequence = r.sequence.Add(1)
	event.AtMs = float64(time.Since(r.started).Microseconds()) / 1000
	for _, listener := range c.listeners {
		if listener != nil {
			listener(event)
		}
	}
}

func codeOr(err error, fallback domain.ErrorCode) domain.ErrorCode {
	if code := domain.CodeOf(err); code != "" {
		return code
	}
	return fallback
}

// Level maps frame RMS to a perceptual 0..1 meter value.
func Level(samples []float32) float32 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	rms := math.Sqrt(sum / float64(len(samples)))
	db := 20 * math.Log10(rms+1e-9)
	return float32(math.Max(0, math.Min(1, (db+60)/50)))
}
