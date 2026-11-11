//go:build darwin

package macos

/*
#include "native.h"
#include "selection.h"
*/
import "C"

import (
	"sync"
	"unsafe"

	"arion.local/core/domain"
)

// selectionReadMax is the cursor-context budget for one mouse-up read.
// The native call measures UTF-16 units against twice this value.
const selectionReadMax = 100000

var selectionWatch struct {
	sync.Mutex
	sink func()
}

//export arionSelectionMouseUp
func arionSelectionMouseUp() {
	selectionWatch.Lock()
	sink := selectionWatch.sink
	selectionWatch.Unlock()
	if sink != nil {
		sink()
	}
}

// StartSelectionWatch observes left mouse-up. sink runs on the tap thread
// and must not block or read the selection. This is not the dictation key.
func StartSelectionWatch(sink func()) error {
	selectionWatch.Lock()
	selectionWatch.sink = sink
	selectionWatch.Unlock()
	if status := C.arion_selection_watch_start(); status != C.ARION_SELECTION_OK {
		StopSelectionWatch()
		if status == C.ARION_SELECTION_DENIED {
			return domain.Fail(domain.CodePermissionDenied, "input monitoring")
		}
		return domain.Fail(domain.CodeTargetUnsupported, "selection listener unavailable")
	}
	return nil
}

// StopSelectionWatch removes the mouse-up tap.
func StopSelectionWatch() {
	C.arion_selection_watch_stop()
	selectionWatch.Lock()
	selectionWatch.sink = nil
	selectionWatch.Unlock()
}

// ReadSelectedText returns the focused field's Accessibility selection.
// A secure field returns no text. The text is not logged.
func ReadSelectedText() (string, bool, error) {
	var native C.arion_target
	switch C.arion_capture_target(&native, 0) {
	case C.ARION_TARGET_UNTRUSTED:
		return "", false, domain.Fail(domain.CodePermissionDenied, "accessibility")
	case C.ARION_TARGET_NONE:
		return "", false, nil
	case C.ARION_TARGET_OK:
	default:
		return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	defer C.arion_release(native.element)
	defer C.arion_release(native.selection)
	if native.secure_input == 1 || C.GoString(&native.subrole[0]) == "AXSecureTextField" {
		return "", true, nil
	}
	if native.element == nil {
		return "", false, nil
	}
	_, _, selected, ok := cursorContextSelection(unsafe.Pointer(native.element), selectionReadMax)
	if !ok {
		return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection unreadable")
	}
	return selected, false, nil
}
