package bootstrap

import (
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/runtime"
)

// dictationMode samples the actual OS input source when a new session begins.
// The selected route remains owned by that session until it ends.
func (a *App) dictationMode() domain.InputMode {
	if a.harness != nil {
		if mode, ok := a.harness.routeOverride.Load().(domain.InputMode); ok {
			return mode
		}
	}
	if a.ime != nil && a.ime.State() == runtime.InputMethodSelected {
		return domain.SystemIMEMode
	}
	return domain.ShortcutMode
}
