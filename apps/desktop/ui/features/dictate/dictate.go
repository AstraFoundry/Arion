// Package dictate starts and finishes dictation from a button.
package dictate

import (
	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Toggle is the main dictation button with its shortcut. It turns into a
// live "Finish" control while the microphone is open.
func Toggle(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	d := store.State.Dictation
	live := d.State == runtime.StateCapturing
	busy := dictation.Active(d) && !live
	label := copy.StartDictate
	if live {
		label = copy.StopDictate
	}
	button := ui.ButtonBase(c.Key("dictate")).Height(40).Padding(0, 14).Gap(10).AlignItems(ui.Center).Radius(10).
		Disabled(busy || store.State.Model.State != runtime.ModelReady)
	face := tokens.Accent
	if live {
		face = tokens.Live
	}
	if button.Hovered() && !button.IsDisabled() {
		face = face.Mix(ui.RGB(255, 255, 255), 0.08)
	}
	button.Background(button.AnimateColor("face", face, theme.Quick)).TextColor(tokens.AccentText)
	kit.Press(button)
	if button.IsDisabled() {
		button.Opacity(0.55)
	}
	button.Children(func() {
		if live {
			kit.LevelBars(c, d.Levels, 5, 3, 16, 2, tokens.AccentText)
		} else {
			ui.Icon(c, icons.Mic).FontSize(16)
		}
		ui.Text(c, label).FontSize(theme.Body).FontWeight(600).SingleLine().ContentTransition(ui.ContentTransition{})
		ui.Spacer(c)
		ui.Row(c).Gap(3).Opacity(0.85).Children(func() {
			for _, key := range kit.Keys(store.State.Shortcut.Accelerator) {
				ui.Text(c, key).FontSize(theme.Caption).FontWeight(500)
			}
		})
	})
	if button.Clicked() {
		store.Actions.ToggleDictation()
	}
}
