package cloudpolish

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"arion.local/desktop/internal/oauthloopback"
)

// The ChatGPT sign-in is the OAuth public client the Codex CLI registers,
// which Republic's codex provider refreshes. Its redirect URI is fixed to
// port 1455 on localhost, so the listener cannot take a random port.
const (
	chatgptIssuer   = "https://auth.openai.com"
	chatgptClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	chatgptListen   = "127.0.0.1:1455"
	chatgptPath     = "/auth/callback"
	chatgptRedirect = "http://localhost:1455" + chatgptPath
	chatgptScope    = "openid profile email offline_access"
	// originator names Arion to the ChatGPT backend.
	originator = "arion"
	// signInTimeout bounds the wait for the browser.
	signInTimeout = 5 * time.Minute
)

// tokens is the token endpoint's answer.
type tokens struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// chatgptAuthorizeURL is the browser's sign-in page.
func chatgptAuthorizeURL(issuer, state, challenge string) string {
	q := url.Values{
		"response_type":              {"code"},
		"client_id":                  {chatgptClientID},
		"redirect_uri":               {chatgptRedirect},
		"scope":                      {chatgptScope},
		"code_challenge":             {challenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {state},
		"originator":                 {originator},
	}
	return issuer + "/oauth/authorize?" + q.Encode()
}

// signInChatGPT runs the browser sign-in and returns the credential to
// store. The listener serves one callback, then closes.
func signInChatGPT(ctx context.Context, issuer string, client *http.Client, browser func(string) error, page func() []byte) (Credential, error) {
	ctx, cancel := context.WithTimeout(ctx, signInTimeout)
	defer cancel()
	verifier, challenge, err := oauthloopback.NewPKCE()
	if err != nil {
		return Credential{}, fail(CodeSignIn, err)
	}
	state, err := oauthloopback.NewState()
	if err != nil {
		return Credential{}, fail(CodeSignIn, err)
	}
	srv, _, err := oauthloopback.Start(ctx, chatgptListen, chatgptPath, page)
	if err != nil {
		if ctx.Err() != nil {
			return Credential{}, stopped(ctx)
		}
		return Credential{}, fail(CodePortInUse, err)
	}
	defer srv.Close()
	if err := browser(chatgptAuthorizeURL(issuer, state, challenge)); err != nil {
		return Credential{}, fail(CodeSignIn, err)
	}
	cb, err := srv.Wait(ctx)
	if err != nil {
		return Credential{}, stopped(ctx)
	}
	switch {
	case !oauthloopback.MatchState(state, cb.State):
		return Credential{}, fail(CodeSignIn, fmt.Errorf("state mismatch"))
	case cb.Error != "" || cb.Code == "":
		return Credential{}, fail(CodeSignIn, fmt.Errorf("authorization %q", cb.Error))
	}
	tok, err := tokenRequest(ctx, issuer, client, url.Values{"grant_type": {"authorization_code"}, "code": {cb.Code},
		"redirect_uri": {chatgptRedirect}, "client_id": {chatgptClientID}, "code_verifier": {verifier}})
	if err != nil {
		return Credential{}, err
	}
	return credentialFrom(tok)
}

// stopped classifies a sign-in that ended before its callback.
func stopped(ctx context.Context) error {
	if ctx.Err() == context.DeadlineExceeded {
		return fail(CodeTimeout, ctx.Err())
	}
	return fail(CodeCancelled, ctx.Err())
}

// refreshChatGPT trades the stored refresh token for fresh tokens; the
// access token is never stored.
func refreshChatGPT(ctx context.Context, issuer string, client *http.Client, refresh string) (tokens, error) {
	tok, err := tokenRequest(ctx, issuer, client, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}, "client_id": {chatgptClientID}})
	if err == nil && tok.RefreshToken == "" {
		tok.RefreshToken = refresh
	}
	return tok, err
}

func tokenRequest(ctx context.Context, issuer string, client *http.Client, form url.Values) (tokens, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return tokens{}, fail(CodeSignIn, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return tokens{}, fail(CodeNetwork, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return tokens{}, fail(CodeRejected, fmt.Errorf("token endpoint HTTP %d", res.StatusCode))
	}
	if res.StatusCode >= 300 {
		return tokens{}, fail(CodeNetwork, fmt.Errorf("token endpoint HTTP %d", res.StatusCode))
	}
	var tok tokens
	if err := json.NewDecoder(res.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		return tokens{}, fail(CodeSignIn, fmt.Errorf("token response without an access token"))
	}
	return tok, nil
}

// credentialFrom reads the ChatGPT account from the ID token and the
// access token, which carry the same claims. Neither signature is verified: they came from the token endpoint over TLS, and the
// backend checks the access token itself.
func credentialFrom(tok tokens) (Credential, error) {
	if tok.RefreshToken == "" {
		return Credential{}, fail(CodeSignIn, fmt.Errorf("no refresh token"))
	}
	c := Credential{Provider: ChatGPT, RefreshToken: tok.RefreshToken}
	for _, jwt := range []string{tok.IDToken, tok.AccessToken} {
		claims := jwtClaims(jwt)
		auth, _ := claims["https://api.openai.com/auth"].(map[string]any)
		if c.AccountID == "" {
			c.AccountID, _ = auth["chatgpt_account_id"].(string)
		}
		if c.Email == "" {
			c.Email, _ = claims["email"].(string)
		}
	}
	if c.AccountID == "" {
		return Credential{}, fail(CodeNoSubscription, fmt.Errorf("no ChatGPT account in the token"))
	}
	return c, nil
}

func jwtClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil
	}
	var claims map[string]any
	if json.Unmarshal(raw, &claims) != nil {
		return nil
	}
	return claims
}
