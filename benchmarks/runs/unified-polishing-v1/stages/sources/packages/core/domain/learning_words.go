package domain

import (
	"strings"
	"unicode/utf8"
)

func validWordSpans(words []WordSpan, length int) bool {
	end := 0
	for _, w := range words {
		if w.Start < end || w.Start < 0 || w.End <= w.Start || w.End > length {
			return false
		}
		end = w.End
	}
	return len(words) > 0
}

// alignWordHunks requires complete words on both sides of the edit. Source
// boundaries help retain a name the tokenizer no longer knows after the edit.
// Overlapping changes are merged before projecting unchanged word edges.
func alignWordHunks(a, b []string, hunks []hunk, source, target []WordSpan) []hunk {
	aw, bw := unitWords(a, source), unitWords(b, target)
	type candidate struct {
		h        hunk
		from, to WordSpan
	}
	var candidates []candidate
	for _, raw := range hunks {
		h := trimHunk(a, b, raw)
		from, to := strings.Join(a[h.i0:h.i1], ""), strings.Join(b[h.j0:h.j1], "")
		if contentWeight(from) == 0 || contentWeight(to) == 0 || (numeric(from) && numeric(to)) || exemptChange(from, to) {
			continue
		}
		left, right := WordSpan{h.i0, h.i1}, WordSpan{h.j0, h.j1}
		if hasCJK(from) || hasCJK(to) {
			var ok bool
			left, ok = coveringWords(aw, left)
			if !ok {
				continue
			}
			right, ok = coveringWords(bw, right)
			if !ok {
				continue
			}
			// A lone character has no reusable lexical boundary. Permit only a
			// complete question-particle phrase; never choose an arbitrary neighbor.
			if left.End-left.Start == 1 && right.End-right.Start == 1 {
				if right.End < len(b) && questionParticle(b[right.End]) && hasCJK(b[right.Start]) {
					right.End++
				} else {
					continue
				}
			}
		}
		if n := len(candidates); n > 0 && (left.Start < candidates[n-1].from.End || right.Start < candidates[n-1].to.End) {
			prev := &candidates[n-1]
			prev.h.i1, prev.h.j1 = h.i1, h.j1
			prev.from.End, prev.to.End = max(prev.from.End, left.End), max(prev.to.End, right.End)
		} else {
			candidates = append(candidates, candidate{h, left, right})
		}
	}
	var out []hunk
	for _, c := range candidates {
		h := c.h
		left := max(h.i0-c.from.Start, h.j0-c.to.Start)
		right := max(c.from.End-h.i1, c.to.End-h.j1)
		valid := true
		for range left {
			if h.i0 == 0 || h.j0 == 0 || a[h.i0-1] != b[h.j0-1] {
				valid = false
				break
			}
			h.i0--
			h.j0--
		}
		for valid && right > 0 {
			if h.i1 == len(a) || h.j1 == len(b) || a[h.i1] != b[h.j1] {
				valid = false
				break
			}
			h.i1++
			h.j1++
			right--
		}
		// Expanding from one side must not cut through a larger word on the other.
		if valid && wordEdges(aw, h.i0, h.i1) && wordEdges(bw, h.j0, h.j1) {
			out = append(out, h)
		}
	}
	return out
}

func questionParticle(word string) bool {
	return len([]rune(word)) == 1 && strings.Contains("吗呢吧啊呀嘛", word)
}

// unitWords converts code-point spans to the same token coordinates as the
// diff. Boundaries inside a Latin token are deliberately not accepted.
func unitWords(units []string, spans []WordSpan) []WordSpan {
	offsets := map[int]int{0: 0}
	position := 0
	for i, u := range units {
		position += utf8.RuneCountInString(u)
		offsets[position] = i + 1
	}
	var words []WordSpan
	for _, span := range spans {
		start, ok := offsets[span.Start]
		if !ok {
			continue
		}
		end, ok := offsets[span.End]
		if !ok {
			continue
		}
		words = append(words, WordSpan{start, end})
	}
	return words
}

func coveringWords(words []WordSpan, change WordSpan) (WordSpan, bool) {
	result := WordSpan{-1, -1}
	for _, w := range words {
		if w.Start < change.End && w.End > change.Start {
			if result.Start < 0 {
				result.Start = w.Start
			}
			result.End = w.End
		}
	}
	return result, result.Start >= 0 && result.Start <= change.Start && result.End >= change.End
}

func wordEdges(words []WordSpan, start, end int) bool {
	for _, w := range words {
		if w.Start < start && w.End > start || w.Start < end && w.End > end {
			return false
		}
	}
	return true
}
