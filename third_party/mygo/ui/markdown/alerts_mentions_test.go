package markdown

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

func TestViewAlert(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) { View(c, "> [!WARNING]\n> Mind the gap.", nil) }, 600, 400)
	for _, s := range []string{"Warning", "Mind the gap."} {
		if !tt.HasText(s) {
			t.Errorf("no %q in %q", s, tt.Texts())
		}
	}
	if tt.HasText("[!WARNING]") {
		t.Errorf("the marker shows: %q", tt.Texts())
	}
	// A marker alone in its paragraph shows nothing either.
	tt = ui.NewTester(func(c *ui.Context) { View(c, "> [!TIP]\n>\n> - one", nil) }, 600, 400)
	if !tt.HasText("Tip") || tt.HasText("[!TIP]") {
		t.Errorf("got %q", tt.Texts())
	}
}

// resolver knows octocat and issue 12 of the document's repository, as
// paths of the app, and whom it knows once told.
type resolver struct{ known []string }

func (r *resolver) resolve(in parser.Inline) (string, bool) {
	switch in.Kind {
	case parser.Mention:
		if slices.Contains(r.known, in.Login()) {
			return "/users/" + in.Login(), true
		}
	case parser.Reference:
		if repo, n := in.Ref(); repo == "" && n == 12 {
			return "/pulls/12", true
		}
	}
	return "", false
}

func TestViewMentions(t *testing.T) {
	r := &resolver{known: []string{"octocat"}}
	var opened []string
	o := &Options{Resolve: r.resolve, OnLink: func(url string) { opened = append(opened, url) }}
	src := "Thanks @octocat and @nobody, see #12 and #13."
	tt := ui.NewTester(func(c *ui.Context) { View(c, src, o) }, 600, 400)
	for _, s := range []string{"@octocat", "#12"} {
		if err := tt.Click(s); err != nil {
			t.Fatalf("%s: %v in %q", s, err, tt.Texts())
		}
	}
	if !slices.Equal(opened, []string{"/users/octocat", "/pulls/12"}) {
		t.Errorf("opened %q", opened)
	}
	// Those it does not know stay text.
	if err := tt.Click("@nobody"); err == nil {
		t.Error("@nobody is a link")
	}
	// Once the app knows them, they link.
	r.known = append(r.known, "nobody")
	tt.Frame()
	if err := tt.Click("@nobody"); err != nil || opened[len(opened)-1] != "/users/nobody" {
		t.Errorf("@nobody once known: %v, opened %q", err, opened)
	}
	// Resolve set later makes them links, and cleared, text again.
	var o2 Options
	tt = ui.NewTester(func(c *ui.Context) { View(c, src, &o2) }, 600, 400)
	if err := tt.Click("@octocat"); err == nil {
		t.Error("@octocat is a link without Resolve")
	}
	o2.Resolve, o2.OnLink = r.resolve, o.OnLink
	tt.Frame()
	if err := tt.Click("@octocat"); err != nil {
		t.Errorf("@octocat once Resolve is set: %v in %q", err, tt.Texts())
	}
	o2.Resolve = nil
	tt.Frame()
	if err := tt.Click("@octocat"); err == nil {
		t.Error("@octocat is still a link once Resolve is cleared")
	}
	// Without Resolve, they are text.
	tt = ui.NewTester(func(c *ui.Context) { View(c, src, nil) }, 600, 400)
	if !tt.HasText(src) {
		t.Errorf("not one text: %q", tt.Texts())
	}
}

func TestViewPreview(t *testing.T) {
	r := &resolver{known: []string{"octocat"}}
	var asked []string
	o := &Options{Resolve: r.resolve, Preview: func(c *ui.Context, dest string) func(tip ui.Element) {
		asked = append(asked, dest)
		if !strings.HasPrefix(dest, "/users/") {
			return nil
		}
		return func(tip ui.Element) {
			tip.Padding(8).Children(func() { ui.Text(c, "The Octocat") })
		}
	}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, "Hi @octocat, see [docs](https://go.dev).", o) }, 600, 400)
	at, ok := tt.Find("@octocat")
	if !ok {
		t.Fatalf("no @octocat in %q", tt.Texts())
	}
	tt.Move(at.X+at.W/2, at.Y+at.H/2)
	for range 10 {
		tt.Advance(100 * time.Millisecond)
		tt.Frame()
	}
	if !tt.HasText("The Octocat") {
		t.Errorf("no card while the pointer rests: %q", tt.Texts())
	}
	if !slices.Contains(asked, "https://go.dev") {
		t.Errorf("Preview not asked of every link: %q", asked)
	}
}

// Part of a mention written with an entity copies as that part of its
// source, as text does.
func TestViewCopyMentionEntity(t *testing.T) {
	src := "Thanks &#64;octocat now"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	text := rvParaText(st, 1)
	if text != "Thanks @octocat now" {
		t.Fatalf("text %q", text)
	}
	if got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, text, 8, 12)}); got != "octo" {
		t.Errorf("copied %q, want %q", got, "octo")
	}
}
