// Package holdkey chooses the dictation key, the only global activation,
// and explains why it is not listening.
package holdkey

import (
	platform "runtime"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Picker displays the dictation shortcut and records a replacement.
func Picker(c *ui.Context, store runtime.Store, state *PickerState) {
	picker(c, store, state, false, false)
}

// TranslatePicker records the translation shortcut, rejecting dictation conflicts.
func TranslatePicker(c *ui.Context, store runtime.Store, state *PickerState) {
	picker(c, store, state, true, false)
}

// Setting keeps dictation feedback below its complete settings row.
func Setting(c *ui.Context, store runtime.Store, state *PickerState) {
	picker(c, store, state, false, true)
}

// TranslateSetting keeps translation feedback inside its collapsible settings.
func TranslateSetting(c *ui.Context, store runtime.Store, state *PickerState) {
	picker(c, store, state, true, true)
}

func picker(c *ui.Context, store runtime.Store, state *PickerState, translation, setting bool) {
	messages, tokens := store.Copy(), theme.Resolve(c.Theme().Dark)
	prefs := store.State.Preferences
	owner, current, blocked := "dictation", prefs.HoldKey, prefs.TranslateKey
	label, detail, edit, conflict := copy.HoldKeyLabel, copy.HoldKeyDetail, copy.HoldKeyEdit, copy.HoldKeyConflict
	set := store.Actions.SetHoldKey
	if translation {
		owner, current, blocked = "translation", prefs.TranslateKey, prefs.HoldKey
		label, detail, edit, conflict = copy.TranslateKeyLabel, copy.TranslateKeyDetail, copy.TranslateKeyEdit, copy.TranslateKeyConflict
		set = store.Actions.SetTranslateKey
		if !TranslationEnabled(store) {
			// Preserve the assignment displayed during the closing animation.
			current = prefs.PreviousTranslateKey
		}
	}
	column := ui.Column(c).Shrink(1)
	if !setting {
		column.Gap(6).AlignItems(ui.Start)
	}
	column.Children(func() {
		control := func() { shortcutField(c, store, state, owner, current, edit, set, blocked, conflict) }
		if setting {
			kit.TipRow(c, tokens, messages.Text(label), messages.Text(detail), control)
		} else {
			control()
		}
		message, ink := state.error, tokens.Danger
		if message == "" {
			ink = tokens.Muted
			switch {
			case state.listening:
				message = copy.HoldKeySystemHint
			case current == runtime.HoldFn:
				message = copy.HoldKeyFnHint
			}
		}
		if message != "" {
			feedback := ui.Column(c)
			if setting {
				feedback.Padding(0, 16, 12, 16)
			} else {
				feedback.MaxWidth(c.Theme().Space(80))
			}
			feedback.Children(func() { ui.Text(c, messages.Text(message)).FontSize(theme.Caption).TextColor(ink) })
		}
	})
}

// FreeTranslatePicker exposes translation enablement and refuses dictation
// conflicts during onboarding rather than moving the key just tested.
func FreeTranslatePicker(c *ui.Context, store runtime.Store, state *PickerState) {
	ui.Column(c).Gap(12).Children(func() {
		TranslationToggle(c, store, state)
		if TranslationEnabled(store) {
			TranslatePicker(c, store, state)
		}
	})
}

// TranslationEnabled reports whether a translation shortcut is assigned.
func TranslationEnabled(store runtime.Store) bool {
	key := store.State.Preferences.TranslateKey
	return key != "" && key != runtime.HoldOff
}

// TranslationToggle keeps the assignment and target language when disabled.
func TranslationToggle(c *ui.Context, store runtime.Store, state *PickerState) {
	enabled := TranslationEnabled(store)
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, store.Copy().Text(copy.TranslateGroup)).FontSize(theme.Body).TextColor(theme.Resolve(c.Theme().Dark).Text).Grow(1)
		if ui.Switch(c.Key("translation-enabled"), &enabled).Label(store.Copy().Text(copy.TranslateEnabled)).Changed() {
			state.Close(store)
			prefs := store.State.Preferences
			prefs.EnableTranslationShortcut(platform.GOOS, enabled)
			store.Actions.SetTranslateKey(prefs.TranslateKey)
		}
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

// ListenerStatus explains a shared listening failure outside individual settings.
func ListenerStatus(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !Unavailable(store) {
		return
	}
	messages := store.Copy()
	hold := store.State.Hold
	ui.Row(c.Key("shortcut-listener-status")).Gap(10).AlignItems(ui.Center).Children(func() {
		message, action := copy.HoldKeyUnavailable, copy.Retry
		repair := store.Actions.RetryShortcutListener
		if hold.Failure == runtime.CodePermissionDenied {
			message, action = copy.HoldKeyNeedsMonitoring, copy.Allow
			repair = store.Actions.RequestInputMonitoring
		}
		ui.Text(c, messages.Text(message)).FontSize(theme.Caption).TextColor(tokens.Danger).Grow(1).Shrink(1)
		if kit.Button(c, tokens, kit.Outline, messages.Text(action), true).Clicked() {
			repair()
		}
	})
}
