package app

import (
	"testing"

	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func clicker(t *testing.T, tt *ui.Tester) func(string) {
	return func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatalf("%s: %v (texts %q)", label, err, tt.Texts())
		}
		settle(tt)
	}
}

// expect checks that each text is, or is not, shown exactly.
func expect(t *testing.T, tt *ui.Tester, show bool, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if _, shown := tt.Find(text); shown != show {
			t.Fatalf("%q shown=%v, want %v: %q", text, !show, show, tt.Texts())
		}
	}
}

func signedIn(state *runtime.State, hosted bool) {
	state.Account = runtime.Account{Issuer: state.Account.Issuer, Portal: state.Account.Portal, Phase: runtime.AccountSignedIn, Subject: "sub-1", Name: "Ada Lovelace", Email: "ada@example.com", HostedSync: hosted}
}

// The account step follows the welcome. It offers Google, GitHub and email,
// which all open the browser sign-in, beside continuing locally; the
// sign-in waits on the browser with a cancel, explains failures, and once
// signed in shows the waitlist or the entitlement before moving on.
func TestOnboardingAccountStep(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	expect(t, tt, true, en.Text(copy.OnbAccountTitle), en.Text(copy.OnbContinueLocally), en.Text(copy.AccContinueGoogle),
		en.Text(copy.AccContinueGitHub), en.Text(copy.AccContinueEmail))
	expect(t, tt, false, en.Text(copy.Continue))
	snapshot(t, tt, "onboarding-account-choice")

	for _, method := range []copy.Key{copy.AccContinueGoogle, copy.AccContinueGitHub, copy.AccContinueEmail} {
		actions.calls = nil
		click(en.Text(method))
		if !actions.called("sign-in") {
			t.Fatalf("%s did not start the browser sign-in", method)
		}
	}
	state.Account.Phase = runtime.AccountSigningIn
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccWaiting), en.Text(copy.AccCancel))
	expect(t, tt, false, en.Text(copy.AccContinueGoogle))
	snapshot(t, tt, "onboarding-account-waiting")
	click(en.Text(copy.AccCancel))
	if !actions.called("cancel-sign-in") {
		t.Fatal("cancel not requested")
	}
	state.Account = runtime.Account{Portal: state.Account.Portal, Phase: runtime.AccountSignedOut, ErrorCode: runtime.AccountCodeCancelled}
	settle(tt)
	expect(t, tt, true, en.Text(copy.OnbContinueLocally), en.Text(copy.AccContinueGoogle))
	expect(t, tt, false, en.Text(copy.AccErrGeneric))

	state.Account = runtime.Account{Portal: state.Account.Portal, Phase: runtime.AccountSignedOut, ErrorCode: runtime.AccountCodeAuthorization, ErrorDetail: "access_denied"}
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccErrDenied))
	snapshot(t, tt, "onboarding-account-denied")

	signedIn(state, false)
	settle(tt)
	expect(t, tt, true, "Ada Lovelace", "ada@example.com", en.Text(copy.AccWaitlistOpen), en.Text(copy.AccSelfHosted), en.Text(copy.Continue))
	expect(t, tt, false, en.Text(copy.AccErrDenied), en.Text(copy.OnbContinueLocally))
	snapshot(t, tt, "onboarding-account-signed-in")
	click(en.Text(copy.AccWaitlistJoin))
	if !actions.called("join-waitlist") {
		t.Fatal("waitlist join not requested")
	}
	state.Account.Waitlist.Phase = runtime.AccountWaitlistJoinedPending
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccWaitlistPending))

	signedIn(state, true)
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccSyncAvailable), en.Text(copy.AccSyncDeviceLater))
	expect(t, tt, false, en.Text(copy.AccWaitlistJoin), en.Text(copy.AccSelfHosted))
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbMicTitle))
}

// A custom issuer's sign-in methods are unknown, so the step offers one
// sign-in instead of Google, GitHub and email.
func TestOnboardingAccountCustomIssuer(t *testing.T) {
	state, actions, store := fixture()
	state.Account = runtime.Account{Issuer: "https://accounts.example.org", Phase: runtime.AccountSignedOut, Credential: runtime.CredentialAbsent}
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	expect(t, tt, false, en.Text(copy.AccContinueGoogle))
	click(en.Text(copy.AccSignIn))
	if !actions.called("sign-in") {
		t.Fatal("sign-in not started")
	}
}

// A sign-in in flight holds the step: Continue waits until it finishes.
func TestOnboardingAccountWaitsForSignIn(t *testing.T) {
	state, _, store := fixture()
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	signedIn(state, false)
	state.Account.Phase = runtime.AccountRefreshing
	settle(tt)
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbAccountTitle))
	state.Account.Phase = runtime.AccountSignedIn
	settle(tt)
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbMicTitle))
}

func openAccount(t *testing.T, store runtime.Store) *ui.Tester {
	t.Helper()
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	// The sidebar's account card names the signed-in account.
	label := store.Copy().Text(copy.NavAccount)
	if store.State.Account.SignedIn() {
		label = store.State.Account.Name
	}
	clicker(t, tt)(label)
	return tt
}

func TestSettingsAccountSignedOut(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	tt := openAccount(t, store)
	expect(t, tt, true, en.Text(copy.AccTitle), en.Text(copy.AccSignIn))
	expect(t, tt, false, en.Text(copy.AccSyncGroup))
	snapshot(t, tt, "settings-account-signed-out")
	clicker(t, tt)(en.Text(copy.AccSignIn))
	if !actions.called("sign-in") {
		t.Fatal("sign-in not started")
	}
	state.Account.Phase = runtime.AccountSigningIn
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccWaiting))
	clicker(t, tt)(en.Text(copy.AccCancel))
	if !actions.called("cancel-sign-in") {
		t.Fatal("cancel not requested")
	}
}

func TestSettingsAccountSignedIn(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	signedIn(state, false)
	tt := openAccount(t, store)
	click := clicker(t, tt)
	expect(t, tt, true, "Ada Lovelace", en.Text(copy.AccSignOut), en.Text(copy.AccSyncGroup), en.Text(copy.AccWaitlistJoin))
	snapshot(t, tt, "settings-account-signed-in")

	for _, tc := range []struct {
		phase   runtime.AccountWaitlistPhase
		status  copy.Key
		action  copy.Key
		request string
	}{
		{runtime.AccountWaitlistJoinedPending, copy.AccWaitlistPending, copy.AccCheckAgain, "refresh-account"},
		{runtime.AccountWaitlistAlreadyEntitled, copy.AccWaitlistEntitled, copy.AccCheckAgain, "refresh-account"},
		{runtime.AccountWaitlistError, copy.AccWaitlistFailed, copy.Retry, "join-waitlist"},
	} {
		actions.calls = nil
		state.Account.Waitlist = runtime.AccountWaitlist{Phase: tc.phase}
		state.Account.ErrorCode = ""
		if tc.phase == runtime.AccountWaitlistError {
			state.Account.ErrorCode = runtime.AccountCodeWaitlist
		}
		settle(tt)
		expect(t, tt, true, en.Text(tc.status), en.Text(copy.AccSelfHosted))
		click(en.Text(tc.action))
		if !actions.called(tc.request) {
			t.Fatalf("%s: %s not requested: %v", tc.phase, tc.request, actions.calls)
		}
		snapshot(t, tt, "settings-account-waitlist-"+string(tc.phase))
	}
	// The waitlist row already says the join failed; no second notice.
	failures := 0
	for _, text := range tt.Texts() {
		if text == en.Text(copy.AccWaitlistFailed) {
			failures++
		}
	}
	if failures != 1 {
		t.Fatalf("waitlist failure shown %d times", failures)
	}

	state.Account.Waitlist = runtime.AccountWaitlist{Phase: runtime.AccountWaitlistClosed}
	state.Account.ErrorCode = ""
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccWaitlistClosed))
	expect(t, tt, false, en.Text(copy.AccWaitlistJoin), en.Text(copy.Retry))

	actions.calls = nil
	state.Account.Phase = runtime.AccountSigningOut
	settle(tt)
	click(en.Text(copy.AccSignOut))
	if actions.called("sign-out") {
		t.Fatal("sign-out requested while one is running")
	}
	state.Account.Phase = runtime.AccountSignedIn
	settle(tt)
	click(en.Text(copy.AccSignOut))
	if !actions.called("sign-out") {
		t.Fatal("sign-out not requested")
	}
}

// A revocation AstraFoundry did not confirm is stated after signing out,
// in either language.
func TestSettingsAccountRevokeFailure(t *testing.T) {
	state, _, store := fixture()
	state.Account = runtime.Account{
		Phase:     runtime.AccountSignedOut,
		ErrorCode: runtime.AccountCodeRevoke, ErrorDetail: "transport",
		SignOut: runtime.AccountSignOut{Attempted: true, RefreshTokenDeleted: true, RevokeAttempted: true, RevokeDetail: "transport"},
	}
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state.Preferences.UILanguage = language
		messages := store.Copy()
		tt := openAccount(t, store)
		expect(t, tt, true, messages.Text(copy.AccRevokeUnconfirmed), messages.Text(copy.AccSignIn))
		snapshot(t, tt, "settings-account-revoke-failed-"+string(language))
	}
}

func TestChineseAccount(t *testing.T) {
	state, _, store := fixture()
	state.Preferences.UILanguage = runtime.LanguageChinese
	zh := copy.For(copy.SimplifiedChinese)
	signedIn(state, true)
	tt := openAccount(t, store)
	expect(t, tt, true, zh.Text(copy.AccSignOut), zh.Text(copy.AccSyncAvailable), zh.Text(copy.AccSyncDeviceLater))
	snapshot(t, tt, "settings-account-zh")
}

// Settings › Account reads the credential store when it shows; a stored
// sign-in offers Reconnect and Sign out instead of Sign in.
func TestSettingsAccountStoredSession(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	state.Account = runtime.Account{Phase: runtime.AccountSignedOut}
	tt := openAccount(t, store)
	if !actions.called("check-stored") {
		t.Fatal("credential store not read")
	}
	state.Account.Credential = runtime.CredentialChecking
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccChecking))
	expect(t, tt, false, en.Text(copy.AccSignIn))

	state.Account.Credential = runtime.CredentialStored
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccStoredDetail), en.Text(copy.AccReconnect), en.Text(copy.AccSignOut))
	expect(t, tt, false, en.Text(copy.AccSignIn), en.Text(copy.AccSyncGroup))
	snapshot(t, tt, "settings-account-stored")
	click := clicker(t, tt)
	click(en.Text(copy.AccReconnect))
	if !actions.called("refresh-account") {
		t.Fatal("reconnect did not refresh")
	}
	click(en.Text(copy.AccSignOut))
	if !actions.called("sign-out") {
		t.Fatal("sign-out not requested")
	}

	actions.calls = nil
	state.Account = runtime.Account{Phase: runtime.AccountSigningOut}
	settle(tt)
	click(en.Text(copy.AccSignOut))
	if actions.called("sign-out") {
		t.Fatal("sign-out requested while one is running")
	}
}

// A credential store that cannot be read is stated, not shown as signed out,
// and is not read again on every frame.
func TestSettingsAccountUnreadableStore(t *testing.T) {
	state, actions, store := fixture()
	state.Account = runtime.Account{Phase: runtime.AccountSignedOut, Credential: runtime.CredentialUnreadable,
		ErrorCode: runtime.AccountCodeStore, ErrorDetail: "read"}
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state.Preferences.UILanguage = language
		messages := store.Copy()
		actions.calls = nil
		tt := openAccount(t, store)
		expect(t, tt, true, messages.Text(copy.AccErrStore), messages.Text(copy.AccSignIn))
		if actions.called("check-stored") {
			t.Fatal("unreadable store read again")
		}
		snapshot(t, tt, "settings-account-unreadable-"+string(language))
	}
}

// The onboarding account step reads the store too, and not before; with a
// stored sign-in it offers Reconnect, and Continue moves on without
// reconnecting.
func TestOnboardingAccountStoredSession(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	if actions.called("check-stored") {
		t.Fatal("credential store read before the account step")
	}
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	if !actions.called("check-stored") {
		t.Fatal("credential store not read on the account step")
	}
	state.Account.Credential = runtime.CredentialStored
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccReconnect), en.Text(copy.AccSignOut))
	expect(t, tt, false, en.Text(copy.AccContinueGoogle))
	snapshot(t, tt, "onboarding-account-stored")
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbMicTitle))
}

func TestDeviceCenterHidesAccountSettings(t *testing.T) {
	state, actions, store := fixture()
	state.Account.Disabled = true
	page := settings.NewPage()
	page.Section = settings.Account
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	messages := copy.For(copy.English)
	expect(t, tt, false, messages.Text(copy.NavAccount), messages.Text(copy.AccSignIn))
	if actions.called("check-stored") || actions.called("sign-in") {
		t.Fatal("device center accessed account state")
	}
}

func TestCustomOIDCDoesNotOfferOfficialWaitlist(t *testing.T) {
	state, _, store := fixture()
	signedIn(state, false)
	state.Account.WaitlistDisabled = true
	page := settings.NewPage()
	page.Section = settings.Account
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	expect(t, tt, false, copy.For(copy.English).Text(copy.AccWaitlistJoin))
}

func TestAccountShowsConfiguredIssuer(t *testing.T) {
	state, _, store := fixture()
	state.Account.Issuer = "https://accounts.example.org"
	state.Account.Credential = runtime.CredentialAbsent
	page := settings.NewPage()
	page.Section = settings.Account
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	expect(t, tt, true, state.Account.Issuer)
}

// Signed in, the sidebar's foot names the account and its section links the
// AstraFoundry account portal; a custom issuer has no portal to link.
func TestSettingsAccountPortal(t *testing.T) {
	state, _, store := fixture()
	en := copy.For(copy.English)
	signedIn(state, false)
	tt := openAccount(t, store)
	expect(t, tt, true, en.Text(copy.AccPortalGroup), en.Text(copy.AccPortalProfile), en.Text(copy.AccPortalSecurity),
		en.Text(copy.AccPortalConnections), en.Text(copy.AccPortalProducts), en.Text(copy.AccPortalData))
	snapshot(t, tt, "settings-account-portal")

	state.Account.Portal = ""
	settle(tt)
	expect(t, tt, false, en.Text(copy.AccPortalGroup))
}
