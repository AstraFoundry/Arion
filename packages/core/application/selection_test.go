package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"arion.local/core/domain"
)

func TestSelectionDefaultsStayOff(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	if prefs.Selection.Enabled || prefs.Version != PreferencesVersion {
		t.Fatalf("selection default %+v version %d", prefs.Selection, prefs.Version)
	}
	if len(prefs.Selection.Clients) != 2 || prefs.Selection.Clients[0].ID != SelectionClientClaude || prefs.Selection.Clients[1].ID != SelectionClientChatGPT {
		t.Fatalf("clients %+v", prefs.Selection.Clients)
	}
	for _, client := range prefs.Selection.Clients {
		template, _ := builtinClient(client.ID)
		if client.Enabled || client.Template != template.Template {
			t.Fatalf("client default %+v", client)
		}
	}
	if prefs.Selection.Actions != nil {
		t.Fatal("nil actions means the built-in list")
	}
	normalized, repairs := prefs.Normalize("darwin")
	if len(repairs) != 0 || normalized.Selection.Enabled {
		t.Fatalf("default selection repaired: %+v %v", normalized.Selection, repairs)
	}
	var old Preferences
	if err := json.Unmarshal([]byte(`{"version":1,"uiLanguage":"en"}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.Selection.Enabled {
		t.Fatal("a profile without selection enabled it")
	}
	loaded := DefaultPreferences("windows")
	if err := json.Unmarshal([]byte(`{"version":1,"uiLanguage":"en"}`), &loaded); err != nil {
		t.Fatal(err)
	}
	normalized, repairs = loaded.Normalize("windows")
	if normalized.Selection.Enabled || len(repairs) != 0 || normalized.Version != 1 {
		t.Fatalf("old profile changed selection: %+v %v", normalized.Selection, repairs)
	}
}

func TestSelectionNormalizeRepairsClientsAndActions(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	prefs.Selection.Enabled = true
	prefs.Selection.Clients = []SelectionClient{
		{ID: "doubao", Enabled: true, Template: "doubao://chat?q={{text}}"},
		{ID: SelectionClientClaude, Enabled: true, Template: "javascript:alert({{text}})"},
		{ID: SelectionClientChatGPT, Enabled: false, Template: SelectionTemplateChatGPT},
	}
	prefs.Selection.Actions = []SelectionAction{{ID: "shell", Kind: "prompt", Template: "rm {{shell}}", Enabled: true}}
	normalized, repairs := prefs.Normalize("darwin")
	if !normalized.Selection.Enabled || len(repairs) != 1 || repairs[0] != "selection" {
		t.Fatalf("repair %+v %v", normalized.Selection, repairs)
	}
	if normalized.Version != PreferencesVersion {
		t.Fatal("selection changed the preferences version")
	}
	clients := normalized.Selection.Clients
	if len(clients) != 2 || clients[0].ID != SelectionClientClaude || !clients[0].Enabled || clients[0].Template != SelectionTemplateClaude {
		t.Fatalf("claude %+v", clients)
	}
	if clients[1].Enabled || clients[1].Template != SelectionTemplateChatGPT {
		t.Fatalf("chatgpt %+v", clients[1])
	}
	if normalized.Selection.Actions[0].ID != SelectionActionTranslate || normalized.Selection.Actions[0].Template != SelectionPromptTranslate {
		t.Fatalf("actions %+v", normalized.Selection.Actions)
	}
}

func TestSelectionURLEncodingCapSchemeAndMissingClient(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	prefs.Selection.Enabled = true
	prefs.Selection.Clients[0].Enabled = true
	raw := "a b&中文?"
	got, err := prefs.Selection.SelectionURL(SelectionActionExplain, raw, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "claude://claude.ai/new?q=") {
		t.Fatalf("url %s", got)
	}
	query := mustSelectionQuery(t, got, "q")
	if query != strings.ReplaceAll(SelectionPromptExplain, "{{selection}}", raw) || strings.Contains(got, " ") || strings.Contains(got, "&中") {
		t.Fatalf("encoded query %q in %s", query, got)
	}
	long := strings.Repeat("á", ClaudeQueryLimit+25)
	capped, err := prefs.Selection.SelectionURL(SelectionActionExplain, long, "")
	if err != nil {
		t.Fatal(err)
	}
	cappedQuery := mustSelectionQuery(t, capped, "q")
	if utf8.RuneCountInString(cappedQuery) != ClaudeQueryLimit {
		t.Fatalf("claude query runes %d", utf8.RuneCountInString(cappedQuery))
	}
	prefs.Selection.Clients[0].Enabled = false
	prefs.Selection.Clients[1].Enabled = true
	chat, err := prefs.Selection.SelectionURL(SelectionActionAskChatGPT, long, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(chat, "codex://new?prompt=") {
		t.Fatal(chat)
	}
	chatQuery := mustSelectionQuery(t, chat, "prompt")
	if utf8.RuneCountInString(chatQuery) != ClaudeQueryLimit+25 {
		t.Fatalf("chatgpt was capped: %d %v", utf8.RuneCountInString(chatQuery), err)
	}
	for _, template := range []string{
		"javascript:alert({{text}})",
		"file:///tmp/{{text}}",
		"http://example.com/?q={{text}}",
		"https://user:pass@example.com/search?q={{text}}",
		"chatgpt://new?prompt={{text}}",
		"doubao://chat?q={{text}}",
		"feishu://applink?q={{text}}",
		"claude://claude.ai/new?q={{text}}&{{shell}}",
	} {
		if _, err := FillSelectionTemplate(template, "hello"); !errors.Is(err, ErrSelectionScheme) {
			t.Fatalf("%s: %v", template, err)
		}
	}
	allowed, err := FillSelectionTemplate("https://example.com/search?q={{text}}", "hello world")
	if err != nil || !strings.Contains(allowed, "q=hello%20world") {
		t.Fatalf("https %s %v", allowed, err)
	}
	prefs.Selection.Clients[1].Enabled = false
	if _, err := prefs.Selection.SelectionURL(SelectionActionTranslate, "hello", ""); !errors.Is(err, ErrSelectionClient) {
		t.Fatalf("missing client: %v", err)
	}
	if _, err := prefs.Selection.SelectionURL(SelectionActionAskChatGPT, "hello", ""); !errors.Is(err, ErrSelectionClient) {
		t.Fatalf("disabled chatgpt: %v", err)
	}
	if _, err := prefs.Selection.SelectionURL("missing", "hello", ""); !errors.Is(err, ErrSelectionAction) {
		t.Fatalf("missing action: %v", err)
	}
}

func TestSelectionMouseUpSwitchSecureCopyAndHotword(t *testing.T) {
	prefs := DefaultPreferences("darwin")
	reads := 0
	read := func() (SelectionReading, error) {
		reads++
		return SelectionReading{Text: "SelectionTerm"}, nil
	}
	if _, ok, err := OfferAfterMouseUp(prefs.Selection, read); ok || err != nil || reads != 0 {
		t.Fatalf("switch off offered a selection: %v %v reads=%d", ok, err, reads)
	}
	prefs.Selection.Enabled = true
	offer, ok, err := OfferAfterMouseUp(prefs.Selection, read)
	if err != nil || !ok || offer.Text != "SelectionTerm" || reads != 1 {
		t.Fatalf("offer %+v %v %v", offer, ok, err)
	}
	ids := map[string]bool{}
	for _, action := range offer.Actions {
		ids[action.ID] = true
		if action.Kind == SelectionKindPrompt || action.Kind == SelectionKindClient {
			t.Fatalf("prompt visible without a client: %+v", action)
		}
	}
	if !ids[SelectionActionCopy] || !ids[SelectionActionVocabulary] {
		t.Fatalf("actions %+v", offer.Actions)
	}
	secure := SelectionReading{Secure: true, Text: "secret"}
	var wrote, added string
	if err := CopySelection(secure, func(text string) error { wrote = text; return nil }); domain.CodeOf(err) != domain.CodeSecureFieldBlocked || wrote != "" {
		t.Fatalf("secure copy %q %v", wrote, err)
	}
	if err := AddSelectionHotword(secure, func(text string) error { added = text; return nil }); domain.CodeOf(err) != domain.CodeSecureFieldBlocked || added != "" {
		t.Fatalf("secure hotword %q %v", added, err)
	}
	if _, ok, err := OfferAfterMouseUp(prefs.Selection, func() (SelectionReading, error) { return secure, nil }); ok || err != nil {
		t.Fatal("secure field produced an offer")
	}
	if err := CopySelection(SelectionReading{Text: "SelectionTerm"}, func(text string) error { wrote = text; return nil }); err != nil || wrote != "SelectionTerm" {
		t.Fatalf("copy %q %v", wrote, err)
	}
	if err := AddSelectionHotword(SelectionReading{Text: "SelectionTerm"}, func(text string) error { added = text; return nil }); err != nil || added != "SelectionTerm" {
		t.Fatalf("hotword %q %v", added, err)
	}
}

func mustSelectionQuery(t *testing.T, raw, key string) string {
	t.Helper()
	query := strings.IndexByte(raw, '?')
	if query < 0 {
		t.Fatal(raw)
	}
	for _, part := range strings.Split(raw[query+1:], "&") {
		name, value, ok := strings.Cut(part, "=")
		if !ok || name != key {
			continue
		}
		decoded, err := decodeSelectionQuery(value)
		if err != nil {
			t.Fatal(err)
		}
		return decoded
	}
	t.Fatal(raw)
	return ""
}

func decodeSelectionQuery(value string) (string, error) {
	var decoded strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			decoded.WriteByte(value[i])
			continue
		}
		if i+2 >= len(value) {
			return "", errors.New("truncated selection escape")
		}
		hi, ok1 := decodeHex(value[i+1])
		lo, ok2 := decodeHex(value[i+2])
		if !ok1 || !ok2 {
			return "", errors.New("invalid selection escape")
		}
		decoded.WriteByte(hi<<4 | lo)
		i += 2
	}
	return decoded.String(), nil
}

func decodeHex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	default:
		return 0, false
	}
}
