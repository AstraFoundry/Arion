package application

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

// CandidatePageSizes are the selectable candidates per page.
var CandidatePageSizes = []int{5, 6, 7, 9}

// KeyboardPreferences configure Arion's own input method (ADR 017). The input
// method runs without the app; it reads them from a file the app keeps
// current, when a client activates it.
type KeyboardPreferences struct {
	PageSize      int                 `json:"pageSize"`
	CandidateSize CandidateSize       `json:"candidateSize"`
	Appearance    CandidateAppearance `json:"appearance"`
	// ContextRanking moves the word the user chose after the same previous
	// word to the front while typing, and learns those choices (上下文调频).
	ContextRanking bool `json:"contextRanking"`
}

// DefaultKeyboardPreferences keep the engine's nine-candidate page.
func DefaultKeyboardPreferences() KeyboardPreferences {
	return KeyboardPreferences{PageSize: 9, CandidateSize: CandidateMedium, Appearance: AppearanceSystem, ContextRanking: true}
}

// normalize repairs fields that cannot be valid.
func (k KeyboardPreferences) normalize() (KeyboardPreferences, bool) {
	defaults := DefaultKeyboardPreferences()
	repaired := false
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
	return k, repaired
}
