package main

import (
	"image"
	"image/color"
	"math/rand/v2"

	"github.com/egoist/mygo/ui"
)

// The sheet is 8×8 cells of 16 pixels, drawn here pixel by pixel so that
// the example needs no files: tiles, crops, the farmer and the toolbar's
// icons.
const cell = 16

// Cells of the sheet.
const (
	cGrass = iota
	cGrass2
	cFlowers
	cPath
	cSoil
	cWetSoil
	cWater0
	cWater1
	cTreeTop
	cBush
	cRock
	cFence
	cSeed
	cSprout
	cGrowing
	cRipe
	cTreeTrunk
	cHoe
	cSeeds
	cCan
	cCoin
	cFrame
	cDrop
	_
	// The farmer: 3 frames walking down, up and sideways (facing right).
	cDown
	_
	_
	cUp
	_
	_
	cSide
)

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

// canvas draws into one cell of the sheet.
type canvas struct {
	img    *image.RGBA
	ox, oy int
}

func (c canvas) px(x, y int, col color.RGBA) {
	if x >= 0 && y >= 0 && x < cell && y < cell {
		c.img.SetRGBA(c.ox+x, c.oy+y, col)
	}
}

func (c canvas) rect(x, y, w, h int, col color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c.px(i, j, col)
		}
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

func at(img *image.RGBA, i int) canvas { return canvas{img, i % 8 * cell, i / 8 * cell} }

// sheet draws the sheet.
func sheet() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 8*cell, 8*cell))
	rng := rand.New(rand.NewPCG(1, 2))
	grass, grassDark, grassLight := rgb(0x6abe30), rgb(0x4b9a2a), rgb(0x8fd14f)
	speckle := func(c canvas, base color.RGBA, spots ...color.RGBA) {
		c.rect(0, 0, cell, cell, base)
		for range 18 {
			c.px(rng.IntN(cell), rng.IntN(cell), spots[rng.IntN(len(spots))])
		}
	}
	speckle(at(img, cGrass), grass, grassDark, grassLight)
	g2 := at(img, cGrass2)
	speckle(g2, grass, grassDark, grassLight)
	for _, p := range [][2]int{{3, 5}, {4, 4}, {5, 5}, {10, 11}, {11, 10}, {12, 11}} {
		g2.px(p[0], p[1], grassDark)
		g2.px(p[0], p[1]+1, grassDark)
	}
	fl := at(img, cFlowers)
	speckle(fl, grass, grassDark, grassLight)
	for _, p := range [][3]int{{4, 4, 0xfbf236}, {11, 6, 0xffffff}, {6, 11, 0xd95763}, {12, 12, 0xfbf236}} {
		col := rgb(uint32(p[2]))
		fl.px(p[0], p[1], col)
		fl.px(p[0]-1, p[1], col)
		fl.px(p[0]+1, p[1], col)
		fl.px(p[0], p[1]-1, col)
		fl.px(p[0], p[1]+1, col)
		fl.px(p[0], p[1], rgb(0xdf7126))
	}
	speckle(at(img, cPath), rgb(0xd9a066), rgb(0xc08a52), rgb(0xe8b77e))
	soil := at(img, cSoil)
	speckle(soil, rgb(0x8f563b), rgb(0x7a4630), rgb(0x9c6545))
	for y := 2; y < cell; y += 4 {
		soil.rect(1, y, 14, 1, rgb(0x6e3f2a))
	}
	wet := at(img, cWetSoil)
	speckle(wet, rgb(0x5a3426), rgb(0x4a2a1e), rgb(0x663c2c))
	for y := 2; y < cell; y += 4 {
		wet.rect(1, y, 14, 1, rgb(0x3e2218))
	}
	for f, i := range []int{cWater0, cWater1} {
		w := at(img, i)
		w.rect(0, 0, cell, cell, rgb(0x5b6ee1))
		for y := 2; y < cell; y += 5 {
			x := (y*3 + f*4) % cell
			w.rect(x, y, 4, 1, rgb(0x8ea3f2))
			w.rect((x+8)%cell, y+2, 3, 1, rgb(0x4a5bc8))
		}
	}
	leaf, leafDark, leafLight := rgb(0x37946e), rgb(0x2a6b50), rgb(0x5fb68a)
	at(img, cTreeTop).rows(map[byte]color.RGBA{'g': leaf, 'd': leafDark, 'l': leafLight},
		"                ",
		"     gggggg     ",
		"   ggllgggggg   ",
		"  gglllggggggg  ",
		" ggllgggggggdgg ",
		" gglggggggggddg ",
		"gggggggggggdddgg",
		"ggggggggggddddgg",
		"gggggggggdddddgg",
		"ggdgggggddddddgg",
		" gddggddddddddg ",
		" ggddddddddddgg ",
		"  ggddddddddgg  ",
		"   gggddddggg   ",
		"     gggggg     ",
		"                ")
	trunk := map[byte]color.RGBA{'b': rgb(0x8f563b), 'd': rgb(0x663931), 's': rgb(0x000000)}
	trunk['s'] = color.RGBA{0, 0, 0, 70}
	at(img, cTreeTrunk).rows(trunk,
		"      bbbb      ",
		"      bbdb      ",
		"      bbdb      ",
		"      bbdb      ",
		"      bbdb      ",
		"     bbbddb     ",
		"    bbb  ddb    ",
		"   ssssssssss   ")
	at(img, cBush).rows(map[byte]color.RGBA{'g': leaf, 'd': leafDark, 'l': leafLight, 'r': rgb(0xd95763), 's': {0, 0, 0, 60}},
		"                ",
		"                ",
		"                ",
		"     gggggg     ",
		"   ggllggggggg  ",
		"  gglgggrgggdgg ",
		"  gggggggggddgg ",
		" ggrgggggdddrdgg",
		" gggggggddddddgg",
		" ggggggdddddddgg",
		"  ggddddddddddg ",
		"   ggdddddddgg  ",
		"    sssssssss   ")
	at(img, cRock).rows(map[byte]color.RGBA{'g': rgb(0x9badb7), 'd': rgb(0x696a6a), 'l': rgb(0xcbdbfc), 's': {0, 0, 0, 60}},
		"                ",
		"                ",
		"                ",
		"                ",
		"                ",
		"      gggg      ",
		"    gglllggg    ",
		"   gglgggggdg   ",
		"  gglggggggddg  ",
		"  ggggggggdddg  ",
		"  gggggddddddg  ",
		"   gdddddddddg  ",
		"    sssssssss   ")
	at(img, cFence).rows(map[byte]color.RGBA{'b': rgb(0xd9a066), 'd': rgb(0x8f563b)},
		"                ",
		"                ",
		"                ",
		"  bb        bb  ",
		"  bd        bd  ",
		"bbbbbbbbbbbbbbbb",
		"dddddddddddddddd",
		"  bd        bd  ",
		"  bd        bd  ",
		"bbbbbbbbbbbbbbbb",
		"dddddddddddddddd",
		"  bd        bd  ",
		"  bd        bd  ",
		"  bd        bd  ")
	crop := map[byte]color.RGBA{'s': rgb(0xeec39a), 'g': rgb(0x6abe30), 'd': rgb(0x37946e), 'r': rgb(0xd95763), 'o': rgb(0xac3232), 'h': rgb(0xf7a6b0)}
	at(img, cSeed).rows(crop,
		"", "", "", "", "", "", "",
		"     s    s     ",
		"                ",
		"        s       ",
		"   s            ",
		"           s    ")
	at(img, cSprout).rows(crop,
		"", "", "", "", "", "",
		"       g g      ",
		"       ggg      ",
		"        g       ",
		"        g       ",
		"        d       ")
	at(img, cGrowing).rows(crop,
		"", "", "",
		"      g   g     ",
		"     ggg ggg    ",
		"      ggggg     ",
		"    g  ggg  g   ",
		"    ggggggggg   ",
		"      ggggg     ",
		"        g       ",
		"        d       ",
		"        d       ")
	at(img, cRipe).rows(crop,
		"",
		"      g   g     ",
		"     ggg ggg    ",
		"      ggdgg     ",
		"     rrrrrrr    ",
		"    rrhrrrrrr   ",
		"   rrhrrrrrrrr  ",
		"   rrrrrrrrrrr  ",
		"   rrrrrrrrror  ",
		"   rrrrrrrrroo  ",
		"    rrrrrrrroo  ",
		"     rrooooo    ",
		"       ddd      ")
	at(img, cHoe).rows(map[byte]color.RGBA{'m': rgb(0x9badb7), 'l': rgb(0xcbdbfc), 'b': rgb(0x8f563b), 'd': rgb(0x663931)},
		"",
		"   mmmmm        ",
		"  mllllmm       ",
		"  mmmmmbmm      ",
		"        bd      ",
		"         bd     ",
		"          bd    ",
		"           bd   ",
		"            bd  ",
		"             bd ",
		"              b ")
	at(img, cSeeds).rows(map[byte]color.RGBA{'p': rgb(0xeec39a), 'd': rgb(0xd9a066), 'g': rgb(0x6abe30), 's': rgb(0x663931)},
		"",
		"     dddddd     ",
		"     pppppp     ",
		"    pppppppp    ",
		"   pppggppppp   ",
		"   ppgggggppp   ",
		"   pppgggpppp   ",
		"   pppppppppp   ",
		"   pppppppppp   ",
		"   ppsppsppsp   ",
		"   pppppppppp   ",
		"    dddddddd    ")
	at(img, cCan).rows(map[byte]color.RGBA{'m': rgb(0x5b6ee1), 'l': rgb(0x8ea3f2), 'd': rgb(0x3f3f74)},
		"",
		"",
		"      mmmmm     ",
		"     m     m    ",
		" m   mmmmmmm    ",
		"  m mllmmmmmm   ",
		"   mmlmmmmmmm   ",
		"    mlmmmmmmmm  ",
		"    mmmmmmmmmm  ",
		"    mmmmmmmmdd  ",
		"    dddddddddd  ")
	at(img, cCoin).rows(map[byte]color.RGBA{'y': rgb(0xfbf236), 'o': rgb(0xdf7126), 'l': rgb(0xffffff)},
		"",
		"",
		"     oooooo     ",
		"    oyyyyyyo    ",
		"   oyllyyyyyo   ",
		"   oylyyoyyyo   ",
		"   oyyyyoyyyo   ",
		"   oyyyyoyyyo   ",
		"   oyyyyoyyyo   ",
		"   oyyyyyyyyo   ",
		"    oyyyyyyo    ",
		"     oooooo     ")
	frame := at(img, cFrame)
	white := color.RGBA{255, 255, 255, 220}
	for i := range 4 {
		frame.px(i, 0, white)
		frame.px(0, i, white)
		frame.px(15-i, 0, white)
		frame.px(15, i, white)
		frame.px(i, 15, white)
		frame.px(0, 15-i, white)
		frame.px(15-i, 15, white)
		frame.px(15, 15-i, white)
	}
	at(img, cDrop).rows(map[byte]color.RGBA{'b': rgb(0x8ea3f2), 'l': rgb(0xffffff)},
		"", "", "",
		"       b        ",
		"      bbb       ",
		"     bbbbb      ",
		"     blbbb      ",
		"     bbbbb      ",
		"      bbb       ")
	farmer(img)
	return img
}

// farmer draws the farmer walking: 3 frames down, up and sideways.
func farmer(img *image.RGBA) {
	pal := map[byte]color.RGBA{
		'h': rgb(0xe8c15a), 'H': rgb(0xb48a2f), 'k': rgb(0xf2c79a), 'e': rgb(0x222034),
		'r': rgb(0x8f563b), 'b': rgb(0x4a7bd1), 'B': rgb(0x30559c), 'p': rgb(0x6b4a2f), 's': rgb(0x3b2a20),
		'S': {0, 0, 0, 60},
	}
	head := map[string][]string{
		"down": {
			"     hhhhhh     ",
			"    hhhhhhhh    ",
			"  HHHHHHHHHHHH  ",
			"     kkkkkk     ",
			"     kekkek     ",
			"     kkkkkk     ",
		},
		"up": {
			"     hhhhhh     ",
			"    hhhhhhhh    ",
			"  HHHHHHHHHHHH  ",
			"     rrrrrr     ",
			"     rrrrrr     ",
			"     krrrrk     ",
		},
		"side": {
			"     hhhhhh     ",
			"    hhhhhhhh    ",
			"   HHHHHHHHHHHHH",
			"    rrrkkkk     ",
			"    rrkkkekk    ",
			"    rrkkkkk     ",
		},
	}
	body := map[string][]string{
		"down": {
			"    bbbbbbbb    ",
			"   kbbbbbbbbk   ",
			"   kbBbbbbBbk   ",
			"    bbbbbbbb    ",
			"    pppppppp    ",
		},
		"side": {
			"     bbbbbb     ",
			"     bbbbbbb    ",
			"     bbkbbbb    ",
			"     bbkbbb     ",
			"     pppppp     ",
		},
	}
	body["up"] = body["down"]
	legs := [3][]string{
		{"    pp    pp    ", "    ss    ss    ", "   SSSSSSSSSS   "},
		{"    pp     pp   ", "    ss     ss   ", "   SSSSSSSSSS   "},
		{"   pp     pp    ", "   ss     ss    ", "   SSSSSSSSSS   "},
	}
	for d, name := range []string{"down", "up", "side"} {
		for f := range 3 {
			art := append(append(append([]string{"", ""}, head[name]...), body[name]...), legs[f]...)
			at(img, cDown+d*3+f).rows(pal, art...)
		}
	}
}

// sprites cuts the sheet into the sprites the game draws.
type sprites struct {
	sheet *ui.Bitmap
}

func newSprites() sprites { return sprites{ui.NewBitmap(sheet()).Pixelated()} }

// A sprite is a cell of the sheet, drawn flipped and tinted as the game
// needs.
type sprite struct {
	src   ui.Rect
	flipX bool
	tint  ui.Color
}

func (s sprites) cell(i int) sprite {
	return sprite{src: ui.Rect{X: float32(i % 8 * cell), Y: float32(i / 8 * cell), W: cell, H: cell}}
}

// draw draws sp stretched into dst, a part of the sheet (ImageRegion).
func (s sprites) draw(p *ui.Painter, sp sprite, dst ui.Rect) {
	p.ImageRegion(s.sheet, sp.src, dst, ui.ImageOptions{FlipX: sp.flipX, Tint: sp.tint})
}
