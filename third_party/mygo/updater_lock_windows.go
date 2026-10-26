package mygo

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	updateLockExclusive       = 0x00000002
	updateLockFailImmediately = 0x00000001
)

var updateLockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

// LockFileEx releases the byte-range lock when its file handle is closed or
// the owning process exits. Fail immediately while another updater owns it.
func lockUpdateFile(file *os.File) error {
	var overlapped syscall.Overlapped
	result, _, err := updateLockFileEx.Call(file.Fd(), updateLockExclusive|updateLockFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if result == 0 {
		return err
	}
	return nil
}
