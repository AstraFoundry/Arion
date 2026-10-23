package app

import (
	"strings"
	"testing"
	"time"

	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/features/transcript"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The polish row states its one-time download, shows progress, readiness,
// unsupported platforms and failures with a retry, in both languages.
func TestPolishRowStates(t *testing.T) {
	state, actions, store := fixture()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() { polish.Row(c, store, theme.Resolve(false)) })
	}, 560, 160)
	en := copy.For(copy.English)
	state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1430 << 20}
	settle(tt)
	if !hasText(tt, en.Text(copy.PolishLabel)) || !hasText(tt, en.DownloadSize(1430<<20)) {
		t.Fatalf("off state: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-polish-off")
	// The switch sits at the row's trailing edge, level with its title.
	title, ok := tt.Find(en.Text(copy.PolishLabel))
	if !ok {
		t.Fatal("polish title not rendered")
	}
	tt.ClickAt(560-40, title.Y+title.H/2)
	settle(tt)
	if !state.Preferences.Polish || !actions.called("prefs") {
		t.Fatal("switch did not turn polishing on")
	}
	if !hasText(tt, en.Text(copy.PolishDownload)) || actions.called("download-polish") {
		t.Fatal("enabling local polishing did not wait for download confirmation")
	}
	if err := tt.Click(en.Text(copy.PolishDownload)); err != nil || !actions.called("download-polish") {
		t.Fatalf("download confirmation: %v", err)
	}
	state.Polish = runtime.Polish{State: runtime.PolishProvisioning, Phase: runtime.PhaseDownloading, Done: 300 << 20, Total: 1430 << 20}
	settle(tt)
	if !hasText(tt, en.Progress(300<<20, 1430<<20)) {
		t.Fatalf("progress: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-polish-downloading")
	state.Polish = runtime.Polish{State: runtime.PolishFailed, Failure: runtime.CodeProvisioningFailed}
	settle(tt)
	if err := tt.Click(en.Text(copy.Retry)); err != nil || !actions.called("retry-polish") {
		t.Fatalf("failure has no working retry: %v", err)
	}
	state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
	settle(tt)
	if !hasText(tt, en.Text(copy.PolishReady)) {
		t.Fatalf("ready: %q", tt.Texts())
	}
	state.Preferences.UILanguage = runtime.LanguageChinese
	state.Polish = runtime.Polish{State: runtime.PolishUnsupported}
	settle(tt)
	if !hasText(tt, copy.For(copy.SimplifiedChinese).Text(copy.PolishUnsupported)) {
		t.Fatalf("unsupported zh: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-polish-unsupported-zh-CN")
}

func TestOnboardingLocalPolishDownload(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = locale
			state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1280835840}
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			settle(tt)
			messages := store.Copy()
			click := func(key copy.Key) {
				t.Helper()
				if err := tt.Click(messages.Text(key)); err != nil {
					t.Fatal(err)
				}
				settle(tt)
			}
			for _, key := range []copy.Key{copy.OnbStart, copy.OnbContinueLocally, copy.Continue, copy.Continue, copy.OnbIMELater, copy.Continue} {
				click(key)
			}
			click(copy.PolishEngineLocal)
			if !state.Preferences.Polish || actions.called("download-polish") || !hasText(tt, messages.Text(copy.PolishDownload)) {
				t.Fatal("selecting local polishing did not offer a separate download")
			}
			download, _ := tt.Find(messages.Text(copy.PolishDownload))
			local, _ := tt.Find(messages.Text(copy.PolishEngineLocal))
			if download.X < local.X+200 || download.Y > local.Y+local.H || download.Y+download.H < local.Y {
				t.Fatal("download action is not at the trailing edge of the local header")
			}
			if hasText(tt, messages.Text(copy.PolishReady)) {
				t.Fatal("unprovisioned local choice already shows a ready check")
			}
			snapshot(t, tt, "onboarding-polish-confirm-"+string(locale))
			click(copy.Continue)
			if !hasText(tt, messages.Text(copy.OnbPolishTitle)) {
				t.Fatal("continued before confirming the local download")
			}
			click(copy.PolishDownload)
			if !actions.called("download-polish") {
				t.Fatal("confirmation did not request the download")
			}
			state.Polish.State, state.Polish.Phase = runtime.PolishProvisioning, runtime.PhaseDownloading
			state.Polish.Done, state.Polish.Total = 4<<20, state.Polish.SizeBytes
			settle(tt)
			assertLocalStatus := func(label string) {
				t.Helper()
				progress, ok := tt.Find(label)
				cloud, cloudOK := tt.Find(messages.Text(copy.PolishEngineCloud))
				local, localOK := tt.Find(messages.Text(copy.PolishEngineLocal))
				if !ok || !cloudOK || !localOK || progress.Y < local.Y || progress.Y+progress.H > cloud.Y {
					t.Fatalf("local status appears outside its option: %q", tt.Texts())
				}
			}
			assertLocalStatus(messages.Progress(state.Polish.Done, state.Polish.Total))
			cancel, ok := tt.Find(messages.Text(copy.PolishCancelDownload))
			local, _ = tt.Find(messages.Text(copy.PolishEngineLocal))
			if !ok || cancel.X < local.X+200 || cancel.Y > local.Y+local.H || cancel.Y+cancel.H < local.Y {
				t.Fatal("cancel action is not at the trailing edge of the local header")
			}
			cancelCount := 0
			for _, text := range tt.Texts() {
				if text == messages.Text(copy.PolishCancelDownload) {
					cancelCount++
				}
			}
			if cancelCount != 1 {
				t.Fatal("cancel action is duplicated below the progress bar")
			}
			snapshot(t, tt, "onboarding-polish-download-"+string(locale))
			click(copy.PolishCancelDownload)
			rest(tt, theme.Disclosure.Duration)
			tt.Frame()
			if state.Preferences.Polish || hasText(tt, messages.Text(copy.PolishCancelDownload)) {
				t.Fatal("cancelling did not turn polishing off")
			}
			state.Polish.State = runtime.PolishOff
			click(copy.PolishEngineLocal)
			state.Polish.State, state.Polish.Failure = runtime.PolishFailed, runtime.CodeProvisioningFailed
			settle(tt)
			assertLocalStatus(messages.Text(copy.Retry))
			snapshot(t, tt, "onboarding-polish-failed-"+string(locale))
			click(copy.Retry)
			if !actions.called("retry-polish") {
				t.Fatal("download failure had no retry action")
			}
			state.Polish.State = runtime.PolishProvisioning
			settle(tt)
			click(copy.Continue)
			if !hasText(tt, messages.Text(copy.OnbHoldKeyTitle)) {
				t.Fatal("confirmed download prevented continuing setup")
			}
			click(copy.Back)
			state.Polish.State, state.Polish.Installed, state.Polish.Failure = runtime.PolishReady, true, ""
			rest(tt, theme.Disclosure.Duration)
			tt.Frame()
			assertLocalStatus(messages.Text(copy.PolishReady))
			check, _ := tt.Find(messages.Text(copy.PolishReady))
			local, _ = tt.Find(messages.Text(copy.PolishEngineLocal))
			if check.X < local.X+200 || check.Y > local.Y+local.H || check.Y+check.H < local.Y {
				t.Fatal("ready check is not at the trailing edge of the local header")
			}
			if hasText(tt, messages.Text(copy.PolishDownload)) || hasText(tt, messages.Text(copy.PolishCancelDownload)) {
				t.Fatal("installed local model still offered download controls")
			}
			snapshot(t, tt, "onboarding-polish-ready-"+string(locale))
		})
	}
}

// Match the app appears only while polishing is on, starts on, and its
// switch writes the preference.
func TestMatchAppRow(t *testing.T) {
	state, actions, store := fixture()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() { polish.MatchApp(c, store, theme.Resolve(false)) })
	}, 560, 120)
	en := copy.For(copy.English)
	state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
	state.Preferences.Polish, state.Preferences.MatchApp = false, true
	settle(tt)
	if hasText(tt, en.Text(copy.MatchAppLabel)) {
		t.Fatal("shown while polishing is off")
	}
	state.Preferences.Polish = true
	settle(tt)
	title, ok := tt.Find(en.Text(copy.MatchAppLabel))
	if !ok || hasText(tt, en.Text(copy.MatchAppDetail)) {
		t.Fatalf("row missing or its detail inline: %q", tt.Texts())
	}
	// The explanation is the hint's tooltip, shown on hover.
	tt.Move(title.X+title.W+14, title.Y+title.H/2)
	for range 40 {
		tt.Frame()
		if hasText(tt, en.Text(copy.MatchAppDetail)) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !hasText(tt, en.Text(copy.MatchAppDetail)) {
		t.Fatalf("tooltip not shown: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-match-app-tooltip")
	tt.Move(0, 0)
	tt.ClickAt(560-40, title.Y+title.H/2)
	settle(tt)
	if state.Preferences.MatchApp || !actions.called("prefs") {
		t.Fatal("switch did not turn Match the app off")
	}
	state.Polish = runtime.Polish{State: runtime.PolishUnsupported}
	settle(tt)
	if hasText(tt, en.Text(copy.MatchAppLabel)) {
		t.Fatal("shown where polishing is unsupported")
	}
}

// Punctuation is a row in the Polish card. It follows Runs on, and follows
// Cloud model when that row is shown. It accepts a choice only while the
// polish switch is on.
func TestPunctuationControl(t *testing.T) {
	state, actions, store := fixture()
	page := settings.NewPage()
	page.Section = settings.Model
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	en := copy.For(copy.English)
	state.Preferences.Punctuation = runtime.PunctuationSmart
	state.Preferences.Polish = false
	settle(tt)
	if !hasText(tt, en.Text(copy.PunctuationLabel)) || !hasText(tt, en.Text(copy.PunctuationSmart)) {
		t.Fatalf("control missing: %q", tt.Texts())
	}
	if punct, runs, cloud := firstExact(tt.Texts(), en.Text(copy.PunctuationLabel)), firstExact(tt.Texts(), en.Text(copy.PolishEngineLabel)), firstExact(tt.Texts(), en.Text(copy.CloudTitle)); punct < 0 || runs < 0 || punct < runs || cloud >= 0 {
		t.Fatalf("punctuation row is not after Runs on: punct %d runs %d cloud %d texts %q", punct, runs, cloud, tt.Texts())
	}
	snapshot(t, tt, "settings-punctuation-disabled")
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	zh := copy.For(copy.SimplifiedChinese)
	if punct, runs := firstExact(tt.Texts(), zh.Text(copy.PunctuationLabel)), firstExact(tt.Texts(), zh.Text(copy.PolishEngineLabel)); punct < 0 || runs < 0 || punct < runs || !hasText(tt, zh.Text(copy.PunctuationSmart)) {
		t.Fatalf("chinese punctuation row while polish is off: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-punctuation-disabled-zh-CN")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	reveal(t, tt, en.Text(copy.PunctuationSmart))
	if err := tt.Click(en.Text(copy.PunctuationSmart)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if hasText(tt, en.Text(copy.PunctuationFull)) || state.Preferences.Punctuation != runtime.PunctuationSmart {
		t.Fatal("punctuation changed while polish is off")
	}
	state.Preferences.Polish = true
	state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
	settle(tt)
	texts := tt.Texts()
	punct, runs, match, context := firstExact(texts, en.Text(copy.PunctuationLabel)), firstExact(texts, en.Text(copy.PolishEngineLabel)), firstExact(texts, en.Text(copy.MatchAppLabel)), firstExact(texts, en.Text(copy.ContextGroup))
	if punct < 0 || runs < 0 || match < 0 || context < 0 || !(runs < punct && punct < match && match < context) || lastExact(texts, en.Text(copy.PunctuationLabel)) > match || firstExact(texts, en.Text(copy.CloudTitle)) >= 0 {
		t.Fatalf("punctuation row order: runs %d punct %d match %d context %d texts %q", runs, punct, match, context, texts)
	}
	label, labelOK := tt.Find(en.Text(copy.PunctuationLabel))
	option, optionOK := tt.Find(en.Text(copy.PunctuationSmart))
	if !labelOK || !optionOK || option.X <= label.X+label.W {
		t.Fatalf("punctuation menu is not on the right of its label: label %v option %v", label, option)
	}
	snapshot(t, tt, "settings-punctuation-smart")
	reveal(t, tt, en.Text(copy.PunctuationSmart))
	if err := tt.Click(en.Text(copy.PunctuationSmart)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if hasText(tt, en.Text(copy.PunctuationSpaces)) || !hasText(tt, en.Text(copy.PunctuationFull)) {
		t.Fatalf("english menu: %q", tt.Texts())
	}
	if err := tt.Click(en.Text(copy.PunctuationOmit)); err != nil {
		t.Fatalf("menu did not open: %v texts %q", err, tt.Texts())
	}
	settle(tt)
	if state.Preferences.Punctuation != runtime.PunctuationOmit || !actions.called("prefs") {
		t.Fatalf("choice not saved: %q", state.Preferences.Punctuation)
	}
	snapshot(t, tt, "settings-punctuation-omit")
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if !hasText(tt, copy.For(copy.SimplifiedChinese).Text(copy.PunctuationOmit)) || hasText(tt, en.Text(copy.PunctuationOmit)) {
		t.Fatalf("chinese label: %q", tt.Texts())
	}
	zh = copy.For(copy.SimplifiedChinese)
	texts = tt.Texts()
	punct, runs, match, context = firstExact(texts, zh.Text(copy.PunctuationLabel)), firstExact(texts, zh.Text(copy.PolishEngineLabel)), firstExact(texts, zh.Text(copy.MatchAppLabel)), firstExact(texts, zh.Text(copy.ContextGroup))
	if punct < 0 || runs < 0 || match < 0 || context < 0 || !(runs < punct && punct < match && match < context) || lastExact(texts, zh.Text(copy.PunctuationLabel)) > match || firstExact(texts, zh.Text(copy.CloudTitle)) >= 0 {
		t.Fatalf("chinese punctuation row order: runs %d punct %d match %d context %d texts %q", runs, punct, match, context, texts)
	}
	snapshot(t, tt, "settings-punctuation-omit-zh-CN")
	reveal(t, tt, zh.Text(copy.PunctuationOmit))
	if err := tt.Click(zh.Text(copy.PunctuationOmit)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !hasText(tt, zh.Text(copy.PunctuationSpaces)) {
		t.Fatalf("chinese menu hides spaces: %q", tt.Texts())
	}
	if err := tt.Click(zh.Text(copy.PunctuationSpaces)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.Punctuation != runtime.PunctuationSpaces {
		t.Fatalf("chinese spaces choice not saved: %q", state.Preferences.Punctuation)
	}
	snapshot(t, tt, "settings-punctuation-spaces-zh-CN")
	saved := len(actions.calls)
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	if state.Preferences.Punctuation != runtime.PunctuationSpaces || len(actions.calls) != saved || hasText(tt, en.Text(copy.PunctuationSpaces)) {
		t.Fatalf("english showed or replaced spaces: %q calls %d", tt.Texts(), len(actions.calls)-saved)
	}
	snapshot(t, tt, "settings-punctuation-spaces-hidden-en")
	title, ok := tt.Find(en.Text(copy.PunctuationLabel))
	if !ok {
		t.Fatal("punctuation label missing")
	}
	tt.ClickAt(SettingsWidth-80, title.Y+title.H/2)
	settle(tt)
	if hasText(tt, en.Text(copy.PunctuationSpaces)) || !hasText(tt, en.Text(copy.PunctuationSmart)) || !hasText(tt, en.Text(copy.PunctuationOmit)) || state.Preferences.Punctuation != runtime.PunctuationSpaces || len(actions.calls) != saved {
		t.Fatalf("english menu while spaces is saved: %q preference %q", tt.Texts(), state.Preferences.Punctuation)
	}
	if err := tt.Click(en.Text(copy.PunctuationOmit)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.Punctuation != runtime.PunctuationOmit {
		t.Fatalf("english choice did not replace spaces: %q", state.Preferences.Punctuation)
	}
	state.Preferences.UILanguage = runtime.LanguageEnglish
	state.Preferences.PolishEngine = runtime.PolishCloud
	settle(tt)
	snapshot(t, tt, "settings-punctuation-cloud")
	texts = tt.Texts()
	punct, engine, cloud, match := firstExact(texts, en.Text(copy.PunctuationLabel)), firstExact(texts, en.Text(copy.PolishEngineLabel)), firstExact(texts, en.Text(copy.CloudTitle)), firstExact(texts, en.Text(copy.MatchAppLabel))
	if punct < 0 || engine < 0 || cloud < 0 || match < 0 || !(engine < cloud && cloud < punct && punct < match) || firstExact(texts, en.Text(copy.ContextGroup)) >= 0 {
		t.Fatalf("punctuation row is not after Cloud model: engine %d cloud %d punct %d match %d texts %q", engine, cloud, punct, match, texts)
	}
}

func firstExact(texts []string, want string) int {
	for i, text := range texts {
		if text == want {
			return i
		}
	}
	return -1
}

func lastExact(texts []string, want string) int {
	last := -1
	for i, text := range texts {
		if text == want {
			last = i
		}
	}
	return last
}

// A rewritten dictation keeps its recognized original one click away.
func TestCopyOriginal(t *testing.T) {
	state, actions, store := fixture()
	state.LastText, state.LastOriginal = "我们周四开会。", "嗯我们周三开会，不对周四。"
	copied := &transcript.Copied{}
	tt := ui.NewTester(func(c *ui.Context) { transcript.Last(c, store, theme.Resolve(false), copied) }, 320, 160)
	settle(tt)
	if err := tt.Click(copy.For(copy.English).Text(copy.CopyOriginal)); err != nil {
		t.Fatal(err)
	}
	if !actions.called("copy:" + state.LastOriginal) {
		t.Fatalf("original not copied: %v", actions.calls)
	}
	state.LastOriginal = ""
	settle(tt)
	if hasText(tt, copy.For(copy.English).Text(copy.CopyOriginal)) {
		t.Fatal("copy original shown without a rewrite")
	}
}

// Engine and model-family names stay out of the interface: the Model page
// and the onboarding model step speak in plain words in both languages.
func TestNoTechnicalModelNames(t *testing.T) {
	banned := []string{"ASR", "LLM", "Qwen", "SenseVoice", "Zipformer", "ONNX", "GenAI", "int4", "160 ms"}
	check := func(where string, texts []string) {
		t.Helper()
		for _, text := range texts {
			for _, word := range banned {
				if strings.Contains(text, word) {
					t.Fatalf("%s shows %q in %q", where, word, text)
				}
			}
		}
	}
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, _, store := fixture()
		state.Preferences.UILanguage = locale
		state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1430 << 20}
		page := settings.NewPage()
		page.Section = settings.Model
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		settle(tt)
		messages := copy.For(copy.Locale(locale))
		check("settings", tt.Texts())
		snapshot(t, tt, "settings-model-"+string(locale))
		if !hasText(tt, messages.Text(copy.PolishLabel)) || hasText(tt, messages.Text(copy.ModelReason)) {
			t.Fatalf("model section: %q", tt.Texts())
		}
		if err := tt.Click(messages.Text(copy.ModelRecognitionPage)); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if !hasText(tt, messages.Text(copy.ModelNameGeneric)) || hasText(tt, messages.Text(copy.PolishLabel)) {
			t.Fatalf("recognition page: %q", tt.Texts())
		}
		check("settings", tt.Texts())
		snapshot(t, tt, "settings-model-recognition-"+string(locale))
		if err := tt.Click(messages.Text(copy.Back)); err != nil {
			t.Fatal(err)
		}
		settle(tt)

		state.Permissions = runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted}
		onb := onboarding.NewPage()
		ot := ui.NewTester(func(c *ui.Context) { onb.Render(c, store) }, OnboardingWidth, OnboardingHeight)
		settle(ot)
		for _, label := range []copy.Key{copy.OnbStart, copy.OnbContinueLocally, copy.Continue, copy.Continue, copy.OnbIMELater} {
			if err := ot.Click(messages.Text(label)); err != nil {
				t.Fatalf("%s: %v", label, err)
			}
			settle(ot)
		}
		if !hasText(ot, messages.Text(copy.OnbModelTitle)) {
			t.Fatalf("onboarding model step: %q", ot.Texts())
		}
		check("onboarding", ot.Texts())
		snapshot(t, ot, "onboarding-5-model-"+string(locale))
		if err := ot.Click(messages.Text(copy.Continue)); err != nil {
			t.Fatal(err)
		}
		settle(ot)
		if !hasText(ot, messages.Text(copy.OnbPolishTitle)) || !hasText(ot, messages.Text(copy.PolishEngineLocal)) {
			t.Fatalf("onboarding polish step: %q", ot.Texts())
		}
		check("onboarding polish", ot.Texts())
		snapshot(t, ot, "onboarding-6-polish-"+string(locale))
	}
}

// The context switches show while polishing is on, start on, write their
// preference, and the screen switch asks for Accessibility when missing.
func TestContextRows(t *testing.T) {
	state, actions, store := fixture()
	state.Permissions.Accessibility = runtime.ConsentDenied
	state.Preferences.Polish = true
	state.Preferences.ContextCursor, state.Preferences.ContextScreen, state.Preferences.ContextHistory = true, true, true
	state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() { polish.Context(c, store, theme.Resolve(false)) })
	}, 560, 360)
	en := copy.For(copy.English)
	settle(tt)
	for _, key := range []copy.Key{copy.ContextCursorLabel, copy.ContextScreenLabel, copy.ContextHistoryLabel, copy.ContextScreenPermission} {
		if !hasText(tt, en.Text(key)) {
			t.Fatalf("missing %s: %q", key, tt.Texts())
		}
	}
	snapshot(t, tt, "settings-context")
	if err := tt.Click(en.Text(copy.Allow)); err != nil || !actions.called("request-ax") {
		t.Fatalf("permission request: %v", err)
	}
	title, _ := tt.Find(en.Text(copy.ContextHistoryLabel))
	tt.ClickAt(560-40, title.Y+title.H/2)
	settle(tt)
	if state.Preferences.ContextHistory {
		t.Fatal("history switch did not turn off")
	}
	state.Permissions.Accessibility = runtime.ConsentGranted
	settle(tt)
	if hasText(tt, en.Text(copy.ContextScreenPermission)) {
		t.Fatal("permission prompt shown after access was granted")
	}
	state.Preferences.Polish = false
	settle(tt)
	if hasText(tt, en.Text(copy.ContextCursorLabel)) {
		t.Fatal("shown with polishing off")
	}
}
