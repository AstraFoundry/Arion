// Package cloudpolish is the optional cloud polishing capability (ADR 022):
// one credential per provider in the OS credential store, the active
// provider's model list, and a ports.Rewriter using the shared staged pipeline. Enabled context
// sources reach spelling selection through Republic; the formatter receives
// only the edited source. Cloud polishing requires an explicit engine choice.
package cloudpolish

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/AstraFoundry/republic"
)

// Provider IDs. ChatGPT signs in with the user's ChatGPT subscription; the
// others take an API key.
const (
	ChatGPT    = "chatgpt"
	OpenAI     = "openai"
	Anthropic  = "anthropic"
	Google     = "google"
	DeepSeek   = "deepseek"
	OpenRouter = "openrouter"
	// OpenAICompatible and AnthropicCompatible take the caller's base URL.
	// Requests use Republic's openai and anthropic providers with that APIBase.
	OpenAICompatible    = "openai-compatible"
	AnthropicCompatible = "anthropic-compatible"
)

// KeyProviders accept an API key, in the order the picker lists them.
var KeyProviders = []string{OpenAI, Anthropic, Google, DeepSeek, OpenRouter, OpenAICompatible, AnthropicCompatible}

// NeedsBase reports a provider whose endpoint comes from the credential.
func NeedsBase(id string) bool {
	return id == OpenAICompatible || id == AnthropicCompatible
}

// knownProviders is every provider that can hold a credential.
func knownProviders() []string {
	return append([]string{ChatGPT}, KeyProviders...)
}

// Known reports a provider this package can connect.
func Known(id string) bool {
	if id == ChatGPT {
		return true
	}
	for _, p := range KeyProviders {
		if p == id {
			return true
		}
	}
	return false
}

// codexClientVersion is sent when listing ChatGPT models; the service lists
// the models available to clients of at least this version.
const codexClientVersion = "1.0.0"

// codexSpec is Republic's codex provider with model listing, which Republic
// leaves unsupported: the ChatGPT backend lists models under "models" with a
// slug, and hidden ones are not offered.
func codexSpec() republic.ProviderSpec {
	return republic.ProviderSpec{Name: "codex", APIBase: "https://chatgpt.com/backend-api/codex", ModelsPath: "/models",
		Formats: []string{"responses"},
		ModelsParams: func(previous any) map[string]string {
			if previous != nil {
				return nil
			}
			return map[string]string{"client_version": codexClientVersion}
		},
		ParseModels: func(data any) ([]republic.ModelInfo, error) {
			root, _ := data.(map[string]any)
			items, _ := root["models"].([]any)
			var models []republic.ModelInfo
			for _, x := range items {
				item, _ := x.(map[string]any)
				slug, _ := item["slug"].(string)
				if slug == "" || item["visibility"] == "hide" {
					continue
				}
				name, _ := item["display_name"].(string)
				models = append(models, republic.ModelInfo{ID: slug, DisplayName: name, Raw: item})
			}
			return models, nil
		}}
}

// retries bounds Republic's retries: a dictation waits on the request.
var retries = 1

// republicName maps an Arion provider id onto Republic's registry. The
// compatible ids use the fixed provider's wire protocol at the caller's base.
func republicName(id string) string {
	switch id {
	case OpenAICompatible:
		return OpenAI
	case AnthropicCompatible:
		return Anthropic
	default:
		return id
	}
}

// normalizeBaseURL accepts an http or https address and drops trailing
// slashes. A query, fragment, or userinfo is rejected: the address is joined
// with a request path.
func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("base URL must be an http or https address")
	}
	return raw, nil
}

// open builds the Republic provider for a credential. auth is the ChatGPT
// signer; API keys go in the config. base replaces the endpoint in tests.
// A compatible credential supplies its own base when base is empty, so the
// product path does not read REPUBLIC_*_API_BASE. OpenAI-compatible calls
// use chat completions; the fixed OpenAI provider keeps Republic's default.
func open(c Credential, auth republic.Auth, client *http.Client, base string) (*republic.Provider, error) {
	apiBase := base
	if apiBase == "" && NeedsBase(c.Provider) {
		apiBase = c.BaseURL
	}
	config := republic.ProviderConfig{HTTPClient: client, APIBase: apiBase, MaxRetries: &retries, Timeout: 30 * time.Second}
	if c.Provider == ChatGPT {
		config.Auth = auth
		config.Headers = http.Header{}
		config.Headers.Set("originator", originator)
		return republic.NewProvider(codexSpec(), config)
	}
	if c.Provider == OpenAICompatible {
		config.APIFormat = "chat"
	}
	config.APIKey = c.APIKey
	return republic.GetProvider(republicName(c.Provider), config)
}

// listModels returns the provider's chat model IDs, sorted.
func (s *Service) listModels(ctx context.Context, p *republic.Provider) ([]string, error) {
	infos, err := p.ListModels(ctx)
	if err != nil {
		err = classify(err, CodeModels)
		status := 0
		var failure *republic.Error
		if errors.As(err, &failure) && failure.StatusCode >= 100 && failure.StatusCode <= 599 {
			status = failure.StatusCode
		}
		s.mu.Lock()
		listeners := append([]func(Code, int){}, s.modelFailures...)
		s.mu.Unlock()
		for _, listener := range listeners {
			listener(CodeOf(err), status)
		}
		return nil, err
	}
	ids := make([]string, 0, len(infos))
	seen := map[string]bool{}
	for _, m := range infos {
		if m.ID != "" && !seen[m.ID] {
			seen[m.ID] = true
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}
