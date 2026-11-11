package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// rvView builds a View of *src and returns its tester and a getter of its
// state.
func rvView(src *string, o *Options) (*ui.Tester, func() *viewState) {
	var st *viewState
	tt := ui.NewTester(func(c *ui.Context) {
		root := View(c, *src, o)
		st = *ui.Local(root, stateKey{}, newState)
	}, 700, 3000)
	return tt, func() *viewState { return st }
}

// rvText returns the text of the element of block i noted for copying.
func rvParaText(st *viewState, i int) string {
	var b strings.Builder
	for _, pc := range st.paras[i].p.pieces {
		for _, s := range pc.spans {
			b.WriteString(s.Text)
		}
	}
	return b.String()
}

// rvPart returns a selected part of the k-th noted element.
func rvPart(st *viewState, k int, text string, from, to int) ui.SelectedText {
	return ui.SelectedText{ID: st.elems[k].id, Text: text, Start: from, End: to}
}

func rvElemOf(t *testing.T, st *viewState, block int) int {
	for k, e := range st.elems {
		if e.block == block {
			return k
		}
	}
	t.Fatalf("no element of block %d in %+v", block, st.elems)
	return -1
}

// A selection ending right before emphasis copies a stray closer.
func TestReviewCopyMarkdownEndBeforeEmphasis(t *testing.T) {
	src := "foo *bar* baz"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	text := rvParaText(st, 1)
	if text != "foo bar baz" {
		t.Fatalf("text %q", text)
	}
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, text, 0, 4)}) // "foo "
	if got != "foo " {
		t.Errorf("copied %q, want %q", got, "foo ")
	}
}

// The same, through a real drag.
func TestReviewCopyMarkdownEndBeforeEmphasisDrag(t *testing.T) {
	src := "foo *bar* baz"
	for dx := float32(10); dx < 40; dx += 0.5 {
		tt, _ := rvView(&src, nil)
		r, _ := tt.Find("foo bar baz")
		y := r.Y + r.H/2
		tt.Press(r.X+0.5, y)
		tt.Move(r.X+dx, y)
		tt.Release(r.X+dx, y)
		tt.Command("copy")
		if tt.Clipboard() != "foo " {
			continue
		}
		tt.Key(ui.Cmd|ui.Shift, ui.KeyC)
		if got := tt.Clipboard(); got != "foo " {
			t.Errorf("dragged over %q, copied as Markdown %q", "foo ", got)
		}
		return
	}
	t.Fatal("never selected \"foo \"")
}

func measure(t *testing.T, s string) (float32, float32) {
	var w, h float32
	ui.NewTester(func(c *ui.Context) {
		w, h = c.MeasureText(0, ui.Span{Text: s, Size: 14})
	}, 100, 100)
	return w, h
}

// A selection inside text with an entity maps to the wrong source.
func TestReviewCopyMarkdownEntity(t *testing.T) {
	src := "Tom &amp; Jerry *ran* home"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	text := rvParaText(st, 1)
	if text != "Tom & Jerry ran home" {
		t.Fatalf("text %q", text)
	}
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, text, 6, 11)}) // "Jerry"
	if got != "Jerry" {
		t.Errorf("copied %q, want %q", got, "Jerry")
	}
}

// A paragraph selected to its end copies the next line too, when a block
// follows without a blank line.
func TestReviewCopyMarkdownParagraphEnd(t *testing.T) {
	src := "para\n- item\n"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	text := rvParaText(st, 1)
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, text, 0, len(text))})
	if got != "para" {
		t.Errorf("copied %q, want %q", got, "para")
	}
}

// Lines 1-32 of a 40-line code block selected copy all 40.
func TestReviewCopyMarkdownCodeChunk(t *testing.T) {
	var lines []string
	for i := range 40 {
		lines = append(lines, "line"+itoa(i))
	}
	src := "```\n" + strings.Join(lines, "\n") + "\n```\n"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	chunk := strings.Join(lines[:32], "\n")
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, chunk, 6, len([]rune(chunk)))}) // from line 1
	want := strings.Join(lines[1:32], "\n")
	if got != want {
		t.Errorf("copied %d lines, want 31: %q", strings.Count(got, "\n")+1, got)
	}
}

// A paragraph that turns into a link once its reference is defined,
// while the source only grows.
func TestReviewStaleParaAfterRefDefined(t *testing.T) {
	src := "[foo]\n"
	tt, _ := rvView(&src, nil)
	if !tt.HasText("[foo]") {
		t.Fatalf("shows %q", tt.Texts())
	}
	src = "[foo]\n\n[foo]: https://go.dev\n"
	tt.Frame()
	whole, _ := rvView(&src, nil)
	if got, want := strings.Join(tt.Texts(), "|"), strings.Join(whole.Texts(), "|"); got != want {
		t.Errorf("grown shows %q, whole shows %q", got, want)
	}
}

// Stream by line asks for frames forever while waiting for a line end.
func TestReviewByLineFramesForever(t *testing.T) {
	text := "Hello world, no newline yet"
	o := &Options{Reveal: &Reveal{Unit: ByLine}}
	frames := 0
	tt := ui.NewTester(func(c *ui.Context) { frames++; Stream(c, text, o) }, 600, 400)
	tt.Advance(400 * time.Millisecond)
	frames = 0
	tt.Frame()
	if frames >= 20 || tt.FramePending() {
		t.Errorf("one Frame ran %d frames, one pending %v: the view keeps asking for them with nothing to show", frames, tt.FramePending())
	}
}

// Links without a scheme open outside a router: on Windows, ShellExecute
// runs programs ("calc", \\host\share\x.exe).
func TestReviewSchemelessLinkOpens(t *testing.T) {
	src := "[a](calc) [b](\\\\\\\\evil\\\\share\\\\x.exe)"
	tt, _ := rvView(&src, nil)
	tt.Click("a")
	tt.Click("b")
	if got := tt.OpenedURLs(); len(got) > 0 {
		t.Errorf("opened %q", got)
	}
}

// A heading's color, zero, takes the window theme's text color rather
// than Theme.Text, as HeadingStyle.Color documents.
func TestReviewHeadingColorFollowsText(t *testing.T) {
	def := DefaultTheme(ui.LightTheme())
	th := resolve(&Theme{Text: ui.RGB(200, 0, 0)}, def)
	if th.Headings[0].Color != th.Text {
		t.Errorf("heading color %v, text %v", th.Headings[0].Color, th.Text)
	}
}

// Part of a code span: its start maps past the span.
func TestReviewCopyMarkdownInsideCodeSpan(t *testing.T) {
	src := "x `hello world` y"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	text := rvParaText(st, 1)
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, text, 8, 13)}) // "world"
	if got != "`world`" {
		t.Errorf("copied %q from %q, want %q", got, text, "`world`")
	}
}

// A backslash escape makes its text inexact.
func TestReviewCopyMarkdownEscape(t *testing.T) {
	src := "1\\. first thing *here*"
	_, get := rvView(&src, nil)
	st := get()
	k := rvElemOf(t, st, 1)
	text := rvParaText(st, 1)
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, text, 3, 8)}) // "first"
	if got != "first" {
		t.Errorf("copied %q from %q, want %q", got, text, "first")
	}
}

// One cell selected whole copies its row.
func TestReviewCopyMarkdownTableCell(t *testing.T) {
	src := "| a | b |\n|---|---|\n| 1 | 2 |\n"
	_, get := rvView(&src, nil)
	st := get()
	var cell int
	for i := 1; i < st.doc.Len(); i++ {
		if st.doc.Block(i).Kind == parser.TableCell && st.doc.Source()[st.doc.Block(i).Start:st.doc.Block(i).End] == "1" {
			cell = i
		}
	}
	k := rvElemOf(t, st, cell)
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, "1", 0, 1)})
	if got != "1" {
		t.Errorf("copied %q, want %q", got, "1")
	}
}

// A code line with a tab partly taken by the list item's indentation.
func TestReviewCopyMarkdownCodeTab(t *testing.T) {
	src := "- foo\n\n\t\tbar\n"
	_, get := rvView(&src, nil)
	st := get()
	var code int
	for i := 1; i < st.doc.Len(); i++ {
		if st.doc.Block(i).Kind == parser.CodeBlock {
			code = i
		}
	}
	line, _ := st.doc.Line(code, 0)
	k := rvElemOf(t, st, code)
	n := len([]rune(line))
	got := st.copyMarkdown([]ui.SelectedText{rvPart(st, k, line, n-3, n-1)}) // "ba"
	if got != "ba" {
		t.Errorf("line %q: copied %q, want %q", line, got, "ba")
	}
}
