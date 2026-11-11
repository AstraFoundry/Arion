package app

import (
	"testing"

	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestOnboardingFlow(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = runtime.UILanguage(locale)
			state.Permissions = runtime.Permissions{Microphone: runtime.ConsentUndetermined, Accessibility: runtime.ConsentUndetermined}
			state.Model.State = runtime.ModelAwaitingDownload
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			messages := store.Copy()
			click := func(key copy.Key) { t.Helper(); clicker(t, tt)(messages.Text(key)) }
			expect(t, tt, true, messages.Text(copy.OnbWelcomeTitle), messages.Text(copy.OnbAccountTitle), messages.Text(copy.AccContinueGoogle), messages.Text(copy.AccTerms), messages.Text(copy.OnbContinueLocally))
			snapshot(t, tt, "onboarding-welcome-"+string(locale))
			click(copy.AccTermsAccept)
			click(copy.AccUsageAgree)
			click(copy.OnbContinueLocally)
			if actions.called("sign-in") || actions.called("sign-in:share") {
				t.Fatal("continuing locally started sign-in")
			}
			expect(t, tt, true, messages.Text(copy.OnbModelTitle), messages.DownloadSize(state.Model.Active.SizeBytes))
			if actions.called("retry") {
				t.Fatal("opening setup downloaded a model")
			}
			download, ok := tt.Find(messages.Text(copy.OnbDownload))
			title, titleOK := tt.Find(messages.Text(copy.ModelNameGeneric))
			if !ok || !titleOK || download.W >= 180 || download.X < title.X+200 || download.Y > title.Y+title.H {
				t.Fatal("download action must stay compact at the model header's trailing edge")
			}
			snapshot(t, tt, "onboarding-download-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbModelTitle))
			click(copy.OnbDownload)
			if !actions.called("retry") {
				t.Fatal("download confirmation did not start provisioning")
			}
			state.Model.State = runtime.ModelProvisioning
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbPolishTitle), messages.Text(copy.PolishOff), messages.Text(copy.PolishEngineLocal), messages.Text(copy.PolishEngineCloud))
			snapshot(t, tt, "onboarding-polish-"+string(locale))
			click(copy.PolishEngineLocal)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbPolishTitle))
			if actions.called("download-polish") {
				t.Fatal("selecting local polish downloaded it")
			}
			click(copy.PolishOff)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbPermissionsTitle), messages.Text(copy.MicPermission), messages.Text(copy.AXPermission), messages.Text(copy.Allow))
			expect(t, tt, false, messages.Text(copy.StatusUndetermined))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbPermissionsTitle))
			click(copy.Allow)
			if !actions.called("request-mic") {
				t.Fatal("microphone not requested")
			}
			state.Permissions.Microphone = runtime.ConsentGranted
			settle(tt)
			expect(t, tt, true, messages.Text(copy.MicPermission), messages.Text(copy.StatusGranted), messages.Text(copy.AXPermission), messages.Text(copy.Allow))
			expect(t, tt, false, messages.Text(copy.StatusUndetermined))
			snapshot(t, tt, "onboarding-permissions-"+string(locale))
			click(copy.Allow)
			if !actions.called("request-ax") {
				t.Fatal("accessibility not requested")
			}
			state.Permissions.Accessibility = runtime.ConsentGranted
			settle(tt)
			expect(t, tt, true, messages.Text(copy.MicPermission), messages.Text(copy.AXPermission), messages.Text(copy.StatusGranted))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbDictateTitle), messages.Text(copy.PolishLabel), messages.Text(copy.HoldKeyEdit))
			expect(t, tt, false, messages.Text(copy.TranslateGroup), messages.Text(copy.OnbTryPlaceholder))
			snapshot(t, tt, "onboarding-practice-downloading-"+string(locale))
			if state.Preferences.HoldKey == "" {
				click(copy.OnbFinish)
				if actions.called("complete") {
					t.Fatal("finished before dictation could run")
				}
				state.Model.State = runtime.ModelReady
				state.Hold = runtime.Hold{Active: true}
				settle(tt)
				click(copy.OnbFinish)
				if !actions.called("complete") {
					t.Fatal("onboarding not completed")
				}
				return
			}
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbDictateTitle))
			state.Model.State = runtime.ModelFailed
			settle(tt)
			click(copy.Retry)
			state.Model.State = runtime.ModelReady
			state.Hold = runtime.Hold{Failure: runtime.CodePermissionDenied}
			settle(tt)
			snapshot(t, tt, "onboarding-practice-permission-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbDictateTitle))
			state.Hold = runtime.Hold{Active: true}
			settle(tt)
			expect(t, tt, true, messages.Text(copy.OnbTryPlaceholder))
			snapshot(t, tt, "onboarding-practice-"+string(locale))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbTranslateTitle), messages.Text(copy.TranslateGroup))
			expect(t, tt, false, messages.Text(copy.TranslateKeyEdit))
			click(copy.TranslateEnabled)
			expect(t, tt, true, messages.Text(copy.TranslateKeyEdit), messages.Text(copy.TranslateLanguageLabel), messages.Text(copy.OnbTryPlaceholder))
			expect(t, tt, false, messages.Text(copy.OnbFinish), messages.Text(copy.OnbDoneBody))
			state.Dictation.State = runtime.StateFinalizing
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbTranslateTitle))
			state.Dictation.State = runtime.StateCompleted
			settle(tt)
			snapshot(t, tt, "onboarding-translate-"+string(locale))
			state.Permissions.Microphone = runtime.ConsentDenied
			settle(tt)
			expect(t, tt, true, messages.Text(copy.OnbTranslateTitle), messages.Text(copy.OpenSettings), messages.Text(copy.MicPermission))
			state.Permissions.Microphone = runtime.ConsentGranted
			settle(tt)
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbSelectionTitle), messages.Text(copy.SelectionEnabled), messages.Text(copy.OnbDoneBody), messages.Text(copy.OnbFinish))
			snapshot(t, tt, "onboarding-complete-"+string(locale))
			click(copy.OnbFinish)
			if state.Preferences.Selection.Enabled {
				t.Fatal("finishing turned selection actions on")
			}
			if !actions.called("complete") {
				t.Fatal("onboarding not completed")
			}
			if !actions.called("check-stored") {
				t.Fatal("welcome page did not look for a stored sign-in")
			}
			for _, call := range []string{"sign-in", "sign-in:share", "install-ime", "download-polish"} {
				if actions.called(call) {
					t.Fatalf("setup triggered %s", call)
				}
			}
		})
	}
}

func TestOnboardingCloudStepAndCustomIssuer(t *testing.T) {
	state, actions, store := fixture()
	state.Account.Portal = ""
	state.Model.State = runtime.ModelReady
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	messages := store.Copy()
	click := func(key copy.Key) { t.Helper(); clicker(t, tt)(messages.Text(key)) }
	expect(t, tt, true, messages.Text(copy.OnbWelcomeTitle), messages.Text(copy.AccSignIn), messages.Text(copy.AccTerms))
	expect(t, tt, false, messages.Text(copy.AccContinueGoogle))
	click(copy.OnbContinueLocally)
	if actions.called("sign-in") {
		t.Fatal("custom issuer local continuation started sign-in")
	}
	click(copy.Continue)
	click(copy.PolishEngineCloud)
	click(copy.Continue)
	expect(t, tt, true, messages.Text(copy.CloudTitle), messages.Text(copy.Skip))
	expect(t, tt, false, messages.Text(copy.OnbPolishTitle))
	click(copy.Back)
	expect(t, tt, true, messages.Text(copy.OnbPolishTitle))
	click(copy.Continue)
	expect(t, tt, true, messages.Text(copy.CloudTitle))
	name := config.CloudProviders[runtime.CloudOpenAICompatible]
	for i := 0; i < 12; i++ {
		if r, ok := tt.Find(name); ok && r.Y >= 0 && r.Y+r.H <= float32(OnboardingHeight)-100 {
			break
		}
		tt.Scroll(float32(OnboardingWidth)*0.7, float32(OnboardingHeight)/2, 0, 80)
		settle(tt)
	}
	if err := tt.Click(name); err != nil {
		t.Fatalf("compatible provider: %v %q", err, tt.Texts())
	}
	settle(tt)
	expect(t, tt, true, messages.Text(copy.CloudBaseURLLabel), messages.Text(copy.CloudKeyLabel))
	click(copy.Back)
	click(copy.Skip)
	click(copy.Continue)
	expect(t, tt, true, messages.Text(copy.OnbDictateTitle), messages.Text(copy.PolishLabel))
	expect(t, tt, false, messages.Text(copy.TranslateGroup))
}

// PRV-A11: both issuer layouts require terms and only explicit selection shares.
func TestOnboardingConsent(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, custom := range []bool{false, true} {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			if custom {
				state.Account.Portal = ""
			}
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			messages := store.Copy()
			click := func(key copy.Key) { t.Helper(); clicker(t, tt)(messages.Text(key)) }
			method := copy.AccContinueGoogle
			if custom {
				method = copy.AccSignIn
			}
			click(method)
			if !actions.called("sign-in") || actions.called("sign-in:share") {
				t.Fatal("default sign-in shared usage")
			}
			terms, termsOK := tt.Find(messages.Text(copy.AccTerms))
			privacy, privacyOK := tt.Find(messages.Text(copy.AccPrivacy))
			usage, usageOK := tt.Find(messages.Text(copy.AccUsageData))
			if !termsOK || !privacyOK || !usageOK || terms.Y != privacy.Y || usage.Y <= terms.Y || privacy.X+privacy.W > OnboardingWidth-onboarding.Inset {
				t.Fatal("consent keywords must remain inline within their separate single-line rows")
			}
			click(copy.AccTerms)
			click(copy.AccPrivacy)
			click(copy.AccUsageData)
			origin := config.SiteURL
			if locale == runtime.LanguageChinese {
				origin += "/zh"
			}
			urls := tt.OpenedURLs()
			if len(urls) != 3 || urls[0] != origin+config.TermsPath || urls[1] != origin+config.PrivacyPath || urls[2] != origin+config.UsageDataPath {
				t.Fatalf("inline consent links = %v", urls)
			}
			actions.calls = nil
			click(method)
			if !actions.called("sign-in") || actions.called("sign-in:share") {
				t.Fatal("following a document link changed a consent checkbox")
			}
			actions.calls = nil
			click(copy.AccTermsAccept)
			click(method)
			if !custom {
				click(copy.AccContinueGitHub)
				click(copy.AccContinueEmail)
			}
			if actions.called("sign-in") || actions.called("sign-in:share") {
				t.Fatal("sign-in ignored unchecked terms")
			}
			click(copy.AccUsageAgree)
			click(copy.AccTermsAccept)
			click(method)
			if !actions.called("sign-in:share") {
				t.Fatal("explicit usage choice was not passed to sign-in")
			}
			state.Account.Phase = runtime.AccountSigningIn
			settle(tt)
			expect(t, tt, true, messages.Text(copy.AccCancel))
			expect(t, tt, false, messages.Text(copy.AccUsageAgree))
			signedIn(state, false)
			settle(tt)
			expect(t, tt, false, messages.Text(copy.AccUsageShare), messages.Text(copy.AccUsageAgree))
			click(copy.Continue)
			expect(t, tt, true, messages.Text(copy.OnbModelTitle))
		}
	}
}
