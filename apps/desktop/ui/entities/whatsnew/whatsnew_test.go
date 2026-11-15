package whatsnew

import (
	"reflect"
	"testing"
)

func ids(list []Slide) []string {
	var out []string
	for _, slide := range list {
		out = append(out, slide.ID+"@"+slide.Version)
	}
	return out
}

var sample = []Release{
	{Version: "0.3.0", Slides: []string{"e", "f", "g", "h"}},
	{Version: "0.2.1", Skip: "Fixes only."},
	{Version: "0.2.0", Slides: []string{"c", "d"}},
	{Version: "0.2.0-beta.1", Slides: []string{"b"}},
	{Version: "0.1.6", Slides: []string{"a"}},
	{Version: "0.1.5", Skip: "Before the catalog."},
}

func TestPendingFollowsTheVersionSpan(t *testing.T) {
	for _, step := range []struct {
		what, seen, current string
		want                []string
	}{
		{"one release", "0.1.5", "0.1.6", []string{"a@0.1.6"}},
		{"several releases, newest first", "0.1.6", "0.2.0", []string{"c@0.2.0", "d@0.2.0", "b@0.2.0-beta.1"}},
		{"a release with nothing to show", "0.2.0", "0.2.1", nil},
		{"a preview shows only what it has", "0.1.6", "0.2.0-beta.1", []string{"b@0.2.0-beta.1"}},
		{"already seen", "0.2.0", "0.2.0", nil},
		{"a downgrade", "0.3.0", "0.2.0", nil},
		{"a profile from before the catalog", "", "0.1.6", []string{"a@0.1.6"}},
		{"an unreleased version between entries", "0.1.6", "0.2.5", []string{"c@0.2.0", "d@0.2.0", "b@0.2.0-beta.1"}},
		{"a development build", "0.1.5", "dev", nil},
		{"a long span keeps the newest slides", "0.1.5", "0.3.0", []string{"e@0.3.0", "f@0.3.0", "g@0.3.0", "h@0.3.0", "c@0.2.0", "d@0.2.0"}},
	} {
		if got := ids(between(sample, step.seen, step.current)); !reflect.DeepEqual(got, step.want) {
			t.Errorf("%s: %s → %s = %v, want %v", step.what, step.seen, step.current, got, step.want)
		}
	}
}

func TestCompare(t *testing.T) {
	ordered := []string{"0.1.0-alpha.9", "0.1.0-alpha.23", "0.1.0-beta", "0.1.0", "0.1.5", "0.1.6", "0.2.0", "1.0.0"}
	for i := range ordered {
		for j := range ordered {
			want := sign(i - j)
			if got := Compare(ordered[i], ordered[j]); got != want {
				t.Errorf("Compare(%s, %s) = %d, want %d", ordered[i], ordered[j], got, want)
			}
		}
	}
	for _, invalid := range []string{"", "dev", "1.2", "1.02.3", "1.2.3-", "v1.2.3"} {
		if Compare(invalid, "0.0.0") != -1 {
			t.Errorf("%q orders before a release", invalid)
		}
	}
}

func TestParseRejectsMalformedCatalogs(t *testing.T) {
	for what, data := range map[string]string{
		"neither slides nor skip": `{"releases":[{"version":"0.1.6"}]}`,
		"both slides and skip":    `{"releases":[{"version":"0.1.6","slides":["a"],"skip":"x"}]}`,
		"blank skip":              `{"releases":[{"version":"0.1.6","skip":" "}]}`,
		"oldest first":            `{"releases":[{"version":"0.1.5","skip":"x"},{"version":"0.1.6","skip":"x"}]}`,
		"duplicate version":       `{"releases":[{"version":"0.1.6","skip":"x"},{"version":"0.1.6","skip":"x"}]}`,
		"repeated slide":          `{"releases":[{"version":"0.1.6","slides":["a"]},{"version":"0.1.5","slides":["a"]}]}`,
		"invalid version":         `{"releases":[{"version":"next","skip":"x"}]}`,
		"unknown field":           `{"releases":[{"version":"0.1.6","skip":"x","notes":"y"}]}`,
	} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("%s: accepted", what)
		}
	}
}

func TestCatalog(t *testing.T) {
	if len(Releases()) == 0 {
		t.Fatal("empty catalog")
	}
	for _, release := range Releases() {
		if len(release.Slides) > MaxSlides {
			t.Errorf("%s has %d slides; one release shows at most %d", release.Version, len(release.Slides), MaxSlides)
		}
	}
	if got := ids(Latest("0.1.6")); len(got) == 0 || got[0] != "selection-actions@0.1.6" {
		t.Errorf("Latest(0.1.6) = %v", got)
	}
	if got := Latest("0.1.5"); got != nil {
		t.Errorf("Latest(0.1.5) = %v", got)
	}
}
