// Package keyboard configures Arion's own input method: installing its input
// source, the candidate bar and smart input (IME-01, KBD-03, ADR 017).
package keyboard

import (
	"strconv"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// State is the section's own view state.
type State struct {
	Error copy.Key
}

// Supported reports whether this system has Arion's input method at all.
func Supported(store runtime.Store) bool {
	return store.State.InputMethod != runtime.InputMethodUnsupported
}

// Ready reports whether the input source is installed and enabled.
func Ready(store runtime.Store) bool {
	s := store.State.InputMethod
	return s == runtime.InputMethodInstalled || s == runtime.InputMethodSelected
}

// Source shows the input source's state with its install, setup and
// removal actions.
func Source(c *ui.Context, store runtime.Store, tokens theme.Tokens, state *State) {
	source(c, store, tokens, state, true)
}

// Setup reuses installation and status controls during first-run setup.
func Setup(c *ui.Context, store runtime.Store, tokens theme.Tokens, state *State) {
	source(c, store, tokens, state, false)
}

func source(c *ui.Context, store runtime.Store, tokens theme.Tokens, state *State, removal bool) {
	messages := store.Copy()
	s := store.State
	kit.Card(c, tokens).Children(func() {
		detail := messages.Text(status(s.InputMethod))
		update := s.InputMethodUpdate && s.InputMethod != runtime.InputMethodAbsent
		if update {
			detail = messages.Text(copy.IMEUpdateAvailable)
		}
		label := messages.Text(copy.IMEName)
		if !removal {
			label, detail = detail, ""
			if s.InputMethod == runtime.InputMethodInstalled && !update {
				label = messages.Text(copy.IMEEnabled)
			}
		}
		kit.Row(c, tokens, label, detail, func() {
			if s.InputMethodInstalling {
				kit.Button(c.Key("installing"), tokens, kit.Outline, messages.Text(copy.IMEInstalling), false)
				return
			}
			switch s.InputMethod {
			case runtime.InputMethodAbsent:
				if kit.Button(c.Key("install"), tokens, kit.Outline, messages.Text(copy.IMEInstall), true).Clicked() {
					state.Error = ""
					if store.Actions.InstallInputMethod() != nil {
						state.Error = copy.IMEFailed
					}
				}
			case runtime.InputMethodPending, runtime.InputMethodInstalled, runtime.InputMethodSelected:
				// Installing over the current source replaces it with the
				// bundled build and restarts the input method.
				if update {
					if kit.Button(c.Key("update"), tokens, kit.Primary, messages.Text(copy.IMEUpdate), true).Clicked() {
						state.Error = ""
						if store.Actions.InstallInputMethod() != nil {
							state.Error = copy.IMEFailed
						}
					}
				}
				if s.InputMethod == runtime.InputMethodPending || s.InputMethod == runtime.InputMethodInstalled {
					if kit.Button(c.Key("setup"), tokens, kit.Primary, messages.Text(copy.IMESetup), true).Clicked() {
						state.Error = ""
						if store.Actions.OpenInputMethodSettings() != nil {
							state.Error = copy.IMESettingsFailed
						}
					}
				}
				if removal && kit.Button(c.Key("remove"), tokens, kit.Outline, messages.Text(copy.IMERemove), true).Clicked() {
					state.Error = ""
					if store.Actions.RemoveInputMethod() != nil {
						state.Error = copy.IMERemoveFailed
					}
				}
			}
		})
	})
	if removal && s.InputMethod == runtime.InputMethodPending && !s.InputMethodInstalling {
		ui.Text(c, messages.Text(copy.IMESetupHint)).FontSize(theme.Caption).TextColor(tokens.Muted)
	}
	if state.Error != "" {
		ui.Text(c, messages.Text(state.Error)).FontSize(theme.Caption).TextColor(tokens.Danger)
	} else if s.InputMethodError != "" {
		ui.Text(c, messages.Text(s.InputMethodError)).FontSize(theme.Caption).TextColor(tokens.Danger)
	}
}

// Candidates chooses the page size, text size and theme of the candidate bar.
func Candidates(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	prefs := store.State.Preferences.Keyboard
	update := func(change func(*runtime.KeyboardPreferences)) {
		store.Actions.UpdatePreferences(func(p *runtime.Preferences) { change(&p.Keyboard) })
	}
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.KbdPageSize), "", func() {
			options := make([]string, len(runtime.CandidatePageSizes))
			for i, size := range runtime.CandidatePageSizes {
				options[i] = strconv.Itoa(size)
			}
			selected := strconv.Itoa(prefs.PageSize)
			if ui.Select(c.Key("page-size"), &selected, options).Label(messages.Text(copy.KbdPageSize)).Width(120).Changed() {
				size, _ := strconv.Atoi(selected)
				update(func(k *runtime.KeyboardPreferences) { k.PageSize = size })
			}
		})
		kit.Divider(c, tokens)
		kit.Row(c, tokens, messages.Text(copy.KbdCandidateSize), "", func() {
			choice(c, "candidate-size", messages.Text(copy.KbdCandidateSize), prefs.CandidateSize, []option[runtime.CandidateSize]{
				{runtime.CandidateSmall, messages.Text(copy.KbdSizeSmall)}, {runtime.CandidateMedium, messages.Text(copy.KbdSizeMedium)},
				{runtime.CandidateLarge, messages.Text(copy.KbdSizeLarge)}}, func(size runtime.CandidateSize) {
				update(func(k *runtime.KeyboardPreferences) { k.CandidateSize = size })
			})
		})
		kit.Divider(c, tokens)
		kit.Row(c, tokens, messages.Text(copy.KbdAppearance), "", func() {
			choice(c, "appearance", messages.Text(copy.KbdAppearance), prefs.Appearance, []option[runtime.CandidateAppearance]{
				{runtime.AppearanceSystem, messages.Text(copy.KbdAppearanceSystem)}, {runtime.AppearanceLight, messages.Text(copy.KbdAppearanceLight)},
				{runtime.AppearanceDark, messages.Text(copy.KbdAppearanceDark)}}, func(appearance runtime.CandidateAppearance) {
				update(func(k *runtime.KeyboardPreferences) { k.Appearance = appearance })
			})
		})
	})
}

// Smart turns context ranking on or off.
func Smart(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	ranking := store.State.Preferences.Keyboard.ContextRanking
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.KbdContextRanking), messages.Text(copy.KbdContextRankingDetail), func() {
			if ui.Switch(c.Key("context-ranking"), &ranking).Label(messages.Text(copy.KbdContextRanking)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Keyboard.ContextRanking = ranking })
			}
		})
	})
}

// Keys lists the input method's fixed keys.
func Keys(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	kit.Card(c, tokens).Children(func() {
		for i, row := range [][2]copy.Key{{copy.KbdSwitchLanguage, copy.KbdSwitchLanguageKeys},
			{copy.KbdChooseKeys, copy.KbdChooseKeysValue}, {copy.KbdPageKeys, copy.KbdPageKeysValue}} {
			if i > 0 {
				kit.Divider(c, tokens)
			}
			kit.Row(c, tokens, messages.Text(row[0]), "", func() {
				ui.Text(c, messages.Text(row[1])).FontSize(theme.Body).TextColor(tokens.Muted)
			})
		}
	})
}

type option[T comparable] struct {
	value T
	label string
}

// choice is a select over labeled values.
func choice[T comparable](c *ui.Context, key, label string, current T, options []option[T], changed func(T)) {
	labels := make([]string, len(options))
	selected := ""
	for i, o := range options {
		labels[i] = o.label
		if o.value == current {
			selected = o.label
		}
	}
	if ui.Select(c.Key(key), &selected, labels).Label(label).Width(140).Changed() {
		for _, o := range options {
			if o.label == selected {
				changed(o.value)
			}
		}
	}
}

func status(s runtime.InputMethodState) copy.Key {
	switch s {
	case runtime.InputMethodAbsent:
		return copy.IMEAbsent
	case runtime.InputMethodPending:
		return copy.IMEPending
	case runtime.InputMethodInstalled:
		return copy.IMEInstalled
	case runtime.InputMethodSelected:
		return copy.IMESelected
	}
	return copy.IMEUnsupported
}
