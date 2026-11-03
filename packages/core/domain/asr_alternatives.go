package domain

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxASRAlternativeTerms = 8
	maxASRAlternativeSites = 128
	maxASRPlanBytes        = 64 << 10
	maxASRResolveStates    = 65536
	maxASRNominations      = 4096
)

// ASRPlan limits acoustic re-decoding to alternatives anchored in independent
// recognition. It is immutable and never treats lexical similarity as confidence.
type ASRPlan struct {
	source string
	sites  []asrAlternativeSite
	terms  []string
}

type asrAlternativeSite struct {
	start, end int
	candidates []string
	alias      bool
}

// PrepareASR nominates whole canonical terms from a recording's frozen snapshot.
// Approved aliases help retrieval only; Terms never exposes an incorrect form.
// Canonical-only English retrieval compares spelling, not cross-script speech.
func (l *SpellingLexicon) PrepareASR(source string, terms []string, aliases *Vocabulary) ASRPlan {
	plan := ASRPlan{source: source}
	if source == "" || len(source) > maxASRPlanBytes || !utf8.ValidString(source) {
		return plan
	}
	eligible := map[string]bool{}
	for _, term := range terms {
		if normalized, err := NormalizeHotword(term); err == nil && normalized == term {
			eligible[term] = true
		}
	}
	var nominations []asrAlternativeSite
	exceeded := false
	add := func(start, end int, canonical string, alias bool) {
		if source[start:end] != canonical && asrMatchText(canonical) != "" {
			if len(nominations) == maxASRNominations {
				exceeded = true
				return
			}
			nominations = append(nominations, asrAlternativeSite{start, end, []string{canonical}, alias})
		}
	}
	if aliases.Len() > 0 {
		var replacements []Replacement
		for _, replacement := range aliases.entries {
			if eligible[replacement.To] {
				replacements = append(replacements, replacement)
			}
		}
		filtered := NewVocabulary(replacements)
		runes := []rune(source)
		lower := lowerRunesOf(runes)
		offsets := asrRuneOffsets(source)
		for k := range runes {
			if match, ok := filtered.match(filtered.from, runes, lower, k); ok {
				add(offsets[k], offsets[k+len(match.runes)], filtered.entries[match.entry].To, true)
			}
		}
	}
	words := spellingWords.FindAllStringIndex(source, -1)
	for _, term := range terms {
		if exceeded {
			return plan
		}
		if !eligible[term] {
			continue
		}
		if l != nil && asrOnlyHan(term) {
			length := utf8.RuneCountInString(term)
			for _, run := range spellingHan.FindAllStringIndex(source, -1) {
				text := source[run[0]:run[1]]
				offsets := asrRuneOffsets(text)
				for i := 0; i+length < len(offsets); i++ {
					start, end := run[0]+offsets[i], run[0]+offsets[i+length]
					if l.matches(source[start:end], term, false) {
						add(start, end, term, false)
					}
				}
			}
		}
		letters, ok := asrLatinLetters(term)
		if !ok {
			continue
		}
		for i := range words {
			for count := 1; count <= 6 && i+count <= len(words); count++ {
				last := i + count - 1
				if count > 1 && strings.TrimSpace(source[words[last-1][1]:words[last][0]]) != "" {
					break
				}
				start, end := words[i][0], words[last][1]
				if !spellingCompoundBoundary(source, start, end) {
					continue
				}
				spoken, _ := asrLatinLetters(source[start:end])
				if asrSimilarLetters(spoken, letters) {
					add(start, end, term, false)
				}
			}
		}
	}
	if exceeded {
		return plan
	}
	// Exact approved aliases take precedence over approximate overlapping sites.
	// Within each class, longer spans win deterministically.
	sort.SliceStable(nominations, func(i, j int) bool {
		a, b := nominations[i], nominations[j]
		if a.alias != b.alias {
			return a.alias
		}
		if a.end-a.start != b.end-b.start {
			return a.end-a.start > b.end-b.start
		}
		return a.start < b.start
	})
	for _, site := range nominations {
		overlap := false
		for i := range plan.sites {
			existing := &plan.sites[i]
			if site.start == existing.start && site.end == existing.end {
				existing.candidates = append(existing.candidates, site.candidates...)
				overlap = true
				break
			}
			if site.start < existing.end && site.end > existing.start {
				overlap = true
				break
			}
		}
		if !overlap && len(plan.sites) < maxASRAlternativeSites {
			plan.sites = append(plan.sites, site)
		}
	}
	selected := map[string]bool{}
	for _, term := range terms {
		if selected[term] || len(plan.terms) == maxASRAlternativeTerms {
			continue
		}
		for _, site := range plan.sites {
			for _, candidate := range site.candidates {
				if candidate == term && !selected[term] {
					selected[term] = true
					plan.terms = append(plan.terms, term)
				}
			}
			if selected[term] {
				break
			}
		}
	}
	var sites []asrAlternativeSite
	for _, site := range plan.sites {
		candidates := spellingUnique(site.candidates)
		site.candidates = nil
		for _, candidate := range candidates {
			if selected[candidate] {
				site.candidates = append(site.candidates, candidate)
			}
		}
		if len(site.candidates) > 0 {
			sites = append(sites, site)
		}
	}
	plan.sites = sites
	sort.Slice(plan.sites, func(i, j int) bool { return plan.sites[i].start < plan.sites[j].start })
	return plan
}

// Terms returns a bounded canonical-only shortlist, in frozen snapshot order.
func (p ASRPlan) Terms() []string { return append([]string(nil), p.terms...) }

// Resolve accepts only a unique complete parse through the source backbone and
// nominated choices. Punctuation and whitespace may vary during matching; all
// unchanged source bytes survive reconstruction. Ambiguity preserves the source.
func (p ASRPlan) Resolve(proposal string) (string, bool) {
	if len(p.sites) == 0 || len(proposal) > maxASRPlanBytes || !utf8.ValidString(proposal) {
		return p.source, false
	}
	target := asrMatchText(proposal)
	type state struct{ site, offset int }
	type result struct {
		count  int
		choice string
		next   state
	}
	memo := map[state]result{}
	states := 0
	exhausted := false
	var visit func(int, int) result
	visit = func(index, offset int) result {
		key := state{index, offset}
		if cached, ok := memo[key]; ok {
			return cached
		}
		states++
		if states > maxASRResolveStates {
			exhausted = true
			return result{}
		}
		start := 0
		if index > 0 {
			start = p.sites[index-1].end
		}
		end := len(p.source)
		if index < len(p.sites) {
			end = p.sites[index].start
		}
		backbone := p.source[start:end]
		match := asrMatchText(backbone)
		out := result{}
		if !strings.HasPrefix(target[offset:], match) {
			memo[key] = out
			return out
		}
		offset += len(match)
		if index == len(p.sites) {
			if offset == len(target) {
				out.count = 1
			}
			memo[key] = out
			return out
		}
		site := p.sites[index]
		choices := append([]string{p.source[site.start:site.end]}, site.candidates...)
		for _, choice := range choices {
			match := asrMatchText(choice)
			if !strings.HasPrefix(target[offset:], match) {
				continue
			}
			next := visit(index+1, offset+len(match))
			if next.count > 0 {
				out.count = min(2, out.count+next.count)
				out.choice = choice
				out.next = state{index + 1, offset + len(match)}
			}
			if out.count == 2 || exhausted {
				break
			}
		}
		memo[key] = out
		return out
	}
	resolved := visit(0, 0)
	if exhausted || resolved.count != 1 {
		return p.source, false
	}
	var rebuilt strings.Builder
	key := state{}
	start := 0
	for index, site := range p.sites {
		chosen := memo[key]
		rebuilt.WriteString(p.source[start:site.start])
		rebuilt.WriteString(chosen.choice)
		key = chosen.next
		start = p.sites[index].end
	}
	rebuilt.WriteString(p.source[start:])
	return rebuilt.String(), true
}

func asrMatchText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			return -1
		}
		return r
	}, text)
}

func asrRuneOffsets(text string) []int {
	var offsets []int
	for offset := range text {
		offsets = append(offsets, offset)
	}
	return append(offsets, len(text))
}

func asrOnlyHan(text string) bool {
	for _, r := range text {
		if r < '\u3400' || r > '\u9fff' {
			return false
		}
	}
	return text != ""
}

func asrLatinLetters(text string) (string, bool) {
	var letters strings.Builder
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if r < 'a' || r > 'z' {
			return "", false
		}
		letters.WriteRune(r)
	}
	return letters.String(), letters.Len() > 0
}

func asrSimilarLetters(source, canonical string) bool {
	if source == canonical {
		return true
	}
	longer := max(len(source), len(canonical))
	if min(len(source), len(canonical)) < 4 || (longer-min(len(source), len(canonical)))*100 > longer*15 {
		return false
	}
	previous := make([]int, len(canonical)+1)
	for i := range previous {
		previous[i] = i
	}
	current := make([]int, len(previous))
	for i := range source {
		current[0] = i + 1
		for j := range canonical {
			cost := 0
			if source[i] != canonical[j] {
				cost = 1
			}
			current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(canonical)]*100 <= longer*15
}
