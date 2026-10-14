// Package scene shows which scene polishing uses for the app that was
// frontmost when the menu bar panel opened, and lets the user change it
// for that site or app.
package scene

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Label is the scene's name.
func Label(s runtime.Scene) copy.Key {
	switch s {
	case runtime.SceneEmail:
		return copy.SceneEmail
	case runtime.SceneWork:
		return copy.SceneWork
	case runtime.ScenePrompt:
		return copy.ScenePrompt
	case runtime.SceneSearch:
		return copy.SceneSearch
	}
	return copy.SceneChat
}

// Icon is the scene's glyph.
func Icon(s runtime.Scene) *ui.SVG {
	switch s {
	case runtime.SceneEmail:
		return icons.Mail
	case runtime.SceneWork:
		return icons.Briefcase
	case runtime.ScenePrompt:
		return icons.Code
	case runtime.SceneSearch:
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

// Card shows the scene and opens a select to choose another one, or to follow
// the built-in rules again.
func Card(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if !Visible(store) {
		return
	}
	messages := store.Copy()
	target := store.State.SceneTarget
	decision := runtime.DecideScene(target.Hints, store.State.Preferences.SceneOverrides)
	source := copy.SceneAutomatic
	if decision.Route == runtime.SceneByUser {
		source = copy.SceneChosen
	}
	place := target.AppName
	if target.Hints.Site != "" {
		place = target.Hints.Site
	}
	// The empty scene stands for the built-in rules.
	var chosen runtime.Scene
	if decision.Route == runtime.SceneByUser {
		chosen = decision.Scene
	}
	selected := chosen
	sel := ui.SelectBase(c.Key("scene"), &selected)
	button := sel.Trigger.Label(messages.Text(copy.SceneCardLabel)).Height(48).Padding(0, 12).Gap(10).
		AlignItems(ui.Center).Radius(theme.RadiusControl).Border(1, tokens.Hairline)
	kit.Press(button)
	if button.Hovered() || sel.Open() {
		button.Background(tokens.AccentSoft)
	}
	button.Transition(ui.ElementTransition{Colors: true, Duration: theme.Quick.Duration})
	button.Children(func() {
		ui.Icon(c, Icon(decision.Scene)).FontSize(16).TextColor(tokens.Text)
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
	kit.SelectPopup(tokens, sel, func() {
		kit.SelectItem(c, tokens, sel, "", messages.Text(copy.SceneAutomatic), chosen == "")
		kit.SelectSeparator(c, tokens)
		for _, s := range runtime.ChoosableScenes {
			kit.SelectItem(c, tokens, sel, s, messages.Text(Label(s)), chosen == s)
		}
	})
	if button.Changed() {
		store.Actions.ChooseScene(selected)
	}
}
