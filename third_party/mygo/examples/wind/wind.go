package main

import "math"

// wind is a visit's wind, which its seed picks: its look, the set of
// characters it shows; the heading it blows toward, in radians with y
// down; its speed, in DIPs a second; the length and width of a gust; and
// how far its path bends, and the size of a bend.
type wind struct {
	seed                                                   uint32
	look                                                   string
	heading, speed, gustLength, gustWidth, bend, bendScale float32
}

// newWind returns the wind of seed, with look.
func newWind(seed uint32, look string) wind {
	next := random(seed, "wind")
	between := func(lo, hi float32) float32 { return lo + (hi-lo)*float32(next()) }
	w := wind{seed: seed, look: look, heading: headingOf(next())}
	w.speed = between(42, 64)
	w.gustLength = between(380, 560)
	w.gustWidth = between(120, 190)
	w.bend = between(60, 110)
	w.bendScale = between(300, 440)
	return w
}

// headingOf returns the heading of a fraction e: across the pane and down
// it, from 18° to 42° below the horizontal, rightward for e below 0.5 and
// leftward above.
func headingOf(e float64) float32 {
	lo := math.Pi / 10
	a := lo + (math.Pi/4.3-lo)*math.Mod(e*2, 1)
	if e < 0.5 {
		return float32(a)
	}
	return float32(math.Pi - a)
}

// carried returns how far the wind has carried its gusts by its time,
// surging and slackening as it goes, and span the span of its long
// swells over a canvas of w×h DIPs, about one to it: what the app's
// vertex shader computes, once a frame.
func (w *wind) carried(time float32) float32 {
	n := noises{seed: w.seed}
	return w.speed * (time + n.value(time*0.09, 3.7, 0, 0x58))
}

func (w *wind) span(cw, ch float32) float32 {
	return 0.58 * (abs(cos(w.heading))*cw + abs(float32(math.Sin(float64(w.heading))))*ch)
}
