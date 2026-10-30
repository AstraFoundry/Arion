package app

import (
	"testing"

	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
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
// signed in shows the profile, plus hosted sync when the account has it.
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
	expect(t, tt, true, "Ada Lovelace", "ada@example.com", en.Text(copy.Continue))
	expect(t, tt, false, en.Text(copy.AccErrDenied), en.Text(copy.OnbContinueLocally), en.Text(copy.AccSyncGroup), en.Text(copy.AccSyncAvailable))
	snapshot(t, tt, "onboarding-account-signed-in")

	signedIn(state, true)
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccSyncAvailable), en.Text(copy.AccSyncDeviceLater))
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
	expect(t, tt, true, "Ada Lovelace", en.Text(copy.AccSignOut))
	expect(t, tt, false, en.Text(copy.AccSyncGroup), en.Text(copy.AccSyncAvailable))
	snapshot(t, tt, "settings-account-signed-in")

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
// sign-in restores automatically and still offers Sign out.
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
	expect(t, tt, true, en.Text(copy.AccStoredDetail), en.Text(copy.AccSignOut))
	expect(t, tt, false, en.Text(copy.AccSignIn), en.Text(copy.AccSyncGroup), en.Text(copy.AccReconnect))
	snapshot(t, tt, "settings-account-stored")
	click := clicker(t, tt)
	expect(t, tt, false, en.Text(copy.AccReconnect))
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

// The onboarding account step shares automatic restoration.
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
	expect(t, tt, true, en.Text(copy.AccSignOut))
	expect(t, tt, false, en.Text(copy.AccReconnect))
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
	en := copy.For(copy.English)
	expect(t, tt, false, en.Text(copy.AccSyncGroup), en.Text(copy.AccSyncAvailable))
	state.Account.HostedSync = true
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccSyncAvailable))
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

// Signed in, Settings says whether consented usage is being reported. Off,
// it links the portal's data page and checks again for a new token; on, it
// only links the page. A custom issuer keeps the status without a link.
func TestSettingsAccountUsageReporting(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	signedIn(state, false)
	tt := openAccount(t, store)
	click := clicker(t, tt)
	expect(t, tt, true, en.Text(copy.AccUsageGroup), en.Text(copy.AccUsageOff), en.Text(copy.AccUsageOffDetail), en.Text(copy.AccUsageManage))
	snapshot(t, tt, "settings-account-usage-off")
	click(en.Text(copy.AccCheckAgain))
	if !actions.called("refresh-account") {
		t.Fatalf("check again did not refresh: %v", actions.calls)
	}
	click(en.Text(copy.AccUsageManage))
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != "https://astrafoundry.si/account/data" {
		t.Fatalf("opened %q", got)
	}

	state.Account.UsageReporting = true
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccUsageOn), en.Text(copy.AccUsageOnDetail), en.Text(copy.AccUsageManage))
	expect(t, tt, false, en.Text(copy.AccUsageOff), en.Text(copy.AccCheckAgain))
	snapshot(t, tt, "settings-account-usage-on")

	state.Account.Portal = ""
	settle(tt)
	expect(t, tt, true, en.Text(copy.AccUsageOn))
	expect(t, tt, false, en.Text(copy.AccUsageManage))

	state.Account.Portal = "https://astrafoundry.si/account"
	state.Account.UsageReporting = false
	state.Preferences.UILanguage = runtime.LanguageChinese
	zh := copy.For(copy.SimplifiedChinese)
	settle(tt)
	expect(t, tt, true, zh.Text(copy.AccUsageGroup), zh.Text(copy.AccUsageOff), zh.Text(copy.AccUsageOffDetail))
	snapshot(t, tt, "settings-account-usage-off-zh")
}

// Product terms are available before authentication, including custom issuers.
func TestOnboardingTermsAndLocalContinuation(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, custom := range []bool{false, true} {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			if custom {
				state.Account.Portal = ""
			}
			messages := store.Copy()
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			click := clicker(t, tt)
			click(messages.Text(copy.OnbStart))
			expect(t, tt, true, messages.Text(copy.AccTermsNotice), messages.Text(copy.AccTerms), messages.Text(copy.AccPrivacy))
			actions.calls = nil
			click(messages.Text(copy.AccTerms))
			click(messages.Text(copy.AccPrivacy))
			origin := config.SiteURL
			if locale == runtime.LanguageChinese {
				origin += "/zh"
			}
			urls := tt.OpenedURLs()
			if len(urls) != 2 || urls[0] != origin+config.TermsPath || urls[1] != origin+config.PrivacyPath {
				t.Fatalf("unexpected legal destinations: %v", urls)
			}
			if actions.called("sign-in") {
				t.Fatal("reading terms must not start authentication")
			}
			if !custom {
				snapshot(t, tt, "onboarding-terms-"+string(locale))
			}
			click(messages.Text(copy.OnbContinueLocally))
			expect(t, tt, true, messages.Text(copy.OnbMicTitle))
		}
	}
}
