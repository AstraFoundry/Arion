//go:build darwin

package macos

/*
#include "hold.h"
*/
import "C"

import (
	"sync"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
)

// PaneInputMonitoring lists apps allowed to observe keyboard events.
const PaneInputMonitoring Pane = "x-apple.systempreferences:com.apple.preference.security?Privacy_ListenEvent"

// InputMonitoring reports whether listen-only event taps are allowed. The
// OS offers no undetermined state for it.
func (Permissions) InputMonitoring() AuthStatus {
	if C.arion_listen_access() == 1 {
		return AuthGranted
	}
	return AuthDenied
}

// RequestInputMonitoring prompts once; later calls open System Settings.
func (p Permissions) RequestInputMonitoring() {
	if C.arion_listen_request() == 0 {
		p.Open(PaneInputMonitoring)
	}
}

var hold struct {
	sync.Mutex
	interpreter *holdkey.Mac
	sink        func(holdkey.Event)
}

//export arionHoldRaw
func arionHoldRaw(eventType C.int, keycode C.longlong, flags C.ulonglong, autorepeat C.int) {
	hold.Lock()
	interpreter, sink := hold.interpreter, hold.sink
	var event holdkey.Event
	ok := false
	if interpreter != nil {
		event, ok = interpreter.Interpret(int(eventType), int64(keycode), uint64(flags), autorepeat != 0)
	}
	hold.Unlock()
	if ok {
		sink(event)
	}
}

// StartHold observes key. sink runs on the tap's thread and must not block.
func StartHold(key application.HoldKey, sink func(holdkey.Event)) error {
	interpreter, ok := holdkey.NewMac(key)
	if !ok {
		return domain.Fail(domain.CodeTargetUnsupported, "hold key not available on macOS")
	}
	hold.Lock()
	hold.interpreter, hold.sink = interpreter, sink
	hold.Unlock()
	if status := C.arion_hold_start(); status != 0 {
		StopHold()
		if status == C.ARION_HOLD_PERMISSION_DENIED {
			return domain.Fail(domain.CodePermissionDenied, "input monitoring")
		}
		return domain.Fail(domain.CodeTargetUnsupported, "dictation key listener unavailable")
	}
	return nil
}

// StopHold removes the tap.
func StopHold() {
	C.arion_hold_stop()
	hold.Lock()
	hold.interpreter, hold.sink = nil, nil
	hold.Unlock()
}
