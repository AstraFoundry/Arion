package cloudpolish

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/internal/account"
)

type memorySecrets struct {
	mu    sync.Mutex
	items map[string]string
	puts  int
}

func (m *memorySecrets) Put(accountName, v string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items == nil {
		m.items = map[string]string{}
	}
	m.items[accountName] = v
	m.puts++
	return nil
}

func (m *memorySecrets) Get(accountName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[accountName]
	if !ok {
		return "", account.ErrNotFound
	}
	return v, nil
}

func (m *memorySecrets) Delete(accountName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, accountName)
	return nil
}

func (m *memorySecrets) credential(t *testing.T) Credential {
	t.Helper()
	raw, err := m.Get(credentialAccountFor(m.activeProvider(t)))
	if err != nil {
		t.Fatal(err)
	}
	var c Credential
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func (m *memorySecrets) activeProvider(t *testing.T) string {
	t.Helper()
	raw, err := m.Get(activeAccount())
	if err != nil {
		t.Fatal(err)
	}
	var c Credential
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatal(err)
	}
	return c.Provider
}

func newTestService(secrets SecretStore, base string) *Service {
	s := New(secrets, func(string) error { return nil }, nil)
	s.base = base
	return s
}

// An OpenAI-compatible fake: the models list and one Responses answer.
func keyServer(t *testing.T, key string, bodies *[]map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"message":"bad key"}}`)
			return
		}
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"model-b"},{"id":"model-a"}]}`)
		case "/responses":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			*bodies = append(*bodies, body)
			text := " Polished. "
			payload := lastCloudPayload(t, body)
			if hypotheses, ok := payload["hypotheses"].([]any); ok {
				choices := map[string]string{}
				for _, item := range hypotheses {
					h := item.(map[string]any)
					choice := "keep"
					if h["status"] == "unique" {
						choice = "replace"
					}
					choices[h["id"].(string)] = choice
				}
				raw, _ := json.Marshal(choices)
				text = string(raw)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "r", "object": "response", "status": "completed", "output": []any{map[string]any{"type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": text}}}}})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestConnectKeyListsModelsAndPolishes(t *testing.T) {
	var bodies []map[string]any
	srv := keyServer(t, "sk-good", &bodies)
	defer srv.Close()
	secrets := &memorySecrets{}
	s := newTestService(secrets, srv.URL)
	if err := s.ConnectKey(OpenAI, "  sk-good "); err != nil {
		t.Fatal(err)
	}
	st := s.Status()
	if st.Phase != Connected || st.Provider != OpenAI || strings.Join(st.Models, ",") != "model-a,model-b" || st.Failure != "" {
		t.Fatalf("status %+v", st)
	}
	if c := secrets.credential(t); c.Provider != OpenAI || c.APIKey != "sk-good" {
		t.Fatalf("stored %+v", c)
	}
	if s.Ready() {
		t.Fatal("ready without a model")
	}
	if _, err := s.Rewrite(context.Background(), ports.RewriteRequest{Text: "x"}); domain.CodeOf(err) != domain.CodeModelMissing {
		t.Fatalf("rewrite without a model: %v", err)
	}
	s.SetModel("model-a")
	result, err := s.Rewrite(context.Background(), ports.RewriteRequest{Text: "检查 write value。", Scene: "email", Terms: []string{"writeValue"},
		Context: domain.RewriteContext{Screen: "writeValue SCREEN_CANARY", Before: "BEFORE_CANARY", After: "AFTER_CANARY", History: []string{"HISTORY_CANARY"}}})
	if err != nil || result.Text != "Polished." || result.Template != "cloud+"+prompt.SelectedVersion {
		t.Fatalf("result %+v %v", result, err)
	}
	if len(bodies) != 2 {
		t.Fatalf("want selector and formatter, got %d", len(bodies))
	}
	selected := lastCloudPayload(t, bodies[0])
	contextData := selected["context"].(map[string]any)
	for key, want := range map[string]string{"Screen": "writeValue SCREEN_CANARY", "Before": "BEFORE_CANARY", "After": "AFTER_CANARY", "History": "HISTORY_CANARY", "Terms": "writeValue"} {
		if contextData[key] != want {
			t.Errorf("%s = %v", key, contextData[key])
		}
	}
	formatted := lastCloudPayload(t, bodies[1])
	if formatted["source"] != "检查 writeValue。" {
		t.Fatalf("formatter source %v", formatted)
	}
	raw, _ := json.Marshal(bodies[1])
	if strings.Contains(string(raw), "CANARY") {
		t.Fatal("context leaked to formatter")
	}
	if bodies[0]["model"] != "model-a" {
		t.Fatalf("model %v", bodies[0]["model"])
	}
}

func TestRejectedKeyIsNotStored(t *testing.T) {
	var bodies []map[string]any
	srv := keyServer(t, "sk-good", &bodies)
	defer srv.Close()
	secrets := &memorySecrets{}
	s := newTestService(secrets, srv.URL)
	if err := s.ConnectKey(Anthropic, ""); CodeOf(err) != CodeRejected {
		t.Fatalf("empty key: %v", err)
	}
	if err := s.ConnectKey(OpenAI, "sk-bad"); CodeOf(err) != CodeRejected {
		t.Fatalf("bad key: %v", err)
	}
	if st := s.Status(); st.Phase != Disconnected || st.Failure != CodeRejected || secrets.puts != 0 {
		t.Fatalf("status %+v puts %d", st, secrets.puts)
	}
}

func TestFailedKeyKeepsExistingConnection(t *testing.T) {
	var bodies []map[string]any
	srv := keyServer(t, "sk-good", &bodies)
	defer srv.Close()
	secrets := &memorySecrets{}
	s := newTestService(secrets, srv.URL)
	if err := s.ConnectKey(OpenAI, "sk-good"); err != nil {
		t.Fatal(err)
	}
	_ = s.ConnectKey(OpenAI, "sk-bad")
	if st := s.Status(); st.Phase != Connected || st.Provider != OpenAI || st.Failure != CodeRejected || len(st.Models) != 2 {
		t.Fatalf("status %+v", st)
	}
	if c := secrets.credential(t); c.APIKey != "sk-good" {
		t.Fatalf("stored %+v", c)
	}
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get(credentialAccountFor(OpenAI)); err != account.ErrNotFound {
		t.Fatalf("credential kept: %v", err)
	}
	if _, err := secrets.Get(activeAccount()); err != account.ErrNotFound {
		t.Fatalf("pointer kept: %v", err)
	}
	if st := s.Status(); st.Phase != Disconnected || st.Models != nil || s.Ready() {
		t.Fatalf("status %+v", st)
	}
}

func TestLoadIsLocal(t *testing.T) {
	secrets := &memorySecrets{}
	s := newTestService(secrets, "http://127.0.0.1:1")
	s.Load()
	if st := s.Status(); st.Phase != Disconnected || st.Failure != "" {
		t.Fatalf("empty store %+v", st)
	}
	if err := secrets.Put(activeAccount(), `{"provider":"deepseek","apiKey":"k"}`); err != nil {
		t.Fatal(err)
	}
	s.Load()
	if st := s.Status(); st.Phase != Connected || st.Provider != DeepSeek || !strings.Contains(strings.Join(st.Saved, ","), DeepSeek) {
		t.Fatalf("loaded %+v", st)
	}
	migrated, err := secrets.Get(credentialAccountFor(DeepSeek))
	if err != nil || !strings.Contains(migrated, `"apiKey":"k"`) {
		t.Fatalf("migrated %q %v", migrated, err)
	}
	pointer, err := secrets.Get(activeAccount())
	if err != nil || strings.Contains(pointer, "apiKey") {
		t.Fatalf("pointer still holds the key: %s", pointer)
	}
	secrets = &memorySecrets{}
	if err := secrets.Put(activeAccount(), `{"provider":"elsewhere","apiKey":"k"}`); err != nil {
		t.Fatal(err)
	}
	s = newTestService(secrets, "")
	s.Load()
	if st := s.Status(); st.Phase != Disconnected || st.Failure != CodeStore {
		t.Fatalf("unknown provider %+v", st)
	}
}

// A legacy single credential is copied to that provider. Connecting another
// provider, switching back, and disconnecting leave the other key in place.
func TestSwitchingProvidersKeepsEachAPIKey(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if r.URL.Path != "/models" || (key != "sk-ds" && key != "sk-or") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"message":"bad key"}}`)
			return
		}
		seen = append(seen, key)
		id := "deepseek-chat"
		if key == "sk-or" {
			id = "router-model"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": id}}})
	}))
	defer srv.Close()
	secrets := &memorySecrets{}
	if err := secrets.Put(activeAccount(), `{"provider":"deepseek","apiKey":"sk-ds"}`); err != nil {
		t.Fatal(err)
	}
	s := newTestService(secrets, srv.URL)
	s.Load()
	s.SetModel("deepseek-chat")
	if c := s.Status(); c.Provider != DeepSeek || c.Model != "deepseek-chat" {
		t.Fatalf("migrated status %+v", c)
	}
	if err := s.ConnectKey(OpenRouter, "sk-or"); err != nil {
		t.Fatal(err)
	}
	deep, err := secrets.Get(credentialAccountFor(DeepSeek))
	if err != nil || !strings.Contains(deep, `"apiKey":"sk-ds"`) || !strings.Contains(deep, `"model":"deepseek-chat"`) {
		t.Fatalf("deepseek key lost: %s %v", deep, err)
	}
	route, err := secrets.Get(credentialAccountFor(OpenRouter))
	if err != nil || !strings.Contains(route, `"apiKey":"sk-or"`) {
		t.Fatalf("openrouter key: %s %v", route, err)
	}
	pointer, err := secrets.Get(activeAccount())
	if err != nil || strings.Contains(pointer, "sk-") {
		t.Fatalf("pointer holds a key: %s", pointer)
	}
	s.SetModel("router-model")
	seen = nil
	if err := s.Activate(DeepSeek); err != nil {
		t.Fatal(err)
	}
	if st := s.Status(); st.Phase != Connected || st.Provider != DeepSeek || st.Model != "deepseek-chat" || strings.Join(st.Models, ",") != "deepseek-chat" {
		t.Fatalf("switched back %+v", st)
	}
	if len(seen) == 0 || seen[len(seen)-1] != "sk-ds" {
		t.Fatalf("list used %v", seen)
	}
	if route, err = secrets.Get(credentialAccountFor(OpenRouter)); err != nil || !strings.Contains(route, `"apiKey":"sk-or"`) || !strings.Contains(route, `"model":"router-model"`) {
		t.Fatalf("openrouter key lost while using deepseek: %s %v", route, err)
	}
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get(credentialAccountFor(DeepSeek)); err != account.ErrNotFound {
		t.Fatalf("deepseek key still stored: %v", err)
	}
	if route, err = secrets.Get(credentialAccountFor(OpenRouter)); err != nil || !strings.Contains(route, "sk-or") {
		t.Fatalf("disconnect removed openrouter: %s %v", route, err)
	}
	if st := s.Status(); st.Phase != Disconnected || strings.Join(st.Saved, ",") != OpenRouter {
		t.Fatalf("after disconnect %+v", st)
	}
	seen = nil
	if err := s.Activate(OpenRouter); err != nil {
		t.Fatal(err)
	}
	if st := s.Status(); st.Phase != Connected || st.Provider != OpenRouter || st.Model != "router-model" || seen[len(seen)-1] != "sk-or" {
		t.Fatalf("openrouter restore %+v keys %v", st, seen)
	}
	if err := s.ConnectKey(Anthropic, "sk-bad"); CodeOf(err) != CodeRejected {
		t.Fatalf("bad key: %v", err)
	}
	if st := s.Status(); st.Phase != Connected || st.Provider != OpenRouter {
		t.Fatalf("rejected key replaced the connection %+v", st)
	}
	if route, err = secrets.Get(credentialAccountFor(OpenRouter)); err != nil || !strings.Contains(route, "sk-or") {
		t.Fatalf("rejected key removed openrouter: %s %v", route, err)
	}
	if _, err := secrets.Get(credentialAccountFor(Anthropic)); err != account.ErrNotFound {
		t.Fatalf("rejected key was stored: %v", err)
	}
}

// ChatGPT sign-in uses the same store as an API key. It must not delete a
// key that was already saved, and switching back must not delete the sign-in.
func TestChatGPTConnectKeepsAPIKeys(t *testing.T) {
	secrets := &memorySecrets{}
	if err := secrets.Put(activeAccount(), `{"provider":"deepseek","apiKey":"sk-ds"}`); err != nil {
		t.Fatal(err)
	}
	s := newTestService(secrets, "http://127.0.0.1:1")
	s.Load()
	s.mu.Lock()
	s.generation = 1
	generation := s.generation
	s.mu.Unlock()
	c := Credential{Provider: ChatGPT, RefreshToken: "rt", AccountID: "acct", Email: "a@example.com"}
	if err := s.connect(context.Background(), generation, c, &connection{models: []string{"gpt"}}); err != nil {
		t.Fatal(err)
	}
	if st := s.Status(); st.Phase != Connected || st.Provider != ChatGPT || st.Email != "a@example.com" {
		t.Fatalf("status %+v", st)
	}
	deep, err := secrets.Get(credentialAccountFor(DeepSeek))
	if err != nil || !strings.Contains(deep, "sk-ds") {
		t.Fatalf("sign-in dropped the API key: %s %v", deep, err)
	}
	if err := s.Activate(DeepSeek); err != nil {
		t.Fatal(err)
	}
	gpt, err := secrets.Get(credentialAccountFor(ChatGPT))
	if err != nil || !strings.Contains(gpt, `"refreshToken":"rt"`) {
		t.Fatalf("switching back dropped ChatGPT: %s %v", gpt, err)
	}
	if st := s.Status(); st.Provider != DeepSeek {
		t.Fatalf("status %+v", st)
	}
}

func jwt(claims map[string]any) string {
	raw, _ := json.Marshal(claims)
	return "e30." + base64.RawURLEncoding.EncodeToString(raw) + ".sig"
}

func TestCredentialFromTokens(t *testing.T) {
	id := jwt(map[string]any{"email": "a@example.com", "https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "acct"}})
	c, err := credentialFrom(tokens{IDToken: id, AccessToken: "opaque", RefreshToken: "rt"})
	if err != nil || c != (Credential{Provider: ChatGPT, RefreshToken: "rt", AccountID: "acct", Email: "a@example.com"}) {
		t.Fatalf("credential %+v %v", c, err)
	}
	if _, err := credentialFrom(tokens{IDToken: jwt(map[string]any{"email": "a@example.com"}), RefreshToken: "rt"}); CodeOf(err) != CodeNoSubscription {
		t.Fatalf("no account: %v", err)
	}
	if _, err := credentialFrom(tokens{IDToken: id}); CodeOf(err) != CodeSignIn {
		t.Fatalf("no refresh token: %v", err)
	}
}

func TestAuthorizeURL(t *testing.T) {
	u, err := url.Parse(chatgptAuthorizeURL(chatgptIssuer, "st", "ch"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "auth.openai.com" || u.Path != "/oauth/authorize" || q.Get("client_id") != chatgptClientID || q.Get("redirect_uri") != "http://localhost:1455/auth/callback" ||
		q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") != "ch" || q.Get("state") != "st" || !strings.Contains(q.Get("scope"), "offline_access") {
		t.Fatalf("authorize URL %s", u)
	}
}

// The ChatGPT sign-in end to end against fakes: the browser follows the
// redirect, the code is exchanged, the refresh token is stored, and a later
// launch trades it for an access token before listing models.
func TestChatGPTSignInAndModels(t *testing.T) {
	id := jwt(map[string]any{"email": "a@example.com", "https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "acct"}})
	var mu sync.Mutex
	var forms []url.Values
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mu.Lock()
		forms = append(forms, r.PostForm)
		mu.Unlock()
		refresh := "rt-1"
		if r.PostForm.Get("grant_type") == "refresh_token" {
			refresh = "rt-2"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id_token": id, "access_token": "at", "refresh_token": refresh})
	}))
	defer issuer.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/responses" {
			if r.Header.Get("originator") != "arion" || r.Header.Get("ChatGPT-Account-Id") != "acct" {
				t.Errorf("responses headers %v", r.Header)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("responses body: %v", err)
			}
			reasoning, _ := body["reasoning"].(map[string]any)
			if reasoning["effort"] != "none" {
				t.Errorf("codex reasoning %+v", body["reasoning"])
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"Done.\",\"output_index\":0,\"content_index\":0}\n\n"+
				"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"Done.\"}]}]}}\n\n")
			return
		}
		if r.URL.Path != "/models" || r.Header.Get("Authorization") != "Bearer at" || r.Header.Get("ChatGPT-Account-Id") != "acct" || r.URL.Query().Get("client_version") == "" {
			t.Errorf("models request %s %v", r.URL, r.Header)
		}
		_, _ = io.WriteString(w, `{"models":[{"slug":"m-1","display_name":"M1"},{"slug":"m-hidden","visibility":"hide"}]}`)
	}))
	defer backend.Close()

	secrets := &memorySecrets{}
	s := newTestService(secrets, backend.URL)
	s.issuer = issuer.URL
	s.browser = func(raw string) error {
		u, _ := url.Parse(raw)
		go func() {
			res, err := http.Get("http://127.0.0.1:1455/auth/callback?code=c&state=" + url.QueryEscape(u.Query().Get("state")))
			if err == nil {
				res.Body.Close()
			}
		}()
		return nil
	}
	if err := s.SignInChatGPT(); err != nil {
		t.Fatal(err)
	}
	if c := secrets.credential(t); c.Provider != ChatGPT || c.AccountID != "acct" || c.Email != "a@example.com" {
		t.Fatalf("stored %+v", c)
	}
	st := s.Status()
	if st.Phase != Connected || st.Email != "a@example.com" || strings.Join(st.Models, ",") != "m-1" {
		t.Fatalf("status %+v", st)
	}
	s.SetModel("m-1")
	if result, err := s.Rewrite(context.Background(), ports.RewriteRequest{Text: "done", Scene: "chat"}); err != nil || result.Text != "Done." {
		t.Fatalf("rewrite %+v %v", result, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(forms) != 2 || forms[0].Get("grant_type") != "authorization_code" || forms[0].Get("code_verifier") == "" || forms[1].Get("refresh_token") != "rt-1" {
		t.Fatalf("token requests %v", forms)
	}
	// The refresh rotated the token; the rotation is stored.
	if c := secrets.credential(t); c.RefreshToken != "rt-2" {
		t.Fatalf("rotation not stored: %+v", c)
	}
}

func TestCancelledSignInRestoresState(t *testing.T) {
	secrets := &memorySecrets{}
	s := newTestService(secrets, "")
	opened := make(chan struct{})
	s.browser = func(string) error { close(opened); return nil }
	done := make(chan error, 1)
	go func() { done <- s.SignInChatGPT() }()
	<-opened
	if st := s.Status(); st.Phase != SigningIn {
		t.Fatalf("status %+v", st)
	}
	s.Cancel()
	if err := <-done; CodeOf(err) != CodeCancelled {
		t.Fatalf("cancel: %v", err)
	}
	if st := s.Status(); st.Phase != Disconnected || st.Failure != "" || secrets.puts != 0 {
		t.Fatalf("status %+v", st)
	}
}

// Read the task data from the last Responses input turn, excluding examples.
func lastCloudPayload(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	payload, err := gatewayPayload(body, "responses")
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestRecipeDisablesThinking(t *testing.T) {
	cases := []struct {
		provider, model, format, path string
		check                         func(*testing.T, map[string]any)
	}{
		{OpenAI, "gpt", "responses", "/responses", expectReasoningNone},
		{OpenAI, "gpt", "chat", "/chat/completions", expectChatReasoningNone},
		{Anthropic, "claude", "", "/messages", expectThinkingDisabled},
		{Google, "gemini-2.5-flash", "", "/models/gemini-2.5-flash:generateContent", expectThinkingBudgetZero},
		{DeepSeek, "deepseek-flash", "chat", "/chat/completions", expectThinkingDisabled},
		{DeepSeek, "deepseek-flash", "responses", "/responses", expectReasoningNone},
		{OpenRouter, "vendor/model", "responses", "/responses", expectReasoningNone},
		{OpenRouter, "vendor/model", "chat", "/chat/completions", expectReasoningNone},
	}
	for _, tc := range cases {
		t.Run(tc.provider+"/"+tc.model+"/"+tc.format, func(t *testing.T) {
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					t.Errorf("path %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("body: %v", err)
				}
				_ = json.NewEncoder(w).Encode(thinkingOffResponse(r.URL.Path))
			}))
			defer srv.Close()
			s, err := NewAPICompletion(tc.provider, "synthetic-key", tc.model, srv.URL, tc.format)
			if err != nil {
				t.Fatal(err)
			}
			defer s.CloseCompletion()
			res, err := s.Rewrite(context.Background(), ports.RewriteRequest{Text: "今天天气很好。", Scene: "chat", Punctuation: "omit"})
			if err != nil || res.Text != "今天天气很好。" {
				t.Fatalf("res=%+v err=%v", res, err)
			}
			tc.check(t, body)
		})
	}
}

func TestEnabledThinkingIsNotOverridden(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(gatewayResponse("chat", "ok", ""))
	}))
	defer srv.Close()
	s, err := NewAPICompletion(DeepSeek, "synthetic-key", "deepseek-flash", srv.URL, "chat")
	if err != nil {
		t.Fatal(err)
	}
	defer s.CloseCompletion()
	_, err = s.Complete(context.Background(), prompt.ChatRequest{
		Messages:      []prompt.Message{{Role: "user", Content: "hi"}},
		MaxTokens:     16,
		AllowThinking: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body["thinking"]; ok {
		t.Fatalf("explicit thinking was disabled: %+v", body["thinking"])
	}
	if _, ok := body["reasoning_effort"]; ok {
		t.Fatalf("reasoning effort sent: %+v", body["reasoning_effort"])
	}
}

func thinkingOffResponse(path string) map[string]any {
	switch {
	case strings.Contains(path, "generateContent"):
		return map[string]any{"candidates": []any{map[string]any{
			"content":      map[string]any{"role": "model", "parts": []any{map[string]any{"text": "今天天气很好。"}}},
			"finishReason": "STOP",
		}}}
	case strings.HasSuffix(path, "/messages"):
		return map[string]any{
			"id": "m", "type": "message", "role": "assistant", "stop_reason": "end_turn",
			"content": []any{map[string]any{"type": "text", "text": "今天天气很好。"}},
		}
	case strings.HasSuffix(path, "/responses"):
		return gatewayResponse("responses", "今天天气很好。", "")
	default:
		return gatewayResponse("chat", "今天天气很好。", "")
	}
}

func expectReasoningNone(t *testing.T, body map[string]any) {
	t.Helper()
	reasoning, _ := body["reasoning"].(map[string]any)
	if reasoning["effort"] != "none" {
		t.Fatalf("reasoning effort: %+v", body["reasoning"])
	}
}

func expectChatReasoningNone(t *testing.T, body map[string]any) {
	t.Helper()
	if body["reasoning_effort"] != "none" {
		t.Fatalf("reasoning_effort: %+v", body["reasoning_effort"])
	}
}

func expectThinkingDisabled(t *testing.T, body map[string]any) {
	t.Helper()
	thinking, _ := body["thinking"].(map[string]any)
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking: %+v", body["thinking"])
	}
	if _, ok := body["reasoning_effort"]; ok {
		t.Fatalf("reasoning_effort must stay unset: %+v", body["reasoning_effort"])
	}
}

func expectThinkingBudgetZero(t *testing.T, body map[string]any) {
	t.Helper()
	if thinkingConfig(t, body)["thinkingBudget"] != float64(0) {
		t.Fatalf("thinkingConfig: %+v", thinkingConfig(t, body))
	}
}

func thinkingConfig(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	cfg, _ := body["generationConfig"].(map[string]any)
	thinking, _ := cfg["thinkingConfig"].(map[string]any)
	if thinking == nil {
		t.Fatalf("missing thinkingConfig: %+v", body["generationConfig"])
	}
	return thinking
}

// A provider starts on its fast tier when no model was chosen: the newest
// undated model of the first tier with one; nothing when no tier matches.
func TestDefaultModel(t *testing.T) {
	for _, c := range []struct {
		provider string
		models   []string
		want     string
	}{
		{OpenAI, []string{"gpt-6-sol", "gpt-5.6-luna", "gpt-6-luna-2026-07-01", "gpt-6-luna", "gpt-6-mini"}, "gpt-6-luna"},
		{ChatGPT, []string{"gpt-6-sol", "gpt-6-luna"}, "gpt-6-luna"},
		{OpenAI, []string{"gpt-6-sol", "gpt-5-mini", "gpt-5-mini-realtime-preview"}, "gpt-5-mini"},
		{Anthropic, []string{"claude-opus-5-5", "claude-haiku-4-5-20251001", "claude-haiku-4-5", "claude-sonnet-5-5"}, "claude-haiku-4-5"},
		{DeepSeek, []string{"deepseek-v4-pro", "deepseek-flash"}, "deepseek-flash"},
		{DeepSeek, []string{"deepseek-reasoner", "deepseek-chat"}, "deepseek-chat"},
		{Google, []string{"gemini-3.6-flash", "gemini-3.5-flash-lite", "gemini-3.1-flash-lite-preview"}, "gemini-3.5-flash-lite"},
		{OpenRouter, []string{"anthropic/claude-haiku-4.5", "openai/gpt-6-luna", "meta/llama"}, "openai/gpt-6-luna"},
		{OpenAI, []string{"model-a", "model-b"}, ""},
	} {
		if got := DefaultModel(c.provider, c.models); got != c.want {
			t.Errorf("%s %v: %q, want %q", c.provider, c.models, got, c.want)
		}
	}
	for _, p := range knownProviders() {
		if len(tiers[p]) == 0 {
			t.Errorf("%s has no default tier", p)
		}
	}
}

// Connecting a provider whose list has a default-tier model makes it the
// model, remembered with the key; a chosen model is kept.
func TestConnectAdoptsDefaultModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"id":"gpt-6-sol"},{"id":"gpt-6-luna"}]}`)
	}))
	defer srv.Close()
	secrets := &memorySecrets{}
	s := newTestService(secrets, srv.URL)
	if err := s.ConnectKey(OpenAI, "sk"); err != nil {
		t.Fatal(err)
	}
	if st := s.Status(); st.Model != "gpt-6-luna" || !s.Ready() {
		t.Fatalf("status %+v", st)
	}
	if c := secrets.credential(t); c.Model != "gpt-6-luna" {
		t.Fatalf("stored %+v", c)
	}
	s.SetModel("gpt-6-sol")
	s.ListModels()
	if st := s.Status(); st.Model != "gpt-6-sol" {
		t.Fatalf("chosen model replaced: %+v", st)
	}
}
