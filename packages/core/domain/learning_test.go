package domain

import (
	"strings"
	"testing"
	"time"
)

func TestExtractCorrections(t *testing.T) {
	cases := []struct {
		name    string
		edit    FieldEdit
		verdict EditVerdict
		pairs   []Replacement
	}{
		{"cjk word fixed", FieldEdit{Inserted: "成功处置深航机上送火事件", Baseline: "新闻：成功处置深航机上送火事件", Settled: "新闻：成功处置深航机上纵火事件"},
			EditLearned, []Replacement{{From: "送火", To: "纵火"}}},
		{"latin name", FieldEdit{Inserted: "ask jason about the release", Baseline: "Hi, ask jason about the release", Settled: "Hi, ask Jayson about the release"},
			EditLearned, []Replacement{{From: "jason", To: "Jayson"}}},
		{"split words joined", FieldEdit{Inserted: "we use open ai models", Baseline: "we use open ai models", Settled: "we use OpenAI models"},
			EditLearned, []Replacement{{From: "open ai", To: "OpenAI"}}},
		{"two fixes", FieldEdit{Inserted: "把阿里昂的配置发给小王", Baseline: "把阿里昂的配置发给小王", Settled: "把Arion的配置发给晓王"},
			EditLearned, []Replacement{{From: "阿里昂", To: "Arion"}, {From: "小王", To: "晓王"}}},
		{"cjk to latin", FieldEdit{Inserted: "打开杰森文件", Baseline: "打开杰森文件", Settled: "打开JSON文件"},
			EditLearned, []Replacement{{From: "杰森", To: "JSON"}}},
		{"continued typing", FieldEdit{Inserted: "明天开会", Baseline: "明天开会", Settled: "明天开会，记得带电脑"}, EditNoPair, nil},
		{"typed before", FieldEdit{Inserted: "明天开会", Baseline: "备注：明天开会", Settled: "重要备注：明天开会"}, EditOutside, nil},
		{"punctuation only", FieldEdit{Inserted: "好的 我知道了", Baseline: "好的 我知道了", Settled: "好的，我知道了。"}, EditNoPair, nil},
		{"deleted word", FieldEdit{Inserted: "this is really good", Baseline: "this is really good", Settled: "this is good"}, EditNoPair, nil},
		{"whole rewrite", FieldEdit{Inserted: "let us meet tomorrow at noon", Baseline: "let us meet tomorrow at noon", Settled: "see you friday"}, EditTooLarge, nil},
		{"changed time", FieldEdit{Inserted: "明天下午三点开会", Baseline: "明天下午三点开会", Settled: "明天下午四点开会"}, EditNoPair, nil},
		{"changed amount", FieldEdit{Inserted: "the budget is 5000 dollars", Baseline: "the budget is 5000 dollars", Settled: "the budget is 6000 dollars"}, EditNoPair, nil},
		{"line start case", FieldEdit{Inserted: "done\nok thanks", Baseline: "done\nok thanks", Settled: "done\nOk thanks"}, EditNoPair, nil},
		{"lowercased name", FieldEdit{Inserted: "ask Cozette now", Baseline: "ask Cozette now", Settled: "ask cozette now"}, EditNoPair, nil},
		{"single letter", FieldEdit{Inserted: "merge to M today", Baseline: "merge to M today", Settled: "merge to main today"}, EditNoPair, nil},
		{"pronoun", FieldEdit{Inserted: "我觉得他也同意", Baseline: "我觉得他也同意", Settled: "我觉得她也同意"}, EditNoPair, nil},
		{"particle", FieldEdit{Inserted: "认真的学习", Baseline: "认真的学习", Settled: "认真地学习"}, EditNoPair, nil},
		{"cleared", FieldEdit{Inserted: "hello", Baseline: "hello", Settled: "  "}, EditCleared, nil},
		{"unchanged", FieldEdit{Inserted: "hello", Baseline: "x hello", Settled: "x hello"}, EditUnchanged, nil},
		{"insertion missing", FieldEdit{Inserted: "hello", Baseline: "goodbye", Settled: "goodbye!"}, EditNotFound, nil},
		{"many changes", FieldEdit{Inserted: "a1 b c d e f g h", Baseline: "a1 b c d e f g h", Settled: "a1 B c D e F g H"}, EditRewrite, nil},
		{"whole insertion replaced", FieldEdit{Inserted: "在", Baseline: "在", Settled: "再"}, EditTooLarge, nil},
		{"single han widened", FieldEdit{Inserted: "收到请回付", Baseline: "收到请回付", Settled: "收到请回复"},
			EditLearned, []Replacement{{From: "回付", To: "回复"}}},
		{"exempt homophone", FieldEdit{Inserted: "我们在来一次", Baseline: "我们在来一次", Settled: "我们再来一次"}, EditNoPair, nil},
		{"long phrase", FieldEdit{Inserted: "please send it", Baseline: "please send it", Settled: "please forward the quarterly planning spreadsheet it"}, EditTooLarge, nil},
		{"identifier", FieldEdit{Inserted: "call get user by id first", Baseline: "call get user by id first", Settled: "call getUserById first"},
			EditLearned, []Replacement{{From: "get user by id", To: "getUserById"}}},
		{"sentence case", FieldEdit{Inserted: "ok sounds good", Baseline: "ok sounds good", Settled: "Ok, sounds good!"}, EditNoPair, nil},
		{"mid-sentence case", FieldEdit{Inserted: "the new iphone is out", Baseline: "the new iphone is out", Settled: "the new iPhone is out"},
			EditLearned, []Replacement{{From: "iphone", To: "iPhone"}}},
		{"short rewrite", FieldEdit{Inserted: "我觉得这个方案还可以再改改", Baseline: "我觉得这个方案还可以再改改", Settled: "方案通过"}, EditTooLarge, nil},
	}
	for _, tc := range cases {
		got := ExtractCorrections(tc.edit)
		if got.Verdict != tc.verdict || !samePairs(got.Pairs, tc.pairs) {
			t.Errorf("%s: got %s %v want %s %v", tc.name, got.Verdict, got.Pairs, tc.verdict, tc.pairs)
		}
	}
}

func samePairs(a, b []Replacement) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestVocabularyApplySinglePass(t *testing.T) {
	v := NewVocabulary([]Replacement{{From: "open ai", To: "OpenAI"}, {From: "open", To: "Open"}, {From: "OpenAI", To: "X"}, {From: "阿里昂", To: "Arion"}})
	got, fired := v.Apply("Open AI and opener from 阿里昂, open now")
	if got != "OpenAI and opener from Arion, Open now" {
		t.Fatalf("got %q", got)
	}
	if len(fired) != 3 {
		t.Fatalf("fired %v", fired)
	}
}

func TestVocabularyTermsRelevantAndBounded(t *testing.T) {
	var reps []Replacement
	for i := 0; i < 500; i++ {
		reps = append(reps, Replacement{From: "word" + strings.Repeat("x", i%7) + string(rune('a'+i%26)), To: "W"})
	}
	reps = append(reps, Replacement{From: "杰森", To: "JSON"}, Replacement{From: "jason", To: "Jayson"})
	v := NewVocabulary(reps)
	terms := v.Terms("打开JSON文件，问 Jayson", MaxRecallTerms)
	if len(terms) != 2 || terms[0] != "杰森 → JSON" || terms[1] != "jason → Jayson" {
		t.Fatalf("terms %v", terms)
	}
	if got := v.Terms("nothing relevant", MaxRecallTerms); got != nil {
		t.Fatalf("irrelevant terms %v", got)
	}
}

func TestLearnRules(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	var entries []MemoryEntry
	entries, r := Learn(entries, Replacement{From: "送火", To: "纵火"}, "a", now)
	if r.Outcome != LearnAdded || len(entries) != 1 {
		t.Fatal(r)
	}
	entries, r = Learn(entries, Replacement{From: "送火", To: "纵火"}, "b", now)
	if r.Outcome != LearnReinforced || entries[0].Uses != 2 {
		t.Fatal(r)
	}
	entries, r = Learn(entries, Replacement{From: "纵火", To: "送火"}, "c", now)
	if r.Outcome != LearnReverted || len(entries) != 0 {
		t.Fatal(r, entries)
	}
	user := []MemoryEntry{{ID: "u", From: "arion", To: "Arion", Origin: OriginUser}}
	if _, r = Learn(user, Replacement{From: "Arion", To: "arion"}, "d", now); r.Outcome != LearnKept {
		t.Fatal(r)
	}
	if _, r = Learn(user, Replacement{From: "ARION", To: "ARI"}, "d", now); r.Outcome != LearnKept {
		t.Fatal(r)
	}
}

func TestPruneMemoryKeepsUserEntries(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	var entries []MemoryEntry
	entries = append(entries, MemoryEntry{ID: "old", From: "a", To: "b", Origin: OriginEdit, Used: now.Add(-LearnedTTL - time.Hour)})
	entries = append(entries, MemoryEntry{ID: "user-old", From: "c", To: "d", Origin: OriginUser, Used: now.Add(-10 * LearnedTTL)})
	for i := 0; i < MaxMemoryEntries+10; i++ {
		entries = append(entries, MemoryEntry{ID: "n", From: "x", To: "y", Origin: OriginEdit, Uses: 1 + i%3, Used: now})
	}
	got := PruneMemory(entries, now)
	if len(got) != MaxMemoryEntries {
		t.Fatalf("len %d", len(got))
	}
	if got[0].ID != "user-old" {
		t.Fatalf("user entry not kept first: %v", got[0])
	}
	for _, e := range got {
		if e.ID == "old" {
			t.Fatal("expired entry kept")
		}
	}
}
