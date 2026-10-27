package bootstrap

import (
	"context"
	"slices"
	"testing"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"path/filepath"

	"arion.local/desktop/infrastructure/cloudpolish"
	"arion.local/desktop/infrastructure/keystore"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/desktop/infrastructure/settings"
	arion "arion.local/desktop/ui/shared/runtime"
)

type noSecrets struct{}

func (noSecrets) Put(string, string) error   { return nil }
func (noSecrets) Get(string) (string, error) { return "", nil }
func (noSecrets) Delete(string) error        { return nil }

var _ cloudpolish.SecretStore = keystore.Catalog{}

// Both engines honor the same context switches; cloud stages run after stop.
func TestCloudPolicyUsesSharedContext(t *testing.T) {
	a := &App{prefs: application.DefaultPreferences("darwin"), rewriter: &rewriting.Service{}}
	a.prefs.Polish = true
	local := a.rewritePolicy()
	if !local.Enabled || !local.Context.Screen || !local.Context.Cursor || !local.Context.History {
		t.Fatalf("local policy %+v", local)
	}
	a.prefs.PolishEngine = application.PolishCloud
	cloud := a.rewritePolicy()
	if !cloud.Enabled || cloud.Context != local.Context || cloud.Timeout != cloudRewriteTimeout {
		t.Fatalf("cloud policy %+v", cloud)
	}
}

// The router follows the engine preference per request.
func TestPolishRouterFollowsEngine(t *testing.T) {
	a := &App{prefs: application.DefaultPreferences("darwin"), rewriter: &rewriting.Service{},
		cloud: cloudpolish.New(noSecrets{}, func(string) error { return nil }, nil)}
	router := polishRouter{a}
	if router.engine() != ports.Rewriter(a.rewriter) {
		t.Fatal("local engine not routed to the worker")
	}
	a.prefs.PolishEngine = application.PolishCloud
	if router.engine() != ports.Rewriter(a.cloud) {
		t.Fatal("cloud engine not routed to the cloud")
	}
	// Without a chosen model the cloud fails explicitly; the coordinator
	// records it as unavailable.
	if _, err := router.Rewrite(context.Background(), ports.RewriteRequest{Text: "x"}); domain.CodeOf(err) != domain.CodeModelMissing {
		t.Fatalf("unconfigured cloud: %v", err)
	}
}

// The UI lists the providers the cloud service connects.
func TestCloudProvidersMatchUI(t *testing.T) {
	if !slices.Equal(arion.CloudKeyProviders, cloudpolish.KeyProviders) || arion.CloudChatGPT != cloudpolish.ChatGPT {
		t.Fatalf("UI %v, service %v", arion.CloudKeyProviders, cloudpolish.KeyProviders)
	}
}

func TestContextSwitchesApplyToBothEngines(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		a := &App{prefs: application.DefaultPreferences("darwin")}
		a.prefs.Polish = true
		a.prefs.ContextCursor = mask&1 != 0
		a.prefs.ContextScreen = mask&2 != 0
		a.prefs.ContextHistory = mask&4 != 0
		want := application.ContextPolicy{Cursor: a.prefs.ContextCursor, Screen: a.prefs.ContextScreen, History: a.prefs.ContextHistory}
		for _, engine := range []application.PolishEngine{application.PolishLocal, application.PolishCloud} {
			a.prefs.PolishEngine = engine
			if got := a.rewritePolicy().Context; got != want {
				t.Fatalf("%s mask %d: %+v", engine, mask, got)
			}
		}
		a.prefs.Polish = false
		if got := a.rewritePolicy(); got.Enabled || got.Context.Any() {
			t.Fatalf("off: %+v", got)
		}
	}
}

// Switching the connected provider restores that provider's model. The first
// load keeps a model that was already chosen, and disconnecting clears it.
func TestApplyCloudRestoresProviderModel(t *testing.T) {
	dir := t.TempDir()
	cloud := cloudpolish.New(noSecrets{}, func(string) error { return nil }, nil)
	a := &App{
		prefs:     application.Preferences{CloudModel: "deepseek-chat"},
		prefStore: settings.Store{Path: filepath.Join(dir, "preferences.json")},
		cloud:     cloud,
		state:     arion.State{Preferences: application.Preferences{CloudModel: "deepseek-chat"}},
	}
	a.applyCloud(cloudpolish.Status{Phase: cloudpolish.Connected, Provider: cloudpolish.DeepSeek, Model: "deepseek-chat",
		Saved: []string{cloudpolish.DeepSeek}})
	if a.cloudProvider != cloudpolish.DeepSeek || a.preferences().CloudModel != "deepseek-chat" || !slices.Equal(a.state.Cloud.Saved, []string{cloudpolish.DeepSeek}) {
		t.Fatalf("load %+v model %q provider %q", a.state.Cloud, a.preferences().CloudModel, a.cloudProvider)
	}
	a.applyCloud(cloudpolish.Status{Phase: cloudpolish.Connected, Provider: cloudpolish.OpenRouter, Model: "or-model",
		Saved: []string{cloudpolish.DeepSeek, cloudpolish.OpenRouter}})
	if a.preferences().CloudModel != "or-model" || a.state.Preferences.CloudModel != "or-model" || a.state.Cloud.Provider != cloudpolish.OpenRouter {
		t.Fatalf("switch model %q state %+v", a.preferences().CloudModel, a.state.Cloud)
	}
	a.applyCloud(cloudpolish.Status{Phase: cloudpolish.Verifying, Provider: cloudpolish.DeepSeek})
	if a.preferences().CloudModel != "or-model" || a.cloudProvider != cloudpolish.OpenRouter {
		t.Fatalf("key check changed model %q provider %q", a.preferences().CloudModel, a.cloudProvider)
	}
	a.applyCloud(cloudpolish.Status{Phase: cloudpolish.Disconnected})
	if a.preferences().CloudModel != "" || a.cloudProvider != "" || a.state.Cloud.Phase != arion.CloudDisconnected {
		t.Fatalf("disconnect model %q provider %q phase %q", a.preferences().CloudModel, a.cloudProvider, a.state.Cloud.Phase)
	}
	a.applyCloud(cloudpolish.Status{Phase: cloudpolish.Disconnected, Failure: cloudpolish.CodeRejected, FailureProvider: cloudpolish.Anthropic})
	if a.state.Cloud.Failure != arion.CloudCodeRejected || a.state.Cloud.FailureProvider != cloudpolish.Anthropic {
		t.Fatalf("failure %+v", a.state.Cloud)
	}
}
