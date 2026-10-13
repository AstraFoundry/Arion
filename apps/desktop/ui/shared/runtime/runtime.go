// Package runtime is the boundary between the native UI and the composition
// root: an observable application state snapshot and the actions views may
// request. Core values enter the UI only through this package.
package runtime

import (
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
)

// Re-exported core types so UI slices depend on this boundary only.
type (
	SessionMode   = domain.InputMode
	SessionState  = domain.SessionState
	ErrorCode     = domain.ErrorCode
	CommitOutcome = contracts.CommitOutcome
	Preferences   = application.Preferences
	Delivery      = application.Delivery
	UILanguage    = application.UILanguage
	HoldKey       = application.HoldKey
)

const (
	StateIdle       = domain.Idle
	StatePreparing  = domain.Preparing
	StateCapturing  = domain.Capturing
	StateFinalizing = domain.Finalizing
	StateReviewing  = domain.Reviewing
	StateCommitting = domain.Committing
	StateCompleted  = domain.Completed
	StateCancelled  = domain.Cancelled
	StateFailed     = domain.Failed

	OutcomeApplied    = contracts.Applied
	OutcomeNotApplied = contracts.NotApplied
	OutcomeUnknown    = contracts.Unknown

	DeliveryAutomatic = application.DeliveryAutomatic
	DeliveryClipboard = application.DeliveryClipboard
	LanguageEnglish   = application.English
	LanguageChinese   = application.SimplifiedChinese

	HoldOff          = application.HoldOff
	HoldFn           = application.HoldFn
	HoldRightOption  = application.HoldRightOption
	HoldRightCommand = application.HoldRightCommand
	HoldRightAlt     = application.HoldRightAlt
	HoldRightCtrl    = application.HoldRightCtrl
)

// HoldKeysFor lists the hold keys a platform supports.
func HoldKeysFor(platform string) []HoldKey { return application.HoldKeysFor(platform) }

// Error codes the UI explains.
const (
	CodePermissionDenied   = domain.CodePermissionDenied
	CodeModelMissing       = domain.CodeModelMissing
	CodeModelUnsupported   = domain.CodeModelUnsupported
	CodeAudioDiscontinuity = domain.CodeAudioDiscontinuity
	CodeTargetChanged      = domain.CodeTargetChanged
	CodeTargetUnsupported  = domain.CodeTargetUnsupported
	CodeDeviceUnavailable  = domain.CodeDeviceUnavailable
	CodeSessionBusy        = domain.CodeSessionBusy
	CodeNoSpeech           = domain.CodeNoSpeech
	CodeSecureFieldBlocked = domain.CodeSecureFieldBlocked
	CodeArtifactUnverified = domain.CodeArtifactUnverified
	CodeProvisioningFailed = domain.CodeProvisioningFailed
	CodeEvaluationMissing  = domain.CodeEvaluationUnavailable
	CodeCancelled          = domain.CodeCancelled
)

// Diagnostic categories adapters attach to PERMISSION_DENIED.
const (
	DetailMicrophone    = "microphone"
	DetailAccessibility = "accessibility"
	DetailInputMethod   = "input method not active"
)

// Dictation is the live session projection. Text stays in RAM only.
type Dictation struct {
	SessionID string
	State     SessionState
	Draft     string
	Final     string
	// Levels are recent input levels (0..1), newest last.
	Levels  []float32
	Failure ErrorCode
	// FailureDetail names the capability behind a failure, e.g. "microphone".
	FailureDetail string
	// Pasted marks a commit delivered through the clipboard.
	Pasted  bool
	Outcome CommitOutcome
	// Changed is when State last changed, for timed dismissals.
	Changed time.Time
}

// ModelState is the readiness of the recognition bundle.
type ModelState string

const (
	ModelUnselected   ModelState = "unselected"
	ModelProvisioning ModelState = "provisioning"
	ModelLoading      ModelState = "loading"
	ModelReady        ModelState = "ready"
	ModelFailed       ModelState = "failed"
)

// ProvisionPhase is the step of a download.
type ProvisionPhase string

const (
	PhaseDownloading ProvisionPhase = "downloading"
	PhaseVerifying   ProvisionPhase = "verifying"
	PhaseExtracting  ProvisionPhase = "extracting"
	PhaseReady       ProvisionPhase = "ready"
)

// Bundle describes an evaluated recognition bundle.
type Bundle struct {
	ID          string
	Name        string
	Streaming   string
	Finalizer   string
	SizeBytes   int64
	ErrorRates  map[string]float64
	ReviewP95Ms float64
	RTF         float64
	Evaluated   bool
	// Recommended marks the automatic choice.
	Recommended bool
}

// Model is the active bundle and its provisioning progress.
type Model struct {
	State           ModelState
	Active          Bundle
	Phase           ProvisionPhase
	Done            int64
	Total           int64
	Failure         ErrorCode
	LatencyMeasured bool
	Automatic       bool
	Bundles         []Bundle
}

// Consent is an OS permission state.
type Consent string

const (
	ConsentUndetermined Consent = "undetermined"
	ConsentDenied       Consent = "denied"
	ConsentRestricted   Consent = "restricted"
	ConsentGranted      Consent = "granted"
)

type Permissions struct {
	Microphone    Consent
	Accessibility Consent
	// InputMonitoring is needed only for a hold key on macOS.
	InputMonitoring Consent
}

// Device is a capture endpoint.
type Device struct {
	ID      string
	Name    string
	Default bool
}

// Replacement is one approved vocabulary correction.
type Replacement struct {
	ID   string
	From string
	To   string
}

// InputMethodState is the system input source's registration state.
type InputMethodState string

const (
	InputMethodUnsupported InputMethodState = "unsupported"
	InputMethodAbsent      InputMethodState = "absent"
	InputMethodInstalled   InputMethodState = "installed"
	InputMethodSelected    InputMethodState = "selected"
)

// Input modes.
const (
	ModeShortcut  = domain.ShortcutMode
	ModeSystemIME = domain.SystemIMEMode
)

// Shortcut is the registered dictation hotkey.
type Shortcut struct {
	Accelerator string
	Registered  bool
	// Pressed is when the shortcut last fired, for the onboarding test.
	Pressed time.Time
	// HoldActive reports the hold-key listener is running.
	HoldActive bool
	// HoldFailure explains why a chosen hold key is not listening:
	// PERMISSION_DENIED means Input Monitoring consent is missing.
	HoldFailure ErrorCode
	// Holding marks a session that dictates until the hold key is released.
	Holding bool
}

// State is everything views read. The composition root mutates it only on
// the main thread and then invalidates windows.
type State struct {
	Version     string
	Dictation   Dictation
	Model       Model
	Permissions Permissions
	Preferences Preferences
	Devices     []Device
	Vocabulary  []Replacement
	Shortcut    Shortcut
	InputMethod InputMethodState
	// MonitorLevels feed the microphone test meter, newest last.
	MonitorLevels []float32
	LastText      string
	Platform      string
}

// Actions are requests views make; implementations return quickly and do
// slow work off the main thread.
type Actions interface {
	ToggleDictation()
	CancelDictation()
	RequestMicrophone()
	RequestAccessibility()
	OpenMicrophoneSettings()
	OpenAccessibilitySettings()
	RetryModel()
	UseBundle(id string)
	UseRecommendedBundle()
	UpdatePreferences(func(*Preferences))
	SetShortcut(accelerator string) error
	// SetHoldKey chooses the hold-to-talk key, or HoldOff.
	SetHoldKey(key HoldKey)
	RequestInputMonitoring()
	// SuspendShortcut releases the global hotkey while a recorder listens.
	SuspendShortcut(suspend bool)
	// TestShortcut makes hotkey presses only report Shortcut.Pressed, for
	// the onboarding check, instead of starting dictation.
	TestShortcut(testing bool)
	RefreshDevices()
	StartLevelMonitor()
	StopLevelMonitor()
	InstallInputMethod() error
	RemoveInputMethod() error
	AddReplacement(from, to string) error
	RemoveReplacement(id string)
	CopyText(text string)
	OpenSettings()
	CompleteOnboarding()
	Quit()
}

// Store pairs the state with the actions for views.
type Store struct {
	State   *State
	Actions Actions
}

// Copy derives an immutable catalog from the current shared language preference.
func (s Store) Copy() copy.Catalog {
	return copy.For(copy.Locale(s.State.Preferences.UILanguage))
}
