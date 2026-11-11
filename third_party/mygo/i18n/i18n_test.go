package i18n

import (
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"en-us":           "en-US",
		"zh_CN.UTF-8":     "zh-CN",
		"zh_TW":           "zh-TW",
		"sr_RS@latin":     "sr-Latn-RS",
		"sr_RS@euro":      "sr-RS",
		"de_DE.utf8@euro": "de-DE",
		"iw":              "he",
		"in":              "id",
		"nb":              "nb",
		"zh-Hant-HK":      "zh-Hant-HK",
		"es-419":          "es-419",
		"C":               "",
		"POSIX":           "",
		"C.UTF-8":         "",
		"":                "",
		"und":             "",
		"x":               "",
		" en ":            "en",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	for _, c := range []struct {
		preferred, supported []string
		want                 string
	}{
		{[]string{"zh-TW"}, []string{"en", "zh-Hans", "zh-Hant"}, "zh-Hant"},
		{[]string{"zh-HK"}, []string{"en", "zh-TW"}, "zh-TW"},
		{[]string{"zh-TW"}, []string{"en", "zh-Hans"}, ""},
		{[]string{"zh"}, []string{"en", "zh-Hant", "zh-Hans"}, "zh-Hans"},
		{[]string{"zh_CN.UTF-8"}, []string{"en", "zh-Hans"}, "zh-Hans"},
		{[]string{"es-MX"}, []string{"en", "es-ES"}, "es-ES"},
		{[]string{"es-MX"}, []string{"en", "es-ES", "es-419"}, "es-419"},
		{[]string{"pt-PT"}, []string{"en", "pt-BR"}, "pt-BR"},
		{[]string{"en-AU"}, []string{"zh-Hans", "en-US", "en-GB"}, "en-GB"},
		{[]string{"fr-CA", "de"}, []string{"en", "de", "fr"}, "fr"},
		{[]string{"fr-CA", "de"}, []string{"en", "de"}, "de"},
		{[]string{"ja"}, []string{"en", "zh-Hans"}, ""},
		{[]string{"ja", "zh-Hans"}, []string{"en", "zh-Hans"}, "zh-Hans"},
		{[]string{"nb-NO"}, []string{"en", "no"}, "no"},
		{[]string{"no"}, []string{"en", "nb"}, "nb"},
		{[]string{"iw"}, []string{"en", "he"}, "he"},
		{[]string{"sr-Cyrl"}, []string{"en", "sr-Latn"}, ""},
		{[]string{"sr_RS@latin"}, []string{"en", "sr-Latn"}, "sr-Latn"},
		{[]string{"en-XA"}, []string{"en", "en-XA"}, "en-XA"},
		{[]string{"C"}, []string{"en"}, ""},
		{nil, []string{"en"}, ""},
		{[]string{"en"}, nil, ""},
		// The tag as the app spells it.
		{[]string{"en-gb"}, []string{"zh-hans", "en-gb"}, "en-gb"},
	} {
		if got := Match(c.preferred, c.supported); got != c.want {
			t.Errorf("Match(%q, %q) = %q, want %q", c.preferred, c.supported, got, c.want)
		}
	}
}

func TestDirection(t *testing.T) {
	var rtl []string
	for _, tag := range []string{"en", "ar", "ar-EG", "he", "iw", "fa", "ur", "ps", "yi", "dv", "ug", "ckb", "zh", "ja", "az-Arab", "az", "ku", "ar-XB", "en-XA", "", "C", "pa-Arab", "nqo"} {
		if Direction(tag) == RightToLeft {
			rtl = append(rtl, tag)
		}
	}
	want := []string{"ar", "ar-EG", "he", "iw", "fa", "ur", "ps", "yi", "dv", "ug", "ckb", "az-Arab", "ar-XB", "pa-Arab", "nqo"}
	if !slices.Equal(rtl, want) {
		t.Errorf("right to left: %q, want %q", rtl, want)
	}
	if RightToLeft.String() != "rtl" || LeftToRight.String() != "ltr" {
		t.Error("Dir.String")
	}
}

func TestBase(t *testing.T) {
	for in, want := range map[string]string{"zh-Hant-TW": "zh", "iw": "he", "en": "en", "": "", "sr_RS@latin": "sr"} {
		if got := Base(in); got != want {
			t.Errorf("Base(%q) = %q, want %q", in, got, want)
		}
	}
}
