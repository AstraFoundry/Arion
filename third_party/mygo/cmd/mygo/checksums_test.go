package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestChecksums(t *testing.T) {
	dist := t.TempDir()
	var artifacts []string
	for _, name := range []string{"app-1.2.0.dmg", "app-setup-1.2.0-amd64.exe", "app-1.2.0-linux-amd64.tar.gz", "update-linux-amd64.json", "App.exe", "linux-amd64/install.sh", "linux-arm64/install.sh"} {
		p := filepath.Join(dist, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("hello"), 0o644)
		artifacts = append(artifacts, p)
	}
	path, err := writeChecksums(dist, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	const hello = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	want := hello + "  app-1.2.0-linux-amd64.tar.gz\n" + hello + "  app-1.2.0.dmg\n" + hello + "  app-setup-1.2.0-amd64.exe\n" + hello + "  install.sh\n"
	if b, _ := os.ReadFile(path); string(b) != want {
		t.Errorf("SHA256SUMS:\n%s\nwant\n%s", b, want)
	}
	if _, err := exec.LookPath("shasum"); err == nil && runtime.GOOS != "windows" {
		cmd := exec.Command("shasum", "-a", "256", "-c", checksumsName)
		cmd.Dir = filepath.Join(dist, "linux-amd64")
		os.WriteFile(filepath.Join(cmd.Dir, checksumsName), []byte(hello+"  install.sh\n"), 0o644)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("shasum -c: %v\n%s", err, out)
		}
	}

	// The lines of an earlier upload stay for the files this one lacks.
	other := strings.Repeat("a", 64)
	if err := mergeChecksums(path, []byte(other+"  app-setup-1.2.0-arm64.exe\r\n"+other+"  install.sh\njunk\n"), ""); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), other+"  app-setup-1.2.0-arm64.exe\n") || !strings.Contains(string(b), hello+"  install.sh\n") || strings.Contains(string(b), "junk") {
		t.Errorf("merged SHA256SUMS:\n%s", b)
	}

	if path, err := writeChecksums(dist, nil); path != "" || err != nil {
		t.Errorf("with nothing to publish: %q, %v", path, err)
	}
}

func TestPublishGitHubChecksums(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as gh")
	}
	bin, dist := t.TempDir(), t.TempDir()
	other := strings.Repeat("b", 64)
	// gh: the release exists, with the SHA256SUMS of another platform.
	script := "#!/bin/sh\ncase \"$2 $*\" in\n\"view \"*--json*) printf 'app-setup-1.2.0-arm64.exe\\nSHA256SUMS\\n' ;;\ndownload*) [ -z \"$GH_FAIL\" ] || exit 1; for a; do last=$a; done; printf '" + other + "  app-setup-1.2.0-arm64.exe\\n' > \"$last/SHA256SUMS\" ;;\nesac\n"
	os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	exe := filepath.Join(dist, "app-setup-1.2.0-amd64.exe")
	os.WriteFile(exe, []byte("hello"), 0o644)
	sums, err := writeChecksums(dist, []string{exe})
	if err != nil {
		t.Fatal(err)
	}
	c := &Config{root: t.TempDir(), Version: "1.2.0", Updates: &Updates{GitHub: "me/app", TagPrefix: "v"}}
	if err := publishGitHub(c, []string{exe, sums}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(sums)
	if !strings.Contains(string(b), other+"  app-setup-1.2.0-arm64.exe\n") || !strings.Contains(string(b), "  app-setup-1.2.0-amd64.exe\n") {
		t.Errorf("SHA256SUMS:\n%s", b)
	}
	// The release's SHA256SUMS that cannot be downloaded stops the upload,
	// which would drop its lines.
	t.Setenv("GH_FAIL", "1")
	if err := publishGitHub(c, []string{exe, sums}); err == nil || !strings.Contains(err.Error(), "gh release download") {
		t.Errorf("with the release's SHA256SUMS failing to download: %v", err)
	}
}

func TestInstallPS1(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	key := base64.StdEncoding.EncodeToString(pub)
	for _, tc := range []struct {
		updates Updates
		want    []string
	}{
		{Updates{PublicKey: key, GitHub: "me/my-app", TagPrefix: "v"}, []string{
			"#   irm https://github.com/me/my-app/releases/latest/download/install.ps1 | iex\n",
			"$releases = 'https://github.com/me/my-app/releases/latest/download/'\n",
			"$api = ''\n",
			"$github = $true\n",
			"$assetsApi = 'https://api.github.com/repos/me/my-app/releases/latest'\n",
		}},
		{Updates{PublicKey: key, GitHub: "me/my-app", TagPrefix: "desktop-v"}, []string{
			"#   irm https://github.com/me/my-app/releases/download/desktop-v1.2.0/install.ps1 | iex\n",
			"$releases = 'https://github.com/me/my-app/releases/download/'\n",
			"$api = 'https://api.github.com/repos/me/my-app/releases?per_page=100'\n",
			"$tagPrefix = 'desktop-v'\n",
			"$assetsApi = 'https://api.github.com/repos/me/my-app/releases/tags/'\n",
		}},
		{Updates{PublicKey: key, URL: "https://dl.example.com/my-app"}, []string{
			"$releases = 'https://dl.example.com/my-app/'\n",
			"$github = $false\n",
		}},
	} {
		c := &Config{Name: "Jane's App", Identifier: "com.example.app", Version: "1.2.0", Updates: &tc.updates}
		script := installPS1(c)
		for _, line := range append(tc.want,
			"$appName = 'Jane''s App'\n",
			"$exeName = 'Jane''s App.exe'\n",
			"$setupPrefix = 'jane-s-app-setup-'\n",
			"$identifier = 'com.example.app'\n",
			"$downloadUrl = $env:JANE_S_APP_DOWNLOAD_URL\n",
		) {
			if !strings.Contains(script, line) {
				t.Errorf("install.ps1 of %+v has no %q", tc.updates, line)
			}
		}
		// It stops the app's processes, those running from its folder, that
		// do not quit when asked.
		for _, line := range []string{
			"$_.ExecutablePath.StartsWith($dir, [StringComparison]::OrdinalIgnoreCase)",
			"$running | Stop-Process -Force",
			"Wait-Exit $running 10\n",
		} {
			if !strings.Contains(script, line) {
				t.Errorf("install.ps1 of %+v has no %q", tc.updates, line)
			}
		}
		// Windows PowerShell 5.1 outputs a JSON array as one object.
		if !strings.Contains(script, `-Uri "$api&page=$page" | ForEach-Object { $_ })`) {
			t.Errorf("install.ps1 of %+v does not enumerate the releases", tc.updates)
		}
		if strings.Contains(script, "60000") {
			t.Errorf("install.ps1 of %+v waits a minute for the app to quit", tc.updates)
		}
		// Windows PowerShell 5.1 reads a script without a BOM in the code
		// page of the system.
		for i, r := range script {
			if r >= 0x80 {
				t.Errorf("install.ps1 of %+v has %q at %d: not ASCII", tc.updates, r, i)
				break
			}
		}
		// The installer it looks for is the one the build names: the
		// prefix, the version, then -<arch>.exe.
		if version, ok := strings.CutPrefix(installerFileName(c, "arm64"), "jane-s-app-setup-"); !ok || version != "1.2.0-arm64.exe" || !strings.Contains(script, `'(\S+)-' + $arch + '\.exe)$'`) {
			t.Errorf("installer %s", installerFileName(c, "arm64"))
		}
		if pwsh, err := exec.LookPath("pwsh"); err == nil {
			path := filepath.Join(t.TempDir(), "install.ps1")
			os.WriteFile(path, []byte(script), 0o644)
			if out, err := exec.Command(pwsh, "-NoProfile", "-Command", "$null = [scriptblock]::Create((Get-Content -Raw '"+path+"'))").CombinedOutput(); err != nil {
				t.Errorf("pwsh cannot parse install.ps1: %v\n%s", err, out)
			}
		}
	}
	// Names of other characters than ASCII: their PowerShell strings stay
	// ASCII, as Windows PowerShell 5.1 reads the script that irm
	// downloads as Latin-1, and a typographic quote ends a string there.
	for s, want := range map[string]string{
		"":                     "''",
		"Jane's App":           "'Jane''s App'",
		"Jane\u2019s App":      "('' + 'Jane' + [char]0x2019 + 's App')",
		"\u5feb\u5199":         "('' + [char]0x5FEB + [char]0x5199)",
		"Caf\u00e9 \U0001F600": "('' + 'Caf' + [char]0x00E9 + ' ' + [char]0xD83D + [char]0xDE00)",
	} {
		if got := psQuote(s); got != want {
			t.Errorf("psQuote(%q) = %s, want %s", s, got, want)
		}
	}
	c := &Config{Name: "\u5feb\u5199 Caf\u00e9", Identifier: "com.example.app", Version: "1.2.0", Updates: &Updates{PublicKey: key, GitHub: "me/my-app"}}
	script := installPS1(c)
	for i, r := range script {
		if r >= 0x80 {
			t.Errorf("install.ps1 of %s has %q at %d: not ASCII", c.Name, r, i)
			break
		}
	}
	if !strings.Contains(script, "$appName = ('' + [char]0x5FEB + [char]0x5199 + ' Caf' + [char]0x00E9)\n") {
		t.Errorf("install.ps1 of %s names the app otherwise:\n%s", c.Name, script)
	}
	if got := envName("my-app"); got != "MY_APP" {
		t.Errorf("envName(my-app) = %s", got)
	}
	if got := envName("2do"); got != "APP_2DO" {
		t.Errorf("envName(2do) = %s", got)
	}
}

// TestPublishS3Checksums checks that an upload to updates.s3 keeps the
// lines of the SHA256SUMS published before, as of another platform, but
// those of the files of other versions that it replaces.
func TestPublishS3Checksums(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	old, other := strings.Repeat("a", 64), strings.Repeat("b", 64)
	published := old + "  app-setup-1.1.0-amd64.exe\n" + other + "  app-setup-1.2.0-arm64.exe\n" + old + "  app-1.1.0-linux-amd64.tar.gz\n" + old + "  install.ps1\n"
	var (
		mu       sync.Mutex
		uploaded string
		status   = http.StatusOK
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/app/SHA256SUMS":
			w.WriteHeader(status)
			if status == http.StatusOK {
				io.WriteString(w, published)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/downloads/app/SHA256SUMS":
			b, _ := io.ReadAll(r.Body)
			uploaded = string(b)
		case r.Method != http.MethodPut:
			t.Errorf("%s %s", r.Method, r.URL)
		}
	}))
	defer srv.Close()
	dist := t.TempDir()
	exe := filepath.Join(dist, "app-setup-1.2.0-amd64.exe")
	os.WriteFile(exe, []byte("hello"), 0o644)
	ps1 := filepath.Join(dist, "windows-amd64", "install.ps1")
	os.MkdirAll(filepath.Dir(ps1), 0o755)
	os.WriteFile(ps1, []byte("hello"), 0o644)
	c := &Config{root: t.TempDir(), Version: "1.2.0", Updates: &Updates{URL: srv.URL + "/app", S3: &S3{Bucket: "downloads", Prefix: "app", Endpoint: srv.URL}}}
	const hello = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	for _, tc := range []struct {
		status int
		want   string
	}{
		// The other platform's installer and the Linux archive of 1.1.0,
		// which this upload does not replace, stay.
		{http.StatusOK, other + "  app-setup-1.2.0-arm64.exe\n" + hello + "  app-setup-1.2.0-amd64.exe\n" + old + "  app-1.1.0-linux-amd64.tar.gz\n" + hello + "  install.ps1\n"},
		{http.StatusNotFound, hello + "  app-setup-1.2.0-amd64.exe\n" + hello + "  install.ps1\n"},
		{http.StatusForbidden, hello + "  app-setup-1.2.0-amd64.exe\n" + hello + "  install.ps1\n"},
	} {
		sums, err := writeChecksums(dist, []string{exe, ps1})
		if err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		status, uploaded = tc.status, ""
		mu.Unlock()
		if err := publish(c, []string{exe, ps1, sums}); err != nil {
			t.Fatal(err)
		}
		if got := parseChecksums([]byte(uploaded)); !maps.Equal(got, parseChecksums([]byte(tc.want))) {
			t.Errorf("with the published SHA256SUMS answering %d, uploaded\n%s\nwant\n%s", tc.status, uploaded, tc.want)
		}
	}
	// Another error stops the upload, as it would drop the lines of the
	// other builds.
	mu.Lock()
	status = http.StatusInternalServerError
	mu.Unlock()
	sums, _ := writeChecksums(dist, []string{exe, ps1})
	if err := publish(c, []string{exe, ps1, sums}); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("with the published SHA256SUMS failing: %v", err)
	}
}
