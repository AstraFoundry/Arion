package main

import (
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
)

// A vec is a point or a direction in the world, in pixels of the art.
type vec struct{ x, y float64 }

func (a vec) add(b vec) vec             { return vec{a.x + b.x, a.y + b.y} }
func (a vec) sub(b vec) vec             { return vec{a.x - b.x, a.y - b.y} }
func (a vec) mul(k float64) vec         { return vec{a.x * k, a.y * k} }
func (a vec) len() float64              { return math.Hypot(a.x, a.y) }
func (a vec) lerp(b vec, t float64) vec { return a.add(b.sub(a).mul(t)) }

// norm returns a with a length of 1, or nothing for nothing.
func (a vec) norm() vec {
	if l := a.len(); l > 0 {
		return a.mul(1 / l)
	}
	return vec{}
}

// The kinds of stage fright, which come as time goes on: static, the
// noise in her head; crows, the whispers; shadows, the doubts.
type kind int

const (
	static kind = iota
	crow
	shadow
)

var kinds = [...]struct {
	hp, speed, radius float64
	stones            int
}{
	static: {hp: 2, speed: 22, radius: 6, stones: 1},
	crow:   {hp: 1, speed: 46, radius: 5, stones: 2},
	shadow: {hp: 7, speed: 30, radius: 6, stones: 5},
}

type enemy struct {
	id        uint32
	kind      kind
	pos, prev vec
	hp        float64
	knock     vec     // knockback, which fades
	flash     float64 // seconds left of the flash of a hit
	pickHit   float64 // seconds before the picks hit it again
	phase     float64 // its animation's, and a crow's weaving
	dead      bool
}

// A note is what the vocalist sings at the noise.
type note struct {
	pos, prev, vel vec
	damage         float64
	pierce         int // enemies it goes through
	life           float64
	hit            []uint32
}

// A stone is what the fallen leave: the vocalist collects them.
type stone struct {
	pos, prev vec
	value     int
	pulled    bool
}

// A puff is the cloud an enemy leaves, a popup a number of damage.
type puff struct {
	pos vec
	t   float64
}

type popup struct {
	pos  vec
	t    float64
	text string
}

// The upgrades a level offers, three at a time. Four bring a bandmate:
// the first time, they join, and follow Tomori from then on.
type upgrade int

const (
	upVoice upgrade = iota
	upTempo
	upHarmony // the guitarist joins
	upBass    // the bassist joins
	upPicks   // the lead guitarist joins
	upDrums   // the drummer joins
	upShoes
	upEye
	upBandaid
	upgradeCount
)

var upgrades = [upgradeCount]struct {
	name, desc string
	max        int
	icon       int
	member     int // the bandmate it brings, or -1
}{
	upVoice:   {"Louder", "Her notes hit harder", 5, cMic, -1},
	upTempo:   {"Tempo up", "She sings faster", 5, cMetronome, -1},
	upHarmony: {"Harmony", "Anon joins on guitar:\none more note a volley", 3, cGuitar, guitarist},
	upBass:    {"Bassline", "Soyo joins on bass:\nnotes go through one more", 3, cBassIcon, bassist},
	upPicks:   {"Guitar picks", "Rāna joins on lead:\na pick circles you", 5, cPick, lead},
	upDrums:   {"Drum beat", "Taki joins on drums:\na beat blasts around you", 4, cDrum, drummer},
	upShoes:   {"Running shoes", "Walk faster", 4, cShoe, -1},
	upEye:     {"An eye for stones", "Stones come from\nfarther away", 3, cStoneIcon, -1},
	upBandaid: {"Band-aid", "One more band-aid,\nand heal two", 5, cBandaid, -1},
}

// The band, after MyGO!!!!!'s: Tomori sings, Anon plays guitar, Soyo the
// bass, Rāna lead guitar and Taki the drums.
const (
	vocalist = iota
	guitarist
	bassist
	lead
	drummer
	memberCount
)

var memberNames = [memberCount]string{"Tomori", "Anon", "Soyo", "Rāna", "Taki"}

// The state of a game: before it, playing, choosing an upgrade, or over.
type state int

const (
	title state = iota
	playing
	choosing
	over
)

// What happened in a step, for its sounds.
type event int

const (
	evSing event = iota
	evHit
	evKill
	evStone
	evLevel
	evHurt
	evBeat
	evChoose
	evJoin
	evOver
)

const (
	maxEnemies = 400
	cellSize   = 16  // of the grid that finds enemies near a point
	trailLen   = 128 // steps of the vocalist's trail the band follows
	trailGap   = 12  // steps between bandmates on it
)

// A world is the game's state, advanced in fixed steps of simulated time.
type world struct {
	rng   *rand.Rand
	state state
	time  float64 // held out, in seconds

	pos, prev  vec // the vocalist, by the middle of their feet
	move       vec // where the player steers, at most 1 long
	facingLeft bool
	walked     float64
	hp, maxHP  int
	hurt       float64 // seconds left unhurtable
	speed      float64
	reach      float64 // how far stones come from

	damage, cooldown float64
	notes, pierce    int
	levels           [upgradeCount]int

	// The band in the order they joined, after the vocalist, and the trail
	// they walk on.
	band  []int
	trail [trailLen]vec
	head  int

	enemies []enemy
	sung    []note
	stones  []stone
	puffs   []puff
	popups  []popup
	nextID  uint32

	spawnTimer, singTimer, beatTimer, waveTimer float64
	pickAngle, beatFlash, shake, overTime       float64

	xp, level, xpNext int
	collected         int // stones
	calmed            int // enemies
	choices           []upgrade
	banner            string // who joined, shown for a while
	bannerTime        float64

	// view is half the view's size in the world: enemies come from
	// beyond it.
	view   vec
	events []event

	grid [256][]int32 // enemies by cell, hashed
}

func newWorld(seed uint64) *world {
	w := &world{rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)), view: vec{180, 110}}
	w.reset()
	w.state = title
	return w
}

// reset starts a game anew.
func (w *world) reset() {
	*w = world{rng: w.rng, view: w.view, events: w.events[:0], enemies: w.enemies[:0], sung: w.sung[:0],
		stones: w.stones[:0], puffs: w.puffs[:0], popups: w.popups[:0], band: w.band[:0], grid: w.grid}
	w.state = playing
	w.hp, w.maxHP = 5, 5
	w.speed, w.reach = 62, 30
	w.damage, w.cooldown, w.notes = 1, 0.75, 1
	w.level, w.xpNext = 1, 5
	w.spawnTimer, w.waveTimer, w.beatTimer = 0.5, 45, 1
}

// step advances the world by dt seconds.
func (w *world) step(dt float64) {
	if w.state == over {
		w.overTime += dt
	}
	if w.state != playing {
		return
	}
	w.time += dt
	w.walk(dt)
	w.spawn(dt)
	w.chase(dt)
	w.sing(dt)
	w.fly(dt)
	w.spinPicks(dt)
	w.drum(dt)
	w.collect(dt)
	w.touch(dt)
	w.age(dt)
}

func (w *world) walk(dt float64) {
	w.prev = w.pos
	if w.move.len() > 0.05 {
		w.pos = w.pos.add(w.move.mul(w.speed * dt))
		w.walked += w.move.len() * w.speed * dt
		if w.move.x < -0.1 {
			w.facingLeft = true
		} else if w.move.x > 0.1 {
			w.facingLeft = false
		}
		w.head = (w.head + 1) % trailLen
		w.trail[w.head] = w.pos
	}
}

// follower returns where bandmate i of the band (0 the first to join)
// walks: on the vocalist's trail, a little behind the one before.
func (w *world) follower(i int) vec {
	return w.trail[(w.head-(i+1)*trailGap+trailLen*4)%trailLen]
}

// spawn brings enemies from just beyond the view's edges, ever more and
// stronger, static in groups, and every 45 seconds a ring of it all
// around.
func (w *world) spawn(dt float64) {
	w.spawnTimer -= dt
	for w.spawnTimer <= 0 {
		w.spawnTimer += max(0.08, 0.7*math.Pow(0.982, w.time))
		k, n := static, 1+w.rng.IntN(1+int(w.time/40))
		switch r := w.rng.Float64(); {
		case w.time > 60 && r < 0.15:
			k, n = shadow, 1
		case w.time > 25 && r < 0.4:
			k, n = crow, 1+w.rng.IntN(2)
		}
		at := w.edge(14)
		for range n {
			w.add(k, at.add(vec{w.rng.Float64()*16 - 8, w.rng.Float64()*16 - 8}))
		}
	}
	w.waveTimer -= dt
	if w.waveTimer <= 0 {
		w.waveTimer = 45
		n := 24 + int(w.time/15)
		for i := range n {
			a := float64(i) / float64(n) * 2 * math.Pi
			w.add(static, w.pos.add(vec{math.Cos(a), math.Sin(a)}.mul(w.view.len()+4)))
		}
	}
}

// edge returns a point margin beyond the view's edges, anywhere around.
func (w *world) edge(margin float64) vec {
	hw, hh := w.view.x+margin, w.view.y+margin
	d := w.rng.Float64() * 4 * (hw + hh)
	var p vec
	switch {
	case d < 2*hw:
		p = vec{d - hw, -hh}
	case d < 4*hw:
		p = vec{d - 3*hw, hh}
	case d < 4*hw+2*hh:
		p = vec{-hw, d - 4*hw - hh}
	default:
		p = vec{hw, d - 4*hw - 3*hh}
	}
	return w.pos.add(p)
}

func (w *world) add(k kind, at vec) {
	if len(w.enemies) >= maxEnemies {
		return
	}
	w.nextID++
	w.enemies = append(w.enemies, enemy{id: w.nextID, kind: k, pos: at, prev: at,
		hp: kinds[k].hp * (1 + w.time/100), phase: w.rng.Float64() * 10})
}

// cellOf hashes the cell of the grid a point is in.
func cellOf(x, y int) int { return int(uint32(x*73856093^y*19349663) % 256) }

func (w *world) cell(p vec) (int, int) {
	return int(math.Floor(p.x / cellSize)), int(math.Floor(p.y / cellSize))
}

// index files the enemies in the grid.
func (w *world) index() {
	for i := range w.grid {
		w.grid[i] = w.grid[i][:0]
	}
	for i := range w.enemies {
		cx, cy := w.cell(w.enemies[i].pos)
		c := cellOf(cx, cy)
		w.grid[c] = append(w.grid[c], int32(i))
	}
}

// near calls fn with the enemies that may be within a cell of p.
func (w *world) near(p vec, fn func(e *enemy)) {
	cx, cy := w.cell(p)
	var seen [9]int
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			c := cellOf(cx+dx, cy+dy)
			if slices.Contains(seen[:n], c) {
				continue // two cells hashed alike
			}
			seen[n] = c
			n++
			for _, i := range w.grid[c] {
				fn(&w.enemies[i])
			}
		}
	}
}

// chase moves the enemies toward the vocalist, apart from each other.
func (w *world) chase(dt float64) {
	far := w.view.len() * 3
	for i := range w.enemies {
		e := &w.enemies[i]
		e.prev = e.pos
		e.phase += dt
		to := w.pos.sub(e.pos)
		if to.len() > far {
			e.dead = true // left behind: others come
			continue
		}
		dir := to.norm()
		if e.kind == crow {
			side := vec{-dir.y, dir.x}.mul(0.7 * math.Sin(e.phase*5))
			dir = dir.add(side)
		}
		e.pos = e.pos.add(dir.mul(kinds[e.kind].speed * dt)).add(e.knock.mul(dt))
		e.knock = e.knock.mul(math.Exp(-9 * dt))
	}
	w.index()
	for i := range w.enemies {
		e := &w.enemies[i]
		w.near(e.pos, func(o *enemy) {
			if o == e {
				return
			}
			d := e.pos.sub(o.pos)
			r := kinds[e.kind].radius + kinds[o.kind].radius
			if l := d.len(); l < r {
				if l == 0 {
					d, l = vec{1, 0}, 1
				}
				e.pos = e.pos.add(d.mul((r - l) / l * 0.5))
			}
		})
	}
}

// sing sends notes at the nearest enemy in reach.
func (w *world) sing(dt float64) {
	w.singTimer -= dt
	if w.singTimer > 0 {
		return
	}
	var target *enemy
	best := 200.0
	for i := range w.enemies {
		if d := w.enemies[i].pos.sub(w.pos).len(); d < best && !w.enemies[i].dead {
			target, best = &w.enemies[i], d
		}
	}
	if target == nil {
		w.singTimer = 0
		return
	}
	w.singTimer = w.cooldown
	from := w.pos.add(vec{0, -8})
	aim := math.Atan2(target.pos.y-from.y, target.pos.x-from.x)
	for i := range w.notes {
		a := aim + (float64(i)-float64(w.notes-1)/2)*0.22
		w.sung = append(w.sung, note{pos: from, prev: from, vel: vec{math.Cos(a), math.Sin(a)}.mul(200),
			damage: w.damage, pierce: w.pierce, life: 1.2})
	}
	w.events = append(w.events, evSing)
}

func (w *world) fly(dt float64) {
	for i := range w.sung {
		n := &w.sung[i]
		n.prev = n.pos
		n.pos = n.pos.add(n.vel.mul(dt))
		n.life -= dt
		w.near(n.pos, func(e *enemy) {
			if n.life <= 0 || e.dead || slices.Contains(n.hit, e.id) ||
				e.pos.sub(n.pos).len() > kinds[e.kind].radius+3 {
				return
			}
			n.hit = append(n.hit, e.id)
			w.hurtEnemy(e, n.damage, n.pos.sub(n.vel))
			if n.pierce--; n.pierce < 0 {
				n.life = 0
			}
		})
	}
	w.sung = slices.DeleteFunc(w.sung, func(n note) bool { return n.life <= 0 })
}

// pickPos returns where pick i of n is.
func (w *world) pickPos(i, n int, around vec, angle float64) vec {
	a := angle + float64(i)/float64(n)*2*math.Pi
	return around.add(vec{math.Cos(a), math.Sin(a) * 0.8}.mul(30)).add(vec{0, -6})
}

func (w *world) spinPicks(dt float64) {
	n := w.levels[upPicks]
	if n == 0 {
		return
	}
	w.pickAngle += 3.4 * dt
	for i := range n {
		at := w.pickPos(i, n, w.pos, w.pickAngle)
		w.near(at, func(e *enemy) {
			if !e.dead && e.pickHit <= 0 && e.pos.sub(at).len() < kinds[e.kind].radius+6 {
				e.pickHit = 0.4
				w.hurtEnemy(e, 1+float64(n)/3, w.pos)
			}
		})
	}
}

// beatRadius is how far the drummer's beat reaches at its level.
func (w *world) beatRadius() float64 { return 44 + 12*float64(w.levels[upDrums]) }

func (w *world) drum(dt float64) {
	lv := w.levels[upDrums]
	if lv == 0 {
		return
	}
	w.beatTimer -= dt
	if w.beatTimer > 0 {
		return
	}
	w.beatTimer = 4.6 - 0.6*float64(lv)
	w.beatFlash, w.shake = 0.4, max(w.shake, 0.12)
	r := w.beatRadius()
	for i := range w.enemies {
		if e := &w.enemies[i]; !e.dead && e.pos.sub(w.pos).len() < r {
			w.hurtEnemy(e, 1+float64(lv), w.pos)
			e.knock = e.knock.mul(2)
		}
	}
	w.events = append(w.events, evBeat)
}

// hurtEnemy hurts e by d, knocking it away from where the hit came from.
func (w *world) hurtEnemy(e *enemy, d float64, from vec) {
	e.hp -= d
	e.flash = 0.12
	push := 90.0
	if e.kind == shadow {
		push = 40
	}
	e.knock = e.pos.sub(from).norm().mul(push)
	w.popups = append(w.popups, popup{pos: e.pos.add(vec{0, -10}), text: strconv.FormatFloat(math.Round(d*10)/10, 'f', -1, 64)})
	if e.hp > 0 {
		w.events = append(w.events, evHit)
		return
	}
	e.dead = true
	w.calmed++
	w.stones = append(w.stones, stone{pos: e.pos, prev: e.pos, value: kinds[e.kind].stones})
	w.puffs = append(w.puffs, puff{pos: e.pos})
	w.events = append(w.events, evKill)
}

// collect draws in the stones within reach, and levels up.
func (w *world) collect(dt float64) {
	for i := range w.stones {
		s := &w.stones[i]
		s.prev = s.pos
		to := w.pos.add(vec{0, -4}).sub(s.pos)
		d := to.len()
		if d < w.reach {
			s.pulled = true
		}
		if !s.pulled {
			continue
		}
		if d < 5 {
			w.xp += s.value
			w.collected++
			s.value = 0
			w.events = append(w.events, evStone)
			continue
		}
		s.pos = s.pos.add(to.norm().mul(min(d, (90+w.speed*1.5)*dt)))
	}
	w.stones = slices.DeleteFunc(w.stones, func(s stone) bool { return s.value == 0 })
	w.levelUp()
}

func (w *world) levelUp() {
	if w.state != playing || w.xp < w.xpNext {
		return
	}
	w.xp -= w.xpNext
	w.level++
	w.xpNext = 3 + w.level*2
	w.choices = w.choices[:0]
	var open, joining []upgrade
	for u := range upgradeCount {
		if w.levels[u] >= upgrades[u].max {
			continue
		}
		if upgrades[u].member >= 0 && w.levels[u] == 0 {
			joining = append(joining, u)
		} else {
			open = append(open, u)
		}
	}
	w.rng.Shuffle(len(open), func(i, j int) { open[i], open[j] = open[j], open[i] })
	w.rng.Shuffle(len(joining), func(i, j int) { joining[i], joining[j] = joining[j], joining[i] })
	// A bandmate is always among the choices until all have joined.
	if len(joining) > 0 {
		w.choices = append(w.choices, joining[0])
		open = append(open, joining[1:]...)
		w.rng.Shuffle(len(open), func(i, j int) { open[i], open[j] = open[j], open[i] })
	}
	w.choices = append(w.choices, open[:min(3-len(w.choices), len(open))]...)
	w.rng.Shuffle(len(w.choices), func(i, j int) { w.choices[i], w.choices[j] = w.choices[j], w.choices[i] })
	if len(w.choices) > 0 {
		w.state = choosing
	}
	w.events = append(w.events, evLevel)
}

// choose takes the upgrade offered at i.
func (w *world) choose(i int) {
	if w.state != choosing || i < 0 || i >= len(w.choices) {
		return
	}
	u := w.choices[i]
	w.levels[u]++
	switch u {
	case upVoice:
		w.damage++
	case upTempo:
		w.cooldown *= 0.8
	case upHarmony:
		w.notes++
	case upBass:
		w.pierce++
	case upShoes:
		w.speed *= 1.12
	case upEye:
		w.reach *= 1.45
	case upBandaid:
		w.maxHP++
		w.hp = min(w.hp+2, w.maxHP)
	}
	w.state = playing
	if m := upgrades[u].member; m >= 0 && w.levels[u] == 1 {
		w.band = append(w.band, m)
		w.banner, w.bannerTime = memberNames[m]+" joins the band!", 2.5
		// They come from where the vocalist walked a moment ago.
		w.events = append(w.events, evJoin)
	} else {
		w.events = append(w.events, evChoose)
	}
	w.levelUp()
}

// joined reports whether bandmate m is in the band.
func (w *world) joined(m int) bool { return slices.Contains(w.band, m) }

// touch hurts the vocalist when an enemy reaches them.
func (w *world) touch(dt float64) {
	if w.hurt > 0 {
		w.hurt -= dt
		return
	}
	w.near(w.pos, func(e *enemy) {
		if w.hurt > 0 || e.dead || e.pos.sub(w.pos).len() > kinds[e.kind].radius+4 {
			return
		}
		w.hp--
		w.hurt, w.shake = 1, 0.3
		w.events = append(w.events, evHurt)
		for i := range w.enemies {
			if o := &w.enemies[i]; o.pos.sub(w.pos).len() < 30 {
				o.knock = o.pos.sub(w.pos).norm().mul(160)
			}
		}
		if w.hp <= 0 {
			w.state = over
			w.events = append(w.events, evOver)
		}
	})
}

// age moves on the effects, and forgets the dead.
func (w *world) age(dt float64) {
	for i := range w.enemies {
		e := &w.enemies[i]
		e.flash -= dt
		e.pickHit -= dt
	}
	w.enemies = slices.DeleteFunc(w.enemies, func(e enemy) bool { return e.dead })
	for i := range w.puffs {
		w.puffs[i].t += dt
	}
	w.puffs = slices.DeleteFunc(w.puffs, func(p puff) bool { return p.t > 0.32 })
	for i := range w.popups {
		w.popups[i].t += dt
	}
	w.popups = slices.DeleteFunc(w.popups, func(p popup) bool { return p.t > 0.6 })
	w.shake = max(w.shake-dt, 0)
	w.beatFlash = max(w.beatFlash-dt, 0)
	w.bannerTime = max(w.bannerTime-dt, 0)
}
