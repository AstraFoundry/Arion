// Package vocabulary projects approved records as the terms Settings shows.
package vocabulary

import (
	"slices"
	"strings"

	"arion.local/desktop/ui/shared/runtime"
)

// Term is one correct spelling with everything recorded for it. Each record
// keeps its identity for independent removal, undo and keyboard binding.
type Term struct {
	Text string
	// Records holds every record of the term, so removing the term removes
	// them all.
	Records []runtime.Replacement
	// Variants are the records that replace an incorrect form.
	Variants []runtime.Replacement
	// Readings are the records that carry a keyboard Pinyin reading. Without
	// one it holds the record a new reading binds to: the correct-only record,
	// else the first variant.
	Readings []runtime.Replacement
}

// Groups lists the most recently added term first. A matching correct
// spelling or variant reveals the complete term, so searching never hides the
// other spellings it owns.
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
		term := &terms[position]
		term.Records = append(term.Records, entry)
		if entry.From != "" {
			term.Variants = append(term.Variants, entry)
		}
		if entry.KeyboardPinyin != "" {
			term.Readings = append(term.Readings, entry)
		}
	}
	for i := range terms {
		if len(terms[i].Readings) == 0 {
			terms[i].Readings = []runtime.Replacement{anchor(terms[i].Records)}
		}
	}
	slices.Reverse(terms)
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

func anchor(records []runtime.Replacement) runtime.Replacement {
	for _, record := range records {
		if record.From == "" {
			return record
		}
	}
	return records[0]
}
