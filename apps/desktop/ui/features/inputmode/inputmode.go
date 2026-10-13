// Package inputmode chooses shortcut or system input-method dictation and
// installs Arion's input source on request (DSK-01, IME-01).
package inputmode

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// State is the choice's own view state.
type State struct {
	mode  runtime.SessionMode
	Error string
}

// Choice draws the two modes and, for the input method, its installation.
func Choice(c *ui.Context, store runtime.Store, tokens theme.Tokens, state *State) {
	s := store.State
	state.mode = s.Preferences.Mode
	supported := s.InputMethod != runtime.InputMethodUnsupported
	kit.Card(c, tokens).Children(func() {
		ui.RadioGroup(c.Key("mode"), func() {
			options := []struct {
				value         runtime.SessionMode
				label, detail string
			}{{runtime.ModeShortcut, copy.ModeShortcut, copy.ModeShortcutDetail}, {runtime.ModeSystemIME, copy.ModeIME, copy.ModeIMEDetail}}
			for i, option := range options {
				if i > 0 {
					kit.Divider(c, tokens)
				}
				value := option.value
				ui.Column(c).Padding(12, 16).Gap(3).Children(func() {
					radio := ui.Radio(c.Key(value), &state.mode, value, option.label).Disabled(value == runtime.ModeSystemIME && !supported)
					if radio.Changed() {
						store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.Mode = value })
					}
					ui.Text(c, option.detail).FontSize(theme.Caption).TextColor(tokens.Muted).Margin(0, 0, 0, 24)
				})
			}
		})
		kit.Divider(c, tokens)
		kit.Row(c, tokens, copy.IMEName, status(s.InputMethod), func() {
			switch s.InputMethod {
			case runtime.InputMethodAbsent:
				if ui.Button(c.Key("install"), copy.IMEInstall).Clicked() {
					state.Error = ""
					if store.Actions.InstallInputMethod() != nil {
						state.Error = copy.IMEFailed
					}
				}
			case runtime.InputMethodInstalled, runtime.InputMethodSelected:
				if ui.Button(c.Key("remove"), copy.IMERemove).Clicked() {
					_ = store.Actions.RemoveInputMethod()
				}
			}
		})
	})
	if state.Error != "" {
		ui.Text(c, state.Error).FontSize(theme.Caption).TextColor(tokens.Danger)
	}
}

func status(s runtime.InputMethodState) string {
	switch s {
	case runtime.InputMethodAbsent:
		return copy.IMEAbsent
	case runtime.InputMethodInstalled:
		return copy.IMEInstalled
	case runtime.InputMethodSelected:
		return copy.IMESelected
	}
	return copy.IMEUnsupported
}
