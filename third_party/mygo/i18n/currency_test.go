package i18n

import (
	"math"
	"testing"
)

func TestCurrencyAmounts(t *testing.T) {
	usd, eur, jpy := CurrencyOf("USD"), CurrencyOf("EUR"), CurrencyOf("JPY")
	for _, c := range []struct {
		locale string
		n      any
		o      NumberOptions
		want   string
	}{
		// What Intl.NumberFormat writes (#1182).
		{"en", 1842, NumberOptions{Currency: usd}, "$1,842.00"},
		{"en", -1842, NumberOptions{Currency: usd}, "-$1,842.00"},
		{"en", -1842, NumberOptions{Currency: usd, Accounting: true}, "($1,842.00)"},
		{"en", 1842, NumberOptions{Currency: usd, Accounting: true}, "$1,842.00"},
		{"de", 1842, NumberOptions{Currency: eur}, "1.842,00\u00a0€"},
		{"ja", 1842, NumberOptions{Currency: jpy}, "￥1,842"},
		{"en-IN", 1234567, NumberOptions{Currency: CurrencyOf("INR")}, "₹12,34,567.00"},
		{"en", 1842, NumberOptions{Currency: eur, CurrencyDisplay: CurrencyCode}, "EUR\u00a01,842.00"},
		{"de", 1842, NumberOptions{Currency: eur, CurrencyDisplay: CurrencyCode}, "1.842,00\u00a0EUR"},
		{"en-GB", 1842, NumberOptions{Currency: usd}, "US$1,842.00"},
		{"en-GB", 1842, NumberOptions{Currency: usd, CurrencyDisplay: CurrencyNarrowSymbol}, "$1,842.00"},
		{"en", 1842, NumberOptions{Currency: usd, CurrencyDisplay: CurrencySymbol}, "$1,842.00"},
		// Names, in the plural form of the amount as written.
		{"en", 1842, NumberOptions{Currency: eur, CurrencyDisplay: CurrencyName}, "1,842.00 euros"},
		{"en", -1, NumberOptions{Currency: eur, CurrencyDisplay: CurrencyName, Accounting: true}, "-1.00 euros"},
		{"fr", 1.5, NumberOptions{Currency: eur, CurrencyDisplay: CurrencyName}, "1,50 euro"},
		{"en", 1842, NumberOptions{Currency: CurrencyOf("xyz"), CurrencyDisplay: CurrencyName}, "1,842.00 XYZ"},
		// The currency's fraction digits, unless the options give some.
		{"en", 1842, NumberOptions{Currency: CurrencyOf("BHD")}, "BHD\u00a01,842.000"},
		{"en", 1842.5, NumberOptions{Currency: usd, Integer: true}, "$1,843"},
		{"en", 1842.56, NumberOptions{Currency: usd, MaxFractionDigits: 1}, "$1,842.6"},
		{"en", 1842.5, NumberOptions{Currency: usd, MinFractionDigits: 3}, "$1,842.500"},
		{"en", 1842.5, NumberOptions{Currency: jpy, MinFractionDigits: 1}, "¥1,842.5"},
		{"en", 1842, NumberOptions{Currency: CurrencyOf("XYZ")}, "XYZ\u00a01,842.00"},
		// Separators of amounts, and formats of a currency's own.
		{"de-AT", 1842.53, NumberOptions{Currency: eur}, "€\u00a01.842,53"},
		{"fr-CH", 1842.53, NumberOptions{Currency: CurrencyOf("CHF")}, "1'842.53\u00a0CHF"},
		{"en-DE", -1842, NumberOptions{Currency: usd}, "-US$1,842.00"},
		{"pt-CV", 1842, NumberOptions{Currency: usd}, "1842$00\u00a0US$"},
		// Compact amounts.
		{"en", 4200000, NumberOptions{Currency: usd, Notation: CompactShort}, "$4.2M"},
		{"en", 4200000, NumberOptions{Currency: usd, Notation: CompactLong}, "$4.2M"},
		{"en", 999, NumberOptions{Currency: usd, Notation: CompactShort}, "$999"},
		{"en", 4200000, NumberOptions{Currency: eur, Notation: CompactShort, CurrencyDisplay: CurrencyCode}, "EUR\u00a04.2M"},
		{"en", 4200000, NumberOptions{Currency: eur, Notation: CompactLong, CurrencyDisplay: CurrencyName}, "4.2 million euros"},
		{"fr", 1000000, NumberOptions{Currency: eur, Notation: CompactLong, CurrencyDisplay: CurrencyName}, "1 million euros"},
		{"de", 1234, NumberOptions{Currency: eur, Notation: CompactShort}, "1234\u00a0€"},
		{"en", -123, NumberOptions{Currency: eur, Notation: CompactShort, Accounting: true}, "(€123)"},
		{"ja", 123456789, NumberOptions{Currency: jpy, Notation: CompactShort}, "￥1.2億"},
		// ECMA-402's maximum of compact amounts is 3 with a minimum, as
		// JavaScriptCore writes it; V8 writes "$1.23K".
		{"en", 1234, NumberOptions{Currency: usd, Notation: CompactShort, MinFractionDigits: 2}, "$1.234K"},
		// NaN and infinities, whose signs are not spaced.
		{"en", math.NaN(), NumberOptions{Currency: usd, CurrencyDisplay: CurrencyCode}, "USDNaN"},
		{"en", math.Inf(-1), NumberOptions{Currency: usd, Accounting: true}, "($∞)"},
		{"en", math.NaN(), NumberOptions{Currency: usd, CurrencyDisplay: CurrencyName}, "NaN US dollars"},
		{"en", math.Copysign(0, -1), NumberOptions{Currency: usd, Accounting: true}, "($0.00)"},
		// Percent is ignored with a currency; the zero Currency is none.
		{"en", 0.5, NumberOptions{Currency: usd, Percent: true}, "$0.50"},
		{"en", 1842, NumberOptions{Currency: CurrencyOf("US")}, "1,842"},
	} {
		if got := FormatNumber(c.locale, c.n, c.o); got != c.want {
			t.Errorf("FormatNumber(%q, %v, %+v) = %q, want %q", c.locale, c.n, c.o, got, c.want)
		}
	}
}

func TestCurrencyOf(t *testing.T) {
	for code, want := range map[string]string{"USD": "USD", "eur": "EUR", "Jpy": "JPY", "US": "", "US$": "", "USDX": "", "": ""} {
		if got := CurrencyOf(code).Code(); got != want {
			t.Errorf("CurrencyOf(%q).Code() = %q, want %q", code, got, want)
		}
	}
}

func TestCurrencyDigits(t *testing.T) {
	// CLDR 48's supplemental currencyData, as V8 takes it; JavaScriptCore
	// takes ISO 4217's minor units, which differ for some (IQD 3, HUF 2).
	for code, want := range map[string]string{
		"USD": "USD\u00a01.00", "JPY": "JPY\u00a01", "BHD": "BHD\u00a01.000", "CLF": "CLF\u00a01.0000",
		"HUF": "HUF\u00a01", "IQD": "IQD\u00a01", "RSD": "RSD\u00a01.00", "XYZ": "XYZ\u00a01.00",
	} {
		if got := FormatNumber("en", 1, NumberOptions{Currency: CurrencyOf(code), CurrencyDisplay: CurrencyCode}); got != want {
			t.Errorf("1 %s = %q, want %q", code, got, want)
		}
	}
}

func TestCurrencyAllocations(t *testing.T) {
	// An amount allocates as a number of its fraction digits does; one
	// with a name, its plural form's operands too.
	usd := CurrencyOf("USD")
	for _, c := range []struct{ amount, number NumberOptions }{
		{NumberOptions{Currency: usd}, NumberOptions{MinFractionDigits: 2}},
		{NumberOptions{Currency: usd, CurrencyDisplay: CurrencyName}, NumberOptions{MinFractionDigits: 2}},
		{NumberOptions{Currency: usd, Notation: CompactShort}, NumberOptions{Notation: CompactShort}},
	} {
		want := testing.AllocsPerRun(100, func() { FormatNumber("en", 1842.5, c.number) })
		if c.amount.CurrencyDisplay == CurrencyName {
			want++
		}
		if n := testing.AllocsPerRun(100, func() { FormatNumber("en", 1842.5, c.amount) }); n > want {
			t.Errorf("FormatNumber with %+v allocates %v times, want %v", c.amount, n, want)
		}
	}
}
