package application

import (
	"strings"

	"arion.local/core/domain"
)

// Delivery selects how shortcut-mode commits reach other applications.
type Delivery string

const (
	// DeliveryAutomatic uses the platform standard: system paste on macOS,
	// Unicode keyboard input on Windows. IME commits do not use this choice.
	DeliveryAutomatic Delivery = "automatic"
	// DeliveryClipboard pastes through the clipboard and restores it unless
	// another app wrote it meanwhile.
	DeliveryClipboard Delivery = "clipboard"
)

// Preferences are the user's persisted choices. Model selection is absent
// on purpose: the best evaluated bundle loads automatically; ModelOverride
// only exists for an explicit choice among evaluated bundles.
type Preferences struct {
	Version             int                        `json:"version"`
	UILanguage          UILanguage                 `json:"uiLanguage"`
	OnboardingCompleted bool                       `json:"onboardingCompleted"`
	HoldKey             HoldKey                    `json:"holdKey"`
	TranslateKey        HoldKey                    `json:"translateKey,omitempty"`
	TranslateLanguage   domain.TranslationLanguage `json:"translateLanguage,omitempty"`
	Delivery            Delivery                   `json:"delivery"`
	Sounds              bool                       `json:"sounds"`
	LaunchAtLogin       bool                       `json:"launchAtLogin"`
	Microphone          string                     `json:"microphone"`
	// MicrophoneGain is one of GainStops' decibels; profiles written before
	// it existed load 0 dB, the device's own level.
	MicrophoneGain int    `json:"microphoneGain"`
	ModelOverride  string `json:"modelOverride"`
	// CheckUpdates asks the release feed for a newer version at launch and
	// periodically. Profiles written before it existed load the default.
	CheckUpdates bool `json:"checkUpdates"`
	// AutoInstallUpdates downloads and verifies updates when automatic checks are
	// enabled. Installation waits for an explicit protected restart.
	AutoInstallUpdates bool `json:"autoInstallUpdates"`
	// AllowPrereleaseUpdates includes published preview releases when checking.
	AllowPrereleaseUpdates bool `json:"allowPrereleaseUpdates"`
	// Polish enables the shared rewrite after the dictation key is released.
	// Neither engine rewrites while the user is speaking.
	Polish bool `json:"polish"`
	// Punctuation chooses how polished text is punctuated. It is used only
	// while Polish is on. Profiles written before it existed load smart.
	Punctuation domain.Punctuation `json:"punctuation,omitempty"`
	// PolishEngine is where polishing runs: on this device, or with the
	// cloud model the user connected (ADR 022). Profiles written before it
	// existed load PolishLocal.
	PolishEngine PolishEngine `json:"polishEngine"`
	// CloudModel is the model ID chosen for cloud polishing, among those the
	// connected provider lists. Credentials are never preferences.
	CloudModel string `json:"cloudModel,omitempty"`
	// MatchApp lets polishing follow the focused app, field and web site
	// (email, work chat, coding prompt, search) instead of always writing
	// plain chat text. Profiles written before it existed load the default.
	MatchApp bool `json:"matchApp"`
	// SceneOverrides are scenes the user chose for a site or app (by
	// domain.SceneKey) from the menu bar panel. Replace the map, never
	// mutate it: preference snapshots share it.
	SceneOverrides domain.SceneOverrides `json:"sceneOverrides,omitempty"`
	// ContextCursor, ContextScreen and ContextHistory let polishing read the
	// field's text around the cursor, the focused window's recognized text
	// and the user's recent dictations in the same place (ADR 013). On by
	// default; cloud polishing sends enabled sources (ADR 024). Profiles written before
	// they existed load the defaults.
	ContextCursor  bool `json:"contextCursor"`
	ContextScreen  bool `json:"contextScreen"`
	ContextHistory bool `json:"contextHistory"`
	// LearnEdits reads a field Arion inserted into until the user stops
	// editing it, and remembers the words they corrected (ADR 014). On by
	// default; profiles written before it existed load the default.
	LearnEdits bool `json:"learnEdits"`
	// Keyboard configures Arion's input method. Profiles written before it
	// existed load the defaults.
	Keyboard KeyboardPreferences `json:"keyboard"`
	// AccountSubject, AccountName and AccountEmail remember the last signed-in
	// profile line for Settings. They are not credentials; tokens stay in the
	// OS store. Profiles written before they existed load empty.
	AccountSubject string `json:"accountSubject,omitempty"`
	AccountName    string `json:"accountName,omitempty"`
	AccountEmail   string `json:"accountEmail,omitempty"`
}

// PolishEngine selects where polishing runs.
type PolishEngine string

const (
	PolishLocal PolishEngine = "local"
	// PolishCloud sends text, recalled terms and enabled context to
	// the connected provider; disabled context sources stay on the device.
	PolishCloud PolishEngine = "cloud"
)

// PreferencesVersion is the current schema; older files are migrated.
const PreferencesVersion = 1

// UILanguage affects desktop copy only, independently of recognition language.
type UILanguage string

const (
	English           UILanguage = "en"
	SimplifiedChinese UILanguage = "zh-CN"
)

// DefaultPreferences are the first-run choices on platform (a GOOS value).
func DefaultPreferences(platform string) Preferences {
	return Preferences{
		Version: PreferencesVersion, UILanguage: English, HoldKey: DefaultHoldKey(platform),
		TranslateKey: DefaultTranslateKey(platform), TranslateLanguage: domain.TargetEnglish,
		Delivery: DeliveryAutomatic, PolishEngine: PolishLocal, Punctuation: domain.PunctuationSmart, Sounds: true, CheckUpdates: true, MatchApp: true,
		ContextCursor: true, ContextScreen: true, ContextHistory: true, LearnEdits: true,
		Keyboard: DefaultKeyboardPreferences(),
	}
}

// Normalize repairs fields that cannot be valid and reports what changed, so
// a damaged file never yields an unusable configuration silently. A hold key
// this platform cannot observe, including the retired "off", becomes the
// platform default.
func (p Preferences) Normalize(platform string) (Preferences, []string) {
	defaults := DefaultPreferences(platform)
	var repaired []string
	if p.UILanguage != English && p.UILanguage != SimplifiedChinese {
		p.UILanguage = defaults.UILanguage
		repaired = append(repaired, "uiLanguage")
	}
	if p.Version != PreferencesVersion {
		p.Version = PreferencesVersion
		repaired = append(repaired, "version")
	}
	if !knownHoldKey(platform, p.HoldKey) {
		p.HoldKey = defaults.HoldKey
		repaired = append(repaired, "holdKey")
	}
	if !knownTranslateKey(platform, p.TranslateKey) {
		p.TranslateKey = defaults.TranslateKey
		repaired = append(repaired, "translateKey")
	}
	if p.TranslateKey != "" && p.TranslateKey != HoldOff && p.TranslateKey == p.HoldKey {
		p.TranslateKey = HoldOff
	}
	if !p.TranslateLanguage.Known() {
		p.TranslateLanguage = defaults.TranslateLanguage
		repaired = append(repaired, "translateLanguage")
	}
	if p.PolishEngine != PolishLocal && p.PolishEngine != PolishCloud {
		p.PolishEngine = defaults.PolishEngine
		repaired = append(repaired, "polishEngine")
	}
	if !p.Punctuation.Known() {
		p.Punctuation = defaults.Punctuation
		repaired = append(repaired, "punctuation")
	}
	if p.Delivery != DeliveryAutomatic && p.Delivery != DeliveryClipboard {
		p.Delivery = defaults.Delivery
		repaired = append(repaired, "delivery")
	}
	for key, scene := range p.SceneOverrides {
		if key == "" || !scene.Known() || scene == domain.SceneList {
			p.SceneOverrides = p.SceneOverrides.WithScene(key, "")
			repaired = append(repaired, "sceneOverrides")
		}
	}
	if keyboard, changed := p.Keyboard.normalize(); changed {
		p.Keyboard = keyboard
		repaired = append(repaired, "keyboard")
	}
	if !knownGain(p.MicrophoneGain) {
		p.MicrophoneGain = defaults.MicrophoneGain
		repaired = append(repaired, "microphoneGain")
	}
	if next := p.WithAccountDisplay(p.AccountSubject, p.AccountName, p.AccountEmail); next.AccountSubject != p.AccountSubject || next.AccountName != p.AccountName || next.AccountEmail != p.AccountEmail {
		p = next
		repaired = append(repaired, "accountDisplay")
	}
	return p, repaired
}

// WithAccountDisplay stores the signed-in profile line, or clears it when
// subject is empty. Tokens are never written here.
func (p Preferences) WithAccountDisplay(subject, name, email string) Preferences {
	p.AccountSubject = strings.TrimSpace(subject)
	p.AccountName = strings.TrimSpace(name)
	p.AccountEmail = strings.TrimSpace(email)
	if p.AccountSubject == "" {
		p.AccountName = ""
		p.AccountEmail = ""
	}
	return p
}

// DisplayForAccount returns the remembered name and email when they belong
// to subject. A different or empty subject is not a match.
func (p Preferences) DisplayForAccount(subject string) (name, email string, ok bool) {
	subject = strings.TrimSpace(subject)
	if subject == "" || p.AccountSubject != subject {
		return "", "", false
	}
	name, email = strings.TrimSpace(p.AccountName), strings.TrimSpace(p.AccountEmail)
	return name, email, name != "" || email != ""
}
