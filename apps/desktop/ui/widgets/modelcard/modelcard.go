// Package modelcard presents the automatically chosen recognition bundle,
// why it was chosen and its provisioning state.
package modelcard

import (
	"arion.local/desktop/ui/features/modelchoice"
	"arion.local/desktop/ui/features/provision"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Render draws the card. Compact hides the metrics, for the menu bar.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens, compact bool) {
	m := store.State.Model
	kit.Card(c, tokens).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Cpu).FontSize(16).TextColor(tokens.Accent)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				name := m.Active.Name
				if name == "" {
					name = copy.ModelNoEvaluation
				}
				ui.Text(c, name).FontSize(theme.Body).FontWeight(600).SingleLine()
				source := copy.PanelAutomatic
				if !m.Automatic {
					source = copy.PanelChosen
				}
				ui.Text(c, source).FontSize(theme.Caption).TextColor(tokens.Muted)
			})
			if !m.Automatic && !compact {
				if ui.Button(c, copy.ModelUseRecommend).Clicked() {
					store.Actions.UseRecommendedBundle()
				}
			}
		})
		if !compact && m.Active.Evaluated {
			reason := copy.ModelReason
			if !m.LatencyMeasured {
				reason = copy.ModelReasonRemote
			}
			ui.Text(c, reason).FontSize(theme.Caption).TextColor(tokens.Muted)
			modelchoice.Metrics(c, tokens, m.Active)
		}
		provision.Progress(c, store, tokens)
	})
}
