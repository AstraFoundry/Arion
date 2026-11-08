// Package holdkey records native dictation, translation and history shortcuts,
// and explains why activation is unavailable.
package holdkey

import (
	platform "runtime"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// role is the activation a recorder assigns.
type role int

const (
	dictation role = iota
	secondDictation
	translation
	clipboard
)

var roles = []role{dictation, secondDictation, translation, clipboard}

// owners name native recording ownership.
var owners = map[role]string{dictation: "dictation", secondDictation: "second-dictation", translation: "translation", clipboard: "clipboard"}

// usedBy is the refusal for a key that role already holds.
var usedBy = map[role]copy.Key{
	dictation: copy.TranslateKeyConflict, secondDictation: copy.TranslateKeyConflict,
	translation: copy.HoldKeyConflict, clipboard: copy.ClipboardKeyConflict,
}

// assignment is the key role holds now; an unassigned role holds none.
func assignment(prefs runtime.Preferences, r role) runtime.HoldKey {
	switch r {
	case dictation:
		return prefs.HoldKey
	case secondDictation:
		return prefs.SecondHoldKey
	case translation:
		return prefs.TranslateKey
	}
	key, _ := runtime.ClipboardShortcutKey(platform.GOOS, prefs.Clipboard.Shortcut)
	return key
}

// Picker displays the dictation shortcut and records a replacement.
func Picker(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, dictation, false, "", "")
}

// TranslatePicker records the translation shortcut, rejecting dictation conflicts.
func TranslatePicker(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, translation, false, "", "")
}

// Setting keeps dictation feedback below its complete settings row.
func Setting(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, dictation, true, "", "")
}

// SecondSetting records an optional second dictation key, active together
// with the first, and can remove it.
func SecondSetting(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, secondDictation, true, "", "")
}

// TranslateSetting keeps translation feedback inside its collapsible settings.
func TranslateSetting(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, translation, true, "", "")
}

// PolishShortcut is the dictation shortcut labeled as polish, for onboarding.
func PolishShortcut(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, dictation, true, copy.PolishLabel, copy.HoldKeyDetail)
}

// ClipboardSetting uses the native recorder shared with dictation and translation.
func ClipboardSetting(c *ui.Context, store runtime.Store, state *PickerState) {
	pickerFor(c, store, state, clipboard, true, "", "")
}

func pickerFor(c *ui.Context, store runtime.Store, state *PickerState, r role, setting bool, title, detail copy.Key) {
	messages, tokens := store.Copy(), theme.Resolve(c.Theme().Dark)
	prefs := store.State.Preferences
	owner, current := owners[r], assignment(prefs, r)
	var label, rowDetail, edit copy.Key
	var set func(runtime.HoldKey)
	var clear func()
	switch r {
	case dictation:
		label, rowDetail, edit = copy.HoldKeyLabel, copy.HoldKeyDetail, copy.HoldKeyEdit
		set = store.Actions.SetHoldKey
	case secondDictation:
		label, rowDetail, edit = copy.SecondHoldKeyLabel, copy.SecondHoldKeyDetail, copy.SecondHoldKeyEdit
		set = store.Actions.SetSecondHoldKey
		if current != "" && current != runtime.HoldOff {
			clear = func() { store.Actions.SetSecondHoldKey(runtime.HoldOff) }
		}
	case translation:
		label, rowDetail, edit = copy.TranslateKeyLabel, copy.TranslateKeyDetail, copy.TranslateKeyEdit
		set = store.Actions.SetTranslateKey
		if !TranslationEnabled(store) {
			// Preserve the assignment displayed during the closing animation.
			current = prefs.PreviousTranslateKey
		}
	case clipboard:
		label, rowDetail, edit = copy.ClipboardShortcut, copy.ClipboardShortcutHint, copy.ClipboardShortcutEdit
		set = func(key runtime.HoldKey) { store.Actions.SetClipboardShortcut(string(key)) }
	}
	if title != "" {
		label, rowDetail = title, detail
	}
	validate := func(key runtime.HoldKey) copy.Key {
		if r == clipboard {
			if _, ok := runtime.ClipboardShortcutKey(platform.GOOS, string(key)); !ok {
				return copy.ClipboardShortcutNeedsChord
			}
		}
		for _, other := range roles {
			if other != r && runtime.HoldKeysConflict(key, assignment(prefs, other)) {
				return usedBy[other]
			}
		}
		return ""
	}
	column := ui.Column(c).Shrink(1)
	if !setting {
		column.Gap(6).AlignItems(ui.Start)
	}
	column.Children(func() {
		control := func() { shortcutField(c, store, state, owner, current, edit, set, clear, validate, setting) }
		if setting {
			kit.TipRow(c, tokens, messages.Text(label), messages.Text(rowDetail), control)
		} else {
			control()
		}
		// A refused key is the field's own error (shortcutField); below the
		// row go guidance in Muted and a failure that is not the field's.
		var message copy.Key
		failed := false
		switch {
		case state.error != "":
		case state.listening:
			message = copy.HoldKeySystemHint
		case r == clipboard && store.State.Clipboard.ShortcutFailed:
			message, failed = copy.ClipboardShortcutFailed, true
		case current == runtime.HoldFn:
			message = copy.HoldKeyFnHint
		}
		if message != "" {
			feedback := ui.Column(c)
			if setting {
				feedback.Padding(0, 16, 12, 16)
			} else {
				feedback.MaxWidth(c.Theme().Space(80))
			}
			feedback.Children(func() {
				if failed {
					kit.Notice(c, tokens, tokens.Danger, messages.Text(message), nil)
				} else {
					ui.Text(c, messages.Text(message)).FontSize(theme.Caption).TextColor(tokens.Muted)
				}
			})
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
// translation; its list scrolls within the settings window.
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
	message, action := copy.HoldKeyUnavailable, copy.Retry
	repair := store.Actions.RetryShortcutListener
	if hold.Failure == runtime.CodePermissionDenied {
		message, action = copy.HoldKeyNeedsMonitoring, copy.Allow
		repair = store.Actions.RequestInputMonitoring
	}
	ui.Row(c.Key("shortcut-listener-status")).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).Shrink(1).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.TriangleAlert).FontSize(13).TextColor(tokens.Danger)
			ui.Text(c, messages.Text(message)).FontSize(theme.Caption).TextColor(tokens.Text).SingleLine()
		})
		if kit.Button(c, tokens, kit.Outline, messages.Text(action), true).Clicked() {
			repair()
		}
	})
}
