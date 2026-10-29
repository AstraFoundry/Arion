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
	"arion.local/desktop/ui/shared/typography"
	"github.com/egoist/mygo/ui"
)

// View keeps the original preview and copy confirmation for the current result.
type View struct {
	text, original string
	expanded       bool
	copiedAt       time.Time
	copiedOriginal bool
}

// Last shows the most recent dictation text.
func Last(c *ui.Context, store runtime.Store, tokens theme.Tokens, view *View) {
	messages := store.Copy()
	text := store.State.LastText
	original := store.State.LastOriginal
	if view.text != text || view.original != original {
		*view = View{text: text, original: original}
	}
	ui.Column(c).Shrink(0).Font(typography.Family).Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, messages.Text(copy.PanelLastDictation)).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted)
			ui.Spacer(c)
			if text != "" {
				copyButton(c.Key("copy"), store, tokens, view, false, messages.Text(copy.Copy), text)
			}
		})
		if text == "" {
			ui.Text(c, messages.Text(copy.PanelEmpty)).FontSize(theme.Caption).TextColor(tokens.Faint)
		} else {
			ui.Text(c.Key(text), text).FontSize(theme.Body).Selectable().
				Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 6}})
		}
		if text == "" || original == "" || original == text {
			return
		}
		ui.Row(c.Key("original-actions")).AlignItems(ui.Center).Children(func() {
			label := messages.Text(copy.ViewOriginalTranscript)
			if view.expanded {
				label = messages.Text(copy.HideOriginalTranscript)
			}
			button := ui.ButtonBase(c.Key("disclosure")).Gap(4).Height(26).Radius(6).AlignItems(ui.Center).Label(label).Expanded(view.expanded)
			kit.Style(button, tokens, kit.Ghost, true)
			button.TextColor(tokens.Muted)
			button.Children(func() {
				ui.Text(c, label).FontSize(theme.Caption).SingleLine()
				icon := ui.Icon(c, icons.ChevronRight).FontSize(12)
				if view.expanded {
					icon.Rotate(90)
				}
			})
			if button.Clicked() {
				view.expanded = !view.expanded
			}
			ui.Spacer(c)
			if view.expanded {
				copyButton(c.Key("original"), store, tokens, view, true, messages.Text(copy.CopyOriginal), original)
			}
		})
		if view.expanded {
			ui.Text(c.Key("original-text"), original).FontSize(theme.Body).Selectable()
		}
	})
}

func copyButton(c *ui.Context, store runtime.Store, tokens theme.Tokens, view *View, original bool, label, text string) {
	justCopied := view.copiedOriginal == original && c.Now().Sub(view.copiedAt) < 1500*time.Millisecond
	if justCopied {
		c.After(1500*time.Millisecond - c.Now().Sub(view.copiedAt))
	}
	icon := icons.Copy
	if justCopied {
		icon, label = icons.Check, store.Copy().Text(copy.Copied)
	}
	button := ui.ButtonBase(c).Gap(4).Height(26).Padding(0, 8).Radius(6).AlignItems(ui.Center).TextColor(tokens.Muted)
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
		view.copiedAt, view.copiedOriginal = c.Now(), original
	}
}
