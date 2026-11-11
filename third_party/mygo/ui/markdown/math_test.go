package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// mathRecorder makes inline math objects of every TeX but "bad", and
// notes the TeX and sizes asked for and the boxes drawn.
type mathRecorder struct {
	asked []string
	sizes []float32
	drawn map[string]ui.Rect
}

func (m *mathRecorder) options() *Options {
	return &Options{
		Extensions: parser.Default | parser.ExtDollarMath,
		InlineMath: func(tex string, size float32) (InlineObject, bool) {
			m.asked = append(m.asked, tex)
			m.sizes = append(m.sizes, size)
			if tex == "bad" {
				return InlineObject{}, false
			}
			return InlineObject{Width: 20, Height: 10, Depth: 6, Draw: func(p *ui.Painter, r ui.Rect) {
				if m.drawn == nil {
					m.drawn = map[string]ui.Rect{}
				}
				m.drawn[tex] = r
			}}, true
		},
	}
}

// TestInlineMath shows math within text as the objects InlineMath makes,
// in links and strong text too, and as TeX where it makes none.
func TestInlineMath(t *testing.T) {
	var m mathRecorder
	src := "Let $x_i$ be **big $y$**, [see $z$](https://go.dev) and $bad$ or \\(w\\).\n\n# Head $h$\n"
	tt, get := rvView(&src, m.options())
	st := get()
	if got, want := rvParaText(st, 1), "Let ￼ be big ￼, see ￼ and bad or ￼."; got != want {
		t.Errorf("text %q, want %q", got, want)
	}
	for _, tex := range []string{"x_i", "y", "z", "w", "h"} {
		r, ok := m.drawn[tex]
		if !ok || r.W != 20 || r.H != 16 {
			t.Errorf("%s drawn in %+v (%v)", tex, r, ok)
		}
	}
	if _, ok := m.drawn["bad"]; ok || !tt.HasText("bad") {
		t.Errorf("bad drawn, or not shown as TeX in %q", tt.Texts())
	}
	// The size of the text around: the heading's is larger.
	for k, tex := range m.asked {
		if want := st.th.FontSize; tex != "h" && m.sizes[k] != want || tex == "h" && m.sizes[k] <= want {
			t.Errorf("%s asked at %v DIPs, text of %v", tex, m.sizes[k], want)
		}
	}
	// Each on the baseline of its line: x_i's box holds its descent.
	if x, y := m.drawn["x_i"], m.drawn["y"]; x.Y != y.Y {
		t.Errorf("x_i at %+v, y at %+v: not on one baseline", x, y)
	}
}

// TestInlineMathCopy copies math drawn as objects as its source, with
// its delimiters, amid the text selected.
func TestInlineMathCopy(t *testing.T) {
	var m mathRecorder
	src := "Let $x_i$ be **big $y$** or \\(w\\) — café.\n"
	tt, get := rvView(&src, m.options())
	st := get()
	r, ok := tt.Find("Let \uFFFC be big \uFFFC or \uFFFC — café.")
	if !ok {
		t.Fatalf("no text in %q", tt.Texts())
	}
	tt.ClickAt(r.X+2, r.Y+r.H/2)
	tt.Command("selectAll")
	tt.Command("copy")
	if got, want := tt.Clipboard(), "Let $x_i$ be big $y$ or \\(w\\) — café."; got != want {
		t.Errorf("copied %q, want %q", got, want)
	}
	text := rvParaText(st, 1)
	k := rvElemOf(t, st, 1)
	for _, c := range []struct {
		from, to int
		text, md string
	}{
		{4, 7, "$x_i$ b", "$x_i$ b"},
		{2, 5, "t $x_i$", "t $x_i$"},
		{4, 5, "$x_i$", "$x_i$"},
		{9, 19, "big $y$ or \\(w\\)", "**big $y$** or \\(w\\)"},
	} {
		part := []ui.SelectedText{rvPart(st, k, text, c.from, c.to)}
		if got := st.copyText(part); got != c.text {
			t.Errorf("runes %d to %d copied %q, want %q", c.from, c.to, got, c.text)
		}
		if got := st.copyMarkdown(part); got != c.md {
			t.Errorf("runes %d to %d copied as Markdown %q, want %q", c.from, c.to, got, c.md)
		}
	}
}

// TestInlineMathStreaming asks for math a single dollar leaves open as
// it streams in, healed, and leaves money as text.
func TestInlineMathStreaming(t *testing.T) {
	var m mathRecorder
	o := m.options()
	o.Reveal = &Reveal{Unit: AtOnce, Fade: -1}
	src := "Costs $5, so $x_i"
	tt := ui.NewTester(func(c *ui.Context) { Stream(c, src, o) }, 600, 400)
	tt.Frame()
	if len(m.asked) == 0 || m.asked[len(m.asked)-1] != "x_i" || strings.Contains(strings.Join(m.asked, " "), "5") {
		t.Errorf("asked for %q", m.asked)
	}
	if !tt.HasText("Costs $5, so ￼") {
		t.Errorf("shows %q", tt.Texts())
	}
}

// TestInlineMathNil keeps math as TeX without InlineMath.
func TestInlineMathNil(t *testing.T) {
	src := "Let $x_i$ be.\n"
	o := &Options{Extensions: parser.Default | parser.ExtDollarMath}
	_, get := rvView(&src, o)
	if got := rvParaText(get(), 1); got != "Let x_i be." {
		t.Errorf("text %q", got)
	}
}
