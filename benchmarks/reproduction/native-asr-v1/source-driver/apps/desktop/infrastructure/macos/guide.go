//go:build darwin

package macos

/*
#include "guide.h"
*/
import "C"

import (
	"math"
	"unsafe"
)

// SettingsWindow is where System Settings shows, in global points with a
// top-left origin on the primary display.
type SettingsWindow struct {
	// Running reports a System Settings process.
	Running bool
	// Visible reports an on-screen settings window with the bounds below.
	Visible bool
	// Frontmost reports System Settings as the active application.
	Frontmost           bool
	X, Y, Width, Height int
}

// FindSettingsWindow reads the System Settings window from the window
// server. It is safe to call from any goroutine.
func FindSettingsWindow() SettingsWindow {
	var native C.arion_settings_window
	C.arion_find_settings_window(&native)
	return SettingsWindow{
		Running: native.running == 1, Visible: native.visible == 1, Frontmost: native.frontmost == 1,
		X: int(math.Round(float64(native.x))), Y: int(math.Round(float64(native.y))),
		Width: int(math.Round(float64(native.width))), Height: int(math.Round(float64(native.height))),
	}
}

// AppBundle is the path of the running .app, or "" outside a bundle.
func AppBundle() string {
	buffer := make([]byte, 4096)
	if C.arion_app_bundle((*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer))) != 1 {
		return ""
	}
	return C.GoString((*C.char)(unsafe.Pointer(&buffer[0])))
}
