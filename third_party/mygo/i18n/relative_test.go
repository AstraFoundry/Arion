package i18n

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"
	"testing"
)

func TestFormatRelativeTime(t *testing.T) {
	for _, c := range []struct {
		locale string
		value  float64
		unit   RelativeTimeUnit
		o      RelativeTimeOptions
		want   string
	}{
		{"en", -3, RelativeMinute, RelativeTimeOptions{}, "3 minutes ago"},
		{"en", 2, RelativeDay, RelativeTimeOptions{}, "in 2 days"},
		{"en", -1, RelativeDay, RelativeTimeOptions{Auto: true}, "yesterday"},
		{"en", 0, RelativeSecond, RelativeTimeOptions{Auto: true}, "now"},
		{"en", 1.0001, RelativeDay, RelativeTimeOptions{Auto: true}, "tomorrow"},
		{"en", -2, RelativeDay, RelativeTimeOptions{Auto: true}, "2 days ago"},
		{"en", math.Copysign(0, -1), RelativeDay, RelativeTimeOptions{}, "0 days ago"},
		{"en", 0, RelativeDay, RelativeTimeOptions{}, "in 0 days"},
		{"en", -3, RelativeMinute, RelativeTimeOptions{Style: RelativeShort}, "3 min. ago"},
		{"en", -3, RelativeMinute, RelativeTimeOptions{Style: RelativeNarrow}, "3m ago"},
		{"fr", -2, RelativeDay, RelativeTimeOptions{Auto: true}, "avant-hier"},
		{"zh", 1, RelativeDay, RelativeTimeOptions{Auto: true}, "明天"},
		{"ru", -21, RelativeMinute, RelativeTimeOptions{}, "21 минуту назад"},
		{"ar", 2, RelativeDay, RelativeTimeOptions{}, "خلال يومين"},
		{"ht", 1, RelativeDay, RelativeTimeOptions{Auto: true}, "demain"},
		{"el-polyton", 1, RelativeDay, RelativeTimeOptions{Auto: true}, "αύριο"},
		{"unknown", -1, RelativeHour, RelativeTimeOptions{}, "1 hour ago"},
		{"en_US.UTF-8", -1, RelativeHour, RelativeTimeOptions{}, "1 hour ago"},
	} {
		if got := FormatRelativeTime(c.locale, c.value, c.unit, c.o); got != c.want {
			t.Errorf("%s %v %d %+v: got %q, want %q", c.locale, c.value, c.unit, c.o, got, c.want)
		}
	}
}

func TestRelativeTimeInvalid(t *testing.T) {
	for _, fn := range []func(){
		func() { FormatRelativeTime("en", math.NaN(), RelativeDay, RelativeTimeOptions{}) },
		func() { FormatRelativeTime("en", math.Inf(1), RelativeDay, RelativeTimeOptions{}) },
		func() { FormatRelativeTime("en", math.Inf(-1), RelativeDay, RelativeTimeOptions{}) },
		func() { FormatRelativeTime("en", 1, RelativeTimeUnit(255), RelativeTimeOptions{}) },
		func() { NewRelativeTimeFormatter("en", RelativeTimeOptions{Style: RelativeTimeStyle(255)}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid input did not panic")
				}
			}()
			fn()
		}()
	}
}

func TestRelativeTimeConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			f := NewRelativeTimeFormatter("de-CH", RelativeTimeOptions{Auto: true})
			for range 100 {
				if got := f.Format(1, RelativeDay); got != "morgen" {
					t.Errorf("got %q, want morgen", got)
				}
			}
		})
	}
	wg.Wait()
}

func TestRelativeTimeMatchesICU(t *testing.T) {
	file, err := os.Open("testdata/relative_time_icu.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	var golden struct {
		ICU, CLDR string
		Cases     []struct {
			Value string
			Unit  RelativeTimeUnit
			Style RelativeTimeStyle
			Auto  bool
		}
		Locales map[string][]string
	}
	if err := json.NewDecoder(gz).Decode(&golden); err != nil {
		t.Fatal(err)
	}
	t.Logf("ICU %s, CLDR %s: %d locales, %d cases", golden.ICU, golden.CLDR, len(golden.Locales), len(golden.Cases))
	for locale, row := range golden.Locales {
		t.Run(locale, func(t *testing.T) {
			// ICU 78 has no Haitian Creole or polytonic Greek data; the
			// existing format golden records the same differences.
			if locale == "ht" || locale == "el-polyton" {
				t.Skip("ICU 78 lacks this CLDR locale")
			}
			if len(row) != len(golden.Cases) {
				t.Fatal("incomplete golden row")
			}
			failures := 0
			for i, c := range golden.Cases {
				v, err := strconv.ParseFloat(c.Value, 64)
				if err != nil {
					t.Fatal(err)
				}
				got := FormatRelativeTime(locale, v, c.Unit, RelativeTimeOptions{Style: c.Style, Auto: c.Auto})
				want := row[i]
				// ICU 78 falls back to be for these two words, while
				// CLDR gives be-tarask its own short and narrow forms.
				if locale == "be-tarask" && c.Style != RelativeLong && c.Auto && math.Abs(v) < 0.005 {
					switch c.Unit {
					case RelativeMinute:
						if want != "у гэту хвіліну" {
							t.Fatal("ICU's be-tarask minute changed")
						}
						want = "у гэтую хвіліну"
					case RelativeHour:
						if want != "у гэту гадзіну" {
							t.Fatal("ICU's be-tarask hour changed")
						}
						want = "у гэтую гадзіну"
					}
				}
				if got != want {
					t.Errorf("%+v: got %q, want %q", c, got, want)
					failures++
					if failures >= 8 {
						break
					}
				}
			}
		})
	}
}

func BenchmarkRelativeTime(b *testing.B) {
	f := NewRelativeTimeFormatter("en", RelativeTimeOptions{})
	b.ReportAllocs()
	for b.Loop() {
		f.Format(-3, RelativeMinute)
	}
}

func ExampleFormatRelativeTime() {
	fmt.Println(FormatRelativeTime("en", -3, RelativeMinute, RelativeTimeOptions{}))
	fmt.Println(FormatRelativeTime("zh", 1, RelativeDay, RelativeTimeOptions{Auto: true}))
	fmt.Println(FormatRelativeTime("fr", -2, RelativeDay, RelativeTimeOptions{Auto: true}))
	// Output:
	// 3 minutes ago
	// 明天
	// avant-hier
}
