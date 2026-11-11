package i18n

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestNewOperands(t *testing.T) {
	for _, c := range []struct {
		in   any
		want Operands
	}{
		{0, Operands{}},
		{1, Operands{I: 1}},
		{-5, Operands{I: 5}},
		{int64(math.MinInt64), Operands{I: 9223372036854775808%big + big}},
		{uint8(7), Operands{I: 7}},
		{uint64(math.MaxUint64), Operands{I: 18446744073709551615%big + big}},
		{1.5, Operands{I: 1, V: 1, W: 1, F: 5, T: 5}},
		{float32(0.25), Operands{V: 2, W: 2, F: 25, T: 25}},
		{1.0, Operands{I: 1}},
		{"1.0", Operands{I: 1, V: 1, F: 0, W: 0, T: 0}},
		{"1.50", Operands{I: 1, V: 2, W: 1, F: 50, T: 5}},
		{"-1.50", Operands{I: 1, V: 2, W: 1, F: 50, T: 5}},
		{"1.03", Operands{I: 1, V: 2, W: 2, F: 3, T: 3}},
		{"1.230", Operands{I: 1, V: 3, W: 2, F: 230, T: 23}},
		{"0.001", Operands{V: 3, W: 3, F: 1, T: 1}},
		{"1200000", Operands{I: 1200000}},
		{"1.2c6", Operands{I: 1200000, C: 6}},
		{"1.2e6", Operands{I: 1200000, C: 6}},
		{"123c6", Operands{I: 123000000, C: 6}},
		{"1.234c2", Operands{I: 123, V: 1, W: 1, F: 4, T: 4, C: 2}},
		{"1c3", Operands{I: 1000, C: 3}},
		{".5", Operands{V: 1, W: 1, F: 5, T: 5}},
		{"1000000000000000000001", Operands{I: 1 + big}},
		{"1.0000000000000000000000", Operands{I: 1, V: 22}},
		{Operands{I: 3}, Operands{I: 3}},
	} {
		got, err := NewOperands(c.in)
		if err != nil {
			t.Errorf("NewOperands(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NewOperands(%#v) = %+v, want %+v", c.in, got, c.want)
		}
	}
	for _, bad := range []any{"", "-", "1.", "1.2.3", "abc", "1e", "1c-2", " ", math.NaN(), math.Inf(1), "1,5", struct{}{}, nil} {
		if o, err := NewOperands(bad); err == nil {
			t.Errorf("NewOperands(%#v) = %+v, want an error", bad, o)
		}
	}
}

func TestParsePlural(t *testing.T) {
	for p := PluralZero; p <= PluralOther; p++ {
		got, ok := ParsePlural(p.String())
		if !ok || got != p {
			t.Errorf("ParsePlural(%q) = %v, %v", p.String(), got, ok)
		}
	}
	if _, ok := ParsePlural("single"); ok {
		t.Error("ParsePlural(single) is a category")
	}
	if Plural(42).String() != "other" {
		t.Error("an unknown Plural is not other")
	}
}

func cats(tag string, ordinal bool) []string {
	var ps []Plural
	if ordinal {
		ps = OrdinalCategories(tag)
	} else {
		ps = PluralCategories(tag)
	}
	var out []string
	for _, p := range ps {
		out = append(out, p.String())
	}
	return out
}

func TestPluralCategories(t *testing.T) {
	for _, c := range []struct {
		tag     string
		ordinal bool
		want    []string
	}{
		{"en", false, []string{"one", "other"}},
		{"en-GB", false, []string{"one", "other"}},
		{"en", true, []string{"one", "two", "few", "other"}},
		{"ja", false, []string{"other"}},
		{"zh-Hant-TW", false, []string{"other"}},
		{"ar", false, []string{"zero", "one", "two", "few", "many", "other"}},
		{"ru", false, []string{"one", "few", "many", "other"}},
		{"fr", false, []string{"one", "many", "other"}},
		{"pt-PT", false, []string{"one", "many", "other"}},
		{"xx", false, []string{"other"}},
		{"", false, []string{"other"}},
		{"de-DE-u-nu-latn", false, []string{"one", "other"}},
	} {
		if got := cats(c.tag, c.ordinal); !slices.Equal(got, c.want) {
			t.Errorf("categories(%q, ordinal %v) = %v, want %v", c.tag, c.ordinal, got, c.want)
		}
	}
}

func category(t *testing.T, tag string, n any, ordinal bool) string {
	t.Helper()
	o, err := NewOperands(n)
	if err != nil {
		t.Fatal(err)
	}
	if ordinal {
		return OrdinalCategory(tag, o).String()
	}
	return PluralCategory(tag, o).String()
}

func TestPluralCategory(t *testing.T) {
	for _, c := range []struct {
		tag     string
		n       any
		ordinal bool
		want    string
	}{
		{"en", 1, false, "one"},
		{"en", "1.0", false, "other"},
		{"en", 2, false, "other"},
		{"en", 0, false, "other"},
		{"en", 1, true, "one"},
		{"en", 22, true, "two"},
		{"en", 113, true, "other"},
		{"en", 103, true, "few"},
		{"fr", 0, false, "one"},
		{"fr", 1.5, false, "one"},
		{"fr", 1000000, false, "many"},
		{"fr", "1c6", false, "many"},
		{"fr", "1.2c6", false, "many"},
		{"fr", "1c3", false, "other"},
		{"ru", 1, false, "one"},
		{"ru", 3, false, "few"},
		{"ru", 5, false, "many"},
		{"ru", 11, false, "many"},
		{"ru", 21, false, "one"},
		{"ru", 1.5, false, "other"},
		{"ar", 0, false, "zero"},
		{"ar", 102, false, "other"},
		{"ar", 103, false, "few"},
		{"ar", 111, false, "many"},
		{"ar", "1.0", false, "one"},
		{"pl", "1000000000000000000021", false, "many"},
		{"lv", "0.1", false, "one"},
		{"lv", "0.10", false, "other"},
		{"lv", 10, false, "zero"},
		{"ja", 1, false, "other"},
	} {
		if got := category(t, c.tag, c.n, c.ordinal); got != c.want {
			t.Errorf("%s %v (ordinal %v) = %s, want %s", c.tag, c.n, c.ordinal, got, c.want)
		}
	}
}

// TestPluralSamples checks the rules against the samples CLDR gives of
// each category.
func TestPluralSamples(t *testing.T) {
	f, err := os.Open("testdata/plural_samples.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	checked := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.SplitN(line, " ", 4)
		if len(fields) != 4 {
			t.Fatalf("bad line %q", line)
		}
		kind, langs, want, samples := fields[0], strings.Split(fields[1], ","), fields[2], fields[3]
		for _, s := range expandSamples(samples) {
			for _, lang := range langs {
				if got := category(t, lang, s, kind == "ordinal"); got != want {
					t.Errorf("%s %s %s = %s, want %s", lang, kind, s, got, want)
				}
				checked++
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if checked < 10000 {
		t.Errorf("checked %d samples only", checked)
	}
}

// expandSamples returns the samples of a rule, as "@integer 0, 2~16, 100,
// … @decimal 0.0~1.5", with each range written out: 2~16 is every
// integer from 2 to 16, 0.0~1.5 every number from 0.0 to 1.5 by tenths.
func expandSamples(s string) []string {
	var out []string
	for _, group := range strings.Split(s, "@") {
		_, list, ok := strings.Cut(strings.TrimSpace(group), " ")
		if !ok {
			continue
		}
		for _, item := range strings.Split(list, ",") {
			item = strings.TrimSpace(item)
			if item == "" || item == "…" {
				continue
			}
			lo, hi, isRange := strings.Cut(item, "~")
			if !isRange {
				out = append(out, item)
				continue
			}
			_, frac, _ := strings.Cut(lo, ".")
			a, _ := strconv.ParseInt(strings.Replace(lo, ".", "", 1), 10, 64)
			z, _ := strconv.ParseInt(strings.Replace(hi, ".", "", 1), 10, 64)
			for n := a; n <= z; n++ {
				s := strconv.FormatInt(n, 10)
				if len(frac) > 0 {
					s = strings.Repeat("0", max(len(frac)+1-len(s), 0)) + s
					s = s[:len(s)-len(frac)] + "." + s[len(s)-len(frac):]
				}
				out = append(out, s)
			}
		}
	}
	return out
}

// TestPluralsMatchICU checks the rules against ICU's, as Bun's
// Intl.PluralRules gives them (scripts/plurals-golden.ts).
func TestPluralsMatchICU(t *testing.T) {
	data, err := os.ReadFile("testdata/plurals_icu.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		ICU      string
		Numbers  []string
		Cardinal map[string]string
		Ordinal  map[string]string
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	initials := map[byte]string{'z': "zero", 'o': "one", 't': "two", 'f': "few", 'm': "many", 'x': "other"}
	langs := 0
	for _, kind := range []struct {
		ordinal bool
		langs   map[string]string
	}{{false, golden.Cardinal}, {true, golden.Ordinal}} {
		for lang, want := range kind.langs {
			langs++
			if len(want) != len(golden.Numbers) {
				t.Fatalf("%s: %d categories for %d numbers", lang, len(want), len(golden.Numbers))
			}
			for i, n := range golden.Numbers {
				if got := category(t, lang, n, kind.ordinal); got != initials[want[i]] {
					t.Errorf("%s %s (ordinal %v) = %s, ICU %s says %s", lang, n, kind.ordinal, got, golden.ICU, initials[want[i]])
				}
			}
		}
	}
	if langs < 250 {
		t.Errorf("compared %d languages only", langs)
	}
}
