// Package account maps the AstraFoundry account projection to what views
// say about it: the stage to draw, the profile line and the copy for each
// failure.
package account

import (
	"strings"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

// Stage is which account controls a view draws.
type Stage int

const (
	// SignedOut offers sign-in.
	SignedOut Stage = iota
	// Waiting is the browser sign-in in flight, which the user may cancel.
	Waiting
	// Checking is a refresh without a profile to show yet, or the
	// credential-store read.
	Checking
	// Stored is signed out with a refresh token kept from an earlier
	// session: restore automatically, or sign out to revoke and delete it.
	Stored
	// Profile is a session: profile, sign-out and hosted sync.
	Profile
)

func StageOf(a runtime.Account) Stage {
	switch {
	case a.Phase == runtime.AccountSigningIn:
		return Waiting
	case a.SignedIn():
		return Profile
	case a.Phase == runtime.AccountRefreshing:
		return Checking
	case a.Phase == runtime.AccountSigningOut:
		return Stored
	case a.Credential == runtime.CredentialChecking:
		return Checking
	case a.Credential == runtime.CredentialStored:
		return Stored
	}
	return SignedOut
}

// NeedsCredentialCheck reports a signed-out account whose credential store
// has not been read since the last account change.
func NeedsCredentialCheck(a runtime.Account) bool {
	return !a.Disabled && a.Phase == runtime.AccountSignedOut && a.Credential == runtime.CredentialUnknown
}

// Identity is the profile line: the name over the email, or the email
// alone. Without either, the title says the account is signed in.
func Identity(a runtime.Account) (title string, detail string, key copy.Key) {
	name, email := strings.TrimSpace(a.Name), strings.TrimSpace(a.Email)
	switch {
	case name != "" && email != "" && name != email:
		return name, email, ""
	case name != "":
		return name, "", ""
	case email != "":
		return email, "", ""
	}
	return "", "", copy.AccSignedIn
}

// PortalPage is a page of the AstraFoundry account portal, below its
// /account root.
type PortalPage struct {
	Path  string
	Title copy.Key
}

// PortalPages are the portal's account pages, in its own order.
var PortalPages = []PortalPage{
	{"/profile", copy.AccPortalProfile},
	{"/security", copy.AccPortalSecurity},
	{"/connections", copy.AccPortalConnections},
	{"/products", copy.AccPortalProducts},
	{"/data", copy.AccPortalData},
}

// ErrorMessage is the copy for an ErrorCode, refined by ErrorDetail. A
// cancelled sign-in has none: the user just asked for it.
func ErrorMessage(code, detail string) copy.Key {
	switch code {
	case "", runtime.AccountCodeCancelled:
		return ""
	case runtime.AccountCodeInProgress:
		return copy.AccErrInProgress
	case runtime.AccountCodeTimeout:
		return copy.AccErrTimeout
	case runtime.AccountCodeStateMismatch:
		return copy.AccErrState
	case runtime.AccountCodeAuthorization:
		if detail == "access_denied" {
			return copy.AccErrDenied
		}
		return copy.AccErrAuthorization
	case runtime.AccountCodeToken:
		return copy.AccErrToken
	case runtime.AccountCodeDiscovery:
		if detail == "transport" || detail == "request" || strings.HasPrefix(detail, "http_") {
			return copy.AccErrDiscovery
		}
		return copy.AccErrIssuer
	case runtime.AccountCodeStore:
		return copy.AccErrStore
	case runtime.AccountCodeNotSignedIn:
		return copy.AccErrNotSignedIn
	case runtime.AccountCodeRefresh:
		if detail == "invalid_grant" || detail == "not_found" {
			return copy.AccErrSessionEnded
		}
		return copy.AccErrRefresh
	case runtime.AccountCodeRevoke:
		return copy.AccRevokeUnconfirmed
	case runtime.AccountCodeWaitlist:
		return copy.AccWaitlistFailed
	case runtime.AccountCodeEntitlement:
		return copy.AccErrEntitlement
	case runtime.AccountCodeBrowser:
		if detail == "open" || detail == "missing" {
			return copy.AccErrBrowser
		}
		return copy.AccErrLoopback
	}
	return copy.AccErrGeneric
}

// Notice is the one account message a view shows above its controls. A
// sign-out whose revocation was not confirmed says so even without an error
// code.
func Notice(a runtime.Account) copy.Key {
	if key := ErrorMessage(a.ErrorCode, a.ErrorDetail); key != "" {
		return key
	}
	if a.SignOut.RevokeAttempted && !a.SignOut.Revoked {
		return copy.AccRevokeUnconfirmed
	}
	return ""
}
