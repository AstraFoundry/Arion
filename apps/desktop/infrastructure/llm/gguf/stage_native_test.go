package gguf

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/spelling"
)

// stageCase describes constructed transcript/context input. Evaluator checks
// never enter the native engine request or the formatter diagnostic request.
type stageCase struct {
	ID       string   `json:"id"`
	Scene    string   `json:"scene"`
	Preview  string   `json:"preview"`
	Final    string   `json:"final"`
	Screen   string   `json:"screen"`
	Terms    []string `json:"terms,omitempty"`
	Previous string   `json:"previousText,omitempty"`
	PairKind string   `json:"pairKind"`
	Checks   []string `json:"checks"`
}

type stageWire struct {
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response,omitempty"`
	Error     string          `json:"error,omitempty"`
	Status    int             `json:"status"`
	ElapsedMs float64         `json:"elapsedMs"`
}

type stageTransport struct {
	base  http.RoundTripper
	calls []stageWire
}

// RoundTrip records actual native completions without substituting inference.
func (r *stageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path != "/v1/chat/completions" {
		return r.base.RoundTrip(req)
	}
	started := time.Now()
	wire := stageWire{}
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	wire.Request = append(json.RawMessage{}, raw...)
	req.Body = io.NopCloser(bytes.NewReader(raw))
	response, err := r.base.RoundTrip(req)
	if err != nil {
		wire.Error = err.Error()
	} else {
		wire.Status = response.StatusCode
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		response.Body = io.NopCloser(bytes.NewReader(body))
		if json.Valid(body) {
			wire.Response = append(json.RawMessage{}, body...)
		} else {
			wire.Error = string(body)
		}
		if readErr != nil {
			wire.Error = readErr.Error()
		}
	}
	wire.ElapsedMs = float64(time.Since(started).Microseconds()) / 1000
	r.calls = append(r.calls, wire)
	return response, err
}

// TestNativeStagePreservation is opt-in development discovery, not acoustic or
// OCR execution evidence. Every stage uses the real pinned native runtime.
func TestNativeStagePreservation(t *testing.T) {
	runtimeDir, modelDir := os.Getenv("ARION_GGUF_RUNTIME"), os.Getenv("ARION_GGUF_MODEL")
	casesPath, outPath := os.Getenv("ARION_STAGE_CASES"), os.Getenv("ARION_STAGE_OUT")
	if runtimeDir == "" || modelDir == "" || casesPath == "" || outPath == "" {
		t.Skip("native artifacts, stage corpus and exclusive output path required")
	}
	data, err := os.ReadFile(casesPath)
	if err != nil {
		t.Fatal(err)
	}
	var cases []stageCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	write := func(value any) {
		t.Helper()
		if err := encoder.Encode(value); err != nil {
			t.Fatal(err)
		}
		if err := out.Sync(); err != nil {
			t.Fatal(err)
		}
	}
	config := engine.Config{ModelDir: modelDir, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}
	write(map[string]any{"kind": "start", "config": config, "runtime": runtimeDir, "cases": casesPath, "formatOnly": os.Getenv("ARION_STAGE_FORMAT_ONLY") == "1"})
	e, loadMs, err := Load(config, runtimeDir)
	if err != nil {
		write(map[string]any{"kind": "load-error", "error": err.Error()})
		t.Fatal(err)
	}
	defer e.Close()
	recorder := &stageTransport{base: e.http.Transport}
	e.http.Transport = recorder
	write(map[string]any{"kind": "loaded", "loadMs": loadMs})
	for _, c := range cases {
		if filter := os.Getenv("ARION_STAGE_FILTER"); filter != "" && !strings.Contains(","+filter+",", ","+c.ID+",") {
			continue
		}
		for _, phase := range []string{"preview", "final"} {
			source := c.Preview
			if phase == "final" {
				source = c.Final
			}
			recorder.calls = nil
			req := engine.Request{Session: c.ID, Scene: c.Scene, Text: source, Terms: c.Terms, Context: domain.RewriteContext{Screen: c.Screen}}
			if phase == "final" && source != c.Preview {
				req.PreviousText = c.Preview
			}
			if c.Previous != "" && phase == "final" {
				req.PreviousText = c.Previous
			}
			if os.Getenv("ARION_STAGE_NO_SCREEN") == "1" {
				req.Context.Screen = ""
			}
			if os.Getenv("ARION_STAGE_NO_PREVIOUS") == "1" {
				req.PreviousText = ""
			}
			started := time.Now()
			var result engine.Result
			var runErr error
			mode := "native-pipeline"
			if os.Getenv("ARION_STAGE_FORMAT_ONLY") == "1" {
				mode = "formatter-only-diagnostic"
				lexicon, loadErr := spelling.Load()
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				recipe, loadErr := prompt.Selected(lexicon)
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				request, formatErr := recipe.Format(source, c.Scene, domain.PunctuationSmart)
				runErr = formatErr
				if runErr == nil {
					completion, generateErr := e.generate(context.Background(), request, false)
					runErr = generateErr
					if len(completion.Choices) == 1 {
						result.Text = strings.TrimSpace(completion.Choices[0].Message.Content)
					}
					result.InputTokens, result.OutputTokens = completion.Usage.Input, completion.Usage.Output
				}
			} else {
				var stop atomic.Bool
				timer := time.AfterFunc(10*time.Second, func() { stop.Store(true) })
				result, runErr = e.Run(req, &stop)
				timer.Stop()
			}
			errorText := ""
			if runErr != nil {
				errorText = runErr.Error()
			}
			checks := map[string]bool{}
			for _, term := range c.Checks {
				checks[term] = strings.Contains(result.Text, term)
			}
			write(map[string]any{"kind": "observation", "id": c.ID, "pairKind": c.PairKind, "phase": phase, "mode": mode, "request": req, "result": result, "wire": recorder.calls, "elapsedMs": float64(time.Since(started).Microseconds()) / 1000, "error": errorText, "deliveryVerdict": domain.ValidateRewriteDelivery(domain.RewriteCheck{Source: source, Candidate: result.Text, Truncated: result.Truncated}), "evaluatorContains": checks})
		}
		if err := e.EndSession(); err != nil {
			write(map[string]any{"kind": "session-error", "id": c.ID, "error": err.Error()})
			t.Fatal(err)
		}
	}
	write(map[string]any{"kind": "complete"})
}
