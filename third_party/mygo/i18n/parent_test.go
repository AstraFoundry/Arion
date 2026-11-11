package i18n

import (
	"slices"
	"testing"
)

func TestParent(t *testing.T) {
	for tag, want := range map[string]string{
		"en-AU":              "en-001",
		"en-001":             "en",
		"en-GB":              "en-001",
		"en-CH":              "en-150",
		"en-150":             "en-001",
		"en":                 "",
		"en-US":              "en",
		"de-CH":              "de",
		"zh-Hant-MO":         "zh-Hant-HK",
		"zh-Hant-HK":         "zh-Hant",
		"zh-Hant":            "",
		"zh-Hans":            "zh",
		"zh-Hans-CN":         "zh-Hans",
		"sr-Latn":            "",
		"sr-Latn-RS":         "sr-Latn",
		"sr-Cyrl":            "sr",
		"ja-Latn":            "",
		"nb":                 "no",
		"pt-AO":              "pt-PT",
		"pt-PT":              "pt",
		"es-MX":              "es-419",
		"es-419":             "es",
		"ca-ES-valencia":     "ca-ES",
		"de-DE-u-co-phonebk": "de",
		"en-XA":              "en",
		"ar-XB":              "ar",
		"":                   "",
		"C":                  "",
	} {
		if got := Parent(tag); got != want {
			t.Errorf("Parent(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestChain(t *testing.T) {
	for tag, want := range map[string][]string{
		"en-AU":      {"en-AU", "en-001", "en"},
		"zh-Hant-MO": {"zh-Hant-MO", "zh-Hant-HK", "zh-Hant"},
		"de":         {"de"},
		"x":          nil,
	} {
		if got := chain(tag); !slices.Equal(got, want) {
			t.Errorf("chain(%q) = %q, want %q", tag, got, want)
		}
	}
}
