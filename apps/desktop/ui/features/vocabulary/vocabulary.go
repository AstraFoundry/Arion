// Package vocabulary edits approved replacements the corrector applies.
package vocabulary

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
	"strings"
)

// Draft holds the add form.
type Draft struct {
	From  string
	To    string
	Error copy.Key
}

var rowMotion = ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}

// Editor lists replacements and adds new ones.
func Editor(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft) {
	messages := store.Copy()
	ui.Column(c).Gap(12).Children(func() {
		kit.Card(c, tokens).Children(func() {
			ui.Row(c).Gap(8).Padding(10, 16).AlignItems(ui.Center).Children(func() {
				ui.Text(c, messages.Text(copy.VocabHeard)).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted).Width(180)
				ui.Text(c, messages.Text(copy.VocabWrite)).FontSize(theme.Caption).FontWeight(600).TextColor(tokens.Muted).Grow(1)
			})
			kit.Divider(c, tokens)
			if len(store.State.Vocabulary) == 0 {
				ui.Text(c.Key("empty"), messages.Text(copy.VocabEmpty)).FontSize(theme.Body).TextColor(tokens.Muted).Padding(16)
			}
			for _, entry := range store.State.Vocabulary {
				id := entry.ID
				ui.Row(c.Key(entry.ID)).Gap(8).Padding(8, 16).AlignItems(ui.Center).Transition(rowMotion).Children(func() {
					ui.Text(c, entry.From).FontSize(theme.Body).Width(180).SingleLine()
					ui.Icon(c, icons.ArrowRight).FontSize(12).TextColor(tokens.Faint)
					ui.Text(c, entry.To).FontSize(theme.Body).FontWeight(500).Grow(1).SingleLine()
					kit.IconButton(c, tokens, icons.X, messages.Text(copy.Remove)).OnClick(func() { store.Actions.RemoveReplacement(id) })
				})
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			from := ui.TextInput(c.Key("from"), &draft.From).Placeholder(messages.Text(copy.VocabFromHint)).Label(messages.Text(copy.VocabFromLabel)).Width(180)
			ui.Icon(c, icons.ArrowRight).FontSize(12).TextColor(tokens.Faint)
			to := ui.TextInput(c.Key("to"), &draft.To).Placeholder(messages.Text(copy.VocabToHint)).Label(messages.Text(copy.VocabToLabel)).Grow(1)
			add := kit.Button(c.Key("add"), tokens, kit.Outline, messages.Text(copy.Add), true).Clicked()
			if add || from.Submitted() || to.Submitted() {
				if strings.TrimSpace(draft.From) == "" || strings.TrimSpace(draft.To) == "" {
					draft.Error = copy.VocabInvalid
				} else if err := store.Actions.AddReplacement(draft.From, draft.To); err != nil {
					draft.Error = copy.VocabSaveFailed
				} else {
					*draft = Draft{}
				}
			}
		})
		if draft.Error != "" {
			ui.Text(c, messages.Text(draft.Error)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
		ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(tokens.Muted).Children(func() {
			ui.Icon(c, icons.Lock).FontSize(12)
			ui.Text(c, messages.Text(copy.VocabStoredLocal)).FontSize(theme.Caption)
		})
	})
}
