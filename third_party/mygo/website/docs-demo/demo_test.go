package main

import (
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// directive is a page's preview: `<!-- preview button 120 -->`.
var directive = regexp.MustCompile(`<!-- preview (\S+) (\d+) -->`)

// TestPagesAndDemos checks that the previews of docs/ui name demos, each
// page's own, and that every demo is a page's.
func TestPagesAndDemos(t *testing.T) {
	pages, err := filepath.Glob("../../docs/ui/*.md")
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	for _, page := range pages {
		b, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range directive.FindAllStringSubmatch(string(b), -1) {
			name := m[1]
			if _, ok := demos[name]; !ok {
				t.Errorf("%s: no demo %q", page, name)
			}
			if want := strings.TrimSuffix(filepath.Base(page), ".md"); name != want {
				t.Errorf("%s previews %q, not its own demo %q", page, name, want)
			}
			used[name] = true
		}
		if strings.Contains(string(b), "<!-- preview") && !directive.Match(b) {
			t.Errorf("%s: a preview comment is not `<!-- preview name height -->`", page)
		}
	}
	for name := range demos {
		if !used[name] {
			t.Errorf("demo %q is no page's preview", name)
		}
	}
}

// TestDemosBuild builds every demo for a few frames, as a window of the
// size of the docs' frames does.
func TestDemosBuild(t *testing.T) {
	names := make([]string, 0, len(demos))
	for name := range demos {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			view := demos[name]
			tt := ui.NewTester(func(c *ui.Context) { preview(c, view) }, 680, 320)
			for range 3 {
				tt.Frame()
			}
		})
	}
}

// TestShots draws demos into PNG files, as their pages' frames show them:
// DEMO_SHOT names them, comma separated, or "all", and DEMO_DIR the
// directory; DEMO_DARK=1 draws them dark.
//
//	DEMO_SHOT=button,link DEMO_DIR=/tmp/shots go test -run TestShots ./website/docs-demo
func TestShots(t *testing.T) {
	list, dir := os.Getenv("DEMO_SHOT"), os.Getenv("DEMO_DIR")
	if list == "" || dir == "" {
		t.Skip("DEMO_SHOT and DEMO_DIR name the demos and where their pictures go")
	}
	heights := map[string]int{}
	pages, _ := filepath.Glob("../../docs/ui/*.md")
	for _, page := range pages {
		b, _ := os.ReadFile(page)
		for _, m := range directive.FindAllStringSubmatch(string(b), -1) {
			heights[m[1]], _ = strconv.Atoi(m[2])
		}
	}
	names := strings.Split(list, ",")
	if list == "all" {
		names = names[:0]
		for name := range demos {
			names = append(names, name)
		}
	}
	for _, name := range names {
		view, ok := demos[name]
		if !ok {
			t.Errorf("no demo %q", name)
			continue
		}
		h := heights[name]
		if h == 0 {
			h = 240
		}
		tt := ui.NewTester(func(c *ui.Context) { preview(c, view) }, 720, h)
		tt.SetScale(2)
		tt.SetDark(os.Getenv("DEMO_DARK") == "1")
		for range 3 {
			tt.Frame()
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}
