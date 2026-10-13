// Package modelchoice lets the user keep the recommended bundle or pick
// another evaluated one explicitly.
package modelchoice

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Metrics shows a bundle's measured error rates and latency.
func Metrics(c *ui.Context, tokens theme.Tokens, bundle runtime.Bundle) {
	ui.Row(c).Gap(18).Children(func() {
		metric := func(label, value string) {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, value).FontSize(theme.Heading).FontWeight(600).FontFeatures("tnum")
				ui.Text(c, label).FontSize(theme.Caption).TextColor(tokens.Muted)
			})
		}
		for _, entry := range []struct{ key, label string }{{"zh", copy.ModelMandarin}, {"en", copy.ModelEnglish}, {"mixed", copy.ModelMixed}} {
			if rate, ok := bundle.ErrorRates[entry.key]; ok {
				metric(entry.label, copy.Percent(rate))
			}
		}
		if bundle.ReviewP95Ms > 0 {
			metric(copy.ModelLatency, copy.Millis(bundle.ReviewP95Ms))
		}
	})
}

// Others lists evaluated alternatives with a Use action.
func Others(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	model := store.State.Model
	kit.Card(c, tokens).Children(func() {
		first := true
		for _, bundle := range model.Bundles {
			if !bundle.Evaluated || bundle.ID == model.Active.ID {
				continue
			}
			if !first {
				kit.Divider(c, tokens)
			}
			first = false
			id := bundle.ID
			ui.Row(c.Key(id)).Gap(12).Padding(12, 16).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(6).Children(func() {
					ui.Text(c, bundle.Name).FontSize(theme.Body).FontWeight(500)
					Metrics(c, tokens, bundle)
				})
				ui.Button(c, copy.ModelUse).Disabled(model.State == runtime.ModelProvisioning || model.State == runtime.ModelLoading).
					OnClick(func() { store.Actions.UseBundle(id) })
			})
		}
	})
}
