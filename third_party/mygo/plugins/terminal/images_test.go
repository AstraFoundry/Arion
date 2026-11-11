package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	imagecolor "image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/egoist/mygo/ui"
)

// kittyImage returns the Kitty graphics sequence that transmits and
// places a w×h image of one color, as RGBA, with more keys.
func kittyImage(id, w, h int, c ui.Color, keys string) string {
	px := bytes.Repeat([]byte{c.R, c.G, c.B, 0xff}, w*h)
	return fmt.Sprintf("\x1b_Ga=T,f=32,s=%d,v=%d,i=%d,q=2%s;%s\x1b\\", w, h, id, keys, base64.StdEncoding.EncodeToString(px))
}

// Images programs place (Kitty graphics) show where they placed them,
// over the text unless placed under it, and go when deleted: grok shows a
// pasted image's preview so.
func TestImages(t *testing.T) {
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
	// Two cells by two of red at row 1, column 2; the cursor stays.
	term.Feed([]byte("\x1b[2;3H" + kittyImage(7, 2*term.v.cellW, 2*term.v.cellH, red, ",C=1")))
	tt.Frame()
	for _, c := range []struct {
		col, row int
		want     ui.Color
	}{{2, 1, red}, {3, 2, red}, {4, 1, bg}, {2, 3, bg}} {
		if got := at(c.col, c.row); got != c.want {
			t.Errorf("cell %d,%d is %v, want %v", c.col, c.row, got, c.want)
		}
	}
	// Under the cells' backgrounds, a cell with its own hides it.
	term.Feed([]byte("\x1b[5;1H" + kittyImage(8, term.v.cellW, term.v.cellH, blue, ",z=-1073741825,C=1") + "\x1b[42m \x1b[0m"))
	tt.Frame()
	if got := at(0, 4); got == blue {
		t.Errorf("an image under the backgrounds shows over a cell's own")
	}
	// Deleted, it goes.
	term.Feed([]byte("\x1b_Ga=d,d=I,i=7,q=2\x1b\\"))
	tt.Frame()
	if got := at(2, 1); got != bg {
		t.Errorf("a deleted image's cell is %v, want %v", got, bg)
	}
}

// Images sent as PNG (f=100), as grok sends a pasted image's preview,
// decode and show.
func TestImagePNG(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), Theme: LightTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	src := image.NewNRGBA(image.Rect(0, 0, 3*term.v.cellW, 2*term.v.cellH))
	draw.Draw(src, src.Bounds(), image.NewUniform(imagecolor.NRGBA{G: 255, A: 255}), image.Point{}, draw.Src)
	var b bytes.Buffer
	png.Encode(&b, src)
	data := base64.StdEncoding.EncodeToString(b.Bytes())
	// In chunks, as programs send big ones.
	term.Feed([]byte("\x1b[2;2H\x1b_Ga=T,f=100,q=2,m=1;" + data[:40] + "\x1b\\\x1b_Gm=0;" + data[40:] + "\x1b\\"))
	tt.Frame()
	x, y := cellCenter(term, 2, 2)
	c := tt.Image().RGBAAt(int(x*term.v.scale), int(y*term.v.scale))
	if got := ui.RGB(c.R, c.G, c.B); got != ui.RGB(0, 255, 0) {
		t.Errorf("the image's cell is %v, want green", got)
	}
}
