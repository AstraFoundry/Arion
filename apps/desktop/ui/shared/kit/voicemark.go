package kit

import (
	"math"
	"time"

	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// MarkMode is what the voice mark is doing.
type MarkMode int

const (
	// MarkListening shows the mark's bars as the input meter beside the
	// pulsing live dot.
	MarkListening MarkMode = iota
	// MarkWaiting folds the bars into three dots that a pulse travels
	// through, while the microphone opens or recognition finishes.
	MarkWaiting
	// MarkOrbiting turns the three dots around a circle, one eased turn and
	// a pause at a time, while the rewrite organizes the text.
	MarkOrbiting
	// MarkResting stands the bars as the full mark once the text is decided.
	MarkResting
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

// markSpin is one orbit cycle: a turn over markTurn of it, then a pause.
// Each dot starts its turn markChase after the one behind it, so the dots
// close up while they accelerate and spread again as they settle.
const (
	markSpin  = 1600 * time.Millisecond
	markTurn  = 0.68
	markChase = 0.045
)

// markEase is the orbit's turn: a slow start, a fast middle and a long
// settle, so each pause reads as the dots arriving rather than stopping.
var markEase = ui.CubicBezier(0.7, 0, 0.25, 1)

// markPose is the mark's motion at one instant. Live, Fold and Orbit are
// spring amounts from 0 to 1; Wave and Spin come from loops.
type markPose struct {
	// Levels are the bars' standing levels, from 0 to 1.
	Levels [5]float32
	// Live is the live dot's presence.
	Live float32
	// Fold is how far the bars have folded into dots.
	Fold float32
	// Orbit is how far the row of dots has moved onto its circle.
	Orbit float32
	// Wave is the waiting pulse's phase.
	Wave float32
	// Spin is each dot's progress through its current turn.
	Spin [3]float32
}

// markShape is one filled rounded shape of the mark.
type markShape struct {
	Rect  ui.Rect
	Alpha float32
}

// markLayout places the mark at height h in pose: the live dot, then the
// five bars standing at their levels, each folded into its dot as far as
// Fold, the dots resting in a row or turning on a circle as far as Orbit.
//
// The waiting pulse is a Gaussian that travels across the dots, as Morph
// Bot's thinking dots: the dot it passes lifts, grows and brightens while
// the others rest half dimmed. On the circle the dots sit 120 degrees apart
// with the center dot at the top, so each moves only a short way from the
// row. Bars and dots are one shape at every pose, a rounded rect whose
// width, height, position and alpha interpolate, so every change of mode
// is one continuous motion and reverses from wherever it is.
func markLayout(h float32, pose markPose) (width float32, live markShape, bars [5]markShape) {
	bar := 0.15 * h
	dot := bar * 5 / 3
	open := clamp01(pose.Live)
	orbit := clamp01(pose.Orbit)
	// The live dot leads the bars while the microphone is open and shrinks
	// away when it closes, taking its gap with it.
	lead := 0.9 * h * open
	radius := 0.2 * h * open
	live = markShape{Rect: ui.Rect{X: 0, Y: h/2 - radius, W: 2 * radius, H: 2 * radius}, Alpha: open}
	center := lead + 4.5*bar
	ring := 0.3 * h
	for i := range bars {
		distance := float32(absInt(i - 2))
		p := clamp01((pose.Fold - markStagger*distance) / (1 - 2*markStagger))
		j := markDots[i]
		d := float32(math.Abs(float64(pose.Wave - float32(j)/3)))
		d = min(d, 1-d)
		// On the circle the dots stay whole: the turn is the motion.
		pulse := lerp(float32(math.Exp(-float64(d*d)/0.045)), 1, orbit)
		size := dot * (0.84 + 0.22*pulse)
		angle := 2 * math.Pi * (float64(j-1)/3 - 0.25 + float64(pose.Spin[j]))
		dx := lerp(center+float32(j-1)*3*bar, center+ring*float32(math.Cos(angle)), orbit)
		dy := lerp(h/2-0.2*h*pulse, h/2+ring*float32(math.Sin(angle)), orbit)
		standing := bar + (h*markBars[i]-bar)*(markRest+(1-markRest)*clamp01(pose.Levels[i]))
		w := lerp(bar, size, p)
		ht := lerp(standing, size, p)
		x := lerp(lead+bar/2+float32(i)*2*bar, dx, p)
		y := lerp(h/2, dy, p)
		alpha := lerp(1, 0.5+0.5*pulse, p)
		if i%2 == 1 {
			// Odd bars merge into a neighbor's dot and leave their alpha behind.
			alpha *= 1 - p
		}
		bars[i] = markShape{Rect: ui.Rect{X: x - w/2, Y: y - ht/2, W: w, H: ht}, Alpha: alpha}
	}
	return lead + 9*bar, live, bars
}

// markSpinAt is each dot's progress through its turn at phase u of the
// orbit cycle: all of them rest at a whole turn through the pause.
func markSpinAt(u float32) (spin [3]float32) {
	for j := range spin {
		spin[j] = markEase(clamp01((u - markChase*float32(j)) / markTurn))
	}
	return spin
}

// VoiceMark draws the app mark come alive at height h in mode. Each mode
// only sets the targets of springs on one set of shapes, so a change of
// mode, even mid-way, is one continuous motion. Calls with the same key in
// the same place share the springs.
func VoiceMark(c *ui.Context, levels []float32, h float32, mode MarkMode, ink, live ui.Color) ui.Element {
	box := ui.Box(c)
	var pose markPose
	pose.Live = box.AnimateTo("live", b2f(mode == MarkListening), theme.Pop)
	pose.Fold = box.AnimateTo("fold", b2f(mode == MarkWaiting || mode == MarkOrbiting), theme.Pop)
	pose.Orbit = box.AnimateTo("orbit", b2f(mode == MarkOrbiting), theme.Pop)
	for i := range pose.Levels {
		target := levelAt(levels, i, len(pose.Levels))
		if mode == MarkResting {
			target = 1
		}
		pose.Levels[i] = box.AnimateTo(barKey(i), target, ui.Animation{Spring: ui.SpringOptions{Stiffness: 520, Damping: 32}})
	}
	if pose.Fold > 0.004 && pose.Orbit < 0.996 {
		pose.Wave = box.Loop("wave", markWave, ui.Linear)
	}
	if pose.Orbit > 0.004 {
		pose.Spin = markSpinAt(box.Loop("spin", markSpin, ui.Linear))
	}
	pulse := float32(1)
	if pose.Live > 0.004 {
		pulse = 0.45 + 0.55*box.Loop("pulse", 1400*time.Millisecond, ui.Bounce(ui.EaseInOut))
	}
	width, dot, bars := markLayout(h, pose)
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
