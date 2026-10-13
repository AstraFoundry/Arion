// Package provision shows model download and verification progress.
package provision

import (
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Progress shows the active bundle's provisioning state with a retry when
// it failed. Ready shows a verified offline badge.
func Progress(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	m := store.State.Model
	ui.Column(c).Gap(8).Children(func() {
		switch m.State {
		case runtime.ModelReady:
			ui.Row(c.Key("ready")).Gap(6).AlignItems(ui.Center).TextColor(tokens.Success).
				Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
				ui.Icon(c, icons.ShieldCheck).FontSize(14)
				ui.Text(c, copy.ModelVerified).FontSize(theme.Caption).FontWeight(500)
			})
		case runtime.ModelFailed:
			ui.Row(c.Key("failed")).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.TriangleAlert).FontSize(14).TextColor(tokens.Danger)
				ui.Text(c, failure(m.Failure)).FontSize(theme.Caption).Grow(1)
				if ui.Button(c, copy.Retry).Clicked() {
					store.Actions.RetryModel()
				}
			})
		case runtime.ModelUnselected:
			ui.Text(c, copy.ModelNoEvaluation).FontSize(theme.Caption).TextColor(tokens.Muted)
		default:
			fraction := model.Fraction(m)
			if m.Phase != runtime.PhaseDownloading || m.State == runtime.ModelLoading {
				fraction = -1
			}
			ui.Progress(c.Key("bar"), fraction).Label(model.PhaseLabel(m))
			ui.Row(c).Children(func() {
				ui.Text(c, model.PhaseLabel(m)).FontSize(theme.Caption).TextColor(tokens.Muted).ContentTransition(ui.ContentTransition{})
				ui.Spacer(c)
				if m.Phase == runtime.PhaseDownloading && m.State == runtime.ModelProvisioning {
					ui.Text(c, copy.Progress(m.Done, m.Total)).FontSize(theme.Caption).TextColor(tokens.Muted).FontFeatures("tnum")
				}
			})
		}
	})
}

func failure(code runtime.ErrorCode) string {
	switch code {
	case runtime.CodeArtifactUnverified:
		return copy.ModelFailChecksum
	case runtime.CodeProvisioningFailed:
		return copy.ModelFailNetwork
	}
	return copy.FailModelTitle
}
