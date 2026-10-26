package bootstrap

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/updatestate"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

// actions implements runtime.Actions. Views call it on the main thread.
type actions struct{ a *App }

func (x actions) ToggleDictation()   { x.a.record("dictation_toggle", nil); x.a.toggleDictation() }
func (x actions) CancelDictation()   { x.a.record("dictation_cancel", nil); x.a.core.Cancel() }
func (x actions) RequestMicrophone() { x.a.platform.RequestMicrophone() }
func (x actions) OpenMicrophoneSettings() {
	if !x.a.openGuide(arion.GuideMicrophone) {
		x.a.platform.OpenMicrophoneSettings()
	}
}
func (x actions) CloseGuide()    { x.a.closeGuide(true) }
func (x actions) CloseTrayHint() { x.a.closeTrayHint() }
func (x actions) RetryModel() {
	if !x.a.updateApplying() {
		x.a.runResource(func() { x.a.activate(x.a.state.Preferences.ModelOverride) })
	}
}
func (x actions) DownloadPolish()      { x.a.startPolish(true) }
func (x actions) RetryPolish()         { x.DownloadPolish() }
func (x actions) StartLevelMonitor()   { x.a.startMonitor() }
func (x actions) StopLevelMonitor()    { x.a.stopMonitor() }
func (x actions) RefreshDevices()      { go x.a.refreshDevicesAsync() }
func (x actions) CopyText(text string) { mygo.Clipboard.WriteText(text) }
func (x actions) OpenSettings()        { x.a.openSettings() }
func (x actions) CheckForUpdates()     { x.a.updates.check(time.Now(), true) }
func (x actions) InstallUpdate()       { x.a.updates.install() }
func (x actions) RestartToUpdate()     { x.a.restartUpdate() }

// OpenUpdate shows Settings › About, where the update is installed.
func (x actions) OpenUpdate() {
	x.a.closeUpdateReminder()
	x.a.clearUpdateNotification()
	if x.a.updatePolicy != nil && x.a.state.Update.Version != "" {
		_ = x.a.saveReminder(x.a.reminderKey(), func(r *updatestate.Reminder) { r.Dismissed = false })
	}
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
func (x actions) TestHoldKey(testing bool) { x.a.testingHold = testing }

func (x actions) UseBundle(id string) {
	if x.a.updateApplying() {
		return
	}
	x.UpdatePreferences(func(p *application.Preferences) { p.ModelOverride = id })
	x.a.runResource(func() { x.a.activate(id) })
}

func (x actions) UseRecommendedBundle() {
	if x.a.updateApplying() {
		return
	}
	x.UpdatePreferences(func(p *application.Preferences) { p.ModelOverride = "" })
	x.a.runResource(func() { x.a.activate("") })
}

// UpdatePreferences persists a change and applies its side effects.
func (x actions) UpdatePreferences(change func(*application.Preferences)) {
	a := x.a
	if a.updatePolicy != nil && a.updatePolicy.applying {
		return
	}
	before := a.state.Preferences
	next := before
	change(&next)
	next, _ = next.Normalize(runtime.GOOS)
	if !a.savePreferences(next) {
		if next.CheckUpdates != before.CheckUpdates || next.AutoInstallUpdates != before.AutoInstallUpdates || next.AllowPrereleaseUpdates != before.AllowPrereleaseUpdates {
			a.state.Update.StateError = true
			if a.updatePolicy != nil {
				a.updatePolicy.failed = true
			}
		}
		return
	}
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
	if (next.Keyboard != before.Keyboard || next.HoldKey != before.HoldKey) && a.ime != nil {
		if err := a.ime.ApplyKeyboard(next.Keyboard, next.HoldKey); err != nil {
			a.record("keyboard_settings", err)
		}
	}
	if next.Polish != before.Polish || next.PolishEngine != before.PolishEngine {
		a.startPolish(false)
	}
	if next.CloudModel != before.CloudModel {
		a.cloud.SetModel(next.CloudModel)
	}
	if before.ContextHistory && !next.ContextHistory {
		a.history.Clear()
	}
	if next.AllowPrereleaseUpdates != before.AllowPrereleaseUpdates {
		a.closeUpdateReminder()
		a.clearUpdateNotification()
		a.updates.setPrereleases(next.AllowPrereleaseUpdates)
	}
	if a.updates != nil {
		a.updates.preferences(next.CheckUpdates, next.AutoInstallUpdates)
	}
	if next.CheckUpdates && !before.CheckUpdates && a.updates.due(time.Now()) {
		a.updates.check(time.Now(), false)
	}
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
func (x actions) RequestScreenRecording() {
	x.a.platform.RequestScreenRecording()
	if x.a.platform.ScreenRecording() != arion.ConsentGranted && x.a.platform.AppBundle() != "" {
		x.a.showGuide(arion.GuideScreenRecording, transfer.Data{})
	}
}

// RequestInputMonitoring asks for the consent a macOS hold key needs.
func (x actions) RequestInputMonitoring() {
	if !x.a.openGuide(arion.GuideInputMonitoring) {
		x.a.platform.RequestInputMonitoring()
	}
	x.a.refreshPermissions()
}

// InstallInputMethod starts installation and publishes completion on the UI thread.
func (x actions) InstallInputMethod() error { return x.installInputMethod(false) }

func (x actions) installInputMethod(automatic bool) error {
	if x.a.ime == nil {
		return errInputMethodUnsupported
	}
	a := x.a
	if a.updateApplying() || a.state.InputMethodInstalling || a.state.InputMethodActivating || a.state.InputMethodRemoving {
		return nil
	}
	if a.core != nil && !automatic {
		a.core.CancelMode(domain.SystemIMEMode)
	}
	generation := a.imeSetupGeneration
	updating := a.state.InputMethodUpdate && a.state.InputMethod != arion.InputMethodAbsent
	a.state.InputMethodInstalling = true
	a.state.InputMethodError = ""
	a.record("ime_install_started", nil)
	go func() {
		var err error
		if automatic {
			err = a.ime.(automaticInputMethod).UpdateAutomatically()
		} else {
			err = a.ime.Install()
		}
		state := a.ime.State()
		update := a.ime.UpdateAvailable()
		a.post(func() {
			a.state.InputMethodInstalling = false
			// Keep Retry available if the machine transaction committed but
			// ordinary-user setup failed after the build already matched.
			a.state.InputMethodUpdate = update || updating && err != nil
			a.setInputMethodState(state)
			var deferred interface{ InputMethodUpdateDeferred() bool }
			if automatic && errors.As(err, &deferred) && deferred.InputMethodUpdateDeferred() {
				return
			}
			a.state.InputMethodUpdateFailed = updating && err != nil
			if err != nil {
				a.state.InputMethodError = copy.IMEFailed
				if domain.CodeOf(err) == domain.CodeArtifactUnverified {
					a.state.InputMethodError = copy.IMEPackageUnverified
				}
				a.record("ime_install_failed", err)
			} else if state == arion.InputMethodPending || state == arion.InputMethodDiscovering {
				a.record("ime_install_pending", nil)
			} else {
				a.record("ime_install_completed", nil)
			}
			if err == nil && !updating && generation == a.imeSetupGeneration {
				_ = x.ActivateInputMethod()
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
		if !inputMethodGuideReady(x.a.ime.State()) {
			x.a.showGuide(arion.GuideKeyboard, transfer.Data{})
		}
	}
	return err
}

// RemoveInputMethod removes the input source; subsequent dictation selects its
// route from the actual current source.
func (x actions) RemoveInputMethod() error {
	a := x.a
	if a.ime == nil {
		return errInputMethodUnsupported
	}
	if a.updateApplying() || a.state.InputMethodInstalling || a.state.InputMethodActivating || a.state.InputMethodRemoving {
		return nil
	}
	if a.core != nil {
		a.core.CancelMode(domain.SystemIMEMode)
	}
	a.imeSetupGeneration++
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	a.imeSetupCancel = cancel
	a.state.InputMethodRemoving = true
	a.state.InputMethodError = ""
	go func() {
		err := a.ime.Remove(ctx)
		cancel()
		state := a.ime.State()
		a.post(func() {
			a.imeSetupCancel = nil
			a.state.InputMethodRemoving = false
			a.setInputMethodState(state)
			a.record("ime_remove", err)
			if err == nil {
				a.state.InputMethodUpdate = false
				return
			}
			if errors.Is(err, context.Canceled) {
				return
			}
			a.state.InputMethodError = copy.IMERemoveFailed
			var pending interface{ InputMethodRemovalPending() bool }
			if errors.As(err, &pending) && pending.InputMethodRemovalPending() {
				a.state.InputMethodError = copy.IMERemovePending
			}
		})
	}()
	return nil
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
	a.openSettings()
	a.showTrayHint()
}

var _ arion.Actions = actions{}
