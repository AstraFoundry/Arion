package bootstrap

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"arion.local/desktop/internal/account"
	arion "arion.local/desktop/ui/shared/runtime"
)

func TestAccountRenewalSchedule(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, tc := range []struct{ lifetime, delay time.Duration }{
		{time.Hour, 59 * time.Minute}, {time.Minute, 54 * time.Second}, {-time.Second, time.Second},
	} {
		m := accountMaintenance{}
		m.renewed(account.Session{Expiry: now.Add(tc.lifetime)}, now)
		if got := m.next.Sub(now); got != tc.delay {
			t.Fatalf("lifetime %v: delay %v", tc.lifetime, got)
		}
	}
	m := accountMaintenance{}
	m.renewed(account.Session{}, now)
	if !m.next.IsZero() {
		t.Fatal("unknown expiry must not cause a refresh loop")
	}
	for range 20 {
		m.failed(now)
		if d := m.next.Sub(now); d < m.retry/2 || d > m.retry || d > 5*time.Minute {
			t.Fatalf("retry delay %v", d)
		}
	}
}

func TestAccountRetryClassification(t *testing.T) {
	for _, tc := range []struct {
		code   account.Code
		detail string
		retry  bool
	}{
		{account.CodeRefresh, "transport", true}, {account.CodeDiscovery, "http_503", true},
		{account.CodeRefresh, "http_429", true}, {account.CodeRefresh, "invalid_grant", false},
		{account.CodeRefresh, "not_found", false}, {account.CodeDiscovery, "issuer_mismatch", false},
		{account.CodeRefresh, "invalid_client", false}, {account.CodeStore, "read", true},
		{account.CodeStore, "unsupported", false},
	} {
		if got := accountRetryable(account.Wrap(tc.code, tc.detail, nil)); got != tc.retry {
			t.Fatalf("%s/%s: %v", tc.code, tc.detail, got)
		}
	}
}

type maintenanceStore struct {
	mu    sync.Mutex
	token string
}

func (s *maintenanceStore) Get() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		return "", account.ErrNotFound
	}
	return s.token, nil
}
func (s *maintenanceStore) Put(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
	return nil
}
func (s *maintenanceStore) Delete() error { return s.Put("") }

// Exercise the production credential check, refresh action, queue and retry
// together. No browser is involved in restoring an existing account.
func TestAccountAutomaticRecovery(t *testing.T) {
	var issuerURL string
	var mode string
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseRefresh) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q}`, issuerURL, issuerURL+"/authorize", issuerURL+"/token")
		case "/token":
			switch mode {
			case "blocked":
				close(refreshStarted)
				select {
				case <-r.Context().Done():
				case <-releaseRefresh:
				}
				w.WriteHeader(503)
				return
			case "offline":
				w.WriteHeader(503)
				fmt.Fprint(w, `{"error":"temporarily_unavailable"}`)
			case "revoked":
				w.WriteHeader(400)
				fmt.Fprint(w, `{"error":"invalid_grant"}`)
			default:
				token := "e30." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user","ent":[]}`)) + ".sig"
				fmt.Fprintf(w, `{"access_token":%q,"refresh_token":"rotated","expires_in":3600}`, token)
			}
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	defer release()
	issuerURL = server.URL
	store := &maintenanceStore{token: "stored"}
	svc, err := account.New(account.Config{Issuer: issuerURL}, account.Dependencies{Store: store, Browser: func(string) error { t.Error("background recovery opened browser"); return nil }})
	if err != nil {
		t.Fatal(err)
	}
	a := &App{accounts: svc, queueReady: make(chan struct{}, 1)}
	a.state.Account.Phase = arion.AccountSignedOut
	pump := func(done func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for !done() {
			if time.Now().After(deadline) {
				t.Fatalf("account operation timed out: %+v", a.state.Account)
			}
			select {
			case <-a.queueReady:
			case <-time.After(time.Millisecond):
			}
			a.queueMu.Lock()
			batch := a.queue
			a.queue = nil
			a.queueMu.Unlock()
			for _, fn := range batch {
				fn()
			}
		}
	}
	a.accountMaintenance.next = time.Now()
	a.maintainAccount(time.Now())
	pump(func() bool { return a.state.Account.Credential == arion.CredentialStored })
	a.maintainAccount(time.Now())
	pump(func() bool { return a.state.Account.Phase == arion.AccountSignedIn })
	if token, _ := store.Get(); token != "rotated" {
		t.Fatal("refresh token was not rotated")
	}
	if a.state.Account.Subject != "user" || a.accountMaintenance.next.IsZero() {
		t.Fatal("restoration did not schedule renewal")
	}
	mode = "offline"
	a.refreshAccount(true)
	pump(func() bool { return !a.state.Account.Busy() })
	if !a.state.Account.SignedIn() || a.state.Account.ErrorCode != "" || a.accountMaintenance.retry == 0 {
		t.Fatal("temporary failure lost identity or did not retry")
	}
	if token, _ := store.Get(); token == "" {
		t.Fatal("temporary failure deleted credential")
	}
	// A due retry recovers without a user action, including after sleep.
	mode = ""
	a.maintainAccount(time.Now().Add(5 * time.Minute))
	pump(func() bool { return !a.state.Account.Busy() })
	if a.accountMaintenance.retry != 0 {
		t.Fatal("successful retry did not reset backoff")
	}

	// Signing out while refresh is in flight cancels it and prevents the
	// queued completion or subsequent timer from restoring the account.
	mode = "blocked"
	a.refreshAccount(true)
	select {
	case <-refreshStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("refresh did not start")
	}
	actions{a}.SignOut()
	release()
	pump(func() bool { return a.state.Account.Phase == arion.AccountSignedOut })
	a.maintainAccount(time.Now().Add(time.Hour))
	if token, _ := store.Get(); token != "" || !a.accountMaintenance.next.IsZero() {
		t.Fatal("sign-out retained credentials or scheduled recovery")
	}
	if _, ok := svc.Session(); ok {
		t.Fatal("late refresh restored signed-out session")
	}
	store.Put("stored")
	mode = "revoked"
	a.refreshAccount(true)
	pump(func() bool { return !a.state.Account.Busy() })
	if a.state.Account.SignedIn() || a.state.Account.Credential != arion.CredentialAbsent || !a.accountMaintenance.next.IsZero() {
		t.Fatal("revocation did not stop recovery")
	}
	if _, ok := svc.Session(); ok {
		t.Fatal("revoked session retained")
	}
	a.state.Account.Phase = arion.AccountSigningOut
	a.accountMaintenance.next = time.Now()
	seq := a.accountGeneration()
	a.maintainAccount(time.Now())
	if a.accountGeneration() != seq {
		t.Fatal("maintenance ran during sign-out")
	}
	a.state.Account.Credential = arion.CredentialChecking
	a.beginAccount()
	a.finishCredential(seq, true, nil)
	if a.state.Account.Credential != arion.CredentialUnknown {
		t.Fatal("stale credential check restored account")
	}
}

func TestAccountMaintenanceGuards(t *testing.T) {
	for _, name := range []string{"offline", "scripted", "disabled", "checking", "stopped"} {
		t.Run(name, func(t *testing.T) {
			a := &App{}
			a.state.Account = arion.Account{Phase: arion.AccountSignedOut, Credential: arion.CredentialStored}
			a.accountMaintenance.next = time.Now()
			switch name {
			case "offline":
				a.opts.Offline = true
			case "scripted":
				a.harness = &harness{}
			case "disabled":
				a.state.Account.Disabled = true
			case "checking":
				a.state.Account.Credential = arion.CredentialChecking
			case "stopped":
				a.accountMaintenance.stopped = true
			}
			a.maintainAccount(time.Now().Add(time.Hour))
			if a.accountGeneration() != 0 || len(a.queue) != 0 || a.state.Account.Phase != arion.AccountSignedOut {
				t.Fatal("maintenance started a guarded account operation")
			}
		})
	}
}
