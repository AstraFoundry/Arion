package domain

import "testing"

func TestDecideScene(t *testing.T) {
	cases := []struct {
		name  string
		hints SceneHints
		want  SceneDecision
	}{
		{"nothing known", SceneHints{}, SceneDecision{Scene: SceneChat, Route: RouteDefault}},
		{"mail app", SceneHints{App: "com.apple.mail", Field: FieldText}, SceneDecision{SceneEmail, RouteApp, "com.apple.mail"}},
		{"work messenger", SceneHints{App: "com.electron.lark"}, SceneDecision{SceneWork, RouteApp, "com.electron.lark"}},
		{"personal messenger", SceneHints{App: "com.tencent.xinWeChat"}, SceneDecision{SceneChat, RouteApp, "com.tencent.xinWeChat"}},
		{"editor", SceneHints{App: "com.todesktop.230313mzl4w4u92"}, SceneDecision{ScenePrompt, RouteApp, "com.todesktop.230313mzl4w4u92"}},
		{"vendor prefix", SceneHints{App: "com.jetbrains.goland"}, SceneDecision{ScenePrompt, RouteApp, "com.jetbrains."}},
		{"search field beats the app", SceneHints{App: "com.apple.mail", Field: FieldSearch}, SceneDecision{SceneSearch, RouteField, "search"}},
		{"site beats the browser", SceneHints{App: "com.google.Chrome", Site: "mail.google.com"}, SceneDecision{SceneEmail, RouteSite, "mail.google.com"}},
		{"www is the same host", SceneHints{Site: "www.google.com"}, SceneDecision{SceneSearch, RouteSite, "google.com"}},
		{"other subdomains are other products", SceneHints{Site: "docs.google.com"}, SceneDecision{Scene: SceneChat, Route: RouteDefault}},
		{"tenant subdomain", SceneHints{Site: "acme.atlassian.net"}, SceneDecision{SceneWork, RouteSite, "atlassian.net"}},
		{"tenant domain itself", SceneHints{Site: "feishu.cn"}, SceneDecision{SceneWork, RouteSite, "feishu.cn"}},
		{"host case and trailing dot", SceneHints{Site: "ChatGPT.com."}, SceneDecision{ScenePrompt, RouteSite, "chatgpt.com"}},
		{"unknown site falls back to the app", SceneHints{App: "com.tinyspeck.slackmacgap", Site: "files.example.org"}, SceneDecision{SceneWork, RouteApp, "com.tinyspeck.slackmacgap"}},
		{"unknown site in a browser", SceneHints{App: "com.apple.Safari", Site: "en.wikipedia.org"}, SceneDecision{Scene: SceneChat, Route: RouteDefault}},
		{"top-level label alone is not a site", SceneHints{Site: "com"}, SceneDecision{Scene: SceneChat, Route: RouteDefault}},
		{"lookalike host", SceneHints{Site: "notgithub.com"}, SceneDecision{Scene: SceneChat, Route: RouteDefault}},
	}
	for _, c := range cases {
		if got := DecideScene(c.hints, nil); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

func TestSceneTablesUseKnownScenes(t *testing.T) {
	for _, table := range []map[string]Scene{hostScenes, tenantScenes, appScenes} {
		for key, scene := range table {
			if !scene.Known() {
				t.Errorf("%s maps to unknown scene %q", key, scene)
			}
		}
	}
}

func TestSceneTagsAreMarkup(t *testing.T) {
	for _, scene := range Scenes {
		if got := ValidateRewrite(RewriteCheck{Source: "你好", Candidate: "[" + string(scene) + "]\n你好"}); got != RewriteMarkup {
			t.Errorf("echoed %s tag: got %s", scene, got)
		}
	}
}

func TestModelScene(t *testing.T) {
	enumerated := "要做两件事，第一是写测试，第二是发版本。"
	plain := "我晚点到。"
	cases := []struct {
		scene  Scene
		source string
		want   Scene
	}{
		{SceneChat, enumerated, SceneChat},
		{SceneEmail, plain, SceneEmail},
		{SceneList, plain, SceneList},
		{SceneWork, enumerated, SceneWork},
		{SceneWork, plain, SceneWork},
		{ScenePrompt, "Add two checks, first the token, second the expiry.", ScenePrompt},
		{ScenePrompt, plain, ScenePrompt},
		{SceneSearch, enumerated, SceneChat},
		{Scene("poem"), plain, SceneChat},
	}
	for _, c := range cases {
		if got := ModelScene(c.scene, c.source); got != c.want {
			t.Errorf("%s %q: got %s want %s", c.scene, c.source, got, c.want)
		}
	}
	for _, s := range Scenes {
		got := ModelScene(s, enumerated)
		if !slicesHas(ModelScenes, got) {
			t.Errorf("%s maps to %s, which the model is not taught", s, got)
		}
	}
}

func TestEnumerated(t *testing.T) {
	for _, s := range []string{"第一回邮件，第二整理报告", "一是成本，二是工期", "首先登录，其次打开设置", "First the API, then second the UI"} {
		if !Enumerated(s) {
			t.Errorf("%q not enumerated", s)
		}
	}
	for _, s := range []string{"第二天再说", "我是第一个到的", "first come, first served", "二是一个数字，一是另一个"} {
		if Enumerated(s) {
			t.Errorf("%q enumerated", s)
		}
	}
}

func TestFinishScene(t *testing.T) {
	cases := []struct {
		scene      Scene
		text, want string
	}{
		{SceneSearch, "杭州明天天气怎么样？", "杭州明天天气怎么样"},
		{SceneSearch, "how to reset a mac.\n", "how to reset a mac"},
		{SceneSearch, "react hooks\ncleanup", "react hooks cleanup"},
		{SceneChat, "我七点到。", "我七点到。"},
		{SceneEmail, "你好：\n\n合同已收到。", "你好：\n\n合同已收到。"},
	}
	for _, c := range cases {
		if got := FinishScene(c.scene, c.text); got != c.want {
			t.Errorf("%s %q: got %q want %q", c.scene, c.text, got, c.want)
		}
	}
}

func TestSceneOverrides(t *testing.T) {
	gmail := SceneHints{App: "com.google.Chrome", Site: "www.Mail.Google.com"}
	if key := SceneKey(gmail); key != "mail.google.com" {
		t.Fatalf("site key %q", key)
	}
	if key := SceneKey(SceneHints{App: "com.apple.mail", Field: FieldSearch}); key != "search@com.apple.mail" {
		t.Fatalf("search key %q", key)
	}
	if key := SceneKey(SceneHints{}); key != "" {
		t.Fatalf("empty key %q", key)
	}
	var none SceneOverrides
	chosen := none.WithScene("mail.google.com", SceneWork)
	if none != nil || len(chosen) != 1 {
		t.Fatal("WithScene changed its receiver")
	}
	if got := DecideScene(gmail, chosen); got != (SceneDecision{SceneWork, RouteUser, "mail.google.com"}) {
		t.Fatalf("override ignored: %+v", got)
	}
	// The choice is for the page, not for the browser's other sites, and
	// not for the page's search box.
	if got := DecideScene(SceneHints{App: "com.google.Chrome", Site: "github.com"}, chosen); got.Route != RouteSite {
		t.Fatalf("override leaked: %+v", got)
	}
	if got := DecideScene(SceneHints{App: "com.google.Chrome", Site: "mail.google.com", Field: FieldSearch}, chosen); got.Scene != SceneSearch {
		t.Fatalf("override applied to search box: %+v", got)
	}
	// Automatic again, and stored values that are not choosable are ignored.
	if got := DecideScene(gmail, chosen.WithScene("mail.google.com", "")); got.Route != RouteSite {
		t.Fatalf("reset ignored: %+v", got)
	}
	if got := DecideScene(gmail, SceneOverrides{"mail.google.com": "poem"}); got.Route != RouteSite {
		t.Fatalf("unknown override used: %+v", got)
	}
}
