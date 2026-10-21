// Package modelcard presents the automatically chosen recognition bundle,
// why it was chosen and its provisioning state.
package modelcard

import (
	"arion.local/desktop/ui/entities/model"
	"arion.local/desktop/ui/features/modelchoice"
	"arion.local/desktop/ui/features/provision"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Render draws the card. Compact hides the metrics and the choice, for a
// summary.
func Render(c *ui.Context, store runtime.Store, tokens theme.Tokens, compact bool) {
	messages := store.Copy()
	m := store.State.Model
	kit.Card(c, tokens).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icons.Cpu).FontSize(16).TextColor(tokens.Muted)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				name := messages.Text(model.NameKey(m.Active.ID))
				if m.Active.ID == "" {
					name = messages.Text(copy.ModelNoEvaluation)
				}
				ui.Text(c, name).FontSize(theme.Body).FontWeight(600).SingleLine()
				source := messages.Text(copy.PanelAutomatic)
				if !m.Automatic {
					source = messages.Text(copy.PanelChosen)
				}
				ui.Text(c, source).FontSize(theme.Caption).TextColor(tokens.Muted)
			})
			if !m.Automatic && !compact {
				if kit.Button(c, tokens, kit.Outline, messages.Text(copy.ModelUseRecommend), true).Clicked() {
					store.Actions.UseRecommendedBundle()
				}
			}
		})
		if !compact && m.Active.Evaluated {
			reason := messages.Text(copy.ModelReason)
			if !m.LatencyMeasured {
				reason = messages.Text(copy.ModelReasonRemote)
			}
			ui.Text(c, reason).FontSize(theme.Caption).TextColor(tokens.Muted)
			modelchoice.Metrics(c, tokens, store.Copy(), m.Active)
		}
		provision.Progress(c, store, tokens)
	})
}
