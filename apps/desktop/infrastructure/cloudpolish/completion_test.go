package cloudpolish

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/ranking"
)

func TestGenericCompletionUsesIndependentRankingPrompt(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer synthetic-test-key" {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.Error(w, "bad request", 400)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		_, _ = io.WriteString(w, `{"id":"r","object":"response","status":"completed","output":[{"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"{\"pick\":\"b\"}"}]}]}`)
	}))
	defer srv.Close()
	s, err := NewAPICompletion(OpenAI, "synthetic-test-key", "model-test", srv.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.CloseCompletion()
	req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "synthetic", Generation: 1}, Revision: 1, Input: "beijing", Before: "这幅图的", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "a", Text: "北京", Source: "test"}, Start: 0, End: len("beijing")},
		{Candidate: domain.Candidate{ID: "b", Text: "背景", Source: "test"}, Start: 0, End: len("beijing")},
	}}
	out, err := (&ranking.Ranker{Chat: s}).RankKeyboard(context.Background(), req)
	if err != nil || len(out.Candidates) != 2 || out.Candidates[0].ID != "b" || out.Decision != "model" || received == nil {
		t.Fatal(out, err)
	}
	body, _ := json.Marshal(received)
	if !strings.Contains(string(body), "Pinyin") || strings.Contains(string(body), "speech") || strings.Contains(string(body), "acceptable") {
		t.Fatalf("wrong task prompt %s", body)
	}
}

func TestCompatibleChatCompletionAndTruncation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected wire path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"id":"synthetic","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"{\"pick\":\"c0\"}"},"finish_reason":"length"}]}`)
	}))
	defer srv.Close()
	s, err := NewAPICompletion(OpenAI, "local-test", "model-test", srv.URL, "chat")
	if err != nil {
		t.Fatal(err)
	}
	defer s.CloseCompletion()
	out, err := s.Complete(context.Background(), prompt.ChatRequest{Messages: []prompt.Message{{Role: "user", Content: "synthetic task"}}, MaxTokens: 48})
	if err != nil || !out.Truncated {
		t.Fatal("truncated API response accepted", out, err)
	}
}
