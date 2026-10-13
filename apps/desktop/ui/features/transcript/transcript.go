// Package transcript shows the last dictation with a copy action, the
// user's way out when insertion was not possible.
package transcript

import (
	"time"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Copied remembers when the copy button was used, for its confirmation.
type Copied struct{ At time.Time }

// Last shows the most recent dictation text.
func Last(c *ui.Context, store runtime.Store, tokens theme.Tokens, copied *Copied) {
	messages := store.Copy()
	text := store.State.LastText
	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, messages.Text(copy.PanelLastDictation)).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted)
			ui.Spacer(c)
			if text != "" {
				justCopied := c.Now().Sub(copied.At) < 1500*time.Millisecond
				if justCopied {
					c.After(1500*time.Millisecond - c.Now().Sub(copied.At))
				}
				icon, label := icons.Copy, messages.Text(copy.Copy)
				if justCopied {
					icon, label = icons.Check, messages.Text(copy.Copied)
				}
				button := ui.ButtonBase(c.Key("copy")).Gap(4).Height(26).Padding(0, 8).Radius(6).AlignItems(ui.Center).TextColor(tokens.Muted)
				kit.Press(button)
				if button.Hovered() {
					button.Background(tokens.AccentSoft).TextColor(tokens.Text)
				}
				button.Children(func() {
					ui.Icon(c, icon).FontSize(12).ContentTransition(ui.ContentTransition{})
					ui.Text(c, label).FontSize(theme.Caption).ContentTransition(ui.ContentTransition{})
				})
				if button.Clicked() {
					store.Actions.CopyText(text)
					copied.At = c.Now()
				}
			}
		})
		if text == "" {
			ui.Text(c, messages.Text(copy.PanelEmpty)).FontSize(theme.Caption).TextColor(tokens.Faint)
		} else {
			ui.Text(c.Key(text), text).FontSize(theme.Body).MaxLines(4).Selectable().
				Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 6}})
		}
	})
}
