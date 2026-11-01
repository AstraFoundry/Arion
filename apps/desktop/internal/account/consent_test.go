package account

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestRecordUsageConsentSendsTheChoiceWithTheAccessToken(t *testing.T) {
	for _, accepted := range []bool{true, false} {
		iss := newIssuer(t)
		svc := iss.service(&memStore{})
		signed, err := svc.SignIn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if signed.UsageConsent {
			t.Fatal("fixture token already carried consent")
		}
		got, err := svc.RecordUsageConsent(context.Background(), accepted)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"productSlug":"arion","accepted":` + map[bool]string{true: "true", false: "false"}[accepted] + `}`
		if iss.consentAuth != "Bearer "+signed.AccessToken || iss.consentRequest != want {
			t.Fatalf("auth %q body %q, want body %s", iss.consentAuth, iss.consentRequest, want)
		}
		if got.UsageConsent != accepted {
			t.Fatalf("session consent = %v, want %v", got.UsageConsent, accepted)
		}
		if current, _ := svc.Session(); current.UsageConsent != accepted {
			t.Fatalf("service session consent = %v, want %v", current.UsageConsent, accepted)
		}
	}
}

func TestRecordUsageConsentKeepsTheSessionOnFailure(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		detail string
	}{
		{"refused", http.StatusForbidden, `{}`, "http_403"},
		{"invalid json", http.StatusOK, `not json`, "invalid_json"},
		{"wrong answer", http.StatusOK, `{"data":{"accepted":false,"decided":true,"version":"usage-v1"}}`, "mismatch"},
		{"unknown version", http.StatusOK, `{"data":{"accepted":true,"decided":true,"version":"old"}}`, "mismatch"},
		{"not recorded", http.StatusOK, `{"data":{"accepted":true,"decided":false,"version":"usage-v1"}}`, "mismatch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			iss := newIssuer(t)
			iss.consentStatus, iss.consentBody = tc.status, tc.body
			svc := iss.service(&memStore{})
			if _, err := svc.SignIn(context.Background()); err != nil {
				t.Fatal(err)
			}
			_, err := svc.RecordUsageConsent(context.Background(), true)
			var failure *Error
			if !errors.As(err, &failure) || failure.Code != CodeUsageConsent || failure.Detail != tc.detail {
				t.Fatalf("err = %v, want %s/%s", err, CodeUsageConsent, tc.detail)
			}
			noSecret(t, err)
			if current, _ := svc.Session(); current.UsageConsent {
				t.Fatal("a failed request recorded consent in the session")
			}
		})
	}
}

func TestRecordUsageConsentNeedsASignedInAccount(t *testing.T) {
	iss := newIssuer(t)
	svc := iss.service(&memStore{})
	if _, err := svc.RecordUsageConsent(context.Background(), true); !errors.Is(err, ErrNotSignedIn) {
		t.Fatalf("err = %v, want not signed in", err)
	}
	if iss.consentHits != 0 {
		t.Fatal("sent a consent request without a token")
	}
}

func TestReadUsageConsentSeparatesUndecidedFromDeclined(t *testing.T) {
	iss := newIssuer(t)
	svc := iss.service(&memStore{})
	signed, err := svc.SignIn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []UsageConsent{{}, {Accepted: false, Decided: true}, {Accepted: true, Decided: true}} {
		iss.mu.Lock()
		iss.consentAccepted, iss.consentDecided = want.Accepted, want.Decided
		iss.mu.Unlock()
		got, answer, err := svc.ReadUsageConsent(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if answer != want || got.UsageConsent != want.Accepted {
			t.Fatalf("answer %+v session %v, want %+v", answer, got.UsageConsent, want)
		}
		if iss.consentAuth != "Bearer "+signed.AccessToken {
			t.Fatalf("auth %q", iss.consentAuth)
		}
	}
}

func TestReadUsageConsentRejectsAnUnknownAcceptance(t *testing.T) {
	iss := newIssuer(t)
	iss.consentBody = `{"data":{"accepted":true,"decided":true,"version":"old"}}`
	svc := iss.service(&memStore{})
	if _, err := svc.SignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ReadUsageConsent(context.Background()); CodeOf(err) != CodeUsageConsent || DetailOf(err) != "mismatch" {
		t.Fatalf("err = %v, want mismatch", err)
	}
	if current, _ := svc.Session(); current.UsageConsent {
		t.Fatal("an unknown version enabled reporting")
	}
}
