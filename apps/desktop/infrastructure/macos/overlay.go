//go:build darwin

package macos

/*
#include "overlay.h"
*/
import "C"

// ConfigureOverlay lets a floating window join other applications' fullscreen
// Spaces without activating Arion. Call on the AppKit main thread with a live
// NSWindow handle, before showing it.
func ConfigureOverlay(handle uintptr) {
	C.arion_configure_overlay(C.uintptr_t(handle))
}

// ConfigureClipboardDialog keeps explicit confirmation free of AppKit's
// automatic window entrance/exit effects. Call before presenting the window.
func ConfigureClipboardDialog(handle uintptr) {
	C.arion_configure_clipboard_dialog(C.uintptr_t(handle))
}
