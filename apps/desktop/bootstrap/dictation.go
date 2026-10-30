package bootstrap

import (
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/runtime"
)

// dictationMode samples the OS input source and native input capability at start.
// The selected route remains owned by that session until it ends.
func (a *App) dictationMode() domain.InputMode {
	if a.harness != nil {
		if mode, ok := a.harness.routeOverride.Load().(domain.InputMode); ok {
			return mode
		}
	}
	selected := false
	if source, ok := a.ime.(interface{ SelectedForSession() bool }); ok {
		selected = source.SelectedForSession()
	} else if a.ime != nil {
		selected = a.ime.State() == runtime.InputMethodSelected
	}
	if selected {
		// Choose the native copy route before acquisition only on a positive
		// no-input observation. An existing IME session never changes transport.
		if input, ok := a.targets.(interface{ NoInput() bool }); ok && input.NoInput() {
			return domain.ShortcutMode
		}
		return domain.SystemIMEMode
	}
	return domain.ShortcutMode
}
