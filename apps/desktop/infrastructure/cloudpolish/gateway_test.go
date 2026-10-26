package cloudpolish

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/spelling"
)

// gatewayFixture mocks only the remote HTTP boundary, leaving the provider SDK,
// cloud adapter, prompt construction and shared spelling validator in production.
type gatewayFixture struct {
	mu       sync.Mutex
	requests []map[string]any
	payloads []map[string]any
	format   string
	mode     string
	stage    int
	entered  chan struct{}
	release  chan struct{}
}

func newGateway(t *testing.T, format, mode string, stage int) (*gatewayFixture, *Service) {
	t.Helper()
	g := &gatewayFixture{format: format, mode: mode, stage: stage, entered: make(chan struct{}, 8), release: make(chan struct{})}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := "/v1/responses"
		if format == "chat" {
			path = "/v1/chat/completions"
		}
		if r.Method != "POST" || r.URL.Path != path || r.Header.Get("Authorization") != "Bearer synthetic-gateway-key" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", 400)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			http.Error(w, "invalid JSON", 400)
			return
		}
		payload, err := gatewayPayload(body, format)
		if err != nil {
			t.Error(err)
			http.Error(w, "invalid messages", 400)
			return
		}
		if body["model"] != "gateway-model" {
			t.Errorf("model: %v", body["model"])
		}
		if body["max_output_tokens"] == nil && body["max_tokens"] == nil && body["max_completion_tokens"] == nil {
			t.Error("missing token cap")
		}
		for _, key := range []string{"grammar", "cache_prompt", "session_id"} {
			if _, ok := body[key]; ok {
				t.Errorf("native field transmitted: %s", key)
			}
		}
		g.mu.Lock()
		g.requests = append(g.requests, body)
		g.payloads = append(g.payloads, payload)
		call := len(g.requests)
		g.mu.Unlock()
		if call == g.stage {
			switch g.mode {
			case "block":
				g.entered <- struct{}{}
				select {
				case <-r.Context().Done():
					return
				case <-g.release:
				}
			case "400", "401", "429", "500":
				var status int
				_, _ = fmt.Sscan(g.mode, &status)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"synthetic gateway failure","type":"invalid_request_error","code":"synthetic"}}`))
				return
			case "malformed":
				_, _ = w.Write([]byte(`{"broken":`))
				return
			}
		}
		text, _ := payload["source"].(string)
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
			encoded, _ := json.Marshal(choices)
			text = string(encoded)
		}
		mode := ""
		if call == g.stage {
			mode = g.mode
		}
		if mode == "invalid-choice" {
			text = `{"e0":"invent"}`
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gatewayResponse(format, text, mode))
	}))
	t.Cleanup(func() { close(g.release); srv.Close() })
	service, err := NewAPICompletion(OpenAI, "synthetic-gateway-key", "gateway-model", srv.URL+"/v1", format)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.CloseCompletion)
	return g, service
}

func gatewayPayload(body map[string]any, format string) (map[string]any, error) {
	key := "input"
	if format == "chat" {
		key = "messages"
	}
	messages, ok := body[key].([]any)
	if !ok || len(messages) == 0 {
		return nil, fmt.Errorf("missing %s", key)
	}
	last, ok := messages[len(messages)-1].(map[string]any)
	if !ok || last["role"] != "user" {
		return nil, fmt.Errorf("missing final user message")
	}
	content, ok := last["content"].(string)
	if !ok {
		parts, _ := last["content"].([]any)
		if len(parts) > 0 {
			part, _ := parts[0].(map[string]any)
			content, _ = part["text"].(string)
		}
	}
	if strings.HasPrefix(content, "<source>") && strings.HasSuffix(content, "</source>") {
		return map[string]any{"source": strings.TrimSuffix(strings.TrimPrefix(content, "<source>"), "</source>")}, nil
	}
	var payload map[string]any
	err := json.Unmarshal([]byte(content), &payload)
	return payload, err
}

func gatewayResponse(format, text, mode string) map[string]any {
	if format == "chat" {
		reason := "stop"
		if mode == "truncated" {
			reason = "length"
		}
		message := map[string]any{"role": "assistant", "content": text}
		if mode == "refusal" {
			message["refusal"] = "synthetic refusal"
			message["content"] = nil
		}
		return map[string]any{"id": "synthetic", "object": "chat.completion", "model": "gateway-model", "created": 1, "choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": reason}}}
	}
	status := "completed"
	if mode == "truncated" {
		status = "incomplete"
	}
	content := map[string]any{"type": "output_text", "text": text, "annotations": []any{}}
	if mode == "refusal" {
		content = map[string]any{"type": "refusal", "refusal": "synthetic refusal"}
	}
	response := map[string]any{"id": "synthetic", "object": "response", "model": "gateway-model", "status": status, "output": []any{map[string]any{"id": "msg", "type": "message", "role": "assistant", "status": status, "content": []any{content}}}}
	if mode == "truncated" {
		response["incomplete_details"] = map[string]any{"reason": "max_output_tokens"}
	}
	return response
}

func (g *gatewayFixture) snapshot() ([]map[string]any, []map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]map[string]any(nil), g.requests...), append([]map[string]any(nil), g.payloads...)
}

func gatewayRequest() ports.RewriteRequest {
	return ports.RewriteRequest{Text: "um 检查 write value。", Scene: "prompt", Terms: []string{"writeValue"}, PreviousText: "之前 writeValue。", Context: domain.RewriteContext{Before: "BEFORE_CANARY", After: "AFTER_CANARY", Screen: "writeValue SCREEN_CANARY", History: []string{"HISTORY_CANARY"}}}
}

func TestOpenAIGatewaySharedStages(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		t.Run(format, func(t *testing.T) {
			g, s := newGateway(t, format, "", 0)
			result, err := s.Rewrite(context.Background(), gatewayRequest())
			if err != nil || result.Text != "检查 writeValue。" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			bodies, payloads := g.snapshot()
			if len(payloads) != 2 {
				t.Fatalf("calls=%d", len(payloads))
			}
			c, ok := payloads[0]["context"].(map[string]any)
			if !ok {
				t.Fatal("selector missing context")
			}
			for key, want := range map[string]string{"Before": "BEFORE_CANARY", "After": "AFTER_CANARY", "Screen": "writeValue SCREEN_CANARY", "History": "HISTORY_CANARY", "Terms": "writeValue", "EarlierSource": "之前 writeValue。"} {
				if c[key] != want {
					t.Errorf("%s=%v", key, c[key])
				}
			}
			raw, _ := json.Marshal(bodies[1])
			if strings.Contains(string(raw), "CANARY") || strings.Contains(string(raw), "之前") || payloads[1]["source"] != "检查 writeValue。" {
				t.Fatalf("formatter leaked context or lost selected spelling: %s", raw)
			}
			t.Logf("HTTP calls=%d; selector context verified; source-only formatter; output=%q", len(bodies), result.Text)
		})
	}
}

func TestOpenAIGatewayFailures(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		for _, mode := range []string{"400", "401", "malformed", "invalid-choice", "truncated", "refusal"} {
			for _, stage := range []int{1, 2} {
				if mode == "invalid-choice" && stage == 2 {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/stage%d", format, mode, stage), func(t *testing.T) {
					g, s := newGateway(t, format, mode, stage)
					result, err := s.Rewrite(context.Background(), gatewayRequest())
					if err == nil || result.Text != "" {
						t.Fatalf("failure accepted: %+v %v", result, err)
					}
					bodies, _ := g.snapshot()
					if len(bodies) != stage {
						t.Fatalf("continued after failure: calls=%d", len(bodies))
					}
					t.Logf("rejected after %d HTTP call(s): %v", len(bodies), err)
				})
			}
		}
	}
}

func TestOpenAIGatewayCancellationAndRecovery(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		for _, stage := range []int{1, 2} {
			for _, deadline := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stage%d/deadline%v", format, stage, deadline), func(t *testing.T) {
					g, s := newGateway(t, format, "block", stage)
					ctx, cancel := context.WithCancel(context.Background())
					if deadline {
						cancel()
						ctx, cancel = context.WithTimeout(context.Background(), 150*time.Millisecond)
					}
					defer cancel()
					type completion struct {
						text string
						err  error
					}
					done := make(chan completion, 1)
					go func() {
						result, err := s.Rewrite(ctx, gatewayRequest())
						done <- completion{text: result.Text, err: err}
					}()
					select {
					case <-g.entered:
					case <-time.After(3 * time.Second):
						t.Fatal("request did not arrive")
					}
					if !deadline {
						cancel()
					}
					select {
					case result := <-done:
						if result.err == nil || result.text != "" {
							t.Fatalf("cancellation returned text=%q err=%v", result.text, result.err)
						}
					case <-time.After(3 * time.Second):
						t.Fatal("cancellation did not stop HTTP")
					}
					bodies, _ := g.snapshot()
					if len(bodies) != stage {
						t.Fatalf("calls after cancellation=%d", len(bodies))
					}
					result, err := s.Rewrite(context.Background(), gatewayRequest())
					if err != nil || result.Text != "检查 writeValue。" {
						t.Fatalf("recovery failed: %+v %v", result, err)
					}
				})
			}
		}
	}
}

func TestOpenAIGatewayRetryableFailure(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		for _, mode := range []string{"429", "500"} {
			t.Run(format+"/"+mode, func(t *testing.T) {
				g, s := newGateway(t, format, mode, 1)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				result, err := s.Rewrite(ctx, gatewayRequest())
				if err != nil || result.Text != "检查 writeValue。" {
					t.Fatalf("retry did not recover: %+v %v", result, err)
				}
				_, payloads := g.snapshot()
				if len(payloads) != 3 {
					t.Fatalf("HTTP calls=%d, want failed selection, retried selection, formatting", len(payloads))
				}
				first, _ := json.Marshal(payloads[0])
				second, _ := json.Marshal(payloads[1])
				if string(first) != string(second) {
					t.Fatal("retry changed selector payload")
				}
			})
		}
	}
}

func TestOpenAIGatewayWithoutSpellingCandidates(t *testing.T) {
	for _, format := range []string{"responses", "chat"} {
		t.Run(format, func(t *testing.T) {
			g, s := newGateway(t, format, "", 0)
			result, err := s.Rewrite(context.Background(), ports.RewriteRequest{Text: "um 今天天气很好。", Scene: "prompt"})
			if err != nil || result.Text != "今天天气很好。" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			_, payloads := g.snapshot()
			if len(payloads) != 1 || payloads[0]["source"] != "今天天气很好。" {
				t.Fatalf("unexpected stage requests: %+v", payloads)
			}
		})
	}
}

// The main-branch punctuation choice must reach the shared formatter unchanged.
func TestOpenAIGatewayPunctuation(t *testing.T) {
	lexicon, err := spelling.Load()
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := prompt.Selected(lexicon)
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"responses", "chat"} {
		for _, punctuation := range []domain.Punctuation{domain.PunctuationSmart, domain.PunctuationFull, domain.PunctuationOmit, domain.PunctuationSpaces} {
			t.Run(format+"/"+string(punctuation), func(t *testing.T) {
				g, s := newGateway(t, format, "", 0)
				req := ports.RewriteRequest{Text: "今天天气很好。", Scene: "prompt", Punctuation: string(punctuation)}
				if _, err := s.Rewrite(context.Background(), req); err != nil {
					t.Fatal(err)
				}
				bodies, _ := g.snapshot()
				if len(bodies) != 1 {
					t.Fatalf("calls=%d", len(bodies))
				}
				expected, err := recipe.Format(req.Text, req.Scene, punctuation)
				if err != nil {
					t.Fatal(err)
				}
				// Compare the serialized system instruction, not just the output: the
				// synthetic formatter echoes text and cannot prove punctuation forwarding.
				key := "input"
				if format == "chat" {
					key = "messages"
				}
				messages := bodies[0][key].([]any)
				first := messages[0].(map[string]any)
				content, ok := first["content"].(string)
				if !ok {
					content = first["content"].([]any)[0].(map[string]any)["text"].(string)
				}
				if content != expected.Messages[0].Content {
					t.Fatal("punctuation instruction lost in cloud adapter")
				}
			})
		}
	}
}
