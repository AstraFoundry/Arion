package i18n

import (
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFormatNumber(t *testing.T) {
	for _, c := range []struct {
		locale string
		n      any
		o      NumberOptions
		want   string
	}{
		{"en", 1234567.891, NumberOptions{}, "1,234,567.891"},
		{"en", int64(-42), NumberOptions{}, "-42"},
		{"en", uint8(7), NumberOptions{}, "7"},
		{"en", "12345678901234567890.25", NumberOptions{}, "12,345,678,901,234,567,890.25"},
		{"en", 1.005, NumberOptions{MaxFractionDigits: 2}, "1.01"},
		{"en", 2.5, NumberOptions{Integer: true}, "3"},
		{"en", 2.5, NumberOptions{Integer: true, HalfEven: true}, "2"},
		{"en", 3.5, NumberOptions{Integer: true, HalfEven: true}, "4"},
		{"en", 0.125, NumberOptions{MaxFractionDigits: 2, HalfEven: true}, "0.12"},
		{"en", 1.5, NumberOptions{MinFractionDigits: 2}, "1.50"},
		{"en", 9.9999, NumberOptions{}, "10"},
		{"en", -0.0001, NumberOptions{}, "-0"},
		{"en", 0.256, NumberOptions{Percent: true}, "26%"},
		{"en", 12345, NumberOptions{NoGrouping: true}, "12345"},
		{"en", 7, NumberOptions{MinIntegerDigits: 2}, "07"},
		{"fa", 7, NumberOptions{MinIntegerDigits: 2}, "۰۷"},
		{"en", math.NaN(), NumberOptions{}, "NaN"},
		{"en", math.NaN(), NumberOptions{Percent: true}, "NaN%"},
		{"de", math.NaN(), NumberOptions{Percent: true}, "NaN\u00a0%"},
		{"tr", math.NaN(), NumberOptions{Percent: true}, "%NaN"},
		{"tr", math.Inf(-1), NumberOptions{Percent: true}, "-%∞"},
		{"en", math.Inf(-1), NumberOptions{}, "-∞"},
		{"en", "lots", NumberOptions{}, "lots"},
		{"de", 1234.5, NumberOptions{}, "1.234,5"},
		{"de-CH", 1234.5, NumberOptions{}, "1'234.5"},
		{"fr", 1234.5, NumberOptions{}, "1 234,5"},
		{"es", 1234, NumberOptions{}, "1234"},
		{"es", 12345, NumberOptions{}, "12.345"},
		{"hi", 12345678, NumberOptions{}, "1,23,45,678"},
		{"en-IN", 12345678, NumberOptions{}, "1,23,45,678"},
		{"fa", -1234.5, NumberOptions{}, "‎−۱٬۲۳۴٫۵"},
		{"fa-u-nu-latn", 1234.5, NumberOptions{}, "1,234.5"},
		{"ar-EG", 12, NumberOptions{}, "١٢"},
		{"bn", 0.5, NumberOptions{Percent: true}, "৫০%"},
		{"tr", 0.25, NumberOptions{Percent: true}, "%25"},
		{"en-XA", 1234.5, NumberOptions{}, "1,234.5"},
		{"xx", 1234.5, NumberOptions{}, "1,234.5"},
	} {
		if got := FormatNumber(c.locale, c.n, c.o); got != c.want {
			t.Errorf("FormatNumber(%q, %v, %+v) = %q, want %q", c.locale, c.n, c.o, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	day := time.Date(2026, 1, 5, 13, 7, 9, 0, time.UTC)
	for _, c := range []struct {
		locale string
		o      DateOptions
		want   string
	}{
		{"en", DateOptions{}, "1/5/2026"},
		{"en", DateOptions{Date: FullStyle}, "Monday, January 5, 2026"},
		{"en", DateOptions{Time: ShortStyle}, "1:07 PM"},
		{"en", DateOptions{Time: ShortStyle, HourCycle: 24}, "13:07"},
		{"en-GB", DateOptions{Time: MediumStyle, HourCycle: 12}, "01:07:09 pm"},
		{"en", DateOptions{Skeleton: "yMMMM"}, "January 2026"},
		{"ru", DateOptions{Skeleton: "yMMMM"}, "январь 2026 г."},
		{"ru", DateOptions{Skeleton: "MMMMd"}, "5 января"},
		{"zh-Hans", DateOptions{Skeleton: "yMMMM"}, "2026年1月"},
		{"en", DateOptions{Skeleton: "jm"}, "1:07 PM"},
		{"de", DateOptions{Skeleton: "jm"}, "13:07"},
		{"de-u-hc-h12", DateOptions{Skeleton: "jm"}, "1:07 PM"},
		{"zh-Hant", DateOptions{Time: ShortStyle}, "下午1:07"},
		{"ja", DateOptions{Skeleton: "Ed"}, "5日(月)"},
		{"fa", DateOptions{Date: ShortStyle}, "۲۰۲۶/۱/۵"},
		{"en", DateOptions{Date: LongStyle, Time: LongStyle}, "January 5, 2026 at 1:07:09 PM UTC"},
		{"ar-XB", DateOptions{Skeleton: "MMMd"}, "Jan 5"},
		{"en", DateOptions{Skeleton: "EEEEEEd"}, "5 Mo"},
	} {
		if got := FormatDate(c.locale, day, c.o); got != c.want {
			t.Errorf("FormatDate(%q, %+v) = %q, want %q", c.locale, c.o, got, c.want)
		}
	}
	// An hour letter of a skeleton is its hour cycle, as V8 (Node 26)
	// writes them.
	half := time.Date(2026, 1, 5, 0, 30, 0, 0, time.UTC)
	for _, c := range []struct{ locale, skeleton, want string }{
		{"ja", "hm", "午前12:30"},
		{"en", "Kms", "0:30:00 AM"},
		{"en", "km", "24:30"},
		{"de", "hm", "12:30 AM"},
		{"en", "Hm", "00:30"},
	} {
		if got := FormatDate(c.locale, half, DateOptions{Skeleton: c.skeleton}); got != c.want {
			t.Errorf("FormatDate(%q, %q) = %q, want %q", c.locale, c.skeleton, got, c.want)
		}
	}
	shanghai := time.FixedZone("CST", 8*3600)
	if got := FormatDate("en", day.In(shanghai), DateOptions{Time: FullStyle}); got != "9:07:09 PM GMT+08:00" {
		t.Errorf("a time in UTC+8 is %q", got)
	}
	if got := NewDateFormatter("en", DateOptions{Skeleton: "yMMMd"}).Pattern(); got != "MMM d, y" {
		t.Errorf("the pattern of yMMMd is %q", got)
	}
}

func TestNames(t *testing.T) {
	if m := Months("ru", Wide, true); m[0] != "январь" {
		t.Errorf("Russian's January alone is %q", m[0])
	}
	if m := Months("ru", Wide, false); m[0] != "января" {
		t.Errorf("Russian's January in a date is %q", m[0])
	}
	if m := Months("en", Narrow, true); strings.Join(m[:], "") != "JFMAMJJASOND" {
		t.Errorf("narrow months: %v", m)
	}
	if w := Weekdays("en", Short, true); w[time.Monday] != "Mo" {
		t.Errorf("short Monday is %q", w[time.Monday])
	}
	if w := Weekdays("zh-Hans", Narrow, true); w[time.Sunday] != "日" {
		t.Errorf("Chinese narrow Sunday is %q", w[time.Sunday])
	}
	if am, pm := DayPeriods("en", Abbreviated); am != "AM" || pm != "PM" {
		t.Errorf("DayPeriods = %q %q", am, pm)
	}
}

func TestRegionData(t *testing.T) {
	for _, c := range []struct {
		locale string
		first  time.Weekday
		hours  int
	}{
		{"en", time.Sunday, 12},
		{"en-US", time.Sunday, 12},
		{"en-GB", time.Monday, 24},
		{"de", time.Monday, 24},
		{"ar-EG", time.Saturday, 12},
		{"zh-TW", time.Sunday, 12},
		{"zh-Hant", time.Sunday, 12},
		{"zh", time.Monday, 24},
		{"und-MV", time.Friday, 24},
		{"en-US-u-fw-mon-hc-h23", time.Monday, 24},
		{"de-u-hc-h12", time.Monday, 12},
	} {
		if got := FirstWeekday(c.locale); got != c.first {
			t.Errorf("FirstWeekday(%q) = %v, want %v", c.locale, got, c.first)
		}
		if got := HourCycle(c.locale); got != c.hours {
			t.Errorf("HourCycle(%q) = %d, want %d", c.locale, got, c.hours)
		}
	}
	if r := RegionFor("en-GB"); r != (Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24}) {
		t.Errorf("RegionFor(en-GB) = %+v", r)
	}
}

func TestAutonym(t *testing.T) {
	for tag, want := range map[string]string{
		"en": "English", "en-GB": "British English", "en-US": "American English", "de": "Deutsch",
		"zh-Hans": "简体中文", "zh-Hant": "繁體中文", "zh-TW": "繁體中文", "ja": "日本語", "ar": "العربية",
		"pt-BR": "português (Brasil)", "es-419": "español latinoamericano", "en-XA": "en-XA", "tlh": "",
	} {
		if got := Autonym(tag); got != want {
			t.Errorf("Autonym(%q) = %q, want %q", tag, got, want)
		}
	}
	if l := FormatLocales(); len(l) < 400 || l[0] != "af" {
		t.Errorf("FormatLocales() = %d locales from %v", len(l), l[:1])
	}
}

func BenchmarkFormatNumber(b *testing.B) {
	FormatNumber("de", 1234.5, NumberOptions{})
	b.ReportAllocs()
	for b.Loop() {
		FormatNumber("de", 1234.5, NumberOptions{})
	}
}

func BenchmarkFormatDate(b *testing.B) {
	day := time.Date(2026, 1, 5, 13, 7, 9, 0, time.UTC)
	FormatDate("de", day, DateOptions{Skeleton: "yMMMM"})
	b.ReportAllocs()
	for b.Loop() {
		FormatDate("de", day, DateOptions{Skeleton: "yMMMM"})
	}
}

func BenchmarkMonths(b *testing.B) {
	Months("de", Wide, true)
	b.ReportAllocs()
	for b.Loop() {
		Months("de", Wide, true)
	}
}

func BenchmarkLoadTables(b *testing.B) {
	for b.Loop() {
		tables.once, tables.datesOnce = sync.Once{}, sync.Once{}
		loadDates()
	}
}

func TestFormatLocale(t *testing.T) {
	for _, c := range []struct {
		r        Region
		language string
		want     string
	}{
		{Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24}, "en", "en-GB-u-fw-mon-hc-h23"},
		{Region{Locale: "de-DE", FirstWeekday: time.Monday, HourCycle: 24}, "en", "en-DE-u-fw-mon-hc-h23"},
		{Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12}, "zh-Hant", "zh-Hant-US-u-fw-sun-hc-h12"},
		{Region{}, "de", "de"},
		{Region{Locale: "de-DE", FirstWeekday: time.Monday, HourCycle: 24}, "zh-HK", "zh-Hant-DE-u-fw-mon-hc-h23"},
		{Region{Locale: "fr-FR", FirstWeekday: time.Monday, HourCycle: 24}, "zh-TW", "zh-Hant-FR-u-fw-mon-hc-h23"},
		{Region{Locale: "zh-TW", FirstWeekday: time.Sunday, HourCycle: 12}, "zh", "zh-Hans-TW-u-fw-sun-hc-h12"},
		{Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12}, "sr-ME", "sr-Latn-US-u-fw-sun-hc-h12"},
		{Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12}, "de-AT", "de-US-u-fw-sun-hc-h12"},
		{Region{Locale: "fr"}, "fr-CA", "fr-CA"},
		// Pseudo locales write the formats in English, as Months does.
		{Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12}, "ar-XB", "en-US-u-fw-sun-hc-h12"},
		{Region{}, "en-XA", "en"},
	} {
		if got := c.r.FormatLocale(c.language); got != c.want {
			t.Errorf("%+v.FormatLocale(%q) = %q, want %q", c.r, c.language, got, c.want)
		}
	}
	if got := FormatDate((Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24}).FormatLocale("en"), time.Date(2026, 1, 5, 13, 7, 0, 0, time.UTC), DateOptions{Date: ShortStyle, Time: ShortStyle}); got != "05/01/2026, 13:07" {
		t.Errorf("English with British formats: %q", got)
	}
	if got := Weekdays((Region{Locale: "de-DE", FirstWeekday: time.Monday, HourCycle: 24}).FormatLocale("zh-HK"), Abbreviated, true)[time.Monday]; got != "週一" {
		t.Errorf("Traditional Chinese with German formats says Monday %q", got)
	}
}

func TestPeriodRulesCrossMidnight(t *testing.T) {
	// morning1 06:00-12:00, afternoon1 12:00-18:00, evening1 18:00-21:00,
	// night1 21:00-06:00, as CLDR writes the rules of English.
	r := parsePeriodRules("4:360-720 6:720-1080 8:1080-1260 10:1260-360 3@720")
	want := [24]int{10, 10, 10, 10, 10, 10, 4, 4, 4, 4, 4, 4, 6, 6, 6, 6, 6, 6, 8, 8, 8, 10, 10, 10}
	if r.byHour != want || !r.noon {
		t.Errorf("rules by hour %v, noon %v", r.byHour, r.noon)
	}
}
