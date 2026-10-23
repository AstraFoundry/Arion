package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// RulesVersion identifies the conservative rule set used for a correction.
// arion-rules-v4 leaves standalone hesitation to speech-spans-v2.
const RulesVersion = "arion-rules-v4"

// LatinCase declares how a recognizer emits Latin letters.
type LatinCase string

const (
	LatinCaseNatural LatinCase = "natural"
	LatinCaseUpper   LatinCase = "upper"
)

// Replacement is a user-approved vocabulary correction. It applies only to
// whole tokens (Latin word boundaries, or exact CJK sequences).
type Replacement struct {
	From string
	To   string
}

// CorrectionPolicy selects deterministic rules. Every rule only rewrites
// spacing, case of recognizer-uppercased Latin and approved replacements.
// Standalone hesitation stays for speech cleanup when polishing runs.
// A rule never adds words (SPH-06).
type CorrectionPolicy struct {
	LatinCase    LatinCase
	Replacements []Replacement
	// Vocabulary is Replacements prebuilt once by the caller; when nil it is
	// built from Replacements for this call.
	Vocabulary *Vocabulary
}

// CorrectionOutcome keeps the raw text separate from the corrected text.
type CorrectionOutcome struct {
	Raw     string
	Text    string
	Applied []string
	Version string
	// Recalled lists the vocabulary entries that fired, in text order.
	Recalled []Replacement
}

// Correct applies the conservative rule pipeline.
func Correct(raw string, policy CorrectionPolicy) CorrectionOutcome {
	out := CorrectionOutcome{Raw: raw, Version: RulesVersion}
	text := raw
	apply := func(id string, fn func(string) string) {
		if next := fn(text); next != text {
			text = next
			out.Applied = append(out.Applied, id)
		}
	}
	apply("whitespace", collapseWhitespace)
	if policy.LatinCase == LatinCaseUpper {
		apply("latin-case", naturalizeUppercaseLatin)
	}
	apply("acronyms", joinSpelledAcronyms)
	vocabulary := policy.Vocabulary
	if vocabulary == nil && len(policy.Replacements) > 0 {
		vocabulary = NewVocabulary(policy.Replacements)
	}
	if vocabulary.Len() > 0 {
		apply("vocabulary", func(s string) string {
			next, fired := vocabulary.Apply(s)
			out.Recalled = fired
			return next
		})
	}
	apply("cjk-spacing", joinCJKSpacing)
	out.Text = text
	return out
}

func collapseWhitespace(s string) string { return strings.Join(strings.Fields(s), " ") }

// IsCJK reports Han, kana, Hangul and CJK punctuation runes.
func IsCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) || isCJKPunct(r)
}

func isCJKPunct(r rune) bool {
	return (r >= 0x3000 && r <= 0x303F) || (r >= 0xFF00 && r <= 0xFFEF)
}

func isClosingPunct(r rune) bool {
	return strings.ContainsRune(",.!?;:%)]}", r) || isCJKPunct(r) && !strings.ContainsRune("「『（【《〈", r)
}

// joinCJKSpacing removes spaces between CJK runes and before punctuation,
// which recognizers emit as token separators.
func joinCJKSpacing(s string) string {
	runes := []rune(collapseWhitespace(s))
	var b strings.Builder
	for i, r := range runes {
		if r == ' ' && i > 0 && i+1 < len(runes) {
			prev, next := runes[i-1], runes[i+1]
			if (IsCJK(prev) && IsCJK(next)) || isClosingPunct(next) || isCJKPunct(prev) {
				continue
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isLatinOrDigit(r rune) bool {
	return r < utf8.RuneSelf && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

// naturalizeUppercaseLatin lowercases all-caps Latin words produced by an
// uppercase recognizer, capitalizes sentence starts and the pronoun "I".
// Mixed-case words are left untouched; they are not recognizer artifacts.
func naturalizeUppercaseLatin(s string) string {
	runes := []rune(s)
	sentenceStart := true
	for i := 0; i < len(runes); {
		r := runes[i]
		if !(r < utf8.RuneSelf && unicode.IsLetter(r)) {
			if strings.ContainsRune(".!?。！？", r) {
				sentenceStart = true
			} else if !unicode.IsSpace(r) && r != '"' && r != '\'' {
				sentenceStart = false
			}
			i++
			continue
		}
		j := i
		allUpper := true
		for j < len(runes) && (runes[j] < utf8.RuneSelf && (unicode.IsLetter(runes[j]) || runes[j] == '\'')) {
			if unicode.IsLower(runes[j]) {
				allUpper = false
			}
			j++
		}
		if allUpper {
			word := strings.ToLower(string(runes[i:j]))
			if word == "i" || strings.HasPrefix(word, "i'") || sentenceStart {
				word = strings.ToUpper(word[:1]) + word[1:]
			}
			copy(runes[i:j], []rune(word))
		}
		sentenceStart = false
		i = j
	}
	return string(runes)
}

// joinSpelledAcronyms joins runs of two or more single uppercase Latin
// letters that a recognizer spells out ("P R" → "PR", "A P I" → "API").
// A lone capital letter between words is left alone.
func joinSpelledAcronyms(s string) string {
	return transformUnquoted(s, joinUnquotedAcronyms)
}

func joinUnquotedAcronyms(s string) string {
	fields := strings.Split(s, " ")
	single := func(f string) bool { return len(f) == 1 && f[0] >= 'A' && f[0] <= 'Z' }
	var out []string
	for i := 0; i < len(fields); {
		j := i
		for j < len(fields) && single(fields[j]) {
			j++
		}
		// Repeated first-person I is ambiguous speech, not an acronym.
		// Leave it for the independent repetition stage.
		firstPerson := j > i
		for _, word := range fields[i:j] {
			if word != "I" {
				firstPerson = false
				break
			}
		}
		if j-i >= 2 && !firstPerson {
			out = append(out, strings.Join(fields[i:j], ""))
			i = j
			continue
		}
		out = append(out, fields[i])
		i++
	}
	return strings.Join(out, " ")
}

var fillerWords = map[string]bool{"嗯": true, "呃": true, "额": true, "um": true, "uh": true, "erm": true, "uhm": true}

// hesitationSpans marks standalone hesitation for speech cleanup.
// Rule correction leaves these tokens in place. CJK hesitations may attach
// to adjacent speech; the ambiguous noun suffix 额 needs both boundaries.
func hesitationSpans(s string) []SpeechSpan {
	runes := []rune(s)
	offsets := make([]int, 0, len(runes)+1)
	for i := range s {
		offsets = append(offsets, i)
	}
	offsets = append(offsets, len(s))
	var out []SpeechSpan
	for i := 0; i < len(runes); {
		start, end := i, i+1
		if r := runes[i]; r < utf8.RuneSelf && unicode.IsLetter(r) {
			for end < len(runes) && runes[end] < utf8.RuneSelf && unicode.IsLetter(runes[end]) {
				end++
			}
			if !fillerWords[strings.ToLower(string(runes[start:end]))] {
				i = end
				continue
			}
		} else if !fillerWords[string(r)] || !standaloneCJK(runes, i) {
			i++
			continue
		}
		end = skipFillerTrail(runes, end)
		out = append(out, SpeechSpan{Start: offsets[start], End: offsets[end], Kind: SpeechHesitation})
		i = end
	}
	return out
}

func standaloneCJK(runes []rune, i int) bool {
	before := i == 0 || !unicode.Is(unicode.Han, runes[i-1])
	after := i+1 >= len(runes) || !unicode.Is(unicode.Han, runes[i+1])
	if runes[i] == '额' {
		// Unlike 嗯/呃, 额 commonly ends words such as 余额 and 名额.
		return before && after
	}
	return before || after
}

// skipFillerTrail also drops the comma that recognizers attach to a filler.
func skipFillerTrail(runes []rune, j int) int {
	if j < len(runes) && strings.ContainsRune(",，、", runes[j]) {
		j++
	}
	return j
}

// JoinText concatenates text segments, inserting a space only between
// Latin letters or digits, never next to CJK characters.
func JoinText(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if b.Len() > 0 {
			last, _ := utf8.DecodeLastRuneInString(b.String())
			first, _ := utf8.DecodeRuneInString(part)
			if !IsCJK(last) && !IsCJK(first) && !unicode.IsSpace(last) {
				b.WriteByte(' ')
			}
		}
		b.WriteString(part)
	}
	return b.String()
}
