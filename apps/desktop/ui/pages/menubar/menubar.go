// Package menubar is the panel the menu bar icon opens.
package menubar

import (
	"runtime"

	"arion.local/desktop/ui/features/dictate"
	"arion.local/desktop/ui/features/transcript"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	arion "arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/modelcard"
	"arion.local/desktop/ui/widgets/statusheader"
	"github.com/egoist/mygo/ui"
)

// Width and Height size the panel.
const (
	Width  = 330
	Height = 340
)

// Panel keeps the panel's own view state.
type Panel struct {
	copied transcript.Copied
}

// Render draws the panel over the popover material on macOS.
func (p *Panel) Render(c *ui.Context, store arion.Store) {
	tokens := theme.Apply(c)
	c.Root().Background(ui.Transparent)
	box := ui.Column(c).Fill()
	switch {
	case c.Vibrancy():
	case runtime.GOOS == "windows":
		box.Background(tokens.Canvas)
	default:
		box.Background(tokens.Canvas).Radius(12).Border(1, tokens.Hairline)
	}
	box.Padding(16).Gap(14).Children(func() {
		statusheader.Render(c, store, tokens)
		dictate.Toggle(c, store, tokens)
		transcript.Last(c, store, tokens, &p.copied)
		ui.Spacer(c)
		modelcard.Render(c, store, tokens, true)
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			footerButton(c, tokens, icons.Settings, copy.SettingsMenu, store.Actions.OpenSettings)
			ui.Spacer(c)
			footerButton(c, tokens, icons.Power, copy.Quit, store.Actions.Quit)
		})
	})
}

func footerButton(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, label string, action func()) {
	button := ui.ButtonBase(c.Key(label)).Gap(6).Height(30).Padding(0, 10).Radius(8).AlignItems(ui.Center).TextColor(tokens.Muted)
	kit.Press(button)
	if button.Hovered() {
		button.Background(tokens.AccentSoft).TextColor(tokens.Text)
	}
	button.Transition(ui.ElementTransition{Colors: true, Duration: theme.Quick.Duration})
	button.Children(func() {
		ui.Icon(c, icon).FontSize(13)
		ui.Text(c, label).FontSize(theme.Caption)
	})
	if button.Clicked() {
		action()
	}
}
