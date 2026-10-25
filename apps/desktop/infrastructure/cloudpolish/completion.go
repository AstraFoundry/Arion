package cloudpolish

import (
	"context"
	"errors"
	"unicode/utf8"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"github.com/AstraFoundry/republic"
)

// Complete reuses the connected provider transport for explicitly authorized
// chat tasks. It does not authorize ranking or send anything on its own.
// The production UI does not wire this method to keyboard context.
func (s *Service) Complete(ctx context.Context, req prompt.ChatRequest) (engine.Result, error) {
	chat, err := s.completionModel(ctx)
	if err != nil {
		return engine.Result{}, err
	}
	return s.completeChat(ctx, chat, req)
}

// Bind the provider and model once for all stages of a rewrite.
func (s *Service) completionModel(ctx context.Context) (*republic.ChatModel, error) {
	s.mu.Lock()
	model := s.model
	format := s.completionFormat
	s.mu.Unlock()
	if model == "" {
		return nil, domain.Fail(domain.CodeModelMissing, "no completion model chosen")
	}
	p, err := s.ready(ctx)
	if err != nil {
		return nil, s.settle(err, CodeNetwork)
	}
	chat, err := p.GetModel(model)
	if err != nil {
		return nil, fail(CodeModels, err)
	}
	if format != "" {
		chat.Format = republic.WireFormat{Format: format, Dialect: p.Name}
	}
	return chat, nil
}

func (s *Service) completeChat(ctx context.Context, chat *republic.ChatModel, req prompt.ChatRequest) (engine.Result, error) {
	if err := ctx.Err(); err != nil {
		return engine.Result{}, err
	}
	messages := make([]republic.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			messages = append(messages, republic.System(m.Content))
		case "user":
			messages = append(messages, republic.User(republic.Text(m.Content)))
		case "assistant":
			messages = append(messages, republic.Assistant(republic.Text(m.Content)))
		default:
			return engine.Result{}, errors.New("unsupported completion role")
		}
	}
	options := republic.Options{}
	// The Codex backend rejects token caps; its request is bounded by the
	// coordinator's deadline instead. Other wire formats translate this cap.
	if chat.Provider.Name != "codex" {
		options["max_tokens"] = req.MaxTokens
	}
	// Republic translates none into each provider's off switch.
	if !req.AllowThinking {
		options["reasoning_effort"] = "none"
	}
	// Selection is validated by the shared bounded-choice protocol. Cloud
	// models receive its JSON instruction in the prompt; native grammar fields
	// are specific to llama.cpp and are never copied into provider requests.
	response, err := chat.Chat(ctx, messages, options)
	if err = s.settle(err, CodeNetwork); err != nil {
		return engine.Result{}, err
	}
	if !utf8.ValidString(response.Text()) {
		return engine.Result{}, errors.New("completion is not UTF-8")
	}
	return engine.Result{Text: response.Text(), Truncated: response.FinishReason != "stop" || response.Refusal != ""}, nil
}

// NewAPICompletion is an explicit evaluation entry point for API-key or local
// compatible endpoints. It never reads stored credentials or signs in.
func NewAPICompletion(provider, key, model, base, format string) (*Service, error) {
	if provider == ChatGPT || !Known(provider) || key == "" || model == "" {
		return nil, errors.New("explicit API provider, key and model required")
	}
	if format != "" && format != "chat" && format != "responses" {
		return nil, errors.New("unsupported completion wire format")
	}
	s := New(nil, nil, nil)
	s.credential = Credential{Provider: provider, APIKey: key}
	s.model = model
	s.base = base
	s.completionFormat = format
	return s, nil
}

// CloseCompletion releases an evaluation provider without changing credentials.
func (s *Service) CloseCompletion() { s.mu.Lock(); defer s.mu.Unlock(); s.closeLocked() }
