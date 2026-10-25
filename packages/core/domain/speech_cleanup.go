package domain

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SpeechCleanupVersion identifies the local span producer, independently
// of rule correction and the organization prompt.
const SpeechCleanupVersion = "speech-spans-v2"

type SpeechSpanKind string

const (
	SpeechHesitation SpeechSpanKind = "hesitation"
	SpeechRepetition SpeechSpanKind = "repetition"
	SpeechRepair     SpeechSpanKind = "repair"
)

// SpeechSpan uses UTF-8 byte offsets in the unchanged source. A span can
// only delete text; it cannot supply a replacement or reorder content.
type SpeechSpan struct {
	Start int            `json:"start"`
	End   int            `json:"end"`
	Kind  SpeechSpanKind `json:"kind"`
}

type SpeechCleanup struct {
	Text    string       `json:"text"`
	Spans   []SpeechSpan `json:"spans,omitempty"`
	Version string       `json:"version"`
}

// ApplySpeechSpans enforces the deletion contract for an ordered annotation
// sequence, trimming outer whitespace. Literal quotes and code spans cannot
// be deleted by annotations.
func ApplySpeechSpans(source string, spans []SpeechSpan) (string, error) {
	quoted := quotedText.FindAllStringIndex(source, -1)
	var b strings.Builder
	cursor := 0
	for _, span := range spans {
		if span.Start < cursor || span.End <= span.Start || span.End > len(source) ||
			!utf8.ValidString(source[cursor:span.Start]) || !utf8.ValidString(source[span.Start:span.End]) {
			return "", errors.New("invalid speech span range")
		}
		if joinsSpeechWords(source, span.Start, span.End) {
			return "", errors.New("speech span joins separate words")
		}
		switch span.Kind {
		case SpeechHesitation, SpeechRepetition, SpeechRepair:
		default:
			return "", errors.New("invalid speech span kind")
		}
		for _, q := range quoted {
			if span.Start < q[1] && q[0] < span.End {
				return "", errors.New("speech span overlaps quoted content")
			}
		}
		b.WriteString(source[cursor:span.Start])
		cursor = span.End
	}
	b.WriteString(source[cursor:])
	return strings.TrimSpace(b.String()), nil
}

var cleanupLeading = regexp.MustCompile(`(?i)(?:那个就是说|就是说|我想说的是|然后的话|然后呢)[，,\s]*|(?:so basically(?: what I wanted to say is(?: that)?)?|what I wanted to say is(?: that)?)\b[,\s]*`)
var cleanupTopic = regexp.MustCompile(`(?:这个|那个)\s*[A-Za-z_][A-Za-z0-9_]*\s*的话`)
var cleanupQuestion = regexp.MustCompile(`那个[，,]\s*(?:我想问|我想确认|我想了解)`)
var cleanupParenthetical = regexp.MustCompile(`(?i),\s*like\s*,`)
var cleanupForeign = regexp.MustCompile(`(?i)(?:(?:euh|eh)[,，]\s*|(?:えーと|えっと|あの)、)`)
var cleanupWords = regexp.MustCompile(`[A-Za-z]+`)
var cleanupRepairCue = regexp.MustCompile(`(?i)[，,、]\s*(?:啊?不对|哦不|说错了|no wait|i mean|sorry|perdón|non pardon|pardon|いや)[，,、\s]+`)
var cleanupQuantity = regexp.MustCompile(`[0-9零〇一二两三四五六七八九十百千万]+(?:点|时|時|个|元|块|天|号|人|张|台|件|分|秒)`)
var cleanupDay = regexp.MustCompile(`(?i)(?:el )?(?:lunes|martes|miércoles|jueves|viernes|sábado|domingo)|(?:lundi|mardi|mercredi|jeudi|vendredi|samedi|dimanche)`)

// CleanSpeech produces high-confidence deletion spans before independent
// organization. Ambiguous discourse and repairs remain model input. This
// is Arion's rule producer, not a reconstructed competitor speech model.
func CleanSpeech(source string) SpeechCleanup {
	out := SpeechCleanup{Text: source, Version: SpeechCleanupVersion}
	// All detectors annotate the unchanged source. Generated content never
	// becomes evidence for any detector.
	removed := make([]bool, len(source))
	mark := func(start, end int, kind SpeechSpanKind) {
		if start < 0 || end <= start || end > len(source) || joinsSpeechWords(source, start, end) {
			return
		}
		for _, q := range quotedText.FindAllStringIndex(source, -1) {
			if start < q[1] && q[0] < end {
				return
			}
		}
		for i := start; i < end; i++ {
			if removed[i] {
				return
			}
		}
		for i := start; i < end; i++ {
			removed[i] = true
		}
		out.Spans = append(out.Spans, SpeechSpan{Start: start, End: end, Kind: kind})
	}
	isLeading := func(start int) bool {
		var before strings.Builder
		for i := 0; i < start; i++ {
			if !removed[i] {
				before.WriteByte(source[i])
			}
		}
		return clauseLeading(before.String(), before.Len())
	}
	// Standalone hesitation is deleted here, not by rule correction.
	for _, span := range hesitationSpans(source) {
		mark(span.Start, span.End, SpeechHesitation)
	}
	for _, pattern := range []*regexp.Regexp{cleanupLeading, cleanupForeign, cleanupParenthetical} {
		for _, loc := range pattern.FindAllStringIndex(source, -1) {
			if pattern == cleanupParenthetical || isLeading(loc[0]) {
				mark(loc[0], loc[1], SpeechHesitation)
			}
		}
	}

	for _, loc := range cleanupQuestion.FindAllStringIndex(source, -1) {
		if !isLeading(loc[0]) {
			continue
		}
		start := loc[0] + strings.Index(source[loc[0]:loc[1]], "那个")
		end := start + len("那个")
		for end < len(source) {
			r, n := utf8.DecodeRuneInString(source[end:])
			if !unicode.IsSpace(r) && !strings.ContainsRune("，,", r) {
				break
			}
			end += n
		}
		mark(start, end, SpeechHesitation)
	}
	for _, loc := range cleanupTopic.FindAllStringIndex(source, -1) {
		if isLeading(loc[0]) {
			mark(loc[1]-len("的话"), loc[1], SpeechHesitation)
		}
	}

	for _, word := range []string{"这个", "那个", "就是", "我"} {
		for cursor := 0; cursor < len(source); {
			i := strings.Index(source[cursor:], word+word)
			if i < 0 {
				break
			}
			i += cursor
			mark(i, i+len(word), SpeechRepetition)
			cursor = i + len(word)
		}
	}
	words := cleanupWords.FindAllStringIndex(source, -1)
	for i := 1; i < len(words); i++ {
		a, b := words[i-1], words[i]
		word := strings.ToLower(source[a[0]:a[1]])
		if word != "i" && word != "the" && word != "a" && word != "we" {
			continue
		}
		if !strings.EqualFold(source[a[0]:a[1]], source[b[0]:b[1]]) {
			continue
		}
		if word == "a" && source[a[0]:a[1]] == "A" && source[b[0]:b[1]] == "A" {
			continue
		}
		if !speechWordBoundary(source, a[0], b[1]) {
			continue
		}

		if strings.Trim(source[a[1]:b[0]], " ,\t") == "" {
			mark(a[0], b[0], SpeechRepetition)
		}
	}
	// Repairs are scoped to one preceding clause and require a repeated
	// frame or a matching typed value. A cue alone never licenses deletion.
	for _, cue := range cleanupRepairCue.FindAllStringIndex(source, -1) {
		start := strings.LastIndexAny(source[:cue[0]], "，,、。.!！?？;；\n")
		if start < 0 {
			start = 0
		} else {
			_, n := utf8.DecodeRuneInString(source[start:])
			start += n
		}
		end := len(source)
		if i := strings.IndexAny(source[cue[1]:], "，,、。.!！?？;；\n"); i >= 0 {
			end = cue[1] + i
		}
		start += len(source[start:cue[0]]) - len(strings.TrimLeftFunc(source[start:cue[0]], unicode.IsSpace))
		left, right := strings.TrimSpace(source[start:cue[0]]), strings.TrimSpace(source[cue[1]:end])
		if left == "" || right == "" {
			continue
		}
		repairStart := -1
		repairEnd := cue[1]
		cueText := strings.ToLower(source[cue[0]:cue[1]])
		englishRepair := strings.Contains(cueText, "no wait") || strings.Contains(cueText, "i mean")
		chineseRepair := strings.Contains(cueText, "不对") || strings.Contains(cueText, "说错") || strings.Contains(cueText, "哦不")
		lw, rw := strings.Fields(strings.ToLower(left)), strings.Fields(strings.ToLower(right))
		if englishRepair && len(lw) >= 3 && len(lw) == len(rw) && lw[0] == rw[0] && lw[1] == rw[1] {
			differences := 0
			for i := range lw {
				if lw[i] != rw[i] {
					differences++
				}
			}
			if differences == 1 {
				repairStart = start
			}
		}
		// A corrected noun phrase can retain the original governing verb.
		if englishRepair && repairStart < 0 && len(rw) >= 3 && (rw[0] == "the" || rw[0] == "a") && len(lw) >= len(rw) && lw[len(lw)-1] == rw[len(rw)-1] {
			if i := lastSpeechArticle(left, rw[0]); i >= 0 {
				repairStart = start + strings.Index(source[start:cue[0]], left) + i
			}
		}
		if repairStart < 0 {
			l, r := []rune(left), []rune(right)
			prefix, suffix := 0, 0
			for prefix < min(len(l), len(r)) && l[prefix] == r[prefix] && IsCJK(l[prefix]) {
				prefix++
			}
			for suffix < min(len(l), len(r))-prefix && l[len(l)-1-suffix] == r[len(r)-1-suffix] && IsCJK(l[len(l)-1-suffix]) {
				suffix++
			}
			if chineseRepair && (prefix >= 2 || prefix >= 1 && suffix >= 2) {
				repairStart = start
			}
		}
		if repairStart < 0 {
			value := right
			copular := chineseRepair && strings.HasPrefix(value, "是")
			if copular {
				value = strings.TrimLeftFunc(value[len("是"):], unicode.IsSpace)
			}
			for _, pattern := range []*regexp.Regexp{cleanupQuantity, cleanupDay} {
				l, r := pattern.FindAllStringIndex(left, -1), pattern.FindStringIndex(value)
				if len(l) == 0 || r == nil || r[0] != 0 {
					continue
				}
				// A copular replacement must be the entire clause, not a
				// different action that merely starts with the same unit.
				if copular && r[1] != len(value) {
					continue
				}
				last := l[len(l)-1]
				if last[1] != len(left) {
					continue
				}
				old, new := left[last[0]:], value[:r[1]]
				if pattern == cleanupQuantity {
					a, _ := utf8.DecodeLastRuneInString(old)
					b, _ := utf8.DecodeLastRuneInString(new)
					if a != b {
						continue
					}
				}
				repairStart = start + strings.Index(source[start:cue[0]], left) + last[0]
				if copular {
					rawRight := source[cue[1]:end]
					leading := len(rawRight) - len(strings.TrimLeftFunc(rawRight, unicode.IsSpace))
					repairEnd += leading + len(right) - len(value)
				}
				break
			}
		}
		if repairStart >= 0 {
			candidate := source[:repairStart] + source[repairEnd:]
			if negations(candidate) >= negations(source) && !losesUncertainty(source, candidate) {
				mark(repairStart, repairEnd, SpeechRepair)
			}
		}
	}
	// Keep annotations in source order; overlapping detections were ignored
	// by mark. Adjacent annotations retain their separate kinds.
	sort.Slice(out.Spans, func(i, j int) bool { return out.Spans[i].Start < out.Spans[j].Start })
	text, err := ApplySpeechSpans(source, out.Spans)
	if err != nil {
		panic("invalid local speech annotations: " + err.Error())
	}
	// Empty cleanup is not evidence for generated content. Preserve the
	// original all-padding behavior and let rewrite validation decide.
	if len(contentRunes(text)) == 0 {
		out.Spans = nil
		return out
	}
	out.Text = text
	return out
}

func clauseLeading(source string, start int) bool {
	before := strings.TrimRightFunc(source[:start], unicode.IsSpace)
	if before == "" {
		return true
	}
	last, _ := utf8.DecodeLastRuneInString(before)
	return strings.ContainsRune("，,、。.!！?？;；", last)
}

// Return offsets in the original UTF-8 string, never in a case-folded copy.
func lastSpeechArticle(text, article string) int {
	words := cleanupWords.FindAllStringIndex(text, -1)
	for i := len(words) - 1; i >= 0; i-- {
		loc := words[i]
		if strings.EqualFold(text[loc[0]:loc[1]], article) && loc[1] < len(text) && text[loc[1]] == ' ' {
			return loc[0]
		}
	}
	return -1
}

func speechWordBoundary(source string, start, end int) bool {
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(source[:start])
		if word(r) {
			return false
		}
	}
	if end < len(source) {
		r, _ := utf8.DecodeRuneInString(source[end:])
		if word(r) {
			return false
		}
	}
	return true
}

func joinsSpeechWords(source string, start, end int) bool {
	if start == 0 || end == len(source) {
		return false
	}
	left, _ := utf8.DecodeLastRuneInString(source[:start])
	right, _ := utf8.DecodeRuneInString(source[end:])
	word := func(r rune) bool { return unicode.IsDigit(r) || unicode.IsLetter(r) && !IsCJK(r) }
	return word(left) && word(right)
}
