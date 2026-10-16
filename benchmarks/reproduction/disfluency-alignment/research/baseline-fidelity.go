package domain

import (
	"regexp"
	"strings"
)

// Padding is discounted only for the deletion checks, never removed from
// the delivered text. Clause-leading discourse markers and these repeated
// demonstratives are narrow allowances; facts and hedges still count.
var speechPadding = regexp.MustCompile(quotedText.String() + `|(?:^|[，,。.!！?？;；]\s*)(?:那个就是说|就是说|我想说的是|然后的话|然后呢)[，,\s]*|(?i:(?:^|[,.;!?]\s*)(?:so basically(?: what I wanted to say is(?: that)?)?|what I wanted to say is(?: that)?)\b[,\s]*)|这个这个|就是就是|那个那个`)

func withoutSpeechPadding(source string) string {
	original := source
	for {
		next := speechPadding.ReplaceAllStringFunc(source, speechContent)
		if next == source {
			// An all-padding utterance supplies no evidence for new text.
			if len(contentRunes(next)) == 0 {
				return original
			}
			return source
		}
		source = next
	}
}

func speechContent(part string) string {
	switch part {
	case "这个这个":
		return "这个"
	case "就是就是":
		return "就是"
	case "那个那个":
		return "那个"
	}
	if quotedText.MatchString(part) {
		return part
	}
	return ""
}

// These are fidelity proxies, like the negation check. Synonymous or
// repeated modal words can become one hedge; losing uncertainty cannot.
var chineseUncertainty = regexp.MustCompile(`可能|也许|或许|大概|似乎|不确定|不一定|无法确定|不能确定|确定不了|确定不下来|说不准|说不定`)

// Only an explicit correction to certainty releases the immediately
// preceding clause's hedge. Earlier clauses still require uncertainty.
var certaintyRepair = regexp.MustCompile(`(?i)[^，,。.!！?？;；]+[，,]\s*(?:不对|哦不|说错了|no wait|sorry|actually)[，,\s]*((?:(?:我)?(?:确定|肯定)|(?:I am |I'm )?(?:certain|sure)|definitely)[^，,。.!！?？;；]*)`)

func losesUncertainty(source, candidate string) bool {
	return uncertain(certaintyRepair.ReplaceAllString(source, "$1")) && !uncertain(candidate)
}

func uncertain(text string) bool {
	if chineseUncertainty.MatchString(text) {
		return true
	}
	for _, word := range latinWords(strings.ToLower(text)) {
		switch word {
		case "might", "may", "perhaps", "maybe", "probably", "possibly", "uncertain", "unsure":
			return true
		}
	}
	return false
}
