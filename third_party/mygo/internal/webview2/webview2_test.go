package webview2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// runtimeDir makes a runtime's directory, with what the loader looks for.
func runtimeDir(t *testing.T, version string) string {
	dir := filepath.Join(t.TempDir(), version)
	touch(t, filepath.Join(dir, browserExe))
	touch(t, filepath.Join(dir, clientDir, archDir(), clientDLL))
	return dir
}

func TestClientDLL(t *testing.T) {
	dir := runtimeDir(t, "154.0.4258.62")
	if got, want := ClientDLL(dir), filepath.Join(dir, clientDir, archDir(), clientDLL); got != want {
		t.Errorf("ClientDLL of a runtime = %q, want %q", got, want)
	}

	// The Edge browser's layout: the library without the runtime's browser
	// process, whose environments fail with ERROR_NOT_SUPPORTED.
	edge := t.TempDir()
	touch(t, filepath.Join(edge, clientDir, archDir(), clientDLL))
	if got := ClientDLL(edge); got != "" {
		t.Errorf("ClientDLL of the library alone = %q, want none", got)
	}

	browser := t.TempDir()
	touch(t, filepath.Join(browser, browserExe))
	if got := ClientDLL(browser); got != "" {
		t.Errorf("ClientDLL of the browser alone = %q, want none", got)
	}

	other := t.TempDir()
	touch(t, filepath.Join(other, browserExe))
	touch(t, filepath.Join(other, clientDir, "other", clientDLL))
	if got := ClientDLL(other); got != "" {
		t.Errorf("ClientDLL of another architecture's runtime = %q, want none", got)
	}
}

func TestFind(t *testing.T) {
	key := func(channel string) string { return `SOFTWARE\Microsoft\EdgeUpdate\ClientState\` + channel }
	registry := func(values map[bool]map[string]string) func(bool, string, string) string {
		return func(machine bool, path, name string) string {
			if name != clientDir {
				return ""
			}
			return values[machine][path]
		}
	}
	older, newer := runtimeDir(t, "153.0.1.2"), runtimeDir(t, "154.0.4258.62")
	beta := runtimeDir(t, "155.0.1.1")
	gone := filepath.Join(t.TempDir(), "154.0.4258.62")

	for _, tc := range []struct {
		name   string
		fixed  string
		values map[bool]map[string]string
		dir    string
		err    string
	}{
		{name: "none", err: "not installed"},
		{
			name:   "machine",
			values: map[bool]map[string]string{true: {key(channels[0]): newer}},
			dir:    newer,
		},
		{
			name: "newest of a channel",
			values: map[bool]map[string]string{
				true:  {key(channels[0]): older},
				false: {key(channels[0]): newer},
			},
			dir: newer,
		},
		{
			name: "stable before beta",
			values: map[bool]map[string]string{
				true: {key(channels[0]): older, key(channels[1]): beta},
			},
			dir: older,
		},
		{
			// EdgeUpdate keeps the record of a runtime whose files are gone.
			name:   "stale",
			values: map[bool]map[string]string{true: {key(channels[0]): gone}},
			err:    gone,
		},
		{
			name: "stale stable, beta installed",
			values: map[bool]map[string]string{
				true: {key(channels[0]): gone, key(channels[1]): beta},
			},
			dir: beta,
		},
		{name: "fixed", fixed: older, dir: older},
		{
			name:   "fixed without a runtime",
			fixed:  gone,
			values: map[bool]map[string]string{true: {key(channels[0]): newer}},
			err:    "WEBVIEW2_BROWSER_EXECUTABLE_FOLDER",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, fixed, err := find(tc.fixed, registry(tc.values))
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("find = %q, %v; want an error naming %q", dir, err, tc.err)
				}
				return
			}
			if err != nil || dir != tc.dir || fixed != (tc.fixed != "") {
				t.Errorf("find = %q, %v, %v; want %q, %v", dir, fixed, err, tc.dir, tc.fixed != "")
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"154.0.4258.62", "154.0.4258.62", 0},
		{"155.0.1.1", "154.9.9.9", 1},
		{"154.0.4258.62", "154.0.4258.100", -1},
		{"154.0", "154.0.1", -1},
		{"not-a-version", "1.0.0.0", -1},
	} {
		if got := compareVersions(parseVersion(tc.a), parseVersion(tc.b)); got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
