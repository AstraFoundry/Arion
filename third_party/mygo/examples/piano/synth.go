package main

import (
	"math"
	"math/rand/v2"
	"sync"

	"github.com/egoist/mygo/audio"
)

const synthRate = 32000

var sounds [highest + 1]struct {
	once sync.Once
	s    *audio.Sound
}

// sound returns the sound of note's key, synthesized the first time.
func sound(note int) *audio.Sound {
	e := &sounds[note]
	e.once.Do(func() { e.s = synth(note) })
	return e.s
}

// synth makes a piano-like tone: partials a little sharper than harmonics,
// as a stiff string's are, from two strings a cent apart, each dying away
// fast and then slowly, the higher the faster, with the thump of a hammer.
func synth(note int) *audio.Sound {
	return audio.NewSound(synthRate, 1, synthSamples(note))
}

func synthSamples(note int) []float32 {
	f := 440 * math.Pow(2, float64(note-69)/12)
	// Low strings ring longer.
	dur := 1.4 + 2.6*float64(highest-note)/float64(highest-lowest)
	n := int(dur * synthRate)
	out := make([]float64, n)
	stiffness := 0.00008 * math.Pow(2, float64(note-lowest)/12)
	for k := 1; k <= 18; k++ {
		fk := float64(k) * f * math.Sqrt(1+stiffness*float64(k*k))
		if fk > synthRate*0.45 {
			break
		}
		amp := 1 / math.Pow(float64(k), 1.1)
		fast := 0.3 * math.Pow(261.6/f, 0.5) / (1 + 0.3*float64(k-1))
		slow := fast * 7
		for _, detune := range []float64{1, 1.0006} {
			// A rotation for the sine and products for the decays, which
			// sin and exp for each sample would make slow.
			w := 2 * math.Pi * fk * detune / synthRate
			cw, sw := math.Cos(w), math.Sin(w)
			s, c := 0.0, 1.0
			e1, e2 := 0.65, 0.35
			d1, d2 := math.Exp(-1/(fast*synthRate)), math.Exp(-1/(slow*synthRate))
			for i := range out {
				out[i] += amp * (e1 + e2) * s
				s, c = s*cw+c*sw, c*cw-s*sw
				e1 *= d1
				e2 *= d2
			}
		}
	}
	// The hammer: a short burst of noise, and an attack of 2 ms.
	rng := rand.New(rand.NewPCG(uint64(note), 1))
	lp := 0.0
	for i := range min(n, synthRate/20) {
		t := float64(i) / synthRate
		lp += 0.3 * (rng.Float64()*2 - 1 - lp)
		out[i] += 0.4 * lp * math.Exp(-t/0.006)
		out[i] *= min(t/0.002, 1)
	}
	// The release of the last 50 ms, and every key as loud as another.
	peak := 0.0
	for i, v := range out {
		if r := n - i; r < synthRate/20 {
			out[i] *= float64(r) / (synthRate / 20)
		}
		peak = max(peak, math.Abs(v))
	}
	samples := make([]float32, n)
	for i, v := range out {
		samples[i] = float32(0.22 * v / peak)
	}
	return samples
}
