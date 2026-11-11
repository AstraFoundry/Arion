package parser

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// checkInvariants checks Next, Parent and segs ranges.
func checkInvariants(t *testing.T, d *Document, ctx string) {
	t.Helper()
	n := len(d.blocks)
	if d.blocks[0].Next != n {
		t.Fatalf("%s: root.Next %d, len %d", ctx, d.blocks[0].Next, n)
	}
	for i := 1; i < n; i++ {
		b := &d.blocks[i]
		if b.Parent < 0 || b.Parent >= i {
			t.Fatalf("%s: block %d parent %d", ctx, i, b.Parent)
		}
		p := &d.blocks[b.Parent]
		if b.Next <= i || b.Next > p.Next {
			t.Fatalf("%s: block %d (%v) Next %d, parent %d Next %d", ctx, i, b.Kind, b.Next, b.Parent, p.Next)
		}
		if b.segA > b.segB || int(b.segB) > len(d.segs) {
			t.Fatalf("%s: block %d segs [%d,%d) of %d", ctx, i, b.segA, b.segB, len(d.segs))
		}
		for _, s := range d.segsOf(b) {
			if s.start > s.end || int(s.end) > len(d.src) {
				t.Fatalf("%s: block %d seg %v src %d", ctx, i, s, len(d.src))
			}
		}
	}
	for k, i := range d.notes {
		if i <= 0 || i >= n || d.blocks[i].Kind != FootnoteDefinition || normalizeLabel(d.blocks[i].info) != k {
			t.Fatalf("%s: notes[%q]=%d bad", ctx, k, i)
		}
	}
}

var zzExt = func() Extensions {
	if os.Getenv("ZZTABLES") != "" {
		return Default
	}
	return Default &^ ExtTables
}()

func zzSeed() uint64 {
	n, _ := strconv.ParseUint(os.Getenv("ZZSEED"), 10, 64)
	return n
}

// zzKnown filters sources hitting a bug already reported: a label defined
// twice.
func zzKnown(src string) bool {
	zzSeen = zzSeen || strings.Count(src, "[a]:") > 1 || strings.Count(src, "[b]:") > 1 || strings.Count(src, "[a]") > 1 && strings.Contains(src, "[a]:") || strings.Count(src, "[b]") > 1 && strings.Contains(src, "[b]:")
	return zzSeen
}

var zzSeen bool

// zzFuzzOnly skips the random differential tests and probes unless
// ZZFUZZ is set (ZZSEED, ZZTABLES and ZZSTRUCT vary them).
func zzFuzzOnly(t *testing.T) {
	if os.Getenv("ZZFUZZ") == "" {
		t.Skip("set ZZFUZZ=1")
	}
}

var zzStruct = os.Getenv("ZZSTRUCT") != ""

// zzDump describes blocks and their inlines with offsets.
func zzDump(d *Document) string {
	var sb strings.Builder
	for i := 1; i < d.Len(); i++ {
		b := d.Block(i)
		fmt.Fprintf(&sb, "%d %v p%d n%d [%d,%d) open=%v L%d %v%d%c t%v task%d f%v h%v %v %q\n", i, b.Kind, b.Parent, b.Next, b.Start, b.End, b.Open, b.Level, b.Ordered, b.Number, b.Marker, b.Tight, b.Task, b.Fenced, b.Header, b.Align, b.info)
		if b.Kind == CodeBlock || b.Kind == MathBlock || b.Kind == HTMLBlock {
			for k := 0; k < d.LineCount(i); k++ {
				l, o := d.Line(i, k)
				fmt.Fprintf(&sb, "   line %q @%d\n", l, o)
			}
		}
		for _, in := range d.Inlines(i) {
			fmt.Fprintf(&sb, "   %v %q %q [%d,%d) n%d\n", in.Kind, in.Text, in.Dest, in.Start, in.End, in.Next)
		}
	}
	return sb.String()
}

func zzStructFails(d *Document, ext Extensions) bool {
	f := New(ext)
	f.Set(d.Source())
	return zzDump(d) != zzDump(f)
}

var zzPieces = []string{
	"", "\n", "\n\n", "\r\n", "\r", "\t", "- ", "* ", "+ ", "> ", ">", "# ", "## ", "```", "~~~", "**", "*", "_", "[a]", "[a]: /u\n", "[b]: /v \"t\"\n",
	"|", "| a | b |\n", "|---|---|\n", "a | b\n", "--- | ---\n", "    ", "  ", "1. ", "2) ", "---\n", "===\n", "text", "word ", "`", "<div>", "</div>\n", "<!-- ", "-->", "<pre>",
	"$$", "$$\n", "\\[", "\\]", "\\[\n", "\\]\n", "[^1]", "[^1]: ", "[^x]: note\n", "    more\n", "é", "日本", "\u00a0", "[ ] ", "[x] ", "<http://a.b>", "www.x.com ", "$x$", "\\(y\\)",
}

func zzRandDoc(r *rand.Rand, n int) string {
	var sb strings.Builder
	for range n {
		sb.WriteString(zzPieces[r.IntN(len(zzPieces))])
	}
	return sb.String()
}

func zzCheck(t *testing.T, d *Document, ext Extensions, ctx string) {
	t.Helper()
	checkInvariants(t, d, ctx)
	if want := render(d.Source(), ext); d.HTML() != want && !zzKnown(d.Source()) {
		t.Fatalf("%s:\nsrc  %q\ngot  %q\nwant %q", ctx, d.Source(), d.HTML(), want)
	}
}

func zzAppendFails(chunks []string, stream bool, ext Extensions) (failed bool) {
	defer func() {
		if recover() != nil {
			failed = true
		}
	}()
	zzSeen = false
	d := New(ext)
	d.SetStreaming(stream)
	src := ""
	for _, c := range chunks {
		d.Append(c)
		src += c
		d.HTML()
		if !stream && d.HTML() != render(src, ext) && !zzKnown(src) {
			return true
		}
	}
	d.SetStreaming(false)
	return (d.HTML() != render(src, ext) || zzStruct && zzStructFails(d, ext)) && !zzKnown(src)
}

func zzMinChunks(chunks []string, stream bool, ext Extensions) []string {
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(chunks); i++ {
			ns := append(append([]string(nil), chunks[:i]...), chunks[i+1:]...)
			if zzAppendFails(ns, stream, ext) {
				chunks, changed, i = ns, true, i-1
				continue
			}
			if i+1 < len(chunks) {
				ns = append(append(append([]string(nil), chunks[:i]...), chunks[i]+chunks[i+1]), chunks[i+2:]...)
				if zzAppendFails(ns, stream, ext) {
					chunks, changed, i = ns, true, i-1
					continue
				}
			}
		}
		for i := range chunks {
			for k := 0; k < len(chunks[i]); k++ {
				ns := append([]string(nil), chunks...)
				ns[i] = chunks[i][:k] + chunks[i][k+1:]
				if ns[i] != "" && utf8.ValidString(ns[i]) && zzAppendFails(ns, stream, ext) {
					chunks, changed, k = ns, true, k-1
				}
			}
		}
	}
	return chunks
}

func zzStreamFails(chunks []string, ext Extensions) (failed bool) {
	defer func() {
		if recover() != nil {
			failed = true
		}
	}()
	zzSeen = false
	d := New(ext)
	d.SetStreaming(true)
	src := ""
	for _, c := range chunks {
		d.Append(c)
		src += c
		f := New(ext)
		f.SetStreaming(true)
		f.Set(src)
		if d.HTML() != f.HTML() && !zzKnown(src) {
			return true
		}
	}
	return false
}

func TestParserReviewStreamMatchesFreshStream(t *testing.T) {
	zzFuzzOnly(t)
	r := rand.New(rand.NewPCG(51+zzSeed(), 52))
	for iter := range 20000 {
		ext := zzExt
		src := zzRandDoc(r, 1+r.IntN(14))
		var chunks []string
		for i := 0; i < len(src); {
			k := min(1+r.IntN(6), len(src)-i)
			chunks = append(chunks, src[i:i+k])
			i += k
		}
		if zzStreamFails(chunks, ext) {
			// Minimize.
			for changed := true; changed; {
				changed = false
				for i := 0; i < len(chunks); i++ {
					ns := append(append([]string(nil), chunks[:i]...), chunks[i+1:]...)
					if zzStreamFails(ns, ext) {
						chunks, changed, i = ns, true, i-1
					}
				}
				for i := range chunks {
					for k := 0; k < len(chunks[i]); k++ {
						ns := append([]string(nil), chunks...)
						ns[i] = chunks[i][:k] + chunks[i][k+1:]
						if ns[i] != "" && utf8.ValidString(ns[i]) && zzStreamFails(ns, ext) {
							chunks, changed, k = ns, true, k-1
						}
					}
				}
			}
			t.Fatalf("iter %d: streaming %q differs from a fresh streaming parse", iter, chunks)
		}
	}
}

func zzTimeAppend(src string, chunk int) time.Duration {
	d := New(Default)
	t0 := time.Now()
	for i := 0; i < len(src); i += chunk {
		d.Append(src[i:min(i+chunk, len(src))])
	}
	return time.Since(t0)
}

func TestParserReviewLineEndings(t *testing.T) {
	zzFuzzOnly(t)
	r := rand.New(rand.NewPCG(61+zzSeed(), 62))
	norm := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	}
	found := 0
	for iter := 0; iter < 30000 && found < 8; iter++ {
		var sb strings.Builder
		for range 1 + r.IntN(10) {
			p := zzPieces[r.IntN(len(zzPieces))]
			if strings.Contains(p, "\r") {
				continue
			}
			sb.WriteString(p)
		}
		s := sb.String()
		for _, eol := range []string{"\r\n", "\r"} {
			c := strings.ReplaceAll(s, "\n", eol)
			if want, got := render(s, Default), norm(render(c, Default)); got != want {
				// Minimize s.
				f := func(s string) bool {
					c := strings.ReplaceAll(s, "\n", eol)
					return render(s, Default) != norm(render(c, Default))
				}
				for changed := true; changed; {
					changed = false
					for k := 0; k < len(s); k++ {
						n := s[:k] + s[k+1:]
						if utf8.ValidString(n) && f(n) {
							s, changed, k = n, true, k-1
						}
					}
				}
				c = strings.ReplaceAll(s, "\n", eol)
				t.Errorf("eol %q: %q\n lf   %q\n eol  %q", eol, s, render(s, Default), render(c, Default))
				found++
				break
			}
		}
	}
}

func TestParserReviewRandomAppend(t *testing.T) {
	zzFuzzOnly(t)
	r := rand.New(rand.NewPCG(11+zzSeed(), 12))
	for iter := range 30000 {
		ext := zzExt
		if r.IntN(3) == 0 {
			ext = zzExt | ExtDollarMath
		}
		src := zzRandDoc(r, 1+r.IntN(14))
		stream := r.IntN(2) == 0
		var chunks []string
		for i := 0; i < len(src); {
			k := min(1+r.IntN(6), len(src)-i)
			chunks = append(chunks, src[i:i+k])
			i += k
		}
		if zzAppendFails(chunks, stream, ext) {
			t.Fatalf("iter %d stream=%v ext=%b minimal chunks %q", iter, stream, ext, zzMinChunks(chunks, stream, ext))
		}
	}
}

// zzEditFails reports whether Set(prev) then Edit(from, to, text) differs from a whole parse.
func zzEditFails(prev string, from, to int, text string, ext Extensions) bool {
	d := New(ext)
	d.Set(prev)
	d.HTML()
	failed := false
	func() {
		defer func() {
			if recover() != nil {
				failed = true
			}
		}()
		d.Edit(from, to, text)
		failed = d.HTML() != render(d.Source(), ext)
	}()
	return failed
}

func zzMinEdit(prev string, from, to int, text string, ext Extensions) (string, int, int) {
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(prev); i++ {
			if i >= from && i < to {
				continue
			}
			np := prev[:i] + prev[i+1:]
			nf, nt := from, to
			if i < from {
				nf--
				nt--
			}
			if utf8.ValidString(np) && zzEditFails(np, nf, nt, text, ext) {
				prev, from, to = np, nf, nt
				changed = true
				i--
			}
		}
		for to > from {
			np := prev[:from] + prev[from+1:]
			if utf8.ValidString(np) && zzEditFails(np, from, to-1, text, ext) {
				prev, to = np, to-1
				changed = true
			} else {
				break
			}
		}
	}
	return prev, from, to
}

// zzReplay sets each state in turn (streaming around it when stream), and
// reports whether any step differs from a whole parse.
func zzReplay(states []string, stream bool, ext Extensions) (failed bool) {
	defer func() {
		if recover() != nil {
			failed = true
		}
	}()
	zzSeen = false
	d := New(ext)
	for _, s := range states {
		if stream {
			d.SetStreaming(true)
		}
		d.Set(s)
		if stream {
			d.HTML()
			d.SetStreaming(false)
		}
		if (d.HTML() != render(s, ext) || zzStruct && zzStructFails(d, ext)) && !zzKnown(s) {
			return true
		}
	}
	return false
}

func zzMinStates(states []string, stream bool, ext Extensions) []string {
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(states); i++ {
			ns := append(append([]string(nil), states[:i]...), states[i+1:]...)
			if len(ns) > 0 && zzReplay(ns, stream, ext) {
				states = ns
				changed = true
				i--
			}
		}
		// Shrink each state by dropping bytes.
		for i := range states {
			for k := 0; k < len(states[i]); k++ {
				ns := append([]string(nil), states...)
				ns[i] = states[i][:k] + states[i][k+1:]
				if utf8.ValidString(ns[i]) && zzReplay(ns, stream, ext) {
					states = ns
					changed = true
					k--
				}
			}
		}
	}
	return states
}

func zzShow(t *testing.T, states []string, stream bool, ext Extensions) {
	d := New(ext)
	for _, s := range states {
		if stream {
			d.SetStreaming(true)
		}
		d.Set(s)
		if stream {
			t.Logf("streaming: %q", d.HTML())
			d.SetStreaming(false)
		}
		t.Logf("set %q\n got  %q\n want %q", s, d.HTML(), render(s, ext))
	}
}

// An edit shrinking a heading before a quiet block shifts a restart point
// that lay before the splice: restart goes negative and the next Append
// panics.
func TestParserReviewEditStaleRestartPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	d := New(0)
	d.Set("# a\nb")
	d.Set("c\n\nb") // Edit(0, 3, "c\n"): delta -1
	t.Logf("restart %d restartAt %d", d.restart, d.restartAt)
	d.Append("x")
	if want := render("c\n\nbx", 0); d.HTML() != want {
		t.Fatalf("got %q want %q", d.HTML(), want)
	}
}

func TestParserReviewEditStaleRestartWrong(t *testing.T) {
	d := New(0)
	d.Set("# a\nb")
	d.Set("# a\n\nb") // insert a blank line: delta +1, restart stays 0 -> 1
	t.Logf("restart %d restartAt %d", d.restart, d.restartAt)
	d.Append("x")
	if want := render("# a\n\nbx", 0); d.HTML() != want {
		t.Fatalf("got %q want %q", d.HTML(), want)
	}
}

// A setext heading whose paragraph began with a definition moves its
// Start past the definition, so the restart point noted from it skips
// the paragraph's first lines.
func TestParserReviewSetextAfterDefinitionAppend(t *testing.T) {
	d := New(0)
	d.Append("[a]: /u\n    b\n=\n")
	d.Append("x")
	if want := render("[a]: /u\n    b\n=\nx", 0); d.HTML() != want {
		t.Fatalf("got %q want %q", d.HTML(), want)
	}
}

// A table whose header was a later line of a paragraph starts at the
// header line, where parsing restarts: alone, "2) a|b" is a list item.
func TestParserReviewTableFromParagraphRestart(t *testing.T) {
	d := New(GFM)
	d.Append("e\n2) a|b\n-|-\n")
	d.Append("x")
	if want := render(d.Source(), GFM); d.HTML() != want {
		t.Errorf("append: got %q want %q", d.HTML(), want)
	}
	d = New(GFM)
	d.Set("e\n2) a|b\n-|-\n\nz\n")
	d.Edit(14, 15, "y")
	if want := render(d.Source(), GFM); d.HTML() != want {
		t.Errorf("edit: got %q want %q", d.HTML(), want)
	}
}

// A second definition of a label, in the blocks an edit keeps, is lost
// when the edit removes the first: refs keeps only the first, so the
// splice has no record of the second.
func TestParserReviewDuplicateDefinitionLost(t *testing.T) {
	d := New(0)
	d.Set("[b]: /1\n\n[b]\n\n[b]: /2\n")
	d.Edit(0, 1, "")
	if want := render(d.Source(), 0); d.HTML() != want {
		t.Fatalf("got %q want %q", d.HTML(), want)
	}
}

// An edit that changes only a continuation line's indentation reuses the
// paragraph's old inlines (same first offset, same text) with the old
// offsets.
func TestParserReviewReusedInlinesStaleOffsets(t *testing.T) {
	d := New(0)
	d.Set("a\nb\n\nc\n")
	d.HTML()
	d.Edit(2, 2, " ")
	f := New(0)
	f.Set(d.Source())
	if got, want := zzDump(d), zzDump(f); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// An HTML block of type 7 cannot interrupt a paragraph, a lazy
// continuation line included (commonmark.js's "maybe lazy", cmark's
// !maybe_lazy): the tag continues the item's paragraph.
func TestParserReviewHTMLBlock7InterruptsLazyParagraph(t *testing.T) {
	for src, want := range map[string]string{
		"- a\n<x y=\"\">\n": "<ul>\n<li>a\n<x y=\"\"></li>\n</ul>\n",
		"> a\n<x>\n":        "<blockquote>\n<p>a\n<x></p>\n</blockquote>\n",
	} {
		if got := render(src, 0); got != want {
			t.Errorf("%q: got %q want %q", src, got, want)
		}
	}
}

// A math block's opening line adds an empty line (or a second, empty
// line after the TeX on it) to its content.
func TestParserReviewMathBlockOpeningLine(t *testing.T) {
	for src, want := range map[string]string{
		"$$\ny\n$$\n":   "y\n",
		"$$ x\ny\n$$\n": "x\ny\n",
	} {
		d := Parse(src)
		if got := d.Lines(1); got != want {
			t.Errorf("%q: lines %q (%d), want %q", src, got, d.LineCount(1), want)
		}
	}
}

// Lines ending in \r alone make the parser look for \n to the end of the
// source on every line.
func TestParserReviewCROnlyQuadratic(t *testing.T) {
	checkLinearWork(t, "lines ending in CR", Default, false, repeat("a\r"))
}

// \[ display math never closes at \]: mathBracket compares with the raw
// string `\\]`, three bytes.
func TestParserReviewBracketMathNeverCloses(t *testing.T) {
	got := render("\\[\na\n\\]\n\ntext\n", Default)
	want := "<div class=\"math display\">a</div>\n<p>text</p>\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParserReviewRandomSetStates(t *testing.T) {
	zzFuzzOnly(t)
	r := rand.New(rand.NewPCG(31+zzSeed(), 32))
	for iter := range 6000 {
		ext := zzExt
		stream := r.IntN(2) == 0
		states := []string{zzRandDoc(r, 1+r.IntN(12))}
		for range 12 {
			cur := states[len(states)-1]
			from := r.IntN(len(cur) + 1)
			to := min(len(cur), from+r.IntN(10))
			if r.IntN(3) == 0 {
				to = from
			}
			next := cur[:from] + zzPieces[r.IntN(len(zzPieces))] + cur[to:]
			if utf8.ValidString(next) {
				states = append(states, next)
			}
		}
		if zzReplay(states, stream, ext) {
			m := zzMinStates(states, stream, ext)
			t.Fatalf("iter %d stream=%v minimal states %q", iter, stream, m)
		}
	}
}

type zzOp struct {
	from, to int
	text     string
}

func zzOpsFail(init string, ops []zzOp, stream bool, ext Extensions) (failed bool) {
	defer func() {
		if recover() != nil {
			failed = true
		}
	}()
	d := New(ext)
	d.Set(init)
	zzSeen = false
	zzKnown(init)
	cur := init
	for _, o := range ops {
		if o.to > len(cur) || o.from > o.to {
			return false
		}
		next := cur[:o.from] + o.text + cur[o.to:]
		if !utf8.ValidString(next) {
			return false
		}
		if stream {
			d.SetStreaming(true)
		}
		d.Edit(o.from, o.to, o.text)
		if stream {
			d.HTML()
			d.SetStreaming(false)
		}
		cur = next
		if (d.HTML() != render(cur, ext) || zzStruct && zzStructFails(d, ext)) && !zzKnown(cur) {
			return true
		}
	}
	return false
}

func zzMinOps(init string, ops []zzOp, stream bool, ext Extensions) (string, []zzOp) {
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(ops); i++ {
			ns := append(append([]zzOp(nil), ops[:i]...), ops[i+1:]...)
			if zzOpsFail(init, ns, stream, ext) {
				ops, changed, i = ns, true, i-1
			}
		}
		for k := 0; k < len(init); k++ {
			ni := init[:k] + init[k+1:]
			ns := append([]zzOp(nil), ops...)
			for x := range ns {
				// Shift ops naively: only valid for the first op.
				if x == 0 {
					if ns[0].from > k {
						ns[0].from--
					}
					if ns[0].to > k {
						ns[0].to--
					}
				}
			}
			if zzOpsFail(ni, ns, stream, ext) {
				init, ops, changed, k = ni, ns, true, k-1
			}
		}
		for x := range ops {
			for k := 0; k < len(ops[x].text); k++ {
				ns := append([]zzOp(nil), ops...)
				ns[x].text = ops[x].text[:k] + ops[x].text[k+1:]
				if zzOpsFail(init, ns, stream, ext) {
					ops, changed, k = ns, true, k-1
				}
			}
			if ops[x].to > ops[x].from {
				ns := append([]zzOp(nil), ops...)
				ns[x].to--
				if zzOpsFail(init, ns, stream, ext) {
					ops, changed = ns, true
				}
			}
		}
	}
	return init, ops
}

func TestParserReviewRandomEditOps(t *testing.T) {
	zzFuzzOnly(t)
	r := rand.New(rand.NewPCG(41+zzSeed(), 42))
	for iter := range 6000 {
		ext := zzExt
		stream := r.IntN(3) == 0
		init := zzRandDoc(r, 1+r.IntN(20))
		cur := init
		var ops []zzOp
		for range 20 {
			from := r.IntN(len(cur) + 1)
			to := min(len(cur), from+r.IntN(12))
			if r.IntN(3) == 0 {
				to = from
			}
			text := zzPieces[r.IntN(len(zzPieces))]
			next := cur[:from] + text + cur[to:]
			if !utf8.ValidString(next) {
				continue
			}
			ops = append(ops, zzOp{from, to, text})
			cur = next
		}
		if zzOpsFail(init, ops, stream, ext) {
			mi, mo := zzMinOps(init, ops, stream, ext)
			t.Fatalf("iter %d stream=%v init %q ops %+v", iter, stream, mi, mo)
		}
	}
}

func TestParserReviewRandomEdit(t *testing.T) {
	zzFuzzOnly(t)
	r := rand.New(rand.NewPCG(21+zzSeed(), 22))
	for iter := range 4000 {
		ext := zzExt
		src := zzRandDoc(r, 1+r.IntN(20))
		d := New(ext)
		d.Set(src)
		zzSeen = false
		zzKnown(src)
		stream := r.IntN(3) == 0
		for e := range 30 {
			cur := d.Source()
			from := r.IntN(len(cur) + 1)
			to := min(len(cur), from+r.IntN(12))
			if r.IntN(3) == 0 {
				to = from
			}
			text := zzPieces[r.IntN(len(zzPieces))]
			next := cur[:from] + text + cur[to:]
			if !utf8.ValidString(next) {
				continue
			}
			if stream {
				d.SetStreaming(true)
			}
			switch r.IntN(3) {
			case 0:
				d.Set(next)
			default:
				d.Edit(from, to, text)
			}
			if stream {
				d.HTML()
				d.SetStreaming(false)
			}
			if d.Source() != next {
				t.Fatalf("source mismatch")
			}
			zzCheck(t, d, ext, fmt.Sprintf("iter %d edit %d [%d,%d)->%q prev %q", iter, e, from, to, text, cur))
		}
	}
}
