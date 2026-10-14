// Package scene shows which scene polishing uses for the app that was
// frontmost when the menu bar panel opened, and lets the user change it
// for that site or app.
package scene

import (
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Label is the scene's name.
func Label(s domain.Scene) copy.Key {
	switch s {
	case domain.SceneEmail:
		return copy.SceneEmail
	case domain.SceneWork:
		return copy.SceneWork
	case domain.ScenePrompt:
		return copy.ScenePrompt
	case domain.SceneSearch:
		return copy.SceneSearch
	}
	return copy.SceneChat
}

func icon(s domain.Scene) *ui.SVG {
	switch s {
	case domain.SceneEmail:
		return icons.Mail
	case domain.SceneWork:
		return icons.Briefcase
	case domain.ScenePrompt:
		return icons.Code
	case domain.SceneSearch:
		return icons.Search
	}
	return icons.MessageCircle
}

// Visible reports whether the card applies: polishing and Match the app are
// on and the panel found another app's field.
func Visible(store runtime.Store) bool {
	s := store.State
	return s.SceneTarget.Known && s.Preferences.Polish && s.Preferences.MatchApp && s.Polish.State != runtime.PolishUnsupported
}

// Card shows the scene and opens a menu to choose another one, or to follow
// the built-in rules again.
func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !Visible(store) {
		return
	}
	messages := store.Copy()
	target := store.State.SceneTarget
	decision := domain.DecideScene(target.Hints, store.State.Preferences.SceneOverrides)
	source := copy.SceneAutomatic
	if decision.Route == domain.RouteUser {
		source = copy.SceneChosen
	}
	place := target.AppName
	if target.Hints.Site != "" {
		place = target.Hints.Site
	}
	button := ui.ButtonBase(c.Key("scene")).Label(messages.Text(copy.SceneCardLabel)).Height(48).Padding(0, 12).Gap(10).
		AlignItems(ui.Center).Radius(theme.RadiusControl).Border(1, tokens.Hairline)
	kit.Press(button)
	if button.Hovered() {
		button.Background(tokens.AccentSoft)
	}
	button.Transition(ui.ElementTransition{Colors: true, Duration: theme.Quick.Duration})
	button.Children(func() {
		ui.Icon(c, icon(decision.Scene)).FontSize(16).TextColor(tokens.Text)
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, messages.Text(Label(decision.Scene))).FontSize(theme.Body).FontWeight(600).SingleLine().
				ContentTransition(ui.ContentTransition{})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				if place != "" {
					ui.Text(c, place).FontSize(theme.Caption).TextColor(tokens.Muted).SingleLine().Shrink(1)
					ui.Box(c).Size(3, 3).Radius(1.5).Background(tokens.Faint)
				}
				ui.Text(c, messages.Text(source)).FontSize(theme.Caption).TextColor(tokens.Muted).SingleLine()
			})
		})
		ui.Icon(c, icons.ChevronsUpDown).FontSize(14).TextColor(tokens.Muted)
	})
	button.Menu(func(m *ui.Menu) {
		if m.Item(messages.Text(copy.SceneAutomatic)).Checked(decision.Route != domain.RouteUser).Chosen() {
			store.Actions.ChooseScene("")
		}
		m.Separator()
		for _, s := range domain.ChoosableScenes {
			if m.Item(messages.Text(Label(s))).Checked(decision.Route == domain.RouteUser && decision.Scene == s).Chosen() {
				store.Actions.ChooseScene(s)
			}
		}
	})
}
