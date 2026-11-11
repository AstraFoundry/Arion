package main

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/game"
	"github.com/egoist/mygo/ui"
)

// inPlay returns a world in play, without enemies coming for a while.
func inPlay() *world {
	w := newWorld(1)
	w.reset()
	w.spawnTimer, w.waveTimer = 1e9, 1e9
	return w
}

func steps(w *world, sec float64) {
	for range int(sec * game.Rate) {
		w.step(stepTime)
	}
}

// Tomori sings on her own at an enemy in reach and calms it, which leaves
// a stone she picks up.
func TestNotesCalm(t *testing.T) {
	w := inPlay()
	w.add(static, vec{60, 0})
	steps(w, 2)
	if w.calmed != 1 || len(w.enemies) != 0 {
		t.Fatalf("%d calmed, %d enemies left", w.calmed, len(w.enemies))
	}
	if len(w.stones) != 1 {
		t.Fatalf("%d stones", len(w.stones))
	}
	w.move = vec{1, 0}
	steps(w, 1)
	if w.xp != 1 || w.collected != 1 || len(w.stones) != 0 {
		t.Errorf("xp %d, %d collected, %d stones after walking to it", w.xp, w.collected, len(w.stones))
	}
}

// Enough experience offers three upgrades, which wait for a choice.
func TestLevelUp(t *testing.T) {
	w := inPlay()
	w.stones = append(w.stones, stone{pos: vec{0, -4}, value: 5})
	w.step(stepTime)
	if w.state != choosing || w.level != 2 || len(w.choices) != 3 {
		t.Fatalf("state %v, level %d, %d choices", w.state, w.level, len(w.choices))
	}
	before := w.time
	steps(w, 1)
	if w.time != before {
		t.Error("the world went on while choosing")
	}
	// Until the band is complete, a bandmate is among the choices.
	if !slices.ContainsFunc(w.choices, func(u upgrade) bool { return upgrades[u].member >= 0 }) {
		t.Errorf("no bandmate among %v", w.choices)
	}
	u := w.choices[0]
	w.choose(0)
	if w.state != playing || w.levels[u] != 1 {
		t.Errorf("after choosing: state %v, level of %v %d", w.state, u, w.levels[u])
	}
}

func TestUpgrades(t *testing.T) {
	w := inPlay()
	for _, u := range []upgrade{upHarmony, upPicks, upDrums, upBandaid, upHarmony} {
		w.state, w.choices = choosing, []upgrade{u}
		w.choose(0)
	}
	if w.notes != 3 || w.levels[upPicks] != 1 || w.maxHP != 6 || w.hp != 6 {
		t.Errorf("notes %d, picks %d, band-aids %d/%d", w.notes, w.levels[upPicks], w.hp, w.maxHP)
	}
	// Anon, Rāna and Taki joined, once each, and follow Tomori.
	if !slices.Equal(w.band, []int{guitarist, lead, drummer}) {
		t.Errorf("band %v", w.band)
	}
	w.move = vec{1, 0}
	steps(w, 2)
	if f := w.follower(0); f.x >= w.pos.x || f.x < w.pos.x-30 {
		t.Errorf("Anon at %v, Tomori at %v", f, w.pos)
	}
	w.move = vec{}
	// The picks and the beat calm what comes near.
	w.beatTimer = 0.5
	w.damage = 0
	w.add(static, w.pos.add(vec{30, -5}))
	w.add(crow, w.pos.add(vec{-30, 20}))
	steps(w, 3)
	if w.calmed != 2 {
		t.Errorf("%d calmed by picks and beat", w.calmed)
	}
}

// An enemy reaching Tomori takes a band-aid, then she cannot be hurt for
// a second; without band-aids, the game is over.
func TestHurt(t *testing.T) {
	w := inPlay()
	w.damage = 0
	w.add(shadow, vec{3, 0})
	steps(w, 0.1)
	if w.hp != 4 || w.hurt <= 0 {
		t.Fatalf("hp %d, hurt %v", w.hp, w.hurt)
	}
	w.hp = 1
	steps(w, 3)
	if w.state != over {
		t.Errorf("state %v with no band-aids", w.state)
	}
}

// Enemies come from beyond the view, more as time goes on, and stay apart.
func TestSpawn(t *testing.T) {
	w := newWorld(2)
	w.reset()
	w.damage = 0
	w.hurt = 1e9
	w.spawn(30)
	if len(w.enemies) < 30 {
		t.Fatalf("%d enemies after 30 s", len(w.enemies))
	}
	for _, e := range w.enemies {
		if d := e.pos.sub(w.pos); math.Abs(d.x) < w.view.x && math.Abs(d.y) < w.view.y {
			t.Fatalf("an enemy came within the view, at %v", d)
		}
	}
	steps(w, 5)
	for i, a := range w.enemies {
		for _, b := range w.enemies[i+1:] {
			if a.pos.sub(b.pos).len() < 2 {
				t.Fatalf("enemies on top of each other at %v", a.pos)
			}
		}
	}
}

// The view starts the game with a key, walks Tomori with the keys, and
// waits without the keyboard.
func TestView(t *testing.T) {
	h := newMayoi(false)
	tt := ui.NewTester(h.view, 960, 640)
	step := func() {
		tt.Advance(3 * time.Second / game.Rate)
	}
	tt.Key(0, ui.KeySpace)
	step()
	if h.w.state != playing || !h.sounding {
		t.Fatalf("state %v after Space, music %v", h.w.state, h.sounding)
	}
	// Paused, the music pauses too.
	tt.Key(0, ui.KeyEscape)
	step()
	if !h.paused || h.sounding {
		t.Errorf("after Esc: paused %v, music %v", h.paused, h.sounding)
	}
	tt.Key(0, ui.KeyEscape)
	step()
	if h.paused || !h.sounding {
		t.Errorf("after Esc again: paused %v, music %v", h.paused, h.sounding)
	}
	x := h.w.pos.x
	tt.KeyDown(ui.KeyD)
	tt.Advance(200 * time.Millisecond)
	if h.w.pos.x <= x || h.w.facingLeft {
		t.Errorf("D held: at %v, facing left %v", h.w.pos.x, h.w.facingLeft)
	}
	tt.KeyUp(ui.KeyD)
	tt.SetFocused(false)
	step()
	before := h.w.time
	step()
	if h.w.time != before || !h.frozen || h.sounding {
		t.Errorf("without the keyboard: the world went on %v, music %v", h.w.time != before, h.sounding)
	}
}
