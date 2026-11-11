// Farm is a little farming game in native UI: walk a farm with the
// keyboard, till the grass, plant seeds, water them and harvest what
// grows, through a camera that follows you, as days turn to night, with
// sound and music. The game plugin runs it, in fixed steps (game.View),
// and its camera follows the farmer (game.Camera). Everything it draws
// and plays is made by its code: pixel art in a pixelated sheet drawn
// with Painter.ImageRegion, and sounds synthesized for package audio.
//
//	go run ./examples/farm
//	go run ./cmd/mygo dev -web examples/farm
//
// Arrows or WASD walk, Space or E uses the tool (or harvests what is
// ripe), 1 2 3 or Q choose the tool, + and - zoom, and M plays or stops
// the music. With a gamepad: the left stick or the d-pad walks, A uses the
// tool, the shoulder buttons choose it, and Start plays or stops the
// music.
package main

import (
	"fmt"
	"log"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/game"
	"github.com/egoist/mygo/plugins/game/gamepad"
	"github.com/egoist/mygo/ui"
)

// stepTime is the world's fixed step, as game.View runs it: 60 a second,
// whatever the display's rate, so that the game plays the same on every
// display.
const stepTime = 1.0 / game.Rate

// zooms are the camera's zooms: pixels of the screen per pixel of art, in
// DIPs. Not all are whole numbers: pixelated sampling keeps the art's
// pixels square at any.
var zooms = []float32{2, 2.5, 3, 4, 5}

// farm is the game game.View runs: Update steps the world with the
// input, Draw draws it.
type farm struct {
	w     *world
	art   sprites
	snd   *sounds // nil in tests
	zoom  int
	music bool
	// pad is the last state of the first gamepad, to tell the buttons
	// pressed since.
	pad gamepad.State
}

func newFarm(withSound bool) *farm {
	g := &farm{w: newWorld(), art: newSprites(), zoom: 2}
	if withSound {
		g.snd = newSounds()
		g.setMusic(true)
	}
	return g
}

func (g *farm) setMusic(on bool) {
	g.music = on
	if g.snd == nil {
		return
	}
	if on {
		g.snd.music.Play()
	} else {
		g.snd.music.Pause()
	}
}

func (g *farm) view(c *ui.Context) {
	game.View(c, g).Fill().Background(ui.Hex("#4b9a2a")).Label("Farm")
}

// Update steps the world: the keys and the gamepad walk and act, then
// what happened sounds.
func (g *farm) Update(in *game.Input) {
	w := g.w
	w.held = [4]bool{
		up:    in.Down(ui.KeyUp) || in.Down(ui.KeyW),
		down:  in.Down(ui.KeyDown) || in.Down(ui.KeyS),
		left:  in.Down(ui.KeyLeft) || in.Down(ui.KeyA),
		right: in.Down(ui.KeyRight) || in.Down(ui.KeyD),
	}
	switch {
	case in.Pressed(ui.KeySpace), in.Pressed(ui.KeyE), in.Pressed(ui.KeyEnter):
		w.use()
	case in.Pressed(ui.Key1):
		w.tool = hoe
	case in.Pressed(ui.Key2):
		w.tool = seeds
	case in.Pressed(ui.Key3):
		w.tool = can
	case in.Pressed(ui.KeyQ), in.Pressed(ui.KeyTab):
		w.tool = (w.tool + 1) % 3
	case in.Pressed(ui.KeyEqual):
		g.zoom = min(g.zoom+1, len(zooms)-1)
	case in.Pressed(ui.KeyMinus):
		g.zoom = max(g.zoom-1, 0)
	case in.Pressed(ui.KeyM):
		g.setMusic(!g.music)
	}
	g.readPad()
	w.step(stepTime)
	if g.snd != nil && len(w.events) > 0 {
		g.snd.play(w.events, 0)
	}
	w.events = w.events[:0]
}

// readPad reads the first gamepad: its stick or d-pad walks, and its
// buttons act as the keys do when just pressed.
func (g *farm) readPad() {
	pads := gamepad.Pads()
	if len(pads) == 0 {
		g.w.stick, g.pad = [2]float64{}, gamepad.State{}
		return
	}
	s := pads[0].State()
	x, y := s.LeftStick()
	if s.Pressed(gamepad.Left) {
		x = -1
	} else if s.Pressed(gamepad.Right) {
		x = 1
	}
	if s.Pressed(gamepad.Up) {
		y = -1
	} else if s.Pressed(gamepad.Down) {
		y = 1
	}
	if m := math.Hypot(float64(x), float64(y)); m > 1 {
		x, y = float32(float64(x)/m), float32(float64(y)/m)
	}
	g.w.stick = [2]float64{float64(x), float64(y)}
	switch {
	case s.JustPressed(g.pad, gamepad.South):
		g.w.use()
	case s.JustPressed(g.pad, gamepad.RightShoulder):
		g.w.tool = (g.w.tool + 1) % 3
	case s.JustPressed(g.pad, gamepad.LeftShoulder):
		g.w.tool = (g.w.tool + 2) % 3
	case s.JustPressed(g.pad, gamepad.Start):
		g.setMusic(!g.music)
	}
	g.pad = s
}

// tint is the color the world takes at the light of the hour: the night
// is blue.
func (g *farm) tint() ui.Color {
	light, _ := g.w.daylight()
	mix := func(night, day float64) uint8 { return uint8(night + (day-night)*light) }
	return ui.RGB(mix(80, 255), mix(95, 255), mix(160, 255))
}

// Draw draws the world through a camera that follows the farmer, within
// the map, and the HUD over it. alpha places the farmer between their last
// two steps, for motion as smooth as the display's frames.
func (g *farm) Draw(p *ui.Painter, r ui.Rect, alpha float64) {
	w := g.w
	fx := float32(w.px + (w.x-w.px)*alpha)
	fy := float32(w.py + (w.y-w.py)*alpha)
	cam := game.Camera{Zoom: zooms[g.zoom]}
	cam.X = follow(fx*cell, mapW*cell, r.W/cam.Zoom)
	cam.Y = follow(fy*cell, mapH*cell, r.H/cam.Zoom)
	tint := g.tint()
	cam.Draw(p, r, func() { g.drawWorld(p, tint, fx, fy) })
	g.drawHUD(p, r)
}

// follow returns where the camera looks along an axis to follow a point at
// at, within a world as long as world in a view as long as view.
func follow(at, world, view float32) float32 {
	if world <= view {
		return world / 2
	}
	return min(max(at, view/2), world-view/2)
}

// drawWorld draws the map in the art's pixels: the ground, crops, then
// what stands, row by row with the farmer among them, at fx, fy, so that
// what stands south of another covers it.
func (g *farm) drawWorld(p *ui.Painter, tint ui.Color, fx, fy float32) {
	w := g.w
	put := func(i int, x, y float32) {
		s := g.art.cell(i)
		s.tint = tint
		g.art.draw(p, s, ui.Rect{X: x, Y: y, W: cell, H: cell})
	}
	waterFrame := cWater0
	if int(w.now*2)%2 == 1 {
		waterFrame = cWater1
	}
	for y := range mapH {
		for x := range mapW {
			t := &w.tiles[y][x]
			tx, ty := float32(x*cell), float32(y*cell)
			switch t.ground {
			case grass:
				put([]int{cGrass, cGrass2, cFlowers}[t.variant], tx, ty)
			case path:
				put(cPath, tx, ty)
			case soil:
				if t.wet {
					put(cWetSoil, tx, ty)
				} else {
					put(cSoil, tx, ty)
				}
			case water:
				put(waterFrame, tx, ty)
			}
			if t.crop >= 0 {
				put(cSeed+int(t.crop), tx, ty)
			}
		}
	}
	// The tile the tool would act on.
	if tx, ty := w.target(); tx >= 0 && ty >= 0 && tx < mapW && ty < mapH {
		put(cFrame, float32(tx*cell), float32(ty*cell))
	}
	feet := int(math.Floor(float64(fy)))
	for y := range mapH {
		for x := range mapW {
			tx, ty := float32(x*cell), float32(y*cell)
			switch w.tiles[y][x].object {
			case tree:
				put(cTreeTrunk, tx, ty)
				put(cTreeTop, tx, ty-cell+4)
			case bush:
				put(cBush, tx, ty)
			case rock:
				put(cRock, tx, ty)
			case fence:
				put(cFence, tx, ty)
			}
		}
		if y == feet {
			g.art.draw(p, g.farmer(tint), ui.Rect{X: fx*cell - cell/2, Y: fy*cell - cell + 2, W: cell, H: cell})
		}
	}
}

// farmer returns the farmer's sprite: their frame of walking, facing
// where they go, flipped to face left. They move smoothly: under the
// camera's transform, sprites are where they are, not on whole pixels.
func (g *farm) farmer(tint ui.Color) sprite {
	w := g.w
	frame := 0
	if w.moving {
		frame = 1 + int(w.walked/0.35)%2
	}
	var s sprite
	switch w.facing {
	case down:
		s = g.art.cell(cDown + frame)
	case up:
		s = g.art.cell(cUp + frame)
	default:
		s = g.art.cell(cSide + frame)
		s.flipX = w.facing == left
	}
	s.tint = tint
	return s
}

// drawHUD draws, over the world, the coins and the hour, and the toolbar.
func (g *farm) drawHUD(p *ui.Painter, r ui.Rect) {
	w := g.w
	panel := ui.RGBA(20, 18, 30, 0.75)
	white, muted := ui.RGB(255, 255, 255), ui.RGB(200, 200, 215)
	_, hour := w.daylight()
	p.Fill(ui.Rect{X: r.X + 12, Y: r.Y + 12, W: 168, H: 40}, panel, 10)
	g.art.draw(p, g.art.cell(cCoin), ui.Rect{X: r.X + 18, Y: r.Y + 16, W: 32, H: 32})
	p.Text(r.X+54, r.Y+21, fmt.Sprint(w.coins), 18, white)
	p.Text(r.X+110, r.Y+23, fmt.Sprintf("%02d:00", hour), 15, muted)
	// The toolbar, at the bottom, the tool in use framed.
	const slot, gap = 52, 8
	bw := float32(3*slot + 2*gap + 16)
	bx, by := r.X+(r.W-bw)/2, r.Y+r.H-slot-50
	p.Fill(ui.Rect{X: bx, Y: by, W: bw, H: slot + 16}, panel, 12)
	for i, icon := range []int{cHoe, cSeeds, cCan} {
		x := bx + 8 + float32(i)*(slot+gap)
		box := ui.Rect{X: x, Y: by + 8, W: slot, H: slot}
		if tool(i) == w.tool {
			p.Fill(box, ui.RGBA(255, 255, 255, 0.15), 8)
			p.Stroke(box, ui.RGB(251, 242, 54), 8, 2)
		}
		g.art.draw(p, g.art.cell(icon), ui.Rect{X: x + 4, Y: by + 12, W: slot - 8, H: slot - 8})
		p.Text(x+4, by+10, fmt.Sprint(i+1), 11, muted)
	}
	hint := toolNames[w.tool] + " · Space uses it · 1 2 3 or Q switch · + − zoom · M music"
	label := ui.Span{Text: hint, Size: 12, Color: muted}
	tw, th := p.MeasureText(0, label)
	p.Fill(ui.Rect{X: r.X + (r.W-tw)/2 - 10, Y: r.Y + r.H - 30, W: tw + 20, H: th + 10}, panel, (th+10)/2)
	p.RichText(r.X+(r.W-tw)/2, r.Y+r.H-25, 0, label)
}

func main() {
	g := newFarm(true)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Farm",
			Width:   960,
			Height:  640,
			Content: ui.View(g.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
