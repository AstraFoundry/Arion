//go:build !js

package mygo

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/internal/bridge"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/ui"

	// MyGo's own strings in the languages the tests show.
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
)

// localeTest gives a test the app written in langs, the user reading
// preferred, a user data directory of its own, and puts back what it
// changed. It returns the directory.
func localeTest(t *testing.T, langs []string, preferred ...string) string {
	t.Helper()
	dir := t.TempDir()
	App.SetPath(PathUserData, dir)
	onMain(func() {
		fb.SetSystemLanguages(preferred...)
		App.SetLanguages(langs...)
	})
	onMain(func() {}) // what they posted ran
	t.Cleanup(func() {
		onMain(func() {
			App.SetLocale("")
			fb.KeepsLocale = false
			fb.SetSavedLocale("")
			fb.SetSystemLanguages()
			fb.SetSystemRegion(platform.Region{Locale: "en-US", FirstWeekday: 0, HourCycle: 12})
			App.SetLanguages()
		})
		onMain(func() {})
		App.SetPath(PathUserData, "")
	})
	return dir
}

// heardList records what listeners hear.
type heardList[T any] struct {
	mu   sync.Mutex
	list []T
}

func (r *heardList[T]) add(v T) { r.mu.Lock(); r.list = append(r.list, v); r.mu.Unlock() }
func (r *heardList[T]) get() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.list)
}

func TestResolveLocale(t *testing.T) {
	for _, c := range []struct {
		override         string
		preferred, langs []string
		want             string
	}{
		{"", []string{"zh-CN", "en-US"}, []string{"en", "zh-Hans"}, "zh-Hans"},
		{"", []string{"zh-TW"}, []string{"en", "zh-Hans"}, "en"},
		{"", []string{"ja", "de-AT"}, []string{"en", "de"}, "de"},
		{"", nil, []string{"fr", "en"}, "fr"},
		{"de", []string{"zh-CN"}, []string{"en", "de", "zh-Hans"}, "de"},
		// An override the app is not written in: the user's languages.
		{"ko", []string{"zh-CN"}, []string{"en", "zh-Hans"}, "zh-Hans"},
		{"ko", []string{"ja"}, []string{"en", "zh-Hans"}, "en"},
		// The pseudo locales, whatever the app is written in.
		{"en-XA", nil, []string{"en"}, "en-XA"},
		{"ar-XB", []string{"de"}, []string{"en", "de"}, "ar-XB"},
	} {
		if got := resolveLocale(c.override, c.preferred, c.langs); got != c.want {
			t.Errorf("resolveLocale(%q, %q, %q) = %q, want %q", c.override, c.preferred, c.langs, got, c.want)
		}
	}
}

func TestLocaleFollowsSystem(t *testing.T) {
	localeTest(t, []string{"en", "de", "zh-Hans"}, "en-US")
	var heard heardList[string]
	off := App.OnLocaleChanged(heard.add)
	defer off()
	if got := App.Locale(); got != "en" {
		t.Fatalf("Locale() = %q, want en", got)
	}
	fb.SetSystemLanguages("zh_CN", "en-US")
	onMain(func() {})
	if got := App.Locale(); got != "zh-Hans" {
		t.Errorf("Locale() = %q, want zh-Hans", got)
	}
	if got := App.PreferredLanguages(); !slices.Equal(got, []string{"zh-CN", "en-US"}) {
		t.Errorf("PreferredLanguages() = %q", got)
	}
	// The user's languages change, the app's does not: nobody hears.
	fb.SetSystemLanguages("zh-Hans-CN", "en-GB")
	onMain(func() {})
	if got := heard.get(); !slices.Equal(got, []string{"zh-Hans"}) {
		t.Errorf("listeners heard %q, want [zh-Hans]", got)
	}
	// The backend hears the app's language first, then the user's.
	onMain(func() {
		if got := fb.UILanguages; !slices.Equal(got, []string{"zh-Hans", "zh-Hans-CN", "en-GB"}) {
			t.Errorf("the backend's languages are %q", got)
		}
	})
}

func TestLocaleDefaultLanguages(t *testing.T) {
	localeTest(t, nil, "de-DE")
	if got := App.Languages(); !slices.Equal(got, []string{"en"}) {
		t.Errorf("Languages() = %q, want [en] without catalogs", got)
	}
	if got := App.Locale(); got != "en" {
		t.Errorf("an app written in English shows %q to a reader of German", got)
	}
}

func TestSetLocaleSavesInFile(t *testing.T) {
	dir := localeTest(t, []string{"en", "de", "fr"}, "fr-CA")
	var heard heardList[string]
	off := App.OnLocaleChanged(heard.add)
	defer off()
	if App.Locale() != "fr" {
		t.Fatalf("Locale() = %q, want fr", App.Locale())
	}
	App.SetLocale("de-AT")
	onMain(func() {})
	if got := App.Locale(); got != "de" {
		t.Errorf("after SetLocale(de-AT), Locale() = %q", got)
	}
	data, err := os.ReadFile(filepath.Join(dir, localeFile))
	if err != nil || strings.TrimSpace(string(data)) != "de-AT" {
		t.Errorf("the file has %q (%v)", data, err)
	}
	// Another tag of the language shown changes nothing anyone sees.
	App.SetLocale("de-CH")
	onMain(func() {})
	App.SetLocale("")
	onMain(func() {})
	if got := heard.get(); !slices.Equal(got, []string{"de", "fr"}) {
		t.Errorf("listeners heard %q, want [de fr]", got)
	}
	if _, err := os.Stat(filepath.Join(dir, localeFile)); !os.IsNotExist(err) {
		t.Errorf("following the system kept the file: %v", err)
	}
	if PathUserData == "" || App.Locale() != "fr" {
		t.Errorf("Locale() = %q after SetLocale(\"\")", App.Locale())
	}
}

func TestSetLocaleSavesWithSystem(t *testing.T) {
	dir := localeTest(t, []string{"en", "ja"}, "en")
	onMain(func() { fb.KeepsLocale = true })
	App.SetLocale("ja")
	onMain(func() {})
	if got, kept := onMainValue(func() string { tag, _ := backend().App().SavedLocale(); return tag }), true; got != "ja" || !kept {
		t.Errorf("the system keeps %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, localeFile)); !os.IsNotExist(err) {
		t.Errorf("a file was written where the system keeps the language: %v", err)
	}
}

func TestSetLocalePanicsOnBadTag(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "not a language tag") {
			t.Errorf("recovered %v", r)
		}
	}()
	App.SetLocale("not a tag")
}

// TestSetLocaleFromGoroutines checks that SetLocale is safe from any
// goroutine and that the calls the main thread gets to together show the
// last tag only.
func TestSetLocaleFromGoroutines(t *testing.T) {
	localeTest(t, []string{"en", "de", "fr", "ja"}, "en")
	var heard heardList[string]
	off := App.OnLocaleChanged(heard.add)
	defer off()
	onMain(func() {
		// The main thread is busy: the calls queue up.
		var wg sync.WaitGroup
		for _, tag := range []string{"de", "fr"} {
			wg.Add(1)
			go func() { defer wg.Done(); App.SetLocale(tag) }()
			wg.Wait()
		}
		wg.Add(1)
		go func() { defer wg.Done(); App.SetLocale("ja") }()
		wg.Wait()
	})
	onMain(func() {})
	if got := heard.get(); !slices.Equal(got, []string{"ja"}) {
		t.Errorf("listeners heard %q, want [ja]", got)
	}
}

// TestLocaleSavedAndEnvironment checks the language an app starts with:
// the one it saved, unless MYGO_LOCALE chooses one, which is never saved.
func TestLocaleSavedAndEnvironment(t *testing.T) {
	dir := localeTest(t, []string{"en", "de", "ja"}, "en")
	restart := func() {
		locales.mu.Lock()
		locales.loaded, locales.override, locales.fromEnv = false, "", false
		locales.mu.Unlock()
		onMain(func() { App.updateLocale(true) })
	}
	t.Cleanup(func() {
		locales.mu.Lock()
		locales.fromEnv = false
		locales.mu.Unlock()
	})
	if err := os.WriteFile(filepath.Join(dir, localeFile), []byte("de\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restart()
	if got := App.Locale(); got != "de" {
		t.Errorf("the app started with %q, saved de", got)
	}
	t.Setenv("MYGO_LOCALE", "ja")
	restart()
	if got := App.Locale(); got != "ja" {
		t.Errorf("the app started with %q, MYGO_LOCALE ja", got)
	}
	// While the app runs, SetLocale shows another language and saves
	// nothing.
	App.SetLocale("en")
	onMain(func() {})
	if got := App.Locale(); got != "en" {
		t.Errorf("SetLocale(en) under MYGO_LOCALE shows %q", got)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, localeFile)); strings.TrimSpace(string(data)) != "de" {
		t.Errorf("the file has %q, want de", data)
	}
}

func TestRegion(t *testing.T) {
	localeTest(t, []string{"en"}, "en-US")
	var heard heardList[Region]
	off := App.OnRegionChanged(heard.add)
	defer off()
	for _, c := range []struct {
		system platform.Region
		want   Region
	}{
		{platform.Region{Locale: "en_GB", FirstWeekday: 1, HourCycle: 24}, Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24}},
		// What the system does not say, the region's.
		{platform.Region{Locale: "en_US.UTF-8", FirstWeekday: -1}, Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12}},
		{platform.Region{Locale: "de-DE", FirstWeekday: -1}, Region{Locale: "de-DE", FirstWeekday: time.Monday, HourCycle: 24}},
		{platform.Region{Locale: "ar_EG", FirstWeekday: -1}, Region{Locale: "ar-EG", FirstWeekday: time.Saturday, HourCycle: 12}},
		{platform.Region{Locale: "zh-TW", FirstWeekday: -1}, Region{Locale: "zh-TW", FirstWeekday: time.Sunday, HourCycle: 12}},
		// macOS's formats of another region.
		{platform.Region{Locale: "en_US@rg=gbzzzz", FirstWeekday: 1, HourCycle: 24}, Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24}},
		{platform.Region{Locale: "en_US@calendar=japanese;rg=gbzzzz", FirstWeekday: 1, HourCycle: 24}, Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24}},
		{platform.Region{Locale: "zh-Hans_CN", FirstWeekday: 0, HourCycle: 24}, Region{Locale: "zh-Hans-CN", FirstWeekday: time.Sunday, HourCycle: 24}},
		{platform.Region{FirstWeekday: -1}, Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12}},
	} {
		fb.SetSystemRegion(c.system)
		onMain(func() {})
		if got := App.Region(); got != c.want {
			t.Errorf("%+v: Region() = %+v, want %+v", c.system, got, c.want)
		}
	}
	if got := heard.get(); len(got) != 8 {
		t.Errorf("listeners heard %d changes, want 8: %+v", len(got), got)
	}
}

// pageLocale returns the language the bridge configuration of the scripts
// of a page's document start tells it.
func pageLocale(t *testing.T, scripts []platform.UserScript) (locale, dir string) {
	t.Helper()
	for _, s := range scripts {
		if i := strings.Index(s.Source, `"locale":"`); i >= 0 && strings.Contains(s.Source, "__MYGO_CONFIG__") {
			rest := s.Source[i+len(`"locale":"`):]
			locale = rest[:strings.IndexByte(rest, '"')]
			j := strings.Index(s.Source, `"dir":"`)
			rest = s.Source[j+len(`"dir":"`):]
			return locale, rest[:strings.IndexByte(rest, '"')]
		}
	}
	t.Fatal("no bridge in the scripts")
	return "", ""
}

func TestPagesLocale(t *testing.T) {
	localeTest(t, []string{"en", "ar", "de"}, "en-US")
	_, fw := readyWindow(t, WindowOptions{Page: PageOptions{PreloadScript: "window.preloaded = 1"}})
	if l, d := pageLocale(t, fw.UserScripts()); l != "en" || d != "ltr" {
		t.Errorf("the page starts with %q %q", l, d)
	}
	fw.ClearScripts()
	App.SetLocale("ar-EG")
	onMain(func() {})
	scripts := fw.UserScripts()
	if l, d := pageLocale(t, scripts); l != "ar" || d != "rtl" {
		t.Errorf("a page loaded later starts with %q %q", l, d)
	}
	onMain(func() {
		if !fb.UIRightToLeft {
			t.Error("the backend lays its widgets out left to right in Arabic")
		}
	})
	if !slices.ContainsFunc(scripts, func(s platform.UserScript) bool { return s.Source == "window.preloaded = 1" }) {
		t.Error("the preload script was lost")
	}
	evaled := strings.Join(fw.Scripts(), "\n")
	if !strings.Contains(evaled, bridge.LocaleEvent) || !strings.Contains(evaled, `"locale":"ar","dir":"rtl"`) {
		t.Errorf("the page heard %q", evaled)
	}
	// A language the app shows already tells the page nothing.
	fw.ClearScripts()
	App.SetLocale("ar")
	onMain(func() {})
	if s := strings.Join(fw.Scripts(), "\n"); strings.Contains(s, bridge.LocaleEvent) {
		t.Errorf("the page heard %q", s)
	}
}

func TestContentLocale(t *testing.T) {
	localeTest(t, []string{"en", "ja"}, "en-US")
	var locale string
	var region i18n.Region
	frames := 0
	_, _, s := contentWindow(t, func(c *ui.Context) {
		locale, region, frames = c.Locale(), c.Region(), frames+1
		ui.Text(c, "直")
	})
	if locale != "en" || region.Locale != "en-US" {
		t.Errorf("the view sees %q %+v", locale, region)
	}
	before := frames
	App.SetLocale("ja")
	onMain(func() { s.Frame() })
	if locale != "ja" || frames == before {
		t.Errorf("after SetLocale(ja) the view sees %q in %d frames", locale, frames-before)
	}
	fb.SetSystemRegion(platform.Region{Locale: "ja_JP", FirstWeekday: 0, HourCycle: 24})
	onMain(func() { s.Frame() })
	if region.Locale != "ja-JP" || region.HourCycle != 24 {
		t.Errorf("after the region changed the view sees %+v", region)
	}
}

func TestMenuRelabel(t *testing.T) {
	localeTest(t, []string{"en", "zh-Hans"}, "en")
	own := &MenuItem{Role: RolePaste, Label: "Insert"}
	edit := &MenuItem{Role: RoleEditMenu, Submenu: []*MenuItem{{Role: RoleCopy}, own}}
	quit := &MenuItem{Role: RoleQuit}
	m := NewMenu([]*MenuItem{edit, quit})
	w, fw := testWindow(t, WindowOptions{})
	w.SetMenu(m)
	onMain(func() {})
	if edit.Label != "Edit" || edit.Submenu[0].Label != "Copy" {
		t.Fatalf("in English the menu says %q, %q", edit.Label, edit.Submenu[0].Label)
	}
	App.SetLocale("zh-Hans")
	onMain(func() {})
	if got := edit.Submenu[0].Label; got != "复制" {
		t.Errorf("Copy says %q", got)
	}
	if edit.Label != "编辑" || own.Label != "Insert" {
		t.Errorf("Edit says %q, the app's own label %q", edit.Label, own.Label)
	}
	if quit.Label == "Quit" || quit.Label == "Exit" || strings.HasPrefix(quit.Label, "Quit ") {
		t.Errorf("Quit says %q", quit.Label)
	}
	onMain(func() {
		menu := fw.Menu()
		if menu == nil || menu.Items[0].Label != "编辑" || menu.Items[0].Submenu.Items[0].Label != "复制" {
			t.Errorf("the window's menu was not shown again: %+v", menu)
		}
	})
	App.SetLocale("")
	onMain(func() {})
	if got := edit.Submenu[0].Label; got != "Copy" {
		t.Errorf("back in English, Copy says %q", got)
	}
}

// TestMenuBeforeRun checks that the items of a menu made before the app
// knew the language it shows, as main makes the app's menu before Run,
// take their roles' labels in that language as it starts.
func TestMenuBeforeRun(t *testing.T) {
	localeTest(t, []string{"en", "zh-Hans"}, "zh-CN")
	onMain(func() { locales.shown.Store(nil) }) // as before Run
	edit := &MenuItem{Role: RoleEditMenu, Submenu: []*MenuItem{{Role: RoleCopy}}}
	NewMenu([]*MenuItem{edit})
	if edit.Label != "Edit" {
		t.Fatalf("before the app knew its language, Edit says %q", edit.Label)
	}
	onMain(App.startLocale)
	if edit.Label != "编辑" || edit.Submenu[0].Label != "复制" {
		t.Errorf("as the app starts, the menu says %q, %q", edit.Label, edit.Submenu[0].Label)
	}
}

// TestSavedLocaleWhereRunPutsIt checks that the language saved in an
// earlier run is read where the app's paths are as it runs, even when
// something asked for the language before they were set.
func TestSavedLocaleWhereRunPutsIt(t *testing.T) {
	dir := localeTest(t, []string{"en", "zh-Hans", "de"}, "en")
	elsewhere := t.TempDir()
	App.SetPath(PathUserData, elsewhere)
	onMain(func() {
		locales.mu.Lock()
		locales.loaded, locales.chosen = false, false
		locales.mu.Unlock()
		loadLocale() // read early, under the paths of then
	})
	App.SetPath(PathUserData, dir)
	App.SetLocale("de") // saved where the app runs
	onMain(func() {})
	onMain(func() {
		locales.mu.Lock()
		locales.override, locales.loaded, locales.chosen = "", true, false
		locales.mu.Unlock()
		App.startLocale()
	})
	if got := App.Locale(); got != "de" {
		t.Errorf("Locale() = %q after starting, want the saved de", got)
	}
}

// TestCatalogsReadLocale checks that the accessors of catalogs say their
// messages in the language the app shows, as MyGo's own strings do.
func TestCatalogsReadLocale(t *testing.T) {
	localeTest(t, []string{"en", "zh-Hans"}, "zh-CN")
	if got := i18n.Locale(); got != "zh-Hans" {
		t.Errorf("i18n.Locale() = %q, want zh-Hans", got)
	}
	App.SetLocale("en-XA")
	onMain(func() {})
	if got := i18n.Locale(); got != "en-XA" {
		t.Errorf("i18n.Locale() = %q under the pseudo locale", got)
	}
}

// TestContentDirection checks that native UI is laid out in the direction
// of the language the app shows.
func TestContentDirection(t *testing.T) {
	localeTest(t, []string{"en", "ar"}, "en-US")
	var dir ui.Direction
	_, _, s := contentWindow(t, func(c *ui.Context) { dir = c.Direction() })
	if dir != ui.LeftToRight {
		t.Errorf("English is laid out %v", dir)
	}
	App.SetLocale("ar")
	onMain(func() { s.Frame() })
	if dir != ui.RightToLeft {
		t.Errorf("Arabic is laid out %v", dir)
	}
}

// TestBackendTexts checks that the backends' own words (platform.Texts)
// and a message dialog's default button follow the language the app
// shows.
func TestBackendTexts(t *testing.T) {
	localeTest(t, []string{"en", "zh-Hans"}, "en")
	onMain(func() {})
	if got := platform.CurrentTexts(); got.OK != "OK" || got.Cancel != "Cancel" || got.About != "About \x00" {
		t.Fatalf("in English: %+v", got)
	}
	App.SetLocale("zh-Hans")
	onMain(func() {})
	if got := platform.CurrentTexts(); got.OK != "好" || got.Cancel != "取消" || got.Open != "打开" || got.SelectFolder != "选择文件夹" || got.About != "关于\x00" {
		t.Errorf("in Chinese: %+v", got)
	}
	var buttons []string
	fb.OnMessageBox = func(o *platform.MessageBoxOptions) platform.MessageBoxResult {
		buttons = o.Buttons
		return platform.MessageBoxResult{}
	}
	t.Cleanup(func() { fb.OnMessageBox = nil })
	if _, err := Dialog.Message(MessageOptions{Message: "x"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(buttons, []string{"好"}) {
		t.Errorf("a message's default button: %q", buttons)
	}
}

// TestMissingTranslations checks which languages of an app the app is
// told, as it starts, it ships without MyGo's translation.
func TestMissingTranslations(t *testing.T) {
	translations := []string{"ja", "zh-Hans", "zh-Hant"}
	got := missingTranslations([]string{"en", "zh-TW", "de", "zh-Hans", "ja"}, translations, []string{"en", "zh-Hans"})
	want := [][2]string{{"zh-TW", "zh-Hant"}, {"ja", "ja"}}
	if !slices.Equal(got, want) {
		t.Errorf("missing %v, want %v", got, want)
	}
	if got := missingTranslations([]string{"en", "zh-TW"}, l10n.Languages, l10n.Catalog.Languages()); got != nil {
		t.Errorf("with the translations the tests import: %v", got)
	}
}
