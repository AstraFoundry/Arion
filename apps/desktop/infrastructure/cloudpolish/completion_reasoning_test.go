package cloudpolish

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"arion.local/core/domain/ports"
)

// reasoningServer refuses every effort in refused with OpenRouter's pre-routing
// 400 and records the efforts it saw.
func reasoningServer(t *testing.T, refused ...string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		reasoning, _ := body["reasoning"].(map[string]any)
		effort, _ := reasoning["effort"].(string)
		mu.Lock()
		seen = append(seen, effort)
		mu.Unlock()
		for _, r := range refused {
			if effort == r {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"Reasoning is mandatory for this endpoint and cannot be disabled.","code":400}}`))
				return
			}
		}
		_ = json.NewEncoder(w).Encode(thinkingOffResponse(r.URL.Path))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string{}, seen...) }
}

func rewriteOnce(t *testing.T, s *Service) error {
	t.Helper()
	_, err := s.Rewrite(context.Background(), ports.RewriteRequest{Text: "今天天气很好。", Scene: "chat", Punctuation: "omit"})
	return err
}

// strictService keeps the default: reasoning stays off and a refusal fails.
func strictService(t *testing.T, base string) *Service {
	t.Helper()
	s, err := NewAPICompletion(OpenRouter, "synthetic-key", "google/mandatory", base, "chat")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.CloseCompletion)
	return s
}

// openRouterService has the user's reasoning fallback switch turned on.
func openRouterService(t *testing.T, base string) *Service {
	t.Helper()
	s := strictService(t, base)
	s.SetReasoningFallback(true)
	return s
}

func TestMandatoryReasoningFailsWhenFallbackIsOff(t *testing.T) {
	srv, seen := reasoningServer(t, "none")
	s := strictService(t, srv.URL)
	if err := rewriteOnce(t, s); err == nil {
		t.Fatal("a model that requires reasoning must fail while the switch is off")
	}
	if got := seen(); len(got) != 1 || got[0] != "none" {
		t.Fatalf("efforts %v, want a single none request", got)
	}
}

func TestTurningFallbackOffIgnoresTheRememberedEffort(t *testing.T) {
	srv, seen := reasoningServer(t, "none")
	s := openRouterService(t, srv.URL)
	if err := rewriteOnce(t, s); err != nil {
		t.Fatal(err)
	}
	s.SetReasoningFallback(false)
	before := len(seen())
	if err := rewriteOnce(t, s); err == nil {
		t.Fatal("a remembered effort must not outlive the switch")
	}
	if got := seen()[before:]; len(got) != 1 || got[0] != "none" {
		t.Fatalf("efforts after turning off: %v", got)
	}
}

func TestFallbackSwitchAppliesWithoutReconnecting(t *testing.T) {
	srv, seen := reasoningServer(t, "none")
	s := strictService(t, srv.URL)
	if err := rewriteOnce(t, s); err == nil {
		t.Fatal("expected failure while off")
	}
	s.SetReasoningFallback(true)
	if err := rewriteOnce(t, s); err != nil {
		t.Fatal(err)
	}
	if got := seen(); got[len(got)-1] != "minimal" {
		t.Fatalf("efforts %v", got)
	}
}

func TestMandatoryReasoningFallsBackToMinimal(t *testing.T) {
	srv, seen := reasoningServer(t, "none")
	s := openRouterService(t, srv.URL)
	if err := rewriteOnce(t, s); err != nil {
		t.Fatal(err)
	}
	if got := seen(); len(got) < 2 || got[0] != "none" || got[1] != "minimal" {
		t.Fatalf("efforts %v", got)
	}
}

func TestMandatoryReasoningFallsBackToLow(t *testing.T) {
	srv, seen := reasoningServer(t, "none", "minimal")
	s := openRouterService(t, srv.URL)
	if err := rewriteOnce(t, s); err != nil {
		t.Fatal(err)
	}
	if got := seen(); len(got) < 3 || got[0] != "none" || got[1] != "minimal" || got[2] != "low" {
		t.Fatalf("efforts %v", got)
	}
}

func TestAcceptedEffortIsReused(t *testing.T) {
	srv, seen := reasoningServer(t, "none")
	s := openRouterService(t, srv.URL)
	if err := rewriteOnce(t, s); err != nil {
		t.Fatal(err)
	}
	first := len(seen())
	if err := rewriteOnce(t, s); err != nil {
		t.Fatal(err)
	}
	for _, effort := range seen()[first:] {
		if effort != "minimal" {
			t.Fatalf("later request sent %q, want the accepted minimal", effort)
		}
	}
}

func TestRefusedEveryEffortFails(t *testing.T) {
	srv, seen := reasoningServer(t, "none", "minimal", "low")
	s := openRouterService(t, srv.URL)
	if err := rewriteOnce(t, s); err == nil {
		t.Fatal("expected failure when no effort is accepted")
	}
	for _, effort := range seen() {
		if effort == "medium" || effort == "high" {
			t.Fatalf("fell back above low: %v", seen())
		}
	}
}

func TestUnrelatedBadRequestIsNotRetried(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid request","code":400}}`))
	}))
	defer srv.Close()
	s := openRouterService(t, srv.URL)
	if err := rewriteOnce(t, s); err == nil {
		t.Fatal("expected failure")
	}
	if calls != 1 {
		t.Fatalf("calls %d, want one", calls)
	}
}
