// Package keyboard configures Arion's own input method: installing its input
// source, the candidate bar and smart input (IME-01, KBD-03, ADR 017).
package keyboard

import (
	"fmt"
	"strconv"
	"strings"

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

// windows selects native installation and recovery guidance.
func windows(store runtime.Store) bool { return strings.HasPrefix(store.State.Platform, "windows-") }

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
		update := s.InputMethodUpdate && (s.InputMethod == runtime.InputMethodInstalled || s.InputMethod == runtime.InputMethodSelected || s.InputMethod == runtime.InputMethodPending)
		if update && !s.InputMethodUpdateFailed && s.InputMethod != runtime.InputMethodPending {
			detail = messages.Text(copy.IMEUpdateAvailable)
		}
		if s.InputMethodActivating {
			detail = messages.Text(copy.IMEActivating)
		}
		label := messages.Text(copy.IMEName)
		if windows(store) {
			label = messages.Text(copy.IMEWindowsName)
			if s.InputMethod == runtime.InputMethodAbsent {
				detail = messages.Text(copy.IMEWindowsInstallHint)
			} else if s.InputMethod == runtime.InputMethodPending && !s.InputMethodActivating {
				detail = messages.Text(copy.IMEWindowsPending)
			}
		}
		if !removal {
			label, detail = detail, ""
			if s.InputMethod == runtime.InputMethodInstalled && !update {
				label = messages.Text(copy.IMEEnabled)
			}
		}
		kit.Row(c, tokens, label, detail, func() {
			if s.InputMethodRemoving {
				kit.Button(c.Key("removing"), tokens, kit.Outline, messages.Text(copy.IMERemoving), false)
				return
			}
			if s.InputMethodInstalling {
				kit.Button(c.Key("installing"), tokens, kit.Outline, messages.Text(copy.IMEInstalling), false)
				return
			}
			if s.InputMethodActivating {
				if kit.Button(c.Key("cancel-setup"), tokens, kit.Outline, messages.Text(copy.IMECancelSetup), true).Clicked() {
					store.Actions.CancelInputMethodSetup()
				}
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
			case runtime.InputMethodDiscovering, runtime.InputMethodPending, runtime.InputMethodInstalled, runtime.InputMethodSelected:
				if update && (s.InputMethodUpdateFailed || s.InputMethod == runtime.InputMethodPending) {
					if kit.Button(c.Key("update"), tokens, kit.Primary, messages.Text(copy.IMEUpdate), true).Clicked() {
						state.Error = ""
						if store.Actions.InstallInputMethod() != nil {
							state.Error = copy.IMEFailed
						}
					}
				}
				if (s.InputMethod != runtime.InputMethodSelected || s.InputMethodError != "") && !update {
					key := copy.IMEEnable
					if s.InputMethod == runtime.InputMethodInstalled {
						key = copy.IMESelect
					}
					if kit.Button(c.Key("activate"), tokens, kit.Primary, messages.Text(key), true).Clicked() {
						state.Error = ""
						if store.Actions.ActivateInputMethod() != nil {
							state.Error = copy.IMEFailed
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
	if !s.InputMethodInstalling && !s.InputMethodActivating && !s.InputMethodRemoving && (s.InputMethod == runtime.InputMethodDiscovering || s.InputMethod == runtime.InputMethodPending || s.InputMethod == runtime.InputMethodInstalled || s.InputMethodError == copy.IMERemovePending) {
		if kit.Button(c.Key("setup"), tokens, kit.Ghost, messages.Text(copy.IMESetup), true).Clicked() {
			state.Error = ""
			if store.Actions.OpenInputMethodSettings() != nil {
				state.Error = copy.IMESettingsFailed
			}
		}
	}

	if removal && !windows(store) && (s.InputMethod == runtime.InputMethodPending || s.InputMethod == runtime.InputMethodDiscovering) && !s.InputMethodInstalling && !s.InputMethodActivating && !s.InputMethodRemoving {
		ui.Text(c, messages.Text(copy.IMESetupHint)).FontSize(theme.Caption).TextColor(tokens.Muted)
	}
	if state.Error != "" {
		kit.Notice(c, tokens, tokens.Danger, messages.Text(errorKey(store, state.Error)), nil)
	} else if s.InputMethodError != "" {
		kit.Notice(c, tokens, tokens.Danger, messages.Text(errorKey(store, s.InputMethodError)), nil)
	}
}

func errorKey(store runtime.Store, key copy.Key) copy.Key {
	if windows(store) {
		switch key {
		case copy.IMEDiscoveryFailed:
			return copy.IMEWindowsDiscoveryFailed
		case copy.IMEEnableFailed:
			return copy.IMEWindowsEnableFailed
		case copy.IMESettingsFailed:
			return copy.IMEWindowsSettingsFailed
		}
	}
	return key
}

// Candidates chooses the page size, text size, theme and highlight color of
// the candidate bar, and whether native Emoji appear among Pinyin candidates.
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
		kit.Divider(c, tokens)
		kit.Row(c, tokens, messages.Text(copy.KbdAccent), "", func() { accent(c, messages, tokens, prefs, update) })
		kit.Divider(c, tokens)
		kit.Row(c, tokens, messages.Text(copy.KbdEmojiSuggestions), messages.Text(copy.KbdEmojiSuggestionsDetail), func() {
			emoji := prefs.EmojiSuggestions
			if ui.Switch(c.Key("emoji-suggestions"), &emoji).Label(messages.Text(copy.KbdEmojiSuggestions)).Changed() {
				update(func(k *runtime.KeyboardPreferences) { k.EmojiSuggestions = emoji })
			}
		})
	})
}

// Phonetics configures spelling expansion without changing the base dictionary.
func Phonetics(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	prefs := store.State.Preferences.Keyboard
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.KbdTypoCorrection), messages.Text(copy.KbdTypoCorrectionDetail), func() {
			enabled := prefs.TypoCorrection
			if ui.Switch(c.Key("typo-correction"), &enabled).Label(messages.Text(copy.KbdTypoCorrection)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Keyboard.TypoCorrection = enabled })
			}
		})
	})
	kit.Group(c, tokens, messages.Text(copy.KbdFuzzyPinyin), func() {
		kit.Card(c, tokens).Children(func() {
			ui.Row(c).Gap(12).AlignItems(ui.Start).Children(func() {
				for _, group := range [][]fuzzyOption{fuzzyOptions[:6], fuzzyOptions[6:]} {
					ui.Column(c).Grow(1).Shrink(1).Children(func() {
						for i, pair := range group {
							if i > 0 {
								kit.Divider(c, tokens)
							}
							kit.Row(c, tokens, messages.Text(pair.label), "", func() {
								enabled := prefs.FuzzyPinyin&pair.bit != 0
								if ui.Switch(c.Key(string(pair.label)), &enabled).Label(messages.Text(pair.label)).Changed() {
									store.Actions.UpdatePreferences(func(p *runtime.Preferences) {
										if enabled {
											p.Keyboard.FuzzyPinyin |= pair.bit
										} else {
											p.Keyboard.FuzzyPinyin &^= pair.bit
										}
									})
								}
							})
						}
					})
				}
			})
		})
	})
}

type fuzzyOption struct {
	bit   uint32
	label copy.Key
}

var fuzzyOptions = []fuzzyOption{
	{runtime.FuzzyZZH, copy.KbdFuzzyZZH}, {runtime.FuzzyCCH, copy.KbdFuzzyCCH},
	{runtime.FuzzySSH, copy.KbdFuzzySSH}, {runtime.FuzzyNL, copy.KbdFuzzyNL},
	{runtime.FuzzyFH, copy.KbdFuzzyFH}, {runtime.FuzzyRL, copy.KbdFuzzyRL},
	{runtime.FuzzyANANG, copy.KbdFuzzyANANG}, {runtime.FuzzyENENG, copy.KbdFuzzyENENG},
	{runtime.FuzzyINING, copy.KbdFuzzyINING}, {runtime.FuzzyIANIANG, copy.KbdFuzzyIANIANG},
	{runtime.FuzzyUANUANG, copy.KbdFuzzyUANUANG},
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

// accentLabels name the candidate highlight colors.
var accentLabels = map[runtime.CandidateAccent]copy.Key{
	runtime.AccentYellow: copy.KbdAccentYellow, runtime.AccentOrange: copy.KbdAccentOrange, runtime.AccentRed: copy.KbdAccentRed,
	runtime.AccentPink: copy.KbdAccentPink, runtime.AccentPurple: copy.KbdAccentPurple, runtime.AccentBlue: copy.KbdAccentBlue,
	runtime.AccentGreen: copy.KbdAccentGreen, runtime.AccentGraphite: copy.KbdAccentGraphite, runtime.AccentCustom: copy.KbdAccentCustom,
}

// accent chooses the candidate highlight color, previewing each fill before
// its name. Custom starts from the current fill and adds a color well.
func accent(c *ui.Context, messages copy.Catalog, tokens theme.Tokens, prefs runtime.KeyboardPreferences, update func(func(*runtime.KeyboardPreferences))) {
	// Swatches show the fill of the appearance the candidate bar wears.
	dark := prefs.Appearance == runtime.AppearanceDark || prefs.Appearance == runtime.AppearanceSystem && tokens.Dark
	fill := func(accent runtime.CandidateAccent) string {
		choice := prefs
		choice.Accent = accent
		return choice.Highlight(dark).Fill
	}
	swatch := func(accent runtime.CandidateAccent) {
		// Custom has no fill until a color is chosen.
		color := ui.Transparent
		if accent != runtime.AccentCustom || prefs.AccentColor != "" {
			color = ui.Hex(fill(accent))
		}
		kit.Swatch(c, tokens, color)
	}
	label := func(accent runtime.CandidateAccent) string { return messages.Text(accentLabels[accent]) }
	ui.Row(c).Gap(theme.GapRelated).AlignItems(ui.Center).Children(func() {
		if prefs.Accent == runtime.AccentCustom {
			color := ui.Hex(prefs.AccentColor)
			if ui.ColorWell(c.Key("accent-color"), &color).Label(messages.Text(copy.KbdAccentCustomColor)).Changed() {
				update(func(k *runtime.KeyboardPreferences) {
					k.AccentColor = fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)
				})
			}
		}
		selected := prefs.Accent
		if kit.Select(c.Key("accent"), &selected, runtime.CandidateAccents, label, swatch).Label(messages.Text(copy.KbdAccent)).Width(140).Changed() {
			current := prefs.Highlight(dark).Fill
			update(func(k *runtime.KeyboardPreferences) {
				k.Accent = selected
				if selected == runtime.AccentCustom && k.AccentColor == "" {
					k.AccentColor = current
				}
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
	case runtime.InputMethodDiscovering:
		return copy.IMEDiscovering
	case runtime.InputMethodPending:
		return copy.IMEPending
	case runtime.InputMethodInstalled:
		return copy.IMEInstalled
	case runtime.InputMethodSelected:
		return copy.IMESelected
	}
	return copy.IMEUnsupported
}

// Neural configures only the resident local model; cloud polishing is unrelated.
func Neural(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	enabled := store.State.Preferences.Keyboard.NeuralRanking
	model := store.State.Polish
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.KbdNeuralRanking), messages.Text(copy.KbdNeuralDetail), func() {
			if ui.Switch(c.Key("neural-ranking"), &enabled).Label(messages.Text(copy.KbdNeuralRanking)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Keyboard.NeuralRanking = enabled })
			}
		})
		// A ready model needs no line: the switch already says ranking is on.
		if enabled && model.State != runtime.PolishReady {
			ui.Column(c).Padding(0, 16, 12, 16).Gap(8).Children(func() {
				key := copy.KbdNeuralUnavailable
				if model.State == runtime.PolishLoading {
					key = copy.PolishLoading
				} else if !model.Installed && model.State != runtime.PolishUnsupported {
					key = copy.KbdNeuralMissing
				}
				ui.Text(c, messages.Text(key)).FontSize(theme.Caption).TextColor(tokens.Muted)
				if !model.Installed && model.State != runtime.PolishUnsupported {
					if kit.Button(c.Key("ranking-download"), tokens, kit.Outline, messages.Text(copy.PolishDownload), model.State != runtime.PolishProvisioning).Clicked() {
						store.Actions.DownloadPolish()
					}
				}
			})
		}
	})
}
