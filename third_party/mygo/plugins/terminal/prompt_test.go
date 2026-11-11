package terminal

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// rowsOf returns the rows of a prompt as a terminal of 10 columns shows
// them, a "+" ending those that soft-wrap.
func rowsOf(lines ...string) []promptRow {
	rows := make([]promptRow, len(lines))
	for i, l := range lines {
		l, wrap := strings.CutSuffix(l, "+")
		text := []rune(l + strings.Repeat(" ", 10-len(l)))
		cols := make([]int, len(text))
		for x := range cols {
			cols[x] = x
		}
		rows[i] = newPromptRow(text, cols, wrap)
	}
	return rows
}

func TestPromptArrows(t *testing.T) {
	for _, c := range []struct {
		name             string
		rows             []promptRow
		cx, cy, col, row int
		application      bool
		want             string
	}{
		{"left on the row", rowsOf("$ echo hi"), 9, 0, 4, 0, false, "\x1b[D\x1b[D\x1b[D\x1b[D\x1b[D"},
		{"right on the row", rowsOf("$ echo hi"), 2, 0, 4, 0, false, "\x1b[C\x1b[C"},
		{"on the cursor", rowsOf("$ echo hi"), 4, 0, 4, 0, false, ""},
		{"application cursor keys", rowsOf("$ echo hi"), 9, 0, 7, 0, true, "\x1bOD\x1bOD"},
		// Past the end of the text: to its end, and no further.
		{"past the end", rowsOf("$ echo hi"), 2, 0, 9, 0, false, strings.Repeat("\x1b[C", 7)},
		{"past the end at the end", rowsOf("$ echo hi"), 9, 0, 9, 0, false, ""},
		// Blanks typed before the cursor are the input's.
		{"typed blanks", rowsOf("$ ab"), 7, 0, 4, 0, false, strings.Repeat("\x1b[D", 3)},
		// Over the rows a command wraps onto, their every column.
		{"up a wrap", rowsOf("$ echo he+", "llo  worl+", "d"), 1, 2, 7, 0, false, strings.Repeat("\x1b[D", 14)},
		{"down a wrap", rowsOf("$ echo he+", "llo  worl+", "d"), 7, 0, 1, 2, false, strings.Repeat("\x1b[C", 14)},
		{"down a wrap past the end", rowsOf("$ echo he+", "llo  worl+", "d"), 7, 0, 6, 2, false, strings.Repeat("\x1b[C", 14)},
		{"onto a wrapped row's end", rowsOf("$ echo he+", "llo  worl+", "d"), 1, 2, 9, 1, false, "\x1b[D\x1b[D"},
		// A line break is a character, the blanks after its text none.
		{"up a line break", rowsOf("$ abc", "de"), 1, 1, 3, 0, false, strings.Repeat("\x1b[D", 4)},
		{"down a line break", rowsOf("$ abc", "de"), 3, 0, 2, 1, false, strings.Repeat("\x1b[C", 5)},
	} {
		if got := string(promptArrows(c.rows, c.cx, c.cy, c.col, c.row, c.application)); got != c.want {
			t.Errorf("%s: sent %q, want %q", c.name, got, c.want)
		}
	}
}

// Alt+click on the input of a shell's prompt (OSC 133) moves its cursor
// there, with arrow keys, and selects nothing; elsewhere it does nothing.
func TestAltClickMovesTheCursor(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	term.mu.Lock()
	cols, _ := term.term.Size()
	term.mu.Unlock()
	// A command that wraps onto a second row, the cursor 3 characters
	// before its end.
	command := "echo " + strings.Repeat("x", cols)
	term.Feed([]byte("out\r\n\x1b]133;A\a$ \x1b]133;B\a" + command + "\x1b[3D"))
	tt.Frame()
	click := func(col, row int) string {
		t.Helper()
		x, y := cellCenter(term, col, row)
		tt.ClickAtWith(ui.Alt, x, y)
		return conn.take(term)
	}
	cx := (2+len(command))%cols - 3
	if got, want := click(2, 1), strings.Repeat("\x1b[D", len(command)-3); got != want {
		t.Errorf("Alt+click on the input's start sent %q, want %q", got, want)
	}
	if got, want := click(cols-1, 2), "\x1b[C\x1b[C\x1b[C"; got != want {
		t.Errorf("Alt+click past the input's end sent %q, want %q", got, want)
	}
	if got := click(cx, 2); got != "" {
		t.Errorf("Alt+click on the cursor sent %q", got)
	}
	term.mu.Lock()
	text, selected := term.term.SelectionText()
	term.mu.Unlock()
	if selected {
		t.Errorf("Alt+click selected %q", text)
	}
	// Outside the input: the output above, the rows below.
	if got := click(1, 0); got != "" {
		t.Errorf("Alt+click on the output sent %q", got)
	}
	if got := click(1, 3); got != "" {
		t.Errorf("Alt+click below the input sent %q", got)
	}
	// Cursor keys in the application's mode, which the shell set.
	term.Feed([]byte("\x1b[?1h"))
	if got := click(cx-1, 2); got != "\x1bOD" {
		t.Errorf("Alt+click with DECCKM sent %q", got)
	}
	term.Feed([]byte("\x1b[?1l"))
	// A click without Alt selects as ever.
	x, y := cellCenter(term, 2, 1)
	tt.ClickAt(x, y)
	if got := conn.take(term); got != "" {
		t.Errorf("a click sent %q", got)
	}
	// Not while the view shows the scrollback, where the rows are others
	// than the cursor's: below, a prompt whose input wraps, its first row
	// where the cursor's shows once scrolled back a row.
	term.Feed([]byte("\x1b]133;C\a" + strings.Repeat("\r\n", 30) + "\x1b]133;A\a$ \x1b]133;B\a" + command))
	term.mu.Lock()
	_, cy := term.term.Cursor()
	term.term.ScrollBy(-1)
	term.mu.Unlock()
	tt.Frame()
	if got := click(2, cy); got != "" {
		t.Errorf("Alt+click on the scrollback sent %q", got)
	}
	term.mu.Lock()
	term.term.ScrollToBottom()
	term.mu.Unlock()
	tt.Frame()
	if got, want := click(2, cy-1), strings.Repeat("\x1b[D", len(command)); got != want {
		t.Errorf("Alt+click at the bottom again sent %q, want %q", got, want)
	}
	// Not on a full-screen program's screen.
	term.Feed([]byte("\x1b[?1049h\x1b[H\x1b]133;A\a$ \x1b]133;B\aecho hi"))
	tt.Frame()
	if got := click(2, 0); got != "" {
		t.Errorf("Alt+click on the alternate screen sent %q", got)
	}
}

// A shell that does not mark its prompt gets no arrows.
func TestAltClickWithoutShellIntegration(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	term.Feed([]byte("$ echo hello"))
	tt.Frame()
	x, y := cellCenter(term, 2, 0)
	tt.ClickAtWith(ui.Alt, x, y)
	if got := conn.take(term); got != "" {
		t.Errorf("Alt+click sent %q", got)
	}
}
