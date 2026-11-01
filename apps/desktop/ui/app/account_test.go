package app

import (
	"testing"

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

	state.Preferences.UILanguage = runtime.LanguageChinese
	tt = openAccount(t, store)
	snapshot(t, tt, "settings-account-portal-zh")

	state.Account.Portal = ""
	settle(tt)
	expect(t, tt, false, store.Copy().Text(copy.AccPortalGroup))
}

// Settings › Account changes the answer with a switch while signed in.
func TestSettingsAccountUsageSharingSwitch(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = locale
		messages := store.Copy()
		tt := openAccount(t, store)
		expect(t, tt, false, messages.Text(copy.AccUsageShare))
		signedIn(state, false)
		state.Account.UsageSharing = runtime.UsageNotShared
		tt = openAccount(t, store)
		expect(t, tt, true, messages.Text(copy.AccUsageShare), messages.Text(copy.AccUsageShareDetail))
		snapshot(t, tt, "settings-account-usage-sharing-"+string(locale))
		// The switch sits at the row's trailing edge, level with its title.
		toggle := func() {
			t.Helper()
			title, ok := tt.Find(messages.Text(copy.AccUsageShare))
			if !ok {
				t.Fatalf("no usage sharing row: %q", tt.Texts())
			}
			tt.ClickAt(SettingsWidth-60, title.Y+title.H)
			settle(tt)
		}
		toggle()
		if !actions.called("usage:share") {
			t.Fatalf("calls = %v, want usage:share", actions.calls)
		}
		actions.calls = nil
		state.Account.UsageSharing = runtime.UsageShared
		settle(tt)
		toggle()
		if !actions.called("usage:private") {
			t.Fatalf("calls = %v, want usage:private", actions.calls)
		}
	}
}

func TestSettingsAccountTerms(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		for _, issuer := range []string{"standard", "custom"} {
			t.Run(string(locale)+"/"+issuer, func(t *testing.T) {
				state, actions, store := fixture()
				if issuer == "custom" {
					state.Account.Portal = ""
				}
				state.Preferences.UILanguage = runtime.UILanguage(locale)
				tt := openAccount(t, store)
				messages := store.Copy()
				click := clicker(t, tt)
				expect(t, tt, true, messages.Text(copy.AccTerms), messages.Text(copy.AccPrivacy))
				click(messages.Text(copy.AccTerms))
				click(messages.Text(copy.AccPrivacy))
				origin := config.SiteURL
				if locale == copy.SimplifiedChinese {
					origin += "/zh"
				}
				urls := tt.OpenedURLs()
				if len(urls) != 2 || urls[0] != origin+config.TermsPath || urls[1] != origin+config.PrivacyPath || actions.called("sign-in") {
					t.Fatalf("terms links: %v", urls)
				}
				click(messages.Text(copy.AccSignIn))
				if !actions.called("sign-in") || actions.called("sign-in:share") {
					t.Fatal("sign-in implied usage consent")
				}
				snapshot(t, tt, "settings-account-terms-"+string(locale))
			})
		}
	}
}
