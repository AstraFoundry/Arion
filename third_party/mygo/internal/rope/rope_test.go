package rope

import (
	"math/rand/v2"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

// sample is text of several chunks, with multi-byte runes, runes outside
// the BMP and empty lines.
func sample() string {
	var b strings.Builder
	for i := range 3000 {
		switch i % 5 {
		case 0:
			b.WriteString("func main() {\n")
		case 1:
			b.WriteString("\tfmt.Println(\"héllo, 世界\")\n")
		case 2:
			b.WriteString("\n")
		case 3:
			b.WriteString("// 😀 emoji\n")
		default:
			b.WriteString("}\n")
		}
	}
	b.WriteString("last")
	return b.String()
}

func TestTextMatchesString(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	s := sample()
	text := New(s)
	pieces := []string{"", "x", "\n", "é\n世", "😀", strings.Repeat("abc\n", 2000)}
	for step := range 400 {
		a := r.IntN(len(s) + 1)
		b := a + r.IntN(min(len(s)-a, 5000)+1)
		// Keep the edges on runes, as the editor does.
		for a > 0 && !utf8.RuneStart(s[a]) {
			a--
		}
		for b < len(s) && !utf8.RuneStart(s[b]) {
			b++
		}
		in := pieces[r.IntN(len(pieces))]
		old := text
		text = text.Replace(a, b, in)
		s = s[:a] + in + s[b:]
		if old.Same(text) {
			t.Fatalf("step %d: Replace returned the same tree", step)
		}
		if err := Check(text.Root()); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		if text.Len() != len(s) || text.String() != s {
			t.Fatalf("step %d: text differs from the string", step)
		}
		if step%40 == 0 {
			checkLines(t, text, s)
		}
	}
	checkLines(t, text, s)
}

func checkLines(t *testing.T, text Text, s string) {
	t.Helper()
	lines := strings.Split(s, "\n")
	if text.LineCount() != len(lines) {
		t.Fatalf("LineCount = %d, want %d", text.LineCount(), len(lines))
	}
	off := 0
	for i, l := range lines {
		if text.LineStart(i) != off || text.LineEnd(i) != off+len(l) || text.Line(i) != l {
			t.Fatalf("line %d: start %d end %d %q, want %d %d %q", i, text.LineStart(i), text.LineEnd(i), text.Line(i), off, off+len(l), l)
		}
		if text.LineOf(off) != i || text.LineOf(off+len(l)) != i {
			t.Fatalf("LineOf(%d) = %d, want %d", off, text.LineOf(off), i)
		}
		off += len(l) + 1
	}
}

func TestEachLine(t *testing.T) {
	testEachLine(t, sample())
	// Lines of up to 90 KB among them, for batches cut short by bytes.
	lines := strings.Split(sample(), "\n")
	for i := 0; i < len(lines); i += 37 {
		lines[i] += strings.Repeat("x", i*30)
	}
	testEachLine(t, strings.Join(lines, "\n"))
}

func testEachLine(t *testing.T, s string) {
	text := New(s)
	lines := strings.Split(s, "\n")
	starts := make([]int, len(lines))
	for i := 1; i < len(lines); i++ {
		starts[i] = starts[i-1] + len(lines[i-1]) + 1
	}
	for _, r := range [][2]int{{0, len(lines)}, {0, 1}, {255, 257}, {700, 1300}, {len(lines) - 3, len(lines) + 10}, {-5, 3}} {
		next := max(0, r[0])
		text.EachLine(r[0], r[1], func(i, start int, l string) bool {
			if i != next || start != starts[i] || l != lines[i] {
				t.Fatalf("EachLine(%d, %d): line %d at %d %q, want %d at %d %q", r[0], r[1], i, start, l, next, starts[next], lines[next])
			}
			next++
			return true
		})
		if want := min(r[1], len(lines)); next != want {
			t.Fatalf("EachLine(%d, %d) stopped at %d, want %d", r[0], r[1], next, want)
		}
	}
	for _, r := range [][2]int{{len(lines) - 1, 0}, {0, 0}, {600, 300}, {len(lines) + 10, len(lines) - 4}, {3, -5}} {
		next := min(r[0], len(lines)-1)
		text.EachLineBack(r[0], r[1], func(i, start int, l string) bool {
			if i != next || start != starts[i] || l != lines[i] {
				t.Fatalf("EachLineBack(%d, %d): line %d at %d %q, want %d at %d %q", r[0], r[1], i, start, l, next, starts[next], lines[next])
			}
			next--
			return true
		})
		if want := max(r[1], 0) - 1; next != want {
			t.Fatalf("EachLineBack(%d, %d) stopped at %d, want %d", r[0], r[1], next, want)
		}
	}
	// Returning false stops at once.
	n := 0
	text.EachLine(0, len(lines), func(i, _ int, _ string) bool { n++; return i < 300 })
	if n != 301 {
		t.Fatalf("EachLine called fn %d times after a stop, want 301", n)
	}
	n = 0
	text.EachLineBack(len(lines)-1, 0, func(i, _ int, _ string) bool { n++; return false })
	if n != 1 {
		t.Fatalf("EachLineBack called fn %d times after a stop, want 1", n)
	}
}

func TestUTF16(t *testing.T) {
	s := sample()
	text := New(s)
	units := 0
	for off, r := range s {
		if got := text.UTF16(off); got != units {
			t.Fatalf("UTF16(%d) = %d, want %d", off, got, units)
		}
		if got := text.FromUTF16(units); got != off {
			t.Fatalf("FromUTF16(%d) = %d, want %d", units, got, off)
		}
		if got, n := text.RuneAt(off); got != r || n != utf8.RuneLen(r) {
			t.Fatalf("RuneAt(%d) = %q %d, want %q", off, got, n, r)
		}
		units += len(utf16.Encode([]rune{r}))
	}
	if text.UTF16(len(s)) != units || text.FromUTF16(units) != len(s) {
		t.Fatalf("UTF16 of the end = %d, want %d", text.UTF16(len(s)), units)
	}
	if r, n := text.RuneBefore(len(s)); r != 't' || n != 1 {
		t.Fatalf("RuneBefore(end) = %q %d", r, n)
	}
}

func TestChunks(t *testing.T) {
	s := sample()
	text := New(s)
	for _, r := range [][2]int{{0, len(s)}, {10, 10}, {ChunkBytes - 3, ChunkBytes + 3}, {5000, 20000}} {
		var b strings.Builder
		text.Chunks(r[0], r[1], func(c string) bool { b.WriteString(c); return true })
		if b.String() != s[r[0]:r[1]] || text.Slice(r[0], r[1]) != s[r[0]:r[1]] {
			t.Fatalf("Chunks(%d, %d) differs from the string", r[0], r[1])
		}
	}
	var w strings.Builder
	if _, err := WriteTo(text.Root(), &w); err != nil || w.String() != s {
		t.Fatalf("WriteTo differs from the string: %v", err)
	}
}
