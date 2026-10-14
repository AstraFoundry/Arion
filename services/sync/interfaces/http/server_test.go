package httpapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"arion.local/sync/application"
	"arion.local/sync/domain"
	"arion.local/sync/infrastructure/logging"
	"arion.local/sync/infrastructure/memory"
	"arion.local/sync/infrastructure/noop"
	"arion.local/sync/infrastructure/oidc"
)

func TestNoopRelay(t *testing.T) {
	env := newNoop(t, 1<<20, time.Hour, 100)
	if rec := env.call(http.MethodPost, "/v1/security-events", "", nil, "", []byte("x")); rec.Code != http.StatusNotFound {
		t.Fatalf("events endpoint %d", rec.Code)
	}
	dev := env.enroll("device-a", "group-a", nil, "")
	if env.store.Counts().Tenants != 0 || env.store.Counts().Links != 0 || env.store.Counts().Events != 0 {
		t.Fatalf("noop stored account rows: %+v", env.store.Counts())
	}
	body := env.pushBody("op-1", "rec-1", 1, 0, nil, []byte("CIPHERTEXT-MARKER"))
	rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("push %d %s", rec.Code, rec.Body.Bytes())
	}
	var pushed struct {
		Results []struct {
			Outcome        string `json:"outcome"`
			ServerRevision int64  `json:"server_revision"`
		} `json:"results"`
		Cursor string `json:"cursor"`
	}
	decode(t, rec.Body.Bytes(), &pushed)
	if pushed.Results[0].Outcome != "applied" {
		t.Fatalf("%+v", pushed)
	}
	cursor0, err := domain.EncodeCursor(0)
	if err != nil {
		t.Fatal(err)
	}
	pulled := env.pull(dev, cursor0)
	if len(pulled.Envelopes) != 1 || pulled.Envelopes[0].Ciphertext != base64.RawURLEncoding.EncodeToString([]byte("CIPHERTEXT-MARKER")) {
		t.Fatalf("%+v", pulled)
	}
	if strings.Contains(env.logs.String(), "CIPHERTEXT-MARKER") || strings.Contains(env.logs.String(), base64.RawURLEncoding.EncodeToString([]byte("CIPHERTEXT-MARKER"))) {
		t.Fatal("log contains ciphertext")
	}

	again := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, body)
	decode(t, again.Body.Bytes(), &pushed)
	if again.Code != http.StatusOK || pushed.Results[0].Outcome != "duplicate" {
		t.Fatalf("duplicate %d %+v", again.Code, pushed)
	}
	replay := env.pushBody("op-2", "rec-1", 1, 0, nil, []byte("CIPHERTEXT-MARKER"))
	if rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, replay); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), domain.CodeReplay) {
		t.Fatalf("replay %d %s", rec.Code, rec.Body.Bytes())
	}
	mismatch := env.pushBody("op-1", "rec-1", 1, 0, nil, []byte("other-ciphertext"))
	if rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, mismatch); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), domain.CodeOperationMismatch) {
		t.Fatalf("mismatch %d %s", rec.Code, rec.Body.Bytes())
	}

	tampered := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, body)
	tampered = env.raw(http.MethodPost, "/v1/sync/push", "", dev.id, env.at, "not-a-signature", body)
	if tampered.Code != http.StatusUnauthorized {
		t.Fatalf("tamper %d", tampered.Code)
	}

	other := env.enrollSponsored("device-b", "group-a", dev)
	sibling := env.pushBody("op-b", "rec-1", 1, 0, nil, []byte("sibling-ciphertext"))
	rec = env.call(http.MethodPost, "/v1/sync/push", "", other.priv, other.id, sibling)
	decode(t, rec.Body.Bytes(), &pushed)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"sibling":true`) {
		t.Fatalf("sibling %d %s", rec.Code, rec.Body.Bytes())
	}
	live := env.pushBodyKind("op-live", "rec-2", 1, 0, nil, []byte("live-ciphertext"), "write")
	if rec = env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, live); rec.Code != http.StatusOK {
		t.Fatalf("write %d %s", rec.Code, rec.Body.Bytes())
	}
	del := env.pushBodyKind("op-tomb", "rec-2", 1, 0, nil, []byte("tomb-ciphertext"), "tombstone")
	rec = env.call(http.MethodPost, "/v1/sync/push", "", other.priv, other.id, del)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"suppressed":true`) {
		t.Fatalf("suppress %d %s", rec.Code, rec.Body.Bytes())
	}
	page := env.pull(other, cursor0)
	var sawTomb bool
	for _, envl := range page.Envelopes {
		if envl.Kind == "tombstone" && envl.RecordID == "rec-2" {
			sawTomb = true
		}
	}
	if !sawTomb {
		t.Fatal("second device cannot see the tombstone")
	}
	gc := env.pushBody("op-gc", "rec-gc", 1, 0, nil, []byte("gc-ciphertext!!"))
	rec = env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, gc)
	decode(t, rec.Body.Bytes(), &pushed)
	gcRev := pushed.Results[0].ServerRevision
	gone := env.pushBodyKind("op-gc-del", "rec-gc", 1, gcRev, []int64{gcRev}, []byte("gc-tomb-cipher!!"), "tombstone")
	rec = env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, gone)
	decode(t, rec.Body.Bytes(), &pushed)
	if page = env.pull(other, cursor0); !containsRecord(page, "rec-gc") {
		t.Fatal("tombstone was collected before the other device acked")
	}
	ackBody, _ := json.Marshal(map[string][]int64{"revisions": {pushed.Results[0].ServerRevision}})
	if rec = env.call(http.MethodPost, "/v1/sync/ack", "", other.priv, other.id, ackBody); rec.Code != http.StatusOK {
		t.Fatalf("ack %d %s", rec.Code, rec.Body.Bytes())
	}
	if page = env.pull(other, cursor0); containsRecord(page, "rec-gc") {
		t.Fatal("tombstone survived every ack")
	}

	revBody := []byte("{}")
	rec = env.call(http.MethodPost, "/v1/sync/devices/"+other.id+"/revoke", "", dev.priv, dev.id, revBody)
	var revoked struct {
		KeyEpoch uint64 `json:"key_epoch"`
	}
	decode(t, rec.Body.Bytes(), &revoked)
	if rec.Code != http.StatusOK || revoked.KeyEpoch != 2 {
		t.Fatalf("revoke %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", "", other.priv, other.id, env.pushBody("op-x", "rec-3", 2, 0, nil, []byte("nope-ciphertext"))); rec.Code != http.StatusForbidden {
		t.Fatalf("revoked device %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, env.pushBody("op-stale", "rec-3", 1, 0, nil, []byte("stale-ciphertext"))); rec.Code != http.StatusConflict {
		t.Fatalf("stale epoch %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, env.pushBody("op-new", "rec-3", 2, 0, nil, []byte("fresh-ciphertext"))); rec.Code != http.StatusOK {
		t.Fatalf("new epoch %d %s", rec.Code, rec.Body.Bytes())
	}
	againRev := env.call(http.MethodPost, "/v1/sync/devices/"+other.id+"/revoke", "", dev.priv, dev.id, revBody)
	decode(t, againRev.Body.Bytes(), &revoked)
	if revoked.KeyEpoch != 2 {
		t.Fatalf("epoch rotated twice: %d", revoked.KeyEpoch)
	}

	if rec := env.call(http.MethodPost, "/v1/sync/devices", "", dev.priv, "device-c", env.enrollBody("device-c", "group-b", dev.pub, "recover-me", "", nil)); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), domain.CodeRecoveryUnsupported) {
		t.Fatalf("recovery %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCursorExpiryQuotaAndRate(t *testing.T) {
	env := newNoop(t, 1<<20, time.Millisecond, 100)
	dev := env.enroll("device-a", "group-a", nil, "")
	rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, env.pushBody("op-1", "rec-1", 1, 0, nil, []byte("CIPHERTEXT-MARKER")))
	var pushed struct {
		Cursor string `json:"cursor"`
	}
	decode(t, rec.Body.Bytes(), &pushed)
	env.at = env.at.Add(time.Hour)
	rec = env.call(http.MethodGet, "/v1/sync/pull?cursor="+pushed.Cursor, "", dev.priv, dev.id, nil)
	if rec.Code != http.StatusGone || !strings.Contains(rec.Body.String(), `"/v1/sync/snapshot"`) {
		t.Fatalf("expiry %d %s", rec.Code, rec.Body.Bytes())
	}
	rec = env.call(http.MethodGet, "/v1/sync/snapshot", "", dev.priv, dev.id, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), base64.RawURLEncoding.EncodeToString([]byte("CIPHERTEXT-MARKER"))) {
		t.Fatalf("snapshot %d %s", rec.Code, rec.Body.Bytes())
	}

	small := newNoop(t, 4, time.Hour, 100)
	one := small.enroll("device-q", "group-q", nil, "")
	if rec := small.call(http.MethodPost, "/v1/sync/push", "", one.priv, one.id, small.pushBody("op-q", "rec-q", 1, 0, nil, []byte("too-much-ciphertext"))); rec.Code != http.StatusForbidden {
		t.Fatalf("quota %d %s", rec.Code, rec.Body.Bytes())
	}

	limited := newNoop(t, 1<<20, time.Hour, 1)
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	body := limited.enrollBody("device-r", "group-r", pub, "", "", nil)
	if rec := limited.call(http.MethodPost, "/v1/sync/devices", "", priv, "device-r", body); rec.Code != http.StatusCreated {
		t.Fatalf("enroll %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := limited.call(http.MethodPost, "/v1/sync/devices", "", priv, "device-r", body); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestOIDCGate(t *testing.T) {
	issuer, priv := newIssuer(t)
	env := newOIDC(t, issuer, priv, 1<<20)
	dev := env.enroll("device-a", "group-a", nil, "alice")
	token := env.access("alice", env.at)
	if strings.Contains(env.logs.String(), token) {
		t.Fatal("log contains access token")
	}
	body := env.pushBody("op-1", "rec-1", 1, 0, nil, []byte("CIPHERTEXT-MARKER"))
	if rec := env.raw(http.MethodPost, "/v1/sync/push", token, dev.id, env.at, "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing device signature %d", rec.Code)
	}
	foreign := env.enroll("device-b", "group-b", nil, "bob")
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", env.at), foreign.priv, foreign.id, body); rec.Code != http.StatusForbidden {
		t.Fatalf("cross tenant %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", token, dev.priv, dev.id, body); rec.Code != http.StatusOK {
		t.Fatalf("push %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestSecurityEvents(t *testing.T) {
	issuer, priv := newIssuer(t)
	env := newOIDC(t, issuer, priv, 1<<20)
	dev := env.enroll("device-a", "group-a", nil, "alice")
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", env.at), dev.priv, dev.id, env.pushBody("op-1", "rec-1", 1, 0, nil, []byte("CIPHERTEXT-MARKER"))); rec.Code != http.StatusOK {
		t.Fatalf("push %d %s", rec.Code, rec.Body.Bytes())
	}
	bad := env.set("alice", "jti-bad", env.at, map[string]any{application.EventEntitlementsChanged: map[string]any{"entitlements": []string{"arion.sync.hosted"}}})
	parts := strings.Split(bad, ".")
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	sig[0] ^= 0xff
	parts[2] = base64.RawURLEncoding.EncodeToString(sig)
	bad = strings.Join(parts, ".")
	if rec := env.postSET(bad); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"err":"invalid_key"`) {
		t.Fatalf("bad signature %d %s", rec.Code, rec.Body.Bytes())
	}
	wrongAud := signSET(t, priv, issuer.URL, "other-client", "alice", "jti-aud", env.at, map[string]any{application.EventEntitlementsChanged: map[string]any{}})
	if rec := env.postSET(wrongAud); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_audience"`) {
		t.Fatalf("aud %d %s", rec.Code, rec.Body.Bytes())
	}
	wrongIss := signSET(t, priv, "https://evil.example", "arion-sync", "alice", "jti-iss", env.at, map[string]any{application.EventEntitlementsChanged: map[string]any{}})
	if rec := env.postSET(wrongIss); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_issuer"`) {
		t.Fatalf("iss %d %s", rec.Code, rec.Body.Bytes())
	}

	changed := env.set("alice", "jti-ent", env.at, map[string]any{application.EventEntitlementsChanged: map[string]any{"entitlements": []string{"arion.sync.hosted"}}})
	if rec := env.postSET(changed); rec.Code != http.StatusAccepted || rec.Body.Len() != 0 {
		t.Fatalf("entitlements %d %q", rec.Code, rec.Body.Bytes())
	}
	if rec := env.postSET(changed); rec.Code != http.StatusAccepted || env.store.Counts().Events != 1 {
		t.Fatalf("replay %d events %d", rec.Code, env.store.Counts().Events)
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", env.at), dev.priv, dev.id, env.pushBody("op-2", "rec-2", 1, 0, nil, []byte("still-ciphertext!"))); rec.Code != http.StatusOK {
		t.Fatalf("push after entitlements %d %s", rec.Code, rec.Body.Bytes())
	}

	before := env.at
	revoked := env.set("alice", "jti-sess", env.at, map[string]any{application.EventSessionRevoked: map[string]any{}})
	if rec := env.postSET(revoked); rec.Code != http.StatusAccepted {
		t.Fatalf("session %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", before), dev.priv, dev.id, env.pushBody("op-3", "rec-3", 1, 0, nil, []byte("old-token-cipher!"))); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old token %d %s", rec.Code, rec.Body.Bytes())
	}
	env.at = env.at.Add(time.Second)
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", env.at), dev.priv, dev.id, env.pushBody("op-4", "rec-3", 1, 0, nil, []byte("new-token-cipher!"))); rec.Code != http.StatusOK {
		t.Fatalf("new token %d %s", rec.Code, rec.Body.Bytes())
	}

	disabledAt := env.at
	if rec := env.postSET(env.set("alice", "jti-off", env.at, map[string]any{application.EventAccountDisabled: map[string]any{}})); rec.Code != http.StatusAccepted {
		t.Fatalf("disable %d %s", rec.Code, rec.Body.Bytes())
	}
	kept := env.store.Counts().Envelopes
	if kept == 0 {
		t.Fatal("disable deleted ciphertext")
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", disabledAt), dev.priv, dev.id, env.pushBody("op-5", "rec-4", 1, 0, nil, []byte("frozen-ciphertext"))); rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled old token %d %s", rec.Code, rec.Body.Bytes())
	}
	env.at = env.at.Add(time.Second)
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", env.at), dev.priv, dev.id, env.pushBody("op-6", "rec-4", 1, 0, nil, []byte("frozen-ciphertext"))); rec.Code != http.StatusForbidden {
		t.Fatalf("frozen %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.postSET(env.set("alice", "jti-on", env.at, map[string]any{application.EventAccountEnabled: map[string]any{}})); rec.Code != http.StatusAccepted {
		t.Fatalf("enable %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", env.at), dev.priv, dev.id, env.pushBody("op-7", "rec-5", 1, 0, nil, []byte("restored-cipher!"))); rec.Code != http.StatusOK {
		t.Fatalf("enabled %d %s", rec.Code, rec.Body.Bytes())
	}
	if rec := env.call(http.MethodPost, "/v1/sync/push", env.access("alice", before), dev.priv, dev.id, env.pushBody("op-8", "rec-6", 1, 0, nil, []byte("still-old-cipher!"))); rec.Code != http.StatusUnauthorized {
		t.Fatalf("pre-disable token %d %s", rec.Code, rec.Body.Bytes())
	}

	if rec := env.postSET(env.set("alice", "jti-purge", env.at, map[string]any{application.EventAccountPurged: map[string]any{}})); rec.Code != http.StatusAccepted {
		t.Fatalf("purge %d %s", rec.Code, rec.Body.Bytes())
	}
	counts := env.store.Counts()
	if counts.Envelopes != 0 || counts.Devices != 0 || counts.Groups != 0 || counts.Links != 0 {
		t.Fatalf("purge left rows %+v", counts)
	}
	if rec := env.call(http.MethodPost, "/v1/sync/devices", env.access("alice", env.at), dev.priv, dev.id, env.enrollBody(dev.id, "group-a", dev.pub, "", "", nil)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("recreate %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestContractFixtureRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../../../../packages/contracts/sync/push-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	env := newNoop(t, 1<<20, time.Hour, 100)
	dev := env.enroll("device-contract", "group-contract", nil, "")
	body := bytes.TrimSpace(raw)
	if rec := env.call(http.MethodPost, "/v1/sync/push", "", dev.priv, dev.id, body); rec.Code != http.StatusOK {
		t.Fatalf("fixture %d %s", rec.Code, rec.Body.Bytes())
	}
	cursor0, err := domain.EncodeCursor(0)
	if err != nil {
		t.Fatal(err)
	}
	page := env.pull(dev, cursor0)
	var fixture struct {
		Operations []struct {
			Ciphertext string `json:"ciphertext"`
			Nonce      string `json:"nonce"`
			Tag        string `json:"tag"`
			RecordID   string `json:"record_id"`
		} `json:"operations"`
	}
	decode(t, body, &fixture)
	if len(page.Envelopes) != 1 || page.Envelopes[0].Ciphertext != fixture.Operations[0].Ciphertext || page.Envelopes[0].Nonce != fixture.Operations[0].Nonce || page.Envelopes[0].Tag != fixture.Operations[0].Tag || page.Envelopes[0].RecordID != fixture.Operations[0].RecordID {
		t.Fatalf("round trip %+v", page)
	}
}

type identity struct {
	id   string
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

type harness struct {
	t      *testing.T
	store  *memory.Store
	at     time.Time
	h      http.Handler
	logs   *bytes.Buffer
	issuer string
	sign   ed25519.PrivateKey
}

func newNoop(t *testing.T, quota int64, cursor time.Duration, rate int) *harness {
	t.Helper()
	return newHarness(t, quota, cursor, rate, "noop", "", nil)
}

func newOIDC(t *testing.T, issuer *httptest.Server, priv ed25519.PrivateKey, quota int64) *harness {
	t.Helper()
	return newHarness(t, quota, time.Hour, 100, "oidc", issuer.URL, priv)
}

func newHarness(t *testing.T, quota int64, cursor time.Duration, rate int, mode, issuer string, sign ed25519.PrivateKey) *harness {
	t.Helper()
	store := memory.New()
	logs := &bytes.Buffer{}
	log, err := logging.New(logs, "info")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	h := &harness{t: t, store: store, at: at, logs: logs, issuer: issuer, sign: sign}
	var gate application.Gate
	var events application.EventVerifier
	var directory application.Directory
	var accounts application.Accounts
	if mode == "noop" {
		gate = noop.New(quota)
	} else {
		provider, err := oidc.New(oidc.Options{
			Issuer: issuer, Audience: "https://api.arion.astrafoundry.si", ClientID: "arion-sync", Entitlement: "arion.sync.hosted",
			JWKSURL: issuer + "/jwks", Algorithms: []string{"EdDSA"}, Leeway: 30 * time.Second, CacheTTL: time.Hour,
			Quota: quota, Directory: store, Now: func() time.Time { return h.at },
		})
		if err != nil {
			t.Fatal(err)
		}
		gate = provider
		events = provider
		directory = store
		accounts = store
	}
	svc, err := application.New(application.Options{
		Store: store, Directory: directory, Accounts: accounts, Gate: gate, Events: events,
		Limits: domain.DefaultLimits(), CursorTTL: cursor, Skew: 5 * time.Minute, PageSize: 100,
		Now: func() time.Time { return h.at }, Log: log,
	})
	if err != nil {
		t.Fatal(err)
	}
	h.h = New(Deps{Service: svc, Ready: func(context.Context) error { return nil }, Log: log, MaxBody: 1 << 20, RatePerMinute: rate, Now: func() time.Time { return h.at }})
	return h
}

func (h *harness) enroll(id, group string, sponsor *identity, subject string) identity {
	h.t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		h.t.Fatal(err)
	}
	var sponsorID, sponsorSig string
	if sponsor != nil {
		sponsorID = sponsor.id
		sponsorSig = base64.RawURLEncoding.EncodeToString(ed25519.Sign(sponsor.priv, domain.EnrollMessage(group, id, domain.EncodePublicKey(pub))))
	}
	body := h.enrollBody(id, group, pub, "", sponsorID, sponsorSig)
	auth := ""
	if subject != "" {
		auth = h.access(subject, h.at)
	}
	rec := h.call(http.MethodPost, "/v1/sync/devices", auth, priv, id, body)
	if rec.Code != http.StatusCreated {
		h.t.Fatalf("enroll %s %d %s", id, rec.Code, rec.Body.Bytes())
	}
	return identity{id: id, pub: pub, priv: priv}
}

func (h *harness) enrollSponsored(id, group string, sponsor identity) identity {
	h.t.Helper()
	return h.enroll(id, group, &sponsor, "")
}

func (h *harness) enrollBody(id, group string, pub ed25519.PublicKey, recovery, sponsorID string, sponsorSig any) []byte {
	h.t.Helper()
	payload := map[string]any{"device_id": id, "group_id": group, "public_key": domain.EncodePublicKey(pub)}
	if recovery != "" {
		payload["recovery"] = recovery
	}
	if sponsorID != "" {
		payload["sponsor_device_id"] = sponsorID
		payload["sponsor_signature"] = sponsorSig
	}
	body, err := json.Marshal(payload)
	if err != nil {
		h.t.Fatal(err)
	}
	return body
}

func (h *harness) pushBody(op, record string, epoch uint64, expected int64, parents []int64, ciphertext []byte) []byte {
	return h.pushBodyKind(op, record, epoch, expected, parents, ciphertext, "write")
}

func (h *harness) pushBodyKind(op, record string, epoch uint64, expected int64, parents []int64, ciphertext []byte, kind string) []byte {
	h.t.Helper()
	if parents == nil {
		parents = []int64{}
	}
	body, err := json.Marshal(map[string]any{"operations": []map[string]any{{
		"operation_id": op, "record_id": record, "schema_version": 1, "key_epoch": epoch,
		"algorithm": "AES-256-GCM", "nonce": base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 12)),
		"ciphertext": base64.RawURLEncoding.EncodeToString(ciphertext), "tag": base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{2}, 16)),
		"logical_revision": 1, "kind": kind, "expected_revision": expected, "parents": parents,
	}}})
	if err != nil {
		h.t.Fatal(err)
	}
	return body
}

func (h *harness) pull(dev identity, cursor string) logDTO {
	h.t.Helper()
	rec := h.call(http.MethodGet, "/v1/sync/pull?cursor="+cursor, h.bearer(dev), dev.priv, dev.id, nil)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("pull %d %s", rec.Code, rec.Body.Bytes())
	}
	var page logDTO
	decode(h.t, rec.Body.Bytes(), &page)
	return page
}

func (h *harness) bearer(identity) string {
	if h.issuer == "" {
		return ""
	}
	return h.access("alice", h.at)
}

func (h *harness) access(subject string, iat time.Time) string {
	h.t.Helper()
	claims := jwt.MapClaims{
		"iss": h.issuer, "sub": subject, "aud": "https://api.arion.astrafoundry.si",
		"iat": iat.Unix(), "nbf": iat.Unix(), "exp": iat.Add(time.Hour).Unix(),
		"ent": []string{"arion.sync.hosted"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "sync-test"
	token.Header["typ"] = "at+jwt"
	signed, err := token.SignedString(h.sign)
	if err != nil {
		h.t.Fatal(err)
	}
	return "Bearer " + signed
}

func (h *harness) set(subject, jti string, iat time.Time, events map[string]any) string {
	return signSET(h.t, h.sign, h.issuer, "arion-sync", subject, jti, iat, events)
}

func (h *harness) postSET(token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/security-events", strings.NewReader(token))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	return rec
}

func (h *harness) call(method, target, auth string, priv ed25519.PrivateKey, deviceID string, body []byte) *httptest.ResponseRecorder {
	sig := ""
	if priv != nil {
		sig = base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, domain.CanonicalRequest(deviceID, h.at.UnixMilli(), method, target, body)))
	}
	return h.raw(method, target, auth, deviceID, h.at, sig, body)
}

func (h *harness) raw(method, target, auth, deviceID string, at time.Time, sig string, body []byte) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("X-Arion-Device-Id", deviceID)
	req.Header.Set("X-Arion-Timestamp", strconvMilli(at))
	if sig != "" {
		req.Header.Set("X-Arion-Signature", sig)
	}
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	return rec
}

func signSET(t *testing.T, priv ed25519.PrivateKey, iss, aud, subject, jti string, iat time.Time, events map[string]any) string {
	t.Helper()
	encoded := map[string]json.RawMessage{}
	for kind, payload := range events {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		encoded[kind] = raw
	}
	claims := jwt.MapClaims{
		"iss": iss, "aud": aud, "iat": iat.Unix(), "jti": jti,
		"sub_id": map[string]string{"format": "iss_sub", "iss": iss, "sub": subject},
		"events": encoded,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "sync-test"
	token.Header["typ"] = "secevent+jwt"
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func newIssuer(t *testing.T) (*httptest.Server, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"keys": []map[string]string{{
		"kty": "OKP", "crv": "Ed25519", "use": "sig", "alg": "EdDSA", "kid": "sync-test",
		"x": base64.RawURLEncoding.EncodeToString(pub),
	}}}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(doc)
	}))
	t.Cleanup(srv.Close)
	return srv, priv
}

func strconvMilli(at time.Time) string {
	return strconv.FormatInt(at.UnixMilli(), 10)
}

func containsRecord(page logDTO, id string) bool {
	for _, envl := range page.Envelopes {
		if envl.RecordID == id {
			return true
		}
	}
	return false
}

func decode(t *testing.T, raw []byte, dest any) {
	t.Helper()
	if err := json.Unmarshal(raw, dest); err != nil {
		t.Fatal(err)
	}
}
