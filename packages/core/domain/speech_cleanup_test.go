package domain

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSpeechCleanup(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"hesitation", "呃，那个，我想问一下，评审几点开始？", "我想问一下，评审几点开始？"},
		{"demonstrative", "我要那个，别拿这个。", "我要那个，别拿这个。"},
		{"attached demonstrative", "あのボタンは削除しないでください。", "あのボタンは削除しないでください。"},
		{"quoted fillers", "文案里的“呃”和\"um\"都保留。", "文案里的“呃”和\"um\"都保留。"},
		{"unclosed quote", "保留“呃，那个，我想问一下", "保留“呃，那个，我想问一下"},
		{"code", "Keep `I I` and `, like,` intact.", "Keep `I I` and `, like,` intact."},
		{"hesitations", "Uh, um, please do not delete it.", "please do not delete it."},
		{"no implicit word joining", "I,like,need it today.", "I,like,need it today."},
		{"parenthetical", "I, like, need it today.", "I need it today."},
		{"meaningful like", "I like that, but I might wait.", "I like that, but I might wait."},
		{"stutter", "我我觉得这个这个可能不行。", "我觉得这个可能不行。"},
		{"english stutter", "I I might need the the earlier train.", "I might need the earlier train."},
		{"deliberate emphasis", "It is very very important.", "It is very very important."},
		{"spelled letters", "Use A A for the label.", "Use A A for the label."},
		{"identifier repetition", "Keep I_I I_I.", "Keep I_I I_I."},
		{"unicode repair offsets", "Keep İtem the red folder, I mean the blue folder.", "Keep İtem the blue folder."},
		{"technical topic", "就是说这个 max_retry_count 的话不要改名，然后的话 timeout 设成 30 秒。", "这个 max_retry_count 不要改名，timeout 设成 30 秒。"},
		{"conditional topic", "如果 timeout 失败的话不要重试。", "如果 timeout 失败的话不要重试。"},
		{"demonstrative condition", "这个 API 失败的话不要重试。", "这个 API 失败的话不要重试。"},
		{"grammatical repetition", "I had had enough of that that day.", "I had had enough of that that day."},
		{"name repair", "Send it to Alice, no wait, send it to Bob.", "send it to Bob."},
		{"noun repair", "Use the red folder, I mean the blue folder.", "Use the blue folder."},
		{"repair scope", "Keep the draft. Send it to Alice, no wait, send it to Bob.", "Keep the draft. send it to Bob."},
		{"number repair", "会議は三時、いや、五時からです。", "会議は五時からです。"},
		{"day repair", "La réunion est mardi, non pardon, jeudi.", "La réunion est jeudi."},
		{"spanish repair", "La reunión es el martes, perdón, el jueves.", "La reunión es el jueves."},
		{"japanese hesitation", "えーと、あの、会議は何時ですか。", "会議は何時ですか。"},
		{"uncertain repair remains", "I might send it to Alice, no wait, I will send it to Bob.", "I might send it to Alice, no wait, I will send it to Bob."},
		{"negated repair remains", "Do not ship Tuesday, no wait, do ship Thursday.", "Do not ship Tuesday, no wait, do ship Thursday."},
		{"apology is not a repair", "I need the report, sorry, I need the spreadsheet too.", "I need the report, sorry, I need the spreadsheet too."},
		{"partial restatement remains", "Send it to Alice in Berlin, no wait, send it to Bob.", "Send it to Alice in Berlin, no wait, send it to Bob."},
		{"unrelated clause", "The report is ready, sorry, I have a question.", "The report is ready, sorry, I have a question."},
		{"unfinished repair", "Send it to Alice, no wait, ", "Send it to Alice, no wait,"},
		{"all padding", "呃，嗯", "呃，嗯"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanSpeech(tt.source)
			if got.Text != tt.want {
				t.Fatalf("got %q want %q; spans=%+v", got.Text, tt.want, got.Spans)
			}
			applied, err := ApplySpeechSpans(tt.source, got.Spans)
			if err != nil || applied != strings.TrimSpace(got.Text) {
				t.Fatalf("span round trip: %q %v", applied, err)
			}
			if twice := CleanSpeech(got.Text); twice.Text != got.Text {
				t.Errorf("not idempotent: %q → %q", got.Text, twice.Text)
			}
		})
	}
}

func TestSpeechSpanContract(t *testing.T) {
	source := "呃，keep \"um\"."
	for _, spans := range [][]SpeechSpan{
		{{Start: -1, End: 1, Kind: SpeechHesitation}},
		{{Start: 0, End: 99, Kind: SpeechHesitation}},
		{{Start: 0, End: 1, Kind: SpeechHesitation}},
		{{Start: 0, End: 3, Kind: SpeechHesitation}, {Start: 0, End: 3, Kind: SpeechRepair}},
		{{Start: 11, End: 13, Kind: SpeechHesitation}},
		{{Start: 0, End: 3, Kind: "unknown"}},
	} {
		if _, err := ApplySpeechSpans(source, spans); err == nil {
			t.Fatalf("accepted invalid annotation: %+v", spans)
		}
	}
}

func TestCleanupDoesNotLicenseDroppingOtherContent(t *testing.T) {
	source := "I, like, need the report before Friday, and the budget must stay under 900 dollars."
	candidate := "I need the report."
	if got := ValidateRewrite(RewriteCheck{Source: source, Candidate: candidate}); got == RewriteAccepted {
		t.Fatal("accepted deletion of facts after cleanup")
	}
	source = "那个，我想问一下，今天的评审几点开始？"
	if got := ValidateRewrite(RewriteCheck{Source: source, Candidate: "我想问一下，今天的评审几点开始？"}); got != RewriteAccepted {
		t.Fatal(got)
	}
}

func FuzzSpeechCleanupContract(f *testing.F) {
	for _, source := range []string{"呃，那个，我想问一下。", "Keep İtem the red folder, I mean the blue folder.", "I_I I_I", "“那个”", "会議は三時、いや、五時です。", "嗯，就是说然后的话这个这个接口"} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if !utf8.ValidString(source) {
			t.Skip()
		}
		cleanup := CleanSpeech(source)
		text, err := ApplySpeechSpans(source, cleanup.Spans)
		if err != nil || text != strings.TrimSpace(cleanup.Text) {
			t.Fatalf("invalid deletion contract: %v", err)
		}
		before, after := []rune(source), []rune(cleanup.Text)
		j := 0
		for _, r := range before {
			if j < len(after) && r == after[j] {
				j++
			}
		}
		if j != len(after) {
			t.Fatal("cleanup introduced or reordered content")
		}
	})
}

func TestSpeechCleanupCopularQuantityRepair(t *testing.T) {
	cases := []struct {
		name, source, want string
		repaired           bool
	}{
		{"quantity", "把超时设成四十秒，不对，是四秒，保留并发数，先别改其他配置。", "把超时设成四秒，保留并发数，先别改其他配置。", true},
		{"digits", "超时40秒，不对，是4秒，先别重试。", "超时4秒，先别重试。", true},
		{"unicode spacing", "超时40秒，不对，　是　4秒，先别重试。", "超时4秒，先别重试。", true},
		{"unrelated units", "预算四十元，不对，是四秒，先别提交。", "预算四十元，不对，是四秒，先别提交。", false},
		{"negated replacement", "超时四十秒，不对，不是四秒，先别改。", "超时四十秒，不对，不是四秒，先别改。", false},
		{"uncertain replacement", "超时四十秒，不对，也许是四秒，先别改。", "超时四十秒，不对，也许是四秒，先别改。", false},
		{"different predicate", "超时四十秒，不对，是四秒后重试，先别提交。", "超时四十秒，不对，是四秒后重试，先别提交。", false},
		{"explicit exclusion", "把超时设成四秒，不是四十秒，先别改配置。", "把超时设成四秒，不是四十秒，先别改配置。", false},
		{"literal", "保留“超时四十秒，不对，是四秒”。", "保留“超时四十秒，不对，是四秒”。", false},
		{"untyped entity", "地点北京，不对，是上海。", "地点北京，不对，是上海。", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanSpeech(tt.source)
			if got.Text != tt.want {
				t.Fatalf("got %q, want %q; spans=%+v", got.Text, tt.want, got.Spans)
			}
			if tt.repaired && (len(got.Spans) != 1 || got.Spans[0].Kind != SpeechRepair) {
				t.Fatalf("missing scoped repair: %+v", got.Spans)
			}
			if !tt.repaired && len(got.Spans) != 0 {
				t.Fatalf("unsupported deletion: %+v", got.Spans)
			}
			applied, err := ApplySpeechSpans(tt.source, got.Spans)
			if err != nil || applied != tt.want || CleanSpeech(got.Text).Text != got.Text {
				t.Fatalf("invalid byte-span replay or idempotence: %q, %v", applied, err)
			}
		})
	}
}
