package domain

import (
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxASRHotwords     = 64
	MaxASRContextRunes = 2048
	MaxHotwordRunes    = 64
)

var ErrInvalidHotword = errors.New("hotword must be a bounded term without control characters or model tokens")
var ErrASRContextBudget = errors.New("ASR hotword context exceeds its budget")

// NormalizeHotword validates one explicit canonical spelling. It never truncates.
func NormalizeHotword(text string) (string, error) {
	if !validVocabularyText(text) || strings.Contains(text, "<|") || strings.Contains(text, "|>") {
		return "", ErrInvalidHotword
	}
	for _, r := range text {
		if unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			return "", ErrInvalidHotword
		}
	}
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > MaxHotwordRunes {
		return "", ErrInvalidHotword
	}
	return text, nil
}

// ASRContext validates a final request's canonical terms before native transport.
// The joined context, including separators, has its own bounded rune budget.
func ASRContext(hotwords []string) (string, error) {
	if len(hotwords) > MaxASRHotwords {
		return "", ErrASRContextBudget
	}
	for _, word := range hotwords {
		normalized, err := NormalizeHotword(word)
		if err != nil || normalized != word {
			return "", ErrInvalidHotword
		}
	}
	context := strings.Join(hotwords, " ")
	if utf8.RuneCountInString(context) > MaxASRContextRunes {
		return "", ErrASRContextBudget
	}
	return context, nil
}

// ASRHotwordEligible reports whether one approved memory record can supply
// a canonical term, independently of the per-session selection budget.
func ASRHotwordEligible(entry MemoryEntry) bool {
	if !entry.Hotword && entry.Origin != OriginEdit {
		return false
	}
	term, err := NormalizeHotword(entry.To)
	if err != nil || term != entry.To {
		return false
	}
	return entry.Origin != OriginEdit || utf8.RuneCountInString(term) <= 16
}

// ASRHotwords selects approved memory's canonical spellings, never aliases.
// Explicit hotwords precede learned terms, then use+recall counts, recency,
// canonical spelling and ID break ties. Exact canonical spellings share a slot;
// case-distinct spellings remain distinct. Invalid legacy terms are ineligible.
func ASRHotwords(entries []MemoryEntry) []string {
	ranked := make([]MemoryEntry, 0, len(entries))
	for _, entry := range entries {
		if ASRHotwordEligible(entry) {
			ranked = append(ranked, entry)
		}
	}
	explicit := func(e MemoryEntry) bool { return e.Origin == OriginUser && e.Hotword }
	sort.Slice(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if explicit(a) != explicit(b) {
			return explicit(a)
		}
		if a.Uses+a.Recalls != b.Uses+b.Recalls {
			return a.Uses+a.Recalls > b.Uses+b.Recalls
		}
		if !a.Used.Equal(b.Used) {
			return a.Used.After(b.Used)
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.ID < b.ID
	})
	var out []string
	seen := map[string]bool{}
	size := 0
	for _, entry := range ranked {
		if seen[entry.To] {
			continue
		}
		n := utf8.RuneCountInString(entry.To)
		if len(out) > 0 {
			n++
		}
		if size+n > MaxASRContextRunes {
			continue
		}
		out = append(out, entry.To)
		seen[entry.To] = true
		size += n
		if len(out) == MaxASRHotwords {
			break
		}
	}
	return out
}
