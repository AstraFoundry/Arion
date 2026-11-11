//go:build windows && (amd64 || arm64)

package windows

import (
	"runtime"
	"testing"
	"unsafe"
)

// TestTaskbarList checks ITaskbarList3's IID against ShObjIdl_core.h and
// that the shell's taskbar list answers to it: with a wrong IID,
// CoCreateInstance fails with E_NOINTERFACE and progress bars never show.
func TestTaskbarList(t *testing.T) {
	want := GUID{0xea1afb91, 0x9e28, 0x4b86, [8]byte{0x90, 0xe9, 0x9e, 0x9f, 0x8a, 0x5e, 0xef, 0xaf}}
	if iidITaskbarList3 != want {
		t.Fatalf("IID_ITaskbarList3 = %#v, want %#v", iidITaskbarList3, want)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, coinitApartmentThreaded)
	var tl uintptr
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidTaskbarList)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidITaskbarList3)), uintptr(unsafe.Pointer(&tl)))
	if failed(hr) || tl == 0 {
		t.Fatalf("CoCreateInstance(TaskbarList, ITaskbarList3): %#x", hr)
	}
	release(tl)
}
