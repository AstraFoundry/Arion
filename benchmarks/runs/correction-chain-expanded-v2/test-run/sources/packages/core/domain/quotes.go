package domain

import (
	"regexp"
	"strings"
)

// Quoted wording is content even when it names a hesitation. Unclosed
// quotes protect the rest of an unfinished utterance. A straight single
// quote only opens after a non-letter, so contractions are not quotes.
var quotedText = regexp.MustCompile("“[^”]*(?:”|$)|「[^」]*(?:」|$)|『[^』]*(?:』|$)|‘[^’]*(?:’|$)|\"[^\"]*(?:\"|$)|`[^`]*(?:`|$)|(?:^|[^\\p{L}\\p{N}])'[^']*(?:'|$)")

func transformUnquoted(text string, transform func(string) string) string {
	var b strings.Builder
	start := 0
	for _, span := range quotedText.FindAllStringIndex(text, -1) {
		b.WriteString(transform(text[start:span[0]]))
		b.WriteString(text[span[0]:span[1]])
		start = span[1]
	}
	b.WriteString(transform(text[start:]))
	return b.String()
}
