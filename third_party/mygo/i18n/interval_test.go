package i18n

import (
	"testing"
	"time"
)

func TestFormatDateRange(t *testing.T) {
	day := func(m time.Month, d, h, min int) time.Time { return time.Date(2026, m, d, h, min, 0, 0, time.UTC) }
	medium := DateOptions{Skeleton: "yMMMd"}
	for _, c := range []struct {
		locale     string
		start, end time.Time
		o          DateOptions
		want       string
	}{
		{"en", day(1, 5, 0, 0), day(1, 9, 0, 0), medium, "Jan 5\u2009–\u20099, 2026"},
		{"en", day(1, 5, 0, 0), day(2, 9, 0, 0), medium, "Jan 5\u2009–\u2009Feb 9, 2026"},
		{"en", day(1, 5, 0, 0), day(1, 5, 23, 0), medium, "Jan 5, 2026"}, // the fields written are the same
		{"en", day(12, 29, 0, 0), day(1, 2, 0, 0).AddDate(1, 0, 0), medium, "Dec 29, 2026\u2009–\u2009Jan 2, 2027"},
		{"en", day(1, 5, 10, 0), day(1, 5, 11, 30), DateOptions{Skeleton: "jm"}, "10:00\u2009–\u200911:30\u202fAM"},
		{"en", day(1, 5, 10, 0), day(1, 5, 13, 0), DateOptions{Skeleton: "jm"}, "10:00\u202fAM\u2009–\u20091:00\u202fPM"},
		{"en", day(1, 5, 10, 0), day(1, 6, 13, 0), DateOptions{Skeleton: "jm"}, "1/5/2026, 10:00\u202fAM\u2009–\u20091/6/2026, 1:00\u202fPM"},
		{"en", day(1, 5, 10, 0), day(1, 5, 11, 0), DateOptions{Date: MediumStyle, Time: ShortStyle}, "Jan 5, 2026, 10:00\u2009–\u200911:00\u202fAM"},
		{"en", day(1, 5, 10, 0), day(1, 5, 13, 0), DateOptions{Skeleton: "jm", HourCycle: 24}, "10:00\u2009–\u200913:00"},
		{"de", day(1, 5, 0, 0), day(1, 9, 0, 0), medium, "5.–9. Jan. 2026"},
		{"fr", day(1, 5, 0, 0), day(2, 9, 0, 0), DateOptions{Skeleton: "yMMMMd"}, "5 janvier\u2009–\u20099 février 2026"},
		{"ja", day(1, 5, 0, 0), day(1, 9, 0, 0), medium, "2026/01/05～2026/01/09"},
	} {
		if got := FormatDateRange(c.locale, c.start, c.end, c.o); got != c.want {
			t.Errorf("FormatDateRange(%q, %v, %v, %+v) = %q, want %q", c.locale, c.start, c.end, c.o, got, c.want)
		}
	}
	// The end in the start's location.
	start := day(1, 5, 10, 0)
	end := day(1, 5, 11, 0).In(time.FixedZone("UTC+9", 9*3600))
	if got, want := FormatDateRange("en", start, end, DateOptions{Skeleton: "jm"}), "10:00\u2009–\u200911:00\u202fAM"; got != want {
		t.Errorf("a range ending in another zone: %q, want %q", got, want)
	}
}
