package terminal

import (
	"fmt"
	"image/png"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
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

// waitFor runs frames until cond holds.
func waitFor(t *testing.T, tt *ui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
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

// take returns what the terminal sent since the last take, once it sent
// want bytes or a second passed.
func (p *pipe) take(want int) string {
	deadline := time.Now().Add(time.Second)
	for {
		p.mu.Lock()
		if len(p.sent) >= want || time.Now().After(deadline) {
			s := string(p.sent)
			p.sent = nil
			p.mu.Unlock()
			return s
		}
		p.mu.Unlock()
		time.Sleep(time.Millisecond)
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
	tt.Scroll(100, 100, 0, -40) // a notch of a mouse wheel up
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
		if got := conn.take(len(want)); got != want {
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
	if got := conn.take(6); got != "世界" {
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
	if got := conn.take(1); got != "" {
		t.Errorf("composing sent %q", got)
	}
	tt.Compose("你", 1)
	tt.Type("你")
	if got := conn.take(3); got != "你" {
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
	if got, want := conn.take(9), "\x1b[<0;3;4M"; got != want {
		t.Errorf("press sent %q, want %q", got, want)
	}
	x2, y2 := cellCenter(term, 5, 3)
	tt.Move(x2, y2)
	if got, want := conn.take(10), "\x1b[<32;6;4M"; got != want {
		t.Errorf("drag sent %q, want %q", got, want)
	}
	tt.Release(x2, y2)
	if got, want := conn.take(9), "\x1b[<0;6;4m"; got != want {
		t.Errorf("release sent %q, want %q", got, want)
	}
	tt.Scroll(x, y, 0, -40)
	if got := conn.take(30); !strings.HasPrefix(got, "\x1b[<64;3;4M") {
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
	case <-time.After(10 * time.Second):
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
	if got := conn.take(6); got != "pasted" {
		t.Errorf("Paste sent %q", got)
	}
	// The context menu.
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	if got := strings.Join(tt.Menu(), ","); !strings.Contains(got, "Copy") || !strings.Contains(got, "Clear Scrollback") {
		t.Errorf("context menu = %q", got)
	}
	tt.SetClipboard("again")
	if err := tt.ChooseMenuItem("Paste"); err != nil {
		t.Fatal(err)
	}
	if got := conn.take(5); got != "again" {
		t.Errorf("the menu's Paste sent %q", got)
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
	if f := k.variant(1); f.Weight != 700 || f.Features != "calt=0" {
		t.Errorf("the bold of weight 300 is %+v", f)
	}
	if f := (&Font{Weight: 800}).key(1).variant(3); f.Weight != 900 || !f.Italic {
		t.Errorf("the bold italic of weight 800 is %+v", f)
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
	case <-time.After(20 * time.Second):
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
	if got := conn.take(1); got != "b" {
		t.Errorf("sent %q", got)
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
	case <-time.After(5 * time.Second):
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
	tt.HoldModifiers(0)
	if c := tt.Cursor(); c != ui.CursorText {
		t.Errorf("the cursor is %v once Command is let go of", c)
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
	got := conn.take(18) // two reports, 22 bytes with Control
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
	term, err := New(Options{Conn: conn, OnPaste: func(text string) bool { seen = append(seen, text); return take }})
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
	if got := conn.take(1); len(got) != 0 {
		t.Errorf("a taken paste sent %q", got)
	}
	take = false
	tt.Key(paste, ui.KeyV)
	tt.Frame()
	if got := conn.take(5); got != "hello" || len(seen) != 3 {
		t.Errorf("a paste OnPaste leaves sent %q (seen %q)", got, seen)
	}
}

// TestOnKeyTakesAKey: Options.OnKey sees a key press before the terminal
// encodes it, in order with the keys after it; a true result sends
// nothing, not even the text the key types.
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
	if got := conn.take(16); got != "<taken>\t<taken>y" {
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
	tt.Scroll(100, 100, 0, -40) // a notch up
	tt.Scroll(100, 100, 0, 40)  // and down
	if !slices.Equal(seen, []int{-3, 3}) {
		t.Errorf("OnScroll saw %v, want [-3 3]", seen)
	}
	if got := offset(); got != bottom {
		t.Errorf("a taken scroll moved the scrollback: offset = %d, want %d", got, bottom)
	}
	// The program takes the mouse: the wheel goes to it.
	term.Feed([]byte("\x1b[?1002h\x1b[?1006h"))
	tt.Frame()
	tt.Scroll(100, 100, 0, -40)
	if len(seen) != 2 {
		t.Errorf("OnScroll saw %v while the program took the mouse", seen)
	}
	if got := conn.take(10); !strings.HasPrefix(got, "\x1b[<64;") {
		t.Errorf("wheel sent %q, want a mouse report", got)
	}
	// A full-screen program gets it as arrows, unless it turns that off.
	term.Feed([]byte("\x1b[?1002l\x1b[?1006l\x1b[?1049h"))
	tt.Frame()
	tt.Scroll(100, 100, 0, -40)
	if len(seen) != 2 {
		t.Errorf("OnScroll saw %v on a full-screen program's screen", seen)
	}
	if got := conn.take(9); strings.Count(got, "A") != 3 {
		t.Errorf("wheel sent %q, want three arrows up", got)
	}
	term.Feed([]byte("\x1b[?1007l"))
	tt.Frame()
	tt.Scroll(100, 100, 0, -40)
	if !slices.Equal(seen, []int{-3, 3, -3}) {
		t.Errorf("OnScroll saw %v without alternate scroll", seen)
	}
}
