package bootstrap

import (
	"arion.local/core/application"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

// SettingsWindow is where the system settings window of a guided consent
// shows, in DIPs from the primary display's top-left corner.
type SettingsWindow struct {
	Running, Visible, Frontmost bool
	Bounds                      mygo.Rectangle
}

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
	// InputMonitoring is the consent to observe keys for hold-to-talk.
	InputMonitoring() runtime.Consent
	RequestInputMonitoring()
	// AppBundle is the application bundle the consent guide offers to drag
	// into a settings list; "" where the process runs outside one.
	AppBundle() string
	// OpenGuidePane opens the settings list of a guided consent.
	OpenGuidePane(pane runtime.GuidePane)
	// SettingsWindow reports the window the guide follows. Consent reads
	// and this run on the guide's worker goroutine.
	SettingsWindow() SettingsWindow
	// StartHoldKey observes key; sink runs on a native thread and must not
	// block. StopHoldKey is idempotent.
	StartHoldKey(key application.HoldKey, sink func(holdkey.Event)) error
	StopHoldKey()
	NewTargets(delivery func() application.Delivery) Targets
	// NewInputMethod returns nil where no input method host exists yet.
	NewInputMethod(onLeave func()) InputMethod
}
