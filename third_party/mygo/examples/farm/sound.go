package main

import (
	"io"
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/audio"
)

const sfxRate = 32000

// sounds are the game's effects, synthesized once, and its music, made as
// it plays.
type sounds struct {
	step, till, plant, water, harvest, nothing *audio.Sound
	music                                      *audio.Player
	rng                                        *rand.Rand
}

func newSounds() *sounds {
	rng := rand.New(rand.NewPCG(3, 5))
	return &sounds{
		step: synth(0.07, func(t float64) float64 { return noise(rng) * math.Exp(-t*60) * 0.5 }),
		till: synth(0.18, func(t float64) float64 {
			return (0.6*math.Sin(2*math.Pi*90*t*(1-t*2)) + 0.5*noise(rng)) * math.Exp(-t*25)
		}),
		plant: synth(0.12, func(t float64) float64 { return math.Sin(2*math.Pi*(700-2500*t)*t) * math.Exp(-t*30) * 0.5 }),
		water: synth(0.35, func(t float64) float64 {
			return noise(rng) * (0.5 + 0.5*math.Sin(2*math.Pi*23*t)) * math.Exp(-t*8) * 0.25
		}),
		harvest: synth(0.5, chime),
		nothing: synth(0.06, func(t float64) float64 { return math.Sin(2*math.Pi*160*t) * math.Exp(-t*50) * 0.3 }),
		// The first bar is an intro: then the rest loops.
		music: audio.NewPlayer(&tune{}).SetLoopRange(4*tuneBeat*time.Second, 0).SetVolume(0.35),
		rng:   rng,
	}
}

func noise(rng *rand.Rand) float64 { return rng.Float64()*2 - 1 }

// synth computes a sound of d seconds from f, a function of the time.
func synth(d float64, f func(t float64) float64) *audio.Sound {
	s := make([]float32, int(d*sfxRate))
	for i := range s {
		t := float64(i) / sfxRate
		// No click at either end.
		edge := min(t/0.002, (d-t)/0.01, 1)
		s[i] = float32(f(t) * edge)
	}
	return audio.NewSound(sfxRate, 1, s)
}

// chime is two bell-like notes, a fifth apart.
func chime(t float64) float64 {
	bell := func(f, t float64) float64 {
		if t < 0 {
			return 0
		}
		return (math.Sin(2*math.Pi*f*t) + 0.4*math.Sin(2*math.Pi*f*2.76*t)*math.Exp(-t*12)) * math.Exp(-t*7)
	}
	return 0.3 * (bell(1046.5, t) + bell(1568, t-0.09))
}

// play plays the sounds of what happened, a little different each time,
// as games vary effects that play often.
func (s *sounds) play(evs []event, pan float64) {
	for _, e := range evs {
		var snd *audio.Sound
		volume := 1.0
		switch e {
		case evStep:
			snd, volume = s.step, 0.35
		case evTill:
			snd = s.till
		case evPlant:
			snd = s.plant
		case evWater:
			snd = s.water
		case evHarvest:
			snd = s.harvest
		case evNothing:
			snd, volume = s.nothing, 0.6
		}
		snd.NewPlayer().SetVolume(volume).SetPan(pan).SetSpeed(0.94 + s.rng.Float64()*0.12).Play()
	}
}

// tune is the music: a slow, gentle loop in C major, chords under a
// pentatonic melody, made as it plays.
type tune struct{ pos int64 }

const (
	tuneRate  = 22050
	tuneBeat  = 0.5 // seconds: 120 beats a minute
	tuneBeats = 32
)

// The melody, in semitones above C5 for each half beat, -1 resting.
var melody = []int{
	7, -1, 4, -1, 2, 4, 7, -1, 9, -1, 7, -1, 4, -1, -1, -1,
	2, -1, 4, -1, 7, 4, 2, -1, 0, -1, 2, -1, 4, -1, -1, -1,
	7, -1, 9, -1, 12, -1, 9, 7, 4, -1, 7, -1, 9, -1, -1, -1,
	12, -1, 9, -1, 7, 4, 2, -1, 4, -1, 2, -1, 0, -1, -1, -1,
}

// The chords, a bar of 4 beats each: C, Am, F, G, as roots in semitones
// above C3 and their thirds and fifths.
var chords = [][3]int{{0, 4, 7}, {-3, 0, 4}, {-7, -3, 0}, {-5, -1, 2}, {0, 4, 7}, {-3, 0, 4}, {-7, -3, 0}, {-5, -1, 2}}

func (t *tune) SampleRate() int { return tuneRate }
func (t *tune) Channels() int   { return 2 }
func (t *tune) Len() int64      { return int64(tuneBeats * tuneBeat * tuneRate) }
func (t *tune) SetPosition(f int64) error {
	t.pos = f
	return nil
}

func freq(semi int, base float64) float64 { return base * math.Pow(2, float64(semi)/12) }

func (t *tune) Read(buf []float32) (int, error) {
	n := 0
	for ; n+2 <= len(buf) && t.pos < t.Len(); n += 2 {
		sec := float64(t.pos) / tuneRate
		beat := sec / tuneBeat
		// The melody: a soft plucked tone for each of its half beats.
		var lead float64
		if i := int(beat * 2); melody[i] >= 0 {
			nt := sec - float64(i)*tuneBeat/2
			f := freq(melody[i], 523.25)
			lead = (math.Sin(2*math.Pi*f*nt) + 0.3*math.Sin(4*math.Pi*f*nt)) * math.Exp(-nt*4) * min(nt*200, 1)
		}
		// The chord of the bar, swelling and fading.
		var pad float64
		bar := int(beat / 4)
		bt := sec - float64(bar)*4*tuneBeat
		swell := math.Sin(math.Pi * bt / (4 * tuneBeat))
		for _, semi := range chords[bar] {
			f := freq(semi, 130.81)
			pad += math.Sin(2*math.Pi*f*sec) + 0.25*math.Sin(2*math.Pi*2*f*sec)
		}
		pad *= swell / 3
		buf[n] = float32(0.18*lead + 0.12*pad)
		buf[n+1] = float32(0.13*lead + 0.15*pad)
		t.pos++
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}
