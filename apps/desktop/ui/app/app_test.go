package app

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	platform "runtime"
	"strings"
	"testing"
	"time"

	"arion.local/desktop/ui/features/holdkey"
	"arion.local/desktop/ui/pages/guide"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/overlay"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// recorder implements runtime.Actions for view tests.
type recorder struct {
	calls    []string
	state    *runtime.State
	shortcut error
}

func (r *recorder) call(name string)            { r.calls = append(r.calls, name) }
func (r *recorder) ToggleDictation()            { r.call("toggle") }
func (r *recorder) CancelDictation()            { r.call("cancel") }
func (r *recorder) RequestMicrophone()          { r.call("request-mic") }
func (r *recorder) RequestAccessibility()       { r.call("request-ax") }
func (r *recorder) OpenMicrophoneSettings()     { r.call("open-mic") }
func (r *recorder) OpenAccessibilitySettings()  { r.call("open-ax") }
func (r *recorder) RetryModel()                 { r.call("retry") }
func (r *recorder) RetryPolish()                { r.call("retry-polish") }
func (r *recorder) UseBundle(id string)         { r.call("use:" + id) }
func (r *recorder) UseRecommendedBundle()       { r.call("use-recommended") }
func (r *recorder) RefreshDevices()             { r.call("devices") }
func (r *recorder) StartLevelMonitor()          { r.call("monitor-start") }
func (r *recorder) StopLevelMonitor()           { r.call("monitor-stop") }
func (r *recorder) CopyText(text string)        { r.call("copy:" + text) }
func (r *recorder) OpenSettings()               { r.call("settings") }
func (r *recorder) CompleteOnboarding()         { r.call("complete") }
func (r *recorder) Quit()                       { r.call("quit") }
func (r *recorder) SuspendShortcut(s bool)      { r.call("suspend") }
func (r *recorder) TestShortcut(testing bool)   { r.call("test-shortcut") }
func (r *recorder) RemoveReplacement(id string) { r.call("remove:" + id) }
func (r *recorder) RequestInputMonitoring()     { r.call("request-monitoring") }
func (r *recorder) CloseGuide()                 { r.call("close-guide") }
func (r *recorder) CheckForUpdates()            { r.call("check-update") }
func (r *recorder) InstallUpdate()              { r.call("install-update") }
func (r *recorder) RestartToUpdate()            { r.call("restart-update") }
func (r *recorder) OpenUpdate() {
	r.call("open-update")
	r.state.Focus = runtime.SettingsFocus{Section: settings.About, Seq: r.state.Focus.Seq + 1}
}
func (r *recorder) SetHoldKey(key runtime.HoldKey) {
	r.call("hold:" + string(key))
	r.state.Preferences.HoldKey = key
}
func (r *recorder) InstallInputMethod() error {
	r.call("install-ime")
	r.state.InputMethod = runtime.InputMethodInstalled
	return nil
}
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
func (r *recorder) SetShortcut(accelerator string) error {
	r.call("shortcut:" + accelerator)
	if r.shortcut != nil {
		return r.shortcut
	}
	r.state.Shortcut.Accelerator = accelerator
	return nil
}
func (r *recorder) AddReplacement(from, to string) error {
	if from == "" || to == "" {
		return errors.New("invalid")
	}
	r.state.Vocabulary = append(r.state.Vocabulary, runtime.Replacement{ID: from, From: from, To: to})
	r.call("add")
	return nil
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
	recommended := runtime.Bundle{ID: "xasr-160-xasr-final", Evaluated: true, Recommended: true,
		ErrorRates: map[string]float64{"zh": 0.041, "en": 0.052, "mixed": 0.087}, ReviewP95Ms: 312, SizeBytes: 270 << 20}
	other := runtime.Bundle{ID: "xasr-160", Evaluated: true,
		ErrorRates: map[string]float64{"zh": 0.06, "en": 0.07, "mixed": 0.11}, ReviewP95Ms: 41, SizeBytes: 134 << 20}
	state := &runtime.State{
		Version:     "0.1.0",
		Permissions: runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted},
		Preferences: runtime.Preferences{Mode: runtime.ModeShortcut, Shortcut: "Alt+Space", Delivery: runtime.DeliveryAutomatic, RemoveFillers: true, Sounds: true, CheckUpdates: true},
		InputMethod: runtime.InputMethodAbsent,
		Shortcut:    runtime.Shortcut{Accelerator: "Alt+Space", Registered: true},
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
	tt := ui.NewTester(func(c *ui.Context) { overlay.Render(c, store) }, OverlayWidth, OverlayHeight)
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
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.StatusReady)) || !hasText(tt, copy.For(copy.English).Text(copy.PanelEmpty)) {
		t.Fatalf("idle panel: %q", tt.Texts())
	}
	if hasText(tt, copy.For(copy.English).Text(copy.ModelNameAccurate)) {
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
	state.LastText = "Ship the release notes before Friday."
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.StopDictate)) || !hasText(tt, copy.For(copy.English).Text(copy.StatusListening)) {
		t.Fatalf("live panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-listening")
	if err := tt.Click(copy.For(copy.English).Text(copy.Copy)); err != nil {
		t.Fatal(err)
	}
	if !actions.called("copy:Ship the release notes before Friday.") {
		t.Fatal("copy did not copy the last dictation")
	}
	state.Dictation = runtime.Dictation{}
	state.Model = runtime.Model{State: runtime.ModelProvisioning, Phase: runtime.PhaseDownloading, Done: 63 << 20, Total: 134 << 20, Active: state.Model.Active, Automatic: true}
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.StatusPreparing)) || !hasText(tt, "63 MB of 134 MB") {
		t.Fatalf("provisioning panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-downloading")

	// A failed download offers its retry above the footer, under the
	// longest last dictation.
	state.LastText = strings.Repeat("Ship the release notes before Friday. ", 12)
	state.Model = runtime.Model{State: runtime.ModelFailed, Failure: runtime.CodeProvisioningFailed, Active: state.Model.Active, Automatic: true}
	settle(tt)
	en := copy.For(copy.English)
	retry, ok := tt.Find(en.Text(copy.Retry))
	quit, _ := tt.Find(en.Text(copy.Quit))
	if !ok || retry.Y+retry.H > quit.Y || quit.Y+quit.H > PanelHeight {
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
		{settings.General, copy.For(copy.English).Text(copy.NavGeneral), copy.For(copy.English).Text(copy.LaunchAtLogin)},
		{settings.Dictation, copy.For(copy.English).Text(copy.NavDictation), copy.For(copy.English).Text(copy.ShortcutLabel)},
		{settings.Microphone, copy.For(copy.English).Text(copy.NavMicrophone), copy.For(copy.English).Text(copy.MicTest)},
		{settings.Model, copy.For(copy.English).Text(copy.NavModel), copy.For(copy.English).Text(copy.ModelReason)},
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
	if err := tt.Click(copy.For(copy.English).Text(copy.VocabFromLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Type("阿里昂")
	if err := tt.Click(copy.For(copy.English).Text(copy.VocabToLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Type("Arion")
	if err := tt.Click(copy.For(copy.English).Text(copy.Add)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if len(state.Vocabulary) != 1 || state.Vocabulary[0].To != "Arion" || !hasText(tt, "阿里昂") {
		t.Fatalf("vocabulary not added: %v %q", state.Vocabulary, tt.Texts())
	}
	snapshot(t, tt, "settings-vocabulary-filled")
	// Dictation: record a new shortcut.
	if err := tt.Click(copy.For(copy.English).Text(copy.NavDictation)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(copy.For(copy.English).Text(copy.ShortcutRecord)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Key(ui.Super|ui.Shift, ui.KeyD)
	settle(tt)
	if state.Shortcut.Accelerator != "Cmd+Shift+D" {
		t.Fatalf("shortcut not recorded: %q calls=%v", state.Shortcut.Accelerator, actions.calls)
	}
	// Plain letters are rejected so typing is never captured.
	if err := tt.Click(copy.For(copy.English).Text(copy.ShortcutRecord)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Key(0, ui.KeyD)
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(modifierHint())) {
		t.Fatalf("modifier rule not shown: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-dictation-recording")
	// The input method is installed on request, then chosen as the mode.
	state.InputMethod = runtime.InputMethodAbsent
	settle(tt)
	if err := tt.Click(copy.For(copy.English).Text(copy.IMEInstall)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !actions.called("install-ime") || !hasText(tt, copy.For(copy.English).Text(copy.IMEInstalled)) {
		t.Fatalf("input method install: %v %q", actions.calls, tt.Texts())
	}
	if err := tt.Click(copy.For(copy.English).Text(copy.ModeIME)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.Mode != runtime.ModeSystemIME {
		t.Fatalf("mode not changed: %v", state.Preferences.Mode)
	}
	snapshot(t, tt, "settings-dictation-input-method")
	if hasText(tt, copy.For(copy.English).Text(copy.DeliveryClipboard)) {
		t.Fatal("shortcut-only insertion options shown in input-method mode")
	}
	if err := tt.Click(copy.For(copy.English).Text(copy.ModeShortcut)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	// Switching insertion to paste persists through the preferences action.
	tt.Key(0, ui.KeyEscape)
	settle(tt)
	reveal(t, tt, copy.For(copy.English).Text(copy.DeliveryClipboard))
	if err := tt.Click(copy.For(copy.English).Text(copy.DeliveryClipboard)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.Delivery != runtime.DeliveryClipboard {
		t.Fatalf("delivery not changed: %v", state.Preferences.Delivery)
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
		if !hasText(tt, messages.Text(copy.MicGainDetail)) || !hasText(tt, messages.Decibels(0)) {
			t.Fatalf("%s: gain row missing: %q", locale, tt.Texts())
		}
		snapshot(t, tt, "settings-microphone-gain-"+string(locale))
		// The slider spans the card under its row; its far end is the loudest stop.
		detail, _ := tt.Find(messages.Text(copy.MicGainDetail))
		readout, _ := tt.Find(messages.Decibels(0))
		tt.ClickAt(readout.X+readout.W-14, detail.Y+detail.H+12+14)
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
		{runtime.Update{Phase: runtime.UpdateCurrent}, en.Text(copy.UpdateCurrent), en.Text(copy.UpdateCheck), "check-update"},
		{runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}, en.UpdateAvailable("0.2.0"), en.Text(copy.UpdateInstall), "install-update"},
		{runtime.Update{Phase: runtime.UpdateDownloading, Version: "0.2.0", Downloaded: 3 << 20, Total: 12 << 20}, en.Progress(3<<20, 12<<20), "", ""},
		{runtime.Update{Phase: runtime.UpdateReady, Version: "0.2.0"}, en.Text(copy.UpdateReady), en.Text(copy.UpdateRestart), "restart-update"},
		{runtime.Update{Phase: runtime.UpdateFailed, Failed: runtime.UpdateStepCheck}, en.Text(copy.UpdateCheckFailed), en.Text(copy.Retry), "check-update"},
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
}

func TestUpdateNotice(t *testing.T) {
	state, actions, store := fixture()
	panel := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
	en := copy.For(copy.English)
	settle(tt)
	if hasText(tt, en.UpdateTo("0.2.0")) {
		t.Fatal("notice without an update")
	}
	state.Update = runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}
	settle(tt)
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
		{"panel", func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight},
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
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbMicTitle)) {
		t.Fatalf("microphone step missing: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-2-microphone")
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
	snapshot(t, tt, "onboarding-2-microphone-granted")
	click(copy.For(copy.English).Text(copy.Continue))
	snapshot(t, tt, "onboarding-3-accessibility")
	click(copy.For(copy.English).Text(copy.OnbAXAllow))
	if !actions.called("request-ax") || !hasText(tt, copy.For(copy.English).Text(copy.OnbAXWaiting)) {
		t.Fatalf("accessibility request: %v %q", actions.calls, tt.Texts())
	}
	snapshot(t, tt, "onboarding-3-accessibility-waiting")
	state.Permissions.Accessibility = runtime.ConsentGranted
	deadline := time.Now().Add(2 * time.Second)
	for !hasText(tt, copy.For(copy.English).Text(copy.OnbModelTitle)) && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(20 * time.Millisecond)
	}
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbModelTitle)) {
		t.Fatalf("did not advance after accessibility grant: %q", tt.Texts())
	}
	settle(tt)
	snapshot(t, tt, "onboarding-4-model")
	click(copy.For(copy.English).Text(copy.Continue))
	if !actions.called("test-shortcut") {
		t.Fatal("shortcut test mode not entered")
	}
	snapshot(t, tt, "onboarding-5-shortcut")
	state.Shortcut.Pressed = time.Now().Add(time.Second)
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(copy.OnbShortcutHeard)) {
		t.Fatalf("shortcut press not acknowledged: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-5-shortcut-heard")
	click(copy.For(copy.English).Text(copy.Continue))
	snapshot(t, tt, "onboarding-6-try")
	click(copy.For(copy.English).Text(copy.Continue))
	snapshot(t, tt, "onboarding-7-done")
	click(copy.For(copy.English).Text(copy.OnbFinish))
	if !actions.called("complete") {
		t.Fatal("onboarding not completed")
	}
}

func TestInterfaceLanguageSwitch(t *testing.T) {
	state, actions, store := fixture()
	page := settings.NewPage()
	panel := &menubar.Panel{}
	onboardingPage := onboarding.NewPage()
	settingsTester := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	panelTester := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
	onboardingTester := ui.NewTester(func(c *ui.Context) { onboardingPage.Render(c, store) }, OnboardingWidth, OnboardingHeight)
	state.LastText = "User text {index}: keep exactly."
	state.Dictation = runtime.Dictation{SessionID: "active", State: runtime.StateCapturing, Draft: state.LastText, Changed: time.Now()}
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
	if state.Dictation.SessionID != "active" || state.Dictation.Draft != state.LastText || !hasText(panelTester, state.LastText) {
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
	state, _, store := fixture()
	state.Preferences.UILanguage = runtime.LanguageChinese
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	zh := copy.For(copy.SimplifiedChinese)
	for _, section := range []struct {
		id  string
		key copy.Key
	}{
		{settings.General, copy.LanguageLabel}, {settings.Dictation, copy.ShortcutLabel},
		{settings.Microphone, copy.MicTest}, {settings.Model, copy.ModelReason},
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
	if err := tt.Click("添加"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !hasText(tt, "请填写两个输入框") {
		t.Fatalf("missing localized validation: %q", tt.Texts())
	}
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	if !hasText(tt, "Both fields are needed") || hasText(tt, "请填写两个输入框") {
		t.Fatalf("stale validation after switch: %q", tt.Texts())
	}

	// A draft entered in English survives the language change and is saved verbatim.
	if err := tt.Click("Heard as"); err != nil {
		t.Fatal(err)
	}
	tt.Type("阿里昂 {value}")
	if err := tt.Click("Write as"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Arion {value}")
	state.Preferences.UILanguage = runtime.LanguageChinese
	settle(tt)
	if err := tt.Click("添加"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if len(state.Vocabulary) != 1 || state.Vocabulary[0].From != "阿里昂 {value}" || state.Vocabulary[0].To != "Arion {value}" {
		t.Fatalf("language switch changed vocabulary draft: %v", state.Vocabulary)
	}
	page.Section = settings.Dictation
	settle(tt)
	if err := tt.Click("修改"); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Key(0, ui.KeyD)
	settle(tt)
	if !hasText(tt, zh.Text(modifierHint())) {
		t.Fatal("shortcut error untranslated")
	}
	state.Preferences.UILanguage = runtime.LanguageEnglish
	settle(tt)
	if !hasText(tt, copy.For(copy.English).Text(modifierHint())) {
		t.Fatal("shortcut error did not change language")
	}
}

func TestChineseOverlayAndDownload(t *testing.T) {
	state, _, store := fixture()
	state.Preferences.UILanguage = runtime.LanguageChinese
	tt := ui.NewTester(func(c *ui.Context) { overlay.Render(c, store) }, OverlayWidth, OverlayHeight)
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
	panelTester := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
	settle(panelTester)
	if !hasText(panelTester, "正在下载") || !hasText(panelTester, "63 MB / 134 MB") {
		t.Fatalf("untranslated download: %q", panelTester.Texts())
	}
}

func modifierHint() copy.Key {
	if platform.GOOS == "windows" {
		return copy.ShortcutNeedsModifierWindows
	}
	return copy.ShortcutNeedsModifier
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
	choice := holdkey.Choices()[1]
	if err := tt.Click(en.Text(copy.HoldKeyOff)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(holdkey.Name(en, choice)); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !actions.called("hold:"+string(choice)) || state.Preferences.HoldKey != choice {
		t.Fatalf("hold key not chosen: %v", actions.calls)
	}
	state.Shortcut.HoldFailure = runtime.CodePermissionDenied
	settle(tt)
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

	pill := ui.NewTester(func(c *ui.Context) { overlay.Render(c, store) }, OverlayWidth, OverlayHeight)
	state.Dictation = runtime.Dictation{SessionID: "h", State: runtime.StateCapturing, Levels: levels(40), Changed: time.Now()}
	state.Shortcut.Holding = true
	settle(pill)
	if !hasText(pill, en.Text(copy.PillReleaseHint)) || !hasText(pill, holdkey.Name(en, choice)) {
		t.Fatalf("release hint missing: %q", pill.Texts())
	}
	snapshot(t, pill, "overlay-holding")
	state.Shortcut.Holding = false
	settle(pill)
	if !hasText(pill, en.Text(copy.PillShortcutHint)) {
		t.Fatalf("toggle hint missing after tap: %q", pill.Texts())
	}
}

func TestPermissionGuide(t *testing.T) {
	state, actions, store := fixture()
	tt := ui.NewTester(func(c *ui.Context) { guide.Render(c, store) }, 480, GuideHeight)
	settle(tt)
	if len(tt.Texts()) != 0 {
		t.Fatalf("guide drawn without a pane: %q", tt.Texts())
	}
	app, err := transfer.FileData("/Applications/Arion.app")
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
