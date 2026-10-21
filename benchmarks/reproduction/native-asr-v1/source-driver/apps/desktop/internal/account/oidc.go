package account

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type provider struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	RevocationEndpoint    string   `json:"revocation_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
}

func (s *Service) discover(ctx context.Context) (provider, error) {
	endpoint := s.cfg.Issuer + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return provider{}, wrap(CodeDiscovery, "request", err)
	}
	req.Header.Set("Accept", "application/json")
	body, status, err := s.do(req)
	if err != nil {
		return provider{}, wrap(CodeDiscovery, "transport", err)
	}
	if status != http.StatusOK {
		return provider{}, &Error{Code: CodeDiscovery, Detail: "http_" + strconv.Itoa(status)}
	}
	var doc provider
	if err := json.Unmarshal(body, &doc); err != nil {
		return provider{}, wrap(CodeDiscovery, "document", err)
	}
	if doc.Issuer != s.cfg.Issuer {
		return provider{}, &Error{Code: CodeDiscovery, Detail: "issuer_mismatch"}
	}
	if err := checkHTTPURL(doc.AuthorizationEndpoint); err != nil {
		return provider{}, wrap(CodeDiscovery, "authorization_endpoint", err)
	}
	if err := checkHTTPURL(doc.TokenEndpoint); err != nil {
		return provider{}, wrap(CodeDiscovery, "token_endpoint", err)
	}
	if doc.RevocationEndpoint != "" {
		if err := checkHTTPURL(doc.RevocationEndpoint); err != nil {
			return provider{}, wrap(CodeDiscovery, "revocation_endpoint", err)
		}
	}
	if doc.UserInfoEndpoint != "" {
		if err := checkHTTPURL(doc.UserInfoEndpoint); err != nil {
			return provider{}, wrap(CodeDiscovery, "userinfo_endpoint", err)
		}
	}
	if len(doc.CodeChallengeMethods) > 0 && !contains(doc.CodeChallengeMethods, "S256") {
		return provider{}, &Error{Code: CodeDiscovery, Detail: "s256_required"}
	}
	return doc, nil
}

func authorizeURL(doc provider, cfg Config, redirect, state, challenge string) (string, error) {
	u, err := url.Parse(doc.AuthorizationEndpoint)
	if err != nil {
		return "", wrap(CodeDiscovery, "authorization_endpoint", err)
	}
	q := u.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirect)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(cfg.Scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("resource", cfg.Resource)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Service) exchange(ctx context.Context, doc provider, redirect, code, verifier string) (tokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirect)
	form.Set("client_id", s.cfg.ClientID)
	form.Set("code_verifier", verifier)
	form.Set("resource", s.cfg.Resource)
	return s.token(ctx, doc.TokenEndpoint, form, true)
}

func (s *Service) refreshToken(ctx context.Context, doc provider, refresh string) (tokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)
	form.Set("client_id", s.cfg.ClientID)
	form.Set("resource", s.cfg.Resource)
	return s.token(ctx, doc.TokenEndpoint, form, false)
}

func (s *Service) token(ctx context.Context, endpoint string, form url.Values, requireRefresh bool) (tokenSet, error) {
	body, status, err := s.postForm(ctx, endpoint, form)
	if err != nil {
		return tokenSet{}, wrap(CodeToken, "transport", err)
	}
	var tok tokenSet
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &tok); err != nil {
			return tokenSet{}, wrap(CodeToken, "document", err)
		}
	}
	if status != http.StatusOK {
		detail := tok.Error
		if detail == "" {
			detail = "http_" + strconv.Itoa(status)
		}
		return tok, &Error{Code: CodeToken, Detail: detail}
	}
	if tok.Error != "" {
		return tok, &Error{Code: CodeToken, Detail: tok.Error}
	}
	if tok.AccessToken == "" {
		return tok, &Error{Code: CodeToken, Detail: "missing_access_token"}
	}
	if requireRefresh && tok.RefreshToken == "" {
		return tok, &Error{Code: CodeToken, Detail: "missing_refresh_token"}
	}
	if tok.RefreshToken != "" && stringsContainTokenBreak(tok.RefreshToken) {
		return tok, &Error{Code: CodeToken, Detail: "refresh_token"}
	}
	return tok, nil
}

// userInfo fills the profile from the UserInfo endpoint, where issuers such
// as Better Auth 1.7 keep the profile and email claims out of the ID token.
// The profile is display data: when the endpoint is missing or fails, the
// session keeps the ID token's claims and sign-in still succeeds.
func (s *Service) userInfo(ctx context.Context, doc provider, session *Session) {
	if doc.UserInfoEndpoint == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, doc.UserInfoEndpoint, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", session.TokenType+" "+session.AccessToken)
	req.Header.Set("Accept", "application/json")
	body, status, err := s.do(req)
	if err != nil || status != http.StatusOK {
		return
	}
	var claims map[string]any
	if json.Unmarshal(body, &claims) != nil || stringClaim(claims, "sub") != session.Subject {
		return
	}
	for _, field := range []struct {
		claim  string
		target *string
	}{{"name", &session.Profile.Name}, {"email", &session.Profile.Email}, {"picture", &session.Profile.Picture}, {"locale", &session.Profile.Locale}} {
		if value := stringClaim(claims, field.claim); value != "" {
			*field.target = value
		}
	}
}

func (s *Service) revoke(ctx context.Context, endpoint, refresh string) error {
	if endpoint == "" {
		return &Error{Code: CodeRevoke, Detail: "missing_revocation_endpoint"}
	}
	form := url.Values{}
	form.Set("token", refresh)
	form.Set("token_type_hint", "refresh_token")
	form.Set("client_id", s.cfg.ClientID)
	body, status, err := s.postForm(ctx, endpoint, form)
	if err != nil {
		return wrap(CodeRevoke, "transport", err)
	}
	if status >= 200 && status < 300 {
		return nil
	}
	detail := oauthError(body)
	if detail == "" {
		detail = "http_" + strconv.Itoa(status)
	}
	return &Error{Code: CodeRevoke, Detail: detail}
}

func oauthError(body []byte) string {
	var parsed struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &parsed) != nil {
		return ""
	}
	return parsed.Error
}

func (s *Service) postForm(ctx context.Context, endpoint string, form url.Values) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return s.do(req)
}

func (s *Service) do(req *http.Request) ([]byte, int, error) {
	res, err := s.cfg.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, err := readBody(res.Body)
	if err != nil {
		return nil, res.StatusCode, err
	}
	return body, res.StatusCode, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (s *Service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}
