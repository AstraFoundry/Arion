package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Edit learning turns a user's correction of inserted dictation into
// vocabulary (ADR 014). Like Typeless and Doubao, Arion compares the field
// right after insertion with the field once the user stops editing; unlike
// them, the comparison and the memory stay on this device.
const (
	// MaxEditRunes bounds the inserted text an edit is learned from.
	MaxEditRunes = 2000
	// MaxPairRunes bounds each side of a learned pair: vocabulary is names,
	// terms and short phrases, not sentences.
	MaxPairRunes = 16
	// MaxEditHunks is the most separate changes one edit may contain; more
	// is a rewrite, not a correction.
	MaxEditHunks = 3
	// minCJKPair is the shortest CJK side: a single Han character carries
	// too little context to be replaced safely everywhere.
	minCJKPair = 2
)

// FieldEdit is one tracked insertion. Baseline is the field's text right
// after Arion inserted Inserted; Settled is the same field once the user
// stopped editing. Both are bounded excerpts around the insertion point.
type FieldEdit struct {
	Inserted string
	Baseline string
	Settled  string
}

// WordSpan is a half-open word range in Unicode code points, not bytes or
// UTF-16 code units. Punctuation and whitespace need not have spans.
type WordSpan struct{ Start, End int }

// WordSegmenter supplies local linguistic word boundaries. Platform adapters
// implement this port without introducing OS dependencies into the domain.
type WordSegmenter interface {
	Words(string) ([]WordSpan, error)
}

// EditVerdict is a content-free outcome of one tracked edit.
type EditVerdict string

const (
	EditLearned          EditVerdict = "learned"
	EditUnchanged        EditVerdict = "unchanged"
	EditCleared          EditVerdict = "cleared"
	EditNotFound         EditVerdict = "not_found"
	EditOutside          EditVerdict = "outside"
	EditTooLarge         EditVerdict = "too_large"
	EditRewrite          EditVerdict = "rewrite"
	EditNoPair           EditVerdict = "no_pair"
	EditWordsUnavailable EditVerdict = "words_unavailable"
)

// EditOutcome carries the pairs to learn when the verdict is EditLearned.
type EditOutcome struct {
	Verdict EditVerdict
	Pairs   []Replacement
}

// ExtractCorrections finds the word-level replacements the user made inside
// the inserted text. Typing before or after it, deleting it, clearing the
// field and large rewrites teach nothing.
func ExtractCorrections(e FieldEdit, segmenter WordSegmenter) EditOutcome {
	if strings.TrimSpace(e.Settled) == "" {
		return EditOutcome{Verdict: EditCleared}
	}
	if e.Settled == e.Baseline {
		return EditOutcome{Verdict: EditUnchanged}
	}
	if strings.TrimSpace(e.Inserted) == "" || utf8.RuneCountInString(e.Inserted) > MaxEditRunes {
		return EditOutcome{Verdict: EditNotFound}
	}
	at := strings.LastIndex(e.Baseline, e.Inserted)
	if at < 0 {
		return EditOutcome{Verdict: EditNotFound}
	}
	end := at + len(e.Inserted)
	prefix, suffix := commonAffixes(e.Baseline, e.Settled)
	// Every change must lie within the insertion; the text around it must
	// be intact on both sides.
	if prefix < at || len(e.Baseline)-suffix > end || len(e.Settled) < at+len(e.Baseline)-end {
		return EditOutcome{Verdict: EditOutside}
	}
	edited := e.Settled[at : len(e.Settled)-(len(e.Baseline)-end)]
	if edited == e.Inserted {
		return EditOutcome{Verdict: EditUnchanged}
	}
	if utf8.RuneCountInString(edited) > MaxEditRunes {
		return EditOutcome{Verdict: EditTooLarge}
	}
	a, b := editUnits(e.Inserted), editUnits(edited)
	ops := diffUnits(a, b)
	removed, added := 0, 0
	for _, op := range ops {
		switch op.kind {
		case opDelete:
			removed += contentWeight(a[op.i])
		case opInsert:
			added += contentWeight(b[op.j])
		}
	}
	// Most of the insertion must survive: when more than two thirds of it
	// is gone, or more than twice its size was typed, the user rewrote the
	// message rather than corrected words in it.
	total := 0
	for _, u := range a {
		total += contentWeight(u)
	}
	if strings.TrimSpace(edited) == "" || removed*3 > total*2 || added > 2*total {
		return EditOutcome{Verdict: EditTooLarge}
	}
	hunks := groupHunks(a, ops)
	if len(hunks) > MaxEditHunks {
		return EditOutcome{Verdict: EditRewrite}
	}
	needsWords := false
	for _, h := range hunks {
		h = trimHunk(a, b, h)
		from, to := strings.Join(a[h.i0:h.i1], ""), strings.Join(b[h.j0:h.j1], "")
		if contentWeight(from) > 0 && contentWeight(to) > 0 &&
			!(numeric(from) && numeric(to)) && !exemptChange(from, to) &&
			(hasCJK(from) || hasCJK(to)) {
			needsWords = true
			break
		}
	}
	if needsWords {
		if segmenter == nil {
			return EditOutcome{Verdict: EditWordsUnavailable}
		}
		sourceWords, sourceErr := segmenter.Words(e.Inserted)
		words, err := segmenter.Words(edited)
		if err != nil || sourceErr != nil || !validWordSpans(sourceWords, utf8.RuneCountInString(e.Inserted)) || !validWordSpans(words, utf8.RuneCountInString(edited)) {
			return EditOutcome{Verdict: EditWordsUnavailable}
		}
		hunks = alignWordHunks(a, b, hunks, sourceWords, words)
	}
	var pairs []Replacement
	for _, h := range hunks {
		h = trimHunk(a, b, h)
		if sentenceCase(a, b, h) {
			continue
		}
		if pair, ok := hunkPair(a, b, h); ok && !containsPair(pairs, pair) {
			pairs = append(pairs, pair)
		}
	}
	if len(pairs) == 0 {
		return EditOutcome{Verdict: EditNoPair}
	}
	return EditOutcome{Verdict: EditLearned, Pairs: pairs}
}

// commonAffixes returns the byte lengths of the longest common prefix and
// suffix that do not overlap, both on rune boundaries.
func commonAffixes(x, y string) (int, int) {
	p := 0
	for p < len(x) && p < len(y) {
		rx, nx := utf8.DecodeRuneInString(x[p:])
		ry, _ := utf8.DecodeRuneInString(y[p:])
		if rx != ry {
			break
		}
		p += nx
	}
	s := 0
	for s < len(x)-p && s < len(y)-p {
		rx, nx := utf8.DecodeLastRuneInString(x[:len(x)-s])
		ry, _ := utf8.DecodeLastRuneInString(y[:len(y)-s])
		if rx != ry {
			break
		}
		s += nx
	}
	return p, s
}

// editUnits splits text into Latin words or numbers, single CJK runes,
// whitespace runs and single other runes (punctuation).
func editUnits(s string) []string {
	runes := []rune(s)
	var out []string
	for i := 0; i < len(runes); {
		j := i + 1
		switch r := runes[i]; {
		case isLatinOrDigit(r):
			for j < len(runes) && (isLatinOrDigit(runes[j]) || innerJoiner(runes, j)) {
				j++
			}
		case unicode.IsSpace(r):
			for j < len(runes) && unicode.IsSpace(runes[j]) {
				j++
			}
		}
		out = append(out, string(runes[i:j]))
		i = j
	}
	return out
}

// innerJoiner keeps apostrophes, hyphens and dots inside a Latin token
// ("don't", "gpt-4o", "v1.2").
func innerJoiner(runes []rune, j int) bool {
	return strings.ContainsRune("'-.", runes[j]) && j+1 < len(runes) && isLatinOrDigit(runes[j+1]) && isLatinOrDigit(runes[j-1])
}

// contentWeight counts a unit's letters and digits: whitespace and
// punctuation changes do not make an edit large.
func contentWeight(u string) int {
	n := 0
	for _, r := range u {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n
}

type opKind int

const (
	opEqual opKind = iota
	opDelete
	opInsert
)

type diffOp struct {
	kind opKind
	i, j int
}

// diffUnits is a longest-common-subsequence diff of two unit lists.
func diffUnits(a, b []string) []diffOp {
	n, m := len(a), len(b)
	lcs := make([][]int32, n+1)
	for i := range lcs {
		lcs[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			ops = append(ops, diffOp{opEqual, i, j})
			i, j = i+1, j+1
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			ops = append(ops, diffOp{opInsert, i, j})
			j++
		default:
			ops = append(ops, diffOp{opDelete, i, j})
			i++
		}
	}
	return ops
}

// hunk is a maximal run of changes: a[i0:i1] became b[j0:j1].
type hunk struct{ i0, i1, j0, j1 int }

// groupHunks joins changes separated by a single whitespace unit, so
// "open ai" → "OpenAI" is one change.
func groupHunks(a []string, ops []diffOp) []hunk {
	var out []hunk
	var gap []diffOp
	for _, op := range ops {
		if op.kind == opEqual {
			gap = append(gap, op)
			continue
		}
		joined := len(out) > 0 && len(gap) <= 1
		if joined && len(gap) == 1 {
			joined = strings.TrimSpace(a[gap[0].i]) == ""
		}
		if !joined {
			out = append(out, hunk{i0: op.i, i1: op.i, j0: op.j, j1: op.j})
		}
		h := &out[len(out)-1]
		if op.kind == opDelete {
			h.i1 = op.i + 1
		} else {
			h.j1 = op.j + 1
		}
		h.i1, h.j1 = max(h.i1, op.i), max(h.j1, op.j)
		gap = gap[:0]
	}
	return out
}

// hunkPair turns an aligned change into a vocabulary pair.
func hunkPair(a, b []string, h hunk) (Replacement, bool) {
	from := strings.TrimSpace(strings.Join(a[h.i0:h.i1], ""))
	to := strings.TrimSpace(strings.Join(b[h.j0:h.j1], ""))
	if contentWeight(from) == 0 || contentWeight(to) == 0 || from == to {
		return Replacement{}, false
	}
	// A changed number or time is content the user changed, not spelling.
	if numeric(from) && numeric(to) {
		return Replacement{}, false
	}
	// Lowering letter case is style, never a spelling to enforce; raising
	// it (iphone → iPhone) is.
	if strings.EqualFold(from, to) && upperCount(to) <= upperCount(from) {
		return Replacement{}, false
	}
	// A single Latin letter is too ambiguous to replace everywhere.
	if len([]rune(from)) == 1 && isLatinOrDigit([]rune(from)[0]) {
		return Replacement{}, false
	}
	// Characters whose correct form depends on the sentence (他/她/它,
	// 的/地/得, 在/再) cannot be fixed by a replacement rule; Doubao keeps an
	// exempt-word dictionary for the same reason.
	if exemptChange(from, to) {
		return Replacement{}, false
	}
	if hasCJK(from) || hasCJK(to) {
		if utf8.RuneCountInString(from) < minCJKPair || utf8.RuneCountInString(to) < minCJKPair {
			return Replacement{}, false
		}
	}
	if utf8.RuneCountInString(from) > MaxPairRunes || utf8.RuneCountInString(to) > MaxPairRunes ||
		strings.ContainsAny(from+to, "\n\r") {
		return Replacement{}, false
	}
	return Replacement{From: from, To: to}, true
}

// trimHunk drops punctuation and whitespace at both ends of a change, so
// typing a comma after a corrected word does not become part of it.
func trimHunk(a, b []string, h hunk) hunk {
	for h.i0 < h.i1 && contentWeight(a[h.i0]) == 0 {
		h.i0++
	}
	for h.i1 > h.i0 && contentWeight(a[h.i1-1]) == 0 {
		h.i1--
	}
	for h.j0 < h.j1 && contentWeight(b[h.j0]) == 0 {
		h.j0++
	}
	for h.j1 > h.j0 && contentWeight(b[h.j1-1]) == 0 {
		h.j1--
	}
	return h
}

// sentenceCase reports a change of letter case at the start of the text or
// of a sentence: formatting, not how a word is spelled.
func sentenceCase(a, b []string, h hunk) bool {
	from, to := strings.Join(a[h.i0:h.i1], ""), strings.Join(b[h.j0:h.j1], "")
	if from == to || !strings.EqualFold(from, to) {
		return false
	}
	for i := h.i0 - 1; i >= 0; i-- {
		if strings.ContainsRune(a[i], '\n') {
			return true
		}
		u := strings.TrimSpace(a[i])
		if u == "" {
			continue
		}
		return strings.ContainsAny(u, ".!?。！？")
	}
	return true
}

// exemptGroups are characters that sound alike and are chosen by grammar
// or meaning, not by spelling.
var exemptGroups = []string{"他她它祂", "的地得", "在再", "那哪", "做作", "以已", "即既", "象像"}

// exemptChange reports a change between characters of one exempt group.
func exemptChange(from, to string) bool {
	a, b := []rune(from), []rune(to)
	if len(a) != len(b) {
		return false
	}
	changed := false
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		changed = true
		same := false
		for _, group := range exemptGroups {
			if strings.ContainsRune(group, a[i]) && strings.ContainsRune(group, b[i]) {
				same = true
				break
			}
		}
		if !same {
			return false
		}
	}
	return changed
}

func upperCount(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			n++
		}
	}
	return n
}

// numeric reports text made only of digits, Chinese numerals and number
// punctuation.
func numeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && !strings.ContainsRune("〇零一二三四五六七八九十百千万亿两.,:：", r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return s != ""
}

func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

func containsPair(pairs []Replacement, p Replacement) bool {
	for _, x := range pairs {
		if x == p {
			return true
		}
	}
	return false
}
