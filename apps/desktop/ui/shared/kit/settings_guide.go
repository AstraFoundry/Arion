package kit

import (
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

var guideMotion = ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Y: -8, Scale: 0.98}}

// SettingsGuide is the shared shell under a System Settings window. The caller
// supplies either an actual drag source or instructions for the current pane.
func SettingsGuide(c *ui.Context, tokens theme.Tokens, shown int, title, closeLabel string, close func(), content func()) {
	Callout(c, tokens, shown, icons.ArrowUp, title, closeLabel, close, content)
}

// Callout is a floating card whose arrow points at what it explains, with
// a close button and optional content under its title.
func Callout(c *ui.Context, tokens theme.Tokens, shown int, arrow *ui.SVG, title, closeLabel string, close func(), content func()) {
	card := ui.Column(c.Key(shown)).Fill().Padding(10).Gap(8).Radius(theme.RadiusCard).Background(tokens.Card).
		Transition(guideMotion)
	if content == nil {
		card.Justify(ui.Center)
	}
	card.Children(func() {
		ui.Row(c).Height(36).Gap(8).Padding(0, 0, 0, 6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, arrow).Size(14, 14).TextColor(tokens.AccentInk)
			ui.Text(c, title).FontSize(theme.Body).FixedLineHeight(18).Grow(1).Shrink(1)
			if IconButton(c.Key("close"), tokens, icons.X, closeLabel).Clicked() {
				close()
			}
		})
		if content != nil {
			content()
		}
	})
}
