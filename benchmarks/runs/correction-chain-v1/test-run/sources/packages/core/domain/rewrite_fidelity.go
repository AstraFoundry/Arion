package domain

import (
	"regexp"
	"strings"
)

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
