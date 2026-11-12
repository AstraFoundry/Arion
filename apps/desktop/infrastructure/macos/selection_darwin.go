//go:build darwin

package macos

/*
#include <stdlib.h>
#include "native.h"
#include "selection.h"
*/
import "C"

import (
	"os"
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

// selectionFocus is the frontmost app's focused element. It is read from
// that app rather than system-wide: the popover is a nonactivating panel
// that takes the keyboard for its question field, and Arion never owns the
// selection it shows. The caller releases a non-nil element.
func selectionFocus() (element unsafe.Pointer, secure bool, err error) {
	if C.arion_ax_trusted(0) != 1 {
		return nil, false, domain.Fail(domain.CodePermissionDenied, "accessibility")
	}
	pid := C.arion_frontmost_pid()
	if pid <= 0 || int(pid) == os.Getpid() {
		return nil, false, nil
	}
	var native C.int
	element = C.arion_history_focus(pid, &native, nil, nil)
	if native == 1 {
		C.arion_release(element)
		return nil, true, nil
	}
	return element, false, nil
}

// ReadSelectedText returns the focused selection: a text field's
// Accessibility selection, otherwise the selected page text of the web
// area around the focus. A secure field returns no text. The text is not
// logged.
func ReadSelectedText() (string, bool, error) {
	element, secure, err := selectionFocus()
	if err != nil || secure || element == nil {
		return "", secure, err
	}
	defer C.arion_release(element)
	_, _, selected, field := cursorContextSelection(element, selectionReadMax)
	if field && selected != "" {
		return selected, false, nil
	}
	var text *C.char
	switch C.arion_web_selection(element, C.long(selectionReadMax), &text) {
	case C.ARION_WEB_OK:
		defer C.free(unsafe.Pointer(text))
		return C.GoString(text), false, nil
	case C.ARION_WEB_NONE:
		if field {
			return "", false, nil
		}
	}
	return "", false, domain.Fail(domain.CodeTargetUnsupported, "selection unreadable")
}

// SelectedTextAnchor is the focused selection's screen rectangle in points,
// from the text field or the web area ReadSelectedText reads. It reads
// geometry only. A secure field and a missing rectangle return ok false.
func SelectedTextAnchor() (x, y, width, height int, ok bool) {
	element, secure, err := selectionFocus()
	if err != nil || secure || element == nil {
		return 0, 0, 0, 0, false
	}
	defer C.arion_release(element)
	var left, top, w, h C.double
	if C.arion_history_anchor(element, &left, &top, &w, &h) != 1 &&
		C.arion_web_selection_bounds(element, &left, &top, &w, &h) != 1 {
		return 0, 0, 0, 0, false
	}
	return int(left), int(top), int(w), int(h), true
}
