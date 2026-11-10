package usage

import (
	"arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The first-run card's measures: the space does the grouping, so the
// figure stands well clear of the words above and below it.
const (
	firstPad     = 24
	figureHeight = 172
	figureWidth  = 300
	figureGap    = 20
	bodyWidth    = 340
)

// first stands in for the totals until the first dictation is committed:
// what is missing and what to do in one two-tone thought, the tray figure,
// what will collect here, and the key that starts a dictation.
func first(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog) ui.Element {
	card := kit.Card(c, tokens)
	card.Children(func() {
		ui.Column(c).Padding(firstPad).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, messages.Text(copy.UsageFirstTitle)).FontSize(theme.Title).FontWeight(500).TextColor(tokens.Text)
				ui.Text(c, messages.Text(copy.UsageFirstPrompt)).FontSize(theme.Title).FontWeight(500).TextColor(tokens.Muted)
			})
			Tray(c, tokens, messages.Text(copy.UsageFirstFigureLabel)).Width(figureWidth).MaxWidthPercent(100).Height(figureHeight).MarginY(figureGap)
			ui.Text(c, messages.Text(copy.UsageFirstBody)).FontSize(theme.Body).TextColor(tokens.Muted).MaxWidth(bodyWidth)
			// A platform without a dictation key has nothing to hold.
			if key := holdkey.Legend(store.State.Preferences.HoldKey); key != "" {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(12, 0, 0, 0).Children(func() {
					kit.Kbd(c, tokens, key, false)
					ui.Text(c, messages.Text(copy.UsageFirstHold)).FontSize(theme.Body).TextColor(tokens.Text)
				})
			}
		})
	})
	return card
}
