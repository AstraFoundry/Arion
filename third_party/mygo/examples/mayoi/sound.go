package main

import (
	"io"
	"math"
	"math/rand/v2"
	"sort"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/audio"
)

const sfxRate = 32000

// sounds are the game's effects, synthesized once, and its music, made as
// it plays.
type sounds struct {
	sing, hit, calm, stone, level, hurt, beat, choose, join, over *audio.Sound
	music                                                         *audio.Player
	tune                                                          *tune
	rng                                                           *rand.Rand
	stones                                                        int     // stones picked up in a row
	lastStone                                                     float64 // when the last one was, in steps
}

func newSounds() *sounds {
	rng := rand.New(rand.NewPCG(9, 7))
	noise := func() float64 { return rng.Float64()*2 - 1 }
	t := newTune()
	return &sounds{
		// A sung "ah": a fifth up, with a little vibrato.
		sing: synth(0.16, func(t float64) float64 {
			f := 660 * (1 + 0.012*math.Sin(2*math.Pi*7*t))
			return (math.Sin(2*math.Pi*f*t) + 0.3*math.Sin(4*math.Pi*f*t) + 0.15*math.Sin(6*math.Pi*f*t)) *
				math.Exp(-t*14) * min(t*60, 1) * 0.16
		}),
		hit: synth(0.05, func(t float64) float64 { return noise() * math.Exp(-t*80) * 0.25 }),
		calm: synth(0.22, func(t float64) float64 {
			return (math.Sin(2*math.Pi*(300+1500*t)*t)*0.4 + noise()*0.15*math.Exp(-t*40)) * math.Exp(-t*16)
		}),
		// A pebble's click.
		stone: synth(0.06, func(t float64) float64 {
			return (math.Sin(2*math.Pi*2300*t) + 0.5*math.Sin(2*math.Pi*3400*t)) * math.Exp(-t*70) * 0.22
		}),
		level: synth(0.8, func(t float64) float64 {
			var s float64
			for i, f := range []float64{587.33, 739.99, 880, 1174.66} {
				if nt := t - float64(i)*0.08; nt > 0 {
					s += ep(f, nt) * math.Exp(-nt*3)
				}
			}
			return s * 0.14
		}),
		hurt: synth(0.3, func(t float64) float64 {
			return (math.Sin(2*math.Pi*(160-220*t)*t)*0.7 + noise()*0.3) * math.Exp(-t*12) * 0.45
		}),
		// Taki's beat: a kick and a crash.
		beat: synth(0.6, func(t float64) float64 {
			kick := math.Sin(2*math.Pi*(55+90*math.Exp(-t*30))*t) * math.Exp(-t*9)
			return (kick*0.8 + noise()*0.3*math.Exp(-t*5)) * 0.45
		}),
		choose: synth(0.35, func(t float64) float64 { return (ep(880, t) + 0.6*ep(1318.5, t-0.05)) * math.Exp(-t*7) * 0.15 }),
		// A bandmate joins: a strummed D major chord.
		join: synth(1.4, func(t float64) float64 {
			var s float64
			for i, m := range []int{50, 57, 62, 66, 69, 74} {
				if nt := t - float64(i)*0.025; nt > 0 {
					s += pluck(midi(m), nt)
				}
			}
			return s * 0.12
		}),
		over: synth(1.6, func(t float64) float64 {
			var s float64
			for i, m := range []int{74, 71, 69, 66, 62} {
				if nt := t - float64(i)*0.25; nt > 0 {
					s += ep(midi(m), nt) * math.Exp(-nt*2.5)
				}
			}
			return s * 0.15
		}),
		// The first bar is an intro: then the rest loops.
		music: audio.NewPlayer(t).SetLoopRange(time.Duration(math.Round(barLen*float64(time.Second))), 0).SetVolume(0.5),
		tune:  t,
		rng:   rng,
	}
}

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

// play plays the sounds of what happened in step n, a little different
// each time, as games vary effects that play often. Stones picked up in a
// row climb in pitch.
func (s *sounds) play(evs []event, n float64) {
	sang := false
	for _, e := range evs {
		var snd *audio.Sound
		volume, speed := 1.0, 0.94+s.rng.Float64()*0.12
		switch e {
		case evSing:
			if sang {
				continue // one for a volley
			}
			sang, snd, volume = true, s.sing, 0.7
			// On the notes of the tune's scale.
			speed = math.Pow(2, float64([]int{0, 2, 4, 7, 9}[s.rng.IntN(5)])/12)
		case evHit:
			snd, volume = s.hit, 0.5
		case evKill:
			snd, volume = s.calm, 0.55
		case evStone:
			if n-s.lastStone > 30 {
				s.stones = 0
			}
			s.lastStone = n
			snd, speed = s.stone, math.Pow(2, float64(min(s.stones, 12))/12)
			s.stones++
		case evLevel:
			snd, speed = s.level, 1
		case evHurt:
			snd = s.hurt
		case evBeat:
			snd = s.beat
		case evChoose:
			snd, speed = s.choose, 1
		case evJoin:
			snd, speed = s.join, 1
		case evOver:
			snd, speed = s.over, 1
		}
		snd.NewPlayer().SetVolume(volume).SetSpeed(speed).Play()
	}
}

// setBand tells the music who plays: each bandmate adds their part.
func (s *sounds) setBand(band []int) {
	var mask uint32
	for _, m := range band {
		mask |= 1 << m
	}
	s.tune.band.Store(mask)
}

func midi(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

// ep is an electric piano's tone: a sine with a soft second harmonic and
// a bell-like partial that fades fast.
func ep(f, t float64) float64 {
	if t < 0 {
		return 0
	}
	return (math.Sin(2*math.Pi*f*t) + 0.25*math.Sin(4*math.Pi*f*t)*math.Exp(-t*3) +
		0.2*math.Sin(2*math.Pi*f*14*t)*math.Exp(-t*25)) * min(t/0.004, 1)
}

// pluck is a soft plucked string.
func pluck(f, t float64) float64 {
	if t < 0 {
		return 0
	}
	return (math.Sin(2*math.Pi*f*t) + 0.5*math.Sin(4*math.Pi*f*t)*math.Exp(-t*6) +
		0.25*math.Sin(6*math.Pi*f*t)*math.Exp(-t*10)) * math.Exp(-t*2.2) * min(t/0.003, 1)
}

// tune is the music: a lo-fi arrangement of a song by MyGo's author,
// transcribed from their recording, in E major and 6/8, with vinyl
// crackle and a wavering tape, made as it plays. Tomori hums the melody
// over a Rhodes; Anon brings guitar arpeggios, Soyo the bass, Taki the
// drums and Rāna an echo of the melody an octave up. Its first bar, the
// Rhodes alone, is an intro.
type tune struct {
	pos  int64
	band atomic.Uint32 // who plays, by bit
	// gains fade the bandmates in and out, per sample.
	gains [memberCount]float64
	parts []part
	rng   *rand.Rand
}

// A part is what one instrument plays, by whom.
type part struct {
	member int
	voice  int
	hits   []hit
}

// The parts' voices.
const (
	rhodes = iota
	hum
	guitar
	bass
	drums
	echo
)

// A hit is a note or a drum of a part, at a time of the loop.
type hit struct {
	t     float64 // seconds
	freqs []float64
	dur   float64 // seconds a hummed note lasts
	drum  int     // 0 for a note; kick, snare or hat
	vel   float64
}

const (
	tuneRate  = 22050
	eighthLen = 0.33 // seconds
	barLen    = 6 * eighthLen
	tuneBars  = 9
)

const (
	_ = iota
	kick
	snare
	hat
)

// Chords, as the Rhodes voices them: from the bass up.
var (
	emaj7   = []int{52, 56, 59, 63}
	fsm7    = []int{54, 57, 61, 64}
	dmaj7   = []int{54, 57, 61, 62} // over F♯
	gsm7    = []int{56, 59, 63, 66}
	b7sus4  = []int{59, 64, 66, 69}
	badd9   = []int{59, 61, 63, 66}
	fs7     = []int{54, 58, 61, 64}
	csm7    = []int{56, 59, 61, 64}
	harmony = [][2][]int{
		{emaj7, emaj7}, {emaj7, fsm7}, {dmaj7, dmaj7}, {gsm7, b7sus4},
		{badd9, badd9}, {badd9, fs7}, {csm7, fs7}, {badd9, fs7},
	}
	// The bass's roots, a half bar each, and the intro's.
	roots     = [][2]int{{40, 40}, {40, 42}, {42, 42}, {44, 47}, {47, 47}, {47, 40}, {49, 42}, {47, 40}}
	introRoot = 40
)

// The melody, a bar each: its notes as eighth, MIDI note and eighths long.
var melody = [][][3]int{
	{{0, 71, 3}, {3, 71, 1}, {4, 73, 1}, {5, 71, 1}},
	{{0, 73, 1}, {1, 74, 1}, {2, 71, 1}, {3, 73, 3}},
	{{0, 73, 2}, {2, 74, 1}, {3, 73, 3}},
	{{0, 68, 3}, {3, 76, 1}, {4, 75, 2}},
	{{0, 75, 6}},
	{{0, 75, 3}, {3, 66, 3}},
	{{0, 66, 1}, {1, 75, 1}, {2, 73, 1}, {3, 71, 1}, {4, 66, 1}, {5, 70, 1}},
	{{0, 71, 6}},
}

func newTune() *tune {
	t := &tune{rng: rand.New(rand.NewPCG(5, 5))}
	parts := map[int]*part{}
	add := func(member, voice int, h hit) {
		p := parts[voice]
		if p == nil {
			p = &part{member: member, voice: voice}
			parts[voice] = p
		}
		p.hits = append(p.hits, h)
	}
	notes := func(ms []int, shift int) []float64 {
		fs := make([]float64, len(ms))
		for i, m := range ms {
			fs[i] = midi(m + shift)
		}
		return fs
	}
	at := func(bar, eighth int) float64 { return float64(bar*6+eighth) * eighthLen }
	add(vocalist, rhodes, hit{t: 0, freqs: notes(fs7, 0), vel: 1})
	add(bassist, bass, hit{t: 0, freqs: []float64{midi(introRoot)}, vel: 1})
	for b := 1; b < tuneBars; b++ {
		for half := range 2 {
			chord, root := harmony[b-1][half], roots[b-1][half]
			e := half * 3
			// The Rhodes, on each half bar, softer on the second.
			add(vocalist, rhodes, hit{t: at(b, e), freqs: notes(chord, 0), vel: 1 - 0.3*float64(half)})
			// Anon's arpeggio, up the chord an octave higher.
			for i, k := range []int{1, 2, 3} {
				add(guitarist, guitar, hit{t: at(b, e+i), freqs: []float64{midi(chord[k] + 12)}, vel: 1 - 0.2*float64(i)})
			}
			// Soyo: the root, then its fifth.
			add(bassist, bass, hit{t: at(b, e), freqs: []float64{midi(root)}, vel: 1})
			add(bassist, bass, hit{t: at(b, e+2), freqs: []float64{midi(root + 7)}, vel: 0.6})
		}
		// Taki: half time, a kick on the bar, the snare on its middle.
		add(drummer, drums, hit{t: at(b, 0), drum: kick, vel: 1})
		add(drummer, drums, hit{t: at(b, 3), drum: snare, vel: 1})
		add(drummer, drums, hit{t: at(b, 5), drum: kick, vel: 0.5})
		for i := range 6 {
			add(drummer, drums, hit{t: at(b, i) + 0.002, drum: hat, vel: []float64{0.9, 0.4, 0.6, 0.9, 0.4, 0.6}[i]})
		}
		for _, n := range melody[b-1] {
			h := hit{t: at(b, n[0]), freqs: []float64{midi(n[1])}, dur: float64(n[2])*eighthLen - 0.04, vel: 1}
			add(vocalist, hum, h)
			// Rāna's echo: an eighth later, an octave up.
			h.t += eighthLen
			h.freqs = []float64{midi(n[1] + 12)}
			add(lead, echo, h)
		}
	}
	for v := range echo + 1 {
		p := parts[v]
		sort.SliceStable(p.hits, func(i, j int) bool { return p.hits[i].t < p.hits[j].t })
		t.parts = append(t.parts, *p)
	}
	return t
}

func (t *tune) SampleRate() int { return tuneRate }
func (t *tune) Channels() int   { return 2 }
func (t *tune) Len() int64      { return int64(math.Round(tuneBars * barLen * tuneRate)) }
func (t *tune) SetPosition(f int64) error {
	t.pos = f
	return nil
}

// noiseAt is noise that is the same at the same time.
func noiseAt(x float64) float64 {
	v := math.Sin(x*12.9898) * 43758.5453
	return (v-math.Floor(v))*2 - 1
}

// voiced is a hummed note: a soft "oo" with a vibrato that grows, held
// for dur, then let go.
func voiced(f, t, dur float64) float64 {
	if t < 0 {
		return 0
	}
	env := min(t/0.05, 1)
	if t > dur {
		env *= math.Exp(-(t - dur) * 12)
	} else {
		env *= 1 - 0.2*t/max(dur, 0.1)
	}
	depth := f * 0.007 * min(t/0.35, 1)
	ph := 2*math.Pi*f*t + depth/5.5*math.Sin(2*math.Pi*5.5*t)
	return (math.Sin(ph) + 0.3*math.Sin(2*ph) + 0.08*math.Sin(3*ph)) * env
}

// sound returns what hit h of a voice sounds like, dt seconds after it.
func sound(voice int, h hit, dt float64) float64 {
	switch h.drum {
	case kick:
		return math.Sin(2*math.Pi*(48+70*math.Exp(-dt*35))*dt) * math.Exp(-dt*8) * min(dt/0.002, 1) * 0.8 * h.vel
	case snare:
		return (noiseAt(dt*977)*0.45*math.Exp(-dt*16) + math.Sin(2*math.Pi*185*dt)*0.25*math.Exp(-dt*25)) * 0.45 * h.vel
	case hat:
		return noiseAt(dt*1931) * math.Exp(-dt*70) * 0.14 * h.vel
	}
	var s float64
	for i, f := range h.freqs {
		switch voice {
		case rhodes:
			// The chord strummed a little.
			at := dt - float64(i)*0.014
			s += ep(f, at) * math.Exp(-max(at, 0)*0.9) * 0.16
		case hum:
			s += voiced(f, dt, h.dur) * 0.22
		case guitar:
			s += pluck(f, dt) * 0.16
		case bass:
			if dt > 0 {
				s += (math.Sin(2*math.Pi*f*dt) + 0.3*math.Sin(4*math.Pi*f*dt)) * math.Exp(-dt*2) * min(dt/0.01, 1) * 0.5
			}
		case echo:
			if dt > 0 { // a clean guitar, far away
				s += (math.Sin(2*math.Pi*f*dt) + 0.2*math.Sin(4*math.Pi*f*dt)) * math.Exp(-dt*3.5) * min(dt/0.004, 1) * 0.06
			}
		}
	}
	return s * h.vel
}

func (t *tune) Read(buf []float32) (int, error) {
	band := t.band.Load() | 1 // Tomori always plays
	n := 0
	for ; n+2 <= len(buf) && t.pos < t.Len(); n += 2 {
		sec := float64(t.pos) / tuneRate
		// The tape's wow: the pitch wavers a little.
		wsec := sec + 0.0018*math.Sin(2*math.Pi*0.55*sec)
		for m := range memberCount {
			want := 0.0
			if band&(1<<m) != 0 {
				want = 1
			}
			t.gains[m] += (want - t.gains[m]) * 0.0002
		}
		var l, r float64
		for _, p := range t.parts {
			g := t.gains[p.member]
			if g < 0.001 {
				continue
			}
			hits := p.hits
			i := sort.Search(len(hits), func(i int) bool { return hits[i].t > wsec }) - 1
			var v float64
			// The hit sounding and those before it, still ringing.
			for j := max(i-3, 0); j <= i; j++ {
				v += sound(p.voice, hits[j], wsec-hits[j].t)
			}
			v *= g
			pan := [...]float64{rhodes: 0, hum: 0, guitar: -0.35, bass: 0, drums: 0.05, echo: 0.4}[p.voice]
			l += v * (1 - pan)
			r += v * (1 + pan)
		}
		// Vinyl: a soft hiss, and now and then a crackle.
		crackle := 0.004 * (t.rng.Float64()*2 - 1)
		if t.rng.IntN(2500) == 0 {
			crackle += (t.rng.Float64()*2 - 1) * 0.1
		}
		// Warm: soft saturation.
		buf[n] = float32(math.Tanh(l*0.9) + crackle)
		buf[n+1] = float32(math.Tanh(r*0.9) + crackle)
		t.pos++
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}
