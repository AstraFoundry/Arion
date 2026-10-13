//go:build windows

package bootstrap

import (
	"arion.local/core/application"
	"arion.local/desktop/infrastructure/windows"
	"arion.local/desktop/ui/shared/runtime"
)

// windowsPlatform: Windows grants microphone access per app in Settings and
// reports a denial when capture opens; typing into other apps needs no
// permission, so Accessibility is granted.
type windowsPlatform struct{}

func newPlatform() Platform { return windowsPlatform{} }

func (windowsPlatform) Microphone() runtime.Consent    { return runtime.ConsentGranted }
func (windowsPlatform) Accessibility() runtime.Consent { return runtime.ConsentGranted }
func (windowsPlatform) MicrophoneAuthorized() bool     { return true }
func (windowsPlatform) RequestMicrophone()             { windows.OpenSettings("ms-settings:privacy-microphone") }
func (windowsPlatform) RequestAccessibility()          {}
func (windowsPlatform) OpenMicrophoneSettings() {
	windows.OpenSettings("ms-settings:privacy-microphone")
}
func (windowsPlatform) OpenAccessibilitySettings() {}

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
