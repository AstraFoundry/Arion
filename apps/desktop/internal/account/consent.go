package account

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"arion.local/desktop/internal/usageupload"
)

// UsageConsentVersion is the notice version the service records. A token
// carries it as `usage_consent_version` once the choice is accepted.
const UsageConsentVersion = "usage-v1"

// UsageConsent is the account's answer to sharing usage data. Sharing is a
// separate choice from signing in: Decided is false until the account has
// answered the current version either way.
type UsageConsent struct {
	Accepted bool
	Decided  bool
}

type usageConsentPayload struct {
	Data struct {
		Accepted bool   `json:"accepted"`
		Decided  bool   `json:"decided"`
		Version  string `json:"version"`
	} `json:"data"`
}

// ReadUsageConsent asks the platform for the signed-in account's choice. The
// platform is the record; the returned session carries its answer, which
// replaces the token's claim until the next sign-in or refresh.
func (s *Service) ReadUsageConsent(ctx context.Context) (Session, UsageConsent, error) {
	query := url.Values{"productSlug": {usageupload.Product}}
	return s.usageConsent(ctx, http.MethodGet, "?"+query.Encode(), nil, nil)
}

// RecordUsageConsent records a choice made in Arion's usage step or settings
// switch with the signed-in account, using its access token. The platform
// keeps a history of acceptance and withdrawal, so a later choice replaces an
// earlier one. A failure leaves the session as it was.
func (s *Service) RecordUsageConsent(ctx context.Context, accepted bool) (Session, error) {
	payload, err := json.Marshal(struct {
		ProductSlug string `json:"productSlug"`
		Accepted    bool   `json:"accepted"`
	}{usageupload.Product, accepted})
	if err != nil {
		return Session{}, wrap(CodeUsageConsent, "request", err)
	}
	session, _, err := s.usageConsent(ctx, http.MethodPut, "", bytes.NewReader(payload), &accepted)
	return session, err
}

// usageConsent calls the platform's consent route with the access token. With
// want set, the answer must echo that choice.
func (s *Service) usageConsent(ctx context.Context, method, query string, body io.Reader, want *bool) (Session, UsageConsent, error) {
	if s.cfg.Disabled {
		return Session{}, UsageConsent{}, ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	current, busy := s.snapshotSession()
	if busy {
		return Session{}, UsageConsent{}, ErrSignInInProgress
	}
	if current.AccessToken == "" {
		return Session{}, UsageConsent{}, ErrNotSignedIn
	}
	req, err := http.NewRequestWithContext(ctx, method, s.cfg.Issuer+"/api/v1/usage/consent"+query, body)
	if err != nil {
		return Session{}, UsageConsent{}, wrap(CodeUsageConsent, "request", err)
	}
	req.Header.Set("Authorization", "Bearer "+current.AccessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	raw, status, err := s.do(req)
	if err != nil {
		return Session{}, UsageConsent{}, classifyWait(ctx, wrap(CodeUsageConsent, "transport", err))
	}
	if status != http.StatusOK {
		return Session{}, UsageConsent{}, &Error{Code: CodeUsageConsent, Detail: "http_" + strconv.Itoa(status)}
	}
	var parsed usageConsentPayload
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Session{}, UsageConsent{}, wrap(CodeUsageConsent, "invalid_json", err)
	}
	// Trust only an answer this client understands: an acceptance names the
	// version it knows, and a recorded choice is echoed and decided.
	answer := UsageConsent{Accepted: parsed.Data.Accepted, Decided: parsed.Data.Decided}
	if (answer.Accepted && (parsed.Data.Version != UsageConsentVersion || !answer.Decided)) ||
		(want != nil && (answer.Accepted != *want || !answer.Decided)) {
		return Session{}, UsageConsent{}, &Error{Code: CodeUsageConsent, Detail: "mismatch"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession || s.session.Subject != current.Subject {
		return Session{}, UsageConsent{}, ErrNotSignedIn
	}
	s.session.UsageConsent = answer.Accepted
	return s.session, answer, nil
}

// snapshotSession copies the session unless a sign-in, refresh or sign-out
// owns the service.
func (s *Service) snapshotSession() (session Session, busy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inflight || s.blocked > 0 {
		return Session{}, true
	}
	if !s.hasSession {
		return Session{}, false
	}
	return s.session, false
}
