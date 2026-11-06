package audio

import (
	"context"
	"encoding/binary"
	"math"
	"sync/atomic"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// captureHardware is initialized but not necessarily running. Stop must drain
// callbacks before returning; Close also drains on a failed Start or Stop.
type captureHardware interface {
	Start() error
	Stop() error
	Close()
	Matches(string) (bool, error)
}

type captureDevice struct {
	hardware captureHardware
	revision uint64
	feed     atomic.Pointer[captureFeed]
	broken   atomic.Bool
}

// Each recording owns fresh counters and a fresh sink. No session, context or
// PCM is retained by an idle device.
type captureFeed struct {
	identity contracts.Identity
	sink     ports.FrameSink
	stopping atomic.Bool
	sequence atomic.Uint64
	offset   atomic.Uint64
}

func (d *captureDevice) data(input []byte, frames uint32, gain float32) {
	f := d.feed.Load()
	if f == nil {
		return
	}
	samples := make([]float32, frames)
	for i := range samples {
		samples[i] = max(-1, min(1, gain*math.Float32frombits(binary.LittleEndian.Uint32(input[i*4:]))))
	}
	f.sink(contracts.Frame{Identity: f.identity, Sequence: f.sequence.Add(1) - 1,
		SampleOffset: f.offset.Add(uint64(frames)) - uint64(frames), SampleRate: SampleRate,
		Channels: 1, Samples: samples})
}

func (d *captureDevice) stopped() {
	f := d.feed.Load()
	if f == nil || f.stopping.Load() {
		return
	}
	d.broken.Store(true)
	f.sink(contracts.Frame{Identity: f.identity, Sequence: f.sequence.Add(1) - 1,
		SampleOffset: f.offset.Load(), SampleRate: SampleRate, Channels: 1, Discontinuity: true})
}

func (d *captureDevice) close() {
	if f := d.feed.Load(); f != nil {
		f.stopping.Store(true)
	}
	d.hardware.Close()
	d.feed.Store(nil)
}

type deviceInitializer func(context.Context, contracts.Identity, string, *captureDevice) (captureHardware, error)

// Called under nativeGate, including all cache transitions and native calls.
func (m *Microphone) openDevice(ctx context.Context, id contracts.Identity, sink ports.FrameSink, initialize deviceInitializer) (ports.AudioStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.permission != nil && !m.permission.MicrophoneAuthorized() {
		m.discardIdle()
		return nil, domain.Fail(domain.CodePermissionDenied, "microphone")
	}
	m.mu.Lock()
	selected, revision, closed := m.deviceID, m.revision, m.closed
	m.mu.Unlock()
	if closed {
		return nil, domain.Fail(domain.CodeCancelled, "microphone closed")
	}
	d := m.idle
	m.idle = nil
	if d != nil {
		matches := false
		var err error
		if d.revision == revision && !d.broken.Load() {
			err = m.operation(id, ports.DiagnosticAudioSelect, func() error {
				matches, err = d.hardware.Matches(selected)
				return err
			})
		}
		if !matches || err != nil {
			d.close()
			d = nil
		}
		if err != nil {
			return nil, err
		}
	}
	if d == nil {
		d = &captureDevice{revision: revision}
		hardware, err := initialize(ctx, id, selected, d)
		if err != nil {
			return nil, err
		}
		d.hardware = hardware
	} else {
		m.observe(id, ports.DiagnosticAudioReuse, time.Now(), nil)
	}
	if err := ctx.Err(); err != nil {
		m.disposeDevice(id, d)
		return nil, err
	}
	m.mu.Lock()
	valid := !m.closed && m.revision == revision
	m.mu.Unlock()
	if !valid {
		m.disposeDevice(id, d)
		return nil, domain.Fail(domain.CodeCancelled, "microphone configuration changed")
	}
	d.feed.Store(&captureFeed{identity: id, sink: sink})
	if err := m.operation(id, ports.DiagnosticAudioStart, d.hardware.Start); err != nil {
		m.disposeDevice(id, d)
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "device start")
	}
	lease := &deviceLease{microphone: m, device: d}
	if m.active == nil {
		m.active = make(map[*deviceLease]struct{})
	}
	m.active[lease] = struct{}{}
	return lease, nil
}

func (m *Microphone) disposeDevice(id contracts.Identity, d *captureDevice) {
	_ = m.operation(id, ports.DiagnosticAudioStop, func() error { d.close(); return nil })
}

type deviceLease struct {
	microphone *Microphone
	device     *captureDevice
	stopped    bool // Owned by nativeGate, including Close and serializedStream.Stop.
	err        error
}

func (s *deviceLease) Stop() error {
	if s.stopped {
		return s.err
	}
	s.stopped = true
	m, d := s.microphone, s.device
	delete(m.active, s)
	d.feed.Load().stopping.Store(true)
	s.err = d.hardware.Stop()
	if s.err != nil {
		d.close()
		return s.err
	}
	d.feed.Store(nil)
	m.mu.Lock()
	reuse := m.reuseDevice && !m.closed && m.revision == d.revision
	m.mu.Unlock()
	if reuse && !d.broken.Load() && m.idle == nil {
		m.idle = d
	} else {
		d.close()
	}
	return nil
}

func (m *Microphone) discardIdle() {
	if m.idle != nil {
		m.idle.close()
		m.idle = nil
	}
}

// Reset retires stopped hardware after suspend, consent or device changes.
// Active leases from the old revision are disposed when their capture stops.
// Native work may block; hosts call this off their UI thread.
func (m *Microphone) Reset() {
	m.mu.Lock()
	m.revision++
	m.mu.Unlock()
	m.nativeOnce.Do(func() { m.nativeGate = make(chan struct{}, 1) })
	m.nativeGate <- struct{}{}
	defer func() { <-m.nativeGate }()
	m.discardIdle()
}

// Close prevents further starts and drains every owned device, including the
// settings monitor. A late Stop on an old lease remains harmless.
func (m *Microphone) Close() {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	m.nativeOnce.Do(func() { m.nativeGate = make(chan struct{}, 1) })
	m.nativeGate <- struct{}{}
	defer func() { <-m.nativeGate }()
	for lease := range m.active {
		_ = lease.Stop()
	}
	m.discardIdle()
}
