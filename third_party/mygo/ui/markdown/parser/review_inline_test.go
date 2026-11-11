package parser

import (
	"strings"
	"testing"
)

// Bugs found reviewing the inline parser; each test fails on 088606d5.

// An email autolink in text after an entity or an escape gets the
// offsets of the decoded text, not of the source.
func TestReviewEmailOffsets(t *testing.T) {
	for _, src := range []string{"a &amp; b x@y.com", `a \* b x@y.com`} {
		d := Parse(src)
		for _, in := range d.Inlines(1) {
			if in.Kind == Link {
				if got := d.Source()[in.Start:in.End]; got != "x@y.com" {
					t.Errorf("%q: link [%d,%d) is %q, want %q", src, in.Start, in.End, got, "x@y.com")
				}
			}
		}
	}
}

// Quadratic time: nested footnote-like brackets, $ math with
// ExtDollarMath, and \( without \).
func TestReviewQuadratic(t *testing.T) {
	checkLinear(t, "footnote brackets", Default, func(n int) string { return strings.Repeat("[^a", n) + strings.Repeat("]", n) })
	checkLinear(t, "dollar openers", Default|ExtDollarMath, repeat("$a "))
	checkLinear(t, "\\( openers", Default, repeat(`\(`))
}

// HTML entities are only names of the table followed by ;, not a
// legacy name's prefix.
func TestReviewEntityPrefix(t *testing.T) {
	if got, want := render("&copyright; &notit;", Default), "<p>&amp;copyright; &amp;notit;</p>\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReviewHealing(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		// A tilde for "about" strikes the rest of the paragraph through.
		{"about ~200 keys. See", "<p>about ~200 keys. See</p>\n"},
		// Strong next to CJK text is not healed: ** shows.
		{"这是**重要", "<p>这是<strong>重要</strong></p>\n"},
		// An open code span skips healing emphasis: ** shows.
		{"Edit **`config.ya", "<p>Edit <strong><code>config.ya</code></strong></p>\n"},
		// $$ inside a code span counts as open math: $$ shows.
		{"Use `$$` for display math", "<p>Use <code>$$</code> for display math</p>\n"},
		// A destination with parentheses: raw syntax, and the added ) shows.
		{"See [Go](https://en.wikipedia.org/wiki/Go_(lang", `<p>See <a href="https://en.wikipedia.org/wiki/Go_(lang)">Go</a></p>` + "\n"},
		// The text of a link being written shows its [.
		{"See the [Go blog", "<p>See the Go blog</p>\n"},
		// Strikethrough being closed shows its tildes.
		{"a ~~struck~", "<p>a <del>struck</del></p>\n"},
		// A < destination: the added ) shows.
		{"see [a](<htt", `<p>see <a href="htt">a</a></p>` + "\n"},
	} {
		if got := streamed(c.src); got != c.want {
			t.Errorf("%q:\ngot  %q\nwant %q", c.src, got, c.want)
		}
	}
}

// Extended autolinks end at punctuation beyond ASCII, such as the CJK
// comma, and at spaces beyond ASCII, but not at letters beyond ASCII.
func TestReviewAutolinkUnicodePunct(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"信中附上链接：https://vishnuai.in/mediakit，并留下电话联系方式。", `<p>信中附上链接：<a href="https://vishnuai.in/mediakit">https://vishnuai.in/mediakit</a>，并留下电话联系方式。</p>` + "\n"},
		{"见 https://example.com/a。", `<p>见 <a href="https://example.com/a">https://example.com/a</a>。</p>` + "\n"},
		{"“www.example.com”", `<p>“<a href="http://www.example.com">www.example.com</a>”</p>` + "\n"},
		{"（www.example.com/路径）", `<p>（<a href="http://www.example.com/%E8%B7%AF%E5%BE%84">www.example.com/路径</a>）</p>` + "\n"},
		{"https://例子.测试/页面", `<p><a href="https://%E4%BE%8B%E5%AD%90.%E6%B5%8B%E8%AF%95/%E9%A1%B5%E9%9D%A2">https://例子.测试/页面</a></p>` + "\n"},
		{"https://en.wikipedia.org/wiki/1990–91_season", `<p><a href="https://en.wikipedia.org/wiki/1990%E2%80%9391_season">https://en.wikipedia.org/wiki/1990–91_season</a></p>` + "\n"},
		{"https://example.com\u3000下一句", `<p><a href="https://example.com">https://example.com</a>` + "\u3000下一句</p>\n"},
		{"https://example.com\u2028下一句", `<p><a href="https://example.com">https://example.com</a>` + "\u2028下一句</p>\n"},
		{"https://example.com/a\u2028下一句", `<p><a href="https://example.com/a">https://example.com/a</a>` + "\u2028下一句</p>\n"},
		{"见\u2029www.example.com", "<p>见\u2029" + `<a href="http://www.example.com">www.example.com</a></p>` + "\n"},
		{"邮箱：a@example.com，谢谢", `<p>邮箱：<a href="mailto:a@example.com">a@example.com</a>，谢谢</p>` + "\n"},
		// Written links keep their whole destination and text.
		{"[文档，下载](https://example.org/文档，下载)", `<p><a href="https://example.org/%E6%96%87%E6%A1%A3%EF%BC%8C%E4%B8%8B%E8%BD%BD">文档，下载</a></p>` + "\n"},
		{"[https://example.org/文档，下载](https://example.org/文档，下载)", `<p><a href="https://example.org/%E6%96%87%E6%A1%A3%EF%BC%8C%E4%B8%8B%E8%BD%BD">https://example.org/文档，下载</a></p>` + "\n"},
		{"<https://example.org/x，y>", `<p><a href="https://example.org/x%EF%BC%8Cy">https://example.org/x，y</a></p>` + "\n"},
		{"[见 https://example.org/a，b](https://c.org)", `<p><a href="https://c.org">见 https://example.org/a，b</a></p>` + "\n"},
	} {
		if got := render(c.src, Default); got != c.want {
			t.Errorf("%q:\ngot  %q\nwant %q", c.src, got, c.want)
		}
	}
}
