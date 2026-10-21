package kit

import (
	"math"
	"time"

	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// markBars is the app mark's bar silhouette, as cmd/arion-icon draws it,
// relative to its tallest bar.
var markBars = [5]float32{230.0 / 560, 420.0 / 560, 1, 360.0 / 560, 250.0 / 560}

// markDots is the waiting dot each bar folds into: the center bar becomes
// the center dot and each outer pair merges into the dot beside it, so no
// shape appears or leaves while the mark folds.
var markDots = [5]int{0, 0, 1, 2, 2}

// markRest is how much of the silhouette stands in silence: the meter
// still reads as the mark before a voice moves it.
const markRest = 0.4

// markStagger delays each bar's fold by its distance from the center, so
// the mark gathers from the middle out rather than moving as one block.
const markStagger = 0.12

// markWave is one pass of the pulse through the waiting dots.
const markWave = 1400 * time.Millisecond

// markShape is one filled rounded shape of the mark.
type markShape struct {
	Rect  ui.Rect
	Alpha float32
}

// markLayout places the mark at height h: the live dot, then the five
// bars standing at levels, each folded fold of the way into its waiting
// dot. wave is the waiting dots' pulse phase, from 0 to 1.
//
// The pulse is a Gaussian that travels across the dots, as Grok's thinking
// dots: the dot it passes lifts, grows and brightens while the others rest
// half dimmed. Bars and dots are one shape at every fold, a rounded rect
// whose width, height, position and alpha interpolate, so folding and
// unfolding are the same motion reversed.
func markLayout(h float32, levels [5]float32, fold, wave float32) (width float32, live markShape, bars [5]markShape) {
	bar := 0.15 * h
	dot := bar * 5 / 3
	open := 1 - clamp01(fold)
	// The live dot leads the bars while the microphone is open and shrinks
	// away as they fold, taking its gap with it.
	lead := 0.9 * h * open
	radius := 0.2 * h * open
	live = markShape{Rect: ui.Rect{X: 0, Y: h/2 - radius, W: 2 * radius, H: 2 * radius}, Alpha: open}
	center := lead + 4.5*bar
	for i := range bars {
		distance := float32(absInt(i - 2))
		p := clamp01((fold - markStagger*distance) / (1 - 2*markStagger))
		j := markDots[i]
		d := float32(math.Abs(float64(wave - float32(j)/3)))
		d = min(d, 1-d)
		pulse := float32(math.Exp(-float64(d*d) / 0.045))
		size := dot * (0.84 + 0.22*pulse)
		standing := bar + (h*markBars[i]-bar)*(markRest+(1-markRest)*clamp01(levels[i]))
		w := lerp(bar, size, p)
		ht := lerp(standing, size, p)
		x := lerp(lead+bar/2+float32(i)*2*bar, center+float32(j-1)*3*bar, p)
		y := h/2 - 0.2*h*pulse*p
		alpha := lerp(1, 0.5+0.5*pulse, p)
		if i%2 == 1 {
			// Odd bars merge into a neighbor's dot and leave their alpha behind.
			alpha *= 1 - p
		}
		bars[i] = markShape{Rect: ui.Rect{X: x - w/2, Y: y - ht/2, W: w, H: ht}, Alpha: alpha}
	}
	return lead + 9*bar, live, bars
}

// VoiceMark draws the app mark come alive at height h. Listening, its five
// bars keep the mark's silhouette and move with the input levels beside a
// pulsing live dot; waiting, as while the microphone opens or recognition
// finishes, they fold into three dots that a pulse travels through. One
// spring drives the fold, so a switch mid-way turns back from where the
// shapes are. Calls with the same key in the same place share the spring.
func VoiceMark(c *ui.Context, levels []float32, h float32, waiting bool, ink, live ui.Color) ui.Element {
	box := ui.Box(c)
	fold := box.AnimateTo("fold", b2f(waiting), theme.Pop)
	var standing [5]float32
	for i := range standing {
		standing[i] = box.AnimateTo(barKey(i), levelAt(levels, i, len(standing)), ui.Animation{Spring: ui.SpringOptions{Stiffness: 520, Damping: 32}})
	}
	wave := float32(0)
	if fold > 0.004 {
		wave = box.Loop("wave", markWave, ui.Linear)
	}
	pulse := float32(1)
	if fold < 0.996 {
		pulse = 0.45 + 0.55*box.Loop("pulse", 1400*time.Millisecond, ui.Bounce(ui.EaseInOut))
	}
	width, dot, bars := markLayout(h, standing, fold, wave)
	return box.Size(width, h).Draw(func(p *ui.Painter, r ui.Rect) {
		if dot.Alpha > 0 {
			p.Fill(offset(dot.Rect, r), live.Alpha(dot.Alpha*pulse), dot.Rect.W/2)
		}
		for _, s := range bars {
			if s.Alpha > 0 {
				p.Fill(offset(s.Rect, r), ink.Alpha(s.Alpha), min(s.Rect.W, s.Rect.H)/2)
			}
		}
	})
}

func offset(s, r ui.Rect) ui.Rect {
	s.X += r.X
	s.Y += r.Y
	return s
}

// lerp lands exactly on b at t = 1, so merged shapes coincide.
func lerp(a, b, t float32) float32 { return a*(1-t) + b*t }

func clamp01(v float32) float32 { return min(1, max(0, v)) }

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
