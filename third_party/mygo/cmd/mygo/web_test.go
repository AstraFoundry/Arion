package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/wait"
)

func TestWebPage(t *testing.T) {
	c := &Config{Name: `The "Demo" <App>`}
	page := webPage(c, "demo.0123abcd.wasm", "icon.png", false, "")
	for _, want := range []string{`<title>The &#34;Demo&#34; &lt;App&gt;</title>`, `fetch("demo.0123abcd.wasm")`, `src="wasm_exec.js"`, `rel="icon" href="icon.png"`, `class="spinner"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %s:\n%s", want, page)
		}
	}
	if strings.Contains(page, devEvents) || strings.Contains(page, "MYGO_ENV") {
		t.Error("a production page listens to mygo dev")
	}
	dev := webPage(c, "app.wasm", "", true, "")
	for _, want := range []string{devEvents, `MYGO_ENV: "development"`, "build-error", `fetch("app.wasm")`} {
		if !strings.Contains(dev, want) {
			t.Errorf("the development page lacks %s", want)
		}
	}
	if strings.Contains(dev, `rel="icon"`) {
		t.Error("a page without an icon links one")
	}
}

func TestIsWebPlatform(t *testing.T) {
	for p, want := range map[string]bool{"web": true, "js/wasm": true, "linux/amd64": false, "wasm": false} {
		if isWebPlatform(p) != want {
			t.Errorf("isWebPlatform(%q) = %v", p, !want)
		}
	}
}

// TestWebDev serves the development page and its build, and tells pages
// of new builds and failed ones.
func TestWebDev(t *testing.T) {
	d := &webDev{c: &Config{Name: "Demo"}, loader: []byte("// wasm_exec.js"), clients: map[chan [2]string]bool{}}
	d.wasm = []byte("\x00asm")
	srv := httptest.NewServer(d)
	defer srv.Close()
	get := func(path string) (string, string) {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return string(b), res.Header.Get("Content-Type")
	}
	if page, typ := get("/"); !strings.Contains(page, devEvents) || !strings.HasPrefix(typ, "text/html") {
		t.Errorf("/ is %s: %s", typ, page)
	}
	if wasm, typ := get("/app.wasm"); wasm != "\x00asm" || typ != "application/wasm" {
		t.Errorf("/app.wasm is %q, %s", wasm, typ)
	}
	if js, _ := get("/wasm_exec.js"); js != "// wasm_exec.js" {
		t.Errorf("/wasm_exec.js is %q", js)
	}

	res, err := http.Get(srv.URL + devEvents)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	events := bufio.NewReader(res.Body)
	next := func() string {
		done := make(chan string, 1)
		go func() {
			var b strings.Builder
			for {
				line, err := events.ReadString('\n')
				if err != nil || line == "\n" {
					break
				}
				b.WriteString(line)
			}
			done <- b.String()
		}()
		select {
		case e := <-done:
			return e
		case <-time.After(wait.Guard):
			t.Fatal("no event")
			return ""
		}
	}
	if e := next(); !strings.HasPrefix(e, "event: hello") {
		t.Fatalf("first event %q", e)
	}
	d.broadcast("build-error", `"main.go:1: oops"`)
	if e := next(); !strings.Contains(e, "event: build-error") || !strings.Contains(e, "oops") {
		t.Errorf("failure %q", e)
	}
	d.broadcast("reload", "{}")
	if e := next(); !strings.HasPrefix(e, "event: reload") {
		t.Errorf("build %q", e)
	}
}

func TestWebDevReloadsConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles programs")
	}
	t.Setenv("GOWORK", "off")
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"go.mod":         "module example.com/webdev\n\n" + goDirective(t) + "\n",
		"mygo.json":      `{"name":"Before","main":"./before","icon":"before.png"}`,
		"before/main.go": "package main\nfunc main() {}\n",
		"before.png":     "before icon",
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	icon, err := webIcon(c)
	if err != nil {
		t.Fatal(err)
	}
	d := &webDev{c: c, icon: icon}
	wasm := filepath.Join(t.TempDir(), "app.wasm")
	if !d.build(context.Background(), wasm, "") {
		t.Fatal(d.failure)
	}
	writeFiles(t, dir, map[string]string{
		"mygo.json":     `{"name":"After","main":"./after","icon":"after.svg"}`,
		"after/main.go": "package main\nfunc main() {}\n",
		"after.svg":     "<svg></svg>",
	})
	if err := os.RemoveAll(filepath.Join(dir, "before")); err != nil {
		t.Fatal(err)
	}
	if !d.build(context.Background(), wasm, "mygo.json changed") {
		t.Fatalf("rebuild did not use the new entry point: %s", d.failure)
	}
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	page := get("/").Body.String()
	if !strings.Contains(page, "<title>After</title>") || !strings.Contains(page, `href="icon.svg"`) {
		t.Errorf("page uses stale config: %s", page)
	}
	if icon := get("/icon.svg"); icon.Code != http.StatusOK || icon.Body.String() != "<svg></svg>" {
		t.Errorf("icon was not reloaded: %d %q", icon.Code, icon.Body.String())
	}
	if old := get("/icon.png"); old.Code != http.StatusNotFound {
		t.Errorf("old icon still served: %d", old.Code)
	}
	good := get("/app.wasm").Body.Bytes()
	if !bytes.HasPrefix(good, []byte("\x00asm")) {
		t.Fatal("rebuild did not serve WebAssembly")
	}
	writeFiles(t, dir, map[string]string{"mygo.json": `{`})
	if d.build(context.Background(), wasm, "mygo.json changed") || d.failure == "" {
		t.Fatal("invalid config did not fail the build")
	}
	if get("/").Body.String() != page || !bytes.Equal(get("/app.wasm").Body.Bytes(), good) {
		t.Error("a failed rebuild replaced the last successful build")
	}
}

// TestBuildWeb builds the native template for browsers: the page loads the
// WebAssembly it names, next to Go's loader.
func TestBuildWeb(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := t.TempDir()
	data := templateData{Name: "Web Demo", Slug: "web-demo", Module: "web-demo", Identifier: "com.example.webdemo"}
	if err := writeTemplate(dir, nativeTemplate, data); err != nil {
		t.Fatal(err)
	}
	checkout, _ := filepath.Abs(filepath.Join("..", ".."))
	sum, _ := os.ReadFile(filepath.Join(checkout, "go.sum"))
	gomod := "module web-demo\n\ngo " + goVersion() + "\n\nrequire github.com/egoist/mygo v0.0.0\n\nreplace github.com/egoist/mygo => " + checkout + "\n"
	if os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644) != nil || os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644) != nil {
		t.Fatal("cannot write the module")
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOWORK", "off")
	if err := runBuild([]string{"-platform", "web", dir}); err != nil {
		t.Fatal(err)
	}
	web := filepath.Join(dir, "build", "web")
	page, err := os.ReadFile(filepath.Join(web, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`fetch\("(web-demo\.[0-9a-f]{12}\.wasm)"\)`).FindSubmatch(page)
	if m == nil {
		t.Fatalf("the page names no WebAssembly:\n%s", page)
	}
	wasm, err := os.ReadFile(filepath.Join(web, string(m[1])))
	if err != nil || !bytes.HasPrefix(wasm, []byte("\x00asm")) {
		t.Fatalf("%s is no WebAssembly: %v", m[1], err)
	}
	if _, err := os.Stat(filepath.Join(web, "wasm_exec.js")); err != nil {
		t.Error(err)
	}

	t.Run("replace read-only files", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("directory permissions are not enforced by chmod on Windows")
		}
		// A previous build may contain files that cannot be deleted.
		// Replacing it must not remove the rest of the site first.
		locked := filepath.Join(web, "old-assets")
		if err := os.Mkdir(locked, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(locked, "old.txt"), []byte("old asset"), 0o644); err != nil {
			t.Fatal(err)
		}
		oldDir, err := os.Open(locked)
		if err != nil {
			t.Fatal(err)
		}
		// The old directory can move during the build. Its open handle
		// lets cleanup restore its permissions wherever it ends up.
		t.Cleanup(func() {
			if err := oldDir.Chmod(0o755); err != nil {
				t.Error(err)
			}
			oldDir.Close()
		})
		if err := oldDir.Chmod(0o555); err != nil {
			t.Fatal(err)
		}
		probe, err := os.Create(filepath.Join(locked, "probe"))
		if err == nil {
			probe.Close()
			t.Skip("directory permissions do not prevent writes for this user")
		}
		if !os.IsPermission(err) {
			t.Fatal(err)
		}
		if err := runBuild([]string{"-platform", "js/wasm", dir}); err != nil {
			t.Error(err)
		}
		for _, name := range []string{"index.html", string(m[1]), "wasm_exec.js"} {
			if _, err := os.Stat(filepath.Join(web, name)); err != nil {
				t.Error(err)
			}
		}
		if _, err := os.Stat(filepath.Join(web, "old-assets")); !os.IsNotExist(err) {
			t.Errorf("old assets remain in the new build: %v", err)
		}
	})
}
