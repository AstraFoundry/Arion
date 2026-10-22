// Package contributors shows who builds Arion: a stack of GitHub avatars in
// Settings › About and the page listing every contributor.
package contributors

import (
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// stackLimit is how many avatars the About row shows before a count.
const stackLimit = 5

// Row opens the contributors page from About. Its avatars follow GitHub's
// order left to right, each one over half of the next.
func Row(c *ui.Context, store runtime.Store, tokens theme.Tokens) bool {
	messages := store.Copy()
	list := assets.Contributors
	shown := list[:min(len(list), stackLimit)]
	return kit.LinkRowWith(c, tokens, messages.Text(copy.AboutContributors), nil, func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			// Reversed, the first contributor is drawn last: leftmost and on top.
			ui.Row(c).Reverse().AlignItems(ui.Center).Children(func() {
				for i := len(shown) - 1; i >= 0; i-- {
					a := avatar(c.Key(shown[i].Login), tokens, shown[i], 24)
					if i > 0 {
						a.Margin(0, 0, 0, -12)
					}
				}
			})
			if more := len(list) - len(shown); more > 0 {
				ui.Text(c, messages.MoreContributors(more)).FontSize(theme.Caption).TextColor(tokens.Muted).SingleLine()
			}
		})
	})
}

// Page lists every contributor. The avatars pop in one after another when
// it opens; hovering one names its GitHub account, and a click opens its
// profile in the browser.
func Page(c *ui.Context, tokens theme.Tokens) {
	kit.Card(c, tokens).Children(func() {
		ui.Row(c).Wrap().Gap(14).Padding(18).Children(func() {
			for i, person := range assets.Contributors {
				button := ui.ButtonBase(c.Key(person.Login)).Radius(26).Label(person.Login).
					Transition(ui.ElementTransition{
						Animation: &ui.Animation{Spring: theme.Pop.Spring, Delay: time.Duration(i) * 35 * time.Millisecond},
						Enter:     &ui.Motion{Scale: 0.6, Y: 6},
					})
				lift := float32(0)
				if button.Hovered() {
					lift = -2
				}
				button.Children(func() {
					avatar(c, tokens, person, 48).Margin(lift, 0, -lift, 0)
				})
				kit.Tooltip(c, tokens, button, person.Login)
				if button.Clicked() {
					c.OpenURL(person.Profile)
				}
			}
		})
	})
}

// avatar is a round GitHub avatar ringed in the card's color, so stacked
// avatars stay apart.
func avatar(c *ui.Context, tokens theme.Tokens, person assets.Contributor, size float32) ui.Element {
	return ui.Image(c, person.Avatar).Size(size, size).Fit(ui.Cover).Radius(size/2).Border(2, tokens.Card).Clip().Shrink(0)
}
