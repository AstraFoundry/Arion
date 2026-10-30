// Package vocabulary edits the terms Arion should spell the user's way: each
// correct term with its incorrect forms and keyboard reading.
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

// Draft holds the editor's unsaved input.
type Draft struct {
	Term    string
	Error   copy.Key
	Query   string
	Variant variantDraft
	Reading readingDraft
}

// variantDraft is the incorrect form being added to one term.
type variantDraft struct {
	Term, Value string
	Error       copy.Key
	Focus       bool
	// Held records that the input has had focus, so it closes on blur
	// only after focus has actually arrived.
	Held bool
}

type readingDraft struct {
	ID, Spelling, Value string
	Error               copy.Key
	Focus               bool
}

var rowMotion = ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}

// chipHeight is shared by incorrect forms, the add control and readings, so
// a term's controls line up whichever are present.
const chipHeight = 24

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

// Editor lists terms and adds new ones. Keyboard shows Pinyin readings,
// which only Arion's own input method uses.
func Editor(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft, keyboard bool) {
	messages := store.Copy()
	ui.Column(c).Gap(12).Children(func() {
		if notice := store.State.VocabularyKeyboardNotice; notice != "" && keyboard {
			color := tokens.Muted
			if notice == copy.VocabKeyboardFailed {
				color = tokens.Danger
			}
			ui.Text(c, messages.Text(notice)).FontSize(theme.Caption).TextColor(color)
		}
		if len(store.State.Vocabulary) > 0 {
			field(c.Key("search"), tokens, icons.Search, &draft.Query, messages.Text(copy.VocabSearch), messages.Text(copy.VocabSearch))
		}
		groups := vocabularymodel.Groups(store.State.Vocabulary, draft.Query)
		kit.Card(c, tokens).Children(func() {
			composer(c, store, tokens, draft)
			if len(groups) == 0 {
				kit.Divider(c, tokens)
				message := copy.VocabEmpty
				if strings.TrimSpace(draft.Query) != "" {
					message = copy.VocabNoMatches
				}
				kit.Empty(c, tokens, messages.Text(message))
			}
			for _, group := range groups {
				ui.Column(c.Key(group.Text)).Transition(rowMotion).Children(func() {
					kit.Divider(c, tokens)
					term(c, store, tokens, draft, group, keyboard)
				})
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).TextColor(tokens.Muted).Children(func() {
			ui.Icon(c, icons.Lock).FontSize(12)
			ui.Text(c, messages.Text(copy.VocabStoredLocal)).FontSize(theme.Caption)
		})
	})
}

// composer adds a correct term. Its incorrect forms and reading are added on
// the term itself, so one kind of entry carries everything.
func composer(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft) {
	messages := store.Copy()
	ui.Column(c).Gap(6).Padding(10, 12, 10, 16).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			var input ui.Element
			ui.Icon(c, icons.Plus).FontSize(14).TextColor(tokens.Muted).OnClick(func() { input.Focus() })
			input = bare(ui.TextInput(c.Key("term"), &draft.Term)).Grow(1).Shrink(1).
				Placeholder(messages.Text(copy.VocabToHint)).Label(messages.Text(copy.VocabToLabel))
			if input.Changed() {
				draft.Error = ""
			}
			add := kit.Button(c.Key("add"), tokens, kit.Outline, messages.Text(copy.Add), strings.TrimSpace(draft.Term) != "").Clicked()
			if add || input.Submitted() {
				addTerm(store, draft)
				input.Focus()
			}
		})
		if draft.Error != "" {
			ui.Text(c, messages.Text(draft.Error)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
	})
}

func term(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft, group vocabularymodel.Term, keyboard bool) {
	messages := store.Copy()
	reading := false
	ui.Column(c).Gap(8).Padding(12, 12, 12, 16).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).MinHeight(28).Children(func() {
			ui.Text(c, group.Text).FontSize(theme.Heading).FontWeight(600).Grow(1).Shrink(1)
			for _, record := range group.Readings {
				if !keyboard {
					break
				}
				if record.ID == draft.Reading.ID {
					reading = true
					readingEditor(c.Key(record.ID), store, tokens, &draft.Reading)
					continue
				}
				readingChip(c.Key(record.ID), store, tokens, &draft.Reading, record)
			}
			if !reading {
				records := group.Records
				kit.IconButton(c, tokens, icons.X, messages.RemoveVocab(group.Text)).OnClick(func() {
					for _, record := range records {
						store.Actions.RemoveReplacement(record.ID)
					}
				})
			}
		})
		ui.Row(c).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
			for _, variant := range group.Variants {
				variantChip(c.Key(variant.ID), store, tokens, variant)
			}
			variantAdder(c, store, tokens, &draft.Variant, group.Text)
		})
		var failure copy.Key
		if draft.Variant.Term == group.Text {
			failure = draft.Variant.Error
		}
		if reading && draft.Reading.Error != "" {
			failure = draft.Reading.Error
		}
		if failure != "" {
			ui.Text(c, messages.Text(failure)).FontSize(theme.Caption).TextColor(tokens.Danger)
		}
	})
}

// variantChip strikes an incorrect form through: it is what Arion replaces.
func variantChip(c *ui.Context, store runtime.Store, tokens theme.Tokens, variant runtime.Replacement) {
	messages := store.Copy()
	id := variant.ID
	ui.Row(c).Height(chipHeight).Padding(0, 3, 0, 10).Gap(2).AlignItems(ui.Center).Radius(theme.RadiusPill).
		Background(tokens.Subtle).Transition(rowMotion).Children(func() {
		ui.Text(c, variant.From).FontSize(theme.Body).TextColor(tokens.Muted).Strikethrough().DecorationColor(tokens.Faint).SingleLine().Shrink(1)
		remove := ui.ButtonBase(c.Key("remove")).Size(18, 18).Radius(9).Justify(ui.Center).AlignItems(ui.Center).
			Label(messages.RemoveVocab(variant.From)).Tooltip(messages.RemoveVocab(variant.From))
		kit.Style(remove, tokens, kit.Ghost, true)
		remove.TextColor(tokens.Faint).Children(func() { ui.Icon(c, icons.X).FontSize(10) }).
			OnClick(func() { store.Actions.RemoveReplacement(id) })
	})
}

// variantAdder is a quiet dashed control that opens in place, so adding a
// form happens where the forms are listed.
func variantAdder(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *variantDraft, term string) {
	messages := store.Copy()
	label := messages.Text(copy.VocabFromLabel)
	if draft.Term != term {
		add := ui.ButtonBase(c.Key("variant-add")).Height(chipHeight).Padding(0, 10, 0, 8).Gap(4).AlignItems(ui.Center).
			Radius(theme.RadiusPill).Border(1, tokens.CardBorder).BorderStyle(ui.BorderDashed).Label(messages.VocabVariantAction(term))
		kit.Style(add, tokens, kit.Ghost, true)
		add.TextColor(tokens.Muted).Children(func() {
			ui.Icon(c, icons.Plus).FontSize(11)
			ui.Text(c, label).FontSize(theme.Caption).SingleLine()
		}).OnClick(func() { *draft = variantDraft{Term: term, Focus: true} })
		return
	}
	input := bare(ui.TextInput(c.Key("variant"), &draft.Value)).Width(160).Height(chipHeight).Padding(0, 10).
		Radius(theme.RadiusPill).Border(1, tokens.CardBorder).BorderStyle(ui.BorderDashed).FontSize(theme.Body).
		Placeholder(label).Label(messages.VocabVariantAction(term)).
		OnShortcut(0, ui.KeyEscape, func() { *draft = variantDraft{} })
	switch {
	case draft.Focus:
		input.Focus()
		draft.Focus = false
	case input.Focused():
		draft.Held = true
	case draft.Held && strings.TrimSpace(draft.Value) == "":
		*draft = variantDraft{}
		return
	}
	if input.Changed() {
		draft.Error = ""
	}
	if input.Submitted() {
		addVariant(store, draft)
	}
}

// readingChip shows the Pinyin that types this term on Arion's keyboard, or
// only the keyboard glyph when none is set.
func readingChip(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *readingDraft, record runtime.Replacement) {
	messages := store.Copy()
	name := messages.VocabReadingAction(record.To)
	if record.From != "" && record.KeyboardPinyin != "" {
		name = messages.VocabReadingVariant(record.To, record.From)
	}
	chip := ui.ButtonBase(c).Height(chipHeight).MinWidth(chipHeight).Padding(0, 6).Gap(6).
		Justify(ui.Center).AlignItems(ui.Center).Radius(theme.RadiusMD).Label(name).Tooltip(name)
	kit.Style(chip, tokens, kit.Ghost, true)
	chip.TextColor(tokens.Muted).Children(func() {
		ui.Icon(c, icons.Keyboard).FontSize(13)
		if record.KeyboardPinyin != "" {
			ui.Text(c, record.KeyboardPinyin).FontSize(theme.Caption).SingleLine()
		}
	}).OnClick(func() {
		*draft = readingDraft{ID: record.ID, Spelling: record.To, Value: record.KeyboardPinyin, Focus: true}
	})
}

// readingEditor edits one record's reading in place; Return saves and
// Escape cancels anywhere within it.
func readingEditor(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *readingDraft) {
	messages := store.Copy()
	cancel := func() { *draft = readingDraft{} }
	var field ui.Element
	var save, dismiss bool
	ui.Row(c).Gap(2).AlignItems(ui.Center).OnShortcut(0, ui.KeyEscape, cancel).Children(func() {
		ui.Icon(c, icons.Keyboard).FontSize(13).TextColor(tokens.Muted).Margin(0, 6, 0, 0)
		field = ui.TextInput(c.Key("reading"), &draft.Value).Width(176).Height(28).Padding(0, 10).FontSize(theme.Body).
			Label(messages.Text(copy.VocabReading)).Placeholder(messages.Text(copy.VocabReadingPlaceholder))
		save = kit.IconButton(c.Key("reading-save"), tokens, icons.Check, messages.Text(copy.VocabReadingSave)).Clicked()
		dismiss = kit.IconButton(c.Key("reading-cancel"), tokens, icons.X, messages.Text(copy.VocabReadingCancel)).Clicked()
	})
	if draft.Focus {
		field.Focus()
		draft.Focus = false
	}
	if field.Changed() {
		draft.Error = ""
	}
	if dismiss {
		cancel()
		return
	}
	if !save && !field.Submitted() {
		return
	}
	err := store.Actions.SetKeyboardPinyin(draft.ID, draft.Spelling, strings.TrimSpace(draft.Value))
	switch {
	case err == nil:
		cancel()
	case errors.Is(err, runtime.ErrVocabularyInvalid):
		draft.Error = copy.VocabReadingInvalid
	case errors.Is(err, runtime.ErrVocabularyConflict):
		draft.Error = copy.VocabReadingStale
	default:
		draft.Error = copy.VocabSaveFailed
	}
}

// field is a search box led by its glyph.
func field(c *ui.Context, tokens theme.Tokens, icon *ui.SVG, value *string, placeholder, label string) {
	var input ui.Element
	ui.Row(c).Height(32).Padding(0, 10).Gap(8).AlignItems(ui.Center).Radius(theme.RadiusControl).
		Background(tokens.Card).Border(1, tokens.CardBorder).Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(tokens.Muted).OnClick(func() { input.Focus() })
		input = bare(ui.TextInput(c.Key("input"), value)).Grow(1).Shrink(1).Placeholder(placeholder).Label(label)
	})
}

// bare strips an input's own box where its container draws one.
func bare(input ui.Element) ui.Element {
	return input.Background(ui.Transparent).Border(0, ui.Transparent).Padding(0, 0)
}

func addTerm(store runtime.Store, draft *Draft) {
	if strings.TrimSpace(draft.Term) == "" {
		return
	}
	switch err := store.Actions.AddHotword(draft.Term); {
	case err == nil:
		draft.Term, draft.Error = "", ""
	case errors.Is(err, runtime.ErrVocabularyInvalid):
		draft.Error = copy.VocabTermInvalid
	default:
		draft.Error = copy.VocabSaveFailed
	}
}

func addVariant(store runtime.Store, draft *variantDraft) {
	if strings.TrimSpace(draft.Value) == "" {
		return
	}
	switch err := store.Actions.AddReplacementVariants([]string{draft.Value}, draft.Term); {
	case err == nil:
		draft.Value, draft.Error = "", ""
	case errors.Is(err, runtime.ErrVocabularyInvalid):
		draft.Error = copy.VocabInvalid
	case errors.Is(err, runtime.ErrVocabularyConflict):
		draft.Error = copy.VocabConflict
	default:
		draft.Error = copy.VocabSaveFailed
	}
}
