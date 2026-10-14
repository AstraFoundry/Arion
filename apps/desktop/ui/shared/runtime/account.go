package runtime

// AccountPhase is the sign-in lifecycle views read.
type AccountPhase string

const (
	AccountSignedOut  AccountPhase = "signed_out"
	AccountSigningIn  AccountPhase = "signing_in"
	AccountSignedIn   AccountPhase = "signed_in"
	AccountRefreshing AccountPhase = "refreshing"
	AccountSigningOut AccountPhase = "signing_out"
)

// AccountWaitlistPhase is the hosted-sync waitlist join views read.
type AccountWaitlistPhase string

const (
	AccountWaitlistIdle            AccountWaitlistPhase = ""
	AccountWaitlistJoining         AccountWaitlistPhase = "joining"
	AccountWaitlistJoinedPending   AccountWaitlistPhase = "joined_pending"
	AccountWaitlistAlreadyEntitled AccountWaitlistPhase = "already_entitled"
	AccountWaitlistClosed          AccountWaitlistPhase = "closed"
	AccountWaitlistError           AccountWaitlistPhase = "error"
)

// Account is the onboarding projection. It carries the profile and whether
// the access token's ent claim includes hosted sync. The access token and
// the refresh token are not here.
type Account struct {
	Phase       AccountPhase
	Subject     string
	Name        string
	Email       string
	Picture     string
	Locale      string
	HostedSync  bool
	ErrorCode   string
	ErrorDetail string
	Waitlist    AccountWaitlist
	SignOut     AccountSignOut
}

// Account.ErrorCode values views map to copy. They equal the account
// service's codes; a bootstrap test keeps the two lists identical.
const (
	AccountCodeInProgress    = "SIGN_IN_IN_PROGRESS"
	AccountCodeTimeout       = "SIGN_IN_TIMEOUT"
	AccountCodeCancelled     = "SIGN_IN_CANCELLED"
	AccountCodeStateMismatch = "STATE_MISMATCH"
	AccountCodeAuthorization = "AUTHORIZATION"
	AccountCodeToken         = "TOKEN"
	AccountCodeDiscovery     = "DISCOVERY"
	AccountCodeStore         = "SECURE_STORE"
	AccountCodeNotSignedIn   = "NOT_SIGNED_IN"
	AccountCodeRefresh       = "REFRESH"
	AccountCodeRevoke        = "REVOKE"
	AccountCodeWaitlist      = "WAITLIST"
	AccountCodeEntitlement   = "ENTITLEMENT"
	AccountCodeBrowser       = "BROWSER"
)

// AccountCodes lists every ErrorCode the service reports.
var AccountCodes = []string{
	AccountCodeInProgress, AccountCodeTimeout, AccountCodeCancelled, AccountCodeStateMismatch,
	AccountCodeAuthorization, AccountCodeToken, AccountCodeDiscovery, AccountCodeStore,
	AccountCodeNotSignedIn, AccountCodeRefresh, AccountCodeRevoke, AccountCodeWaitlist,
	AccountCodeEntitlement, AccountCodeBrowser,
}

// Busy reports a sign-in, refresh or sign-out in flight. Starting another
// one fails with SIGN_IN_IN_PROGRESS, so views wait instead.
func (a Account) Busy() bool {
	return a.Phase == AccountSigningIn || a.Phase == AccountRefreshing || a.Phase == AccountSigningOut
}

// SignedIn reports a session with a profile to show, including while it is
// refreshed or signed out.
func (a Account) SignedIn() bool {
	return a.Subject != "" && (a.Phase == AccountSignedIn || a.Phase == AccountRefreshing || a.Phase == AccountSigningOut)
}

// AccountWaitlist is the last join attempt.
type AccountWaitlist struct {
	Phase      AccountWaitlistPhase
	EntryID    string
	Code       string
	HTTPStatus int
}

// AccountSignOut is the last sign-out. A failed revocation sets RevokeDetail
// and leaves Revoked false; the refresh-token item is still removed.
type AccountSignOut struct {
	Attempted           bool
	RefreshTokenDeleted bool
	RevokeAttempted     bool
	Revoked             bool
	RevokeDetail        string
}
