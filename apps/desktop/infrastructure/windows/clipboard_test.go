//go:build windows

package windows

import (
	"syscall"
	"testing"
	"unsafe"
)

var procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")

// The transient paste text must carry the clipboard-history exclusion format.
// This writes the real clipboard and restores its previous Unicode text.
func TestTransientClipboardTextIsExcludedFromHistory(t *testing.T) {
	if ok, _, _ := procOpenClipboard.Call(0); ok == 0 {
		t.Fatal("clipboard unavailable")
	}
	previous, hadText := readClipboardText()
	written := writeClipboardText("arion transient canary") && excludeFromClipboardHistory()
	procCloseClipboard.Call()
	t.Cleanup(func() {
		if ok, _, _ := procOpenClipboard.Call(0); ok != 0 {
			procEmptyClipboard.Call()
			if hadText {
				writeClipboardText(previous)
			}
			procCloseClipboard.Call()
		}
	})
	if !written {
		t.Fatal("transient clipboard text was not written with its exclusion format")
	}
	name, err := syscall.UTF16PtrFromString(clipboardHistoryExclusion)
	if err != nil {
		t.Fatal(err)
	}
	format, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))
	if available, _, _ := procIsClipboardFormatAvailable.Call(format); format == 0 || available == 0 {
		t.Fatal("clipboard history exclusion format is missing")
	}
}
