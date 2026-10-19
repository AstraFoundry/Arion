// Package inputmode chooses how dictation reaches the field: the shortcut
// route or the system input method (DSK-01, IME-01). Installing the input
// method belongs to the keyboard settings.
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
	mode runtime.SessionMode
}

// Choice draws the two modes. While the input method mode is chosen but its
// input source is not ready, it points to where it is installed.
func Choice(c *ui.Context, store runtime.Store, tokens theme.Tokens, state *State, imeReady bool, openKeyboard func()) {
	messages := store.Copy()
	s := store.State
	state.mode = s.Preferences.Mode
	supported := s.InputMethod != runtime.InputMethodUnsupported
	kit.Card(c, tokens).Children(func() {
		ui.RadioGroup(c.Key("mode"), func() {
			options := []struct {
				value         runtime.SessionMode
				label, detail string
			}{{runtime.ModeShortcut, messages.Text(copy.ModeShortcut), messages.Text(copy.ModeShortcutDetail)}, {runtime.ModeSystemIME, messages.Text(copy.ModeIME), messages.Text(copy.ModeIMEDetail)}}
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
		if s.Preferences.Mode == runtime.ModeSystemIME && supported && !imeReady {
			kit.Divider(c, tokens)
			kit.Row(c, tokens, messages.Text(copy.ModeIMENeedsKeyboard), "", func() {
				if kit.Button(c.Key("open-keyboard"), tokens, kit.Outline, messages.Text(copy.ModeIMEOpenKeyboard), true).Clicked() {
					openKeyboard()
				}
			})
		}
	})
}
