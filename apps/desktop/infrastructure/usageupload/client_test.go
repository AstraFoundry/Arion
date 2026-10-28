package usageupload

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	policy "arion.local/desktop/internal/usageupload"
)

func TestReportUsesOnlyAggregateFieldsAndNeverFollowsRedirects(t *testing.T) {
	var seen map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/api/v1/usage" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("unexpected request")
		}
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"productSlug":"arion","date":"2026-10-28","visited":true,"sessions":1,"speechMs":60000,"words":120},"error":null}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	session := policy.Session{Issuer: server.URL, Subject: "user-1", Token: "test-token"}
	day := policy.Day{Date: "2026-10-28", Visited: true, Sessions: 1, SpeechMs: 60000, Words: 120}
	if _, err := client.Report(context.Background(), session, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", day); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 7 || seen["sourceId"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || seen["productSlug"] != "arion" {
		t.Fatalf("unexpected fields: %v", seen)
	}
	var redirected bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected = true }))
	defer other.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	client, _ = NewClient(redirect.URL, nil)
	session.Issuer = redirect.URL
	if _, err := client.Report(context.Background(), session, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", day); err == nil || redirected {
		t.Fatal("redirect was followed")
	}
}

func TestReportRefusesCrossIssuerAndMismatchedAcknowledgement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"productSlug":"arion","date":"2026-10-27","visited":true},"error":null}`))
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, nil)
	session := policy.Session{Issuer: "https://other.example", Subject: "u", Token: "test"}
	day := policy.Day{Date: "2026-10-28", Visited: true}
	if _, err := client.Report(context.Background(), session, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", day); !errors.Is(err, policy.ErrAuthorization) {
		t.Fatal(err)
	}
	session.Issuer = server.URL
	if _, err := client.Report(context.Background(), session, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", day); !errors.Is(err, policy.ErrPayload) {
		t.Fatal(err)
	}
	if _, err := NewClient("http://external.example", nil); err == nil {
		t.Fatal("external HTTP accepted")
	}
}

func TestLedgerSurvivesRestartWithStableSource(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "ledger.json")}
	ledger, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	ledger.Account = "account-1"
	if err := store.Save(ledger); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Source != ledger.Source || len(restored.Source) != 32 || restored.Account != "account-1" {
		t.Fatalf("unstable ledger: %+v", restored)
	}
}

func TestReportRequiresProductAndMonotonicAcknowledgement(t *testing.T) {
	for _, test := range []struct {
		name      string
		product   string
		sessions  int
		speech    int64
		words     int
		wantError bool
	}{
		{"exact", "arion", 1, 60000, 120, false},
		{"newer server snapshot", "arion", 2, 90000, 180, false},
		{"wrong product", "other", 1, 60000, 120, true},
		{"missing product", "", 1, 60000, 120, true},
		{"fewer sessions", "arion", 0, 60000, 120, true},
		{"less microphone time", "arion", 1, 0, 120, true},
		{"fewer words", "arion", 1, 60000, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "error": nil, "data": map[string]any{
					"productSlug": test.product, "date": "2026-10-28", "visited": true, "sessions": test.sessions, "speechMs": test.speech, "words": test.words,
				}})
			}))
			defer server.Close()
			client, err := NewClient(server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Report(context.Background(), policy.Session{Issuer: server.URL, Subject: "user", Token: "token"}, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", policy.Day{Date: "2026-10-28", Visited: true, Sessions: 1, SpeechMs: 60000, Words: 120})
			if test.wantError && !errors.Is(err, policy.ErrPayload) {
				t.Fatalf("expected payload error, got %v", err)
			}
			if !test.wantError && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReportRequiresVisitAcknowledgement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"productSlug":"arion","date":"2026-10-28","visited":false,"sessions":0,"speechMs":0,"words":0},"error":null}`))
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, nil)
	_, err := client.Report(context.Background(), policy.Session{Issuer: server.URL, Subject: "user", Token: "token"}, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", policy.Day{Date: "2026-10-28", Visited: true})
	if !errors.Is(err, policy.ErrPayload) {
		t.Fatalf("missing visit accepted: %v", err)
	}
}
