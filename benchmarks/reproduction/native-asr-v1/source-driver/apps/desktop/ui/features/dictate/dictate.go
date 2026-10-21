// Package dictate starts and finishes dictation from a button.
package dictate

import (
	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Toggle is the main dictation button with its key. It turns into a
// live "Finish" control while the microphone is open.
func Toggle(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	d := store.State.Dictation
	live := d.State == runtime.StateCapturing
	busy := dictation.Active(d) && !live
	label := messages.Text(copy.StartDictate)
	if live {
		label = messages.Text(copy.StopDictate)
	}
	enabled := !busy && store.State.Model.State == runtime.ModelReady
	button := ui.ButtonBase(c.Key("dictate")).Height(40).Padding(0, 14).Gap(10).AlignItems(ui.Center).Radius(theme.RadiusControl).
		Disabled(!enabled)
	variant := kit.Primary
	if live {
		variant = kit.Live
	}
	kit.Style(button, tokens, variant, enabled)
	button.Children(func() {
		if live {
			kit.LevelBars(c, d.Levels, 5, 3, 16, 2, tokens.OnFill)
		} else {
			ui.Icon(c, icons.Mic).FontSize(16)
		}
		ui.Text(c, label).FontSize(theme.Body).FontWeight(600).SingleLine().ContentTransition(ui.ContentTransition{})
		ui.Spacer(c)
		kit.Kbd(c, tokens, holdkey.Legend(store.State.Preferences.HoldKey), false)
	})
	if button.Clicked() {
		store.Actions.ToggleDictation()
	}
}
