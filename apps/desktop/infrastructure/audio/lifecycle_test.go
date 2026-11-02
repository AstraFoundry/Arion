package audio

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain/ports"
)

type lifecycleStream struct{ stop func() error }

func (s lifecycleStream) Stop() error { return s.stop() }

type captureResult struct {
	stream ports.AudioStream
	err    error
}

func receiveCapture(t *testing.T, result <-chan captureResult) captureResult {
	t.Helper()
	select {
	case r := <-result:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("capture lifecycle did not complete")
		return captureResult{}
	}
}

// Cancellation during an uninterruptible native open must suppress frames and
// dispose its device before the next session can enter native initialization.
func TestCancelledMicrophoneOpenDrainsBeforeNextStart(t *testing.T) {
	m := NewMicrophone(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opening, finishOpen := make(chan struct{}), make(chan struct{})
	cleaning, finishCleanup := make(chan struct{}), make(chan struct{})
	var frames, stops atomic.Int32
	first := make(chan captureResult, 1)
	go func() {
		s, err := m.start(ctx, contracts.Identity{}, func(contracts.Frame) { frames.Add(1) },
			func(_ context.Context, _ contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
				close(opening)
				<-finishOpen
				sink(contracts.Frame{})
				return lifecycleStream{stop: func() error {
					stops.Add(1)
					close(cleaning)
					<-finishCleanup
					return nil
				}}, nil
			})
		first <- captureResult{s, err}
	}()
	<-opening
	cancel()
	next := make(chan captureResult, 1)
	var nextOpened atomic.Bool
	go func() {
		s, err := m.start(context.Background(), contracts.Identity{}, func(contracts.Frame) {},
			func(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error) {
				nextOpened.Store(true)
				return lifecycleStream{stop: func() error { return nil }}, nil
			})
		next <- captureResult{s, err}
	}()
	close(finishOpen)
	<-cleaning
	if nextOpened.Load() {
		t.Error("new native initialization overlapped cancelled open or cleanup")
	}
	close(finishCleanup)
	r := receiveCapture(t, first)
	if !errors.Is(r.err, context.Canceled) || r.stream != nil || frames.Load() != 0 || stops.Load() != 1 {
		t.Errorf("cancelled open: stream=%v error=%v frames=%d stops=%d", r.stream, r.err, frames.Load(), stops.Load())
	}
	r = receiveCapture(t, next)
	if r.err != nil || r.stream == nil || !nextOpened.Load() {
		t.Fatalf("next capture: %+v", r)
	}
	if err := r.stream.Stop(); err != nil {
		t.Fatal(err)
	}
}

// Waiting for a prior device operation must honor cancellation without opening
// hardware; regular Stop also holds ownership until teardown has completed.
func TestMicrophoneCancellationWhileWaitingForTeardown(t *testing.T) {
	m := NewMicrophone(nil)
	stopping, finishStop := make(chan struct{}), make(chan struct{})
	var stops atomic.Int32
	s, err := m.start(context.Background(), contracts.Identity{}, func(contracts.Frame) {},
		func(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error) {
			return lifecycleStream{stop: func() error {
				stops.Add(1)
				close(stopping)
				<-finishStop
				return nil
			}}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	stopped := make(chan struct{})
	go func() { _ = s.Stop(); close(stopped) }()
	<-stopping
	ctx, cancel := context.WithCancel(context.Background())
	queued := make(chan captureResult, 1)
	var opens atomic.Int32
	go func() {
		stream, err := m.start(ctx, contracts.Identity{}, func(contracts.Frame) {},
			func(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error) {
				opens.Add(1)
				return nil, errors.New("cancelled request reached native open")
			})
		queued <- captureResult{stream, err}
	}()
	cancel()
	r := receiveCapture(t, queued)
	close(finishStop)
	<-stopped
	if !errors.Is(r.err, context.Canceled) || r.stream != nil || opens.Load() != 0 {
		t.Fatalf("cancelled queue: %+v; native opens=%d", r, opens.Load())
	}
	if err := s.Stop(); err != nil || stops.Load() != 1 {
		t.Fatalf("repeated stop: %v; native stops=%d", err, stops.Load())
	}
}

func TestFailedMicrophoneOpenReleasesOwnership(t *testing.T) {
	m := NewMicrophone(nil)
	failure := errors.New("native initialization failed")
	_, err := m.start(context.Background(), contracts.Identity{}, nil,
		func(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error) {
			return nil, failure
		})
	if !errors.Is(err, failure) {
		t.Fatalf("native failure lost: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s, err := m.start(ctx, contracts.Identity{}, nil,
		func(context.Context, contracts.Identity, ports.FrameSink) (ports.AudioStream, error) {
			return lifecycleStream{stop: func() error { return nil }}, nil
		})
	if err != nil {
		t.Fatalf("failed open retained native ownership: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}
