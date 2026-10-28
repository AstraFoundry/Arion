// Package vocabulary edits approved corrections and recognition hotwords.
package vocabulary

import (
	"errors"
	"strings"

	vocabularymodel "arion.local/desktop/ui/entities/vocabulary"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Draft holds the add form.
type Draft struct {
	Hotword bool
	From    string
	To      string
	Error   copy.Key
	Query   string
}

var rowMotion = ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}

// Learning is the switch for learning entries from the user's corrections
// of inserted text (ADR 014).
func Learning(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	on := store.State.Preferences.LearnEdits
	kit.Card(c, tokens).Children(func() {
		kit.TipRow(c, tokens, messages.Text(copy.LearnEditsLabel), messages.Text(copy.LearnEditsDetail), func() {
			if ui.Switch(c.Key("learn-edits"), &on).Label(messages.Text(copy.LearnEditsLabel)).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.LearnEdits = on })
			}
		})
	})
}

// Editor lists replacements and adds new ones.
func Editor(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft) {
	messages := store.Copy()
	ui.Column(c).Gap(12).Children(func() {
		kit.Card(c, tokens).Padding(12, 16).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, messages.Text(copy.VocabHotwordMode)).FontSize(theme.Body).Grow(1).OnClick(func() { draft.Hotword = !draft.Hotword; draft.Error = "" })
					if ui.Switch(c.Key("hotword-mode"), &draft.Hotword).Label(messages.Text(copy.VocabHotwordMode)).Changed() {
						draft.Error = ""
					}
				})
				var to, from ui.Element
				ui.Text(c, messages.Text(copy.VocabToLabel)).FontSize(theme.Caption).TextColor(tokens.Muted).OnClick(func() { to.Focus() })
				to = ui.TextInput(c.Key("to"), &draft.To).Placeholder(messages.Text(copy.VocabToHint)).Label(messages.Text(copy.VocabToLabel))
				if !draft.Hotword {
					ui.Text(c, messages.Text(copy.VocabFromLabel)).FontSize(theme.Caption).TextColor(tokens.Muted).OnClick(func() { from.Focus() })
					from = ui.TextArea(c.Key("from"), &draft.From).Placeholder(messages.Text(copy.VocabFromHint)).Label(messages.Text(copy.VocabFromLabel)).Height(80)
				}
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					hint := copy.VocabVariantsHint
					if draft.Hotword {
						hint = copy.VocabHotwordHint
					}
					ui.Text(c, messages.Text(hint)).FontSize(theme.Caption).TextColor(tokens.Muted).Grow(1)
					add := kit.Button(c.Key("add"), tokens, kit.Outline, messages.Text(copy.Add), true).Clicked()
					if add || to.Submitted() {
						save(store, draft)
					}
				})
			})
		})
		if draft.Error != "" {
			ui.Text(c, messages.Text(draft.Error)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
		ui.TextInput(c.Key("search"), &draft.Query).Placeholder(messages.Text(copy.VocabSearch)).Label(messages.Text(copy.VocabSearch))
		groups := vocabularymodel.Groups(store.State.Vocabulary, draft.Query)
		kit.Card(c, tokens).Children(func() {
			if len(groups) == 0 {
				message := copy.VocabEmpty
				if strings.TrimSpace(draft.Query) != "" {
					message = copy.VocabNoMatches
				}
				kit.Empty(c, tokens, messages.Text(message))
			}
			for index, group := range groups {
				if index > 0 {
					kit.Divider(c, tokens)
				}
				ui.Column(c.Key(group.Text)).Gap(2).Padding(12, 16).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, group.Text).FontSize(theme.Heading).FontWeight(600).Grow(1).Shrink(1)
						if group.Hotword {
							ui.Text(c, messages.Text(copy.VocabHotwordStatus)).FontSize(theme.Caption).TextColor(tokens.Muted)
						}
						if len(group.HotwordIDs) > 0 {
							ids := append([]string(nil), group.HotwordIDs...)
							kit.IconButton(c, tokens, icons.X, messages.RemoveHotword(group.Text)).OnClick(func() {
								for _, id := range ids {
									store.Actions.RemoveReplacement(id)
								}
							})
						}
					})
					for _, variant := range group.Variants {
						id := variant.ID
						ui.Row(c.Key(id)).Gap(8).AlignItems(ui.Center).Transition(rowMotion).Children(func() {
							ui.Text(c, variant.From).FontSize(theme.Body).TextColor(tokens.Muted).Grow(1).Shrink(1)
							kit.IconButton(c, tokens, icons.X, messages.RemoveVocabVariant(variant.From)).OnClick(func() { store.Actions.RemoveReplacement(id) })
						})
					}
				})
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(tokens.Muted).Children(func() {
			ui.Icon(c, icons.Lock).FontSize(12)
			ui.Text(c, messages.Text(copy.VocabStoredLocal)).FontSize(theme.Caption)
		})
	})
}

func save(store runtime.Store, draft *Draft) {
	var variants []string
	for _, line := range strings.Split(draft.From, "\n") {
		if variant := strings.TrimSpace(line); variant != "" {
			variants = append(variants, variant)
		}
	}
	if (!draft.Hotword && len(variants) == 0) || strings.TrimSpace(draft.To) == "" {
		draft.Error = copy.VocabInvalid
		if draft.Hotword {
			draft.Error = copy.VocabHotwordInvalid
		}
		return
	}
	var err error
	if draft.Hotword {
		err = store.Actions.AddHotword(draft.To)
	} else {
		err = store.Actions.AddReplacementVariants(variants, draft.To)
	}
	if err != nil {
		switch {
		case errors.Is(err, runtime.ErrVocabularyInvalid):
			draft.Error = copy.VocabInvalid
			if draft.Hotword {
				draft.Error = copy.VocabHotwordInvalid
			}
		case errors.Is(err, runtime.ErrVocabularyConflict):
			draft.Error = copy.VocabConflict
		default:
			draft.Error = copy.VocabSaveFailed
		}
		return
	}
	draft.From, draft.To, draft.Error = "", "", ""
}
