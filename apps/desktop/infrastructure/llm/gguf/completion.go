package gguf

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
)

// StopContext bridges the worker's out-of-band cancellation to native HTTP.
func StopContext(stop *atomic.Bool) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	if stop.Load() {
		cancel()
		return ctx, cancel
	}
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if stop.Load() {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel
}

// Complete runs an independent bounded chat task on the resident model.
// Erasing the slot prevents ranking and rewriting from sharing private state.
func (e *Engine) Complete(ctx context.Context, req prompt.ChatRequest) (result engine.Result, err error) {
	if len(req.Messages) == 0 || len(req.Messages) > 16 || req.MaxTokens <= 0 || req.MaxTokens > 256 {
		return result, errors.New("invalid completion bounds")
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = e.EndSession(); err != nil {
		return result, err
	}
	defer func() {
		if eraseErr := e.EndSession(); eraseErr != nil && err == nil {
			err = eraseErr
		}
	}()
	req.CachePrompt = false
	req.Stream = false
	req.TemplateArgs = map[string]bool{"enable_thinking": false}
	out, err := e.generate(ctx, req, false)
	if err != nil {
		return result, err
	}
	result = engine.Result{Text: out.Choices[0].Message.Content, InputTokens: out.Usage.Input, OutputTokens: out.Usage.Output, PrefillMs: out.Timings.PromptMs, DecodeMs: out.Timings.DecodeMs}
	return result, nil
}
