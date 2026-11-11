package i18n

import "testing"

func TestCompactNumbers(t *testing.T) {
	for _, c := range []struct {
		locale string
		n      any
		o      NumberOptions
		want   string
	}{
		{"en", 1234, NumberOptions{Notation: CompactShort}, "1.2K"},
		{"en", 12345, NumberOptions{Notation: CompactShort}, "12K"},
		{"en", 999999, NumberOptions{Notation: CompactShort}, "1M"}, // rounds into millions
		{"en", 999.9, NumberOptions{Notation: CompactShort}, "1K"},
		{"en", 1e15, NumberOptions{Notation: CompactShort}, "1000T"}, // trillions the largest
		{"en", 0.001234, NumberOptions{Notation: CompactShort}, "0.0012"},
		{"en", -1234, NumberOptions{Notation: CompactShort}, "-1.2K"},
		{"en", 1234, NumberOptions{Notation: CompactLong}, "1.2 thousand"},
		{"en", "1234567", NumberOptions{Notation: CompactShort, MaxFractionDigits: 2}, "1.23M"},
		{"en", 1000, NumberOptions{Notation: CompactShort, MinFractionDigits: 2}, "1.00K"},
		{"en", 1500, NumberOptions{Notation: CompactShort, Integer: true}, "2K"},
		{"en", 2500, NumberOptions{Notation: CompactShort, Integer: true, HalfEven: true}, "2K"},
		{"en", 123.45, NumberOptions{Notation: CompactShort, Percent: true}, "12K%"},
		{"tr", -0.5, NumberOptions{Notation: CompactShort, Percent: true}, "%-50"},
		{"fr", 1000, NumberOptions{Notation: CompactLong}, "mille"}, // the form of exactly 1
		{"fr", 1234, NumberOptions{Notation: CompactLong}, "1,2 millier"},
		{"fr", 2000000, NumberOptions{Notation: CompactLong}, "2 millions"},
		{"fr", -1000, NumberOptions{Notation: CompactLong}, "-1 millier"},
		{"ja", 12345, NumberOptions{Notation: CompactShort}, "1.2万"},
		{"de", 12345, NumberOptions{Notation: CompactShort}, "12.345"}, // no compact thousands; grouped from five digits
		{"sw", -1234, NumberOptions{Notation: CompactShort}, "elfu\u00a0-1.2"},
		{"hi", 123456, NumberOptions{Notation: CompactShort}, "1.2\u00a0लाख"},
		{"ar-EG", 1234, NumberOptions{Notation: CompactShort}, "١٫٢\u00a0ألف"},
		{"ar-EG-u-nu-latn", 1234, NumberOptions{Notation: CompactShort}, "1.2\u00a0ألف"},
	} {
		if got := FormatNumber(c.locale, c.n, c.o); got != c.want {
			t.Errorf("FormatNumber(%q, %v, %+v) = %q, want %q", c.locale, c.n, c.o, got, c.want)
		}
	}
}

func TestMaxFractionDigitsDefault(t *testing.T) {
	// MaxFractionDigits 0 is 3, or MinFractionDigits when more, as Intl's
	// maximumFractionDigits: "1,234.568", not "1,234.57".
	if got := FormatNumber("en", 1234.5678, NumberOptions{MinFractionDigits: 2}); got != "1,234.568" {
		t.Errorf("got %q", got)
	}
}
