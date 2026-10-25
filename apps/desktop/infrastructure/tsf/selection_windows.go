//go:build windows

package tsf

/*
#cgo CXXFLAGS: -std=c++17
#cgo LDFLAGS: -lole32 -luuid -ladvapi32
long arion_tsf_selected(int *selected);
*/
import "C"

import (
	"fmt"
	"runtime"
)

func selectedNative() (bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var selected C.int
	result := int32(C.arion_tsf_selected(&selected))
	if result < 0 {
		return false, fmt.Errorf("Windows TSF source observation failed: 0x%x", uint32(result))
	}
	return selected != 0, nil
}
