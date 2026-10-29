package domain

import (
	"regexp"
	"strings"
	"unicode"
)

// MaxContextTerms bounds the spellings taken from context.
const MaxContextTerms = 12

var (
	latinTerm  = regexp.MustCompile(`[A-Za-z][A-Za-z0-9]*(?:[-_./'][A-Za-z0-9]+)*`)
	quotedTerm = regexp.MustCompile(`[「“"『]([^」”"』\n]{2,12})[」”"』]`)
	// leadingHan is a short Han run starting a line, as chat and mail
	// windows show a sender's name.
	leadingHan = regexp.MustCompile(`^(\p{Han}{2,3})(?:[：:\s]|$)`)
)

// commonCapitalized are capitalized words that are not names or terms.
var commonCapitalized = map[string]bool{
	"the": true, "this": true, "that": true, "these": true, "those": true, "hi": true, "hello": true, "dear": true, "from": true,
	"to": true, "subject": true, "re": true, "fw": true, "fwd": true, "cc": true, "thanks": true, "thank": true, "can": true,
	"could": true, "would": true, "please": true, "original": true, "sent": true, "reply": true, "inbox": true, "draft": true,
	"we": true, "you": true, "i": true, "it": true, "if": true, "and": true, "or": true, "but": true, "so": true, "ok": true,
	"yes": true, "no": true, "best": true, "regards": true, "cheers": true, "seeing": true, "check": true, "what": true,
	"when": true, "where": true, "who": true, "why": true, "how": true, "are": true, "is": true, "do": true, "did": true,
	"mail": true, "today": true, "tomorrow": true, "monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true, "note": true, "file": true, "edit": true, "view": true, "new": true,
}

// ContextTerms extracts spellings from context that a dictation may have
// misheard: names in sender lines and quotes, and Latin names, identifiers
// and codes. Terms already in the source are left out. History and the
// cursor's text come before the screen's.
func ContextTerms(c RewriteContext, source string) []string {
	c = c.Bounded()
	lowerSource := strings.ToLower(source)
	seen := map[string]bool{}
	var out []string
	add := func(term string) {
		term = strings.Trim(term, ".-_/'")
		key := strings.ToLower(term)
		if len([]rune(term)) < 2 || seen[key] || strings.Contains(lowerSource, key) || len(out) >= MaxContextTerms {
			return
		}
		seen[key] = true
		out = append(out, term)
	}
	texts := append(append([]string{}, c.History...), c.Before, c.After, c.Screen)
	for _, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if m := leadingHan.FindStringSubmatch(line); m != nil {
				add(m[1])
			}
			for _, m := range quotedTerm.FindAllStringSubmatch(line, -1) {
				add(strings.TrimSpace(m[1]))
			}
			words := latinTerm.FindAllStringIndex(line, -1)
			for i, w := range words {
				t := line[w[0]:w[1]]
				// A capitalized first word of a sentence is not a name;
				// a short line such as a sender's name is kept.
				if i == 0 && w[0] == 0 && len(words) > 3 && !markedLatin(t) {
					continue
				}
				if notableLatin(t) {
					add(t)
				}
			}
		}
	}
	return out
}

// notableLatin keeps tokens that look like names, identifiers or codes:
// inner capitals, digits or joining punctuation, or a capitalized word
// that is not a common one.
func notableLatin(t string) bool {
	if len(t) < 3 {
		return false
	}
	return markedLatin(t) || unicode.IsUpper([]rune(t)[0]) && !commonCapitalized[strings.ToLower(t)]
}

// markedLatin has inner capitals, digits or joining punctuation, which only
// names, identifiers and codes have.
func markedLatin(t string) bool {
	for i, x := range []rune(t) {
		if unicode.IsDigit(x) || strings.ContainsRune("-_./", x) || (i > 0 && unicode.IsUpper(x)) {
			return true
		}
	}
	return false
}
