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

// Cloud polishing sends one request per dictation and reads no context;
// local polishing keeps previews and the context switches (ADR 022).
func TestCloudPolicyIsFinalOnlyWithoutContext(t *testing.T) {
	a := &App{prefs: application.DefaultPreferences("darwin"), rewriter: &rewriting.Service{}}
	a.prefs.Polish = true
	local := a.rewritePolicy()
	if !local.Enabled || local.FinalOnly || !local.Context.Screen || !local.Context.Cursor || !local.Context.History {
		t.Fatalf("local policy %+v", local)
	}
	a.prefs.PolishEngine = application.PolishCloud
	cloud := a.rewritePolicy()
	if !cloud.Enabled || !cloud.FinalOnly || cloud.Context != (application.ContextPolicy{}) || cloud.Timeout != cloudRewriteTimeout {
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
