package cloudpolish

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"arion.local/core/domain"
)

func TestCompletionFailureCategories(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"reasoning required", 400, `{"error":{"message":"Reasoning is mandatory for this endpoint and cannot be disabled.","code":400}}`, domain.PolishReasoningRequired},
		{"rate limited", 429, `{"error":{"message":"slow down","code":429}}`, domain.PolishRateLimited},
		{"credits exhausted", 402, `{"error":{"message":"insufficient credits","code":402}}`, domain.PolishQuotaExhausted},
		{"key rejected", 401, `{"error":{"message":"bad key","code":401}}`, domain.PolishCredentialRejected},
		{"forbidden stays generic", 403, `{"error":{"message":"forbidden","code":403}}`, ""},
		{"model not found", 404, `{"error":{"message":"no such model","code":404}}`, domain.PolishModelNotFound},
		{"overloaded", 503, `{"error":{"message":"overloaded","code":503}}`, domain.PolishModelUnavailable},
		{"other bad request", 400, `{"error":{"message":"invalid request","code":400}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			err := rewriteOnce(t, strictService(t, srv.URL))
			if err == nil {
				t.Fatal("expected failure")
			}
			if got := domain.DetailOf(err); got != tc.want {
				t.Fatalf("detail %q, want %q (%v)", got, tc.want, err)
			}
		})
	}
}
