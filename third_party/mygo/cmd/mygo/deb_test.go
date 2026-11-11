package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/update"
)

// debData returns the files of the data archive of a Debian package: their
// contents, and for links the target.
func debData(t *testing.T, deb string) map[string]string { return debMember(t, deb, "data.tar.gz") }

// debMember returns the files of an archive of a Debian package, such as
// control.tar.gz or data.tar.gz.
func debMember(t *testing.T, deb, archive string) map[string]string {
	t.Helper()
	files := map[string]string{}
	debEntries(t, deb, archive, func(hdr *tar.Header, content []byte) {
		if hdr.Typeflag == tar.TypeSymlink {
			content = []byte(hdr.Linkname)
		}
		files[strings.TrimPrefix(hdr.Name, "./")] = string(content)
	})
	return files
}

// debEntries calls f with each entry of an archive of a Debian package and
// its content.
func debEntries(t *testing.T, deb, archive string, f func(hdr *tar.Header, content []byte)) {
	t.Helper()
	b, err := os.ReadFile(deb)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := bytes.CutPrefix(b, []byte("!<arch>\n"))
	if !ok {
		t.Fatal("not an ar archive")
	}
	for len(b) >= 60 {
		name := strings.TrimSpace(string(b[:16]))
		size, err := strconv.Atoi(strings.TrimSpace(string(b[48:58])))
		if err != nil {
			t.Fatal(err)
		}
		member := b[60 : 60+size]
		b = b[60+size+size%2:]
		if name != archive {
			continue
		}
		gz, err := gzip.NewReader(bytes.NewReader(member))
		if err != nil {
			t.Fatal(err)
		}
		tr := tar.NewReader(gz)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			content, _ := io.ReadAll(tr)
			f(hdr, content)
		}
	}
	t.Fatal("no " + archive)
}

func TestDebCommand(t *testing.T) {
	for _, commands := range []commandNames{nil, {"my-app-gui"}, {"my-app", "ma"}} {
		c := &Config{root: t.TempDir(), Name: "My App", Version: "1.0.0"}
		c.Linux.Command = commands
		stage := t.TempDir()
		if err := os.WriteFile(filepath.Join(stage, "my-app"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		deb, err := writeDeb(c, stage, "my-app", "amd64", []string{"my-app"})
		if err != nil {
			t.Fatal(err)
		}
		files := debData(t, deb)
		var linked []string
		for name, target := range files {
			if rest, ok := strings.CutPrefix(name, "usr/bin/"); ok && rest != "" {
				linked = append(linked, rest)
				if target != "../../opt/my-app/my-app" {
					t.Errorf("with the commands %q, %s links to %q", commands, name, target)
				}
			}
		}
		slices.Sort(linked)
		if !slices.Equal(linked, slices.Sorted(slices.Values(commands))) {
			t.Errorf("with the commands %q, the package has %v", commands, linked)
		}
		if entry := files["usr/share/applications/my-app.desktop"]; !strings.Contains(entry, "\nExec=/opt/my-app/my-app\n") {
			t.Errorf("desktop entry:\n%s", entry)
		}
	}
}

// An app that shows no web page depends on GTK alone, and install.sh does
// not ask for WebKitGTK (#551).
func TestLinuxWebView(t *testing.T) {
	for _, webview := range []*bool{nil, new(true), new(false)} {
		c := &Config{root: t.TempDir(), Name: "My App", Version: "1.0.0"}
		c.Linux.WebView = webview
		c.Linux.Depends = []string{"libayatana-appindicator3-1"}
		stage := t.TempDir()
		if err := os.WriteFile(filepath.Join(stage, "my-app"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		deb, err := writeDeb(c, stage, "my-app", "amd64", []string{"my-app"})
		if err != nil {
			t.Fatal(err)
		}
		want := "\nDepends: libgtk-3-0 | libgtk-3-0t64, libwebkit2gtk-4.1-0, libayatana-appindicator3-1\n"
		asks := true
		if webview != nil && !*webview {
			want = "\nDepends: libgtk-3-0 | libgtk-3-0t64, libayatana-appindicator3-1\n"
			asks = false
		}
		if control := debMember(t, deb, "control.tar.gz")["control"]; !strings.Contains(control, want) {
			t.Errorf("webview %v: control:\n%s", c.Linux.webView(), control)
		}
		if got := strings.Contains(installScript(c), "if ! has_webkit; then"); got != asks {
			t.Errorf("webview %v: install.sh checks for WebKitGTK: %v", c.Linux.webView(), got)
		}
	}
}

func TestLinuxCommandName(t *testing.T) {
	for _, tc := range []struct {
		command commandNames
		ok      bool
	}{
		{nil, true},
		{commandNames{"my-app"}, true},
		{commandNames{"my_app2.1+"}, true},
		{commandNames{"my-app", "ma"}, true},
		{commandNames{"-v"}, false},
		{commandNames{"my app"}, false},
		{commandNames{"../app"}, false},
		{commandNames{"bin/app"}, false},
		{commandNames{"my-app", ""}, false},
		{commandNames{"ma", "my-app", "ma"}, false},
	} {
		c := &Config{}
		c.Linux.Command = tc.command
		if err := c.validate(); (err == nil) != tc.ok {
			t.Errorf("linux.command %q: %v", tc.command, err)
		}
	}
}

// linux.command takes a name, or a list of them.
func TestLinuxCommandJSON(t *testing.T) {
	load := func(command string) (*Config, error) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, jsonConfig), []byte(`{"name": "My App", "linux": {"command": `+command+`}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		return loadConfig(dir)
	}
	for command, want := range map[string]commandNames{`"my-app"`: {"my-app"}, `["my-app", "ma"]`: {"my-app", "ma"}, `""`: nil, `[]`: nil, `null`: nil} {
		c, err := load(command)
		if err != nil {
			t.Fatalf("linux.command %s: %v", command, err)
		}
		if !slices.Equal(c.Linux.Command, want) {
			t.Errorf("linux.command %s loaded as %q", command, c.Linux.Command)
		}
	}
	// Anything else fails, naming the field.
	for _, command := range []string{`1`, `["my-app", 1]`} {
		if _, err := load(command); err == nil || !strings.Contains(err.Error(), "linux.command") {
			t.Errorf("linux.command %s: %v", command, err)
		}
	}
}

// TestBuildLinuxSlug builds an app with a slug of its own, which names the
// executable, the Debian package and its directory, the update archive,
// the PKGBUILD and install.sh's files and variables.
func TestBuildLinuxSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := t.TempDir()
	data := templateData{Name: "云音乐", Slug: "cloud-music", Module: "cloud-music", Identifier: "com.example.cloudmusic"}
	if err := writeTemplate(dir, nativeTemplate, data); err != nil {
		t.Fatal(err)
	}
	config := `{"name": "云音乐", "slug": "cloud-music", "identifier": "com.example.cloudmusic", "out": "build", "linux": {"webview": false}}`
	if err := os.WriteFile(filepath.Join(dir, jsonConfig), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	checkout, _ := filepath.Abs(filepath.Join("..", ".."))
	sum, _ := os.ReadFile(filepath.Join(checkout, "go.sum"))
	gomod := "module cloud-music\n\ngo " + goVersion() + "\n\nrequire github.com/egoist/mygo v0.0.0\n\nreplace github.com/egoist/mygo => " + checkout + "\n"
	if os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644) != nil || os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644) != nil {
		t.Fatal("cannot write the module")
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOWORK", "off")
	t.Setenv("MYGO_UPDATER_PRIVATE_KEY", "")
	if err := runBuild([]string{"-platform", "linux/amd64", dir}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "build", "linux-amd64")
	for _, name := range []string{"cloud-music", "cloud-music.desktop", "cloud-music-0.1.0-linux-amd64.tar.gz", "cloud-music_0.1.0_amd64.deb", pkgbuildName} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Error(err)
		}
	}
	if t.Failed() {
		return
	}
	files := debData(t, filepath.Join(out, "cloud-music_0.1.0_amd64.deb"))
	if _, ok := files["opt/cloud-music/cloud-music"]; !ok {
		t.Error("the package has no opt/cloud-music/cloud-music")
	}
	// The list of what the version installs, from which its update knows
	// what to remove, is in the app's archive.
	f, err := os.Open(filepath.Join(out, "cloud-music-0.1.0-linux-amd64.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	unpacked := filepath.Join(t.TempDir(), "app")
	err = update.ExtractArchive(f, unpacked)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	list, err := os.ReadFile(filepath.Join(unpacked, update.FileList))
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := entryNames(unpacked); len(entries) < 2 || !slices.Equal(update.ParseFileList(list), entries) {
		t.Errorf("the list names %q, the archive holds %q", list, entries)
	}
	if !strings.HasPrefix(debMember(t, filepath.Join(out, "cloud-music_0.1.0_amd64.deb"), "control.tar.gz")["control"], "Package: cloud-music\n") {
		t.Error("the package is not named cloud-music")
	}
	pkgbuild, err := os.ReadFile(filepath.Join(out, pkgbuildName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkgbuild), "\npkgname='cloud-music'\n") || !strings.Contains(string(pkgbuild), "\nsource=('cloud-music-0.1.0-linux-amd64.tar.gz')\n") || !strings.Contains(string(pkgbuild), "\ndepends=('gtk3')\n") {
		t.Errorf("the PKGBUILD does not package cloud-music:\n%s", pkgbuild)
	}
	script, err := os.ReadFile(filepath.Join(out, installScriptName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "\nname='cloud-music'\n") {
		t.Errorf("install.sh does not install cloud-music:\n%s", script)
	}
	// Its variables for tests, which it has with updates.
	c := &Config{Name: "云音乐", Slug: "cloud-music", Updates: &Updates{PublicKey: "key", GitHub: "me/cloud-music"}}
	if script := installScript(c); !strings.Contains(script, "${CLOUD_MUSIC_DOWNLOAD_URL:-}") {
		t.Errorf("install.sh has no CLOUD_MUSIC_DOWNLOAD_URL:\n%s", script)
	}
}

func TestSlug(t *testing.T) {
	for slug, ok := range map[string]bool{"": true, "my-app": true, "app2": true, "My-App": false, "-app": false, "app-": false, "my--app": false, "my_app": false, "云音乐": false} {
		c := &Config{Slug: slug}
		if err := c.validate(); (err == nil) != ok {
			t.Errorf("slug %q: %v", slug, err)
		}
	}
}

// TestSlugify gives names in Chinese, Japanese and Korean slugs of their
// readings, which they had none of before but "app" or what Latin they have.
func TestSlugify(t *testing.T) {
	for name, want := range map[string]string{
		"Hello, World!": "hello-world",
		"!!!":           "app",
		"MyGo 云音乐":      "mygo-yun-yin-le",
		"云音乐Pro 2":      "yun-yin-le-pro-2",
		"ラーメン":          "ramen",
		"とうきょう":         "toukyou",
		"キャッシュ・マッチ":     "kyasshu-matchi",
		"じゃんけん":         "janken",
		"ファイル":          "fairu",
		"ウィキ":           "wiki",
		"서울":            "seoul",
		"한국어 사전":        "hangugeo-sajeon",
		"東京タワー":         "dong-jing-tawa",
	} {
		if got := slugify(name); got != want {
			t.Errorf("slugify(%q) = %q, want %q", name, got, want)
		}
	}
	if len(kanaRomaji) != 0x3096-0x3041+1 {
		t.Errorf("%d romanizations of kana, for %d", len(kanaRomaji), 0x3096-0x3041+1)
	}
}
