package audio

import (
	"testing"
	"time"
)

// heldFrames is what the output holds ahead in these tests: 100 ms.
const heldFrames = sampleRate / 10

// atFrame checks that p's position is source frame f.
func atFrame(t *testing.T, p *Player, f int64, when string) {
	t.Helper()
	got, want := p.Position(), frames(f, sampleRate)
	if d := got - want; d < -time.Microsecond || d > time.Microsecond {
		t.Errorf("%s: Position = %v, want %v (frame %d)", when, got, want, f)
	}
}

func longSound() *Sound { return NewSound(sampleRate, 1, make([]float32, 10*sampleRate)) }

// Position is what is heard: it stands still while the device fills, and
// then lags what was mixed by what the device holds.
func TestPositionIsHeard(t *testing.T) {
	setup(t)
	hold(heldFrames)
	p := longSound().Play()
	render(heldFrames / 2)
	atFrame(t, p, 0, "half the device filled")
	render(heldFrames / 2)
	atFrame(t, p, 0, "the device filled")
	render(3000)
	atFrame(t, p, 3000, "3000 frames on")
	// A device opening again holds nothing of what was mixed before,
	// which it dropped: the position goes on from what was mixed.
	hold(heldFrames)
	atFrame(t, p, heldFrames+3000, "a device opened again")
}

// Right after a seek, Position is where it seeked to, and goes on from
// there once the device plays what was mixed since.
func TestPositionHoldsAtSeek(t *testing.T) {
	setup(t)
	hold(heldFrames)
	p := longSound().Play()
	render(2 * heldFrames)
	p.Seek(2 * time.Second)
	atFrame(t, p, 2*sampleRate, "right after Seek")
	render(heldFrames - 100)
	atFrame(t, p, 2*sampleRate, "before the seek is heard")
	render(600)
	atFrame(t, p, 2*sampleRate+500, "500 frames into the seek heard")
	p.Stop()
	atFrame(t, p, 0, "after Stop")
}

// Paused, a player goes on for as long as the device plays what it holds,
// then stands still; playing again, it stands still until that is heard.
func TestPositionPlaysOutPause(t *testing.T) {
	setup(t)
	hold(heldFrames)
	p := longSound().Play()
	render(10000)
	p.Pause()
	atFrame(t, p, 10000-heldFrames, "at Pause")
	render(2000)
	atFrame(t, p, 12000-heldFrames, "2000 frames after Pause")
	// It faded out over rampFrames after the pause.
	render(2 * heldFrames)
	atFrame(t, p, 10000+rampFrames, "played out")
	render(1000)
	atFrame(t, p, 10000+rampFrames, "paused, played out")
	p.Play()
	render(heldFrames - 100)
	atFrame(t, p, 10000+rampFrames, "before playing again is heard")
	render(600)
	atFrame(t, p, 10000+rampFrames+500, "500 frames into playing again")
}

// With a speed, what the device holds is that many more frames of the
// source.
func TestPositionAtSpeed(t *testing.T) {
	setup(t)
	hold(heldFrames)
	p := longSound().NewPlayer().SetSpeed(2)
	p.Play()
	render(3 * heldFrames)
	atFrame(t, p, 4*heldFrames, "200 ms heard at speed 2")
	p.SetSpeed(0.5)
	render(heldFrames * 3 / 2)
	// Of the 150 ms heard since, 100 ms were mixed at 2, 50 ms at 0.5.
	atFrame(t, p, 4*heldFrames+2*heldFrames+heldFrames/4, "after slowing down")
}

// A looping player's position heard comes back to the loop's start.
func TestPositionWrapsLoops(t *testing.T) {
	setup(t)
	hold(50)
	p := NewSound(sampleRate, 1, ramp(100)).NewPlayer().
		SetLoopRange(frames(30, sampleRate), frames(60, sampleRate))
	p.Play()
	render(160)
	atFrame(t, p, 50, "110 frames heard")
	render(10)
	// 120 heard: 60, then two loops of 30.
	atFrame(t, p, 30, "120 frames heard")
}

// A stream that is late stands still while it is, as heard.
func TestLateStreamPositionIsHeard(t *testing.T) {
	setup(t)
	hold(heldFrames)
	src := &lateSource{ready: 1000}
	p := newPlayer(src, sampleRate, 1, -1)
	p.Play()
	// It plays 998 frames of the 1000, reading two ahead to interpolate,
	// then waits for 1002.
	render(heldFrames + 2000)
	src.ready = 1 << 30
	render(500)
	atFrame(t, p, 998, "the device playing the wait")
	render(heldFrames)
	atFrame(t, p, 998+500, "the wait heard")
}

// A lateSource has ready frames decoded, and then waits for more.
type lateSource struct{ ready int }

func (s *lateSource) read(dst []float32) (int, bool) {
	n := min(len(dst), s.ready)
	clear(dst[:n])
	s.ready -= n
	return n, false
}

func (*lateSource) seek(int64)                 {}
func (*lateSource) setLoop(bool, int64, int64) {}
func (*lateSource) close()                     {}
