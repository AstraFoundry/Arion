//go:build windows

package bootstrap

import (
	"arion.local/core/application"
	"arion.local/desktop/infrastructure/diagnostics"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/infrastructure/windows"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// windowsPlatform: Windows grants microphone access per app in Settings and
// reports a denial when capture opens; typing into other apps needs no
// permission, so Accessibility is granted.
type windowsPlatform struct{}

func newPlatform() Platform { return windowsPlatform{} }

func configureInputMethodDiagnostics(func(diagnostics.Entry)) {}

func configureClipboardWindow(*mygo.Window) {}

func configureOverlay(window *mygo.Window) { windows.ConfigureOverlay(window.NativeHandle()) }

func (windowsPlatform) Microphone() runtime.Consent    { return runtime.ConsentGranted }
func (windowsPlatform) Accessibility() runtime.Consent { return runtime.ConsentGranted }
func (windowsPlatform) MicrophoneAuthorized() bool     { return true }
func (windowsPlatform) RequestMicrophone()             { windows.OpenSettings("ms-settings:privacy-microphone") }
func (windowsPlatform) RequestAccessibility()          {}

// Windows needs no consent to capture another window.
func (windowsPlatform) ScreenRecording() runtime.Consent { return runtime.ConsentGranted }
func (windowsPlatform) RequestScreenRecording()          {}
func (windowsPlatform) OpenMicrophoneSettings() {
	windows.OpenSettings("ms-settings:privacy-microphone")
}
func (windowsPlatform) OpenAccessibilitySettings() {}

// A low-level keyboard hook needs no consent.
func (windowsPlatform) InputMonitoring() runtime.Consent { return runtime.ConsentGranted }
func (windowsPlatform) RequestInputMonitoring()          {}

// Windows has no consent that a settings list takes by drag.
func (windowsPlatform) AppBundle() string               { return "" }
func (windowsPlatform) OpenGuidePane(runtime.GuidePane) {}
func (windowsPlatform) SettingsWindow() SettingsWindow  { return SettingsWindow{} }
func (windowsPlatform) StartHoldKeys(keys []application.HoldKey, sink func(application.HoldKey, holdkey.Event)) error {
	return windows.StartHoldKeys(keys, sink)
}
func (windowsPlatform) StopHoldKey() { windows.StopHold() }

// NewTargets: the Windows adapter reads no web site or cursor text; it
// recognizes the window's text when gates.Screen allows.
func (windowsPlatform) NewTargets(delivery func() application.Delivery, gates TargetGates) Targets {
	return windows.NewTargets(gates.Screen, func() windows.Delivery {
		if delivery() == application.DeliveryClipboard {
			return windows.DeliveryClipboard
		}
		return windows.DeliveryAutomatic
	})
}

// Windows ships a separate x64 in-process TSF service for owned voice ranges.
func (windowsPlatform) NewInputMethod(onLeave, onDictate func()) InputMethod {
	return newWindowsIME(onLeave, onDictate)
}
