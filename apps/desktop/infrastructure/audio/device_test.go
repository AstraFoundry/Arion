package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type testHardware struct {
	device                *captureDevice
	starts, stops, closes int
	startError, stopError error
	matchError            error
	matches               bool
	onStart               func()
}

func (h *testHardware) Start() error {
	h.starts++
	if h.onStart != nil {
		h.onStart()
	}
	return h.startError
}
func (h *testHardware) Stop() error                  { h.stops++; h.device.stopped(); return h.stopError }
func (h *testHardware) Close()                       { h.closes++ }
func (h *testHardware) Matches(string) (bool, error) { return h.matches, h.matchError }

type deviceRig struct {
	m       *Microphone
	devices []*testHardware
}

func newDeviceRig(t *testing.T) *deviceRig {
	t.Helper()
	r := &deviceRig{m: NewMicrophone(nil)}
	r.m.reuseDevice = true
	t.Cleanup(r.m.Close)
	return r
}

func (r *deviceRig) initialize(_ context.Context, _ contracts.Identity, _ string, d *captureDevice) (captureHardware, error) {
	h := &testHardware{device: d, matches: true}
	r.devices = append(r.devices, h)
	return h, nil
}

func (r *deviceRig) start(ctx context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	return r.m.start(ctx, id, sink, func(ctx context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
		return r.m.openDevice(ctx, id, sink, r.initialize)
	})
}

func (r *deviceRig) mustStart(t *testing.T, id contracts.Identity, sink ports.FrameSink) ports.AudioStream {
	t.Helper()
	s, err := r.start(context.Background(), id, sink)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func testPCM(samples ...float32) []byte {
	b := make([]byte, len(samples)*4)
	for i, v := range samples {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	return b
}

func TestStoppedDeviceReuseIsolatesRecordingFrames(t *testing.T) {
	r := newDeviceRig(t)
	var first, next []contracts.Frame
	id1 := contracts.Identity{SessionID: "first", Generation: 1}
	id2 := contracts.Identity{SessionID: "next", Generation: 2}
	s := r.mustStart(t, id1, func(f contracts.Frame) { first = append(first, f) })
	d := r.devices[0].device
	d.data(testPCM(.5, -.25), 2, 1)
	d.data(testPCM(.75), 1, 1)
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if d.feed.Load() != nil || r.devices[0].closes != 0 {
		t.Fatal("idle hardware retained a session or was destroyed")
	}
	d.data(testPCM(1), 1, 1)
	d.stopped()
	n := r.mustStart(t, id2, func(f contracts.Frame) { next = append(next, f) })
	if len(r.devices) != 1 || r.devices[0].starts != 2 {
		t.Fatal("second recording reinitialized hardware")
	}
	// A delayed cleanup from the previous session cannot stop the new one.
	_ = s.Stop()
	d.data(testPCM(.75, -.75), 2, 2)
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[1].Sequence != 1 || first[1].SampleOffset != 2 || first[0].Identity != id1 {
		t.Fatalf("first session frames: %+v", first)
	}
	if len(next) != 1 || next[0].Identity != id2 || next[0].Sequence != 0 || next[0].SampleOffset != 0 || next[0].Samples[0] != 1 || next[0].Samples[1] != -1 {
		t.Fatalf("fresh session counters, routing or gain: %+v", next)
	}
	if r.devices[0].stops != 2 {
		t.Fatal("each recording must stop native capture exactly once")
	}
}

func TestDeviceCacheInvalidation(t *testing.T) {
	for _, reason := range []string{"selection", "default_changed", "reset_idle", "reset_active", "unexpected_stop", "no_reuse"} {
		t.Run(reason, func(t *testing.T) {
			r := newDeviceRig(t)
			s := r.mustStart(t, contracts.Identity{}, func(contracts.Frame) {})
			h := r.devices[0]
			switch reason {
			case "selection":
				r.m.Select("different-device")
			case "default_changed":
				h.matches = false
			case "reset_active":
				r.m.Reset()
			case "unexpected_stop":
				h.device.stopped()
			case "no_reuse":
				r.m.reuseDevice = false
			}
			if err := s.Stop(); err != nil {
				t.Fatal(err)
			}
			if reason == "reset_idle" {
				r.m.Reset()
			}
			n := r.mustStart(t, contracts.Identity{}, func(contracts.Frame) {})
			if len(r.devices) != 2 || h.closes != 1 {
				t.Fatalf("stale device reused: initialized=%d disposed=%d", len(r.devices), h.closes)
			}
			_ = n.Stop()
		})
	}
}

func TestResumeFailureDisposesWithoutRetry(t *testing.T) {
	for _, stage := range []string{"start", "selection", "stop"} {
		t.Run(stage, func(t *testing.T) {
			r := newDeviceRig(t)
			s := r.mustStart(t, contracts.Identity{}, func(contracts.Frame) {})
			h := r.devices[0]
			failure := errors.New("native failure")
			if stage == "stop" {
				h.stopError = failure
				if !errors.Is(s.Stop(), failure) {
					t.Fatal("stop failure hidden")
				}
			} else {
				_ = s.Stop()
				if stage == "start" {
					h.startError = failure
				} else {
					h.matchError = failure
				}
				n, err := r.start(context.Background(), contracts.Identity{}, func(contracts.Frame) {})
				if err == nil || n != nil {
					t.Fatal("resume failure hidden")
				}
			}
			if len(r.devices) != 1 || h.closes != 1 || r.m.idle != nil || h.device.feed.Load() != nil {
				t.Fatal("failed device retained or native failure silently retried")
			}
			_ = s.Stop()
			if h.closes != 1 {
				t.Fatal("device freed twice")
			}
		})
	}
}

func TestCancelledInitializationNeverStartsOrCachesHardware(t *testing.T) {
	r := newDeviceRig(t)
	ctx, cancel := context.WithCancel(context.Background())
	opening, finish := make(chan struct{}), make(chan struct{})
	result := make(chan captureResult, 1)
	go func() {
		s, err := r.m.start(ctx, contracts.Identity{}, func(contracts.Frame) {}, func(ctx context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
			return r.m.openDevice(ctx, id, sink, func(ctx context.Context, id contracts.Identity, selected string, d *captureDevice) (captureHardware, error) {
				close(opening)
				<-finish
				return r.initialize(ctx, id, selected, d)
			})
		})
		result <- captureResult{s, err}
	}()
	<-opening
	cancel()
	close(finish)
	out := receiveCapture(t, result)
	h := r.devices[0]
	if !errors.Is(out.err, context.Canceled) || out.stream != nil || h.starts != 0 || h.closes != 1 || r.m.idle != nil {
		t.Fatalf("cancelled initialization: %+v; starts=%d closes=%d", out, h.starts, h.closes)
	}
}

func TestCancellationDuringResumeSuppressesCallbacks(t *testing.T) {
	r := newDeviceRig(t)
	s := r.mustStart(t, contracts.Identity{}, func(contracts.Frame) {})
	_ = s.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := r.devices[0]
	h.onStart = func() { cancel(); h.device.data(testPCM(.5), 1, 1) }
	frames := 0
	n, err := r.start(ctx, contracts.Identity{}, func(contracts.Frame) { frames++ })
	if !errors.Is(err, context.Canceled) || n != nil || frames != 0 || h.stops != 2 || h.device.feed.Load() != nil {
		t.Fatalf("cancelled resume: stream=%v error=%v frames=%d stops=%d", n, err, frames, h.stops)
	}
}

func TestCloseDrainsActiveAndIdleDevicesAndRejectsNewStarts(t *testing.T) {
	r := newDeviceRig(t)
	a := r.mustStart(t, contracts.Identity{}, func(contracts.Frame) {})
	b := r.mustStart(t, contracts.Identity{}, func(contracts.Frame) {})
	_ = a.Stop()
	r.m.Close()
	_ = b.Stop()
	r.m.Close()
	for _, h := range r.devices {
		if h.stops != 1 || h.closes != 1 || h.device.feed.Load() != nil {
			t.Fatalf("close did not drain exactly once: %+v", h)
		}
	}
	if n, err := r.start(context.Background(), contracts.Identity{}, func(contracts.Frame) {}); n != nil || domain.CodeOf(err) != domain.CodeCancelled {
		t.Fatalf("closed microphone accepted a start: %v %v", n, err)
	}
}
