//go:build linux && (amd64 || arm64)

package linux

import (
	"os/exec"
	"slices"
	"testing"
)

func TestHourCycleOf(t *testing.T) {
	for format, want := range map[string]int{"%H:%M:%S": 24, "%r": 12, "%I:%M:%S %p": 12, "%l:%M %p": 12, "%T": 24, "%-H.%M": 24, "%Ex": 0, "": 0} {
		if got := hourCycleOf(format); got != want {
			t.Errorf("hourCycleOf(%q) = %d, want %d", format, got, want)
		}
	}
}

func TestPreferredLanguages(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "de_DE.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LANGUAGE", "fr:es")
	if got := (appController{}).PreferredLanguages(); !slices.Equal(got, []string{"fr", "es", "de_DE.UTF-8"}) {
		t.Errorf("PreferredLanguages() = %q", got)
	}
	// LANGUAGE does not count in the C locale.
	t.Setenv("LC_MESSAGES", "C")
	if got := (appController{}).PreferredLanguages(); len(got) != 0 {
		t.Errorf("PreferredLanguages() = %q in the C locale", got)
	}
}

// TestRegion reads the formats of locales glibc has, which the system must
// have generated (locale-gen): the first day of the week and the hour
// cycle.
func TestRegion(t *testing.T) {
	out, _ := exec.Command("locale", "-a").Output()
	for _, c := range []struct {
		locale, has  string
		first, cycle int
	}{
		{"en_US.UTF-8", "en_US.utf8", 0, 12},
		{"de_DE.UTF-8", "de_DE.utf8", 1, 24},
		{"en_GB.UTF-8", "en_GB.utf8", 1, 24},
		{"ar_EG.UTF-8", "ar_EG.utf8", 6, 12},
	} {
		if !slices.Contains(splitLines(string(out)), c.has) {
			t.Logf("skipping %s, which the system has not generated", c.locale)
			continue
		}
		t.Setenv("LC_ALL", "")
		t.Setenv("LANG", "C")
		t.Setenv("LC_TIME", c.locale)
		r := (appController{}).Region()
		if r.Locale != c.locale || r.FirstWeekday != c.first || r.HourCycle != c.cycle {
			t.Errorf("%s: Region() = %+v, want first weekday %d, hours %d", c.locale, r, c.first, c.cycle)
		}
	}
	t.Setenv("LC_TIME", "xx_YY.UTF-8") // not generated
	if r := (appController{}).Region(); r.FirstWeekday != -1 || r.HourCycle != 0 {
		t.Errorf("a locale that is not there: %+v", r)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}
