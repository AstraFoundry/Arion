package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/game"
	"github.com/egoist/mygo/ui"
)

// Tests play no sound: the game is made without its sounds.

// The farmer tills, plants, waters, and harvests what grows.
func TestFarming(t *testing.T) {
	w := newWorld()
	w.facing = left
	tx, ty := w.target()
	tile := &w.tiles[ty][tx]
	if tile.ground != grass || tile.object != none {
		t.Fatalf("the farmer starts facing %v", *tile)
	}
	w.tool = hoe
	w.use()
	w.tool = seeds
	w.use()
	w.tool = can
	w.use()
	if tile.ground != soil || tile.crop != 0 || !tile.wet {
		t.Fatalf("after hoe, seeds and water: %+v", *tile)
	}
	// Watered, it grows a stage, then waits for water again.
	for range int(growTime/stepTime) + 2 {
		w.step(stepTime)
	}
	if tile.crop != 1 || tile.wet {
		t.Fatalf("after growing a stage: %+v", *tile)
	}
	for tile.crop < ripe {
		w.use()
		for range int(growTime/stepTime) + 2 {
			w.step(stepTime)
		}
	}
	w.tool = hoe
	w.use()
	if tile.crop != -1 || w.coins != 15 {
		t.Errorf("after harvesting: crop %d, %d coins", tile.crop, w.coins)
	}
	want := []event{evTill, evPlant, evWater, evWater, evWater, evHarvest}
	if len(w.events) != len(want) {
		t.Errorf("events %v, want %v", w.events, want)
	}
}

// The farmer walks, and stops at the pond and the trees.
func TestWalking(t *testing.T) {
	w := newWorld()
	w.held[right] = true
	for range 5 * game.Rate {
		w.step(stepTime)
	}
	if w.facing != right || w.x <= 21 {
		t.Errorf("after walking right: at %v facing %v", w.x, w.facing)
	}
	if w.x > mapW-2 {
		t.Errorf("walked into the trees: %v", w.x)
	}
	w.held = [4]bool{up: true}
	for range 20 * game.Rate {
		w.step(stepTime)
	}
	if w.y < 2 {
		t.Errorf("walked into the trees north: %v", w.y)
	}
	if len(w.events) == 0 || w.events[0] != evStep {
		t.Error("no footsteps")
	}
}

// A gamepad's stick walks as far as it leans, facing where it leans most.
func TestStickWalks(t *testing.T) {
	w := newWorld()
	x, y := w.x, w.y
	w.stick = [2]float64{-0.5, 0.2}
	for range game.Rate / 2 {
		w.step(stepTime)
	}
	if w.facing != left || w.x >= x || w.y <= y {
		t.Errorf("at %v, %v facing %v", w.x, w.y, w.facing)
	}
	// Half way leaned, half as fast as the keys.
	if d := x - w.x; d < 0.5*speed*0.5*0.9 || d > 0.5*speed*0.5*1.1 {
		t.Errorf("walked %v in half a second", d)
	}
}

// The game's view draws, takes keys, and walks the farmer while they are
// held.
func TestView(t *testing.T) {
	g := newFarm(false)
	tt := ui.NewTester(g.view, 960, 640)
	// Keys act in the game's next step.
	step := func() {
		tt.Advance(3 * time.Second / game.Rate)
	}
	tt.Key(0, ui.Key3)
	step()
	if g.w.tool != can {
		t.Errorf("key 3 chose %v", g.w.tool)
	}
	tt.Key(0, ui.KeyEqual)
	step()
	if g.zoom != 3 {
		t.Errorf("zoom %d after +", g.zoom)
	}
	x := g.w.x
	tt.KeyDown(ui.KeyD)
	tt.Advance(200 * time.Millisecond)
	if g.w.x <= x || g.w.facing != right {
		t.Errorf("D held: at %v facing %v", g.w.x, g.w.facing)
	}
}
