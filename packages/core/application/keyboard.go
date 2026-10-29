package application

import (
	"math"
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

// Text on a highlight: white where it holds 3:1 against the fill, as
// Chatterfly's blue carries it, else Sunbeam's ink.
const (
	highlightWhite = "#ffffff"
	highlightInk   = "#14110f"
)

// Highlight is the highlighted candidate's fill and text as #rrggbb.
type Highlight struct {
	Fill string `json:"fill"`
	Text string `json:"text"`
}

// Highlight resolves the accent for an appearance.
func (k KeyboardPreferences) Highlight(dark bool) Highlight {
	fill := k.AccentColor
	if k.Accent != AccentCustom {
		fills := accentFills[k.Accent]
		fill = fills[0]
		if dark {
			fill = fills[1]
		}
	}
	text := highlightInk
	if contrast(fill, highlightWhite) >= 3 {
		text = highlightWhite
	}
	return Highlight{Fill: fill, Text: text}
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

// contrast is the WCAG contrast ratio of two valid #rrggbb colors.
func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

func luminance(s string) float64 {
	c, _ := rgb(s)
	for i, v := range c {
		if v <= 0.04045 {
			c[i] = v / 12.92
		} else {
			c[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

// CandidatePageSizes are the selectable candidates per page.
var CandidatePageSizes = []int{5, 6, 7, 9}

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
	// EmojiSuggestions adds a small reviewed set of Unicode Emoji alternatives
	// to Chinese Pinyin candidates. The OS renders the characters.
	EmojiSuggestions bool `json:"emojiSuggestions"`
	TypoCorrection   bool `json:"typoCorrection"`
	// FuzzyPinyin is a stable bit set shared with the native keyboard ABI.
	FuzzyPinyin uint32 `json:"fuzzyPinyin"`
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
// app's yellow accent.
func DefaultKeyboardPreferences() KeyboardPreferences {
	return KeyboardPreferences{PageSize: 9, CandidateSize: CandidateMedium, Appearance: AppearanceSystem, Accent: AccentYellow, EmojiSuggestions: true}
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
	return k, repaired
}
