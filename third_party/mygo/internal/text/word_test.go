package text

import (
	"slices"
	"testing"
)

func TestStandsAlone(t *testing.T) {
	for _, r := range "我们公园今日直骨々あをはカタナーｶｰ" {
		if !StandsAlone(r) || !IsWordRune(r) {
			t.Errorf("%q: StandsAlone %v, IsWordRune %v", r, StandsAlone(r), IsWordRune(r))
		}
	}
	// Spaced scripts, Hangul among them, digits, punctuation and spaces.
	for _, r := range "aZé_9жω한글ไทย，。、「」 \n　！１" {
		if StandsAlone(r) {
			t.Errorf("%q stands alone", r)
		}
	}
}

func TestWordBoundary(t *testing.T) {
	for _, c := range []struct {
		a, b rune
		want bool
	}{
		{'a', 'b', false}, {'a', '_', false}, {'9', 'x', false}, {'한', '글', false},
		{'a', ' ', true}, {',', ' ', false}, {'a', '中', true}, {'中', 'a', true},
		{'今', '日', true}, {'は', '直', true}, {'カ', 'タ', true}, {'中', '，', true},
	} {
		if got := WordBoundary(c.a, c.b); got != c.want {
			t.Errorf("WordBoundary(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// words walks s with next from 0 (and prev from its end) and returns the
// offsets it stops at.
func words(s string, step func(*Boundaries, int) int, from int) []int {
	var b Boundaries
	b.Reset([]rune(s))
	var stops []int
	for i := from; ; {
		j := step(&b, i)
		if j == i {
			return stops
		}
		stops, i = append(stops, j), j
	}
}

func TestWordMovement(t *testing.T) {
	next := (*Boundaries).NextWord
	prev := (*Boundaries).PrevWord
	for _, c := range []struct {
		s           string
		right, left []int
	}{
		{"hello brave new_world", []int{5, 11, 21}, []int{12, 6, 0}},
		{"한국어 문장", []int{3, 6}, []int{4, 0}},
		{"我们去公园", []int{1, 2, 3, 4, 5}, []int{4, 3, 2, 1, 0}},
		{"今天，好", []int{1, 2, 4}, []int{3, 1, 0}},
		{"今日は直接骨を見た", []int{1, 2, 3, 4, 5, 6, 7, 8, 9}, []int{8, 7, 6, 5, 4, 3, 2, 1, 0}},
		{"Go语言 is fun", []int{2, 3, 4, 7, 11}, []int{8, 5, 3, 2, 0}},
		{"カタカナとabc", []int{1, 2, 3, 4, 5, 8}, []int{5, 4, 3, 2, 1, 0}},
	} {
		n := len([]rune(c.s))
		if got := words(c.s, next, 0); !slices.Equal(got, c.right) {
			t.Errorf("%q: NextWord stops at %v, want %v", c.s, got, c.right)
		}
		if got := words(c.s, prev, n); !slices.Equal(got, c.left) {
			t.Errorf("%q: PrevWord stops at %v, want %v", c.s, got, c.left)
		}
	}
}

func TestWordAtCJK(t *testing.T) {
	s := "say 我们去，ok_1\nx"
	var b Boundaries
	b.Reset([]rune(s))
	for _, c := range []struct{ i, start, end int }{
		{0, 0, 3}, {3, 3, 4}, {4, 4, 5}, {5, 5, 6}, {6, 6, 7}, {7, 7, 8}, {8, 8, 12}, {11, 8, 12}, {12, 12, 12}, {13, 13, 14},
	} {
		if a, z := b.WordAt(c.i); a != c.start || z != c.end {
			t.Errorf("WordAt(%d) = %d, %d, want %d, %d", c.i, a, z, c.start, c.end)
		}
	}
}
