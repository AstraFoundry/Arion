package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/update"
)

type updateTestTransport func(*http.Request) (*http.Response, error)

func (f updateTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// A platform published after another platform still gets deltas from its
// earlier release, including the archives that manifest passes on.
func TestWriteDeltasGitHub(t *testing.T) {
	const target = "darwin-universal"
	name := update.ManifestName(target)
	release := func(tag, extra string, assets ...string) string {
		var names []string
		for _, asset := range assets {
			names = append(names, fmt.Sprintf(`{"name":%q}`, asset))
		}
		return fmt.Sprintf(`{"tag_name":%q,"assets":[%s]%s}`, tag, strings.Join(names, ","), extra)
	}
	page := func(releases ...string) string { return "[" + strings.Join(releases, ",") + "]" }
	partial := page(
		release("v1.2.0", "", "update-windows-amd64.json"),
		release("v1.1.0", "", name),
	)
	full := make([]string, 100)
	for i := range full {
		full[i] = release(fmt.Sprintf("cli-v2.0.%d", i), "", name)
	}
	for _, tc := range []struct {
		name          string
		prefix        string
		customURL     bool
		latestStatus  int
		latestBody    string
		pages         []string
		apiStatus     int
		candidateCode int
		badSignature  bool
		wantDeltas    int
		wantPrevious  int
		wantPages     int
	}{
		{name: "published manifest", latestStatus: 200, wantDeltas: 2, wantPrevious: 2},
		{name: "latest has only Windows", pages: []string{partial}, wantDeltas: 2, wantPrevious: 2, wantPages: 1},
		{name: "tag prefix", prefix: "desktop-v", pages: []string{page(
			release("cli-v3.0.0", "", name),
			release("desktop-v1.2.0", "", "update-windows-amd64.json"),
			release("desktop-v1.1.0", "", name),
		)}, wantDeltas: 2, wantPrevious: 2, wantPages: 2},
		{name: "eligible releases and missing assets", pages: []string{page(
			release("cli-v3.0.0", "", name),
			release("v", "", name),
			release("vnightly", "", name),
			release("v1.1.9", `,"draft":true`, name),
			release("v1.1.8-beta.1", `,"prerelease":true`, name),
			release("v2.0.0", "", name),
			release("v1.2.0", "", name),
			release("v1.1.7", "", "update-windows-amd64.json"),
			release("v1.1.6", "", name), // advertised but returns 404
			release("v1.1.5", "", name), // manifest names a newer version
			release("v1.1.0", "", name),
		)}, wantDeltas: 2, wantPrevious: 2, wantPages: 1},
		{name: "pagination", pages: []string{page(full...), partial}, wantDeltas: 2, wantPrevious: 2, wantPages: 2},
		{name: "no platform release", pages: []string{page(release("v1.1.0", "", "update-windows-amd64.json"))}, wantPages: 1},
		{name: "bounded pagination", pages: slices.Repeat([]string{page(full...)}, 10), wantPages: 10},
		{name: "latest server error", latestStatus: 503},
		{name: "malformed latest manifest", latestStatus: 200, latestBody: "{"},
		{name: "custom URL missing", customURL: true},
		{name: "API error", pages: []string{partial}, apiStatus: 403, wantPages: 1},
		{name: "malformed API response", pages: []string{"{"}, wantPages: 1},
		{name: "candidate server error", pages: []string{partial}, candidateCode: 503, wantPages: 1},
		{name: "wrong signing key", pages: []string{partial}, badSignature: true, wantPrevious: 2, wantPages: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pub, key, err := ed25519.GenerateKey(nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
			// Incompressible code makes the full archive much larger than
			// a delta that changes only the version at its end.
			code := make([]byte, 64<<10)
			if _, err := rand.New(rand.NewSource(1)).Read(code); err != nil {
				t.Fatal(err)
			}
			app := func(root, version string) []string {
				t.Helper()
				bundle := filepath.Join(root, "App.app")
				if err := os.MkdirAll(bundle, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(bundle, "app"), append(slices.Clone(code), version...), 0o755); err != nil {
					t.Fatal(err)
				}
				return []string{"App.app"}
			}
			var archives [][]byte
			var bases []update.Archive
			for _, v := range []string{"1.1.0", "1.0.0"} {
				dir := t.TempDir()
				var archive bytes.Buffer
				if err := update.WriteArchive(&archive, dir, app(dir, v)); err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(archive.Bytes())
				signer := key
				if tc.badSignature {
					_, signer, err = ed25519.GenerateKey(nil)
					if err != nil {
						t.Fatal(err)
					}
				}
				archives = append(archives, archive.Bytes())
				bases = append(bases, update.Archive{
					Version: v, URL: "https://github.com/me/app/releases/download/v" + v + "/app.tar.gz",
					Size: int64(archive.Len()), Signature: update.Sign(signer, sum[:]),
				})
			}
			manifest, err := json.Marshal(update.Manifest{
				Version: bases[0].Version, URL: bases[0].URL, Size: bases[0].Size,
				Signature: bases[0].Signature, Previous: bases[1:],
			})
			if err != nil {
				t.Fatal(err)
			}
			prefix := tc.prefix
			if prefix == "" {
				prefix = "v"
			}
			latestPath := "/me/app/releases/latest/download/" + name
			if prefix != "v" {
				latestPath = "/me/app/releases/download/" + prefix + "1.2.0/" + name
			}
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			apiReads := 0
			var requests []string
			http.DefaultTransport = updateTestTransport(func(r *http.Request) (*http.Response, error) {
				requests = append(requests, r.URL.String())
				status, body := http.StatusOK, manifest
				switch {
				case r.URL.Host == "api.github.com":
					pageNumber := apiReads + 1
					if prefix != "v" && apiReads > 0 {
						pageNumber-- // ResolveFeed already read the first page.
					}
					if r.URL.Path != "/repos/me/app/releases" || r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != fmt.Sprint(pageNumber) || pageNumber > len(tc.pages) {
						t.Errorf("unexpected API request: %s", r.URL)
						return nil, fmt.Errorf("unexpected API request: %s", r.URL)
					}
					body = []byte(tc.pages[pageNumber-1])
					apiReads++
					if tc.apiStatus != 0 {
						status = tc.apiStatus
					}
				case r.URL.Path == latestPath || r.URL.Host == "downloads.example.com":
					status = tc.latestStatus
					if status == 0 {
						status = http.StatusNotFound
					}
					if tc.latestBody != "" {
						body = []byte(tc.latestBody)
					}
				case r.URL.Path == "/me/app/releases/download/"+prefix+"1.1.0/"+name:
					if tc.candidateCode != 0 {
						status = tc.candidateCode
					}
				case r.URL.Path == "/me/app/releases/download/v1.1.6/"+name:
					status = http.StatusNotFound
				case r.URL.Path == "/me/app/releases/download/v1.1.5/"+name:
					body = []byte(`{"version":"2.0.0"}`)
				case r.URL.Path == "/me/app/releases/download/v1.1.0/app.tar.gz":
					body = archives[0]
				case r.URL.Path == "/me/app/releases/download/v1.0.0/app.tar.gz":
					body = archives[1]
				default:
					t.Errorf("unexpected request: %s", r.URL)
					return nil, fmt.Errorf("unexpected request: %s", r.URL)
				}
				return &http.Response{
					StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)),
					Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header), Request: r,
				}, nil
			})
			c := &Config{
				root: t.TempDir(), Name: "App", Slug: "app", Version: "1.2.0",
				Updates: &Updates{GitHub: "me/app", TagPrefix: prefix, PublicKey: base64.StdEncoding.EncodeToString(pub)},
			}
			if tc.customURL {
				c.Updates.GitHub, c.Updates.URL = "", "https://downloads.example.com/app"
			}
			stage := t.TempDir()
			if _, err := writeArchive(c, stage, target, app(stage, c.Version)); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(stage, name))
			if err != nil {
				t.Fatal(err)
			}
			var m update.Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			if len(m.Deltas) != tc.wantDeltas || len(m.Previous) != tc.wantPrevious || apiReads != tc.wantPages {
				t.Fatalf("deltas %v, previous %v, API pages %d; want %d, %d, %d; requests %v", m.Deltas, m.Previous, apiReads, tc.wantDeltas, tc.wantPrevious, tc.wantPages, requests)
			}
			for i, a := range m.Previous {
				if a != bases[i] {
					t.Errorf("previous archive = %+v, want %+v", a, bases[i])
				}
			}
			for i, d := range m.Deltas {
				if d.From != bases[i].Version {
					t.Errorf("delta from %s, want %s", d.From, bases[i].Version)
				}
				f, err := os.Open(filepath.Join(stage, filepath.Base(d.URL)))
				if err != nil {
					t.Fatal(err)
				}
				oldDir := t.TempDir()
				app(oldDir, d.From)
				out := filepath.Join(t.TempDir(), "updated")
				err = update.ApplyDelta(f, d.Size, d.From, c.Version, filepath.Join(oldDir, "App.app"), out)
				f.Close()
				if err != nil {
					t.Fatal(err)
				}
				got, err := os.ReadFile(filepath.Join(out, "app"))
				if err != nil || !bytes.Equal(got, append(slices.Clone(code), c.Version...)) {
					t.Fatalf("delta from %s reconstructed another app: %v", d.From, err)
				}
			}
		})
	}
}
