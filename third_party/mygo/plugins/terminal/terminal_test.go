package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"

	// MyGo's own strings in Traditional Chinese, which a test shows.
	_ "github.com/egoist/mygo/locales/zh-Hant"
)

// loadLib loads libghostty-vt, or skips the test where it cannot.
func loadLib(t *testing.T) {
	t.Helper()
	if testing.Short() && os.Getenv("MYGO_GHOSTTY_VT") == "" {
		t.Skip("downloads libghostty-vt")
	}
	if err := Load(); err != nil {
		t.Fatal(err)
	}
}

// waitFor runs frames until cond holds: what a program writes arrives
// when it does, on the terminal's reader. The deadline is a hang guard.
func waitFor(t *testing.T, tt *ui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
		tt.Frame()
	}
}

func shell(t *testing.T, opts Options) *Terminal {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	if opts.Command == nil {
		opts.Command = []string{"/bin/sh"}
	}
	opts.Env = append(opts.Env, "PS1=$ ", "ENV=")
	term, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Close() })
	return term
}

func save(t *testing.T, tt *ui.Tester, name string) {
	dir := os.Getenv("MYGO_TEST_IMAGES")
	if dir == "" {
		return
	}
	f, err := os.Create(dir + "/" + name + ".png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, tt.Image())
}

func TestTypeInShell(t *testing.T) {
	loadLib(t)
	term := shell(t, Options{})
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 480, 240)
	waitFor(t, tt, "the prompt", func() bool { return strings.Contains(term.Text(), "$") })
	for _, r := range "echo hi there" {
		tt.Type(string(r))
	}
	tt.Key(0, ui.KeyEnter)
	waitFor(t, tt, "the output", func() bool { return strings.Contains(term.Text(), "\nhi there\n") })
	if cols, rows := term.Size(); cols < 40 || rows < 10 {
		t.Errorf("size = %d×%d", cols, rows)
	}
	save(t, tt, "shell")
}

// pipe is a Conn whose output the test writes with Feed, and which keeps
// what the terminal sends.
type pipe struct {
	closed chan struct{}
	mu     sync.Mutex
	sent   []byte
}

func newPipe() *pipe { return &pipe{closed: make(chan struct{})} }

func (p *pipe) Read([]byte) (int, error) {
	<-p.closed
	return 0, os.ErrClosed
}

func (p *pipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	p.sent = append(p.sent, b...)
	p.mu.Unlock()
	return len(b), nil
}

// take returns what term sent since the last take: all that it queued
// for its Conn before take, which waits until the queue wrote it (see
// written), and nothing queued after.
func (p *pipe) take(term *Terminal) string {
	written(term)
	p.mu.Lock()
	defer p.mu.Unlock()
	s := string(p.sent)
	p.sent = nil
	return s
}

// written returns once the input queue of term, which writes to its Conn
// (or program) from a goroutine of its own, wrote what was queued before,
// and ran what was queued to run once that was written, as the statuses
// of a paste: it queues a marker, which the queue reaches in order. A
// closed queue writes nothing more.
func written(term *Terminal) {
	done := make(chan struct{})
	q := &term.in
	q.mu.Lock()
	q.init()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.queue = append(q.queue, queued{done: func() { close(done) }})
	q.cond.Signal()
	q.mu.Unlock()
	select {
	case <-done:
	case <-time.After(time.Minute): // a hang guard
		panic("the terminal's input queue wrote nothing for a minute")
	}
}
func (p *pipe) Close() error {
	select {
	case <-p.closed:
	default:
		close(p.closed)
	}
	return nil
}

func TestPaint(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	var b strings.Builder
	for i := range 16 {
		b.WriteString("\x1b[48;5;" + itoa(i) + "m  ")
	}
	b.WriteString("\x1b[0m\r\n")
	for i := range 32 {
		b.WriteString("\x1b[48;2;" + itoa(i*8) + ";100;" + itoa(255-i*8) + "m ")
	}
	b.WriteString("\x1b[0m\r\n")
	b.WriteString("\x1b[1mbold\x1b[0m \x1b[3mitalic\x1b[0m \x1b[1;3mboth\x1b[0m \x1b[2mfaint\x1b[0m \x1b[4munder\x1b[0m \x1b[4:3mcurly\x1b[0m \x1b[21mdouble\x1b[0m \x1b[9mstrike\x1b[0m \x1b[7minverse\x1b[0m\r\n")
	b.WriteString("\x1b[31mred \x1b[32mgreen \x1b[33myellow \x1b[34mblue \x1b[35mmagenta \x1b[36mcyan \x1b[91mbright\x1b[0m\r\n")
	b.WriteString("wide: 世界 こんにちは 한국어 | emoji: 👍🏽 🇯🇵 👨‍👩‍👧 | combining: é́ ä\r\n")
	b.WriteString("┌──┬──┐ ╭──╮ ╔══╦══╗ ┏━━┓ █▓▒░ ▀▄▌▐ ▖▗▘▝\r\n")
	b.WriteString("│ab│cd│ │  │ ║  ║  ║ ┃  ┃ \r\n")
	b.WriteString("├──┼──┤ ╰──╯ ╠══╬══╣ ┗━━┛ ╱╲╳\r\n")
	b.WriteString("└──┴──┘      ╚══╩══╝\r\n")
	b.WriteString("ligatures? -> => != === <= >=  www  fi fl")
	term.Feed([]byte(b.String()))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 640, 280)
	tt.SetScale(2)
	save(t, tt, "paint")
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Repeat(" ", 0) + string(rune('0'+i/100%10)) + string(rune('0'+i/10%10)) + string(rune('0'+i%10)))
}

func TestScrollback(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	offset := func() uint64 {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.term.Scrollbar().Offset
	}
	bottom := offset()
	tt.Wheel(100, 100, 0, -hostNotch()) // a notch of a mouse wheel up
	if got := offset(); got != bottom-3 {
		t.Errorf("after scrolling a notch up, offset = %d, want %d", got, bottom-3)
	}
	tt.Key(ui.Shift, ui.KeyPageUp)
	if got := offset(); got >= bottom-3 {
		t.Errorf("Shift+Page Up did not scroll: offset = %d", got)
	}
	save(t, tt, "scrolled")
	// Typing goes back to the bottom.
	tt.Type("x")
	if got := offset(); got != bottom {
		t.Errorf("after typing, offset = %d, want %d", got, bottom)
	}
}

// TestScrollbarDrag: the scroll bar shows while the pointer is over the
// terminal, as the code editor's does, and goes as it leaves; over its
// strip the pointer is the arrow rather than the I-beam. It drags the
// scrollback along: by the thumb, or from where the track is pressed.
func TestScrollbarDrag(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	offset := func() uint64 {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.term.Scrollbar().Offset
	}
	// shows reports whether the bar paints at (395, y), by the pixel off
	// its strip.
	shows := func(y int) bool {
		img := tt.Image()
		s := img.Bounds().Dx() / 400
		return img.RGBAAt(395*s, y*s) != img.RGBAAt(380*s, y*s)
	}
	bottom := offset()
	if shows(188) {
		t.Error("the bar shows, the pointer away")
	}
	tt.Move(100, 100)
	tt.Frame()
	if !shows(188) {
		t.Error("the bar doesn't show with the pointer over the terminal")
	}
	// Over the strip, at the bottom: the bar, and the arrow.
	tt.Move(395, 100)
	tt.Frame()
	if c := tt.Cursor(); c != ui.CursorDefault {
		t.Errorf("over the bar, the pointer is %v, want the arrow", c)
	}
	if !shows(188) {
		t.Error("the bar doesn't show with the pointer over it")
	}
	// The thumb, at the bottom, dragged halfway up and past the top.
	tt.Press(395, 188)
	tt.Move(395, 100)
	if got := offset(); got == 0 || got >= bottom {
		t.Errorf("halfway up, offset = %d, want between 0 and %d", got, bottom)
	}
	// The pointer off the strip: the drag still has it.
	tt.Move(100, -50)
	if got := offset(); got != 0 {
		t.Errorf("dragged past the top, offset = %d, want 0", got)
	}
	tt.Release(100, -50)
	term.mu.Lock()
	_, sel := term.term.SelectionText()
	term.mu.Unlock()
	if sel {
		t.Error("dragging the bar selected text")
	}
	tt.Move(100, 100)
	tt.Frame()
	if c := tt.Cursor(); c != ui.CursorText {
		t.Errorf("over the text, the pointer is %v, want the I-beam", c)
	}
	if !shows(12) {
		t.Error("the bar went with the pointer still over the terminal")
	}
	// The pointer away, it goes though the viewport is up, and comes back
	// with it.
	tt.Move(100, -50)
	tt.Frame()
	if shows(12) {
		t.Error("the bar stayed with the pointer away")
	}
	tt.Wheel(100, 100, 0, 40)
	tt.Frame()
	if !shows(15) {
		t.Error("the bar didn't show with the pointer back")
	}
	// A press low on the track brings the thumb there.
	tt.Press(395, 190)
	tt.Release(395, 190)
	if got := offset(); got < bottom/2 {
		t.Errorf("after a press low on the track, offset = %d, want past %d", got, bottom/2)
	}
	// A full-screen program's screen has no scrollback, and no bar.
	term.Feed([]byte("\x1b[?1049h"))
	tt.Move(395, 120)
	tt.Frame()
	if c := tt.Cursor(); c != ui.CursorText {
		t.Errorf("over a full-screen program's edge, the pointer is %v, want the I-beam", c)
	}
}

// A match that wraps from a row above the viewport shows on the rows of it
// that the viewport shows.
func TestSearchWrappedMatch(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	cols, _ := term.Size()
	term.Feed([]byte(lines(0, 50, "line ") + strings.Repeat("x", cols-3) + "needle\r\n" + lines(0, 50, "after ")))
	term.Search("needle")
	for range 100 {
		if s := term.SearchStatus(); !s.Searching && s.Selected > 0 {
			break
		}
		tt.Frame()
	}
	// The match starts on row 50, from the top of the scrollback, and ends
	// on row 51: the viewport from row 51 shows its end alone.
	term.mu.Lock()
	term.term.ScrollToRow(51)
	term.mu.Unlock()
	tt.Frame()
	want := matchSpan{y: 0, x0: 0, x1: 2, selected: true}
	if len(term.v.matches) != 1 || term.v.matches[0] != want {
		t.Errorf("matches show at %+v, want %+v", term.v.matches, want)
	}
}

// A search finds the matches in the scrollback and on the screen, selects
// the newest, highlights those that show, and goes to the older ones,
// scrolling; OnSearch follows it, and ending it clears the highlights.
func TestSearch(t *testing.T) {
	loadLib(t)
	var got []SearchStatus
	term, err := New(Options{Conn: newPipe(), OnSearch: func(s SearchStatus) { got = append(got, s) }})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		if i%10 == 5 {
			b.WriteString("a NEEDLE " + itoa(i) + "\r\n")
		} else {
			b.WriteString("line " + itoa(i) + "\r\n")
		}
	}
	b.WriteString("needle at the prompt")
	term.Feed([]byte(b.String()))
	tt.Frame()
	term.Search("needle")
	for range 100 {
		if s := term.SearchStatus(); !s.Searching && s.Selected > 0 {
			break
		}
		tt.Frame()
	}
	tt.Frame()
	want := SearchStatus{Text: "needle", Total: 11, Selected: 1}
	if s := term.SearchStatus(); s != want {
		t.Fatalf("status = %+v, want %+v", s, want)
	}
	if len(got) == 0 || got[len(got)-1] != want {
		t.Errorf("OnSearch was told %+v", got)
	}
	selected := func() (matchSpan, bool) {
		for _, m := range term.v.matches {
			if m.selected {
				return m, true
			}
		}
		return matchSpan{}, false
	}
	if m, ok := selected(); !ok || m.x0 != 0 || m.x1 != 5 {
		t.Errorf("selected match shows at %+v, %v", m, ok)
	}
	// The selected match's color shows.
	img := tt.Image()
	peach := false
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y && !peach; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if r, g, bl, _ := img.At(x, y).RGBA(); r>>8 == 0xf2 && g>>8 == 0xa5 && bl>>8 == 0x7e {
				peach = true
				break
			}
		}
	}
	if !peach {
		t.Error("the selected match is not highlighted")
	}
	save(t, tt, "search")
	// Going to older matches scrolls up into the scrollback, wrapping
	// around past the oldest.
	term.mu.Lock()
	bottom := term.term.Scrollbar().Offset
	term.mu.Unlock()
	for range 3 {
		term.SearchNext()
	}
	tt.Frame()
	term.mu.Lock()
	offset := term.term.Scrollbar().Offset
	term.mu.Unlock()
	if s := term.SearchStatus(); s.Selected != 4 || offset >= bottom {
		t.Errorf("after 3 next: %+v, offset %d of %d", s, offset, bottom)
	}
	_, rows := term.Size()
	if m, ok := selected(); !ok || m.y != rows/2 {
		t.Errorf("the selected match shows at %+v (%v), not in the middle of %d rows", m, ok, rows)
	}
	term.SearchPrevious()
	tt.Frame()
	if s := term.SearchStatus(); s.Selected != 3 {
		t.Errorf("after previous: %+v", s)
	}
	for range 8 {
		term.SearchNext()
	}
	tt.Frame()
	if s := term.SearchStatus(); s.Selected != 11 {
		t.Errorf("at the oldest: %+v", s)
	}
	term.SearchNext()
	tt.Frame()
	if s := term.SearchStatus(); s.Selected != 1 {
		t.Errorf("after wrapping around: %+v", s)
	}
	term.EndSearch()
	tt.Frame()
	if len(term.v.matches) != 0 || got[len(got)-1] != (SearchStatus{}) {
		t.Errorf("after ending: %d matches, told %+v", len(term.v.matches), got[len(got)-1])
	}
}

// A snapshot of a terminal no view shows restores what it shows in
// another, as a server keeping sessions sends it to a window attaching.
func TestSnapshot(t *testing.T) {
	loadLib(t)
	newTerm := func() *Terminal {
		term, err := New(Options{Conn: newPipe()})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { term.Close() })
		term.Resize(40, 6)
		return term
	}
	a := newTerm()
	for i := range 10 {
		a.Feed([]byte("line " + itoa(i) + " \x1b[31mred\x1b[0m \x1b[1;4mbold\x1b[0m\r\n"))
	}
	a.Feed([]byte("prompt % \x1b[?2004h"))
	snap := a.Snapshot()
	b := newTerm()
	b.Feed(snap)
	// The cursor and the modes come too: what follows lands alike.
	a.Feed([]byte("x"))
	b.Feed([]byte("x"))
	if a.Text() != b.Text() {
		t.Errorf("restored text\n%q, want\n%q", b.Text(), a.Text())
	}
	b.mu.Lock()
	paste := b.term.Mode(vt.ModeBracketedPaste)
	b.mu.Unlock()
	if !paste {
		t.Error("bracketed paste was not restored")
	}
	// The blank rows at the end of the screen come too: the cursor on one
	// under the last line, as while a program prints lines, keeps the line
	// above from what is written next, and a screen cleared shows blank,
	// what it showed in the scrollback.
	for _, last := range []string{"", "\x1b[H\x1b[2J"} {
		d := newTerm()
		for i := range 10 {
			d.Feed([]byte("row " + itoa(i) + "\r\n"))
		}
		d.Feed([]byte(last))
		e := newTerm()
		e.Feed(d.Snapshot())
		d.Feed([]byte("next"))
		e.Feed([]byte("next"))
		if d.Text() != e.Text() {
			t.Errorf("after %q, restored text\n%q, want\n%q", last, e.Text(), d.Text())
		}
		d.mu.Lock()
		want := d.term.Scrollbar()
		d.mu.Unlock()
		e.mu.Lock()
		got := e.term.Scrollbar()
		e.mu.Unlock()
		if got != want {
			t.Errorf("after %q, restored scrollbar %+v, want %+v", last, got, want)
		}
	}
	// A full-screen program's screen, at a new size.
	a.Feed([]byte("\x1b[?1049h\x1b[H\x1b[2Jfull screen\x1b[3;5Hhere"))
	a.Resize(30, 5)
	c := newTerm()
	c.Resize(30, 5)
	c.Feed(a.Snapshot())
	c.Feed([]byte("!"))
	if got := c.Text(); !strings.Contains(got, "full screen") || !strings.Contains(got, "here!") {
		t.Errorf("restored alternate screen %q", got)
	}
}

// cellCenter returns the middle of a cell of the view, in DIPs.
func cellCenter(term *Terminal, col, row int) (float32, float32) {
	v := term.v
	return (float32(v.ox) + (float32(col)+0.5)*float32(v.cellW)) / v.scale, (float32(v.oy) + (float32(row)+0.5)*float32(v.cellH)) / v.scale
}

func TestSelectAndCopy(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world\r\nsecond line"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	copyMods := ui.Ctrl | ui.Shift
	if runtime.GOOS == "darwin" {
		copyMods = ui.Super
	}
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 4, 0)
	tt.Press(x0-2, y0)
	tt.Move(x1, y1)
	tt.Move(x1+2, y1)
	tt.Release(x1+2, y1)
	tt.Key(copyMods, ui.KeyC)
	if got := tt.Clipboard(); got != "hello" {
		t.Errorf("dragging copied %q", got)
	}
	// A double click selects a word, a triple click a line.
	x, y := cellCenter(term, 8, 0)
	tt.Press(x, y)
	tt.Release(x, y)
	tt.Press(x, y)
	tt.Release(x, y)
	tt.Key(copyMods, ui.KeyC)
	if got := tt.Clipboard(); got != "world" {
		t.Errorf("double click copied %q", got)
	}
	x, y = cellCenter(term, 2, 1)
	for range 3 {
		tt.Press(x, y)
		tt.Release(x, y)
	}
	tt.Key(copyMods, ui.KeyC)
	if got := tt.Clipboard(); got != "second line" {
		t.Errorf("triple click copied %q", got)
	}
	save(t, tt, "selected")
	// The Edit menu's Copy and Select All.
	tt.Key(0, ui.KeyEscape) // typing clears the selection
}

func TestPaste(t *testing.T) {
	loadLib(t)
	term := shell(t, Options{Command: []string{"/bin/cat"}})
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.SetClipboard("pasted text")
	pasteMods := ui.Ctrl | ui.Shift
	if runtime.GOOS == "darwin" {
		pasteMods = ui.Super
	}
	tt.Key(pasteMods, ui.KeyV)
	tt.Key(0, ui.KeyEnter)
	waitFor(t, tt, "the echo of cat", func() bool { return strings.Count(term.Text(), "pasted text") == 2 })
}

func TestKeys(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	key := func(mods ui.Modifiers, k ui.Key, text, want string) {
		t.Helper()
		if text != "" {
			tt.TypeKey(mods, k, text)
		} else {
			tt.Key(mods, k)
		}
		if got := conn.take(term); got != want {
			t.Errorf("%v %v %q sent %q, want %q", mods, k, text, got, want)
		}
	}
	key(0, ui.KeyA, "a", "a")
	key(ui.Shift, ui.KeyA, "A", "A")
	key(ui.Ctrl, ui.KeyC, "", "\x03")
	key(0, ui.KeyEnter, "", "\r")
	key(0, ui.KeyBackspace, "", "\x7f")
	key(0, ui.KeyUp, "", "\x1b[A")
	key(ui.Shift, ui.KeyTab, "", "\x1b[Z")
	key(0, ui.KeyF5, "", "\x1b[15~")
	if runtime.GOOS != "darwin" {
		key(ui.Alt, ui.KeyB, "", "\x1bb")
	}
	// Text without its key, as input methods and Linux send it.
	tt.Type("世界")
	if got := conn.take(term); got != "世界" {
		t.Errorf("typed %q", got)
	}
	// Programs that ask for the Kitty keyboard protocol get it.
	term.Feed([]byte("\x1b[>1u"))
	key(0, ui.KeyEscape, "", "\x1b[27u")
	key(ui.Ctrl, ui.KeyC, "", "\x1b[99;5u")
	// Programs in cursor key mode get application keys.
	term.Feed([]byte("\x1b[<u\x1b[?1h"))
	key(0, ui.KeyUp, "", "\x1bOA")
}

func TestComposition(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	term.Feed([]byte("abc"))
	tt.Frame()
	caret, ok := tt.TextCaret()
	x, y := cellCenter(term, 3, 0)
	if !ok || !caret.Contains(x, y) {
		t.Errorf("input methods compose at %+v (%v), not in the cell of the cursor at %v, %v", caret, ok, x, y)
	}
	tt.Compose("ni", 2)
	save(t, tt, "composing")
	if got := conn.take(term); got != "" {
		t.Errorf("composing sent %q", got)
	}
	tt.Compose("你", 1)
	tt.Type("你")
	if got := conn.take(term); got != "你" {
		t.Errorf("committing sent %q", got)
	}
}

func TestMouse(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	term.Feed([]byte("\x1b[?1002h\x1b[?1006h"))
	tt.Frame()
	x, y := cellCenter(term, 2, 3)
	tt.Press(x, y)
	if got, want := conn.take(term), "\x1b[<0;3;4M"; got != want {
		t.Errorf("press sent %q, want %q", got, want)
	}
	x2, y2 := cellCenter(term, 5, 3)
	tt.Move(x2, y2)
	if got, want := conn.take(term), "\x1b[<32;6;4M"; got != want {
		t.Errorf("drag sent %q, want %q", got, want)
	}
	tt.Release(x2, y2)
	if got, want := conn.take(term), "\x1b[<0;6;4m"; got != want {
		t.Errorf("release sent %q, want %q", got, want)
	}
	tt.Wheel(x, y, 0, -40)
	if got := conn.take(term); !strings.HasPrefix(got, "\x1b[<64;3;4M") {
		t.Errorf("wheel sent %q", got)
	}
	// Focus reports.
	term.Feed([]byte("\x1b[?1004h"))
	tt.Frame()
}

func TestExit(t *testing.T) {
	loadLib(t)
	exited := make(chan int, 1)
	var titles []string
	var mu sync.Mutex
	term := shell(t, Options{
		Command: []string{"/bin/sh", "-c", "printf '\\033]2;the title\\007'; echo bye; exit 3"},
		OnExit:  func(code int) { exited <- code },
		OnTitle: func(s string) { mu.Lock(); titles = append(titles, s); mu.Unlock() },
	})
	select {
	case code := <-exited:
		if code != 3 {
			t.Errorf("exit code = %d", code)
		}
	case <-time.After(time.Minute): // a hang guard
		t.Fatal("no exit")
	}
	<-term.Done()
	if term.ExitCode() != 3 || !strings.Contains(term.Text(), "bye\n[Process exited with code 3]") {
		t.Errorf("code %d, text %q", term.ExitCode(), term.Text())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(titles) != 1 || titles[0] != "the title" || term.Title() != "the title" {
		t.Errorf("titles = %q", titles)
	}
}

func TestMenus(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("some text"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	// The Edit menu's roles.
	tt.Command("selectAll")
	tt.Command("copy")
	if got := tt.Clipboard(); got != "some text" {
		t.Errorf("Select All and Copy copied %q", got)
	}
	tt.SetClipboard("pasted")
	tt.Command("paste")
	if got := conn.take(term); got != "pasted" {
		t.Errorf("Paste sent %q", got)
	}
	// The context menu.
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	if got := strings.Join(tt.Menu(), ","); !strings.Contains(got, "Copy") || !strings.HasSuffix(got, ",-,Clear") {
		t.Errorf("context menu = %q", got)
	}
	tt.SetClipboard("again")
	if err := tt.ChooseMenuItem("Paste"); err != nil {
		t.Fatal(err)
	}
	if got := conn.take(term); got != "again" {
		t.Errorf("the menu's Paste sent %q", got)
	}
	// In the window's language.
	tt.SetLocale("zh-Hant")
	tt.RightClickAt(x, y)
	if got := strings.Join(tt.Menu(), ","); !strings.Contains(got, "複製") || !strings.HasSuffix(got, ",-,清除") {
		t.Errorf("context menu in Traditional Chinese = %q", got)
	}
}

func TestCopyOnSelect(t *testing.T) {
	loadLib(t)
	for _, on := range []CopyOnSelect{CopyOnSelectNone, CopyOnSelectClipboard} {
		conn := newPipe()
		term, err := New(Options{Conn: conn, CopyOnSelect: on})
		if err != nil {
			t.Fatal(err)
		}
		term.Feed([]byte("hello world"))
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
		x0, y0 := cellCenter(term, 0, 0)
		x1, y1 := cellCenter(term, 4, 0)
		tt.Press(x0-2, y0)
		tt.Move(x1+2, y1)
		tt.Release(x1+2, y1)
		want := ""
		if on == CopyOnSelectClipboard {
			want = "hello"
		}
		if got := tt.Clipboard(); got != want {
			t.Errorf("CopyOnSelect %v: selecting copied %q, want %q", on, got, want)
		}
		tt.Command("selectAll")
		if on == CopyOnSelectClipboard {
			want = "hello world"
		}
		if got := tt.Clipboard(); got != want {
			t.Errorf("CopyOnSelect %v: Select All copied %q, want %q", on, got, want)
		}
		term.Close()
	}
}

// OnCopy tells of each copy, and what made it.
func TestOnCopy(t *testing.T) {
	loadLib(t)
	type copy struct {
		text string
		by   CopySource
	}
	var got []copy
	conn := newPipe()
	term, err := New(Options{Conn: conn, CopyOnSelect: CopyOnSelectClipboard, OnCopy: func(text string, by CopySource) {
		got = append(got, copy{text, by})
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 4, 0)
	tt.Press(x0-2, y0)
	tt.Move(x1+2, y1)
	tt.Release(x1+2, y1)
	copyMods := ui.Ctrl | ui.Shift
	if runtime.GOOS == "darwin" {
		copyMods = ui.Super
	}
	tt.Key(copyMods, ui.KeyC)
	tt.Command("selectAll")
	want := []copy{{"hello", CopySelect}, {"hello", CopyUser}, {"hello world", CopySelect}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OnCopy saw %v, want %v", got, want)
	}
}

func TestSetCopyOnSelect(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	sel := func(from, to int) {
		x0, y0 := cellCenter(term, from, 0)
		x1, y1 := cellCenter(term, to, 0)
		tt.Press(x0-2, y0)
		tt.Move(x1+2, y1)
		tt.Release(x1+2, y1)
	}
	term.SetCopyOnSelect(CopyOnSelectClipboard)
	sel(0, 4)
	if got := tt.Clipboard(); got != "hello" {
		t.Errorf("turned on, selecting copied %q, want %q", got, "hello")
	}
	term.SetCopyOnSelect(CopyOnSelectNone)
	sel(6, 10)
	if got := tt.Clipboard(); got != "hello" {
		t.Errorf("turned off, selecting left %q on the clipboard, want %q", got, "hello")
	}
}

// TestCopyAndPasteKeys: Windows Terminal's keys, performable Control-C
// and -V, Control-Insert and Shift-Insert, and a secondary click that
// copies or pastes.
func TestCopyAndPasteKeys(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, RightClick: RightClickCopyOrPaste, Keybinds: []Keybind{
		{Mods: ui.Ctrl, Key: ui.KeyC, Action: ActionCopy, Performable: true},
		{Mods: ui.Ctrl, Key: ui.KeyV, Action: ActionPaste, Performable: true},
		{Mods: ui.Ctrl, Key: ui.KeyInsert, Action: ActionCopy},
		{Mods: ui.Shift, Key: ui.KeyInsert, Action: ActionPaste},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("some text"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	// Control-C copies a selection and lets go of it; without one it
	// interrupts.
	tt.Command("selectAll")
	tt.Key(ui.Ctrl, ui.KeyC)
	if got := tt.Clipboard(); got != "some text" {
		t.Errorf("Control-C copied %q", got)
	}
	if got := conn.take(term); got != "" {
		t.Errorf("Control-C with a selection sent %q", got)
	}
	tt.Key(ui.Ctrl, ui.KeyC)
	if got := conn.take(term); got != "\x03" {
		t.Errorf("Control-C without a selection sent %q", got)
	}
	tt.SetClipboard("by V")
	tt.Key(ui.Ctrl, ui.KeyV)
	if got := conn.take(term); got != "by V" {
		t.Errorf("Control-V sent %q", got)
	}
	// With nothing to paste, Control-V reaches the program.
	tt.SetClipboard("")
	tt.Key(ui.Ctrl, ui.KeyV)
	if got := conn.take(term); got != "\x16" {
		t.Errorf("Control-V with an empty clipboard sent %q", got)
	}
	tt.Command("selectAll")
	tt.Key(ui.Ctrl, ui.KeyInsert)
	if got := tt.Clipboard(); got != "some text" {
		t.Errorf("Control-Insert copied %q", got)
	}
	tt.SetClipboard("by Insert")
	tt.Key(ui.Shift, ui.KeyInsert)
	if got := conn.take(term); got != "by Insert" {
		t.Errorf("Shift-Insert sent %q", got)
	}
	// A secondary click copies a selection, else pastes, and shows no menu.
	x, y := cellCenter(term, 1, 0)
	tt.SetClipboard("")
	tt.Command("selectAll")
	tt.RightClickAt(x, y)
	if got := tt.Clipboard(); got != "some text" {
		t.Errorf("a secondary click over a selection copied %q", got)
	}
	tt.RightClickAt(x, y)
	if got := conn.take(term); got != "some text" {
		t.Errorf("a secondary click without a selection sent %q", got)
	}
	if menu := tt.Menu(); menu != nil {
		t.Errorf("a secondary click showed the menu %q", menu)
	}
}

// TestUnboundKeys: keys bound to ActionNone reach the program, as do
// Control-C and -V outside Windows, and a secondary click shows the menu.
func TestUnboundKeys(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, Keybinds: []Keybind{
		{Mods: ui.Ctrl, Key: ui.KeyC}, {Mods: ui.Ctrl, Key: ui.KeyV},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("some text"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Command("selectAll")
	tt.SetClipboard("clipboard")
	tt.Key(ui.Ctrl, ui.KeyC)
	if got := conn.take(term); got != "\x03" {
		t.Errorf("Control-C sent %q", got)
	}
	tt.Key(ui.Ctrl, ui.KeyV)
	if got := conn.take(term); got != "\x16" {
		t.Errorf("Control-V sent %q", got)
	}
	if got := tt.Clipboard(); got != "clipboard" {
		t.Errorf("the clipboard became %q", got)
	}
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	if tt.Menu() == nil {
		t.Error("a secondary click showed no menu")
	}
}

// TestKeybinds: Options.Keybinds replace the defaults of their keys, in
// order, ActionNone unbinds, and NoDefaultKeybinds starts from none.
func TestKeybinds(t *testing.T) {
	copyKey := Keybind{Mods: ui.Ctrl | ui.Shift, Key: ui.KeyC, Action: ActionCopy}
	if runtime.GOOS == "darwin" {
		copyKey.Mods = ui.Super
	}
	if !slices.Contains(DefaultKeybinds(), copyKey) {
		t.Errorf("the defaults have no %v", copyKey)
	}
	win := defaultKeybinds("windows")
	for _, k := range []Keybind{
		{Mods: ui.Ctrl, Key: ui.KeyC, Action: ActionCopy, Performable: true},
		{Mods: ui.Ctrl, Key: ui.KeyV, Action: ActionPaste, Performable: true},
		{Mods: ui.Shift, Key: ui.KeyInsert, Action: ActionPaste},
	} {
		if !slices.Contains(win, k) {
			t.Errorf("Windows' defaults have no %v", k)
		}
	}
	if slices.ContainsFunc(defaultKeybinds("linux"), func(k Keybind) bool { return k.Mods == ui.Ctrl && k.Key == ui.KeyC }) {
		t.Error("Linux binds Control-C")
	}
	opts := Options{Keybinds: []Keybind{
		{Mods: copyKey.Mods, Key: ui.KeyC, Action: ActionPaste},
		{Mods: ui.Alt, Key: ui.KeyC, Action: ActionCopy},
		{Mods: ui.Alt, Key: ui.KeyC, Action: ActionSelectAll},
		{Mods: copyKey.Mods, Key: ui.KeyA},
	}}
	b := keybinds(&opts)
	term := &Terminal{binds: b}
	if k, ok := term.binding(copyKey.Mods, ui.KeyC); !ok || k.Action != ActionPaste {
		t.Errorf("the copy key is bound to %v, %v", k, ok)
	}
	if k, _ := term.binding(ui.Alt, ui.KeyC); k.Action != ActionSelectAll {
		t.Errorf("a key bound twice does %v", k.Action)
	}
	if _, ok := term.binding(copyKey.Mods, ui.KeyA); ok {
		t.Error("ActionNone left the key bound")
	}
	if len(b) != len(DefaultKeybinds()) {
		t.Errorf("%d bindings, want %d", len(b), len(DefaultKeybinds()))
	}
	opts.NoDefaultKeybinds = true
	if b := keybinds(&opts); len(b) != 2 {
		t.Errorf("without the defaults, bound %v", b)
	}
}

// TestKeybindActions: the defaults select all, clear the screen above the
// prompt, scroll the scrollback but for a full-screen program, and show in
// the context menu.
func TestKeybindActions(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, Keybinds: []Keybind{
		{Mods: ui.Alt, Key: ui.KeyK, Action: ActionClearScreen, Performable: true},
		{Mods: ui.Alt, Key: ui.KeyA, Action: ActionSelectAll},
		{Mods: ui.Alt, Key: ui.KeyX, Action: ActionIgnore},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	b.WriteString("$ ls")
	term.Feed([]byte(b.String()))
	tt.Frame()
	offset := func() uint64 {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.term.Scrollbar().Offset
	}
	bottom := offset()
	tt.Key(ui.Shift, ui.KeyPageUp)
	if offset() >= bottom {
		t.Error("Shift-Page Up did not scroll")
	}
	tt.Key(ui.Shift, ui.KeyPageDown)
	if offset() != bottom {
		t.Error("Shift-Page Down did not scroll back")
	}
	tt.Key(ui.Alt, ui.KeyX)
	if got := conn.take(term); got != "" {
		t.Errorf("an ignored key sent %q", got)
	}
	tt.Key(ui.Alt, ui.KeyA)
	tt.Command("copy")
	if got := tt.Clipboard(); !strings.HasPrefix(got, "line 000\n") {
		t.Errorf("Select All copied %.20q", got)
	}
	tt.Key(ui.Alt, ui.KeyK)
	tt.Frame()
	if got := strings.TrimSpace(term.Text()); got != "$ ls" {
		t.Errorf("clearing left %q", got)
	}
	if x, y := term.term.Cursor(); x != 4 || y != 0 {
		t.Errorf("the cursor is at %d,%d after clearing, want 4,0", x, y)
	}
	// A full-screen program gets the keys.
	term.Feed([]byte("\x1b[?1049h"))
	tt.Frame()
	tt.Key(ui.Shift, ui.KeyPageUp)
	if got := conn.take(term); got != "\x1b[5;2~" {
		t.Errorf("Shift-Page Up on the alternate screen sent %q", got)
	}
	tt.Key(ui.Alt, ui.KeyK)
	if got := conn.take(term); got == "" {
		t.Error("a performable clear took the key on the alternate screen")
	}
	// The context menu shows the keys.
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	if got := strings.Join(tt.Menu(), ","); !strings.Contains(got, "Copy") {
		t.Errorf("context menu = %q", got)
	}
}

// TestOnClear: a clear OnClear takes clears nothing here, for a server's
// terminal to clear (TestMirrorClearScreen); one it leaves clears as
// before; and a full-screen program's screen neither asks nor clears.
func TestOnClear(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var asked []Clear
	take := true
	term, err := New(Options{Conn: conn, Keybinds: []Keybind{
		{Mods: ui.Alt, Key: ui.KeyK, Action: ActionClearScreen, Performable: true},
	}, OnClear: func(c Clear) bool {
		asked = append(asked, c)
		return take
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	b.WriteString("$ ls")
	term.Feed([]byte(b.String()))
	tt.Frame()
	full := term.Text()
	tt.Key(ui.Alt, ui.KeyK)
	if len(asked) != 1 || asked[0] != ClearScreen {
		t.Fatalf("the clear keys asked %v", asked)
	}
	if got := conn.take(term); got != "" {
		t.Errorf("a clear taken sent %q", got)
	}
	if term.Text() != full {
		t.Error("a clear taken cleared")
	}
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	if err := tt.ChooseMenuItem("Clear"); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 2 || asked[1] != ClearScreen {
		t.Fatalf("the menu's Clear asked %v", asked)
	}
	if term.Text() != full {
		t.Error("the menu's Clear taken cleared")
	}
	take = false
	tt.Key(ui.Alt, ui.KeyK)
	tt.Frame()
	if got := strings.TrimSpace(term.Text()); got != "$ ls" {
		t.Errorf("a clear left to the terminal left %q", got)
	}
	// The menu's Clear clears as the keys do.
	term.Feed([]byte("\r\n" + b.String()))
	tt.Frame()
	tt.RightClickAt(x, y)
	if err := tt.ChooseMenuItem("Clear"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if got := strings.TrimSpace(term.Text()); got != "$ ls" {
		t.Errorf("the menu's Clear left to the terminal left %q", got)
	}
	term.Feed([]byte("\x1b[?1049h"))
	tt.Frame()
	tt.Key(ui.Alt, ui.KeyK)
	if len(asked) != 4 {
		t.Errorf("the alternate screen asked %v", asked[4:])
	}
	tt.RightClickAt(x, y)
	if err := tt.ChooseMenuItem("Clear"); err == nil {
		t.Error("the menu's Clear is enabled on the alternate screen")
	}
	tt.CloseMenu()
	if got := conn.take(term); got == "" {
		t.Error("a performable clear took the key on the alternate screen")
	}
	if term.ClearScreen() {
		t.Error("ClearScreen cleared the alternate screen")
	}
}

func TestAppearance(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	bg := func() [3]uint8 {
		c := tt.Image().RGBAAt(200, 190)
		return [3]uint8{c.R, c.G, c.B}
	}
	light := bg()
	tt.SetDark(true)
	if dark := bg(); dark == light || dark[0] > 64 {
		t.Errorf("the background did not follow the dark appearance: %v, then %v", light, dark)
	}
	cols, _ := term.Size()
	term.SetFont(Font{Size: 26})
	tt.Frame()
	if c2, _ := term.Size(); c2 >= cols {
		t.Errorf("a bigger font left %d columns, from %d", c2, cols)
	}
	// A closed terminal shows nothing, and takes no input.
	term.Close()
	tt.Frame()
	tt.Type("x")
	if term.Text() != "" {
		t.Error("a closed terminal has text")
	}
}

func TestThemes(t *testing.T) {
	loadLib(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	latte, err := GhosttyTheme("Catppuccin Latte")
	if err != nil {
		t.Fatal(err)
	}
	mocha, _ := GhosttyTheme("Catppuccin Mocha")
	term, err := New(Options{Conn: newPipe(), Theme: latte, DarkTheme: mocha, NoBlink: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	// A block under the cursor, and two below it, before a space.
	term.Feed([]byte("\x1b[2;1H██ x\x1b[1;1H█\x1b[D"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	at := func(x, y float32) ui.Color {
		s := term.v.scale
		c := tt.Image().RGBAAt(int(x*s), int(y*s))
		return ui.RGB(c.R, c.G, c.B)
	}
	cell := func(col, row int) ui.Color { return at(cellCenter(term, col, row)) }
	near := func(name string, got, want ui.Color) {
		t.Helper()
		d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
		if d(got.R, want.R) > 2 || d(got.G, want.G) > 2 || d(got.B, want.B) > 2 {
			t.Errorf("%s is %v, want %v", name, got, want)
		}
	}
	near("the light background", at(200, 190), latte.Background)
	tt.SetDark(true)
	near("the dark background", at(200, 190), mocha.Background)

	th := &Theme{
		Foreground: ui.Hex("#c0c0c0"), Background: ui.Hex("#101010"),
		Cursor: ui.Hex("#ff0000"), CursorText: ui.Hex("#00ff00"),
		Selection: ui.Hex("#0000ff"), SelectionText: ui.Hex("#ffff00"),
	}
	term.SetTheme(th, nil)
	tt.Frame()
	near("the background", at(200, 190), th.Background)
	near("the text", cell(0, 1), th.Foreground)
	near("the text under the cursor", cell(0, 0), th.CursorText)
	tt.Command("selectAll")
	tt.Frame()
	near("the selected text", cell(0, 1), th.SelectionText)
	near("the selection", cell(2, 1), th.Selection)
	near("the selected text under the cursor", cell(0, 0), th.CursorText)
}

func TestFont(t *testing.T) {
	loadLib(t)
	for _, f := range [][]byte{gomono.TTF, gomonobold.TTF} {
		if err := ui.RegisterFont(f, "MyGo Test Mono"); err != nil {
			t.Fatal(err)
		}
	}
	// render returns the size of a cell in a font, and the ink of a row of
	// dark text on white.
	render := func(f Font) (w, h int, ink int) {
		f.Family, f.Size = "MyGo Test Mono", 14
		term, err := New(Options{Conn: newPipe(), Font: f, Theme: LightTheme(), NoBlink: true})
		if err != nil {
			t.Fatal(err)
		}
		defer term.Close()
		term.Feed([]byte("MMMMMMMMMM\r\n"))
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 120)
		tt.SetScale(2)
		v := term.v
		img := tt.Image()
		for y := v.oy; y < v.oy+v.cellH; y++ {
			for x := v.ox; x < v.ox+10*v.cellW; x++ {
				ink += 255 - int(img.RGBAAt(x, y).G)
			}
		}
		return v.cellW, v.cellH, ink
	}
	w, h, ink := render(Font{})
	if w2, h2, _ := render(Font{LineHeight: 1.5}); w2 != w || h2 < h*3/2-1 || h2 > h*3/2+1 {
		t.Errorf("rows 1.5 times as high are %dx%d, from %dx%d", w2, h2, w, h)
	}
	if w2, h2, spaced := render(Font{CellWidth: 1.5}); w2 < w*3/2-1 || w2 > w*3/2+1 || h2 != h {
		t.Errorf("columns 1.5 times as wide are %dx%d, from %dx%d", w2, h2, w, h)
	} else if spaced < ink*95/100 || spaced > ink*105/100 {
		t.Errorf("spacing changed glyph ink from %d to %d", ink, spaced)
	}
	if _, _, bold := render(Font{Weight: 700}); bold < ink*11/10 {
		t.Errorf("weight 700 has ink %d, 400 %d", bold, ink)
	}
	_, _, thick := render(Font{Thicken: true})
	if runtime.GOOS == "darwin" && thick < ink*103/100 {
		t.Errorf("thickened text has ink %d, else %d", thick, ink)
	} else if runtime.GOOS != "darwin" && thick != ink {
		t.Errorf("thickening changed text off macOS: ink %d, else %d", thick, ink)
	}
	if got := featureList([]string{"-calt", "+ss01", " cv05=2 ", "liga=0", ""}); got != "calt=0,ss01,cv05=2,liga=0" {
		t.Errorf("features are %q", got)
	}
	k := (&Font{Weight: 300, Features: []string{"-calt"}}).key(1)
	features := "calt=0"
	if runtime.GOOS == "darwin" {
		features = "kern=0,calt=0"
	}
	if f := k.variant(1); f.Weight != 700 || f.Features != features {
		t.Errorf("the bold of weight 300 is %+v", f)
	}
	// Kerning is off unless the features name it, on macOS.
	for in, want := range map[string]string{"": "kern=0", "calt=0": "kern=0,calt=0", "kern": "kern", "kern=1": "kern=1", " kern =0": " kern =0"} {
		if got := unkerned(in); got != want {
			t.Errorf("unkerned(%q) = %q, want %q", in, got, want)
		}
	}
	if f := (&Font{Weight: 800}).key(1).variant(3); f.Weight != 900 || !f.Italic {
		t.Errorf("the bold italic of weight 800 is %+v", f)
	}
}

// TestCJKPunctuationKeepsItsPlace checks that a wide character sits at
// the same place in its cells wherever it is in a run. Core Text shrank
// the CJK punctuation of the system's monospaced font that ends a run, as
// a 。 being typed does, to half its width, and the first of two in a
// row: centered in its two cells, it jumped when text followed. Each
// character is checked against itself, so that it holds whatever fonts
// the system falls back to.
func TestCJKPunctuationKeepsItsPlace(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Core Text's kerning")
	}
	loadLib(t)
	type place struct{ off, adv float32 }
	for _, size := range []float32{12, 13} {
		seen := map[rune]place{}
		for _, in := range []string{"清理。", "清理。地方", "清理。\x1b[31m地方", "清理。。地", "日本語。", "日本語。地"} {
			term, err := New(Options{Conn: newPipe(), Font: Font{Size: size}, NoBlink: true})
			if err != nil {
				t.Fatal(err)
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 120)
			term.Feed([]byte(in))
			tt.Frame()
			term.mu.Lock()
			v := term.v
			cw := float32(v.cellW) / v.scale
			var got []place
			line := v.lines[0]
			for _, r := range line.runs {
				for i, g := range r.glyphs {
					if slices.Contains(line.wide, r.cols[i]) {
						got = append(got, place{g.X - float32(r.cols[i])*cw, g.Advance})
					}
				}
			}
			term.mu.Unlock()
			term.Close()
			chars := []rune(strings.ReplaceAll(in, "\x1b[31m", ""))
			if len(got) != len(chars) {
				t.Errorf("size %v, %q: %d wide glyphs, want %d", size, in, len(got), len(chars))
				continue
			}
			for i, r := range chars {
				p, ok := seen[r]
				if !ok {
					seen[r] = got[i]
				} else if math.Abs(float64(got[i].off-p.off)) > 0.01 || math.Abs(float64(got[i].adv-p.adv)) > 0.01 {
					t.Errorf("size %v, %q: %c %d at offset %v, advance %v, elsewhere %v, %v", size, in, r, i, got[i].off, got[i].adv, p.off, p.adv)
				}
			}
		}
	}
}

func TestWindowsShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cmd.exe")
	}
	loadLib(t)
	term, err := New(Options{Command: []string{"cmd.exe", "/c", "echo hello from cmd"}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 640, 300)
	select {
	case <-term.Done():
	case <-time.After(time.Minute): // a hang guard
		t.Fatal("cmd did not exit")
	}
	tt.Frame()
	if !strings.Contains(term.Text(), "hello from cmd") {
		t.Errorf("text = %q", term.Text())
	}
}

func TestStartsAtTheViewsSize(t *testing.T) {
	loadLib(t)
	term := shell(t, Options{Command: []string{"/bin/sh", "-c", "stty size; sleep 5"}})
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 300, 200)
	cols, rows := term.Size()
	want := fmt.Sprintf("%d %d", rows, cols)
	waitFor(t, tt, "stty", func() bool { return strings.Contains(term.Text(), " ") })
	if got := strings.TrimSpace(term.Text()); got != want {
		t.Errorf("the program saw %q, not the view's %q", got, want)
	}
}

func TestBlurEndsSelecting(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 4, 0)
	tt.Press(x0-2, y0)
	tt.Move(x1+2, y1)
	tt.SetFocused(false)
	if term.v.selecting {
		t.Error("still selecting after the window lost the keyboard")
	}
	tt.SetFocused(true)
	x2, _ := cellCenter(term, 9, 0)
	tt.Move(x2, y1)
	tt.Command("copy")
	if got := tt.Clipboard(); got != "hello" {
		t.Errorf("the selection followed the pointer with no button held: %q", got)
	}
}

func TestKeyWithoutText(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	// A key that types text, whose text never comes, goes alone when
	// released.
	tt.Key(0, ui.KeyB)
	if got := conn.take(term); got != "b" {
		t.Errorf("sent %q", got)
	}
}

// TestOptionAsAltDropsText: with Options.OptionAsAlt on macOS, Option
// and a letter go as Alt (Esc and the letter), and what macOS still types
// for the key, or composes for a dead key as ´ for Option-E, is dropped
// (#1000). The composition is refused, which has macOS give it up rather
// than compose the next key with it. The key's release goes as any key's.
func TestOptionAsAltDropsText(t *testing.T) {
	loadLib(t)
	defer func(was string) { goos = was }(goos)
	goos = "darwin"
	conn := newPipe()
	term, err := New(Options{Conn: conn, OptionAsAlt: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	v := term.v
	// press presses key with mods as macOS does, its text coming from
	// interpretKeyEvents: typed as text, or composed for a dead key until
	// the next key, which macOS then composes with it unless the
	// composition was refused.
	composing := false
	press := func(mods ui.Modifiers, key ui.Key, text string, dead bool) {
		t.Helper()
		if composing {
			// Composed with the dead key: no key, the composition's end,
			// then both characters.
			composing = false
			v.input(ui.InputEvent{Kind: ui.InputCompose})
			v.input(ui.InputEvent{Kind: ui.InputText, Text: "´" + text})
		} else {
			v.input(ui.InputEvent{Kind: ui.InputKeyDown, Mods: mods, Key: key})
			if dead {
				composing = v.input(ui.InputEvent{Kind: ui.InputCompose, Text: text, Caret: 1})
			} else {
				v.input(ui.InputEvent{Kind: ui.InputText, Text: text})
			}
		}
		v.input(ui.InputEvent{Kind: ui.InputKeyUp, Mods: mods, Key: key})
	}
	press(ui.Alt, ui.KeyE, "´", true)
	press(0, ui.KeyX, "x", false)
	if got, want := conn.take(term), "\x1bex"; got != want {
		t.Errorf("Option-E and x sent %q, want %q", got, want)
	}
	// Option-X types ≈ on a US layout, as text.
	press(ui.Alt, ui.KeyX, "≈", false)
	if got, want := conn.take(term), "\x1bx"; got != want {
		t.Errorf("Option-X sent %q, want %q", got, want)
	}
	// Programs that ask for the Kitty keyboard protocol's releases get
	// them.
	term.Feed([]byte("\x1b[>11u"))
	press(ui.Alt, ui.KeyE, "´", true)
	if got, want := conn.take(term), "\x1b[101;3u\x1b[101;3:3u"; got != want {
		t.Errorf("Option-E with the Kitty keyboard protocol sent %q, want %q", got, want)
	}
}

// TestAltGrTypesText: on Windows, where AltGr is Control and Alt, the
// characters AltGr types on a German layout go as text, not as Control
// sequences of them (#1203). Control, Alt and a key that types nothing
// still go as the key with both.
func TestAltGrTypesText(t *testing.T) {
	loadLib(t)
	defer func(was string) { goos = was }(goos)
	goos = "windows"
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	v := term.v
	// press presses key with mods as Windows does, the text it types, if
	// any, coming as WM_CHAR between the key's press and its release.
	press := func(mods ui.Modifiers, key ui.Key, text string) {
		t.Helper()
		v.input(ui.InputEvent{Kind: ui.InputKeyDown, Mods: mods, Key: key})
		if text != "" {
			v.input(ui.InputEvent{Kind: ui.InputText, Text: text})
		}
		v.input(ui.InputEvent{Kind: ui.InputKeyUp, Mods: mods, Key: key})
	}
	altGr := ui.Ctrl | ui.Alt
	for _, c := range []struct {
		name string
		key  ui.Key
		text string
	}{
		{"AltGr+Q", ui.KeyQ, "@"},
		{"AltGr+E", ui.KeyE, "€"},
		{"AltGr+7", ui.Key7, "{"},
		{"AltGr+ß", ui.KeyMinus, `\`},
	} {
		press(altGr, c.key, c.text)
		if got := conn.take(term); got != c.text {
			t.Errorf("%s sent %q, want %q", c.name, got, c.text)
		}
	}
	press(altGr, ui.KeyA, "")
	if got, want := conn.take(term), "\x1b\x01"; got != want {
		t.Errorf("Control, Alt and A sent %q, want %q", got, want)
	}
	// The Kitty keyboard protocol's disambiguation.
	term.Feed([]byte("\x1b[>1u"))
	press(altGr, ui.KeyQ, "@")
	if got, want := conn.take(term), "@"; got != want {
		t.Errorf("AltGr+Q with the Kitty keyboard protocol sent %q, want %q", got, want)
	}
	press(altGr, ui.KeyA, "")
	if got, want := conn.take(term), "\x1b[97;7u"; got != want {
		t.Errorf("Control, Alt and A with the Kitty keyboard protocol sent %q, want %q", got, want)
	}
}

func TestCloseKillsWhatIgnoresHangups(t *testing.T) {
	loadLib(t)
	term := shell(t, Options{Command: []string{"/bin/sh", "-c", "trap '' HUP; echo ready; sleep 30"}})
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 300, 200)
	waitFor(t, tt, "the program", func() bool { return strings.Contains(term.Text(), "ready") })
	term.Close()
	select {
	case <-term.Done():
	case <-time.After(time.Minute): // a hang guard: Close kills it after 2 s
		t.Fatal("Done was not closed: the program outlived Close")
	}
}

// TestCommandClickLinks: Command+click hands URLs and printed paths to
// Options.OpenLink, and URLs it leaves to the system.
func TestCommandClickLinks(t *testing.T) {
	loadLib(t)
	var opened []string
	term, err := New(Options{Conn: newPipe(), OpenLink: func(link string) bool {
		opened = append(opened, link)
		return link != "https://x.org"
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("see src/app.go:12 or https://x.org\r\nplain words"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	for _, col := range []int{6, 24} {
		x, y := cellCenter(term, col, 0)
		tt.ClickAtWith(ui.Cmd, x, y)
	}
	x, y := cellCenter(term, 2, 1)
	tt.ClickAtWith(ui.Cmd, x, y)
	if want := []string{"src/app.go:12", "https://x.org"}; !slices.Equal(opened, want) {
		t.Errorf("opened %q, want %q", opened, want)
	}
	if u := tt.OpenedURLs(); !slices.Equal(u, []string{"https://x.org"}) {
		t.Errorf("system opened %q", u)
	}
}

// Command-hover underlines the same cells that Command-click opens,
// including bare filenames and paths to ignored output directories.
func TestCommandHoverFilePath(t *testing.T) {
	loadLib(t)
	for _, path := range []string{"notes.md", ".log-outputs/takumi-session-analysis.md"} {
		t.Run(path, func(t *testing.T) {
			term, err := New(Options{Conn: newPipe(), OpenLink: func(string) bool { return true }})
			if err != nil {
				t.Fatal(err)
			}
			defer term.Close()
			term.Feed([]byte("write(" + path + ")\r\n"))
			tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 800, 200)
			tt.Frame()
			x, y := cellCenter(term, 8, 0)
			tt.Move(x, y)
			plain := slices.Clone(tt.Image().Pix)
			tt.HoldModifiers(ui.Cmd)
			if h := term.v.hover; h != (linkSpan{0, 6, 6 + len(path)}) {
				t.Fatalf("hovered %+v, want the file path's cells", h)
			}
			if tt.Cursor() != ui.CursorPointer {
				t.Error("hovering a file path with Command did not show a hand")
			}
			if bytes.Equal(plain, tt.Image().Pix) {
				t.Error("hovering a file path with Command did not paint its underline")
			}
			tt.HoldModifiers(0)
			if term.v.hover.on() || !bytes.Equal(plain, tt.Image().Pix) {
				t.Error("releasing Command did not remove the underline")
			}
		})
	}
}

// TestCommandClickLinkWithoutOpenLink: without Options.OpenLink, the system
// opens hyperlinks of any scheme, paths are no links, the pointer is a hand
// over a link while Command is held, and a click on a link goes before a
// program that takes the mouse, its release too.
func TestCommandClickLinkWithoutOpenLink(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("\x1b]8;;vscode://file/a.go\x1b\\edit\x1b]8;;\x1b\\ src/app.go:12"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	x, y := cellCenter(term, 1, 0)
	tt.Move(x, y)
	if c := tt.Cursor(); c != ui.CursorText {
		t.Errorf("the cursor is %v over a link without Command", c)
	}
	// Command pressed and let go of with the pointer still.
	tt.HoldModifiers(ui.Cmd)
	if c := tt.Cursor(); c != ui.CursorPointer {
		t.Errorf("the cursor is %v over a link with Command", c)
	}
	// The link under it shows underlined, all its cells.
	if h := term.v.hover; h != (linkSpan{0, 0, 4}) {
		t.Errorf("hovered %+v, want the hyperlink's cells 0..4", h)
	}
	x0, y0 := cellCenter(term, 0, 0)
	x4, _ := cellCenter(term, 4, 0)
	if link, path, at := term.HoveredLink(); link != "vscode://file/a.go" || path || !at.Contains(x0, y0) || at.Contains(x4, y0) {
		t.Errorf("HoveredLink is %q, %v, %+v, want the hyperlink over cells 0..4", link, path, at)
	}
	tt.HoldModifiers(0)
	if c := tt.Cursor(); c != ui.CursorText {
		t.Errorf("the cursor is %v once Command is let go of", c)
	}
	if h := term.v.hover; h.on() {
		t.Errorf("hovered %+v once Command is let go of", h)
	}
	if link, _, _ := term.HoveredLink(); link != "" {
		t.Errorf("HoveredLink is %q once Command is let go of", link)
	}
	term.Feed([]byte("\x1b[?1002h\x1b[?1006h"))
	tt.Frame()
	tt.ClickAtWith(ui.Cmd, x, y)
	x2, y2 := cellCenter(term, 8, 0)
	tt.ClickAtWith(ui.Cmd, x2, y2)
	if u := tt.OpenedURLs(); !slices.Equal(u, []string{"vscode://file/a.go"}) {
		t.Errorf("system opened %q", u)
	}
	// The click on the path, no link without OpenLink, went to the
	// program; the one on the link did not, nor its release.
	got := conn.take(term) // two reports, 22 bytes with Control
	if strings.Count(got, "\x1b[<") != 2 || strings.Count(got, ";9;1M") != 1 || strings.Count(got, ";9;1m") != 1 {
		t.Errorf("sent %q, want the press and the release of the path's click", got)
	}
}

// TestOnPasteTakesAPaste: Options.OnPaste sees a paste and the clipboard's
// text, and a true result pastes nothing.
func TestOnPasteTakesAPaste(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var seen []string
	take := true
	term, err := New(Options{Conn: conn, OnPaste: func(d transfer.Data) bool {
		text, _ := d.Read(transfer.Text)
		png, _ := d.Read(transfer.PNG)
		seen = append(seen, string(text)+string(png))
		return take
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.SetClipboard("hello")
	tt.Frame()
	paste := ui.Super
	if runtime.GOOS != "darwin" {
		paste = ui.Ctrl | ui.Shift
	}
	tt.Key(paste, ui.KeyV)
	tt.Command("paste") // the Edit menu's Paste
	tt.Frame()
	if !slices.Equal(seen, []string{"hello", "hello"}) {
		t.Fatalf("OnPaste saw %q", seen)
	}
	if got := conn.take(term); len(got) != 0 {
		t.Errorf("a taken paste sent %q", got)
	}
	take = false
	tt.Key(paste, ui.KeyV)
	tt.Frame()
	if got := conn.take(term); got != "hello" || len(seen) != 3 {
		t.Errorf("a paste OnPaste leaves sent %q (seen %q)", got, seen)
	}
	// It sees every format, as an image.
	take = true
	tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, []byte("PNG")))))
	tt.Key(paste, ui.KeyV)
	if seen[len(seen)-1] != "PNG" {
		t.Errorf("OnPaste saw %q of an image", seen[len(seen)-1])
	}
}

// TestOnKeyTakesAKey: Options.OnKey sees a key press before the terminal
// encodes it, in order with the keys after it; a true result sends
// nothing, not even the text the key types.
// A modifier on its own is no key to the terminal: nothing goes to the shell
// or to OnKey, and Alt is left to the menu bar.
func TestModifierKeyAlone(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var seen []ui.Key
	term, err := New(Options{Conn: conn, OnKey: func(mods ui.Modifiers, key ui.Key) bool {
		seen = append(seen, key)
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	tt.CodeDown(ui.Shift, ui.CodeShiftLeft)
	tt.CodeUp(0, ui.CodeShiftLeft)
	tt.Frame()
	if len(seen) != 0 {
		t.Errorf("OnKey saw %v", seen)
	}
	if got := conn.take(term); got != "" {
		t.Errorf("the shell got %q", got)
	}
}

func TestOnKeyTakesAKey(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var seen []ui.Key
	var term *Terminal
	term, err := New(Options{Conn: conn, OnKey: func(mods ui.Modifiers, key ui.Key) bool {
		seen = append(seen, key)
		if mods == ui.Ctrl && key == ui.KeyV || key == ui.KeyX {
			term.Send([]byte("<taken>")) // in order with what is typed
			return true
		}
		return false
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	tt.Key(ui.Ctrl, ui.KeyV)
	tt.Key(0, ui.KeyTab)
	tt.TypeKey(0, ui.KeyX, "x")
	tt.TypeKey(0, ui.KeyY, "y")
	tt.Frame()
	if got := conn.take(term); got != "<taken>\t<taken>y" {
		t.Errorf("sent %q, want the taken keys' marks in order with Tab and y", got)
	}
	if !slices.Equal(seen, []ui.Key{ui.KeyV, ui.KeyTab, ui.KeyX, ui.KeyY}) {
		t.Errorf("OnKey saw %v", seen)
	}
}

// TestOnScrollTakesTheWheel: Options.OnScroll sees the wheel's whole rows
// before the scrollback moves, unless a program takes the wheel: as the
// mouse, or as the arrows of a full-screen program.
func TestOnScrollTakesTheWheel(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var seen []int
	term, err := New(Options{Conn: conn, OnScroll: func(rows int) bool {
		seen = append(seen, rows)
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	offset := func() uint64 {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.term.Scrollbar().Offset
	}
	bottom := offset()
	tt.Wheel(100, 100, 0, -hostNotch()) // a notch up
	tt.Wheel(100, 100, 0, hostNotch())  // and down
	if !slices.Equal(seen, []int{-3, 3}) {
		t.Errorf("OnScroll saw %v, want [-3 3]", seen)
	}
	if got := offset(); got != bottom {
		t.Errorf("a taken scroll moved the scrollback: offset = %d, want %d", got, bottom)
	}
	// The program takes the mouse: the wheel goes to it.
	term.Feed([]byte("\x1b[?1002h\x1b[?1006h"))
	tt.Frame()
	tt.Wheel(100, 100, 0, -hostNotch())
	if len(seen) != 2 {
		t.Errorf("OnScroll saw %v while the program took the mouse", seen)
	}
	if got := conn.take(term); !strings.HasPrefix(got, "\x1b[<64;") {
		t.Errorf("wheel sent %q, want a mouse report", got)
	}
	// A full-screen program gets it as arrows, unless it turns that off.
	term.Feed([]byte("\x1b[?1002l\x1b[?1006l\x1b[?1049h"))
	tt.Frame()
	tt.Wheel(100, 100, 0, -hostNotch())
	if len(seen) != 2 {
		t.Errorf("OnScroll saw %v on a full-screen program's screen", seen)
	}
	if got := conn.take(term); strings.Count(got, "A") != 3 {
		t.Errorf("wheel sent %q, want three arrows up", got)
	}
	term.Feed([]byte("\x1b[?1007l"))
	tt.Frame()
	tt.Wheel(100, 100, 0, -hostNotch())
	if !slices.Equal(seen, []int{-3, 3, -3}) {
		t.Errorf("OnScroll saw %v without alternate scroll", seen)
	}
}

// A program that reads the clipboard itself (mode 5522) is told what a
// paste offers and reads the image with the event's password; a
// terminal that did not paste, as another window of the program, answers
// nothing; without the mode the text is pasted, bracketed.
func TestPasteImage(t *testing.T) {
	loadLib(t)
	image := bytes.Repeat([]byte("\x89PNG\r\n\x1a\nxyz"), 1000)
	conn := newPipe()
	var mu sync.Mutex
	var statuses []string
	var term *Terminal
	var err error
	var last PasteProgress
	term, err = New(Options{Conn: conn, OnPasteStatus: func(p PasteProgress) {
		term.Title() // the terminal is not locked
		mu.Lock()
		statuses = append(statuses, fmt.Sprint(p.Status, p.Sent == p.Total && p.Total > 0))
		if p.Status == PasteSent {
			last = p
		}
		mu.Unlock()
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.Text, []byte("some text")), transfer.Bytes(transfer.PNG, image))))
	pasteMods := ui.Ctrl | ui.Shift
	if runtime.GOOS == "darwin" {
		pasteMods = ui.Super
	}

	term.Feed([]byte("\x1b[?5522h"))
	tt.Key(pasteMods, ui.KeyV)
	event := conn.take(term)
	_, after, ok := strings.Cut(event, ":pw=")
	if !ok || !strings.HasSuffix(event, "\x1b\\") {
		t.Fatalf("paste sent %q, want a paste event", event)
	}
	pw := after[:strings.Index(after, "\x1b")]
	listing, _ := base64.StdEncoding.DecodeString(event[strings.Index(event, "Lg==:pw="+pw+";")+len("Lg==:pw="+pw+";") : strings.Index(event, "\x1b\\\x1b]5522;type=read:status=DONE")])
	if string(listing) != "text/plain image/png\n" {
		t.Errorf("offered %q", listing)
	}

	name := base64.StdEncoding.EncodeToString([]byte("Paste event"))
	mime := base64.StdEncoding.EncodeToString([]byte("image/png"))
	read := "\x1b]5522;type=read:pw=" + pw + ":name=" + name + ";" + mime + "\x1b\\"
	term.Feed([]byte(read))
	reply := conn.take(term)
	var got []byte
	for _, pkt := range strings.Split(reply, "\x1b\\") {
		if _, data, ok := strings.Cut(pkt, "status=DATA:mime="+mime+";"); ok {
			b, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, b...)
		}
	}
	if !bytes.Equal(got, image) || !strings.HasSuffix(reply, "status=DONE\x1b\\") {
		t.Errorf("the program read %d bytes of %d; reply ends %q", len(got), len(image), reply[max(0, len(reply)-40):])
	}
	// The statuses of the pieces ran on the queue, before take's marker.
	mu.Lock()
	if last.Total != int64(len(reply)) {
		t.Errorf("sent %d bytes of %d, but the reply is %d", last.Sent, last.Total, len(reply))
	}
	mu.Unlock()

	// Another terminal of the same program (a mirror) sees the read too,
	// and must not answer it.
	conn2 := newPipe()
	other, err := New(Options{Conn: conn2})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	other.Feed([]byte("\x1b[?5522h" + read))
	if sent := conn2.take(other); sent != "" {
		t.Errorf("a terminal that did not paste answered %q", sent)
	}

	// Without the mode, the text, bracketed when asked.
	term.Feed([]byte("\x1b[?5522l\x1b[?2004h"))
	tt.Key(pasteMods, ui.KeyV)
	if sent := conn.take(term); sent != "\x1b[200~some text\x1b[201~" {
		t.Errorf("without the mode, pasted %q", sent)
	}
	// Only an image, to a program that takes bracketed pastes: an empty
	// one, which it reads as a paste of the image, as Claude Code does.
	tt.SetClipboardData(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, image))))
	tt.Key(pasteMods, ui.KeyV)
	if sent := conn.take(term); sent != "\x1b[200~\x1b[201~" {
		t.Errorf("with only an image and bracketed paste, pasted %q", sent)
	}
	// Only an image, and no program to read it: nothing, and told so.
	term.Feed([]byte("\x1b[?2004l"))
	tt.Key(pasteMods, ui.KeyV)
	if sent := conn.take(term); sent != "" {
		t.Errorf("with only an image and no mode, pasted %q", sent)
	}

	mu.Lock()
	seen := strings.Join(statuses, " ")
	mu.Unlock()
	// Offered, sending from nothing, then pieces, the last of them all sent;
	// then the paste of no text.
	if !strings.HasPrefix(seen, fmt.Sprint(PasteOffered, false)+" "+fmt.Sprint(PasteSending, false)) ||
		!strings.Contains(seen, fmt.Sprint(PasteSent, true)) || !strings.HasSuffix(seen, fmt.Sprint(PasteNoText, false)) {
		t.Errorf("statuses %q", seen)
	}
}

// hostNotch is the DIPs a notch of a mouse wheel scrolls on this
// platform at its defaults, as its backend sends them.
func hostNotch() float32 {
	switch runtime.GOOS {
	case "windows":
		return 3 * 33 // 3 lines to scroll, 33 DIPs a line
	case "linux":
		return 50
	}
	return 40
}

// TestWheelRows: a notch of a mouse wheel scrolls three rows on every
// platform at its defaults, though their backends send it as 40, 50 or 99
// DIPs, and on Windows follows "lines to scroll": a notch at one line is a
// row (#783).
func TestWheelRows(t *testing.T) {
	for _, c := range []struct {
		goos string
		dy   float32
		want float32
	}{
		{"windows", 99, 3},   // a notch at 3 lines
		{"windows", -99, -3}, // and up
		{"windows", 33, 1},   // a notch at 1 line
		{"windows", 66, 2},   // two notches at 1 line, or one at 2
		{"windows", 165, 5},  // a notch at 5 lines
		{"darwin", 40, 3},
		{"darwin", -80, -6}, // a line accelerated to two
		{"linux", 50, 3},
		{"linux", -50, -3},
	} {
		if got := wheelRows(c.dy, c.goos); got != c.want {
			t.Errorf("wheelRows(%v, %q) = %v, want %v", c.dy, c.goos, got, c.want)
		}
	}
}

// TestWheelNotchScrollsThreeRows: the view scrolls three whole rows a
// notch of this platform's wheel, and as many again the next notch, with
// nothing left over.
func TestWheelNotchScrollsThreeRows(t *testing.T) {
	loadLib(t)
	var seen []int
	term, err := New(Options{Conn: newPipe(), OnScroll: func(rows int) bool {
		seen = append(seen, rows)
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	for _, dy := range []float32{hostNotch(), hostNotch(), -hostNotch()} {
		term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: dy})
	}
	if !slices.Equal(seen, []int{3, 3, -3}) {
		t.Errorf("notches scrolled %v rows, want [3 3 -3]", seen)
	}
}

// TestScrollMultiplier: Options.ScrollMultiplier's Discrete is the rows a
// notch scrolls, its Precision scales a touchpad's distance, and
// SetScrollMultiplier changes them in a running terminal, as Ghostty's
// mouse-scroll-multiplier.
func TestScrollMultiplier(t *testing.T) {
	loadLib(t)
	var seen []int
	term, err := New(Options{Conn: newPipe(), ScrollMultiplier: ScrollMultiplier{Discrete: 1}, OnScroll: func(rows int) bool {
		seen = append(seen, rows)
		return true
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	ch := float32(term.v.cellH) / term.v.scale
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: hostNotch()})
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -hostNotch()})
	term.SetScrollMultiplier(ScrollMultiplier{Precision: 0.5, Discrete: 6})
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: hostNotch()})
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: 4 * ch, Precise: true})
	if !slices.Equal(seen, []int{1, -1, 6, 2}) {
		t.Errorf("scrolled %v rows, want [1 -1 6 2]", seen)
	}
}

// TestScrollMultiplierRows: a fraction of a notch's rows carries over, and
// zero is the default.
func TestScrollMultiplierRows(t *testing.T) {
	for _, c := range []struct {
		m       ScrollMultiplier
		dy      float32
		precise bool
		goos    string
		want    float32
	}{
		{ScrollMultiplier{}, 40, false, "darwin", 3},
		{ScrollMultiplier{}, 20, true, "darwin", 1},
		{ScrollMultiplier{Discrete: 1.5}, 40, false, "darwin", 1.5},
		{ScrollMultiplier{Discrete: 1}, 50, false, "linux", 1},
		{ScrollMultiplier{Discrete: 6}, 165, false, "windows", 10}, // a notch at 5 lines, doubled
		{ScrollMultiplier{Precision: 0.5}, 40, true, "darwin", 1},
		{ScrollMultiplier{Precision: 2}, 40, false, "darwin", 3}, // a wheel's alone
	} {
		if got := c.m.rows(c.dy, 20, c.precise, c.goos); got != c.want {
			t.Errorf("%+v.rows(%v, precise %v, %q) = %v, want %v", c.m, c.dy, c.precise, c.goos, got, c.want)
		}
	}
}

// TestOnZoom: Options.OnZoom takes the wheel with Control (Command on
// macOS) in notches, rolling up positive, before OnScroll and a program
// that takes the mouse, and the font size keys.
func TestOnZoom(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var zooms []Zoom
	var scrolls []int
	term, err := New(Options{
		Conn:     conn,
		OnZoom:   func(z Zoom) bool { zooms = append(zooms, z); return true },
		OnScroll: func(rows int) bool { scrolls = append(scrolls, rows); return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	zoomMod := ui.Cmd
	wheel := func(mods ui.Modifiers, dy float32, precise bool) {
		term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: dy, Mods: mods, Precise: precise})
	}
	steps := func() []int {
		var s []int
		for _, z := range zooms {
			s = append(s, z.Steps)
		}
		return s
	}
	wheel(zoomMod, -40, false) // a notch up
	wheel(zoomMod, 40, false)  // and down
	wheel(zoomMod, -120, false)
	if !slices.Equal(steps(), []int{1, -1, 1}) {
		t.Errorf("notches zoomed %v, want [1 -1 1]: one step an event", steps())
	}
	// A trackpad's moves add up to notches of 40.
	zooms = nil
	for range 6 {
		wheel(zoomMod, -15, true)
	}
	if !slices.Equal(steps(), []int{1, 1}) {
		t.Errorf("a trackpad zoomed %v, want [1 1]", steps())
	}
	if scrolls != nil {
		t.Errorf("OnScroll saw %v while zooming", scrolls)
	}
	// Without the modifier the wheel scrolls.
	zooms = nil
	wheel(0, -hostNotch(), false)
	if zooms != nil || !slices.Equal(scrolls, []int{-3}) {
		t.Errorf("a plain notch zoomed %v and scrolled %v", zooms, scrolls)
	}
	// A program that takes the mouse gets no report of a zoom.
	term.Feed([]byte("\x1b[?1002h\x1b[?1006h"))
	tt.Frame()
	wheel(zoomMod, -40, false)
	if !slices.Equal(steps(), []int{1}) {
		t.Errorf("zoomed %v while the program took the mouse", steps())
	}
	if got := conn.take(term); got != "" {
		t.Errorf("a zoom sent %q to the program", got)
	}
	// The font size keys.
	zooms = nil
	tt.Key(ui.Cmd, ui.KeyEqual)
	tt.Key(ui.Cmd, ui.KeyMinus)
	tt.Key(ui.Cmd, ui.Key0)
	if !slices.Equal(zooms, []Zoom{{Steps: 1}, {Steps: -1}, {Reset: true}}) {
		t.Errorf("the keys zoomed %v", zooms)
	}
	if got := conn.take(term); got != "" {
		t.Errorf("the zoom keys sent %q to the program", got)
	}
	if term.v.font.size != 13 {
		t.Errorf("a zoom OnZoom took changed the font to %v", term.v.font.size)
	}
}

// A touchpad scroll that began without Command keeps scrolling, even after
// the fingers leave it and Command comes down while its momentum runs.
func TestOnZoomIgnoresMomentum(t *testing.T) {
	loadLib(t)
	var zooms []Zoom
	var scrolls []int
	term, err := New(Options{
		Conn:     newPipe(),
		OnZoom:   func(z Zoom) bool { zooms = append(zooms, z); return true },
		OnScroll: func(rows int) bool { scrolls = append(scrolls, rows); return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line " + itoa(i) + "\r\n")
	}
	term.Feed([]byte(b.String()))
	tt.Frame()
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Precise: true})
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Mods: ui.Cmd, Precise: true, Momentum: true})
	if zooms != nil {
		t.Errorf("momentum scrolling zoomed %v", zooms)
	}
	if len(scrolls) != 2 || scrolls[0] >= 0 || scrolls[1] >= 0 {
		t.Errorf("momentum scrolling scrolled %v, want two scrolls up", scrolls)
	}
}

// Command at the start of a touchpad scroll zooms through its momentum.
func TestOnZoomKeepsMomentumOfZoomGesture(t *testing.T) {
	loadLib(t)
	var zooms []Zoom
	term, err := New(Options{
		Conn:   newPipe(),
		OnZoom: func(z Zoom) bool { zooms = append(zooms, z); return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Mods: ui.Cmd, Precise: true})
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Mods: ui.Cmd, Precise: true, Momentum: true})
	if !slices.Equal(zooms, []Zoom{{Steps: 1}, {Steps: 1}}) {
		t.Errorf("momentum scrolling zoomed %v, want two steps up", zooms)
	}
}

// TestZoom: without OnZoom the terminal zooms its font a DIP a step, and
// back to its font's size, which SetFont changes.
func TestZoom(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), Font: Font{Size: 14}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	cols, _ := term.Size()
	size := func() float32 { tt.Frame(); return term.v.font.size }
	term.v.scroll(ui.InputEvent{Kind: ui.InputScroll, DY: -40, Mods: ui.Cmd})
	tt.Key(ui.Cmd, ui.KeyEqual)
	if got := size(); got != 16 {
		t.Errorf("zoomed in twice to %v, want 16", got)
	}
	if c, _ := term.Size(); c >= cols {
		t.Errorf("%d columns at a larger size, %d before", c, cols)
	}
	tt.Key(ui.Cmd, ui.KeyMinus)
	if got := size(); got != 15 {
		t.Errorf("zoomed out to %v, want 15", got)
	}
	tt.Key(ui.Cmd, ui.Key0)
	if got := size(); got != 14 {
		t.Errorf("reset to %v, want 14", got)
	}
	tt.Key(ui.Cmd, ui.KeyEqual)
	term.SetFont(Font{Size: 20})
	if got := size(); got != 20 {
		t.Errorf("SetFont after a zoom gave %v, want 20", got)
	}
	for range 30 {
		tt.Key(ui.Cmd, ui.KeyMinus)
	}
	if got := size(); got != 1 {
		t.Errorf("zoomed out to %v, want 1 at the least", got)
	}
}

// TestPasteData: a program that reads the clipboard is offered every
// format, a file list as text/uri-list, and reads a lazy one only when it
// asks; a paste it does not read is told after a moment, and still read.
func TestPasteData(t *testing.T) {
	loadLib(t)
	notTakenAfter = 50 * time.Millisecond
	defer func() { notTakenAfter = 5 * time.Second }()
	conn := newPipe()
	var mu sync.Mutex
	var statuses []PasteStatus
	notTaken := make(chan struct{})
	term, err := New(Options{Conn: conn, OnPasteStatus: func(p PasteProgress) {
		mu.Lock()
		statuses = append(statuses, p.Status)
		mu.Unlock()
		if p.Status == PasteNotTaken {
			close(notTaken)
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	read := 0
	path := filepath.Join(t.TempDir(), "a.png")
	files, err := transfer.FileData(path)
	if err != nil {
		t.Fatal(err)
	}
	// The file URL as FileData wrote it: Windows paths have a drive letter.
	uri, err := files.Items()[0].Read(transfer.FileList)
	if err != nil {
		t.Fatal(err)
	}
	d := transfer.New(append(files.Items(), transfer.NewItem(
		transfer.Lazy("image/jpeg", func() ([]byte, error) { read++; return []byte("JPEG"), nil }),
		transfer.Bytes(transfer.Text, []byte("a.png")),
	))...)
	term.Feed([]byte("\x1b[?5522h"))
	term.PasteData(d)
	event := conn.take(term)
	_, after, ok := strings.Cut(event, ":pw=")
	if !ok {
		t.Fatalf("paste sent %q, want a paste event", event)
	}
	pw := after[:strings.Index(after, "\x1b")]
	start := strings.Index(event, "Lg==:pw="+pw+";") + len("Lg==:pw="+pw+";")
	listing, _ := base64.StdEncoding.DecodeString(event[start : strings.Index(event[start:], "\x1b")+start])
	if string(listing) != "text/uri-list image/jpeg text/plain\n" {
		t.Errorf("offered %q", listing)
	}
	select {
	case <-notTaken:
	case <-time.After(time.Minute):
		t.Fatal("no PasteNotTaken")
	}
	if read != 0 {
		t.Error("the lazy format was read before the program asked")
	}
	mu.Lock()
	if !slices.Equal(statuses, []PasteStatus{PasteOffered, PasteNotTaken}) {
		t.Errorf("statuses %v before the read", statuses)
	}
	mu.Unlock()
	b64 := base64.StdEncoding.EncodeToString
	term.Feed([]byte("\x1b]5522;type=read:pw=" + pw + ":name=" + b64([]byte("Paste event")) + ";" + b64([]byte("image/jpeg text/uri-list")) + "\x1b\\"))
	reply := conn.take(term)
	if read != 1 || !strings.Contains(reply, b64([]byte("JPEG"))) || !strings.Contains(reply, b64(uri)) {
		t.Errorf("read %d times, replied %q", read, reply)
	}
}

// TestNoClipboardWrite: programs write to the clipboard (OSC 52) unless
// NoClipboardWrite.
func TestNoClipboardWrite(t *testing.T) {
	loadLib(t)
	for _, deny := range []bool{false, true} {
		term, err := New(Options{Conn: newPipe(), NoClipboardWrite: deny})
		if err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
		term.Feed([]byte("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("copied")) + "\x1b\\"))
		tt.Frame()
		want := "copied"
		if deny {
			want = ""
		}
		if got := tt.Clipboard(); got != want {
			t.Errorf("NoClipboardWrite %v: a program copied %q", deny, got)
		}
		term.Close()
	}
}

// TestClipboardReadAllow: with ClipboardRead ClipboardAllow the terminal
// answers a program's read of the clipboard (OSC 52) with it, from the
// view, on the main thread.
func TestClipboardReadAllow(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, ClipboardRead: ClipboardAllow})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.SetClipboard("hello")
	term.Feed([]byte("\x1b]52;c;?\x1b\\"))
	tt.Frame()
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("hello")) + "\x1b\\"
	if got := conn.take(term); got != want {
		t.Errorf("the program read %q, want %q", got, want)
	}
}

// TestClipboardReadDeny: reads are refused by default and with
// ClipboardDeny: an empty OSC 52 reply, without reading the clipboard.
func TestClipboardReadDeny(t *testing.T) {
	loadLib(t)
	for _, opts := range []Options{{}, {ClipboardRead: ClipboardDeny}} {
		conn := newPipe()
		opts.Conn = conn
		term, err := New(opts)
		if err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
		tt.SetClipboard("secret")
		term.Feed([]byte("\x1b]52;c;?\x1b\\"))
		tt.Frame()
		if got := conn.take(term); got != "\x1b]52;c;\x1b\\" {
			t.Errorf("ClipboardRead %v: the program read %q", opts.ClipboardRead, got)
		}
		if got := tt.Clipboard(); got != "secret" {
			t.Errorf("the read changed the clipboard to %q", got)
		}
		term.Close()
	}
}

// TestClipboardReadAsk: with ClipboardRead ClipboardAsk,
// Options.OnClipboardRequest sees the read, with what it asks for, and
// Allow or Deny answers it, in the hook or later, from any goroutine.
func TestClipboardReadAsk(t *testing.T) {
	loadLib(t)
	for _, answer := range []int8{1, 2, 0, 3} {
		conn := newPipe()
		var requests []ClipboardRequest
		var answerNow func()
		term, err := New(Options{
			Conn:          conn,
			ClipboardRead: ClipboardAsk,
			OnClipboardRequest: func(req ClipboardRequest) {
				requests = append(requests, req)
				switch answer {
				case 1:
					req.Allow()
				case 2:
					req.Deny()
				case 0:
					answerNow = req.Allow // later, from the test's goroutine
				case 3:
					answerNow = req.Deny
				}
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
		tt.SetClipboard("hello")
		term.Feed([]byte("\x1b]52;c;?\x1b\\"))
		tt.Frame()
		if len(requests) != 1 {
			t.Fatalf("the hook saw %d requests", len(requests))
		}
		if req := requests[0]; !slices.Equal(req.Mimes, []string{"text/plain"}) || req.List || req.Text != "hello" {
			t.Errorf("the hook saw %+v", req)
		}
		if answer == 0 || answer == 3 {
			answerNow()
			tt.Frame()
		}
		want := "\x1b]52;c;\x1b\\"
		if answer == 1 || answer == 0 {
			want = "\x1b]52;c;aGVsbG8=\x1b\\"
		}
		if got := conn.take(term); got != want {
			t.Errorf("answer %d: the program read %q, want %q", answer, got, want)
		}
		term.Close()
	}
}

// TestClipboardReadOrder: reads are answered in the order they came, since
// an OSC 52 reply doesn't tell which read it answers: one the app is asked
// about holds back those after it.
func TestClipboardReadOrder(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	var requests []ClipboardRequest
	term, err := New(Options{
		Conn:               conn,
		ClipboardRead:      ClipboardAsk,
		OnClipboardRequest: func(req ClipboardRequest) { requests = append(requests, req) },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.SetClipboard("hello")
	term.Feed([]byte("\x1b]52;c;?\x1b\\\x1b]52;p;?\x1b\\"))
	tt.Frame()
	if len(requests) != 1 || requests[0].Primary {
		t.Fatalf("asked %+v before the first read was answered", requests)
	}
	requests[0].Deny()
	tt.Frame()
	if got := conn.take(term); got != "\x1b]52;c;\x1b\\" {
		t.Errorf("the first read was answered %q", got)
	}
	if len(requests) != 2 || !requests[1].Primary {
		t.Fatalf("the second read was not asked about: %+v", requests)
	}
	requests[1].Allow()
	tt.Frame()
	want := "\x1b]52;p;aGVsbG8=\x1b\\"
	if got := conn.take(term); got != want {
		t.Errorf("the second read was answered %q, want %q", got, want)
	}
}

// TestClipboardReadPrompt: without OnClipboardRequest the terminal asks
// with its own prompt, whose default button (Enter) allows and whose
// Escape denies, keeping the read pending across frames until then.
func TestClipboardReadPrompt(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn, ClipboardRead: ClipboardAsk})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.SetClipboard("hello")
	term.Feed([]byte("\x1b]52;c;?\x1b\\"))
	tt.Frame() // the prompt opens
	if got := conn.take(term); got != "" {
		t.Fatalf("the prompt answered %q on its own", got)
	}
	if !tt.HasText("hello") {
		t.Error("the prompt does not show the clipboard's text")
	}
	tt.Key(0, ui.KeyEnter)
	want := "\x1b]52;c;aGVsbG8=\x1b\\"
	if got := conn.take(term); got != want {
		t.Errorf("Enter allowed the read: %q, want %q", got, want)
	}
	// Escape dismisses the alert, which denies the read.
	term.Feed([]byte("\x1b]52;c;?\x1b\\"))
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if got := conn.take(term); got != "\x1b]52;c;\x1b\\" {
		t.Errorf("Escape answered %q, want an empty clipboard", got)
	}
}

// TestClipboardReadKitty: the Kitty clipboard protocol's reads (OSC 5522)
// follow the same policy: allowed reads are answered with the text, and
// refused ones with EPERM.
func TestClipboardReadKitty(t *testing.T) {
	loadLib(t)
	read := "\x1b]5522;type=read;" + base64.StdEncoding.EncodeToString([]byte("text/plain")) + "\x1b\\"
	for _, policy := range []ClipboardAccess{ClipboardAllow, ClipboardDeny} {
		conn := newPipe()
		term, err := New(Options{Conn: conn, ClipboardRead: policy})
		if err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
		tt.SetClipboard("hello")
		term.Feed([]byte("\x1b[?5522h"))
		term.Feed([]byte(read))
		tt.Frame()
		want := "\x1b]5522;type=read:status=EPERM\x1b\\"
		if policy == ClipboardAllow {
			mime := base64.StdEncoding.EncodeToString([]byte("text/plain"))
			want = "\x1b]5522;type=read:status=OK\x1b\\" +
				"\x1b]5522;type=read:status=DATA:mime=" + mime + ";aGVsbG8=\x1b\\" +
				"\x1b]5522;type=read:status=DONE\x1b\\"
		}
		if got := conn.take(term); got != want {
			t.Errorf("ClipboardRead %v: the program read %q, want %q", policy, got, want)
		}
		term.Close()
	}
}

func TestPadding(t *testing.T) {
	loadLib(t)
	th := LightTheme()
	red, green, blue := th.Palette[1], th.Palette[2], th.Palette[4]
	near := func(name string, got, want ui.Color) {
		t.Helper()
		d := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
		if d(got.R, want.R) > 2 || d(got.G, want.G) > 2 || d(got.B, want.B) > 2 {
			t.Errorf("%s is %v, want %v", name, got, want)
		}
	}
	for _, mode := range []PaddingColor{PaddingBackground, PaddingExtend, PaddingExtendAlways} {
		term, err := New(Options{Conn: newPipe(), Theme: th, NoBlink: true, Padding: Padding{X: 20, Y: 12}, PaddingColor: mode})
		if err != nil {
			t.Fatal(err)
		}
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
		// At the view's size: row 0 red across, as a full-width bar; row
		// 1 green in its first two cells only, as a shell's highlight.
		term.Feed([]byte("\x1b[41m\x1b[2K\x1b[0m\x1b[2;1H\x1b[42mab\x1b[0m\x1b[5;1H"))
		tt.Frame()
		at := func(x, y float32) ui.Color {
			s := term.v.scale
			c := tt.Image().RGBAAt(int(x*s), int(y*s))
			return ui.RGB(c.R, c.G, c.B)
		}
		if want := int(20 * term.v.scale); term.v.ox != want {
			t.Errorf("padding across %d pixels, want %d", term.v.ox, want)
		}
		_, y0 := cellCenter(term, 0, 0)
		_, y1 := cellCenter(term, 0, 1)
		switch mode {
		case PaddingBackground:
			near("the padding beside a red row", at(5, y0), th.Background)
		case PaddingExtend:
			near("the padding beside a red row", at(5, y0), red)
			near("the padding past a red row's end", at(397, y0), red)
			near("the padding above a red row", at(200, 3), red)
			near("the padding beside a row half green", at(5, y1), th.Background)
			// A full-screen program's colors extend from every row.
			term.Feed([]byte("\x1b[?1049h\x1b[44m\x1b[2J\x1b[0m\x1b[3;1H\x1b[42mab\x1b[0m"))
			tt.Frame()
			_, y2 := cellCenter(term, 0, 2)
			near("the padding beside a full-screen program's green", at(5, y2), green)
			near("the padding below a full-screen program", at(200, 197), blue)
			// A program's own cursor, a character in inverse video, stays
			// in its cell, even at the edge of a full-screen program's row.
			term.Feed([]byte("\x1b[2J\x1b[H\x1b[7m \x1b[0m\x1b[K"))
			tt.Frame()
			_, y0 = cellCenter(term, 0, 0)
			near("the padding beside a program's cursor", at(5, y0), th.Background)
			near("the padding above a program's cursor", at(5, 3), th.Background)
		case PaddingExtendAlways:
			near("the padding beside a row half green", at(5, y1), green)
		}
		term.Close()
	}
}

// The pointer is Ghostty's: the I-beam, the arrow while a program takes
// the mouse (the I-beam with Shift, which selects then), and the shape a
// program asks for with OSC 22 until the mouse mode changes again.
func TestPointerShape(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	tt.Move(cellCenter(term, 1, 0))
	step := func(feed string, mods ui.Modifiers, want ui.Cursor, why string) {
		t.Helper()
		term.Feed([]byte(feed))
		tt.HoldModifiers(mods)
		tt.Frame()
		if c := tt.Cursor(); c != want {
			t.Errorf("%s: the pointer is %v, want %v", why, c, want)
		}
	}
	step("", 0, ui.CursorText, "a shell")
	step("\x1b[?1000h\x1b[?1006h", 0, ui.CursorDefault, "a program taking the mouse")
	step("", ui.Shift, ui.CursorText, "Shift over a program taking the mouse")
	step("", 0, ui.CursorDefault, "Shift let go of")
	step("\x1b]22;pointer\x07", 0, ui.CursorPointer, "OSC 22 pointer")
	step("\x1b]22;col-resize\x07", 0, ui.CursorResizeColumn, "OSC 22 col-resize")
	step("\x1b[?1000l\x1b[?1006l", 0, ui.CursorText, "the mouse given back")
	step("\x1b]22;crosshair\x07", 0, ui.CursorCrosshair, "OSC 22 without the mouse")
	step("\x1b]22;\x07", 0, ui.CursorText, "OSC 22 reset, to the I-beam")
}

// sizedPipe is a pipe told the size, as a server's connection is.
type sizedPipe struct {
	*pipe
	sizes chan [2]int
}

func (p sizedPipe) Resize(cols, rows int) error {
	p.sizes <- [2]int{cols, rows}
	return nil
}

// With ResizeByConn, the view asks Conn for its size, once, and the
// screen takes a size only from Resize, which Conn is not told of.
func TestResizeByConn(t *testing.T) {
	loadLib(t)
	conn := sizedPipe{newPipe(), make(chan [2]int, 16)}
	term, err := New(Options{Conn: conn, ResizeByConn: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Resize(30, 10)
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	tt.Frame()
	var asked [2]int
	select {
	case asked = <-conn.sizes:
	case <-time.After(2 * time.Second):
		t.Fatal("the view did not ask Conn for its size")
	}
	if asked == [2]int{30, 10} {
		t.Fatalf("asked %v, the screen's own size: the test's view should fit another", asked)
	}
	if c, r := term.Size(); c != 30 || r != 10 {
		t.Errorf("the screen took the view's size, %dx%d, before Resize", c, r)
	}
	term.Resize(asked[0], asked[1])
	tt.Frame()
	if c, r := term.Size(); c != asked[0] || r != asked[1] {
		t.Errorf("after Resize the screen is %dx%d, want %v", c, r, asked)
	}
	select {
	case s := <-conn.sizes:
		t.Errorf("Conn told %v again: by the view twice, or by Resize", s)
	case <-time.After(200 * time.Millisecond):
	}
}

// pixelPipe is a pipe told the size with the size of a cell in pixels.
type pixelPipe struct {
	*pipe
	sizes chan [4]int
}

func (p pixelPipe) ResizePixels(cols, rows, cellWidth, cellHeight int) error {
	p.sizes <- [4]int{cols, rows, cellWidth, cellHeight}
	return nil
}

// A terminal standing for another, as a server's for a window, takes the
// size of the window's cells, which programs get from size queries and
// count pointer reports in pixels (mode 1016) in; the window's terminal
// tells its Conn that size.
func TestResizePixels(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	server, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	server.ResizePixels(40, 6, 18, 36)
	server.Feed([]byte("\x1b[16t\x1b[14t"))
	if got, want := conn.take(server), "\x1b[6;36;18t\x1b[4;216;720t"; got != want {
		t.Errorf("size queries answered %q, want %q", got, want)
	}
	server.ResizePixels(30, 5, 0, 0)
	server.Feed([]byte("\x1b[16t"))
	if got, want := conn.take(server), "\x1b[6;36;18t"; got != want {
		t.Errorf("a zero cell size answered %q, want %q", got, want)
	}

	client := pixelPipe{newPipe(), make(chan [4]int, 8)}
	term, err := New(Options{Conn: client, ResizeByConn: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	tt.Frame()
	select {
	case s := <-client.sizes:
		term.mu.Lock()
		want := [4]int{term.asked.cols, term.asked.rows, term.size.cellW, term.size.cellH}
		term.mu.Unlock()
		if s != want {
			t.Errorf("Conn told %v, not the view's %v", s, want)
		}
	case <-time.After(10 * time.Second): // a hang guard
		t.Fatal("Conn not told the size")
	}
}

// themePipe is a pipe told the view's theme.
type themePipe struct {
	*pipe
	themes chan *Theme
}

func (p themePipe) UseTheme(theme *Theme) error {
	p.themes <- theme
	return nil
}

// A terminal standing for another, as a server's for a window, answers
// programs' questions about the colors with the window's, which its view
// tells its Conn: Copilot CLI, asking the background, drew its input box
// dark on a light one.
func TestUseTheme(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	server, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	light := &Theme{Foreground: ui.Hex("#202020"), Background: ui.Hex("#fafafa"), Cursor: ui.Hex("#ff0000")}
	light.Palette[1] = ui.Hex("#cd3131")
	server.UseTheme(light)
	server.Feed([]byte("\x1b]10;?\x1b\\\x1b]11;?\x1b\\\x1b]12;?\x1b\\\x1b]4;1;?\x1b\\\x1b[?996n"))
	want := "\x1b]10;rgb:2020/2020/2020\x1b\\\x1b]11;rgb:fafa/fafa/fafa\x1b\\\x1b]12;rgb:ffff/0000/0000\x1b\\\x1b]4;1;rgb:cdcd/3131/3131\x1b\\\x1b[?997;2n"
	if got := conn.take(server); got != want {
		t.Errorf("color queries answered %q, want %q", got, want)
	}

	client := themePipe{newPipe(), make(chan *Theme, 8)}
	term, err := New(Options{Conn: client, Theme: light})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	select {
	case th := <-client.themes:
		if th != light {
			t.Errorf("Conn told %+v, not the view's theme", th)
		}
	case <-time.After(10 * time.Second): // a hang guard
		t.Fatal("Conn not told the theme")
	}
	tt.Frame()
	select {
	case th := <-client.themes:
		t.Errorf("Conn told the same theme again: %+v", th)
	case <-time.After(100 * time.Millisecond):
	}
}

// A full-screen program's rows with cells of the default background keep
// their colors in their cells, as in Ghostty: Copilot CLI's tabs on its
// top row grew up into the padding, and the row of its input box, which
// starts with a border on the default background, stuck out past the box.
func TestPaddingPartialRows(t *testing.T) {
	loadLib(t)
	th := LightTheme()
	term, err := New(Options{Conn: newPipe(), Theme: th, NoBlink: true, Padding: Padding{X: 20, Y: 12}, PaddingColor: PaddingExtend})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	pad := strings.Repeat(" ", term.v.cols)
	term.Feed([]byte("\x1b[?1049h\x1b[H \x1b[44m Current \x1b[0m \x1b[47m Gists \x1b[0m\x1b[3;1H┃\x1b[47m" + pad[1:] + "\x1b[0m"))
	tt.Frame()
	at := func(x, y float32) ui.Color {
		s := term.v.scale
		c := tt.Image().RGBAAt(int(x*s), int(y*s))
		return ui.RGB(c.R, c.G, c.B)
	}
	x, _ := cellCenter(term, 3, 0)
	_, y2 := cellCenter(term, 0, 2)
	for _, c := range []struct {
		name string
		x, y float32
	}{
		{"above a tab", x, 3},
		{"past the end of a box's row", 397, y2},
	} {
		if got := at(c.x, c.y); got != th.Background {
			t.Errorf("the padding %s is %v, want the background %v", c.name, got, th.Background)
		}
	}
}

// A program that asked (mode 2031) is told when the colors go from light
// to dark or back, as Ghostty tells it: a server's terminal as its window
// tells it the theme, and a terminal a view shows as the window's
// appearance changes. Claude Code and Copilot CLI ask, to recolor.
func TestColorSchemeChanges(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	server, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	server.UseTheme(LightTheme())
	server.UseTheme(DarkTheme())
	if got := conn.take(server); got != "" {
		t.Errorf("told %q without asking", got)
	}
	server.Feed([]byte("\x1b[?2031h"))
	server.UseTheme(LightTheme())
	server.UseTheme(LightTheme())
	if got, want := conn.take(server), "\x1b[?997;2n"; got != want {
		t.Errorf("going light told %q, want %q", got, want)
	}

	conn = newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	term.Feed([]byte("\x1b[?2031h"))
	tt.Frame()
	tt.SetDark(true)
	tt.Frame()
	if got, want := conn.take(term), "\x1b[?997;1n"; got != want {
		t.Errorf("a dark window told %q, want %q", got, want)
	}
}

// A transparent terminal draws a background a program set (OSC 11), as
// Ghostty shows it, and none again once the program resets it.
func TestTransparentProgramBackground(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), Theme: LightTheme(), Transparent: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(0, 0, 255)).Children(func() { View(c, term).Fill() })
	}, 400, 200)
	at := func() ui.Color {
		c := tt.Image().RGBAAt(int(300*term.v.scale), int(150*term.v.scale))
		return ui.RGB(c.R, c.G, c.B)
	}
	red, blue := ui.RGB(255, 0, 0), ui.RGB(0, 0, 255)
	for _, c := range []struct {
		label, out string
		want       ui.Color
	}{
		{"at the start", "", blue},
		{"set", "\x1b]11;#ff0000\x07", red},
		{"reset", "\x1b]111\x07", blue},
	} {
		term.Feed([]byte(c.out))
		tt.Frame()
		if got := at(); got != c.want {
			t.Errorf("%s: the background is %v, want %v", c.label, got, c.want)
		}
	}
}

// Dropped files' paths are quoted for the program's shell, so that each
// is one argument, whatever it holds: a newline does not end the command,
// nor do PowerShell's $(…) or $var expand. Where the shells are installed,
// they read each quoted path back as it was.
func TestQuotePaths(t *testing.T) {
	cases := []struct {
		q          quoting
		path, want string
	}{
		{quotePOSIX, "/Users/me/My Shots/shot (1).png", `/Users/me/My\ Shots/shot\ \(1\).png`},
		{quotePOSIX, "/tmp/a&b.txt", `/tmp/a\&b.txt`},
		{quotePOSIX, "/tmp/$(id)`id`;x|y*", "/tmp/\\$\\(id\\)\\`id\\`\\;x\\|y\\*"},
		{quotePOSIX, "/tmp/日本 語", `/tmp/日本\ 語`},
		{quotePOSIX, "/tmp/x\ntouch pwned", `$'/tmp/x\x0atouch pwned'`},
		{quotePOSIX, "/tmp/it's\tthere\\", `$'/tmp/it\'s\x09there\\'`},
		{quotePOSIX, "/tmp/a\rb\x1b[201~c", `$'/tmp/a\x0db\x1b[201~c'`},
		{quotePOSIX, "/tmp/\xff.bin", `$'/tmp/\xff.bin'`},
		{quotePowerShell, `C:\tmp\ab.txt`, `C:\tmp\ab.txt`},
		{quotePowerShell, `C:\Users\me\My Shots\shot (1).png`, `'C:\Users\me\My Shots\shot (1).png'`},
		{quotePowerShell, `C:\dl\$(Start-Process calc) x.txt`, `'C:\dl\$(Start-Process calc) x.txt'`},
		{quotePowerShell, `C:\dl\$env:USERNAME.txt`, `'C:\dl\$env:USERNAME.txt'`},
		{quotePowerShell, "C:\\dl\\a`b;c.txt", "'C:\\dl\\a`b;c.txt'"},
		{quotePowerShell, `C:\dl\it's ‘q’.txt`, `'C:\dl\it''s ‘‘q’’.txt'`},
		{quoteCmd, `C:\tmp\ab.txt`, `C:\tmp\ab.txt`},
		{quoteCmd, `C:\a b\c&d.txt`, `"C:\a b\c&d.txt"`},
	}
	for _, c := range cases {
		if got := c.q.quote(c.path); got != c.want {
			t.Errorf("quoting %d of %q = %s, want %s", c.q, c.path, got, c.want)
		}
	}
	for _, c := range []struct {
		goos, program string
		want          quoting
	}{
		{"linux", "/bin/zsh", quotePOSIX},
		{"darwin", "/usr/local/bin/pwsh", quotePowerShell},
		{"windows", `C:\Program Files\PowerShell\7\pwsh.exe`, quotePowerShell},
		{"windows", `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, quotePowerShell},
		{"windows", `C:\Windows\System32\cmd.exe`, quoteCmd},
		{"windows", `C:\Program Files\Git\bin\bash.exe`, quotePOSIX},
		{"windows", `C:\tools\nu.exe`, quotePowerShell},
	} {
		if got := quotingFor(c.goos, c.program); got != c.want {
			t.Errorf("quotingFor(%s, %s) = %d, want %d", c.goos, c.program, got, c.want)
		}
	}
	// The shells read the paths back.
	type shell struct {
		names []string
		q     quoting
		args  func(quoted string) []string
	}
	shells := []shell{
		{[]string{"bash", "zsh"}, quotePOSIX, func(q string) []string { return []string{"-c", "printf %s " + q} }},
		{[]string{"pwsh", "powershell"}, quotePowerShell, func(q string) []string {
			return []string{"-NoProfile", "-NonInteractive", "-Command", "& { [Console]::Out.Write($args[0]) } " + q}
		}},
	}
	for _, sh := range shells {
		for _, name := range sh.names {
			bin, err := exec.LookPath(name)
			if err != nil {
				continue
			}
			for _, c := range cases {
				if c.q != sh.q || (runtime.GOOS == "windows" && strings.ContainsFunc(c.path, unicode.IsControl)) {
					continue
				}
				out, err := exec.Command(bin, sh.args(c.want)...).Output()
				if err != nil || string(out) != c.path {
					t.Errorf("%s read %s as %q (%v), want %q", name, c.want, out, err, c.path)
				}
			}
		}
	}
}

// Files dropped on a terminal paste their paths, escaped for a shell and
// between spaces, as Ghostty does: bracketed for a program that asks, so
// that an agent takes them as files it attaches.
func TestDropFiles(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	paths := []string{"/Users/me/My Shots/shot (1).png", "/tmp/a&b.txt"}
	want := []string{`/Users/me/My\ Shots/shot\ \(1\).png /tmp/a\&b.txt`, `/tmp/a\&b.txt`}
	if runtime.GOOS == "windows" {
		paths = []string{`C:\Users\me\My Shots\shot (1).png`, `C:\tmp\ab.txt`}
		want = []string{`'C:\Users\me\My Shots\shot (1).png' C:\tmp\ab.txt`, `C:\tmp\ab.txt`}
	}
	if !tt.DropFiles(100, 100, paths...) {
		t.Fatal("the terminal did not take the files")
	}
	tt.Frame()
	if got := conn.take(term); got != want[0] {
		t.Errorf("dropping pasted %q, want %q", got, want[0])
	}
	term.Feed([]byte("\x1b[?2004h"))
	tt.Frame()
	tt.DropFiles(100, 100, paths[1])
	tt.Frame()
	if got, want := conn.take(term), "\x1b[200~"+want[1]+"\x1b[201~"; got != want {
		t.Errorf("dropping in bracketed paste pasted %q, want %q", got, want)
	}
}
