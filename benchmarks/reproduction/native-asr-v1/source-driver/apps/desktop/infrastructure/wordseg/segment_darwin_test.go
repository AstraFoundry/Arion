//go:build darwin

package wordseg

import (
	"testing"

	"arion.local/core/domain"
)

func TestNativeWordBoundaries(t *testing.T) {
	for _, text := range []string{"公示需要修改", "公式需要修改", "这是全棉的衣服", "codex能报吗", "把Arion的配置发给晓王", "🙂收到请回复", "成功处置深航机上纵火事件"} {
		spans, err := (Segmenter{}).Words(text)
		if err != nil {
			t.Fatal(err)
		}
		var words []string
		for _, span := range spans {
			words = append(words, string([]rune(text)[span.Start:span.End]))
		}
		t.Logf("%s: %q", text, words)
	}
}

func TestNativeCorrectionWords(t *testing.T) {
	cases := []struct {
		from, to string
		pair     domain.Replacement
	}{
		{"公示需要修改", "公式需要修改", domain.Replacement{From: "公示", To: "公式"}},
		{"把Arion的配置发给小王", "把Arion的配置发给晓王", domain.Replacement{From: "小王", To: "晓王"}},
		{"codex能爆吗", "codex能报吗", domain.Replacement{From: "爆吗", To: "报吗"}},
		{"成功处置深航机上送火事件", "成功处置深航机上纵火事件", domain.Replacement{From: "送火", To: "纵火"}},
	}
	for _, c := range cases {
		got := domain.ExtractCorrections(domain.FieldEdit{Inserted: c.from, Baseline: c.from, Settled: c.to}, Segmenter{})
		if got.Verdict != domain.EditLearned || len(got.Pairs) != 1 || got.Pairs[0] != c.pair {
			t.Errorf("%s: %+v want %+v", c.from, got, c.pair)
		}
	}
}

func TestAmbiguousSingleCharacterIsNotWidened(t *testing.T) {
	got := domain.ExtractCorrections(domain.FieldEdit{Inserted: "这是全免的衣服", Baseline: "这是全免的衣服", Settled: "这是全棉的衣服"}, Segmenter{})
	if got.Verdict != domain.EditNoPair || len(got.Pairs) != 0 {
		t.Fatalf("ambiguous boundary learned: %+v", got)
	}
}

func TestUnicodeWordOffsets(t *testing.T) {
	const text = "🙂收到请回复"
	spans, err := (Segmenter{}).Words(text)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range spans {
		if string([]rune(text)[w.Start:w.End]) == "回复" {
			if w.Start != 4 || w.End != 6 {
				t.Fatalf("UTF-16 leaked into domain offsets: %+v", w)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("reply word missing")
	}
}
