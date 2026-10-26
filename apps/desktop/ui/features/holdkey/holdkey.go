// Package holdkey chooses the dictation key, the only global activation,
// and explains why it is not listening.
package holdkey

import (
	platform "runtime"
	"slices"

	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func holdSelect(c *ui.Context, id string, choices []runtime.HoldKey, current runtime.HoldKey, messages copy.Catalog, label copy.Key, set func(runtime.HoldKey)) {
	labels := make([]string, len(choices))
	selected := holdname.Name(messages, current)
	for i, key := range choices {
		labels[i] = holdname.Name(messages, key)
	}
	sel := ui.Select(c.Key(id), &selected, labels)
	if label != "" {
		sel = sel.Label(messages.Text(label))
	}
	if !sel.Changed() {
		return
	}
	for i, item := range labels {
		if item == selected {
			set(choices[i])
		}
	}
}

// Picker selects the hold key among this platform's keys.
func Picker(c *ui.Context, store runtime.Store) {
	holdSelect(c, "hold", runtime.HoldKeysFor(platform.GOOS), store.State.Preferences.HoldKey, store.Copy(), "", func(key runtime.HoldKey) {
		store.Actions.SetHoldKey(key)
	})
}

// TranslatePicker selects the translation hold key, or HoldOff to disable it.
// Choosing the dictation key swaps the two.
func TranslatePicker(c *ui.Context, store runtime.Store) {
	translatePicker(c, store, runtime.TranslateKeysFor(platform.GOOS))
}

// FreeTranslatePicker is TranslatePicker without the dictation key, so the
// two cannot collide.
func FreeTranslatePicker(c *ui.Context, store runtime.Store) {
	hold := store.State.Preferences.HoldKey
	choices := slices.DeleteFunc(runtime.TranslateKeysFor(platform.GOOS), func(k runtime.HoldKey) bool { return k == hold })
	translatePicker(c, store, choices)
}

func translatePicker(c *ui.Context, store runtime.Store, choices []runtime.HoldKey) {
	holdSelect(c, "translate-hold", choices, store.State.Preferences.TranslateKey, store.Copy(), copy.TranslateKeyLabel, func(key runtime.HoldKey) {
		store.Actions.SetTranslateKey(key)
	})
}

// languages name the translation target languages.
var languages = map[runtime.TranslationLanguage]copy.Key{
	runtime.TargetEnglish: copy.TranslateEnglish, runtime.TargetChinese: copy.TranslateChinese,
	runtime.TargetChineseTraditional: copy.TranslateChineseTraditional, runtime.TargetJapanese: copy.TranslateJapanese,
	runtime.TargetKorean: copy.TranslateKorean, runtime.TargetSpanish: copy.TranslateSpanish,
	runtime.TargetFrench: copy.TranslateFrench, runtime.TargetGerman: copy.TranslateGerman,
	runtime.TargetPortuguese: copy.TranslatePortuguese, runtime.TargetItalian: copy.TranslateItalian,
	runtime.TargetRussian: copy.TranslateRussian, runtime.TargetArabic: copy.TranslateArabic,
}

// LanguageName is a translation target language's display name.
func LanguageName(messages copy.Catalog, language runtime.TranslationLanguage) string {
	return messages.Text(languages[language])
}

// TranslateLanguagePicker selects the target language for voice
// translation; its list scrolls, so it fits the onboarding window.
func TranslateLanguagePicker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	selected := store.State.Preferences.TranslateLanguage
	name := func(l runtime.TranslationLanguage) string { return LanguageName(messages, l) }
	if kit.Select(c.Key("translate-language"), &selected, runtime.TranslationLanguages, name, nil).
		Label(messages.Text(copy.TranslateLanguageLabel)).Changed() {
		setLanguage(store, selected)
	}
}

func setLanguage(store runtime.Store, language runtime.TranslationLanguage) {
	store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.TranslateLanguage = language })
}

// Unavailable reports a dictation key that is not listening.
func Unavailable(store runtime.Store) bool { return !store.State.Hold.Active }

// HasNotes reports whether Notes has anything to say.
func HasNotes(store runtime.Store) bool {
	prefs := store.State.Preferences
	return Unavailable(store) || prefs.HoldKey == runtime.HoldFn || prefs.TranslateKey == runtime.HoldFn
}

// Notes explain what the chosen key still needs.
func Notes(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	hold := store.State.Hold
	ui.Column(c).Gap(8).Children(func() {
		switch {
		case hold.Failure == runtime.CodePermissionDenied:
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, messages.Text(copy.HoldKeyNeedsMonitoring)).FontSize(theme.Caption).TextColor(tokens.Danger).Grow(1)
				if kit.Button(c.Key("monitoring"), tokens, kit.Outline, messages.Text(copy.Allow), true).Clicked() {
					store.Actions.RequestInputMonitoring()
				}
			})
		case !hold.Active:
			ui.Text(c, messages.Text(copy.HoldKeyUnavailable)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
		if store.State.Preferences.HoldKey == runtime.HoldFn || store.State.Preferences.TranslateKey == runtime.HoldFn {
			ui.Text(c, messages.Text(copy.HoldKeyFnHint)).FontSize(theme.Caption).TextColor(tokens.Muted)
		}
	})
}
