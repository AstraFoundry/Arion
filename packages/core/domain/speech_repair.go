package domain

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// markSpokenRepairs deletes a clear mid-utterance revision and keeps the
// final span. It does not add words. Content exclusions ("四秒，不是四十秒"),
// identity statements ("我不是张三，是李四"), conjunctions, and quoted text stay.
// Ambiguous gaps remain model input.
func markSpokenRepairs(source string, mark func(int, int, SpeechSpanKind)) {
	markRejectedValueRuns(source, mark)
	markNotAIsB(source, mark)
	markEnglishNotRepair(source, mark)
}

type repairValue struct {
	start, end int
	kind       string
	text       string
}

var (
	zhMonthValue    = regexp.MustCompile(`(?:[0-9]{1,2}|[零〇一二两三四五六七八九十百千]+)个?月`)
	zhQuantityValue = regexp.MustCompile(`(?:[0-9]+|[零〇一二两三四五六七八九十百千万亿]+)(?:点[0-9零〇一二三四五六七八九]+)?(?:个|元|块|天|号|人|张|台|件|分|秒)`)
	zhNumberValue   = regexp.MustCompile(`[0-9]+|[零〇一二两三四五六七八九十百千万亿]+`)
	enMonthValue    = regexp.MustCompile(`\b(?:January|February|March|April|May|June|July|August|September|October|November|December)\b`)
	enWeekdayValue  = regexp.MustCompile(`\b(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\b`)
	enNumberValue   = regexp.MustCompile(`(?i)\b(?:zero|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety|hundred|thousand|[0-9]+)\b(?:\s+(?:people|seconds|minutes|hours|days|dollars|percent))?`)
	repairGapSplit  = regexp.MustCompile(`[，,、。.!！?？;；\n]+|…+|\.{3,}`)
	notAIsBPattern  = regexp.MustCompile(`不是([^，,、。.!！?？;；\n…]{1,48}?)[，,、\s]*是([^，,、。.!！?？;；\n…]{1,48})`)
	enNotPattern    = regexp.MustCompile(`(?i)\bnot\s+([A-Za-z0-9]+(?:\s+[A-Za-z0-9]+){0,3})\s*[，,]\s*(?:it(?:'s| is)\s+)?([A-Za-z0-9]+(?:\s+[A-Za-z0-9]+){0,3})`)
)

func findRepairValues(source string) []repairValue {
	var found []repairValue
	add := func(pattern *regexp.Regexp, kind string, kindFrom func(string) string, skip func(int) bool) {
		for _, loc := range pattern.FindAllStringIndex(source, -1) {
			if skip != nil && skip(loc[0]) {
				continue
			}
			text := source[loc[0]:loc[1]]
			k := kind
			if kindFrom != nil {
				k = kindFrom(text)
			}
			found = append(found, repairValue{start: loc[0], end: loc[1], kind: k, text: text})
		}
	}
	add(zhMonthValue, "zh-month", nil, nil)
	add(zhQuantityValue, "", func(text string) string {
		r, _ := utf8.DecodeLastRuneInString(text)
		return "zhq:" + string(r)
	}, nil)
	add(zhNumberValue, "zh-num", nil, func(start int) bool {
		return precededBy(source, start, "什么", "没有", "哪有", "不是")
	})
	add(enMonthValue, "en-month", nil, nil)
	add(enWeekdayValue, "en-weekday", nil, nil)
	add(enNumberValue, "", func(text string) string {
		fields := strings.Fields(text)
		if len(fields) > 1 {
			return "en-num:" + strings.ToLower(fields[len(fields)-1])
		}
		return "en-num"
	}, nil)
	return keepLongestRepairValues(found)
}

func keepLongestRepairValues(found []repairValue) []repairValue {
	// Earlier start first; at the same start keep the longer span so a
	// month wins over the numeral inside it.
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].start != found[j].start {
			return found[i].start < found[j].start
		}
		return found[i].end-found[i].start > found[j].end-found[j].start
	})
	var kept []repairValue
	for _, v := range found {
		overlap := false
		for _, prior := range kept {
			if v.start < prior.end && prior.start < v.end {
				overlap = true
				break
			}
		}
		if !overlap {
			kept = append(kept, v)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].start < kept[j].start })
	return kept
}

func precededBy(source string, start int, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if start >= len(prefix) && source[start-len(prefix):start] == prefix {
			return true
		}
	}
	return false
}

func markRejectedValueRuns(source string, mark func(int, int, SpeechSpanKind)) {
	values := findRepairValues(source)
	for i := 0; i < len(values); {
		last := i
		saw := false
		for next := i + 1; next < len(values) && values[next].kind == values[i].kind; next++ {
			// An abandoned attempt ends the clause. A copular replacement
			// ("是四秒") must be that whole clause, not the start of a
			// different predicate ("是四秒后重试").
			if !repairClauseFinal(source, values[next-1].end) {
				break
			}
			gap := source[values[next-1].end:values[next].start]
			if gapHasCopula(gap) && !repairClauseFinal(source, values[next].end) {
				break
			}
			allow, rejection := gapAllowsRepair(gap, values[i:next+1])
			if !allow {
				break
			}
			if rejection {
				saw = true
			}
			last = next
		}
		if saw && last > i {
			mark(values[i].start, values[last].start, SpeechRepair)
			i = last + 1
			continue
		}
		i++
	}
}

func repairClauseFinal(source string, end int) bool {
	if end >= len(source) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(source[end:])
	return unicode.IsSpace(r) || strings.ContainsRune("，,、。.!！?？;；\n…", r)
}

func gapHasCopula(gap string) bool {
	for _, part := range repairGapSplit.Split(gap, -1) {
		if isRepairCopula(strings.TrimSpace(part)) {
			return true
		}
	}
	return false
}

func gapAllowsRepair(gap string, values []repairValue) (bool, bool) {
	if repairGluedExclusion(gap) {
		return false, false
	}
	rejection := false
	for _, part := range repairGapSplit.Split(gap, -1) {
		part = strings.TrimSpace(part)
		if part == "" || isRepairCopula(part) {
			continue
		}
		if isRepairRejection(part, values) {
			rejection = true
			continue
		}
		return false, false
	}
	return true, rejection
}

// repairGluedExclusion reports "不是四十秒" / "not fifty", where the next
// value is the excluded alternative rather than the correction.
func repairGluedExclusion(gap string) bool {
	trimmed := strings.TrimRightFunc(gap, unicode.IsSpace)
	if strings.HasSuffix(trimmed, "不是") {
		return true
	}
	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, "not") {
		if len(lower) == len("not") {
			return true
		}
		prev, _ := utf8.DecodeLastRuneInString(lower[:len(lower)-len("not")])
		return !unicode.IsLetter(prev) && !unicode.IsDigit(prev)
	}
	return false
}

func isRepairCopula(part string) bool {
	switch strings.ToLower(part) {
	case "是", "it's", "it is":
		return true
	default:
		return false
	}
}

func isRepairRejection(part string, values []repairValue) bool {
	switch strings.ToLower(part) {
	case "不对", "啊不对", "哦不", "说错了", "不是", "no", "no wait", "wait", "i mean", "sorry", "actually", "scratch that", "pardon", "perdón", "non pardon", "いや":
		return true
	}
	for _, prefix := range []string{"什么", "没有", "哪有", "不是", "what ", "no ", "not ", "there's no ", "there is no "} {
		if strings.HasPrefix(strings.ToLower(part), strings.ToLower(prefix)) {
			rest := strings.TrimSpace(part[len(prefix):])
			if rest != "" && refersToRepairValue(rest, values) {
				return true
			}
		}
	}
	return false
}

func refersToRepairValue(rest string, values []repairValue) bool {
	for _, v := range values {
		if strings.EqualFold(v.text, rest) || strings.EqualFold(repairCore(v.text), rest) {
			return true
		}
	}
	return false
}

func repairCore(text string) string {
	for _, suffix := range []string{"个月", "月", "人", "秒", "分", "个", "天", "元", "块", "号"} {
		if strings.HasSuffix(text, suffix) && len([]rune(text)) > len([]rune(suffix)) {
			return strings.TrimSuffix(text, suffix)
		}
	}
	fields := strings.Fields(text)
	if len(fields) > 1 {
		return fields[0]
	}
	return text
}

func markNotAIsB(source string, mark func(int, int, SpeechSpanKind)) {
	for _, loc := range notAIsBPattern.FindAllStringSubmatchIndex(source, -1) {
		a := strings.TrimSpace(source[loc[2]:loc[3]])
		b := strings.TrimSpace(source[loc[4]:loc[5]])
		if a == "" || b == "" || replacementBlocked(b) {
			continue
		}
		bStart := loc[4]
		bStart += len(source[loc[4]:loc[5]]) - len(strings.TrimLeftFunc(source[loc[4]:loc[5]], unicode.IsSpace))
		start, ok := repeatedAttemptStart(source[:loc[0]], a)
		if ok {
			mark(start, bStart, SpeechRepair)
			continue
		}
		if !sameWholeRepairKind(a, b) {
			continue
		}
		start = extendSameKind(source, loc[0], repairKindOf(b))
		mark(start, bStart, SpeechRepair)
	}
}

func markEnglishNotRepair(source string, mark func(int, int, SpeechSpanKind)) {
	for _, loc := range enNotPattern.FindAllStringSubmatchIndex(source, -1) {
		a := strings.TrimSpace(source[loc[2]:loc[3]])
		b := strings.TrimSpace(source[loc[4]:loc[5]])
		if a == "" || b == "" || replacementBlocked(b) {
			continue
		}
		start, ok := repeatedAttemptStart(source[:loc[0]], a)
		if !ok {
			continue
		}
		bStart := loc[4]
		bStart += len(source[loc[4]:loc[5]]) - len(strings.TrimLeftFunc(source[loc[4]:loc[5]], unicode.IsSpace))
		mark(start, bStart, SpeechRepair)
	}
}

func repeatedAttemptStart(prefix, attempt string) (int, bool) {
	trimmed := strings.TrimRightFunc(prefix, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("，,、。.!！?？;；…", r)
	})
	if attempt == "" || !strings.HasSuffix(trimmed, attempt) {
		return 0, false
	}
	// A Latin attempt must end on a word boundary so "ice" does not match
	// the tail of "notice".
	start := len(trimmed) - len(attempt)
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(trimmed[:start])
		first, _ := utf8.DecodeRuneInString(attempt)
		if unicode.IsLetter(r) && unicode.IsLetter(first) && !isHanRune(r) && !isHanRune(first) {
			return 0, false
		}
	}
	return start, true
}

func isHanRune(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

func replacementBlocked(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range []string{"不", "没", "别", "未", "也许", "可能", "大概", "或许", "maybe", "perhaps", "might"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func sameWholeRepairKind(a, b string) bool {
	va, vb := findRepairValues(a), findRepairValues(b)
	if len(va) != 1 || len(vb) != 1 {
		return false
	}
	return va[0].start == 0 && va[0].end == len(a) && vb[0].start == 0 && vb[0].end == len(b) && va[0].kind == vb[0].kind
}

func repairKindOf(text string) string {
	values := findRepairValues(text)
	if len(values) == 1 {
		return values[0].kind
	}
	return ""
}

func extendSameKind(source string, from int, kind string) int {
	if kind == "" {
		return from
	}
	head := strings.TrimRightFunc(source[:from], func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("，,、。.!！?？;；…", r)
	})
	values := findRepairValues(head)
	if len(values) == 0 {
		return from
	}
	last := values[len(values)-1]
	if last.kind == kind && last.end == len(head) {
		return last.start
	}
	return from
}
