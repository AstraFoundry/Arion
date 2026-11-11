package main

import (
	"archive/tar"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/update"
)

func TestArchVersion(t *testing.T) {
	for v, want := range map[string]string{
		"1.2.0":                "1.2.0",
		"v1.2.0":               "1.2.0",
		"1.2.0-beta.1":         "1.2.0beta.1",
		"1.2.0-rc.1":           "1.2.0rc.1",
		"1.2.0-alpha-2":        "1.2.0alpha_2",
		"1.2.0-1":              "1.2.0pre1",
		"1.2.0-0.3.7":          "1.2.0pre0.3.7",
		"1.2.0+build.5":        "1.2.0_build.5",
		"1.2.0-beta.1+exp.sha": "1.2.0beta.1_exp.sha",
		"1.2 final":            "1.2_final",
		"":                     "0",
	} {
		if got := archVersion(v); got != want {
			t.Errorf("archVersion(%q) = %q, want %q", v, got, want)
		}
	}
	// vercmp, when it is there, sorts them as semantic versioning does.
	vercmp, err := exec.LookPath("vercmp")
	if err != nil {
		return
	}
	sorted := []string{"1.1.9", "1.2.0-alpha", "1.2.0-beta", "1.2.0-beta.1", "1.2.0-beta.2", "1.2.0-beta.10", "1.2.0-rc.1", "1.2.0", "1.2.1-beta.1", "1.2.1"}
	for i := 1; i < len(sorted); i++ {
		a, b := archVersion(sorted[i-1]), archVersion(sorted[i])
		out, err := exec.Command(vercmp, a, b).Output()
		if err != nil || strings.TrimSpace(string(out)) != "-1" {
			t.Errorf("vercmp %s %s = %s, %v; want -1", a, b, out, err)
		}
	}
}

func TestArchPkgname(t *testing.T) {
	for slug, want := range map[string]string{"my-app": "my-app", "cloud-music": "cloud-music", "My App": "my-app", "-app": "app", ".app_2": "app_2", "云": "app"} {
		if got := archPkgname(slug); got != want {
			t.Errorf("archPkgname(%q) = %q, want %q", slug, got, want)
		}
	}
}

func TestArchDepends(t *testing.T) {
	for _, tc := range []struct {
		webview           *bool
		depends           []string
		want, wantUnknown []string
	}{
		{nil, nil, []string{"gtk3", "webkit2gtk-4.1"}, nil},
		{new(false), nil, []string{"gtk3"}, nil},
		{new(false), []string{"libnotify4", "libasound2t64 (>= 1.2)", "libfoo1 | libsecret-1-0:any", "libgtk-3-0", "libfoo2"}, []string{"gtk3", "libnotify", "alsa-lib", "libsecret"}, []string{"libfoo2"}},
	} {
		l := Linux{WebView: tc.webview, Depends: tc.depends}
		got, unknown := l.archDepends()
		if !slices.Equal(got, tc.want) || !slices.Equal(unknown, tc.wantUnknown) {
			t.Errorf("Debian %q: depends %q, unknown %q; want %q, %q", l.debDepends(), got, unknown, tc.want, tc.wantUnknown)
		}
	}
}

func TestArchLicenseAndURL(t *testing.T) {
	for _, tc := range []struct {
		config       string
		packageJSON  string
		github       string
		license, url string
	}{
		{"", "", "", "LicenseRef-custom", ""},
		{"", `{"license": "MIT", "homepage": "https://example.com"}`, "me/app", "MIT", "https://example.com"},
		{"", `{"license": "(MIT OR Apache-2.0)"}`, "me/app", "MIT OR Apache-2.0", "https://github.com/me/app"},
		{"", `{"license": "UNLICENSED"}`, "", "LicenseRef-custom", ""},
		{"", `{"license": "SEE LICENSE IN LICENSE.txt"}`, "", "LicenseRef-custom", ""},
		{"", `{"license": {"type": "MIT"}}`, "", "LicenseRef-custom", ""},
		// mygo.json's license wins over the one of package.json, which a
		// frontend may declare for npm rather than for the app.
		{"MIT", "", "", "MIT", ""},
		{"(MPL-2.0)", `{"license": "MIT"}`, "", "MPL-2.0", ""},
		{"UNLICENSED", "", "", "LicenseRef-custom", ""},
	} {
		c := &Config{root: t.TempDir(), License: tc.config}
		if tc.packageJSON != "" {
			if err := os.WriteFile(filepath.Join(c.root, "package.json"), []byte(tc.packageJSON), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if tc.github != "" {
			c.Updates = &Updates{GitHub: tc.github}
		}
		if license, url := archLicense(c), archURL(c); license != tc.license || url != tc.url {
			t.Errorf("mygo.json license %q, package.json %s, updates.github %q: license %q, url %q",
				tc.config, tc.packageJSON, tc.github, license, url)
		}
	}
}

// TestPKGBUILD writes the PKGBUILD of an app, as a build does, and checks
// its fields and checksums, then runs its package() as makepkg would and
// compares what it installs with the Debian package.
func TestPKGBUILD(t *testing.T) {
	root := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300))
	for i := range img.Pix {
		img.Pix[i] = uint8(i)
	}
	img.Set(0, 0, color.NRGBA{0, 0, 0, 0})
	f, err := os.Create(filepath.Join(root, "icon.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"license": "MIT", "homepage": "https://example.com/note-pad"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Config{
		root: root, Name: "Note's Pad", Slug: "note-pad", Version: "1.2.0-beta.1", Icon: "icon.png",
		URLSchemes:       []string{"notepad"},
		FileAssociations: []FileAssociation{{Ext: []string{"npad"}, Name: "Note <Pad>"}},
	}
	c.Linux.Comment = "Take $notes `now` & 'then'"
	c.Linux.Command = commandNames{"note-pad", "np"}
	c.Linux.Depends = []string{"libnotify4"}
	c.Linux.Maintainer = "Jane Doe <jane@example.com>"

	// The app as a build stages it.
	stage := t.TempDir()
	for name, mode := range map[string]fs.FileMode{"note-pad": 0o755, "tool.sh": 0o700, "data/it's a file.txt": 0o600, "-v": 0o644} {
		p := filepath.Join(stage, name)
		if os.MkdirAll(filepath.Dir(p), 0o755) != nil || os.WriteFile(p, []byte(name), mode) != nil {
			t.Fatal("cannot write " + name)
		}
	}
	if err := os.Symlink("data", filepath.Join(stage, "data-link")); err != nil {
		t.Fatal(err)
	}
	if _, err := writeLinuxDesktop(c, stage, "note-pad"); err != nil {
		t.Fatal(err)
	}
	app, err := entryNames(stage)
	if err != nil {
		t.Fatal(err)
	}
	deb, err := writeDeb(c, stage, "note-pad", "amd64", app)
	if err != nil {
		t.Fatal(err)
	}
	archives, err := writeArchive(c, stage, "linux-amd64", app)
	if err != nil {
		t.Fatal(err)
	}
	written, err := writePKGBUILD(c, stage, "note-pad", "amd64", app, archives[0])
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, w := range written {
		names = append(names, filepath.Base(w))
	}
	if want := []string{"note-pad-256x256.png", "note-pad-512x512.png", pkgbuildName}; !slices.Equal(names, want) {
		t.Errorf("wrote %q, want %q", names, want)
	}
	pkgbuild := filepath.Join(stage, pkgbuildName)

	// The PKGBUILD is not published: it builds from the archive next to it,
	// and every architecture has one of the same name.
	if files, _, _ := uploads(append([]string{deb}, append(archives, written...)...)); slices.ContainsFunc(files, func(f string) bool { return !strings.HasSuffix(f, ".deb") && !strings.HasSuffix(f, ".tar.gz") }) {
		t.Errorf("uploads %q", files)
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash to read the PKGBUILD")
	}
	if out, err := exec.Command(bash, "-n", pkgbuild).CombinedOutput(); err != nil {
		t.Fatalf("bash -n PKGBUILD: %v\n%s", err, out)
	}
	out, err := exec.Command(bash, "-c", `set -eu; source "$1"; printf '%s\n' "$pkgname" "$pkgver" "$pkgrel" "$pkgdesc" "$url" "${arch[*]}" "${license[*]}" "${depends[*]}" "${options[*]}" "${source[*]}" "${sha256sums[*]}"`, "-", pkgbuild).Output()
	if err != nil {
		t.Fatalf("reading the PKGBUILD: %v", err)
	}
	fields := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	want := []string{
		"note-pad", "1.2.0beta.1", "1", "Take $notes `now` & 'then'", "https://example.com/note-pad",
		"x86_64", "MIT", "gtk3 webkit2gtk-4.1 libnotify", "!strip !debug",
		"note-pad-1.2.0-beta.1-linux-amd64.tar.gz note-pad-256x256.png note-pad-512x512.png",
	}
	if len(fields) != len(want)+1 || !slices.Equal(fields[:len(want)], want) {
		t.Fatalf("PKGBUILD fields %q, want %q", fields, want)
	}
	sources, sums := strings.Fields(fields[len(want)-1]), strings.Fields(fields[len(want)])
	for i, s := range sources {
		sum, err := fileSHA256(filepath.Join(stage, s))
		if err != nil || i >= len(sums) || sums[i] != sum {
			t.Errorf("sha256sums of %s: %q, want %s (%v)", s, sums, sum, err)
		}
	}

	if runtime.GOOS != "linux" {
		return // package() runs the GNU tools of Arch Linux
	}
	// makepkg extracts the archive into srcdir, with links to the other
	// sources, and runs package() there with a umask of 022.
	srcdir, pkgdir := t.TempDir(), t.TempDir()
	a, err := os.Open(archives[0])
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := update.ExtractArchive(a, srcdir); err != nil {
		t.Fatal(err)
	}
	for _, s := range sources {
		if err := os.Symlink(filepath.Join(stage, s), filepath.Join(srcdir, s)); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(bash, "-c", `set -eu; umask 022; source "$1"; package`, "-", pkgbuild)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "srcdir="+srcdir, "pkgdir="+pkgdir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("package(): %v\n%s", err, out)
	}

	// The same entries as the Debian package, with the same modes and
	// contents.
	installed := map[string]string{}
	err = filepath.WalkDir(pkgdir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == pkgdir {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(pkgdir, p)
		rel = filepath.ToSlash(rel)
		switch {
		case d.IsDir():
			installed[rel+"/"] = fmt.Sprintf("dir %o", info.Mode().Perm())
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			installed[rel] = "link " + target
			return err
		default:
			b, err := os.ReadFile(p)
			installed[rel] = fmt.Sprintf("file %o %s", info.Mode().Perm(), b)
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	packaged := map[string]string{}
	debEntries(t, deb, "data.tar.gz", func(hdr *tar.Header, content []byte) {
		name := strings.TrimPrefix(hdr.Name, "./")
		switch hdr.Typeflag {
		case tar.TypeDir:
			packaged[name] = fmt.Sprintf("dir %o", hdr.Mode)
		case tar.TypeSymlink:
			packaged[name] = "link " + hdr.Linkname
		default:
			packaged[name] = fmt.Sprintf("file %o %s", hdr.Mode, content)
		}
	})
	if !maps.Equal(installed, packaged) {
		for _, name := range slices.Sorted(maps.Keys(packaged)) {
			if installed[name] != packaged[name] {
				t.Errorf("%s: package() installs %.80q, the Debian package has %.80q", name, installed[name], packaged[name])
			}
		}
		for _, name := range slices.Sorted(maps.Keys(installed)) {
			if _, ok := packaged[name]; !ok {
				t.Errorf("package() installs %s, which the Debian package has not", name)
			}
		}
	}
	for _, name := range []string{"opt/note-pad/note-pad", "opt/note-pad/-v", "usr/bin/np", "usr/share/applications/note-pad.desktop", "usr/share/mime/packages/note-pad.xml", "usr/share/icons/hicolor/512x512/apps/note-pad.png"} {
		if _, ok := installed[name]; !ok {
			t.Errorf("package() does not install %s", name)
		}
	}
}
