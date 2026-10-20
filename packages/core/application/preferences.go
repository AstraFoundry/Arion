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
	Version             int              `json:"version"`
	UILanguage          UILanguage       `json:"uiLanguage"`
	OnboardingCompleted bool             `json:"onboardingCompleted"`
	Mode                domain.InputMode `json:"mode"`
	Shortcut            string           `json:"shortcut"`
	HoldKey             HoldKey          `json:"holdKey"`
	Delivery            Delivery         `json:"delivery"`
	RemoveFillers       bool             `json:"removeFillers"`
	Sounds              bool             `json:"sounds"`
	LaunchAtLogin       bool             `json:"launchAtLogin"`
	Microphone          string           `json:"microphone"`
	// MicrophoneGain is one of GainStops' decibels; profiles written before
	// it existed load 0 dB, the device's own level.
	MicrophoneGain int    `json:"microphoneGain"`
	ModelOverride  string `json:"modelOverride"`
	// CheckUpdates asks the release feed for a newer version at launch and
	// periodically. Profiles written before it existed load the default.
	CheckUpdates bool `json:"checkUpdates"`
	// Polish turns on the local LLM pipeline: live rewriting while speaking
	// and a validated final rewrite. Off until the user downloads the model.
	Polish bool `json:"polish"`
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
	// default; everything stays on this device. Profiles written before
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
}

// PreferencesVersion is the current schema; older files are migrated.
const PreferencesVersion = 1

// UILanguage affects desktop copy only, independently of recognition language.
type UILanguage string

const (
	English           UILanguage = "en"
	SimplifiedChinese UILanguage = "zh-CN"
)

// DefaultShortcut is Option+Space on macOS and Alt+Space on Windows.
const DefaultShortcut = "Alt+Space"

// DefaultPreferences are the first-run choices.
func DefaultPreferences() Preferences {
	return Preferences{
		Version: PreferencesVersion, UILanguage: English, Mode: domain.ShortcutMode, Shortcut: DefaultShortcut,
		Delivery: DeliveryAutomatic, RemoveFillers: true, Sounds: true, CheckUpdates: true, MatchApp: true,
		ContextCursor: true, ContextScreen: true, ContextHistory: true, LearnEdits: true,
		Keyboard: DefaultKeyboardPreferences(),
	}
}

// Normalize repairs fields that cannot be valid and reports what changed, so
// a damaged file never yields an unusable configuration silently.
func (p Preferences) Normalize() (Preferences, []string) {
	defaults := DefaultPreferences()
	var repaired []string
	if p.UILanguage != English && p.UILanguage != SimplifiedChinese {
		p.UILanguage = defaults.UILanguage
		repaired = append(repaired, "uiLanguage")
	}
	if p.Version != PreferencesVersion {
		p.Version = PreferencesVersion
		repaired = append(repaired, "version")
	}
	if p.Mode != domain.ShortcutMode && p.Mode != domain.SystemIMEMode {
		p.Mode = defaults.Mode
		repaired = append(repaired, "mode")
	}
	if strings.TrimSpace(p.Shortcut) == "" {
		p.Shortcut = defaults.Shortcut
		repaired = append(repaired, "shortcut")
	}
	if !knownHoldKey(p.HoldKey) {
		p.HoldKey = HoldOff
		repaired = append(repaired, "holdKey")
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
	return p, repaired
}
