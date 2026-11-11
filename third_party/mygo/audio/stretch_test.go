package audio

import (
	"io"
	"math"
	"testing"
	"time"
)

// sine returns secs of sines of amplitude 0.5 at rate, one frequency for
// each channel.
func sine(rate int, secs float64, freqs ...float64) []float32 {
	ch := len(freqs)
	n := int(secs * float64(rate))
	s := make([]float32, n*ch)
	for i := range n {
		for c, f := range freqs {
			s[i*ch+c] = float32(0.5 * math.Sin(2*math.Pi*f*float64(i)/float64(rate)))
		}
	}
	return s
}

// channel returns channel c of out, interleaved stereo, from frame from
// to frame to.
func channel(out []float32, c, from, to int) []float32 {
	x := make([]float32, to-from)
	for i := range x {
		x[i] = out[2*(from+i)+c]
	}
	return x
}

// dominant returns the frequency, to the hertz, of x's strongest
// component between 50 Hz and 2 kHz, by the Goertzel algorithm.
func dominant(x []float32) float64 {
	best, at := 0.0, 0.0
	for f := 50.0; f <= 2000; f++ {
		w := 2 * math.Pi * f / sampleRate
		c := 2 * math.Cos(w)
		var s1, s2 float64
		for _, v := range x {
			s1, s2 = float64(v)+c*s1-s2, s1
		}
		if p := s1*s1 + s2*s2 - c*s1*s2; p > best {
			best, at = p, f
		}
	}
	return at
}

// smooth checks that channel c of out, a sine of amplitude 0.5 at up to
// maxHz, has no click and no gap from frame from to frame to: no step
// between frames much larger than the sine's steepest, and no period
// without its peaks.
func smooth(t *testing.T, out []float32, c, from, to int, maxHz float64, what string) {
	t.Helper()
	steep := 2 * math.Pi * maxHz * 0.5 / sampleRate
	for i := from + 1; i < to; i++ {
		if d := math.Abs(float64(out[2*i+c] - out[2*(i-1)+c])); d > 1.6*steep {
			t.Fatalf("%s: a click at frame %d: %v to %v", what, i, out[2*(i-1)+c], out[2*i+c])
		}
	}
	period := int(sampleRate/maxHz*2) + 1
	for i := from; i+period <= to; i += period / 2 {
		peak := float32(0)
		for _, v := range channel(out, c, i, i+period) {
			peak = max(peak, v, -v)
		}
		if peak < 0.35 {
			t.Fatalf("%s: a gap at frame %d: peak %v", what, i, peak)
		}
	}
}

// renderToEnd mixes p until it ends, and returns what it mixed.
func renderToEnd(t *testing.T, p *Player) []float32 {
	t.Helper()
	var out []float32
	for p.Playing() {
		out = append(out, render(256)...)
		if len(out) > 60*sampleRate*channels {
			t.Fatal("a minute and not at its end")
		}
	}
	return out
}

// framesPlayed returns the frames of out up to its last that is not silent.
func framesPlayed(out []float32) int {
	for i := len(out)/2 - 1; i >= 0; i-- {
		if out[2*i] != 0 || out[2*i+1] != 0 {
			return i + 1
		}
	}
	return 0
}

// At 1.5×, a sine stretched keeps its frequency, and one resampled, the
// default, rises a fifth.
func TestPreservesPitch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rate  int
		freqs []float64
	}{
		{"mono", sampleRate, []float64{220}},
		{"stereo", sampleRate, []float64{200, 300}},
		{"mono at 24 kHz", 24000, []float64{220}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSound(tc.rate, len(tc.freqs), sine(tc.rate, 2, tc.freqs...))
			for _, keep := range []bool{true, false} {
				setup(t)
				p := s.NewPlayer().SetSpeed(1.5)
				if keep {
					p.SetPreservesPitch(true)
				}
				p.Play()
				out := render(sampleRate)
				for c := range 2 {
					f := tc.freqs[min(c, len(tc.freqs)-1)]
					want := f
					if !keep {
						want = f * 1.5
					}
					if got := dominant(channel(out, c, sampleRate/4, sampleRate*3/4)); math.Abs(got-want) > want*0.01 {
						t.Errorf("preserving pitch %v: channel %d of %v Hz plays at %v Hz, want %v", keep, c, f, got, want)
					}
					if keep {
						smooth(t, out, c, 0, sampleRate, f, "stretched")
					}
				}
			}
		})
	}
}

// Stretched, a sound plays for its duration over the speed, and ends
// where it ends.
func TestPreservesPitchDuration(t *testing.T) {
	for _, ch := range []int{1, 2} {
		freqs := []float64{220, 330}[:ch]
		s := NewSound(sampleRate, ch, sine(sampleRate, 1, freqs...))
		for _, speed := range []float64{0.25, 0.5, 1, 1.25, 1.5, 2, 4, 16} {
			setup(t)
			p := s.NewPlayer().SetSpeed(speed).SetPreservesPitch(true)
			ended := make(chan struct{}, 1)
			p.OnEnd(func() { ended <- struct{}{} })
			p.Play()
			out := renderToEnd(t, p)
			got := frames(int64(framesPlayed(out)), sampleRate)
			want := time.Duration(float64(time.Second) / speed)
			// Within a hop.
			if d := got - want; d < -25*time.Millisecond || d > 25*time.Millisecond {
				t.Errorf("%d channels at %v×: played %v, want %v", ch, speed, got, want)
			}
			if p.Position() != p.Duration() {
				t.Errorf("%d channels at %v×: Position = %v at the end, want %v", ch, speed, p.Position(), p.Duration())
			}
			select {
			case <-ended:
			case <-time.After(time.Minute):
				t.Fatalf("%d channels at %v×: no OnEnd", ch, speed)
			}
		}
	}
}

// Resampling stays the default: a player gets no stretcher until it
// preserves pitch, and at speed 1 a stretched one plays as decoded.
func TestPreservesPitchIsOptIn(t *testing.T) {
	setup(t)
	samples := sine(sampleRate, 0.5, 220)
	p := NewSound(sampleRate, 1, samples).NewPlayer().SetSpeed(1.5).SetPreservesPitch(false)
	if p.PreservesPitch() || p.v.st != nil {
		t.Fatalf("PreservesPitch = %v, stretcher %v, by default", p.PreservesPitch(), p.v.st)
	}
	p.SetSpeed(1).SetPreservesPitch(true).Play()
	if !p.PreservesPitch() {
		t.Error("PreservesPitch = false after SetPreservesPitch(true)")
	}
	out := render(sampleRate / 4)
	for i := range sampleRate / 4 {
		if !near(out[2*i], samples[i]) {
			t.Fatalf("frame %d at 1× = %v, want %v as decoded", i, out[2*i], samples[i])
		}
	}
}

// Position follows the source at the speed while stretched, as resampled.
func TestPreservesPitchPosition(t *testing.T) {
	setup(t)
	hold(heldFrames)
	p := NewSound(sampleRate, 1, sine(sampleRate, 4, 220)).NewPlayer().SetSpeed(1.5).SetPreservesPitch(true)
	p.Play()
	render(3 * heldFrames)
	atFrame(t, p, 3*heldFrames, "200 ms heard at 1.5×")
	p.SetSpeed(2)
	render(heldFrames)
	// Of the 100 ms heard since, all were mixed at 1.5: what changed is
	// a hop or so behind.
	atFrame(t, p, 3*heldFrames+heldFrames*3/2, "100 ms more at 1.5×")
	p.Seek(time.Second)
	atFrame(t, p, sampleRate, "right after Seek")
	render(heldFrames + 1000)
	atFrame(t, p, sampleRate+2000, "1000 frames into the seek at 2×")
}

// The stretch goes on without a click or a gap as the speed changes, over
// a loop's seam, and as it turns on and off; a seek starts it over at the
// frame seeked to.
func TestStretchIsContinuous(t *testing.T) {
	setup(t)
	// 100 periods of 200 Hz: a seamless loop.
	samples := sine(sampleRate, 0.5, 200)
	p := NewSound(sampleRate, 1, samples).NewPlayer().SetLoop(true).SetPreservesPitch(true)
	p.Play()
	var out []float32
	for _, speed := range []float64{1, 1.25, 1.5, 2, 3, 0.75, 0.5, 4} {
		p.SetSpeed(speed)
		out = append(out, render(sampleRate/5)...)
	}
	// 8 × 200 ms at 1.95× on average: 6 loops.
	smooth(t, out, 0, 0, len(out)/2, 200, "changing speed and looping")
	if got := dominant(channel(out, 0, 0, len(out)/2)); got != 200 {
		t.Errorf("plays at %v Hz, want 200", got)
	}

	p.SetSpeed(1.5)
	out = render(sampleRate / 5)
	p.SetPreservesPitch(false)
	out = append(out, render(sampleRate/5)...)
	p.SetPreservesPitch(true)
	out = append(out, render(sampleRate/5)...)
	smooth(t, out, 0, 0, len(out)/2, 300, "turning on and off")
	if got := dominant(channel(out, 0, sampleRate/5+2000, 2*sampleRate/5)); got != 300 {
		t.Errorf("resampled at 1.5× again: %v Hz, want 300", got)
	}
	if got := dominant(channel(out, 0, 2*sampleRate/5+2000, 3*sampleRate/5)); got != 200 {
		t.Errorf("stretched again: %v Hz, want 200", got)
	}

	setup(t)
	ramped := ramp(sampleRate)
	q := NewSound(sampleRate, 1, ramped).NewPlayer().SetSpeed(2).SetPreservesPitch(true)
	q.Play()
	render(5000)
	q.Seek(500 * time.Millisecond)
	out = render(rampFrames + 20)
	// What a stretch begins with plays as it is.
	if i := rampFrames + 10; !near(out[2*i], ramped[24000+i]) {
		t.Errorf("frame %d after seeking = %v, want %v", i, out[2*i], ramped[24000+i])
	}
}

// sineDecoder decodes a sine of amplitude 0.5 in each of ch channels.
type sineDecoder struct {
	ch          int
	hz          float64
	pos, length int64
}

func (d *sineDecoder) SampleRate() int { return sampleRate }
func (d *sineDecoder) Channels() int   { return d.ch }
func (d *sineDecoder) Len() int64      { return d.length }

func (d *sineDecoder) Read(buf []float32) (int, error) {
	n := 0
	for ; n+d.ch <= len(buf) && d.pos < d.length; n += d.ch {
		for c := range d.ch {
			buf[n+c] = float32(0.5 * math.Sin(2*math.Pi*d.hz*float64(d.pos)/sampleRate))
		}
		d.pos++
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (d *sineDecoder) SetPosition(f int64) error { d.pos = f; return nil }

// A stream stretches as a sound does, to its end.
func TestStreamStretches(t *testing.T) {
	setup(t)
	p := NewPlayer(&sineDecoder{ch: 2, hz: 250, length: sampleRate}).SetSpeed(1.5).SetPreservesPitch(true)
	defer p.Close()
	p.Play()
	s := p.v.src.(*stream)
	var out []float32
	for p.Playing() {
		// The decoder keeps a second ahead: wait for it, as the device
		// would not.
		for {
			s.mu.Lock()
			ok := s.n >= 8192 || s.atEOF
			s.mu.Unlock()
			if ok {
				break
			}
			time.Sleep(time.Millisecond)
		}
		out = append(out, render(256)...)
		if len(out) > 10*sampleRate*channels {
			t.Fatal("10 s and not at its end")
		}
	}
	n := framesPlayed(out)
	if got, want := frames(int64(n), sampleRate), time.Second*2/3; got < want-25*time.Millisecond || got > want+25*time.Millisecond {
		t.Errorf("played %v, want %v", got, want)
	}
	smooth(t, out, 1, 0, n-2*sampleRate/250, 250, "a stream")
	if got := dominant(channel(out, 0, 0, n)); got != 250 {
		t.Errorf("plays at %v Hz, want 250", got)
	}
}

// Stretching allocates nothing as it mixes.
func TestStretchAllocatesNothing(t *testing.T) {
	setup(t)
	p := NewSound(sampleRate, 2, sine(sampleRate, 0.5, 200, 300)).NewPlayer().
		SetLoop(true).SetSpeed(1.5).SetPreservesPitch(true)
	p.Play()
	buf := make([]float32, 480*channels)
	speeds := []float64{1.5, 2, 0.75}
	i := 0
	if n := testing.AllocsPerRun(500, func() {
		mix(buf)
		if i++; i%50 == 0 {
			p.SetSpeed(speeds[i/50%len(speeds)])
		}
	}); n != 0 {
		t.Errorf("%v allocations a buffer", n)
	}
}
