package oidc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"arion.local/sync/application"
	"arion.local/sync/domain"
	"arion.local/sync/infrastructure/memory"
)

const (
	testAudience    = "https://api.arion.astrafoundry.si"
	testClient      = "arion-sync"
	testEntitlement = "arion.sync.hosted"
)

type keyRing struct {
	mu   sync.Mutex
	keys []edKey
}

type edKey struct {
	kid string
	pub ed25519.PublicKey
}

func (r *keyRing) serve(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc := struct {
		Keys []map[string]string `json:"keys"`
	}{}
	for _, key := range r.keys {
		doc.Keys = append(doc.Keys, map[string]string{
			"kty": "OKP",
			"crv": "Ed25519",
			"use": "sig",
			"alg": "EdDSA",
			"kid": key.kid,
			"x":   base64.RawURLEncoding.EncodeToString(key.pub),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func (r *keyRing) set(keys ...edKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys = append([]edKey(nil), keys...)
}

func newProvider(t *testing.T, store *memory.Store, issuer, jwks string, now func() time.Time) *Provider {
	t.Helper()
	provider, err := New(Options{
		Issuer:      issuer,
		Audience:    testAudience,
		ClientID:    testClient,
		Entitlement: testEntitlement,
		JWKSURL:     jwks,
		Algorithms:  []string{"EdDSA"},
		Leeway:      30 * time.Second,
		CacheTTL:    time.Hour,
		Quota:       1 << 20,
		Directory:   store,
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func signAccess(t *testing.T, priv ed25519.PrivateKey, kid, iss, aud, sub, typ string, iat, exp, nbf time.Time, ent []string) string {
	t.Helper()
	claims := accessClaims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    iss,
		Subject:   sub,
		Audience:  jwt.ClaimStrings{aud},
		IssuedAt:  jwt.NewNumericDate(iat),
		ExpiresAt: jwt.NewNumericDate(exp),
		NotBefore: jwt.NewNumericDate(nbf),
	}, Ent: ent}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = kid
	token.Header["typ"] = typ
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestAuthorizeAndTenantMapping(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ring := &keyRing{}
	ring.set(edKey{kid: "k1", pub: pub})
	jwks := httptest.NewServer(http.HandlerFunc(ring.serve))
	t.Cleanup(jwks.Close)
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			_ = json.NewEncoder(w).Encode(map[string]string{"jwks_uri": jwks.URL})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(discovery.Close)
	other := httptest.NewServer(http.HandlerFunc(ring.serve))
	t.Cleanup(other.Close)

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := now
	store := memory.New()
	primary := newProvider(t, store, discovery.URL, "", func() time.Time { return clock })
	secondary := newProvider(t, store, other.URL, other.URL, func() time.Time { return clock })
	ctx := context.Background()
	token := signAccess(t, priv, "k1", discovery.URL, testAudience, "alice", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement})
	first, err := primary.Authorize(ctx, "Bearer "+token)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := primary.Authorize(ctx, "Bearer "+token)
	if err != nil || repeat.ID != first.ID {
		t.Fatalf("repeat tenant %q err %v", repeat.ID, err)
	}
	otherToken := signAccess(t, priv, "k1", other.URL, testAudience, "alice", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement})
	second, err := secondary.Authorize(ctx, "Bearer "+otherToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatal("another issuer reused the tenant")
	}

	cases := []struct {
		name string
		tok  string
		code string
	}{
		{"wrong iss", signAccess(t, priv, "k1", "https://evil.example", testAudience, "alice", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement}), domain.CodeUnauthorized},
		{"wrong aud", signAccess(t, priv, "k1", discovery.URL, "https://other.example", "bob", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement}), domain.CodeUnauthorized},
		{"expired", signAccess(t, priv, "k1", discovery.URL, testAudience, "bob", "at+jwt", now.Add(-2*time.Hour), now.Add(-time.Hour), now.Add(-2*time.Hour), []string{testEntitlement}), domain.CodeUnauthorized},
		{"nbf", signAccess(t, priv, "k1", discovery.URL, testAudience, "bob", "at+jwt", now, now.Add(2*time.Hour), now.Add(time.Hour), []string{testEntitlement}), domain.CodeUnauthorized},
		{"missing ent", signAccess(t, priv, "k1", discovery.URL, testAudience, "bob", "at+jwt", now, now.Add(time.Hour), now, nil), application.CodeEntitlementRequired},
		{"wrong typ", signAccess(t, priv, "k1", discovery.URL, testAudience, "bob", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement}), ""},
	}
	cases[5].tok = signAccess(t, priv, "k1", discovery.URL, testAudience, "bob", "JWT", now, now.Add(time.Hour), now, []string{testEntitlement})
	cases[5].code = domain.CodeUnauthorized
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := primary.Authorize(ctx, "Bearer "+tc.tok)
			if !coded(err, tc.code) {
				t.Fatalf("got %v", err)
			}
		})
	}

	pub2, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ring.set(edKey{kid: "k2", pub: pub2})
	rotated := signAccess(t, priv2, "k2", discovery.URL, testAudience, "cara", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement})
	if _, err := primary.Authorize(ctx, "Bearer "+rotated); err != nil {
		t.Fatal(err)
	}
	clock = now.Add(2 * time.Hour)
	ring.set(edKey{kid: "k2", pub: pub})
	stale := signAccess(t, priv2, "k2", discovery.URL, testAudience, "cara", "at+jwt", clock, clock.Add(time.Hour), clock, []string{testEntitlement})
	if _, err := primary.Authorize(ctx, "Bearer "+stale); err == nil {
		t.Fatal("stale cached key was accepted")
	}
	fresh := signAccess(t, priv, "k2", discovery.URL, testAudience, "cara", "at+jwt", clock, clock.Add(time.Hour), clock, []string{testEntitlement})
	if _, err := primary.Authorize(ctx, "Bearer "+fresh); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizeFrozenAndSessionFloor(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ring := &keyRing{}
	ring.set(edKey{kid: "k1", pub: pub})
	jwks := httptest.NewServer(http.HandlerFunc(ring.serve))
	t.Cleanup(jwks.Close)
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	provider := newProvider(t, store, "https://astrafoundry.si", jwks.URL, func() time.Time { return now })
	ctx := context.Background()
	token := signAccess(t, priv, "k1", "https://astrafoundry.si", testAudience, "alice", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement})
	if _, err := provider.Authorize(ctx, "Bearer "+token); err != nil {
		t.Fatal(err)
	}
	tx, err := store.BeginAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.SetFrozen(ctx, "https://astrafoundry.si", "alice", true, now); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = provider.Authorize(ctx, "Bearer "+token)
	if !coded(err, application.CodeTenantFrozen) {
		t.Fatalf("got %v", err)
	}
	tx, err = store.BeginAccount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.SetFrozen(ctx, "https://astrafoundry.si", "alice", false, now); err != nil {
		t.Fatal(err)
	}
	if err := tx.RevokeSessions(ctx, "https://astrafoundry.si", "alice", now); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = provider.Authorize(ctx, "Bearer "+token)
	if !coded(err, application.CodeSessionRevoked) {
		t.Fatalf("got %v", err)
	}
	later := signAccess(t, priv, "k1", "https://astrafoundry.si", testAudience, "alice", "at+jwt", now.Add(time.Second), now.Add(time.Hour), now.Add(time.Second), []string{testEntitlement})
	if _, err := provider.Authorize(ctx, "Bearer "+later); err != nil {
		t.Fatal(err)
	}
}

func TestNewRejectsHMAC(t *testing.T) {
	store := memory.New()
	_, err := New(Options{
		Issuer: "https://astrafoundry.si", Audience: testAudience, ClientID: testClient, Entitlement: testEntitlement,
		Algorithms: []string{"HS256"}, Leeway: time.Second, CacheTTL: time.Minute, Directory: store, Now: time.Now,
	})
	if err == nil {
		t.Fatal("expected hmac to be rejected")
	}
}

func TestSETKeyFailureAndEntitlementsPayload(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ring := &keyRing{}
	ring.set(edKey{kid: "k1", pub: pub})
	jwks := httptest.NewServer(http.HandlerFunc(ring.serve))
	t.Cleanup(jwks.Close)
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	store := memory.New()
	down := newProvider(t, store, "https://astrafoundry.si", broken.URL, func() time.Time { return now })
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","typ":"secevent+jwt","kid":"k1"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	if _, err := down.Verify(context.Background(), header+"."+body+".sig"); !setCode(err, "invalid_key") {
		t.Fatalf("jwks failure: %v", err)
	}

	provider := newProvider(t, store, "https://astrafoundry.si", jwks.URL, func() time.Time { return now })
	claims := jwt.MapClaims{
		"iss": "https://astrafoundry.si", "aud": testClient, "iat": now.Unix(), "jti": "jti-ent",
		"sub_id": map[string]string{"format": "iss_sub", "iss": "https://astrafoundry.si", "sub": "alice"},
		"events": map[string]any{application.EventEntitlementsChanged: map[string]any{"entitlements": []string{"arion.sync.hosted", "other"}}},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "k1"
	token.Header["typ"] = "secevent+jwt"
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	event, err := provider.Verify(context.Background(), signed)
	if err != nil {
		t.Fatal(err)
	}
	if len(event.Types) != 1 || event.Types[0] != application.EventEntitlementsChanged || event.Subject != "alice" {
		t.Fatalf("%+v", event)
	}
}

func TestNoneAlgorithmRejected(t *testing.T) {
	store := memory.New()
	provider := newProvider(t, store, "https://astrafoundry.si", "http://127.0.0.1/jwks", time.Now)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"at+jwt","kid":"k1"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://astrafoundry.si","sub":"alice","aud":"https://api.arion.astrafoundry.si","exp":9999999999,"iat":1,"ent":["arion.sync.hosted"]}`))
	_, err := provider.Authorize(context.Background(), "Bearer "+header+"."+payload+".")
	if !coded(err, domain.CodeUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func setCode(err error, code string) bool {
	set, ok := application.AsSET(err)
	return ok && set.Code == code
}

func coded(err error, code string) bool {
	if err == nil {
		return false
	}
	var de *domain.Error
	if ok := errorAs(err, &de); ok && de.Code == code {
		return true
	}
	var ae *application.Error
	return errorAs(err, &ae) && ae.Code == code
}

func errorAs(err error, target any) bool {
	switch dest := target.(type) {
	case **domain.Error:
		return asDomain(err, dest)
	case **application.Error:
		return asApplication(err, dest)
	default:
		return false
	}
}

func asDomain(err error, dest **domain.Error) bool {
	for err != nil {
		if de, ok := err.(*domain.Error); ok {
			*dest = de
			return true
		}
		unwrap, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrap.Unwrap()
	}
	return false
}

func TestFetchURLRejectsMetadataTargets(t *testing.T) {
	for _, raw := range []string{
		"http://169.254.169.254/latest/meta-data",
		"https://metadata.google.internal/computeMetadata/v1/",
		"file:///etc/passwd",
		"https://user:pass@example.com/jwks",
		"http://0.0.0.0/jwks",
	} {
		if err := ValidateFetchURL(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, raw := range []string{"http://127.0.0.1:9/jwks", "https://login.example.com/jwks", "http://issuer.internal:8080/jwks"} {
		if err := ValidateFetchURL(raw); err != nil {
			t.Errorf("rejected %s: %v", raw, err)
		}
	}
	if _, err := New(Options{
		Issuer: "https://login.example.com", Audience: testAudience, ClientID: testClient, Entitlement: testEntitlement,
		JWKSURL: "http://169.254.169.254/jwks", Algorithms: []string{"EdDSA"}, Leeway: time.Second, CacheTTL: time.Hour,
		Directory: memory.New(), Now: time.Now,
	}); err == nil {
		t.Fatal("link-local JWKS URL was configured")
	}
}

func TestDiscoveryDoesNotFetchLinkLocalJWKS(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	fetched := false
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			fetched = true
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"jwks_uri": "http://169.254.169.254/latest"})
	}))
	t.Cleanup(discovery.Close)
	provider, err := New(Options{
		Issuer: discovery.URL, Audience: testAudience, ClientID: testClient, Entitlement: testEntitlement,
		Algorithms: []string{"EdDSA"}, Leeway: 30 * time.Second, CacheTTL: time.Hour, Quota: 1 << 20,
		Directory: memory.New(), Now: time.Now,
		HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Hostname() == "169.254.169.254" {
				fetched = true
				return nil, errors.New("metadata fetched")
			}
			return http.DefaultTransport.RoundTrip(r)
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	token := signAccess(t, priv, "k1", discovery.URL, testAudience, "alice", "at+jwt", now, now.Add(time.Hour), now, []string{testEntitlement})
	if _, err := provider.Authorize(context.Background(), "Bearer "+token); err == nil {
		t.Fatal("authorized with an unfetched key")
	}
	if fetched {
		t.Fatal("relay requested the link-local JWKS URL")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func asApplication(err error, dest **application.Error) bool {
	for err != nil {
		if ae, ok := err.(*application.Error); ok {
			*dest = ae
			return true
		}
		unwrap, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrap.Unwrap()
	}
	return false
}
