//go:build windows

package raster

import (
	"testing"
	"unsafe"
)

// memoryBasicInformation is MEMORY_BASIC_INFORMATION.
type memoryBasicInformation struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	PartitionId       uint16
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
}

// TestAllocPixelsOneReservation checks that a frame larger than a heap
// arena lies in one VirtualAlloc, which SetDIBitsToDevice needs (#690).
func TestAllocPixelsOneReservation(t *testing.T) {
	procVirtualQuery := kernel32.NewProc("VirtualQuery")
	for range 3 {
		n := 4 * 1264 * 812 * 2 // past 4 MB
		p := allocPixels(n)
		if !p.mapped || len(p.b) != n {
			t.Fatalf("allocPixels(%d): mapped %v, %d bytes", n, p.mapped, len(p.b))
		}
		for i := range p.b {
			p.b[i] = byte(i)
		}
		base := uintptr(unsafe.Pointer(&p.b[0]))
		var mbi memoryBasicInformation
		if r, _, err := procVirtualQuery.Call(base, uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi)); r == 0 {
			t.Fatalf("VirtualQuery: %v", err)
		}
		if mbi.AllocationBase != base || mbi.RegionSize < uintptr(n) {
			t.Fatalf("frame at %#x, %d bytes: region from %#x of %d bytes", base, n, mbi.AllocationBase, mbi.RegionSize)
		}
		p.free()
		if p.b != nil {
			t.Fatal("free kept the slice")
		}
	}
	var nilp *pixels
	nilp.free()
}
