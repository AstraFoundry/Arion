package bootstrap

import (
	"context"
	"slices"
	"testing"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/cloudpolish"
	"arion.local/desktop/infrastructure/rewriting"
	arion "arion.local/desktop/ui/shared/runtime"
)

type noSecrets struct{}

func (noSecrets) Put(string) error     { return nil }
func (noSecrets) Get() (string, error) { return "", nil }
func (noSecrets) Delete() error        { return nil }

// Both engines honor the same context switches; cloud stages run after stop.
func TestCloudPolicyUsesSharedContext(t *testing.T) {
	a := &App{prefs: application.DefaultPreferences("darwin"), rewriter: &rewriting.Service{}}
	a.prefs.Polish = true
	local := a.rewritePolicy()
	if !local.Enabled || !local.FinalOnly || !local.Context.Screen || !local.Context.Cursor || !local.Context.History {
		t.Fatalf("local policy %+v", local)
	}
	a.prefs.PolishEngine = application.PolishCloud
	cloud := a.rewritePolicy()
	if !cloud.Enabled || !cloud.FinalOnly || cloud.Context != local.Context || cloud.Timeout != cloudRewriteTimeout {
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
