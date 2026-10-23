package app

import (
	"strings"
	"testing"

	"arion.local/desktop/ui/features/cloud"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The cloud pages: the provider list, a provider's sign-in or masked key,
// then, connected, Disconnect and the model; failures in words.
func TestCloudPages(t *testing.T) {
	state, actions, store := fixture()
	nav := &cloud.Nav{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Padding(16).Children(func() { cloud.Pages(c, store, theme.Resolve(false), nav) })
	}, 560, 420)
	en := copy.For(copy.English)
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	settle(tt)
	for _, id := range append([]string{runtime.CloudChatGPT}, runtime.CloudKeyProviders...) {
		if !hasText(tt, config.CloudProviders[id]) {
			t.Fatalf("missing %s: %q", id, tt.Texts())
		}
	}
	if hasText(tt, en.Text(copy.CloudConnect)) || hasText(tt, en.Text(copy.CloudChatGPTSignIn)) {
		t.Fatal("list shows a provider's controls")
	}
	snapshot(t, tt, "cloud-providers")

	if err := tt.Click(config.CloudProviders[runtime.CloudChatGPT]); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if nav.Provider != runtime.CloudChatGPT {
		t.Fatalf("nav %+v", nav)
	}
	snapshot(t, tt, "cloud-chatgpt")
	if err := tt.Click(en.Text(copy.CloudChatGPTSignIn)); err != nil || !actions.called("cloud-sign-in") {
		t.Fatalf("sign in: %v", err)
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudSigningIn, Provider: runtime.CloudChatGPT}
	settle(tt)
	if err := tt.Click(en.Text(copy.AccCancel)); err != nil || !actions.called("cloud-cancel") || !hasText(tt, en.Text(copy.CloudSigningIn)) {
		t.Fatalf("waiting: %v %q", err, tt.Texts())
	}

	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	nav.Back()
	settle(tt)
	if err := tt.Click(config.CloudProviders[runtime.CloudAnthropic]); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	connect, ok := tt.Find(en.Text(copy.CloudConnect))
	if !ok {
		t.Fatalf("no connect: %q", tt.Texts())
	}
	tt.ClickAt(connect.X-80, connect.Y+connect.H/2)
	tt.Type("sk-secret")
	settle(tt)
	for _, text := range tt.Texts() {
		if strings.Contains(text, "sk-secret") {
			t.Fatalf("key shown in %q", text)
		}
	}
	snapshot(t, tt, "cloud-key")
	if err := tt.Click(en.Text(copy.CloudConnect)); err != nil || !actions.called("cloud-key:anthropic:sk-secret") || nav.Key != "" {
		t.Fatalf("connect: %v %v", err, actions.calls)
	}

	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudAnthropic}
	settle(tt)
	if !actions.called("cloud-models") {
		t.Fatal("models not listed")
	}
	state.Cloud.Models = []string{"model-a", "model-b"}
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudConnected)) || !hasText(tt, en.Text(copy.CloudModelPlaceholder)) {
		t.Fatalf("connected: %q", tt.Texts())
	}
	snapshot(t, tt, "cloud-connected")
	if err := tt.Click(en.Text(copy.CloudDisconnect)); err != nil || !actions.called("cloud-disconnect") {
		t.Fatalf("disconnect: %v", err)
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudAnthropic, Models: []string{}, Failure: runtime.CloudCodeModels}
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudFailModels)) {
		t.Fatalf("failure: %q", tt.Texts())
	}
	if err := tt.Click(en.Text(copy.Retry)); err != nil {
		t.Fatal(err)
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected, Failure: runtime.CloudCodeCancelled}
	settle(tt)
	for _, code := range runtime.CloudCodes {
		if key := cloud.FailureKey(code); key != "" && hasText(tt, en.Text(key)) {
			t.Fatalf("cancel shows %s", key)
		}
	}
}

// Settings › Model keeps its settings on one page; the recognition model
// and the cloud model each open their own pages, and back leaves them one
// page at a time. Context switches also show for the cloud, which works where
// the local runtime is not shipped.
func TestSettingsCloudModel(t *testing.T) {
	state, _, store := fixture()
	state.Polish = runtime.Polish{State: runtime.PolishUnsupported}
	state.Preferences.Polish, state.Preferences.PolishEngine = true, runtime.PolishCloud
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudDeepSeek, Models: []string{"deepseek-chat"}}
	state.Preferences.CloudModel = "deepseek-chat"
	page := settings.NewPage()
	page.Section = settings.Model
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	en := copy.For(copy.English)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatalf("%s: %v (%q)", label, err, tt.Texts())
		}
		settle(tt)
	}
	if !hasText(tt, en.Text(copy.ModelRecognitionPage)) || !hasText(tt, en.Text(copy.MatchAppLabel)) || !hasText(tt, en.Text(copy.ContextScreenLabel)) ||
		hasText(tt, en.Text(copy.PolishUnsupported)) || !hasText(tt, "deepseek-chat") || hasText(tt, en.Text(copy.CloudDisconnect)) ||
		hasText(tt, en.Text(copy.PolishCloudDetail)) || hasText(tt, en.Text(copy.ModelReason)) {
		t.Fatalf("model section: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-model-cloud")
	click(en.Text(copy.CloudTitle))
	if !hasText(tt, config.CloudProviders[runtime.CloudChatGPT]) || hasText(tt, en.Text(copy.MatchAppLabel)) {
		t.Fatalf("cloud list: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-cloud-providers")
	click(config.CloudProviders[runtime.CloudDeepSeek])
	if !hasText(tt, en.Text(copy.CloudDisconnect)) {
		t.Fatalf("provider page: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-cloud-deepseek")
	click(en.Text(copy.Back))
	click(en.Text(copy.Back))
	if !hasText(tt, en.Text(copy.MatchAppLabel)) {
		t.Fatalf("back to Model: %q", tt.Texts())
	}
	click(en.Text(copy.ModelRecognitionPage))
	if !hasText(tt, en.Text(copy.ModelReason)) {
		t.Fatalf("recognition page: %q", tt.Texts())
	}
	click(en.Text(copy.Back))
	state.Preferences.PolishEngine = runtime.PolishLocal
	settle(tt)
	if hasText(tt, en.Text(copy.MatchAppLabel)) || !hasText(tt, en.Text(copy.PolishUnsupported)) || hasText(tt, en.Text(copy.CloudTitle)) {
		t.Fatalf("local on an unsupported platform: %q", tt.Texts())
	}
}

// Onboarding asks about polishing on its own page and adds the cloud page
// only when the cloud is chosen; back leaves a provider page first.
func TestOnboardingCloudPolish(t *testing.T) {
	state, _, store := fixture()
	state.Permissions = runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted}
	state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1430 << 20}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	en := copy.For(copy.English)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatalf("%s: %v (%q)", label, err, tt.Texts())
		}
		settle(tt)
	}
	for _, label := range []copy.Key{copy.OnbStart, copy.OnbContinueLocally, copy.Continue, copy.Continue, copy.OnbIMELater, copy.Continue} {
		click(en.Text(label))
	}
	if !hasText(tt, en.Text(copy.OnbPolishTitle)) || !hasText(tt, en.DownloadSize(1430<<20)) {
		t.Fatalf("polish step: %q", tt.Texts())
	}
	click(en.Text(copy.PolishEngineCloud))
	if !state.Preferences.Polish || state.Preferences.PolishEngine != runtime.PolishCloud {
		t.Fatalf("cloud not chosen: %+v", state.Preferences)
	}
	snapshot(t, tt, "onboarding-6-polish-cloud")
	click(en.Text(copy.Continue))
	if !hasText(tt, en.Text(copy.CloudTitle)) {
		t.Fatalf("cloud step: %q", tt.Texts())
	}
	if !hasText(tt, en.Text(copy.Skip)) {
		t.Fatalf("cloud step offers no skip: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-7-cloud")
	click(config.CloudProviders[runtime.CloudChatGPT])
	if !hasText(tt, en.Text(copy.CloudChatGPTSignIn)) {
		t.Fatalf("chatgpt page: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-7-cloud-chatgpt")
	click(en.Text(copy.Back))
	if !hasText(tt, en.Text(copy.CloudTitle)) {
		t.Fatalf("back to the list: %q", tt.Texts())
	}
	click(en.Text(copy.Back))
	click(en.Text(copy.PolishOff))
	if state.Preferences.Polish {
		t.Fatal("off kept polishing on")
	}
	click(en.Text(copy.Continue))
	if !hasText(tt, en.Text(copy.OnbHoldKeyTitle)) {
		t.Fatalf("cloud step not skipped: %q", tt.Texts())
	}
}

func TestContextSettingsSharedAcrossEngines(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.Locale("zh-CN")} {
		t.Run(string(locale), func(t *testing.T) {
			state, _, store := fixture()
			state.Preferences.UILanguage = runtime.UILanguage(locale)
			state.Preferences.Polish = true
			state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
			state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudDeepSeek}
			state.Preferences.CloudModel = "deepseek-chat"
			state.Permissions.Accessibility = runtime.ConsentGranted
			state.Preferences.ContextCursor, state.Preferences.ContextScreen, state.Preferences.ContextHistory = true, true, true
			page := settings.NewPage()
			page.Section = settings.Model
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			messages := copy.For(locale)
			for _, engine := range []runtime.PolishEngine{runtime.PolishLocal, runtime.PolishCloud} {
				state.Preferences.PolishEngine = engine
				settle(tt)
				for _, key := range []copy.Key{copy.ContextCursorLabel, copy.ContextScreenLabel, copy.ContextHistoryLabel} {
					if !hasText(tt, messages.Text(key)) {
						t.Fatalf("%s missing %s", engine, key)
					}
				}
				snapshot(t, tt, "settings-context-"+string(engine)+"-"+string(locale))
			}
			// Changing a source in cloud mode changes the same preference seen locally.
			reveal(t, tt, messages.Text(copy.ContextHistoryLabel))
			title, ok := tt.Find(messages.Text(copy.ContextHistoryLabel))
			if !ok {
				t.Fatal("history row missing")
			}
			tt.ClickAt(SettingsWidth-54, title.Y+title.H/2)
			settle(tt)
			if state.Preferences.ContextHistory {
				t.Fatal("cloud history switch did not turn off")
			}
			state.Preferences.PolishEngine = runtime.PolishLocal
			settle(tt)
			if state.Preferences.ContextHistory {
				t.Fatal("engine change reset context preference")
			}
		})
	}
}
