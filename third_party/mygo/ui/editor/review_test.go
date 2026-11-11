package editor

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Regressions found by reviewing the editor.

// Clicking a completion other than the last chose it while the rows after
// it were still being built from the list it closed.
func TestReviewCompletionClickMiddle(t *testing.T) {
	ct := newEditorTest(t, "go", "|")
	ct.st.OnCompletion = func(r CompletionRequest) {
		ct.st.ShowCompletions(r.Offset-len(r.Prefix), []Completion{{Label: "alpha"}, {Label: "alpine"}, {Label: "alps"}})
	}
	ct.typ("al")
	if err := ct.tt.Click("alpha"); err != nil {
		t.Fatal(err)
	}
	ct.want("alpha|")
}

// The lines a highlighter reports final hold the tokens a fresh
// tokenization gives, whatever edits and passes came before: a pass that
// stopped short left lines past it unknown, which a later edit's
// converging must not confirm.
func TestReviewHighlighterConfirmed(t *testing.T) {
	lang := LanguageNamed("go")
	pieces := []string{"/*", "*/", "\n", "x := 1", "`", "\"", "y", "\n\n"}
	for seed := range uint64(300) {
		r := rand.New(rand.NewPCG(seed, 7))
		lines := make([]string, 80)
		for i := range lines {
			lines[i] = "x := 1"
		}
		buf := NewBuffer(strings.Join(lines, "\n"))
		buf.SetLanguage(lang)
		for step := range 12 {
			h := buf.highlighter()
			h.Highlight(r.IntN(buf.LineCount()), time.Second)
			off := r.IntN(buf.Len() + 1)
			end := min(buf.Len(), off+r.IntN(4))
			buf.Edit(Edit{Start: off, End: end, Text: pieces[r.IntN(len(pieces))]})
			h = buf.highlighter()
			h.Highlight(r.IntN(buf.LineCount()), time.Second)
			var state uint64
			for i := range buf.LineCount() {
				want, next := lang.Tokenizer.Tokenize(buf.Line(i), state, nil)
				state = next
				got, ok := h.Tokens(i, buf.Line(i), nil)
				if ok && !slices.Equal(got, want) {
					t.Fatalf("seed %d step %d: line %d final with %v, want %v", seed, step, i, got, want)
				}
			}
		}
	}
}

// Swapping a State's Buffer kept the old buffer's carets, past the new
// text's end, and its find matches.
func TestReviewSwapBuffer(t *testing.T) {
	long := strings.Repeat("line\n", 100)
	st := &State{Buffer: NewBuffer(long)}
	st.Buffer.SetLanguage(LanguageNamed("go"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 500, 300)
	tt.Frame()
	st.Select(len(long), len(long))
	st.Find("line")
	tt.Frame()
	nb := NewBuffer("short line")
	nb.SetLanguage(LanguageNamed("go"))
	st.Buffer = nb
	tt.Frame()
	if n, _ := st.FindMatches(); n != 1 {
		t.Fatalf("%d matches in the new buffer, want 1", n)
	}
	st.Select(0, 0)
	tt.Type("x")
	if got := nb.String(); got != "xshort line" {
		t.Fatalf("typed into the new buffer: %q", got)
	}
}

// Diagnostics given for a version are moved through the edits since, a
// new buffer's version 0 too, and keep End at or after Start.
func TestReviewDiagnosticsVersionZero(t *testing.T) {
	ct := newEditorTest(t, "go", "|abc def")
	v0 := ct.st.Buffer.Version()
	ct.typ("xx")
	ct.st.SetDiagnostics(v0, []Diagnostic{{Start: 4, End: 7}})
	d := ct.st.Diagnostics()[0]
	if got := ct.st.Buffer.Slice(d.Start, d.End); got != "def" {
		t.Fatalf("diagnostic covers %q, want def", got)
	}
}

// Several insertions at one offset: a left-biased offset stays before
// them all.
func TestReviewMapOffsetInsertions(t *testing.T) {
	b := NewBuffer("ab")
	b.Edit(Edit{1, 1, "X"}, Edit{1, 1, "Y"})
	cs, _, _ := b.changesSince(0)
	if got := mapOffset(1, cs[0], biasLeft); got != 1 {
		t.Fatalf("left of two insertions at 1: %d, want 1", got)
	}
	if got := mapOffset(1, cs[0], biasRight); got != 3 {
		t.Fatalf("right of two insertions at 1: %d, want 3", got)
	}
	ct := newEditorTest(t, "go", "[abc][de]")
	ct.typ("(")
	ct.want("([abc])([de])")
}

// A selection keeps what it holds as text is inserted at its edges, by
// another editor or a command that leaves the selections to the buffer.
func TestReviewSelectionBias(t *testing.T) {
	buf := NewBuffer("hello world")
	a, b := &State{Buffer: buf}, &State{Buffer: buf}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			View(c, a).Grow(1)
			View(c, b).Grow(1)
		})
	}, 800, 300)
	tt.Frame()
	b.Select(0, 5)
	a.Select(5, 5)
	tt.Frame()
	a.Insert("X")
	tt.Frame()
	if s := b.Selection(); s.Start() != 0 || s.End() != 5 {
		t.Fatalf("b's selection %+v grew over what a typed", s)
	}
	ct := newEditorTest(t, "go", "x := [foo]")
	ct.run(ToggleBlockComment)
	ct.want("x := /* [foo] */")
}

// Undo Selection takes back selections moved through the edits since.
func TestReviewUndoSelectionMapped(t *testing.T) {
	ct := newEditorTest(t, "", "abc def|")
	ct.st.Select(0, 0)
	ct.tt.Frame()
	ct.typ("XX")
	ct.key(ui.Cmd, ui.KeyU)
	ct.want("XXabc def|")
}

// Find in the selection: the selection found in moves with the edits,
// so that replacing stays within it.
func TestReviewFindScopeMoves(t *testing.T) {
	ct := newEditorTest(t, "", "[aaa aaa aaa] aaa")
	f := &ct.st.ed.find
	ct.run(FindInSelection)
	f.query, f.replace, f.stale = "aaa", "b", true
	ct.tt.Frame()
	for range 4 {
		f.replaceOne(&ct.st.ed)
		ct.tt.Frame()
	}
	if got := ct.st.Buffer.String(); got != "b b b aaa" {
		t.Fatalf("replaced in the selection: %q", got)
	}
}

// An editor scrolled down stays on the lines it shows as another editor
// of the buffer adds lines above them.
func TestReviewScrollAnchorMoves(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "line " + strings.Repeat("x", i%7)
	}
	buf := NewBuffer(strings.Join(lines, "\n"))
	a, b := &State{Buffer: buf}, &State{Buffer: buf}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			View(c, a).Grow(1).AutoFocus()
			View(c, b).Grow(1)
		})
	}, 800, 300)
	tt.Frame()
	b.ScrollTo(100)
	tt.Frame()
	top := func() string { return buf.Line(b.ed.view.heights.lineAt(b.ed.scrollY)) }
	before := top()
	a.Select(0, 0)
	tt.Frame()
	for range 5 {
		tt.Key(0, ui.KeyEnter)
	}
	tt.Frame()
	if got := top(); got != before {
		t.Fatalf("b's top line %q, was %q", got, before)
	}
}

// A language set after the first frame highlights the lines shown.
func TestReviewSetLanguageLate(t *testing.T) {
	ct := newEditorTest(t, "", "|func main() {}")
	ct.st.Buffer.SetLanguage(LanguageNamed("go"))
	ct.tt.Frame()
	ct.tt.Frame()
	if toks := ct.st.ed.view.line(0).toks; len(toks) == 0 || toks[0].Kind != TokenKeyword {
		t.Fatalf("tokens %v after setting the language", toks)
	}
}

// An editor not built while more edits than the log keeps were made
// follows them by the lines that changed, rather than keeping offsets of
// the old text.
func TestReviewFarBehindEditor(t *testing.T) {
	buf := NewBuffer("hello world\nsecond\n")
	a, b := &State{Buffer: buf}, &State{Buffer: buf}
	showB := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			View(c, a).Grow(1).AutoFocus()
			if showB {
				View(c, b).Grow(1)
			}
		})
	}, 800, 300)
	tt.Frame()
	b.Select(len("hello world\n"), len("hello world\nsecond"))
	tt.Frame()
	showB = false
	a.Select(0, 0)
	tt.Frame()
	for range 300 {
		tt.Type("x")
	}
	showB = true
	tt.Frame()
	if got := b.SelectedText(); got != "second" {
		t.Fatalf("b selects %q, want second", got)
	}
}

// Line commands with whole-line selections, as Cmd+L makes, and with
// adjacent carets.
func TestReviewLineCommands(t *testing.T) {
	ct := newEditorTest(t, "go", "one\n[two\n]three")
	ct.run(MoveLinesUp)
	ct.want("[two\n]one\nthree")
	ct = newEditorTest(t, "go", "[one\n]two\nthree")
	ct.run(MoveLinesDown)
	ct.want("two\n[one\n]three")
	ct = newEditorTest(t, "go", "one\n[two\n]three")
	ct.run(DuplicateLinesDown)
	ct.want("one\ntwo\n[two\n]three")
	ct = newEditorTest(t, "go", "one\n[two\n]three")
	ct.run(DuplicateLinesUp)
	ct.want("one\n[two\n]two\nthree")
	// Transposing at carets next to each other.
	ct = newEditorTest(t, "", "ab|c|d")
	ct.run(Transpose)
	if got := ct.st.Buffer.String(); len(got) != 4 {
		t.Fatalf("transposed into %q", got)
	}
	// Sorting keeps the lines selected.
	ct = newEditorTest(t, "", "[c\nb\na]")
	ct.run(SortLines)
	ct.want("[a\nb\nc]")
	// Select Line again extends to the next line.
	ct = newEditorTest(t, "", "o|ne\ntwo\nthree")
	ct.run(SelectLine)
	ct.run(SelectLine)
	ct.want("[one\ntwo\n]three")
	// Delete Line keeps the column.
	ct = newEditorTest(t, "", "one\ntw|o\nthree")
	ct.run(DeleteLine)
	ct.want("one\nth|ree")
}

// Tab goes to the next tab stop past hard tabs, and indents the line of
// a selection, as Zed's.
func TestReviewTab(t *testing.T) {
	ct := newEditorTest(t, "python", "\tx|")
	ct.key(0, ui.KeyTab)
	ct.want("\tx   |")
	ct = newEditorTest(t, "python", "x = [foo]")
	ct.key(0, ui.KeyTab)
	ct.want("    x = [foo]")
}

// Turning soft wrap on scrolls back to the lines' start.
func TestReviewWrapResetsScrollX(t *testing.T) {
	ct := newEditorTest(t, "", "|"+strings.Repeat("word ", 100))
	ct.run(MoveLineEnd)
	ct.tt.Frame()
	ct.run(ToggleSoftWrap)
	ct.tt.Frame()
	ct.tt.Frame()
	if x := ct.st.ed.scrollX; x != 0 {
		t.Fatalf("scrolled %v sideways with soft wrap on", x)
	}
}

// Folding keeps carets on the line a fold starts at.
func TestReviewFoldCarets(t *testing.T) {
	ct := newEditorTest(t, "go", "func a() {\n\tx()\n}\nfunc b() {\n\ty|()\n}\n")
	ct.run(FoldAll)
	if p := ct.st.CursorPos(); p.Line != 3 {
		t.Fatalf("caret at %+v after Fold All, want line 3", p)
	}
	ct = newEditorTest(t, "", "[a\nb\nc\nd]\ne")
	ct.run(FoldSelection)
	ct.tt.Frame()
	if _, ok := ct.st.ed.caretRect(ct.st.Cursor()); !ok {
		t.Fatal("the caret is in the lines Fold Selection hid")
	}
}

// Toggle Line Numbers gives the numbers room in the gutter.
func TestReviewToggleLineNumbers(t *testing.T) {
	st := &State{Buffer: NewBuffer("a\nb"), LineNumbers: LineNumbersOff, NoFolding: true}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	st.Run(ToggleLineNumbers)
	tt.Frame()
	if st.ed.gutterW == 0 {
		t.Fatal("no gutter for the numbers toggled on")
	}
}

// No hover rests where no text is: over the gutter, or past a line's end.
func TestReviewHoverOffText(t *testing.T) {
	st := &State{Buffer: NewBuffer("first line\nfoo := bar\n")}
	var asked []int
	st.OnHover = func(r HoverRequest) { asked = append(asked, r.Offset) }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 600, 300)
	tt.Frame()
	ed := &st.ed
	y := float32(ed.view.heights.geometry(1).rows-ed.scrollY) + ed.view.lineHeight/2
	rest := func() {
		tt.Advance(time.Hour)
	}
	tt.Move(ed.textX+40, y)
	tt.Move(5, y) // to the gutter
	rest()
	tt.Move(500, y) // past the line's end
	rest()
	if len(asked) != 0 {
		t.Fatalf("hover asked for %v off the text", asked)
	}
}

// Go to line's column counts characters.
func TestReviewGoToColumn(t *testing.T) {
	ct := newEditorTest(t, "", "|héllo")
	ed := &ct.st.ed
	ed.goTo.show(ed)
	ed.goTo.text = "1:3"
	(&GoToLineBar{ed: ed}).Go()
	ct.tt.Frame()
	ct.typ("x")
	if got := ct.st.Buffer.String(); got != "héxllo" {
		t.Fatalf("typed at column 3: %q", got)
	}
}

// Finding scrolls sideways to the match it selects.
func TestReviewFindScrollsSideways(t *testing.T) {
	ct := newEditorTest(t, "", "|"+strings.Repeat("x", 300)+"needle")
	ct.st.Find("needle")
	ct.tt.Frame()
	ct.tt.Frame()
	cr, _ := ct.st.ed.caretRect(ct.st.Cursor())
	if cr.X < ct.st.ed.scrollX || cr.X > ct.st.ed.scrollX+ct.st.ed.r.W {
		t.Fatalf("match at x %v, view from %v", cr.X, ct.st.ed.scrollX)
	}
}

// Lua's --[[ comments are block comments.
func TestReviewLuaBlockComment(t *testing.T) {
	tk := LanguageNamed("lua").Tokenizer
	var state uint64
	var toks []Token
	for _, line := range []string{"--[[ a", "b ]]"} {
		toks, state = tk.Tokenize(line, state, toks[:0])
	}
	if len(toks) != 1 || toks[0].Kind != TokenComment {
		t.Fatalf("the second line of a block comment: %v", toks)
	}
}

// With soft wrap, a press past a row's end and Down put the caret on that
// row, not at the start of the next.
func TestReviewWrapRowEnd(t *testing.T) {
	st := &State{Buffer: NewBuffer(strings.Repeat("x", 38) + "\nabcdefghij " + strings.Repeat("z", 45)), SoftWrap: true, WrapColumn: 40}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 800, 400)
	tt.Frame()
	st.Select(35, 35)
	tt.Frame()
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	ed := &st.ed
	if cr, _ := ed.caretRect(st.Cursor()); cr.Y != float32(ed.rowTop(1)) {
		t.Fatalf("Down to y %v, want the first row of line 1 at %v", cr.Y, ed.rowTop(1))
	}
	st2 := &State{Buffer: NewBuffer("abcdefghij " + strings.Repeat("z", 45)), SoftWrap: true, WrapColumn: 40}
	tt = ui.NewTester(func(c *ui.Context) { View(c, st2).Fill().AutoFocus() }, 800, 400)
	tt.Frame()
	ed = &st2.ed
	tt.ClickAt(ed.textX+ed.view.charWidth*30, float32(ed.rowTop(0))+ed.view.lineHeight/2)
	tt.Frame()
	if cr, _ := ed.caretRect(st2.Cursor()); cr.Y != float32(ed.rowTop(0)) {
		t.Fatalf("a press past row 0's end put the caret at y %v", cr.Y)
	}
}

// Moving sideways goes over a fold, rather than into it.
func TestReviewMoveOverFold(t *testing.T) {
	ct := newEditorTest(t, "go", "func a() {\n\tx()\n\ty()\n}\n")
	ct.st.Fold(0)
	ct.tt.Frame()
	ct.st.Select(10, 10)
	ct.tt.Frame()
	ct.key(0, ui.KeyRight)
	ct.tt.Frame()
	if len(ct.st.Folded()) != 1 || ct.st.CursorPos().Line != 3 {
		t.Fatalf("folds %v, caret %+v", ct.st.Folded(), ct.st.CursorPos())
	}
	ct.key(0, ui.KeyLeft)
	ct.tt.Frame()
	if len(ct.st.Folded()) != 1 || ct.st.Cursor() != 10 {
		t.Fatalf("back: folds %v, caret %d", ct.st.Folded(), ct.st.Cursor())
	}
}

// A suggestion goes as the caret moves, and the caret shows before it.
func TestReviewSuggestion(t *testing.T) {
	ct := newEditorTest(t, "", "abc|")
	ct.st.SetSuggestion(3, "defgh")
	ct.tt.Frame()
	ed := &ct.st.ed
	if cr, _ := ed.caretRect(3); cr.X > 3*ed.view.charWidth+1 {
		t.Fatalf("caret at x %v, after the suggestion", cr.X)
	}
	ct.key(0, ui.KeyLeft)
	if ed.suggestion != "" || len(ed.view.inlays) != 0 {
		t.Fatal("the suggestion stayed as the caret moved")
	}
}

// Completing at several carets replaces the word typed only where it is.
func TestReviewCompletionCarets(t *testing.T) {
	ct := newEditorTest(t, "", "ab x|\nfoo|")
	ct.st.OnCompletion = func(r CompletionRequest) {
		ct.st.ShowCompletions(r.Offset-len(r.Prefix), []Completion{{Label: "fooBar"}})
	}
	ct.tt.Frame()
	ct.st.Run(ShowCompletions)
	ct.tt.Frame()
	ct.key(0, ui.KeyEnter)
	if got := ct.st.Buffer.String(); got != "ab xfooBar\nfooBar" {
		t.Fatalf("completed into %q", got)
	}
}

// A regular expression's replacement expands its groups as found in the
// text, \B and all.
func TestReviewRegexReplace(t *testing.T) {
	ct := newEditorTest(t, "", "|fooBar bazQux")
	f := &ct.st.ed.find
	f.open, f.regex, f.caseSens, f.query, f.replace, f.stale = true, true, true, `\B([A-Z])`, "_$1", true
	f.replaceAll(&ct.st.ed)
	if got := ct.st.Buffer.String(); got != "foo_Bar baz_Qux" {
		t.Fatalf("replaced into %q", got)
	}
}

// After edits, the matches found by searching only the lines changed are
// those of a search of the whole text.
func TestReviewIncrementalFind(t *testing.T) {
	pieces := []string{"cat", "Cat", " ", "\n", "dog", "ca", "t", "é", "catalog"}
	for seed := range uint64(150) {
		r := rand.New(rand.NewPCG(seed, 11))
		ct := newEditorTest(t, "", "|cat dog\ncat cat\nnothing\ncatalog Cat\n")
		f := &ct.st.ed.find
		ct.st.Find("cat")
		f.caseSens, f.wholeWord = r.IntN(2) == 0, r.IntN(2) == 0
		if r.IntN(3) == 0 {
			f.query = "é"
		}
		ct.tt.Frame()
		for step := range 15 {
			b := ct.st.Buffer
			off := r.IntN(b.Len() + 1)
			end := min(b.Len(), off+r.IntN(5))
			b.Edit(Edit{Start: off, End: end, Text: pieces[r.IntN(len(pieces))]})
			ct.tt.Frame()
			got := slices.Clone(f.matches)
			f.stale = true
			f.search(&ct.st.ed)
			if !slices.Equal(got, f.matches) {
				t.Fatalf("seed %d step %d: %q: found %v, a full search %v", seed, step, b.String(), got, f.matches)
			}
		}
	}
}

// Lines copied whole paste as lines, from several carets at one, and in
// another editor.
func TestReviewWholeLinePaste(t *testing.T) {
	ct := newEditorTest(t, "", "a|\nb|\nc")
	ct.key(ui.Cmd, ui.KeyC)
	ct.st.Select(5, 5)
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("a\nb\na\nb\nc|")
	a := newEditorTest(t, "", "o|ne\ntwo")
	a.key(ui.Cmd, ui.KeyC)
	b := newEditorTest(t, "", "abc\nd|ef")
	b.tt.SetClipboard(a.tt.Clipboard())
	b.key(ui.Cmd, ui.KeyV)
	b.want("abc\none\nd|ef")
}

// Backspace takes a pair together only when the editor closed it.
func TestReviewBackspacePair(t *testing.T) {
	ct := newEditorTest(t, "go", "f(|)")
	ct.key(0, ui.KeyBackspace)
	ct.want("f|)")
}

// Select Next skips matches overlapping a selection.
func TestReviewSelectNextOverlap(t *testing.T) {
	ct := newEditorTest(t, "", "[aa]aa")
	ct.run(SelectNext)
	ct.run(SelectNext)
	ct.want("[aa][aa]")
}

// A flag is one character to the caret.
func TestReviewFlag(t *testing.T) {
	ct := newEditorTest(t, "", "|🇺🇸🇫🇷x")
	ct.run(MoveRight)
	ct.want("🇺🇸|🇫🇷x")
	ct.run(MoveRight)
	ct.want("🇺🇸🇫🇷|x")
	ct.key(0, ui.KeyBackspace)
	ct.want("🇺🇸|x")
}

// Text with \r\n line endings edits as lines ending in \n, and is
// written back with \r\n.
func TestReviewCRLF(t *testing.T) {
	ct := newEditorTest(t, "", "a|b\r\ncd")
	ct.run(MoveLineEnd)
	ct.typ("x")
	b := ct.st.Buffer
	if got := b.String(); got != "abx\ncd" {
		t.Fatalf("text %q", got)
	}
	var out strings.Builder
	b.WriteTo(&out)
	if out.String() != "abx\r\ncd" || b.LineEnding() != "\r\n" {
		t.Fatalf("written %q, line ending %q", out.String(), b.LineEnding())
	}
	ct = newEditorTest(t, "", "one\r\ntw|o\r\nthree")
	ct.key(ui.Cmd, ui.KeyC)
	ct.st.Select(1, 1)
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("two\no|ne\ntwo\nthree")
}

// Set replaces the whole text at once, edits from other goroutines
// before or after it.
func TestReviewSetAtomic(t *testing.T) {
	b := NewBuffer("")
	done := make(chan bool)
	go func() {
		for range 2000 {
			b.Edit(Edit{Start: 0, End: 0, Text: "y"})
		}
		close(done)
	}()
	for {
		select {
		case <-done:
			return
		default:
		}
		b.Set("x")
		if n := strings.Count(b.String(), "x"); n > 1 {
			t.Fatalf("%d x after Set", n)
		}
	}
}

// Typing while another goroutine edits the buffer keeps every keystroke,
// in order.
func TestReviewConcurrentTyping(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	b := ct.st.Buffer
	stop := make(chan bool)
	done := make(chan bool)
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			b.Edit(Edit{Start: 0, End: 0, Text: "z"})
			time.Sleep(20 * time.Microsecond)
		}
	}()
	letters := "abcdefghijklmnopqrstuvwxy"
	var typed strings.Builder
	for i := range 1500 {
		c := string(letters[i%len(letters)])
		typed.WriteString(c)
		ct.tt.Type(c)
	}
	close(stop)
	<-done
	got := strings.ReplaceAll(b.String(), "z", "")
	if got != typed.String() {
		t.Fatalf("typed %d letters, the text has %d of them: ...%q", typed.Len(), len(got), got[max(0, len(got)-40):])
	}
}

// An input method composing in a range it names, with several carets,
// composes there alone and commits there.
func TestReviewMarkedRangeCarets(t *testing.T) {
	ct := newEditorTest(t, "", "e|\ne|")
	ed := &ct.st.ed
	ed.SetMarkedText(&ui.TextInputRange{Start: 2, End: 3}, "ab", ui.TextInputRange{Start: 2, End: 2})
	if got := ct.st.Buffer.String(); got != "e\nab" {
		t.Fatalf("marked into %q", got)
	}
	ed.ReplaceText(nil, "X")
	if got := ct.st.Buffer.String(); got != "e\nX" {
		t.Fatalf("committed into %q", got)
	}
}

// Undo and redo of a step of several edits give a highlighter each
// edit's own texts, as an incremental parser needs.
func TestReviewUndoHighlighterTexts(t *testing.T) {
	var bad []string
	lang := &Language{Name: "Check", Highlighter: func(text Snapshot, invalidate func()) Highlighter {
		return &checkingHighlighter{bad: &bad}
	}}
	ct := newEditorTest(t, "", "|")
	ct.st.Buffer.SetLanguage(lang)
	ct.tt.Frame()
	ct.typ("abc")
	ct.tt.Frame()
	ct.key(ui.Cmd, ui.KeyZ)
	ct.tt.Frame()
	ct.key(ui.Cmd|ui.Shift, ui.KeyZ)
	ct.tt.Frame()
	if len(bad) > 0 {
		t.Fatalf("edits that do not agree: %v", bad)
	}
}

// checkingHighlighter checks that each edit's old text and changes give
// its new text.
type checkingHighlighter struct{ bad *[]string }

func (h *checkingHighlighter) Edit(old, new Snapshot, changes []Change) {
	text := old.String()
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		text = text[:c.OldStart] + new.Slice(c.NewStart, c.NewEnd) + text[c.OldEnd:]
	}
	if text != new.String() {
		*h.bad = append(*h.bad, old.String()+" -> "+text+", not "+new.String())
	}
}
func (h *checkingHighlighter) Highlight(int, time.Duration) bool { return true }
func (h *checkingHighlighter) Tokens(_ int, _ string, dst []Token) ([]Token, bool) {
	return dst, true
}
