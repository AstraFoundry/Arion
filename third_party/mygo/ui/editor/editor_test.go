package editor

import (
	"bytes"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/internal/rope"
	"github.com/egoist/mygo/internal/text"
	"github.com/egoist/mygo/ui"
)

// editorTest is an editor in a tester, focused, with the text given; | in
// it marks the carets, and [ ] selections.
type editorTest struct {
	t  *testing.T
	st *State
	tt *ui.Tester
}

func newEditorTest(t *testing.T, lang, text string) *editorTest {
	t.Helper()
	plain, sels := parseCarets(text)
	st := &State{Buffer: NewBuffer(plain)}
	if lang != "" {
		st.Buffer.SetLanguage(LanguageNamed(lang))
	}
	ct := &editorTest{t: t, st: st}
	ct.tt = ui.NewTester(func(c *ui.Context) {
		View(c, st).Fill().AutoFocus()
	}, 600, 400)
	if len(sels) > 0 {
		st.SetSelections(sels...)
	}
	ct.tt.Frame()
	return ct
}

// parseCarets takes the carets and selections out of text.
func parseCarets(text string) (string, []Selection) {
	var b strings.Builder
	var sels []Selection
	open := -1
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '|':
			sels = append(sels, Selection{Anchor: b.Len(), Head: b.Len()})
		case '[':
			open = b.Len()
		case ']':
			sels = append(sels, Selection{Anchor: open, Head: b.Len()})
		default:
			b.WriteByte(text[i])
		}
	}
	return b.String(), sels
}

// show writes the text with its carets and selections, as parseCarets
// reads them.
func (ct *editorTest) show() string {
	text := ct.st.Buffer.String()
	sels := ct.st.Selections()
	var b strings.Builder
	for i := 0; i <= len(text); i++ {
		// A selection ending here first, then those starting.
		for _, s := range sels {
			if !s.Empty() && s.End() == i {
				b.WriteByte(']')
			}
		}
		for _, s := range sels {
			if s.Empty() && s.Head == i {
				b.WriteByte('|')
			} else if !s.Empty() && s.Start() == i {
				b.WriteByte('[')
			}
		}
		if i < len(text) {
			b.WriteByte(text[i])
		}
	}
	return b.String()
}

func (ct *editorTest) want(want string) {
	ct.t.Helper()
	if got := ct.show(); got != want {
		ct.t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func (ct *editorTest) key(mods ui.Modifiers, key ui.Key) { ct.tt.Key(mods, key) }
func (ct *editorTest) typ(s string) {
	for _, r := range s {
		ct.tt.Type(string(r))
	}
}
func (ct *editorTest) run(cmd Command) { ct.st.Run(cmd); ct.tt.Frame() }

func TestCodeEditorTyping(t *testing.T) {
	ct := newEditorTest(t, "go", "|")
	ct.typ("fmt.Println(")
	ct.want("fmt.Println(|)")
	ct.typ(`"hi`)
	ct.want(`fmt.Println("hi|")`)
	// Closers typed step over those typed.
	ct.typ(`")`)
	ct.want(`fmt.Println("hi")|`)
	ct.key(0, ui.KeyBackspace)
	ct.want(`fmt.Println("hi"|`)
	// Backspace takes an empty pair.
	ct = newEditorTest(t, "go", "x|")
	ct.typ("[")
	ct.want("x[|]")
	ct.key(0, ui.KeyBackspace)
	ct.want("x|")
	// A selection wraps in brackets.
	ct = newEditorTest(t, "go", "[abc]")
	ct.typ("(")
	ct.want("([abc])")
}

func TestCodeEditorNewline(t *testing.T) {
	ct := newEditorTest(t, "go", "func main() {|}")
	ct.key(0, ui.KeyEnter)
	ct.want("func main() {\n\t|\n}")
	ct.typ("x")
	ct.key(0, ui.KeyEnter)
	ct.want("func main() {\n\tx\n\t|\n}")
	// A line comment goes on.
	ct = newEditorTest(t, "go", "\t// a comment|")
	ct.key(0, ui.KeyEnter)
	ct.want("\t// a comment\n\t// |")
	// Python indents after a colon, with spaces.
	ct = newEditorTest(t, "python", "def f():|")
	ct.key(0, ui.KeyEnter)
	ct.want("def f():\n    |")
}

// TestCodeEditorOutdent checks that typing what closes a block outdents
// its line, as Zed's auto-indent on input does.
func TestCodeEditorOutdent(t *testing.T) {
	ct := newEditorTest(t, "go", "func main() {\n\tif x {\n\t\ty()\n\t\t|")
	ct.typ("}")
	ct.want("func main() {\n\tif x {\n\t\ty()\n\t}|")
	// Undo takes the bracket back with its outdent.
	ct.key(ui.Cmd, ui.KeyZ)
	ct.want("func main() {\n\tif x {\n\t\ty()\n\t\t|")
	// Not after other text, nor inside a string.
	ct = newEditorTest(t, "go", "func main() {\n\t\tx|")
	ct.typ("}")
	ct.want("func main() {\n\t\tx}|")
	ct = newEditorTest(t, "go", "f(`\n\t\t|")
	ct.typ("}")
	ct.want("f(`\n\t\t}|")
	// Each caret outdents its line.
	ct = newEditorTest(t, "go", "{\n\t{\n\t\t|\n\t\t|")
	ct.typ("}")
	ct.want("{\n\t{\n\t}|\n}|")
	// Words of the language's DecreaseIndent.
	ct = newEditorTest(t, "shell", "if x; then\n  echo\n  |")
	ct.typ("fi")
	ct.want("if x; then\n  echo\nfi|")
	ct = newEditorTest(t, "ruby", "if x\n  y\n  |")
	ct.typ("else")
	ct.want("if x\n  y\nelse|")
	ct.typ("\n  z\n  end")
	ct.want("if x\n  y\nelse\n  z\nend|")
}

// TestCodeEditorScrollbar checks that the scroll bar of a text shorter
// than the editor, which scrolls past its end, takes the pointer: an
// arrow over it, and dragging it scrolls rather than selects.
func TestCodeEditorScrollbar(t *testing.T) {
	ct := newEditorTest(t, "go", "|package main\n\nfunc main() {}\n")
	x := float32(600 - scrollbarWidth/2)
	ct.tt.Move(300, 5)
	if c := ct.tt.Cursor(); c != ui.CursorText {
		t.Fatalf("cursor over the text %v, want the I-beam", c)
	}
	ct.tt.Move(x, 5)
	if c := ct.tt.Cursor(); c != ui.CursorDefault {
		t.Fatalf("cursor over the scroll bar %v, want the arrow", c)
	}
	ct.tt.Press(x, 5)
	ct.tt.Move(x, 300)
	ct.tt.Release(x, 300)
	if ct.st.ed.scrollY <= 0 {
		t.Fatal("dragging the scroll bar did not scroll")
	}
	ct.want("|package main\n\nfunc main() {}\n")
	// The horizontal one shows for a line running under the vertical one.
	ct = newEditorTest(t, "", "|"+strings.Repeat("x", 200))
	y := float32(400 - scrollbarWidth/2)
	ct.tt.Move(300, y)
	if c := ct.tt.Cursor(); c != ui.CursorDefault {
		t.Fatalf("cursor over the horizontal scroll bar %v, want the arrow", c)
	}
	ct.tt.Press(300, y)
	ct.tt.Move(500, y)
	ct.tt.Release(500, y)
	if ct.st.ed.scrollX <= 0 {
		t.Fatal("dragging the horizontal scroll bar did not scroll")
	}
	ct.want("|" + strings.Repeat("x", 200))
}

// TestCodeEditorAutoscroll checks that a selection dragged past the
// text's right or left edge scrolls the editor that way.
func TestCodeEditorAutoscroll(t *testing.T) {
	ct := newEditorTest(t, "", "|"+strings.Repeat("x", 200))
	ct.tt.Press(300, 5)
	ct.tt.Move(650, 5)
	ct.tt.Advance(time.Second)
	right := ct.st.ed.scrollX
	if right <= 0 {
		t.Fatal("dragging past the right edge did not scroll")
	}
	if sel := ct.st.Selections()[0]; sel.Head <= sel.Anchor {
		t.Fatalf("selection %+v, want it to grow to the right", sel)
	}
	ct.tt.Move(5, 5)
	ct.tt.Advance(time.Second)
	ct.tt.Release(5, 5)
	if ct.st.ed.scrollX >= right {
		t.Fatal("dragging over the gutter did not scroll back")
	}
}

// TestCodeEditorScrollbarOnHover checks that the scroll bar's thumb
// shows while the pointer is over the editor, as a scroll container's
// does, and not otherwise, and over no track.
func TestCodeEditorScrollbarOnHover(t *testing.T) {
	ct := newEditorTest(t, "go", "|package main\n\nfunc main() {}\n")
	thumb := func() bool {
		img := ct.tt.Image()
		s := img.Bounds().Dx() / 600
		x, y := (600-scrollbarWidth/2)*s, 5*s
		return img.RGBAAt(x, y) != img.RGBAAt(x-scrollbarWidth*s, y)
	}
	ct.tt.Frame()
	if thumb() {
		t.Fatal("the thumb shows with the pointer away")
	}
	ct.tt.Move(300, 200)
	ct.tt.Frame()
	if !thumb() {
		t.Fatal("the thumb doesn't show with the pointer over the editor")
	}
	ct.tt.Move(-10, -10)
	ct.tt.Frame()
	if thumb() {
		t.Fatal("the thumb stayed as the pointer left")
	}
	// Over the bar, the thumb and no track: below the thumb, the bar is
	// the editor's background.
	x := float32(600 - scrollbarWidth/2)
	ct.tt.Move(x, 5)
	ct.tt.Frame()
	img := ct.tt.Image()
	s := img.Bounds().Dx() / 600
	if img.RGBAAt(int(x)*s, 390*s) != img.RGBAAt((int(x)-scrollbarWidth)*s, 390*s) {
		t.Error("the bar shows a track under its thumb")
	}
}

// TestCodeEditorKeepScrollbars checks that KeepScrollbars shows the
// thumb with the pointer away, as it shows on hover, and that
// NoScrollbars shows no bar, takes no press on it and leaves its room to
// the text, while the wheel still scrolls.
func TestCodeEditorKeepScrollbars(t *testing.T) {
	ct := newEditorTest(t, "go", "|package main\n\nfunc main() {}\n")
	bar := func() []uint32 {
		img := ct.tt.Image()
		s := img.Bounds().Dx() / 600
		var px []uint32
		for y := 0; y < 400*s; y++ {
			for x := (600 - scrollbarWidth) * s; x < 600*s; x++ {
				c := img.RGBAAt(x, y)
				px = append(px, uint32(c.R)<<24|uint32(c.G)<<16|uint32(c.B)<<8|uint32(c.A))
			}
		}
		return px
	}
	ct.tt.Move(-10, -10)
	ct.tt.Frame()
	none := bar()
	ct.st.KeepScrollbars = true
	ct.tt.Frame()
	away := bar()
	if slices.Equal(away, none) {
		t.Fatal("a kept thumb doesn't show with the pointer away")
	}
	ct.tt.Move(300, 200)
	ct.tt.Frame()
	if hovered := bar(); !slices.Equal(away, hovered) {
		t.Fatal("a kept thumb looks otherwise with the pointer away than on hover")
	}
	ct.st.KeepScrollbars = false
	ct.st.NoScrollbars = true
	ct.tt.Frame()
	if v, h := ct.st.ed.scrollbars(); v || h || ct.st.ed.trackWidth() != 0 {
		t.Fatalf("NoScrollbars: bars %v %v, room %v", v, h, ct.st.ed.trackWidth())
	}
	if !slices.Equal(bar(), none) {
		t.Fatal("NoScrollbars painted a bar on hover")
	}
	x := float32(600 - scrollbarWidth/2)
	ct.tt.Move(x, 5)
	if c := ct.tt.Cursor(); c != ui.CursorText {
		t.Fatalf("cursor where the bar was %v, want the I-beam", c)
	}
	ct.tt.Press(x, 5)
	ct.tt.Move(x, 300)
	ct.tt.Release(x, 300)
	if ct.st.ScrollY() != 0 {
		t.Fatal("dragging where the bar was scrolled")
	}
	ct.tt.Scroll(300, 200, 0, 20)
	ct.tt.Frame()
	if ct.st.ScrollY() != 20 {
		t.Fatalf("the wheel scrolled to %v without bars, want 20", ct.st.ScrollY())
	}
	if ct.st.ViewportHeight() != 400 {
		t.Fatalf("ViewportHeight %v, want 400", ct.st.ViewportHeight())
	}
}

func TestCodeEditorMultiCursor(t *testing.T) {
	ct := newEditorTest(t, "go", "foo := 1\nbar := foo\nfoo++")
	ct.st.Select(0, 0)
	// Select next occurrence: the word, then each next.
	ct.key(ui.Cmd, ui.KeyD)
	ct.want("[foo] := 1\nbar := foo\nfoo++")
	ct.key(ui.Cmd, ui.KeyD)
	ct.key(ui.Cmd, ui.KeyD)
	ct.want("[foo] := 1\nbar := [foo]\n[foo]++")
	ct.typ("x")
	ct.want("x| := 1\nbar := x|\nx|++")
	ct.key(0, ui.KeyEscape)
	ct.want("x := 1\nbar := x\nx|++")
	// Add carets below, and type at each.
	ct = newEditorTest(t, "", "|ab\ncd\nef")
	ct.key(ui.Cmd|ui.Alt, ui.KeyDown)
	ct.key(ui.Cmd|ui.Alt, ui.KeyDown)
	ct.typ("-")
	ct.want("-|ab\n-|cd\n-|ef")
	// Going back up takes the last added away.
	ct.key(ui.Cmd|ui.Alt, ui.KeyUp)
	ct.want("-|ab\n-|cd\n-ef")
	// Select all occurrences.
	ct = newEditorTest(t, "", "a b a c |a")
	ct.key(ui.Cmd|ui.Shift, ui.KeyL)
	ct.want("[a] b [a] c [a]")
}

func TestCodeEditorLines(t *testing.T) {
	ct := newEditorTest(t, "go", "one\ntw|o\nthree")
	ct.key(ui.Alt, ui.KeyUp)
	ct.want("tw|o\none\nthree")
	ct.key(ui.Alt, ui.KeyDown)
	ct.key(ui.Alt, ui.KeyDown)
	ct.want("one\nthree\ntw|o")
	ct.key(ui.Alt|ui.Shift, ui.KeyUp)
	ct.want("one\nthree\ntw|o\ntwo")
	// Deleting a line keeps the caret's column, as Zed's.
	ct.key(ui.Cmd|ui.Shift, ui.KeyK)
	ct.want("one\nthree\ntw|o")
	ct.key(ui.Ctrl, ui.KeyJ)
	ct.want("one\nthree\ntw|o")
	ct = newEditorTest(t, "go", "a|\n  b\nc")
	ct.key(ui.Ctrl, ui.KeyJ)
	ct.want("a |b\nc")
	// Toggle comments at the least indentation, and back.
	ct = newEditorTest(t, "go", "\tif x {\n\t\t[y()\n\t}]")
	ct.key(ui.Cmd, ui.KeySlash)
	if got := ct.st.Buffer.String(); got != "\tif x {\n\t// \ty()\n\t// }" {
		t.Fatalf("commented %q", got)
	}
	ct.key(ui.Cmd, ui.KeySlash)
	if got := ct.st.Buffer.String(); got != "\tif x {\n\t\ty()\n\t}" {
		t.Fatalf("uncommented %q", got)
	}
	// Indent and outdent.
	ct = newEditorTest(t, "python", "[a\nb]")
	ct.key(ui.Cmd, ui.KeyBracketRight)
	ct.want("[    a\n    b]")
	ct.key(ui.Cmd, ui.KeyBracketLeft)
	ct.want("[a\nb]")
}

// The caret blinks by the frames' time, and shows again as it moves.
func TestCodeEditorCaretBlinks(t *testing.T) {
	ct := newEditorTest(t, "", "hello|")
	tt := ct.tt
	ct.key(0, ui.KeyLeft)
	shown := bytes.Clone(tt.Image().Pix)
	tt.Advance(blinkInterval)
	if bytes.Equal(tt.Image().Pix, shown) {
		t.Fatal("the caret still shows a blink on")
	}
	ct.key(0, ui.KeyRight)
	ct.key(0, ui.KeyLeft)
	if !bytes.Equal(tt.Image().Pix, shown) {
		t.Error("the caret does not show as it moved")
	}
}

func TestCodeEditorUndo(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	ct.typ("abc")
	// Typing in a row is one step.
	ct.key(ui.Cmd, ui.KeyZ)
	ct.want("|")
	ct.key(ui.Cmd|ui.Shift, ui.KeyZ)
	ct.want("abc|")
	// A pause makes another.
	time.Sleep(groupTimeout + 50*time.Millisecond)
	ct.typ("d")
	ct.key(ui.Cmd, ui.KeyZ)
	ct.want("abc|")
	// Undo puts the selections back.
	ct = newEditorTest(t, "", "x [hello] y")
	ct.typ("b")
	ct.want("x b| y")
	ct.key(ui.Cmd, ui.KeyZ)
	ct.want("x [hello] y")
}

func TestCodeEditorClipboard(t *testing.T) {
	ct := newEditorTest(t, "", "one\ntw|o\nthree")
	// Copying a caret alone copies its line, which pastes above a line.
	ct.key(ui.Cmd, ui.KeyC)
	if got := ct.tt.Clipboard(); got != "two\n" {
		t.Fatalf("copied %q", got)
	}
	ct.st.Select(1, 1)
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("two\no|ne\ntwo\nthree")
	// Cut of several selections pastes back piece by piece.
	ct = newEditorTest(t, "", "[a] [b] [c]")
	ct.key(ui.Cmd, ui.KeyX)
	ct.want("| | |")
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("a| b| c|")
	// Pasting text from elsewhere.
	ct.tt.SetClipboard("zz")
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("azz| bzz| czz|")
	// Pasting what was just copied over the selection still copied puts
	// the caret after it, and pasting again pastes it again.
	ct = newEditorTest(t, "", "say [hello] there")
	ct.key(ui.Cmd, ui.KeyC)
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("say hello| there")
	ct.key(ui.Cmd, ui.KeyV)
	ct.want("say hellohello| there")
	// As does typing over a selection the character typed.
	ct = newEditorTest(t, "", "a[b]c")
	ct.typ("b")
	ct.want("ab|c")
}

func TestCodeEditorComposition(t *testing.T) {
	ct := newEditorTest(t, "", "a|b\nc|d")
	ct.tt.Compose("に", 1)
	ct.want("aに|b\ncに|d")
	ct.tt.Compose("にほ", 2)
	ct.want("aにほ|b\ncにほ|d")
	ct.tt.Type("日本")
	ct.want("a日本|b\nc日本|d")
	ct.key(ui.Cmd, ui.KeyZ)
	ct.want("a|b\nc|d")
}

func TestCodeEditorFind(t *testing.T) {
	ct := newEditorTest(t, "", "|cat Cat cat dog")
	ct.st.Find("cat")
	ct.tt.Frame()
	if n, cur := ct.st.FindMatches(); n != 3 || cur != 0 {
		t.Fatalf("matches %d, %d", n, cur)
	}
	ct.st.ed.find.caseSens = true
	ct.st.ed.find.stale = true
	if n, _ := ct.st.FindMatches(); n != 2 {
		t.Fatalf("case-sensitive matches %d", n)
	}
	ct.st.ed.find.replace = "cow"
	ct.st.ed.find.replaceAll(&ct.st.ed)
	if got := ct.st.Buffer.String(); got != "cow Cat cow dog" {
		t.Fatalf("replaced %q", got)
	}
	ct.st.ed.find.regex, ct.st.ed.find.query, ct.st.ed.find.replace, ct.st.ed.find.stale = true, `c(\w+)`, "<$1>", true
	ct.st.ed.find.replaceAll(&ct.st.ed)
	if got := ct.st.Buffer.String(); got != "<ow> Cat <ow> dog" {
		t.Fatalf("replaced with a regular expression %q", got)
	}
}

func TestCodeEditorFold(t *testing.T) {
	ct := newEditorTest(t, "go", "func a() {\n\tx()\n\ty()\n}\nfunc b() {}")
	h := &ct.st.ed.view.heights
	before := h.total()
	ct.st.Fold(0)
	ct.tt.Frame()
	if got := ct.st.Folded(); len(got) != 1 || got[0] != 0 {
		t.Fatalf("folded %v", got)
	}
	if h.total() != before-2*float64(ct.st.ed.view.lineHeight) {
		t.Fatalf("folded height %v of %v", h.total(), before)
	}
	// Moving down skips what the fold hides.
	ct.st.Select(0, 0)
	ct.key(0, ui.KeyDown)
	if p := ct.st.CursorPos(); p.Line != 3 {
		t.Fatalf("caret went to line %d", p.Line)
	}
	ct.run(UnfoldAll)
	if h.total() != before {
		t.Fatalf("unfolded height %v of %v", h.total(), before)
	}
}

// TestCodeEditorCJKWords moves and deletes by words in Chinese and
// Japanese: a character at a time, as they have no spaces (#668).
func TestCodeEditorCJKWords(t *testing.T) {
	ct := newEditorTest(t, "", "|Go语言 今日は")
	for _, want := range []string{"Go|语言 今日は", "Go语|言 今日は", "Go语言| 今日は", "Go语言 今|日は", "Go语言 今日|は", "Go语言 今日は|"} {
		ct.run(MoveWordRight)
		ct.want(want)
	}
	ct.run(MoveWordLeft)
	ct.want("Go语言 今日|は")
	ct.run(DeleteWordLeft)
	ct.want("Go语言 今|は")
	ct = newEditorTest(t, "", "我们去公园散步|")
	ct.run(DeleteWordLeft)
	ct.want("我们去公园散|")
	ct.run(MoveSubwordLeft)
	ct.want("我们去公园|散")
}

func TestCJKWordAt(t *testing.T) {
	r := rope.TextOf(rope.Build("say 我们，ok"))
	for _, c := range []struct{ off, a, z int }{
		{0, 0, 3}, {4, 4, 7}, {5, 4, 7}, {7, 7, 10}, {10, 7, 10}, {13, 13, 15},
	} {
		if a, z := wordAt(r, c.off, ""); a != c.a || z != c.z {
			t.Errorf("wordAt(%d) = %d, %d, want %d, %d", c.off, a, z, c.a, c.z)
		}
	}
}

func TestCodeEditorFoldImports(t *testing.T) {
	for _, tc := range []struct {
		lang, text string
		want       []fold
	}{
		// A block in brackets alone folds by them, as before.
		{"go", "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {}", []fold{{2, 4}}},
		// Several, to the closing line of the last.
		{"go", "package main\n\nimport \"fmt\"\nimport (\n\t\"os\"\n)\n\nfunc main() {}", []fold{{2, 5}}},
		// One import alone does not fold.
		{"go", "package main\n\nimport \"fmt\"\n\nfunc main() {}", nil},
		// A run of them, a long one in it, and blank lines between.
		{"typescript", "// x\nimport a from './a'\nimport {\n  b,\n  c,\n} from './b'\n\nimport './d'\n\nconst e = a", []fold{{1, 7}}},
		{"python", "\"\"\"Doc.\"\"\"\nimport os\nfrom x import (\n    a,\n)\nimport sys\n\nprint(os)", []fold{{1, 5}}},
		{"rust", "use std::io;\npub use crate::x;\n\nfn main() {}", []fold{{0, 1}}},
		{"c", "#include <stdio.h>\n#include \"x.h\"\n\nint main() {}", []fold{{0, 1}}},
		{"java", "package a;\n\nimport a.B;\nimport a.C;\n\nclass D {}", []fold{{2, 3}}},
		// Not the lines of a docstring.
		{"python", "\"\"\"\nfrom here on\nimport this\n\"\"\"\nimport os\nimport sys\n", []fold{{4, 5}}},
		{"go", "// import a\n// import b\npackage main\n", nil},
		// None in a language without imports.
		{"markdown", "import a\nimport b\n", nil},
	} {
		ct := newEditorTest(t, tc.lang, tc.text)
		ct.st.FoldImports()
		if got := ct.st.ed.folds; !slices.Equal(got, tc.want) {
			t.Errorf("%s: folds %v, want %v", tc.lang, got, tc.want)
		}
	}
	// The gutter marks the run's first line, not the others.
	ct := newEditorTest(t, "javascript", "import a from 'a'\nimport b from 'b'\nimport c from 'c'\n\nf(a, b, c)")
	if !ct.st.ed.foldable(0) || ct.st.ed.foldable(1) {
		t.Errorf("foldable %v %v, want the first line only", ct.st.ed.foldable(0), ct.st.ed.foldable(1))
	}
}

func TestCodeEditorWords(t *testing.T) {
	ct := newEditorTest(t, "", "|fooBar_baz qux")
	ct.run(MoveSubwordRight)
	ct.want("foo|Bar_baz qux")
	ct.run(MoveSubwordRight)
	ct.want("fooBar|_baz qux")
	ct.run(MoveWordRight)
	ct.want("fooBar_baz| qux")
	ct.run(MoveWordLeft)
	ct.want("|fooBar_baz qux")
	// Selecting larger: the word, the string, its quotes, the brackets.
	ct = newEditorTest(t, "go", `f(x, "a b|c")`)
	ct.run(SelectLarger)
	ct.want(`f(x, "a [bc]")`)
	ct.run(SelectLarger)
	ct.want(`f(x, "[a bc]")`)
	ct.run(SelectLarger)
	ct.want(`f(x, ["a bc"])`)
	ct.run(SelectLarger)
	ct.want(`f([x, "a bc"])`)
	ct.run(SelectSmaller)
	ct.want(`f(x, ["a bc"])`)
	// A selection of a pair grows to the pair around it, and a caret
	// after a pair selects that pair, as Zed does.
	ct = newEditorTest(t, "go", `g(f[(x)])`)
	ct.run(SelectLarger)
	ct.want(`g([f(x)])`)
	ct.run(SelectLarger)
	ct.want(`g[(f(x))]`)
	ct = newEditorTest(t, "go", `g(f(x)|)`)
	ct.run(SelectLarger)
	ct.want(`g(f[(x)])`)
	// Moving to the bracket goes to the same side of the other one.
	ct = newEditorTest(t, "go", `g(f(x)|)`)
	ct.run(MoveToBracket)
	ct.want(`g(f|(x))`)
	ct.run(MoveToBracket)
	ct.want(`g(f(x)|)`)
	ct = newEditorTest(t, "go", `g(f(|x))`)
	ct.run(MoveToBracket)
	ct.want(`g(f(x|))`)
	ct.run(MoveToBracket)
	ct.want(`g(f(|x))`)
	// Case conversions.
	ct = newEditorTest(t, "", "[fooBarBaz]")
	ct.run(SnakeCase)
	ct.want("[foo_bar_baz]")
	ct.run(KebabCase)
	ct.want("[foo-bar-baz]")
	ct.run(PascalCase)
	ct.want("[FooBarBaz]")
}

func TestCodeEditorBracketsFar(t *testing.T) {
	// 70 KB in 71 lines: past the 64 KB bound, within the 1000 lines.
	far := strings.Repeat(strings.Repeat("x", 1000)+"\n", 70)
	for _, text := range []string{"f(\n" + far + "|)", "f(|\n" + far + ")", "f(\n" + far + ")|"} {
		ct := newEditorTest(t, "go", text)
		at := ct.st.Selections()[0].Head
		ct.run(MoveToBracket)
		if got := ct.st.Selections()[0].Head; got != at {
			t.Errorf("caret at %d moved to %d", at, got)
		}
	}
}

// Two editors of one buffer follow each other's edits.
func TestCodeEditorSharedBuffer(t *testing.T) {
	buf := NewBuffer("hello world")
	a, b := &State{Buffer: buf}, &State{Buffer: buf}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			View(c, a).Grow(1).AutoFocus()
			View(c, b).Grow(1)
		})
	}, 800, 300)
	b.Select(6, 11)
	a.Select(0, 0)
	tt.Frame()
	tt.Type("¡")
	if got := buf.String(); got != "¡hello world" {
		t.Fatalf("text %q", got)
	}
	if s := b.Selection(); s.Start() != 8 || s.End() != 13 {
		t.Fatalf("other editor's selection %+v", s)
	}
}

// A double click selects the word under it and a triple click its line,
// on a backend that does not count clicks.
func TestCodeEditorMultiClick(t *testing.T) {
	ct := newEditorTest(t, "", "|select customers from orders\nx")
	ed := &ct.st.ed
	x, y := ed.textX+ed.view.charWidth*10, float32(ed.rowTop(0))+ed.view.lineHeight/2
	ct.tt.ClickAt(x, y)
	ct.tt.ClickAt(x, y)
	ct.tt.Frame()
	ct.want("select [customers] from orders\nx")
	ct.tt.ClickAt(x, y)
	ct.tt.Frame()
	ct.want("[select customers from orders\n]x")
}

// Soft wraps of long lines are measured a few lines a frame, as the
// budget allows, rather than all at once, and every frame measures some.
func TestCodeEditorWrapLongLines(t *testing.T) {
	ct := newEditorTest(t, "", strings.Repeat(strings.Repeat("word ", 20_000)+"\n", 10))
	ct.st.SoftWrap = true
	ct.tt.Frame()
	v := &ct.st.ed.view
	// Measure again, as after a resize, with a budget spent from the start.
	v.setWrap(true, v.wrapWidth-10)
	if v.measureWraps(0, -1) {
		t.Fatal("measured all the long lines at once")
	}
	for calls := 2; !v.measureWraps(0, -1); calls++ {
		if calls > v.text.LineCount() {
			t.Fatalf("lines not all measured after %d calls", calls)
		}
	}
}

// Tokens whose style is bold or italic are shaped in those faces, and
// carets and hits find the same places across the runs of faces, on one
// row and on a line wrapped into rows measured before they are shaped.
func TestCodeEditorTokenFaces(t *testing.T) {
	const line = "plain **strong** and *emph* téxt"
	ct := newEditorTest(t, "markdown", line)
	v := &ct.st.ed.view
	l := v.line(0)
	if len(l.faces) == 0 {
		t.Fatalf("no faces for %q: %+v", line, l.toks)
	}
	lay := v.piece(l, &l.rows[0].pieces[0])
	spans := text.DecodeSpans(lay.Params.Spans)
	face := func(rune int) (weight int, italic bool) {
		for _, sp := range spans {
			if rune < sp.End {
				return sp.Weight, sp.Italic
			}
		}
		return 0, false
	}
	runeOf := func(sub string) int { return utf8.RuneCountInString(line[:strings.Index(line, sub)]) }
	for _, c := range []struct {
		sub    string
		weight int
		italic bool
	}{{"plain", 0, false}, {"strong", 700, false}, {"and", 0, false}, {"emph", 0, true}, {"téxt", 0, false}} {
		if w, it := face(runeOf(c.sub)); w != c.weight || it != c.italic {
			t.Errorf("%q: weight %d, italic %v; want %d, %v", c.sub, w, it, c.weight, c.italic)
		}
	}
	plain := newEditorTest(t, "", line)
	pv := &plain.st.ed.view
	pl := pv.line(0)
	if s := pv.piece(pl, &pl.rows[0].pieces[0]).Params.Spans; len(pl.faces) != 0 || s != "" {
		t.Errorf("plain text has faces %+v, spans %q", pl.faces, s)
	}

	check := func(v *view, l *lineLayout) {
		t.Helper()
		for ri := range l.rows {
			r := &l.rows[ri]
			prev := float32(-1)
			for d := r.from; d < r.to; d++ {
				if !utf8.RuneStart(l.disp[d]) {
					continue
				}
				x := v.xOf(l, r, d)
				if x <= prev {
					t.Fatalf("row %d: x of %d is %v, after %v", ri, d, x, prev)
				}
				prev = x
				if got := v.dispAt(l, r, x+0.5); got != d {
					t.Fatalf("row %d: hit at x %v of %d is %d", ri, x, d, got)
				}
			}
			if w := v.xOf(l, r, r.to) - r.indent; math.Abs(float64(w-r.width)) > 1 {
				t.Errorf("row %d: shaped %v wide, measured %v", ri, w, r.width)
			}
		}
	}
	check(v, l)

	ct.st.SoftWrap = true
	ct.tt.Frame()
	v.setWrap(true, v.runsWidth(line, nil, 0, len(line))/2)
	l = v.line(0)
	if len(l.rows) < 2 {
		t.Fatalf("%d rows, want the line wrapped", len(l.rows))
	}
	check(v, l)
}

// An editor as high as its lines, in a scroll view, shows its caret as it
// takes the focus, not its first line.
func TestFitContentFocusRevealsCaret(t *testing.T) {
	st := &State{Buffer: NewBuffer(strings.Repeat("line\n", 300)), FitContent: true}
	var scroll ui.ScrollState
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().TrackScroll(&scroll).Children(func() {
			ui.Box(c).Height(400).Shrink(0)
			View(c, st).Shrink(0)
		})
	}, 600, 400)
	tt.Frame()
	st.Select(200*5, 200*5)
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if !st.Focused() {
		t.Fatal("Tab didn't focus the editor")
	}
	lh := st.ed.view.lineHeight
	top := 400 + float32(st.LineTop(200))
	if scroll.Y > top || scroll.Y+400 < top+lh {
		t.Fatalf("focused, the view scrolled to %v, which doesn't show the caret at %v", scroll.Y, top)
	}
}

// TestCodeEditorNoScrollbarsCancelsDrag checks that hiding the bars while
// a thumb is dragged lets the drag go: the pointer's moves scroll no
// more.
func TestCodeEditorNoScrollbarsCancelsDrag(t *testing.T) {
	ct := newEditorTest(t, "", "|"+strings.Repeat("line\n", 200))
	x := float32(600 - scrollbarWidth/2)
	ct.tt.Move(x, 5)
	ct.tt.Press(x, 5)
	ct.tt.Move(x, 50)
	if ct.st.ScrollY() <= 0 {
		t.Fatal("dragging the thumb did not scroll")
	}
	ct.st.NoScrollbars = true
	ct.tt.Frame()
	before := ct.st.ScrollY()
	ct.tt.Move(x, 300)
	if ct.st.ed.scrollbar.active || ct.st.ScrollY() != before {
		t.Fatalf("hidden bars kept the drag: active %v, scrolled from %v to %v", ct.st.ed.scrollbar.active, before, ct.st.ScrollY())
	}
	ct.tt.Release(x, 300)
}
