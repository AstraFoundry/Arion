package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/i18ngen"
)

func TestI18nConfig(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{jsonConfig: `{"i18n": {}}`})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.I18n.Dir != "locales" || c.I18n.Source != "en" {
		t.Errorf("defaults: %+v", c.I18n)
	}
	for config, want := range map[string]string{
		`{"i18n": {"source": "EN"}}`:    `i18n.source "EN" is not a BCP 47 tag`,
		`{"i18n": {"source": "en-XA"}}`: `i18n.source "en-XA" is not a BCP 47 tag`,
		`{"i18n": {"source": "zh_CN"}}`: `i18n.source "zh_CN" is not a BCP 47 tag`,
		`{"i18n": {"dir": "."}}`:        `i18n.dir "." is not a directory of the project`,
		`{"i18n": {"dir": "../x"}}`:     `i18n.dir "../x" is not a directory of the project`,
	} {
		writeFiles(t, dir, map[string]string{jsonConfig: config})
		if _, err := loadConfig(dir); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want an error saying %q", config, err, want)
		}
	}
	writeFiles(t, dir, map[string]string{jsonConfig: `{}`})
	if c, err := loadConfig(dir); err != nil || c.I18n != nil || i18nModule(c) != "" {
		t.Errorf("without i18n: %+v, %v", c, err)
	}
}

func TestI18nModule(t *testing.T) {
	tests := []struct {
		name   string
		config string
		files  map[string]string
		want   string // relative to the project, "" for no module
	}{
		{"without a frontend", `{"i18n": {}}`, nil, ""},
		{"a workspace root", `{"i18n": {}}`, map[string]string{"package.json": `{"private": true, "workspaces": ["website"]}`}, ""},
		{"a Yarn workspace root", `{"i18n": {}}`, map[string]string{"package.json": `{"workspaces": {"packages": ["site"]}}`}, ""},
		{"the app's own package.json", `{"i18n": {}}`, map[string]string{"package.json": `{"name": "app", "dependencies": {"mygo-runtime": "^1"}}`}, "src/locales.ts"},
		{"a package.json of native UI", `{"i18n": {}}`, map[string]string{"package.json": `{"name": "app", "devDependencies": {"mygo-cli": "^1"}}`}, ""},
		{"a frontend directory", `{"i18n": {}}`, map[string]string{"frontend/package.json": `{"dependencies": {"mygo-runtime": "^1"}}`}, "frontend/src/locales.ts"},
		{"the frontend options", `{"i18n": {}, "frontendDist": "dist"}`, nil, "locales.ts"},
		{"an explicit bindings path", `{"bindings": "web/api.ts", "i18n": {}}`, nil, "web/locales.ts"},
		{"explicit bindings at a workspace root", `{"bindings": "web/api.ts", "i18n": {}}`, map[string]string{"package.json": `{"workspaces": ["website"]}`}, "web/locales.ts"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{jsonConfig: tc.config, "locales/en.json": `{}`})
			writeFiles(t, dir, tc.files)
			c, err := loadConfig(dir)
			if err != nil {
				t.Fatal(err)
			}
			want := ""
			if tc.want != "" {
				want = filepath.Join(dir, filepath.FromSlash(tc.want))
			}
			if got := i18nModule(c); got != want {
				t.Errorf("i18nModule = %q, want %q", got, want)
			}
		})
	}
}

func TestWriteI18n(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		jsonConfig:            `{"i18n": {"dir": "app/strings", "source": "en"}, "bindings": "web/src/mygo.ts"}`,
		"app/strings/en.json": `{"hello": "Hello, {name}!", "files": "{n, plural, one {# file} other {# files}}", "mygo.undo": "Revert"}`,
		"app/strings/de.json": `{"hello": "Hallo, {name}!", "mygo.undo": "Rückgängig"}`,
		"app/strings/doc.go":  "// Package strs holds the app's messages.\npackage strs\n",
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := i18nModule(c), filepath.Join(dir, "web", "src", "strings.ts"); got != want {
		t.Errorf("i18nModule = %s, want %s", got, want)
	}
	out, err := writeI18n(c)
	if err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join("app", "strings", i18ngen.GoFile)
	// Written paths are as the CLI shows them, with slashes.
	if want := []string{filepath.ToSlash(goFile), "web/src/strings.ts"}; !slices.Equal(out.written, want) {
		t.Errorf("wrote %v, want %v", out.written, want)
	}
	if len(out.warnings) != 1 || !strings.Contains(out.warnings[0].String(), "1 of 2 messages missing") {
		t.Errorf("warnings: %v", out.warnings)
	}
	src, _ := os.ReadFile(filepath.Join(dir, goFile))
	for _, want := range []string{"package strs", "func Hello(name string) string", "func Files(n int) string", "i18n.Register(Catalog)"} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("the accessors lack %q", want)
		}
	}
	ts, _ := os.ReadFile(filepath.Join(dir, "web", "src", "strings.ts"))
	if !bytes.Contains(ts, []byte(`hello(name: string): string;`)) || bytes.Contains(ts, []byte("mygo.undo")) {
		t.Errorf("the TypeScript module:\n%s", ts)
	}
	// Unchanged catalogs write nothing.
	if out, err := writeI18n(c); err != nil || len(out.written) != 0 {
		t.Errorf("again: wrote %v, %v", out.written, err)
	}

	// Errors fail it, and keep what it wrote.
	writeFiles(t, dir, map[string]string{"app/strings/de.json": `{"hello": "Hallo, {nom}!", "mygo.nope": "x"}`})
	_, err = writeI18n(c)
	if err == nil || !strings.Contains(err.Error(), `app/strings/de.json:1: "hello" uses the argument nom`) || !strings.Contains(err.Error(), `"mygo.nope": mygo has no message nope`) {
		t.Errorf("errors: %v", err)
	}
	if err := runI18n([]string{"check", dir}); err == nil || err.Error() != "2 errors in the catalogs" {
		t.Errorf("mygo i18n check: %v", err)
	}
	writeFiles(t, dir, map[string]string{"app/strings/de.json": `{"hello": "Hallo, {name}!", "files": "{n, plural, one {# Datei} other {# Dateien}}"}`})
	if err := runI18n([]string{"check", dir}); err != nil {
		t.Errorf("mygo i18n check: %v", err)
	}
	if err := runI18n([]string{"lint", dir}); err == nil {
		t.Error("mygo i18n lint")
	}

	// A missing directory.
	writeFiles(t, dir, map[string]string{jsonConfig: `{"i18n": {"dir": "nowhere"}}`})
	c, _ = loadConfig(dir)
	if _, err := writeI18n(c); err == nil || !strings.Contains(err.Error(), "i18n.dir nowhere is not a directory") {
		t.Errorf("a missing directory: %v", err)
	}
}

// TestI18nApp generates the accessors of an app's catalogs and builds the
// app, which the package of the catalogs registers with.
func TestI18nApp(t *testing.T) {
	if testing.Short() {
		t.Skip("building the app")
	}
	dir := t.TempDir()
	checkout, _ := filepath.Abs(filepath.Join("..", ".."))
	sum, _ := os.ReadFile(filepath.Join(checkout, "go.sum"))
	writeFiles(t, dir, map[string]string{
		"go.mod":          "module demo-app\n\ngo " + goVersion() + "\n\nrequire github.com/egoist/mygo v0.0.0\n\nreplace github.com/egoist/mygo => " + checkout + "\n",
		"go.sum":          string(sum),
		jsonConfig:        `{"i18n": {}}`,
		"locales/en.json": `{"files": "{n, plural, one {# file} other {# files}}"}`,
		"locales/de.json": `{"files": "{n, plural, one {# Datei} other {# Dateien}}"}`,
		"main.go": `package main

import (
	"fmt"

	"demo-app/locales"
	"github.com/egoist/mygo/i18n"
)

func main() {
	fmt.Println(locales.Files(1), locales.In("de").Files(2), i18n.Languages())
}
`,
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writeI18n(c); err != nil {
		t.Fatal(err)
	}
	env := []string{"GOFLAGS=-mod=mod", "GOPROXY=off", "GOWORK=off"}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOWORK", "off")
	cmd := goCommand(dir, env, "run", ".")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run: %v\n%s", err, out.String())
	}
	if got := out.String(); got != "1 file 2 Dateien [en de]\n" {
		t.Errorf("the app says %q", got)
	}
	if imported, path := catalogImported(c, compiledDirs(c)); !imported || path != "demo-app/locales" {
		t.Errorf("catalogImported = %v, %q", imported, path)
	}
	writeFiles(t, dir, map[string]string{"main.go": "package main\n\nfunc main() {}\n"})
	if imported, _ := catalogImported(c, compiledDirs(c)); imported {
		t.Error("an app that does not import its catalogs imports them")
	}
}

func TestDevLocale(t *testing.T) {
	for in, want := range map[string]string{"": "", "de": "de", "zh_TW": "zh-TW", "en-xa": "en-XA", "ar-XB": "ar-XB"} {
		if got, err := devLocale(in); err != nil || got != want {
			t.Errorf("devLocale(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
	if _, err := devLocale("C"); err == nil {
		t.Error("devLocale(C)")
	}
	page := webPage(&Config{Name: "Demo"}, "app.wasm", "", true, "ar-XB")
	if !strings.Contains(page, `go.env = { MYGO_ENV: "development", MYGO_LOCALE: "ar-XB" }`) {
		t.Errorf("the dev page does not pass the locale:\n%s", page)
	}
}

func TestWatchCatalogs(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		jsonConfig:                  `{"i18n": {}}`,
		"locales/en.json":           `{}`,
		"locales/" + i18ngen.GoFile: "package locales\n",
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	in := &buildInputs{}
	if c.I18n != nil {
		d := c.path(c.I18n.Dir)
		in.fileDirs = []string{d}
		in.sourceDirs = []string{d}
		in.generated = []string{filepath.Join(d, i18ngen.GoFile)}
	}
	s := snapshot(in)
	if _, ok := s[filepath.Join(dir, "locales", "en.json")]; !ok {
		t.Errorf("the catalogs are not watched: %v", s)
	}
	if _, ok := s[filepath.Join(dir, "locales", i18ngen.GoFile)]; ok {
		t.Error("the accessors written from the catalogs are watched")
	}
}
