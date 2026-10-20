package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SpellingHypothesis is an unverified, source-relative spelling alternative.
// Offsets refer to UTF-8 bytes in the unchanged cleaned source.
type SpellingHypothesis struct {
	ID         string   `json:"id"`
	Start      int      `json:"startUtf8"`
	End        int      `json:"endUtf8"`
	Source     string   `json:"sourceSpan"`
	Candidates []string `json:"candidateSpellings"`
	Status     string   `json:"status"`
	Kind       string   `json:"kind"`
}

// SpellingPreparation keeps reference context separate from editable source.
type SpellingPreparation struct {
	Source     string               `json:"source"`
	Context    map[string]string    `json:"context"`
	Scene      string               `json:"scene"`
	Hypotheses []SpellingHypothesis `json:"hypotheses"`
	Unknown    []string             `json:"unknownChineseCharacters"`
}

// SpellingLexicon contains fixed text-derived pronunciations, never acoustic truth.
type SpellingLexicon struct {
	Readings map[string][]string
	English  map[string][][]string
}

var spellingHan = regexp.MustCompile(`[\x{3400}-\x{9fff}]+`)
var spellingWords = regexp.MustCompile(`[A-Za-z]+`)
var spellingSpoken = regexp.MustCompile(`[A-Za-z]+(?:[ \t]+[A-Za-z]+)+`)
var spellingIdentifier = regexp.MustCompile(`\b[A-Za-z_][A-Za-z_0-9]*\b`)
var spellingNumbers = regexp.MustCompile(`[0-9零〇一二三四五六七八九十百千万亿两点]+(?:元|块|秒|天|小时|分钟|米|户|个|次|年|月|日)`)
var spellingQuotes = regexp.MustCompile("“[^”]*(?:”|$)|「[^」]*(?:」|$)|『[^』]*(?:』|$)|‘[^’]*(?:’|$)|\"[^\"]*(?:\"|$)|`[^`]*(?:`|$)|'[^']*(?:'|$)")

func spellingOverlap(a, b [2]int) bool { return a[0] < b[1] && a[1] > b[0] }
func spellingQuoted(source string) [][2]int {
	out := [][2]int{}
	for _, m := range spellingQuotes.FindAllStringIndex(source, -1) {
		if source[m[0]] == '\'' && m[0] > 0 {
			r, _ := utf8.DecodeLastRuneInString(source[:m[0]])
			if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
				continue
			}
		}
		out = append(out, [2]int{m[0], m[1]})
	}
	return out
}
func spellingBlocked(start, end int, spans [][2]int) bool {
	for _, span := range spans {
		if spellingOverlap([2]int{start, end}, span) {
			return true
		}
	}
	return false
}
func spellingUnique(items []string) []string {
	sort.Strings(items)
	out := []string{}
	for _, s := range items {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}
func spellingParts(s string) []string {
	out := []string{}
	for i := 0; i < len(s); {
		start := i
		switch {
		case s[i] >= 'A' && s[i] <= 'Z':
			i++
			for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
				i++
			}
			if i-start > 1 && i < len(s) && s[i] >= 'a' && s[i] <= 'z' {
				i--
			} else {
				for i < len(s) && s[i] >= 'a' && s[i] <= 'z' {
					i++
				}
			}
		case s[i] >= 'a' && s[i] <= 'z':
			for i < len(s) && s[i] >= 'a' && s[i] <= 'z' {
				i++
			}
		case s[i] >= '0' && s[i] <= '9':
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
		default:
			return nil
		}
		out = append(out, s[start:i])
	}
	return out
}
func (l *SpellingLexicon) phones(words []string) map[string]bool {
	combinations := [][]string{{}}
	for _, w := range words {
		choices := l.English[strings.ToLower(w)]
		if len(choices) == 0 || len(combinations)*len(choices) > 4096 {
			return nil
		}
		next := [][]string{}
		for _, a := range combinations {
			for _, b := range choices {
				next = append(next, append(append([]string{}, a...), b...))
			}
		}
		combinations = next
	}
	out := map[string]bool{}
	for _, p := range combinations {
		out[strings.Join(p, " ")] = true
	}
	return out
}
func (l *SpellingLexicon) matches(a, b string, tones bool) bool {
	ar, br := []rune(a), []rune(b)
	if len(ar) != len(br) {
		return false
	}
	for i, c := range ar {
		found := false
		for _, x := range l.Readings[string(c)] {
			for _, y := range l.Readings[string(br[i])] {
				if !tones {
					x = strings.TrimRight(x, "012345")
					y = strings.TrimRight(y, "012345")
				}
				if x == y {
					found = true
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func spellingHanWindows(s string) []SpellingHypothesis {
	out := []SpellingHypothesis{}
	for _, span := range spellingHan.FindAllStringIndex(s, -1) {
		text := s[span[0]:span[1]]
		positions := []int{}
		for i := range text {
			positions = append(positions, i)
		}
		positions = append(positions, len(text))
		for n := 2; n <= 6; n++ {
			for i := 0; i+n < len(positions); i++ {
				start, end := span[0]+positions[i], span[0]+positions[i+n]
				out = append(out, SpellingHypothesis{Start: start, End: end, Source: s[start:end]})
			}
		}
	}
	return out
}

// Prepare derives maximal nonoverlapping alternatives using the evaluated
// tone-insensitive Chinese and 2..6-word English producer. No decisions or
// replacements occur here; ambiguity is represented explicitly.
func (l *SpellingLexicon) Prepare(source string, context map[string]string, scene string) SpellingPreparation {
	if context == nil {
		context = map[string]string{}
	}
	p := SpellingPreparation{Source: source, Context: context, Scene: scene, Hypotheses: []SpellingHypothesis{}, Unknown: []string{}}
	keys := []string{}
	for k := range context {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := []string{}
	for _, k := range keys {
		values = append(values, context[k])
	}
	screen := strings.Join(values, "\n")
	quoted := spellingQuoted(source)
	blocked := append([][2]int{}, quoted...)
	for _, m := range spellingNumbers.FindAllStringIndex(source, -1) {
		blocked = append(blocked, [2]int{m[0], m[1]})
	}
	hypotheses := []SpellingHypothesis{}
	if scene == string(SceneWork) || scene == string(ScenePrompt) {
		ids := []string{}
		phones := map[string]map[string]bool{}
		for _, id := range spellingIdentifier.FindAllString(screen, -1) {
			if parts := spellingParts(id); len(parts) >= 2 {
				ids = append(ids, id)
				phones[id] = l.phones(parts)
			}
		}
		ids = spellingUnique(ids)
		for _, run := range spellingSpoken.FindAllStringIndex(source, -1) {
			words := spellingWords.FindAllStringIndex(source[run[0]:run[1]], -1)
			for n := 2; n <= min(6, len(words)); n++ {
				for i := 0; i+n <= len(words); i++ {
					start, end := run[0]+words[i][0], run[0]+words[i+n-1][1]
					if spellingBlocked(start, end, quoted) {
						continue
					}
					text := source[start:end]
					letters := strings.Join(strings.Fields(text), "")
					same := []string{}
					matches := []string{}
					pron := l.phones(strings.Fields(text))
					for _, id := range ids {
						if strings.EqualFold(id, letters) {
							same = append(same, id)
						}
						for ph := range pron {
							if phones[id][ph] {
								matches = append(matches, id)
								break
							}
						}
					}
					if len(same) > 0 {
						matches = same
					}
					matches = spellingUnique(matches)
					if len(matches) > 0 {
						hypotheses = append(hypotheses, SpellingHypothesis{Start: start, End: end, Source: text, Candidates: matches, Kind: "english-identifier"})
					}
				}
			}
		}
	}
	targets := []string{}
	for _, h := range spellingHanWindows(screen) {
		targets = append(targets, h.Source)
	}
	targets = spellingUnique(targets)
	// Index by length and the first segmental reading; this is the same
	// exhaustive match set without a source-window × OCR-window scan.
	targetIndex := map[int]map[string][]string{}
	for _, target := range targets {
		runes := []rune(target)
		if targetIndex[len(runes)] == nil {
			targetIndex[len(runes)] = map[string][]string{}
		}
		for _, reading := range l.Readings[string(runes[0])] {
			key := strings.TrimRight(reading, "012345")
			targetIndex[len(runes)][key] = append(targetIndex[len(runes)][key], target)
		}
	}
	for _, h := range spellingHanWindows(source) {
		if spellingBlocked(h.Start, h.End, blocked) || strings.Contains(screen, h.Source) {
			continue
		}
		runes := []rune(h.Source)
		possible := []string{}
		for _, reading := range l.Readings[string(runes[0])] {
			possible = append(possible, targetIndex[len(runes)][strings.TrimRight(reading, "012345")]...)
		}
		matches := []string{}
		for _, target := range spellingUnique(possible) {
			if l.matches(h.Source, target, false) {
				matches = append(matches, target)
			}
		}
		if len(matches) > 0 {
			h.Candidates = matches
			h.Kind = "chinese-character-readings"
			for _, target := range matches {
				if !l.matches(h.Source, target, true) {
					h.Kind = "chinese-segmental-readings"
					break
				}
			}
			hypotheses = append(hypotheses, h)
		}
	}

	sort.SliceStable(hypotheses, func(i, j int) bool {
		a, b := hypotheses[i], hypotheses[j]
		if a.End-a.Start != b.End-b.Start {
			return a.End-a.Start > b.End-b.Start
		}
		return a.Start < b.Start
	})
	for _, h := range hypotheses {
		overlap := false
		for _, c := range p.Hypotheses {
			if spellingOverlap([2]int{h.Start, h.End}, [2]int{c.Start, c.End}) {
				overlap = true
				break
			}
		}
		if !overlap {
			p.Hypotheses = append(p.Hypotheses, h)
		}
	}
	sort.Slice(p.Hypotheses, func(i, j int) bool { return p.Hypotheses[i].Start < p.Hypotheses[j].Start })
	for i := range p.Hypotheses {
		h := &p.Hypotheses[i]
		h.ID = "e" + strconv.Itoa(i)
		h.Status = "unique"
		if len(h.Candidates) != 1 {
			h.Status = "ambiguous"
		}
	}
	for _, r := range source + screen {
		if r >= 0x3400 && r <= 0x9fff && len(l.Readings[string(r)]) == 0 {
			p.Unknown = append(p.Unknown, string(r))
		}
	}
	p.Unknown = spellingUnique(p.Unknown)
	return p
}

// ApplySpellingChoices checks protocol identity and exact source spans, then
// applies only uniquely nominated replacements. It performs no semantic veto.
func ApplySpellingChoices(raw string, p SpellingPreparation) (string, error) {
	var choices map[string]string
	if !utf8.ValidString(raw) || json.Unmarshal([]byte(raw), &choices) != nil || len(choices) != len(p.Hypotheses) {
		return "", errors.New("invalid spelling-choice coverage")
	}
	source := p.Source
	for i := len(p.Hypotheses) - 1; i >= 0; i-- {
		h := p.Hypotheses[i]
		choice, ok := choices[h.ID]
		if !ok || (choice != "keep" && choice != "replace") || (choice == "replace" && len(h.Candidates) != 1) {
			return "", errors.New("invalid spelling-choice value")
		}
		if h.Start < 0 || h.End > len(source) || h.End < h.Start || source[h.Start:h.End] != h.Source {
			return "", errors.New("spelling source identity mismatch")
		}
		if choice == "replace" {
			source = source[:h.Start] + h.Candidates[0] + source[h.End:]
		}
	}
	return source, nil
}
