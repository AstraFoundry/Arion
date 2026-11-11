//go:build !js

package mygo

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"
)

const blockAds = `[{"trigger":{"url-filter":"ads"},"action":{"type":"block"}}]`

func compileRules(t *testing.T, id, rules string) *ContentRules {
	t.Helper()
	r, err := CompileContentRules(id, rules)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func lists(fw *fake.Window) []string {
	var out []string
	for _, l := range fw.ContentRules() {
		out = append(out, l.Identifier)
	}
	return out
}

func TestContentRules(t *testing.T) {
	ads := compileRules(t, "ads", blockAds)
	hide := compileRules(t, "hide", `[{"trigger":{"url-filter":".*"},"action":{"type":"css-display-none","selector":".ad"}}]`)
	if ads.ID() != "ads" {
		t.Errorf("ID = %q", ads.ID())
	}
	adsList := platform.ContentRulesIdentifier("ads", blockAds)

	w, fw := testWindow(t, WindowOptions{URL: "https://example.com/", Page: PageOptions{ContentRules: []*ContentRules{ads}}})
	if got := lists(fw); !slices.Equal(got, []string{adsList}) {
		t.Fatalf("a window's lists = %v, want PageOptions.ContentRules", got)
	}
	w.Page().SetContentRules(hide, nil, ads)
	if got := lists(fw); len(got) != 2 || got[1] != adsList || !strings.HasPrefix(got[0], "hide-") {
		t.Errorf("lists after SetContentRules = %v", got)
	}
	w.Page().SetContentRules()
	if got := lists(fw); len(got) != 0 {
		t.Errorf("SetContentRules() left %v", got)
	}

	// A window without rules is left alone until it gets some.
	_, plain := testWindow(t, WindowOptions{})
	if n := onMainValue(func() int { return plain.ContentRuleUpdates }); n != 0 {
		t.Errorf("a window without rules had them set %d times", n)
	}
}

func TestContentRulesWebView(t *testing.T) {
	ads := compileRules(t, "ads", blockAds)
	w, _, s := contentWindow(t, func(c *ui.Context) {})
	v, err := w.NewWebView(WebViewOptions{Page: PageOptions{ContentRules: []*ContentRules{ads}}})
	if err != nil {
		t.Fatal(err)
	}
	fv := s.WebViews()[0]
	if got := lists(fv.Window); len(got) != 1 {
		t.Fatalf("a web view's lists = %v, want PageOptions.ContentRules", got)
	}
	v.Page().SetContentRules()
	if got := lists(fv.Window); len(got) != 0 {
		t.Errorf("the web view kept %v", got)
	}
	if w.Page() != nil {
		t.Fatal("a window of native UI has a page")
	}
}

// The same rules load what was compiled; other rules of the same id
// compile again and replace the earlier list.
func TestContentRulesKept(t *testing.T) {
	before := onMainValue(func() int { return fb.ContentRuleCompiles })
	compileRules(t, "kept", blockAds)
	compileRules(t, "kept", blockAds)
	if n := onMainValue(func() int { return fb.ContentRuleCompiles }) - before; n != 1 {
		t.Errorf("the same rules compiled %d times, want once", n)
	}
	other := `[{"trigger":{"url-filter":"trackers"},"action":{"type":"block"}}]`
	compileRules(t, "kept", other)
	compileRules(t, "kept-more", blockAds)
	var kept []string
	for _, name := range fb.ContentRuleLists() {
		if strings.HasPrefix(name, "kept-") {
			kept = append(kept, name)
		}
	}
	slices.Sort(kept)
	want := []string{platform.ContentRulesIdentifier("kept", other), platform.ContentRulesIdentifier("kept-more", blockAds)}
	slices.Sort(want)
	if !slices.Equal(kept, want) {
		t.Errorf("lists kept = %v, want %v", kept, want)
	}
}

func TestContentRulesErrors(t *testing.T) {
	for _, id := range []string{"", "a/b", "spaced id", strings.Repeat("x", 65)} {
		if _, err := CompileContentRules(id, blockAds); err == nil {
			t.Errorf("id %q was taken", id)
		}
	}
	if _, err := CompileContentRules("bad", `{"trigger":{}}`); err == nil || !strings.Contains(err.Error(), "content rules bad") {
		t.Errorf("rules that are not an array: %v", err)
	}
}

func TestStaleContentRules(t *testing.T) {
	current := platform.ContentRulesIdentifier("ads", "[1]")
	for name, want := range map[string]bool{
		platform.ContentRulesIdentifier("ads", "[2]"): true,
		current: false,
		platform.ContentRulesIdentifier("ads-more", "[2]"): false,
		"ads-notahash0000000":                              false,
		"ads":                                              false,
	} {
		if got := platform.StaleContentRules("ads", current, name); got != want {
			t.Errorf("StaleContentRules(%q) = %v, want %v", name, got, want)
		}
	}
}
