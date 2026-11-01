package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// KeyboardCompletion is provisional text covering the entire raw composition.
// Reading is a complete, space-separated pronunciation, never a model score.
type KeyboardCompletion struct {
	Input   string `json:"input"`
	Text    string `json:"text"`
	Reading string `json:"reading"`
}

// MatchKeyboardReading matches full syllables or initials against raw input.
// Explicit apostrophes are boundaries; no deletion, correction or extra suffix
// is accepted. Dynamic programming preserves ambiguous syllable boundaries.
func MatchKeyboardReading(input, reading string) bool {
	if len(input) == 0 || len(input) > 256 || len(reading) > 448 {
		return false
	}
	for _, c := range input {
		if (c < 'a' || c > 'z') && c != '\'' {
			return false
		}
	}
	syllables := strings.Split(reading, " ")
	if len(syllables) < 2 || len(syllables) > 32 {
		return false
	}
	positions := map[int]bool{0: true}
	for _, syllable := range syllables {
		if len(syllable) == 0 || len(syllable) > 6 {
			return false
		}
		for _, c := range syllable {
			if c < 'a' || c > 'z' {
				return false
			}
		}
		choices := []string{syllable, syllable[:1]}
		if strings.HasPrefix(syllable, "zh") || strings.HasPrefix(syllable, "ch") || strings.HasPrefix(syllable, "sh") {
			choices = append(choices, syllable[:2])
		}
		next := map[int]bool{}
		for p := range positions {
			if p > 0 && p < len(input) && input[p] == '\'' {
				p++
			}
			for _, choice := range choices {
				if strings.HasPrefix(input[p:], choice) {
					next[p+len(choice)] = true
				}
			}
		}
		positions = next
	}
	return positions[len(input)]
}

func ValidKeyboardCompletion(c KeyboardCompletion, input string) bool {
	if c.Input != input || !utf8.ValidString(c.Text) || utf8.RuneCountInString(c.Text) != len(strings.Split(c.Reading, " ")) || !MatchKeyboardReading(input, c.Reading) {
		return false
	}
	for _, r := range c.Text {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return true
}
