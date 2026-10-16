package domain

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RewriteVerdict is a content-free validation outcome of a generated
// proposal (009 step 5). Rejection keeps the source text.
type RewriteVerdict string

const (
	RewriteAccepted    RewriteVerdict = "accepted"
	RewriteEmpty       RewriteVerdict = "empty"
	RewriteTruncated   RewriteVerdict = "truncated"
	RewriteMarkup      RewriteVerdict = "markup"
	RewriteLength      RewriteVerdict = "length"
	RewriteNewNumber   RewriteVerdict = "new_number"
	RewriteUnsupported RewriteVerdict = "unsupported_content"
	RewriteNegation    RewriteVerdict = "negation_dropped"
	RewriteFinalValue  RewriteVerdict = "final_value_dropped"
	RewriteDropped     RewriteVerdict = "content_dropped"
	RewriteUncertainty RewriteVerdict = "uncertainty_dropped"
	// RewriteContextCopied adds content from the context that the dictation
	// does not contain.
	RewriteContextCopied RewriteVerdict = "context_copied"
)

// RewriteCheck is a proposal and the evidence it may draw on.
type RewriteCheck struct {
	Source    string
	Candidate string
	Truncated bool
	// Terms are approved vocabulary/memory strings; their content counts as
	// supported.
	Terms []string
	// Context is the rewrite's context; its text supports spelling fixes
	// but not added content.
	Context RewriteContext
}

// markers are chat/prompt structures that must never reach a target,
// including every scene tag of the rewrite prompt.
var markers = func() []string {
	out := append([]string{"<think", "</think", "<|", "|>", "[terms", `"transcript"`}, contextTags...)
	for _, scene := range Scenes {
		out = append(out, "["+string(scene)+"]")
	}
	return out
}()

// listMarkers are numbered-list labels a list scene may add.
var listMarkers = regexp.MustCompile(`(?m)^\s*\d{1,2}[.、)]\s*`)

// zhNegations and enNegations are meaning-bearing negators. Repair cues such
// as 不对/不是/no are deliberately excluded: removing them is the point of a
// self-correction.
var zhNegations = []string{"不要", "不能", "不用", "不会", "不该", "不必", "不可", "不准", "别", "没", "未", "勿"}
var enNegations = []string{"not", "n't", "never", "nothing", "nobody", "cannot"}

// ValidateRewrite applies conservative fidelity proxies. Passing does not
// prove the meaning is preserved; failing proves the proposal is unsafe to
// insert without review.
func ValidateRewrite(c RewriteCheck) RewriteVerdict {
	candidate := strings.TrimSpace(c.Candidate)
	if c.Truncated {
		return RewriteTruncated
	}
	if candidate == "" {
		return RewriteEmpty
	}
	lower := strings.ToLower(candidate)
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return RewriteMarkup
		}
	}
	evidence := c.Source + "\n" + strings.Join(c.Terms, "\n") + "\n" + c.Context.Evidence()
	src, cand := contentRunes(c.Source), contentRunes(candidate)
	if len(cand) > len(src)*5/4+6 || len(cand)*4 < len(contentRunes(withoutSpeechPadding(c.Source))) {
		return RewriteLength
	}
	for _, n := range digitRuns(listMarkers.ReplaceAllString(candidate, "")) {
		if !strings.Contains(evidence, n) {
			return RewriteNewNumber
		}
	}
	if !c.Context.Empty() && contextCopied(c.Source+"\n"+strings.Join(c.Terms, "\n"), candidate) {
		return RewriteContextCopied
	}
	if unsupported(evidence, candidate) {
		return RewriteUnsupported
	}
	if negations(candidate) < negations(c.Source) {
		return RewriteNegation
	}
	if losesUncertainty(c.Source, candidate) {
		return RewriteUncertainty
	}
	// A self-correction keeps the last value spoken; dropping it reverts
	// the speaker's correction.
	if values := quantities(c.Source); len(values) > 0 && !strings.Contains(candidate, values[len(values)-1]) {
		return RewriteFinalValue
	}
	kept := withoutReplaced(c.Source, c.Terms)
	// Speech that repeats the text just before the cursor may be dropped:
	// the field already has it.
	kept = strings.TrimPrefix(kept, repeatedBefore(c.Context.Before, kept))
	kept = withoutSpeechPadding(kept)
	if len(listMarkers.FindAllString(candidate, 2)) == 2 {
		// Numbered lines replace the spoken ordinals.
		kept = ordinals.ReplaceAllString(kept, "")
	}
	covered := coverage(kept, candidate)
	if !c.Context.Empty() {
		// A misheard word replaced by its spelling from the context is not
		// dropped content.
		// The credit is capped: a spelling fix replaces a few characters,
		// while context standing in for the dictation replaces most of it.
		n := len(contentRunes(kept))
		credit := min(contextSupported(c.Source, c.Context.Evidence(), candidate), max(4, n/4))
		covered += float64(credit) / float64(max(1, n))
	}
	if !hasRepairCue(c.Source) && covered*10 < 7 {
		return RewriteDropped
	}
	return RewriteAccepted
}

// quantities finds digit runs and Chinese numerals that denote an amount:
// two or more numeral characters, or one followed by a measure word.
// Ordinals (第一) and fixed words (一下, 一起) are not quantities.
var quantity = regexp.MustCompile(`\d+(?:[.:]\d+)?|[零一二两三四五六七八九十百千万亿]{2,}|[零一二两三四五六七八九十][个张台点块元人天月年号次件位名本岁分秒]`)

func quantities(s string) []string {
	var out []string
	for _, m := range quantity.FindAllStringIndex(s, -1) {
		if m[0] >= len("第") && s[m[0]-len("第"):m[0]] == "第" {
			continue
		}
		out = append(out, strings.TrimRight(s[m[0]:m[1]], "个张台点块元人天月年号次件位名本岁分秒"))
	}
	return out
}

// repairCues are spoken self-correction markers; their presence licenses
// deleting the superseded part.
var repairCues = []string{"不对", "不是", "改成", "改为", "哦不", "等一下", "等等", "说错", "抱歉", "sorry", "actually", "no wait", "i mean", "rather", "scratch that", "make it"}

func hasRepairCue(s string) bool {
	lower := strings.ToLower(s)
	for _, cue := range repairCues {
		if strings.Contains(lower, cue) {
			return true
		}
	}
	return false
}

// ordinals are spoken enumeration markers a numbered list replaces.
var ordinals = regexp.MustCompile(`(?i)第[一二三四五六七八九十][是个点]?|[一二三四五六七八九十]是|首先|其次|\b(first|second|third|fourth|fifth)(ly)?\b`)

// withoutReplaced removes the misheard side of "from → to" terms, which a
// correct rewrite legitimately replaces.
func withoutReplaced(source string, terms []string) string {
	for _, term := range terms {
		if from, _, ok := strings.Cut(term, "→"); ok && strings.TrimSpace(from) != "" {
			source = strings.ReplaceAll(source, strings.TrimSpace(from), "")
		}
	}
	return source
}

// coverage is the share of the source's content characters (as a
// multiset) that remain in the candidate.
func coverage(source, candidate string) float64 {
	src := contentRunes(strings.ToLower(source))
	if len(src) == 0 {
		return 1
	}
	have := map[rune]int{}
	for _, r := range contentRunes(strings.ToLower(candidate)) {
		have[r]++
	}
	kept := 0
	for _, r := range src {
		if have[r] > 0 {
			have[r]--
			kept++
		}
	}
	return float64(kept) / float64(len(src))
}

func contentRunes(s string) []rune {
	var out []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return out
}

func digitRuns(s string) []string {
	var out []string
	start := -1
	for i, r := range s + " " {
		if r >= '0' && r <= '9' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			out = append(out, s[start:i])
			start = -1
		}
	}
	return out
}

// unsupported reports content absent from the evidence beyond a small
// allowance for scene formatting (salutations, list labels, punctuation).
// CJK is compared by character, Latin by lowercase word.
func unsupported(evidence, candidate string) bool {
	chars := map[rune]bool{}
	words := map[string]bool{}
	for _, r := range evidence {
		chars[r] = true
	}
	for _, w := range latinWords(evidence) {
		words[w] = true
	}
	novel, total := 0, 0
	for _, r := range candidate {
		if IsCJK(r) {
			total++
			if !chars[r] {
				novel++
			}
		}
	}
	for _, w := range latinWords(candidate) {
		total++
		if !words[w] {
			novel++
		}
	}
	return novel > 3 && novel*7 > total
}

func latinWords(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r < unicode.MaxASCII && (unicode.IsLetter(r) || r == '\''))
	})
}

func negations(s string) int {
	n := 0
	for _, m := range zhNegations {
		n += strings.Count(s, m)
	}
	lower := " " + strings.ToLower(s) + " "
	for _, m := range enNegations {
		if m == "n't" {
			n += strings.Count(lower, m)
			continue
		}
		for _, w := range latinWords(lower) {
			if w == m {
				n++
			}
		}
	}
	return n
}

// sentenceEnds close a sentence; a live proposal may not add one.
const sentenceEnds = "。．.！!？?"

// AlignLiveEnding keeps a live proposal's terminal punctuation consistent
// with its still-open source: speech continues after a pause, so a period
// or question mark the model appended would be premature.
func AlignLiveEnding(source, candidate string) string {
	source, candidate = strings.TrimSpace(source), strings.TrimSpace(candidate)
	last, _ := utf8.DecodeLastRuneInString(source)
	if strings.ContainsRune(sentenceEnds, last) {
		return candidate
	}
	trimmed := strings.TrimRightFunc(candidate, func(r rune) bool { return strings.ContainsRune(sentenceEnds+"，,、；;", r) })
	if strings.ContainsRune("，,、；;", last) {
		return trimmed + string(last)
	}
	return trimmed
}
