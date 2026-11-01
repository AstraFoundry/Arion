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

// The answer to the usage notice belongs to one sign-in: a finished,
// failed or cancelled sign-in consumes it, so a later refresh never reuses it.
func TestUsageChoiceIsConsumedByTheSignInItBelongsTo(t *testing.T) {
	svc, err := account.New(account.Config{Issuer: account.DefaultIssuer}, account.Dependencies{Store: &maintenanceStore{}, Browser: func(string) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		session account.Session
		err     error
	}{
		{"recorded already", account.Session{Issuer: account.DefaultIssuer, Subject: "user-1", UsageConsent: true}, nil},
		{"failed", account.Session{}, account.Wrap(account.CodeAuthorization, "access_denied", nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{accounts: svc, prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
			a.state.Preferences = application.DefaultPreferences("darwin")
			agreed := true
			a.usageChoice = &agreed
			a.finishAccount(tc.session, tc.err)
			if a.usageChoice != nil {
				t.Fatal("the choice outlived its sign-in")
			}
		})
	}
}
