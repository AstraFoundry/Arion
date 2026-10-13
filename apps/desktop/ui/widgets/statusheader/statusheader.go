// Package statusheader shows Arion's brand and a one-word readiness state.
package statusheader

import (
	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Status derives the label and color shown next to the brand: a flat dot
// and a word. The pulse that marks an open microphone lives on the pill.
func Status(s *runtime.State, tokens theme.Tokens) (string, ui.Color) {
	messages := copy.For(copy.Locale(s.Preferences.UILanguage))
	switch {
	case s.Dictation.State == runtime.StateCapturing:
		return messages.Text(copy.StatusListening), tokens.Live
	case s.Permissions.Microphone != runtime.ConsentGranted || s.Permissions.Accessibility != runtime.ConsentGranted:
		return messages.Text(copy.StatusPermission), tokens.Warning
	case dictation.Active(s.Dictation):
		return messages.Text(copy.StatusTranscribing), tokens.Muted
	case model.Busy(s.Model):
		return messages.Text(copy.StatusPreparing), tokens.Muted
	case s.Model.State == runtime.ModelFailed || s.Model.State == runtime.ModelUnselected:
		return messages.Text(copy.StatusModelFailed), tokens.Danger
	}
	return messages.Text(copy.StatusReady), tokens.Success
}

// Render draws the header row.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	label, color := Status(store.State, tokens)
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		kit.Mark(c, 30).Margin(-2)
		ui.Text(c, messages.Text(copy.Title)).FontSize(theme.Heading).FontWeight(600)
		ui.Spacer(c)
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			dot := ui.Box(c).Size(6, 6).Radius(3)
			// The dot eases between states so a change reads as one signal.
			dot.Background(dot.AnimateColor("color", color, theme.Quick))
			ui.Text(c, label).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.Muted).SingleLine().ContentTransition(ui.ContentTransition{})
		})
	})
}
