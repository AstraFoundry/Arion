// Package provision shows model download and verification progress.
package provision

import (
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Progress shows the active bundle's provisioning state with a retry when
// it failed. Ready shows a verified offline badge.
func Progress(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	m := store.State.Model
	ui.Column(c).Gap(8).Children(func() {
		switch m.State {
		case runtime.ModelReady:
			kit.Status(c.Key("ready"), tokens, icons.ShieldCheck, tokens.Success, messages.Text(copy.ModelVerified)).
				Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}})
		case runtime.ModelFailed:
			kit.Notice(c.Key("failed"), tokens, tokens.Danger, messages.Text(model.FailureKey(m.Failure)), func() {
				if kit.Button(c, tokens, kit.Outline, messages.Text(copy.Retry), true).Clicked() {
					store.Actions.RetryModel()
				}
			})
		case runtime.ModelUnselected:
			ui.Text(c, messages.Text(copy.ModelNoEvaluation)).FontSize(theme.Caption).TextColor(tokens.Muted)
		default:
			fraction := model.Fraction(m)
			if m.Phase != runtime.PhaseDownloading || m.State == runtime.ModelLoading {
				fraction = -1
			}
			ui.Progress(c.Key("bar"), fraction).Label(messages.Text(model.PhaseLabel(m)))
			ui.Row(c).Children(func() {
				ui.Text(c, messages.Text(model.PhaseLabel(m))).FontSize(theme.Caption).TextColor(tokens.Muted).ContentTransition(ui.ContentTransition{})
				ui.Spacer(c)
				if m.Phase == runtime.PhaseDownloading && m.State == runtime.ModelProvisioning {
					ui.Text(c, messages.Progress(m.Done, m.Total)).FontSize(theme.Caption).TextColor(tokens.Muted).FontFeatures("tnum")
				}
			})
		}
	})
}
