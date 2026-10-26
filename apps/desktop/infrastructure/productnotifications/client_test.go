package productnotifications

import (
	policy "arion.local/desktop/internal/productnotifications"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixtureMessage() policy.Message {
	return policy.Message{ID: "12345678-1234-1234-1234-123456789abc", ProductSlug: policy.Product, Title: "PRIVATE_TITLE_CANARY", Body: "PRIVATE_BODY_CANARY", PublishedAt: time.Now().Add(-time.Hour), Buttons: []policy.Button{{ID: "visit", Label: "View", Action: "open_url", URL: "https://example.com/news"}}}
}
func envelope(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data, "error": nil})
}
func TestClientStoreAndActionLoopbackLifecycle(t *testing.T) {
	m := fixtureMessage()
	var events []string
	blocked := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer TOKEN_CANARY" {
			w.WriteHeader(401)
			return
		}
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("productSlug") != "arion" || r.URL.Query().Get("limit") != "20" {
				t.Error("wrong product/limit")
			}
			envelope(w, policy.Page{Enabled: true, AllowedActionHosts: []string{"example.com"}, Items: []policy.Message{m}})
			return
		}
		var body struct{ ProductSlug, Event, ActionID string }
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.ProductSlug != "arion" {
			t.Error("invalid receipt")
		}
		events = append(events, body.Event)
		if blocked {
			w.WriteHeader(404)
			return
		}
		now := time.Now()
		receipt := policy.Receipt{}
		switch body.Event {
		case "received":
			receipt.ReceivedAt = &now
		case "read":
			receipt.ReadAt = &now
		case "action":
			if body.ActionID != "visit" {
				t.Error("wrong action")
			}
			receipt.ActionIDs = []string{body.ActionID}
		}
		envelope(w, receipt)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Path: filepath.Join(t.TempDir(), "notices.json")}
	native := &recordTransport{}
	opened := []string{}
	service, err := policy.New(client, store, native, func(url string) error { opened = append(opened, url); return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Stop()
	session := policy.Session{Issuer: server.URL, Subject: "account-a", Token: "TOKEN_CANARY"}
	service.Configure(session, true, true, true)
	if err := service.Poll(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(native.ids) != 1 || strings.Join(events, ",") != "received" {
		t.Fatal("fetch/submission/receipt pipeline failed")
	}
	raw, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{m.Title, m.Body, session.Token, session.Subject} {
		if strings.Contains(string(raw), canary) {
			t.Fatal("ledger persisted body, credentials, or raw subject")
		}
	}
	blocked = true
	if err := service.Activate(context.Background(), native.ids[0], "button-0", time.Now()); err == nil {
		t.Fatal("opened expired/revoked notification")
	}
	if len(opened) != 0 {
		t.Fatal("URL opened on backend refusal")
	}
	blocked = false
	if err := service.Activate(context.Background(), native.ids[0], "button-0", time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != m.Buttons[0].URL {
		t.Fatal("authorized URL not opened")
	}
}

type recordTransport struct{ ids []string }

func (t *recordTransport) Show(ctx context.Context, m policy.Message, id string) error {
	t.ids = append(t.ids, id)
	return nil
}
func (t *recordTransport) Remove(string) {}
func TestClientRefusesRedirectAndInvalidPayload(t *testing.T) {
	tests := []struct {
		name   string
		status int
		data   any
	}{{"cross product", 200, policy.Page{Enabled: true, Items: []policy.Message{{ID: fixtureMessage().ID, ProductSlug: "other", Title: "x", Body: "x", PublishedAt: time.Now()}}, AllowedActionHosts: []string{}}}, {"authorization", 403, nil}, {"redirect", 302, nil}, {"malformed", 200, map[string]any{"items": []any{}}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != 200 {
					w.Header().Set("Location", "https://untrusted.invalid")
					w.WriteHeader(tt.status)
					return
				}
				envelope(w, tt.data)
			}))
			defer server.Close()
			client, err := NewClient(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Inbox(context.Background(), policy.Session{Issuer: server.URL, Subject: "a", Token: "canary"}, "")
			if err == nil {
				t.Fatal("accepted invalid inbox")
			}
		})
	}
	if _, err := NewClient("http://127.0.0.1:3000", nil); !errors.Is(err, policy.ErrPayload) {
		t.Fatal("production client permits plaintext origin")
	}
}
func TestStoreRejectsCorruptionAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	store := Store{Path: path}
	if _, err := store.Load(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema":99,"records":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, policy.ErrStore) {
		t.Fatal("corrupt ledger silently reset")
	}
	target := filepath.Join(dir, "target.json")
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skip("symlinks unavailable")
	}
	if _, err := store.Load(); !errors.Is(err, policy.ErrStore) {
		t.Fatal("followed symlink")
	}
}
