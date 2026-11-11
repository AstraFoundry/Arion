package i18ngen

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// golden compares got with the file, or writes it with -update.
func golden(t *testing.T, file string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(file, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from what was generated (go test -update rewrites it):\n%s", file, diffLines(string(want), string(got)))
	}
}

// diffLines shows the first lines that differ.
func diffLines(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < max(len(w), len(g)); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return "line " + strconv.Itoa(i+1) + ":\n- " + a + "\n+ " + b
		}
	}
	return ""
}

// library is the catalog of a library with the namespace mygo.
func library(t *testing.T) *Catalog {
	t.Helper()
	lib, diags, err := Load(fstest.MapFS{
		"en.json": {Data: []byte(`{"cancel": "Cancel", "quitApp": "Quit {app}"}`)},
	}, Options{Dir: "l10n", Source: "en", Namespace: "mygo"})
	if err != nil || len(diags) > 0 {
		t.Fatalf("library: %v %v", err, diags)
	}
	return lib
}

func loadDir(t *testing.T, dir string, opts Options) (*Catalog, Diagnostics) {
	t.Helper()
	c, diags, err := Load(os.DirFS(dir), opts)
	if err != nil {
		t.Fatal(err)
	}
	return c, diags
}

func TestApp(t *testing.T) {
	c, diags := loadDir(t, "testdata/app", Options{Dir: "locales", Source: "en", Libraries: map[string]*Catalog{"mygo": library(t)}, Translations: []string{"ja", "zh-Hans", "zh-Hant"}})
	var got []string
	for _, d := range diags {
		got = append(got, d.String())
	}
	want := []string{
		`locales/de.json: warning: 6 of 12 messages missing, which say the language they fall back to: progress, positional, keyword, untyped, object, save_as`,
		`locales/en-GB.json: warning: 11 of 12 messages missing, which say the language they fall back to: title, filesSelected, greeting, replied, place, guests, positional, keyword and 3 more`,
		`locales/zh-Hant.json: warning: 9 of 12 messages missing, which say the language they fall back to: replied, progress, place, guests, positional, keyword, untyped, object and 1 more`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("diagnostics:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if diags.Errors() != 0 || diags.Err() != nil || len(diags.Warnings()) != len(want) {
		t.Errorf("errors: %d %v", diags.Errors(), diags.Err())
	}
	if got, want := c.Languages, []string{"en", "de", "en-GB", "zh-Hant"}; !slices.Equal(got, want) {
		t.Errorf("Languages = %v, want %v", got, want)
	}
	var params []string
	for _, k := range c.Keys {
		var ps []string
		for _, p := range k.Params {
			ps = append(ps, p.Name+":"+p.Type+":"+p.Go+":"+p.TS)
		}
		params = append(params, k.GoName+"("+strings.Join(ps, ", ")+")")
	}
	wantParams := []string{
		"Title()",
		"FilesSelected(count:int:count:count)",
		"Greeting(name:String:name:name)",
		"Replied(gender:String:gender:gender, when:DateTime:when:when)",
		"Progress(done:double:done:done)",
		"Place(n:int:n:n)",
		"Guests(host:String:host:host, n:num:n:n)",
		"Positional(0:String:arg0:arg0, 1:String:arg1:arg1)",
		"Keyword(type:String:type_:type, default:String:default_:default_, string:String:string_:string, any:String:any_:any)",
		"Untyped(value:String:value:value)",
		"Object(item:Object:item:item)",
		"SaveAs()",
	}
	if !slices.Equal(params, wantParams) {
		t.Errorf("keys:\n%s\nwant:\n%s", strings.Join(params, "\n"), strings.Join(wantParams, "\n"))
	}
	// The app ships zh-Hant: its accessors link MyGo's zh-Hant.
	if !slices.Equal(c.Translations, []string{"zh-Hant"}) {
		t.Errorf("Translations = %v, want [zh-Hant]", c.Translations)
	}
	if got := Fallbacks(c, "en-GB"); !slices.Equal(got, []string{"en-GB", "en"}) {
		t.Errorf("Fallbacks(en-GB) = %v", got)
	}

	src, err := Go(c, "locales")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "testdata/app/"+GoFile, src)
	ts, err := TS(c)
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "testdata/app/locales.ts", ts)
}

// TestGoldenBuilds compiles the Go accessors of the golden catalog, with a
// program that says its messages.
func TestGoldenBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a program")
	}
	out, err := exec.Command("go", "vet", "./testdata/app").CombinedOutput()
	if err != nil {
		t.Fatalf("go vet: %v\n%s", err, out)
	}
	out, err = exec.Command("go", "run", "./testdata/app/say").CombinedOutput()
	if err != nil {
		t.Fatalf("go run: %v\n%s", err, out)
	}
	want := `Notes
Notizen
3 files selected
Keine Dateien ausgewählt
已選取 1 個檔案
Ada and 2 others
Dismiss
Abbrechen
[en de en-GB zh-Hant]
`
	if string(out) != want {
		t.Errorf("the program says:\n%s\nwant:\n%s", out, want)
	}
}

func TestBad(t *testing.T) {
	_, diags := loadDir(t, "testdata/bad", Options{Dir: "locales", Source: "en", Libraries: map[string]*Catalog{"mygo": library(t)}})
	var b strings.Builder
	for _, d := range diags {
		b.WriteString(d.String() + "\n")
	}
	golden(t, "testdata/bad.txt", []byte(b.String()))
	if diags.Errors() == 0 || diags.Err() == nil {
		t.Error("no errors")
	}
}

func TestLoadSource(t *testing.T) {
	for _, c := range []struct {
		files  fstest.MapFS
		source string
		want   string
	}{
		{fstest.MapFS{"de.json": {Data: []byte(`{}`)}}, "en", "locales/en.json: missing: it holds the messages of the source language, en"},
		{fstest.MapFS{"en.json": {Data: []byte(`{}`)}}, "EN", `locales: the source language "EN" is not a BCP 47 tag in its canonical form (as en, de-CH or zh-Hans)`},
		{fstest.MapFS{"en.json": {Data: []byte(`{}`)}}, "en-XA", `locales: the source language "en-XA" is not a BCP 47 tag in its canonical form (as en, de-CH or zh-Hans)`},
	} {
		_, diags, err := Load(c.files, Options{Dir: "locales", Source: c.source})
		if err != nil {
			t.Fatal(err)
		}
		if len(diags) != 1 || diags[0].String() != c.want {
			t.Errorf("source %q: %v, want %q", c.source, diags, c.want)
		}
	}
	// The source defaults to English.
	c, diags, err := Load(fstest.MapFS{"en.json": {Data: []byte(`{"a": "A"}`)}}, Options{})
	if err != nil || len(diags) > 0 || c.Source != "en" {
		t.Errorf("default source: %v %v %v", c, diags, err)
	}
}

func TestLibraryKeys(t *testing.T) {
	_, diags, err := Load(fstest.MapFS{"en.json": {Data: []byte(`{"a": "A", "other.b": "B"}`)}}, Options{Dir: "l10n", Namespace: "mygo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Msg, `"other.b" is not a key`) {
		t.Errorf("diagnostics: %v", diags)
	}
	src, err := Go(library(t), "l10n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`i18n.NewLibraryCatalog(files, "en", "mygo")`, "func QuitApp(app string) string", `"mygo.<key>"`} {
		if !bytes.Contains(src, []byte(want)) {
			t.Errorf("the library's accessors lack %q:\n%s", want, src)
		}
	}
	if bytes.Contains(src, []byte("Register")) {
		t.Error("a library's catalog is registered")
	}
}

func TestPackageName(t *testing.T) {
	dir := t.TempDir()
	for name, want := range map[string]string{"locales": "locales", "My-Strings": "mystrings", "2l10n": "l10n", "---": "locales"} {
		d := filepath.Join(dir, name)
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if got, err := PackageName(d); err != nil || got != want {
			t.Errorf("PackageName(%s) = %q, %v, want %q", name, got, err, want)
		}
	}
	d := filepath.Join(dir, "locales")
	os.WriteFile(filepath.Join(d, GoFile), []byte("package ignored\n"), 0o644)
	os.WriteFile(filepath.Join(d, "x_test.go"), []byte("package ignored_test\n"), 0o644)
	os.WriteFile(filepath.Join(d, "doc.go"), []byte("// Package strings.\npackage strs\n"), 0o644)
	if got, err := PackageName(d); err != nil || got != "strs" {
		t.Errorf("PackageName with doc.go = %q, %v", got, err)
	}
	os.WriteFile(filepath.Join(d, "doc.go"), []byte("package main\n"), 0o644)
	if _, err := PackageName(d); err == nil || !strings.Contains(err.Error(), "main package") {
		t.Errorf("PackageName of a main package: %v", err)
	}
}
