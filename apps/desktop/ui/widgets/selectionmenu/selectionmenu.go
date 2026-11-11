// Package selectionmenu is the bar shown over text selected in another app:
// one row of the enabled assistants, copy and vocabulary. Choosing an
// assistant opens a second, smaller menu beneath the bar with the prompt
// rows, the question field and that assistant's own actions. It renders the
// actions it is given; the backend filters them, fills and opens URLs, and
// writes the vocabulary.
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

// Width and BarHeight size the bar, and the window while no menu is open:
// one row, as tall as a toolbar, wide enough for the English labels of both
// assistants, copy and vocabulary. Bar buttons do not shrink, so a longer
// label shows past the bar rather than being cut. MenuWidth is the second
// menu's width.
const (
	Width     = 400
	BarHeight = 37
	MenuWidth = 224

	barPadding = 4
	toolHeight = BarHeight - 2*barPadding
	menuRow    = 26
	menuPad    = 4
	menuGap    = 1
	stackGap   = 4
	notice     = 24
)

// Menu keeps the open assistant, the highlight and the question between
// frames. A new offer resets them.
type Menu struct {
	Question  string
	client    string
	highlight int
	refocus   bool
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

// own is the client's actions that the second menu lists after the prompts.
func (l layout) own(client string) []runtime.SelectionAction {
	var out []runtime.SelectionAction
	for _, action := range l.clients {
		if action.ClientID == client {
			out = append(out, action)
		}
	}
	return out
}

// opens reports whether choosing client has a second menu to show.
func (l layout) opens(client string) bool {
	return len(l.prompts) > 0 || l.ask != nil || len(l.own(client)) > 0
}

func (l layout) menuHeight(client string) int {
	rows := len(l.prompts) + len(l.own(client))
	if l.ask != nil {
		rows++
	}
	return 2*menuPad + rows*menuRow + max(0, rows-1)*menuGap
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

func clientName(id string) copy.Key {
	switch id {
	case runtime.SelectionClientClaude:
		return copy.SelectionClaude
	case runtime.SelectionClientChatGPT:
		return copy.SelectionChatGPT
	}
	return ""
}

func clientIcon(id string) *ui.SVG {
	if id == runtime.SelectionClientClaude {
		return icons.Anthropic
	}
	return icons.OpenAI
}

// Render draws the current offer, dark in either appearance, and returns
// the window height it needs: the bar, the open menu and a failure line.
func (m *Menu) Render(c *ui.Context, store runtime.Store) int {
	messages := store.Copy()
	offer := store.State.Selection
	if offer.Text != m.offer {
		m.offer, m.Question, m.client, m.highlight, m.failure = offer.Text, "", "", 0, ""
	}
	l := split(offer.Actions)
	var clients []string
	for _, id := range offer.Clients {
		if clientName(id) != "" && l.opens(id) {
			clients = append(clients, id)
		}
	}
	if !contains(clients, m.client) {
		m.client = ""
	}
	open := m.client != ""
	m.highlight = max(0, min(m.highlight, len(l.prompts)-1))
	tokens := theme.Apply(c)
	outer := c.Theme()
	scoped := *outer
	scoped.Dark = true
	scoped.Text, scoped.TextMuted, scoped.Border = tokens.PillText, tokens.PillMuted, tokens.PillBorder
	scoped.Surface, scoped.SurfaceHover, scoped.SurfacePressed = tokens.PillBorder, tokens.PillBorder, tokens.PillBorder
	c.SetTheme(&scoped)
	ui.Column(c).Fill().Gap(stackGap).AlignItems(ui.Start).TextColor(tokens.PillText).Children(func() {
		m.bar(c, store, tokens, messages, l, clients)
		if open {
			m.menu(c, store, tokens, messages, l)
		}
		if m.failure != "" {
			ui.Text(c, messages.Text(m.failure)).FontSize(theme.Caption).TextColor(tokens.PillMuted).Height(notice).Padding(4, 10).
				Radius(theme.RadiusMD).Background(tokens.PillSurface).SingleLine()
		}
	})
	c.SetTheme(outer)
	c.Root().Background(ui.Transparent)
	switch {
	case open && c.Shortcut(0, ui.KeyEscape):
		m.client = ""
	case open && len(l.prompts) > 0 && c.Shortcut(0, ui.KeyDown):
		m.highlight, m.refocus = min(m.highlight+1, len(l.prompts)-1), true
	case open && len(l.prompts) > 0 && c.Shortcut(0, ui.KeyUp):
		m.highlight, m.refocus = max(m.highlight-1, 0), true
	case c.Shortcut(0, ui.KeyEnter):
		switch {
		case open && len(l.prompts) > 0:
			m.run(store, l.prompts[m.highlight])
		case !open && l.copy != nil:
			m.run(store, *l.copy)
		}
	}
	height := BarHeight
	if m.client != "" {
		height += stackGap + l.menuHeight(m.client)
	}
	if m.failure != "" {
		height += stackGap + notice
	}
	return height
}

func contains(list []string, id string) bool {
	for _, item := range list {
		if item == id {
			return true
		}
	}
	return false
}

// bar is the first surface: the assistants, which open the second menu,
// then copy and vocabulary, which act at once.
func (m *Menu) bar(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, l layout, clients []string) {
	ui.Row(c).Width(Width).Height(BarHeight).Padding(barPadding).Gap(2).AlignItems(ui.Center).Radius(theme.RadiusLG).
		Background(tokens.PillSurface).Border(1, tokens.PillBorder).Children(func() {
		for _, id := range clients {
			if m.clientButton(c, tokens, messages, id) {
				if m.client == id {
					m.client = ""
				} else {
					m.client, m.highlight, m.failure = id, 0, ""
				}
			}
		}
		if len(clients) > 0 && (l.copy != nil || l.vocab != nil) {
			ui.Box(c).Width(1).Height(16).Margin(0, 3).Background(tokens.PillBorder)
		}
		if l.copy != nil && m.tool(c, tokens, messages, *l.copy, icons.Copy) {
			m.run(store, *l.copy)
		}
		if l.vocab != nil && m.tool(c, tokens, messages, *l.vocab, icons.BookOpen) {
			m.run(store, *l.vocab)
		}
	})
}

func (m *Menu) clientButton(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, id string) bool {
	name := messages.Text(clientName(id))
	chosen := m.client == id
	button := ui.ButtonBase(c.Key("client-"+id)).Height(toolHeight).Padding(0, 7).Gap(5).Radius(theme.RadiusMD).
		AlignItems(ui.Center).Shrink(0).Label(name).Selected(chosen)
	switch {
	case chosen:
		button.Background(tokens.PillText.Alpha(0.14))
	case button.Hovered():
		button.Background(tokens.PillText.Alpha(0.08))
	}
	button.Children(func() {
		ui.Icon(c, clientIcon(id)).FontSize(14).TextColor(tokens.PillText)
		ui.Text(c, name).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
		ui.Icon(c, icons.ChevronDown).FontSize(12).TextColor(tokens.PillMuted)
	})
	return button.Clicked()
}

// tool is a bar action that runs on click.
func (m *Menu) tool(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, icon *ui.SVG) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+action.ID)).Height(toolHeight).Padding(0, 7).Gap(5).Radius(theme.RadiusMD).
		AlignItems(ui.Center).Shrink(0).Label(name)
	if button.Hovered() {
		button.Background(tokens.PillText.Alpha(0.08))
	}
	button.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(tokens.PillMuted)
		ui.Text(c, name).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.PillText).SingleLine()
	})
	return button.Clicked()
}

// menu is the second surface for the chosen assistant. Clicking the
// question field only focuses it; submitting it asks.
func (m *Menu) menu(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, l layout) {
	ui.Column(c).Width(MenuWidth).Padding(menuPad).Gap(menuGap).Radius(theme.RadiusLG).Background(tokens.PillSurface).
		Border(1, tokens.PillBorder).Children(func() {
		for i, action := range l.prompts {
			if m.row(c, tokens, messages, action, i == m.highlight) {
				m.highlight = i
				m.run(store, action)
			}
		}
		if l.ask != nil {
			m.askField(c, store, tokens, messages, *l.ask)
		}
		for _, action := range l.own(m.client) {
			if m.row(c, tokens, messages, action, false) {
				m.run(store, action)
			}
		}
	})
}

// row is one menu entry; the highlighted prompt carries the Return glyph.
// Rows are keyed by the open assistant, so each menu appears anew and its
// highlighted row takes the keyboard from the assistant's button, and
// keeps it as the arrows move the highlight: Return runs that row.
func (m *Menu) row(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction, highlighted bool) bool {
	name := messages.Text(title(action.ID))
	button := ui.ButtonBase(c.Key("action-"+m.client+"-"+action.ID)).Height(menuRow).Padding(0, 8).Gap(8).Radius(theme.RadiusMD).
		Justify(ui.Start).AlignItems(ui.Center).Label(name).Selected(highlighted)
	ink := tokens.PillText
	switch {
	case highlighted:
		if m.refocus {
			button.Focus()
			m.refocus = false
		}
		button.AutoFocus().Background(tokens.Accent)
		ink = tokens.AccentText
	case button.Hovered():
		button.Background(tokens.PillText.Alpha(0.08))
	}
	button.Children(func() {
		ui.Text(c, name).FontSize(theme.Caption).TextColor(ink).SingleLine().Grow(1).Shrink(1)
		if highlighted {
			ui.Icon(c, icons.CornerDownLeft).FontSize(12).TextColor(ink).Label(messages.Text(copy.SelectionReturn))
		}
	})
	return button.Clicked()
}

func (m *Menu) askField(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, action runtime.SelectionAction) {
	name := messages.Text(title(action.ID))
	field := ui.TextInput(c.Key("action-"+m.client+"-ask"), &m.Question).Label(name).Placeholder(name).Height(menuRow).FontSize(theme.Caption).
		Background(tokens.PillText.Alpha(0.06)).Border(0, ui.Transparent).TextColor(tokens.PillText)
	if field.Submitted() && strings.TrimSpace(m.Question) != "" {
		m.run(store, action)
	}
}

func (m *Menu) run(store runtime.Store, action runtime.SelectionAction) {
	var err error
	switch action.Kind {
	case runtime.SelectionKindPrompt, runtime.SelectionKindClient:
		question := ""
		if action.ID == runtime.SelectionActionAsk {
			question = strings.TrimSpace(m.Question)
		}
		err = store.Actions.OpenSelection(action.ID, m.client, question)
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
