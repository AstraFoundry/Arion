package bootstrap

import (
	"arion.local/core/domain"
	arion "arion.local/desktop/ui/shared/runtime"
)

// Platform adapters enforce their native handover boundary. A stable Windows
// loader may publish a core while selected; other adapters defer until idle.
type automaticInputMethod interface{ UpdateAutomatically() error }
type selectedInputMethodUpdater interface{ CanUpdateWhileSelected() bool }

func (a *App) updateInputMethodAutomatically() {
	if _, ok := a.ime.(automaticInputMethod); !ok {
		return
	}
	available := a.state.InputMethod == arion.InputMethodInstalled
	if updater, ok := a.ime.(selectedInputMethodUpdater); ok && a.state.InputMethod == arion.InputMethodSelected {
		available = updater.CanUpdateWhileSelected()
	}
	if a.updateApplying() || !a.state.InputMethodUpdate || a.state.InputMethodUpdateFailed || !available ||
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
