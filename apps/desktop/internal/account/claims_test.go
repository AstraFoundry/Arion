package account

import (
	"testing"
	"time"
)

func TestEntitlementDecodeDoesNotVerifySignature(t *testing.T) {
	token := unsignedJWT(map[string]any{"sub": "user-1", "ent": []any{HostedSyncEntitlement}})
	token = token[:len(token)-1] + "x"
	claims, err := decodeClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	got, err := entitlements(claims)
	if err != nil {
		t.Fatal(err)
	}
	session, err := sessionFromTokens(DefaultIssuer, tokenSet{AccessToken: token, TokenType: "Bearer", ExpiresIn: 30}, time.Unix(10, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !session.HasHostedSync() || len(got) != 1 || got[0] != HostedSyncEntitlement {
		t.Fatalf("ent = %v hosted %v", got, session.HasHostedSync())
	}
}

func TestEntitlementRejectsNonArray(t *testing.T) {
	token := unsignedJWT(map[string]any{"sub": "user-1", "ent": "arion.sync.hosted"})
	_, err := sessionFromTokens(DefaultIssuer, tokenSet{AccessToken: token, RefreshToken: "rt"}, time.Unix(10, 0))
	if DetailOf(err) != "ent_type" {
		t.Fatalf("err = %v", err)
	}
	absent := unsignedJWT(map[string]any{"sub": "user-1"})
	session, err := sessionFromTokens(DefaultIssuer, tokenSet{AccessToken: absent, RefreshToken: "rt"}, time.Unix(10, 0))
	if err != nil {
		t.Fatal(err)
	}
	if session.HasHostedSync() || session.Entitlements != nil {
		t.Fatalf("ent = %#v", session.Entitlements)
	}
}

func TestMapWaitlist(t *testing.T) {
	cases := []struct {
		status   int
		body     string
		entitled bool
		want     WaitlistStatus
	}{
		{201, `{"id":"e","status":"pending"}`, false, WaitlistJoinedPending},
		{201, `{}`, false, WaitlistJoinedPending},
		{201, `{"status":"active"}`, false, WaitlistError},
		{409, `{"code":"already_entitled"}`, false, WaitlistAlreadyEntitled},
		{409, `{"code":"already_pending"}`, true, WaitlistJoinedPending},
		{409, `{"entitled":true}`, false, WaitlistAlreadyEntitled},
		{409, `{}`, true, WaitlistAlreadyEntitled},
		{409, `{}`, false, WaitlistJoinedPending},
		{409, `{"code":"nope"}`, true, WaitlistError},
		{403, `{"code":"waitlist_closed"}`, false, WaitlistClosed},
		{200, `{"status":"paused"}`, false, WaitlistClosed},
		{500, `{}`, false, WaitlistError},
	}
	for _, tc := range cases {
		got := mapWaitlist(tc.status, []byte(tc.body), tc.entitled)
		if got.Status != tc.want {
			t.Errorf("status %d body %s entitled %v = %s, want %s", tc.status, tc.body, tc.entitled, got.Status, tc.want)
		}
	}
}
