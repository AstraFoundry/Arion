package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/internal/text"
)

// TestLang holds the language of text to the language the app shows,
// which an element's Lang overrides for its text and that of the elements
// inside it, and a span's for its runes.
func TestLang(t *testing.T) {
	var plain, quote, inner, reset text.Params
	var spans []text.Span
	var locale string
	var region i18n.Region
	tt := coreNewTester(func(c *context) {
		locale, region = c.Locale(), c.Region()
		coreColumn(c).Children(func() {
			plain = coreText(c, "直").textParams(0)
			coreColumn(c).Lang("ja").Children(func() {
				quote = coreText(c, "直").textParams(0)
				inner = coreText(c, "直").Lang("zh-Hant").textParams(0)
				reset = coreText(c, "直").Lang("").textParams(0)
			})
			rich := coreRichText(c, Span{Text: "直"}, Span{Text: "骨", Lang: "ja"})
			spans = text.DecodeSpans(rich.textParams(0).Spans)
		})
	}, 200, 200)
	if plain.Style.Lang != "" || locale != "" {
		t.Errorf("a Tester shows %q, text in %q: want none", locale, plain.Style.Lang)
	}
	tt.SetLocale("zh-Hans")
	tt.SetRegion(i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24})
	if locale != "zh-Hans" || region.Locale != "en-GB" {
		t.Errorf("the view sees %q %+v", locale, region)
	}
	for _, c := range []struct {
		name string
		p    text.Params
		want string
	}{{"plain", plain, "zh-Hans"}, {"quote", quote, "ja"}, {"inner", inner, "zh-Hant"}, {"reset", reset, "ja"}} {
		if c.p.Style.Lang != c.want {
			t.Errorf("%s text is in %q, want %q", c.name, c.p.Style.Lang, c.want)
		}
	}
	if len(spans) != 2 || spans[0].Lang != "" || spans[1].Lang != "ja" {
		t.Errorf("the spans are %+v", spans)
	}
}
