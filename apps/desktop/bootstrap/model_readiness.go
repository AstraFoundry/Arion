package bootstrap

import (
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/modelruntime"
	arion "arion.local/desktop/ui/shared/runtime"
)

// toggleDictation keeps the same readiness rule for buttons, the tray and the
// input menu. An existing session can still be stopped while a model changes.
func (a *App) toggleDictation() {
	if !a.core.Snapshot().Active {
		if a.state.Model.State != arion.ModelReady {
			return
		}
		a.setTranslating(false)
	}
	a.core.Toggle()
}

// clearUnavailableModelFailure removes only a completed attempt made before a
// verified model was loaded. A runtime failure or retained text remains visible.
// apply also calls this for an older failure delivered after the ready update.
func (a *App) clearUnavailableModelFailure() {
	d := &a.state.Dictation
	if a.state.Model.State != arion.ModelReady || d.State != domain.Failed ||
		d.Failure != domain.CodeModelMissing || d.FailureDetail != modelruntime.DetailNotLoaded ||
		d.Draft != "" || d.Final != "" || d.Original != "" || a.core.Snapshot().Active {
		return
	}
	*d = arion.Dictation{SessionID: d.SessionID, State: domain.Idle}
}
