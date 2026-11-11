package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// icuCase is a case of testdata/format_icu.json, which
// scripts/format-golden.ts writes with Node.js's Intl.
type icuCase struct {
	Kind    string         `json:"kind"`
	Value   string         `json:"value"`
	Time    string         `json:"time"`
	End     string         `json:"end"`
	Zone    string         `json:"zone"`
	Options map[string]any `json:"options"`
	Times   []string       `json:"times"`
}

// knownDifference returns why package i18n writes a case of a locale
// otherwise than ICU 78 (Node.js's), or "" when it should not.
func knownDifference(locale string, c icuCase) string {
	switch {
	case locale == "ht":
		return "ICU 78 has no data of Haitian Creole, which CLDR 48 added, and writes it as English"
	case locale == "el-polyton":
		return "ICU 78 writes polytonic Greek as Greek: it has no data of el-polyton"
	case locale == "sr-Cyrl-ME" && c.Options["timeStyle"] == "full" && c.Zone == "":
		return "ICU 78's sr-Cyrl-ME names UTC in the Latin script; CLDR's names it in Cyrillic"
	case locale == "sr-Cyrl-ME" && c.Options["currencyDisplay"] == "name":
		return "ICU 78's sr-Cyrl-ME names currencies in the Latin script; CLDR's names them in Cyrillic"
	case locale == "my" && c.Options["currencyDisplay"] == "name":
		return "ICU 78's Burmese writes the name of a currency after the amount, as the root's unit pattern does; CLDR's puts it before (\"{1} {0}\")"
	case locale == "fr-ML" && c.Options["dateStyle"] == "short" && c.Options["timeStyle"] != nil,
		locale == "fr-ML" && c.Kind == "range" && c.Options["dateStyle"] == nil && c.Time[:10] != c.End[:10] && c.Options["year"] == nil && (c.Options["hour"] != nil || c.Options["timeStyle"] != nil):
		return "CLDR 48.2 puts fr-ML's short dates and times together with a comma, as ranges of times of different days write them; ICU 78 has CLDR 48.0"
	case (locale == "ps" || locale == "ps-PK") && c.Options["compactDisplay"] == "long":
		return "ICU 78's Pashto writes long compact thousands and millions whole, and trillions in billions (G); CLDR 48.2's are the root's"
	case (locale == "fa" || locale == "fa-AF") && c.Options["timeZoneName"] == "short":
		return "ICU 78 writes Persian times with a zone in parentheses, \"H:mm (z)\", though CLDR's pattern of the skeleton Hmv, which ICU picks for a generic zone (v), has none"
	}
	return ""
}

func TestKnownDifferencesDiffer(t *testing.T) {
	// The test below skips the cases knownDifference names, which must
	// still differ: when ICU or CLDR changes, drop what agrees. Of numbers,
	// some magnitudes differ, not all.
	data, err := os.ReadFile("testdata/format_icu.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Cases   []icuCase           `json:"cases"`
		Locales map[string][]string `json:"locales"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	for l, row := range golden.Locales {
		for i, want := range row {
			c := golden.Cases[i]
			got := formatCase(l, c)
			if c.Kind == "range" {
				got, want = rangeSpaces.Replace(got), rangeSpaces.Replace(want)
			}
			if c.Kind != "number" && knownDifference(l, c) != "" && got == want {
				if l != "ht" && l != "el-polyton" {
					t.Errorf("%s %v agrees with ICU now: %s", l, c, knownDifference(l, c))
				}
			}
		}
	}
}

// icuCaseName names a case in failures.
func (c icuCase) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// numberOptions returns the options of Intl.NumberFormat as package i18n's.
func numberOptions(o map[string]any) NumberOptions {
	var n NumberOptions
	if o["style"] == "percent" {
		n.Percent = true
	}
	if v, ok := o["minimumFractionDigits"].(float64); ok {
		n.MinFractionDigits = int(v)
	}
	if v, ok := o["maximumFractionDigits"].(float64); ok {
		if v == 0 {
			n.Integer = true
		}
		n.MaxFractionDigits = int(v)
	}
	if o["roundingMode"] == "halfEven" {
		n.HalfEven = true
	}
	if o["useGrouping"] == false {
		n.NoGrouping = true
	}
	if o["style"] == "currency" {
		n.Currency = CurrencyOf(o["currency"].(string))
		switch o["currencyDisplay"] {
		case "narrowSymbol":
			n.CurrencyDisplay = CurrencyNarrowSymbol
		case "code":
			n.CurrencyDisplay = CurrencyCode
		case "name":
			n.CurrencyDisplay = CurrencyName
		}
		n.Accounting = o["currencySign"] == "accounting"
	}
	if o["notation"] == "compact" {
		n.Notation = CompactShort
		if o["compactDisplay"] == "long" {
			n.Notation = CompactLong
		}
	}
	return n
}

var styleNames = map[string]Style{"full": FullStyle, "long": LongStyle, "medium": MediumStyle, "short": ShortStyle}

// dateOptions returns the options of Intl.DateTimeFormat as package
// i18n's: their styles, or the skeleton V8 makes of them.
func dateOptions(o map[string]any) DateOptions {
	var d DateOptions
	switch o["hourCycle"] {
	case "h12":
		d.HourCycle = 12
	case "h23":
		d.HourCycle = 24
	}
	if s, ok := o["dateStyle"].(string); ok {
		d.Date = styleNames[s]
	}
	if s, ok := o["timeStyle"].(string); ok {
		d.Time = styleNames[s]
	}
	if d.Date != NoStyle || d.Time != NoStyle {
		return d
	}
	var b strings.Builder
	add := func(key string, forms map[string]string) {
		if v, ok := o[key].(string); ok {
			b.WriteString(forms[v])
		}
	}
	add("weekday", map[string]string{"narrow": "EEEEE", "short": "EEE", "long": "EEEE"})
	add("year", map[string]string{"2-digit": "yy", "numeric": "y"})
	add("month", map[string]string{"2-digit": "MM", "numeric": "M", "narrow": "MMMMM", "short": "MMM", "long": "MMMM"})
	add("day", map[string]string{"2-digit": "dd", "numeric": "d"})
	add("hour", map[string]string{"2-digit": "jj", "numeric": "j"})
	add("minute", map[string]string{"2-digit": "mm", "numeric": "m"})
	add("second", map[string]string{"2-digit": "ss", "numeric": "s"})
	add("timeZoneName", map[string]string{"short": "z", "long": "zzzz"})
	d.Skeleton = b.String()
	return d
}

func zoneOf(name string) *time.Location {
	switch name {
	case "", "UTC":
		return time.UTC
	case "Etc/GMT-8":
		return time.FixedZone(name, 8*3600)
	case "Etc/GMT+5":
		return time.FixedZone(name, -5*3600)
	}
	panic("unknown zone " + name)
}

// formatCase formats a case as package i18n does.
func formatCase(locale string, c icuCase) string {
	switch c.Kind {
	case "number":
		return FormatNumber(locale, c.Value, numberOptions(c.Options))
	case "date":
		t, err := time.Parse(time.RFC3339, c.Time)
		if err != nil {
			panic(err)
		}
		return FormatDate(locale, t.In(zoneOf(c.Zone)), dateOptions(c.Options))
	case "range":
		start, err := time.Parse(time.RFC3339, c.Time)
		if err != nil {
			panic(err)
		}
		end, err := time.Parse(time.RFC3339, c.End)
		if err != nil {
			panic(err)
		}
		return FormatDateRange(locale, start, end, dateOptions(c.Options))
	case "names":
		f := NewDateFormatter(locale, dateOptions(c.Options))
		var out []string
		for _, s := range c.Times {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				panic(err)
			}
			out = append(out, f.Format(t))
		}
		return strings.Join(out, "|")
	case "firstDay":
		return fmt.Sprint(int(FirstWeekday(locale)))
	case "hourCycle":
		return fmt.Sprint(HourCycle(locale))
	}
	return "?"
}

var rangeSpaces = strings.NewReplacer("\u2009", " ", "\u202f", " ")

func TestFormatMatchesICU(t *testing.T) {
	data, err := os.ReadFile("testdata/format_icu.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		ICU      string              `json:"icu"`
		Cases    []icuCase           `json:"cases"`
		Locales  map[string][]string `json:"locales"`
		Autonyms map[string]string   `json:"autonyms"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	locales := make([]string, 0, len(golden.Locales))
	for l := range golden.Locales {
		locales = append(locales, l)
	}
	sort.Strings(locales)
	fails := map[string]int{}
	shown := 0
	for _, l := range locales {
		for i, want := range golden.Locales[l] {
			c := golden.Cases[i]
			got := formatCase(l, c)
			if c.Kind == "range" {
				// V8 keeps CLDR's thin and narrow no-break spaces in
				// ranges, as this does, and JavaScriptCore writes spaces.
				got, want = rangeSpaces.Replace(got), rangeSpaces.Replace(want)
			}
			if got == want {
				continue
			}
			if knownDifference(l, c) != "" {
				continue
			}
			fails[c.Kind]++
			if shown < 400 {
				shown++
				t.Errorf("%s %v:\n got %q\nwant %q", l, c, got, want)
			}
		}
	}
	for tag, want := range golden.Autonyms {
		// ICU 78 has no data of ht and el-polyton (see knownDifference).
		if got := Autonym(tag); got != want && Base(tag) != "ht" && tag != "el-polyton" {
			fails["autonym"]++
			t.Errorf("Autonym(%q) = %q, want %q", tag, got, want)
		}
	}
	if len(fails) > 0 {
		t.Logf("ICU %s; failures by kind: %v", golden.ICU, fails)
	}
}
