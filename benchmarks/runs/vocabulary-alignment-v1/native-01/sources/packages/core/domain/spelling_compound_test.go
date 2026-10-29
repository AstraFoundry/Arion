package domain

import (
	"reflect"
	"testing"
)

func TestContextualCompoundSpellingAcrossScenes(t *testing.T) {
	lexicon := &SpellingLexicon{}
	for _, scene := range []Scene{SceneChat, SceneEmail, SceneWork, ScenePrompt, SceneSearch} {
		t.Run(string(scene), func(t *testing.T) {
			source := "检查 chatter fly 和 super whisper。"
			context := map[string]string{"Screen": "Chatterfly superwhisper"}
			preparation := lexicon.PrepareContextual(source, context, string(scene))
			if len(preparation.Hypotheses) != 2 {
				t.Fatalf("hypotheses = %#v", preparation.Hypotheses)
			}
			for i, want := range []string{"Chatterfly", "superwhisper"} {
				h := preparation.Hypotheses[i]
				if !reflect.DeepEqual(h.Candidates, []string{want}) || h.Status != "unique" || source[h.Start:h.End] != h.Source {
					t.Fatalf("hypothesis = %#v", h)
				}
			}
			// Nomination never alters the source, even with a unique spelling.
			if preparation.Source != source {
				t.Fatalf("source changed: %q", preparation.Source)
			}
			kept, err := ApplySpellingChoices(`{"e0":"keep","e1":"keep"}`, preparation)
			if err != nil || kept != source {
				t.Fatalf("keep = %q, %v", kept, err)
			}
			selected, err := ApplySpellingChoices(`{"e0":"replace","e1":"replace"}`, preparation)
			if err != nil || selected != "检查 Chatterfly 和 superwhisper。" {
				t.Fatalf("selection = %q, %v", selected, err)
			}
		})
	}
}

func TestContextualCompoundRejectsUnsupportedAndLiteralChanges(t *testing.T) {
	lexicon := &SpellingLexicon{}
	for _, source := range []string{
		"检查 chad fly。", "检查 chat fly。", "检查 chatter fly2。", "检查 _chatter fly。", "检查 chatter fly_。",
		"保留“chatter fly”。", "保留 `chatter fly`。", "保留 \"chatter fly\"。", "检查 Chatterfly。",
	} {
		t.Run(source, func(t *testing.T) {
			p := lexicon.PrepareContextual(source, map[string]string{"Screen": "Chatterfly"}, string(ScenePrompt))
			if len(p.Hypotheses) != 0 {
				t.Fatalf("unsupported hypothesis = %#v", p.Hypotheses)
			}
		})
	}
	for _, context := range []map[string]string{nil, {}, {"Screen": "Shopping list: milk bread"}} {
		p := lexicon.PrepareContextual("检查 chatter fly。", context, string(ScenePrompt))
		if len(p.Hypotheses) != 0 {
			t.Fatalf("unrelated context hypothesis = %#v", p.Hypotheses)
		}
	}
}

func TestContextualCompoundRetainsAmbiguousCapitalization(t *testing.T) {
	p := (&SpellingLexicon{}).PrepareContextual("检查 chatter fly。", map[string]string{"Screen": "Chatterfly ChatterFly"}, string(SceneChat))
	if len(p.Hypotheses) != 1 {
		t.Fatalf("hypotheses = %#v", p.Hypotheses)
	}
	h := p.Hypotheses[0]
	if h.Status != "ambiguous" || !reflect.DeepEqual(h.Candidates, []string{"ChatterFly", "Chatterfly"}) {
		t.Fatalf("hypothesis = %#v", h)
	}
	if _, err := ApplySpellingChoices(`{"e0":"replace"}`, p); err == nil {
		t.Fatal("ambiguous alternative accepted as certain")
	}
}

func TestContextualCompoundWordWindowBounds(t *testing.T) {
	lexicon := &SpellingLexicon{}
	for _, tc := range []struct {
		source, context string
		count           int
	}{
		{"read config", "readConfig", 1},
		{"get current user account display name", "getCurrentUserAccountDisplayName", 1},
		{"get current user account display name now", "getCurrentUserAccountDisplayNameNow", 0},
	} {
		p := lexicon.PrepareContextual(tc.source, map[string]string{"Screen": tc.context}, string(SceneChat))
		if len(p.Hypotheses) != tc.count {
			t.Fatalf("%q hypotheses = %#v", tc.source, p.Hypotheses)
		}
	}
}

func TestHistoricalSpellingPreparationKeepsCompoundBoundary(t *testing.T) {
	lexicon := &SpellingLexicon{}
	for _, scene := range []string{string(ScenePrompt), string(SceneChat)} {
		old := lexicon.Prepare("检查 chatter fly。", map[string]string{"Screen": "Chatterfly"}, scene)
		if len(old.Hypotheses) != 0 {
			t.Fatalf("historical producer changed: %#v", old.Hypotheses)
		}
	}
	p := lexicon.Prepare("检查 chatter fly。", map[string]string{"Screen": "ChatterFly"}, string(ScenePrompt))
	if len(p.Hypotheses) != 1 || !reflect.DeepEqual(p.Hypotheses[0].Candidates, []string{"ChatterFly"}) {
		t.Fatalf("historical identifier behavior changed: %#v", p.Hypotheses)
	}
}

func TestEarlierSourceIdentifierRequiresIndependentMatchingFrame(t *testing.T) {
	lexicon := &SpellingLexicon{}
	source := "检查一下 chad fly，然后导出日志。"
	context := map[string]string{"EarlierSource": "检查一下 Chatterfly", "Screen": "软件 Chatterfly"}
	p := lexicon.PrepareContextual(source, context, string(SceneChat))
	if len(p.Hypotheses) != 1 || p.Hypotheses[0].Kind != "earlier-source-identifier" || !reflect.DeepEqual(p.Hypotheses[0].Candidates, []string{"Chatterfly"}) {
		t.Fatalf("hypotheses = %#v", p.Hypotheses)
	}
	selected, err := ApplySpellingChoices(`{"e0":"keep"}`, p)
	if err != nil || selected != source {
		t.Fatalf("unverified alternative was applied: %q %v", selected, err)
	}
	legacy := lexicon.Prepare(source, context, string(SceneChat))
	if len(legacy.Hypotheses) != 0 {
		t.Fatalf("legacy producer changed: %#v", legacy.Hypotheses)
	}
	for _, tc := range []struct{ name, source, earlier, screen string }{
		{"no earlier", source, "", "软件 Chatterfly"},
		{"no independent", source, "检查一下 Chatterfly", ""},
		{"other frame", source, "打开 Chatterfly", "软件 Chatterfly"},
		{"different current frame", "这是 chad fly。", "检查一下 Chatterfly", "软件 Chatterfly"},
		{"short frame", "看 chad fly。", "看 Chatterfly", "软件 Chatterfly"},
		{"quoted", "检查一下“chad fly”。", "检查一下 Chatterfly", "软件 Chatterfly"},
		{"not a prior token", source, "检查一下 ChatterflyExtra", "软件 Chatterfly"},
		{"known new name", source, "检查一下 Chatterfly", "软件 Chatterfly Chadfly"},
		{"real product change", "检查一下 Typeless。", "检查一下 Chatterfly", "软件 Chatterfly Typeless"},
		{"missing initial evidence", "检查一下 other fly。", "检查一下 Chatterfly", "软件 Chatterfly"},
		{"missing final evidence", "检查一下 chad bug。", "检查一下 Chatterfly", "软件 Chatterfly"},
		{"source boundary", "检查一下 chad fly2。", "检查一下 Chatterfly", "软件 Chatterfly"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := lexicon.PrepareContextual(tc.source, map[string]string{"EarlierSource": tc.earlier, "Screen": tc.screen}, string(SceneChat))
			for _, h := range got.Hypotheses {
				if h.Kind == "earlier-source-identifier" {
					t.Fatalf("unsupported prior nomination = %#v", h)
				}
			}
		})
	}
}

func TestEarlierSourceIsNotOrdinaryIndependentContext(t *testing.T) {
	p := (&SpellingLexicon{}).PrepareContextual("检查 chatter fly。", map[string]string{"EarlierSource": "检查 Chatterfly。"}, string(SceneChat))
	if len(p.Hypotheses) != 0 {
		t.Fatalf("earlier source leaked into independent context: %#v", p.Hypotheses)
	}
}

func TestEarlierSourceLocalRepairAndAmbiguity(t *testing.T) {
	lexicon := &SpellingLexicon{}
	context := map[string]string{"EarlierSource": "检查一下 Chatterfly", "Screen": "Chatterfly"}
	for _, source := range []string{
		"检查一下 chad fly，不对，改用 Typeless。",
		"检查一下 chad fly，改用 Typeless。",
		"不是检查一下 chad fly，而是检查 Typeless。",
	} {
		p := lexicon.PrepareContextual(source, context, string(SceneChat))
		for _, h := range p.Hypotheses {
			if h.Kind == "earlier-source-identifier" {
				t.Fatalf("repair cue ignored for %q: %#v", source, h)
			}
		}
	}
	// A correction two clauses away does not invalidate the local evidence.
	p := lexicon.PrepareContextual("检查一下 chad fly，导出日志，别处说错了。", context, string(SceneChat))
	if len(p.Hypotheses) != 1 {
		t.Fatalf("unrelated repair vetoed local evidence: %#v", p.Hypotheses)
	}
	ambiguous := lexicon.PrepareContextual("检查一下 chad fly。", map[string]string{
		"EarlierSource": "检查一下 Chatterfly。检查一下 Chatzzfly。",
		"Screen":        "Chatterfly Chatzzfly",
	}, string(SceneChat))
	if len(ambiguous.Hypotheses) != 1 || ambiguous.Hypotheses[0].Status != "ambiguous" || len(ambiguous.Hypotheses[0].Candidates) != 2 {
		t.Fatalf("ambiguity lost: %#v", ambiguous.Hypotheses)
	}
}

func TestEarlierSourceSharedEdgesAreBounded(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"chadfly", "chatterfly", true},
		{"chadfly", "chabcdefghijklmnopfly", false},
		{"chafly", "chatterfly", false},
		{"abcdefg", "abcdefgh", false},
		{"zxadfly", "chatterfly", false},
	} {
		if got := spellingSharedEdges(tc.a, tc.b); got != tc.want {
			t.Fatalf("edges(%q,%q)=%v", tc.a, tc.b, got)
		}
	}
	// A full Latin word provides an exact frame independently of CJK counting.
	p := (&SpellingLexicon{}).PrepareContextual("Inspect chad fly.", map[string]string{"EarlierSource": "Inspect Chatterfly.", "Before": "Chatterfly"}, string(SceneChat))
	if len(p.Hypotheses) != 1 {
		t.Fatalf("Latin frame not recognized: %#v", p.Hypotheses)
	}
}

func TestContextualSpellingDiscussionBoundsCandidateGeneration(t *testing.T) {
	lexicon := &SpellingLexicon{}
	for _, source := range []string{
		"请检查 chatter fly 的拼写。",
		"请检查 chatter fly 这个短语。",
		"请检查 chatter fly，保留字面内容。",
		"请检查 chatter fly；这两个单词保持分开。",
		"请检查 chatter fly，保留内部空格。",
		"请检查 chatter fly，当前写法优先。",
		"请检查 chatter fly，新名称以这里为准。",
		"Inspect chatter fly as a phrase.",
		"Inspect chatter fly as an intentional input.",
		"Inspect chatter fly; preserve its two words.",
		"Inspect chatter fly, keep the internal space.",
		"Inspect chatter fly; discuss its spelling.",
		"Inspect chatter fly as a literal.",
	} {
		t.Run(source, func(t *testing.T) {
			got := lexicon.PrepareContextual(source, map[string]string{"Screen": "Chatterfly"}, string(ScenePrompt))
			if got.Source != source || len(got.Hypotheses) != 0 {
				t.Fatalf("language discussion nominated an edit: %#v", got)
			}
		})
	}
	// The historical method still performs its previous candidate generation.
	legacy := lexicon.Prepare("检查 chatter fly 的拼写。", map[string]string{"Screen": "ChatterFly"}, string(ScenePrompt))
	if len(legacy.Hypotheses) != 1 {
		t.Fatalf("historical Prepare changed: %#v", legacy.Hypotheses)
	}
}

func TestContextualSourceIntentIsSentenceLocal(t *testing.T) {
	lexicon := &SpellingLexicon{}
	for _, source := range []string{
		"请检查 chatter fly 的日志。另一个测试保留“alpha beta”这个字面量。",
		"讨论单词的写法。请检查 chatter fly 的日志。",
		"Inspect chatter fly. Preserve the literal \"alpha beta\" elsewhere.",
		"Discuss the phrase elsewhere! Inspect chatter fly.",
		"请检查 chatter fly 的名称和函数名，重试 2 次。",
		"Inspect chatter fly and keep two retries with a 250 millisecond timeout.",
	} {
		t.Run(source, func(t *testing.T) {
			got := lexicon.PrepareContextual(source, map[string]string{"Screen": "Chatterfly"}, string(ScenePrompt))
			if got.Source != source || len(got.Hypotheses) != 1 {
				t.Fatalf("ordinary product context was blocked: %#v", got)
			}
		})
	}
}

func TestEarlierSourceMistakeInvalidatesPriorCandidateOnly(t *testing.T) {
	lexicon := &SpellingLexicon{}
	context := map[string]string{"EarlierSource": "请检查 Chatterfly。", "Screen": "Chatterfly"}
	for _, source := range []string{
		"请检查 chad fly，刚才的识别错了。",
		"请检查 chad fly；前面的识别不对。",
	} {
		got := lexicon.PrepareContextual(source, context, string(ScenePrompt))
		if got.Source != source || len(got.Hypotheses) != 0 {
			t.Fatalf("invalidated recognition nominated: %#v", got)
		}
	}
	english := map[string]string{"EarlierSource": "Inspect Chatterfly.", "Screen": "Chatterfly"}
	for _, source := range []string{
		"Inspect chad fly; the earlier transcript was mistaken.",
		"Inspect chad fly, the earlier recognition was wrong.",
	} {
		got := lexicon.PrepareContextual(source, english, string(ScenePrompt))
		if len(got.Hypotheses) != 0 {
			t.Fatalf("invalidated recognition nominated: %#v", got)
		}
	}
	got := lexicon.PrepareContextual("请检查 chad fly 的日志。另一个功能的识别错了。", context, string(ScenePrompt))
	if len(got.Hypotheses) != 1 || got.Hypotheses[0].Kind != "earlier-source-identifier" {
		t.Fatalf("unrelated sentence invalidated prior: %#v", got.Hypotheses)
	}
}
