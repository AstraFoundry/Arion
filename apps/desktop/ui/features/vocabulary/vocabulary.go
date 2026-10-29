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
	Reading readingDraft
}

type readingDraft struct {
	ID, Spelling, Value string
	Error               copy.Key
	Focus               bool
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
		if notice := store.State.VocabularyKeyboardNotice; notice != "" {
			color := tokens.Muted
			if notice == copy.VocabKeyboardFailed {
				color = tokens.Danger
			}
			ui.Text(c, messages.Text(notice)).FontSize(theme.Caption).TextColor(color)
		}
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
						if len(group.Hotwords) > 0 {
							entries := append([]runtime.Replacement(nil), group.Hotwords...)
							kit.IconButton(c, tokens, icons.X, messages.RemoveHotword(group.Text)).OnClick(func() {
								for _, entry := range entries {
									store.Actions.RemoveReplacement(entry.ID)
								}
							})
						}
					})
					for _, entry := range group.Hotwords {
						keyboardReading(c.Key(entry.ID), store, tokens, &draft.Reading, entry)
					}
					for _, variant := range group.Variants {
						id := variant.ID
						ui.Column(c.Key(id)).Gap(4).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Transition(rowMotion).Children(func() {
								ui.Text(c, variant.From).FontSize(theme.Body).TextColor(tokens.Muted).Grow(1).Shrink(1)
								if draft.Reading.ID != id {
									keyboardReading(c, store, tokens, &draft.Reading, variant)
								}
								kit.IconButton(c, tokens, icons.X, messages.RemoveVocabVariant(variant.From)).OnClick(func() { store.Actions.RemoveReplacement(id) })
							})
							if draft.Reading.ID == id {
								keyboardReading(c, store, tokens, &draft.Reading, variant)
							}
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

func keyboardReading(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *readingDraft, entry runtime.Replacement) {
	messages := store.Copy()
	if draft.ID != entry.ID {
		label := entry.KeyboardPinyin
		if label == "" {
			label = messages.Text(copy.VocabReading)
		}
		name := messages.VocabReadingAction(entry.To)
		if entry.From != "" {
			name = messages.VocabReadingVariant(entry.To, entry.From)
		}
		kit.Button(c.Key("reading-edit"), tokens, kit.Ghost, label, true).Label(name).OnClick(func() {
			*draft = readingDraft{ID: entry.ID, Spelling: entry.To, Value: entry.KeyboardPinyin, Focus: true}
		})
		return
	}
	ui.Column(c.Key("reading-form")).Gap(6).Children(func() {
		field := ui.TextInput(c.Key("reading"), &draft.Value).Label(messages.Text(copy.VocabReading)).Placeholder(messages.Text(copy.VocabReadingPlaceholder))
		if draft.Focus {
			field.Focus()
			draft.Focus = false
		}
		ui.Text(c, messages.Text(copy.VocabReadingHint)).FontSize(theme.Caption).TextColor(tokens.Muted)
		if draft.Error != "" {
			ui.Text(c, messages.Text(draft.Error)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
		ui.Row(c).Gap(8).Children(func() {
			submit := kit.Button(c.Key("reading-save"), tokens, kit.Outline, messages.Text(copy.VocabReadingSave), true).Clicked()
			cancel := kit.Button(c.Key("reading-cancel"), tokens, kit.Ghost, messages.Text(copy.VocabReadingCancel), true).Clicked()
			if cancel {
				*draft = readingDraft{}
				return
			}
			if submit || field.Submitted() {
				err := store.Actions.SetKeyboardPinyin(draft.ID, draft.Spelling, strings.TrimSpace(draft.Value))
				switch {
				case err == nil:
					*draft = readingDraft{}
				case errors.Is(err, runtime.ErrVocabularyInvalid):
					draft.Error = copy.VocabReadingInvalid
				case errors.Is(err, runtime.ErrVocabularyConflict):
					draft.Error = copy.VocabReadingStale
				default:
					draft.Error = copy.VocabSaveFailed
				}
			}
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
