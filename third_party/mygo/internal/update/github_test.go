package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestResolveFeed(t *testing.T) {
	feed := TaggedFeed("me/app", "desktop-v", "update-linux-amd64.json")
	if want := "https://github.com/me/app/releases/download/desktop-v{version}/update-linux-amd64.json"; feed != want {
		t.Fatalf("feed = %s, want %s", feed, want)
	}
	// The releases of the repository, newest first, a page for each list.
	var pages [][]string
	get := func(_ context.Context, u string) (io.ReadCloser, error) {
		for i, page := range pages {
			if u == fmt.Sprintf("https://api.github.com/repos/me/app/releases?per_page=100&page=%d", i+1) {
				return io.NopCloser(strings.NewReader("[" + strings.Join(page, ",") + "]")), nil
			}
		}
		if u == fmt.Sprintf("https://api.github.com/repos/me/app/releases?per_page=100&page=%d", len(pages)+1) {
			return io.NopCloser(strings.NewReader("[]")), nil
		}
		return nil, fmt.Errorf("GET %s", u)
	}
	rel := func(tag string, draft, pre bool) string {
		return fmt.Sprintf(`{"tag_name": %q, "draft": %t, "prerelease": %t, "body": "not \"tag_name\": \"desktop-v9.0.0\""}`, tag, draft, pre)
	}

	pages = [][]string{{
		rel("cli-v2.0.0", false, false),
		rel("desktop-v1.3.0", true, false),
		rel("desktop-v1.2.0-beta.1", false, true),
		rel("desktop-v1.1.0", false, false),
		rel("desktop-v1.0.0", false, false),
	}}
	got, err := ResolveFeed(context.Background(), feed, get)
	if want := "https://github.com/me/app/releases/download/desktop-v1.1.0/update-linux-amd64.json"; err != nil || got != want {
		t.Errorf("ResolveFeed = %s, %v, want %s", got, err, want)
	}

	// A full page leads to the next.
	var full []string
	for i := range releasesPerPage {
		full = append(full, rel(fmt.Sprintf("cli-v1.0.%d", i), false, false))
	}
	pages = [][]string{full, {rel("desktop-v0.9.0", false, false)}}
	if got, err := ResolveFeed(context.Background(), feed, get); err != nil || !strings.Contains(got, "/desktop-v0.9.0/") {
		t.Errorf("ResolveFeed = %s, %v", got, err)
	}

	pages = [][]string{{rel("cli-v2.0.0", false, false)}}
	if _, err := ResolveFeed(context.Background(), feed, get); !errors.Is(err, ErrNotPublished) {
		t.Errorf("without a release, err = %v", err)
	}

	// Other feeds are the manifest.
	plain := "https://github.com/me/app/releases/latest/download/update-linux-amd64.json"
	if got, err := ResolveFeed(context.Background(), plain, nil); err != nil || got != plain {
		t.Errorf("ResolveFeed = %s, %v", got, err)
	}
	if _, err := ResolveFeed(context.Background(), "https://example.com/{version}/update.json", get); err == nil {
		t.Error("a feed with {version} outside GitHub releases was resolved")
	}
}

func TestPrereleaseChannel(t *testing.T) {
	feed := TaggedFeed("me/app", "v", "update-darwin-arm64.json")
	get := func(context.Context, string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`[{"tag_name":"v0.3.0-alpha.1","prerelease":true,"assets":[{"name":"update-windows-amd64.json"}]},{"tag_name":"v0.2.0-alpha.2","draft":true,"prerelease":true},{"tag_name":"v0.2.0-alpha.1","prerelease":true,"assets":[{"name":"update-darwin-arm64.json"}]},{"tag_name":"v0.1.0"}]`)), nil
	}
	for _, tc := range []struct {
		include bool
		tag     string
	}{{false, "v0.1.0"}, {true, "v0.2.0-alpha.1"}} {
		got, err := ResolveFeedWithPrereleases(context.Background(), feed, get, tc.include)
		if err != nil || !strings.Contains(got, "/"+tc.tag+"/") {
			t.Fatalf("include=%t: %s %v", tc.include, got, err)
		}
	}
	onlyPreview := func(context.Context, string) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`[{"tag_name":"v0.2.0-alpha.1","prerelease":true,"assets":[{"name":"update-darwin-arm64.json"}]}]`)), nil
	}
	if _, err := ResolveFeedWithPrereleases(context.Background(), feed, onlyPreview, false); !errors.Is(err, ErrNotPublished) {
		t.Fatalf("stable channel: %v", err)
	}
	if _, err := ResolveFeedWithPrereleases(context.Background(), feed, func(context.Context, string) (io.ReadCloser, error) { return nil, context.DeadlineExceeded }, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("network failure hidden: %v", err)
	}
}

func TestResolvePrereleaseFeed(t *testing.T) {
	var releases []string
	get := func(_ context.Context, u string) (io.ReadCloser, error) {
		if u == "https://api.github.com/repos/me/app/releases?per_page=100&page=1" {
			return io.NopCloser(strings.NewReader("[" + strings.Join(releases, ",") + "]")), nil
		}
		return nil, fmt.Errorf("GET %s", u)
	}
	rel := func(tag string, draft, pre bool) string {
		return fmt.Sprintf(`{"tag_name": %q, "draft": %t, "prerelease": %t}`, tag, draft, pre)
	}
	latest := "https://github.com/me/app/releases/latest/download/update-linux-amd64.json"
	tagged := TaggedFeed("me/app", "desktop-v", "update-linux-amd64.json")
	for _, c := range []struct {
		name, feed string
		releases   []string
		want       string
	}{
		{"a prerelease newer than the latest release", latest,
			[]string{rel("v1.3.0", true, false), rel("v1.2.0-beta.2", false, true), rel("v1.2.0-beta.10", false, true), rel("v1.1.0", false, false)},
			"https://github.com/me/app/releases/download/v1.2.0-beta.10/update-linux-amd64.json"},
		{"a release newer than the prereleases", latest,
			[]string{rel("v1.2.0", false, false), rel("v1.2.0-beta.2", false, true)},
			"https://github.com/me/app/releases/download/v1.2.0/update-linux-amd64.json"},
		{"by version, not by date", latest,
			[]string{rel("v1.1.1", false, false), rel("v1.2.0-beta.1", false, true), rel("nightly", false, true)},
			"https://github.com/me/app/releases/download/v1.2.0-beta.1/update-linux-amd64.json"},
		{"tagged", tagged,
			[]string{rel("cli-v3.0.0-beta.1", false, true), rel("desktop-v2.0.0-rc.1", false, true), rel("desktop-v1.0.0", false, false)},
			"https://github.com/me/app/releases/download/desktop-v2.0.0-rc.1/update-linux-amd64.json"},
	} {
		releases = c.releases
		if got, err := ResolvePrereleaseFeed(context.Background(), c.feed, get); err != nil || got != c.want {
			t.Errorf("%s: ResolvePrereleaseFeed = %s, %v, want %s", c.name, got, err, c.want)
		}
	}

	releases = []string{rel("v1.0.0", true, false)}
	if _, err := ResolvePrereleaseFeed(context.Background(), latest, get); !errors.Is(err, ErrNotPublished) {
		t.Errorf("with drafts only, err = %v", err)
	}
	plain := "https://downloads.example.com/my-app/update-linux-amd64.json"
	if got, err := ResolvePrereleaseFeed(context.Background(), plain, nil); err != nil || got != plain {
		t.Errorf("ResolvePrereleaseFeed = %s, %v", got, err)
	}
}

func TestResolvePrereleaseFeedPages(t *testing.T) {
	rel := func(tag string, pre bool) string {
		return fmt.Sprintf(`{"tag_name": %q, "draft": false, "prerelease": %t}`, tag, pre)
	}
	// nightlies returns n prereleases, newest first.
	nightlies := func(n int) []string {
		rs := make([]string, n)
		for i := range rs {
			rs[i] = rel(fmt.Sprintf("v2.0.0-nightly.%d", n-i), true)
		}
		return rs
	}
	page := func(rs ...string) string { return "[" + strings.Join(rs, ",") + "]" }
	latest := "https://github.com/me/app/releases/latest/download/update-linux-amd64.json"
	for _, c := range []struct {
		name  string
		pages []string
		want  string
		reads int
	}{
		{"a full page with a stable release", []string{
			page(append(nightlies(releasesPerPage-1), rel("v1.9.0", false))...),
			page(rel("v3.0.0-beta.1", true)), // listed after v1.9.0: not read
		}, "v2.0.0-nightly.99", 1},
		{"prereleases only on the first page", []string{
			page(nightlies(releasesPerPage)...),
			page(rel("v2.0.0-beta.1", true), rel("v1.9.0", false)),
			page(rel("v1.0.0", false)),
		}, "v2.0.0-nightly.100", 2},
		{"a stable release after the prereleases", []string{
			page(nightlies(releasesPerPage)...),
			page(rel("v2.0.0", false)),
		}, "v2.0.0", 2},
	} {
		reads := 0
		get := func(_ context.Context, u string) (io.ReadCloser, error) {
			reads++
			for i, p := range c.pages {
				if u == fmt.Sprintf("https://api.github.com/repos/me/app/releases?per_page=100&page=%d", i+1) {
					return io.NopCloser(strings.NewReader(p)), nil
				}
			}
			return nil, fmt.Errorf("GET %s", u)
		}
		want := "https://github.com/me/app/releases/download/" + c.want + "/update-linux-amd64.json"
		if got, err := ResolvePrereleaseFeed(context.Background(), latest, get); err != nil || got != want {
			t.Errorf("%s: ResolvePrereleaseFeed = %s, %v, want %s", c.name, got, err, want)
		}
		if reads != c.reads {
			t.Errorf("%s: read %d pages, want %d", c.name, reads, c.reads)
		}
	}

	// An error reading the releases, such as the API's rate limit, is the
	// check's.
	get := func(context.Context, string) (io.ReadCloser, error) { return nil, errors.New("GET: 403 Forbidden") }
	if _, err := ResolvePrereleaseFeed(context.Background(), latest, get); err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("rate limited: err = %v", err)
	}
}
