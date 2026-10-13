package bootstrap

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/audio"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// The E2E harness exists only when the app is launched with an explicit
// control socket (ARION_E2E_CONTROL). It replaces the microphone with
// scripted fixture audio and adds fault injection for owned-scenario tests;
// recognition, correction, target capture and insertion stay the product's.
type harness struct {
	app          *App
	audio        *scriptedAudio
	delayFinal   atomic.Bool
	release      chan struct{}
	releaseMu    sync.Mutex
	loseAck      atomic.Bool
	commits      atomic.Int32
	audioStarts  atomic.Int32
	captureLive  atomic.Bool
	lastOutcome  atomic.Value
	lastFailure  atomic.Value
	sessionState atomic.Value
}

func newHarness() *harness {
	h := &harness{audio: &scriptedAudio{}, release: make(chan struct{})}
	h.audio.h = h
	h.lastOutcome.Store("")
	h.lastFailure.Store("")
	h.sessionState.Store(string(domain.Idle))
	return h
}

// scriptedAudio is the session audio source in E2E mode: frames come from
// fixture files the driver plays, paced in real time.
type scriptedAudio struct {
	h    *harness
	mu   sync.Mutex
	sink ports.FrameSink
	id   contracts.Identity
	done chan struct{}
	stop chan struct{}
}

type scriptedStream struct{ a *scriptedAudio }

func (a *scriptedAudio) Start(_ context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sink, a.id = sink, id
	a.stop = make(chan struct{})
	a.h.audioStarts.Add(1)
	a.h.captureLive.Store(true)
	return scriptedStream{a}, nil
}

func (s scriptedStream) Stop() error {
	s.a.mu.Lock()
	stop, done := s.a.stop, s.a.done
	s.a.sink = nil
	s.a.mu.Unlock()
	if stop != nil {
		select {
		case <-stop:
		default:
			close(stop)
		}
	}
	if done != nil {
		<-done
	}
	s.a.h.captureLive.Store(false)
	return nil
}

// play replays a 16 kHz mono WAV into the active session in 20 ms frames.
func (a *scriptedAudio) play(path string) error {
	samples, err := audio.ReadWAV(path)
	if err != nil {
		return err
	}
	a.mu.Lock()
	sink, id, stop := a.sink, a.id, a.stop
	done := make(chan struct{})
	a.done = done
	a.mu.Unlock()
	if sink == nil {
		close(done)
		return domain.Fail(domain.CodeDeviceUnavailable, "no active capture")
	}
	go func() {
		defer close(done)
		chunk := audio.SampleRate / 50
		start := time.Now()
		for offset, sequence := 0, uint64(0); offset < len(samples); offset, sequence = offset+chunk, sequence+1 {
			end := min(offset+chunk, len(samples))
			select {
			case <-stop:
				return
			case <-time.After(time.Until(start.Add(time.Duration(end) * time.Second / audio.SampleRate))):
			}
			sink(contracts.Frame{Identity: id, Sequence: sequence, SampleOffset: uint64(offset), SampleRate: audio.SampleRate, Channels: 1,
				Samples: append([]float32(nil), samples[offset:end]...)})
		}
	}()
	return nil
}

// gatedFinalizer holds the final recognition result while delayFinal is on,
// ignoring cancellation, so a late result arrives after the session ended.
type gatedFinalizer struct {
	h     *harness
	inner ports.Finalizer
}

func (g gatedFinalizer) Finalize(ctx context.Context, request ports.FinalizeRequest) (string, error) {
	if g.h.delayFinal.Load() {
		g.h.releaseMu.Lock()
		release := g.h.release
		g.h.releaseMu.Unlock()
		<-release
	}
	return g.inner.Finalize(context.WithoutCancel(ctx), request)
}

// countingTargets counts insertion attempts and can lose the acknowledgement
// of a real insertion, turning its outcome into unknown.
type countingTargets struct {
	Targets
	h *harness
}

func (t countingTargets) Commit(ctx context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	t.h.commits.Add(1)
	result, err := t.Targets.Commit(ctx, request)
	if err == nil && t.h.loseAck.Load() {
		result.Outcome, result.Failure = contracts.Unknown, ""
	}
	return result, err
}

func (h *harness) observe(e contracts.Event) {
	switch e.Kind {
	case contracts.SessionStateChanged, contracts.CommitResultReady, contracts.ErrorOccurred, contracts.CorrectionReady:
		if e.State != "" {
			h.sessionState.Store(string(e.State))
		}
	}
	if e.Kind == contracts.CommitResultReady {
		h.lastOutcome.Store(string(e.Outcome))
	}
	if e.Error != "" && e.Error != domain.CodeSessionBusy {
		h.lastFailure.Store(string(e.Error))
	}
}

func (h *harness) observations() map[string]any {
	status := h.app.manager.Status()
	model := map[string]any{"state": string(status.State), "bundle": status.Bundle}
	modeValue := string(h.app.preferences().Mode)
	return map[string]any{
		"session":               map[string]any{"state": h.sessionState.Load()},
		"commit":                map[string]any{"status": h.lastOutcome.Load()},
		"error":                 map[string]any{"code": h.lastFailure.Load()},
		"commitCount":           int(h.commits.Load()),
		"insertionAttemptCount": int(h.commits.Load()),
		"capture":               map[string]any{"active": h.captureLive.Load(), "starts": int(h.audioStarts.Load())},
		"input":                 map[string]any{"mode": modeValue},
		"model":                 model,
	}
}

// serve answers one JSON command per line on a local socket owned by the
// launching driver.
func (h *harness) serve(path string) error {
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go h.client(conn)
		}
	}()
	return nil
}

type command struct {
	Op   string `json:"op"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

func (h *harness) client(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)
	for scanner.Scan() {
		var c command
		reply := map[string]any{}
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			reply["error"] = "invalid command"
			_ = encoder.Encode(reply)
			continue
		}
		var err error
		core := h.app.core
		switch c.Op {
		case "hello":
			reply["pid"] = os.Getpid()
		case "reset":
			h.commits.Store(0)
			h.audioStarts.Store(0)
			h.lastOutcome.Store("")
			h.lastFailure.Store("")
			h.loseAck.Store(false)
			h.delayFinal.Store(false)
			core.Cancel()
		case "start":
			core.Start()
		case "toggle":
			core.Toggle()
		case "stop":
			core.Stop()
		case "cancel":
			core.Cancel()
		case "commit-reviewed":
			core.CommitReviewed()
		case "play":
			err = h.audio.play(c.Path)
		case "wait-audio":
			h.audio.mu.Lock()
			done := h.audio.done
			h.audio.mu.Unlock()
			if done != nil {
				<-done
			}
		case "delay-final":
			h.releaseMu.Lock()
			h.release = make(chan struct{})
			h.releaseMu.Unlock()
			h.delayFinal.Store(true)
		case "release-final":
			h.delayFinal.Store(false)
			h.releaseMu.Lock()
			close(h.release)
			h.releaseMu.Unlock()
		case "lose-ack":
			h.loseAck.Store(true)
		case "set-mode":
			mode := domain.InputMode(c.Mode)
			done := make(chan struct{})
			h.app.post(func() {
				actions{h.app}.UpdatePreferences(func(p *arion.Preferences) { p.Mode = mode })
				close(done)
			})
			<-done
		case "state":
		case "quit":
			h.app.post(mygo.App.Quit)
		default:
			reply["error"] = "unknown op"
		}
		if err != nil {
			// Never report a failure as an empty string: uncoded errors carry
			// their message (paths of owned fixtures, not user content).
			message := string(domain.CodeOf(err))
			if message == "" {
				message = err.Error()
			}
			reply["error"] = message
		}
		reply["observations"] = h.observations()
		_ = encoder.Encode(reply)
	}
}
