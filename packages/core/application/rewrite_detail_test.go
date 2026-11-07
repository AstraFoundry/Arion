package application

import (
	"errors"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

type detailedError struct{ detail string }

func (e detailedError) Error() string         { return "provider refused" }
func (e detailedError) FailureDetail() string { return e.detail }

// A rewrite that falls back to local cleanup reports a content-free category
// on the correction event so the UI can say why polishing did not run.
func TestFallbackCarriesFailureDetail(t *testing.T) {
	cases := []struct {
		name     string
		rewriter *scriptedRewriter
		want     string
	}{
		{"categorized", &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
			return "", detailedError{domain.PolishRateLimited}
		}}, domain.PolishRateLimited},
		{"uncategorized", &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) {
			return "", errors.New("boom")
		}}, ""},
		{"timeout", &scriptedRewriter{block: make(chan struct{}), answer: func(req ports.RewriteRequest) (string, error) {
			return req.Text, nil
		}}, domain.PolishTimeout},
		{"accepted", &scriptedRewriter{answer: func(ports.RewriteRequest) (string, error) { return "你好，世界。", nil }}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, audio, events, _, _ := liveSetup(t, []string{"你好，世界。"}, tc.rewriter,
				RewritePolicy{Enabled: true, Scene: "chat", Timeout: 30 * time.Millisecond})
			c.Start()
			events.wait(t, stateIs(domain.Capturing))
			audio.push(1, 0.2)
			c.Stop()
			ready := events.wait(t, kindIs(contracts.CorrectionReady))
			if ready.Detail != tc.want {
				t.Fatalf("detail %q, want %q", ready.Detail, tc.want)
			}
		})
	}
}
