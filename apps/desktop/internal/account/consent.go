package account

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"arion.local/desktop/internal/usageupload"
)

// UsageConsentVersion is the notice version the service records. A token
// carries it as `usage_consent_version` once the choice is accepted.
const UsageConsentVersion = "usage-v1"

type usageConsentPayload struct {
	Data struct {
		Accepted bool   `json:"accepted"`
		Version  string `json:"version"`
	} `json:"data"`
}

// RecordUsageConsent records the choice the user made in the app's own notice
// with the signed-in account, using its access token. The service stores a
// version and timestamp and keeps a history of acceptance and withdrawal, so a
// later choice replaces an earlier one. The returned session carries the
// recorded choice; a failure leaves the session as it was.
func (s *Service) RecordUsageConsent(ctx context.Context, accepted bool) (Session, error) {
	if s.cfg.Disabled {
		return Session{}, ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	current, busy := s.snapshotSession()
	if busy {
		return Session{}, ErrSignInInProgress
	}
	if current.AccessToken == "" {
		return Session{}, ErrNotSignedIn
	}
	payload, err := json.Marshal(struct {
		ProductSlug string `json:"productSlug"`
		Accepted    bool   `json:"accepted"`
	}{usageupload.Product, accepted})
	if err != nil {
		return Session{}, wrap(CodeUsageConsent, "request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.Issuer+"/api/v1/usage/consent", bytes.NewReader(payload))
	if err != nil {
		return Session{}, wrap(CodeUsageConsent, "request", err)
	}
	req.Header.Set("Authorization", "Bearer "+current.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	body, status, err := s.do(req)
	if err != nil {
		return Session{}, classifyWait(ctx, wrap(CodeUsageConsent, "transport", err))
	}
	if status != http.StatusOK {
		return Session{}, &Error{Code: CodeUsageConsent, Detail: "http_" + strconv.Itoa(status)}
	}
	var parsed usageConsentPayload
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Session{}, wrap(CodeUsageConsent, "invalid_json", err)
	}
	// Trust the service's record, not the request: the answer must name the
	// choice and, for an acceptance, the notice version this client knows.
	if parsed.Data.Accepted != accepted || (accepted && parsed.Data.Version != UsageConsentVersion) {
		return Session{}, &Error{Code: CodeUsageConsent, Detail: "mismatch"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession || s.session.Subject != current.Subject {
		return Session{}, ErrNotSignedIn
	}
	s.session.UsageConsent = accepted
	return s.session, nil
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
