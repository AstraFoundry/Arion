//go:build !darwin && !windows

package clipboardos

// Linux has no clipboard sequence in the GTK backend MyGo uses. Polling
// treats an unchanged zero as "no new copy". Capture has no focused-field
// adapter, so history stays copy-only. This is not a paste shortcut.

var clipboardGate = make(chan struct{}, 1)

func Revision() (uint64, bool) { return 0, false }
func Sequence() uint64         { return 0 }

func TryLock() (func(), bool) {
	select {
	case clipboardGate <- struct{}{}:
		return func() { <-clipboardGate }, true
	default:
		return nil, false
	}
}

func CaptureTarget() Target { return nil }
