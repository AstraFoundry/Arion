package app

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	platform "runtime"
	"slices"
	"strings"
	"testing"
	"time"

	holdname "arion.local/desktop/ui/entities/holdkey"
	diagnosticui "arion.local/desktop/ui/features/diagnostics"
	keyboardui "arion.local/desktop/ui/features/keyboard"
	"arion.local/desktop/ui/pages/guide"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/overlay"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// recorder implements runtime.Actions for view tests.
type recorder struct {
	calls      []string
	state      *runtime.State
	vocabError error
}

func (r *recorder) call(name string)           { r.calls = append(r.calls, name) }
func (r *recorder) ExportDiagnostics()         { r.call("export-diagnostics") }
func (r *recorder) ToggleDictation()           { r.call("toggle") }
func (r *recorder) CancelDictation()           { r.call("cancel") }
func (r *recorder) RequestMicrophone()         { r.call("request-mic") }
func (r *recorder) RequestAccessibility()      { r.call("request-ax") }
func (r *recorder) OpenMicrophoneSettings()    { r.call("open-mic") }
func (r *recorder) OpenAccessibilitySettings() { r.call("open-ax") }
func (r *recorder) RetryModel()                { r.call("retry") }
func (r *recorder) RetryShortcutListener()     { r.call("retry-shortcuts") }
func (r *recorder) RetryPolish()               { r.call("retry-polish") }
func (r *recorder) DownloadPolish()            { r.call("download-polish") }
func (r *recorder) UseBundle(id string)        { r.call("use:" + id) }
func (r *recorder) UseRecommendedBundle()      { r.call("use-recommended") }
func (r *recorder) RefreshDevices()            { r.call("devices") }
func (r *recorder) StartLevelMonitor()         { r.call("monitor-start") }
func (r *recorder) StopLevelMonitor()          { r.call("monitor-stop") }
func (r *recorder) CopyText(text string) bool  { r.call("copy:" + text); return true }
func (r *recorder) OpenSettings()              { r.call("settings") }
func (r *recorder) CompleteOnboarding()        { r.call("complete") }
func (r *recorder) SignIn()                    { r.call("sign-in") }
func (r *recorder) CancelSignIn()              { r.call("cancel-sign-in") }
func (r *recorder) RefreshAccount()            { r.call("refresh-account") }
func (r *recorder) SignOut()                   { r.call("sign-out") }
func (r *recorder) JoinHostedSyncWaitlist()    { r.call("join-waitlist") }
func (r *recorder) CheckStoredAccount()        { r.call("check-stored") }
func (r *recorder) SignInChatGPT()             { r.call("cloud-sign-in") }
func (r *recorder) CancelCloud()               { r.call("cloud-cancel") }
func (r *recorder) DisconnectCloud()           { r.call("cloud-disconnect") }
func (r *recorder) ActivateCloud(provider string) {
	r.call("cloud-activate:" + provider)
}
func (r *recorder) ListCloudModels() { r.call("cloud-models") }
func (r *recorder) ConnectCloudKey(provider, key string) {
	r.call("cloud-key:" + provider + ":" + key)
}
func (r *recorder) Quit()                    { r.call("quit") }
func (r *recorder) TestHoldKey(testing bool) { r.call("test-hold") }
func (r *recorder) RemoveReplacement(id string) {
	r.call("remove:" + id)
	r.state.Vocabulary = slices.DeleteFunc(r.state.Vocabulary, func(entry runtime.Replacement) bool { return entry.ID == id })
}
func (r *recorder) UndoLearned(id string)   { r.call("undo:" + id) }
func (r *recorder) RequestInputMonitoring() { r.call("request-monitoring") }
func (r *recorder) CloseGuide()             { r.call("close-guide") }
func (r *recorder) CloseTrayHint()          { r.call("close-tray-hint") }
func (r *recorder) CheckForUpdates()        { r.call("check-update") }
func (r *recorder) InstallUpdate()          { r.call("install-update") }
func (r *recorder) RestartToUpdate()        { r.call("restart-update") }
func (r *recorder) CancelUpdateRestart()    { r.call("cancel-update-restart") }
func (r *recorder) LaterUpdate()            { r.call("later-update") }
func (r *recorder) DismissUpdateReminder()  { r.call("dismiss-update-reminder") }

func (r *recorder) OpenUpdate() {
	r.call("open-update")
	r.state.Focus = runtime.SettingsFocus{Section: settings.About, Seq: r.state.Focus.Seq + 1}
}
func (r *recorder) RecordShortcut(owner string) {
	r.call("record:" + owner)
	r.state.Hold.Recording = owner
}

func (r *recorder) SetHoldKey(key runtime.HoldKey) {
	r.call("hold:" + string(key))
	r.state.Preferences.ChooseHoldKey(key)
}
func (r *recorder) SetTranslateKey(key runtime.HoldKey) {
	r.call("translate-hold:" + string(key))
	r.state.Preferences.ChooseTranslateKey(platform.GOOS, key)
}
func (r *recorder) InstallInputMethod() error {
	r.call("install-ime")
	r.state.InputMethod = runtime.InputMethodInstalled
	return nil
}
func (r *recorder) ActivateInputMethod() error     { r.call("activate-ime"); return nil }
func (r *recorder) CancelInputMethodSetup()        { r.call("cancel-ime-setup") }
func (r *recorder) OpenInputMethodSettings() error { r.call("ime-settings"); return nil }
func (r *recorder) RemoveInputMethod() error {
	r.call("remove-ime")
	r.state.InputMethod = runtime.InputMethodAbsent
	return nil
}
func (r *recorder) RequestScreenRecording() { r.call("screen-recording") }
func (r *recorder) ChooseScene(scene runtime.Scene) {
	key := runtime.SceneKey(r.state.SceneTarget.Hints)
	r.state.Preferences.SceneOverrides = r.state.Preferences.SceneOverrides.WithScene(key, scene)
	r.call("scene:" + string(scene))
}
func (r *recorder) UpdatePreferences(change func(*runtime.Preferences)) {
	change(&r.state.Preferences)
	r.call("prefs")
}
func (r *recorder) AddReplacement(from, to string) error {
	return r.AddReplacementVariants([]string{from}, to)
}

func (r *recorder) AddReplacementVariants(froms []string, to string) error {
	if r.vocabError != nil {
		return r.vocabError
	}
	for _, from := range froms {
		r.state.Vocabulary = append(r.state.Vocabulary, runtime.Replacement{ID: from, From: from, To: to})
	}
	r.call("add")
	return nil
}

func (r *recorder) AddHotword(text string) error {
	if r.vocabError != nil {
		return r.vocabError
	}
	r.state.Vocabulary = append(r.state.Vocabulary, runtime.Replacement{ID: text, To: text, Hotword: true})
	r.call("add-hotword")
	return nil
}

func (r *recorder) SetKeyboardPinyin(id, spelling, reading string) error {
	if r.vocabError != nil {
		return r.vocabError
	}
	for i, entry := range r.state.Vocabulary {
		if entry.ID == id && entry.To == spelling {
			r.state.Vocabulary[i].KeyboardPinyin = reading
			r.call("keyboard-reading")
			return nil
		}
	}
	return runtime.ErrVocabularyConflict
}

func (r *recorder) called(name string) bool {
	for _, call := range r.calls {
		if call == name {
			return true
		}
	}
	return false
}

func fixture() (*runtime.State, *recorder, runtime.Store) {
	recommended := runtime.Bundle{ID: "qwen3-asr-1.7b-mlx-8bit", Evaluated: true, Recommended: true,
		ErrorRates: map[string]float64{"zh": 0.041, "en": 0.052, "mixed": 0.087}, ReviewP95Ms: 312, SizeBytes: 270 << 20}
	other := runtime.Bundle{ID: "qwen3-asr-1.7b-gguf-q8", Evaluated: true,
		ErrorRates: map[string]float64{"zh": 0.06, "en": 0.07, "mixed": 0.11}, ReviewP95Ms: 41, SizeBytes: 134 << 20}
	state := &runtime.State{
		Version:     "0.1.0",
		Permissions: runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted},
		Preferences: runtime.Preferences{HoldKey: runtime.DefaultHoldKey(platform.GOOS), Delivery: runtime.DeliveryAutomatic, Sounds: true, CheckUpdates: true,
			Keyboard: runtime.KeyboardPreferences{PageSize: 9, CandidateSize: runtime.CandidateMedium, Appearance: runtime.AppearanceSystem, Accent: runtime.AccentYellow, EmojiSuggestions: true}},
		InputMethod: runtime.InputMethodAbsent,
		Account:     runtime.Account{Phase: runtime.AccountSignedOut, Issuer: "https://astrafoundry.si", Portal: "https://astrafoundry.si/account"},
		Hold:        runtime.Hold{Active: true},
		Model:       runtime.Model{State: runtime.ModelReady, Active: recommended, Automatic: true, LatencyMeasured: true, Bundles: []runtime.Bundle{recommended, other}},
		Devices:     []runtime.Device{{ID: "a", Name: "MacBook Pro Microphone", Default: true}, {ID: "b", Name: "AirPods Pro"}},
	}
	actions := &recorder{state: state}
	return state, actions, runtime.Store{State: state, Actions: actions}
}

func levels(n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = float32((i*37)%100) / 100
	}
	return out
}

// snapshot saves the tester's last frame when ARION_UI_SNAPSHOTS names a
// directory, for visual review.
func snapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	directory := os.Getenv("ARION_UI_SNAPSHOTS")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name+".png"), buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func settle(tt *ui.Tester) {
	for i := 0; i < 40; i++ {
		tt.Frame()
	}
}

// rest runs frames for d of wall time: the tester's animations follow the
// clock, so an entrance finishes only once its duration has passed.
func rest(tt *ui.Tester, d time.Duration) {
	for deadline := time.Now().Add(d); time.Now().Before(deadline); {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
}

// reveal scrolls the settings content until text is inside the viewport.
func reveal(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	for i := 0; i < 20; i++ {
		if r, ok := tt.Find(text); ok && r.Y >= 0 && r.Y+r.H <= SettingsHeight {
			return
		}
		tt.Scroll(SettingsWidth*0.7, SettingsHeight/2, 0, 120)
		settle(tt)
	}
	t.Fatalf("%q not reachable: %q", text, tt.Texts())
}

func hasText(tt *ui.Tester, text string) bool {
	for _, t := range tt.Texts() {
		if strings.Contains(t, text) {
			return true
		}
	}
	return false
}

func TestOverlayPhases(t *testing.T) {
	state, _, store := fixture()
	overlayPage := overlay.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { overlayPage.Render(c, store) }, OverlayWidth, OverlayHeight)
	now := time.Now()
	cases := []struct {
		name string
		d    runtime.Dictation
		want string
	}{
		{"listening", runtime.Dictation{SessionID: "1", State: runtime.StateCapturing, Levels: levels(40), Changed: now}, copy.For(copy.English).Text(copy.PillListening)},
		{"draft", runtime.Dictation{SessionID: "1", State: runtime.StateCapturing, Levels: levels(40), Draft: "把这个 PR merge 到 main 分支之前先跑一下 CI", Changed: now}, "merge"},
		{"transcribing", runtime.Dictation{SessionID: "1", State: runtime.StateFinalizing, Draft: "今天的 standup 改到上午十点半", Changed: now}, "standup"},
		{"inserted", runtime.Dictation{SessionID: "1", State: runtime.StateCompleted, Outcome: runtime.OutcomeApplied, Changed: now}, copy.For(copy.English).Text(copy.PillInserted)},
		{"typed", runtime.Dictation{SessionID: "2", State: runtime.StateCompleted, Outcome: runtime.OutcomeUnknown, Changed: now}, copy.For(copy.English).Text(copy.PillTyped)},
		{"focus-moved", runtime.Dictation{SessionID: "3", State: runtime.StateFailed, Failure: runtime.CodeTargetChanged, Changed: now}, copy.For(copy.English).Text(copy.FailTargetTitle)},
		{"no-speech", runtime.Dictation{SessionID: "4", State: runtime.StateFailed, Failure: runtime.CodeNoSpeech, Changed: now}, copy.For(copy.English).Text(copy.FailNoSpeechTitle)},
	}
	for _, tc := range cases {
		state.Dictation = tc.d
		state.Dictation.Changed = time.Now()
		settle(tt)
		if tc.d.Draft != "" {
			// The draft types out rather than appearing whole.
			if tc.name == "draft" && hasText(tt, tc.d.Draft) {
				t.Errorf("draft appeared at once: %q", tt.Texts())
			}
			rest(tt, 1500*time.Millisecond)
		}
		if !hasText(tt, tc.want) {
			t.Errorf("%s: want %q in %q", tc.name, tc.want, tt.Texts())
		}
		snapshot(t, tt, "overlay-"+tc.name)
	}
	// Outcomes leave after their display time.
	state.Dictation = runtime.Dictation{SessionID: "5", State: runtime.StateCompleted, Outcome: runtime.OutcomeApplied, Changed: now.Add(-5 * time.Second)}
	settle(tt)
	if hasText(tt, copy.For(copy.English).Text(copy.PillInserted)) {
		t.Error("stale outcome still visible")
	}
	// A denied microphone is named precisely.
	state.Permissions.Microphone = runtime.ConsentDenied
	state.Dictation = runtime.Dictation{SessionID: "6", State: runtime.StateFailed, Failure: runtime.CodePermissionDenied, Changed: time.Now()}
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.FailMicTitle)) {
		t.Errorf("permission failure not explained: %q", tt.Texts())
	}
}

func TestMenuBarPanel(t *testing.T) {
	state, actions, store := fixture()
	panel := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store))
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.StatusReady)) {
		t.Fatalf("idle panel: %q", tt.Texts())
	}
	if hasText(tt, copy.For(copy.English).Text(copy.ModelNameGeneric)) {
		t.Fatalf("ready panel shows the model, which settings owns: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-ready")
	if err := tt.Click(copy.For(copy.English).Text(copy.StartDictate)); err != nil {
		t.Fatal(err)
	}
	if !actions.called("toggle") {
		t.Fatal("start button did not toggle dictation")
	}
	state.Dictation = runtime.Dictation{SessionID: "1", State: runtime.StateCapturing, Levels: levels(40), Changed: time.Now()}
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.StopDictate)) || !hasText(tt, copy.For(copy.English).Text(copy.StatusListening)) {
		t.Fatalf("live panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-listening")
	state.Dictation = runtime.Dictation{}
	state.Model = runtime.Model{State: runtime.ModelProvisioning, Phase: runtime.PhaseDownloading, Done: 63 << 20, Total: 134 << 20, Active: state.Model.Active, Automatic: true}
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.StatusPreparing)) || !hasText(tt, "63 MB of 134 MB") {
		t.Fatalf("provisioning panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-downloading")

	// A failed download offers its retry above the footer.
	state.Model = runtime.Model{State: runtime.ModelFailed, Failure: runtime.CodeProvisioningFailed, Active: state.Model.Active, Automatic: true}
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	en := copy.For(copy.English)
	retry, ok := tt.Find(en.Text(copy.Retry))
	quit, _ := tt.Find(en.Text(copy.Quit))
	if !ok || retry.Y+retry.H > quit.Y || quit.Y+quit.H > float32(PanelHeightFor(store)) {
		t.Fatalf("retry %+v and footer %+v do not fit the panel", retry, quit)
	}
	if err := tt.Click(en.Text(copy.Retry)); err != nil || !actions.called("retry") {
		t.Fatalf("retry: %v %q", err, actions.calls)
	}
	snapshot(t, tt, "panel-failed")
}

func TestSettingsSections(t *testing.T) {
	state, actions, store := fixture()
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	for _, section := range []struct{ id, label, want string }{
		{settings.Overview, copy.For(copy.English).Text(copy.NavOverview), copy.For(copy.English).Text(copy.UsageActivity)},
		{settings.General, copy.For(copy.English).Text(copy.NavGeneral), copy.For(copy.English).Text(copy.LaunchAtLogin)},
		{settings.Dictation, copy.For(copy.English).Text(copy.NavDictation), copy.For(copy.English).Text(copy.HoldKeyLabel)},
		{settings.Microphone, copy.For(copy.English).Text(copy.NavMicrophone), copy.For(copy.English).Text(copy.MicTest)},
		{settings.Model, copy.For(copy.English).Text(copy.NavModel), copy.For(copy.English).Text(copy.PanelAutomatic)},
		{settings.Vocabulary, copy.For(copy.English).Text(copy.NavVocabulary), copy.For(copy.English).Text(copy.VocabEmpty)},
		{settings.About, copy.For(copy.English).Text(copy.NavAbout), copy.For(copy.English).Text(copy.PrivacyLocal)},
	} {
		if err := tt.Click(section.label); err != nil {
			t.Fatal(err)
		}
		settle(tt)
		if page.Section != section.id || !hasText(tt, section.want) {
			t.Errorf("%s: section %s texts %q", section.id, page.Section, tt.Texts())
		}
		snapshot(t, tt, "settings-"+section.id)
	}
	// The level monitor runs only while the microphone section shows.
	if !actions.called("monitor-start") || !actions.called("monitor-stop") {
		t.Errorf("monitor lifecycle %v", actions.calls)
	}
	// Vocabulary: add a replacement through the form.
	if err := tt.Click(copy.For(copy.English).Text(copy.NavVocabulary)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(copy.For(copy.English).Text(copy.VocabToLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Type("Arion")
	if err := tt.Click(copy.For(copy.English).Text(copy.Add)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	rest(tt, 500*time.Millisecond)
	if err := tt.Click(copy.For(copy.English).VocabVariantAction("Arion")); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Type("阿里昂")
	tt.Key(0, ui.KeyEnter)
	settle(tt)
	if len(state.Vocabulary) != 2 || state.Vocabulary[1].To != "Arion" || !hasText(tt, "阿里昂") {
		t.Fatalf("vocabulary not added: %v %q", state.Vocabulary, tt.Texts())
	}
	snapshot(t, tt, "settings-vocabulary-filled")
	// Keyboard setup remains available independently of dictation routing.
	state.InputMethod = runtime.InputMethodAbsent
	if err := tt.Click(copy.For(copy.English).Text(copy.NavKeyboard)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(copy.For(copy.English).Text(copy.IMEInstall)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !actions.called("install-ime") || !hasText(tt, copy.For(copy.English).Text(copy.IMEInstalled)) {
		t.Fatalf("input method install: %v %q", actions.calls, tt.Texts())
	}
}

// TestDarkAppearance renders the main surfaces with the dark tokens.
func TestMicrophoneGain(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = locale
		messages := store.Copy()
		page := settings.NewPage()
		page.Section = settings.Microphone
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		settle(tt)
		if !hasText(tt, messages.Text(copy.MicGain)) || !hasText(tt, messages.Decibels(0)) {
			t.Fatalf("%s: gain row missing: %q", locale, tt.Texts())
		}
		snapshot(t, tt, "settings-microphone-gain-"+string(locale))
		// The slider spans the card under its row; its far end is the loudest stop.
		title, _ := tt.Find(messages.Text(copy.MicGain))
		readout, _ := tt.Find(messages.Decibels(0))
		tt.ClickAt(readout.X+readout.W-14, title.Y+title.H+12+14)
		settle(tt)
		loudest := runtime.GainStops[len(runtime.GainStops)-1]
		if state.Preferences.MicrophoneGain != loudest.DB || !actions.called("prefs") || !hasText(tt, messages.Decibels(loudest.DB)) {
			t.Fatalf("%s: gain %d, texts %q", locale, state.Preferences.MicrophoneGain, tt.Texts())
		}
		snapshot(t, tt, "settings-microphone-gain-loudest-"+string(locale))
	}
}

func TestAboutLinks(t *testing.T) {
	_, _, store := fixture()
	page := settings.NewPage()
	page.Section = settings.About
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	en := copy.For(copy.English)
	links := []struct{ label, url string }{
		{config.Team, config.TeamURL}, {config.Repository, config.RepositoryURL}, {config.XHandle, config.XURL},
		{en.Text(copy.AboutFeedbackLink), config.IssuesURL}, {config.License, config.LicenseURL},
	}
	// The links close the section, below the fold.
	tt.Scroll(SettingsWidth/2, SettingsHeight/2, 0, 2000)
	settle(tt)
	var want []string
	for _, link := range links {
		if err := tt.Click(link.label); err != nil {
			t.Fatalf("%s: %v %q", link.label, err, tt.Texts())
		}
		want = append(want, link.url)
	}
	if got := tt.OpenedURLs(); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("opened %q, want %q", got, want)
	}
	snapshot(t, tt, "settings-about-links")
	// The banner loops a dictation; reduced motion holds its final frame.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	settle(tt)
	if !hasText(tt, en.Text(copy.PillInserted)) {
		t.Errorf("reduced motion: %q", tt.Texts())
	}
	tt.SetDark(true)
	settle(tt)
	snapshot(t, tt, "dark-about")
}

func TestUpdateStates(t *testing.T) {
	state, actions, store := fixture()
	page := settings.NewPage()
	page.Section = settings.About
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	en := copy.For(copy.English)
	for _, step := range []struct {
		update runtime.Update
		want   string
		click  string
		call   string
	}{
		{runtime.Update{Phase: runtime.UpdateUnsupported}, en.Text(copy.UpdateUnsupported), "", ""},
		{runtime.Update{Phase: runtime.UpdateIdle}, en.Text(copy.AboutUpdates), en.Text(copy.UpdateCheck), "check-update"},
		{runtime.Update{Phase: runtime.UpdateNoRelease}, en.Text(copy.UpdateNoRelease), en.Text(copy.UpdateCheck), "check-update"},
		{runtime.Update{Phase: runtime.UpdateCurrent}, en.Text(copy.UpdateCurrent), en.Text(copy.UpdateCheck), "check-update"},
		{runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}, en.UpdateAvailable("0.2.0"), en.Text(copy.UpdateInstall), "install-update"},
		{runtime.Update{Phase: runtime.UpdateDownloading, Version: "0.2.0", Downloaded: 3 << 20, Total: 12 << 20}, en.Progress(3<<20, 12<<20), "", ""},
		{runtime.Update{Phase: runtime.UpdateReady, Version: "0.2.0"}, en.Text(copy.UpdateReady), en.Text(copy.UpdateRestart), "restart-update"},
		{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepCheck}, en.Text(copy.UpdateCheckFailed), en.Text(copy.UpdateCheck), "check-update"},
		{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepInstall, Version: "0.2.0"}, en.Text(copy.UpdateInstallFailed), en.Text(copy.Retry), "install-update"},
	} {
		state.Update = step.update
		actions.calls = nil
		settle(tt)
		if !hasText(tt, step.want) {
			t.Errorf("%s: want %q in %q", step.update.Phase, step.want, tt.Texts())
		}
		// The automatic check is offered wherever the build can update.
		if hasText(tt, en.Text(copy.UpdateAuto)) == (step.update.Phase == runtime.UpdateUnsupported) {
			t.Errorf("%s: automatic check row %q", step.update.Phase, tt.Texts())
		}
		if step.click != "" {
			if err := tt.Click(step.click); err != nil || !actions.called(step.call) {
				t.Errorf("%s: click %q: %v %q", step.update.Phase, step.click, err, actions.calls)
			}
		}
		snapshot(t, tt, "settings-about-update-"+string(step.update.Phase)+string(step.update.Failed))
	}
	state.Update = runtime.Update{Phase: runtime.UpdateCurrent}
	settle(tt)
	reveal(t, tt, en.Text(copy.UpdatePrerelease))
	label, ok := tt.Find(en.Text(copy.UpdatePrerelease))
	if !ok {
		t.Fatal("missing prerelease label")
	}
	tt.ClickAt(SettingsWidth-60, label.Y+label.H)
	settle(tt)
	if !state.Preferences.AllowPrereleaseUpdates {
		t.Fatal("prerelease switch not persisted")
	}
	// The switch sits at the row's trailing edge, level with its title and detail.
	title, ok := tt.Find(en.Text(copy.UpdateAuto))
	if !ok {
		t.Fatalf("no automatic check row: %q", tt.Texts())
	}
	tt.ClickAt(SettingsWidth-60, title.Y+title.H)
	settle(tt)
	if state.Preferences.CheckUpdates {
		t.Error("automatic check not turned off")
	}
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	zh := copy.For(copy.SimplifiedChinese)
	if !hasText(tt, zh.Text(copy.UpdatePrerelease)) {
		t.Fatal("missing translated prerelease switch")
	}
	snapshot(t, tt, "settings-about-update-chinese")
}

func TestUpdateNotice(t *testing.T) {
	state, actions, store := fixture()
	panel := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store))
	en := copy.For(copy.English)
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	if hasText(tt, en.UpdateTo("0.2.0")) {
		t.Fatal("notice without an update")
	}
	quit, _ := tt.Find(en.Text(copy.Quit))
	state.Update = runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.1.0-alpha.6"}
	state.Preferences.UILanguage = runtime.LanguageChinese
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	zh := copy.For(copy.SimplifiedChinese)
	if q, ok := tt.Find(zh.Text(copy.Quit)); !ok || q.X+q.W > PanelWidth-16 {
		t.Fatalf("a long update crowds Quit: %+v", q)
	}
	snapshot(t, tt, "panel-update-zh-CN")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	state.Update = runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}
	tt.SetSize(PanelWidth, PanelHeightFor(store))
	settle(tt)
	if q, _ := tt.Find(en.Text(copy.Quit)); q.W != quit.W || q.X != quit.X {
		t.Fatalf("the update moved Quit: %+v from %+v", q, quit)
	}
	snapshot(t, tt, "panel-update")
	if err := tt.Click(en.UpdateTo("0.2.0")); err != nil || !actions.called("open-update") {
		t.Fatalf("notice: %v %q", err, actions.calls)
	}
	// The settings window follows the request to the update.
	page := settings.NewPage()
	settingsTester := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(settingsTester)
	if page.Section != settings.About {
		t.Fatalf("section %q", page.Section)
	}
	page.Section = settings.General
	settle(settingsTester)
	if page.Section != settings.General {
		t.Fatal("an applied request moved the page again")
	}
}

// usageFixture fills a week of counts ending today and a few replacements.
func usageFixture(state *runtime.State) {
	now := time.Now()
	for i, words := range []int{120, 0, 860, 430, 1210, 640, 380} {
		day := now.AddDate(0, 0, i-6)
		if words > 0 {
			state.Usage = state.Usage.Add(day, runtime.Usage{SpeechMs: int64(words) * 400, Words: words / 2})
			state.Usage = state.Usage.Add(day, runtime.Usage{SpeechMs: int64(words) * 300, Words: words / 4, Scene: runtime.SceneWork})
			state.Usage = state.Usage.Add(day, runtime.Usage{SpeechMs: int64(words) * 150, Words: words - words/2 - words/4, Scene: runtime.ScenePrompt})
		}
	}
	state.Usage = state.Usage.Add(now.AddDate(0, -2, 0), runtime.Usage{SpeechMs: 600000, Words: 1800, Scene: runtime.SceneEmail})
	// Earlier months: busier on weekdays, with quiet stretches.
	for back := 7; back < 210; back++ {
		day := now.AddDate(0, 0, -back)
		words := (back * 7919) % 900
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			words /= 4
		}
		if back%23 < 4 || words < 60 {
			continue
		}
		state.Usage = state.Usage.Add(day, runtime.Usage{SpeechMs: int64(words) * 700, Words: words})
	}
	state.Vocabulary = []runtime.Replacement{{ID: "1", From: "cloud code", To: "Claude Code"}, {ID: "2", From: "a rion", To: "Arion"},
		{ID: "3", From: "my go", To: "MyGo"}}
}

func TestUsageOverview(t *testing.T) {
	state, _, store := fixture()
	en := copy.For(copy.English)
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	if page.Section != settings.Overview || !hasText(tt, en.Text(copy.UsageActivity)) || !hasText(tt, en.Text(copy.UsageScenesEmpty)) {
		t.Fatalf("empty overview: %s %q", page.Section, tt.Texts())
	}
	snapshot(t, tt, "settings-overview-empty")

	// Entering the overview lights the heatmap up from the oldest week.
	usageFixture(state)
	if err := tt.Click(en.Text(copy.NavGeneral)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(en.Text(copy.NavOverview)); err != nil {
		t.Fatal(err)
	}
	// A filmstrip of the entrance, each frame named by its time.
	for entered := time.Now(); time.Since(entered) < 1600*time.Millisecond; {
		tt.Frame()
		snapshot(t, tt, fmt.Sprintf("settings-overview-entering-%04d", time.Since(entered).Milliseconds()))
	}
	rest(tt, 300*time.Millisecond)
	total := state.Usage.Since(time.Time{})
	if !hasText(tt, en.Count(total.Words)) || !hasText(tt, "Claude Code") {
		t.Fatalf("filled overview: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-overview")
	// Hovering this week's Monday names its words.
	title, ok := tt.Find(en.Text(copy.UsageActivity))
	if !ok {
		t.Fatalf("no heatmap: %q", tt.Texts())
	}
	// Header row (44, title centered), the map's 4 DIP margin and its
	// month labels (18), then half a cell.
	tt.Move(SettingsWidth-theme.Gutter-17-6, title.Y+title.H/2+22+4+18+6)
	rest(tt, 900*time.Millisecond)
	snapshot(t, tt, "settings-overview-hover")
	now := time.Now()
	thisMonday := now.AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
	tip := en.DayWords(thisMonday, state.Usage.Days(thisMonday, 1)[0].Words)
	if !hasText(tt, tip) {
		t.Errorf("no tooltip %q: %q", tip, tt.Texts())
	}
	tt.Move(10, 10)
	// A dictation committed while the overview shows rolls the figures on
	// without replaying the entrance.
	state.Usage = state.Usage.Add(time.Now(), runtime.Usage{SpeechMs: 4000, Words: 256})
	rest(tt, 1200*time.Millisecond)
	if grown := state.Usage.Since(time.Time{}).Words; !hasText(tt, en.Count(grown)) {
		t.Fatalf("figure did not follow %d: %q", grown, tt.Texts())
	}
	snapshot(t, tt, "settings-overview-grown")
	tt.Scroll(SettingsWidth*0.7, SettingsHeight/2, 0, 600)
	settle(tt)
	snapshot(t, tt, "settings-overview-scenes")
	// Today has no email; all time does.
	if hasText(tt, en.Text(copy.SceneEmail)) {
		t.Fatalf("today shows an older scene: %q", tt.Texts())
	}
	if err := tt.Click(en.Text(copy.UsageRangeToday)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(en.Text(copy.UsageRangeAll)); err != nil {
		t.Fatal(err)
	}
	rest(tt, 600*time.Millisecond)
	if !hasText(tt, en.Text(copy.SceneEmail)) {
		t.Fatalf("all time misses email: %q", tt.Texts())
	}
	if err := tt.Click(en.Text(copy.UsageVocabEdit)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if page.Section != settings.Vocabulary {
		t.Fatalf("edit opened %s", page.Section)
	}

	state.Preferences.UILanguage = runtime.LanguageChinese
	zh := settings.NewPage()
	zhTester := ui.NewTester(func(c *ui.Context) { zh.Render(c, store) }, SettingsWidth, SettingsHeight)
	rest(zhTester, 1300*time.Millisecond)
	if !hasText(zhTester, copy.For(copy.SimplifiedChinese).Text(copy.UsageSpeed)) {
		t.Fatalf("zh overview: %q", zhTester.Texts())
	}
	snapshot(t, zhTester, "settings-overview-zh-CN")
	zhTester.Scroll(SettingsWidth*0.7, SettingsHeight/2, 0, 600)
	settle(zhTester)
	snapshot(t, zhTester, "settings-overview-zh-CN-scenes")
	dark := settings.NewPage()
	darkTester := ui.NewTester(func(c *ui.Context) { dark.Render(c, store) }, SettingsWidth+300, SettingsHeight)
	darkTester.SetDark(true)
	rest(darkTester, 1300*time.Millisecond)
	snapshot(t, darkTester, "dark-overview-wide")
}

// TestHeatmapReducedMotion shows every lit day at once when the desktop
// asks for less motion.
func TestHeatmapReducedMotion(t *testing.T) {
	state, _, store := fixture()
	usageFixture(state)
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	rest(tt, 700*time.Millisecond)
	snapshot(t, tt, "settings-overview-reduced-motion")
}

func TestDarkAppearance(t *testing.T) {
	_, _, store := fixture()
	panel := &menubar.Panel{}
	settingsPage := settings.NewPage()
	settingsPage.Section = settings.Dictation
	onboardingPage := onboarding.NewPage()
	for _, view := range []struct {
		name          string
		render        func(c *ui.Context)
		width, height int
	}{
		{"panel", func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store)},
		{"settings", func(c *ui.Context) { settingsPage.Render(c, store) }, SettingsWidth, SettingsHeight},
		{"onboarding", func(c *ui.Context) { onboardingPage.Render(c, store) }, OnboardingWidth, OnboardingHeight},
	} {
		tt := ui.NewTester(view.render, view.width, view.height)
		tt.SetDark(true)
		settle(tt)
		if len(tt.Texts()) == 0 {
			t.Errorf("%s: nothing drawn in the dark appearance", view.name)
		}
		snapshot(t, tt, "dark-"+view.name)
	}
}

func TestOnboardingFlow(t *testing.T) {
	state, actions, store := fixture()
	state.Permissions = runtime.Permissions{Microphone: runtime.ConsentUndetermined, Accessibility: runtime.ConsentUndetermined}
	state.Model = runtime.Model{State: runtime.ModelProvisioning, Phase: runtime.PhaseDownloading, Done: 21 << 20, Total: 134 << 20, Active: state.Model.Active, Automatic: true, LatencyMeasured: true}
	page := onboarding.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	settle(tt)
	snapshot(t, tt, "onboarding-1-welcome")
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatalf("%s: %v (texts %q)", label, err, tt.Texts())
		}
		settle(tt)
	}
	click(copy.For(copy.English).Text(copy.OnbStart))
	snapshot(t, tt, "onboarding-2-account")
	click(copy.For(copy.English).Text(copy.OnbContinueLocally))
	if actions.called("sign-in") {
		t.Fatal("continuing locally started a sign-in")
	}
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbMicTitle)) {
		t.Fatalf("microphone step missing: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-3-microphone")
	// Continue stays disabled until consent exists.
	click(copy.For(copy.English).Text(copy.Continue))
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbMicTitle)) {
		t.Fatal("advanced without microphone consent")
	}
	click(copy.For(copy.English).Text(copy.OnbMicAllow))
	if !actions.called("request-mic") {
		t.Fatal("microphone not requested")
	}
	state.Permissions.Microphone = runtime.ConsentGranted
	state.MonitorLevels = levels(40)
	settle(tt)
	snapshot(t, tt, "onboarding-3-microphone-granted")
	click(copy.For(copy.English).Text(copy.Continue))
	snapshot(t, tt, "onboarding-4-accessibility")
	click(copy.For(copy.English).Text(copy.OnbAXAllow))
	if !actions.called("request-ax") || !hasText(tt, copy.For(copy.English).Text(copy.OnbAXWaiting)) {
		t.Fatalf("accessibility request: %v %q", actions.calls, tt.Texts())
	}
	snapshot(t, tt, "onboarding-4-accessibility-waiting")
	state.Permissions.Accessibility = runtime.ConsentGranted
	deadline := time.Now().Add(2 * time.Second)
	for !hasText(tt, copy.For(copy.English).Text(copy.OnbIMETitle)) && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(20 * time.Millisecond)
	}
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbIMETitle)) {
		t.Fatalf("did not advance after accessibility grant: %q", tt.Texts())
	}
	settle(tt)
	snapshot(t, tt, "onboarding-input-method-absent")
	click(copy.For(copy.English).Text(copy.Continue))
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbIMETitle)) {
		t.Fatal("unconfigured input source advanced without an explicit skip")
	}
	click(copy.For(copy.English).Text(copy.IMEInstall))
	if !actions.called("install-ime") {
		t.Fatal("onboarding did not reuse installation or changed dictation mode")
	}
	click(copy.For(copy.English).Text(copy.Continue))
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbIMETitle)) {
		t.Fatal("enabled source counted as current selection")
	}
	state.InputMethod = runtime.InputMethodPending
	settle(tt)
	click(copy.For(copy.English).Text(copy.IMESetup))
	if !actions.called("ime-settings") {
		t.Fatal("onboarding did not open the shared settings guide action")
	}
	snapshot(t, tt, "onboarding-input-method-pending")
	state.InputMethod = runtime.InputMethodSelected
	settle(tt)
	snapshot(t, tt, "onboarding-input-method-selected")
	click(copy.For(copy.English).Text(copy.Continue))
	snapshot(t, tt, "onboarding-5-model")
	click(copy.For(copy.English).Text(copy.Continue))
	// Polishing stays off; the cloud step is skipped.
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbPolishTitle)) {
		t.Fatalf("polish step missing: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-5-polish")
	click(copy.For(copy.English).Text(copy.Continue))
	if !actions.called("test-hold") {
		t.Fatal("hold key test mode not entered")
	}
	// A key that cannot listen holds the step until Input Monitoring is granted.
	state.Hold = runtime.Hold{Failure: runtime.CodePermissionDenied}
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.HoldKeyNeedsMonitoring)) {
		t.Fatalf("missing consent not explained: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-6-hold-key-consent")
	click(copy.For(copy.English).Text(copy.Continue))
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbHoldKeyTitle)) {
		t.Fatal("onboarding moved past a hold key that cannot listen")
	}
	state.Hold = runtime.Hold{Active: true}
	settle(tt)
	snapshot(t, tt, "onboarding-6-hold-key")
	state.Hold.Pressed, state.Hold.PressedKey = time.Now().Add(time.Second), state.Preferences.HoldKey
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbHoldKeyHeard)) {
		t.Fatalf("hold key press not acknowledged: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-6-hold-key-heard")
	click(copy.For(copy.English).Text(copy.Continue))
	// Polishing stayed off, so voice translation is skipped.
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbTryTitle)) {
		t.Fatalf("try step missing: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-7-try")
	click(copy.For(copy.English).Text(copy.Continue))
	snapshot(t, tt, "onboarding-8-done")
	click(copy.For(copy.English).Text(copy.OnbFinish))
	if !actions.called("complete") {
		t.Fatal("onboarding not completed")
	}
}

func TestOnboardingLanguageClearsCaptionButtons(t *testing.T) {
	for _, chrome := range []struct {
		name string
		bar  ui.TitleBar
	}{
		{"windows", ui.TitleBar{Height: 32, Right: 138}},
		{"tall-caption", ui.TitleBar{Height: 48, Right: 180}},
		{"macos", ui.TitleBar{Height: 28, Left: 80}},
	} {
		t.Run(chrome.name, func(t *testing.T) {
			state, _, store := fixture()
			page := onboarding.NewPage()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
			tt.SetTitleBar(chrome.bar)
			settle(tt)
			for _, label := range []string{"English", "简体中文"} {
				r, ok := tt.Find(store.Copy().Text(copy.LanguageLabel))
				if !ok {
					t.Fatal("language picker missing")
				}
				if r.Y < chrome.bar.Height && (r.X < chrome.bar.Left || r.X+r.W > OnboardingWidth-chrome.bar.Right) {
					t.Fatalf("language picker %+v intersects native caption controls %+v", r, chrome.bar)
				}
				snapshot(t, tt, "onboarding-caption-"+chrome.name+"-"+label)
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				settle(tt)
				next := "简体中文"
				if label == next {
					next = "English"
				}
				if err := tt.Click(next); err != nil {
					t.Fatal(err)
				}
				settle(tt)
			}
			if state.Preferences.UILanguage != runtime.LanguageEnglish {
				t.Fatal("language round trip failed")
			}
		})
	}
}

func TestInterfaceLanguageSwitch(t *testing.T) {
	state, actions, store := fixture()
	page := settings.NewPage()
	panel := &menubar.Panel{}
	onboardingPage := onboarding.NewPage()
	settingsTester := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	panelTester := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store))
	onboardingTester := ui.NewTester(func(c *ui.Context) { onboardingPage.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	page.Section = settings.General
	draft := "User text {index}: keep exactly."
	state.Dictation = runtime.Dictation{SessionID: "active", State: runtime.StateCapturing, Draft: draft, Changed: time.Now()}
	settle(settingsTester)
	if err := settingsTester.Click("English"); err != nil {
		t.Fatal(err)
	}
	settle(settingsTester)
	if err := settingsTester.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	settle(settingsTester)
	if state.Preferences.UILanguage != runtime.LanguageChinese || !actions.called("prefs") {
		t.Fatalf("picker did not update preference: %v", state.Preferences)
	}
	settle(panelTester)
	settle(onboardingTester)
	if !hasText(settingsTester, "界面语言") || !hasText(panelTester, "结束") || !hasText(onboardingTester, "开始设置") {
		t.Fatalf("views did not switch: %q %q %q", settingsTester.Texts(), panelTester.Texts(), onboardingTester.Texts())
	}
	if state.Dictation.SessionID != "active" || state.Dictation.Draft != draft {
		t.Fatal("language change altered user content/session")
	}
	snapshot(t, settingsTester, "settings-general-zh-CN")
	snapshot(t, panelTester, "panel-zh-CN")
	snapshot(t, onboardingTester, "onboarding-zh-CN")
	if err := settingsTester.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	settle(settingsTester)
	if err := settingsTester.Click("English"); err != nil {
		t.Fatal(err)
	}
	settle(settingsTester)
	settle(panelTester)
	if state.Preferences.UILanguage != runtime.LanguageEnglish || !hasText(panelTester, "Finish") || !hasText(settingsTester, "Interface language") {
		t.Fatal("switch back to English failed")
	}
}

func TestChineseSettingsAndTransientErrors(t *testing.T) {
	state, actions, store := fixture()
	state.Preferences.UILanguage = runtime.LanguageChinese
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	zh := copy.For(copy.SimplifiedChinese)
	for _, section := range []struct {
		id  string
		key copy.Key
	}{
		{settings.General, copy.LanguageLabel}, {settings.Dictation, copy.HoldKeyLabel},
		{settings.Microphone, copy.MicTest}, {settings.Model, copy.PanelAutomatic},
		{settings.Vocabulary, copy.VocabEmpty}, {settings.About, copy.PrivacyLocal},
	} {
		page.Section = section.id
		settle(tt)
		if !hasText(tt, zh.Text(section.key)) {
			t.Errorf("%s: %q", section.id, tt.Texts())
		}
		snapshot(t, tt, "settings-"+section.id+"-zh-CN")
	}
	page.Section = settings.Vocabulary
	settle(tt)
	if err := tt.Click(zh.Text(copy.VocabToLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Type("<|system|>")
	actions.vocabError = runtime.ErrVocabularyInvalid
	if err := tt.Click("添加"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !hasText(tt, zh.Text(copy.VocabTermInvalid)) {
		t.Fatalf("missing localized validation: %q", tt.Texts())
	}
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.VocabTermInvalid)) || hasText(tt, zh.Text(copy.VocabTermInvalid)) {
		t.Fatalf("stale validation after switch: %q", tt.Texts())
	}
	actions.vocabError = nil

	// Drafts entered in English survive the language change and are saved verbatim.
	if err := tt.Click(copy.For(copy.English).Text(copy.VocabToLabel)); err != nil {
		t.Fatal(err)
	}
	for range "<|system|>" {
		tt.Key(0, ui.KeyBackspace)
	}
	tt.Type("Arion {value}")
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if err := tt.Click("添加"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	rest(tt, 500*time.Millisecond)
	if err := tt.Click(zh.VocabVariantAction("Arion {value}")); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Type("阿里昂 {value}")
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	tt.Key(0, ui.KeyEnter)
	settle(tt)
	if len(state.Vocabulary) != 2 || state.Vocabulary[0].To != "Arion {value}" || state.Vocabulary[1].From != "阿里昂 {value}" || state.Vocabulary[1].To != "Arion {value}" {
		t.Fatalf("language switch changed vocabulary draft: %v", state.Vocabulary)
	}
}

func TestChineseOverlayAndDownload(t *testing.T) {
	state, _, store := fixture()
	state.Preferences.UILanguage = runtime.LanguageChinese
	overlayPage := overlay.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { overlayPage.Render(c, store) }, OverlayWidth, OverlayHeight)
	for _, tc := range []struct {
		session runtime.Dictation
		want    string
	}{
		{runtime.Dictation{State: runtime.StateCapturing}, "正在聆听…"},
		{runtime.Dictation{State: runtime.StateFinalizing}, "正在转写…"},
		{runtime.Dictation{State: runtime.StateCompleted, Outcome: runtime.OutcomeApplied}, "已插入"},
		{runtime.Dictation{State: runtime.StateFailed, Failure: runtime.CodeNoSpeech}, "未检测到语音"},
	} {
		state.Dictation = tc.session
		state.Dictation.SessionID = tc.want
		state.Dictation.Changed = time.Now()
		settle(tt)
		if !hasText(tt, tc.want) {
			t.Fatalf("want %q: %q", tc.want, tt.Texts())
		}
	}
	state.Dictation = runtime.Dictation{}
	state.Model.State = runtime.ModelProvisioning
	state.Model.Phase = runtime.PhaseDownloading
	state.Model.Done, state.Model.Total = 63<<20, 134<<20
	panel := &menubar.Panel{}
	panelTester := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store))
	settle(panelTester)
	if !hasText(panelTester, "正在下载") || !hasText(panelTester, "63 MB / 134 MB") {
		t.Fatalf("untranslated download: %q", panelTester.Texts())
	}
}

// Hold-to-talk: the Dictation section offers this platform's keys, explains
// missing Input Monitoring with a way to grant it, and the pill says to
// release the key while a hold dictates.
func TestHoldKeySettingsAndPill(t *testing.T) {
	state, actions, store := fixture()
	en := copy.For(copy.English)
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	if err := tt.Click(en.Text(copy.NavDictation)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !hasText(tt, en.Text(copy.HoldKeyLabel)) {
		t.Fatalf("hold row missing: %q", tt.Texts())
	}
	keys := runtime.HoldKeysFor(platform.GOOS)
	choice := keys[0]
	if err := tt.Click(en.Text(copy.HoldKeyEdit)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	state.Hold.RecordedKey = choice
	state.Hold.RecordedSeq++
	settle(tt)
	if !actions.called("hold:"+string(choice)) || state.Preferences.HoldKey != choice {
		t.Fatalf("hold key not chosen: %v", actions.calls)
	}
	state.Hold = runtime.Hold{Failure: runtime.CodePermissionDenied}
	// Finish the card height transition before clicking the revealed consent.
	rest(tt, 200*time.Millisecond)
	if !hasText(tt, en.Text(copy.HoldKeyNeedsMonitoring)) {
		t.Fatalf("missing consent not explained: %q", tt.Texts())
	}
	if err := tt.Click(en.Text(copy.Allow)); err != nil {
		t.Fatal(err)
	}
	if !actions.called("request-monitoring") {
		t.Fatalf("consent not requested: %v", actions.calls)
	}
	if choice == runtime.HoldFn && !hasText(tt, en.Text(copy.HoldKeyFnHint)) {
		t.Fatalf("globe key hint missing: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-dictation-hold")

	overlayPage := overlay.NewPage()
	pill := ui.NewTester(func(c *ui.Context) { overlayPage.Render(c, store) }, OverlayWidth, OverlayHeight)
	state.Dictation = runtime.Dictation{SessionID: "h", State: runtime.StateCapturing, Levels: levels(40), Changed: time.Now()}
	state.Hold.Holding = true
	settle(pill)
	if !hasText(pill, en.Text(copy.PillReleaseHint)) || !hasText(pill, holdname.Legend(choice)) {
		t.Fatalf("release hint missing: %q", pill.Texts())
	}
	snapshot(t, pill, "overlay-holding")
	state.Hold.Holding = false
	settle(pill)
	if !hasText(pill, en.Text(copy.PillFinishHint)) || !hasText(pill, holdname.Legend(choice)) {
		t.Fatalf("toggle hint missing after tap: %q", pill.Texts())
	}
}

func TestTranslationSettingsAndPill(t *testing.T) {
	state, actions, store := fixture()
	state.Preferences.TranslateKey = runtime.HoldOff
	state.Preferences.TranslateLanguage = runtime.TargetEnglish
	en := copy.For(copy.English)
	page := settings.NewPage()
	page.Section = settings.Dictation
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	if hasText(tt, en.Text(copy.TranslateKeyLabel)) || hasText(tt, en.Text(copy.TranslateLanguageLabel)) {
		t.Fatal("disabled translation retained settings")
	}
	if err := tt.Click(en.Text(copy.TranslateEnabled)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	for _, key := range []copy.Key{copy.TranslateGroup, copy.TranslateKeyLabel, copy.TranslateLanguageLabel, copy.TranslateEnglish} {
		if !hasText(tt, en.Text(key)) {
			t.Fatalf("missing translation control %s", key)
		}
	}
	tKey := runtime.DefaultTranslateKey(platform.GOOS)
	if !actions.called("translate-hold:"+string(tKey)) || state.Preferences.TranslateKey != tKey {
		t.Fatal("translation not enabled")
	}
	// Record a replacement using the same native capture projection as dictation.
	if err := tt.Click(en.Text(copy.TranslateKeyEdit)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tKey = "shift+f8"
	state.Hold.RecordedKey = tKey
	state.Hold.RecordedSeq++
	settle(tt)
	if state.Preferences.TranslateKey != tKey {
		t.Fatal("translation key not recorded")
	}

	// Select Simplified Chinese as the translation target language.
	if err := tt.Click(en.Text(copy.TranslateEnglish)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(en.Text(copy.TranslateChinese)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.TranslateLanguage != runtime.TargetChinese {
		t.Fatalf("translation language not saved: %q", state.Preferences.TranslateLanguage)
	}

	// Verify the pill shows the translation key legend during a translation session and explains translation failures.
	overlayPage := overlay.NewPage()
	pill := ui.NewTester(func(c *ui.Context) { overlayPage.Render(c, store) }, OverlayWidth, OverlayHeight)
	state.Dictation = runtime.Dictation{SessionID: "tr1", State: runtime.StateCapturing, Translating: true, Levels: levels(40), Changed: time.Now()}
	state.Hold.Holding = true
	settle(pill)
	if !hasText(pill, holdname.Legend(tKey)) || !hasText(pill, en.Text(copy.PillReleaseHint)) {
		t.Fatalf("translation release hint missing: %q", pill.Texts())
	}
	state.Dictation = runtime.Dictation{SessionID: "tr2", State: runtime.StateFailed, Translating: true, Failure: runtime.CodeRewriteTimeout, Changed: time.Now()}
	settle(pill)
	if !hasText(pill, en.Text(copy.FailTranslateTimeoutTitle)) || !hasText(pill, en.Text(copy.FailTranslateHint)) {
		t.Fatalf("translation timeout notice missing: %q", pill.Texts())
	}
}

func TestPermissionGuide(t *testing.T) {
	state, actions, store := fixture()
	tt := ui.NewTester(func(c *ui.Context) { guide.Render(c, store) }, 480, GuideHeight)
	settle(tt)
	if len(tt.Texts()) != 0 {
		t.Fatalf("guide drawn without a pane: %q", tt.Texts())
	}
	app, err := transfer.FileData(filepath.Join(t.TempDir(), "Arion.app"))
	if err != nil {
		t.Fatal(err)
	}
	state.Guide = runtime.Guide{Pane: runtime.GuideAccessibility, App: app, Shown: 1}
	rest(tt, 600*time.Millisecond)
	en := copy.For(copy.English)
	if !hasText(tt, en.Text(copy.GuideDrag)) || !hasText(tt, en.Text(copy.Title)) {
		t.Fatalf("guide: %q", tt.Texts())
	}
	snapshot(t, tt, "guide")
	if err := tt.Click(en.Text(copy.GuideClose)); err != nil {
		t.Fatalf("%v (texts %q)", err, tt.Texts())
	}
	if !actions.called("close-guide") {
		t.Fatal("close button did not close the guide")
	}

	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if !hasText(tt, copy.For(copy.SimplifiedChinese).Text(copy.GuideDrag)) {
		t.Fatalf("Chinese guide: %q", tt.Texts())
	}
	snapshot(t, tt, "guide-zh-CN")
	tt.SetDark(true)
	rest(tt, 200*time.Millisecond)
	snapshot(t, tt, "dark-guide")
}

func TestOnboardingInputMethodDeferral(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = runtime.UILanguage(locale)
		page := onboarding.NewPage()
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, OnboardingWidth, OnboardingHeight)
		settle(tt)
		messages := copy.For(locale)
		click := func(key copy.Key) {
			t.Helper()
			if err := tt.Click(messages.Text(key)); err != nil {
				t.Fatalf("%s: %v", key, err)
			}
			settle(tt)
		}
		for _, key := range []copy.Key{copy.OnbStart, copy.OnbContinueLocally, copy.Continue, copy.Continue} {
			click(key)
		}
		state.InputMethod = runtime.InputMethodPending
		settle(tt)
		snapshot(t, tt, "onboarding-input-method-pending-"+string(locale))
		state.InputMethodInstalling = true
		settle(tt)
		click(copy.Back)
		click(copy.OnbIMELater)
		if !hasText(tt, messages.Text(copy.OnbIMETitle)) {
			t.Fatal("left input setup while installation was in flight")
		}
		state.InputMethodInstalling = false
		state.InputMethod = runtime.InputMethodSelected
		settle(tt)
		snapshot(t, tt, "onboarding-input-method-selected-"+string(locale))
		state.InputMethod = runtime.InputMethodPending
		settle(tt)
		click(copy.OnbIMELater)
		if !hasText(tt, messages.Text(copy.OnbModelTitle)) || actions.called("install-ime") {
			t.Fatal("deferring setup installed an input source or changed dictation mode")
		}
	}
}

func TestSwitchPermissionGuides(t *testing.T) {
	for pane, key := range map[runtime.GuidePane]copy.Key{
		runtime.GuideMicrophone:      copy.GuideMicrophone,
		runtime.GuideScreenRecording: copy.GuideScreenRecording,
	} {
		state, actions, store := fixture()
		state.Guide = runtime.Guide{Pane: pane, Shown: 1}
		tt := ui.NewTester(func(c *ui.Context) { guide.Render(c, store) }, 360, guide.HeightFor(pane))
		for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
			state.Preferences.UILanguage = runtime.UILanguage(locale)
			rest(tt, 600*time.Millisecond)
			messages := copy.For(locale)
			if !hasText(tt, messages.Text(key)) || hasText(tt, messages.Text(copy.GuideDrag)) {
				t.Fatalf("switch guide had missing or redundant content: %q", tt.Texts())
			}
			for _, text := range tt.Texts() {
				if text == messages.Text(copy.Title) {
					t.Fatal("switch guide exposed a redundant app tile")
				}
			}
			snapshot(t, tt, "guide-"+string(pane)+"-"+string(locale))
		}
		if err := tt.Click(store.Copy().Text(copy.GuideClose)); err != nil || !actions.called("close-guide") {
			t.Fatalf("switch guide could not close: %v", err)
		}
	}
}

func TestKeyboardSpellingSettings(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(language), func(t *testing.T) {
			state, _, store := fixture()
			state.Preferences.UILanguage = language
			tt := ui.NewTester(func(c *ui.Context) {
				tokens := theme.Apply(c)
				ui.Column(c).Padding(16).Gap(12).Children(func() { keyboardui.Phonetics(c, store, tokens) })
			}, 600, 600)
			settle(tt)
			for _, key := range []copy.Key{copy.KbdTypoCorrection, copy.KbdFuzzyPinyin, copy.KbdFuzzyZZH, copy.KbdFuzzyIANIANG, copy.KbdFuzzyUANUANG} {
				if !hasText(tt, store.Copy().Text(key)) {
					t.Fatalf("missing spelling setting %s", key)
				}
			}
			snapshot(t, tt, "keyboard-spelling-"+string(language))
			row, ok := tt.Find(store.Copy().Text(copy.KbdTypoCorrection))
			if !ok {
				t.Fatal("typing correction row missing")
			}
			tt.ClickAt(560, row.Y+row.H)
			settle(tt)
			if !state.Preferences.Keyboard.TypoCorrection {
				t.Fatal("typing correction did not persist through preference action")
			}
			for _, pair := range []struct {
				key copy.Key
				x   float32
			}{{copy.KbdFuzzyZZH, 260}, {copy.KbdFuzzyIANIANG, 550}} {
				row, ok := tt.Find(store.Copy().Text(pair.key))
				if !ok {
					t.Fatalf("missing pair %s", pair.key)
				}
				tt.ClickAt(pair.x, row.Y+row.H/2)
				settle(tt)
			}
			if state.Preferences.Keyboard.FuzzyPinyin != runtime.FuzzyZZH|runtime.FuzzyIANIANG {
				t.Fatalf("pair toggles were not independent: %d", state.Preferences.Keyboard.FuzzyPinyin)
			}
			snapshot(t, tt, "keyboard-spelling-selected-"+string(language))
			row, _ = tt.Find(store.Copy().Text(copy.KbdFuzzyZZH))
			tt.ClickAt(260, row.Y+row.H/2)
			settle(tt)
			if state.Preferences.Keyboard.FuzzyPinyin != runtime.FuzzyIANIANG {
				t.Fatal("disabling one fuzzy pair changed another")
			}
		})
	}
}

func TestKeyboardSettingsGuide(t *testing.T) {
	state, actions, store := fixture()
	state.Guide = runtime.Guide{Pane: runtime.GuideKeyboard, Shown: 1}
	tt := ui.NewTester(func(c *ui.Context) { guide.Render(c, store) }, 360, GuideHeight)
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		state.Preferences.UILanguage = runtime.UILanguage(locale)
		rest(tt, 600*time.Millisecond)
		messages := copy.For(locale)
		if !hasText(tt, messages.Text(copy.GuideKeyboardAdd)) || hasText(tt, messages.Text(copy.GuideDrag)) {
			t.Fatalf("keyboard guide offered the wrong settings operation: %q", tt.Texts())
		}
		snapshot(t, tt, "guide-keyboard-"+string(locale))
	}
	if err := tt.Click(store.Copy().Text(copy.GuideClose)); err != nil || !actions.called("close-guide") {
		t.Fatalf("keyboard guide could not close: %v", err)
	}
}

// ADR 014: a learned correction shows with an undo, in both languages, and
// a running session hides it.
func TestLearnedNoticeUndo(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = runtime.UILanguage(locale)
		messages := copy.For(locale)
		state.Learned = runtime.Learned{ID: "e1", From: "jason", To: "Jayson", At: time.Now()}
		overlayPage := overlay.NewPage()
		pill := ui.NewTester(func(c *ui.Context) { overlayPage.Render(c, store) }, OverlayWidth, OverlayHeight)
		settle(pill)
		if !hasText(pill, messages.Text(copy.PillLearned)) || !hasText(pill, "jason") || !hasText(pill, "Jayson") {
			t.Fatalf("%s: notice missing: %q", locale, pill.Texts())
		}
		snapshot(t, pill, "overlay-learned-"+string(locale))
		if err := pill.Click(messages.Text(copy.Undo)); err != nil || !actions.called("undo:e1") {
			t.Fatalf("%s: undo not requested: %v %v", locale, err, actions.calls)
		}
		state.Dictation = runtime.Dictation{SessionID: "n", State: runtime.StateCapturing, Levels: levels(8), Changed: time.Now()}
		settle(pill)
		if hasText(pill, messages.Text(copy.PillLearned)) {
			t.Fatalf("%s: notice shown during a session", locale)
		}
	}
}

func TestAboutDiagnosticExport(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		state, actions, store := fixture()
		state.Preferences.UILanguage = locale
		messages := store.Copy()
		page := settings.NewPage()
		page.Section = settings.About
		tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
		settle(tt)
		tt.Scroll(SettingsWidth/2, SettingsHeight/2, 0, 230)
		settle(tt)
		if !hasText(tt, messages.Text(copy.DiagnosticsTitle)) {
			t.Fatalf("About missing diagnostics: %v", tt.Texts())
		}
		if err := tt.Click(messages.Text(copy.DiagnosticsExport)); err != nil || !actions.called("export-diagnostics") {
			t.Fatalf("export click %v %v", err, actions.calls)
		}
		snapshot(t, tt, "settings-about-diagnostics-"+string(locale))
		tt.Scroll(SettingsWidth/2, SettingsHeight/2, 0, 220)
		settle(tt)
		netTitle, ok := tt.Find(messages.Text(copy.PrivacyNetwork))
		if !ok || hasText(tt, messages.Text(copy.PrivacyNetTxt)) {
			t.Fatalf("privacy row missing or its detail inline: %v", tt.Texts())
		}
		snapshot(t, tt, "settings-about-privacy-"+string(locale))
		tt.Move(netTitle.X+netTitle.W+14, netTitle.Y+netTitle.H/2)
		for range 40 {
			tt.Frame()
			if hasText(tt, messages.Text(copy.PrivacyNetTxt)) {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		if !hasText(tt, messages.Text(copy.PrivacyNetTxt)) {
			t.Fatalf("privacy tooltip not shown: %v", tt.Texts())
		}
		snapshot(t, tt, "settings-about-privacy-tooltip-"+string(locale))
		// Exercise progress, result and error states at the actual content width.
		card := ui.NewTester(func(c *ui.Context) { diagnosticui.Card(c, store, theme.Apply(c)) }, 500, 180)
		for _, phase := range []runtime.DiagnosticsPhase{runtime.DiagnosticsIdle, runtime.DiagnosticsExporting, runtime.DiagnosticsSaved, runtime.DiagnosticsFailed} {
			state.Diagnostics.Phase = phase
			actions.calls = nil
			settle(card)
			label := copy.DiagnosticsExport
			if phase == runtime.DiagnosticsExporting {
				label = copy.DiagnosticsExporting
			}
			if !hasText(card, messages.Text(label)) {
				t.Fatalf("%s missing label: %v", phase, card.Texts())
			}
			err := card.Click(messages.Text(label))
			if phase == runtime.DiagnosticsExporting {
				if actions.called("export-diagnostics") {
					t.Fatal("busy button invoked export")
				}
			} else if err != nil || !actions.called("export-diagnostics") {
				t.Fatalf("%s click: %v", phase, err)
			}
			snapshot(t, card, "diagnostics-"+string(locale)+"-"+string(phase))
		}
	}
}

func TestInputMethodSetupStates(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(language), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = language
			state.InputMethod = runtime.InputMethodPending
			viewState := &keyboardui.State{}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Padding(16).Gap(12).Children(func() {
					keyboardui.Source(c, store, theme.Apply(c), viewState)
				})
			}, 620, 440)
			settle(tt)
			messages := store.Copy()
			if !hasText(tt, messages.Text(copy.IMEPending)) || !hasText(tt, messages.Text(copy.IMESetupHint)) || hasText(tt, messages.Text(copy.IMEInstalled)) {
				t.Fatalf("pending source misreported: %q", tt.Texts())
			}
			if err := tt.Click(messages.Text(copy.IMESetup)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if !actions.called("ime-settings") {
				t.Fatal("setup did not open system settings")
			}
			snapshot(t, tt, "input-method-pending-"+string(language))
			if err := tt.Click(messages.Text(copy.IMEEnable)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if !actions.called("activate-ime") || actions.called("install-ime") {
				t.Fatal("enable replaced installed files")
			}
			state.InputMethodActivating = true
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEActivating)) || hasText(tt, messages.Text(copy.IMEEnable)) || hasText(tt, messages.Text(copy.IMERemove)) {
				t.Fatalf("activation exposed conflicting controls: %q", tt.Texts())
			}
			if err := tt.Click(messages.Text(copy.IMECancelSetup)); err != nil {
				t.Fatal(err)
			}
			if !actions.called("cancel-ime-setup") {
				t.Fatal("cancel did not revoke setup")
			}
			snapshot(t, tt, "input-method-activating-"+string(language))
			state.InputMethodActivating = false
			state.InputMethod = runtime.InputMethodDiscovering
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEDiscovering)) {
				t.Fatal("undiscovered source collapsed into enablement")
			}
			state.InputMethod = runtime.InputMethodInstalled
			settle(tt)
			if err := tt.Click(messages.Text(copy.IMESelect)); err != nil {
				t.Fatal(err)
			}
			state.InputMethod = runtime.InputMethodSelected
			state.InputMethodError = copy.IMEConnectionFailed
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEEnable)) || !hasText(tt, messages.Text(copy.IMEConnectionFailed)) {
				t.Fatal("selected but disconnected source lost retry")
			}
			state.InputMethodError = ""
			state.InputMethod = runtime.InputMethodPending

			state.InputMethodInstalling = true
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEInstalling)) || hasText(tt, messages.Text(copy.IMESetup)) || hasText(tt, messages.Text(copy.IMERemove)) {
				t.Fatalf("busy source exposes competing actions: %q", tt.Texts())
			}
			state.InputMethodInstalling = false
			state.InputMethodRemoving = true
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMERemoving)) || slices.Contains(tt.Texts(), messages.Text(copy.IMERemove)) || slices.Contains(tt.Texts(), messages.Text(copy.IMEEnable)) {
				t.Fatalf("removal exposed competing controls: %q", tt.Texts())
			}
			snapshot(t, tt, "input-method-removing-"+string(language))
			state.InputMethodRemoving = false
			state.InputMethod = runtime.InputMethodInstalled
			state.InputMethodError = copy.IMERemovePending
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMERemovePending)) || !hasText(tt, messages.Text(copy.IMESetup)) || !hasText(tt, messages.Text(copy.IMERemove)) {
				t.Fatalf("pending removal lost retained source or recovery: %q", tt.Texts())
			}
			snapshot(t, tt, "input-method-removal-pending-"+string(language))
			state.InputMethod = runtime.InputMethodAbsent
			state.InputMethodError = copy.IMEFailed
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEFailed)) || !hasText(tt, messages.Text(copy.IMEInstall)) {
				t.Fatalf("failure lost retry: %q", tt.Texts())
			}
		})
	}
}

func TestWindowsKeyboardSettings(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(language), func(t *testing.T) {
			state, actions, store := fixture()
			state.Platform = "windows-amd64"
			state.Preferences.UILanguage = language
			page := settings.NewPage()
			page.Section = settings.Keyboard
			messages := store.Copy()
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			settle(tt)
			for _, key := range []copy.Key{copy.IMEWindowsName, copy.IMEWindowsInstallHint, copy.IMEInstall} {
				if !hasText(tt, messages.Text(key)) {
					t.Fatalf("Windows setup missing %s: %q", key, tt.Texts())
				}
			}
			for _, key := range []copy.Key{copy.KbdPageSize, copy.KbdCandidateSize, copy.KbdAppearance, copy.KbdAccent, copy.KbdEmojiSuggestions, copy.KbdSwitchLanguage} {
				if !hasText(tt, messages.Text(key)) {
					t.Fatalf("Windows keyboard setting missing %s", key)
				}
			}
			if hasText(tt, messages.Text(copy.IMESetupHint)) {
				t.Fatal("Windows setup exposes macOS instructions")
			}
			snapshot(t, tt, "settings-windows-keyboard-"+string(language))
			emojiRow, ok := tt.Find(messages.Text(copy.KbdEmojiSuggestions))
			if !ok || !state.Preferences.Keyboard.EmojiSuggestions {
				t.Fatal("Windows Emoji suggestions should start enabled")
			}
			tt.ClickAt(SettingsWidth-60, emojiRow.Y+emojiRow.H)
			settle(tt)
			if state.Preferences.Keyboard.EmojiSuggestions {
				t.Fatal("Windows Emoji preference did not switch off")
			}
			snapshot(t, tt, "settings-windows-keyboard-emoji-off-"+string(language))
			tt.ClickAt(SettingsWidth-60, emojiRow.Y+emojiRow.H)
			settle(tt)
			if !state.Preferences.Keyboard.EmojiSuggestions {
				t.Fatal("Windows Emoji preference did not switch on")
			}
			if err := tt.Click("9"); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if err := tt.Click("6"); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if state.Preferences.Keyboard.PageSize != 6 {
				t.Fatalf("Windows page-size preference was not saved: %+v", state.Preferences.Keyboard)
			}
			if err := tt.Click(messages.Text(copy.IMEInstall)); err != nil {
				t.Fatal(err)
			}
			if !actions.called("install-ime") {
				t.Fatalf("install action missing: %v", actions.calls)
			}
			state.InputMethod = runtime.InputMethodInstalled
			state.InputMethodUpdate = true
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEUpdateAvailable)) || hasText(tt, messages.Text(copy.IMEUpdate)) {
				t.Fatalf("Windows update requires an unnecessary manual action: %q", tt.Texts())
			}
			state.InputMethodUpdateFailed = true
			settle(tt)
			if hasText(tt, messages.Text(copy.IMEUpdateAvailable)) || !hasText(tt, messages.Text(copy.IMEUpdate)) {
				t.Fatalf("failed Windows update must offer a retry: %q", tt.Texts())
			}
			beforeRetry := len(actions.calls)
			if err := tt.Click(messages.Text(copy.IMEUpdate)); err != nil {
				t.Fatal(err)
			}
			if len(actions.calls) != beforeRetry+1 || actions.calls[beforeRetry] != "install-ime" {
				t.Fatal("Windows update retry did not invoke installation")
			}
			snapshot(t, tt, "settings-windows-keyboard-update-"+string(language))
			state.InputMethod = runtime.InputMethodPending
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEUpdate)) {
				t.Fatal("failed enablement after update lost its retry action")
			}
			state.InputMethodUpdateFailed = false
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEUpdate)) || hasText(tt, messages.Text(copy.IMEUpdateAvailable)) {
				t.Fatal("interrupted pending upgrade must offer explicit recovery without claiming automatic enablement")
			}
			state.InputMethodUpdate = false
			state.InputMethodUpdateFailed = false
			state.InputMethod = runtime.InputMethodPending
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEWindowsPending)) || hasText(tt, messages.Text(copy.IMEPending)) || hasText(tt, messages.Text(copy.IMEUpdate)) {
				t.Fatalf("Windows registration status uses the wrong platform copy: %q", tt.Texts())
			}
			for original, translated := range map[copy.Key]copy.Key{
				copy.IMEDiscoveryFailed: copy.IMEWindowsDiscoveryFailed,
				copy.IMEEnableFailed:    copy.IMEWindowsEnableFailed,
				copy.IMESettingsFailed:  copy.IMEWindowsSettingsFailed,
			} {
				state.InputMethodError = original
				settle(tt)
				if !hasText(tt, messages.Text(translated)) || hasText(tt, messages.Text(original)) {
					t.Fatalf("Windows failure uses the wrong platform copy: %q", tt.Texts())
				}
			}
		})
	}
}

func TestKeyboardSettings(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(language), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.UILanguage = language
			state.InputMethod = runtime.InputMethodSelected
			messages := store.Copy()
			page := settings.NewPage()
			page.Section = settings.Keyboard
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			settle(tt)
			for _, key := range []copy.Key{copy.IMESelected, copy.KbdPageSize, copy.KbdCandidateSize, copy.KbdAppearance, copy.KbdAccent,
				copy.KbdEmojiSuggestions, copy.KbdSwitchLanguage} {
				if !hasText(tt, messages.Text(key)) {
					t.Fatalf("keyboard section missing %s: %q", key, tt.Texts())
				}
			}
			if !state.Preferences.Keyboard.EmojiSuggestions {
				t.Fatal("Emoji suggestions should start enabled on macOS")
			}
			snapshot(t, tt, "settings-keyboard-"+string(language))
			emojiRow, ok := tt.Find(messages.Text(copy.KbdEmojiSuggestions))
			if !ok {
				t.Fatal("Emoji row not found")
			}
			tt.ClickAt(SettingsWidth-60, emojiRow.Y+emojiRow.H)
			settle(tt)
			if state.Preferences.Keyboard.EmojiSuggestions {
				t.Fatal("Emoji preference did not switch off")
			}
			snapshot(t, tt, "settings-keyboard-emoji-off-"+string(language))
			tt.ClickAt(SettingsWidth-60, emojiRow.Y+emojiRow.H)
			settle(tt)
			if !state.Preferences.Keyboard.EmojiSuggestions {
				t.Fatal("Emoji preference did not switch on")
			}
			if err := tt.Click("9"); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if err := tt.Click("6"); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if state.Preferences.Keyboard.PageSize != 6 {
				t.Fatalf("page size not chosen: %+v", state.Preferences.Keyboard)
			}
			if err := tt.Click(messages.Text(copy.KbdSizeMedium)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if err := tt.Click(messages.Text(copy.KbdSizeLarge)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if state.Preferences.Keyboard.CandidateSize != runtime.CandidateLarge {
				t.Fatalf("size not chosen: %+v", state.Preferences.Keyboard)
			}
			if err := tt.Click(messages.Text(copy.KbdAccentYellow)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			snapshot(t, tt, "settings-keyboard-accent-"+string(language))
			if err := tt.Click(messages.Text(copy.KbdAccentBlue)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if state.Preferences.Keyboard.Accent != runtime.AccentBlue {
				t.Fatalf("accent not chosen: %+v", state.Preferences.Keyboard)
			}
			// Custom starts from the current fill and adds a color well.
			if err := tt.Click(messages.Text(copy.KbdAccentBlue)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if err := tt.Click(messages.Text(copy.KbdAccentCustom)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if k := state.Preferences.Keyboard; k.Accent != runtime.AccentCustom || k.AccentColor != "#0088ff" {
				t.Fatalf("custom accent not started from the fill: %+v", k)
			}
			if _, ok := tt.Find(messages.Text(copy.KbdAccentCustomColor)); !ok {
				t.Fatalf("no custom color well: %q", tt.Texts())
			}
			snapshot(t, tt, "settings-keyboard-accent-custom-"+string(language))
			// A rebuilt app updates automatically; only a failed attempt offers retry.
			if hasText(tt, messages.Text(copy.IMEUpdate)) {
				t.Fatalf("update offered for the current build: %q", tt.Texts())
			}
			state.InputMethodUpdate = true
			settle(tt)
			if !hasText(tt, messages.Text(copy.IMEUpdateAvailable)) {
				t.Fatalf("update not explained: %q", tt.Texts())
			}
			snapshot(t, tt, "settings-keyboard-update-"+string(language))
			if hasText(tt, messages.Text(copy.IMEUpdate)) {
				t.Fatal("automatic update requires a manual action")
			}
			state.InputMethodUpdateFailed = true
			settle(tt)
			snapshot(t, tt, "settings-keyboard-update-retry-"+string(language))
			if err := tt.Click(messages.Text(copy.IMEUpdate)); err != nil || !actions.called("install-ime") {
				t.Fatalf("update click: %v %v", err, actions.calls)
			}
		})
	}
	// Without an input method on this system the section is not offered.
	state, _, store := fixture()
	state.InputMethod = runtime.InputMethodUnsupported
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	settle(tt)
	if hasText(tt, copy.For(copy.English).Text(copy.NavKeyboard)) {
		t.Fatalf("unsupported system offers Keyboard: %q", tt.Texts())
	}
}

func TestDictationHidesDeliveryInternals(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, platform := range []string{"darwin-arm64", "windows-amd64"} {
			for _, source := range []runtime.InputMethodState{runtime.InputMethodAbsent, runtime.InputMethodInstalled, runtime.InputMethodSelected} {
				state, _, store := fixture()
				state.Preferences.UILanguage = locale
				state.Platform, state.InputMethod = platform, source
				page := settings.NewPage()
				page.Section = settings.Dictation
				tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
				settle(tt)
				messages := store.Copy()
				for _, key := range []copy.Key{copy.ModeLabel, copy.ModeShortcut, copy.ModeIME, copy.DeliveryLabel, copy.DeliveryAutomatic, copy.DeliveryClipboard, copy.DeliveryMacDetail} {
					if slices.Contains(tt.Texts(), messages.Text(key)) {
						t.Fatalf("%s/%s/%s exposes delivery internals: %q", locale, platform, source, tt.Texts())
					}
				}
				if !hasText(tt, messages.Text(copy.HoldKeyLabel)) {
					t.Fatalf("%s/%s lost the dictation key", locale, platform)
				}
			}
		}
	}
}

func TestKeyboardNeuralSettings(t *testing.T) {
	for _, language := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, mode := range []string{"off", "missing", "loading", "ready", "unavailable"} {
			t.Run(string(language)+"/"+mode, func(t *testing.T) {
				state, _, store := fixture()
				state.Preferences.UILanguage = language
				state.Preferences.Keyboard.NeuralRanking = mode != "off"
				state.Polish.Installed = mode != "missing"
				state.Polish.State = map[string]runtime.PolishState{"off": runtime.PolishOff, "missing": runtime.PolishOff, "loading": runtime.PolishLoading, "ready": runtime.PolishReady, "unavailable": runtime.PolishFailed}[mode]
				tt := ui.NewTester(func(c *ui.Context) {
					tokens := theme.Apply(c)
					ui.Column(c).Padding(16).Gap(12).Children(func() { keyboardui.Neural(c, store, tokens) })
				}, 600, 260)
				settle(tt)
				if !hasText(tt, store.Copy().Text(copy.KbdNeuralRanking)) {
					t.Fatal("missing ranking control")
				}
				snapshot(t, tt, "keyboard-neural-"+mode+"-"+string(language))
				row, ok := tt.Find(store.Copy().Text(copy.KbdNeuralRanking))
				if !ok {
					t.Fatal("missing ranking row")
				}
				tt.ClickAt(560, row.Y+row.H)
				settle(tt)
				if state.Preferences.Keyboard.NeuralRanking != (mode == "off") {
					t.Fatal("ranking preference did not toggle")
				}
			})
		}
	}
}
