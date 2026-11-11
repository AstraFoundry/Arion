package i18n

import (
	"math"
	"testing"
)

// TestNumberSignDisplay checks NumberOptions.SignDisplay against
// Intl.NumberFormat's signDisplay, which Node (V8, ICU) writes as the
// comments say.
func TestNumberSignDisplay(t *testing.T) {
	usd, eur, jpy := CurrencyOf("USD"), CurrencyOf("EUR"), CurrencyOf("JPY")
	chf, ils := CurrencyOf("CHF"), CurrencyOf("ILS")
	negZero := math.Copysign(0, -1)
	for _, c := range []struct {
		locale string
		n      any
		o      NumberOptions
		want   string
	}{
		// SignAuto, the default.
		{"en", 1234, NumberOptions{}, "1,234"},
		{"en", -1234, NumberOptions{}, "-1,234"},
		{"en", negZero, NumberOptions{}, "-0"},

		// SignAlways: a plus before a number that is not negative, zero
		// included; a negative zero keeps its minus.
		{"en", 1234, NumberOptions{SignDisplay: SignAlways}, "+1,234"},
		{"en", -1234, NumberOptions{SignDisplay: SignAlways}, "-1,234"},
		{"en", 0, NumberOptions{SignDisplay: SignAlways}, "+0"},
		{"en", negZero, NumberOptions{SignDisplay: SignAlways}, "-0"},
		{"de", 1234, NumberOptions{SignDisplay: SignAlways}, "+1.234"},
		{"hi", 123456, NumberOptions{SignDisplay: SignAlways}, "+1,23,456"},
		{"ar", 1234, NumberOptions{SignDisplay: SignAlways}, "‎+1,234"},
		{"fa", 1234, NumberOptions{SignDisplay: SignAlways}, "‎+۱٬۲۳۴"},
		{"en", math.NaN(), NumberOptions{SignDisplay: SignAlways}, "+NaN"},
		{"en", math.Inf(1), NumberOptions{SignDisplay: SignAlways}, "+∞"},
		{"en", math.Inf(-1), NumberOptions{SignDisplay: SignAlways}, "-∞"},

		// SignExceptZero: no plus before a number that rounds to zero.
		{"en", 1234, NumberOptions{SignDisplay: SignExceptZero}, "+1,234"},
		{"en", 0, NumberOptions{SignDisplay: SignExceptZero}, "0"},
		{"en", negZero, NumberOptions{SignDisplay: SignExceptZero}, "0"},
		{"en", -0.4, NumberOptions{SignDisplay: SignExceptZero, Integer: true}, "0"},
		{"en", 0.001, NumberOptions{SignDisplay: SignExceptZero, MaxFractionDigits: 2}, "0"},
		{"en", -0.4, NumberOptions{SignDisplay: SignExceptZero}, "-0.4"},

		// SignNegative: as SignAuto, but no minus before a negative zero.
		{"en", 1234, NumberOptions{SignDisplay: SignNegative}, "1,234"},
		{"en", -1234, NumberOptions{SignDisplay: SignNegative}, "-1,234"},
		{"en", negZero, NumberOptions{SignDisplay: SignNegative}, "0"},

		// SignNever: no sign at all.
		{"en", 1234, NumberOptions{SignDisplay: SignNever}, "1,234"},
		{"en", -1234, NumberOptions{SignDisplay: SignNever}, "1,234"},
		{"en", negZero, NumberOptions{SignDisplay: SignNever}, "0"},
		{"en", math.Inf(-1), NumberOptions{SignDisplay: SignNever}, "∞"},

		// Percents, where the plus goes where the minus does: before the
		// percent sign where the locale puts it first.
		{"en", 0.25, NumberOptions{SignDisplay: SignAlways, Percent: true}, "+25%"},
		{"de", 0.25, NumberOptions{SignDisplay: SignAlways, Percent: true}, "+25 %"},
		{"tr", 0.25, NumberOptions{SignDisplay: SignAlways, Percent: true}, "+%25"},
		{"en", 0.25, NumberOptions{SignDisplay: SignExceptZero, Percent: true}, "+25%"},
		{"en", 0.001, NumberOptions{SignDisplay: SignExceptZero, Percent: true}, "0%"},
		{"en", math.NaN(), NumberOptions{SignDisplay: SignAlways, Percent: true}, "+NaN%"},

		// Amounts, in the pattern of the locale, accounting included.
		{"en", 1234, NumberOptions{SignDisplay: SignAlways, Currency: usd}, "+$1,234.00"},
		{"en", -1234, NumberOptions{SignDisplay: SignAlways, Currency: usd}, "-$1,234.00"},
		{"en", -1234, NumberOptions{SignDisplay: SignAlways, Currency: usd, Accounting: true}, "($1,234.00)"},
		{"en", 1234, NumberOptions{SignDisplay: SignAlways, Currency: usd, Accounting: true}, "+$1,234.00"},
		{"de", 1234, NumberOptions{SignDisplay: SignAlways, Currency: eur}, "+1.234,00 €"},
		{"ja", 1234, NumberOptions{SignDisplay: SignAlways, Currency: jpy}, "+￥1,234"},
		// The plus takes the minus's place, not the prefix's: Dutch writes
		// "€ -1.234,00" and so "€ +1.234,00" (with a no-break space), Swiss German "CHF-1'234.00"
		// and so "CHF+1'234.00", Hebrew "‏‎-1,234.00 ‏₪" and so
		// "‏‎+1,234.00 ‏₪".
		{"nl", 1234, NumberOptions{SignDisplay: SignAlways, Currency: eur}, "€\u00a0+1.234,00"},
		{"de-CH", 1234, NumberOptions{SignDisplay: SignAlways, Currency: chf}, "CHF+1'234.00"},
		{"he", 1234, NumberOptions{SignDisplay: SignAlways, Currency: ils}, "\u200f\u200e+1,234.00\u00a0\u200f\u20aa"},
		{"en", math.NaN(), NumberOptions{SignDisplay: SignAlways, Currency: usd}, "+$NaN"},
		{"en", negZero, NumberOptions{SignDisplay: SignAlways, Currency: usd}, "-$0.00"},
		{"en", negZero, NumberOptions{SignDisplay: SignAlways, Currency: usd, Accounting: true}, "($0.00)"},

		// Compact numbers, and compact percents.
		{"en", 1234, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort}, "+1.2K"},
		{"en", -1234, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort}, "-1.2K"},
		{"en", 999999, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort}, "+1M"},
		{"en", 0, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort}, "+0"},
		{"en", 1.234, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort, Percent: true}, "+123%"},
		{"tr", 0.5, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort, Percent: true}, "%+50"},
		{"tr", -0.5, NumberOptions{SignDisplay: SignAlways, Notation: CompactShort, Percent: true}, "%-50"},
	} {
		if got := FormatNumber(c.locale, c.n, c.o); got != c.want {
			t.Errorf("FormatNumber(%q, %v, %+v) = %q, want %q", c.locale, c.n, c.o, got, c.want)
		}
	}
}

// TestNumberSignDisplayFormatter checks that a NumberFormatter writes the
// sign of each number, and that one of amounts keeps the option.
func TestNumberSignDisplayFormatter(t *testing.T) {
	f := NewNumberFormatter("en", NumberOptions{SignDisplay: SignAlways, Currency: CurrencyOf("USD")})
	for _, c := range []struct {
		n    any
		want string
	}{
		{1234, "+$1,234.00"},
		{-1234, "-$1,234.00"},
		{0, "+$0.00"},
	} {
		if got := f.Format(c.n); got != c.want {
			t.Errorf("Format(%v) = %q, want %q", c.n, got, c.want)
		}
	}
}
