package dictation

import "time"

// Typewriter paces streaming recognition text so it flows out at a steady,
// readable speed instead of jumping by whole partials, after Doubao's
// ASRTypeWriter and TypewriterSpring (docs/research/voice-typewriter.md).
// The spring keeps a small backlog of unshown characters: the further the
// display falls behind the recognizer, the faster it types; when it catches
// up, it slows to a calm pace. Recognition keeps revising text already shown
// (refined segments, the rewrite after release, the streaming tail); those characters
// change in place, so the line never retreats and only new characters are
// typed. A revision shorter than what is shown cuts the display to it.
type Typewriter struct {
	session string
	target  []rune
	shown   float64
	at      []time.Time // when each shown rune appeared, for its fade
	last    time.Time
}

const (
	// balance is the backlog the spring settles around, in runes.
	balance = 6
	// interval is the pace at balance; elastic shortens or lengthens it by
	// this fraction per rune of backlog above or below balance.
	interval = 45 * time.Millisecond
	elastic  = 0.25
	// The pace stays between these. A finished recognition drains at the
	// fastest pace, so the final text never waits on the animation long.
	minInterval = 12 * time.Millisecond
	maxInterval = 90 * time.Millisecond
	// FadeIn is how long a new character takes to reach full opacity, from
	// fadeFrom: a short, partial fade reads as ink settling, not blinking.
	FadeIn   = 150 * time.Millisecond
	fadeFrom = 0.35
)

// Frame is what the pill draws now.
type Frame struct {
	Text string
	// Opacity holds the opacities of the runes at the end of Text that are
	// still fading in, oldest first; the runes before them are opaque.
	Opacity []float32
	// Animating asks for another frame.
	Animating bool
}

// Update advances to now toward text of session. final means recognition has
// finished, so the rest drains at full speed.
func (t *Typewriter) Update(session, text string, final bool, now time.Time) Frame {
	if session != t.session {
		*t = Typewriter{session: session, last: now}
	}
	target := []rune(text)
	if n := len(target); int(t.shown) > n {
		t.shown = float64(n)
		t.at = t.at[:n]
	}
	t.target = target
	elapsed := now.Sub(t.last)
	if elapsed < 0 || elapsed > 250*time.Millisecond {
		// A stalled or first frame advances one frame's worth, not a jump.
		elapsed = 16 * time.Millisecond
	}
	t.last = now
	backlog := float64(len(target)) - t.shown
	pace := minInterval
	if !final {
		scale := 1 + elastic*(backlog-balance)
		if scale < 0.1 {
			scale = 0.1
		}
		pace = time.Duration(float64(interval) / scale)
		pace = max(minInterval, min(maxInterval, pace))
	}
	t.shown = min(float64(len(target)), t.shown+float64(elapsed)/float64(pace))
	for len(t.at) < int(t.shown) {
		t.at = append(t.at, now)
	}
	frame := Frame{Text: string(target[:int(t.shown)]), Animating: int(t.shown) < len(target)}
	for i := len(t.at) - 1; i >= 0; i-- {
		age := now.Sub(t.at[i])
		if age >= FadeIn {
			break
		}
		frame.Opacity = append([]float32{fadeFrom + (1-fadeFrom)*float32(age)/float32(FadeIn)}, frame.Opacity...)
		frame.Animating = true
	}
	return frame
}
