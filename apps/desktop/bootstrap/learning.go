package bootstrap

import (
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/vocabulary"
	"arion.local/desktop/ui/entities/dictation"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// learnedOverlay is the overlay's size while it shows the learned notice:
// only the pill's strip takes clicks, and only for LearnedVisible.
const (
	learnedOverlayWidth  = 600
	learnedOverlayHeight = 76
)

// learning reports whether edit learning is on; the coordinator reads it at
// each commit.
func (a *App) learning() bool { return a.preferences().LearnEdits }

// memoryStore is the vocabulary store for worker goroutines.
func (a *App) memoryStore() *vocabulary.Store {
	a.snapshot.Lock()
	defer a.snapshot.Unlock()
	return a.memory
}

// learnEdit persists the corrections of one tracked edit off the main
// thread and shows the newest learned entry. Logs carry no text (PRV-05).
func (a *App) learnEdit(report application.EditReport) {
	a.recordEntry(diagnostics.Entry{Event: "edit_learning", Reason: string(report.Settlement), Outcome: string(report.Outcome.Verdict), Count: len(report.Outcome.Pairs), Reads: report.Reads, Failures: report.Failures})
	if report.Outcome.Verdict != domain.EditLearned || !a.learning() {
		return
	}
	pairs := report.Outcome.Pairs
	a.resourceJobs.Add(1)
	a.post(func() {
		// The first learned entry creates the encrypted store.
		a.loadVocabulary(true)
		store := a.memoryStore()
		if store == nil {
			a.resourceJobs.Add(-1)
			return
		}
		go func() {
			defer a.resourceJobs.Add(-1)
			results, err := store.Learn(pairs, time.Now())
			if err != nil {
				a.record("vocabulary_save", err)
				return
			}
			a.post(func() {
				a.loadVocabulary(false)
				for i := len(results) - 1; i >= 0; i-- {
					if r := results[i]; r.Outcome == domain.LearnAdded || r.Outcome == domain.LearnReplaced {
						a.showLearned(r.Entry)
						break
					}
				}
			})
		}()
	})
}

// recalled counts the dictations each entry corrected, which keeps used
// entries from expiring or being evicted.
func (a *App) recalled(fired []domain.Replacement) {
	if store := a.memoryStore(); store != nil {
		if err := store.Recall(fired, time.Now()); err != nil {
			a.record("vocabulary_recall", err)
		}
	}
}

// showLearned displays the notice and lets the overlay's pill strip take
// the undo click until the notice expires.
func (a *App) showLearned(entry domain.MemoryEntry) {
	if dictation.Active(a.state.Dictation) {
		return
	}
	a.state.Learned = arion.Learned{ID: entry.ID, From: entry.From, To: entry.To, At: time.Now()}
	if a.overlay != nil {
		display := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
		area := display.WorkArea
		a.overlay.SetBounds(mygo.Rectangle{X: area.X + (area.Width-learnedOverlayWidth)/2,
			Y: area.Y + area.Height - learnedOverlayHeight, Width: learnedOverlayWidth, Height: learnedOverlayHeight})
		a.overlay.SetIgnoreMouseEvents(false)
		if !a.overlay.IsVisible() {
			a.overlay.ShowInactive()
		}
	}
	id := entry.ID
	time.AfterFunc(dictation.LearnedVisible, func() {
		a.post(func() {
			if a.state.Learned.ID == id {
				a.dismissLearned()
			}
		})
	})
}

// dismissLearned hides the notice and makes the overlay click-through again.
func (a *App) dismissLearned() {
	if a.state.Learned.ID == "" {
		return
	}
	a.state.Learned = arion.Learned{}
	if a.overlay != nil {
		a.overlay.SetIgnoreMouseEvents(true)
		a.positionOverlay()
	}
}
