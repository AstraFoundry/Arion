// Package audio implements microphone capture and fixture replay sources.
package audio

import (
	"context"
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"github.com/gen2brain/malgo"
)

// SampleRate is the mono rate Arion's recognizers declare. miniaudio's
// linear resampler converts the device's native format explicitly (SPH-01).
const SampleRate = 16000

// periodMs is the capture callback period, which bounds frame size.
const periodMs = 20

// Device names one capture endpoint. ID is empty for the system default.
type Device struct {
	ID      string
	Name    string
	Default bool
}

// Permission reports and requests OS microphone consent before any device
// is opened (DSK-04).
type Permission interface {
	MicrophoneAuthorized() bool
}

// Microphone captures from the selected device through miniaudio
// (CoreAudio on macOS, WASAPI on Windows).
type Microphone struct {
	permission  Permission
	mu          sync.Mutex
	deviceID    string
	diagnostics ports.Diagnostics
	nativeOnce  sync.Once
	nativeGate  chan struct{}
	// gain holds the float32 bits of the factor samples are scaled by; an
	// open stream picks a change up at its next callback.
	gain atomic.Uint32
}

func NewMicrophone(permission Permission) *Microphone {
	m := &Microphone{permission: permission}
	m.SetGain(0)
	return m
}

// Select chooses a device by ID; empty selects the system default.
func (m *Microphone) Select(id string) {
	m.mu.Lock()
	m.deviceID = id
	m.mu.Unlock()
}

// SetGain sets the software gain in decibels. It scales Arion's capture
// only; the OS input volume is unchanged.
func (m *Microphone) SetGain(db int) {
	m.gain.Store(math.Float32bits(application.GainFactor(db)))
}

// Devices lists capture devices.
func Devices() ([]Device, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "audio context")
	}
	defer func() { _ = ctx.Uninit(); ctx.Free() }()
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "device enumeration")
	}
	devices := make([]Device, 0, len(infos))
	for _, info := range infos {
		devices = append(devices, Device{ID: info.ID.String(), Name: info.Name(), Default: info.IsDefault != 0})
	}
	return devices, nil
}

type stream struct {
	ctx      *malgo.AllocatedContext
	device   *malgo.Device
	stopping atomic.Bool
	once     sync.Once
}

// Start opens the device. The data callback only converts and enqueues; a
// device that stops by itself (unplug, sleep) produces a discontinuity frame
// so the session ends with an explicit reason (DSK-A08).
func (m *Microphone) Start(ctx context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.permission != nil && !m.permission.MicrophoneAuthorized() {
		return nil, domain.Fail(domain.CodePermissionDenied, "microphone")
	}
	return m.start(ctx, id, sink, m.open)
}

func (m *Microphone) open(ctx context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Permission may have changed while waiting for a prior native operation.
	if m.permission != nil && !m.permission.MicrophoneAuthorized() {
		return nil, domain.Fail(domain.CodePermissionDenied, "microphone")
	}
	var native *malgo.AllocatedContext
	err := m.operation(id, ports.DiagnosticAudioContext, func() error {
		var err error
		native, err = malgo.InitContext(nil, malgo.ContextConfig{}, nil)
		if err != nil {
			return domain.Fail(domain.CodeDeviceUnavailable, "audio context")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s := &stream{ctx: native}
	ready := false
	defer func() {
		if !ready {
			_ = m.operation(id, ports.DiagnosticAudioStop, s.Stop)
		}
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config := malgo.DefaultDeviceConfig(malgo.Capture)
	config.Capture.Format = malgo.FormatF32
	config.Capture.Channels = 1
	config.SampleRate = SampleRate
	config.PeriodSizeInMilliseconds = periodMs
	config.Resampling.Algorithm = malgo.ResampleAlgorithmLinear

	m.mu.Lock()
	selected := m.deviceID
	m.mu.Unlock()
	if selected != "" {
		found := false
		err := m.operation(id, ports.DiagnosticAudioSelect, func() error {
			infos, err := native.Devices(malgo.Capture)
			if err != nil {
				return domain.Fail(domain.CodeDeviceUnavailable, "device enumeration")
			}
			for i := range infos {
				if infos[i].ID.String() == selected {
					config.Capture.DeviceID = infos[i].ID.Pointer()
					found = true
					break
				}
			}
			if !found {
				return domain.Fail(domain.CodeDeviceUnavailable, "selected microphone missing")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var sequence, offset uint64
	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, frames uint32) {
			gain := math.Float32frombits(m.gain.Load())
			samples := make([]float32, frames)
			for i := range samples {
				// Amplified samples clip at full scale, as the recognizers expect.
				samples[i] = max(-1, min(1, gain*math.Float32frombits(binary.LittleEndian.Uint32(input[i*4:]))))
			}
			sink(contracts.Frame{Identity: id, Sequence: sequence, SampleOffset: offset, SampleRate: SampleRate, Channels: 1, Samples: samples})
			sequence++
			offset += uint64(frames)
		},
		Stop: func() {
			if !s.stopping.Load() {
				sink(contracts.Frame{Identity: id, Sequence: sequence, SampleOffset: offset, SampleRate: SampleRate, Channels: 1, Discontinuity: true})
			}
		},
	}
	err = m.operation(id, ports.DiagnosticAudioDevice, func() error {
		device, err := malgo.InitDevice(native.Context, config, callbacks)
		if err != nil {
			return domain.Fail(domain.CodeDeviceUnavailable, "device init")
		}
		s.device = device
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := m.operation(id, ports.DiagnosticAudioStart, func() error {
		if err := s.device.Start(); err != nil {
			return domain.Fail(domain.CodeDeviceUnavailable, "device start")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	ready = true
	return s, nil
}

// Stop halts the device; miniaudio guarantees no data callback after it
// returns, so every delivered frame precedes Stop's return.
func (s *stream) Stop() error {
	s.once.Do(func() {
		s.stopping.Store(true)
		if s.device != nil {
			_ = s.device.Stop()
			s.device.Uninit()
		}
		_ = s.ctx.Uninit()
		s.ctx.Free()
	})
	return nil
}
