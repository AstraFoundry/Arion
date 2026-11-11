package contentrules

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func mustParse(t testing.TB, rules string) *List {
	t.Helper()
	l, err := Parse(rules)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return l
}

const page = "https://news.example/article"

func TestMatch(t *testing.T) {
	tests := []struct {
		name  string
		rules string
		req   Request
		want  Result
	}{
		{"domain blocks", `[{"trigger":{"url-filter":"^[^:]+://+([^:/]+\\.)?ads\\.test[:/]"},"action":{"type":"block"}}]`,
			Request{URL: "https://cdn.ads.test/x.js", TopURL: page, Type: Script}, Result{Block: true}},
		{"domain spares others", `[{"trigger":{"url-filter":"^[^:]+://+([^:/]+\\.)?ads\\.test[:/]"},"action":{"type":"block"}}]`,
			Request{URL: "https://notads.test/x.js", TopURL: page, Type: Script}, Result{}},
		{"domain spares a longer host", `[{"trigger":{"url-filter":"^[^:]+://+([^:/]+\\.)?ads\\.test[:/]"},"action":{"type":"block"}}]`,
			Request{URL: "https://ads.test.example/x.js", TopURL: page, Type: Script}, Result{}},
		{"path", `[{"trigger":{"url-filter":"/banner/.*\\.gif"},"action":{"type":"block"}}]`,
			Request{URL: "https://a.example/img/BANNER/top.gif", TopURL: page, Type: Image}, Result{Block: true}},
		{"case-sensitive", `[{"trigger":{"url-filter":"/Banner/","url-filter-is-case-sensitive":true},"action":{"type":"block"}}]`,
			Request{URL: "https://a.example/banner/top.gif", TopURL: page, Type: Image}, Result{}},
		{"resource-type matches", `[{"trigger":{"url-filter":"track","resource-type":["script","raw"]},"action":{"type":"block"}}]`,
			Request{URL: "https://a.example/track.js", TopURL: page, Type: Script}, Result{Block: true}},
		{"resource-type spares", `[{"trigger":{"url-filter":"track","resource-type":["script"]},"action":{"type":"block"}}]`,
			Request{URL: "https://a.example/track.png", TopURL: page, Type: Image}, Result{}},
		{"third-party", `[{"trigger":{"url-filter":"pixel","load-type":["third-party"]},"action":{"type":"block"}}]`,
			Request{URL: "https://t.other.example/pixel", TopURL: page, Type: Image}, Result{Block: true}},
		{"third-party spares the site", `[{"trigger":{"url-filter":"pixel","load-type":["third-party"]},"action":{"type":"block"}}]`,
			Request{URL: "https://static.news.example/pixel", TopURL: page, Type: Image}, Result{}},
		{"first-party", `[{"trigger":{"url-filter":"pixel","load-type":["first-party"]},"action":{"type":"block"}}]`,
			Request{URL: "https://static.news.example/pixel", TopURL: page, Type: Image}, Result{Block: true}},
		{"public suffix sites differ", `[{"trigger":{"url-filter":"pixel","load-type":["third-party"]},"action":{"type":"block"}}]`,
			Request{URL: "https://b.github.io/pixel", TopURL: "https://a.github.io/", Type: Image}, Result{Block: true}},
		{"if-domain", `[{"trigger":{"url-filter":"ad","if-domain":["*news.example"]},"action":{"type":"block"}}]`,
			Request{URL: "https://x.example/ad.js", TopURL: "https://www.news.example/", Type: Script}, Result{Block: true}},
		{"if-domain exact", `[{"trigger":{"url-filter":"ad","if-domain":["news.example"]},"action":{"type":"block"}}]`,
			Request{URL: "https://x.example/ad.js", TopURL: "https://www.news.example/", Type: Script}, Result{}},
		{"unless-domain", `[{"trigger":{"url-filter":"ad","unless-domain":["*news.example"]},"action":{"type":"block"}}]`,
			Request{URL: "https://x.example/ad.js", TopURL: page, Type: Script}, Result{}},
		{"if-top-url", `[{"trigger":{"url-filter":"ad","if-top-url":["^https://news\\.example/art"]},"action":{"type":"block"}}]`,
			Request{URL: "https://x.example/ad.js", TopURL: page, Type: Script}, Result{Block: true}},
		{"unless-top-url", `[{"trigger":{"url-filter":"ad","unless-top-url":["/article"]},"action":{"type":"block"}}]`,
			Request{URL: "https://x.example/ad.js", TopURL: page, Type: Script}, Result{}},
		{"ignore-previous-rules", `[
			{"trigger":{"url-filter":"^[^:]+://+([^:/]+\\.)?ads\\.test[:/]"},"action":{"type":"block"}},
			{"trigger":{"url-filter":".*","if-domain":["*news.example"]},"action":{"type":"ignore-previous-rules"}}]`,
			Request{URL: "https://ads.test/x.js", TopURL: page, Type: Script}, Result{}},
		{"block after ignore", `[
			{"trigger":{"url-filter":".*","if-domain":["*news.example"]},"action":{"type":"ignore-previous-rules"}},
			{"trigger":{"url-filter":"^[^:]+://+([^:/]+\\.)?ads\\.test[:/]"},"action":{"type":"block"}}]`,
			Request{URL: "https://ads.test/x.js", TopURL: page, Type: Script}, Result{Block: true}},
		{"block-cookies", `[{"trigger":{"url-filter":"tracker"},"action":{"type":"block-cookies"}}]`,
			Request{URL: "https://tracker.example/", TopURL: page, Type: Raw}, Result{BlockCookies: true}},
		{"make-https", `[{"trigger":{"url-filter":"^http://plain\\.example/"},"action":{"type":"make-https"}}]`,
			Request{URL: "http://plain.example/a", TopURL: page, Type: Image}, Result{MakeHTTPS: true}},
		{"hiding does not block", `[{"trigger":{"url-filter":".*"},"action":{"type":"css-display-none","selector":".ad"}}]`,
			Request{URL: "https://x.example/", TopURL: page, Type: Document}, Result{}},
		{"page is its own top", `[{"trigger":{"url-filter":"ad","load-type":["first-party"]},"action":{"type":"block"}}]`,
			Request{URL: "https://ad.example/", Type: Document}, Result{Block: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match([]*List{mustParse(t, tt.rules)}, tt.req); got != tt.want {
				t.Errorf("Match = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Each list's ignore-previous-rules undoes only its own rules.
func TestMatchLists(t *testing.T) {
	blocks := mustParse(t, `[{"trigger":{"url-filter":"ads"},"action":{"type":"block"}}]`)
	ignores := mustParse(t, `[{"trigger":{"url-filter":".*"},"action":{"type":"ignore-previous-rules"}}]`)
	if got := Match([]*List{blocks, ignores}, Request{URL: "https://ads.example/", TopURL: page}); !got.Block {
		t.Errorf("a later list's ignore-previous-rules undid an earlier list's block")
	}
}

func TestParseErrors(t *testing.T) {
	for _, rules := range []string{
		`{}`,
		`[{"trigger":{},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":"("},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":".*"},"action":{"type":"redirect"}}]`,
		`[{"trigger":{"url-filter":".*"},"action":{"type":"css-display-none"}}]`,
		`[{"trigger":{"url-filter":".*","resource-type":["video"]},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":".*","load-type":["second-party"]},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":".*","if-domain":["a"],"unless-domain":["b"]},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":".*","if-domain":["a"],"if-top-url":["b"]},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":".*","if-domain":["Example.com"]},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":"a|b"},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":"a{2}"},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":"a\\d"},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":"a^b"},"action":{"type":"block"}}]`,
		`[{"trigger":{"url-filter":"é"},"action":{"type":"block"}}]`,
		`[]`,
	} {
		if _, err := Parse(rules); err == nil {
			t.Errorf("Parse(%s) took it", rules)
		}
	}
}

// What WebKit takes, as tried on macOS.
func TestParseTakes(t *testing.T) {
	for _, filter := range []string{`[^a-z]x`, `(ab)?c`, `(ab)+c`, `.*?x`, `x$`, `^x`, `\\/x`, `[a\\-z]`, `a\\.b\\?c`,
		`[.]x`, `a\\|b`, `[|{^]`, `^[^:]+://+([^:/]+\\.)?ex\\.com[/:]`} {
		rules := `[{"trigger":{"url-filter":"` + filter + `","unknown":1},"action":{"type":"block"}}]`
		if _, err := Parse(rules); err != nil {
			t.Errorf("Parse(%s): %v", rules, err)
		}
	}
}

func TestFilterDomain(t *testing.T) {
	for filter, want := range map[string]string{
		`^[^:]+://+([^:/]+\.)?ads\.example\.com[:/]`:    "ads.example.com",
		`^[^:]+://+([^:/]+\.)?Ads\.Example\.com[/:]`:    "ads.example.com",
		`^[^:]+://+([^:/]+\.)?ads\.example\.com/banner`: "ads.example.com",
		`^[^:]+://+([^:/]+\.)?ads\.example\.com`:        "",
		`^[^:]+://+([^:/]+\.)?ads\.example\.co.*[:/]`:   "",
		`^[^:]+://+([^:/]+\.)?ads\.example\.com/*x`:     "",
		`^https?://ads\.example\.com/`:                  "",
	} {
		if got := filterDomain(filter); got != want {
			t.Errorf("filterDomain(%s) = %q, want %q", filter, got, want)
		}
	}
}

func TestLongestLiteral(t *testing.T) {
	for source, want := range map[string]string{
		`/banner/.*\.gif`:              "/banner/",
		`^https?://ads\.example\.com/`: "://ads.example.com/",
		`.*`:                           "",
		`(foo)+bar`:                    "foo",
	} {
		p, err := newPattern(source, true)
		if err != nil {
			t.Fatal(err)
		}
		if p.literal != want {
			t.Errorf("literal of %s = %q, want %q", source, p.literal, want)
		}
	}
}

func TestHidingScript(t *testing.T) {
	l := mustParse(t, `[
		{"trigger":{"url-filter":".*"},"action":{"type":"css-display-none","selector":".ad"}},
		{"trigger":{"url-filter":".*"},"action":{"type":"css-display-none","selector":"#banner"}},
		{"trigger":{"url-filter":".*","if-domain":["*news.example"]},"action":{"type":"css-display-none","selector":".promo"}},
		{"trigger":{"url-filter":".*","if-domain":["*shop.example"]},"action":{"type":"ignore-previous-rules"}},
		{"trigger":{"url-filter":"ads"},"action":{"type":"block"}}]`)
	var entries []cosmeticEntry
	if err := json.Unmarshal([]byte(l.cosmetic), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || fmt.Sprint(entries[0].Selectors) != "[.ad #banner]" || entries[0].URL != "" ||
		fmt.Sprint(entries[1].IfDomain) != "[*news.example]" || !entries[2].Ignore {
		t.Errorf("entries = %+v", entries)
	}
	if s := HidingScript([]*List{l}); !strings.Contains(s, `".promo"`) || !strings.HasPrefix(s, "(()=>{") {
		t.Errorf("script = %.200s", s)
	}
	if s := HidingScript([]*List{mustParse(t, `[{"trigger":{"url-filter":"ads"},"action":{"type":"block"}}]`)}); s != "" {
		t.Errorf("a list hiding nothing made a script: %.100s", s)
	}
}

// BenchmarkMatch matches a load against a list of the size of EasyList's
// network rules.
func BenchmarkMatch(b *testing.B) {
	var rules []string
	for i := range 30000 {
		rules = append(rules, fmt.Sprintf(`{"trigger":{"url-filter":"^[^:]+://+([^:/]+\\.)?ads%d\\.example[:/]"},"action":{"type":"block"}}`, i))
		if i%2 == 0 {
			rules = append(rules, fmt.Sprintf(`{"trigger":{"url-filter":"/banner%d/"},"action":{"type":"block"}}`, i))
		}
	}
	l := mustParse(b, "["+strings.Join(rules, ",")+"]")
	req := Request{URL: "https://cdn.news.example/assets/app.js?v=3", TopURL: page, Type: Script}
	b.ResetTimer()
	for b.Loop() {
		Match([]*List{l}, req)
	}
}
