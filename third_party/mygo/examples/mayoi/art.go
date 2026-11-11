package main

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/egoist/mygo/ui"
)

// The sheet is 8×8 cells of 16 pixels, drawn here pixel by pixel so that
// the example needs no files.
const cell = 16

// Cells of the sheet.
const (
	cPave = iota
	cPave2
	cPetals
	cPetals2
	cCrack
	cGrate
	_
	_
	cMember // 2 frames for each bandmate, in the band's order
)

const (
	cStatic = cMember + 2*memberCount + iota*2 // 2 frames each
	cCrow
	cShadow
)

const (
	cNote = cShadow + 2 + iota
	cPick
	cStone // 3 kinds, by value
	_
	_
	cBandaid
	cBandaidEmpty
	cPuff // 3 frames
	_
	_
	cMic
	cMetronome
	cGuitar
	cBassIcon
	cDrum
	cShoe
	cStoneIcon
)

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

// canvas draws into one cell of the sheet.
type canvas struct {
	img    *image.RGBA
	ox, oy int
}

func at(img *image.RGBA, i int) canvas { return canvas{img, i % 8 * cell, i / 8 * cell} }

func (c canvas) px(x, y int, col color.RGBA) {
	if x >= 0 && y >= 0 && x < cell && y < cell {
		c.img.SetRGBA(c.ox+x, c.oy+y, col)
	}
}

// rows draws pixel art given as rows of characters, each a color of pal;
// spaces are transparent.
func (c canvas) rows(pal map[byte]color.RGBA, art ...string) {
	for y, row := range art {
		for x := 0; x < len(row); x++ {
			if col, ok := pal[row[x]]; ok {
				c.px(x, y, col)
			}
		}
	}
}

var shadowColor = color.RGBA{0, 0, 0, 80}

// sheet draws the sheet.
func sheet() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8*cell, 8*cell))
	ground(img)
	band(img)
	fright(img)
	things(img)
	return img
}

// ground draws the street at night: paving stones, sakura petals, a crack
// and a drain.
func ground(img *image.RGBA) {
	rng := rand.New(rand.NewPCG(4, 2))
	base, mortar, light := rgb(0x353b52), rgb(0x272c3f), rgb(0x3f465f)
	pave := func(i, n int) canvas {
		c := at(img, i)
		for y := range cell {
			for x := range cell {
				col := base
				// Bricks of 8×8, every other row offset by half.
				off := 0
				if y >= 8 {
					off = 4
				}
				if y%8 == 0 || (x+off)%8 == 0 {
					col = mortar
				} else if y%8 == 1 || (x+off)%8 == 1 {
					col = light
				}
				c.px(x, y, col)
			}
		}
		for range n {
			c.px(rng.IntN(cell), rng.IntN(cell), mortar)
		}
		return c
	}
	pave(cPave, 4)
	pave(cPave2, 12)
	pal := map[byte]color.RGBA{'p': rgb(0xf6b8cf), 'P': rgb(0xe48aae), 'w': rgb(0xffe4ee),
		'm': mortar, 'g': rgb(0x1b1f2d), 'G': rgb(0x4a5170)}
	pave(cPetals, 4).rows(pal,
		"",
		"   pP           ",
		"   Pp       wp  ",
		"            pP  ",
		"",
		"       pw       ",
		"       Pp       ",
		"",
		"",
		"  wp            ",
		"  pP      pP    ",
		"          Pw    ",
		"",
		"      pP        ",
		"      PP        ",
	)
	pave(cPetals2, 6).rows(pal,
		"",
		"",
		"         pw     ",
		"         Pp     ",
		"",
		"   pP           ",
		"   pp           ",
		"",
		"            pP  ",
		"            wP  ",
	)
	pave(cCrack, 4).rows(pal,
		"",
		"",
		"     m          ",
		"      m         ",
		"      mm        ",
		"        m       ",
		"        m m     ",
		"         m      ",
		"         m      ",
		"          m     ",
	)
	pave(cGrate, 2).rows(pal,
		"",
		"",
		"",
		"   GGGGGGGGGG   ",
		"   GgGgGgGgGG   ",
		"   GgGgGgGgGG   ",
		"   GgGgGgGgGG   ",
		"   GgGgGgGgGG   ",
		"   GgGgGgGgGG   ",
		"   GGGGGGGGGG   ",
	)
}

// A look is how a bandmate looks: their hair and eyes, the hair's shape
// above and beside the body, and what they hold.
type look struct {
	hair, hairDark, eye, eye2 uint32
	head                      []string // rows 0 to 8
	sides                     []string // hair beside the body, rows 9 on
	item                      []string // what they hold, rows 8 on
	itemPal                   map[byte]uint32
}

var looks = [memberCount]look{
	vocalist: {
		hair: 0x93a9cc, hairDark: 0x6d83aa, eye: 0xd9a441,
		head: []string{
			"",
			"     HHHHHH     ",
			"    HHHHHHHH    ",
			"   HHHHHHHHHH   ",
			"   HHHHHHHHHH   ",
			"   HhkkkkkkhH   ",
			"   HkekkkkEkH   ",
			"   Hkkkkkkkkh   ",
			"   hH kkkk Hh   ",
		},
		sides:   []string{"   h        h   "},
		item:    []string{"             M  ", "             m  ", "            km  "},
		itemPal: map[byte]uint32{'M': 0xd9dde8, 'm': 0x3a3f55},
	},
	guitarist: {
		hair: 0xf3a5c4, hairDark: 0xd47ea2, eye: 0x6aa86e,
		head: []string{
			"",
			"     HHHHHH     ",
			"    HHHHHHHH    ",
			"   HHHHHHHHHH   ",
			"  HHHHHHHHHHHH  ",
			"  HHhkkkkkkhHH  ",
			"  HHkekkkkEkHH  ",
			"  HHkkkkkkkkHH  ",
			"  HH  kkkk  HH  ",
		},
		sides: []string{"  HH        HH  ", "  HH        HH  ", "  hH        Hh  ", "   h        h   "},
		item: []string{
			"              l ",
			"             l  ",
			"            l   ",
			"        GGGl    ",
			"       GGGGGG   ",
			"        GGGG    ",
		},
		itemPal: map[byte]uint32{'G': 0xe0465e, 'l': 0x8f563b},
	},
	bassist: {
		hair: 0xd2a679, hairDark: 0xa97f54, eye: 0x4a8fd6,
		head: []string{
			"",
			"     HHHHHH     ",
			"    HHHHHHHH    ",
			"   HHHHHHHHHH   ",
			"  HHHHHHHHHHHH  ",
			"  HHHHkkkkkkHH  ",
			"  HHkekkkkEkHH  ",
			"  HHkkkkkkkkHH  ",
			"  HHH kkkk HHH  ",
		},
		sides: []string{"  HHH      HHH  ", "  HHH      HHH  ", "  hHH      HHh  ", "  hH        Hh  "},
		item: []string{
			"               l",
			"              l ",
			"             l  ",
			"            l   ",
			"       BBBBl    ",
			"      BBBBBBB   ",
			"       BBBBB    ",
		},
		itemPal: map[byte]uint32{'B': 0x7b4a8f, 'l': 0x5c3a24},
	},
	lead: {
		hair: 0xe6e9f0, hairDark: 0xa9b0c3, eye: 0xf2c94c, eye2: 0x56a0d3,
		head: []string{
			"   h        h   ",
			"   Hh HHHH hH   ",
			"   HHHHHHHHHH   ",
			"  HHHHHHHHHHHH  ",
			"  HHhHHHHHHhHH  ",
			"  HhkkkkkkkkhH  ",
			"  HHkekkkkEkHH  ",
			"  hHkkkkkkkkHh  ",
			"  hh  kkkk  hh  ",
		},
		sides: []string{"  hh        hh  "},
		item: []string{
			"              l ",
			"             l  ",
			"            l   ",
			"        TTTl    ",
			"       TTTTTT   ",
			"        TTTT    ",
		},
		itemPal: map[byte]uint32{'T': 0x3fc2b5, 'l': 0xe8e2d0},
	},
	drummer: {
		hair: 0x3c4170, hairDark: 0x272a4a, eye: 0xa05ad6,
		head: []string{
			"",
			"    HHHHHHHH    ",
			"   HHHHHHHHHH   ",
			"   HHHHHHHHHH   ",
			"   HHHHHHHHHH   ",
			"   HHkkkkkkHH   ",
			"   HkeekkEEkH   ",
			"   HkkkkkkkkH   ",
			"   HH kkkk HH   ",
		},
		sides:   []string{"   HH      HH   ", "   HH      HH   ", "   hH      Hh   ", "   hh      hh   "},
		item:    []string{"  d          d  ", "   d        d   ", "    d      d    "},
		itemPal: map[byte]uint32{'d': 0xe6c48f},
	},
}

// band draws the bandmates in their school uniforms, standing and
// stepping, facing right.
func band(img *image.RGBA) {
	body := []string{
		"   kNWWrrWWNk   ",
		"   kNWWWWWWNk   ",
		"    NNNNNNNN    ",
		"    NnNNNNnN    ",
	}
	legs := [2][]string{
		{"     ss  ss     ", "     bb  bb     ", "    SSSSSSSS    "},
		{"    ss    ss    ", "    bb    bb    ", "    SSSSSSSS    "},
	}
	for m, l := range looks {
		eye2 := l.eye2
		if eye2 == 0 {
			eye2 = l.eye
		}
		pal := map[byte]color.RGBA{
			'H': rgb(l.hair), 'h': rgb(l.hairDark), 'e': rgb(l.eye), 'E': rgb(eye2),
			'k': rgb(0xf6d5bd), 'W': rgb(0xf4f4f8), 'N': rgb(0x2b3557), 'n': rgb(0x232b47),
			'r': rgb(0xd94a5a), 's': rgb(0x232b47), 'b': rgb(0x3a2a30), 'S': shadowColor,
		}
		for name, hex := range l.itemPal {
			pal[name] = rgb(hex)
		}
		for f := range 2 {
			c := at(img, cMember+m*2+f)
			c.rows(pal, append(append(append([]string{}, l.head...), body...), legs[f]...)...)
			c.rows(pal, append(make([]string, 9), l.sides...)...)
			c.rows(pal, append(make([]string, 8), l.item...)...)
		}
	}
}

// fright draws stage fright, two frames each: static, a crow and a
// shadow.
func fright(img *image.RGBA) {
	rng := rand.New(rand.NewPCG(8, 8))
	grays := []color.RGBA{rgb(0x8a86a8), rgb(0x6b6788), rgb(0xb2aed0), rgb(0x4d4a66)}
	white, black := rgb(0xffffff), rgb(0x16141f)
	for f := range 2 {
		c := at(img, cStatic+f)
		r := 5.6 + float64(f)*0.6
		for y := range cell {
			for x := range cell {
				dx, dy := float64(x)-7.5, (float64(y)-9.5)*(1.1-float64(f)*0.15)
				jag := 0.8 * math.Sin(math.Atan2(dy, dx)*7+float64(f)*2)
				if math.Hypot(dx, dy) < r+jag {
					c.px(x, y, grays[rng.IntN(len(grays))])
				}
			}
		}
		c.rows(map[byte]color.RGBA{'w': white, 'e': black, 'S': shadowColor},
			"", "", "", "", "", "", "", "",
			"     we  we     ",
			"     ee  ee     ",
			"", "", "", "", "",
			"   SSSSSSSSSS   ",
		)
	}
	pal := map[byte]color.RGBA{
		'b': rgb(0x1e1c2b), 'B': rgb(0x34304a), 'y': rgb(0xffd95a), 'o': rgb(0xf2a03d),
		'd': rgb(0x120f1c), 'D': rgb(0x2a2340), 'r': rgb(0xff4d6a), 'S': shadowColor,
	}
	at(img, cCrow).rows(pal,
		"", "",
		" B            B ",
		" BB          BB ",
		" bBB  bbb   BBb ",
		"  bbBbbbybbBbb  ",
		"   bbbbbbbbbboo ",
		"     bbbbbb     ",
		"      b  b      ",
		"", "", "", "", "",
		"     SSSSSS     ",
	)
	at(img, cCrow+1).rows(pal,
		"", "", "", "",
		"      bbb       ",
		"     bbbybb     ",
		"  BBbbbbbbbboo  ",
		" BbbbbbbbbbbbB  ",
		" Bbb   b  b  bB ",
		" B            B ",
		"", "", "", "",
		"     SSSSSS     ",
	)
	hood := []string{
		"",
		"      DDDD      ",
		"     DddddD     ",
		"    DddddddD    ",
		"    DdrddrdD    ",
		"    DddddddD    ",
		"    DddddddD    ",
		"   DdddddddD    ",
		"   DddddddddD   ",
		"   DddddddddD   ",
		"  DdddddddddD   ",
		"  DddddddddddD  ",
	}
	at(img, cShadow).rows(pal, append(append([]string{}, hood...), "  Dd dd dd dD   ", "   d  d  d  d   ", "", "    SSSSSSSS    ")...)
	at(img, cShadow+1).rows(pal, append(append([]string{}, hood...), "  dD dd dd Dd   ", "  d  d  d  d    ", "", "    SSSSSSSS    ")...)
}

// things draws the note, the pick, stones, band-aids, the puff and the
// upgrades' icons.
func things(img *image.RGBA) {
	pal := map[byte]color.RGBA{
		'o': rgb(0x9ff0ff), 'O': rgb(0xffffff), 'c': rgb(0x3aa7e0),
		't': rgb(0x3fc2b5), 'T': rgb(0x2a8a80),
		'g': rgb(0x9a9cab), 'G': rgb(0x6d6f80), 'b': rgb(0x8fb4e8), 'B': rgb(0x5b7fb8),
		'p': rgb(0xf6b8cf), 'P': rgb(0xd47ea2),
		'a': rgb(0xe8b98f), 'A': rgb(0xc9926a), 'w': rgb(0xfff6e0), 'k': rgb(0x3b3550), 'K': rgb(0x2a2540),
		'm': rgb(0xd9dde8), 'M': rgb(0x3a3f55), 'y': rgb(0xffd95a), 'r': rgb(0xe0465e), 'l': rgb(0x8f563b),
		'v': rgb(0x7b4a8f), 'd': rgb(0xe6c48f), 'n': rgb(0x2b3557), 'W': rgb(0xd8dde8),
	}
	at(img, cNote).rows(pal,
		"", "",
		"        ooo     ",
		"        oOoo    ",
		"        o  oo   ",
		"        o   o   ",
		"        o       ",
		"        o       ",
		"        o       ",
		"     oooo       ",
		"    oOOOo       ",
		"    ooooo       ",
		"     ooo        ",
	)
	at(img, cPick).rows(pal,
		"", "", "",
		"   TTTTTTTTTT   ",
		"   TttttttttT   ",
		"   TtOttttttT   ",
		"    TttttttT    ",
		"    TttttttT    ",
		"     TttttT     ",
		"     TttttT     ",
		"      TttT      ",
		"       TT       ",
	)
	stone := func(i int, light, dark byte) {
		p := map[byte]color.RGBA{'l': pal[light], 'L': pal[dark], 'O': pal['O']}
		at(img, i).rows(p,
			"", "", "", "", "", "",
			"      llll      ",
			"     lOllll     ",
			"    llllllLL    ",
			"    lllllLLL    ",
			"     LLLLLL     ",
		)
	}
	stone(cStone, 'g', 'G')
	stone(cStone+1, 'b', 'B')
	stone(cStone+2, 'p', 'P')
	bandaid := []string{
		"", "", "", "",
		"   aaaaaaaaaa   ",
		"  aaaawwwwaaaa  ",
		"  aAaawAAwaaAa  ",
		"  aaaawAAwaaaa  ",
		"  aAaawwwwaaAa  ",
		"   aaaaaaaaaa   ",
	}
	at(img, cBandaid).rows(pal, bandaid...)
	at(img, cBandaidEmpty).rows(map[byte]color.RGBA{'a': pal['k'], 'A': pal['K'], 'w': pal['K']}, bandaid...)
	for f := range 3 {
		c := at(img, cPuff+f)
		r := 3 + f*2
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				d := x*x + y*y
				if d <= r*r && (f == 0 || d >= (r-2)*(r-2)) {
					c.px(8+x, 8+y, pal['w'])
				}
			}
		}
	}
	at(img, cMic).rows(pal,
		"",
		"      mmmm      ",
		"     mWmWmm     ",
		"     mmWmWm     ",
		"     mWmWmm     ",
		"      mmmm      ",
		"       MM       ",
		"       MM       ",
		"       MM       ",
		"       MM       ",
		"       MM       ",
		"       MM       ",
		"      MMMM      ",
	)
	at(img, cMetronome).rows(pal,
		"",
		"       ll       ",
		"       ll   y   ",
		"      llll y    ",
		"      llllyl    ",
		"     lllllyl    ",
		"     llllylll   ",
		"    lllllllll   ",
		"    llllyyllll  ",
		"   llllllllllll ",
		"   AAAAAAAAAAAA ",
	)
	at(img, cGuitar).rows(pal,
		"             ll ",
		"            ll  ",
		"           l    ",
		"          l     ",
		"         l      ",
		"    rrr l       ",
		"   rrrrrl       ",
		"  rrrrkrr       ",
		"  rrrkkrrr      ",
		"   rrrrrrrr     ",
		"  rrrrrrrr      ",
		"  rrrrrrr       ",
		"   rrrrr        ",
	)
	at(img, cBassIcon).rows(pal,
		"              ll",
		"             ll ",
		"            l   ",
		"           l    ",
		"          l     ",
		"         l      ",
		"    vvv l       ",
		"   vvvvvl       ",
		"  vvvvkvv       ",
		"  vvvkkvvv      ",
		"   vvvvvvvv     ",
		"  vvvvvvvvv     ",
		"  vvvvvvvv      ",
		"   vvvvvv       ",
	)
	at(img, cDrum).rows(pal,
		" d            d ",
		"  d          d  ",
		"   d        d   ",
		"    WWWWWWWW    ",
		"  WWWWWWWWWWWW  ",
		"  rWWWWWWWWWWr  ",
		"  rrrrrrrrrrrr  ",
		"  ryrrryrrryrr  ",
		"  rrryrrryrrrr  ",
		"  rrrrrrrrrrrr  ",
		"  rrrrrrrrrrrr  ",
		"   yyyyyyyyyy   ",
	)
	at(img, cShoe).rows(pal,
		"", "", "", "",
		"     nnnnn      ",
		"     nWnWn      ",
		"     nnnnn      ",
		"     nnnnnnn    ",
		"    nnnnnnnnnn  ",
		"    WWWWWWWWWW  ",
	)
	at(img, cStoneIcon).rows(pal,
		"",
		"            y   ",
		"           yyy  ",
		"            y   ",
		"",
		"     bbbbb      ",
		"    bObbbbb     ",
		"   bbbbbbbBB    ",
		"   bbbbbbBBB  y ",
		"    BBBBBBB     ",
		"",
		"  gggg    pppp  ",
		" gOgggG  pOpppP ",
		"  GGGG    PPPP  ",
	)
}

// art is the sheet, as a bitmap of pixel art.
type art struct{ sheet *ui.Bitmap }

func newArt() art { return art{ui.NewBitmap(sheet()).Pixelated()} }

func src(i int) ui.Rect {
	return ui.Rect{X: float32(i % 8 * cell), Y: float32(i / 8 * cell), W: cell, H: cell}
}

// draw draws cell i into dst, flipped and tinted as asked.
func (a art) draw(p *ui.Painter, i int, dst ui.Rect, o ui.ImageOptions) {
	p.ImageRegion(a.sheet, src(i), dst, o)
}
