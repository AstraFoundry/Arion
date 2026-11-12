package application

import (
	"errors"
	"strings"
	"unicode/utf8"

	"arion.local/core/domain"
)

// Selection action kinds. A prompt or client row fills one allowlisted URL.
// Copy and vocabulary stay inside Arion.
const (
	SelectionKindPrompt     = "prompt"
	SelectionKindClient     = "client"
	SelectionKindCopy       = "copy"
	SelectionKindVocabulary = "vocabulary"
)

// Built-in selection clients. Both stay off until the user enables one.
const (
	SelectionClientClaude    = "claude"
	SelectionClientChatGPT   = "chatgpt"
	SelectionTemplateClaude  = "claude://claude.ai/new?q={{text}}"
	SelectionTemplateChatGPT = "codex://new?prompt={{text}}"
)

// Built-in selection actions, in menu order.
const (
	SelectionActionTranslate   = "translate-zh"
	SelectionActionExplain     = "explain"
	SelectionActionSummarize   = "summarize"
	SelectionActionRewrite     = "rewrite"
	SelectionActionAsk         = "ask"
	SelectionActionAskChatGPT  = "ask-chatgpt"
	SelectionActionCopy        = "copy"
	SelectionActionVocabulary  = "add-vocabulary"
	SelectionShortcutReturn    = "return"
	SelectionPlaceholderText   = "{{text}}"
	SelectionPlaceholderSelect = "{{selection}}"
	SelectionPlaceholderAsk    = "{{question}}"
)

// ClaudeQueryLimit is the documented Claude prefill cap, in Unicode
// characters of the query value before URL encoding.
const ClaudeQueryLimit = 14000

// Prompt bodies are sent to the client. They are not interface titles.
const (
	SelectionPromptTranslate = "Translate the following text into Simplified Chinese:\n\n{{selection}}"
	SelectionPromptExplain   = "Explain the following text:\n\n{{selection}}"
	SelectionPromptSummarize = "Summarize the key points of the following text:\n\n{{selection}}"
	SelectionPromptRewrite   = "Rewrite the following text:\n\n{{selection}}"
	SelectionPromptAsk       = "{{question}}\n\n{{selection}}"
)

var (
	// ErrSelectionScheme is a URL whose scheme is not https or an allowlisted client.
	ErrSelectionScheme = errors.New("selection url scheme is not allowed")
	// ErrSelectionClient is a prompt or client action with no enabled allowlisted client.
	ErrSelectionClient = errors.New("selection client is not enabled")
	// ErrSelectionAction is an unknown or disabled selection action.
	ErrSelectionAction = errors.New("selection action is unavailable")
	// ErrSelectionEmpty is a mouse-up with no selected text.
	ErrSelectionEmpty = errors.New("selection is empty")
)

// SelectionPreferences is the global selection popover. The zero Enabled
// value is off, so profiles written before the field existed stay off.
type SelectionPreferences struct {
	Enabled bool              `json:"enabled"`
	Clients []SelectionClient `json:"clients,omitempty"`
	// Actions nil means the built-in list.
	Actions []SelectionAction `json:"actions,omitempty"`
}

// SelectionClient is one summon target. Template is the built-in URL; a
// stored edit does not become a shell command.
type SelectionClient struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	Template string `json:"template"`
}

// SelectionAction is one popover row. Titles are copy-catalog keys of ID,
// not stored here.
type SelectionAction struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	ClientID string `json:"clientId,omitempty"`
	Template string `json:"template,omitempty"`
	Shortcut string `json:"shortcut,omitempty"`
	Enabled  bool   `json:"enabled"`
}

// SelectionReading is the focused field at mouse-up. Text is never logged.
type SelectionReading struct {
	Text   string
	Secure bool
}

// SelectionOffer is the in-memory popover input. It is absent when the
// master switch is off, the field is secure, or nothing is selected.
// Clients are the enabled client ids, in built-in order.
type SelectionOffer struct {
	Text    string
	Actions []SelectionAction
	Clients []string
}

// DefaultSelectionPreferences is off, with both clients present and disabled.
func DefaultSelectionPreferences() SelectionPreferences {
	return SelectionPreferences{Clients: builtinSelectionClients()}
}

func builtinSelectionClients() []SelectionClient {
	return []SelectionClient{
		{ID: SelectionClientClaude, Template: SelectionTemplateClaude},
		{ID: SelectionClientChatGPT, Template: SelectionTemplateChatGPT},
	}
}

func builtinSelectionActions() []SelectionAction {
	return []SelectionAction{
		{ID: SelectionActionTranslate, Kind: SelectionKindPrompt, Template: SelectionPromptTranslate, Shortcut: SelectionShortcutReturn, Enabled: true},
		{ID: SelectionActionExplain, Kind: SelectionKindPrompt, Template: SelectionPromptExplain, Enabled: true},
		{ID: SelectionActionSummarize, Kind: SelectionKindPrompt, Template: SelectionPromptSummarize, Enabled: true},
		{ID: SelectionActionRewrite, Kind: SelectionKindPrompt, Template: SelectionPromptRewrite, Enabled: true},
		{ID: SelectionActionAsk, Kind: SelectionKindPrompt, Template: SelectionPromptAsk, Enabled: true},
		{ID: SelectionActionAskChatGPT, Kind: SelectionKindClient, ClientID: SelectionClientChatGPT, Enabled: true},
		{ID: SelectionActionCopy, Kind: SelectionKindCopy, Enabled: true},
		{ID: SelectionActionVocabulary, Kind: SelectionKindVocabulary, Enabled: true},
	}
}

func builtinClient(id string) (SelectionClient, bool) {
	for _, client := range builtinSelectionClients() {
		if client.ID == id {
			return client, true
		}
	}
	return SelectionClient{}, false
}

func builtinAction(id string) (SelectionAction, bool) {
	for _, action := range builtinSelectionActions() {
		if action.ID == id {
			return action, true
		}
	}
	return SelectionAction{}, false
}

// normalize drops unknown clients, restores each built-in template, and
// keeps a missing Enabled false. A nil action list stays nil.
func (p SelectionPreferences) normalize() (SelectionPreferences, bool) {
	clients, clientsChanged := canonicalSelectionClients(p.Clients)
	actions, actionsChanged := canonicalSelectionActions(p.Actions)
	p.Clients = clients
	p.Actions = actions
	return p, clientsChanged || actionsChanged
}

func canonicalSelectionClients(in []SelectionClient) ([]SelectionClient, bool) {
	stored := map[string]SelectionClient{}
	changed := false
	seen := map[string]bool{}
	for _, client := range in {
		builtin, ok := builtinClient(client.ID)
		if !ok || seen[client.ID] {
			changed = true
			continue
		}
		seen[client.ID] = true
		if client.Template != builtin.Template {
			changed = true
			client.Template = builtin.Template
		}
		stored[client.ID] = client
	}
	out := builtinSelectionClients()
	for i, builtin := range out {
		if client, ok := stored[builtin.ID]; ok {
			out[i].Enabled = client.Enabled
		} else {
			changed = true
		}
	}
	if len(in) != len(out) {
		changed = true
	} else if !changed {
		for i := range in {
			if in[i] != out[i] {
				changed = true
			}
		}
	}
	return out, changed
}

func canonicalSelectionActions(in []SelectionAction) ([]SelectionAction, bool) {
	if in == nil {
		return nil, false
	}
	stored := map[string]bool{}
	changed := false
	seen := map[string]bool{}
	for _, action := range in {
		if _, ok := builtinAction(action.ID); !ok || seen[action.ID] {
			changed = true
			continue
		}
		seen[action.ID] = true
		stored[action.ID] = action.Enabled
	}
	out := builtinSelectionActions()
	for i, builtin := range out {
		enabled, ok := stored[builtin.ID]
		if !ok {
			changed = true
			continue
		}
		out[i].Enabled = enabled
	}
	if len(in) != len(out) {
		changed = true
	} else {
		for i := range in {
			repaired := out[i]
			if in[i] != repaired {
				changed = true
			}
		}
	}
	return out, changed
}

// ResolvedActions returns the stored list, or the built-in list when Actions is nil.
func (p SelectionPreferences) ResolvedActions() []SelectionAction {
	if p.Actions == nil {
		return builtinSelectionActions()
	}
	actions, _ := canonicalSelectionActions(p.Actions)
	return actions
}

// VisibleActions hides disabled rows. Prompt and client rows are hidden
// when no allowlisted client is enabled. Copy and vocabulary stay.
func (p SelectionPreferences) VisibleActions() []SelectionAction {
	_, hasDefault := p.defaultClient()
	out := make([]SelectionAction, 0, 8)
	for _, action := range p.ResolvedActions() {
		if !action.Enabled {
			continue
		}
		switch action.Kind {
		case SelectionKindPrompt:
			if !hasDefault {
				continue
			}
		case SelectionKindClient:
			if _, ok := p.enabledClient(action.ClientID); !ok {
				continue
			}
		}
		out = append(out, action)
	}
	return out
}

// EnabledClients lists the enabled built-in client ids. Claude precedes ChatGPT.
func (p SelectionPreferences) EnabledClients() []string {
	var out []string
	for _, builtin := range builtinSelectionClients() {
		if _, ok := p.enabledClient(builtin.ID); ok {
			out = append(out, builtin.ID)
		}
	}
	return out
}

// defaultClient is the first enabled built-in client. Claude precedes ChatGPT.
func (p SelectionPreferences) defaultClient() (SelectionClient, bool) {
	for _, builtin := range builtinSelectionClients() {
		if client, ok := p.enabledClient(builtin.ID); ok {
			return client, true
		}
	}
	return SelectionClient{}, false
}

func (p SelectionPreferences) enabledClient(id string) (SelectionClient, bool) {
	builtin, ok := builtinClient(id)
	if !ok {
		return SelectionClient{}, false
	}
	for _, client := range p.Clients {
		if client.ID == id && client.Enabled {
			builtin.Enabled = true
			return builtin, true
		}
	}
	return SelectionClient{}, false
}

// SelectionURL fills the action's built-in template. A prompt goes to
// clientID, or to the default client when clientID is empty; a client
// action goes only to its own client. It does not open the URL.
func (p SelectionPreferences) SelectionURL(actionID, clientID, selection, question string) (string, error) {
	action, ok := findSelectionAction(p.ResolvedActions(), actionID)
	if !ok || !action.Enabled {
		return "", ErrSelectionAction
	}
	var client SelectionClient
	var text string
	switch action.Kind {
	case SelectionKindPrompt:
		var has bool
		if clientID == "" {
			client, has = p.defaultClient()
		} else {
			client, has = p.enabledClient(clientID)
		}
		if !has {
			return "", ErrSelectionClient
		}
		filled, err := fillPrompt(action.Template, selection, question)
		if err != nil {
			return "", err
		}
		text = filled
	case SelectionKindClient:
		if clientID != "" && clientID != action.ClientID {
			return "", ErrSelectionAction
		}
		var has bool
		client, has = p.enabledClient(action.ClientID)
		if !has {
			return "", ErrSelectionClient
		}
		text = selection
	default:
		return "", ErrSelectionAction
	}
	if strings.TrimSpace(text) == "" {
		return "", ErrSelectionEmpty
	}
	if client.ID == SelectionClientClaude {
		text = truncateRunes(text, ClaudeQueryLimit)
	}
	return FillSelectionTemplate(client.Template, text)
}

func findSelectionAction(actions []SelectionAction, id string) (SelectionAction, bool) {
	for _, action := range actions {
		if action.ID == id {
			return action, true
		}
	}
	return SelectionAction{}, false
}

func fillPrompt(template, selection, question string) (string, error) {
	if template == "" || strings.Contains(stripSelectionPlaceholders(template), "{{") {
		return "", ErrSelectionAction
	}
	return strings.NewReplacer(SelectionPlaceholderSelect, selection, SelectionPlaceholderAsk, question).Replace(template), nil
}

func stripSelectionPlaceholders(template string) string {
	return strings.NewReplacer(SelectionPlaceholderText, "", SelectionPlaceholderSelect, "", SelectionPlaceholderAsk, "").Replace(template)
}

// FillSelectionTemplate substitutes {{text}} once and refuses any scheme
// other than https, claude, or codex. The substitution is percent-encoded.
// The function does not launch a process.
func FillSelectionTemplate(template, text string) (string, error) {
	if strings.Count(template, SelectionPlaceholderText) != 1 || strings.Contains(stripSelectionPlaceholders(template), "{{") {
		return "", ErrSelectionScheme
	}
	raw := strings.ReplaceAll(template, SelectionPlaceholderText, encodeSelectionQuery(text))
	return validateSelectionURL(raw)
}

// ValidateSelectionURL accepts an absolute https URL or an allowlisted
// client scheme. The error never includes the URL.
func ValidateSelectionURL(raw string) error {
	_, err := validateSelectionURL(raw)
	return err
}

func validateSelectionURL(raw string) (string, error) {
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok || !allowedSelectionScheme(scheme) || strings.ContainsAny(raw, " \t\r\n\\") {
		return "", ErrSelectionScheme
	}
	authority := rest
	if cut := strings.IndexAny(authority, "/?#"); cut >= 0 {
		authority = authority[:cut]
	}
	if authority == "" || strings.EqualFold(scheme, "https") && strings.Contains(authority, "@") {
		return "", ErrSelectionScheme
	}
	return raw, nil
}

func allowedSelectionScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "https", "claude", "codex":
		return true
	default:
		return false
	}
}

func encodeSelectionQuery(text string) string {
	var encoded strings.Builder
	for i := 0; i < len(text); i++ {
		c := text[i]
		if selectionUnescaped(c) {
			encoded.WriteByte(c)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hexDigits[c>>4])
		encoded.WriteByte(hexDigits[c&15])
	}
	return encoded.String()
}

func selectionUnescaped(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~'
}

const hexDigits = "0123456789ABCDEF"

func truncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	count := 0
	for i := range text {
		if count == limit {
			return text[:i]
		}
		count++
	}
	return text
}

// OfferAfterMouseUp reads the focused selection only while the master
// switch is on. A secure field or an empty selection produces no offer.
// The selected text is not retained in that case, and a read error is
// returned without an offer.
func OfferAfterMouseUp(prefs SelectionPreferences, read func() (SelectionReading, error)) (SelectionOffer, bool, error) {
	if !prefs.Enabled || read == nil {
		return SelectionOffer{}, false, nil
	}
	reading, err := read()
	if err != nil {
		return SelectionOffer{}, false, err
	}
	if reading.Secure || strings.TrimSpace(reading.Text) == "" {
		return SelectionOffer{}, false, nil
	}
	return SelectionOffer{Text: reading.Text, Actions: prefs.VisibleActions(), Clients: prefs.EnabledClients()}, true, nil
}

// CopySelection writes text with the existing clipboard writer. A secure
// field is refused and nothing is written.
func CopySelection(reading SelectionReading, write func(string) error) error {
	if reading.Secure {
		return domain.Fail(domain.CodeSecureFieldBlocked, "secure field")
	}
	if write == nil || strings.TrimSpace(reading.Text) == "" {
		return ErrSelectionEmpty
	}
	return write(reading.Text)
}

// AddSelectionHotword saves the selection through AddHotword. A secure
// field is refused before the vocabulary store is called.
func AddSelectionHotword(reading SelectionReading, add func(string) error) error {
	if reading.Secure {
		return domain.Fail(domain.CodeSecureFieldBlocked, "secure field")
	}
	if add == nil || strings.TrimSpace(reading.Text) == "" {
		return ErrSelectionEmpty
	}
	return add(reading.Text)
}
