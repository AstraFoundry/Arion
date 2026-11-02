// Package menubar is the panel the menu bar icon opens.
package menubar

import (
	"runtime"

	"arion.local/desktop/ui/features/dictate"
	"arion.local/desktop/ui/features/provision"
	"arion.local/desktop/ui/features/scene"
	"arion.local/desktop/ui/features/update"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	arion "arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/widgets/statusheader"
	"github.com/egoist/mygo/ui"
)

// Width and Height size the ready panel. Optional blocks add only their own space.
const (
	Width  = 330
	Height = 160
)

// HeightFor follows the visible blocks rather than reserving transcript space.
func HeightFor(store arion.Store) int {
	height := Height
	if scene.Visible(store) {
		height += 62
	}
	if store.State.Model.State != arion.ModelReady {
		height += 64
	}
	if _, ok := update.Notice(store); ok {
		height += 34
	}
	return height
}

// Panel keeps the panel's own view state.
type Panel struct{}

// Render draws the panel over the popover material on macOS.
func (p *Panel) Render(c *ui.Context, store arion.Store) {
	messages := store.Copy()
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
		scene.Card(c, store, tokens)
		// The bundle itself is chosen in settings; the panel only shows the
		// download or failure that keeps dictation from starting.
		if store.State.Model.State != arion.ModelReady {
			provision.Progress(c, store, tokens)
		}
		// A pending update takes its own line, so a long version never
		// crowds Settings and Quit off the footer.
		if label, ok := update.Notice(store); ok {
			ui.Row(c.Key("update")).Margin(0, 0, -10, 0).Children(func() {
				footerButton(c, tokens, icons.Download, label, store.Actions.OpenUpdate)
			})
		}
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			footerButton(c, tokens, icons.Settings, messages.Text(copy.SettingsMenu), store.Actions.OpenSettings)
			ui.Spacer(c)
			footerButton(c, tokens, icons.Power, messages.Text(copy.Quit), store.Actions.Quit)
		})
	})
}

func footerButton(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, label string, action func()) {
	button := ui.ButtonBase(c.Key(label)).Gap(6).Height(30).Padding(0, 10).Radius(8).AlignItems(ui.Center).Shrink(0).TextColor(tokens.Muted)
	kit.Press(button)
	if button.Hovered() {
		button.Background(tokens.AccentSoft).TextColor(tokens.Text)
	}
	button.Transition(ui.ElementTransition{Colors: true, Duration: theme.Quick.Duration})
	button.Children(func() {
		ui.Icon(c, icon).FontSize(13)
		ui.Text(c, label).FontSize(theme.Caption).SingleLine()
	})
	if button.Clicked() {
		action()
	}
}
