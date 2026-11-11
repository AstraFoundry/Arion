// Mayoi is a little survivor game in native UI, a fan game after the
// anime BanG Dream! It's MyGO!!!!!, whose band MyGo is named after.
// Tomori, lost, holds out on a street at night against stage fright
// (static, crows and shadows) for as long as she can. She sings on her
// own, at the nearest of it; the player walks her and picks up the stones
// it leaves, which she collects, and at each level chooses one of three
// upgrades. Four bring a bandmate, who joins, follows her and plays their
// part of the music: Anon on guitar, Soyo on bass, Rāna on lead and Taki
// on drums.
//
// The game plugin runs it, in fixed steps (game.View), and its camera
// follows Tomori (game.Camera), shaking when she is hit. Everything it
// draws and plays is made by its code: pixel art in a pixelated sheet,
// drawn with Painter.ImageRegion, and sounds and an original lo-fi tune
// synthesized for package audio. The website's page of the game plugin
// plays it.
//
//	go run ./examples/mayoi
//	go run ./cmd/mygo dev -web examples/mayoi
//
// Arrows or WASD walk, Esc pauses and M plays or stops the music; at a
// level, 1 2 3, a click, or the arrows and Space choose. With a gamepad:
// the left stick or the d-pad walks, A chooses, Start pauses.
package main

import (
	"cmp"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/game"
	"github.com/egoist/mygo/plugins/game/gamepad"
	"github.com/egoist/mygo/ui"
)

// stepTime is the world's fixed step, as game.View runs it.
const stepTime = 1.0 / game.Rate

// mayoi is the game game.View runs: Update steps the world with the
// input, Draw draws it.
type mayoi struct {
	w        *world
	art      art
	snd      *sounds // nil in tests
	music    bool    // the player wants it (M)
	sounding bool    // it plays
	paused   bool    // by the player
	frozen   bool    // the world did not step: nothing moves between steps
	steps    float64
	pick     int // the upgrade chosen with keys or a pad
	pad      gamepad.State
	rng      *rand.Rand
}

func newMayoi(withSound bool) *mayoi {
	g := &mayoi{w: newWorld(rand.Uint64()), art: newArt(), music: true, rng: rand.New(rand.NewPCG(1, 1))}
	if withSound {
		g.snd = newSounds()
	}
	return g
}

func (g *mayoi) view(c *ui.Context) {
	game.View(c, g).Fill().Background(ui.Hex("#272c3f")).Label("Mayoi")
}

func (g *mayoi) start() {
	g.w.reset()
	g.paused, g.frozen, g.pick = false, false, 0
}

// playMusic plays or pauses the music, as the game wants it.
func (g *mayoi) playMusic(on bool) {
	if on == g.sounding {
		return
	}
	g.sounding = on
	if g.snd == nil {
		return
	}
	if on {
		g.snd.music.Play()
	} else {
		g.snd.music.Pause()
	}
}

// Update reads the keys, the pointer and the gamepad, and steps the world
// unless it waits for the player, then plays what happened.
func (g *mayoi) Update(in *game.Input) {
	w := g.w
	g.steps++
	pad, prev := g.readPad()
	padPressed := func(b gamepad.Button) bool { return pad.JustPressed(prev, b) }
	confirm := in.Pressed(ui.KeySpace) || in.Pressed(ui.KeyEnter) || padPressed(gamepad.South)
	if in.Pressed(ui.KeyM) {
		g.music = !g.music
	}
	g.frozen = true
	switch w.state {
	case title:
		if confirm || in.PointerPressed(0) || padPressed(gamepad.Start) {
			g.start()
		}
	case over:
		if w.overTime > 0.8 && (confirm || in.PointerPressed(0) || padPressed(gamepad.Start)) {
			g.start()
		}
		w.step(stepTime)
	case choosing:
		g.choose(in, pad, prev, confirm)
	case playing:
		if in.Pressed(ui.KeyEscape) || in.Pressed(ui.KeyP) || padPressed(gamepad.Start) {
			g.paused = !g.paused
		}
		if g.paused && (confirm || in.PointerPressed(0)) {
			g.paused = false
		}
		var move vec
		if in.Down(ui.KeyLeft) || in.Down(ui.KeyA) || pad.Pressed(gamepad.Left) {
			move.x--
		}
		if in.Down(ui.KeyRight) || in.Down(ui.KeyD) || pad.Pressed(gamepad.Right) {
			move.x++
		}
		if in.Down(ui.KeyUp) || in.Down(ui.KeyW) || pad.Pressed(gamepad.Up) {
			move.y--
		}
		if in.Down(ui.KeyDown) || in.Down(ui.KeyS) || pad.Pressed(gamepad.Down) {
			move.y++
		}
		sx, sy := pad.LeftStick()
		move = move.add(vec{float64(sx), float64(sy)})
		if move.len() > 1 {
			move = move.norm()
		}
		w.move = move
		// Without the keyboard, the player is elsewhere: wait for them.
		if !g.paused && in.Focused() {
			w.step(stepTime)
			g.frozen = false
		}
	}
	// The music plays with the game: not before it, nor while it waits for
	// a player who paused it or is elsewhere.
	g.playMusic(g.music && w.state != title && in.Focused() && !(w.state == playing && g.paused))
	if g.snd != nil {
		g.snd.setBand(w.band)
		if len(w.events) > 0 {
			g.snd.play(w.events, g.steps)
		}
	}
	w.events = w.events[:0]
}

// choose takes the upgrade the player picks: by its number, a click, or
// moving to it and confirming.
func (g *mayoi) choose(in *game.Input, pad, prev gamepad.State, confirm bool) {
	w := g.w
	n := len(w.choices)
	for i, k := range []ui.Key{ui.Key1, ui.Key2, ui.Key3} {
		if i < n && in.Pressed(k) {
			g.pick = i
			confirm = true
		}
	}
	left := in.Pressed(ui.KeyLeft) || in.Pressed(ui.KeyA) || pad.JustPressed(prev, gamepad.Left)
	right := in.Pressed(ui.KeyRight) || in.Pressed(ui.KeyD) || pad.JustPressed(prev, gamepad.Right)
	px, _ := prev.LeftStick()
	x, _ := pad.LeftStick()
	left = left || px > -0.5 && x <= -0.5
	right = right || px < 0.5 && x >= 0.5
	if left {
		g.pick = (g.pick + n - 1) % n
	}
	if right {
		g.pick = (g.pick + 1) % n
	}
	vw, vh := in.Size()
	if x, y, over := in.Pointer(); over {
		for i, c := range cards(n, ui.Rect{W: vw, H: vh}) {
			if x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H {
				g.pick = i
				if in.PointerPressed(0) {
					confirm = true
				}
			}
		}
	}
	if confirm {
		w.choose(g.pick)
		g.pick = 0
	}
}

// readPad reads the first gamepad, returning it and how it was the step
// before.
func (g *mayoi) readPad() (s, prev gamepad.State) {
	prev = g.pad
	if pads := gamepad.Pads(); len(pads) > 0 {
		s = pads[0].State()
	}
	g.pad = s
	return s, prev
}

// zoom returns how many DIPs a pixel of the art takes: a whole number of
// the screen's, about 200 of them across the view's shorter side.
func zoom(p *ui.Painter, r ui.Rect) float32 {
	s := p.Scale()
	k := max(2, float32(math.Round(float64(min(r.W, r.H)*s/200))))
	return k / s
}

// Draw draws the world through a camera that follows Tomori, then the HUD
// and what waits for the player over it.
func (g *mayoi) Draw(p *ui.Painter, r ui.Rect, alpha float64) {
	w := g.w
	if g.frozen {
		alpha = 1
	}
	z := zoom(p, r)
	w.view = vec{float64(r.W / 2 / z), float64(r.H / 2 / z)}
	tomori := w.prev.lerp(w.pos, alpha)
	cam := game.Camera{X: float32(tomori.x), Y: float32(tomori.y - 6), Zoom: z}
	if w.shake > 0 {
		cam.X += float32((g.rng.Float64()*2 - 1) * w.shake * 10)
		cam.Y += float32((g.rng.Float64()*2 - 1) * w.shake * 10)
	}
	cam.Draw(p, r, func() { g.drawWorld(p, cam.Visible(r), alpha, tomori) })
	// Numbers of damage, in the screen's pixels, sharp at any zoom.
	for _, pu := range w.popups {
		x := r.X + r.W/2 + (float32(pu.pos.x)-cam.X)*z
		y := r.Y + r.H/2 + (float32(pu.pos.y-pu.t*20)-cam.Y)*z
		fade := float32(1 - pu.t/0.6)
		center(p, x+1, y+1, ui.Span{Text: pu.text, Size: 13, Weight: 700, Color: ui.RGBA(0, 0, 0, fade*0.6)})
		center(p, x, y, ui.Span{Text: pu.text, Size: 13, Weight: 700, Color: ui.RGBA(255, 240, 200, fade)})
	}
	if w.state != title {
		g.drawHUD(p, r)
	}
	switch {
	case w.state == title:
		g.drawTitle(p, r)
	case w.state == choosing:
		g.drawChoices(p, r)
	case w.state == over:
		g.drawOver(p, r)
	case g.frozen:
		dim(p, r)
		center(p, r.X+r.W/2, r.Y+r.H/2-24, ui.Span{Text: "Paused", Size: 28, Weight: 700, Color: white})
		center(p, r.X+r.W/2, r.Y+r.H/2+14, ui.Span{Text: "Click or press Space to play", Size: 14, Color: muted})
	}
}

// tileAt is the street's tile at x, y: paving, here and there sakura
// petals, a crack or a drain.
func tileAt(x, y int) int {
	n := uint64(int64(x))*0x9e3779b97f4a7c15 ^ uint64(int64(y))*0xc2b2ae3d27d4eb4f
	n = (n ^ n>>31) * 0xbf58476d1ce4e5b9
	switch v := (n ^ n>>27) % 100; {
	case v < 3:
		return cPetals
	case v < 5:
		return cPetals2
	case v < 7:
		return cCrack
	case v < 8:
		return cGrate
	case v < 50:
		return cPave2
	}
	return cPave
}

// drawWorld draws what the camera sees of the world, in its pixels: the
// street, the stones, the beat, the enemies and the band back to front,
// then the picks and the notes.
func (g *mayoi) drawWorld(p *ui.Painter, vis ui.Rect, a float64, tomori vec) {
	w := g.w
	none := ui.ImageOptions{}
	x0, y0 := int(math.Floor(float64(vis.X/cell)))-1, int(math.Floor(float64(vis.Y/cell)))-1
	x1, y1 := int(math.Ceil(float64((vis.X+vis.W)/cell)))+1, int(math.Ceil(float64((vis.Y+vis.H)/cell)))+1
	for ty := y0; ty < y1; ty++ {
		for tx := x0; tx < x1; tx++ {
			g.art.draw(p, tileAt(tx, ty), ui.Rect{X: float32(tx * cell), Y: float32(ty * cell), W: cell, H: cell}, none)
		}
	}
	box := func(v vec, dy float32) ui.Rect {
		return ui.Rect{X: float32(v.x) - cell/2, Y: float32(v.y) - cell + dy, W: cell, H: cell}
	}
	if w.beatFlash > 0 {
		t := w.beatFlash / 0.4
		rr := float32(w.beatRadius() * (1 - 0.4*t))
		ring := ui.Rect{X: float32(tomori.x) - rr, Y: float32(tomori.y-5) - rr, W: 2 * rr, H: 2 * rr}
		p.Fill(ring, ui.RGBA(160, 90, 214, float32(t)*0.18), rr)
		p.Stroke(ring, ui.RGBA(220, 190, 255, float32(t)), rr, 2)
	}
	for i, s := range w.stones {
		c := cStone
		switch {
		case s.value >= 5:
			c = cStone + 2
		case s.value >= 2:
			c = cStone + 1
		}
		bob := float32(math.Sin(g.steps*0.08+float64(i)) * 1)
		g.art.draw(p, c, box(s.prev.lerp(s.pos, a), 4+bob), none)
	}
	for _, pu := range w.puffs {
		g.art.draw(p, cPuff+min(int(pu.t/0.32*3), 2), box(pu.pos, 4), ui.ImageOptions{Tint: ui.RGBA(255, 255, 255, float32(1-pu.t/0.32))})
	}
	// The band and the enemies, by their feet, back to front.
	type figure struct {
		at     vec
		enemy  int // or -1 for a bandmate
		member int
	}
	figures := make([]figure, 0, len(w.enemies)+memberCount)
	walking := w.move.len() > 0.05 && w.state == playing
	if w.state != title {
		figures = append(figures, figure{at: tomori, enemy: -1, member: vocalist})
		for i, m := range w.band {
			figures = append(figures, figure{at: w.follower(i), enemy: -1, member: m})
		}
	}
	for i := range w.enemies {
		figures = append(figures, figure{at: w.enemies[i].prev.lerp(w.enemies[i].pos, a), enemy: i})
	}
	slices.SortFunc(figures, func(x, y figure) int { return cmp.Compare(x.at.y, y.at.y) })
	for _, f := range figures {
		if f.enemy < 0 {
			if f.member == vocalist && w.hurt > 0 && int(w.hurt*14)%2 == 0 {
				continue // blinking, unhurtable
			}
			frame := 0
			if walking {
				frame = int(w.walked/7+float64(f.member)) % 2
			}
			left := w.facingLeft
			if f.member != vocalist {
				left = f.at.x > tomori.x+2 || (f.at.x >= tomori.x-2 && w.facingLeft)
			}
			g.art.draw(p, cMember+f.member*2+frame, box(f.at, 1), ui.ImageOptions{FlipX: left})
			continue
		}
		e := &w.enemies[f.enemy]
		c, speed := cStatic, 6.0
		switch e.kind {
		case crow:
			c, speed = cCrow, 9
		case shadow:
			c, speed = cShadow, 3
		}
		o := ui.ImageOptions{FlipX: f.at.x > tomori.x}
		if e.flash > 0 {
			o.Tint = ui.RGB(255, 140, 170)
		}
		g.art.draw(p, c+int(e.phase*speed)%2, box(f.at, 1), o)
	}
	if n := w.levels[upPicks]; n > 0 {
		for i := range n {
			b := w.pickPos(i, n, tomori, w.pickAngle)
			deg := float32(w.pickAngle*180/math.Pi) + float32(i)*360/float32(n)
			p.Transform(ui.Rotation(deg), float32(b.x), float32(b.y), func() {
				g.art.draw(p, cPick, ui.Rect{X: float32(b.x) - 6, Y: float32(b.y) - 7, W: 12, H: 12}, none)
			})
		}
	}
	for i, n := range w.sung {
		at := n.prev.lerp(n.pos, a)
		wob := float32(math.Sin(g.steps*0.3+float64(i)) * 1.5)
		g.art.draw(p, cNote, ui.Rect{X: float32(at.x) - 7, Y: float32(at.y) - 9 + wob, W: 14, H: 14}, none)
	}
}

var (
	white = ui.RGB(255, 255, 255)
	muted = ui.RGB(200, 196, 220)
	gold  = ui.RGB(255, 217, 90)
	pink  = ui.RGB(246, 184, 207)
)

// center draws a line of text centered at x.
func center(p *ui.Painter, x, y float32, s ui.Span) {
	w, _ := p.MeasureText(0, s)
	p.RichText(x-w/2, y, 0, s)
}

func dim(p *ui.Painter, r ui.Rect) { p.Fill(r, ui.RGBA(14, 12, 28, 0.7), 0) }

func clock(sec float64) string { return fmt.Sprintf("%d:%02d", int(sec)/60, int(sec)%60) }

// drawHUD draws the experience bar, the band-aids, the time, the band,
// the stones collected, and who just joined.
func (g *mayoi) drawHUD(p *ui.Painter, r ui.Rect) {
	w := g.w
	bar := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 8}
	p.Fill(bar, ui.RGBA(0, 0, 0, 0.55), 0)
	bar.W *= float32(w.xp) / float32(w.xpNext)
	p.Fill(bar, pink, 0)
	for i := range w.maxHP {
		c := cBandaid
		if i >= w.hp {
			c = cBandaidEmpty
		}
		g.art.draw(p, c, ui.Rect{X: r.X + 6 + float32(i)*24, Y: r.Y + 8, W: 30, H: 30}, ui.ImageOptions{})
	}
	center(p, r.X+r.W/2, r.Y+16, ui.Span{Text: clock(w.time), Size: 18, Weight: 700, Color: white})
	center(p, r.X+r.W/2, r.Y+40, ui.Span{Text: fmt.Sprintf("Lv %d", w.level), Size: 13, Weight: 600, Color: gold})
	stones := ui.Span{Text: fmt.Sprint(w.collected), Size: 16, Weight: 700, Color: white}
	sw, _ := p.MeasureText(0, stones)
	p.RichText(r.X+r.W-14-sw, r.Y+17, 0, stones)
	g.art.draw(p, cStone+1, ui.Rect{X: r.X + r.W - 46 - sw, Y: r.Y + 4, W: 32, H: 32}, ui.ImageOptions{})
	// The band so far, by the bottom left corner: who has yet to join is a
	// silhouette.
	for m := range memberCount {
		o := ui.ImageOptions{Tint: ui.RGBA(0, 0, 0, 0.45)}
		if m == vocalist || w.joined(m) {
			o = ui.ImageOptions{}
		}
		g.art.draw(p, cMember+m*2, ui.Rect{X: r.X + 8 + float32(m)*30, Y: r.Y + r.H - 42, W: 32, H: 32}, o)
	}
	if w.bannerTime > 0 {
		fade := float32(min(w.bannerTime/0.4, 1))
		span := ui.Span{Text: w.banner, Size: 20, Weight: 800, Color: ui.RGBA(255, 217, 90, fade)}
		bw, bh := p.MeasureText(0, span)
		y := r.Y + r.H*0.24
		p.Fill(ui.Rect{X: r.X + (r.W-bw)/2 - 16, Y: y - 8, W: bw + 32, H: bh + 16}, ui.RGBA(14, 12, 28, 0.75*fade), (bh+16)/2)
		p.RichText(r.X+(r.W-bw)/2, y, 0, span)
	}
}

func (g *mayoi) drawTitle(p *ui.Painter, r ui.Rect) {
	dim(p, r)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	for m := range memberCount {
		bob := float32(math.Sin(g.steps*0.08+float64(m)*1.2) * 3)
		x := cx + (float32(m)-2)*52 - 20
		g.art.draw(p, cMember+m*2+int(g.steps/20+float64(m))%2, ui.Rect{X: x, Y: cy - 128 + bob, W: 40, H: 40}, ui.ImageOptions{FlipX: m > 2})
		center(p, x+20, cy-84, ui.Span{Text: memberNames[m], Size: 11, Weight: 600, Color: muted})
	}
	center(p, cx, cy-58, ui.Span{Text: "MAYOI", Size: 44, Weight: 800, Color: white, LetterSpacing: 8})
	center(p, cx, cy-6, ui.Span{Text: "迷子でもいい、前へ進め", Size: 15, Color: pink})
	center(p, cx, cy+18, ui.Span{Text: "Hold out against stage fright, gather stones, and find your band.", Size: 14, Color: muted})
	if int(g.steps/30)%2 == 0 {
		center(p, cx, cy+60, ui.Span{Text: "Click or press Space to start", Size: 17, Weight: 700, Color: gold})
	}
	center(p, cx, r.Y+r.H-34, ui.Span{Text: "A MyGO!!!!! fan game · WASD or arrows walk · Esc pauses · M music · gamepads too", Size: 12, Color: muted})
}

// cards returns where the upgrades offered show in the view r.
func cards(n int, r ui.Rect) []ui.Rect {
	const gap, ch = 14, 176
	cw := min(200, (r.W-40-gap*2)/3)
	total := float32(n)*cw + float32(n-1)*gap
	x, y := r.X+(r.W-total)/2, r.Y+r.H/2-ch/2+16
	out := make([]ui.Rect, n)
	for i := range out {
		out[i] = ui.Rect{X: x + float32(i)*(cw+gap), Y: y, W: cw, H: ch}
	}
	return out
}

func (g *mayoi) drawChoices(p *ui.Painter, r ui.Rect) {
	w := g.w
	dim(p, r)
	cs := cards(len(w.choices), r)
	if len(cs) > 0 {
		center(p, r.X+r.W/2, cs[0].Y-50, ui.Span{Text: fmt.Sprintf("Level %d!", w.level), Size: 28, Weight: 800, Color: gold})
	}
	for i, c := range cs {
		u := upgrades[w.choices[i]]
		p.Fill(c, ui.RGBA(40, 34, 64, 0.96), 12)
		if i == g.pick {
			p.Stroke(c, gold, 12, 2)
		}
		p.Text(c.X+10, c.Y+8, fmt.Sprint(i+1), 12, muted)
		cx := c.X + c.W/2
		icon := ui.Rect{X: cx - 24, Y: c.Y + 14, W: 48, H: 48}
		if m := u.member; m >= 0 && !w.joined(m) {
			// Who joins, with what they play.
			g.art.draw(p, cMember+m*2, ui.Rect{X: cx - 44, Y: c.Y + 14, W: 48, H: 48}, ui.ImageOptions{})
			icon = ui.Rect{X: cx + 2, Y: c.Y + 26, W: 36, H: 36}
		}
		g.art.draw(p, u.icon, icon, ui.ImageOptions{})
		center(p, cx, c.Y+72, ui.Span{Text: u.name, Size: 15, Weight: 700, Color: white})
		for j, line := range strings.Split(u.desc, "\n") {
			center(p, cx, c.Y+98+float32(j)*17, ui.Span{Text: line, Size: 12, Color: muted})
		}
		lv := w.levels[w.choices[i]]
		label := "New"
		if lv > 0 {
			label = fmt.Sprintf("Level %d → %d", lv, lv+1)
		}
		center(p, cx, c.Y+c.H-24, ui.Span{Text: label, Size: 11, Weight: 600, Color: gold})
	}
}

func (g *mayoi) drawOver(p *ui.Painter, r ui.Rect) {
	w := g.w
	dim(p, r)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	center(p, cx, cy-70, ui.Span{Text: "It's okay to be lost.", Size: 30, Weight: 800, Color: white})
	center(p, cx, cy-22, ui.Span{Text: fmt.Sprintf("You held out %s with a band of %d, reached level %d,", clock(w.time), len(w.band)+1, w.level), Size: 15, Color: muted})
	center(p, cx, cy+2, ui.Span{Text: fmt.Sprintf("calmed %d frights and collected %d stones.", w.calmed, w.collected), Size: 15, Color: muted})
	if w.overTime > 0.8 {
		center(p, cx, cy+52, ui.Span{Text: "Click or press Space to go on", Size: 17, Weight: 700, Color: gold})
	}
}

func main() {
	g := newMayoi(true)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mayoi",
			Width:   960,
			Height:  640,
			Content: ui.View(g.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
