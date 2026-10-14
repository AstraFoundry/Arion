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
	"github.com/egoist/mygo/transfer"
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

// GainStops are the selectable microphone gains, GainStep dB apart, and
// the speaking distances they suit.
var GainStops = application.GainStops

const GainStep = application.GainStep

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
	// Original is the recognized text when a rewrite replaced it.
	Original string
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

// PolishState is the readiness of the local rewrite model.
type PolishState string

const (
	PolishUnsupported  PolishState = "unsupported"
	PolishOff          PolishState = "off"
	PolishProvisioning PolishState = "provisioning"
	PolishLoading      PolishState = "loading"
	PolishReady        PolishState = "ready"
	PolishFailed       PolishState = "failed"
)

// Polish is the live rewrite capability and its download progress.
type Polish struct {
	State     PolishState
	Phase     ProvisionPhase
	Done      int64
	Total     int64
	Failure   ErrorCode
	Installed bool
	SizeBytes int64
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
	// ScreenRecording is needed for screen context on macOS.
	ScreenRecording Consent
}

// GuidePane is a System Settings list that the user adds Arion to by hand.
type GuidePane string

const (
	GuideAccessibility   GuidePane = "accessibility"
	GuideInputMonitoring GuidePane = "input-monitoring"
)

// Guide is the consent guide that floats under System Settings while the
// user adds Arion to a privacy list.
type Guide struct {
	// Pane is the list being guided; empty when no guide is open.
	Pane GuidePane
	// App is Arion's .app as file data, which the list takes as a drop.
	App transfer.Data
	// Shown counts the times the guide appeared, so it enters each time.
	Shown int
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

// UpdatePhase is where the app is in updating itself.
type UpdatePhase string

const (
	// UpdateUnsupported: this build cannot replace itself (a development
	// build, or an install location the user cannot write).
	UpdateUnsupported UpdatePhase = "unsupported"
	UpdateIdle        UpdatePhase = "idle"
	UpdateChecking    UpdatePhase = "checking"
	UpdateCurrent     UpdatePhase = "current"
	UpdateAvailable   UpdatePhase = "available"
	UpdateDownloading UpdatePhase = "downloading"
	// UpdateReady: the new version replaced the app on disk and runs after
	// a restart.
	UpdateReady  UpdatePhase = "ready"
	UpdateFailed UpdatePhase = "failed"
)

// Update is the newest release known and how far installing it got.
type Update struct {
	Phase UpdatePhase
	// Version and Notes (Markdown) describe the newer release.
	Version string
	Notes   string
	// Downloaded and Total count bytes while Phase is UpdateDownloading.
	Downloaded, Total int64
	// Checked is when the release feed last answered.
	Checked time.Time
	// Failed is the step that failed while Phase is UpdateFailed.
	Failed UpdateStep
}

// UpdateStep is a step of updating that can fail.
type UpdateStep string

const (
	UpdateStepCheck   UpdateStep = "check"
	UpdateStepInstall UpdateStep = "install"
)

// Pending reports a newer version the user has not restarted into yet.
func (u Update) Pending() bool {
	return u.Phase == UpdateAvailable || u.Phase == UpdateDownloading || u.Phase == UpdateReady
}

// SceneTarget is the field the menu bar panel shows the scene of: the app
// that was frontmost when the panel opened. It lives only in memory.
type SceneTarget struct {
	Known bool
	// AppName is the app's display name.
	AppName string
	Hints   domain.SceneHints
}

// SettingsFocus asks the settings window to show a section. Seq increases
// with each request, so a page applies each one once.
type SettingsFocus struct {
	Section string
	Seq     int
}

// State is everything views read. The composition root mutates it only on
// the main thread and then invalidates windows.
type State struct {
	Version     string
	Dictation   Dictation
	Model       Model
	Polish      Polish
	Permissions Permissions
	Preferences Preferences
	Devices     []Device
	Vocabulary  []Replacement
	Shortcut    Shortcut
	InputMethod InputMethodState
	Guide       Guide
	Update      Update
	// Focus is the settings section last asked for from outside settings.
	Focus SettingsFocus
	// MonitorLevels feed the microphone test meter, newest last.
	MonitorLevels []float32
	LastText      string
	// LastOriginal is the recognized text of the last dictation when a
	// rewrite replaced it; empty otherwise.
	LastOriginal string
	Platform     string
	// SceneTarget is set each time the panel opens.
	SceneTarget SceneTarget
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
	// RetryPolish downloads or restarts the rewrite model after a failure.
	RetryPolish()
	UseBundle(id string)
	UseRecommendedBundle()
	UpdatePreferences(func(*Preferences))
	SetShortcut(accelerator string) error
	// SetHoldKey chooses the hold-to-talk key, or HoldOff.
	SetHoldKey(key HoldKey)
	RequestInputMonitoring()
	// RequestScreenRecording asks for the consent screen context needs.
	RequestScreenRecording()
	// CloseGuide dismisses the consent guide and returns to Arion.
	CloseGuide()
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
	// ChooseScene keeps scene for SceneTarget's site or app; empty follows
	// the built-in rules again.
	ChooseScene(scene domain.Scene)
	OpenSettings()
	// OpenUpdate shows the update in Settings › About.
	OpenUpdate()
	// CheckForUpdates asks the release feed now.
	CheckForUpdates()
	// InstallUpdate downloads, verifies and installs the available version.
	InstallUpdate()
	// RestartToUpdate relaunches into the installed version.
	RestartToUpdate()
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

// Scene selection for the menu bar card (ADR 012), re-exported so views
// never import the core domain.
type (
	Scene          = domain.Scene
	SceneHints     = domain.SceneHints
	SceneOverrides = domain.SceneOverrides
	SceneDecision  = domain.SceneDecision
	FieldKind      = domain.FieldKind
)

const (
	SceneChat   = domain.SceneChat
	SceneEmail  = domain.SceneEmail
	SceneWork   = domain.SceneWork
	ScenePrompt = domain.ScenePrompt
	SceneSearch = domain.SceneSearch
	SceneByUser = domain.RouteUser
	FieldText   = domain.FieldText
	FieldSearch = domain.FieldSearch
)

// ChoosableScenes are the scenes the card offers.
var ChoosableScenes = domain.ChoosableScenes

// DecideScene is the scene a target gets with the user's choices.
func DecideScene(h SceneHints, o SceneOverrides) SceneDecision { return domain.DecideScene(h, o) }

// SceneKey names the place a choice applies to.
func SceneKey(h SceneHints) string { return domain.SceneKey(h) }
