//go:build windows

package windows

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	procRegisterClipboardFormat    = user32.NewProc("RegisterClipboardFormatW")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procGlobalSize                 = kernel32.NewProc("GlobalSize")
)

func historyFormat(name string) uintptr {
	s, _ := syscall.UTF16PtrFromString(name)
	id, _, _ := procRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(s)))
	return id
}

// Called while the native clipboard lock is held, after every internal write.
func markInternalClipboard() {
	id := historyFormat("application/x-arion-internal")
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, 1)
	if h != 0 {
		p, _, _ := procGlobalLock.Call(h)
		if p != 0 {
			*(*byte)(memory(p)) = 1
			procGlobalUnlock.Call(h)
			procSetClipboardData.Call(id, h)
		}
	}
	rev, _, _ := procGetClipboardSequence.Call()
	internalClipboardRevision.Store(uint64(rev))
}

func TryClipboard() (func(), bool) {
	select {
	case clipboardGate <- struct{}{}:
		return func() { <-clipboardGate }, true
	default:
		return nil, false
	}
}

func ClipboardRevision() (uint64, bool) {
	rev, _, _ := procGetClipboardSequence.Call()
	if uint64(rev) == internalClipboardRevision.Load() && rev != 0 {
		return uint64(rev), true
	}
	for _, name := range []string{"application/x-arion-internal", "ExcludeClipboardContentFromMonitorProcessing"} {
		if ok, _, _ := procIsClipboardFormatAvailable.Call(historyFormat(name)); ok != 0 {
			return uint64(rev), true
		}
	}
	id := historyFormat("CanIncludeInClipboardHistory")
	if available, _, _ := procIsClipboardFormatAvailable.Call(id); available != 0 {
		if ok, _, _ := procOpenClipboard.Call(0); ok == 0 {
			return uint64(rev), true
		}
		defer procCloseClipboard.Call()
		h, _, _ := procGetClipboardData.Call(id)
		size, _, _ := procGlobalSize.Call(h)
		if size < 4 {
			return uint64(rev), true
		}
		p, _, _ := procGlobalLock.Call(h)
		if p == 0 {
			return uint64(rev), true
		}
		value := *(*uint32)(memory(p))
		procGlobalUnlock.Call(h)
		if value == 0 {
			return uint64(rev), true
		}
	}
	return uint64(rev), false
}

func ClipboardSequence() uint64 { rev, _, _ := procGetClipboardSequence.Call(); return uint64(rev) }

type HistoryTarget struct {
	window, focus uintptr
	secure        bool
	input         *historyInput
}

func CaptureHistoryTarget() *HistoryTarget {
	w, _, _ := procGetForegroundWindow.Call()
	var pid uint32
	procGetWindowThreadProcessId.Call(w, uintptr(unsafe.Pointer(&pid)))
	if w == 0 || int(pid) == os.Getpid() {
		return nil
	}
	f := focusOf(w)
	t := &HistoryTarget{window: w, focus: f}
	t.input, _ = readHistoryInput(pid)
	if t.input != nil {
		t.secure = t.input.secure
	}
	if className(f) == "Edit" {
		style, _, _ := procGetWindowLongW.Call(f, gwlStyle)
		t.secure = t.secure || style&esPassword != 0
	}
	return t
}
func (t *HistoryTarget) Close() {
	if t != nil && t.input != nil {
		t.input.close()
		t.input = nil
	}
}
func (t *HistoryTarget) Activate() error {
	if t == nil {
		return errors.New("clipboard target missing")
	}
	if ok, _, _ := procSetForegroundWindow.Call(t.window); ok == 0 {
		return errors.New("clipboard target activation denied")
	}
	return nil
}
func (t *HistoryTarget) Ready() bool {
	w, _, _ := procGetForegroundWindow.Call()
	return t != nil && w == t.window
}
func (t *HistoryTarget) Check() error {
	if !t.Ready() || t.secure || t.input == nil || focusOf(t.window) != t.focus {
		return errors.New("clipboard target changed")
	}
	var current *historyCOM
	if !historyOK(t.input.client.call(8, uintptr(unsafe.Pointer(&current)))) || current == nil {
		return errors.New("clipboard focus unavailable")
	}
	defer current.release()
	var password int32
	if !historyOK(current.call(35, uintptr(unsafe.Pointer(&password)))) || password != 0 {
		return errors.New("clipboard input became protected")
	}
	var same int32
	if !historyOK(t.input.client.call(3, uintptr(unsafe.Pointer(t.input.element)), uintptr(unsafe.Pointer(current)), uintptr(unsafe.Pointer(&same)))) || same == 0 {
		return errors.New("clipboard input changed")
	}
	if className(t.focus) == "Edit" {
		style, _, _ := procGetWindowLongW.Call(t.focus, gwlStyle)
		if style&esPassword != 0 {
			return errors.New("clipboard target secure")
		}
	}
	return nil
}
func (t *HistoryTarget) Paste(revision uint64) error {
	if err := t.Check(); err != nil {
		return err
	}
	rev, _, _ := procGetClipboardSequence.Call()
	if uint64(rev) != revision {
		return errors.New("clipboard ownership changed")
	}
	if !send([]keyboardInput{{Type: inputKeyboard, VK: vkControl}, {Type: inputKeyboard, VK: vkV}, {Type: inputKeyboard, VK: vkV, Flags: keyeventfKeyUp}, {Type: inputKeyboard, VK: vkControl, Flags: keyeventfKeyUp}}) {
		return errors.New("clipboard paste uncertain")
	}
	return nil
}
