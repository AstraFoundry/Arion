package bootstrap

import (
	"arion.local/core/application"
	"arion.local/core/domain/ports"
	"arion.local/desktop/ui/shared/runtime"
)

// InputMethod is the system input-method mode adapter: it owns marked text
// in the focused client and commits through the OS input-method protocol.
type InputMethod interface {
	ports.TargetProvider
	ports.InputTarget
	ports.CompositionTarget
	Listen() error
	State() runtime.InputMethodState
	Install() error
	Remove() error
}

// Targets is a platform's editor capture and insertion adapter.
type Targets interface {
	ports.TargetProvider
	ports.InputTarget
}

// Platform is the OS-specific consent and insertion surface.
type Platform interface {
	Microphone() runtime.Consent
	Accessibility() runtime.Consent
	MicrophoneAuthorized() bool
	RequestMicrophone()
	RequestAccessibility()
	OpenMicrophoneSettings()
	OpenAccessibilitySettings()
	NewTargets(delivery func() application.Delivery) Targets
	// NewInputMethod returns nil where no input method host exists yet.
	NewInputMethod(onLeave func()) InputMethod
}
