package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/update"
)

func TestInstallScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs install.sh")
	}
	if os.Geteuid() == 0 {
		t.Skip("install.sh refuses to install for root")
	}
	// uname answers as an x86-64 Linux machine would.
	bin := t.TempDir()
	uname := "#!/bin/sh\ncase \"$1\" in -s) echo Linux ;; -m) echo x86_64 ;; esac\n"
	if err := os.WriteFile(filepath.Join(bin, "uname"), []byte(uname), 0o755); err != nil {
		t.Fatal(err)
	}
	// ldconfig lists $LD_CACHE as the library cache, and fails without it.
	ldconfig := "#!/bin/sh\n[ \"$1\" = -p ] && [ -n \"${LD_CACHE:-}\" ] || exit 1\nprintf '%s' \"$LD_CACHE\"\n"
	if err := os.WriteFile(filepath.Join(bin, "ldconfig"), []byte(ldconfig), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LD_CACHE", "")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	home := filepath.Join(t.TempDir(), "Jane Doe")
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mygo.json"), []byte(`{
		"name": "My App",
		"icon": "icon.png",
		"urlSchemes": ["my-app"],
		"fileAssociations": [{"ext": ["note"], "name": "Note"}]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "icon.png"), []byte("png"), 0o644)
	// build stages a fake app of version and writes its archive and
	// install.sh, returning where they are.
	build := func(c *Config, version string) string {
		t.Helper()
		c.Version = version
		stage := filepath.Join(t.TempDir(), "linux-amd64")
		files := map[string]string{
			"my-app":                "#!/bin/sh\necho " + version + "\n",
			"only-" + version:       "",
			"resources/config.json": "{}",
		}
		for name, content := range files {
			os.MkdirAll(filepath.Join(stage, filepath.Dir(name)), 0o755)
			if err := os.WriteFile(filepath.Join(stage, name), []byte(content), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := writeLinuxDesktop(c, stage, "my-app"); err != nil {
			t.Fatal(err)
		}
		entries, _ := os.ReadDir(stage)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if _, err := writeArchive(c, stage, "linux-amd64", names); err != nil {
			t.Fatal(err)
		}
		if _, err := writeInstallScript(c, stage); err != nil {
			t.Fatal(err)
		}
		return stage
	}
	run := func(script string, args ...string) string {
		t.Helper()
		out, err := exec.Command("sh", append([]string{script}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("install.sh %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}
	appDir := filepath.Join(home, ".local", "my-app.app")
	binDir := filepath.Join(home, ".local", "bin")
	installed := func(version string) {
		t.Helper()
		if out, err := exec.Command(filepath.Join(appDir, "my-app")).Output(); err != nil || string(out) != version+"\n" {
			t.Errorf("my-app printed %q, %v, want %s", out, err, version)
		}
		if _, err := os.Stat(filepath.Join(appDir, "resources", "config.json")); err != nil {
			t.Error(err)
		}
		entries, _ := os.ReadDir(appDir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "only-") && e.Name() != "only-"+version {
				t.Errorf("%s of another version stayed", e.Name())
			}
		}
		entry, _ := os.ReadFile(filepath.Join(home, ".local", "share", "applications", "my-app.desktop"))
		for _, line := range []string{"Name=My App", `Exec="` + appDir + `/my-app" %U`, "Icon=" + appDir + "/my-app.png", "MimeType=application/x-my-app-note;x-scheme-handler/my-app;"} {
			if !strings.Contains(string(entry), line+"\n") {
				t.Errorf("the desktop entry has no %s:\n%s", line, entry)
			}
		}
		if mime, _ := os.ReadFile(filepath.Join(home, ".local", "share", "mime", "packages", "my-app.xml")); !strings.Contains(string(mime), `<glob pattern="*.note"/>`) {
			t.Errorf("MIME package:\n%s", mime)
		}
		// Updates register the entry again as install.sh does.
		os.WriteFile(filepath.Join(home, ".local", "share", "applications", "my-app.desktop"), append(entry, "X-Stale=true\n"...), 0o644)
		if err := update.RefreshDesktopEntry(appDir, "my-app"); err != nil {
			t.Fatal(err)
		}
		if again, _ := os.ReadFile(filepath.Join(home, ".local", "share", "applications", "my-app.desktop")); string(again) != string(entry) {
			t.Errorf("updates register the entry as\n%s\ninstall.sh as\n%s", again, entry)
		}
	}

	// Without updates, the archive next to the script.
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	local := build(c, "1.0.0")
	script, _ := os.ReadFile(filepath.Join(local, "install.sh"))
	if strings.Contains(string(script), "curl -fsSL http") {
		t.Errorf("install.sh without updates tells to download it:\n%s", script)
	}
	// Without linux.command, no command, and none left from earlier
	// scripts, which linked one named after the app.
	os.MkdirAll(binDir, 0o755)
	if err := os.Symlink(filepath.Join(appDir, "my-app"), filepath.Join(binDir, "my-app")); err != nil {
		t.Fatal(err)
	}
	if out := run(filepath.Join(local, "install.sh")); !strings.HasSuffix(out, "Installed My App: open it from the applications menu\n") {
		t.Errorf("install.sh printed:\n%s", out)
	}
	if entries, _ := os.ReadDir(binDir); len(entries) != 0 {
		t.Errorf("~/.local/bin holds %v", entries)
	}
	installed("1.0.0")

	// With updates, the latest version, which the manifest names.
	if _, err := exec.LookPath("curl"); err != nil {
		if _, err := exec.LookPath("wget"); err != nil {
			t.Skip("downloading needs curl or wget")
		}
	}
	opensslEd25519(t)
	pub, priv, _ := ed25519.GenerateKey(nil)
	t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(priv))
	served := t.TempDir()
	srv := httptest.NewServer(http.FileServer(http.Dir(served)))
	defer srv.Close()
	none := 0
	c.Updates = &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: srv.URL, Deltas: &none}
	c.Linux.Command = commandNames{"my-app", "ma"}
	published := build(c, "1.1.0")
	for _, name := range []string{"my-app-1.1.0-linux-amd64.tar.gz", "update-linux-amd64.json"} {
		b, err := os.ReadFile(filepath.Join(published, name))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(served, name), b, 0o644)
	}
	script, _ = os.ReadFile(filepath.Join(published, "install.sh"))
	if !strings.Contains(string(script), "curl -fsSL "+srv.URL+"/install.sh | sh") {
		t.Errorf("install.sh does not tell how to run it:\n%s", script)
	}
	if !strings.Contains(string(script), "the my-app and ma commands in ~/.local/bin,") {
		t.Errorf("install.sh does not say what it installs:\n%s", script)
	}
	alone := filepath.Join(t.TempDir(), "install.sh")
	os.WriteFile(alone, script, 0o755)
	if out := run(alone); !strings.Contains(out, "Downloading My App 1.1.0") || !strings.Contains(out, "run "+filepath.Join(binDir, "my-app")+"\n") {
		t.Errorf("install.sh printed:\n%s", out)
	}
	installed("1.1.0")
	// The commands of linux.command.
	for _, command := range c.Linux.Command {
		if out, err := exec.Command(filepath.Join(binDir, command)).Output(); err != nil || string(out) != "1.1.0\n" {
			t.Errorf("the %s command printed %q, %v", command, out, err)
		}
	}

	// A command that another program has is left alone, and the app runs
	// by the next.
	notes := filepath.Join(binDir, "notes")
	os.WriteFile(notes, []byte("#!/bin/sh\necho notes\n"), 0o755)
	c.Linux.Command = commandNames{"notes", "ma"}
	withNotes := t.TempDir()
	if _, err := writeInstallScript(c, withNotes); err != nil {
		t.Fatal(err)
	}
	if out := run(filepath.Join(withNotes, "install.sh"), filepath.Join(local, "my-app-1.0.0-linux-amd64.tar.gz")); !strings.Contains(out, "Leaving "+notes+" alone: it is not My App's\n") ||
		!strings.HasSuffix(out, "open it from the applications menu, or run "+filepath.Join(binDir, "ma")+"\n") {
		t.Errorf("install.sh printed:\n%s", out)
	}
	if b, _ := os.ReadFile(notes); string(b) != "#!/bin/sh\necho notes\n" {
		t.Errorf("the notes command became %q", b)
	}

	// The archive given, over the installed version; with ~/.local/bin on
	// the PATH, it says to run the command by its name.
	path := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+path)
	if out := run(alone, filepath.Join(local, "my-app-1.0.0-linux-amd64.tar.gz")); !strings.HasSuffix(out, "open it from the applications menu, or run my-app\n") {
		t.Errorf("install.sh printed:\n%s", out)
	}
	t.Setenv("PATH", path)
	installed("1.0.0")

	// It warns when the library cache has no WebKitGTK, and only then.
	libc := "\tlibc.so.6 (libc6,x86-64, OS ABI: Linux 3.2.0) => /lib/x86_64-linux-gnu/libc.so.6\n"
	for _, cache := range []struct {
		libs  string
		warns bool
	}{
		{libs: "", warns: false}, // no cache
		{libs: "0 libs found in cache `/etc/ld.so.cache'\n", warns: false},
		{libs: libc + "\tlibwebkit2gtk-4.1.so.0 (libc6,x86-64) => /lib/x86_64-linux-gnu/libwebkit2gtk-4.1.so.0\n", warns: false},
		{libs: libc + "\tlibwebkit2gtk-4.0.so.37 (libc6,x86-64) => /lib/x86_64-linux-gnu/libwebkit2gtk-4.0.so.37\n", warns: false},
		{libs: libc + "\tlibgtk-3.so.0 (libc6,x86-64) => /lib/x86_64-linux-gnu/libgtk-3.so.0\n", warns: true},
	} {
		t.Setenv("LD_CACHE", cache.libs)
		out := run(alone, filepath.Join(local, "my-app-1.0.0-linux-amd64.tar.gz"))
		if warns := strings.Contains(out, "My App needs WebKitGTK, which is not installed. Install it with:\n  "); warns != cache.warns {
			t.Errorf("with the library cache\n%s\ninstall.sh printed:\n%s", cache.libs, out)
		}
	}
	t.Setenv("LD_CACHE", "")
	installed("1.0.0")
	// The libraries it looks for are those the app loads.
	ffi, err := os.ReadFile(filepath.Join("..", "..", "internal", "linux", "ffi.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ffi), `open("libwebkit2gtk-4.1.so.0", "libwebkit2gtk-4.0.so.37")`) {
		t.Error("install.sh looks for other WebKitGTK libraries than the app loads")
	}

	// Uninstalling removes what install.sh and the app made, and only that:
	// with the links of the commands, one of a name the app no longer has.
	handler := filepath.Join(home, ".local", "share", "applications", "com.mygo.my-app.url-handler.desktop")
	other := filepath.Join(home, ".local", "share", "applications", "other.url-handler.desktop")
	os.WriteFile(handler, []byte(`Exec="`+appDir+`/my-app" %u`+"\n"), 0o644)
	os.WriteFile(other, []byte("Exec=/usr/bin/other %u\n"), 0o644)
	dropped := filepath.Join(binDir, "my-old-app")
	if err := os.Symlink(filepath.Join(appDir, "my-app"), dropped); err != nil {
		t.Fatal(err)
	}
	run(alone, "--uninstall")
	for _, p := range []string{appDir, filepath.Join(home, ".local", "bin", "my-app"), filepath.Join(home, ".local", "bin", "ma"), dropped, handler,
		filepath.Join(home, ".local", "share", "applications", "my-app.desktop"),
		filepath.Join(home, ".local", "share", "mime", "packages", "my-app.xml")} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("%s is still there", p)
		}
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("the entry of another app was removed: %v", err)
	}
	if _, err := os.Stat(notes); err != nil {
		t.Errorf("the command of another program was removed: %v", err)
	}
	if out, err := exec.Command("sh", alone, "--uninstall").CombinedOutput(); err == nil {
		t.Errorf("uninstalling twice succeeded:\n%s", out)
	}
}

// The opening comment of install.sh names the commands of linux.command.
func TestCommandsPhrase(t *testing.T) {
	for list, want := range map[string]string{
		"":          "",
		"my-app":    "the my-app command",
		"my-app ma": "the my-app and ma commands",
		"a b c":     "the a, b and c commands",
	} {
		if got := commandsPhrase(list); got != want {
			t.Errorf("commandsPhrase(%q) = %q, want %q", list, got, want)
		}
	}
}

func TestInstallScriptTagged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs install.sh")
	}
	if os.Geteuid() == 0 {
		t.Skip("install.sh refuses to install for root")
	}
	// uname answers as an x86-64 Linux machine would, and curl serves the
	// GitHub API and release downloads of me/my-app from $SERVED.
	bin := t.TempDir()
	scripts := map[string]string{
		"uname": "#!/bin/sh\ncase \"$1\" in -s) echo Linux ;; -m) echo x86_64 ;; esac\n",
		"curl": `#!/bin/sh
for url; do :; done
case "$url" in
"https://api.github.com/repos/me/my-app/releases?per_page=100&page="*) file="$SERVED/releases-${url##*page=}.json" ;;
"https://github.com/me/my-app/releases/download/"*) file="$SERVED/${url#https://github.com/me/my-app/releases/download/}" ;;
*) exit 22 ;;
esac
[ -f "$file" ] || exit 22
cat "$file"
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	served := t.TempDir()
	t.Setenv("SERVED", served)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	opensslEd25519(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	pub, priv, _ := ed25519.GenerateKey(nil)
	t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(priv))
	none := 0
	c := &Config{root: t.TempDir(), Name: "My App", Version: "1.1.0",
		Updates: &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), GitHub: "me/my-app", TagPrefix: "desktop-v", Deltas: &none}}
	stage := filepath.Join(t.TempDir(), "linux-amd64")
	os.MkdirAll(stage, 0o755)
	if err := os.WriteFile(filepath.Join(stage, "my-app"), []byte("#!/bin/sh\necho 1.1.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := writeArchive(c, stage, "linux-amd64", []string{"my-app"}); err != nil {
		t.Fatal(err)
	}
	script, err := writeInstallScript(c, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	release := filepath.Join(served, "desktop-v1.1.0")
	os.MkdirAll(release, 0o755)
	for _, name := range []string{"my-app-1.1.0-linux-amd64.tar.gz", "update-linux-amd64.json"} {
		b, err := os.ReadFile(filepath.Join(stage, name))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(release, name), b, 0o644)
	}
	// The API's releases, newest first, as it prints them.
	rel := func(tag string, draft, pre bool) string {
		return fmt.Sprintf("  {\n    \"tag_name\": %q,\n    \"name\": \"not \\\"tag_name\\\": \\\"desktop-v9.0.0\\\"\",\n    \"draft\": %t,\n    \"prerelease\": %t\n  }", tag, draft, pre)
	}
	page := func(releases ...string) {
		os.WriteFile(filepath.Join(served, "releases-1.json"), []byte("[\n"+strings.Join(releases, ",\n")+"\n]\n"), 0o644)
	}

	page(rel("cli-v2.0.0", false, false))
	if out, err := exec.Command("sh", script).CombinedOutput(); err == nil || !strings.Contains(string(out), "no release of My App is published") {
		t.Errorf("without a release, install.sh printed %v:\n%s", err, out)
	}
	page(rel("cli-v2.0.0", false, false), rel("desktop-v1.2.0", true, false), rel("desktop-v1.2.0-beta.1", false, true),
		rel("desktop-v1.1.0", false, false), rel("desktop-v1.0.0", false, false))
	out, err := exec.Command("sh", script).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Downloading My App 1.1.0") {
		t.Fatalf("install.sh printed %v:\n%s", err, out)
	}
	if out, err := exec.Command(filepath.Join(home, ".local", "my-app.app", "my-app")).Output(); err != nil || string(out) != "1.1.0\n" {
		t.Errorf("my-app printed %q, %v", out, err)
	}
}

func TestInstallScriptMacOS(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("installs on macOS")
	}
	if os.Geteuid() == 0 {
		t.Skip("install.sh refuses to install for root")
	}
	home := filepath.Join(t.TempDir(), "Jane Doe")
	apps := filepath.Join(home, "Applications")
	t.Setenv("HOME", home)
	t.Setenv("MY_APP_INSTALL_DIR", apps)
	t.Setenv("MY_APP_NO_OPEN", "1")
	pub, priv, _ := ed25519.GenerateKey(nil)
	t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(priv))
	served := t.TempDir()
	srv := httptest.NewServer(http.FileServer(http.Dir(served)))
	defer srv.Close()
	none := 0
	c := &Config{root: t.TempDir(), Name: "My App", Identifier: "com.example.mygo-install-test",
		Updates: &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: srv.URL, Deltas: &none}}
	c.applyDefaults()
	// publish builds the bundle of version, a script printing it, and
	// serves its archive and manifest.
	publish := func(version, target string) {
		t.Helper()
		c.Version = version
		stage := t.TempDir()
		bin := filepath.Join(stage, "exe")
		os.WriteFile(bin, []byte("#!/bin/sh\necho "+version+"\n"), 0o755)
		app, err := writeBundle(c, stage, bin, []byte("icns"), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(app, "Contents", "Resources", "only-"+version), nil, 0o644)
		if _, err := writeArchive(c, stage, target, []string{"My App.app"}); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"my-app-" + version + "-" + target + ".tar.gz", "update-" + target + ".json"} {
			b, err := os.ReadFile(filepath.Join(stage, name))
			if err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(served, name), b, 0o644)
		}
	}
	script := filepath.Join(t.TempDir(), "install.sh")
	// uname and sysctl answer as the Mac of machine (arm64 or x86_64)
	// would, translated by Rosetta when translated is "1".
	bin := t.TempDir()
	for name, body := range map[string]string{
		"uname":  "#!/bin/sh\ncase \"$1\" in -s) echo Darwin ;; -m) echo \"$FAKE_MACHINE\" ;; esac\n",
		"sysctl": "#!/bin/sh\n[ \"$2\" = sysctl.proc_translated ] && echo \"$FAKE_TRANSLATED\"\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	machine, translated := "arm64", "0"
	run := func(args ...string) (string, error) {
		cmd := exec.Command("sh", append([]string{script}, args...)...)
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
			"FAKE_MACHINE="+machine, "FAKE_TRANSLATED="+translated)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	app := filepath.Join(apps, "My App.app")
	installed := func(version string) {
		t.Helper()
		if out, err := exec.Command(filepath.Join(app, "Contents", "MacOS", "My App")).Output(); err != nil || string(out) != version+"\n" {
			t.Errorf("the app printed %q, %v, want %s", out, err, version)
		}
		entries, _ := os.ReadDir(filepath.Join(app, "Contents", "Resources"))
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "only-") && e.Name() != "only-"+version {
				t.Errorf("%s of another version stayed", e.Name())
			}
		}
		if entries, _ := os.ReadDir(apps); len(entries) != 1 {
			t.Errorf("%s holds %v", apps, entries)
		}
	}

	// Only a universal build is published: Apple silicon installs it.
	publish("1.0.0", "darwin-universal")
	os.WriteFile(script, []byte(installScript(c)), 0o755)
	if out, err := run(); err != nil || !strings.Contains(out, "Downloading My App 1.0.0") || !strings.HasSuffix(out, "Installed My App 1.0.0 in "+app+"\n") {
		t.Fatalf("install.sh: %v\n%s", err, out)
	}
	installed("1.0.0")

	// Again, the newer version replaces it, with the script of either: the
	// build of the Mac's architecture comes before the universal one, and
	// a shell under Rosetta on Apple silicon installs arm64's too.
	publish("1.1.0", "darwin-arm64")
	machine, translated = "x86_64", "1"
	if out, err := run(); err != nil || !strings.HasSuffix(out, "Installed My App 1.1.0 in "+app+"\n") {
		t.Fatalf("install.sh under Rosetta: %v\n%s", err, out)
	}
	installed("1.1.0")
	machine, translated = "arm64", "0"
	if out, err := run(); err != nil || !strings.HasSuffix(out, "Installed My App 1.1.0 in "+app+"\n") {
		t.Fatalf("install.sh: %v\n%s", err, out)
	}
	installed("1.1.0")
	// An Intel Mac has no build of its own but the universal one.
	machine = "x86_64"
	if out, err := run(); err != nil || !strings.Contains(out, "Downloading My App 1.0.0") {
		t.Fatalf("install.sh on an Intel Mac: %v\n%s", err, out)
	}
	installed("1.0.0")
	machine = "arm64"
	if out, err := run(); err != nil || !strings.HasSuffix(out, "Installed My App 1.1.0 in "+app+"\n") {
		t.Fatalf("install.sh: %v\n%s", err, out)
	}
	installed("1.1.0")

	// An archive other than the one signed is refused, and the installed
	// version stays.
	archive := filepath.Join(served, "my-app-1.1.0-darwin-arm64.tar.gz")
	good, _ := os.ReadFile(archive)
	bad := slices.Clone(good)
	bad[len(bad)/2] ^= 1
	os.WriteFile(archive, bad, 0o644)
	if out, err := run(); err == nil || !strings.Contains(out, "is not signed with the key of My App: not installing it") {
		t.Errorf("a tampered archive: %v\n%s", err, out)
	}
	os.WriteFile(archive, append(slices.Clone(good), 0), 0o644)
	if out, err := run(); err == nil || !strings.Contains(out, "does not have the size of the manifest") {
		t.Errorf("a longer archive: %v\n%s", err, out)
	}
	installed("1.1.0")
	// Signed with another key.
	_, other, _ := ed25519.GenerateKey(nil)
	t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(other))
	c.Updates.PublicKey = base64.StdEncoding.EncodeToString(other.Public().(ed25519.PublicKey))
	publish("1.2.0", "darwin-arm64")
	if out, err := run(); err == nil || !strings.Contains(out, "is not signed with the key of My App") {
		t.Errorf("another key: %v\n%s", err, out)
	}
	installed("1.1.0")

	// Another app of the same name is left alone.
	c.Updates.PublicKey = base64.StdEncoding.EncodeToString(pub)
	other2 := *c
	other2.Identifier = "com.example.other"
	os.WriteFile(script, []byte(installScript(&other2)), 0o755)
	if out, err := run("--uninstall"); err == nil || !strings.Contains(out, "My App is not installed in "+apps) {
		t.Errorf("uninstalling another app: %v\n%s", err, out)
	}
	installed("1.1.0")

	// Uninstalling removes the app, and keeps its settings.
	os.WriteFile(script, []byte(installScript(c)), 0o755)
	settings := filepath.Join(home, "Library", "Application Support", c.Identifier, "settings.json")
	os.MkdirAll(filepath.Dir(settings), 0o755)
	os.WriteFile(settings, []byte("{}"), 0o644)
	if out, err := run("--uninstall"); err != nil || !strings.Contains(out, "Uninstalled My App. Its settings and data are still in place.") {
		t.Errorf("install.sh --uninstall: %v\n%s", err, out)
	}
	if _, err := os.Stat(app); err == nil {
		t.Error("the app is still there")
	}
	if _, err := os.Stat(settings); err != nil {
		t.Error(err)
	}
	if out, err := run("--uninstall"); err == nil {
		t.Errorf("uninstalling twice succeeded:\n%s", out)
	}
}

// TestInstallScriptVerify checks install.sh's Ed25519 verification on macOS
// against Go's signatures.
func TestInstallScriptVerify(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("verifies on macOS")
	}
	dir := t.TempDir()
	for i := range 8 {
		pub, priv, _ := ed25519.GenerateKey(nil)
		c := &Config{Name: "App", Updates: &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: "https://example.com"}}
		script := strings.TrimSuffix(installScript(c), "main \"$@\"\n") + "work=\"$(mktemp -d)\"\nif verify \"$1\" \"$2\"; then echo ok; else echo bad; fi\nrm -rf \"$work\"\n"
		path := filepath.Join(dir, "verify.sh")
		os.WriteFile(path, []byte(script), 0o755)
		file := filepath.Join(dir, "file")
		data := []byte(fmt.Sprint("file ", i))
		os.WriteFile(file, data, 0o644)
		sum := sha256.Sum256(data)
		sig := update.Sign(priv, sum[:])
		raw, _ := base64.StdEncoding.DecodeString(sig)
		flipped := slices.Clone(raw)
		flipped[i*7%64] ^= 1 << (i % 8)
		for _, tc := range []struct {
			sig  string
			want string
		}{{sig, "ok"}, {base64.StdEncoding.EncodeToString(flipped), "bad"}, {"", "bad"}, {"not base64", "bad"}} {
			out, err := exec.Command("sh", path, file, tc.sig).CombinedOutput()
			if err != nil || string(out) != tc.want+"\n" {
				t.Errorf("verifying %q: %v %q, want %s", tc.sig, err, out, tc.want)
			}
		}
	}
}

// TestInstallScriptVerifier checks the JavaScript that install.sh runs on
// macOS to check Ed25519 signatures (verifier) with Node.js, which runs
// the same JavaScript as JavaScript for Automation: against the test
// vectors of RFC 8032 and Go's signatures, and that it rejects what
// crypto/ed25519 rejects.
func TestInstallScriptVerifier(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("needs Node.js")
	}
	script := installScript(&Config{Name: "App", Updates: &Updates{PublicKey: "AAAA", URL: "https://example.com"}})
	_, js, ok := strings.Cut(script, "verifier() {\n\tcat <<'EOF'\n")
	js, _, ok2 := strings.Cut(js, "\nEOF\n")
	if !ok || !ok2 {
		t.Fatal("install.sh has no verifier")
	}
	path := filepath.Join(t.TempDir(), "verify.js")
	if err := os.WriteFile(path, []byte(js+"\nconsole.log(run(process.argv.slice(2)))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// verify runs the verifier as install.sh does: with the key, R and S
	// of the signature, and k, the SHA-512 of R, the key and the message.
	verify := func(key, sig, msg []byte) string {
		t.Helper()
		h := sha512.New()
		h.Write(sig[:32])
		h.Write(key)
		h.Write(msg)
		out, err := exec.Command(node, path, hex.EncodeToString(key), hex.EncodeToString(sig[:32]), hex.EncodeToString(sig[32:]), hex.EncodeToString(h.Sum(nil))).CombinedOutput()
		if err != nil {
			t.Fatalf("node: %v\n%s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	unhex := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	// Section 7.1 of RFC 8032: TEST 1, 2, 3 and SHA(abc).
	for _, v := range []struct{ seed, key, msg, sig string }{
		{"9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60", "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a", "",
			"e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b"},
		{"4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb", "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c", "72",
			"92a009a9f0d4cab8720e820b5f642540a2b27b5416503f8fb3762223ebdb69da085ac1e43e15996e458f3613d0f11d8c387b2eaeb4302aeeb00d291612bb0c00"},
		{"c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7", "fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025", "af82",
			"6291d657deec24024827e69c3abe01a30ce548a284743a445e3680d7db5ac3ac18ff9b538d16f290ae67f760984dc6594a7c15e9716ed28dc027beceea1ec40a"},
		{"833fe62409237b9d62ec77587520911e9a759cec1d19755b7da901b96dca3d42", "ec172b93ad5e563bf4932c70e1245034c35467ef2efd4d64ebf819683467e2bf",
			"ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f",
			"dc2a4459e7369633a52b1bf277839a00201009a3efbf3ecb69bea2186c26b58909351fc9ac90b3ecfdfbc7c66431e0303dca179c138ac17ad9bef1177331a704"},
	} {
		key, msg, sig := unhex(v.key), unhex(v.msg), unhex(v.sig)
		priv := ed25519.NewKeyFromSeed(unhex(v.seed))
		if !slices.Equal(priv.Public().(ed25519.PublicKey), key) || !slices.Equal(ed25519.Sign(priv, msg), sig) {
			t.Fatalf("the vector of key %s does not match crypto/ed25519", v.key)
		}
		if got := verify(key, sig, msg); got != "ok" {
			t.Errorf("RFC 8032 key %s: %s, want ok", v.key, got)
		}
		if got := verify(key, sig, append(slices.Clone(msg), 0)); got != "bad" {
			t.Errorf("RFC 8032 key %s, another message: %s, want bad", v.key, got)
		}
	}

	// Go's signatures of digests, as of the archives of updates, and the
	// same signatures altered.
	l, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	for i := range 8 {
		pub, priv, _ := ed25519.GenerateKey(nil)
		_, other, _ := ed25519.GenerateKey(nil)
		digest := sha256.Sum256([]byte(fmt.Sprint("archive ", i)))
		sig := ed25519.Sign(priv, digest[:])
		if got := verify(pub, sig, digest[:]); got != "ok" {
			t.Errorf("signature %d: %s, want ok", i, got)
		}
		flipped := slices.Clone(sig)
		flipped[i*9%64] ^= 1 << (i % 8)
		// S + l, which is the same signature modulo l: RFC 8032 and
		// crypto/ed25519 reject it, as S must be less than l.
		s := new(big.Int).SetBytes(reverse(sig[32:]))
		malleable := append(slices.Clone(sig[:32]), reverse(new(big.Int).Add(s, l).FillBytes(make([]byte, 32)))...)
		for name, tc := range map[string]struct{ key, sig []byte }{
			"flipped bit":   {pub, flipped},
			"other key":     {pub, ed25519.Sign(other, digest[:])},
			"S + l":         {pub, malleable},
			"key not on it": {notOnCurve, sig},
		} {
			if ed25519.Verify(ed25519.PublicKey(tc.key), digest[:], tc.sig) {
				t.Fatalf("signature %d, %s: crypto/ed25519 accepts it", i, name)
			}
			if got := verify(tc.key, tc.sig, digest[:]); got != "bad" {
				t.Errorf("signature %d, %s: %s, want bad", i, name, got)
			}
		}
	}
}

// reverse returns b with its bytes in reverse order, between the
// little-endian numbers of Ed25519 and big.Int's big-endian ones.
func reverse(b []byte) []byte {
	r := slices.Clone(b)
	slices.Reverse(r)
	return r
}

// notOnCurve encodes no point of Ed25519: y = 2, for which
// (y² - 1) / (d y² + 1) has no square root.
var notOnCurve = append([]byte{2}, make([]byte, 31)...)

// verifyScript writes install.sh of c without its last line, running
// instead the shell code of run, and returns its path.
func verifyScript(t *testing.T, c *Config, run string) string {
	t.Helper()
	script := strings.TrimSuffix(installScript(c), "main \"$@\"\n") + "work=\"$(mktemp -d)\"\n" + run + "\nrm -rf \"$work\"\n"
	path := filepath.Join(t.TempDir(), "verify.sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// opensslEd25519 makes sure that the openssl of $PATH checks Ed25519
// signatures, as install.sh does on Linux, putting that of Homebrew first
// when the system's has none (LibreSSL on macOS), else skips the test.
func opensslEd25519(t *testing.T) {
	t.Helper()
	check := verifyScript(t, &Config{Name: "App", Updates: &Updates{URL: "https://example.com"}}, "has_ed25519")
	for _, dir := range []string{"", "/opt/homebrew/opt/openssl@3/bin", "/usr/local/opt/openssl@3/bin"} {
		if dir != "" {
			if _, err := os.Stat(filepath.Join(dir, "openssl")); err != nil {
				continue
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
		if exec.Command("sh", check).Run() == nil {
			return
		}
	}
	t.Skip("no openssl that checks Ed25519 signatures (OpenSSL 3.0 or later)")
}

// TestInstallScriptVerifyOpenSSL checks install.sh's Ed25519 verification
// with openssl, as on Linux, against Go's signatures.
func TestInstallScriptVerifyOpenSSL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs install.sh")
	}
	opensslEd25519(t)
	dir := t.TempDir()
	_, other, _ := ed25519.GenerateKey(nil)
	for i := range 8 {
		pub, priv, _ := ed25519.GenerateKey(nil)
		c := &Config{Name: "App", Updates: &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: "https://example.com"}}
		path := verifyScript(t, c, `if verify_openssl "$1" "$2"; then echo ok; else echo bad; fi`)
		file := filepath.Join(dir, "file")
		data := []byte(fmt.Sprint("file ", i))
		os.WriteFile(file, data, 0o644)
		sum := sha256.Sum256(data)
		sig := update.Sign(priv, sum[:])
		raw, _ := base64.StdEncoding.DecodeString(sig)
		flipped := slices.Clone(raw)
		flipped[i*7%64] ^= 1 << (i % 8)
		for _, tc := range []struct {
			sig  string
			want string
		}{
			{sig, "ok"},
			{base64.StdEncoding.EncodeToString(flipped), "bad"},
			{update.Sign(other, sum[:]), "bad"},
			{update.Sign(priv, data), "bad"}, // of the data, not its digest
			{"", "bad"},
			{"not base64", "bad"},
			{sig[:len(sig)-4], "bad"},
		} {
			out, err := exec.Command("sh", path, file, tc.sig).CombinedOutput()
			if err != nil || string(out) != tc.want+"\n" {
				t.Errorf("verifying %q: %v %q, want %s", tc.sig, err, out, tc.want)
			}
		}
	}
	// A key that is not 32 bytes of base64.
	c := &Config{Name: "App", Updates: &Updates{PublicKey: "AAAA", URL: "https://example.com"}}
	if out, _ := exec.Command("sh", verifyScript(t, c, `if verify_openssl "$1" "$2"; then echo ok; else echo bad; fi`), filepath.Join(dir, "file"), "").CombinedOutput(); string(out) != "bad\n" {
		t.Errorf("verifying with a bad key: %q", out)
	}
}

// TestInstallScriptNoEd25519 checks that install.sh on Linux downloads
// nothing when its openssl cannot check Ed25519 signatures.
func TestInstallScriptNoEd25519(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs install.sh")
	}
	if os.Geteuid() == 0 {
		t.Skip("install.sh refuses to install for root")
	}
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "downloads")
	for name, script := range map[string]string{
		"uname": "#!/bin/sh\ncase \"$1\" in -s) echo Linux ;; -m) echo x86_64 ;; esac\n",
		// An openssl that cannot check Ed25519 signatures, as before 3.0.
		"openssl": "#!/bin/sh\necho 'pkeyutl: unsupported algorithm' >&2\nexit 1\n",
		"curl":    "#!/bin/sh\necho \"$@\" >>\"$DOWNLOADS\"\nexit 22\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DOWNLOADS", log)
	t.Setenv("HOME", t.TempDir())
	pub, _, _ := ed25519.GenerateKey(nil)
	c := &Config{root: t.TempDir(), Name: "My App", Version: "1.0.0",
		Updates: &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: "https://example.com/my-app"}}
	script, err := writeInstallScript(c, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("sh", script).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "checking the signature of My App needs OpenSSL 3.0 or later") {
		t.Errorf("without Ed25519, install.sh printed %v:\n%s", err, out)
	}
	if b, err := os.ReadFile(log); err == nil {
		t.Errorf("without Ed25519, install.sh downloaded:\n%s", b)
	}
}

// TestInstallScriptLinux runs install.sh in Linux containers, which
// download the latest version from a server of the test: it installs a
// signed archive, and refuses one with a flipped byte, another size or
// another key, and, with OpenSSL 1.1.1, whose pkeyutl has no -rawin, any.
// It needs Docker, whose containers reach the host at host.docker.internal.
// MYGO_TEST_LINUX_IMAGES, separated by spaces, are the images to run in
// instead of Debian 13 (OpenSSL 3.5) and 11 (OpenSSL 1.1.1), with curl;
// one whose name ends in ! has an openssl expected to check no Ed25519.
func TestInstallScriptLinux(t *testing.T) {
	if testing.Short() {
		t.Skip("runs Docker")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("runs Docker")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("Docker is not running")
	}
	images := strings.Fields(os.Getenv("MYGO_TEST_LINUX_IMAGES"))
	if len(images) == 0 {
		images = []string{"buildpack-deps:trixie-curl", "buildpack-deps:bullseye-curl!"}
	}

	pub, priv, _ := ed25519.GenerateKey(nil)
	_, other, _ := ed25519.GenerateKey(nil)
	served := t.TempDir()
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.FileServer(http.Dir(served)))
	srv.Listener.Close()
	srv.Listener = l
	srv.Start()
	defer srv.Close()
	url := "http://host.docker.internal:" + strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
	none := 0
	c := &Config{root: t.TempDir(), Name: "My App", Version: "1.1.0",
		Updates: &Updates{PublicKey: base64.StdEncoding.EncodeToString(pub), URL: url, Deltas: &none}}
	// publish serves the archives of a fake app of version, signed with key,
	// for both architectures, and install.sh with that key.
	publish := func(version string, key ed25519.PrivateKey) {
		t.Helper()
		t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
		c.Updates.PublicKey = base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
		c.Version = version
		for _, target := range []string{"linux-amd64", "linux-arm64"} {
			stage := filepath.Join(t.TempDir(), target)
			os.MkdirAll(stage, 0o755)
			if err := os.WriteFile(filepath.Join(stage, "my-app"), []byte("#!/bin/sh\necho "+version+"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := writeArchive(c, stage, target, []string{"my-app"}); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"my-app-" + version + "-" + target + ".tar.gz", "update-" + target + ".json"} {
				b, err := os.ReadFile(filepath.Join(stage, name))
				if err != nil {
					t.Fatal(err)
				}
				os.WriteFile(filepath.Join(served, name), b, 0o644)
			}
		}
		if _, err := writeInstallScript(c, served); err != nil {
			t.Fatal(err)
		}
	}
	// tamper changes the served archives of version.
	tamper := func(version string, change func([]byte) []byte) {
		for _, target := range []string{"linux-amd64", "linux-arm64"} {
			path := filepath.Join(served, "my-app-"+version+"-"+target+".tar.gz")
			b, _ := os.ReadFile(path)
			os.WriteFile(path, change(b), 0o644)
		}
	}
	// The script downloads install.sh, installs, and prints what the
	// installed app prints, as a user other than root.
	const steps = `set -e
curl -fsSL "$MY_APP_DOWNLOAD_URL/install.sh" >/tmp/install.sh
status=0
sh /tmp/install.sh || status=$?
echo "installed: $("$HOME/.local/my-app.app/my-app" 2>/dev/null || echo none)"
exit $status
`
	for _, image := range images {
		image, unsupported := strings.CutSuffix(image, "!")
		t.Run(image, func(t *testing.T) {
			if err := exec.Command("docker", "image", "inspect", image).Run(); err != nil {
				if out, err := exec.Command("docker", "pull", image).CombinedOutput(); err != nil {
					t.Skipf("docker pull %s: %v\n%s", image, err, out)
				}
			}
			home := "/tmp/home"
			container := "mygo-install-test-" + strconv.FormatInt(time.Now().UnixNano(), 36)
			started, err := exec.Command("docker", "run", "-d", "--rm", "--name", container, "--user", "1000:1000",
				"--add-host", "host.docker.internal:host-gateway", "-e", "HOME="+home, "-e", "MY_APP_DOWNLOAD_URL="+url,
				image, "sleep", "600").CombinedOutput()
			if err != nil {
				t.Fatalf("docker run: %v\n%s", err, started)
			}
			defer exec.Command("docker", "rm", "-f", container).Run()
			run := func() (string, error) {
				out, err := exec.Command("docker", "exec", container, "sh", "-c", steps).CombinedOutput()
				return string(out), err
			}

			publish("1.0.0", priv)
			out, err := run()
			if unsupported {
				if err == nil || !strings.Contains(out, "checking the signature of My App needs OpenSSL 3.0 or later") || !strings.Contains(out, "installed: none") {
					t.Errorf("without Ed25519: %v\n%s", err, out)
				}
				return
			}
			if err != nil || !strings.Contains(out, "Downloading My App 1.0.0") || !strings.Contains(out, "installed: 1.0.0\n") {
				t.Fatalf("install.sh: %v\n%s", err, out)
			}
			publish("1.1.0", priv)
			if out, err := run(); err != nil || !strings.Contains(out, "installed: 1.1.0\n") {
				t.Fatalf("install.sh: %v\n%s", err, out)
			}

			// What is not the signed archive is refused, and the installed
			// version stays.
			for _, tc := range []struct {
				name   string
				change func([]byte) []byte
				key    ed25519.PrivateKey
				want   string
			}{
				{"a flipped byte", func(b []byte) []byte { b[len(b)/2] ^= 1; return b }, priv, "is not signed with the key of My App: not installing it"},
				{"another size", func(b []byte) []byte { return append(b, 0) }, priv, "does not have the size of the manifest: not installing it"},
				{"another key", nil, other, "is not signed with the key of My App: not installing it"},
			} {
				publish("1.2.0", tc.key)
				if tc.change != nil {
					tamper("1.2.0", tc.change)
				}
				// install.sh with the key of the app.
				c.Updates.PublicKey = base64.StdEncoding.EncodeToString(pub)
				if _, err := writeInstallScript(c, served); err != nil {
					t.Fatal(err)
				}
				if out, err := run(); err == nil || !strings.Contains(out, tc.want) || !strings.Contains(out, "installed: 1.1.0\n") {
					t.Errorf("%s: %v\n%s", tc.name, err, out)
				}
			}
		})
	}
}
