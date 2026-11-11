package editor

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// parserHighlighter stands for a parser's highlighter, as tree-sitter's
// would be: it follows the edits, works "in the background" for a
// while, and knows the structure of the text.
type parserHighlighter struct {
	text       Snapshot
	edits      int
	pending    int // calls to Highlight until it is ready
	invalidate func()
}

func (h *parserHighlighter) Edit(old, new Snapshot, changes []Change) {
	// The changes and the texts give back the new text.
	var b strings.Builder
	at := 0
	for _, c := range changes {
		b.WriteString(old.Slice(at, c.OldStart))
		b.WriteString(new.Slice(c.NewStart, c.NewEnd))
		at = c.OldEnd
	}
	b.WriteString(old.Slice(at, old.Len()))
	if b.String() != new.String() {
		panic("changes do not make the new text: " + b.String())
	}
	h.text = new
	h.edits++
	h.pending = 2
}

func (h *parserHighlighter) Highlight(last int, budget time.Duration) bool {
	if h.pending > 0 {
		h.pending--
		return false
	}
	return true
}

// Tokens makes "fn" a keyword.
func (h *parserHighlighter) Tokens(i int, text string, dst []Token) ([]Token, bool) {
	from := 0
	for {
		k := strings.Index(text[from:], "fn")
		if k < 0 {
			break
		}
		if from+k > 0 {
			dst = append(dst, Token{End: from + k, Kind: TokenText})
		}
		dst = append(dst, Token{End: from + k + 2, Kind: TokenKeyword})
		from += k + 2
	}
	if from < len(text) {
		dst = append(dst, Token{End: len(text), Kind: TokenText})
	}
	return dst, h.pending == 0
}

// The whole text is the node around anything.
func (h *parserHighlighter) Enclosing(start, end int) (Range, bool) {
	return Range{0, h.text.Len()}, true
}

// Line 0 folds to the end.
func (h *parserHighlighter) FoldEnd(line int) (int, bool) {
	if line == 0 {
		return h.text.LineCount() - 1, true
	}
	return 0, false
}

// The first and last bytes are brackets around everything.
func (h *parserHighlighter) EnclosingBrackets(start, end int) (Range, Range, bool) {
	return Range{0, 2}, Range{h.text.Len() - 3, h.text.Len()}, true
}

// Return indents two levels.
func (h *parserHighlighter) IndentAfter(offset int) (int, bool) { return 2, true }

func TestHighlighterAPI(t *testing.T) {
	var made *parserHighlighter
	lang := &Language{Name: "Parsed", Highlighter: func(text Snapshot, invalidate func()) Highlighter {
		made = &parserHighlighter{text: text, invalidate: invalidate}
		return made
	}}
	st := &State{Buffer: NewBuffer("fn a() do\n  body\nend"), Indent: IndentSpaces, TabSize: 2}
	st.Buffer.SetLanguage(lang)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, st).Fill().AutoFocus()
	}, 500, 300)
	tt.Frame()
	if made == nil {
		t.Fatal("no highlighter made")
	}
	if toks := st.ed.view.line(0).toks; len(toks) == 0 || toks[0] != (Token{End: 2, Kind: TokenKeyword}) {
		t.Fatalf("tokens %+v", toks)
	}
	// Edits reach it, and the frames go on while it works.
	st.Select(0, 0)
	tt.Type("x")
	if made.edits != 1 || made.pending != 0 {
		t.Fatalf("edits %d, pending %d", made.edits, made.pending)
	}
	if !st.ed.view.line(0).exact {
		t.Fatal("tokens of line 0 still a guess once the highlighter is done")
	}
	// Select Larger takes its node.
	st.Select(5, 5)
	st.Run(SelectLarger)
	if s := st.Selection(); s.Start() != 0 || s.End() != st.Buffer.Len() {
		t.Fatalf("selected %+v", s)
	}
	// Moving to the bracket takes its brackets.
	st.Select(5, 5)
	st.Run(MoveToBracket)
	if c := st.Cursor(); c != st.Buffer.Len()-3 {
		t.Fatalf("moved to %d", c)
	}
	// Folding takes its folds.
	st.Fold(0)
	tt.Frame()
	if f := st.Folded(); len(f) != 1 || f[0] != 0 {
		t.Fatalf("folds %v", f)
	}
	st.Run(UnfoldAll)
	// Return takes its indentation.
	st.Select(st.Buffer.Len(), st.Buffer.Len())
	st.Run(Newline)
	if got := st.Buffer.String(); !strings.HasSuffix(got, "end\n    ") {
		t.Fatalf("text after Return %q", got)
	}
}

// The tokenizer of a language is its default highlighter.
func TestTokenHighlighterDefault(t *testing.T) {
	b := NewBuffer("x := `a\nb`")
	b.SetLanguage(LanguageNamed("go"))
	h, ok := b.highlighter().(*tokenHighlighter)
	if !ok {
		t.Fatalf("highlighter %T", b.highlighter())
	}
	h.Highlight(1, time.Hour)
	toks, final := h.Tokens(1, b.Line(1), nil)
	if !final || len(toks) == 0 || toks[0].Kind != TokenString {
		t.Fatalf("line 1 tokens %+v, final %v", toks, final)
	}
	// A language without a tokenizer or highlighter leaves text plain.
	b.SetLanguage(&Language{Name: "Plain"})
	if b.highlighter() != nil {
		t.Fatal("plain text highlighted")
	}
}

// The find bar's buttons take clicks, not the editor around them.
func TestFindBarButtons(t *testing.T) {
	// Opening the bar finds the word at the caret, and selects its match.
	ct := newEditorTest(t, "", "|a a a")
	ct.key(ui.Cmd, ui.KeyF)
	ct.tt.Frame()
	ct.want("[a] a a")
	if err := ct.tt.Click("Next match"); err != nil {
		t.Fatal(err)
	}
	if _, cur := ct.st.FindMatches(); cur != 1 {
		t.Fatalf("current match %d after Next", cur)
	}
	if err := ct.tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	if ct.st.ed.find.open {
		t.Fatal("find bar open after Close")
	}
}

// snapshotHighlighter colors from the text it follows, as a highlighter
// that parses the whole text does, rather than from the line it is
// given; behind marks a line asked of it before it heard of its text.
type snapshotHighlighter struct {
	text   Snapshot
	behind []string
}

func (h *snapshotHighlighter) Edit(_, new Snapshot, _ []Change)  { h.text = new }
func (h *snapshotHighlighter) Highlight(int, time.Duration) bool { return true }
func (h *snapshotHighlighter) Tokens(i int, text string, dst []Token) ([]Token, bool) {
	if i >= h.text.LineCount() || h.text.Slice(h.text.LineStart(i), h.text.LineEnd(i)) != text {
		h.behind = append(h.behind, text)
	}
	return append(dst, Token{End: len(text), Kind: TokenKeyword}), true
}

// An edit reaches the highlighter before a line of its text is laid out,
// as when the selection is set before the next frame.
func TestHighlighterHearsEditsBeforeLayout(t *testing.T) {
	h := &snapshotHighlighter{}
	lang := &Language{Name: "Snapshot", Highlighter: func(text Snapshot, _ func()) Highlighter {
		h.text = text
		return h
	}}
	st := &State{Buffer: NewBuffer("")}
	st.Buffer.SetLanguage(lang)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, st).Fill().AutoFocus()
	}, 500, 300)
	tt.Frame()
	st.Buffer.Set("SELECT 1")
	st.Select(8, 8)
	tt.Frame()
	if len(h.behind) > 0 {
		t.Fatalf("asked for the tokens of %q before hearing of the edit", h.behind)
	}
}
