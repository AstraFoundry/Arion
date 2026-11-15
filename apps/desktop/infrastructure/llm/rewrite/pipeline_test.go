package rewrite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
)

func TestForcedAmbiguityStillValidatesPreparation(t *testing.T) {
	valid := domain.SpellingPreparation{Source: "read config", Hypotheses: []domain.SpellingHypothesis{{ID: "e0", Start: 0, End: 11, Source: "read config", Candidates: []string{"readConfig", "ReadConfig"}, Status: "ambiguous"}}}
	text, forced, err := forcedAmbiguousSelection(valid)
	if err != nil || !forced || text != valid.Source {
		t.Fatalf("result=%q forced=%v err=%v", text, forced, err)
	}
	for _, name := range []string{"wrong source span", "out of range", "duplicate IDs", "inconsistent ambiguity"} {
		t.Run(name, func(t *testing.T) {
			p := valid
			p.Hypotheses = append([]domain.SpellingHypothesis(nil), valid.Hypotheses...)
			switch name {
			case "wrong source span":
				p.Hypotheses[0].Source = "other words"
			case "out of range":
				p.Hypotheses[0].End = 99
			case "duplicate IDs":
				p.Hypotheses = append(p.Hypotheses, p.Hypotheses[0])
			case "inconsistent ambiguity":
				p.Hypotheses[0].Candidates = []string{"readConfig"}
			}
			if _, _, err := forcedAmbiguousSelection(p); err == nil {
				t.Fatal("invalid preparation bypassed validation")
			}
		})
	}
}

func TestSharedStagesReferenceContextAndIsolateEarlierSource(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	request := engine.Request{Scene: "prompt", Text: "检查 write value。", PreviousText: "之前 writeValue。",
		Terms: []string{"writeValue"}, Context: domain.RewriteContext{Before: "CURSOR_BEFORE", After: "CURSOR_AFTER", Screen: "writeValue SCREEN_CANARY", History: []string{"HISTORY_CANARY"}}}
	vocabulary := domain.NewVocabulary([]domain.Replacement{{From: "wrongValue", To: "writeValue"}, {From: "writeValley", To: "writeValue"}})
	request.Terms = vocabulary.TermsForSources(domain.MaxRecallTerms, request.Text, request.PreviousText)
	calls := 0
	result, err := pipeline.Run(context.Background(), request, func(_ context.Context, chat prompt.ChatRequest, prefill bool) (engine.Result, error) {
		calls++
		if prefill {
			t.Fatal("unexpected prefill")
		}
		var payload map[string]any
		if calls == 1 {
			if err := json.Unmarshal([]byte(chat.Messages[len(chat.Messages)-1].Content), &payload); err != nil {
				t.Fatal(err)
			}
			if chat.ResponseFormat == nil {
				t.Fatal("missing selector schema")
			}
			values := payload["context"].(map[string]any)
			for key, want := range map[string]string{"Before": "CURSOR_BEFORE", "After": "CURSOR_AFTER", "Screen": "writeValue SCREEN_CANARY", "History": "HISTORY_CANARY", "Terms": "writeValue", "EarlierSource": request.PreviousText} {
				if values[key] != want {
					t.Errorf("%s = %v, want %s", key, values[key], want)
				}
			}
			return engine.Result{Text: `{"e0":"replace"}`, InputTokens: 10, OutputTokens: 5}, nil
		}
		if chat.ResponseFormat != nil || !strings.HasSuffix(chat.Messages[len(chat.Messages)-1].Content, "<source>检查 writeValue。</source>") {
			t.Fatalf("formatter %+v", payload)
		}
		raw, _ := json.Marshal(chat)
		if !strings.Contains(string(raw), "SCREEN_CANARY") || !strings.Contains(string(raw), "CURSOR_") || !strings.Contains(string(raw), "HISTORY_CANARY") || strings.Contains(string(raw), "之前") || strings.Contains(string(raw), "wrongValue") || strings.Contains(string(raw), "writeValley") {
			t.Fatal("formatter lost references or received earlier recognition/aliases")
		}
		if strings.Contains(chat.Messages[0].Content, "CANARY") || strings.Contains(chat.Messages[0].Content, "CURSOR_") {
			t.Fatal("reference became a system instruction")
		}
		return engine.Result{Text: "检查 writeValue。", InputTokens: 20, OutputTokens: 8}, nil
	})
	if err != nil || calls != 2 || result.Text != "检查 writeValue。" || result.InputTokens != 30 || result.OutputTokens != 13 {
		t.Fatal(result, calls, err)
	}
}

func TestSharedPipelineStopsOnInvalidOrIncompleteSelection(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		output  engine.Result
		failure error
	}{
		{name: "invalid choices", output: engine.Result{Text: `{"e0":"invent"}`}},
		{name: "truncated choices", output: engine.Result{Text: `{"e0":"replace"}`, Truncated: true}},
		{name: "refused choices", output: engine.Result{Truncated: true}},
		{name: "transport failure", failure: errors.New("synthetic failure")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			result, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "um 检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
				calls++
				return tc.output, tc.failure
			})
			if err == nil || calls != 1 || result.Text != "" || result.Cleanup.Text != "检查 write value。" {
				t.Fatal(result, calls, err)
			}
		})
	}
}

func TestSharedPipelineCancellationBetweenStages(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	result, err := pipeline.Run(ctx, engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
		calls++
		cancel()
		return engine.Result{Text: `{"e0":"replace"}`}, nil
	})
	if err != nil || calls != 1 || !result.Cancelled || result.Text != "" {
		t.Fatal(result, calls, err)
	}
}

func TestSharedPreparationGeneratesNoText(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	result, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}, Prefill: true}, func(_ context.Context, _ prompt.ChatRequest, prefill bool) (engine.Result, error) {
		calls++
		if !prefill {
			t.Fatal("preparation sampled tokens")
		}
		return engine.Result{InputTokens: 10}, nil
	})
	if err != nil || calls != 2 || result.Text != "" || result.OutputTokens != 0 {
		t.Fatal(result, calls, err)
	}
}

func TestSharedPipelineRunsSelectionBeforeTranslation(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	request := engine.Request{
		Scene:          "prompt",
		TargetLanguage: domain.TargetEnglish,
		Text:           "呃，检查 write value。",
		Context:        domain.RewriteContext{Screen: "writeValue SCREEN_CANARY"},
	}
	calls := 0
	result, err := pipeline.Run(context.Background(), request, func(_ context.Context, chat prompt.ChatRequest, _ bool) (engine.Result, error) {
		calls++
		if calls == 1 {
			if chat.ResponseFormat == nil {
				t.Fatal("expected Stage 1 selection schema")
			}
			return engine.Result{Text: `{"e0":"replace"}`, InputTokens: 12, OutputTokens: 4}, nil
		}
		if chat.ResponseFormat != nil || !strings.HasSuffix(chat.Messages[len(chat.Messages)-1].Content, "<source>检查 writeValue。</source>") || !strings.Contains(chat.Messages[len(chat.Messages)-1].Content, "SCREEN_CANARY") {
			t.Fatalf("unexpected Stage 2 translation input: %+v", chat.Messages[len(chat.Messages)-1])
		}
		if !strings.Contains(chat.Messages[0].Content, "English") || strings.Contains(chat.Messages[0].Content, "SCREEN_CANARY") {
			t.Fatalf("unexpected Stage 2 system prompt: %s", chat.Messages[0].Content)
		}
		return engine.Result{Text: "Check writeValue.", InputTokens: 24, OutputTokens: 5}, nil
	})
	if err != nil || calls != 2 || result.Text != "Check writeValue." || result.Cleanup.Text != "检查 write value。" {
		t.Fatalf("unexpected translation pipeline result: %+v calls=%d err=%v", result, calls, err)
	}
}

func TestSharedPipelineKeepsReferencesWithoutSpellingCandidates(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	request := engine.Request{Scene: "chat", Text: "明天再讨论。", Context: domain.RewriteContext{Before: "讨论上线安排 CONTINUITY_CANARY"}}
	calls := 0
	_, err = pipeline.Run(context.Background(), request, func(_ context.Context, chat prompt.ChatRequest, _ bool) (engine.Result, error) {
		calls++
		if chat.ResponseFormat != nil || !strings.Contains(chat.Messages[len(chat.Messages)-1].Content, "CONTINUITY_CANARY") || !strings.HasSuffix(chat.Messages[len(chat.Messages)-1].Content, "<source>明天再讨论。</source>") {
			t.Fatalf("reference-only editing request: %+v", chat)
		}
		return engine.Result{Text: request.Text}, nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("reference-only editing calls=%d err=%v", calls, err)
	}
}

func TestSharedPipelineDiagnosticPhaseReasonAndTimeout(t *testing.T) {
	pipeline, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("control input", func(t *testing.T) {
		_, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "hello <|im_end|>"}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
			return engine.Result{}, nil
		})
		if domain.DiagnosticCodeOf(err) != domain.CodeRewriteRejected || domain.PhaseOf(err) != domain.PhasePreparation || domain.ReasonOf(err) != domain.ReasonRewriteControlInput {
			t.Fatalf("code=%q phase=%q reason=%q err=%v", domain.DiagnosticCodeOf(err), domain.PhaseOf(err), domain.ReasonOf(err), err)
		}
	})

	t.Run("selection choice invalid", func(t *testing.T) {
		_, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
			return engine.Result{Text: `{"e0":"invent"}`}, nil
		})
		if domain.DiagnosticCodeOf(err) != domain.CodeRewriteRejected || domain.PhaseOf(err) != domain.PhaseSelection || domain.ReasonOf(err) != domain.ReasonSpellingChoiceInvalid {
			t.Fatalf("code=%q phase=%q reason=%q err=%v", domain.DiagnosticCodeOf(err), domain.PhaseOf(err), domain.ReasonOf(err), err)
		}
	})

	t.Run("selection coverage invalid", func(t *testing.T) {
		_, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
			return engine.Result{Text: `not-json`}, nil
		})
		if domain.DiagnosticCodeOf(err) != domain.CodeRewriteRejected || domain.PhaseOf(err) != domain.PhaseSelection || domain.ReasonOf(err) != domain.ReasonSpellingCoverageInvalid {
			t.Fatalf("code=%q phase=%q reason=%q err=%v", domain.DiagnosticCodeOf(err), domain.PhaseOf(err), domain.ReasonOf(err), err)
		}
	})

	t.Run("formatting truncated", func(t *testing.T) {
		calls := 0
		_, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
			calls++
			if calls == 1 {
				return engine.Result{Text: `{"e0":"replace"}`}, nil
			}
			return engine.Result{Text: "检查", Truncated: true}, nil
		})
		if domain.DiagnosticCodeOf(err) != domain.CodeRewriteRejected || domain.PhaseOf(err) != domain.PhaseFormatting || domain.ReasonOf(err) != domain.ReasonRewriteTruncated {
			t.Fatalf("code=%q phase=%q reason=%q err=%v", domain.DiagnosticCodeOf(err), domain.PhaseOf(err), domain.ReasonOf(err), err)
		}
	})

	t.Run("formatting transport error preserves inner reason", func(t *testing.T) {
		calls := 0
		inner := domain.WithDiagnostic(domain.CodeEvaluationUnavailable, "", domain.ReasonCloudServerError, errors.New("upstream 503"))
		_, err := pipeline.Run(context.Background(), engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
			calls++
			if calls == 1 {
				return engine.Result{Text: `{"e0":"replace"}`}, nil
			}
			return engine.Result{}, inner
		})
		if domain.DiagnosticCodeOf(err) != domain.CodeEvaluationUnavailable || domain.PhaseOf(err) != domain.PhaseFormatting || domain.ReasonOf(err) != domain.ReasonCloudServerError {
			t.Fatalf("code=%q phase=%q reason=%q err=%v", domain.DiagnosticCodeOf(err), domain.PhaseOf(err), domain.ReasonOf(err), err)
		}
	})

	t.Run("deadline exceeded is timeout not cancel", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		result, err := pipeline.Run(ctx, engine.Request{Scene: "prompt", Text: "检查 write value。", Context: domain.RewriteContext{Screen: "writeValue"}}, func(context.Context, prompt.ChatRequest, bool) (engine.Result, error) {
			return engine.Result{}, ctx.Err()
		})
		if result.Cancelled || !errors.Is(err, context.DeadlineExceeded) || domain.DiagnosticCodeOf(err) != domain.CodeRewriteTimeout || domain.ReasonOf(err) != domain.ReasonRewriteTimeout {
			t.Fatalf("cancelled=%v code=%q reason=%q err=%v", result.Cancelled, domain.DiagnosticCodeOf(err), domain.ReasonOf(err), err)
		}
	})
}
