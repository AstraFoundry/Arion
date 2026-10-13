//go:build windows

package bootstrap

import (
	"arion.local/core/application"
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

func configureOverlay(*mygo.Window) {}

func (windowsPlatform) Microphone() runtime.Consent    { return runtime.ConsentGranted }
func (windowsPlatform) Accessibility() runtime.Consent { return runtime.ConsentGranted }
func (windowsPlatform) MicrophoneAuthorized() bool     { return true }
func (windowsPlatform) RequestMicrophone()             { windows.OpenSettings("ms-settings:privacy-microphone") }
func (windowsPlatform) RequestAccessibility()          {}
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
func (windowsPlatform) StartHoldKey(key application.HoldKey, sink func(holdkey.Event)) error {
	return windows.StartHold(key, sink)
}
func (windowsPlatform) StopHoldKey() { windows.StopHold() }

func (windowsPlatform) NewTargets(delivery func() application.Delivery) Targets {
	return windows.NewTargets(func() windows.Delivery {
		if delivery() == application.DeliveryClipboard {
			return windows.DeliveryClipboard
		}
		return windows.DeliveryAutomatic
	})
}

// NewInputMethod: the TSF text service is not implemented (TSF-01), so the
// input-method mode is reported unsupported rather than simulated.
func (windowsPlatform) NewInputMethod(func()) InputMethod { return nil }
