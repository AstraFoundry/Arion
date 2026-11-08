package bootstrap

import (
	"testing"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func TestPolishNoteIsToldOncePerModel(t *testing.T) {
	a := &App{prefs: application.Preferences{PolishEngine: application.PolishCloud, CloudModel: "m1"}, cloudProvider: "openrouter"}
	event := contracts.Event{Rewrite: application.RewriteUnavailable, Detail: domain.PolishRateLimited}
	if got := a.polishNote(event); got != domain.PolishRateLimited {
		t.Fatalf("first failure: %q", got)
	}
	if got := a.polishNote(event); got != domain.PolishRateLimited {
		t.Fatalf("a note that was never shown must not be spent: %q", got)
	}
	a.polishNoteTold(domain.PolishRateLimited)
	if got := a.polishNote(event); got != "" {
		t.Fatalf("repeat told again: %q", got)
	}
	a.prefs.CloudModel = "m2"
	if got := a.polishNote(event); got != domain.PolishRateLimited {
		t.Fatalf("another model: %q", got)
	}
	for _, e := range []contracts.Event{
		{Rewrite: string(domain.RewriteAccepted), Detail: domain.PolishRateLimited},
		{Rewrite: application.RewriteUnavailable},
	} {
		if got := a.polishNote(e); got != "" {
			t.Fatalf("%+v told %q", e, got)
		}
	}
	a.prefs.PolishEngine = application.PolishLocal
	if got := a.polishNote(contracts.Event{Rewrite: application.RewriteTimeout, Detail: domain.PolishTimeout}); got != "" {
		t.Fatalf("local engine told %q", got)
	}
}
