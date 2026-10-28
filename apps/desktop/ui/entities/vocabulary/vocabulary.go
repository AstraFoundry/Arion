// Package vocabulary projects approved correction pairs as canonical terms.
package vocabulary

import (
	"strings"

	"arion.local/desktop/ui/shared/runtime"
)

// Term preserves each variant's identity for independent removal and undo.
type Term struct {
	Text       string
	Hotword    bool
	HotwordIDs []string
	Variants   []runtime.Replacement
}

// Groups keeps the store's order. A matching canonical term or variant reveals
// the complete group, so searching never hides the other spellings it owns.
func Groups(entries []runtime.Replacement, query string) []Term {
	var terms []Term
	positions := make(map[string]int)
	for _, entry := range entries {
		position, exists := positions[entry.To]
		if !exists {
			position = len(terms)
			positions[entry.To] = position
			terms = append(terms, Term{Text: entry.To})
		}
		terms[position].Hotword = terms[position].Hotword || entry.Hotword
		if entry.From == "" && entry.Hotword {
			terms[position].HotwordIDs = append(terms[position].HotwordIDs, entry.ID)
		} else {
			terms[position].Variants = append(terms[position].Variants, entry)
		}
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return terms
	}
	var matched []Term
	for _, term := range terms {
		matches := strings.Contains(strings.ToLower(term.Text), query)
		for _, variant := range term.Variants {
			matches = matches || strings.Contains(strings.ToLower(variant.From), query)
		}
		if matches {
			matched = append(matched, term)
		}
	}
	return matched
}
