//go:build !js

package e2e

import (
	"image/color"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

type themeColorChange struct {
	c  color.NRGBA
	ok bool
}

// A page's theme color comes from its <meta name="theme-color">, with the
// media of the app's appearance, as the engine computes the color, and
// changes with the metas, the appearance and the page, telling only of
// changes.
func TestPageThemeColor(t *testing.T) {
	defer mygo.Theme.SetSource(mygo.Theme.Source())
	mygo.Theme.SetSource(mygo.ThemeLight)
	w := newWindow(t, mygo.WindowOptions{Title: "Theme color", Width: 300, Height: 200})
	p := w.Page()
	changes := make(chan themeColorChange, 64) // listeners must not block
	defer p.OnThemeColorChanged(func(c color.NRGBA, ok bool) { changes <- themeColorChange{c, ok} })()
	next := func(what string, c color.NRGBA, ok bool) {
		t.Helper()
		got := wait.For(t, what, changes)
		if want := (themeColorChange{c, ok}); got != want {
			t.Fatalf("%s: changed to %v, want %v", what, got, want)
		}
		if gc, gok := p.ThemeColor(); gc != c || gok != ok {
			t.Fatalf("%s: ThemeColor = %v, %v; want %v, %v", what, gc, gok, c, ok)
		}
	}
	dodgerBlue := color.NRGBA{R: 30, G: 144, B: 255, A: 255}
	dark := color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}
	const metas = `<!doctype html><html><head>
<meta name="theme-color" media="(prefers-color-scheme: dark)" content="#123">
<meta name="Theme-Color" content="dodgerblue">
<meta name="color-scheme" content="light dark"></head><body>%s</body></html>`

	p.LoadHTML(strings.Replace(metas, "%s", "one", 1), "")
	next("the page loaded", dodgerBlue, true)

	mygo.Theme.SetSource(mygo.ThemeDark)
	next("dark appearance", dark, true)
	mygo.Theme.SetSource(mygo.ThemeLight)
	next("light appearance", dodgerBlue, true)

	// Colors in any syntax, which the engine computes.
	mustEval(t, w, `document.querySelectorAll('meta[name="theme-color" i]')[1].content = " color(srgb 0 0 1) "`)
	next("the content changed", color.NRGBA{B: 255, A: 255}, true)
	mustEval(t, w, `document.querySelectorAll('meta[name="theme-color" i]')[1].content = "not a color"`)
	next("not a color", color.NRGBA{}, false)
	mustEval(t, w, `document.querySelectorAll('meta[name="theme-color" i]')[1].content = "rebeccapurple"`)
	next("a named color", color.NRGBA{R: 102, G: 51, B: 153, A: 255}, true)
	mustEval(t, w, `document.querySelectorAll('meta[name="theme-color" i]').forEach((m) => m.remove())`)
	next("the metas removed", color.NRGBA{}, false)
	mustEval(t, w, `document.head.append(Object.assign(document.createElement("meta"), {name: "theme-color", content: "rgba(255, 0, 0, 0.5)"}))`)
	next("a meta added", color.NRGBA{R: 255, A: 128}, true)

	// A page without one has none; going between pages of the same color
	// tells nothing.
	p.LoadHTML(`<p>no theme color</p>`, "")
	next("a page without a theme color", color.NRGBA{}, false)
	p.LoadHTML(strings.Replace(metas, "%s", "two", 1), "")
	next("a page with one", dodgerBlue, true)
	p.LoadHTML(strings.Replace(metas, "%s", "three", 1), "")
	waitFor(t, w, `document.body.textContent === "three" && document.readyState === "complete"`)
	mustEval(t, w, `document.querySelectorAll('meta[name="theme-color" i]').forEach((m) => m.remove())`)
	next("after a page of the same color", color.NRGBA{}, false)

	// A page that loads in an appearance other than the system's has it
	// from its start: it never tells the system's color first.
	mygo.Theme.SetSource(mygo.ThemeDark)
	p.LoadHTML(strings.Replace(metas, "%s", "four", 1), "")
	next("a page loaded in the dark appearance", dark, true)
	mygo.Theme.SetSource(mygo.ThemeLight)
	next("light appearance again", dodgerBlue, true)
	p.LoadHTML(strings.Replace(metas, "%s", "five", 1), "")
	waitFor(t, w, `document.body.textContent === "five" && document.readyState === "complete"`)
	mustEval(t, w, `document.querySelectorAll('meta[name="theme-color" i]').forEach((m) => m.remove())`)
	next("after a page loaded in the light appearance", color.NRGBA{}, false)
}
