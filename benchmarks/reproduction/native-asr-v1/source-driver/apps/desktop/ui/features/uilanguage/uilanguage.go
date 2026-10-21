// Package uilanguage owns the interface language preference scenario.
package uilanguage

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// Picker shows language autonyms so users can recover from an unfamiliar UI.
func Picker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	options := []string{messages.Text(copy.LanguageEnglish), messages.Text(copy.LanguageChinese)}
	selected := options[0]
	if store.State.Preferences.UILanguage == runtime.LanguageChinese {
		selected = options[1]
	}
	if ui.Select(c.Key("ui-language"), &selected, options).Label(messages.Text(copy.LanguageLabel)).Width(180).Changed() {
		language := runtime.LanguageEnglish
		if selected == options[1] {
			language = runtime.LanguageChinese
		}
		store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.UILanguage = language })
	}
}
