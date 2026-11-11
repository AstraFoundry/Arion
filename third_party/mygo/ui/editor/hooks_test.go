package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// OnInput sees keys and typed text before the editor: what it takes the
// editor leaves, the rest goes on as without it.
func TestOnInputTakesKeys(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	var texts, keys int
	ct.st.OnInput = func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputText:
			texts++
			return ev.Text == "x"
		case ui.InputKeyDown:
			keys++
			return ev.Key == ui.KeyBackspace
		}
		return false
	}
	ct.tt.Frame()
	ct.typ("x")
	ct.key(0, ui.KeyBackspace)
	ct.want("ab|")
	ct.typ("y")
	ct.key(0, ui.KeyLeft)
	ct.want("ab|y")
	if texts != 2 || keys != 2 {
		t.Fatalf("OnInput saw %d texts and %d keys, want 2 and 2", texts, keys)
	}
}

// OnInput hears of an event once, even when what it does with one it
// leaves, as edit the buffer, has the editor run its own handling again on
// the text as it is.
func TestOnInputOnce(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	calls := 0
	ct.st.OnInput = func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputText {
			calls++
			ct.st.Buffer.Edit(Edit{Start: 0, End: 0, Text: ">"})
		}
		return false
	}
	ct.tt.Frame()
	ct.typ("y")
	ct.want(">aby|")
	if calls != 1 {
		t.Fatalf("OnInput heard of the text %d times, want once", calls)
	}
}

// VisibleLines are the lines the last frame showed, scrolled or not.
func TestVisibleLines(t *testing.T) {
	ct := newEditorTest(t, "", strings.Repeat("line\n", 200))
	lh := float64(ct.st.ed.view.lineHeight)
	first, last := ct.st.VisibleLines()
	if want := int(400 / lh); first != 0 || last != want {
		t.Fatalf("visible lines %d to %d, want 0 to %d", first, last, want)
	}
	ct.st.ScrollTo(50)
	ct.tt.Frame()
	first, last = ct.st.VisibleLines()
	if want := 50 + int(400/lh); first != 50 || last != want {
		t.Fatalf("scrolled, visible lines %d to %d, want 50 to %d", first, last, want)
	}
}

// pixel returns the color of the last frame at x, y in DIPs of an
// editor test's window, 600 wide.
func (ct *editorTest) pixel(x, y float32) (r, g, b uint8) {
	img := ct.tt.Image()
	s := float32(img.Bounds().Dx()) / 600
	c := img.RGBAAt(int(x*s), int(y*s))
	return c.R, c.G, c.B
}

// redTheme paints carets red on white, for tests to find them.
var redTheme = &Theme{Background: ui.RGB(255, 255, 255), Cursor: ui.RGB(255, 0, 0)}

// A block caret covers the character after it, or a space's width at the
// end of a line, in the color of carets made translucent; an underline
// caret underlines it.
func TestCursorShape(t *testing.T) {
	ct := newEditorTest(t, "", "|abc\nab")
	ct.st.Theme, ct.st.NoActiveLine = redTheme, true
	ct.tt.Frame()
	ed := &ct.st.ed
	cw, lh := ed.view.charWidth, ed.view.lineHeight
	red := func(line, col int, y float32) bool {
		r, g, _ := ct.pixel(ed.textX+(float32(col)+0.6)*cw, float32(line)*lh+y)
		return r > 200 && g < 230
	}
	inBlock := func(line, col int) bool { return red(line, col, 2) }
	if inBlock(0, 0) {
		t.Fatal("a bar caret painted a block")
	}
	ct.st.CursorShape = CursorBlock
	ct.tt.Frame()
	if !inBlock(0, 0) || inBlock(0, 1) {
		t.Fatal("the block caret did not cover the first character alone")
	}
	if r, g, b := ct.pixel(ed.textX+0.6*cw, 2); r < 250 || g < 60 || g != b {
		t.Fatalf("the block caret is %d,%d,%d, not translucent red", r, g, b)
	}
	// At the end of a line, a space's width.
	ct.st.Select(7, 7)
	ct.tt.Frame()
	if !inBlock(1, 2) || inBlock(1, 3) || inBlock(0, 0) {
		t.Fatal("the block caret at the end of a line is not a space wide")
	}
	// An underline under the character, not over it.
	ct.st.CursorShape = CursorUnderline
	ct.st.Select(0, 0)
	ct.tt.Frame()
	if !red(0, 0, lh-1) || red(0, 0, 2) || red(0, 1, lh-1) {
		t.Fatal("the underline caret does not underline the first character alone")
	}
	// Without the keyboard, a block is outlined, not filled.
	ct.st.CursorShape = CursorBlock
	ct.tt.SetFocused(false)
	ct.tt.Frame()
	if ct.st.Focused() || inBlock(0, 0) || !red(0, 0, 0.5) || !red(0, 0, lh-0.5) {
		t.Fatal("the block caret of an unfocused editor is not outlined")
	}
}

// A block caret covers a whole character, with its combining marks.
func TestBlockCaretCombining(t *testing.T) {
	ct := newEditorTest(t, "", "|e\u0301x")
	ct.st.Theme, ct.st.NoActiveLine, ct.st.CursorShape = redTheme, true, CursorBlock
	ct.tt.Frame()
	ed := &ct.st.ed
	l := ed.view.line(0)
	want := ed.view.xOf(l, &l.rows[0], l.toDisp(3, false))
	if w := ed.caretWidth(l, &l.rows[0], 0, ed.view.xOf(l, &l.rows[0], 0)); w != want {
		t.Fatalf("the block caret is %v wide, want %v, that of é", w, want)
	}
}

// With NoInputMethod, the editor turns the system's input methods off
// while it takes text: keys type text, which comes as InputText for
// OnInput, as keys do in Vim's normal mode, and nothing is composed.
func TestNoInputMethod(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	ct.st.NoInputMethod = true
	var typed []string
	ct.st.OnInput = func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputText {
			typed = append(typed, ev.Text)
			return true
		}
		return false
	}
	ct.tt.Frame()
	if active, off := ct.tt.TextInputActive(); !active || !off {
		t.Fatalf("text input %v, input methods off %v; want on, off", active, off)
	}
	ct.tt.Compose("に", 1)
	ct.tt.Frame()
	ct.want("ab|")
	ct.tt.TypeKey(0, ui.KeyJ, "j")
	ct.tt.Frame()
	ct.want("ab|")
	if fmt.Sprint(typed) != "[j]" {
		t.Fatalf("OnInput saw %q", typed)
	}
	// Back on, input methods compose in the text.
	ct.st.NoInputMethod = false
	ct.tt.Frame()
	if active, off := ct.tt.TextInputActive(); !active || off {
		t.Fatalf("text input %v, input methods off %v; want on, on", active, off)
	}
	ct.tt.Compose("に", 1)
	ct.tt.Frame()
	ct.want("abに|")
}

// An input method replacing text other than the selection, as the accents
// of press and hold replace the letter before the caret, edits the text
// itself: OnInput hears only of text typed at the carets.
func TestOnInputReplacement(t *testing.T) {
	ct := newEditorTest(t, "", "cafe|")
	var typed []string
	ct.st.OnInput = func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputText {
			typed = append(typed, ev.Text)
		}
		return false
	}
	ct.tt.Frame()
	ct.st.ed.ReplaceText(&ui.TextInputRange{Start: 3, End: 4}, "é")
	ct.want("café|")
	ct.st.ed.ReplaceText(&ui.TextInputRange{Start: 4, End: 4}, "!")
	ct.want("café!|")
	if fmt.Sprint(typed) != "[!]" {
		t.Fatalf("OnInput saw %q, want the text typed at the caret alone", typed)
	}
}

// The key after the first of a chord is the chord's, not OnInput's.
// The modifier the second key of a chord is held with comes on its own first:
// the chord waits for the key, and OnInput sees it.
func TestChordOverModifierKey(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	ct.st.Bindings = []Binding{{Mods: ui.Cmd, Key: ui.KeyK, ThenMods: ui.Cmd, ThenKey: ui.KeyU, Command: Uppercase}}
	ct.tt.Frame()
	ct.st.Select(0, 2)
	ct.key(ui.Cmd, ui.KeyK)
	ct.tt.CodeUp(0, ui.CodeControlLeft)
	ct.tt.CodeDown(ui.Cmd, ui.CodeControlLeft)
	ct.key(ui.Cmd, ui.KeyU)
	ct.key(0, ui.KeyRight)
	ct.want("AB|")
}

func TestOnInputChord(t *testing.T) {
	ct := newEditorTest(t, "", "ab|")
	var keys []ui.Key
	ct.st.OnInput = func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputKeyDown {
			keys = append(keys, ev.Key)
		}
		return false
	}
	ct.st.Bindings = []Binding{{Mods: ui.Cmd, Key: ui.KeyK, ThenMods: ui.Cmd, ThenKey: ui.KeyU, Command: Uppercase}}
	ct.tt.Frame()
	ct.st.Select(0, 2)
	ct.key(ui.Cmd, ui.KeyK)
	ct.key(ui.Cmd, ui.KeyU)
	ct.key(0, ui.KeyRight)
	ct.want("AB|")
	if fmt.Sprint(keys) != fmt.Sprint([]ui.Key{ui.KeyK, ui.KeyRight}) {
		t.Fatalf("OnInput saw keys %v, want K and Right", keys)
	}
}
