package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/internal/platform"

	// MyGo's own strings in the languages the tests show.
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
)

// TestControlsSayTheLanguage checks that the controls say MyGo's own
// strings in the window's language, assistive technology's included, and
// that saying them allocates nothing frame after frame.
func TestControlsSayTheLanguage(t *testing.T) {
	query, open, current := "x", true, 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreFindBar(c, &open, &query, 0, &current)
			coreSearchField(c, new(string))
		})
	}, 600, 200)
	if !tt.HasText("No matches") || !tt.HasText("Done") {
		t.Fatalf("in English: %q", tt.Texts())
	}
	for _, lang := range []string{"ja", "zh-Hans", "zh-Hant"} {
		tt.SetLocale(lang)
		tx := l10n.In(lang)
		if !tt.HasText(tx.NoMatches()) || !tt.HasText(tx.Done()) || tt.HasText("Done") {
			t.Errorf("in %s: %q", lang, tt.Texts())
		}
		tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
		accessNode(t, tt.h.access, platform.RoleToolbar, tx.Find())
		accessNode(t, tt.h.access, platform.RoleButton, tx.Previous())
	}
	var c context
	c.rt = &engine{lang: "zh-Hans"}
	c.texts().Search()
	if allocs := testing.AllocsPerRun(100, func() { _ = c.texts().Search() }); allocs != 0 {
		t.Errorf("a control's string allocates %v times a frame", allocs)
	}
}

func BenchmarkControlText(b *testing.B) {
	var c context
	c.rt = &engine{lang: "zh-Hans"}
	c.texts().Search()
	b.ReportAllocs()
	for b.Loop() {
		_ = c.texts().Search()
	}
}
