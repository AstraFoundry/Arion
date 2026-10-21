package app

import (
	"strings"
	"testing"

	"arion.local/desktop/ui/features/cloud"
	"arion.local/desktop/ui/features/polish"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Choosing the cloud shows the ChatGPT sign-in and the API key row; a
// pasted key is masked, submitted once and leaves the draft; a connection
// shows its account, the model choice and Disconnect.
func TestCloudPolishRows(t *testing.T) {
	state, actions, store := fixture()
	draft := &cloud.Draft{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			polish.Row(c, store, theme.Resolve(false))
			polish.Engine(c, store, theme.Resolve(false))
			cloud.Rows(c, store, theme.Resolve(false), draft)
		})
	}, 560, 360)
	en := copy.For(copy.English)
	state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1430 << 20}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	settle(tt)
	if hasText(tt, config.CloudProviders[runtime.CloudChatGPT]) {
		t.Fatal("cloud rows shown with the local engine")
	}
	if err := tt.Click(en.Text(copy.PolishEngineCloud)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.PolishEngine != runtime.PolishCloud || !hasText(tt, en.Text(copy.PolishCloudDetail)) || hasText(tt, en.DownloadSize(1430<<20)) {
		t.Fatalf("cloud engine: %q", tt.Texts())
	}
	for _, text := range []string{config.CloudProviders[runtime.CloudChatGPT], en.Text(copy.CloudChatGPTDetail), config.CloudProviders[runtime.CloudOpenAI]} {
		if !hasText(tt, text) {
			t.Fatalf("missing %q: %q", text, tt.Texts())
		}
	}
	snapshot(t, tt, "settings-polish-cloud-disconnected")
	if err := tt.Click(en.Text(copy.AccSignIn)); err != nil || !actions.called("cloud-sign-in") {
		t.Fatalf("sign in: %v", err)
	}
	// The key field masks what it holds.
	connect, ok := tt.Find(en.Text(copy.CloudConnect))
	if !ok {
		t.Fatalf("no connect button: %q", tt.Texts())
	}
	tt.ClickAt(connect.X-60, connect.Y+connect.H/2)
	tt.Type("sk-secret")
	settle(tt)
	for _, text := range tt.Texts() {
		if strings.Contains(text, "sk-secret") {
			t.Fatalf("key shown in %q", text)
		}
	}
	if err := tt.Click(en.Text(copy.CloudConnect)); err != nil || !actions.called("cloud-key:openai:sk-secret") || draft.Key != "" {
		t.Fatalf("connect: %v %v draft %q", err, actions.calls, draft.Key)
	}

	state.Preferences.Polish = true
	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudChatGPT, Email: "ada@example.com"}
	settle(tt)
	if !actions.called("cloud-models") {
		t.Fatal("models not listed for a connection")
	}
	state.Cloud.Models = []string{"model-a", "model-b"}
	settle(tt)
	if !hasText(tt, "ada@example.com") || !hasText(tt, en.Text(copy.CloudModelPlaceholder)) || hasText(tt, en.Text(copy.CloudConnect)) {
		t.Fatalf("connected: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-polish-cloud-connected")
	state.Preferences.CloudModel = "model-b"
	settle(tt)
	if hasText(tt, en.Text(copy.CloudModelPlaceholder)) {
		t.Fatalf("chosen model: %q", tt.Texts())
	}
	if err := tt.Click(en.Text(copy.CloudDisconnect)); err != nil || !actions.called("cloud-disconnect") {
		t.Fatalf("disconnect: %v", err)
	}

	state.Cloud = runtime.Cloud{Phase: runtime.CloudConnected, Provider: runtime.CloudOpenAI, Models: []string{}, Failure: runtime.CloudCodeModels}
	settle(tt)
	if !hasText(tt, en.Text(copy.CloudFailModels)) || !hasText(tt, config.CloudProviders[runtime.CloudOpenAI]) {
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

// The cloud never receives context, so its switches hide with the cloud
// engine, while polishing still works where the local runtime is not
// shipped.
func TestCloudEngineWithoutLocalRuntime(t *testing.T) {
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
	if !hasText(tt, en.Text(copy.MatchAppLabel)) || hasText(tt, en.Text(copy.ContextScreenLabel)) || hasText(tt, en.Text(copy.PolishUnsupported)) {
		t.Fatalf("cloud on an unsupported platform: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-model-cloud")
	state.Preferences.PolishEngine = runtime.PolishLocal
	settle(tt)
	if hasText(tt, en.Text(copy.MatchAppLabel)) || !hasText(tt, en.Text(copy.PolishUnsupported)) {
		t.Fatalf("local on an unsupported platform: %q", tt.Texts())
	}
}

// The onboarding model step fits the cloud rows above its buttons.
func TestOnboardingCloudPolish(t *testing.T) {
	state, actions, store := fixture()
	state.Permissions = runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted}
	state.Polish = runtime.Polish{State: runtime.PolishOff, SizeBytes: 1430 << 20}
	state.Cloud = runtime.Cloud{Phase: runtime.CloudDisconnected}
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	en := copy.For(copy.English)
	for _, label := range []copy.Key{copy.OnbStart, copy.OnbContinueLocally, copy.Continue, copy.Continue, copy.OnbIMELater} {
		if err := tt.Click(en.Text(label)); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		settle(tt)
	}
	if err := tt.Click(en.Text(copy.PolishEngineCloud)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(en.Text(copy.AccSignIn)); err != nil || !actions.called("cloud-sign-in") {
		t.Fatalf("sign in: %v %q", err, tt.Texts())
	}
	key, okKey := tt.Find(config.CloudProviders[runtime.CloudOpenAI])
	back, okBack := tt.Find(en.Text(copy.Back))
	if !okKey || !okBack || key.Y+key.H+16 > back.Y {
		t.Fatalf("cloud rows reach the buttons: key %+v back %+v", key, back)
	}
	snapshot(t, tt, "onboarding-5-model-cloud")
}
