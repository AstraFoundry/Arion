package l10n_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/egoist/mygo/internal/i18ngen"
	. "github.com/egoist/mygo/internal/l10n"

	// The translations, which apps import for the languages they ship.
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
)

// TestCatalogs checks the catalogs as `mygo generate` checks an app's: no
// errors, no warnings, every message in every language, and the code
// generated from them: the accessors, the list of the translations and
// their packages.
func TestCatalogs(t *testing.T) {
	c, diags, err := i18ngen.Load(FS(), i18ngen.Options{Dir: "internal/l10n", Source: "en", Namespace: Namespace, Own: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diags {
		t.Error(d)
	}
	src, err := i18ngen.Go(c, "l10n")
	if err != nil {
		t.Fatal(err)
	}
	if old, _ := os.ReadFile(i18ngen.GoFile); !bytes.Equal(old, src) {
		t.Errorf("%s is out of date: go generate ./internal/l10n", i18ngen.GoFile)
	}
	messages := filepath.Join(i18ngen.MessagesDir, "en.json")
	if old, _ := os.ReadFile(messages); !bytes.Equal(old, i18ngen.MessagesJSON(c.Files["en"])) {
		t.Errorf("%s is out of date: go generate ./internal/l10n", messages)
	}
	// The catalog links no formatting of dates (internal/i18nlink).
	for _, k := range c.Keys {
		for _, p := range k.Params {
			if p.Type == "DateTime" {
				t.Errorf("%s formats a date, %s, which MyGo's own strings do not link: pass a string", k.Name, p.Name)
			}
		}
	}

	all := fstest.MapFS{}
	en, _ := os.ReadFile("en.json")
	all["en.json"] = &fstest.MapFile{Data: en}
	paths, _ := filepath.Glob("../../locales/*/*.json")
	var tags []string
	for _, p := range paths {
		tag := strings.TrimSuffix(filepath.Base(p), ".json")
		data, _ := os.ReadFile(p)
		all[tag+".json"] = &fstest.MapFile{Data: data}
		tags = append(tags, tag)
		if _, err := os.Stat(filepath.Join(filepath.Dir(p), "locale_gen.go")); err != nil {
			t.Errorf("%s has no package: go generate ./internal/l10n", p)
		}
	}
	if !slices.Equal(tags, Languages) {
		t.Errorf("Languages = %v, the translations are %v: go generate ./internal/l10n", Languages, tags)
	}
	_, diags, err = i18ngen.Load(all, i18ngen.Options{Dir: "locales", Source: "en", Namespace: Namespace})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range diags {
		t.Error(d)
	}
	if got, want := Catalog.Languages(), append([]string{"en"}, tags...); !slices.Equal(got, want) {
		t.Errorf("with the translations imported, the catalog has %v, want %v", got, want)
	}
}

func TestMessages(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{Undo(), "Undo"}, // the source language before package mygo tells the app's
		{In("zh-Hans").Undo(), "撤销"},
		{In("zh-TW").Undo(), "復原"},
		{In("zh-HK").MenuOpenRecent(), "打開最近使用過的檔案"},
		{In("de").Undo(), "Undo"},
		{In("en").MenuQuitApp("Notes"), "Quit Notes"},
		{In("zh-Hans").MenuQuitApp("Notes"), "退出Notes"},
		{In("zh-Hant").MenuAbout("Notes"), "關於Notes"},
		{In("en").UnmodifiedLines(1), "1 unmodified line"},
		{In("en").UnmodifiedLines(4), "4 unmodified lines"},
		{In("zh-Hans").UnmodifiedLines(4), "4 行未修改"},
		{In("en").Matches(3, 12), "3 of 12"},
		{In("zh-Hant").MatchesMore(3, 1000), "第 3 項，共 1,000 項以上"},
		{In("en").Line(12345), "Line 12345"},
		{In("en").UntitledNumbered(2), "Untitled 2"},
		{In("zh-Hans").DocumentFailed("moved", "a.txt"), "无法移动文稿“a.txt”。"},
		{In("en").DocumentFailed("saved", "a.txt"), "The document “a.txt” could not be saved."},
		{In("en").ColorValue(0.4, 0.755), "40% saturation, 76% brightness"},
		{In("en").CurrentLine(3, 120, 5), "Current line 3 of 120, column 5"},
		{In("zh-Hans").LineRange(120), "第 1 到 120 行，可加 :列"},
		{In("en").RemoveToken("Ada"), "Remove Ada"},
		{In("en").HunkAction(In("en").Revert()), "Revert change"},
		{In("en-XA").Close(), "[Çļöšé one]"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

// TestTextsAllocateNothing checks what the controls rely on as they say
// their strings every frame: a string without arguments, said again in
// the same language, allocates nothing.
func TestTextsAllocateNothing(t *testing.T) {
	for _, locale := range []string{"", "en", "zh-Hans", "zh-TW", "en-XA"} {
		want := In(locale).Undo()
		if allocs := testing.AllocsPerRun(100, func() {
			if In(locale).Undo() != want {
				t.Fatal("the string changed")
			}
		}); allocs != 0 {
			t.Errorf("Undo in %q allocates %v times", locale, allocs)
		}
	}
}

func BenchmarkText(b *testing.B) {
	In("zh-Hans").Undo()
	b.ReportAllocs()
	for b.Loop() {
		_ = In("zh-Hans").Undo()
	}
}
