// Package statusheader shows Arion's brand and a one-word readiness state.
package statusheader

import (
	"time"

	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Status derives the label and color shown next to the brand. Only an open
// microphone pulses; other states are a flat dot and a word.
func Status(s *runtime.State, tokens theme.Tokens) (string, ui.Color, bool) {
	switch {
	case s.Dictation.State == runtime.StateCapturing:
		return copy.StatusListening, tokens.Live, true
	case s.Permissions.Microphone != runtime.ConsentGranted || s.Permissions.Accessibility != runtime.ConsentGranted:
		return copy.StatusPermission, tokens.Warning, false
	case dictation.Active(s.Dictation):
		return copy.StatusTranscribing, tokens.Muted, false
	case model.Busy(s.Model):
		return copy.StatusPreparing, tokens.Muted, false
	case s.Model.State == runtime.ModelFailed || s.Model.State == runtime.ModelUnselected:
		return copy.StatusModelFailed, tokens.Danger, false
	}
	return copy.StatusReady, tokens.Success, false
}

// Render draws the header row.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	label, color, live := Status(store.State, tokens)
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Size(28, 28).Radius(8).Background(tokens.Accent).Justify(ui.Center).AlignItems(ui.Center).
			TextColor(tokens.AccentText).Children(func() { ui.Icon(c, icons.AudioLines).FontSize(16) })
		ui.Text(c, copy.Title).FontSize(theme.Heading).FontWeight(600)
		ui.Spacer(c)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			dot := ui.Box(c).Size(6, 6).Radius(3)
			// The dot eases between states so a change reads as one signal.
			dot.Background(dot.AnimateColor("color", color, theme.Quick))
			if live {
				dot.Opacity(0.4 + 0.6*dot.Loop("pulse", 1200*time.Millisecond, ui.Bounce(ui.EaseInOut)))
			}
			ui.Text(c, label).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.Muted).SingleLine().ContentTransition(ui.ContentTransition{})
		})
	})
}
