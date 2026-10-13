package application

import (
	"strings"

	"arion.local/core/domain"
)

// Delivery selects how shortcut-mode commits reach other applications.
type Delivery string

const (
	// DeliveryAutomatic picks Accessibility insertion for native editors and
	// typed keystrokes elsewhere, once, when dictation starts.
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
	ModelOverride       string           `json:"modelOverride"`
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
		Delivery: DeliveryAutomatic, RemoveFillers: true, Sounds: true,
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
	return p, repaired
}
