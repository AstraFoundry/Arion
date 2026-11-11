package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

func TestCellWidthLiveGrid(t *testing.T) {
	loadLib(t)
	if err := ui.RegisterFont(gomono.TTF, "MyGo Cell Width Mono"); err != nil {
		t.Fatal(err)
	}
	conn := newPipe()
	font := Font{Family: "MyGo Cell Width Mono", Size: 14}
	term, err := New(Options{Conn: conn, Font: font, CopyOnSelect: CopyOnSelectClipboard, NoBlink: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("A中文B"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 640, 240)
	baseW, baseH := term.v.cellW, term.v.cellH
	baseCols, baseRows := term.Size()
	for _, width := range []float32{1.03, 1.5, 0.8, 1} {
		font.CellWidth = width
		term.SetFont(font)
		tt.Frame()
		v := term.v
		cols, rows := term.Size()
		if v.cellH != baseH || rows != baseRows {
			t.Fatalf("width %g changed rows from %d at %d pixels to %d at %d pixels", width, baseRows, baseH, rows, v.cellH)
		}
		if width == 1.5 && (v.cellW <= baseW || cols >= baseCols) {
			t.Fatalf("wider cells left width %d and %d columns, from %d and %d", v.cellW, cols, baseW, baseCols)
		}
		if width == 0.8 && (v.cellW >= baseW || cols <= baseCols) {
			t.Fatalf("narrower cells left width %d and %d columns, from %d and %d", v.cellW, cols, baseW, baseCols)
		}
		if width == 1 && (v.cellW != baseW || cols != baseCols) {
			t.Fatalf("reset left width %d and %d columns, want %d and %d", v.cellW, cols, baseW, baseCols)
		}
		term.Feed([]byte("\x1b[16t\x1b[6n"))
		want := fmt.Sprintf("\x1b[6;%d;%dt\x1b[1;7R", v.cellH, v.cellW)
		if got := conn.take(term); got != want {
			t.Fatalf("width %g: cell and mixed-width cursor queries = %q, want %q", width, got, want)
		}
		tt.SetClipboard("")
		x, y := cellCenter(term, 0, 0)
		tt.Press(x-2, y)
		x, y = cellCenter(term, 4, 0)
		tt.Move(x+2, y)
		tt.Release(x+2, y)
		if got := tt.Clipboard(); got != "A中文" {
			t.Fatalf("width %g: mixed-width selection = %q", width, got)
		}
		term.Feed([]byte("\x1b[?1000h\x1b[?1006h"))
		tt.Frame()
		x, y = cellCenter(term, 12, 2)
		tt.ClickAt(x, y)
		if got := conn.take(term); !strings.Contains(got, "\x1b[<0;13;3M") {
			t.Fatalf("width %g: click in column 13, row 3 reported %q", width, got)
		}
		term.Feed([]byte("\x1b[?1000l\x1b[?1006l"))
		tt.Frame()
		// Move the selection gesture's last click away from the next drag.
		tt.ClickAt(x, y)
	}
}

// Glyphs too wide for a cell shrink to the cell the font's own advance
// makes, so that wider columns space them out without enlarging them.
func TestCellWidthKeepsFittedGlyphs(t *testing.T) {
	loadLib(t)
	if err := ui.RegisterFont(goregular.TTF, "MyGo Cell Width Sans"); err != nil {
		t.Fatal(err)
	}
	advance := func(width float32) float32 {
		term, err := New(Options{Conn: newPipe(), Font: Font{Family: "MyGo Cell Width Sans", Size: 14, CellWidth: width}, NoBlink: true})
		if err != nil {
			t.Fatal(err)
		}
		defer term.Close()
		tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 120)
		// Go Regular draws Ǆ wider than any of its ASCII glyphs.
		term.Feed([]byte("Ǆ"))
		tt.Frame()
		term.mu.Lock()
		defer term.mu.Unlock()
		for _, r := range term.v.lines[0].runs {
			for i, g := range r.glyphs {
				if r.cols[i] == 0 {
					return g.Advance
				}
			}
		}
		t.Fatalf("width %g: no glyph in the first cell", width)
		return 0
	}
	// Narrower cells leave it its size too, as they do every glyph.
	fitted := advance(1)
	for _, width := range []float32{1.5, 0.8} {
		if got := advance(width); got != fitted {
			t.Errorf("a glyph too wide for its cell is %v wide in cells %g times as wide, %v in the font's", got, width, fitted)
		}
	}
}
