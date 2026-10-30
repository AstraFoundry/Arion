//go:build darwin

package macos

/*
#include "native.h"
*/
import "C"

import (
	"errors"
	"os"
	"unsafe"
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
	pid     int
	element unsafe.Pointer
	secure  bool
}

func CaptureHistoryTarget() *HistoryTarget {
	pid := int(C.arion_frontmost_pid())
	if pid <= 0 || pid == os.Getpid() {
		return nil
	}
	var secure C.int
	element := C.arion_history_focus(C.int(pid), &secure, nil)
	return &HistoryTarget{pid: pid, element: element, secure: secure != 0}
}
func (t *HistoryTarget) Close() {
	if t != nil && t.element != nil {
		C.arion_release(t.element)
		t.element = nil
	}
}
func (t *HistoryTarget) Activate() error {
	if t == nil || C.arion_history_activate(C.int(t.pid)) != C.ARION_TARGET_OK {
		return errors.New("clipboard target activation denied")
	}
	return nil
}
func (t *HistoryTarget) Ready() bool { return t != nil && int(C.arion_frontmost_pid()) == t.pid }
func (t *HistoryTarget) Check() error {
	if t == nil || t.secure || C.arion_history_check(C.int(t.pid), t.element) != 1 {
		return errors.New("clipboard target changed or secure")
	}
	return nil
}
func (t *HistoryTarget) Paste(revision uint64) error {
	if t == nil || C.arion_history_post(C.int(t.pid), t.element, C.longlong(revision)) != C.ARION_PASTE_OK {
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
	return int(x), int(y), int(w), int(h), ok
}

// NoInput never treats permission or provider uncertainty as an empty editor.
func (t *Targets) NoInput() bool { return C.arion_no_input() == 1 }
