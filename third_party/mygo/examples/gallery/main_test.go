package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

func TestSidebarNavigation(t *testing.T) {
	g := &gallery{router: ui.NewRouter("/overview"), sectionsOpen: [2]bool{true, true}}
	g.router.Transition = ui.TransitionNone
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			g.sidebar(c)
			ui.Column(c).Grow(1).Children(func() {
				g.toolbar(c)
				g.router.View(c, func(r *ui.Route) {
					r.Title(pageOf(r.Path()))
					ui.Text(c, "Page: "+r.Path())
				})
			})
		})
	}, 980, 720)
	check := func(want string) {
		t.Helper()
		if got := g.router.Path(); got != want || !tt.HasText("Page: "+want) {
			t.Fatalf("want %s, router at %s, texts %q", want, got, tt.Texts())
		}
	}
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	for _, page := range pages {
		click(page)
		check(pagePath(page))
	}
	// Clicking the current page adds no history entry.
	click("Languages")
	click("Back")
	check("/motion")
	click("Forward")
	check("/languages")
	// Arrow navigation uses the selection that survived the click's rebuild.
	click("Text")
	tt.Key(0, ui.KeyDown)
	check("/rich-text")
	// History and other navigation synchronize the sidebar's selection.
	g.router.Back()
	tt.Frame()
	check("/text")
	tt.Key(0, ui.KeyDown)
	check("/rich-text")
	g.router.Push("/list/42")
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	check("/charts")
}

func TestRowStarAndBreadcrumb(t *testing.T) {
	g := &gallery{router: ui.NewRouter("/list/42"), starred: map[int]bool{}}
	g.router.Transition = ui.TransitionNone
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			g.toolbar(c)
			g.router.View(c, func(r *ui.Route) {
				if r.Match("/list/{row}") {
					g.listRow(c, r)
				}
			})
		})
	}, 600, 400)
	for _, want := range []bool{true, false} {
		if err := tt.Click("Starred"); err != nil {
			t.Fatal(err)
		}
		if g.starred[42] != want {
			t.Fatalf("row's star = %v, want %v", g.starred[42], want)
		}
	}
	if err := tt.Click("List"); err != nil {
		t.Fatal(err)
	}
	if got := g.router.Path(); got != "/list" {
		t.Fatalf("breadcrumb went to %q", got)
	}
}

func TestListDerivedSelection(t *testing.T) {
	g := &gallery{router: ui.NewRouter("/list"), picked: -1, tree: map[string]bool{}, starred: map[int]bool{}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { g.list(c) })
	}, 980, 720)
	if err := tt.Click("Row 2"); err != nil {
		t.Fatal(err)
	}
	if g.picked != 2 {
		t.Fatalf("click picked %d", g.picked)
	}
	tt.Key(0, ui.KeyDown)
	if g.picked != 3 {
		t.Fatalf("Down picked %d", g.picked)
	}
	if err := tt.Click("Filter"); err != nil {
		t.Fatal(err)
	}
	tt.Type("99")
	if err := tt.Click("Row 99"); err != nil {
		t.Fatal(err)
	}
	if g.picked != 99 {
		t.Fatalf("filtered row picked %d", g.picked)
	}
	tt.Key(0, ui.KeyEnter)
	if got := g.router.Path(); got != "/list/99" {
		t.Fatalf("Enter went to %q", got)
	}
}

func TestOverviewCounter(t *testing.T) {
	g := &gallery{router: ui.NewRouter("/overview")}
	tt := ui.NewTester(g.overview, 980, 720)
	for _, step := range []struct{ click, want string }{{"Increment", "1"}, {"Increment", "2"}, {"−", "1"}} {
		if err := tt.Click(step.click); err != nil {
			t.Fatal(err)
		}
		if !tt.HasText(step.want) {
			t.Fatalf("after %s, want %s, texts %q", step.click, step.want, tt.Texts())
		}
	}
}

// TestLanguagesPage checks the Languages page in English and in Chinese:
// the formats and MyGo's own strings follow the window's language.
func TestLanguagesPage(t *testing.T) {
	g := &gallery{router: ui.NewRouter("/languages"), now: time.Date(2026, 1, 5, 13, 7, 0, 0, time.UTC), meeting: time.Date(2026, 1, 5, 9, 30, 0, 0, time.UTC)}
	tt := ui.NewTester(g.languages, 980, 1400)
	tt.SetRegion(i18n.Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12})
	for _, want := range []string{"1,234,567.891", "Monday, January 5, 2026", "1:07 PM", "January 2026", "简体中文 (zh-Hans)", "Pseudo right to left (ar-XB)", "12,34,567.891"} {
		if !tt.HasText(want) {
			t.Errorf("no %q in %q", want, tt.Texts())
		}
	}
	tt.SetLocale("zh-Hans")
	for _, want := range []string{"2026年1月5日星期一", "2026年1月", "上个月", "上午/下午"} {
		if !tt.HasText(want) {
			t.Errorf("in Chinese, no %q in %q", want, tt.Texts())
		}
	}
	// The menu of languages chooses one.
	if err := tt.Click("Deutsch (de)"); err != nil {
		t.Fatal(err)
	}
	if g.language != "de" {
		t.Errorf("after clicking Deutsch, the language chosen is %q", g.language)
	}
}
