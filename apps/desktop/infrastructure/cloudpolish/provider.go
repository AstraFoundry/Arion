// Package cloudpolish is the optional cloud polishing capability (ADR 022):
// the connected provider's credential in the OS credential store, its model
// list, and a ports.Rewriter using the shared staged pipeline. Enabled context
// sources reach spelling selection through Republic; the formatter receives
// only the edited source. Cloud polishing requires an explicit engine choice.
package cloudpolish

import (
	"context"
	"net/http"
	"sort"
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
)

// KeyProviders accept an API key, in the order the picker lists them.
var KeyProviders = []string{OpenAI, Anthropic, Google, DeepSeek, OpenRouter}

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

// open builds the Republic provider for a credential. auth is the ChatGPT
// signer; API keys go in the config. base replaces the endpoint in tests.
func open(c Credential, auth republic.Auth, client *http.Client, base string) (*republic.Provider, error) {
	config := republic.ProviderConfig{HTTPClient: client, APIBase: base, MaxRetries: &retries, Timeout: 30 * time.Second}
	if c.Provider == ChatGPT {
		config.Auth = auth
		config.Headers = http.Header{}
		config.Headers.Set("originator", originator)
		return republic.NewProvider(codexSpec(), config)
	}
	config.APIKey = c.APIKey
	return republic.GetProvider(c.Provider, config)
}

// listModels returns the provider's chat model IDs, sorted.
func listModels(ctx context.Context, p *republic.Provider) ([]string, error) {
	infos, err := p.ListModels(ctx)
	if err != nil {
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
