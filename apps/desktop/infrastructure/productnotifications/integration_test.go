package productnotifications

import (
	policy "arion.local/desktop/internal/productnotifications"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

type rewriteTransport struct {
	origin *url.URL
	base   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	u := *req.URL
	u.Scheme = t.origin.Scheme
	u.Host = t.origin.Host
	copy.URL = &u
	copy.Host = t.origin.Host
	return t.base.RoundTrip(copy)
}

// TestProductNotificationBackendIntegration consumes only a root-owned loopback
// fixture. It validates actual bearer authorization and inbox/receipt contracts,
// not OS notification delivery. It never contacts a production service.
func TestProductNotificationBackendIntegration(t *testing.T) {
	path := os.Getenv("ARION_NOTIFICATION_E2E_CONFIG")
	if path == "" {
		t.Skip("set ARION_NOTIFICATION_E2E_CONFIG to an isolated loopback fixture config")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("read fixture config")
	}
	var cfg struct{ Origin, Token, Subject, NotificationID string }
	if json.Unmarshal(raw, &cfg) != nil {
		t.Fatal("invalid fixture config")
	}
	origin, err := url.Parse(cfg.Origin)
	if err != nil || origin.Scheme != "http" || !net.ParseIP(origin.Hostname()).IsLoopback() || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || cfg.Token == "" || cfg.Subject == "" || !policy.ValidMessageID(cfg.NotificationID) {
		t.Fatal("fixture must be a loopback-only isolated backend")
	}
	const logicalOrigin = "https://notification-e2e.invalid"
	client, err := NewClient(logicalOrigin, &http.Client{Transport: rewriteTransport{origin, http.DefaultTransport}, Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	session := policy.Session{Issuer: logicalOrigin, Subject: cfg.Subject, Token: cfg.Token}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var found *policy.Message
	cursor := ""
	for i := 0; i < 20; i++ {
		page, err := client.Inbox(ctx, session, cursor)
		if err != nil {
			t.Fatal(err)
		}
		if !page.Enabled {
			t.Fatal("fixture notifications disabled")
		}
		for _, m := range page.Items {
			if m.ID == cfg.NotificationID {
				copy := m
				found = &copy
			}
		}
		if found != nil || page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if found == nil {
		t.Fatal("target notification not in authenticated product inbox")
	}
	receipt, err := client.Receipt(ctx, session, found.ID, "received", "")
	if err != nil || receipt.ReceivedAt == nil {
		t.Fatalf("received receipt: %v", err)
	}
	receipt, err = client.Receipt(ctx, session, found.ID, "read", "")
	if err != nil || receipt.ReadAt == nil {
		t.Fatalf("read receipt: %v", err)
	}
	for _, button := range found.Buttons {
		if button.Action == "open_url" {
			receipt, err = client.Receipt(ctx, session, found.ID, "action", button.ID)
			if err != nil {
				t.Fatal(err)
			}
			match := false
			for _, id := range receipt.ActionIDs {
				if id == button.ID {
					match = true
				}
			}
			if !match {
				t.Fatal("authorized action missing from receipt")
			}
			break
		}
	}
	receipt, err = client.Receipt(ctx, session, found.ID, "dismissed", "")
	if err != nil || receipt.DismissedAt == nil {
		t.Fatalf("dismissed receipt: %v", err)
	}
}
