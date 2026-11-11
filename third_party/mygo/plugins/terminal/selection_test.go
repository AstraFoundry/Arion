package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestShiftClickExtendsSelection(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), CopyOnSelect: CopyOnSelectClipboard})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x, y := cellCenter(term, 0, 0)
	tt.Press(x-2, y)
	x, y = cellCenter(term, 4, 0)
	tt.Move(x+2, y)
	tt.Release(x+2, y)
	if got := tt.Clipboard(); got != "hello" {
		t.Fatalf("initial selection = %q", got)
	}
	for _, step := range []struct {
		col, row int
		want     string
	}{
		{5, 1, "hello world\nsecond"},
		{9, 2, "hello world\nsecond line\nthird line"},
		{2, 0, "hel"},
	} {
		x, y = cellCenter(term, step.col, step.row)
		tt.ClickAtWith(ui.Shift, x, y)
		if got := tt.Clipboard(); got != step.want {
			t.Errorf("Shift-click (%d,%d) copied %q, want %q", step.col, step.row, got, step.want)
		}
		tt.SetClipboard("sentinel")
		tt.Command("copy")
		if got := tt.Clipboard(); got != step.want {
			t.Errorf("Copy after Shift-click = %q, want %q", got, step.want)
		}
	}
}

func TestShiftClickKeepsSelectionAnchor(t *testing.T) {
	loadLib(t)
	for _, mode := range []string{"reverse", "word", "line", "copy off", "output", "mouse tracking"} {
		t.Run(mode, func(t *testing.T) {
			term, err := New(Options{Conn: newPipe()})
			if err != nil {
				t.Fatal(err)
			}
			defer term.Close()
			term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
			tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
			x, y := cellCenter(term, 0, 0)
			want := "hello world\nsecond"
			switch mode {
			case "reverse":
				x, y = cellCenter(term, 4, 1)
				tt.Press(x+2, y)
				x, y = cellCenter(term, 0, 1)
				tt.Move(x-2, y)
				tt.Release(x-2, y)
				want = "hello world\nsecon"
			case "word", "line":
				clicks := 2
				if mode == "line" {
					clicks = 3
				}
				for range clicks {
					tt.ClickAt(x, y)
				}
			default:
				tt.Press(x-2, y)
				x, y = cellCenter(term, 4, 0)
				tt.Move(x+2, y)
				tt.Release(x+2, y)
			}
			if mode == "output" {
				term.Feed([]byte("\r\nmore output"))
				tt.Frame()
			}
			if mode == "mouse tracking" {
				term.Feed([]byte("\x1b[?1000h\x1b[?1006h"))
			}
			x, y = cellCenter(term, 5, 1)
			if mode == "reverse" {
				x, y = cellCenter(term, 0, 0)
			}
			tt.SetClipboard("untouched")
			tt.ClickAtWith(ui.Shift, x, y)
			if got := tt.Clipboard(); got != "untouched" {
				t.Errorf("copy-on-select disabled: clipboard = %q", got)
			}
			tt.Command("copy")
			if got := tt.Clipboard(); got != want {
				t.Errorf("extended selection = %q, want %q", got, want)
			}
			term.Feed([]byte("\x1b[?1000l\x1b[?1006l"))
			// An ordinary click clears the selection. Shift-click must not
			// revive the previous selection's anchor after it is gone.
			x, y = cellCenter(term, 3, 2)
			tt.ClickAt(x, y)
			x, y = cellCenter(term, 8, 2)
			tt.ClickAtWith(ui.Shift, x, y)
			tt.SetClipboard("untouched")
			tt.Command("copy")
			if got := tt.Clipboard(); got != "untouched" {
				t.Errorf("Shift-click revived a cleared selection: %q", got)
			}
		})
	}
}

func TestShiftClickDragAutoscroll(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), CopyOnSelect: CopyOnSelectClipboard})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	for row := range 40 {
		term.Feed(fmt.Appendf(nil, "row%02d\r\n", row))
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	term.mu.Lock()
	term.term.ScrollToTop()
	term.mu.Unlock()
	tt.Frame()
	x, y := cellCenter(term, 0, 0)
	tt.Press(x-2, y)
	x, y = cellCenter(term, 4, 0)
	tt.Move(x+2, y)
	tt.Release(x+2, y)
	if got := tt.Clipboard(); got != "row00" {
		t.Fatalf("initial selection = %q", got)
	}
	v := term.v
	x, y = cellCenter(term, 4, 1)
	// A held Shift-click, followed by dragging beyond the bottom edge.
	// Use the view's real input handler so the button remains pressed.
	v.input(ui.InputEvent{Kind: ui.InputPointerDown, Mods: ui.Shift, X: x, Y: y})
	v.input(ui.InputEvent{Kind: ui.InputPointerMove, X: x, Y: 240})
	tt.Frame()
	tt.Advance(120 * time.Millisecond)
	v.input(ui.InputEvent{Kind: ui.InputPointerUp, X: x, Y: 240})
	got := tt.Clipboard()
	if !strings.HasPrefix(got, "row00\nrow01\n") || !strings.Contains(got, fmt.Sprintf("row%02d", v.rows)) {
		t.Errorf("drag did not keep the anchor and scroll to another row: %q", got)
	}
	if v.selecting || v.ticking {
		t.Error("release left selection dragging or autoscroll active")
	}
}
