package bootstrap

import (
	"arion.local/core/application"
	"arion.local/core/contracts"
)

// polishNote returns the failure category to explain after a cloud rewrite
// fell back to local cleanup, or "" when nothing is to be said. Each category
// is told once per provider and model, so a bad connection does not repeat it
// on every dictation.
func (a *App) polishNote(e contracts.Event) string {
	if e.Detail == "" || (e.Rewrite != application.RewriteUnavailable && e.Rewrite != application.RewriteTimeout) {
		return ""
	}
	if a.preferences().PolishEngine != application.PolishCloud {
		return ""
	}
	if a.polishNoted[a.polishNoteKey(e.Detail)] {
		return ""
	}
	return e.Detail
}

func (a *App) polishNoteKey(detail string) string {
	return a.cloudProvider + "/" + a.preferences().CloudModel + "/" + detail
}

// polishNoteTold records that a note will have been on screen, so it is not
// repeated. It runs when the dictation completes, not when the note is
// prepared: a cancelled or failed commit never shows it.
func (a *App) polishNoteTold(detail string) {
	if detail == "" {
		return
	}
	if a.polishNoted == nil {
		a.polishNoted = map[string]bool{}
	}
	a.polishNoted[a.polishNoteKey(detail)] = true
}
