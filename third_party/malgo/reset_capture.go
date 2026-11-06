package malgo

// #cgo darwin LDFLAGS: -framework AudioToolbox
// #include "reset_capture.h"
import "C"

// ResetCapture discards buffered input and resampler history between recording
// sessions without reopening the hardware. The caller must serialize device
// operations and stop a capture-only device before calling this method.
func (dev *Device) ResetCapture() error {
	return errorFromResult(C.malgo_reset_capture(dev.cptr()))
}
