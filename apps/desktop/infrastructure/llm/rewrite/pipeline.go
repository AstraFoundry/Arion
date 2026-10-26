// Package rewrite owns the staged polishing pipeline shared by all model transports.
package rewrite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/spelling"
)

// Pipeline keeps immutable spelling resources and reviewed prompt recipes.
type Pipeline struct {
	lexicon *domain.SpellingLexicon
	recipe  *prompt.SelectedTemplates
}

var load = sync.OnceValues(func() (*Pipeline, error) {
	lexicon, err := spelling.Load()
	if err != nil {
		return nil, err
	}
	recipe, err := prompt.Selected(lexicon)
	if err != nil {
		return nil, err
	}
	return &Pipeline{lexicon: lexicon, recipe: recipe}, nil
})

// Load shares immutable resources without storing any request or session data.
func Load() (*Pipeline, error) { return load() }

// Generate executes one model stage. Prefill must never sample output tokens.
type Generate func(context.Context, prompt.ChatRequest, bool) (engine.Result, error)

// Run cleans speech, selects bounded spelling edits, and formats the edited
// source. Context reaches only selection; it never reaches the formatter.
func (p *Pipeline) Run(ctx context.Context, req engine.Request, generate Generate) (engine.Result, error) {
	if strings.Contains(req.Text, "<|") || strings.Contains(req.Text, "|>") {
		return engine.Result{}, engine.ErrControlInput
	}
	if ctx.Err() != nil {
		return engine.Result{Cancelled: true}, nil
	}
	cleanup := domain.CleanSpeech(req.Text)
	contextData := map[string]string{}
	c := req.Context.Bounded()
	if c.Screen != "" {
		contextData["Screen"] = c.Screen
	}
	if c.Before != "" {
		contextData["Before"] = c.Before
	}
	if c.After != "" {
		contextData["After"] = c.After
	}
	if len(c.History) > 0 {
		contextData["History"] = strings.Join(c.History, "\n")
	}
	if len(req.Terms) > 0 {
		contextData["Terms"] = strings.Join(req.Terms, "\n")
	}
	if req.PreviousText != "" && req.PreviousText != req.Text {
		contextData["EarlierSource"] = (domain.RewriteContext{Screen: req.PreviousText}).Bounded().Screen
	}
	prepared := p.lexicon.PrepareContextual(cleanup.Text, contextData, req.Scene)
	result := engine.Result{Cleanup: cleanup, Template: prompt.SelectedVersion}
	edited := cleanup.Text
	add := func(out engine.Result) {
		result.InputTokens += out.InputTokens
		result.OutputTokens += out.OutputTokens
		result.ReusedTokens += out.ReusedTokens
		result.PrefillMs += out.PrefillMs
		result.DecodeMs += out.DecodeMs
	}
	forced, allAmbiguous, err := forcedAmbiguousSelection(prepared)
	if err != nil {
		return result, err
	}
	if allAmbiguous {
		edited = forced
	} else if len(prepared.Hypotheses) > 0 {
		out, err := generate(ctx, p.recipe.Selection(prepared), req.Prefill)
		if err != nil {
			if ctx.Err() != nil {
				result.Cancelled = true
				return result, nil
			}
			return result, err
		}
		add(out)
		if out.Truncated || out.Cancelled {
			result.Truncated, result.Cancelled = out.Truncated, out.Cancelled
			return result, errors.New("incomplete spelling selection")
		}
		if !req.Prefill {
			edited, err = domain.ApplySpellingChoices(out.Text, prepared)
			if err != nil {
				return result, err
			}
		}
	}
	if ctx.Err() != nil {
		result.Cancelled = true
		return result, nil
	}
	var request prompt.ChatRequest
	if req.TargetLanguage != "" {
		request, err = p.recipe.FormatTranslation(edited, req.Scene, req.Punctuation, req.TargetLanguage)
	} else {
		request, err = p.recipe.Format(edited, req.Scene, req.Punctuation)
	}
	if err != nil {
		return result, err
	}
	out, err := generate(ctx, request, req.Prefill)
	if err != nil {
		if ctx.Err() != nil {
			result.Cancelled = true
			return result, nil
		}
		return result, err
	}
	add(out)
	if out.Truncated || out.Cancelled {
		result.Truncated, result.Cancelled = out.Truncated, out.Cancelled
		return result, errors.New("incomplete formatting")
	}
	if !req.Prefill {
		result.Text = strings.TrimSpace(out.Text)
	}
	result.Cancelled = ctx.Err() != nil
	return result, nil
}

// forcedAmbiguousSelection avoids inference only when the protocol has exactly
// one legal result: every hypothesis must keep its original source span. The
// normal domain validator still verifies coverage and UTF-8 source identity.
func forcedAmbiguousSelection(p domain.SpellingPreparation) (string, bool, error) {
	if len(p.Hypotheses) == 0 {
		return "", false, nil
	}
	for _, h := range p.Hypotheses {
		if h.Status != "ambiguous" {
			return "", false, nil
		}
		if len(h.Candidates) < 2 {
			return "", false, errors.New("invalid ambiguous spelling candidates")
		}
	}
	choices := make(map[string]string, len(p.Hypotheses))
	for _, h := range p.Hypotheses {
		choices[h.ID] = "keep"
	}
	raw, err := json.Marshal(choices)
	if err != nil {
		return "", false, err
	}
	edited, err := domain.ApplySpellingChoices(string(raw), p)
	return edited, true, err
}
