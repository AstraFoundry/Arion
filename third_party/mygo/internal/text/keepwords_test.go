package text

import (
	"slices"
	"strings"
	"testing"

	"github.com/go-text/typesetting/segmenter"
)

// keptBreaks returns where the lines of text may break in language lang
// with spans, but at its end.
func keptBreaks(text []rune, lang string, spans []Span) []int {
	var seg segmenter.Segmenter
	breaks := lineBreaks(&seg, nil, text)
	keepWordsWhole(breaks, text, lang, spans)
	var at []int
	for i, ok := range breaks[:len(text)] {
		if ok {
			at = append(at, i)
		}
	}
	return at
}

func TestKeepWordsWholeBreaksKoreanAtSpaces(t *testing.T) {
	text := []rune("안녕하세요 반갑습니다, Stacia2 앱")
	var seg segmenter.Segmenter
	breaks := lineBreaks(&seg, nil, text)
	if !breaks[2] {
		t.Fatal("UAX #14 should allow a break between Hangul syllables")
	}
	if !keepWordsWhole(breaks, text, "ko", nil) {
		t.Fatal("no break taken out")
	}
	var at []int
	for i, ok := range breaks {
		if ok {
			at = append(at, i)
		}
	}
	if want := []int{6, 13, 21, len(text)}; !slices.Equal(at, want) {
		t.Fatalf("breaks at %v, want %v", at, want)
	}
	for _, lang := range []string{"ko", "ko-KR", "KO", "ko_KR", "ko-Kore-KR"} {
		if !keepsWords(lang) {
			t.Errorf("keepsWords(%q) = false", lang)
		}
	}
	for _, lang := range []string{"", "ja", "zh-Hans", "kok", "en-KO"} {
		if keepsWords(lang) {
			t.Errorf("keepsWords(%q) = true", lang)
		}
	}
}

func TestKeepWordsWhole(t *testing.T) {
	for _, c := range []struct {
		text, lang string
		spans      []Span
		want       []int
	}{
		// A break after punctuation stays, with or without a space.
		{"가나,다라 마바", "ko", nil, []int{3, 6}},
		{"(가나)다라", "ko", nil, []int{4}},
		// Han characters and kana are letters of a word too.
		{"大韓民國 만세", "ko", nil, []int{5}},
		// Jamo, compatibility jamo, and a syllable with a combining mark.
		{"ㅋㅋㅋ ᄀᄀ", "ko", nil, []int{4}},
		{"가́나 다", "ko", nil, []int{4}},
		// An emoji is no letter: lines break around it.
		{"가😀나", "ko", nil, []int{1, 2}},
		// After a hyphen, as in English.
		{"가-나", "ko", nil, []int{2}},
		// Thai, which breaks by a dictionary, is left to it.
		{"가สวัสดี", "ko", nil, []int{1}},
		// Other languages are left as they are.
		{"가나 다", "ja", nil, []int{1, 3}},
		{"가나 다", "", nil, []int{1, 3}},
		// The language of spans.
		{"가나 다라", "en", []Span{{End: 3, Lang: "ko"}}, []int{3, 4}},
		{"가나 다라", "ko", []Span{{End: 3}, {End: 5, Lang: "ja"}}, []int{3, 4}},
		{"가나다", "en", []Span{{End: 1}, {End: 3, Lang: "ko"}}, []int{1}},
	} {
		if got := keptBreaks([]rune(c.text), c.lang, c.spans); !slices.Equal(got, c.want) {
			t.Errorf("%q in %q with %v breaks at %v, want %v", c.text, c.lang, c.spans, got, c.want)
		}
	}
}

// keepAllText is Korean, of words up to five syllables long.
const keepAllText = "안녕하세요 반갑습니다 오늘은 날씨가 정말 좋네요 우리 함께 산책하러 갈까요"

// Korean text breaks its lines at spaces, with every engine, and other
// languages' text between syllables (#1403). A word longer than a line
// still breaks, unless words are kept whole anyway.
func TestKeepAllLayout(t *testing.T) {
	s := Shared()
	style := Style{Size: 16, Lang: "ko"}
	width := s.Layout(Params{Text: "가나다라마", Style: style}).Width + 0.5
	text := []rune(keepAllText)
	breaksAt := func(l *Layout) (atSpace, inWord int) {
		for i, line := range l.Lines[:len(l.Lines)-1] {
			if line.End != l.Lines[i+1].Start {
				t.Errorf("line %d ends at %d, the next starts at %d", i, line.End, l.Lines[i+1].Start)
			}
			if text[line.End-1] == ' ' {
				atSpace++
			} else {
				inWord++
			}
		}
		return atSpace, inWord
	}
	ko := s.Layout(Params{Text: keepAllText, Style: style, Width: width})
	if atSpace, inWord := breaksAt(ko); inWord > 0 || atSpace < 4 {
		t.Errorf("in Korean, %d lines break at spaces, %d within words", atSpace, inWord)
		for _, line := range ko.Lines {
			t.Logf("%q", string(text[line.Start:line.End]))
		}
	}
	for _, line := range ko.Lines {
		if line.Width > width+0.5 {
			t.Errorf("a line of Korean is %v wide, wider than %v", line.Width, width)
		}
	}
	en := s.Layout(Params{Text: keepAllText, Style: Style{Size: 16, Lang: "en"}, Width: width})
	if _, inWord := breaksAt(en); inWord == 0 {
		t.Error("in English, no line breaks within a Korean word")
	}

	// A word longer than the line breaks within itself, not the line
	// before it.
	long := "가나 " + strings.Repeat("다라마바사", 3) + " 아자"
	text = []rune(long)
	l := s.Layout(Params{Text: long, Style: style, Width: width})
	if len(l.Lines) < 4 || l.Lines[0].End != 3 {
		t.Errorf("a long word took lines %v", lineRanges(l))
	}
	for _, line := range l.Lines {
		if line.Width > width+0.5 {
			t.Errorf("a line of the long word is %v wide, wider than %v", line.Width, width)
		}
	}
	// Kept whole anyway, it overflows its own line.
	l = s.Layout(Params{Text: long, Style: style, Width: width, NoBreakWords: true})
	if got, want := lineRanges(l), [][2]int{{0, 3}, {3, 19}, {19, 21}}; !slices.Equal(got, want) {
		t.Errorf("with words kept whole, lines %v, want %v", got, want)
	}
	// Spaces kept, the lines of Korean break where they did.
	l = s.Layout(Params{Text: keepAllText, Style: style, Width: width, KeepSpaces: true})
	text = []rune(keepAllText)
	if _, inWord := breaksAt(l); inWord > 0 {
		t.Errorf("with spaces kept, %d lines break within words", inWord)
	}
}

func lineRanges(l *Layout) [][2]int {
	var out [][2]int
	for _, line := range l.Lines {
		out = append(out, [2]int{line.Start, line.End})
	}
	return out
}
