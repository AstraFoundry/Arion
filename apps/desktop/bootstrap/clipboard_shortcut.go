package bootstrap

import (
	"runtime"
	"strings"

	"arion.local/core/application"
)

// clipboardAccelerator maps the recorder's physical key names to the native
// global-shortcut transport. Validation happens before this translation.
func clipboardAccelerator(key application.HoldKey) string {
	parts := strings.Split(string(key), "+")
	names := map[string]string{
		"page_up": "PageUp", "page_down": "PageDown",
		"minus": "-", "equal": "=", "comma": ",", "period": ".",
		"slash": "/", "semicolon": ";", "quote": "'", "bracket_left": "[",
		"bracket_right": "]", "backslash": "\\", "grave": "`",
	}
	if name, ok := names[parts[len(parts)-1]]; ok {
		parts[len(parts)-1] = name
	}
	return strings.Join(parts, "+")
}

func (a *App) clipboardKeyConflicts(key application.HoldKey) bool {
	assigned, ok := application.ClipboardShortcutKey(runtime.GOOS, a.state.Preferences.Clipboard.Shortcut)
	return ok && application.HoldKeysConflict(key, assigned)
}
