package main

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestWebPage(t *testing.T) {
	c := &Config{Name: `The "Demo" <App>`}
	page := webPage(c, "demo.0123abcd.wasm", "icon.png", false)
	for _, want := range []string{`<title>The &#34;Demo&#34; &lt;App&gt;</title>`, `fetch("demo.0123abcd.wasm")`, `src="wasm_exec.js"`, `rel="icon" href="icon.png"`, `class="spinner"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %s:\n%s", want, page)
		}
	}
	if strings.Contains(page, devEvents) || strings.Contains(page, "MYGO_ENV") {
		t.Error("a production page listens to mygo dev")
	}
	dev := webPage(c, "app.wasm", "", true)
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
		case <-time.After(5 * time.Second):
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
}
