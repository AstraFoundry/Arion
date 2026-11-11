package richtext

import (
	"github.com/egoist/mygo/ui"
	"math/rand/v2"
	"strings"
	"testing"
)

// cell is a rune of a naive attributed text: its attributes and the style
// of the paragraph it came from.
type cell struct {
	r     rune
	a     Attributes
	style ParagraphStyle
}

// naive is an attributed text as a cell per rune, and a last one for the
// end of the text.
type naive []cell

func naiveOf(t Text) naive {
	var out naive
	for _, p := range t.Paragraphs() {
		_ = p
	}
	i := 0
	for start, p := range t.Paragraphs() {
		_ = start
		for _, r := range p.Runs {
			for _, c := range r.Text {
				out = append(out, cell{c, r.Attributes, p.Style})
				i++
			}
		}
		end := t.AttributesAt(i)
		r := '\n'
		if i == t.Len() {
			r = 0
		}
		out = append(out, cell{r, end, p.Style})
		i++
	}
	return out
}

// normalize gives every cell of a paragraph the style of its first, as
// Text keeps one style a paragraph.
func (n naive) normalize() naive {
	start := 0
	for i, c := range n {
		if c.r == '\n' || c.r == 0 {
			for k := start; k <= i; k++ {
				n[k].style = n[start].style
			}
			start = i + 1
		}
	}
	return n
}

func (n naive) String() string {
	var b strings.Builder
	for _, c := range n[:len(n)-1] {
		b.WriteRune(c.r)
	}
	return b.String()
}

func checkInvariants(t *testing.T, at Text) {
	t.Helper()
	s := at.String()
	lines := strings.Split(s, "\n")
	if at.ParagraphCount() != len(lines) || seqLen(at.seq()) != len(lines) {
		t.Fatalf("%d paragraphs, %d in the sequence, %d lines", at.ParagraphCount(), seqLen(at.seq()), len(lines))
	}
	seqEach(at.seq(), 0, func(i int, p *richPara) bool {
		sum := 0
		for k, r := range p.runs {
			if r.n <= 0 {
				t.Fatalf("paragraph %d: empty run", i)
			}
			if k > 0 && p.runs[k-1].a == r.a && r.a.Attachment == nil {
				t.Fatalf("paragraph %d: runs %d and %d not joined", i, k-1, k)
			}
			sum += r.n
		}
		if want := len([]rune(lines[i])); p.n != want || sum != want+1 {
			t.Fatalf("paragraph %d: n %d, runs %d, line %q", i, p.n, sum, lines[i])
		}
		return true
	})
}

func compareNaive(t *testing.T, at Text, want naive) {
	t.Helper()
	checkInvariants(t, at)
	got := naiveOf(at)
	if got.String() != want.String() {
		t.Fatalf("text %q, want %q", got.String(), want.String())
	}
	for i := range got {
		if got[i].a != want[i].a {
			t.Fatalf("rune %d of %q: attributes %+v, want %+v", i, got.String(), got[i].a, want[i].a)
		}
		if got[i].style != want[i].style {
			t.Fatalf("rune %d of %q: style %+v, want %+v", i, got.String(), got[i].style, want[i].style)
		}
	}
}

var testAttrs = []Attributes{{}, {Weight: 700}, {Italic: true}, {Link: "https://a"}, {Weight: 700, Italic: true}, {Color: ui.RGB(200, 0, 0)}}
var testStyles = []ParagraphStyle{{}, {Heading: 1}, {List: Bullets}, {List: Numbers, Indent: 1}, {Align: ui.Center}}

func randomText(rnd *rand.Rand) Text {
	var paras []Paragraph
	for range 1 + rnd.IntN(4) {
		p := Paragraph{Style: testStyles[rnd.IntN(len(testStyles))]}
		for range rnd.IntN(4) {
			words := []string{"a", "bc", "déf", "ghij", " ", "😀"}
			p.Runs = append(p.Runs, Run{Text: words[rnd.IntN(len(words))], Attributes: testAttrs[rnd.IntN(len(testAttrs))]})
		}
		paras = append(paras, p)
	}
	return TextOf(paras...)
}

func TestAttributedTextOf(t *testing.T) {
	at := TextOf(
		Paragraph{Style: ParagraphStyle{Heading: 1}, Runs: []Run{{Text: "Title"}}},
		Paragraph{Runs: []Run{{Text: "Hello "}, {Text: "bold", Attributes: Attributes{Weight: 700}}, {Text: " and"}, {Text: " more"}}},
		Paragraph{},
	)
	checkInvariants(t, at)
	if got := at.String(); got != "Title\nHello bold and more\n" {
		t.Fatalf("text %q", got)
	}
	if at.ParagraphStyleAt(2).Heading != 1 || at.ParagraphStyleAt(8).Heading != 0 {
		t.Fatal("paragraph styles")
	}
	var runs []string
	for _, r := range at.Runs(6, 25) {
		runs = append(runs, r.Text)
	}
	if got := strings.Join(runs, "|"); got != "Hello |bold| and more" {
		t.Fatalf("runs %q", got)
	}
	if !at.AttributesAt(12).Bold() || at.AttributesAt(16).Bold() {
		t.Fatal("attributes")
	}
	if (Text{}).Len() != 0 || (Text{}).ParagraphCount() != 1 {
		t.Fatal("the zero text")
	}
	if !TextOf(Paragraph{Runs: []Run{{Text: "a\nb"}}}).Equal(NewText("a\nb")) {
		t.Fatal("a newline in a run starts a paragraph")
	}
}

func TestAttributedTextJoinParagraphs(t *testing.T) {
	at := TextOf(
		Paragraph{Style: ParagraphStyle{Heading: 1}, Runs: []Run{{Text: "Title"}}},
		Paragraph{Runs: []Run{{Text: "body"}}},
	)
	// Deleting the newline after a heading keeps the heading.
	joined := at.Delete(5, 6)
	if joined.String() != "Titlebody" || joined.ParagraphStyleAt(0).Heading != 1 {
		t.Fatalf("%q %+v", joined.String(), joined.ParagraphStyleAt(0))
	}
	// Deleting the heading whole, newline and all, leaves the body.
	gone := at.Delete(0, 6)
	if gone.String() != "body" || gone.ParagraphStyleAt(0).Heading != 0 {
		t.Fatalf("%q %+v", gone.String(), gone.ParagraphStyleAt(0))
	}
	// Enter at the end of the heading: both paragraphs are headings, until
	// the editor makes the new one a body paragraph.
	split := at.ReplaceString(5, 5, "\n", Attributes{})
	if split.String() != "Title\n\nbody" || split.ParagraphStyleAt(6).Heading != 1 || split.ParagraphStyleAt(7).Heading != 0 {
		t.Fatalf("%q", split.String())
	}
	if at.String() != "Title\nbody" {
		t.Fatal("the text changed")
	}
}

func TestAttributedTextAttachments(t *testing.T) {
	img := &Attachment{Name: "photo", Width: 20, Height: 10}
	at := NewText("ab").ReplaceString(1, 1, "￼￼x", Attributes{Attachment: img})
	checkInvariants(t, at)
	if at.AttributesAt(1).Attachment != img || at.AttributesAt(2).Attachment != img || at.AttributesAt(3).Attachment != nil {
		t.Fatal("only object replacement characters hold the attachment")
	}
	f := at.Format(0, at.Len(), func(a *Attributes) { a.Weight = 700; a.Attachment = nil })
	if f.AttributesAt(1).Attachment != img || !f.AttributesAt(1).Bold() {
		t.Fatal("Format keeps attachments")
	}
}

func TestAttributedTextRandom(t *testing.T) {
	for seed := range uint64(300) {
		rnd := rand.New(rand.NewPCG(seed, 7))
		at := randomText(rnd)
		model := naiveOf(at)
		compareNaive(t, at, model)
		for step := range 30 {
			n := at.Len()
			a := rnd.IntN(n + 1)
			z := a + rnd.IntN(n-a+1)
			switch rnd.IntN(4) {
			case 0, 1:
				with := randomText(rnd)
				if rnd.IntN(3) == 0 {
					// A piece of the text itself, as a paste.
					b := rnd.IntN(n + 1)
					with = at.Slice(b, b+rnd.IntN(n-b+1))
				}
				wm := naiveOf(with)
				next := append(append(append(naive{}, model[:a]...), wm[:len(wm)-1]...), model[z:]...)
				at, model = at.Replace(a, z, with), next.normalize()
			case 2:
				attrs := testAttrs[rnd.IntN(len(testAttrs))]
				at = at.Format(a, z, func(x *Attributes) { *x = attrs })
				if z == n && a < z {
					z++ // to the end of the text, what typing gets there
				}
				for i := a; i < z; i++ {
					model[i].a = attrs
					model[i].a.Attachment = nil
				}
			case 3:
				style := testStyles[rnd.IntN(len(testStyles))]
				pa, pz := at.paragraphSpan(a, z)
				at = at.FormatParagraphs(a, z, func(p *ParagraphStyle) { *p = style })
				para := 0
				for i := range model {
					if para >= pa && para <= pz {
						model[i].style = style
					}
					if model[i].r == '\n' {
						para++
					}
				}
			}
			t.Run("", func(t *testing.T) {
				defer func() {
					if t.Failed() {
						t.Logf("seed %d step %d", seed, step)
					}
				}()
				compareNaive(t, at, model)
			})
			if t.Failed() {
				return
			}
		}
	}
}

func TestAttributedTextSliceReplace(t *testing.T) {
	for seed := range uint64(200) {
		rnd := rand.New(rand.NewPCG(seed, 9))
		at := randomText(rnd)
		n := at.Len()
		a := rnd.IntN(n + 1)
		z := a + rnd.IntN(n-a+1)
		// Putting a slice back where it came from changes nothing.
		if got := at.Replace(a, z, at.Slice(a, z)); got.String() != at.String() || !runsEqual(got, at) {
			t.Fatalf("seed %d: %q [%d, %d)", seed, at.String(), a, z)
		}
	}
}

func runsEqual(a, b Text) bool {
	ma, mb := naiveOf(a), naiveOf(b)
	if len(ma) != len(mb) {
		return false
	}
	for i := range ma {
		if ma[i].a != mb[i].a {
			return false
		}
	}
	return true
}
