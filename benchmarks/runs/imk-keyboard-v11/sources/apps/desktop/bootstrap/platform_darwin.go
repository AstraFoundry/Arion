//go:build darwin

package bootstrap

import (
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

func (p darwinPlatform) NewTargets(delivery func() application.Delivery, gates TargetGates) Targets {
	targets := macos.NewTargets(func() macos.Delivery {
		if delivery() == application.DeliveryClipboard {
			return macos.DeliveryClipboard
		}
		return macos.DeliveryAutomatic
	})
	targets.ReadSite, targets.ReadCursor, targets.ReadScreen = gates.Site, gates.Cursor, gates.Screen
	return targets
}

type darwinIME struct{ *macos.IME }

func (darwinPlatform) NewInputMethod(onLeave func()) InputMethod {
	return darwinIME{macos.NewIME(macos.DefaultIMESocket(), onLeave)}
}

func (darwinIME) State() runtime.InputMethodState {
	switch macos.QueryIME() {
	case macos.IMESelected:
		return runtime.InputMethodSelected
	case macos.IMECopied, macos.IMERegistered:
		return runtime.InputMethodPending
	case macos.IMEEnabled:
		return runtime.InputMethodInstalled
	}
	return runtime.InputMethodAbsent
}

// Install copies the input method shipped in the app's resources.
func (darwinIME) Install() error {
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return err
	}
	return macos.InstallIME(filepath.Join(resources, "ArionInput.app"))
}

func (darwinIME) Remove() error { return macos.RemoveIME() }

func (darwinIME) OpenSettings() error { return macos.OpenIMEKeyboardSettings() }
