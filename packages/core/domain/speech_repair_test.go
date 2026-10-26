package domain

import (
	"strings"
	"testing"
)

func TestSpokenSelfRepairCollapsesToFinalSpan(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"month cascade", "十二月，十三月，什么十三，没有十三，十一月", "十一月"},
		{"month ellipsis", "十二月…十三月…什么十三，没有十三…十一月", "十一月"},
		{"month with frame", "开会定在十二月，十三月，什么十三，没有十三，十一月下午", "开会定在十一月下午"},
		{"month periods", "十二月。十三月。什么十三，没有十三。十一月", "十一月"},
		{"wrong count", "来了三十人，不对，五十人。", "来了五十人。"},
		{"bare numbers", "预算报三十，不对，五十。", "预算报五十。"},
		{"not A is B", "帮我告诉张三，不是张三，是李四，明天不用来了。", "帮我告诉李四，明天不用来了。"},
		{"not A is B numbers", "不是三十人，是五十人。", "五十人。"},
		{"typed not A is B after false start", "见十二月，不是十三月，是十一月。", "见十一月。"},
		{"english month", "Meet in December, no, November.", "Meet in November."},
		{"english count", "We need three people, no, five people.", "We need five people."},
		{"english not A it's B", "Send it to Alice, not Alice, it's Bob.", "Send it to Bob."},
		{"english frame", "Send it to Alice, no, send it to Bob.", "send it to Bob."},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanSpeech(tt.source)
			if got.Text != tt.want {
				t.Fatalf("got %q want %q spans=%+v", got.Text, tt.want, got.Spans)
			}
			if strings.Contains(got.Text, "十二月") || strings.Contains(got.Text, "十三") || strings.Contains(got.Text, "什么十三") || strings.Contains(got.Text, "没有十三") {
				t.Fatalf("superseded repair talk remains: %q", got.Text)
			}
			applied, err := ApplySpeechSpans(tt.source, got.Spans)
			if err != nil || applied != strings.TrimSpace(got.Text) {
				t.Fatalf("span round trip: %q %v", applied, err)
			}
			if twice := CleanSpeech(got.Text); twice.Text != got.Text {
				t.Fatalf("not idempotent: %q → %q", got.Text, twice.Text)
			}
		})
	}
}

func TestSpokenRepairLeavesContentAndWording(t *testing.T) {
	cases := []struct{ name, source string }{
		{"exclusion", "把超时设成四秒，不是四十秒，先别改配置。"},
		{"contrast", "读数是负一点五度，不是正一点五度。"},
		{"identity", "我不是张三，是李四。"},
		{"conjunction", "十二月和十一月都要。"},
		{"list without repair", "十二月，十一月。"},
		{"quoted repair", "保留“十二月，十三月，什么十三，没有十三，十一月”。"},
		{"original terms", "请核对账目，别急着提交。"},
		{"english wording", "Please reconcile the ledger before Friday."},
		{"english exclusion", "Set pollingDelay to five seconds, not fifty seconds, keep the attempt limit."},
		{"unfinished", "十二月，十三月，什么十三，没有十三"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanSpeech(tt.source)
			if got.Text != tt.source && strings.TrimSpace(got.Text) != strings.TrimSpace(tt.source) {
				t.Fatalf("changed wording: got %q want %q spans=%+v", got.Text, tt.source, got.Spans)
			}
		})
	}
	kept := CleanSpeech("请核对账目，别急着提交。").Text
	if !strings.Contains(kept, "核对") || !strings.Contains(kept, "提交") || strings.Contains(kept, "检查") || strings.Contains(kept, "发送") {
		t.Fatalf("original terms were swapped: %q", kept)
	}
	ledger := CleanSpeech("Please reconcile the ledger before Friday.").Text
	if !strings.Contains(ledger, "reconcile") || !strings.Contains(ledger, "ledger") {
		t.Fatalf("original English terms were dropped: %q", ledger)
	}
}

func TestASRCorrectionDoesNotSwapUnrelatedWords(t *testing.T) {
	// Protocol value "replace" is a spelling correction of an ASR mishear.
	// It is not permission to substitute a synonym for a word that was heard correctly.
	lexicon := &SpellingLexicon{Readings: map[string][]string{
		"飞": {"fei1"},
		"鼠": {"shu3"},
		"书": {"shu1"},
		"纪": {"ji4"},
		"要": {"yao4"},
		"请": {"qing3"},
		"把": {"ba3"},
		"发": {"fa1"},
		"到": {"dao4"},
		"群": {"qun2"},
	}}
	source := "请把纪要发到飞鼠。"
	prepared := lexicon.PrepareContextual(source, map[string]string{"Screen": "请发到飞书群"}, string(SceneChat))
	if len(prepared.Hypotheses) != 1 || prepared.Hypotheses[0].Status != "unique" {
		t.Fatalf("expected one ASR correction candidate, got %#v", prepared.Hypotheses)
	}
	h := prepared.Hypotheses[0]
	if len(h.Candidates) != 1 || !strings.Contains(h.Candidates[0], "飞书") {
		t.Fatalf("correction candidate = %#v", h)
	}
	if strings.Contains(h.Source, "纪要") {
		t.Fatalf("correction span swallowed an unrelated word: %q", h.Source)
	}
	choices := `{"` + h.ID + `":"replace"}`
	corrected, err := ApplySpellingChoices(choices, prepared)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(corrected, "飞书") || strings.Contains(corrected, "飞鼠") {
		t.Fatalf("mishear was not corrected: %q", corrected)
	}
	if !strings.Contains(corrected, "纪要") {
		t.Fatalf("correct original term was replaced: %q", corrected)
	}
	keptChoices := `{"` + h.ID + `":"keep"}`
	kept, err := ApplySpellingChoices(keptChoices, prepared)
	if err != nil || kept != source {
		t.Fatalf("keep changed the source: %q %v", kept, err)
	}
}
