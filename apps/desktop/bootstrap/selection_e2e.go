//go:build arion_e2e

package bootstrap

import (
	"arion.local/desktop/infrastructure/secrets"
	"arion.local/desktop/infrastructure/vocabulary"
	"github.com/egoist/mygo"
)

// NotifySelectionMouseUp runs the production mouse-up read and waits until
// the popover decision is on the UI thread. It does not synthesize a click.
func (a *App) NotifySelectionMouseUp() { a.applySelectionRead() }

// ReopenedVocabulary reads the profile's vocabulary file again with the key
// the running app uses.
func (a *App) ReopenedVocabulary() ([]vocabulary.Entry, error) {
	key, err := secrets.Key(desktopProfile(mygo.IsDev()), "vocabulary-key")
	if err != nil {
		return nil, err
	}
	store, err := vocabulary.Open(a.vocabPath, key)
	if err != nil {
		return nil, err
	}
	return store.List()
}
