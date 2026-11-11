package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// linkIcons makes an object for links but those to anchors, and notes
// the box the last one drew in.
func linkIcons(drawn *ui.Rect) *Options {
	return &Options{LinkIcon: func(dest string, size float32) (InlineObject, bool) {
		if strings.HasPrefix(dest, "#") {
			return InlineObject{}, false
		}
		return InlineObject{Width: size, Height: size * 0.8, Depth: size * 0.2, Draw: func(p *ui.Painter, r ui.Rect) {
			if drawn != nil {
				*drawn = r
			}
		}}, true
	}}
}

// TestLinkIconCopy copies a link's icon, and the joiner after it, as
// nothing, as text and as Markdown, wherever the selection ends.
func TestLinkIconCopy(t *testing.T) {
	src := "See [docs](a.md) now\n"
	tt, get := rvView(&src, linkIcons(nil))
	st := get()
	text := "See \uFFFC\u2060docs now"
	if !tt.HasText(text) {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	k := rvElemOf(t, st, 1)
	for _, c := range []struct {
		from, to int
		text, md string
	}{
		{0, 14, "See docs now", "See [docs](a.md) now"},
		{4, 10, "docs", "[docs](a.md)"}, // from the icon
		{5, 10, "docs", "[docs](a.md)"}, // from the joiner
		{6, 8, "do", "[do](a.md)"},      // within the text
		{0, 5, "See ", "See "},          // to the icon
		{4, 6, "", ""},                  // the icon alone
		{8, 14, "cs now", "[cs](a.md) now"},
	} {
		parts := []ui.SelectedText{rvPart(st, k, text, c.from, c.to)}
		if got := st.copyText(parts); got != c.text {
			t.Errorf("[%d,%d) as text %q, want %q", c.from, c.to, got, c.text)
		}
		if got := st.copyMarkdown(parts); got != c.md {
			t.Errorf("[%d,%d) as Markdown %q, want %q", c.from, c.to, got, c.md)
		}
	}
	if got := st.copyMarkdown(nil); got != src {
		t.Errorf("the document as Markdown %q, want %q", got, src)
	}
}

// TestLinkIconSelectAll copies the whole view without its icons.
func TestLinkIconSelectAll(t *testing.T) {
	src := "See [docs](a.md) now\n"
	tt, _ := rvView(&src, linkIcons(nil))
	r, ok := tt.Find("See \uFFFC\u2060docs now")
	if !ok {
		t.Fatalf("no text in %q", tt.Texts())
	}
	tt.ClickAt(r.X+2, r.Y+r.H/2)
	tt.Command("selectAll")
	tt.Command("copy")
	if got, want := tt.Clipboard(), "See docs now"; got != want {
		t.Errorf("copied %q, want %q", got, want)
	}
}

// TestLinkIconWraps keeps an icon on the line of the link's first word
// wherever the line ends, and asks for it inside the link: a click on
// it opens the link.
func TestLinkIconWraps(t *testing.T) {
	for n := range 40 {
		src := strings.Repeat("i", 2*n) + " aaaa bbbb cccc dddd [Supercali](x.md)\n"
		var icon ui.Rect
		var clicked string
		o := linkIcons(&icon)
		o.OnLink = func(url string) { clicked = url }
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Width(260).Children(func() { View(c, src, o) })
		}, 300, 200)
		link, ok := tt.Find("\uFFFC\u2060Supercali")
		if !ok {
			t.Fatalf("no link in %q", tt.Texts())
		}
		if link.H > 2*icon.H || icon.Y < link.Y-1 || icon.X > link.X+1 {
			t.Fatalf("n=%d: icon at %+v, link at %+v: not on one line", n, icon, link)
		}
		tt.ClickAt(icon.X+icon.W/2, icon.Y+icon.H/2)
		if clicked != "x.md" {
			t.Fatalf("n=%d: a click on the icon opened %q", n, clicked)
		}
	}
}

// TestLinkIconNone leaves out icons where LinkIcon makes none, in code,
// and in links still being written.
func TestLinkIconNone(t *testing.T) {
	src := "[up](#top) and `[x](x.md)` and https://go.dev\n"
	tt, _ := rvView(&src, linkIcons(nil))
	if !tt.HasText("up") || !tt.HasText("\uFFFC\u2060https://go.dev") {
		t.Errorf("anchors or autolinks wrong in %q", tt.Texts())
	}
	for _, s := range tt.Texts() {
		if strings.Contains(s, "\uFFFC\u2060[x]") {
			t.Errorf("code has an icon in %q", s)
		}
	}
	stream := "See [do"
	o := linkIcons(nil)
	tt = ui.NewTester(func(c *ui.Context) { Stream(c, stream, o) }, 700, 300)
	for _, s := range tt.Texts() {
		if strings.ContainsRune(s, '\uFFFC') {
			t.Errorf("a pending link has an icon in %q", s)
		}
	}
}

// TestLinkIconNil keeps links as they are without LinkIcon.
func TestLinkIconNil(t *testing.T) {
	src := "See [docs](a.md) now\n"
	tt, _ := rvView(&src, &Options{})
	if !tt.HasText("See docs now") {
		t.Errorf("text in %q", tt.Texts())
	}
}
