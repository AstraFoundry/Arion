//go:build !js

package e2e

import (
	"bytes"
	"os/exec"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// localeLanguages makes the app written in langs for a test, following
// the user's languages again after it.
func localeLanguages(t *testing.T, langs ...string) {
	t.Helper()
	mygo.App.SetLanguages(langs...)
	t.Cleanup(func() {
		mygo.App.SetLocale("")
		mygo.App.SetLanguages()
	})
}

// localePage records the language the bridge has as the page starts, and
// what its listeners hear of mygo:locale.
const localePage = `<!doctype html><p>locale</p><script>
window.started = mygo.locale + " " + mygo.dir;
window.heard = [];
mygo.on("mygo:locale", (p) => window.heard.push(p.locale + " " + p.dir));
</script>`

// TestLocaleAtDocumentStart checks, in a real web view, that a page loaded
// after App.SetLocale starts in the new language, and that its listeners
// hear of a language the page did not have only, not as it loads.
func TestLocaleAtDocumentStart(t *testing.T) {
	localeLanguages(t, "en", "ar", "ja")
	// SetLanguages queues its update. Choose the initial language on the
	// main thread after it, rather than reading the previous locale or
	// assuming the user's preferred language is written left to right.
	mygo.RunOnMain(func() { mygo.App.SetLocale("en") })
	w := newWindow(t, mygo.WindowOptions{Hidden: true})
	w.Page().LoadHTML(localePage, "app://localhost/")
	waitFor(t, w, `window.started === "en ltr"`)

	mygo.App.SetLocale("ar")
	waitFor(t, w, `mygo.locale === "ar" && mygo.dir === "rtl" && window.heard.join() === "ar rtl"`)
	if v, err := w.Page().Eval("navigator.languages.join()"); err == nil {
		t.Logf("the page's languages: %v", v)
	}

	w.Page().Reload()
	waitFor(t, w, `window.started === "ar rtl" && document.readyState === "complete"`)
	// Go tells a loading page the language again, which it has.
	time.Sleep(300 * time.Millisecond)
	waitFor(t, w, `window.heard.length === 0`)

	mygo.App.SetLocale("ja")
	waitFor(t, w, `window.heard.join() === "ja ltr"`)
	// Another page, in a new window, starts in it too.
	w2 := newWindow(t, mygo.WindowOptions{Hidden: true})
	w2.Page().LoadHTML(localePage, "app://localhost/")
	waitFor(t, w2, `window.started === "ja ltr"`)
}

// TestTextLanguage checks that native UI draws text in the language the
// app shows, which an element's Lang overrides: 直 and 骨 are drawn apart
// in Japanese and in Chinese.
func TestTextLanguage(t *testing.T) {
	if runtime.GOOS == "linux" && (!hasFonts("ja") || !hasFonts("zh-cn")) {
		t.Skip("fontconfig has no Japanese or Chinese fonts (fonts-noto-cjk)")
	}
	localeLanguages(t, "en", "ja", "zh-Hans")
	var seen atomic.Value
	var override atomic.Value
	override.Store("")
	view := func(c *ui.Context) {
		seen.Store(c.Locale())
		ui.Text(c, "直骨").FontSize(48).Lang(override.Load().(string))
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Text language", Width: 200, Height: 100, Content: ui.View(view)})
	capture := func(locale, lang string) []byte {
		t.Helper()
		override.Store(lang)
		mygo.App.SetLocale(locale)
		eventually(t, "the view in "+locale, func() bool { return seen.Load() == locale })
		w.Update(func() {})
		time.Sleep(100 * time.Millisecond)
		data, err := w.CapturePage()
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	ja, hans := capture("ja", ""), capture("zh-Hans", "")
	if bytes.Equal(ja, hans) {
		t.Fatal("直骨 looks the same in Japanese and in Chinese")
	}
	// A Japanese quote in a Chinese app, and the other way round.
	if got := capture("zh-Hans", "ja"); !bytes.Equal(got, ja) {
		t.Error("Lang(ja) in a Chinese app draws other glyphs than Japanese")
	}
	if got := capture("ja", "zh-Hans"); !bytes.Equal(got, hans) {
		t.Error("Lang(zh-Hans) in a Japanese app draws other glyphs than Chinese")
	}
}

// hasFonts reports whether fontconfig has fonts for a language.
func hasFonts(lang string) bool {
	out, err := exec.Command("fc-list", ":lang="+lang, "family").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}
