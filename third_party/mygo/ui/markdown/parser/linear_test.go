package parser

import (
	"runtime"
	"strings"
	"testing"
)

// pathological are inputs that take quadratic time in parsers without
// cmark's guards, or took it in this one, made of n repeats.
var pathological = []struct {
	name string
	ext  Extensions
	src  func(n int) string
}{
	{"mismatched openers", Default, repeat("*a_ ")},
	{"emphasis openers", Default, repeat("*a ")},
	{"many emphasis", Default, repeat("**a** ")},
	{"underscores", Default, func(n int) string { return strings.Repeat("_a ", n) + strings.Repeat("a_ ", n) }},
	{"strikethrough openers", Default, repeat("~~a ")},
	{"unclosed links", Default, repeat("[a](<b")},
	{"unclosed titles", Default, repeat(`[a](b "`)},
	{"brackets", Default, repeat("[")},
	{"unclosed brackets", Default, func(n int) string { return strings.Repeat("[", n) + "a" + strings.Repeat("]", n) }},
	{"image brackets", Default, func(n int) string { return strings.Repeat("![", n) + strings.Repeat("]", n) }},
	{"backticks", Default, repeat("`a``")},
	{"backtick runs", Default, backtickRuns},
	{"escapes", Default, repeat(`\*`)},
	{"unclosed comments", Default, func(n int) string { return "a " + strings.Repeat("<!--", n) }},
	{"math spans", Default, func(n int) string { return "a " + strings.Repeat("$$a ", n) }},
	{"emails", Default, repeat("a@b.")},
	{"urls", Default, repeat("x://")},
	{"nested quotes", Default, func(n int) string { return strings.Repeat(">", n) + " a" }},
	{"nested lists", Default, func(n int) string { return strings.Repeat("* ", n) + "a" }},
	{"link definitions", Default, repeat("[a]: u\n")},
	{"tables", Default, func(n int) string { return "| a |\n|---|\n" + strings.Repeat("|\n", n) }},
	{"pipes", Default, func(n int) string { return strings.Repeat("| ", n) + "\n|" + strings.Repeat("-|", n) }},
	{"lines ending in CR", Default, repeat("a\r")},
	{"indented lines", Default, repeat("a\n ")},
	{"quoted lines", Default, repeat("> a\n")},
	{"openers then links", Default, func(n int) string { return strings.Repeat("[", n) + strings.Repeat("[a](b)", n) }},
	{"escaped pipes", Default, func(n int) string { return "| a |\n|---|\n| " + strings.Repeat(`\| *a* `, n) + "|" }},
	{"mentions", Default, repeat("@a ")},
	{"repository references", Default, repeat("aaaaaaaa/bbbbbbbb#")},
	{"words before references", Default, repeat("a#")},
	{"alerts", Default, repeat("> [!NOTE]\n")},
}

func repeat(unit string) func(int) string {
	return func(n int) string { return strings.Repeat(unit, n) }
}

// backtickRuns returns runs of backticks of every length up to n/40,
// none closed, as cmark's pathological tests.
func backtickRuns(n int) string {
	var b strings.Builder
	for k := 1; k <= n/40; k++ {
		b.WriteString("e")
		b.WriteString(strings.Repeat("`", k))
	}
	return b.String()
}

// work parses src, and the inlines of every block with inlines, and
// returns the steps it took (Document.steps) and the bytes it allocated,
// which grow quadratically where text is copied over and over, as steps
// do not show.
func work(src string, ext Extensions, inlines bool) (steps, bytes int) {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	d := New(ext)
	d.Set(src)
	if inlines {
		inlinesAll(d)
	}
	runtime.ReadMemStats(&after)
	return d.steps, int(after.TotalAlloc - before.TotalAlloc)
}

// checkLinear checks that parsing src(n), inlines included, takes work
// that grows linearly with the input: about the same a byte for src(4n)
// as for src(n), where a quadratic parser takes four times as much, and
// a few steps a byte. It counts what the parser does rather than timing
// it, so that a quadratic regression fails on every machine and a busy
// machine never fails.
func checkLinear(t *testing.T, name string, ext Extensions, src func(n int) string) {
	t.Helper()
	checkLinearWork(t, name, ext, true, src)
}

// checkLinearWork is checkLinear, the inlines parsed with inlines.
func checkLinearWork(t *testing.T, name string, ext Extensions, inlines bool, src func(n int) string) {
	t.Helper()
	// The steps of linear inputs are an affine function of the input's
	// length, under 5 a byte; allocations grow less evenly, as slices
	// double.
	const n, stepsPerByte = 5000, 8
	small, large := src(n), src(4*n)
	sa, ba := work(small, ext, inlines)
	sb, bb := work(large, ext, inlines)
	perByte := func(w int, src string) float64 { return float64(w) / float64(len(src)) }
	if perByte(sb, large) > 1.25*perByte(sa, small) || sb > stepsPerByte*len(large) {
		t.Errorf("%s: %d steps for %d bytes, %d for %d", name, sa, len(small), sb, len(large))
	}
	if perByte(bb, large) > 1.5*perByte(ba, small)+16 {
		t.Errorf("%s: %d bytes allocated for %d bytes, %d for %d", name, ba, len(small), bb, len(large))
	}
}

// TestLinearTime checks that inputs quadratic in parsers without cmark's
// guards take linear work.
func TestLinearTime(t *testing.T) {
	for _, c := range pathological {
		checkLinear(t, c.name, c.ext, c.src)
	}
}
