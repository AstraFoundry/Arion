package i18n

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

// resetRegistry forgets the catalogs a test registered.
func resetRegistry(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		registry.mu.Lock()
		registry.catalogs = nil
		registry.version.Add(1)
		registry.mu.Unlock()
		languages.mu.Lock()
		languages.list = nil
		languages.mu.Unlock()
	})
}

func files(m map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, data := range m {
		fsys[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return fsys
}

var appFiles = map[string]string{
	"en.json": `{
		"@@locale": "en",
		"hello": "Hello, {name}!",
		"@hello": {"description": "A greeting.", "placeholders": {"name": {"type": "String"}}},
		"color": "Color",
		"files": "{n, plural, one {# file} other {# files}}",
		"onlyEnglish": "Only in English",
		"broken": "Fine in English"
	}`,
	"en-001.json":  `{"color": "Colour"}`,
	"de.json":      `{"@@locale": "de", "hello": "Hallo, {name}!", "files": "{n, plural, one {# Datei} other {# Dateien}}", "broken": "{kaputt"}`,
	"zh-Hans.json": `{"hello": "你好，{name}！"}`,
	"zh-Hant.json": `{"hello": "你好，{name}！", "color": "顏色"}`,
	"pt.json":      `{"color": "Cor"}`,
	"pt-PT.json":   `{"hello": "Olá, {name}!"}`,
}

func TestNewCatalog(t *testing.T) {
	c, err := NewCatalog(files(appFiles), "en")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Languages(), []string{"en", "de", "en-001", "pt", "pt-PT", "zh-Hans", "zh-Hant"}; !slices.Equal(got, want) {
		t.Errorf("Languages() = %v, want %v", got, want)
	}
	if c.Source() != "en" || c.Namespace() != "" {
		t.Errorf("Source() = %q, Namespace() = %q", c.Source(), c.Namespace())
	}
	for _, bad := range []struct {
		files  map[string]string
		source string
		want   string
	}{
		{map[string]string{"en.json": "{}"}, "", "not a language"},
		{map[string]string{"en.json": "{}"}, "en-XA", "not a language"},
		{map[string]string{"de.json": "{}"}, "en", "no en.json"},
		{map[string]string{"en.json": "{}", "zh_CN.json": "{}"}, "en", "zh_CN.json is not named"},
		{map[string]string{"en.json": "{}", "EN-gb.json": "{}"}, "en", "EN-gb.json is not named"},
		{map[string]string{"en.json": "{}", "messages.json": "{}"}, "en", "messages.json is not named"},
		{map[string]string{"en.json": "{}", "en-XA.json": "{}"}, "en", "pseudo"},
	} {
		_, err := NewCatalog(files(bad.files), bad.source)
		if err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("NewCatalog(%v, %q) = %v, want an error saying %q", slices.Collect(func(yield func(string) bool) {
				for k := range bad.files {
					yield(k)
				}
			}), bad.source, err, bad.want)
		}
	}
	if _, err := NewLibraryCatalog(files(appFiles), "en", "my.go"); err == nil {
		t.Error("a namespace with a dot")
	}
}

func TestCatalogFormat(t *testing.T) {
	c, err := NewCatalog(files(appFiles), "en")
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		locale, key string
		args        map[string]any
		want        string
	}{
		{"en", "hello", map[string]any{"name": "Ada"}, "Hello, Ada!"},
		{"", "hello", map[string]any{"name": "Ada"}, "Hello, Ada!"},
		{"de", "hello", map[string]any{"name": "Ada"}, "Hallo, Ada!"},
		{"de-CH", "hello", map[string]any{"name": "Ada"}, "Hallo, Ada!"},
		{"de", "color", nil, "Color"},            // falls back to the source
		{"de", "broken", nil, "Fine in English"}, // de's does not parse
		{"en-AU", "color", nil, "Colour"},        // en-AU -> en-001
		{"en-GB", "hello", map[string]any{"name": "Ada"}, "Hello, Ada!"},
		{"en-US", "color", nil, "Color"}, // en-US -> en
		{"zh-TW", "color", nil, "顏色"},    // matched to zh-Hant
		{"zh-Hant-HK", "color", nil, "顏色"},
		{"zh-CN", "color", nil, "Color"}, // zh-Hans has none; never zh-Hant's
		{"zh", "hello", map[string]any{"name": "Ada"}, "你好，Ada！"},
		{"pt-AO", "hello", map[string]any{"name": "Ada"}, "Olá, Ada!"}, // pt-AO -> pt-PT
		{"pt-AO", "color", nil, "Cor"},                                 // pt-PT -> pt
		{"pt-BR", "hello", map[string]any{"name": "Ada"}, "Hello, Ada!"},
		{"fr", "hello", map[string]any{"name": "Ada"}, "Hello, Ada!"},
		{"C", "color", nil, "Color"},
		{"de", "missing", nil, "missing"},
		{"de", "files", map[string]any{"n": 1}, "1 Datei"},
		// English's plural rules for an English message, whatever the
		// locale: French says one for 0.
		{"fr", "files", map[string]any{"n": 0}, "0 files"},
		{"en-XA", "color", nil, "[Çöļöŕ one]"},
		{"en-XA", "missing", nil, "missing"},
		{"ar-XB", "color", nil, rlm + rlo + "Color" + pdf + rlm},
	} {
		if got := c.Format(x.locale, x.key, x.args); got != x.want {
			t.Errorf("Format(%q, %q) = %q, want %q", x.locale, x.key, got, x.want)
		}
	}
	m, lang, ok := c.Message("en-AU", "color")
	if !ok || lang != "en-001" || m.String() != "Colour" {
		t.Errorf("Message(en-AU, color) = %v, %q, %v", m, lang, ok)
	}
	if _, _, ok := c.Message("en", "nope"); ok {
		t.Error("Message of a missing key")
	}
}

// countingFS counts the files it opens.
type countingFS struct {
	fs.FS
	mu     sync.Mutex
	opened map[string]int
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.mu.Lock()
	c.opened[name]++
	c.mu.Unlock()
	return c.FS.Open(name)
}

func TestCatalogLazyAndConcurrent(t *testing.T) {
	fsys := &countingFS{FS: files(appFiles), opened: map[string]int{}}
	c, err := NewCatalog(fsys, "en")
	if err != nil {
		t.Fatal(err)
	}
	delete(fsys.opened, ".") // listed, not read
	if len(fsys.opened) != 0 {
		t.Fatalf("NewCatalog read %v", fsys.opened)
	}
	var wg sync.WaitGroup
	var bad atomic.Int32
	for i := range 32 {
		wg.Go(func() {
			locale := []string{"de", "en", "zh-TW", "en-AU"}[i%4]
			want := map[string]string{"de": "Hallo, Ada!", "en": "Hello, Ada!", "zh-TW": "你好，Ada！", "en-AU": "Hello, Ada!"}[locale]
			for range 100 {
				if c.Format(locale, "hello", map[string]any{"name": "Ada"}) != want {
					bad.Add(1)
				}
			}
		})
	}
	wg.Wait()
	if bad.Load() != 0 {
		t.Errorf("%d wrong messages", bad.Load())
	}
	for name, n := range fsys.opened {
		if n != 1 {
			t.Errorf("read %s %d times", name, n)
		}
	}
	if fsys.opened["pt.json"] != 0 || fsys.opened["de.json"] != 1 {
		t.Errorf("read %v", fsys.opened)
	}
}

var libFiles = map[string]string{
	"en.json":      `{"cancel": "Cancel", "quitApp": "Quit {app}", "close": "Close"}`,
	"zh-Hans.json": `{"cancel": "取消", "quitApp": "退出{app}"}`,
	"zh-Hant.json": `{"cancel": "取消", "quitApp": "結束{app}"}`,
}

func TestLibraryCatalog(t *testing.T) {
	resetRegistry(t)
	lib, err := NewLibraryCatalog(files(libFiles), "en", "mygo")
	if err != nil {
		t.Fatal(err)
	}
	// Without an app's catalog: the library's own.
	if got := lib.Format("zh-TW", "cancel", nil); got != "取消" {
		t.Errorf("zh-TW cancel = %q", got)
	}
	if got := lib.Format("de", "cancel", nil); got != "Cancel" {
		t.Errorf("de cancel = %q", got)
	}

	app, err := NewCatalog(files(map[string]string{
		"de.json":      `{"title": "Titel", "mygo.cancel": "Abbrechen"}`,
		"en.json":      `{"title": "Title", "mygo.close": "Dismiss"}`,
		"zh-Hans.json": `{"title": "标题", "mygo.quitApp": "退出 {app}"}`,
		"zh-TW.json":   `{"title": "標題"}`,
	}), "de")
	if err != nil {
		t.Fatal(err)
	}
	Register(app)
	Register(app) // once
	if got, want := Languages(), []string{"de", "en", "zh-Hans", "zh-TW"}; !slices.Equal(got, want) {
		t.Errorf("Languages() = %v, want %v", got, want)
	}
	for _, x := range []struct {
		locale, key string
		want        string
	}{
		{"de", "cancel", "Abbrechen"},          // the app's translation
		{"de-AT", "cancel", "Abbrechen"},       // along the chain
		{"en", "close", "Dismiss"},             // the app's change
		{"en", "cancel", "Cancel"},             // the library's
		{"zh-Hans", "quitApp", "退出 Notes"},     // the app's change
		{"zh-Hans", "cancel", "取消"},            // the library's
		{"zh-TW", "quitApp", "結束Notes"},        // the library's zh-Hant
		{"ja", "cancel", "Abbrechen"},          // the app's source language
		{"ja", "close", "Dismiss"},             // the library's source, as the app changed it
		{"ja", "quitApp", "Quit Notes"},        // the library's source
		{"", "cancel", "Abbrechen"},            // "": the app's source language
		{"en-XA", "cancel", "[Åƀƀŕéçĥéñ one]"}, // the app's source language, transformed
		{"de", "missing", "missing"},
	} {
		if got := lib.Format(x.locale, x.key, map[string]any{"app": "Notes"}); got != x.want {
			t.Errorf("library Format(%q, %q) = %q, want %q", x.locale, x.key, got, x.want)
		}
	}
	// The app's own keys are not the library's.
	if got := lib.Format("de", "title", nil); got != "title" {
		t.Errorf("library Format(de, title) = %q", got)
	}
	if got := app.Format("zh-TW", "title", nil); got != "標題" {
		t.Errorf("app Format(zh-TW, title) = %q", got)
	}

	defer func() {
		if recover() == nil {
			t.Error("Register of a library's catalog did not panic")
		}
	}()
	Register(lib)
}

func TestLocaleFunc(t *testing.T) {
	t.Cleanup(func() { SetLocaleFunc(nil) })
	if Locale() != "" {
		t.Errorf("Locale() = %q before SetLocaleFunc", Locale())
	}
	SetLocaleFunc(func() string { return "de" })
	if Locale() != "de" {
		t.Errorf("Locale() = %q", Locale())
	}
	SetLocaleFunc(nil)
	if Locale() != "" {
		t.Errorf("Locale() = %q after SetLocaleFunc(nil)", Locale())
	}
}

func BenchmarkCatalogFormat(b *testing.B) {
	c, err := NewCatalog(files(appFiles), "en")
	if err != nil {
		b.Fatal(err)
	}
	args := map[string]any{"n": 3}
	for b.Loop() {
		_ = c.Format("de", "files", args)
	}
}

func ExampleCatalog_Format() {
	c, _ := NewCatalog(fstest.MapFS{
		"en.json": {Data: []byte(`{"files": "{n, plural, =0 {No files} one {# file} other {# files}}"}`)},
		"de.json": {Data: []byte(`{"files": "{n, plural, =0 {Keine Dateien} one {# Datei} other {# Dateien}}"}`)},
	}, "en")
	fmt.Println(c.Format("en", "files", map[string]any{"n": 1}))
	fmt.Println(c.Format("de-AT", "files", map[string]any{"n": 0}))
	fmt.Println(c.Format("en-XA", "files", map[string]any{"n": 3}))
	// Output:
	// 1 file
	// Keine Dateien
	// [3 ƒîļéš one]
}

func TestCatalogFormatKeepsTexts(t *testing.T) {
	c, err := NewCatalog(files(appFiles), "en")
	if err != nil {
		t.Fatal(err)
	}
	first := c.Format("de", "title", nil)
	if allocs := testing.AllocsPerRun(100, func() {
		if c.Format("de", "title", nil) != first {
			t.Fatal("the message changed")
		}
	}); allocs != 0 {
		t.Errorf("a kept message allocates %v times", allocs)
	}
}

func BenchmarkCatalogText(b *testing.B) {
	c, err := NewCatalog(files(appFiles), "en")
	if err != nil {
		b.Fatal(err)
	}
	c.Format("de", "title", nil)
	b.ReportAllocs()
	for b.Loop() {
		_ = c.Format("de", "title", nil)
	}
}

// TestCatalogAdd checks the translations a catalog takes from elsewhere,
// as MyGo's own strings take those of the packages an app imports.
func TestCatalogAdd(t *testing.T) {
	c, err := NewLibraryCatalog(files(map[string]string{"en.json": `{"undo": "Undo", "redo": "Redo"}`}), "en", "lib")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Format("de", "undo", nil); got != "Undo" {
		t.Fatalf("before German was added: %q", got)
	}
	if err := c.Add(files(map[string]string{"de.json": `{"undo": "Rückgängig"}`})); err != nil {
		t.Fatal(err)
	}
	if err := c.Add(files(map[string]string{"zh-Hant.json": `{"undo": "復原"}`})); err != nil {
		t.Fatal(err)
	}
	if got, want := c.Languages(), []string{"en", "de", "zh-Hant"}; !slices.Equal(got, want) {
		t.Errorf("Languages() = %v, want %v", got, want)
	}
	for _, tc := range [][3]string{{"de", "undo", "Rückgängig"}, {"de-AT", "undo", "Rückgängig"}, {"de", "redo", "Redo"}, {"zh-TW", "undo", "復原"}} {
		if got := c.Format(tc[0], tc[1], nil); got != tc[2] {
			t.Errorf("Format(%q, %q) = %q, want %q", tc[0], tc[1], got, tc[2])
		}
	}
	if err := c.Add(files(map[string]string{"de.json": `{}`})); err == nil {
		t.Error("adding German twice did not fail")
	}
	if err := c.Add(files(map[string]string{"german.json": `{}`})); err == nil {
		t.Error("a file not named after a language was added")
	}
}
