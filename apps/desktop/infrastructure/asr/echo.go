package asr

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// hotwordEcho reports whether a transcript is Qwen3 reading the session
// hotword list back. A single matching term is kept: that is the bias working.
// Two or more contiguous snapshot terms count when they are the whole
// transcript, a leading run, or a trailing run. Punctuation and a leading
// 气声 label are ignored; the label is model text for unvoiced audio, not a
// vocabulary term.
func hotwordEcho(text string, hotwords []string) bool {
	if len(hotwords) < 2 || strings.TrimSpace(text) == "" {
		return false
	}
	body := stripBreathLabel(text)
	if strings.TrimSpace(body) == "" {
		return false
	}
	for i := 0; i < len(hotwords); i++ {
		for j := i + 2; j <= len(hotwords); j++ {
			slice := hotwords[i:j]
			if exactSlice(body, slice) || prefixSlice(body, slice) || suffixSlice(body, slice) {
				return true
			}
		}
	}
	return false
}

func stripBreathLabel(text string) string {
	body := strings.TrimSpace(text)
	for {
		rest := strings.TrimSpace(body)
		const label = "气声"
		if !strings.HasPrefix(rest, label) {
			return body
		}
		after := strings.TrimSpace(rest[len(label):])
		r, size := utf8.DecodeRuneInString(after)
		if r != '：' && r != ':' {
			return body
		}
		body = strings.TrimSpace(after[size:])
	}
}

func exactSlice(text string, terms []string) bool {
	rest, ok := consumeSlice(skipGaps(text), terms)
	return ok && !hasSpeech(skipGaps(rest))
}

func prefixSlice(text string, terms []string) bool {
	rest, ok := consumeSlice(skipGaps(text), terms)
	return ok && hasSpeech(rest)
}

func suffixSlice(text string, terms []string) bool {
	runes := []rune(text)
	for start := 0; start < len(runes); start++ {
		if !hasSpeech(string(runes[:start])) {
			continue
		}
		rest, ok := consumeSlice(string(runes[start:]), terms)
		if ok && !hasSpeech(skipGaps(rest)) {
			return true
		}
	}
	return false
}

func consumeSlice(text string, terms []string) (string, bool) {
	rest := text
	for i, term := range terms {
		if term == "" {
			return "", false
		}
		if i > 0 {
			skipped := skipGaps(rest)
			if skipped != rest {
				rest = skipped
			} else if !cjkTerm(terms[i-1]) || !cjkTerm(term) {
				return "", false
			}
		}
		next, ok := matchTerm(rest, term)
		if !ok {
			return "", false
		}
		if next == "" {
			rest = next
			continue
		}
		r, _ := utf8.DecodeRuneInString(next)
		if isGap(r) {
			rest = next
			continue
		}
		if i+1 < len(terms) && cjkTerm(term) && cjkTerm(terms[i+1]) {
			if _, ok := matchTerm(next, terms[i+1]); ok {
				rest = next
				continue
			}
		}
		return "", false
	}
	return rest, true
}

func matchTerm(text, term string) (string, bool) {
	sr := []rune(text)
	tr := []rune(term)
	si, ti := 0, 0
	for ti < len(tr) {
		if unicode.IsSpace(tr[ti]) {
			if si >= len(sr) || !unicode.IsSpace(sr[si]) {
				return "", false
			}
			for ti < len(tr) && unicode.IsSpace(tr[ti]) {
				ti++
			}
			for si < len(sr) && unicode.IsSpace(sr[si]) {
				si++
			}
			continue
		}
		if si >= len(sr) || !strings.EqualFold(string(sr[si]), string(tr[ti])) {
			return "", false
		}
		si++
		ti++
	}
	return string(sr[si:]), true
}

func cjkTerm(term string) bool {
	r, _ := utf8.DecodeRuneInString(term)
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

func skipGaps(text string) string {
	return strings.TrimLeftFunc(text, isGap)
}

func isGap(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func hasSpeech(text string) bool {
	for _, r := range text {
		if !isGap(r) {
			return true
		}
	}
	return false
}
