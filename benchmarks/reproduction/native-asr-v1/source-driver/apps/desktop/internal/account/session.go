package account

import "time"

// Profile is display data taken from the ID token without signature checks.
type Profile struct {
	Subject string
	Name    string
	Email   string
	Picture string
	Locale  string
}

// Session is the in-memory result of sign-in or refresh. The refresh token
// is not here; it lives only in RefreshTokenStore.
type Session struct {
	Entitlement  string
	Issuer       string
	Subject      string
	AccessToken  string
	TokenType    string
	Expiry       time.Time
	Entitlements []string
	Profile      Profile
}

// HasHostedSync checks the configured entitlement; legacy sessions use HostedSyncEntitlement.
func (s Session) HasHostedSync() bool {
	entitlement := s.Entitlement
	if entitlement == "" {
		entitlement = HostedSyncEntitlement
	}
	for _, key := range s.Entitlements {
		if key == entitlement {
			return true
		}
	}
	return false
}

// SignOutResult is the explicit outcome of SignOut. A failed revocation is
// RevokeError and is also returned from SignOut; it is not dropped.
type SignOutResult struct {
	RefreshTokenDeleted bool
	RevokeAttempted     bool
	Revoked             bool
	RevokeError         error
}

type tokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

func sessionFromTokens(issuer string, tok tokenSet, now time.Time) (Session, error) {
	if tok.AccessToken == "" {
		return Session{}, &Error{Code: CodeToken, Detail: "missing_access_token"}
	}
	if stringsContainTokenBreak(tok.AccessToken) {
		return Session{}, &Error{Code: CodeToken, Detail: "access_token"}
	}
	claims, err := decodeClaims(tok.AccessToken)
	if err != nil {
		return Session{}, err
	}
	subject := stringClaim(claims, "sub")
	if subject == "" {
		return Session{}, &Error{Code: CodeEntitlement, Detail: "missing_sub"}
	}
	ents, err := entitlements(claims)
	if err != nil {
		return Session{}, err
	}
	session := Session{
		Issuer:       issuer,
		Subject:      subject,
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		Entitlements: ents,
		Profile:      Profile{Subject: subject},
	}
	if tok.TokenType == "" {
		session.TokenType = "Bearer"
	}
	if tok.ExpiresIn > 0 {
		session.Expiry = now.Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	if tok.IDToken != "" {
		if idClaims, err := decodeClaims(tok.IDToken); err == nil {
			session.Profile.Name = stringClaim(idClaims, "name")
			session.Profile.Email = stringClaim(idClaims, "email")
			session.Profile.Picture = stringClaim(idClaims, "picture")
			session.Profile.Locale = stringClaim(idClaims, "locale")
		}
	}
	return session, nil
}

func stringsContainTokenBreak(token string) bool {
	for _, r := range token {
		if r == ' ' || r == '\r' || r == '\n' || r == '\t' {
			return true
		}
	}
	return false
}
