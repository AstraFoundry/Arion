// Package selectionmenu is the popover shown over text selected in another
// app: prompt rows, the question field and a bottom row of the assistant,
// copy and vocabulary. It renders the actions it is given; the backend
// filters them, fills and opens URLs, and writes the vocabulary.
package selectionmenu

import (
	"errors"
	"strings"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Width and the row metrics size the popover window.
const (
	Width     = 260
	rowHeight = 30
	padding   = 6
	gap       = 2
	divider   = 9
	notice    = 24
)

// Menu keeps the highlight and the question between frames. A new offer
// resets both.
type Menu struct {
	Question  string
	highlight int
	offer     string
	failure   copy.Key
}

type layout struct {
	prompts []runtime.SelectionAction
	ask     *runtime.SelectionAction
	clients []runtime.SelectionAction
	copy    *runtime.SelectionAction
	vocab   *runtime.SelectionAction
}

func split(actions []runtime.SelectionAction) layout {
	var l layout
	for i := range actions {
		action := &actions[i]
		switch action.Kind {
		case runtime.SelectionKindPrompt:
			if action.ID == runtime.SelectionActionAsk {
				l.ask = action
			} else {
				l.prompts = append(l.prompts, *action)
			}
		case runtime.SelectionKindClient:
			l.clients = append(l.clients, *action)
		case runtime.SelectionKindCopy:
			l.copy = action
		case runtime.SelectionKindVocabulary:
			l.vocab = action
		}
	}
	return l
}

func (l layout) bottom() bool { return len(l.clients) > 0 || l.copy != nil || l.vocab != nil }

// Height is the popover's height for actions without a failure line.
func Height(actions []runtime.SelectionAction) int {
	l := split(actions)
	rows := len(l.prompts)
	if l.ask != nil {
		rows++
	}
	if l.bottom() {
		rows++
	}
	height := 2*padding + rows*rowHeight + max(0, rows-1)*gap
	if l.bottom() && rows > 1 {
		height += divider
	}
	return height
}

// title names a built-in action; unknown ids have no title and are not drawn.
func title(id string) copy.Key {
	switch id {
	case runtime.SelectionActionTranslate:
		return copy.SelectionTranslate
	case runtime.SelectionActionExplain:
		return copy.SelectionExplain
	case runtime.SelectionActionSummarize:
		return copy.SelectionSummarize
	case runtime.SelectionActionRewrite:
		return copy.SelectionRewrite
	case runtime.SelectionActionAsk:
		return copy.SelectionAsk
	case runtime.SelectionActionAskChatGPT:
		return copy.SelectionAskChatGPT
	case runtime.SelectionActionCopy:
		return copy.Copy
	case runtime.SelectionActionVocabulary:
		return copy.SelectionVocabulary
	}
	return ""
}

// Render draws the current offer as a dark menu in either appearance and
// returns the height it needs, including a failure line.
func (m *Menu) Render(c *ui.Context, store runtime.Store) int {
	messages := store.Copy()
	offer := store.State.Selection
	if offer.Text != m.offer {
		m.offer, m.Question, m.highlight, m.failure = offer.Text, "", 0, ""
	}
	l := split(offer.Actions)
	m.highlight = max(0, min(m.highlight, len(l.prompts)-1))
	tokens := theme.Apply(c)
	outer := c.Theme()
	scoped := *outer
	scoped.Dark = true
	scoped.Text, scoped.TextMuted, scoped.Border = tokens.PillText, tokens.PillMuted, tokens.PillBorder
	scoped.Surface, scoped.SurfaceHover, scoped.SurfacePressed = tokens.PillBorder, tokens.PillBorder, tokens.PillBorder
	c.SetTheme(&scoped)
	ui.Column(c).Fill().Padding(padding).Gap(gap).Radius(theme.RadiusLG).Background(tokens.PillSurface).Border(1, tokens.PillBorder).
		TextColor(tokens.PillText).Children(func() {
		for i, action := range l.prompts {
			if m.row(c, tokens, messages, action, i == m.highlight) {
				m.highlight = i
				m.run(store, action)
			}
		}
		if l.ask != nil {
			m.askField(c, store, tokens, messages, *l.ask)
		}
		if l.bottom() {
			if len(l.prompts) > 0 || l.ask != nil {
				ui.Box(c).Height(1).Margin(4, 4).Background(tokens.PillBorder)
			}
			m.bottomRow(c, store, tokens, messages, l)
		}
		if m.failure != "" {
			ui.Text(c, messages.Text(m.failure)).FontSize(theme.Caption).TextColor(tokens.PillMuted).Height(notice-gap).Padding(4, 8).SingleLine()
		}
	})
	c.SetTheme(outer)
	c.Root().Background(ui.Transparent)
	if c.Shortcut(0, ui.KeyDown) && len(l.prompts) > 0 {
		m.highlight = min(m.highlight+1, len(l.prompts)-1)
	}
	if c.Shortcut(0, ui.KeyUp) && len(l.prompts) > 0 {
		m.highlight = max(m.highlight-1, 0)
	}
	if c.Shortcut(0, ui.KeyEnter) {
		switch {
		case len(l.prompts) > 0:
			m.run(store, l.prompts[m.highlight])
		case l.copy != nil:
			m.run(store, *l.copy)
		}
	}
	if m.failure != "" {
		return Height(offer.Actions) + notice
	}
	return Height(offer.Actions)
}

// row is one prompt; the highlighted row carries the Return glyph.
func (m *Menu) row(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, highlighted bool) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+action.ID)).Height(rowHeight).Padding(0, 10).Gap(8).Radius(theme.RadiusMD).
		Justify(ui.Start).AlignItems(ui.Center).Label(name).Selected(highlighted)
	ink := tokens.PillText
	switch {
	case highlighted:
		button.Background(tokens.Accent)
		ink = tokens.AccentText
	case button.Hovered():
		button.Background(tokens.PillText.Alpha(0.08))
	}
	button.Children(func() {
		ui.Text(c, name).FontSize(theme.Body).TextColor(ink).SingleLine().Grow(1).Shrink(1)
		if highlighted {
			ui.Icon(c, icons.CornerDownLeft).FontSize(13).TextColor(ink).Label(messages.Text(copy.SelectionReturn))
		}
	})
	return button.Clicked()
}

func (m *Menu) askField(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction) {
	name := messages.Text(title(action.ID))
	field := ui.TextInput(c.Key("action-ask"), &m.Question).Label(name).Placeholder(name).Height(rowHeight).
		Background(tokens.PillText.Alpha(0.06)).Border(0, ui.Transparent).TextColor(tokens.PillText)
	if field.Submitted() && strings.TrimSpace(m.Question) != "" {
		m.run(store, action)
	}
}

func (m *Menu) bottomRow(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, l layout) {
	ui.Row(c).Height(rowHeight).Gap(4).AlignItems(ui.Center).Children(func() {
		for _, action := range l.clients {
			if m.tool(c, tokens, messages, action, icons.OpenAI, true) {
				m.run(store, action)
			}
		}
		ui.Spacer(c)
		if l.copy != nil && m.tool(c, tokens, messages, *l.copy, icons.Copy, false) {
			m.run(store, *l.copy)
		}
		if l.vocab != nil && m.tool(c, tokens, messages, *l.vocab, icons.BookOpen, true) {
			m.run(store, *l.vocab)
		}
	})
}

// tool is a bottom-row button; copy shows only its glyph and keeps the name
// as its label.
func (m *Menu) tool(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, icon *ui.SVG, text bool) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+action.ID)).Height(rowHeight-4).Padding(0, 8).Gap(6).Radius(theme.RadiusMD).
		AlignItems(ui.Center).Label(name)
	if button.Hovered() {
		button.Background(tokens.PillText.Alpha(0.08))
	}
	button.Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(tokens.PillMuted)
		if text {
			ui.Text(c, name).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
		}
	})
	return button.Clicked()
}

func (m *Menu) run(store runtime.Store, action runtime.SelectionAction) {
	var err error
	switch action.Kind {
	case runtime.SelectionKindPrompt, runtime.SelectionKindClient:
		question := ""
		if action.ID == runtime.SelectionActionAsk {
			question = strings.TrimSpace(m.Question)
		}
		err = store.Actions.OpenSelection(action.ID, question)
	case runtime.SelectionKindCopy:
		err = store.Actions.CopySelection()
	case runtime.SelectionKindVocabulary:
		err = store.Actions.AddSelectionVocabulary()
	}
	m.failure = failure(action, err)
}

func failure(action runtime.SelectionAction, err error) copy.Key {
	switch {
	case err == nil:
		return ""
	case runtime.CodeOf(err) == runtime.CodeSecureFieldBlocked:
		return copy.FailSecureTitle
	case action.Kind == runtime.SelectionKindVocabulary && errors.Is(err, runtime.ErrVocabularyInvalid):
		return copy.VocabTermInvalid
	case action.Kind == runtime.SelectionKindVocabulary:
		return copy.VocabSaveFailed
	case action.Kind == runtime.SelectionKindPrompt || action.Kind == runtime.SelectionKindClient:
		return copy.SelectionOpenFailed
	}
	return copy.SelectionFailed
}
