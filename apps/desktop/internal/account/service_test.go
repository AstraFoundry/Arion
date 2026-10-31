package account

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	refreshCanary = "rt-canary-value"
	refreshNext   = "rt-canary-rotated"
	secretCanary  = "TOKEN_CANARY"
)

type memStore struct {
	mu         sync.Mutex
	token      string
	present    bool
	vocabulary string
}

func (m *memStore) Put(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = token
	m.present = true
	return nil
}

func (m *memStore) Get() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.present {
		return "", ErrNotFound
	}
	return m.token, nil
}

func (m *memStore) Delete() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = ""
	m.present = false
	return nil
}

func (m *memStore) current() (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.token, m.present
}

type issuer struct {
	t              *testing.T
	srv            *httptest.Server
	svc            *Service
	mu             sync.Mutex
	mode           string
	entered        chan struct{}
	release        chan struct{}
	auth           url.Values
	authHits       int
	tokenHits      int
	browserCalls   int
	followed       int
	revokeTokens   []string
	revokeStatus   int
	refreshMode    string
	lastRefresh    string
	catalogHits    int
	catalogStatus  string
	catalogID      string
	waitHits       int
	waitStatus     int
	waitBody       string
	waitAuth       string
	waitRequest    string
	withEnt        bool
	ent            any
	badID          bool
	omitRefresh    bool
	redirectToken  bool
	includeMethods bool
	methods        []string
	issuerOverride string
	authEndpoint   string
	now            time.Time
	// bareID issues ID tokens with only sub, as Better Auth 1.7 does, and
	// serves the profile from UserInfo.
	bareID       bool
	userInfoAuth []string
}

func newIssuer(t *testing.T) *issuer {
	t.Helper()
	s := &issuer{
		t:              t,
		entered:        make(chan struct{}, 1),
		release:        make(chan struct{}),
		includeMethods: true,
		methods:        []string{"S256"},
		catalogStatus:  "open",
		catalogID:      "wl_hosted_sync",
		waitStatus:     http.StatusCreated,
		waitBody:       `{"id":"entry-1","status":"pending"}`,
		withEnt:        true,
		now:            time.Unix(1_700_000_000, 0).UTC(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", s.discovery)
	mux.HandleFunc("/api/auth/oauth2/authorize", s.authorize)
	mux.HandleFunc("/api/auth/oauth2/token", s.token)
	mux.HandleFunc("/api/auth/oauth2/revoke", s.revoke)
	mux.HandleFunc("/api/auth/oauth2/userinfo", s.userinfo)
	mux.HandleFunc("/api/v1/catalog/products/"+CatalogProduct, s.catalog)
	mux.HandleFunc("/api/v1/me/waitlists/", s.waitlist)
	mux.HandleFunc("/stolen", s.stolen)
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

func (s *issuer) config() Config {
	client := s.srv.Client()
	client.Timeout = 5 * time.Second
	return Config{
		Issuer:        s.srv.URL + "/",
		HTTP:          client,
		SignInTimeout: 10 * time.Second,
		Resource:      DefaultResource,
	}
}

func (s *issuer) service(store RefreshTokenStore) *Service {
	s.t.Helper()
	svc, err := New(s.config(), Dependencies{Store: store, Browser: s.browser, Now: func() time.Time { return s.now }})
	if err != nil {
		s.t.Fatal(err)
	}
	s.svc = svc
	return svc
}

func (s *issuer) browser(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	q := u.Query()
	if q.Get("client_secret") != "" {
		s.t.Error("authorize URL contains client_secret")
	}
	s.mu.Lock()
	s.browserCalls++
	s.auth = q
	mode := s.mode
	s.mu.Unlock()
	redirect := q.Get("redirect_uri")
	state := q.Get("state")
	switch mode {
	case "silent", "hold":
		if mode == "hold" {
			s.entered <- struct{}{}
			<-s.release
		}
		return nil
	case "mismatch":
		return s.callback(redirect, url.Values{"code": {"auth-code"}, "error": {"access_denied"}, "error_description": {secretCanary}, "state": {"wrong-state"}})
	case "error":
		return s.callback(redirect, url.Values{"error": {"access_denied"}, "error_description": {secretCanary}, "state": {state}})
	case "late":
		s.svc.Cancel()
		return s.callback(redirect, url.Values{"code": {"auth-code"}, "state": {state}})
	default:
		return s.callback(redirect, url.Values{"code": {"auth-code"}, "state": {state}})
	}
}

func (s *issuer) callback(redirect string, q url.Values) error {
	u, err := url.Parse(redirect)
	if err != nil {
		return err
	}
	u.RawQuery = q.Encode()
	res, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return &Error{Code: CodeBrowser, Detail: "callback"}
	}
	if len(strings.TrimSpace(string(body))) != 0 || res.Header.Get("Cache-Control") != "no-store" {
		s.t.Errorf("callback status %d body %q cache %q", res.StatusCode, body, res.Header.Get("Cache-Control"))
	}
	return nil
}

func (s *issuer) discovery(w http.ResponseWriter, r *http.Request) {
	iss := s.srv.URL
	s.mu.Lock()
	if s.issuerOverride != "" {
		iss = s.issuerOverride
	}
	auth := s.srv.URL + "/api/auth/oauth2/authorize"
	if s.authEndpoint != "" {
		auth = s.authEndpoint
	}
	doc := map[string]any{
		"issuer":                 iss,
		"authorization_endpoint": auth,
		"token_endpoint":         s.srv.URL + "/api/auth/oauth2/token",
		"revocation_endpoint":    s.srv.URL + "/api/auth/oauth2/revoke",
	}
	if s.bareID {
		doc["userinfo_endpoint"] = s.srv.URL + "/api/auth/oauth2/userinfo"
	}
	if s.includeMethods {
		doc["code_challenge_methods_supported"] = s.methods
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func (s *issuer) authorize(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.authHits++
	s.mu.Unlock()
	http.NotFound(w, r)
}

func (s *issuer) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.t.Error(err)
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.tokenHits++
	challenge := s.auth.Get("code_challenge")
	redirect := s.auth.Get("redirect_uri")
	mode := s.refreshMode
	omit := s.omitRefresh
	redirectToken := s.redirectToken
	s.lastRefresh = r.Form.Get("refresh_token")
	s.mu.Unlock()
	if r.Form.Get("client_secret") != "" {
		s.t.Error("token request contains client_secret")
	}
	if r.Form.Get("client_id") != DefaultClientID || r.Form.Get("resource") != DefaultResource {
		s.t.Errorf("client %q resource %q", r.Form.Get("client_id"), r.Form.Get("resource"))
	}
	if redirectToken {
		w.Header().Set("Location", s.srv.URL+"/stolen")
		w.WriteHeader(http.StatusFound)
		return
	}
	grant := r.Form.Get("grant_type")
	switch grant {
	case "authorization_code":
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
			s.t.Errorf("pkce challenge mismatch")
		}
		if r.Form.Get("code") != "auth-code" || r.Form.Get("redirect_uri") != redirect {
			s.t.Errorf("code %q redirect %q", r.Form.Get("code"), r.Form.Get("redirect_uri"))
		}
		refresh := refreshCanary
		if omit {
			refresh = ""
		}
		s.writeTokens(w, refresh)
	case "refresh_token":
		if mode == "invalid" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": secretCanary})
			return
		}
		refresh := ""
		if mode != "keep" {
			refresh = refreshNext
		}
		s.writeTokens(w, refresh)
	default:
		s.t.Errorf("grant %q", grant)
		http.Error(w, "bad", http.StatusBadRequest)
	}
}

func (s *issuer) writeTokens(w http.ResponseWriter, refresh string) {
	s.mu.Lock()
	subject := "user-1"
	claims := map[string]any{"sub": subject}
	if s.withEnt {
		ent := s.ent
		if ent == nil {
			ent = []string{HostedSyncEntitlement}
		}
		claims["ent"] = ent
	}
	badID, bareID := s.badID, s.bareID
	s.mu.Unlock()
	out := map[string]any{
		"access_token": unsignedJWT(claims),
		"token_type":   "Bearer",
		"expires_in":   60,
	}
	if refresh != "" {
		out["refresh_token"] = refresh
	}
	switch {
	case badID:
		out["id_token"] = "not-a-jwt"
	case bareID:
		out["id_token"] = unsignedJWT(map[string]any{"sub": subject})
	default:
		out["id_token"] = unsignedJWT(map[string]any{
			"sub": subject, "name": "Ada", "email": "ada@example.com",
			"picture": "https://example.com/a.png", "locale": "en",
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *issuer) userinfo(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.userInfoAuth = append(s.userInfoAuth, r.Header.Get("Authorization"))
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"sub": "user-1", "name": "Grace", "picture": "https://example.com/g.png"})
}

func (s *issuer) revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.t.Error(err)
		return
	}
	s.mu.Lock()
	s.revokeTokens = append(s.revokeTokens, r.Form.Get("token"))
	status := s.revokeStatus
	s.mu.Unlock()
	if r.Form.Get("client_secret") != "" || r.Form.Get("token_type_hint") != "refresh_token" || r.Form.Get("client_id") != DefaultClientID {
		s.t.Errorf("revoke form client %q hint %q", r.Form.Get("client_id"), r.Form.Get("token_type_hint"))
	}
	if status == 0 {
		status = http.StatusOK
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "server_error", "error_description": secretCanary + " " + r.Form.Get("token")})
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *issuer) catalog(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "" {
		s.t.Error("catalog request carried a bearer token")
	}
	s.mu.Lock()
	s.catalogHits++
	id, status := s.catalogID, s.catalogStatus
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"waitlists": []map[string]string{{"id": id, "key": HostedSyncWaitlistKey, "status": status}},
	})
}

func (s *issuer) waitlist(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.waitHits++
	s.waitAuth = r.Header.Get("Authorization")
	s.waitRequest = string(body)
	status, payload := s.waitStatus, s.waitBody
	s.mu.Unlock()
	if r.Method != http.MethodPost {
		s.t.Errorf("waitlist method %s", r.Method)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(payload))
}

func (s *issuer) stolen(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.followed++
	s.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *issuer) authValues() url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auth
}

func unsignedJWT(claims any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		panic(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".not-a-signature"
}

func noSecret(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	text := err.Error()
	if strings.Contains(text, refreshCanary) || strings.Contains(text, refreshNext) || strings.Contains(text, secretCanary) {
		t.Fatalf("error leaked secret material: %s", text)
	}
}

func TestNewDoesNotUseTheNetwork(t *testing.T) {
	hit := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit++ }))
	defer srv.Close()
	svc, err := New(Config{Issuer: srv.URL, HTTP: srv.Client()}, Dependencies{
		Store:   &memStore{},
		Browser: func(string) error { t.Fatal("browser opened"); return nil },
	})
	if err != nil || svc == nil {
		t.Fatal(err)
	}
	if hit != 0 {
		t.Fatalf("new made %d requests", hit)
	}
}

func TestSignInHappyPath(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{vocabulary: "local-key"}
	svc := iss.service(store)
	session, err := svc.SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	q := iss.authValues()
	if q.Get("client_id") != DefaultClientID || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize query = %v", q)
	}
	if q.Get("resource") != DefaultResource || q.Get("scope") != strings.Join(DefaultScopes(), " ") {
		t.Fatalf("resource %q scope %q", q.Get("resource"), q.Get("scope"))
	}
	redirect, err := url.Parse(q.Get("redirect_uri"))
	if err != nil || redirect.Hostname() != "127.0.0.1" || redirect.Path != callbackPath || redirect.Port() == "" {
		t.Fatalf("redirect = %q", q.Get("redirect_uri"))
	}
	if iss.authHits != 0 || iss.tokenHits != 1 {
		t.Fatalf("authorize hits %d token hits %d", iss.authHits, iss.tokenHits)
	}
	if !session.HasHostedSync() || session.Subject != "user-1" || session.Profile.Name != "Ada" || session.Profile.Email != "ada@example.com" {
		t.Fatalf("session = %#v", session)
	}
	if session.TokenType != "Bearer" || !session.Expiry.Equal(iss.now.Add(60*time.Second)) {
		t.Fatalf("type %s expiry %s", session.TokenType, session.Expiry)
	}
	if session.AccessToken == "" || strings.Contains(session.AccessToken, refreshCanary) {
		t.Fatal("access token missing or contains the refresh token")
	}
	got, ok := store.current()
	if !ok || got != refreshCanary {
		t.Fatalf("stored %#v %v", got, ok)
	}
	if store.vocabulary != "local-key" {
		t.Fatal("sign-in touched the vocabulary key")
	}
	loaded, ok := svc.Session()
	if !ok || loaded.AccessToken != session.AccessToken {
		t.Fatal("session was not published")
	}
}

func TestSignInClosesLoopback(t *testing.T) {
	iss := newIssuer(t)
	svc := iss.service(&memStore{})
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	redirect, err := url.Parse(iss.authValues().Get("redirect_uri"))
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialTimeout("tcp", redirect.Host, 200*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		t.Fatal("loopback still accepting")
	}
}

func TestSignInStateMismatch(t *testing.T) {
	iss := newIssuer(t)
	iss.mode = "mismatch"
	store := &memStore{}
	svc := iss.service(store)
	// A wrong state must not finish the attempt. The listener keeps waiting
	// so a later, matching redirect can still be accepted.
	svc.cfg.SignInTimeout = 200 * time.Millisecond
	_, err := svc.SignIn(context.Background())
	if !errors.Is(err, ErrSignInTimeout) {
		t.Fatalf("err = %v", err)
	}
	noSecret(t, err)
	if iss.tokenHits != 0 {
		t.Fatalf("exchanged on mismatch")
	}
	if _, ok := store.current(); ok {
		t.Fatal("stored a token")
	}
}

func TestSignInAuthorizationError(t *testing.T) {
	iss := newIssuer(t)
	iss.mode = "error"
	_, err := iss.service(&memStore{}).SignIn(context.Background())
	if CodeOf(err) != CodeAuthorization || DetailOf(err) != "access_denied" {
		t.Fatalf("err = %v", err)
	}
	noSecret(t, err)
	if iss.tokenHits != 0 {
		t.Fatal("exchanged after an error callback")
	}
}

func TestSignInTimeout(t *testing.T) {
	iss := newIssuer(t)
	iss.mode = "silent"
	svc, err := New(iss.config(), Dependencies{Store: &memStore{}, Browser: iss.browser, Now: func() time.Time { return iss.now }})
	if err != nil {
		t.Fatal(err)
	}
	svc.cfg.SignInTimeout = 200 * time.Millisecond
	start := time.Now()
	_, err = svc.SignIn(context.Background())
	if !errors.Is(err, ErrSignInTimeout) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("timeout waited too long")
	}
	if iss.tokenHits != 0 {
		t.Fatal("exchanged after timeout")
	}
}

func TestSignInCancelBeforeExchange(t *testing.T) {
	iss := newIssuer(t)
	iss.mode = "late"
	store := &memStore{}
	_, err := iss.service(store).SignIn(context.Background())
	if !errors.Is(err, ErrSignInCancelled) {
		t.Fatalf("err = %v", err)
	}
	if iss.tokenHits != 0 {
		t.Fatal("exchanged after cancel")
	}
	if _, ok := store.current(); ok {
		t.Fatal("stored a token after cancel")
	}
}

func TestCancelAfterTokenIssueDropsRefreshToken(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{vocabulary: "local-key"}
	svc := iss.service(store)
	svc.testAfterPut = func() { svc.Cancel() }
	_, err := svc.SignIn(context.Background())
	if !errors.Is(err, ErrSignInCancelled) {
		t.Fatalf("err = %v", err)
	}
	noSecret(t, err)
	if _, ok := store.current(); ok {
		t.Fatal("refresh token kept after cancel")
	}
	if store.vocabulary != "local-key" {
		t.Fatal("cancel deleted the vocabulary key")
	}
	if _, ok := svc.Session(); ok {
		t.Fatal("session published after cancel")
	}
	iss.mu.Lock()
	revoked := append([]string(nil), iss.revokeTokens...)
	iss.mu.Unlock()
	if len(revoked) != 1 || revoked[0] != refreshCanary {
		t.Fatalf("revoked %#v", revoked)
	}
}

func TestSignInSingleFlight(t *testing.T) {
	iss := newIssuer(t)
	iss.mode = "hold"
	svc := iss.service(&memStore{})
	done := make(chan error, 1)
	go func() {
		_, err := svc.SignIn(context.Background())
		done <- err
	}()
	select {
	case <-iss.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("sign-in did not reach the browser")
	}
	if _, err := svc.SignIn(context.Background()); !errors.Is(err, ErrSignInInProgress) {
		t.Fatalf("second sign-in = %v", err)
	}
	if _, err := svc.Refresh(context.Background()); !errors.Is(err, ErrSignInInProgress) {
		t.Fatalf("refresh = %v", err)
	}
	if _, err := svc.JoinHostedSyncWaitlist(context.Background()); !errors.Is(err, ErrSignInInProgress) {
		t.Fatalf("join = %v", err)
	}
	svc.Cancel()
	close(iss.release)
	select {
	case err := <-done:
		if !errors.Is(err, ErrSignInCancelled) {
			t.Fatalf("first = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first sign-in did not finish")
	}
	if iss.browserCalls != 1 || iss.tokenHits != 0 {
		t.Fatalf("browser %d token %d", iss.browserCalls, iss.tokenHits)
	}
}

func TestRefreshRotates(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{}
	svc := iss.service(store)
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	iss.refreshMode = "rotate"
	session, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	iss.mu.Lock()
	sent := iss.lastRefresh
	iss.mu.Unlock()
	if sent != refreshCanary {
		t.Fatalf("sent refresh %q", sent)
	}
	got, ok := store.current()
	if !ok || got != refreshNext {
		t.Fatalf("stored %#v", got)
	}
	if !session.HasHostedSync() || session.AccessToken == "" {
		t.Fatalf("session %#v", session)
	}
	if iss.browserCalls != 1 {
		t.Fatal("refresh opened the browser")
	}
}

func TestRefreshKeepsTokenWhenOmitted(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{}
	svc := iss.service(store)
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	iss.refreshMode = "keep"
	if _, err := svc.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, ok := store.current()
	if !ok || got != refreshCanary {
		t.Fatalf("stored %#v", got)
	}
}

func TestRefreshInvalidGrantDropsRefreshToken(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{}
	svc := iss.service(store)
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	iss.refreshMode = "invalid"
	_, err := svc.Refresh(context.Background())
	if CodeOf(err) != CodeRefresh || DetailOf(err) != "invalid_grant" {
		t.Fatalf("err = %v", err)
	}
	noSecret(t, err)
	if _, ok := store.current(); ok {
		t.Fatal("refresh token kept after invalid_grant")
	}
	if _, ok := svc.Session(); ok {
		t.Fatal("session kept after invalid_grant")
	}
}

func TestSignOutRevokesAndDeletesRefreshToken(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{vocabulary: "local-key"}
	svc := iss.service(store)
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := svc.SignOut(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Revoked || !result.RevokeAttempted || !result.RefreshTokenDeleted || result.RevokeError != nil {
		t.Fatalf("result = %#v", result)
	}
	if _, ok := store.current(); ok {
		t.Fatal("refresh token remains")
	}
	if store.vocabulary != "local-key" {
		t.Fatal("sign-out deleted the vocabulary key")
	}
	if _, ok := svc.Session(); ok {
		t.Fatal("session remains")
	}
	iss.mu.Lock()
	revoked := append([]string(nil), iss.revokeTokens...)
	iss.mu.Unlock()
	if len(revoked) != 1 || revoked[0] != refreshCanary {
		t.Fatalf("revoked %#v", revoked)
	}
}

func TestSignOutReportsRevokeFailure(t *testing.T) {
	iss := newIssuer(t)
	iss.revokeStatus = http.StatusServiceUnavailable
	store := &memStore{vocabulary: "local-key"}
	svc := iss.service(store)
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := svc.SignOut(context.Background(), nil)
	if err == nil || result.RevokeError == nil || result.Revoked {
		t.Fatalf("result %#v err %v", result, err)
	}
	if CodeOf(err) != CodeRevoke || DetailOf(err) != "server_error" || DetailOf(result.RevokeError) != "server_error" {
		t.Fatalf("err = %v", err)
	}
	noSecret(t, err)
	if !result.RefreshTokenDeleted {
		t.Fatal("local item kept after a failed revoke")
	}
	if _, ok := store.current(); ok || store.vocabulary != "local-key" {
		t.Fatal("store lost the vocabulary key or kept the refresh token")
	}
	if _, ok := svc.Session(); ok {
		t.Fatal("session remains")
	}
}

func TestSignOutDoesNotDeleteLocalKeys(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{vocabulary: "vocabulary-key-material"}
	svc := iss.service(store)
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SignOut(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if store.vocabulary != "vocabulary-key-material" {
		t.Fatal("vocabulary material changed")
	}
	if _, ok := store.current(); ok {
		t.Fatal("refresh token remains")
	}
}

func TestSignOutDuringCommitReportsRevoke(t *testing.T) {
	iss := newIssuer(t)
	iss.revokeStatus = http.StatusServiceUnavailable
	store := &memStore{}
	svc := iss.service(store)
	out := make(chan SignOutResult, 1)
	errs := make(chan error, 1)
	svc.testAfterPut = func() {
		go func() {
			result, err := svc.SignOut(context.Background(), nil)
			out <- result
			errs <- err
		}()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			svc.mu.Lock()
			reason := svc.stopReason
			svc.mu.Unlock()
			if reason == stopSignOut {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("sign-out did not stop the in-flight commit")
	}
	_, err := svc.SignIn(context.Background())
	if !errors.Is(err, ErrSignInCancelled) {
		t.Fatalf("sign-in = %v", err)
	}
	var result SignOutResult
	select {
	case result = <-out:
		err = <-errs
	case <-time.After(5 * time.Second):
		t.Fatal("sign-out did not finish")
	}
	if err == nil || !result.RevokeAttempted || result.Revoked || !result.RefreshTokenDeleted {
		t.Fatalf("result %#v err %v", result, err)
	}
	noSecret(t, err)
	if _, ok := store.current(); ok {
		t.Fatal("refresh token remains")
	}
}

func TestJoinWaitlistFromCatalog(t *testing.T) {
	iss := newIssuer(t)
	iss.withEnt = false
	svc := iss.service(&memStore{})
	session, err := svc.SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.HasHostedSync() {
		t.Fatal("fixture granted hosted sync")
	}
	result, err := svc.JoinHostedSyncWaitlist(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != WaitlistJoinedPending || result.EntryID != "entry-1" || result.HTTPStatus != http.StatusCreated {
		t.Fatalf("result = %#v", result)
	}
	if iss.catalogHits != 1 || iss.waitHits != 1 {
		t.Fatalf("catalog %d wait %d", iss.catalogHits, iss.waitHits)
	}
	if iss.waitAuth != "Bearer "+session.AccessToken || iss.waitRequest != `{"answers":{}}` {
		t.Fatalf("auth %q body %q", iss.waitAuth, iss.waitRequest)
	}
}

func TestJoinWaitlistClosed(t *testing.T) {
	iss := newIssuer(t)
	iss.withEnt = false
	iss.catalogStatus = "paused"
	svc := iss.service(&memStore{})
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := svc.JoinHostedSyncWaitlist(context.Background())
	if err != nil || result.Status != WaitlistClosed {
		t.Fatalf("result %#v err %v", result, err)
	}
	if iss.waitHits != 0 {
		t.Fatal("posted to a closed waitlist")
	}
}

func TestJoinWaitlistConflict(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		withEnt  bool
		want     WaitlistStatus
		wantCode string
	}{
		{name: "explicit entitled", body: `{"code":"already_entitled"}`, want: WaitlistAlreadyEntitled, wantCode: "already_entitled"},
		{name: "explicit pending", body: `{"code":"already_pending"}`, withEnt: true, want: WaitlistJoinedPending, wantCode: "already_pending"},
		{name: "empty while entitled", body: `{}`, withEnt: true, want: WaitlistAlreadyEntitled},
		{name: "empty while pending", body: `{}`, want: WaitlistJoinedPending},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			iss := newIssuer(t)
			iss.withEnt = tc.withEnt
			iss.waitStatus = http.StatusConflict
			iss.waitBody = tc.body
			svc := iss.service(&memStore{})
			if _, err := svc.SignIn(context.Background()); err != nil {
				t.Fatal(err)
			}
			result, err := svc.JoinHostedSyncWaitlist(context.Background())
			if err != nil || result.Status != tc.want || result.Code != tc.wantCode {
				t.Fatalf("result %#v err %v", result, err)
			}
		})
	}
}

func TestJoinWaitlistConfiguredIDSkipsCatalog(t *testing.T) {
	iss := newIssuer(t)
	iss.withEnt = false
	cfg := iss.config()
	cfg.WaitlistID = "wl_override"
	svc, err := New(cfg, Dependencies{Store: &memStore{}, Browser: iss.browser, Now: func() time.Time { return iss.now }})
	if err != nil {
		t.Fatal(err)
	}
	iss.svc = svc
	session, err := svc.SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	iss.catalogHits = 0
	result, err := svc.JoinHostedSyncWaitlist(context.Background())
	if err != nil || result.Status != WaitlistJoinedPending {
		t.Fatalf("result %#v err %v", result, err)
	}
	if iss.catalogHits != 0 {
		t.Fatal("catalog was queried despite a configured id")
	}
	if iss.waitAuth != "Bearer "+session.AccessToken {
		t.Fatalf("auth %q", iss.waitAuth)
	}
}

func TestJoinNotSignedIn(t *testing.T) {
	iss := newIssuer(t)
	_, err := iss.service(&memStore{}).JoinHostedSyncWaitlist(context.Background())
	if !errors.Is(err, ErrNotSignedIn) {
		t.Fatalf("err = %v", err)
	}
	if iss.catalogHits != 0 || iss.waitHits != 0 {
		t.Fatal("join dialed before sign-in")
	}
}

func TestSignInRejectsRedirect(t *testing.T) {
	iss := newIssuer(t)
	iss.redirectToken = true
	_, err := iss.service(&memStore{}).SignIn(context.Background())
	if CodeOf(err) != CodeToken || DetailOf(err) != "http_302" {
		t.Fatalf("err = %v", err)
	}
	if iss.followed != 0 {
		t.Fatal("followed a token redirect")
	}
}

func TestDiscoveryRejectsMissingS256(t *testing.T) {
	iss := newIssuer(t)
	iss.methods = []string{"plain"}
	_, err := iss.service(&memStore{}).SignIn(context.Background())
	if CodeOf(err) != CodeDiscovery || DetailOf(err) != "s256_required" {
		t.Fatalf("err = %v", err)
	}
	if iss.browserCalls != 0 {
		t.Fatal("opened the browser without S256")
	}
}

func TestDiscoveryIssuerMismatch(t *testing.T) {
	iss := newIssuer(t)
	iss.issuerOverride = "https://evil.example"
	_, err := iss.service(&memStore{}).SignIn(context.Background())
	if DetailOf(err) != "issuer_mismatch" {
		t.Fatalf("err = %v", err)
	}
	if iss.browserCalls != 0 {
		t.Fatal("opened the browser for a mismatched issuer")
	}
}

func TestFileAuthorizationEndpointNotOpened(t *testing.T) {
	iss := newIssuer(t)
	iss.authEndpoint = "file:///tmp/authorize"
	_, err := iss.service(&memStore{}).SignIn(context.Background())
	if CodeOf(err) != CodeDiscovery {
		t.Fatalf("err = %v", err)
	}
	if iss.browserCalls != 0 {
		t.Fatal("opened a file authorization endpoint")
	}
}

func TestBadIDTokenDoesNotFailSignIn(t *testing.T) {
	iss := newIssuer(t)
	iss.badID = true
	session, err := iss.service(&memStore{}).SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.Subject != "user-1" || session.Profile.Name != "" {
		t.Fatalf("session %#v", session)
	}
}

func TestRejectsBadWaitlistID(t *testing.T) {
	iss := newIssuer(t)
	cfg := iss.config()
	cfg.WaitlistID = ".."
	if _, err := New(cfg, Dependencies{Store: &memStore{}, Browser: iss.browser}); err == nil {
		t.Fatal("accepted ..")
	}
}

// failingStore fails every read with err.
type failingStore struct{ err error }

func (f failingStore) Put(string) error     { return f.err }
func (f failingStore) Get() (string, error) { return "", f.err }
func (f failingStore) Delete() error        { return f.err }

func TestStoredReadsOnlyTheCredentialStore(t *testing.T) {
	hit := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit++ }))
	defer srv.Close()
	for _, tc := range []struct {
		name   string
		store  RefreshTokenStore
		stored bool
		code   Code
		detail string
	}{
		{"empty", &memStore{}, false, "", ""},
		{"token", &memStore{token: refreshCanary, present: true}, true, "", ""},
		{"read error", failingStore{errors.New("keychain locked")}, false, CodeStore, "read"},
		{"unsupported", failingStore{ErrUnsupported}, false, CodeStore, "unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := New(Config{Issuer: srv.URL, HTTP: srv.Client()}, Dependencies{
				Store:   tc.store,
				Browser: func(string) error { t.Fatal("browser opened"); return nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			stored, err := svc.Stored()
			if stored != tc.stored || CodeOf(err) != tc.code || DetailOf(err) != tc.detail {
				t.Fatalf("Stored() = %v, %v", stored, err)
			}
			if err != nil && strings.Contains(err.Error(), refreshCanary) {
				t.Fatal("error leaked the refresh token")
			}
			if _, ok := svc.Session(); ok {
				t.Fatal("reading the store created a session")
			}
		})
	}
	if hit != 0 {
		t.Fatalf("Stored made %d requests", hit)
	}
}

// A refresh token left from an earlier run is revoked and deleted by
// SignOut without a session in memory.
func TestSignOutRevokesStoredTokenWithoutSession(t *testing.T) {
	iss := newIssuer(t)
	store := &memStore{token: refreshCanary, present: true}
	svc := iss.service(store)
	result, err := svc.SignOut(context.Background(), nil)
	if err != nil || !result.RevokeAttempted || !result.Revoked || !result.RefreshTokenDeleted {
		t.Fatalf("SignOut = %+v, %v", result, err)
	}
	if stored, err := svc.Stored(); stored || err != nil {
		t.Fatalf("Stored after sign-out = %v, %v", stored, err)
	}
}

// Better Auth 1.7 keeps profile claims out of the ID token; sign-in and
// refresh read them from UserInfo with the access token.
func TestProfileFromUserInfo(t *testing.T) {
	iss := newIssuer(t)
	iss.bareID = true
	svc := iss.service(&memStore{})
	session, err := svc.SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.Profile.Name != "Grace" || session.Profile.Picture != "https://example.com/g.png" || session.Profile.Email != "" {
		t.Fatalf("profile = %#v", session.Profile)
	}
	iss.refreshMode = "rotate"
	session, err = svc.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.Profile.Name != "Grace" {
		t.Fatalf("refreshed profile = %#v", session.Profile)
	}
	iss.mu.Lock()
	defer iss.mu.Unlock()
	if len(iss.userInfoAuth) != 2 || !strings.HasPrefix(iss.userInfoAuth[0], "Bearer ") {
		t.Fatalf("userinfo auth = %q", iss.userInfoAuth)
	}
}

type accountRoundTripFunc func(*http.Request) (*http.Response, error)

func (f accountRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRefreshPersistsRotationBeforeUserInfo(t *testing.T) {
	iss := newIssuer(t)
	iss.bareID = true
	store := &memStore{}
	svc := iss.service(store)
	before, err := svc.SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	iss.refreshMode = "rotate"
	original := svc.cfg.HTTP.Transport
	if original == nil {
		original = http.DefaultTransport
	}
	svc.cfg.HTTP.Transport = accountRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/auth/oauth2/userinfo" {
			if token, ok := store.current(); !ok || token != refreshNext {
				t.Error("rotation was not persisted before UserInfo")
			}
			return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		}
		return original.RoundTrip(r)
	})
	after, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after.Profile != before.Profile {
		t.Fatalf("temporary UserInfo failure lost profile: %#v", after.Profile)
	}
	if current, ok := svc.Session(); !ok || current.Profile != before.Profile {
		t.Fatal("in-memory profile was not retained")
	}
}
