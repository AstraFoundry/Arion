package app

import (
	"strings"
	"testing"

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
	if !ok || !hasText(tt, en.Text(copy.MatchAppDetail)) {
		t.Fatalf("row missing: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-match-app")
	// The switch is centered on the row, whose detail wraps.
	detail, _ := tt.Find(en.Text(copy.MatchAppDetail))
	tt.ClickAt(560-40, (title.Y+detail.Y+detail.H)/2)
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
		if !hasText(tt, messages.Text(copy.ModelNameGeneric)) || !hasText(tt, messages.Text(copy.PolishLabel)) {
			t.Fatalf("model page lacks recognition or polish: %q", tt.Texts())
		}
		check("settings", tt.Texts())
		snapshot(t, tt, "settings-model-"+string(locale))

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
		if !hasText(ot, messages.Text(copy.OnbModelTitle)) || !hasText(ot, messages.Text(copy.PolishLabel)) {
			t.Fatalf("onboarding model step lacks polish: %q", ot.Texts())
		}
		check("onboarding", ot.Texts())
		snapshot(t, ot, "onboarding-5-model-"+string(locale))
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
	detail, _ := tt.Find(en.Text(copy.ContextHistoryDetail))
	tt.ClickAt(560-40, (title.Y+detail.Y+detail.H)/2)
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
