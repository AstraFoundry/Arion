package domain

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

var ErrInvalidKeyboardPinyin = errors.New("keyboard reading requires bounded lowercase pinyin syllables separated by spaces")

// KeyboardTerm is a projection of approved memory, not an independent registry.
// Recognition counts and correction aliases do not become keyboard frequency.
type KeyboardTerm struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Reading string `json:"reading"`
}

var ErrInvalidKeyboardVocabulary = errors.New("invalid or over-budget keyboard vocabulary projection")

// KeyboardTerms takes already-approved memory and returns a deterministic full
// snapshot. Invalid explicitly bound records reject the snapshot atomically.
func KeyboardTerms(entries []MemoryEntry) ([]KeyboardTerm, error) {
	terms := make([]KeyboardTerm, 0)
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.KeyboardPinyin == "" {
			continue
		}
		if entry.ID == "" || len(entry.ID) > 128 || !utf8.ValidString(entry.ID) || strings.ContainsRune(entry.ID, 0) || seen[entry.ID] ||
			(entry.Origin != OriginUser && entry.Origin != OriginEdit) || ValidateKeyboardPinyin(entry.To, entry.KeyboardPinyin) != nil {
			return nil, ErrInvalidKeyboardVocabulary
		}
		seen[entry.ID] = true
		terms = append(terms, KeyboardTerm{ID: entry.ID, Text: entry.To, Reading: entry.KeyboardPinyin})
		if len(terms) > MaxMemoryEntries {
			return nil, ErrInvalidKeyboardVocabulary
		}
	}
	sort.Slice(terms, func(i, j int) bool { return terms[i].ID < terms[j].ID })
	return terms, nil
}

// NormalizeKeyboardPinyin puts a typed reading in transport form without
// changing what it says. Full-width letters and spaces, as a Chinese input
// method types them, become ASCII; letters become lowercase; ü becomes v,
// as Pinyin keyboards spell it; and runs of spaces or syllable apostrophes
// (xi'an) become one space. Anything else is left for
// ValidateKeyboardPinyin to reject.
func NormalizeKeyboardPinyin(reading string) string {
	var b strings.Builder
	for _, r := range reading {
		switch {
		case r >= '\uff01' && r <= '\uff5e':
			r -= 0xfee0
		case r == '\u3000':
			r = ' '
		}
		switch {
		case r >= 'A' && r <= 'Z':
			r += 'a' - 'A'
		case r == '\u00fc' || r == '\u00dc':
			r = 'v'
		case r == '\'' || r == '\u2019':
			r = ' '
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// ValidateKeyboardPinyin checks the transport form, not linguistic correctness.
// The native dictionary must additionally resolve each syllable before use.
// Empty explicitly removes the reading. No character-to-Pinyin guessing occurs.
func ValidateKeyboardPinyin(text, reading string) error {
	if reading == "" {
		return nil
	}
	term, err := NormalizeHotword(text)
	if err != nil || term != text || len(reading) > 448 {
		return ErrInvalidKeyboardPinyin
	}
	syllables := strings.Split(reading, " ")
	if len(syllables) > 64 {
		return ErrInvalidKeyboardPinyin
	}
	for _, syllable := range syllables {
		if len(syllable) == 0 || len(syllable) > 6 {
			return ErrInvalidKeyboardPinyin
		}
		for _, letter := range syllable {
			if letter < 'a' || letter > 'z' {
				return ErrInvalidKeyboardPinyin
			}
		}
	}
	return nil
}
