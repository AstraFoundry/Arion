// Package audio implements microphone capture and fixture replay sources.
package audio

import (
	"context"
	"encoding/binary"
	"math"
	"sync"
	"sync/atomic"

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
	permission Permission
	mu         sync.Mutex
	deviceID   string
}

func NewMicrophone(permission Permission) *Microphone { return &Microphone{permission: permission} }

// Select chooses a device by ID; empty selects the system default.
func (m *Microphone) Select(id string) {
	m.mu.Lock()
	m.deviceID = id
	m.mu.Unlock()
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
func (m *Microphone) Start(_ context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	if m.permission != nil && !m.permission.MicrophoneAuthorized() {
		return nil, domain.Fail(domain.CodePermissionDenied, "microphone")
	}
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "audio context")
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
		infos, err := ctx.Devices(malgo.Capture)
		found := false
		if err == nil {
			for i := range infos {
				if infos[i].ID.String() == selected {
					config.Capture.DeviceID = infos[i].ID.Pointer()
					found = true
					break
				}
			}
		}
		if !found {
			_ = ctx.Uninit()
			ctx.Free()
			return nil, domain.Fail(domain.CodeDeviceUnavailable, "selected microphone missing")
		}
	}

	s := &stream{ctx: ctx}
	var sequence, offset uint64
	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, frames uint32) {
			samples := make([]float32, frames)
			for i := range samples {
				samples[i] = math.Float32frombits(binary.LittleEndian.Uint32(input[i*4:]))
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
	device, err := malgo.InitDevice(ctx.Context, config, callbacks)
	if err != nil {
		_ = ctx.Uninit()
		ctx.Free()
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "device init")
	}
	s.device = device
	if err := device.Start(); err != nil {
		device.Uninit()
		_ = ctx.Uninit()
		ctx.Free()
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "device start")
	}
	return s, nil
}

// Stop halts the device; miniaudio guarantees no data callback after it
// returns, so every delivered frame precedes Stop's return.
func (s *stream) Stop() error {
	s.once.Do(func() {
		s.stopping.Store(true)
		_ = s.device.Stop()
		s.device.Uninit()
		_ = s.ctx.Uninit()
		s.ctx.Free()
	})
	return nil
}
