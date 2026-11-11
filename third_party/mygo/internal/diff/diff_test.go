package diff

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// apply turns a into b with the hunks, checking that what they leave
// alone is the same on both sides.
func apply(t *testing.T, a, b []string, hunks []Hunk) {
	t.Helper()
	var out []string
	i, j := 0, 0
	for _, h := range hunks {
		if h.A0 < i || h.B0 < j || h.A0-i != h.B0-j || h.A1 < h.A0 || h.B1 < h.B0 {
			t.Fatalf("hunk %+v out of order after %d, %d", h, i, j)
		}
		for ; i < h.A0; i, j = i+1, j+1 {
			if a[i] != b[j] {
				t.Fatalf("unchanged line %d of a %q is %q in b", i, a[i], b[j])
			}
			out = append(out, a[i])
		}
		out = append(out, b[h.B0:h.B1]...)
		i, j = h.A1, h.B1
	}
	if len(a)-i != len(b)-j {
		t.Fatalf("tails differ: %d of a, %d of b", len(a)-i, len(b)-j)
	}
	out = append(out, a[i:]...)
	if !reflect.DeepEqual(out, b) && !(len(out) == 0 && len(b) == 0) {
		t.Fatalf("applying %v gives %q, want %q", hunks, out, b)
	}
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestLines(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want []Hunk
	}{
		{"", "", nil},
		{"a", "a", nil},
		{"", "a\nb", []Hunk{{0, 0, 0, 2}}},
		{"a\nb", "", []Hunk{{0, 2, 0, 0}}},
		{"a\nb\nc", "a\nx\nc", []Hunk{{1, 2, 1, 2}}},
		{"a\nb\nc", "a\nc", []Hunk{{1, 2, 1, 1}}},
		{"a\nc", "a\nb\nc", []Hunk{{1, 1, 1, 2}}},
		{"a\nb\nc\nd", "b\nc\nd\ne", []Hunk{{0, 1, 0, 0}, {4, 4, 3, 4}}},
	} {
		got := Lines(lines(c.a), lines(c.b))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Lines(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
		apply(t, lines(c.a), lines(c.b), got)
	}
}

// A function added between two others shows as the function and the
// blank line after it, not as the end of one function and the start of
// the next, as git's indent heuristic places it.
func TestLinesSlide(t *testing.T) {
	a := lines("func a() {\n\tx()\n}\n\nfunc c() {\n\tx()\n}")
	b := lines("func a() {\n\tx()\n}\n\nfunc b() {\n\tx()\n}\n\nfunc c() {\n\tx()\n}")
	got := Lines(a, b)
	want := []Hunk{{4, 4, 4, 8}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines = %v, want %v: %q", got, want, b[got[0].B0:got[0].B1])
	}
	apply(t, a, b, got)
}

// Histogram anchors on rare lines: moving a block keeps the unique lines
// matched rather than the braces that repeat.
func TestLinesHistogram(t *testing.T) {
	a := lines("one\n{\n}\ntwo\n{\n}\nthree")
	b := lines("two\n{\n}\none\n{\n}\nthree")
	got := Lines(a, b)
	apply(t, a, b, got)
	changed := 0
	for _, h := range got {
		changed += h.A1 - h.A0 + h.B1 - h.B0
	}
	if changed > 6 {
		t.Fatalf("Lines = %v, %d lines changed", got, changed)
	}
}

func TestLinesRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	words := []string{"a", "b", "c", "{", "}", "", "x := 1", "return"}
	gen := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = words[r.IntN(len(words))]
		}
		return out
	}
	for range 2000 {
		a := gen(r.IntN(40))
		b := append([]string(nil), a...)
		// Edit some.
		for range r.IntN(6) {
			if len(b) == 0 || r.IntN(2) == 0 {
				at := r.IntN(len(b) + 1)
				b = append(b[:at], append(gen(r.IntN(4)+1), b[at:]...)...)
			} else {
				at := r.IntN(len(b))
				b = append(b[:at], b[min(len(b), at+r.IntN(3)+1):]...)
			}
		}
		if r.IntN(5) == 0 {
			b = gen(r.IntN(40))
		}
		apply(t, a, b, Lines(a, b))
		ia, ib := Intern(a, b)
		d := differ{a: ia, b: ib}
		d.myers(0, len(ia), 0, len(ib))
		apply(t, a, b, d.hunks())
	}
}

// Myers's cost limit keeps unrelated texts quick and the result valid.
func TestMyersCostLimit(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	gen := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = string(rune('a' + r.IntN(3)))
		}
		return out
	}
	a, b := gen(5000), gen(5000)
	apply(t, a, b, Lines(a, b))
}

func TestWords(t *testing.T) {
	a := "return foo(bar, baz)"
	b := "return foo(qux, baz)"
	ra, rb, same := Words(a, b)
	if !reflect.DeepEqual(ra, []Range{{11, 14}}) || !reflect.DeepEqual(rb, []Range{{11, 14}}) {
		t.Fatalf("Words = %v, %v", ra, rb)
	}
	if same < 0.8 {
		t.Fatalf("same = %v", same)
	}
	// Changed words across whitespace join.
	ra, rb, _ = Words("a b c d", "a x y d")
	if !reflect.DeepEqual(ra, []Range{{2, 5}}) || !reflect.DeepEqual(rb, []Range{{2, 5}}) {
		t.Fatalf("Words joined = %v, %v", ra, rb)
	}
	ra, rb, _ = Chars("colour", "color")
	if !reflect.DeepEqual(ra, []Range{{4, 5}}) || rb != nil {
		t.Fatalf("Chars = %v, %v", ra, rb)
	}
}

func BenchmarkLines(b *testing.B) {
	var sb strings.Builder
	for i := range 20000 {
		sb.WriteString("line ")
		sb.WriteString(strings.Repeat("x", i%40))
		sb.WriteString("\n")
	}
	a := lines(sb.String())
	c := append([]string(nil), a...)
	for i := 100; i < len(c); i += 997 {
		c[i] = "changed"
	}
	b.ResetTimer()
	for range b.N {
		Lines(a, c)
	}
}

// The rarity of a line is its record's count in the region, as git's,
// so that the last of repeated lines is not taken for a unique anchor.
func TestHistogramRarity(t *testing.T) {
	a := []string{"", "", "", "\tv14 := f13()", "\tv22 := f25()", "}", ""}
	b := []string{"\t\treturn err", "\treturn nil", "", "", "\tv22 := f25()", "}", ""}
	got, want := Lines(a, b), []Hunk{{0, 0, 0, 2}, {2, 4, 4, 4}} // git diff --histogram's
	if !slices.Equal(got, want) {
		t.Errorf("hunks %v, want git's %v", got, want)
	}
}
