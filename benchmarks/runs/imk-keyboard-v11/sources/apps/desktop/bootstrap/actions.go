package bootstrap

import (
	"strings"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// actions implements runtime.Actions. Views call it on the main thread.
type actions struct{ a *App }

func (x actions) ToggleDictation()        { x.a.record("dictation_toggle", nil); x.a.core.Toggle() }
func (x actions) CancelDictation()        { x.a.record("dictation_cancel", nil); x.a.core.Cancel() }
func (x actions) RequestMicrophone()      { x.a.platform.RequestMicrophone() }
func (x actions) OpenMicrophoneSettings() { x.a.platform.OpenMicrophoneSettings() }
func (x actions) CloseGuide()             { x.a.closeGuide(true) }
func (x actions) RetryModel()             { go x.a.activate(x.a.state.Preferences.ModelOverride) }
func (x actions) RetryPolish()            { go x.a.startPolish() }
func (x actions) StartLevelMonitor()      { x.a.startMonitor() }
func (x actions) StopLevelMonitor()       { x.a.stopMonitor() }
func (x actions) RefreshDevices()         { go x.a.refreshDevicesAsync() }
func (x actions) CopyText(text string)    { mygo.Clipboard.WriteText(text) }
func (x actions) OpenSettings()           { x.a.openSettings() }
func (x actions) CheckForUpdates()        { x.a.updates.check(time.Now(), true) }
func (x actions) InstallUpdate()          { x.a.updates.install() }
func (x actions) RestartToUpdate()        { mygo.App.Relaunch() }

// OpenUpdate shows Settings › About, where the update is installed.
func (x actions) OpenUpdate() {
	x.a.state.Focus = arion.SettingsFocus{Section: settings.About, Seq: x.a.state.Focus.Seq + 1}
	x.a.openSettings()
}
func (x actions) Quit() { mygo.App.Quit() }

// ChooseScene stores the user's scene for the panel's target place.
func (x actions) ChooseScene(scene domain.Scene) {
	key := domain.SceneKey(x.a.state.SceneTarget.Hints)
	if key == "" {
		return
	}
	x.UpdatePreferences(func(p *application.Preferences) { p.SceneOverrides = p.SceneOverrides.WithScene(key, scene) })
}
func (x actions) TestShortcut(testing bool) { x.a.testingShortcut = testing }

func (x actions) UseBundle(id string) {
	x.UpdatePreferences(func(p *application.Preferences) { p.ModelOverride = id })
	go x.a.activate(id)
}

func (x actions) UseRecommendedBundle() {
	x.UpdatePreferences(func(p *application.Preferences) { p.ModelOverride = "" })
	go x.a.activate("")
}

// UpdatePreferences persists a change and applies its side effects.
func (x actions) UpdatePreferences(change func(*application.Preferences)) {
	a := x.a
	before := a.state.Preferences
	next := before
	change(&next)
	next, _ = next.Normalize()
	a.savePreferences(next)
	if next.UILanguage != before.UILanguage {
		a.updateLanguage()
	}
	if next.LaunchAtLogin != before.LaunchAtLogin {
		_ = mygo.App.SetOpenAtLogin(next.LaunchAtLogin)
	}
	if next.Microphone != before.Microphone {
		a.mic.Select(next.Microphone)
		if a.monitor != nil {
			a.stopMonitor()
			a.startMonitor()
		}
	}
	if next.MicrophoneGain != before.MicrophoneGain {
		a.mic.SetGain(next.MicrophoneGain)
	}
	if next.Mode != before.Mode {
		a.core.SetMode(next.Mode)
	}
	if next.Polish != before.Polish {
		if next.Polish {
			go a.startPolish()
		} else {
			go a.rewriter.Disable()
		}
	}
	if before.ContextHistory && !next.ContextHistory {
		a.history.Clear()
	}
	if next.CheckUpdates && !before.CheckUpdates && a.updates.due(time.Now()) {
		a.updates.check(time.Now(), false)
	}
}

// SetShortcut registers a new hotkey, keeping the old one when it fails.
func (x actions) SetShortcut(accelerator string) error {
	a := x.a
	previous := a.state.Shortcut.Accelerator
	if err := a.registerShortcut(accelerator); err != nil {
		_ = a.registerShortcut(previous)
		return err
	}
	x.UpdatePreferences(func(p *application.Preferences) { p.Shortcut = accelerator })
	return nil
}

// SetHoldKey persists the hold-to-talk key and restarts its listener.
func (x actions) SetHoldKey(key application.HoldKey) {
	x.UpdatePreferences(func(p *application.Preferences) { p.HoldKey = key })
	x.a.startHoldKey()
}

// RequestAccessibility guides the user to add Arion to the Accessibility
// list. A process outside an app bundle has nothing to drag and uses the
// system prompt instead.
func (x actions) RequestAccessibility() {
	if !x.a.openGuide(arion.GuideAccessibility) {
		x.a.platform.RequestAccessibility()
	}
}

func (x actions) OpenAccessibilitySettings() {
	if !x.a.openGuide(arion.GuideAccessibility) {
		x.a.platform.OpenAccessibilitySettings()
	}
}

// RequestScreenRecording asks for the consent screen context needs.
func (x actions) RequestScreenRecording() { x.a.platform.RequestScreenRecording() }

// RequestInputMonitoring asks for the consent a macOS hold key needs.
func (x actions) RequestInputMonitoring() {
	if !x.a.openGuide(arion.GuideInputMonitoring) {
		x.a.platform.RequestInputMonitoring()
	}
	x.a.refreshPermissions()
}

func (x actions) SuspendShortcut(suspend bool) {
	a := x.a
	if suspend && a.shortcutActive {
		mygo.GlobalShortcut.Unregister(a.state.Shortcut.Accelerator)
		a.shortcutActive = false
		return
	}
	if !suspend && !a.shortcutActive {
		_ = a.registerShortcut(a.state.Shortcut.Accelerator)
	}
}

// InstallInputMethod starts installation and publishes completion on the UI thread.
func (x actions) InstallInputMethod() error {
	if x.a.ime == nil {
		return errInputMethodUnsupported
	}
	a := x.a
	if a.state.InputMethodInstalling {
		return nil
	}
	a.state.InputMethodInstalling = true
	a.state.InputMethodError = ""
	a.record("ime_install_started", nil)
	go func() {
		err := a.ime.Install()
		state := a.ime.State()
		a.post(func() {
			a.state.InputMethodInstalling = false
			a.setInputMethodState(state)
			if err != nil {
				a.state.InputMethodError = copy.IMEFailed
				if domain.CodeOf(err) == domain.CodeArtifactUnverified {
					a.state.InputMethodError = copy.IMEPackageUnverified
				}
				a.record("ime_install_failed", err)
			} else if state == arion.InputMethodPending {
				a.record("ime_install_pending", nil)
			} else {
				a.record("ime_install_completed", nil)
			}
		})
	}()
	return nil
}

func (x actions) OpenInputMethodSettings() error {
	if x.a.ime == nil {
		return errInputMethodUnsupported
	}
	err := x.a.ime.OpenSettings()
	x.a.record("ime_settings_open", err)
	if err == nil {
		x.a.state.InputMethodError = ""
	}
	return err
}

// RemoveInputMethod removes the input source and returns to shortcut mode.
func (x actions) RemoveInputMethod() error {
	if x.a.ime == nil {
		return errInputMethodUnsupported
	}
	if x.a.state.InputMethodInstalling {
		return nil
	}
	x.a.state.InputMethodError = ""
	err := x.a.ime.Remove()
	x.a.record("ime_remove", err)
	x.a.state.InputMethod = x.a.ime.State()
	if err == nil && x.a.state.Preferences.Mode == arion.ModeSystemIME {
		x.UpdatePreferences(func(p *application.Preferences) { p.Mode = arion.ModeShortcut })
	}
	return err
}

func (x actions) AddReplacement(from, to string) error {
	a := x.a
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return errInvalidReplacement
	}
	a.loadVocabulary(true)
	if a.vocab == nil {
		return errVocabularyUnavailable
	}
	if _, err := a.vocab.Add(from, to); err != nil {
		return err
	}
	a.loadVocabulary(false)
	return nil
}

func (x actions) UndoLearned(id string) {
	x.a.dismissLearned()
	x.RemoveReplacement(id)
}

func (x actions) RemoveReplacement(id string) {
	a := x.a
	if a.vocab == nil {
		return
	}
	if err := a.vocab.Remove(id); err == nil {
		a.loadVocabulary(false)
	}
}

func (x actions) CompleteOnboarding() {
	a := x.a
	x.UpdatePreferences(func(p *application.Preferences) { p.OnboardingCompleted = true })
	if a.onboarding != nil {
		a.onboarding.Close()
	}
}

var _ arion.Actions = actions{}
