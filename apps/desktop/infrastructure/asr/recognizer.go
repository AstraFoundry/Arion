package asr

import (
	"context"
	"math"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// Streaming schedules bounded Qwen batch inference in the background. It is
// chunked preview, rather than a causal streaming acoustic model.
type Streaming struct {
	engine  *engine
	rate    int
	silence int
}

func LoadStreaming(spec Spec) (*Streaming, error) {
	return LoadStreamingContext(context.Background(), spec)
}
func LoadStreamingContext(ctx context.Context, spec Spec) (*Streaming, error) {
	e, err := load(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &Streaming{engine: e, rate: spec.SampleRate, silence: int(spec.EndpointSilence * float32(spec.SampleRate))}, nil
}
func (s *Streaming) Close() { s.engine.close() }
func (s *Streaming) NewRecognizer(ctx context.Context, request ports.RecognitionRequest) (ports.Recognizer, error) {
	ctx, cancel := context.WithCancel(ctx)
	r := &session{engine: s.engine, rate: s.rate, silence: s.silence, id: request.Identity, hotwords: append([]string(nil), request.Hotwords...), ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1), done: make(chan struct{})}
	go r.preview()
	return r, nil
}

type session struct {
	hotwords  []string
	engine    *engine
	rate      int
	silence   int
	id        contracts.Identity
	ctx       context.Context
	cancel    context.CancelFunc
	wake      chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	samples   []float32
	segments  []string
	tail      string
	decoded   int
	scheduled int
	revision  uint64
	failure   error
	compute   time.Duration
	closed    bool
	finishing bool
	quiet     int
	speech    bool
	endpoints []int
}

func (s *session) Accept(ctx context.Context, f contracts.Frame) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if int(f.SampleRate) != s.rate || f.Channels != 1 {
		return domain.Fail(domain.CodeModelUnsupported, "frame format differs from model")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.finishing {
		return domain.Fail(domain.CodeModelMissing, "recognizer closed")
	}
	if s.failure != nil {
		return s.failure
	}
	s.samples = append(s.samples, f.Samples...)
	var energy float64
	for _, sample := range f.Samples {
		energy += float64(sample) * float64(sample)
	}
	// -42 dBFS RMS and 800 ms quiet delimit stable chunks without dropping
	// onset or tail samples. The actual cut is at the middle of that pause.
	if len(f.Samples) > 0 && math.Sqrt(energy/float64(len(f.Samples))) >= 0.008 {
		s.speech, s.quiet = true, 0
	} else {
		s.quiet += len(f.Samples)
		if s.speech && s.quiet >= s.silence {
			s.endpoints = append(s.endpoints, len(s.samples)-s.quiet/2)
			s.speech = false
			select {
			case s.wake <- struct{}{}:
			default:
			}
		}
	}
	// One second of newly captured audio triggers a replaceable preview request.
	if len(s.samples)-s.scheduled >= s.rate {
		s.scheduled = len(s.samples)
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
	return nil
}
func (s *session) preview() {
	defer close(s.done)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
		}
		s.mu.Lock()
		if s.finishing || s.closed {
			s.mu.Unlock()
			return
		}
		n := len(s.samples)
		commit := n > s.rate*20 || len(s.endpoints) > 0
		if len(s.endpoints) > 0 {
			n = s.endpoints[0]
		} else if commit {
			n = len(SplitAtQuiet(s.samples, s.rate, 20)[0])
		}
		samples := append([]float32(nil), s.samples[:n]...)
		s.mu.Unlock()
		start := time.Now()
		text, err := s.engine.decode(s.ctx, samples, s.hotwords...)
		elapsed := time.Since(start)
		s.mu.Lock()
		s.compute += elapsed
		if s.ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.failure = err
			s.mu.Unlock()
			return
		}
		if commit {
			s.segments = append(s.segments, text)
			var remaining []int
			for _, endpoint := range s.endpoints {
				if endpoint > n {
					remaining = append(remaining, endpoint-n)
				}
			}
			s.endpoints = remaining
			s.samples = append([]float32(nil), s.samples[n:]...)
			s.decoded = 0
			s.scheduled = 0
			s.tail = ""
			if len(s.samples) >= s.rate {
				select {
				case s.wake <- struct{}{}:
				default:
				}
			}
		} else {
			s.tail = text
			s.decoded = n
		}
		s.revision++
		s.mu.Unlock()
	}
}
func (s *session) draft(final bool) contracts.Draft {
	return contracts.Draft{Identity: s.id, Revision: s.revision, Text: JoinSegments(append(append([]string(nil), s.segments...), s.tail)), Final: final, Segments: append([]string(nil), s.segments...), Tail: s.tail}
}
func (s *session) Partial(context.Context) (contracts.Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.draft(false), s.failure
}
func (s *session) Finish(ctx context.Context) (contracts.Draft, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return contracts.Draft{}, domain.Fail(domain.CodeModelMissing, "recognizer closed")
	}
	s.finishing = true
	s.mu.Unlock()
	// Cancel an obsolete preview, then decode every uncommitted sample. This
	// prevents capture from blocking on inference and never drops the final tail.
	s.cancel()
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure != nil {
		return contracts.Draft{}, s.failure
	}
	if s.decoded != len(s.samples) {
		var parts []string
		for _, samples := range SplitAtQuiet(s.samples, s.rate, 25) {
			start := time.Now()
			text, err := s.engine.decode(ctx, samples, s.hotwords...)
			s.compute += time.Since(start)
			if err != nil {
				return contracts.Draft{}, err
			}
			parts = append(parts, text)
		}
		s.tail = JoinSegments(parts)
	}
	s.revision++
	return s.draft(true), nil
}
func (s *session) Close() error {
	s.cancel()
	<-s.done
	s.mu.Lock()
	s.closed = true
	s.samples = nil
	s.mu.Unlock()
	return nil
}

// ComputeTime includes completed and cancelled background inference, and the
// final tail decode. It excludes time spent capturing or waiting for audio.
func (s *session) ComputeTime() time.Duration { s.mu.Lock(); defer s.mu.Unlock(); return s.compute }

// Offline exposes the same native engine for explicit evaluation callers.
type Offline struct {
	engine *engine
	rate   int
}

func LoadOffline(spec Spec) (*Offline, error) {
	e, err := load(context.Background(), spec)
	if err != nil {
		return nil, err
	}
	return &Offline{e, spec.SampleRate}, nil
}
func (o *Offline) Close() { o.engine.close() }
func (o *Offline) Finalize(ctx context.Context, r ports.FinalizeRequest) (string, error) {
	if r.SampleRate != o.rate {
		return "", domain.Fail(domain.CodeModelUnsupported, "sample rate differs from model")
	}
	var parts []string
	for _, samples := range SplitAtQuiet(r.Samples, o.rate, 25) {
		text, err := o.engine.decode(ctx, samples, r.Hotwords...)
		if err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	return JoinSegments(parts), nil
}
