package main

import (
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
)

// The edge glow's colors, and how its stops move: every shuffleEvery
// seconds they head for places of their own, picked at random, over
// shuffleTime, easing in and out, as the SwiftUI recreations of Apple
// Intelligence's glow move theirs.
var glowColors = []ui.Color{
	ui.Hex("#BC82F3"), ui.Hex("#F5B9EA"), ui.Hex("#8D9FFF"),
	ui.Hex("#FF6778"), ui.Hex("#FFBA71"), ui.Hex("#C686FF"),
}

const shuffleEvery, shuffleTime = 0.5, 1.0

// stop is a stop of the gradient: its color in Oklab, in which SwiftUI
// blends a gradient's colors, and where it lies around the window, from 0
// to 1.
type stop struct {
	l, a, b, at float64
}

// stops are the gradient's stops as they move from where they were (from)
// to where they head (to), since when (since), the next move due at next.
type stops struct {
	from, to    []stop
	since, next float64
	random      func() float64
}

func newStops(random func() float64) stops {
	s := stops{random: random}
	s.to = s.shuffle()
	s.from = s.to
	return s
}

// shuffle returns the colors at places picked at random, in order around
// the window: the stop in each place takes another color as the order
// changes, which the move blends.
func (s *stops) shuffle() []stop {
	out := make([]stop, len(glowColors))
	for i, c := range glowColors {
		l, a, b := oklab(c)
		out[i] = stop{l, a, b, s.random()}
	}
	slices.SortFunc(out, func(a, b stop) int {
		switch {
		case a.at < b.at:
			return -1
		case a.at > b.at:
			return 1
		}
		return 0
	})
	return out
}

// step starts the next move once it is due at now, from where the stops
// show.
func (s *stops) step(now float64) {
	if now < s.next {
		return
	}
	s.from = s.show(now)
	s.to, s.since, s.next = s.shuffle(), now, now+shuffleEvery
}

// show returns the stops as they show at now.
func (s *stops) show(now float64) []stop {
	k := ease(min(max((now-s.since)/shuffleTime, 0), 1))
	out := make([]stop, len(s.to))
	for i := range out {
		f, t := s.from[i], s.to[i]
		out[i] = stop{f.l + (t.l-f.l)*k, f.a + (t.a-f.a)*k, f.b + (t.b-f.b)*k, f.at + (t.at-f.at)*k}
	}
	return out
}

// gradient returns the stops at now as a gradient's: their colors at
// their places, and at 0 and at 1 the color between the last and the
// first, blended in Oklab, so that the colors go round without a seam
// where the gradient starts.
func (s *stops) gradient(now float64) []ui.GradientStop {
	st := s.show(now)
	first, last := st[0], st[len(st)-1]
	k := (1 - last.at) / max(first.at+1-last.at, 1e-6)
	seam := srgb(last.l+(first.l-last.l)*k, last.a+(first.a-last.a)*k, last.b+(first.b-last.b)*k)
	out := make([]ui.GradientStop, 0, len(st)+2)
	out = append(out, ui.GradientStop{Color: seam, At: 0})
	for _, x := range st {
		out = append(out, ui.GradientStop{Color: srgb(x.l, x.a, x.b), At: float32(x.at)})
	}
	return append(out, ui.GradientStop{Color: seam, At: 1})
}

// srgb returns the color of Oklab's (l, a, b) in sRGB.
func srgb(l, a, b float64) ui.Color {
	lc := l + 0.3963377774*a + 0.2158037573*b
	mc := l - 0.1055613458*a - 0.0638541728*b
	sc := l - 0.0894841775*a - 1.2914855480*b
	lc, mc, sc = lc*lc*lc, mc*mc*mc, sc*sc*sc
	enc := func(v float64) uint8 {
		v = min(max(v, 0), 1)
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		return uint8(math.Round(v * 255))
	}
	return ui.RGB(enc(4.0767416621*lc-3.3077115913*mc+0.2309699292*sc),
		enc(-1.2684380046*lc+2.6097574011*mc-0.3413193965*sc),
		enc(-0.0041960863*lc-0.7034186147*mc+1.7076147010*sc))
}

// oklab returns c, in sRGB, in Oklab.
func oklab(c ui.Color) (l, a, b float64) {
	lin := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	r, g, bl := lin(c.R), lin(c.G), lin(c.B)
	lc := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*bl)
	mc := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*bl)
	sc := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*bl)
	return 0.2104542553*lc + 0.7936177850*mc - 0.0040720468*sc,
		1.9779984951*lc - 2.4285922050*mc + 0.4505937099*sc,
		0.0259040371*lc + 0.7827717662*mc - 0.8086757660*sc
}
