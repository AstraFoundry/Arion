package main

import (
	"math"
	"math/rand/v2"
)

// The map, in tiles.
const (
	mapW = 40
	mapH = 30
)

type ground uint8

const (
	grass ground = iota
	path
	soil
	water
)

type object uint8

const (
	none object = iota
	tree
	bush
	rock
	fence
)

// A tile is a square of the map: its ground, what stands on it, and the
// crop growing in its soil.
type tile struct {
	ground  ground
	variant uint8 // of grass: plain, tufts or flowers
	object  object
	wet     bool    // watered soil
	crop    int8    // the crop's stage, 0 to ripe, or -1
	grows   float64 // when the crop grows next, if wet
}

const ripe = 3

// growTime is how long a watered crop takes to grow a stage, in seconds.
const growTime = 4.0

type dir uint8

const (
	down dir = iota
	up
	left
	right
)

type tool uint8

const (
	hoe tool = iota
	seeds
	can
)

var toolNames = [...]string{"Hoe", "Seeds", "Watering can"}

// A world is the game's state, advanced in fixed steps of simulated time.
type world struct {
	tiles [mapH][mapW]tile
	// The farmer, by the middle of their feet, in tiles, and where they
	// were a step before.
	x, y   float64
	px, py float64
	facing dir
	moving bool
	walked float64 // distance walked, for the frames and the footsteps
	tool   tool
	coins  int
	// held are the directions held, from the keys, and stick where a
	// gamepad's stick leans, each from -1 to 1.
	held  [4]bool
	stick [2]float64
	// now is the simulated time, in seconds.
	now float64
	// events are what happened in the last steps, for sounds.
	events []event
}

type event uint8

const (
	evStep event = iota
	evTill
	evPlant
	evWater
	evHarvest
	evNothing
)

// newWorld lays out a farm: a field of grass with a pond, a path, trees
// around it and some bushes and rocks, and the farmer by the house.
func newWorld() *world {
	w := &world{x: 20.5, y: 15.8, px: 20.5, py: 15.8, facing: down}
	rng := rand.New(rand.NewPCG(7, 11))
	for y := range mapH {
		for x := range mapW {
			t := &w.tiles[y][x]
			t.crop = -1
			if r := rng.IntN(10); r < 2 {
				t.variant = 1
			} else if r < 3 {
				t.variant = 2
			}
			if x < 2 || y < 2 || x >= mapW-2 || y >= mapH-2 {
				t.object = tree
			}
		}
	}
	// A pond.
	for y := 5; y < 10; y++ {
		for x := 27; x < 35; x++ {
			if dx, dy := float64(x)-30.5, float64(y)-7; dx*dx/16+dy*dy/6 < 1 {
				w.tiles[y][x].ground = water
			}
		}
	}
	// A path from the south edge to the middle.
	for y := 16; y < mapH-2; y++ {
		w.tiles[y][20].ground = path
		w.tiles[y][21].ground = path
	}
	// A fence along the field's north side, with a gap.
	for x := 8; x < 18; x++ {
		if x != 12 {
			w.tiles[11][x].object = fence
		}
	}
	for range 25 {
		x, y := 2+rng.IntN(mapW-4), 2+rng.IntN(mapH-4)
		if t := &w.tiles[y][x]; t.ground == grass && t.object == none && (x < 18 || x > 23) {
			t.object = []object{bush, rock}[rng.IntN(2)]
		}
	}
	return w
}

// solid reports whether the farmer cannot stand on tile x, y.
func (w *world) solid(x, y int) bool {
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		return true
	}
	t := &w.tiles[y][x]
	return t.ground == water || t.object != none
}

// blocked reports whether the farmer's feet, a box 0.6 by 0.3 tiles,
// overlap a solid tile at x, y.
func (w *world) blocked(x, y float64) bool {
	const hw, hh = 0.3, 0.15
	for _, p := range [4][2]float64{{x - hw, y - hh}, {x + hw, y - hh}, {x - hw, y + hh}, {x + hw, y + hh}} {
		if w.solid(int(math.Floor(p[0])), int(math.Floor(p[1]))) {
			return true
		}
	}
	return false
}

// speed is how fast the farmer walks, in tiles a second.
const speed = 4.5

// step advances the world by dt seconds.
func (w *world) step(dt float64) {
	w.now += dt
	w.px, w.py = w.x, w.y
	var dx, dy float64
	if w.held[left] {
		dx--
	}
	if w.held[right] {
		dx++
	}
	if w.held[up] {
		dy--
	}
	if w.held[down] {
		dy++
	}
	if dx != 0 && dy != 0 {
		dx, dy = dx*math.Sqrt2/2, dy*math.Sqrt2/2
	}
	if dx == 0 && dy == 0 {
		// A gamepad's stick, as far as it leans.
		dx, dy = w.stick[0], w.stick[1]
	}
	w.moving = dx != 0 || dy != 0
	if w.moving {
		switch {
		case math.Abs(dy) >= math.Abs(dx) && dy > 0:
			w.facing = down
		case math.Abs(dy) >= math.Abs(dx):
			w.facing = up
		case dx < 0:
			w.facing = left
		default:
			w.facing = right
		}
		// Each axis on its own, so that the farmer slides along walls.
		before := w.walked
		if nx := w.x + dx*speed*dt; !w.blocked(nx, w.y) {
			w.walked += math.Abs(nx - w.x)
			w.x = nx
		}
		if ny := w.y + dy*speed*dt; !w.blocked(w.x, ny) {
			w.walked += math.Abs(ny - w.y)
			w.y = ny
		}
		// A footstep every 0.6 tiles.
		if math.Floor(w.walked/0.6) != math.Floor(before/0.6) {
			w.events = append(w.events, evStep)
		}
	}
	for y := range mapH {
		for x := range mapW {
			t := &w.tiles[y][x]
			if t.crop >= 0 && t.crop < ripe && t.wet && w.now >= t.grows {
				t.crop++
				t.wet = false
			}
		}
	}
}

// target returns the tile in front of the farmer.
func (w *world) target() (int, int) {
	x, y := int(math.Floor(w.x)), int(math.Floor(w.y))
	switch w.facing {
	case down:
		y++
	case up:
		y--
	case left:
		x--
	case right:
		x++
	}
	return x, y
}

// use uses the tool on the tile in front of the farmer, or harvests a ripe
// crop there whatever the tool.
func (w *world) use() {
	x, y := w.target()
	if x < 0 || y < 0 || x >= mapW || y >= mapH {
		w.events = append(w.events, evNothing)
		return
	}
	t := &w.tiles[y][x]
	switch {
	case t.crop == ripe:
		t.crop = -1
		w.coins += 15
		w.events = append(w.events, evHarvest)
	case w.tool == hoe && t.ground == grass && t.object == none:
		t.ground = soil
		w.events = append(w.events, evTill)
	case w.tool == seeds && t.ground == soil && t.crop < 0:
		t.crop = 0
		if t.wet {
			t.grows = w.now + growTime
		}
		w.events = append(w.events, evPlant)
	case w.tool == can && t.ground == soil:
		if !t.wet {
			t.wet = true
			t.grows = w.now + growTime
		}
		w.events = append(w.events, evWater)
	default:
		w.events = append(w.events, evNothing)
	}
}

// dayLength is how long a day lasts, in seconds.
const dayLength = 150.0

// daylight returns how light it is, from 0 at midnight to 1 by day, and
// the hour.
func (w *world) daylight() (light float64, hour int) {
	// The game starts at 8 in the morning.
	t := math.Mod(w.now/dayLength+8.0/24, 1)
	hour = int(t * 24)
	// Dark from 20 to 5, with an hour of dusk and of dawn.
	switch h := t * 24; {
	case h >= 21 || h < 5:
		light = 0
	case h >= 20:
		light = 21 - h
	case h < 6:
		light = h - 5
	default:
		light = 1
	}
	return light, hour
}
