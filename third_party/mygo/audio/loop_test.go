package audio

import (
	"testing"
	"time"
)

// A tune's intro plays once, then its loop again and again, without a gap.
func TestLoopRange(t *testing.T) {
	setup(t)
	samples := ramp(100) // frame i is i/200
	p := NewSound(sampleRate, 1, samples).NewPlayer().
		SetLoopRange(frames(30, sampleRate), frames(60, sampleRate))
	p.Play()
	out := render(200)
	for i := range 200 {
		want := i
		if i >= 60 {
			want = 30 + (i-60)%30
		}
		if !near(out[2*i], samples[want]) {
			t.Fatalf("frame %d = %v, want frame %d's %v", i, out[2*i], want, samples[want])
		}
	}
	// 200 frames in: 60 + 140 = 4 loops of 30 and 20 more.
	if got, want := p.Position(), frames(50, sampleRate); got != want {
		t.Errorf("Position = %v, want %v", got, want)
	}
	// Looping no more, it plays on past the loop's end, to the end (after
	// what it read ahead, a few milliseconds).
	p.SetLoop(false)
	out = render(1000)
	last := -1
	for i := range 1000 {
		if out[2*i] != 0 {
			last = i
		}
	}
	if last < 0 || !near(out[2*last], samples[99]) || p.Playing() {
		t.Errorf("after SetLoop(false): last frame %d = %v, playing %v", last, out[2*max(last, 0)], p.Playing())
	}
	if !near(out[2*(last-39)], samples[60]) {
		t.Errorf("frames 60 to 99 do not end it: %v", out[2*(last-39)])
	}
}

func TestStreamLoopRange(t *testing.T) {
	setup(t)
	d := &genDecoder{ch: 1, length: 5000} // frame i is (i%1000)/1000
	p := NewPlayer(d).SetLoopRange(frames(1000, sampleRate), frames(3000, sampleRate))
	waitDecoded(t, p)
	p.Play()
	out := make([]float32, 0, 9000*channels)
	for len(out) < 9000*channels {
		chunk := make([]float32, 64*channels)
		s := p.v.src.(*stream)
		for {
			s.mu.Lock()
			n := s.n
			s.mu.Unlock()
			if n >= 64 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		mix(chunk)
		out = append(out, chunk...)
	}
	for i := range 9000 {
		f := i
		if i >= 3000 {
			f = 1000 + (i-3000)%2000
		}
		if want := float32(f%1000) / 1000; !near(out[2*i], want) {
			t.Fatalf("frame %d = %v, want frame %d's %v", i, out[2*i], f, want)
		}
	}
	p.Close()
}

func TestLoopRangeOutOfRange(t *testing.T) {
	setup(t)
	p := NewSound(sampleRate, 1, ramp(100)).NewPlayer()
	// An end before the start loops it all.
	p.SetLoopRange(frames(60, sampleRate), frames(30, sampleRate))
	if p.v.loopStart != 0 || p.v.loopEnd != 0 || !p.Loop() {
		t.Errorf("loop %d to %d, looping %v", p.v.loopStart, p.v.loopEnd, p.Loop())
	}
	// Past the end, clamped to it.
	p.SetLoopRange(frames(10, sampleRate), time.Hour)
	if p.v.loopStart != 10 || p.v.loopEnd != 100 {
		t.Errorf("loop %d to %d", p.v.loopStart, p.v.loopEnd)
	}
}
