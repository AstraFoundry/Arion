// Package statusheader shows Arion's brand, a one-word state when Arion is
// not simply ready, and the way to report a problem.
package statusheader

import (
	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Status derives the label and color shown next to the brand: a flat dot
// and a word. Ready says nothing the panel does not, so it shows nothing
// (ok is false). The pulse that marks an open microphone lives on the pill.
func Status(s *runtime.State, tokens theme.Tokens) (label string, color ui.Color, ok bool) {
	messages := copy.For(copy.Locale(s.Preferences.UILanguage))
	switch {
	case s.Dictation.State == runtime.StateCapturing:
		return messages.Text(copy.StatusListening), tokens.Live, true
	case s.Permissions.Microphone != runtime.ConsentGranted || s.Permissions.Accessibility != runtime.ConsentGranted:
		return messages.Text(copy.StatusPermission), tokens.Warning, true
	case dictation.Active(s.Dictation):
		return messages.Text(copy.StatusTranscribing), tokens.Muted, true
	case model.Busy(s.Model):
		return messages.Text(copy.StatusPreparing), tokens.Muted, true
	case s.Model.State == runtime.ModelFailed || s.Model.State == runtime.ModelUnselected:
		return messages.Text(copy.StatusModelFailed), tokens.Danger, true
	}
	return "", ui.Color{}, false
}

// Render draws the header row.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	label, color, shown := Status(store.State, tokens)
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		kit.Mark(c, 30).Margin(-2)
		ui.Text(c, messages.Text(copy.Title)).FontSize(theme.Heading).FontWeight(600)
		ui.Spacer(c)
		if shown {
			ui.Row(c.Key("status")).Gap(6).AlignItems(ui.Center).Transition(ui.ElementTransition{Animation: &theme.Quick, Enter: &ui.Motion{}, Exit: &ui.Motion{}}).Children(func() {
				dot := ui.Box(c).Size(6, 6).Radius(3)
				// The dot eases between states so a change reads as one signal.
				dot.Background(dot.AnimateColor("color", color, theme.Quick))
				ui.Text(c, label).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.Muted).SingleLine().ContentTransition(ui.ContentTransition{})
			})
		}
		// A problem shows up as this state, so the way to report one sits
		// beside it.
		help := kit.IconButton(c.Key("feedback"), tokens, icons.CircleHelp, messages.Text(copy.FeedbackHelp)).Size(22, 22).Margin(0, -4, 0, -2)
		if help.Clicked() {
			store.Actions.OpenFeedback()
		}
	})
}
