//go:build windows

package clipboardos

import "arion.local/desktop/infrastructure/windows"

func Revision() (uint64, bool) { return windows.ClipboardRevision() }
func Sequence() uint64         { return windows.ClipboardSequence() }
func TryLock() (func(), bool)  { return windows.TryClipboard() }
func CaptureTarget() Target {
	t := windows.CaptureHistoryTarget()
	if t == nil {
		return nil
	}
	return t
}
