// Package holdkey chooses the dictation key, the only global activation,
// and explains why it is not listening.
package holdkey

import (
	platform "runtime"
	"slices"

	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Picker selects the hold key among this platform's keys.
func Picker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	choices := runtime.HoldKeysFor(platform.GOOS)
	labels := make([]string, len(choices))
	selected := holdname.Name(messages, store.State.Preferences.HoldKey)
	for i, key := range choices {
		labels[i] = holdname.Name(messages, key)
	}
	if ui.Select(c.Key("hold"), &selected, labels).Changed() {
		for i, label := range labels {
			if label == selected {
				store.Actions.SetHoldKey(choices[i])
			}
		}
	}
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
	messages := store.Copy()
	labels := make([]string, len(choices))
	selected := holdname.Name(messages, store.State.Preferences.TranslateKey)
	for i, key := range choices {
		labels[i] = holdname.Name(messages, key)
	}
	if ui.Select(c.Key("translate-hold"), &selected, labels).Label(messages.Text(copy.TranslateKeyLabel)).Changed() {
		for i, label := range labels {
			if label == selected {
				store.Actions.SetTranslateKey(choices[i])
			}
		}
	}
}

// targets are the translation languages with their names.
var targets = []struct {
	language runtime.TranslationLanguage
	name     copy.Key
}{
	{runtime.TargetEnglish, copy.TranslateEnglish},
	{runtime.TargetChinese, copy.TranslateChinese},
}

// TranslateLanguagePicker selects the target language for voice translation.
func TranslateLanguagePicker(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	labels := make([]string, len(targets))
	selected := messages.Text(copy.TranslateEnglish)
	for i, target := range targets {
		labels[i] = messages.Text(target.name)
		if store.State.Preferences.TranslateLanguage == target.language {
			selected = labels[i]
		}
	}
	if ui.Select(c.Key("translate-language"), &selected, labels).Label(messages.Text(copy.TranslateLanguageLabel)).Changed() {
		for i, label := range labels {
			if label == selected {
				setLanguage(store, targets[i].language)
			}
		}
	}
}

// TranslateLanguageChoice lays the target languages out as a card of
// options, for onboarding.
func TranslateLanguageChoice(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	selected := store.State.Preferences.TranslateLanguage
	if !selected.Known() {
		selected = runtime.TargetEnglish
	}
	kit.Card(c, tokens).Children(func() {
		ui.RadioGroup(c, func() {
			for i, target := range targets {
				if i > 0 {
					kit.Divider(c, tokens)
				}
				name := messages.Text(target.name)
				row := ui.RadioBase(c.Key("translate-"+string(target.language)), &selected, target.language).
					Gap(12).Padding(12, 16).AlignItems(ui.Center).Label(name)
				if row.Hovered() && selected != target.language {
					row.Background(tokens.AccentSoft.Alpha(0.5).Over(tokens.Card))
				}
				row.Children(func() {
					ui.Text(c, name).FontSize(theme.Body).FontWeight(500).Grow(1)
					if selected == target.language {
						ui.Icon(c, icons.Check).FontSize(16).TextColor(tokens.AccentInk)
					}
				})
				if row.Changed() {
					setLanguage(store, target.language)
				}
			}
		}).Gap(0).Label(messages.Text(copy.TranslateLanguageLabel))
	})
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
