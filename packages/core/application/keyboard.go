package application

import (
	"slices"
	"strconv"
)

// CandidateSize is one of the candidate bar's size levels.
type CandidateSize string

const (
	CandidateSmall  CandidateSize = "small"
	CandidateMedium CandidateSize = "medium"
	CandidateLarge  CandidateSize = "large"
)

// CandidateAppearance chooses the candidate bar's theme.
type CandidateAppearance string

const (
	AppearanceSystem CandidateAppearance = "system"
	AppearanceLight  CandidateAppearance = "light"
	AppearanceDark   CandidateAppearance = "dark"
)

// CandidateAccent chooses the fill of the highlighted candidate: a preset,
// or AccentCustom with KeyboardPreferences.AccentColor.
type CandidateAccent string

const (
	// AccentYellow is Sunbeam's yellow, as the app's own primary actions
	// (ADR 007).
	AccentYellow   CandidateAccent = "yellow"
	AccentOrange   CandidateAccent = "orange"
	AccentRed      CandidateAccent = "red"
	AccentPink     CandidateAccent = "pink"
	AccentPurple   CandidateAccent = "purple"
	AccentBlue     CandidateAccent = "blue"
	AccentGreen    CandidateAccent = "green"
	AccentGraphite CandidateAccent = "graphite"
	AccentCustom   CandidateAccent = "custom"
)

// CandidateAccents are the selectable accents in menu order, the custom
// color last.
var CandidateAccents = []CandidateAccent{AccentYellow, AccentOrange, AccentRed, AccentPink, AccentPurple,
	AccentBlue, AccentGreen, AccentGraphite, AccentCustom}

// accentFills are the presets' fills in the light and dark appearances as
// #rrggbb. Blue is Chatterfly's focus box; the others are macOS's system
// colors, so the bar reads as part of the desktop.
var accentFills = map[CandidateAccent][2]string{
	AccentYellow:   {"#ffcc00", "#ffcc00"},
	AccentOrange:   {"#ff9500", "#ff9f0a"},
	AccentRed:      {"#ff3b30", "#ff453a"},
	AccentPink:     {"#ff2d55", "#ff375f"},
	AccentPurple:   {"#af52de", "#bf5af2"},
	AccentBlue:     {"#0088ff", "#0099f2"},
	AccentGreen:    {"#34c759", "#30d158"},
	AccentGraphite: {"#8e8e93", "#98989d"},
}

// CandidateTextColor chooses the text on the highlighted candidate.
type CandidateTextColor string

const (
	TextBlack CandidateTextColor = "black"
	TextWhite CandidateTextColor = "white"
)

// CandidateTextColors are the selectable highlight text colors, black first.
var CandidateTextColors = []CandidateTextColor{TextBlack, TextWhite}

// textColors are the highlight text colors as #rrggbb.
var textColors = map[CandidateTextColor]string{TextBlack: "#000000", TextWhite: "#ffffff"}

// Highlight is the highlighted candidate's fill and text as #rrggbb.
type Highlight struct {
	Fill string `json:"fill"`
	Text string `json:"text"`
}

// Highlight resolves the accent and text color for an appearance. The text
// color is the same in both appearances.
func (k KeyboardPreferences) Highlight(dark bool) Highlight {
	fill := k.AccentColor
	if k.Accent != AccentCustom {
		fills := accentFills[k.Accent]
		fill = fills[0]
		if dark {
			fill = fills[1]
		}
	}
	return Highlight{Fill: fill, Text: textColors[k.ActiveTextColor()]}
}

// ActiveTextColor reports the highlight text color in effect. Empty and
// unknown values are black.
func (k KeyboardPreferences) ActiveTextColor() CandidateTextColor {
	if k.TextColor == TextWhite {
		return TextWhite
	}
	return TextBlack
}

// ValidAccentColor reports whether s is an opaque #rrggbb color.
func ValidAccentColor(s string) bool {
	_, ok := rgb(s)
	return ok
}

// rgb parses #rrggbb into channels between 0 and 1.
func rgb(s string) ([3]float64, bool) {
	var c [3]float64
	if len(s) != 7 || s[0] != '#' {
		return c, false
	}
	for i := range c {
		v, err := strconv.ParseUint(s[1+2*i:3+2*i], 16, 8)
		if err != nil {
			return c, false
		}
		c[i] = float64(v) / 255
	}
	return c, true
}

// CandidatePageSizes are the selectable candidates per page.
var CandidatePageSizes = []int{5, 6, 7, 9}

// LanguageSwitch is the key that toggles Chinese and English inside the
// input method (KBD-04). An empty value means Shift.
type LanguageSwitch string

const (
	LanguageSwitchShift    LanguageSwitch = "shift"
	LanguageSwitchCapsLock LanguageSwitch = "caps_lock"
)

// LanguageSwitches are the selectable keys, Shift first.
var LanguageSwitches = []LanguageSwitch{LanguageSwitchShift, LanguageSwitchCapsLock}

// ActiveLanguageSwitch reports the key in effect. Empty and unknown values,
// including a stored control binding, are Shift.
func (k KeyboardPreferences) ActiveLanguageSwitch() LanguageSwitch {
	if k.LanguageSwitch == LanguageSwitchCapsLock {
		return k.LanguageSwitch
	}
	return LanguageSwitchShift
}

// KeyboardPreferences configure Arion's own input method (ADR 017). The input
// method runs without the app; it reads them from a file the app keeps
// current, when a client activates it.
type KeyboardPreferences struct {
	PageSize      int                 `json:"pageSize"`
	CandidateSize CandidateSize       `json:"candidateSize"`
	Appearance    CandidateAppearance `json:"appearance"`
	Accent        CandidateAccent     `json:"accent"`
	// AccentColor is the custom accent as #rrggbb. It stays when a preset
	// is chosen, so choosing Custom again returns to it.
	AccentColor string `json:"accentColor,omitempty"`
	// TextColor is the text on the highlighted candidate. Empty means black.
	TextColor CandidateTextColor `json:"textColor,omitempty"`
	// EmojiSuggestions adds a small reviewed set of Unicode Emoji alternatives
	// to Chinese Pinyin candidates. The OS renders the characters.
	EmojiSuggestions bool `json:"emojiSuggestions"`
	TypoCorrection   bool `json:"typoCorrection"`
	NeuralRanking    bool `json:"neuralRanking"`
	// Context choices default on, gated by neural ranking and OS permission.
	ContextCursor bool `json:"contextCursor"`
	ContextScreen bool `json:"contextScreen"`
	// FuzzyPinyin is a stable bit set shared with the native keyboard ABI.
	FuzzyPinyin uint32 `json:"fuzzyPinyin"`
	// LanguageSwitch chooses the Chinese/English key. Empty means Shift, so
	// profiles written before the choice keep the original gesture.
	LanguageSwitch LanguageSwitch `json:"languageSwitch,omitempty"`
}

const (
	FuzzyZZH uint32 = 1 << iota
	FuzzyCCH
	FuzzySSH
	FuzzyNL
	FuzzyFH
	FuzzyRL
	FuzzyANANG
	FuzzyENENG
	FuzzyINING
	FuzzyIANIANG
	FuzzyUANUANG
	FuzzyPinyinMask = (1 << iota) - 1
)

// DefaultKeyboardPreferences keep the engine's nine-candidate page and the
// app's yellow accent. The empty text color is black.
func DefaultKeyboardPreferences() KeyboardPreferences {
	return KeyboardPreferences{PageSize: 6, CandidateSize: CandidateMedium, Appearance: AppearanceSystem, Accent: AccentYellow, EmojiSuggestions: true, ContextCursor: true, ContextScreen: true}
}

// normalize repairs fields that cannot be valid.
func (k KeyboardPreferences) normalize() (KeyboardPreferences, bool) {
	defaults := DefaultKeyboardPreferences()
	repaired := false
	if k.FuzzyPinyin & ^uint32(FuzzyPinyinMask) != 0 {
		k.FuzzyPinyin &= FuzzyPinyinMask
		repaired = true
	}
	known := false
	for _, size := range CandidatePageSizes {
		known = known || size == k.PageSize
	}
	if !known {
		k.PageSize, repaired = defaults.PageSize, true
	}
	if k.CandidateSize != CandidateSmall && k.CandidateSize != CandidateMedium && k.CandidateSize != CandidateLarge {
		k.CandidateSize, repaired = defaults.CandidateSize, true
	}
	if k.Appearance != AppearanceSystem && k.Appearance != AppearanceLight && k.Appearance != AppearanceDark {
		k.Appearance, repaired = defaults.Appearance, true
	}
	if k.AccentColor != "" && !ValidAccentColor(k.AccentColor) {
		k.AccentColor, repaired = "", true
	}
	if !slices.Contains(CandidateAccents, k.Accent) || k.Accent == AccentCustom && k.AccentColor == "" {
		k.Accent, repaired = defaults.Accent, true
	}
	if k.TextColor != "" && !slices.Contains(CandidateTextColors, k.TextColor) {
		k.TextColor, repaired = "", true
	}
	switch k.LanguageSwitch {
	case "", LanguageSwitchShift, LanguageSwitchCapsLock:
	default:
		k.LanguageSwitch, repaired = "", true
	}
	return k, repaired
}
