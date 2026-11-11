package parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type specExample struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	Example  int    `json:"example"`
	Section  string `json:"section"`
}

func loadSpec(t testing.TB, name string) []specExample {
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var ex []specExample
	if err := json.Unmarshal(data, &ex); err != nil {
		t.Fatal(err)
	}
	return ex
}

func render(src string, ext Extensions) string {
	d := New(ext)
	d.Set(src)
	return d.HTML()
}

// TestCommonMark runs the examples of the CommonMark spec, 0.31.2.
func TestCommonMark(t *testing.T) {
	failed := 0
	for _, ex := range loadSpec(t, "commonmark.json") {
		if got := render(ex.Markdown, 0); got != ex.HTML {
			failed++
			t.Errorf("example %d (%s):\n%q\ngot  %q\nwant %q", ex.Example, ex.Section, ex.Markdown, got, ex.HTML)
		}
	}
	if failed > 0 {
		t.Logf("%d failed", failed)
	}
}

// TestGFM runs the examples of GitHub's extensions in the GFM spec.
func TestGFM(t *testing.T) {
	for _, ex := range loadSpec(t, "gfm.json") {
		if !strings.Contains(ex.Section, "(extension)") || ex.Section == "Disallowed Raw HTML (extension)" {
			continue
		}
		if got := render(ex.Markdown, GFM); got != ex.HTML {
			t.Errorf("example %d (%s):\n%q\ngot  %q\nwant %q", ex.Example, ex.Section, ex.Markdown, got, ex.HTML)
		}
	}
}
