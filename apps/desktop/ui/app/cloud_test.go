package app

import (
	"strings"
	"testing"
	"time"

	"arion.local/desktop/ui/features/cloud"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The cloud pages: subscriptions apart from API keys; a provider's sign-in
// or masked key; once connected, the page returns to the list, where the
// provider in use has Disconnect and the model; failures in words.
func TestCloudPages(t *testing.T) {
	state, actions, store := fixture()
	nav := &cloud.Nav{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Padding(16).Children(func() { cloud.Pages(c, store, theme.Resolve(false), nav) })
	}, 560, 520)
	en := copy.For(copy.English)
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	settle(tt)
	for _, id := range append([]string{runtime.CloudChatGPT}, runtime.CloudKeyProviders...) {
		if !hasText(tt, config.CloudProviders[id]) {
			t.Fatalf("missing %s: %q", id, tt.Texts())
		}
	}
	// ChatGPT is under Subscriptions; every key provider under API keys.
	texts := tt.Texts()
	subs, keys := firstExact(texts, en.Text(copy.CloudSubscriptions)), firstExact(texts, en.Text(copy.CloudAPIKeys))
	chatgpt := firstExact(texts, config.CloudProviders[runtime.CloudChatGPT])
	if subs < 0 || keys < 0 || !(subs < chatgpt && chatgpt < keys) || hasText(tt, en.Text(copy.CloudInUse)) {
		t.Fatalf("groups: %q", texts)
	}
	for _, id := range runtime.CloudKeyProviders {
		if firstExact(texts, config.CloudProviders[id]) < keys {
			t.Fatalf("%s outside API keys: %q", id, texts)
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

	// Connected, the page returns to the list with the provider in use on
	// top: its default model, Disconnect, and a check on its row.
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudAnthropic, Saved: []string{runtime.CloudAnthropic}}
	state.Preferences.CloudModel = "claude-haiku-4-5"
	settle(tt)
	if nav.Provider != "" || !actions.called("cloud-models") {
		t.Fatalf("connected: nav %+v calls %v", nav, actions.calls)
	}
	state.Cloud.Models = []string{"claude-haiku-4-5", "claude-sonnet-5-5"}
	settle(tt)
	texts = tt.Texts()
	if inUse := firstExact(texts, en.Text(copy.CloudInUse)); inUse < 0 || inUse > firstExact(texts, en.Text(copy.CloudSubscriptions)) ||
		!hasText(tt, en.Text(copy.CloudDisconnect)) || hasText(tt, en.Text(copy.CloudModelPlaceholder)) {
		t.Fatalf("in use: %q", texts)
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

// A refused key or sign-in stays on the page that submitted it. Leaving
// that page, opening another provider, or editing the key drops it. The
// connection in use keeps its own model-list failure.
func TestCloudFailureStaysOnItsProvider(t *testing.T) {
	state, _, store := fixture()
	nav := &cloud.Nav{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Padding(16).Children(func() { cloud.Pages(c, store, theme.Resolve(false), nav) })
	}, 560, 520)
	en := copy.For(copy.English)
	refused := en.Text(copy.CloudFailRejected)
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	settle(tt)
	open := func(id string) {
		t.Helper()
		if err := tt.Click(config.CloudProviders[id]); err != nil {
			t.Fatalf("%s: %v (%q)", id, err, tt.Texts())
		}
		settle(tt)
	}
	leave := func() {
		t.Helper()
		if !nav.Back() {
			t.Fatal("nothing to leave")
		}
		settle(tt)
	}
	typeKey := func(value string) {
		t.Helper()
		connect, ok := tt.Find(en.Text(copy.CloudConnect))
		if !ok {
			t.Fatalf("no connect: %q", tt.Texts())
		}
		tt.ClickAt(connect.X-80, connect.Y+connect.H/2)
		tt.Type(value)
		settle(tt)
	}
	absent := func(where string, text string) {
		t.Helper()
		if hasText(tt, text) {
			t.Fatalf("%s shows %q in %q", where, text, tt.Texts())
		}
	}
	open(runtime.CloudAnthropic)
	typeKey("sk-secret")
	if err := tt.Click(en.Text(copy.CloudConnect)); err != nil {
		t.Fatal(err)
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected, Failure: runtime.CloudCodeRejected, FailureProvider: runtime.CloudAnthropic}
	settle(tt)
	if !hasText(tt, refused) {
		t.Fatalf("attempt: %q", tt.Texts())
	}
	snapshot(t, tt, "cloud-key-rejected")
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if !hasText(tt, copy.For(copy.SimplifiedChinese).Text(copy.CloudFailRejected)) {
		t.Fatalf("zh attempt: %q", tt.Texts())
	}
	snapshot(t, tt, "cloud-key-rejected-zh-CN")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	typeKey("sk-next")
	absent("draft", refused)
	nav.Key = ""
	settle(tt)
	absent("cleared draft", refused)

	leave()
	absent("list", refused)
	open(runtime.CloudGoogle)
	absent("google", refused)
	snapshot(t, tt, "cloud-other-provider-clear")
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	absent("google zh", copy.For(copy.SimplifiedChinese).Text(copy.CloudFailRejected))
	snapshot(t, tt, "cloud-other-provider-clear-zh-CN")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	leave()
	open(runtime.CloudAnthropic)
	absent("reopened", refused)

	leave()
	open(runtime.CloudChatGPT)
	if err := tt.Click(en.Text(copy.CloudChatGPTSignIn)); err != nil {
		t.Fatal(err)
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected, Failure: runtime.CloudCodeSignIn, FailureProvider: runtime.CloudChatGPT}
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudFailSignIn)) {
		t.Fatalf("sign-in: %q", tt.Texts())
	}
	leave()
	open(runtime.CloudAnthropic)
	absent("after sign-in", en.Text(copy.CloudFailSignIn))

	leave()
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudOpenRouter, Models: []string{"m"},
		Failure: runtime.CloudCodeRejected, FailureProvider: runtime.CloudAnthropic}
	settle(tt)
	absent("connection in use", refused)
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudOpenRouter,
		Failure: runtime.CloudCodeModels, FailureProvider: runtime.CloudOpenRouter}
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudFailModels)) {
		t.Fatalf("model list: %q", tt.Texts())
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudOpenRouter, Models: []string{"m"},
		Failure: runtime.CloudCodeNetwork, FailureProvider: runtime.CloudAnthropic}
	settle(tt)
	absent("attempt network", en.Text(copy.CloudFailNetwork))
	if hasText(tt, en.Text(copy.Retry)) {
		t.Fatalf("attempt network offers retry: %q", tt.Texts())
	}
	state.Cloud.FailureProvider = runtime.CloudOpenRouter
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudFailNetwork)) || !hasText(tt, en.Text(copy.Retry)) {
		t.Fatalf("connection network: %q", tt.Texts())
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected, Failure: runtime.CloudCodeStore}
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudFailStore)) {
		t.Fatalf("store: %q", tt.Texts())
	}
}

// Opening a saved provider's page never switches to it: only its Use
// button does, on the list or on its page.
func TestSavedCloudProviderNeedsUse(t *testing.T) {
	state, actions, store := fixture()
	nav := &cloud.Nav{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Padding(16).Children(func() { cloud.Pages(c, store, theme.Resolve(false), nav) })
	}, 560, 560)
	en := copy.For(copy.English)
	zh := copy.For(copy.SimplifiedChinese)
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudOpenRouter,
		Models: []string{"openai/gpt-6-luna"}, Saved: []string{runtime.CloudDeepSeek, runtime.CloudOpenRouter}}
	state.Preferences.CloudModel = "openai/gpt-6-luna"
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if !hasText(tt, zh.Text(copy.CloudInUse)) || !hasText(tt, zh.Text(copy.CloudUse)) || !hasText(tt, zh.Text(copy.CloudSubscriptions)) {
		t.Fatalf("chinese list: %q", tt.Texts())
	}
	snapshot(t, tt, "cloud-providers-saved-zh-CN")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	snapshot(t, tt, "cloud-providers-saved")
	activated := func() bool {
		return actions.called("cloud-activate:" + runtime.CloudDeepSeek)
	}
	if err := tt.Click(config.CloudProviders[runtime.CloudDeepSeek]); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if activated() || nav.Provider != runtime.CloudDeepSeek || !hasText(tt, en.Text(copy.CloudSaved)) || hasText(tt, en.Text(copy.CloudKeyPlaceholder)) {
		t.Fatalf("opening switched or asked for a key: %v %q", actions.calls, tt.Texts())
	}
	snapshot(t, tt, "cloud-provider-saved")
	if err := tt.Click(en.Text(copy.CloudUse)); err != nil || !activated() {
		t.Fatalf("use on the page: %v %v", err, actions.calls)
	}
	nav.Back()
	actions.calls = nil
	settle(tt)
	// The Use button on the list switches without opening the page.
	if err := tt.Click(en.Text(copy.CloudUse)); err != nil || !activated() || nav.Provider != "" {
		t.Fatalf("use on the list: %v %v %+v", err, actions.calls, nav)
	}
}

// Settings › Speech & Polish follows the text: recognition, polishing
// and where it runs, formatting, then context, the last two only while
// polishing runs. The recognition model and the cloud model each open a
// page, and back leaves them one page at a time.
func TestSettingsCloudModel(t *testing.T) {
	state, _, store := fixture()
	state.Polish = runtime.Polish{State: runtime.PolishUnsupported}
	state.Preferences.Polish, state.Preferences.PolishEngine = true, runtime.PolishCloud
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudDeepSeek, Models: []string{"deepseek-flash"}, Saved: []string{runtime.CloudDeepSeek}}
	state.Preferences.CloudModel = "deepseek-flash"
	state.Preferences.Punctuation = runtime.PunctuationSmart
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
	if !hasText(tt, en.Text(copy.NavModel)) || !hasText(tt, en.Text(copy.ModelRecognitionPage)) || !hasText(tt, en.Text(copy.MatchAppLabel)) ||
		hasText(tt, en.Text(copy.PolishUnsupported)) || !hasText(tt, "deepseek-flash") || hasText(tt, en.Text(copy.CloudDisconnect)) ||
		hasText(tt, en.Text(copy.PolishCloudDetail)) || hasText(tt, en.Text(copy.ModelReason)) {
		t.Fatalf("section: %q", tt.Texts())
	}
	texts := tt.Texts()
	order := []string{en.Text(copy.ModelRecognitionGroup), en.Text(copy.ModelPolishGroup), en.Text(copy.PolishEngineLabel), en.Text(copy.CloudTitle),
		en.Text(copy.ModelFormatGroup), en.Text(copy.PunctuationLabel), en.Text(copy.MatchAppLabel), en.Text(copy.ContextGroup)}
	for i := 1; i < len(order); i++ {
		if a, b := firstExact(texts, order[i-1]), firstExact(texts, order[i]); a < 0 || b < 0 || a > b {
			t.Fatalf("%q not before %q: %q", order[i-1], order[i], texts)
		}
	}
	snapshot(t, tt, "settings-model-cloud")
	click(en.Text(copy.CloudTitle))
	if !hasText(tt, config.CloudProviders[runtime.CloudChatGPT]) || hasText(tt, en.Text(copy.MatchAppLabel)) ||
		!hasText(tt, en.Text(copy.CloudInUse)) || !hasText(tt, en.Text(copy.CloudDisconnect)) || hasText(tt, en.Text(copy.CloudModelPlaceholder)) {
		t.Fatalf("cloud list: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-cloud-providers")
	click(config.CloudProviders[runtime.CloudOpenAI])
	if !hasText(tt, en.Text(copy.CloudConnect)) {
		t.Fatalf("provider page: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-cloud-openai")
	click(en.Text(copy.Back))
	click(en.Text(copy.Back))
	if !hasText(tt, en.Text(copy.MatchAppLabel)) {
		t.Fatalf("back to the section: %q", tt.Texts())
	}
	click(en.Text(copy.ModelRecognitionPage))
	if !hasText(tt, en.Text(copy.ModelReason)) {
		t.Fatalf("recognition page: %q", tt.Texts())
	}
	click(en.Text(copy.Back))
	// Polishing off hides formatting and context.
	state.Preferences.Polish = false
	settle(tt)
	if hasText(tt, en.Text(copy.PunctuationLabel)) || hasText(tt, en.Text(copy.MatchAppLabel)) || hasText(tt, en.Text(copy.ContextGroup)) {
		t.Fatalf("polishing off: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-model-off")
	state.Preferences.Polish, state.Preferences.PolishEngine = true, runtime.PolishLocal
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
	// Once a provider is in use, the step folds the lists behind Change and
	// continues on the provider's default model.
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudChatGPT, Email: "me@example.com",
		Models: []string{"gpt-6-luna", "gpt-6-sol"}, Saved: []string{runtime.CloudChatGPT}}
	state.Preferences.CloudModel = "gpt-6-luna"
	settle(tt)
	rest(tt, time.Second)
	if !hasText(tt, en.Text(copy.CloudInUse)) || hasText(tt, en.Text(copy.CloudAPIKeys)) || !hasText(tt, en.Text(copy.Continue)) || hasText(tt, en.Text(copy.Skip)) {
		t.Fatalf("connected step: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-7-cloud-connected")
	click(en.Text(copy.CloudChange))
	if !hasText(tt, en.Text(copy.CloudAPIKeys)) || hasText(tt, en.Text(copy.CloudDisconnect)) {
		t.Fatalf("change: %q", tt.Texts())
	}
	rest(tt, time.Second)
	snapshot(t, tt, "onboarding-7-cloud-change")
	// Back leaves Change before the step.
	click(en.Text(copy.Back))
	if !hasText(tt, en.Text(copy.CloudDisconnect)) || hasText(tt, en.Text(copy.CloudAPIKeys)) {
		t.Fatalf("back from change: %q", tt.Texts())
	}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	state.Preferences.CloudModel = ""
	settle(tt)
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
