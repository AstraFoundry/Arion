package cloudpolish

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestModelListFailuresDoNotRejectValidCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		code   Code
	}{
		{"unauthorized", 401, `{"error":"invalid key"}`, CodeRejected},
		{"cdn forbidden", 403, `<html>The request could not be satisfied.</html>`, CodeModels},
		{"rate limited", 429, `{"error":"rate limited"}`, CodeModels},
		{"server error", 500, `{"error":"unavailable"}`, CodeModels},
		{"unavailable", 503, `{"error":"unavailable"}`, CodeModels},
		{"invalid response", 200, `not json`, CodeModels},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var failing atomic.Bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil || r.Method != http.MethodGet || r.URL.Path != "/models" || len(body) != 0 || r.ContentLength != 0 {
					t.Error("model list must use GET without a body")
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if failing.Load() {
					w.WriteHeader(tc.status)
					_, _ = io.WriteString(w, tc.body)
					return
				}
				_, _ = io.WriteString(w, `{"data":[{"id":"deepseek-flash"}]}`)
			}))
			defer srv.Close()
			secrets := &memorySecrets{}
			s := newTestService(secrets, srv.URL)
			var failures []string
			s.OnModelListFailure(func(code Code, status int) {
				_ = s.Status() // Observers must run outside the service lock.
				failures = append(failures, fmt.Sprintf("%s:%d", code, status))
			})
			if err := s.ConnectKey(DeepSeek, "synthetic-original-key", ""); err != nil {
				t.Fatal(err)
			}
			puts := secrets.puts
			failing.Store(true)
			if err := s.ConnectKey(DeepSeek, "synthetic-new-key", ""); CodeOf(err) != tc.code {
				t.Fatalf("key check code %s, want %s", CodeOf(err), tc.code)
			}
			st := s.Status()
			if st.Phase != Connected || st.Failure != tc.code || st.FailureProvider != DeepSeek || secrets.puts != puts {
				t.Fatalf("failed check changed connection: %+v, writes %d", st, secrets.puts)
			}
			if c := secrets.credential(t); c.APIKey != "synthetic-original-key" {
				t.Fatal("failed check replaced the saved key")
			}
			s.ListModels()
			if st := s.Status(); st.Failure != tc.code || st.Phase != Connected || len(st.Models) != 1 {
				t.Fatalf("failed refresh lost connection/models: %+v", st)
			}
			status := tc.status
			if status == http.StatusOK {
				status = 0 // A JSON parsing error carries no HTTP status.
			}
			want := fmt.Sprintf("%s:%d", tc.code, status)
			if len(failures) != 2 || failures[0] != want || failures[1] != want {
				t.Fatalf("diagnostics %v, want two %s failures", failures, want)
			}
			failing.Store(false)
			s.ListModels()
			if st := s.Status(); st.Failure != "" || len(st.Models) != 1 || len(failures) != 2 {
				t.Fatalf("retry did not recover: %+v, diagnostics %v", st, failures)
			}
		})
	}
}
