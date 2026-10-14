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
