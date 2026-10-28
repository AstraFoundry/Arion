package holdkey

import (
	platform "runtime"
	"testing"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

func TestCombinationLegendsUsePlatformModifierNames(t *testing.T) {
	for _, tc := range []struct {
		os   string
		key  runtime.HoldKey
		want string
	}{
		{"darwin", "alt+i", "Option+I"},
		{"darwin", "alt+super+1", "Option+Command+1"},
		{"windows", "alt+i", "Alt+I"},
		{"windows", "alt+super+1", "Alt+Win+1"},
		{"darwin", "alt+period", "Option+."},
		{"darwin", runtime.HoldRightOption, "⌥"},
		{"windows", runtime.HoldRightAlt, "Alt"},
	} {
		if got := legend(tc.key, tc.os); got != tc.want {
			t.Errorf("%s %s: got %q, want %q", tc.os, tc.key, got, tc.want)
		}
	}
}

func TestOptionAltNameMatchesTheLegendInBothLocales(t *testing.T) {
	want := "Alt+I"
	if platform.GOOS == "darwin" {
		want = "Option+I"
	}
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		if got := Name(copy.For(locale), "alt+i"); got != want {
			t.Errorf("%s name: got %q, want %q", locale, got, want)
		}
	}
	if got := Legend("alt+i"); got != want {
		t.Errorf("legend: got %q, want %q", got, want)
	}
}
