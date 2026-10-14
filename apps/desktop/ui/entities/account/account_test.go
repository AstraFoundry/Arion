package account

import (
	"testing"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

func TestStageOf(t *testing.T) {
	signedIn := runtime.Account{Phase: runtime.AccountSignedIn, Subject: "sub"}
	for _, tc := range []struct {
		account runtime.Account
		want    Stage
	}{
		{runtime.Account{}, SignedOut},
		{runtime.Account{Phase: runtime.AccountSignedOut}, SignedOut},
		{runtime.Account{Phase: runtime.AccountSigningIn}, Waiting},
		{runtime.Account{Phase: runtime.AccountRefreshing}, Checking},
		{signedIn, Profile},
		{runtime.Account{Phase: runtime.AccountRefreshing, Subject: "sub"}, Profile},
		{runtime.Account{Phase: runtime.AccountSigningOut, Subject: "sub"}, Profile},
	} {
		if got := StageOf(tc.account); got != tc.want {
			t.Errorf("StageOf(%+v) = %d, want %d", tc.account, got, tc.want)
		}
	}
}

func TestIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, email, title, detail string
		key                        copy.Key
	}{
		{"Ada", "ada@example.com", "Ada", "ada@example.com", ""},
		{"Ada", "", "Ada", "", ""},
		{"", "ada@example.com", "ada@example.com", "", ""},
		{"ada@example.com", "ada@example.com", "ada@example.com", "", ""},
		{" ", "", "", "", copy.AccSignedIn},
	} {
		title, detail, key := Identity(runtime.Account{Name: tc.name, Email: tc.email})
		if title != tc.title || detail != tc.detail || key != tc.key {
			t.Errorf("Identity(%q, %q) = %q %q %q", tc.name, tc.email, title, detail, key)
		}
	}
}

// Every code the service reports has copy in both catalogs; only a
// cancellation, which the user asked for, has none.
func TestEveryErrorCodeHasCopy(t *testing.T) {
	for _, code := range runtime.AccountCodes {
		key := ErrorMessage(code, "")
		if code == runtime.AccountCodeCancelled {
			if key != "" {
				t.Errorf("cancellation has copy %q", key)
			}
			continue
		}
		if key == "" || key == copy.AccErrGeneric {
			t.Errorf("%s has no specific copy", code)
		}
		for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
			if copy.For(locale).Text(key) == "" {
				t.Errorf("%s: %s empty", locale, key)
			}
		}
	}
	if ErrorMessage("", "") != "" || ErrorMessage("NEW_CODE", "") != copy.AccErrGeneric {
		t.Fatal("empty or unknown code mapping")
	}
}

func TestErrorDetailRefinesCopy(t *testing.T) {
	for _, tc := range []struct {
		code, detail string
		want         copy.Key
	}{
		{runtime.AccountCodeAuthorization, "access_denied", copy.AccErrDenied},
		{runtime.AccountCodeAuthorization, "server_error", copy.AccErrAuthorization},
		{runtime.AccountCodeDiscovery, "transport", copy.AccErrDiscovery},
		{runtime.AccountCodeDiscovery, "http_503", copy.AccErrDiscovery},
		{runtime.AccountCodeDiscovery, "issuer_mismatch", copy.AccErrIssuer},
		{runtime.AccountCodeDiscovery, "s256_required", copy.AccErrIssuer},
		{runtime.AccountCodeRefresh, "invalid_grant", copy.AccErrSessionEnded},
		{runtime.AccountCodeRefresh, "not_found", copy.AccErrSessionEnded},
		{runtime.AccountCodeRefresh, "http_500", copy.AccErrRefresh},
		{runtime.AccountCodeBrowser, "open", copy.AccErrBrowser},
		{runtime.AccountCodeBrowser, "listen", copy.AccErrLoopback},
		{runtime.AccountCodeRevoke, "transport", copy.AccRevokeUnconfirmed},
	} {
		if got := ErrorMessage(tc.code, tc.detail); got != tc.want {
			t.Errorf("ErrorMessage(%s, %s) = %s, want %s", tc.code, tc.detail, got, tc.want)
		}
	}
}

func TestNotice(t *testing.T) {
	for _, tc := range []struct {
		name    string
		account runtime.Account
		want    copy.Key
	}{
		{"none", runtime.Account{Phase: runtime.AccountSignedIn}, ""},
		{"cancelled", runtime.Account{ErrorCode: runtime.AccountCodeCancelled}, ""},
		{"timeout", runtime.Account{ErrorCode: runtime.AccountCodeTimeout}, copy.AccErrTimeout},
		{"waitlist row tells it", runtime.Account{ErrorCode: runtime.AccountCodeWaitlist,
			Waitlist: runtime.AccountWaitlist{Phase: runtime.AccountWaitlistError}}, ""},
		{"revoke failed", runtime.Account{ErrorCode: runtime.AccountCodeRevoke,
			SignOut: runtime.AccountSignOut{Attempted: true, RefreshTokenDeleted: true, RevokeAttempted: true}}, copy.AccRevokeUnconfirmed},
		{"revoke unconfirmed without code", runtime.Account{
			SignOut: runtime.AccountSignOut{Attempted: true, RefreshTokenDeleted: true, RevokeAttempted: true}}, copy.AccRevokeUnconfirmed},
		{"revoked", runtime.Account{
			SignOut: runtime.AccountSignOut{Attempted: true, RefreshTokenDeleted: true, RevokeAttempted: true, Revoked: true}}, ""},
		{"nothing to revoke", runtime.Account{SignOut: runtime.AccountSignOut{Attempted: true}}, ""},
	} {
		if got := Notice(tc.account); got != tc.want {
			t.Errorf("%s: Notice = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestWaitlistStatus(t *testing.T) {
	for phase, want := range map[runtime.AccountWaitlistPhase]copy.Key{
		runtime.AccountWaitlistIdle:            copy.AccWaitlistOpen,
		runtime.AccountWaitlistJoining:         copy.AccWaitlistOpen,
		runtime.AccountWaitlistJoinedPending:   copy.AccWaitlistPending,
		runtime.AccountWaitlistAlreadyEntitled: copy.AccWaitlistEntitled,
		runtime.AccountWaitlistClosed:          copy.AccWaitlistClosed,
		runtime.AccountWaitlistError:           copy.AccWaitlistFailed,
	} {
		if got := WaitlistStatus(phase); got != want {
			t.Errorf("WaitlistStatus(%q) = %s, want %s", phase, got, want)
		}
	}
}
