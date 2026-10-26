package bootstrap

import (
	"context"

	"arion.local/core/application"
	"arion.local/core/contracts"
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
	Close()
	State() runtime.InputMethodState
	Install() error
	// Activate retries installed-source setup without replacing its bundle.
	Activate(context.Context) error
	OpenSettings() error
	Remove(context.Context) error
	// ApplyKeyboard hands the keyboard preferences and the dictation key
	// that titles the menu's Voice Input item to the input method, which
	// reads them when a client next activates it.
	ApplyKeyboard(application.KeyboardPreferences, application.HoldKey) error
	// UpdateAvailable reports an installed input method older or otherwise
	// different from the one this app bundles. It reads files; call it off
	// the UI thread.
	UpdateAvailable() bool
}

// Targets is a platform's editor capture and insertion adapter.
type Targets interface {
	ports.TargetProvider
	ports.InputTarget
	// FieldReader follows committed fields for edit learning; adapters that
	// cannot read fields refuse every Follow.
	ports.FieldReader
	// FocusOwner returns the frontmost app's process before an Arion window
	// takes the focus; Peek then describes its focused field for display.
	FocusOwner() int
	Peek(ctx context.Context, owner int) (contracts.TargetSnapshot, error)
}

// TargetGates say what the target adapter may read beyond the focused
// field's identity; each is checked when a read happens.
type TargetGates struct {
	// Site is the host of the focused web page, for scene selection.
	Site func() bool
	// Cursor is the field's text around the cursor; Screen is the focused
	// window's recognized text.
	Cursor func() bool
	Screen func() bool
}

// Platform is the OS-specific consent and insertion surface.
type Platform interface {
	Microphone() runtime.Consent
	Accessibility() runtime.Consent
	MicrophoneAuthorized() bool
	RequestMicrophone()
	RequestAccessibility()
	// ScreenRecording is the consent window text recognition needs.
	ScreenRecording() runtime.Consent
	RequestScreenRecording()
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
	// StartHoldKeys observes the dictation key and an optional translation
	// key. sink runs on a native thread and must not block. StopHoldKey
	// is idempotent.
	StartHoldKeys(keys []application.HoldKey, sink func(application.HoldKey, holdkey.Event)) error
	StopHoldKey()
	NewTargets(delivery func() application.Delivery, gates TargetGates) Targets
	// NewInputMethod returns nil where no input method host exists yet.
	// onDictate runs off the UI thread when the input menu's Voice Input
	// item is chosen.
	NewInputMethod(onLeave, onDictate func()) InputMethod
}
