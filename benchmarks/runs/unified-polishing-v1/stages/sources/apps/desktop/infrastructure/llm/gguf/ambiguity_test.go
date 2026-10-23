package gguf

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/rewrite"
)

func TestAmbiguousOnlySkipsSelectorAndFormatsUnchangedSource(t *testing.T) {
	testSpellingGenerationCalls(t, "检查 read config。", "readConfig ReadConfig", 0, "检查 read config。")
}

func TestMixedAmbiguityStillRunsSelectorForRealDecision(t *testing.T) {
	testSpellingGenerationCalls(t, "检查 read config 和 write value。", "readConfig ReadConfig writeValue", 1, "检查 read config 和 writeValue。")
}

func testSpellingGenerationCalls(t *testing.T, source, screen string, wantSelectors int, wantFormatted string) {
	t.Helper()
	pipeline, err := rewrite.Load()
	if err != nil {
		t.Fatal(err)
	}
	selectors, formatters := 0, 0
	worker := &Engine{pipeline: pipeline, config: engine.Config{MaxLength: 4096}, http: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		body := `{"prompt":"rendered"}`
		switch r.URL.Path {
		case "/tokenize":
			body = `{"tokens":[1]}`
		case "/v1/chat/completions":
			var request prompt.ChatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			output := wantFormatted
			if request.ResponseFormat != nil {
				selectors++
				output = `{"e0":"keep","e1":"replace"}`
			} else {
				formatters++
				var payload struct {
					Source string `json:"source"`
				}
				if err := json.Unmarshal([]byte(request.Messages[len(request.Messages)-1].Content), &payload); err != nil {
					t.Fatal(err)
				}
				if payload.Source != wantFormatted {
					t.Fatalf("formatter source=%q want=%q", payload.Source, wantFormatted)
				}
			}
			raw, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": output}, "finish_reason": "stop"}}})
			body = string(raw)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	var stop atomic.Bool
	result, err := worker.Run(engine.Request{Text: source, Scene: string(domain.ScenePrompt), Context: domain.RewriteContext{Screen: screen}}, &stop)
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != wantFormatted || selectors != wantSelectors || formatters != 1 {
		t.Fatalf("result=%q selectors=%d formatters=%d", result.Text, selectors, formatters)
	}
}
