package audio

// #include <stdlib.h>
import "C"

import (
	"context"
	"math"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"github.com/gen2brain/malgo"
)

type nativeCapture struct {
	context *malgo.AllocatedContext
	device  *malgo.Device
	uid     string
	retain  bool
}

func (n *nativeCapture) Start() error { return n.device.Start() }
func (n *nativeCapture) Stop() error {
	if err := n.device.Stop(); err != nil {
		return err
	}
	if n.retain {
		return n.device.ResetCapture()
	}
	return nil
}

func (n *nativeCapture) Close() {
	if n.device != nil {
		n.device.Uninit()
	}
	_ = n.context.Uninit()
	n.context.Free()
}

func (n *nativeCapture) selectedDevice(selected string) (malgo.DeviceID, error) {
	infos, err := n.context.Devices(malgo.Capture)
	if err != nil {
		return malgo.DeviceID{}, domain.Fail(domain.CodeDeviceUnavailable, "device enumeration")
	}
	for _, info := range infos {
		if (selected == "" && info.IsDefault != 0) || (selected != "" && info.ID.String() == selected) {
			return info.ID, nil
		}
	}
	return malgo.DeviceID{}, domain.Fail(domain.CodeDeviceUnavailable, "selected microphone missing")
}

func (n *nativeCapture) Matches(selected string) (bool, error) {
	id, err := n.selectedDevice(selected)
	return err == nil && id.String() == n.uid, err
}

func (m *Microphone) initDevice(ctx context.Context, id contracts.Identity, selected string, d *captureDevice) (captureHardware, error) {
	n := &nativeCapture{retain: m.reuseDevice}
	if err := m.operation(id, ports.DiagnosticAudioContext, func() error {
		var err error
		n.context, err = malgo.InitContext(nil, malgo.ContextConfig{}, nil)
		if err != nil {
			return domain.Fail(domain.CodeDeviceUnavailable, "audio context")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			_ = m.operation(id, ports.DiagnosticAudioStop, func() error { n.Close(); return nil })
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
	// Pin the resolved endpoint for reusable devices. Check the current system
	// default again on every resume, rather than letting an idle AudioUnit
	// reroute itself and reinitialize in the background.
	if selected != "" || m.reuseDevice {
		if err := m.operation(id, ports.DiagnosticAudioSelect, func() error {
			deviceID, err := n.selectedDevice(selected)
			if err != nil {
				return err
			}
			n.uid = deviceID.String()
			config.Capture.DeviceID = deviceID.Pointer()
			return nil
		}); err != nil {
			return nil, err
		}
		defer C.free(config.Capture.DeviceID)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, frames uint32) {
			d.data(input, frames, math.Float32frombits(m.gain.Load()))
		},
		Stop: d.stopped,
	}
	if err := m.operation(id, ports.DiagnosticAudioDevice, func() error {
		var err error
		n.device, err = malgo.InitDevice(n.context.Context, config, callbacks)
		if err != nil {
			return domain.Fail(domain.CodeDeviceUnavailable, "device init")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	ready = true
	return n, nil
}
