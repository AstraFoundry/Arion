package domain

import (
	"reflect"
	"testing"
)

func TestCorrectionRules(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		policy CorrectionPolicy
		want   string
	}{
		{"cjk token spaces", "今天 天气 很 好 。", CorrectionPolicy{}, "今天天气很好。"},
		{"keeps cjk latin boundary", "把 PR merge 到 main 分支", CorrectionPolicy{}, "把 PR merge 到 main 分支"},
		{"space before punctuation", "hello , world !", CorrectionPolicy{}, "hello, world!"},
		{"uppercase recognizer", "I THINK WE SHOULD SHIP IT. OK THEN", CorrectionPolicy{LatinCase: LatinCaseUpper}, "I think we should ship it. Ok then"},
		{"uppercase mixed", "我们 用 GITHUB 部署", CorrectionPolicy{LatinCase: LatinCaseUpper}, "我们用 github 部署"},
		{"natural case untouched", "Use GitHub API", CorrectionPolicy{}, "Use GitHub API"},
		{"fillers", "嗯，我觉得 um 这个可以", CorrectionPolicy{RemoveFillers: true}, "我觉得这个可以"},
		{"filler inside word kept", "umbrella and hum", CorrectionPolicy{RemoveFillers: true}, "umbrella and hum"},
		{"vocabulary whole word", "push to arion and arionic", CorrectionPolicy{Replacements: []Replacement{{From: "arion", To: "Arion"}}}, "push to Arion and arionic"},
		{"vocabulary cjk", "打开阿里昂设置", CorrectionPolicy{Replacements: []Replacement{{From: "阿里昂", To: "Arion"}}}, "打开Arion设置"},
		{"numbers and urls untouched", "visit https://a.io/x?y=1 at 3.14 pm", CorrectionPolicy{RemoveFillers: true}, "visit https://a.io/x?y=1 at 3.14 pm"},
		{"negation kept", "我不 想 去", CorrectionPolicy{RemoveFillers: true}, "我不想去"},
		{"spelled acronym", "请把这个 P R merge 到 main 分支。", CorrectionPolicy{}, "请把这个 PR merge 到 main 分支。"},
		{"spelled acronym run", "call the A P I now", CorrectionPolicy{}, "call the API now"},
		{"single letters kept", "I think a B grade is fine", CorrectionPolicy{}, "I think a B grade is fine"},
	}
	for _, tc := range cases {
		got := Correct(tc.raw, tc.policy)
		if got.Text != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got.Text, tc.want)
		}
		if got.Raw != tc.raw || got.Version != RulesVersion {
			t.Errorf("%s: raw/version not preserved", tc.name)
		}
	}
}

func TestCorrectionReportsAppliedRules(t *testing.T) {
	got := Correct("hello", CorrectionPolicy{})
	if len(got.Applied) != 0 {
		t.Fatalf("no-op correction reported rules %v", got.Applied)
	}
	got = Correct("嗯 好", CorrectionPolicy{RemoveFillers: true})
	if !reflect.DeepEqual(got.Applied, []string{"fillers", "cjk-spacing"}) || got.Text != "好" {
		t.Fatalf("got %q %v", got.Text, got.Applied)
	}
}

func TestSelectDefaultModel(t *testing.T) {
	eval := func(zh, en, mixed, p95, rtf float64) ModelEvaluation {
		return ModelEvaluation{RunID: "r", Hardware: "darwin-arm64", Observations: 50, StopToReviewP95Ms: p95, RealTimeFactor: rtf,
			ErrorRates: map[string]float64{"zh": zh, "en": en, "mixed": mixed}}
	}
	all := []string{"zh", "en", "mixed"}
	options := []ModelOption{
		{ID: "accurate-but-slow", Platforms: []string{"darwin-arm64"}, Languages: all, Evaluation: []ModelEvaluation{eval(0.02, 0.03, 0.04, 1800, 0.5)}},
		{ID: "good", Platforms: []string{"darwin-arm64"}, Languages: all, SizeBytes: 200, Evaluation: []ModelEvaluation{eval(0.05, 0.06, 0.08, 400, 0.2)}},
		{ID: "good-small", Platforms: []string{"darwin-arm64"}, Languages: all, SizeBytes: 100, Evaluation: []ModelEvaluation{eval(0.05, 0.06, 0.08, 400, 0.2)}},
		{ID: "worse", Platforms: []string{"darwin-arm64"}, Languages: all, Evaluation: []ModelEvaluation{eval(0.10, 0.10, 0.10, 300, 0.1)}},
		{ID: "unmeasured", Platforms: []string{"darwin-arm64"}, Languages: all},
		{ID: "windows-only", Platforms: []string{"windows-amd64"}, Languages: all, Evaluation: []ModelEvaluation{eval(0, 0, 0, 1, 0.1)}},
	}
	got, err := SelectDefaultModel(options, DefaultSelectionPolicy("darwin-arm64"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Option.ID != "good-small" {
		t.Fatalf("selected %s", got.Option.ID)
	}
	if _, err := SelectDefaultModel(options[4:5], DefaultSelectionPolicy("darwin-arm64")); CodeOf(err) != CodeEvaluationUnavailable {
		t.Fatalf("unmeasured bundle selected: %v", err)
	}
	if !got.LatencyMeasured {
		t.Fatal("local run must mark latency measured")
	}
	// Intel Macs: no local run, so accuracy measured on Apple silicon decides
	// and latency is reported as unmeasured. Gate failures elsewhere do not
	// matter because latency there says nothing about this hardware.
	options[1].Platforms = append(options[1].Platforms, "darwin-amd64")
	options[0].Platforms = append(options[0].Platforms, "darwin-amd64")
	got, err = SelectDefaultModel(options, DefaultSelectionPolicy("darwin-amd64"))
	if err != nil || got.Option.ID != "accurate-but-slow" || got.LatencyMeasured {
		t.Fatalf("cross-hardware selection: %+v %v", got.Option.ID, err)
	}
	if _, err := SelectDefaultModel(options, DefaultSelectionPolicy("linux-amd64")); err == nil {
		t.Fatal("selected a bundle for an unsupported platform")
	}
}
