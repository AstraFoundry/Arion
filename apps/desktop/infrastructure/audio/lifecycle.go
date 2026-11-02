package audio

import (
	"context"
	"errors"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// SetDiagnostics attaches the content-free observer used by device operations.
func (m *Microphone) SetDiagnostics(observer ports.Diagnostics) {
	m.mu.Lock()
	m.diagnostics = observer
	m.mu.Unlock()
}

func (m *Microphone) observe(id contracts.Identity, stage ports.DiagnosticStage, started time.Time, err error) {
	m.mu.Lock()
	observer := m.diagnostics
	m.mu.Unlock()
	if observer == nil {
		return
	}
	code := domain.CodeOf(err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = domain.CodeCancelled
	} else if err != nil && code == "" {
		code = domain.CodeDeviceUnavailable
	}
	observer.Observe(ports.Diagnostic{Identity: id, Stage: stage, Duration: time.Since(started), Failure: code})
}

func (m *Microphone) operation(id contracts.Identity, stage ports.DiagnosticStage, operation func() error) error {
	started := time.Now()
	err := operation()
	m.observe(id, stage, started, err)
	return err
}

// Serialize native initialization and teardown, including those from the level
// monitor. Live callbacks never take this gate. Cancellation cannot interrupt a
// native call safely; its resources must be cleaned up before another opener.
func (m *Microphone) start(ctx context.Context, id contracts.Identity, sink ports.FrameSink, open func(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error)) (ports.AudioStream, error) {
	m.nativeOnce.Do(func() { m.nativeGate = make(chan struct{}, 1) })
	started := time.Now()
	select {
	case m.nativeGate <- struct{}{}:
	case <-ctx.Done():
		m.observe(id, ports.DiagnosticAudioWait, started, ctx.Err())
		return nil, ctx.Err()
	}
	defer func() { <-m.nativeGate }()
	m.observe(id, ports.DiagnosticAudioWait, started, ctx.Err())
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stream, err := open(ctx, id, func(frame contracts.Frame) {
		if ctx.Err() == nil {
			sink(frame)
		}
	})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, m.operation(id, ports.DiagnosticAudioStop, stream.Stop))
	}
	return &serializedStream{AudioStream: stream, microphone: m, identity: id}, nil
}

type serializedStream struct {
	ports.AudioStream
	microphone *Microphone
	identity   contracts.Identity
	once       sync.Once
	err        error
}

func (s *serializedStream) Stop() error {
	s.once.Do(func() {
		started := time.Now()
		s.microphone.nativeGate <- struct{}{}
		defer func() { <-s.microphone.nativeGate }()
		s.err = s.AudioStream.Stop()
		s.microphone.observe(s.identity, ports.DiagnosticAudioStop, started, s.err)
	})
	return s.err
}
