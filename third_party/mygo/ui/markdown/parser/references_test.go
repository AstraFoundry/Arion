package parser

import (
	"fmt"
	"strings"
	"testing"
)

// refsOf lists the mentions and references of src's blocks, each as
// kind:text@start, checking that its offsets are its source.
func refsOf(t *testing.T, ext Extensions, src string) string {
	t.Helper()
	d := New(ext)
	d.Set(src)
	var out []string
	for i := range d.Len() {
		for _, in := range d.Inlines(i) {
			if in.Kind != Mention && in.Kind != Reference {
				continue
			}
			if got := src[in.Start:in.End]; got != in.Text {
				t.Errorf("%q: %v %q is %q in the source", src, in.Kind, in.Text, got)
			}
			out = append(out, fmt.Sprintf("%v:%s@%d", in.Kind, in.Text, in.Start))
		}
	}
	return strings.Join(out, " ")
}

// The cases are GitHub's, as its Markdown API rendered them, but for
// whether the person or the issue exists, which a view resolves.
func TestMentions(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"hi @octocat and @hubot.", "mention:@octocat@3 mention:@hubot@16"},
		{"foo@octocat x@y _@octocat", ""},
		{"(@a) \"@b\" /@c .@d", "mention:@a@1 mention:@b@6 mention:@c@11 mention:@d@15"},
		{"@a, @b! @c? @d: @e.", "mention:@a@0 mention:@b@4 mention:@c@8 mention:@d@12 mention:@e@16"},
		{"@foo- @foo_bar @-foo", "mention:@foo-@0"},
		{"cc @github/docs", "mention:@github/docs@3"},
		{"@octocat@x @octocat.dev", "mention:@octocat@0"},
		{"mail me@example.com", ""},
		{"`@octocat` [@octocat](https://x.y) **@octocat** <@octocat>", "mention:@octocat@37 mention:@octocat@49"},
		{"é@octocat", ""},
		{"@" + strings.Repeat("a", 39) + " @" + strings.Repeat("a", 40), "mention:@" + strings.Repeat("a", 39) + "@0"},
		{"# Hi @octocat", "mention:@octocat@5"},
		{"| @octocat |\n|---|", "mention:@octocat@2"},
		{"_a_@octocat **b**@octocat", "mention:@octocat@3 mention:@octocat@17"},
		{`<a href="u">@octocat</a> <code>@octocat</code> <b>@octocat</b> <br/>@octocat`, "mention:@octocat@50 mention:@octocat@68"},
		{"<pre>\n@a</pre>", ""},
	} {
		if got := refsOf(t, Default, c.src); got != c.want {
			t.Errorf("%q gives %q, want %q", c.src, got, c.want)
		}
	}
	if got := refsOf(t, Default&^ExtMentions, "hi @octocat #1"); got != "reference:#1@12" {
		t.Errorf("without ExtMentions: %q", got)
	}
}

func TestReferences(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"see #874, (#871). a#874 #874abc #874. #0", "reference:#874@4 reference:#871@11 reference:#874@32"},
		{"GH-874 gh-874 xGH-1", "reference:GH-874@0 reference:gh-874@7"},
		{"octocat/hello-world#12 (b/c_d.e#3) (a.b/c#3)", "reference:octocat/hello-world#12@0 reference:b/c_d.e#3@24"},
		{"x.org/a/b#1 /a/b#1 _#874", ""},
		{"`#1` [#1](u) *#1*", "reference:#1@14"},
		{"#12345678901 #1234567890", "reference:#1234567890@13"},
		{"_a_#1 me@x.com#2 #3#4", "reference:#1@3 reference:#2@14 reference:#3@17"},
		{"<code>#1</code> <A HREF=u>#2</a> #3", "reference:#3@33"},
	} {
		if got := refsOf(t, Default, c.src); got != c.want {
			t.Errorf("%q gives %q, want %q", c.src, got, c.want)
		}
	}
	if got := refsOf(t, Default&^ExtReferences, "hi @octocat #1"); got != "mention:@octocat@3" {
		t.Errorf("without ExtReferences: %q", got)
	}
}

func TestLoginAndRef(t *testing.T) {
	d := New(Default)
	d.Set("@octocat @github/docs #12 GH-7 octocat/hello#3")
	type ref struct {
		login, repo string
		n           int
	}
	var got []ref
	for _, in := range d.Inlines(1) {
		if in.Kind == Mention || in.Kind == Reference {
			r, n := in.Ref()
			got = append(got, ref{in.Login(), r, n})
		}
	}
	want := []ref{{"octocat", "", 0}, {"github/docs", "", 0}, {"", "", 12}, {"", "", 7}, {"", "octocat/hello", 3}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Inlines made by hand do not panic.
	if login := (Inline{Kind: Mention}).Login(); login != "" {
		t.Errorf("an empty mention's login is %q", login)
	}
	if repo, n := (Inline{Kind: Reference, Text: "x"}).Ref(); repo != "" || n != 0 {
		t.Errorf("a reference without a number is %q, %d", repo, n)
	}
}
