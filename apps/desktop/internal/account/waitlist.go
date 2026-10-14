package account

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// WaitlistStatus is the typed result of joining the hosted-sync waitlist.
type WaitlistStatus string

const (
	WaitlistJoinedPending   WaitlistStatus = "joined_pending"
	WaitlistAlreadyEntitled WaitlistStatus = "already_entitled"
	WaitlistClosed          WaitlistStatus = "closed"
	WaitlistError           WaitlistStatus = "error"
)

// WaitlistResult is the mapped join response. StatusError still returns a
// non-nil error from JoinHostedSyncWaitlist. The other statuses return nil.
type WaitlistResult struct {
	Status     WaitlistStatus
	EntryID    string
	HTTPStatus int
	Code       string
}

type waitlistPayload struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Code     string `json:"code"`
	Entitled *bool  `json:"entitled"`
}

type catalogProduct struct {
	Waitlists []catalogWaitlist `json:"waitlists"`
}

type catalogWaitlist struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Status string `json:"status"`
}

func (s *Service) JoinHostedSyncWaitlist(ctx context.Context) (WaitlistResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	token, entitled, busy := s.snapshotForJoin()
	if busy {
		return WaitlistResult{}, ErrSignInInProgress
	}
	if token == "" {
		return WaitlistResult{}, ErrNotSignedIn
	}
	id := s.cfg.WaitlistID
	if id == "" {
		found, status, err := s.lookupWaitlist(ctx)
		if err != nil {
			return WaitlistResult{Status: WaitlistError, Code: DetailOf(err)}, classifyWait(ctx, err)
		}
		if closedStatus(status) {
			return WaitlistResult{Status: WaitlistClosed, Code: status}, nil
		}
		id = found
	}
	if !validWaitlistID(id) {
		return WaitlistResult{Status: WaitlistError, Code: "waitlist_id"}, &Error{Code: CodeWaitlist, Detail: "waitlist_id"}
	}
	result, err := s.postWaitlist(ctx, id, token, entitled)
	if err != nil {
		return result, classifyWait(ctx, err)
	}
	return result, nil
}

func (s *Service) lookupWaitlist(ctx context.Context) (id, status string, err error) {
	endpoint := s.cfg.Issuer + "/api/v1/catalog/products/" + CatalogProduct
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", wrap(CodeWaitlist, "catalog", err)
	}
	req.Header.Set("Accept", "application/json")
	body, statusCode, err := s.do(req)
	if err != nil {
		return "", "", wrap(CodeWaitlist, "catalog", err)
	}
	if statusCode != http.StatusOK {
		return "", "", &Error{Code: CodeWaitlist, Detail: "catalog_http_" + strconv.Itoa(statusCode)}
	}
	var product catalogProduct
	if err := json.Unmarshal(body, &product); err != nil {
		return "", "", wrap(CodeWaitlist, "catalog", err)
	}
	for _, item := range product.Waitlists {
		if item.Key != HostedSyncWaitlistKey {
			continue
		}
		if !validWaitlistID(item.ID) {
			return "", "", &Error{Code: CodeWaitlist, Detail: "waitlist_id"}
		}
		return item.ID, item.Status, nil
	}
	return "", "", &Error{Code: CodeWaitlist, Detail: "waitlist_missing"}
}

func (s *Service) postWaitlist(ctx context.Context, id, token string, entitled bool) (WaitlistResult, error) {
	endpoint := s.cfg.Issuer + "/api/v1/me/waitlists/" + url.PathEscape(id) + "/entries"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(`{"answers":{}}`))
	if err != nil {
		return WaitlistResult{Status: WaitlistError, Code: "request"}, wrap(CodeWaitlist, "request", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	body, statusCode, err := s.do(req)
	if err != nil {
		return WaitlistResult{Status: WaitlistError, Code: "transport"}, wrap(CodeWaitlist, "transport", err)
	}
	result := mapWaitlist(statusCode, body, entitled)
	if result.Status == WaitlistError {
		return result, &Error{Code: CodeWaitlist, Detail: result.Code}
	}
	return result, nil
}

func mapWaitlist(status int, body []byte, entitled bool) WaitlistResult {
	var parsed waitlistPayload
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &parsed); err != nil {
			return WaitlistResult{Status: WaitlistError, HTTPStatus: status, Code: "invalid_json"}
		}
	}
	if waitlistClosed(parsed) {
		code := parsed.Code
		if code == "" {
			code = parsed.Status
		}
		return WaitlistResult{Status: WaitlistClosed, HTTPStatus: status, Code: code}
	}
	switch status {
	case http.StatusCreated:
		if parsed.Status != "" && parsed.Status != "pending" {
			return WaitlistResult{Status: WaitlistError, HTTPStatus: status, Code: parsed.Status}
		}
		return WaitlistResult{Status: WaitlistJoinedPending, HTTPStatus: status, EntryID: parsed.ID}
	case http.StatusConflict:
		switch {
		case parsed.Code == "already_entitled" || (parsed.Entitled != nil && *parsed.Entitled):
			return WaitlistResult{Status: WaitlistAlreadyEntitled, HTTPStatus: status, Code: parsed.Code}
		case parsed.Code == "already_pending" || parsed.Status == "pending":
			return WaitlistResult{Status: WaitlistJoinedPending, HTTPStatus: status, Code: parsed.Code}
		case parsed.Code != "":
			return WaitlistResult{Status: WaitlistError, HTTPStatus: status, Code: parsed.Code}
		case entitled:
			return WaitlistResult{Status: WaitlistAlreadyEntitled, HTTPStatus: status}
		default:
			return WaitlistResult{Status: WaitlistJoinedPending, HTTPStatus: status}
		}
	default:
		code := parsed.Code
		if code == "" {
			code = "http_" + strconv.Itoa(status)
		}
		return WaitlistResult{Status: WaitlistError, HTTPStatus: status, Code: code}
	}
}

func waitlistClosed(parsed waitlistPayload) bool {
	return closedStatus(parsed.Code) || closedStatus(parsed.Status)
}

func closedStatus(value string) bool {
	switch strings.ToLower(value) {
	case "closed", "paused", "draft", "not_open", "waitlist_closed":
		return true
	default:
		return false
	}
}

func validWaitlistID(id string) bool {
	if id == "" || id == "." || id == ".." || len(id) > 200 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == '~':
		default:
			return false
		}
	}
	return true
}

func classifyWait(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ErrSignInCancelled
	}
	return err
}

func readBody(r io.Reader) ([]byte, error) {
	const max = 1 << 20
	body, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if len(body) > max {
		return nil, errTooLarge
	}
	return body, nil
}
