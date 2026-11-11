package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
	"github.com/egoist/mygo/ui"
)

// halves returns the Kitty graphics sequence that transmits a w×h image,
// its top half red and its bottom half blue, with more keys.
func halves(id, w, h int, keys string) string {
	var px bytes.Buffer
	for y := range h {
		c := []byte{255, 0, 0, 255}
		if y >= h/2 {
			c = []byte{0, 0, 255, 255}
		}
		px.Write(bytes.Repeat(c, w))
	}
	return fmt.Sprintf("\x1b_Ga=T,f=32,s=%d,v=%d,i=%d,q=2%s;%s\x1b\\", w, h, id, keys, base64.StdEncoding.EncodeToString(px.Bytes()))
}

// Placeholder cells (Kitty graphics' Unicode placeholders, U+10EEEE)
// show their parts of the image placed virtually, by the image's id in
// their foreground color and the row and column in their combining
// marks, which the cells after the first may leave out: yazi shows its
// previews so, and showed tofu (kero#228).
func TestPlaceholders(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), Theme: LightTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	at := func(col, row int) ui.Color {
		x, y := cellCenter(term, col, row)
		c := tt.Image().RGBAAt(int(x*term.v.scale), int(y*term.v.scale))
		return ui.RGB(c.R, c.G, c.B)
	}
	red, blue := ui.RGB(255, 0, 0), ui.RGB(0, 0, 255)
	bg := LightTheme().Background
	const p = "\U0010EEEE"
	row0, row1, col2 := "̅", "̍", "̎"
	// Image 7, four cells by two, placed virtually in four by two.
	term.Feed([]byte(halves(7, 4*term.v.cellW, 2*term.v.cellH, ",U=1,c=4,r=2")))
	// Its first row at row 1, column 2, then its second; the cells after
	// the first of a row take their row and column from it.
	term.Feed([]byte("\x1b[2;3H\x1b[38;5;7m" + p + row0 + row0 + strings.Repeat(p, 3) +
		"\x1b[3;3H" + p + row1 + strings.Repeat(p, 3) +
		// Its third column of its second row alone, at row 5.
		"\x1b[5;1H" + p + row1 + col2 + "\x1b[0m"))
	tt.Frame()
	for _, c := range []struct {
		col, row int
		want     ui.Color
	}{
		{2, 1, red}, {5, 1, red}, {2, 2, blue}, {5, 2, blue},
		{1, 1, bg}, {6, 1, bg}, {2, 3, bg},
		{0, 4, blue}, {1, 4, bg},
	} {
		if got := at(c.col, c.row); got != c.want {
			t.Errorf("cell %d,%d is %v, want %v", c.col, c.row, got, c.want)
		}
	}
	// Placeholders of an image without a virtual placement show nothing.
	term.Feed([]byte("\x1b[7;1H\x1b[38;5;9m" + p + p + "\x1b[0m"))
	tt.Frame()
	if got := at(0, 6); got != bg {
		t.Errorf("a placeholder of no placement is %v, want %v", got, bg)
	}
}

// The other terminal of a Mirror gets the virtual placements, by their
// ids, and the placeholder cells that show them, and drops a placement
// when the program deletes it: yazi's previews in a server's session.
func TestMirrorPlaceholders(t *testing.T) {
	loadLib(t)
	src, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	dst, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	m, err := src.NewMirror()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close(); src.Close(); dst.Close() })
	src.ResizePixels(40, 10, 8, 16)
	dst.ResizePixels(40, 10, 8, 16)
	frame := func() string {
		f, _ := m.Frame()
		dst.Feed(f)
		return string(f)
	}
	frame()
	virtual := func(term *Terminal) []string {
		term.mu.Lock()
		defer term.mu.Unlock()
		var out []string
		term.term.Placements(func(p vt.Placement) {
			if p.Virtual {
				out = append(out, fmt.Sprintf("i=%d p=%d c=%d r=%d", p.ImageID, p.ID, p.ParamCols, p.ParamRows))
			}
		})
		return out
	}
	src.Feed([]byte(halves(5, 16, 32, ",U=1,p=3,c=2,r=2") + "\x1b[38;5;5m\x1b[58;5;3m\U0010EEEE̅\U0010EEEE\x1b[0m"))
	frame()
	if a, b := virtual(src), virtual(dst); len(a) != 1 || fmt.Sprint(a) != fmt.Sprint(b) {
		t.Errorf("the other terminal places %v virtually, want %v", b, a)
	}
	if f := frame(); strings.Contains(f, "_G") {
		t.Errorf("a frame with nothing new sent images again: %q", f)
	}
	src.Feed([]byte("\x1b_Ga=d,d=i,i=5,p=3,q=2\x1b\\"))
	frame()
	if b := virtual(dst); len(b) != 0 {
		t.Errorf("the other terminal still places %v virtually", b)
	}
}
