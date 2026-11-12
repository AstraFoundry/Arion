// Package whatsnew owns which releases have something new to show and
// which of it a person has not seen: the catalog of highlights per release
// and the rule that picks the slides for an update across any number of
// versions. It draws nothing; features/whatsnew owns the slides' scenes.
package whatsnew

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"arion.local/desktop/ui/shared/runtime"
)

// MaxSlides bounds one presentation. An update across many releases keeps
// the newest slides.
const MaxSlides = 6

// Release is one version's entry in the catalog: the slides for what its
// users can now see or do, or the reason it has none.
type Release struct {
	Version string   `json:"version"`
	Slides  []string `json:"slides,omitempty"`
	Skip    string   `json:"skip,omitempty"`
}

// Slide is one highlight and the release that introduced it.
type Slide = runtime.WhatsNewSlide

//go:embed releases.json
var source []byte

var releases = mustParse(source)

func mustParse(data []byte) []Release {
	list, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return list
}

// Parse reads and validates a catalog. Releases are listed newest first;
// each has slides or a skip reason, never both, and a slide belongs to one
// release. scripts/lib/release.mjs applies the same rules before a release
// is packaged.
func Parse(data []byte) ([]Release, error) {
	var file struct {
		Releases []Release `json:"releases"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("what's new catalog: %w", err)
	}
	seen := map[string]bool{}
	for i, release := range file.Releases {
		if _, ok := parse(release.Version); !ok {
			return nil, fmt.Errorf("what's new catalog: invalid version %q", release.Version)
		}
		if i > 0 && Compare(file.Releases[i-1].Version, release.Version) <= 0 {
			return nil, fmt.Errorf("what's new catalog: %s must be listed before older releases", release.Version)
		}
		if (len(release.Slides) == 0) == (strings.TrimSpace(release.Skip) == "") {
			return nil, fmt.Errorf("what's new catalog: %s needs slides or a skip reason, not both", release.Version)
		}
		for _, id := range release.Slides {
			if id == "" || seen[id] {
				return nil, fmt.Errorf("what's new catalog: %s repeats or omits a slide id %q", release.Version, id)
			}
			seen[id] = true
		}
	}
	return file.Releases, nil
}

// Releases is the catalog, newest first.
func Releases() []Release { return releases }

// Pending is what to show a person who last saw What's new at seen and now
// runs current: the slides of every release after seen up to current,
// newest release first, at most MaxSlides. An empty seen is a profile from
// before What's new existed, which has seen none of it. A current the
// catalog cannot order (a development build) and a seen at or after
// current show nothing.
func Pending(seen, current string) []Slide { return between(releases, seen, current) }

// Latest is the slides of the newest release at or before current that has
// any: what About shows on request.
func Latest(current string) []Slide {
	if _, ok := parse(current); !ok {
		return nil
	}
	for _, release := range releases {
		if len(release.Slides) > 0 && Compare(release.Version, current) <= 0 {
			return slides(release)
		}
	}
	return nil
}

func between(list []Release, seen, current string) []Slide {
	if _, ok := parse(current); !ok {
		return nil
	}
	_, known := parse(seen)
	var out []Slide
	for _, release := range list {
		if Compare(release.Version, current) > 0 || (known && Compare(release.Version, seen) <= 0) {
			continue
		}
		out = append(out, slides(release)...)
	}
	if len(out) > MaxSlides {
		out = out[:MaxSlides]
	}
	return out
}

func slides(release Release) []Slide {
	out := make([]Slide, 0, len(release.Slides))
	for _, id := range release.Slides {
		out = append(out, Slide{ID: id, Version: release.Version})
	}
	return out
}

// version is a release number: major.minor.patch and, for a preview, its
// dot-separated prerelease identifiers.
type version struct {
	core [3]int
	pre  []string
}

func parse(text string) (version, bool) {
	var v version
	core, pre, preview := strings.Cut(text, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || strconv.Itoa(n) != part {
			return v, false
		}
		v.core[i] = n
	}
	if preview {
		v.pre = strings.Split(pre, ".")
		for _, id := range v.pre {
			if id == "" {
				return v, false
			}
		}
	}
	return v, true
}

// Compare orders two release numbers as Semantic Versioning does: -1 when
// a is older than b, 1 when newer, 0 when equal. A preview precedes its
// release. A number that is not a release number orders before every one
// that is.
func Compare(a, b string) int {
	left, leftOK := parse(a)
	right, rightOK := parse(b)
	switch {
	case !leftOK && !rightOK:
		return 0
	case !leftOK:
		return -1
	case !rightOK:
		return 1
	}
	for i := range left.core {
		if left.core[i] != right.core[i] {
			return sign(left.core[i] - right.core[i])
		}
	}
	if len(left.pre) == 0 || len(right.pre) == 0 {
		return sign(len(right.pre) - len(left.pre))
	}
	for i := 0; i < len(left.pre) && i < len(right.pre); i++ {
		if c := compareIdentifier(left.pre[i], right.pre[i]); c != 0 {
			return c
		}
	}
	return sign(len(left.pre) - len(right.pre))
}

// compareIdentifier orders prerelease identifiers: numbers by value and
// before words, words by their bytes.
func compareIdentifier(a, b string) int {
	x, errA := strconv.Atoi(a)
	y, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		return sign(x - y)
	case errA == nil:
		return -1
	case errB == nil:
		return 1
	}
	return strings.Compare(a, b)
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
