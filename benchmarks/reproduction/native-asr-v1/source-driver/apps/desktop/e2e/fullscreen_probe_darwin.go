//go:build arion_e2e && darwin

package e2e

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics
#include "fullscreen_probe_darwin.h"
*/
import "C"

type overlayObservation struct {
	OnActiveSpace, OnScreen, AboveFixture, OverlapsFixture bool
	FrontmostPID                                           int
	Behavior                                               uint64
}

// observeOverlay must run on the AppKit main thread.
func observeOverlay(handle uintptr, fixturePID int) overlayObservation {
	var out C.arion_overlay_observation
	C.arion_observe_overlay(C.uintptr_t(handle), C.int(fixturePID), &out)
	return overlayObservation{out.on_active_space != 0, out.on_screen != 0,
		out.above_fixture != 0, out.overlaps_fixture != 0, int(out.frontmost_pid), uint64(out.behavior)}
}
