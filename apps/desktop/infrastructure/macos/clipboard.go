//go:build darwin

package macos

/*
#include "native.h"
#include <ApplicationServices/ApplicationServices.h>
*/
import "C"

import (
	"errors"
	"os"
	"unsafe"

	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/focusclass"
)

// TryClipboard holds the same gate as dictation's paste transaction. The UI
// never waits for that worker, so it cannot deadlock main-thread providers.
func TryClipboard() (func(), bool) {
	select {
	case pasteGate <- struct{}{}:
		return func() { <-pasteGate }, true
	default:
		return nil, false
	}
}

func ClipboardRevision() (uint64, bool) {
	return uint64(C.arion_clipboard_revision()), C.arion_clipboard_excluded() != 0
}

// ClipboardSequence reads only the counter and is safe on the observer worker.
func ClipboardSequence() uint64 { return uint64(C.arion_clipboard_revision()) }

type HistoryTarget struct {
	diagnostic     diagnostics.ClipboardTargetDiagnostic
	pid            int
	element        unsafe.Pointer
	window         unsafe.Pointer
	secure         bool
	windowFallback bool
}

func CaptureHistoryTarget() *HistoryTarget {
	pid := int(C.arion_frontmost_pid())
	if pid <= 0 || pid == os.Getpid() {
		return nil
	}
	var secure, focusError, windowError C.int
	element := C.arion_history_focus(C.int(pid), &secure, nil, &focusError)
	window := C.arion_history_window(C.int(pid), &windowError)
	return &HistoryTarget{pid: pid, element: element, window: window, secure: secure != 0,
		windowFallback: element == nil && window != nil && C.arion_history_window_fallback(focusError) == 1,
		diagnostic: diagnostics.ClipboardTargetDiagnostic{
			Focus: historyQueryStatus(focusError, element != nil), Window: historyQueryStatus(windowError, window != nil),
			Verification: "not_checked", Trusted: C.arion_ax_trusted(0) == 1, Secure: secure != 0,
		}}
}
func (t *HistoryTarget) Close() {
	if t != nil && t.element != nil {
		C.arion_release(t.element)
		t.element = nil
	}
	if t != nil && t.window != nil {
		C.arion_release(t.window)
		t.window = nil
	}
}
func (t *HistoryTarget) Activate() error {
	if t == nil || C.arion_history_activate(C.int(t.pid)) != C.ARION_TARGET_OK {
		return errors.New("clipboard target activation denied")
	}
	return nil
}
func (t *HistoryTarget) Ready() bool { return t != nil && int(C.arion_frontmost_pid()) == t.pid }

func (t *HistoryTarget) usableSnapshot() bool {
	return t != nil && !t.secure && (t.element != nil || (t.windowFallback && t.window != nil))
}

func (t *HistoryTarget) Check() error {
	if t == nil {
		return errors.New("clipboard target missing")
	}
	if !t.usableSnapshot() {
		switch {
		case t.secure:
			t.diagnostic.Verification = "secure_input"
		case !t.diagnostic.Trusted:
			t.diagnostic.Verification = "permission_denied"
		default:
			t.diagnostic.Verification = "missing_focus"
		}
		return errors.New("clipboard target changed or secure")
	}
	status := C.arion_history_status(C.int(t.pid), t.element, t.window)
	t.diagnostic.Verification = historyVerificationStatus(status)
	if status != C.ARION_HISTORY_READY {
		return errors.New("clipboard target changed or secure")
	}
	return nil
}
func (t *HistoryTarget) Paste(revision uint64) error {
	if !t.usableSnapshot() || C.arion_history_post(C.int(t.pid), t.element, t.window, C.longlong(revision)) != C.ARION_PASTE_OK {
		return errors.New("clipboard paste rejected")
	}
	return nil
}

// Anchor reads geometry only; clipboard history does not read editor content.
func (t *HistoryTarget) Anchor() (int, int, int, int, bool) {
	if t == nil || t.secure || t.element == nil {
		return 0, 0, 0, 0, false
	}
	var x, y, w, h C.double
	ok := C.arion_history_anchor(t.element, &x, &y, &w, &h) == 1
	t.diagnostic.Geometry = ok
	return int(x), int(y), int(w), int(h), ok
}

// ClipboardDiagnostic returns captured metadata without further native queries.
func (t *HistoryTarget) ClipboardDiagnostic() diagnostics.ClipboardTargetDiagnostic {
	return t.diagnostic
}

func historyQueryStatus(status C.int, present bool) string {
	switch C.AXError(status) {
	case C.kAXErrorSuccess:
		if present {
			return "available"
		}
		return "empty"
	case C.kAXErrorNoValue:
		return "absent"
	case C.kAXErrorAttributeUnsupported:
		return "unsupported"
	case C.kAXErrorCannotComplete:
		return "cannot_complete"
	case C.kAXErrorAPIDisabled:
		return "api_disabled"
	default:
		return "failed"
	}
}

func historyVerificationStatus(status C.int) string {
	switch status {
	case C.ARION_HISTORY_READY:
		return "ready"
	case C.ARION_HISTORY_PERMISSION_DENIED:
		return "permission_denied"
	case C.ARION_HISTORY_SECURE_INPUT:
		return "secure_input"
	case C.ARION_HISTORY_OWNER_CHANGED:
		return "owner_changed"
	case C.ARION_HISTORY_MISSING_FOCUS:
		return "missing_focus"
	case C.ARION_HISTORY_WINDOW_UNAVAILABLE:
		return "window_unavailable"
	case C.ARION_HISTORY_WINDOW_CHANGED:
		return "window_changed"
	case C.ARION_HISTORY_FOCUS_UNAVAILABLE:
		return "focus_unavailable"
	case C.ARION_HISTORY_FOCUS_CHANGED:
		return "focus_changed"
	default:
		return "verification_failed"
	}
}

// NoInput never treats permission or provider uncertainty as an empty editor.
func (t *Targets) NoInput() bool {
	var native C.arion_target
	if C.arion_capture_target(&native, 0) != C.ARION_TARGET_OK {
		return false
	}
	C.arion_release(native.element)
	C.arion_release(native.selection)
	return focusclass.DecideMac(macObservation(native)).NoInput
}
