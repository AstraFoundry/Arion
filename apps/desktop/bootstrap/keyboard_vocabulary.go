package bootstrap

import (
	"errors"
	"slices"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/imebridge"
	"arion.local/desktop/infrastructure/vocabulary"
	"arion.local/desktop/ui/shared/copy"
)

// The main thread owns this projection identity. An acknowledgement queued
// before another edit must not clear that edit's pending or failed notice.
type keyboardVocabularyProjection struct {
	terms   []domain.KeyboardTerm
	visible bool
}

// Platforms without vocabulary transport keep their existing input contract.
// Native capability admission still determines whether an actual peer can
// receive a snapshot; a Go interface alone does not authorize delivery.
type keyboardVocabularyReceiver interface {
	SetKeyboardVocabulary([]domain.KeyboardTerm, func(error)) error
}

func (a *App) queueKeyboardVocabulary(entries []vocabulary.Entry) {
	terms, err := domain.KeyboardTerms(vocabulary.Memory(entries))
	if err != nil {
		a.keyboardVocabulary = nil
		a.state.VocabularyKeyboardNotice = copy.VocabKeyboardFailed
		a.record("keyboard_vocabulary_projection", err)
		return
	}
	previous := a.keyboardVocabulary
	if previous != nil && slices.Equal(previous.terms, terms) {
		return
	}
	projection := &keyboardVocabularyProjection{terms: terms, visible: len(terms) > 0 || previous != nil && len(previous.terms) > 0}
	a.keyboardVocabulary = projection
	a.state.VocabularyKeyboardNotice = ""
	if projection.visible {
		a.state.VocabularyKeyboardNotice = copy.VocabKeyboardPending
	}
	receiver, ok := a.ime.(keyboardVocabularyReceiver)
	if !ok {
		a.keyboardVocabularyOutcome(projection, imebridge.ErrVocabularyRejected)
		return
	}
	err = receiver.SetKeyboardVocabulary(terms, func(err error) {
		a.record("keyboard_vocabulary_delivery", err)
		a.post(func() { a.keyboardVocabularyOutcome(projection, err) })
	})
	if err != nil {
		a.record("keyboard_vocabulary_projection", err)
		a.keyboardVocabularyOutcome(projection, err)
	}
}

func (a *App) keyboardVocabularyOutcome(projection *keyboardVocabularyProjection, err error) {
	if a.keyboardVocabulary != projection || !projection.visible {
		return
	}
	key := copy.Key("")
	if errors.Is(err, imebridge.ErrVocabularyUnavailable) {
		key = copy.VocabKeyboardPending
	} else if err != nil {
		key = copy.VocabKeyboardFailed
	}
	a.state.VocabularyKeyboardNotice = key
}
