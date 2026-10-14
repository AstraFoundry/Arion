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
	state.Account = runtime.Account{Phase: runtime.AccountSignedIn, Subject: "sub-1", Name: "Ada Lovelace", Email: "ada@example.com", HostedSync: hosted}
}

// The sync step offers continuing locally first and signing in second; the
// sign-in waits on the browser with a cancel, explains failures, and once
// signed in shows the waitlist or the entitlement before finishing.
func TestOnboardingSyncStep(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	for range 5 {
		click(en.Text(copy.Continue))
	}
	expect(t, tt, true, en.Text(copy.OnbSyncTitle), en.Text(copy.OnbContinueLocally), en.Text(copy.OnbSignInSync), en.Text(copy.AccNoDecrypt))
	local, _ := tt.Find(en.Text(copy.OnbContinueLocally))
	signIn, _ := tt.Find(en.Text(copy.OnbSignInSync))
	if local.X >= signIn.X {
		t.Fatal("continuing locally is not the first path")
	}
	expect(t, tt, false, en.Text(copy.Continue))
	snapshot(t, tt, "onboarding-sync-choice")

	click(en.Text(copy.OnbSignInSync))
	if !actions.called("sign-in") {
		t.Fatal("sign-in not started")
	}
	state.Account.Phase = runtime.AccountSigningIn
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccWaiting), en.Text(copy.AccCancel))
	expect(t, tt, false, en.Text(copy.OnbContinueLocally))
	snapshot(t, tt, "onboarding-sync-waiting")
	click(en.Text(copy.AccCancel))
	if !actions.called("cancel-sign-in") {
		t.Fatal("cancel not requested")
	}
	state.Account = runtime.Account{Phase: runtime.AccountSignedOut, ErrorCode: runtime.AccountCodeCancelled}
	settle(tt)
	expect(t, tt, true, en.Text(copy.OnbContinueLocally))
	expect(t, tt, false, en.Text(copy.AccErrGeneric))

	state.Account = runtime.Account{Phase: runtime.AccountSignedOut, ErrorCode: runtime.AccountCodeAuthorization, ErrorDetail: "access_denied"}
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccErrDenied))
	snapshot(t, tt, "onboarding-sync-denied")

	signedIn(state, false)
	settle(tt)
	expect(t, tt, true, "Ada Lovelace", "ada@example.com", en.Text(copy.AccWaitlistOpen), en.Text(copy.AccSelfHosted), en.Text(copy.AccSignOutKeeps))
	expect(t, tt, false, en.Text(copy.AccErrDenied), en.Text(copy.AccNoDecrypt))
	snapshot(t, tt, "onboarding-sync-signed-in")
	click(en.Text(copy.AccWaitlistJoin))
	if !actions.called("join-waitlist") {
		t.Fatal("waitlist join not requested")
	}
	state.Account.Waitlist.Phase = runtime.AccountWaitlistJoinedPending
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccWaitlistPending))
	snapshot(t, tt, "onboarding-sync-pending")

	signedIn(state, true)
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccSyncAvailable), en.Text(copy.AccSyncDeviceLater))
	expect(t, tt, false, en.Text(copy.AccWaitlistJoin), en.Text(copy.AccSelfHosted))
	snapshot(t, tt, "onboarding-sync-entitled")
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbDoneTitle))
}

// A sign-in in flight holds the step: Continue waits until it finishes.
func TestOnboardingSyncWaitsForSignIn(t *testing.T) {
	state, _, store := fixture()
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	for range 5 {
		click(en.Text(copy.Continue))
	}
	signedIn(state, false)
	state.Account.Phase = runtime.AccountRefreshing
	settle(tt)
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbSyncTitle))
	state.Account.Phase = runtime.AccountSignedIn
	settle(tt)
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbDoneTitle))
}

func openAccount(t *testing.T, store runtime.Store) *ui.Tester {
	t.Helper()
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	clicker(t, tt)(store.Copy().Text(copy.NavAccount))
	return tt
}

func TestSettingsAccountSignedOut(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	tt := openAccount(t, store)
	expect(t, tt, true, en.Text(copy.AccTitle), en.Text(copy.AccNoDecrypt), en.Text(copy.AccSignIn))
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
	expect(t, tt, true, "Ada Lovelace", en.Text(copy.AccSignOut), en.Text(copy.AccSignOutKeeps), en.Text(copy.AccSyncGroup), en.Text(copy.AccWaitlistJoin))
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
	expect(t, tt, true, zh.Text(copy.AccSignOut), zh.Text(copy.AccSignOutKeeps), zh.Text(copy.AccSyncAvailable), zh.Text(copy.AccSyncDeviceLater))
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
	expect(t, tt, true, en.Text(copy.AccStoredDetail), en.Text(copy.AccReconnect), en.Text(copy.AccSignOut), en.Text(copy.AccSignOutKeeps))
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

// The onboarding sync step reads the store too; with a stored sign-in it
// offers Reconnect, and Continue moves on without reconnecting.
func TestOnboardingSyncStoredSession(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	click := clicker(t, tt)
	click(en.Text(copy.OnbStart))
	for range 4 {
		click(en.Text(copy.Continue))
	}
	if actions.called("check-stored") {
		t.Fatal("credential store read before the sync step")
	}
	state.Account = runtime.Account{Phase: runtime.AccountSignedOut}
	click(en.Text(copy.Continue))
	if !actions.called("check-stored") {
		t.Fatal("credential store not read on the sync step")
	}
	state.Account.Credential = runtime.CredentialStored
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccReconnect), en.Text(copy.AccSignOut))
	expect(t, tt, false, en.Text(copy.OnbSignInSync))
	snapshot(t, tt, "onboarding-sync-stored")
	click(en.Text(copy.Continue))
	expect(t, tt, true, en.Text(copy.OnbDoneTitle))
}
