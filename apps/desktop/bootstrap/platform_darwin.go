//go:build darwin

package bootstrap

import (
	"context"
	"path/filepath"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/infrastructure/macos"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

type darwinPlatform struct{ permissions macos.Permissions }

func newPlatform() Platform { return darwinPlatform{} }

func configureOverlay(window *mygo.Window) {
	macos.ConfigureOverlay(window.NativeHandle())
}

func consent(status macos.AuthStatus) runtime.Consent {
	switch status {
	case macos.AuthGranted:
		return runtime.ConsentGranted
	case macos.AuthDenied:
		return runtime.ConsentDenied
	case macos.AuthRestricted:
		return runtime.ConsentRestricted
	}
	return runtime.ConsentUndetermined
}

func (p darwinPlatform) Microphone() runtime.Consent { return consent(p.permissions.Microphone()) }
func (p darwinPlatform) Accessibility() runtime.Consent {
	return consent(p.permissions.Accessibility())
}
func (p darwinPlatform) MicrophoneAuthorized() bool { return p.permissions.MicrophoneAuthorized() }
func (p darwinPlatform) RequestMicrophone()         { p.permissions.RequestMicrophone() }
func (p darwinPlatform) RequestAccessibility()      { p.permissions.RequestAccessibility() }
func (p darwinPlatform) ScreenRecording() runtime.Consent {
	return consent(p.permissions.ScreenRecording())
}
func (p darwinPlatform) RequestScreenRecording()    { p.permissions.RequestScreenRecording() }
func (p darwinPlatform) OpenMicrophoneSettings()    { p.permissions.Open(macos.PaneMicrophone) }
func (p darwinPlatform) OpenAccessibilitySettings() { p.permissions.Open(macos.PaneAccessibility) }

func (p darwinPlatform) InputMonitoring() runtime.Consent {
	return consent(p.permissions.InputMonitoring())
}
func (p darwinPlatform) RequestInputMonitoring() { p.permissions.RequestInputMonitoring() }

func (darwinPlatform) AppBundle() string { return macos.AppBundle() }

func (p darwinPlatform) OpenGuidePane(pane runtime.GuidePane) {
	switch pane {
	case runtime.GuideAccessibility:
		p.permissions.Open(macos.PaneAccessibility)
	case runtime.GuideInputMonitoring:
		p.permissions.Open(macos.PaneInputMonitoring)
	case runtime.GuideMicrophone:
		p.permissions.Open(macos.PaneMicrophone)
	}
}

func (darwinPlatform) SettingsWindow() SettingsWindow {
	w := macos.FindSettingsWindow()
	return SettingsWindow{Running: w.Running, Visible: w.Visible, Frontmost: w.Frontmost,
		Bounds: mygo.Rectangle{X: w.X, Y: w.Y, Width: w.Width, Height: w.Height}}
}
func (darwinPlatform) StartHoldKey(key application.HoldKey, sink func(holdkey.Event)) error {
	return macos.StartHold(key, sink)
}
func (darwinPlatform) StopHoldKey() { macos.StopHold() }

func (p darwinPlatform) NewTargets(_ func() application.Delivery, gates TargetGates) Targets {
	targets := macos.NewTargets()
	targets.ReadSite, targets.ReadCursor, targets.ReadScreen = gates.Site, gates.Cursor, gates.Screen
	return targets
}

type darwinIME struct{ *macos.IME }

func (darwinPlatform) NewInputMethod(onLeave func()) InputMethod {
	return darwinIME{macos.NewIME(macos.DefaultIMESocket(), onLeave)}
}

func (darwinIME) ApplyKeyboard(k application.KeyboardPreferences) error {
	return macos.WriteKeyboardSettings(k)
}

func (darwinIME) State() runtime.InputMethodState {
	switch macos.QueryIME() {
	case macos.IMESelected:
		return runtime.InputMethodSelected
	case macos.IMECopied:
		return runtime.InputMethodDiscovering
	case macos.IMERegistered:
		return runtime.InputMethodPending
	case macos.IMEEnabled:
		return runtime.InputMethodInstalled
	}
	return runtime.InputMethodAbsent
}

// Install copies the input method shipped in the app's resources.
func (m darwinIME) Install() error {
	if err := m.EnsureListening(); err != nil {
		return err
	}
	m.BeginUpdate()
	defer m.EndUpdate()
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return err
	}
	return macos.InstallIME(filepath.Join(resources, "ArionInput.app"), m.WaitReady)
}

func (darwinIME) Remove() error { return macos.RemoveIME() }

func (darwinIME) UpdateAvailable() bool {
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return false
	}
	return macos.IMEUpdateAvailable(filepath.Join(resources, "ArionInput.app"))
}

func (darwinIME) OpenSettings() error { return macos.OpenIMEKeyboardSettings() }

func (m darwinIME) Activate(ctx context.Context) error {
	if err := m.EnsureListening(); err != nil {
		return &macos.IMESetupError{Stage: "connection", Err: err}
	}
	m.BeginUpdate()
	defer m.EndUpdate()
	return macos.ActivateIME(ctx, m.WaitReady)
}
