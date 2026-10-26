package bootstrap

import (
	"arion.local/core/domain"
	arion "arion.local/desktop/ui/shared/runtime"
)

// automaticInputMethod guarantees deferral if the source becomes selected
// during staging. Explicit installation remains a separate user action.
type automaticInputMethod interface{ UpdateAutomatically() error }

func (a *App) updateInputMethodAutomatically() {
	if _, ok := a.ime.(automaticInputMethod); !ok {
		return
	}
	if a.updateApplying() || !a.state.InputMethodUpdate || a.state.InputMethodUpdateFailed ||
		a.state.InputMethod != arion.InputMethodInstalled ||
		a.state.InputMethodInstalling || a.state.InputMethodRemoving || a.state.InputMethodActivating ||
		a.state.Hold.Holding {
		return
	}
	switch a.state.Dictation.State {
	case domain.Preparing, domain.Capturing, domain.Finalizing, domain.Reviewing, domain.Committing:
		return
	}
	_ = (actions{a}).installInputMethod(true)
}
