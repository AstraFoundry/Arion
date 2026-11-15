package cloudpolish

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"arion.local/core/domain"
)

func TestCompletionFailureCategories(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		want       string
		wantCode   domain.ErrorCode
		wantReason string
	}{
		{"reasoning required", 400, `{"error":{"message":"Reasoning is mandatory for this endpoint and cannot be disabled.","code":400}}`, domain.PolishReasoningRequired, domain.CodeModelUnsupported, domain.ReasonCloudReasoningRequired},
		{"rate limited", 429, `{"error":{"message":"slow down","code":429}}`, domain.PolishRateLimited, domain.CodeRewriteRejected, domain.ReasonCloudRateLimited},
		{"credits exhausted", 402, `{"error":{"message":"insufficient credits","code":402}}`, domain.PolishQuotaExhausted, domain.CodeRewriteRejected, domain.ReasonCloudQuotaExhausted},
		{"key rejected", 401, `{"error":{"message":"bad key","code":401}}`, domain.PolishCredentialRejected, domain.CodeRewriteRejected, domain.ReasonCloudRejected},
		{"forbidden stays generic", 403, `{"error":{"message":"forbidden","code":403}}`, "", domain.CodeRewriteRejected, domain.ReasonCloudForbidden},
		{"model not found", 404, `{"error":{"message":"no such model","code":404}}`, domain.PolishModelNotFound, domain.CodeModelMissing, domain.ReasonCloudModelNotFound},
		{"overloaded", 503, `{"error":{"message":"overloaded","code":503}}`, domain.PolishModelUnavailable, domain.CodeEvaluationUnavailable, domain.ReasonCloudServerError},
		{"other bad request", 400, `{"error":{"message":"invalid request","code":400}}`, "", domain.CodeRewriteRejected, domain.ReasonCloudBadRequest},
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
			if got := domain.DiagnosticCodeOf(err); got != tc.wantCode {
				t.Fatalf("diagnostic code %q, want %q (%v)", got, tc.wantCode, err)
			}
			if got := domain.PhaseOf(err); got != domain.PhaseFormatting {
				t.Fatalf("phase %q, want %q (%v)", got, domain.PhaseFormatting, err)
			}
			if got := domain.ReasonOf(err); got != tc.wantReason {
				t.Fatalf("reason %q, want %q (%v)", got, tc.wantReason, err)
			}
		})
	}
}
