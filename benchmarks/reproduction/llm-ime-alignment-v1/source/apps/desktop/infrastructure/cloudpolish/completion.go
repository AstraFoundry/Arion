package cloudpolish

import (
	"context"
	"errors"

	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"github.com/AstraFoundry/republic"
)

// Complete reuses the connected provider transport for explicitly authorized
// chat tasks. It does not authorize ranking or send anything on its own.
// The production UI does not wire this method to keyboard context.
func (s *Service) Complete(ctx context.Context, req prompt.ChatRequest) (engine.Result, error) {
	s.mu.Lock()
	model := s.model
	format := s.completionFormat
	s.mu.Unlock()
	if model == "" {
		return engine.Result{}, errors.New("no completion model chosen")
	}
	p, err := s.ready(ctx)
	if err != nil {
		return engine.Result{}, s.settle(err, CodeNetwork)
	}
	chat, err := p.GetModel(model)
	if err != nil {
		return engine.Result{}, fail(CodeModels, err)
	}
	if format != "" {
		chat.Format = republic.WireFormat{Format: format, Dialect: p.Name}
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
	response, err := chat.Chat(ctx, messages, nil)
	if err = s.settle(err, CodeNetwork); err != nil {
		return engine.Result{}, err
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
