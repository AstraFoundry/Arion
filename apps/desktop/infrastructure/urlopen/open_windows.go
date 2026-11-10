//go:build windows

package urlopen

import (
	"errors"
	"syscall"
	"unsafe"
)

var procShellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

func openValidatedURL(raw string) error {
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return errors.New("selection url open failed")
	}
	target, err := syscall.UTF16PtrFromString(raw)
	if err != nil {
		return errors.New("selection url open failed")
	}
	result, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if result <= 32 {
		return errors.New("selection url open failed")
	}
	return nil
}
