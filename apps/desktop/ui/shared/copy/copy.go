// Package copy centralizes all user-facing desktop text.
package copy

import "fmt"

const (
	Title   = "Arion"
	Tagline = "Local voice input"
)

// Common actions.
const (
	Continue      = "Continue"
	Back          = "Back"
	Skip          = "Skip"
	Done          = "Done"
	Retry         = "Try again"
	Copy          = "Copy"
	Copied        = "Copied"
	Add           = "Add"
	Remove        = "Remove"
	Allow         = "Allow"
	OpenSettings  = "Open System Settings"
	SettingsMenu  = "Settings…"
	Quit          = "Quit Arion"
	StartDictate  = "Start dictation"
	StopDictate   = "Finish"
	CancelDictate = "Cancel"
)

// Status words shared by the menu bar and settings.
const (
	StatusReady        = "Ready"
	StatusListening    = "Listening"
	StatusTranscribing = "Transcribing"
	StatusInserting    = "Inserting"
	StatusPreparing    = "Preparing model"
	StatusModelFailed  = "Model unavailable"
	StatusPermission   = "Needs permission"
	StatusGranted      = "Allowed"
	StatusDenied       = "Not allowed"
	StatusUndetermined = "Not set"
)

// Overlay pill.
const (
	PillListening    = "Listening…"
	PillTranscribing = "Transcribing…"
	PillInserted     = "Inserted"
	PillTyped        = "Typed"
	PillPasted       = "Pasted"
	PillCancelled    = "Cancelled"
	PillShortcutHint = "to finish"
	PillEscHint      = "esc"
)

// Failure explanations: a title and the next step.
const (
	FailMicTitle           = "Microphone access needed"
	FailMicHint            = "Allow it in Arion's settings"
	FailAXTitle            = "Accessibility access needed"
	FailAXHint             = "Arion needs it to type into other apps"
	FailModelTitle         = "Speech model not ready"
	FailModelHint          = "Setup continues in the menu bar"
	FailDiscontinuityTitle = "Microphone stopped"
	FailDiscontinuityHint  = "The input device changed or disconnected"
	FailDeviceTitle        = "Microphone unavailable"
	FailDeviceHint         = "Choose another input in Settings"
	FailTargetTitle        = "Focus moved, nothing inserted"
	FailTargetHint         = "Copy the text from the menu bar"
	FailUnsupportedTitle   = "This field can't take text"
	FailSecureTitle        = "Password fields are off-limits"
	FailSecureHint         = "Arion never listens into secure fields"
	FailNoSpeechTitle      = "No speech heard"
	FailNoSpeechHint       = "Check the microphone level in Settings"
	FailBusyTitle          = "Still transcribing"
	FailIMETitle           = "Arion Voice isn't the active input source"
	FailIMEHint            = "Choose it in the input menu, then try again"
	FailGenericTitle       = "Dictation stopped"
)

// Menu bar panel.
const (
	PanelLastDictation = "Last dictation"
	PanelEmpty         = "Your dictations appear here until you quit."
	PanelModel         = "Model"
	PanelAutomatic     = "Chosen automatically"
	PanelChosen        = "Chosen by you"
	TrayToolTip        = "Arion"
	TrayStart          = "Start Dictation"
	TrayStop           = "Finish Dictation"
	TraySettings       = "Settings…"
	TrayQuit           = "Quit Arion"
)

// Settings navigation.
const (
	SettingsTitle = "Arion Settings"
	NavGeneral    = "General"
	NavDictation  = "Dictation"
	NavMicrophone = "Microphone"
	NavModel      = "Model"
	NavVocabulary = "Vocabulary"
	NavPrivacy    = "Privacy"
	NavAbout      = "About"
)

// General.
const (
	LaunchAtLogin     = "Launch at login"
	Sounds            = "Sounds"
	SoundsDetail      = "A soft cue when listening starts and stops"
	PermissionsHeader = "Permissions"
	MicPermission     = "Microphone"
	MicPermissionText = "Used only while you dictate"
	AXPermission      = "Accessibility"
	AXPermissionText  = "Lets Arion type into the app you're using"
)

// Dictation.
const (
	ShortcutLabel         = "Dictation shortcut"
	ShortcutDetail        = "Press once to start, again to finish. Esc cancels."
	ShortcutRecord        = "Change"
	ShortcutRecording     = "Press a key combination…"
	ShortcutTaken         = "That shortcut is in use by another app"
	ShortcutNeedsModifier = "Include ⌘, ⌥, ⌃ or ⇧"
	ModeLabel             = "Input mode"
	ModeShortcut          = "Shortcut"
	ModeShortcutDetail    = "Press your shortcut in any app; the text arrives when you finish"
	ModeIME               = "Input method"
	ModeIMEDetail         = "Choose Arion Voice in the input menu; your words appear underlined as you speak"
	IMEName               = "Arion Voice input source"
	IMEAbsent             = "Not installed"
	IMEInstalled          = "Installed. Choose it in the input menu."
	IMESelected           = "Active"
	IMEUnsupported        = "Not available on this system yet"
	IMEInstall            = "Install"
	IMERemove             = "Remove"
	IMEFailed             = "The input source could not be installed"
	DeliveryLabel         = "Insertion"
	DeliveryAutomatic     = "Automatic"
	DeliveryAutoDetail    = "Accessibility for native apps, typing everywhere else"
	DeliveryClipboard     = "Paste"
	DeliveryClipDetail    = "Uses the clipboard and puts your previous contents back"
	FillersLabel          = "Remove filler words"
	FillersDetail         = "嗯, 呃, um and uh, when they stand alone"
)

// Microphone.
const (
	MicDevice = "Input device"
	MicSystem = "System default"
	MicTest   = "Speak to test the level"
	MicNone   = "No microphone found"
)

// Model.
const (
	ModelRecommended   = "Recommended for this Mac"
	ModelReason        = "Lowest error rate on Arion's Chinese and English tests."
	ModelReasonRemote  = "Lowest error rate on Arion's tests; speed not yet measured on this Mac."
	ModelMandarin      = "Mandarin"
	ModelEnglish       = "English"
	ModelMixed         = "Mixed"
	ModelLatency       = "Finish to text"
	ModelUse           = "Use"
	ModelUseRecommend  = "Use recommended"
	ModelOthers        = "Other evaluated models"
	ModelPreview       = "Live preview"
	ModelFinalPass     = "Final pass"
	ModelStreamingOnly = "Streaming only"
	ModelVerified      = "Verified and offline"
	ModelDownloading   = "Downloading"
	ModelVerifying     = "Verifying"
	ModelUnpacking     = "Unpacking"
	ModelLoading       = "Loading"
	ModelNoEvaluation  = "No evaluated model for this Mac"
	ModelFailChecksum  = "The download didn't match its pinned checksum."
	ModelFailNetwork   = "The download stopped. Check your connection."
)

// Vocabulary.
const (
	VocabHeard       = "When Arion hears"
	VocabWrite       = "Write"
	VocabEmpty       = "Add names and terms Arion should spell your way."
	VocabFromHint    = "e.g. 阿里昂"
	VocabFromLabel   = "Heard as"
	VocabToLabel     = "Write as"
	VocabToHint      = "e.g. Arion"
	VocabInvalid     = "Both fields are needed"
	VocabStoredLocal = "Encrypted on this Mac."
)

// Privacy.
const (
	PrivacyLocal    = "Speech is recognized on this Mac"
	PrivacyLocalTxt = "No audio or text is sent to any server."
	PrivacyAudio    = "Audio is never saved"
	PrivacyAudioTxt = "Recordings live in memory and are dropped when a dictation ends."
	PrivacyLogs     = "Nothing you say is logged"
	PrivacyLogsTxt  = "Diagnostics hold only timings and error codes."
	PrivacyNetwork  = "One network use: model downloads"
	PrivacyNetTxt   = "Each file is checked against a pinned SHA-256 before it loads."
)

// About.
const (
	AboutVersion  = "Version"
	AboutLicenses = "Open-source components"
	AboutCredits  = "Speech recognition by sherpa-onnx and ONNX Runtime. Icons by Lucide."
)

// Onboarding.
const (
	OnbStepWelcome    = "Welcome"
	OnbStepMic        = "Microphone"
	OnbStepAX         = "Accessibility"
	OnbStepModel      = "Model"
	OnbStepShortcut   = "Shortcut"
	OnbStepTry        = "Try it"
	OnbWelcomeTitle   = "Speak. Arion types."
	OnbWelcomeBody    = "Dictation that runs entirely on your Mac, in Chinese, English, or both at once."
	OnbStart          = "Get started"
	OnbMicTitle       = "Let Arion hear you"
	OnbMicBody        = "The microphone is on only while you dictate."
	OnbMicAllow       = "Allow microphone"
	OnbMicDenied      = "Microphone access is off. Turn it on for Arion in System Settings."
	OnbMicTest        = "Say something. The bars should move."
	OnbAXTitle        = "Let Arion type for you"
	OnbAXBody         = "Accessibility access lets Arion put text where your cursor is."
	OnbAXAllow        = "Open Accessibility settings"
	OnbAXWaiting      = "Waiting for access. Switch Arion on in the list."
	OnbAXGranted      = "Access granted"
	OnbModelTitle     = "Setting up recognition"
	OnbModelBody      = "Arion picked the model that scored best in its Chinese–English tests. It downloads once and then works offline."
	OnbModelReady     = "Ready to use offline"
	OnbShortcutTitle  = "Your dictation key"
	OnbShortcutBody   = "Press it once to start and again to finish. Esc cancels."
	OnbShortcutPress  = "Press it now to try"
	OnbShortcutHeard  = "That works"
	OnbTryTitle       = "Try it"
	OnbTryBody        = "Click the box, press your shortcut, and say a sentence."
	OnbTryPlaceholder = "Your words will appear here"
	OnbDoneTitle      = "You're set"
	OnbDoneBody       = "Arion waits in the menu bar. Press your shortcut in any app."
	OnbFinish         = "Start dictating"
)

// Bytes formats a size for people.
func Bytes(n int64) string {
	const mb = 1 << 20
	if n >= 1<<30 {
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	}
	return fmt.Sprintf("%d MB", (n+mb/2)/mb)
}

// Progress formats a download such as "63 MB of 134 MB".
func Progress(done, total int64) string { return fmt.Sprintf("%s of %s", Bytes(done), Bytes(total)) }

// Percent formats an error rate.
func Percent(rate float64) string { return fmt.Sprintf("%.1f%%", rate*100) }

// Millis formats a latency.
func Millis(ms float64) string { return fmt.Sprintf("%.0f ms", ms) }

// Version formats the app version line.
func Version(v string) string { return fmt.Sprintf("%s %s", AboutVersion, v) }

// Step formats onboarding progress for assistive technology.
func Step(index, total int) string { return fmt.Sprintf("Step %d of %d", index, total) }
