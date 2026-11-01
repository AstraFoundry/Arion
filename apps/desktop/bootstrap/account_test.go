package bootstrap

import (
	"errors"
	"path/filepath"
	"testing"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/settings"
	"arion.local/desktop/internal/account"
	arion "arion.local/desktop/ui/shared/runtime"
)

// A credential-store read lands only while it is still the current check;
// a failed read is an error, not "nothing stored".
func TestFinishCredential(t *testing.T) {
	unreadable := account.Wrap(account.CodeStore, "read", errors.New("keychain locked"))
	for _, tc := range []struct {
		name   string
		stale  bool
		stored bool
		err    error
		want   arion.AccountCredential
		code   string
	}{
		{"stored", false, true, nil, arion.CredentialStored, ""},
		{"absent", false, false, nil, arion.CredentialAbsent, ""},
		{"unreadable", false, false, unreadable, arion.CredentialUnreadable, string(account.CodeStore)},
		{"stale", true, true, nil, arion.CredentialUnknown, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{}
			a.state.Account = arion.Account{Phase: arion.AccountSignedOut, Credential: arion.CredentialChecking}
			seq := a.accountGeneration()
			if tc.stale {
				a.beginAccount()
			}
			a.finishCredential(seq, tc.stored, tc.err)
			if got := a.state.Account; got.Credential != tc.want || got.ErrorCode != tc.code {
				t.Fatalf("account = %+v", got)
			}
		})
	}
}

func TestApplySessionRemembersAccountDisplay(t *testing.T) {
	a := &App{prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
	a.state.Preferences = application.DefaultPreferences("darwin")
	a.applySession(account.Session{Issuer: account.DefaultIssuer, Subject: "user-1", Profile: account.Profile{Name: "Ada", Email: "ada@example.com"}}, arion.AccountSignedIn)
	if a.state.Account.Name != "Ada" || a.state.Preferences.AccountSubject != "user-1" || a.state.Preferences.AccountName != "Ada" {
		t.Fatalf("session not remembered: %+v %+v", a.state.Account, a.state.Preferences)
	}
	a.state.Account.Name = ""
	a.state.Account.Email = ""
	a.applySession(account.Session{Issuer: account.DefaultIssuer, Subject: "user-1"}, arion.AccountSignedIn)
	if a.state.Account.Name != "Ada" || a.state.Account.Email != "ada@example.com" {
		t.Fatalf("restart lost display: %+v", a.state.Account)
	}
	a.applySession(account.Session{Issuer: account.DefaultIssuer, Subject: "user-2"}, arion.AccountSignedIn)
	if a.state.Account.Name != "" || a.state.Preferences.AccountName != "" {
		t.Fatalf("other subject kept Ada: %+v %+v", a.state.Account, a.state.Preferences)
	}
}

// Usage sharing is the account's own answer: another account signing in
// starts unknown.
func TestApplySessionForgetsAnotherAccountsUsageAnswer(t *testing.T) {
	a := &App{prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
	a.state.Preferences = application.DefaultPreferences("darwin")
	a.applySession(account.Session{Issuer: account.DefaultIssuer, Subject: "user-1"}, arion.AccountSignedIn)
	a.state.Account.UsageSharing = arion.UsageShared
	a.applySession(account.Session{Issuer: account.DefaultIssuer, Subject: "user-1"}, arion.AccountSignedIn)
	if a.state.Account.UsageSharing != arion.UsageShared {
		t.Fatalf("a refresh of the same account lost its answer: %+v", a.state.Account)
	}
	a.applySession(account.Session{Issuer: account.DefaultIssuer, Subject: "user-2"}, arion.AccountSignedIn)
	if a.state.Account.UsageSharing != arion.UsageUnknown {
		t.Fatalf("user-2 inherited user-1's answer: %+v", a.state.Account)
	}
}

func TestUsageSharingSeparatesUndecidedFromDeclined(t *testing.T) {
	for answer, want := range map[account.UsageConsent]arion.UsageSharing{
		{}:                              arion.UsageUndecided,
		{Decided: true}:                 arion.UsageNotShared,
		{Accepted: true, Decided: true}: arion.UsageShared,
	} {
		if got := usageSharingOf(answer); got != want {
			t.Fatalf("%+v = %s, want %s", answer, got, want)
		}
	}
}

// The sign-in page's box belongs to that sign-in: checked, a successful
// sign-in records consent unless the account already shares; unchecked, it
// only reads the account's answer; a failed sign-in drops it.
func TestSignInBoxRecordsConsentOnlyWhenChecked(t *testing.T) {
	svc, err := account.New(account.Config{Issuer: account.DefaultIssuer}, account.Dependencies{Store: &maintenanceStore{}, Browser: func(string) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		share, shares bool
		err           error
		saving        bool
		loading       bool
	}{
		{"checked", true, false, nil, true, false},
		{"checked, already sharing", true, true, nil, false, true},
		{"unchecked", false, false, nil, false, true},
		{"failed", true, false, account.Wrap(account.CodeAuthorization, "access_denied", nil), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{accounts: svc, prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
			a.state.Preferences = application.DefaultPreferences("darwin")
			a.shareOnSignIn = tc.share
			session := account.Session{Issuer: account.DefaultIssuer, Subject: "user-1", UsageConsent: tc.shares}
			if tc.err != nil {
				session = account.Session{}
			}
			a.finishAccount(session, tc.err)
			if a.shareOnSignIn {
				t.Fatal("the box outlived its sign-in")
			}
			if a.state.Account.UsageSaving != tc.saving || a.state.Account.UsageLoading != tc.loading {
				t.Fatalf("saving %v loading %v, want %v %v", a.state.Account.UsageSaving, a.state.Account.UsageLoading, tc.saving, tc.loading)
			}
		})
	}
}
