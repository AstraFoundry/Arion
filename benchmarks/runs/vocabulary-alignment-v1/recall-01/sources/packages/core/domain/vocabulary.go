package domain

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxRecallTerms bounds the memory entries shown to the rewrite model for
// one text, so a large memory never grows the prompt.
const MaxRecallTerms = 8

// Vocabulary indexes replacements by their first lowercased rune so that
// applying and recalling them scans the text once, whatever the number of
// entries (edit-learning-v1 measures this against the entry count).
// It is immutable once built and safe for concurrent use.
type Vocabulary struct {
	entries []Replacement
	// from indexes each entry's From; both indexes each From and To.
	from map[rune][]pattern
	both map[rune][]pattern
}

type pattern struct {
	runes []rune
	entry int
}

// NewVocabulary builds the index. Empty and identity entries are skipped;
// when two entries share a From, the later one wins.
func NewVocabulary(replacements []Replacement) *Vocabulary {
	v := &Vocabulary{from: map[rune][]pattern{}, both: map[rune][]pattern{}}
	seen := map[string]int{}
	for _, r := range replacements {
		r.From, r.To = strings.TrimSpace(r.From), strings.TrimSpace(r.To)
		if r.From == "" || r.To == "" || r.From == r.To {
			continue
		}
		key := strings.ToLower(r.From)
		if at, ok := seen[key]; ok {
			v.entries[at] = r
			continue
		}
		seen[key] = len(v.entries)
		v.entries = append(v.entries, r)
	}
	for i, r := range v.entries {
		v.add(v.from, r.From, i)
		v.add(v.both, r.From, i)
		v.add(v.both, r.To, i)
	}
	// Longest patterns first, so the longest match wins at a position.
	for _, index := range []map[rune][]pattern{v.from, v.both} {
		for k := range index {
			list := index[k]
			sort.SliceStable(list, func(a, b int) bool { return len(list[a].runes) > len(list[b].runes) })
		}
	}
	return v
}

func (v *Vocabulary) add(index map[rune][]pattern, text string, entry int) {
	runes := lowerRunes(text)
	index[runes[0]] = append(index[runes[0]], pattern{runes: runes, entry: entry})
}

// Len is the number of usable entries.
func (v *Vocabulary) Len() int {
	if v == nil {
		return 0
	}
	return len(v.entries)
}

// Apply replaces every From in one left-to-right pass: case-insensitive
// whole-word Latin matches and exact CJK matches, the longest match first.
// A replaced span is never matched again, so entries do not chain. It
// returns the entries that fired, in text order, without duplicates.
func (v *Vocabulary) Apply(s string) (string, []Replacement) {
	if v.Len() == 0 || s == "" {
		return s, nil
	}
	text := []rune(s)
	lower := lowerRunesOf(text)
	var b strings.Builder
	var fired []Replacement
	used := map[int]bool{}
	last := 0
	for k := 0; k < len(text); {
		p, ok := v.match(v.from, text, lower, k)
		if !ok {
			k++
			continue
		}
		b.WriteString(string(text[last:k]))
		b.WriteString(v.entries[p.entry].To)
		if !used[p.entry] {
			used[p.entry] = true
			fired = append(fired, v.entries[p.entry])
		}
		k += len(p.runes)
		last = k
	}
	if fired == nil {
		return s, nil
	}
	b.WriteString(string(text[last:]))
	return b.String(), fired
}

// Terms returns canonical spellings whose From or To occurs in text.
// Multiple aliases share one recall slot. Incorrect forms remain replacement
// evidence only; they must not become spelling candidates in model context.
func (v *Vocabulary) Terms(text string, limit int) []string {
	if v.Len() == 0 || text == "" || limit <= 0 {
		return nil
	}
	runes := []rune(text)
	lower := lowerRunesOf(runes)
	var out []string
	used := map[string]bool{}
	for k := 0; k < len(runes) && len(out) < limit; {
		p, ok := v.match(v.both, runes, lower, k)
		if !ok {
			k++
			continue
		}
		canonical := v.entries[p.entry].To
		if !used[canonical] {
			used[canonical] = true
			out = append(out, canonical)
		}
		k += len(p.runes)
	}
	return out
}

// TermsForSources shares a canonical recall budget across recognized sources in
// priority order. Callers must never supply generated text as recall evidence.
func (v *Vocabulary) TermsForSources(limit int, sources ...string) []string {
	var out []string
	seen := map[string]bool{}
	for _, source := range sources {
		for _, term := range v.Terms(source, limit) {
			if len(out) >= limit {
				return out
			}
			if !seen[term] {
				seen[term] = true
				out = append(out, term)
			}
		}
	}
	return out
}

// match finds the longest pattern starting at k that sits on word
// boundaries.
func (v *Vocabulary) match(index map[rune][]pattern, text, lower []rune, k int) (pattern, bool) {
	for _, p := range index[lower[k]] {
		end := k + len(p.runes)
		if end > len(lower) || !equalRunes(lower[k:end], p.runes) {
			continue
		}
		if runeBoundary(text, k, end) {
			return p, true
		}
	}
	return pattern{}, false
}

// runeBoundary rejects a match that splits a Latin word or number.
func runeBoundary(text []rune, start, end int) bool {
	if start > 0 && isLatinOrDigit(text[start-1]) && isLatinOrDigit(text[start]) {
		return false
	}
	if end < len(text) && isLatinOrDigit(text[end]) && isLatinOrDigit(text[end-1]) {
		return false
	}
	return true
}

func equalRunes(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func lowerRunes(s string) []rune { return lowerRunesOf([]rune(s)) }

func lowerRunesOf(runes []rune) []rune {
	out := make([]rune, len(runes))
	for i, r := range runes {
		if r < utf8.RuneSelf {
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
		} else {
			r = unicode.ToLower(r)
		}
		out[i] = r
	}
	return out
}
