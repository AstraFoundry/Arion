package parser

import (
	"strings"
	"testing"
)

// The cases are GitHub's, as its Markdown API rendered them.
func TestAlerts(t *testing.T) {
	alert := func(kind, title, body string) string {
		return `<div class="markdown-alert markdown-alert-` + kind + `">` + "\n" +
			`<p class="markdown-alert-title">` + title + "</p>\n" + body + "</div>\n"
	}
	quote := func(body string) string { return "<blockquote>\n" + body + "</blockquote>\n" }
	for _, c := range []struct{ name, src, want string }{
		{"note", "> [!NOTE]\n> Body.", alert("note", "Note", "<p>Body.</p>\n")},
		{"every kind", "> [!TIP]\n> a\n\n> [!IMPORTANT]\n> a\n\n> [!WARNING]\n> a\n\n> [!CAUTION]\n> a",
			alert("tip", "Tip", "<p>a</p>\n") + alert("important", "Important", "<p>a</p>\n") +
				alert("warning", "Warning", "<p>a</p>\n") + alert("caution", "Caution", "<p>a</p>\n")},
		{"any case", "> [!note]\n> Body.", alert("note", "Note", "<p>Body.</p>\n")},
		{"spaces around the marker", ">  [!WARNING]   \n> Body.", alert("warning", "Warning", "<p>Body.</p>\n")},
		{"a blank line first", ">\n> [!NOTE]\n> Body.", alert("note", "Note", "<p>Body.</p>\n")},
		{"a lazy line", "> [!NOTE]\nlazy line", alert("note", "Note", "<p>lazy line</p>\n")},
		{"a blank line after", "> [!NOTE]\n>\n> Body.", alert("note", "Note", "<p>Body.</p>\n")},
		{"a list after", "> [!NOTE]\n> - one", alert("note", "Note", "<ul>\n<li>one</li>\n</ul>\n")},
		{"a heading after", "> [!CAUTION]\n> # Head", alert("caution", "Caution", "<h1>Head</h1>\n")},
		{"a second marker is text", "> [!NOTE]\n> [!TIP]\n> Body.", alert("note", "Note", "<p>[!TIP]\nBody.</p>\n")},
		{"text after the marker", "> [!NOTE] Body.", quote("<p>[!NOTE] Body.</p>\n")},
		{"the marker alone", "> [!TIP]", quote("<p>[!TIP]</p>\n")},
		{"an unknown kind", "> [!FOO]\n> Body.", quote("<p>[!FOO]\nBody.</p>\n")},
		{"nested in a quote", "> > [!NOTE]\n> > Body.", quote(quote("<p>[!NOTE]\nBody.</p>\n"))},
		{"in a list", "- item\n\n  > [!NOTE]\n  > Body.", "<ul>\n<li>\n<p>item</p>\n" + quote("<p>[!NOTE]\nBody.</p>\n") + "</li>\n</ul>\n"},
		{"after a paragraph", "> a\n> [!NOTE]\n> Body.", quote("<p>a\n[!NOTE]\nBody.</p>\n")},
		{"a marker after the marker", "> [!NOTE]\n>\n> [!TIP]\n> x", alert("note", "Note", "<p>[!TIP]\nx</p>\n")},
		{"a definition after the marker", "> [!NOTE]\n>\n> [a]: /u", quote("<p>[!NOTE]</p>\n")},
		{"a footnote after the marker", "> [!NOTE]\n> [^1]: x\n\ny[^1]", quote("<p>[!NOTE]</p>\n") + "<p>y<sup class=\"footnote-ref\"><a href=\"#fn-1\" id=\"fnref-1\" data-footnote-ref>1</a></sup></p>\n" +
			"<section class=\"footnotes\" data-footnotes>\n<ol>\n<li id=\"fn-1\">\n<p>x</p>\n<a href=\"#fnref-1\" class=\"footnote-backref\">↩</a>\n</li>\n</ol>\n</section>\n"},
	} {
		d := New(Default)
		d.Set(c.src)
		if got := d.HTML(); got != c.want {
			t.Errorf("%s: %q gives\n%s\nwant\n%s", c.name, c.src, got, c.want)
		}
		off := New(Default &^ ExtAlerts)
		off.Set(c.src)
		for i := range off.Len() {
			if off.Block(i).Alert != NoAlert {
				t.Errorf("%s: an alert without ExtAlerts", c.name)
			}
		}
	}
}

// An alert written a byte at a time, as a reply streams in, ends as it
// parses at once, and is one as soon as its text starts.
func TestAlertStreaming(t *testing.T) {
	src := "Intro.\n\n> [!WARNING]\n>\n> Mind the **gap**.\n> - and this\n\nAfter."
	want := New(Default)
	want.Set(src)
	d := New(Default)
	d.SetStreaming(true)
	for k := range len(src) {
		d.Append(src[k : k+1])
		if got := d.Source(); len(got) == len("Intro.\n\n> [!WARNING]\n>\n> M") {
			if q := d.Block(2); q.Kind != Quote || q.Alert != AlertWarning {
				t.Errorf("at %q: block 2 is %v, alert %v", got, q.Kind, q.Alert)
			}
		}
	}
	d.SetStreaming(false)
	if got, w := d.HTML(), want.HTML(); got != w {
		t.Errorf("streamed\n%s\nwant\n%s", got, w)
	}
}

// While streaming, a marker alone shows as the alert its quote will
// most likely become; once the text is complete, it is text as GitHub
// has it.
func TestAlertMarkerStreaming(t *testing.T) {
	d := New(Default)
	d.SetStreaming(true)
	d.Set("> [!WARNING]")
	if q := d.Block(1); q.Kind != Quote || q.Alert != AlertWarning {
		t.Errorf("streaming: %v, alert %v", q.Kind, q.Alert)
	}
	if got := d.HTML(); strings.Contains(got, "[!WARNING]") {
		t.Errorf("the marker shows while streaming: %s", got)
	}
	d.SetStreaming(false)
	if q := d.Block(1); q.Alert != NoAlert {
		t.Errorf("complete: alert %v in %s", q.Alert, d.HTML())
	}
}

// Editing the marker away, or back, changes the quote.
func TestAlertEdit(t *testing.T) {
	d := New(Default)
	d.Set("a\n\n> [!NOTE]\n> Body.\n\nb")
	if d.Block(2).Alert != AlertNote {
		t.Fatalf("no alert in %s", d.HTML())
	}
	d.Set("a\n\n> [NOTE]\n> Body.\n\nb")
	if d.Block(2).Alert != NoAlert {
		t.Errorf("an alert without its marker: %s", d.HTML())
	}
	d.Set("a\n\n> [!TIP]\n> Body.\n\nb")
	if d.Block(2).Alert != AlertTip {
		t.Errorf("no tip once edited back: %s", d.HTML())
	}
}
