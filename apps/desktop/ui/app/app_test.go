package app

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/overlay"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
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
	recommended := runtime.Bundle{ID: "xasr-160-xasr-final", Name: "X-ASR 160 ms + X-ASR final pass", Evaluated: true, Recommended: true,
		ErrorRates: map[string]float64{"zh": 0.041, "en": 0.052, "mixed": 0.087}, ReviewP95Ms: 312, SizeBytes: 270 << 20}
	other := runtime.Bundle{ID: "xasr-160", Name: "X-ASR streaming 160 ms", Evaluated: true,
		ErrorRates: map[string]float64{"zh": 0.06, "en": 0.07, "mixed": 0.11}, ReviewP95Ms: 41, SizeBytes: 134 << 20}
	state := &runtime.State{
		Version:     "0.1.0",
		Permissions: runtime.Permissions{Microphone: runtime.ConsentGranted, Accessibility: runtime.ConsentGranted},
		Preferences: runtime.Preferences{Mode: runtime.ModeShortcut, Shortcut: "Alt+Space", Delivery: runtime.DeliveryAutomatic, RemoveFillers: true, Sounds: true},
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
		{"listening", runtime.Dictation{SessionID: "1", State: runtime.StateCapturing, Levels: levels(40), Changed: now}, copy.PillListening},
		{"draft", runtime.Dictation{SessionID: "1", State: runtime.StateCapturing, Levels: levels(40), Draft: "把这个 PR merge 到 main 分支之前先跑一下 CI", Changed: now}, "merge"},
		{"transcribing", runtime.Dictation{SessionID: "1", State: runtime.StateFinalizing, Draft: "今天的 standup 改到上午十点半", Changed: now}, "standup"},
		{"inserted", runtime.Dictation{SessionID: "1", State: runtime.StateCompleted, Outcome: runtime.OutcomeApplied, Changed: now}, copy.PillInserted},
		{"typed", runtime.Dictation{SessionID: "2", State: runtime.StateCompleted, Outcome: runtime.OutcomeUnknown, Changed: now}, copy.PillTyped},
		{"focus-moved", runtime.Dictation{SessionID: "3", State: runtime.StateFailed, Failure: runtime.CodeTargetChanged, Changed: now}, copy.FailTargetTitle},
		{"no-speech", runtime.Dictation{SessionID: "4", State: runtime.StateFailed, Failure: runtime.CodeNoSpeech, Changed: now}, copy.FailNoSpeechTitle},
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
	if hasText(tt, copy.PillInserted) {
		t.Error("stale outcome still visible")
	}
	// A denied microphone is named precisely.
	state.Permissions.Microphone = runtime.ConsentDenied
	state.Dictation = runtime.Dictation{SessionID: "6", State: runtime.StateFailed, Failure: runtime.CodePermissionDenied, Changed: time.Now()}
	settle(tt)
	if !hasText(tt, copy.FailMicTitle) {
		t.Errorf("permission failure not explained: %q", tt.Texts())
	}
}

func TestMenuBarPanel(t *testing.T) {
	state, actions, store := fixture()
	panel := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeight)
	settle(tt)
	if !hasText(tt, copy.StatusReady) || !hasText(tt, copy.PanelEmpty) {
		t.Fatalf("idle panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-ready")
	if err := tt.Click(copy.StartDictate); err != nil {
		t.Fatal(err)
	}
	if !actions.called("toggle") {
		t.Fatal("start button did not toggle dictation")
	}
	state.Dictation = runtime.Dictation{SessionID: "1", State: runtime.StateCapturing, Levels: levels(40), Changed: time.Now()}
	state.LastText = "Ship the release notes before Friday."
	settle(tt)
	if !hasText(tt, copy.StopDictate) || !hasText(tt, copy.StatusListening) {
		t.Fatalf("live panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-listening")
	if err := tt.Click(copy.Copy); err != nil {
		t.Fatal(err)
	}
	if !actions.called("copy:Ship the release notes before Friday.") {
		t.Fatal("copy did not copy the last dictation")
	}
	state.Dictation = runtime.Dictation{}
	state.Model = runtime.Model{State: runtime.ModelProvisioning, Phase: runtime.PhaseDownloading, Done: 63 << 20, Total: 134 << 20, Active: state.Model.Active, Automatic: true}
	settle(tt)
	if !hasText(tt, copy.StatusPreparing) || !hasText(tt, "63 MB of 134 MB") {
		t.Fatalf("provisioning panel: %q", tt.Texts())
	}
	snapshot(t, tt, "panel-downloading")
}

func TestSettingsSections(t *testing.T) {
	state, actions, store := fixture()
	page := settings.NewPage()
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
	for _, section := range []struct{ id, label, want string }{
		{settings.General, copy.NavGeneral, copy.LaunchAtLogin},
		{settings.Dictation, copy.NavDictation, copy.ShortcutLabel},
		{settings.Microphone, copy.NavMicrophone, copy.MicTest},
		{settings.Model, copy.NavModel, copy.ModelReason},
		{settings.Vocabulary, copy.NavVocabulary, copy.VocabEmpty},
		{settings.Privacy, copy.NavPrivacy, copy.PrivacyLocal},
		{settings.About, copy.NavAbout, copy.AboutCredits},
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
	if err := tt.Click(copy.NavVocabulary); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(copy.VocabFromLabel); err != nil {
		t.Fatal(err)
	}
	tt.Type("阿里昂")
	if err := tt.Click(copy.VocabToLabel); err != nil {
		t.Fatal(err)
	}
	tt.Type("Arion")
	if err := tt.Click(copy.Add); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if len(state.Vocabulary) != 1 || state.Vocabulary[0].To != "Arion" || !hasText(tt, "阿里昂") {
		t.Fatalf("vocabulary not added: %v %q", state.Vocabulary, tt.Texts())
	}
	snapshot(t, tt, "settings-vocabulary-filled")
	// Dictation: record a new shortcut.
	if err := tt.Click(copy.NavDictation); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if err := tt.Click(copy.ShortcutRecord); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Key(ui.Super|ui.Shift, ui.KeyD)
	settle(tt)
	if state.Shortcut.Accelerator != "Cmd+Shift+D" {
		t.Fatalf("shortcut not recorded: %q calls=%v", state.Shortcut.Accelerator, actions.calls)
	}
	// Plain letters are rejected so typing is never captured.
	if err := tt.Click(copy.ShortcutRecord); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	tt.Key(0, ui.KeyD)
	settle(tt)
	if !hasText(tt, copy.ShortcutNeedsModifier) {
		t.Fatalf("modifier rule not shown: %q", tt.Texts())
	}
	snapshot(t, tt, "settings-dictation-recording")
	// The input method is installed on request, then chosen as the mode.
	state.InputMethod = runtime.InputMethodAbsent
	settle(tt)
	if err := tt.Click(copy.IMEInstall); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if !actions.called("install-ime") || !hasText(tt, copy.IMEInstalled) {
		t.Fatalf("input method install: %v %q", actions.calls, tt.Texts())
	}
	if err := tt.Click(copy.ModeIME); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.Mode != runtime.ModeSystemIME {
		t.Fatalf("mode not changed: %v", state.Preferences.Mode)
	}
	snapshot(t, tt, "settings-dictation-input-method")
	if hasText(tt, copy.DeliveryClipboard) {
		t.Fatal("shortcut-only insertion options shown in input-method mode")
	}
	if err := tt.Click(copy.ModeShortcut); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	// Switching insertion to paste persists through the preferences action.
	tt.Key(0, ui.KeyEscape)
	settle(tt)
	if err := tt.Click(copy.DeliveryClipboard); err != nil {
		t.Fatal(err)
	}
	settle(tt)
	if state.Preferences.Delivery != runtime.DeliveryClipboard {
		t.Fatalf("delivery not changed: %v", state.Preferences.Delivery)
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
	click(copy.OnbStart)
	if !hasText(tt, copy.OnbMicTitle) {
		t.Fatalf("microphone step missing: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-2-microphone")
	// Continue stays disabled until consent exists.
	click(copy.Continue)
	if !hasText(tt, copy.OnbMicTitle) {
		t.Fatal("advanced without microphone consent")
	}
	click(copy.OnbMicAllow)
	if !actions.called("request-mic") {
		t.Fatal("microphone not requested")
	}
	state.Permissions.Microphone = runtime.ConsentGranted
	state.MonitorLevels = levels(40)
	settle(tt)
	snapshot(t, tt, "onboarding-2-microphone-granted")
	click(copy.Continue)
	snapshot(t, tt, "onboarding-3-accessibility")
	click(copy.OnbAXAllow)
	if !actions.called("request-ax") || !hasText(tt, copy.OnbAXWaiting) {
		t.Fatalf("accessibility request: %v %q", actions.calls, tt.Texts())
	}
	snapshot(t, tt, "onboarding-3-accessibility-waiting")
	state.Permissions.Accessibility = runtime.ConsentGranted
	deadline := time.Now().Add(2 * time.Second)
	for !hasText(tt, copy.OnbModelTitle) && time.Now().Before(deadline) {
		tt.Frame()
		time.Sleep(20 * time.Millisecond)
	}
	if !hasText(tt, copy.OnbModelTitle) {
		t.Fatalf("did not advance after accessibility grant: %q", tt.Texts())
	}
	settle(tt)
	snapshot(t, tt, "onboarding-4-model")
	click(copy.Continue)
	if !actions.called("test-shortcut") {
		t.Fatal("shortcut test mode not entered")
	}
	snapshot(t, tt, "onboarding-5-shortcut")
	state.Shortcut.Pressed = time.Now().Add(time.Second)
	settle(tt)
	if !hasText(tt, copy.OnbShortcutHeard) {
		t.Fatalf("shortcut press not acknowledged: %q", tt.Texts())
	}
	snapshot(t, tt, "onboarding-5-shortcut-heard")
	click(copy.Continue)
	snapshot(t, tt, "onboarding-6-try")
	click(copy.Continue)
	snapshot(t, tt, "onboarding-7-done")
	click(copy.OnbFinish)
	if !actions.called("complete") {
		t.Fatal("onboarding not completed")
	}
}
