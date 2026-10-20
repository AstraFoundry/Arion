package main

import (
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// app is the window's state: the bursts in the air, those still to come,
// and the options of the stage's bursts.
type app struct {
	bursts   []*burst
	emitters []*emitter
	nextID   uint64
	// w and h are the window's size as the view last ran, and dark
	// whether its theme was dark, for demos whose white pieces would not
	// show on a light background.
	w, h float32
	dark bool
	// pressX and pressY are where the stage was last pressed.
	pressX, pressY float32
	// The stage's options, as its sliders set them.
	count, angle, spread, velocity, decay, gravity, drift, scalar float64
	// clock replaces the frame's time in tests.
	clock func() time.Time
}

func newApp() *app {
	s := &app{}
	s.resetOptions()
	return s
}

// resetOptions sets the stage's options as they start: a ring of
// canvas-confetti's default pieces.
func (s *app) resetOptions() {
	d := Defaults()
	s.count, s.angle, s.spread, s.velocity = 150, d.Angle, 360, 30
	s.decay, s.gravity, s.drift, s.scalar = d.Decay, d.Gravity, d.Drift, d.Scalar
}

func (s *app) now(c *ui.Context) time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return c.Now()
}

// maxPieces is how many pieces can be in the air: every piece costs each
// frame that paints, so a button pressed again and again lets the oldest,
// most faded pieces go first.
const maxPieces = 1500

// burst throws a burst of confetti as o says.
func (s *app) burst(o Options) {
	s.bursts = append(s.bursts, newBurst(o, float32(o.X)*s.w, float32(o.Y)*s.h, s.nextID))
	s.nextID++
	for over := s.inAir() - maxPieces; over > 0; {
		b := s.bursts[0]
		if len(b.pieces) > over {
			b.pieces = b.pieces[over:]
			break
		}
		over -= len(b.pieces)
		s.bursts = s.bursts[1:]
	}
}

// inAir returns how many pieces are in the air.
func (s *app) inAir() int {
	n := 0
	for _, b := range s.bursts {
		n += len(b.pieces)
	}
	return n
}

// emitter fires n times, every so often, for demos that throw confetti
// for a while.
type emitter struct {
	next  time.Time
	every time.Duration
	i, n  int
	fire  func(i, n int)
}

// repeat calls fire(i, n) n times, now and every d after.
func (s *app) repeat(now time.Time, d time.Duration, n int, fire func(i, n int)) {
	fire(0, n)
	if n > 1 {
		s.emitters = append(s.emitters, &emitter{next: now.Add(d), every: d, i: 1, n: n, fire: fire})
	}
}

// preset is one of canvas-confetti's demos.
type preset struct {
	name, about string
	run         func(s *app, now time.Time)
}

var presets = []preset{
	{"Basic Cannon", "One burst, up from below", func(s *app, now time.Time) {
		o := Defaults()
		o.Count, o.Spread, o.Y = 100, 70, 0.6
		s.burst(o)
	}},
	{"Random Direction", "Another angle every time", func(s *app, now time.Time) {
		o := Defaults()
		o.Angle = 55 + rand.Float64()*70
		o.Spread = 50 + rand.Float64()*20
		o.Count = 50 + rand.IntN(50)
		o.Y = 0.6
		s.burst(o)
	}},
	{"Realistic Look", "Five bursts, fast to slow", func(s *app, now time.Time) {
		const total = 200
		fire := func(share float64, change func(o *Options)) {
			o := Defaults()
			o.Count, o.Y = int(total*share), 0.7
			change(&o)
			s.burst(o)
		}
		fire(0.25, func(o *Options) { o.Spread, o.StartVelocity = 26, 55 })
		fire(0.2, func(o *Options) { o.Spread = 60 })
		fire(0.35, func(o *Options) { o.Spread, o.Decay, o.Scalar = 100, 0.91, 0.8 })
		fire(0.1, func(o *Options) { o.Spread, o.StartVelocity, o.Decay, o.Scalar = 120, 25, 0.92, 1.2 })
		fire(0.1, func(o *Options) { o.Spread, o.StartVelocity = 120, 45 })
	}},
	{"Fireworks", "Bursts all around for 5 s", func(s *app, now time.Time) {
		s.repeat(now, 250*time.Millisecond, 20, func(i, n int) {
			for _, x := range [][2]float64{{0.1, 0.3}, {0.7, 0.9}} {
				o := Defaults()
				o.Count = 50 * (n - i) / n
				o.StartVelocity, o.Spread, o.Ticks = 30, 360, 60
				o.X = x[0] + rand.Float64()*(x[1]-x[0])
				o.Y = rand.Float64() - 0.2
				s.burst(o)
			}
		})
	}},
	{"Stars", "Golden stars floating out", func(s *app, now time.Time) {
		s.repeat(now, 100*time.Millisecond, 3, func(int, int) {
			o := Defaults()
			o.Spread, o.Ticks, o.Gravity, o.Decay, o.StartVelocity = 360, 50, 0, 0.94, 30
			o.Colors = []ui.Color{ui.Hex("#ffe400"), ui.Hex("#ffbd00"), ui.Hex("#e89400"), ui.Hex("#ffca6c"), ui.Hex("#fdffb8")}
			o.Count, o.Scalar, o.Shapes = 40, 1.2, []Shape{Star}
			s.burst(o)
			o.Count, o.Scalar, o.Shapes = 10, 0.75, []Shape{Circle}
			s.burst(o)
		})
	}},
	{"Snow", "Flakes drifting down for 8 s", func(s *app, now time.Time) {
		const n = 270
		skew := 1.0
		s.repeat(now, 30*time.Millisecond, n, func(i, n int) {
			left := float64(n-i) / float64(n)
			skew = max(0.8, skew-0.001)
			o := Defaults()
			o.Count, o.StartVelocity, o.Shapes = 1, 0, []Shape{Circle}
			o.Ticks = int(max(200, 500*left))
			o.X, o.Y = rand.Float64(), rand.Float64()*skew-0.2
			o.Colors = []ui.Color{ui.Hex("#ffffff"), ui.Hex("#dbeafe"), ui.Hex("#bfdbfe")}
			if !s.dark {
				o.Colors = []ui.Color{ui.Hex("#93c5fd"), ui.Hex("#bfdbfe"), ui.Hex("#cbd5e1")}
			}
			o.Gravity = 0.4 + rand.Float64()*0.2
			o.Scalar = 0.4 + rand.Float64()*0.6
			o.Drift = -0.4 + rand.Float64()*0.8
			s.burst(o)
		})
	}},
	{"School Pride", "Cannons from both sides", func(s *app, now time.Time) {
		s.repeat(now, 25*time.Millisecond, 160, func(int, int) {
			for _, side := range []struct{ angle, x float64 }{{60, 0}, {120, 1}} {
				o := Defaults()
				o.Count, o.Angle, o.Spread, o.X = 2, side.angle, 55, side.x
				o.Colors = []ui.Color{ui.Hex("#bb0000"), ui.Hex("#ffffff")}
				if !s.dark {
					o.Colors[1] = ui.Hex("#1e293b")
				}
				s.burst(o)
			}
		})
	}},
	{"Emoji", "Unicorns and rainbows", func(s *app, now time.Time) {
		s.repeat(now, 100*time.Millisecond, 3, func(int, int) {
			o := Defaults()
			o.Spread, o.Ticks, o.Gravity, o.Decay, o.StartVelocity = 360, 60, 0, 0.96, 20
			o.Shapes, o.Scalar = []Shape{Text("🦄"), Text("🌈")}, 2
			o.Count = 30
			s.burst(o)
			o.Count, o.Flat = 5, true
			s.burst(o)
			o.Count, o.Flat, o.Scalar, o.Shapes = 15, false, 1, []Shape{Circle}
			s.burst(o)
		})
	}},
	{"Hearts", "A shape drawn as a path", func(s *app, now time.Time) {
		s.repeat(now, 100*time.Millisecond, 3, func(int, int) {
			o := Defaults()
			o.Spread, o.Ticks, o.Gravity, o.Decay, o.StartVelocity = 360, 70, 0.3, 0.95, 25
			o.Count, o.Scalar, o.Shapes, o.Flat = 25, 1.5, []Shape{Heart}, true
			o.Colors = []ui.Color{ui.Hex("#ff4d6d"), ui.Hex("#ff758f"), ui.Hex("#c9184a"), ui.Hex("#ffb3c1")}
			s.burst(o)
		})
	}},
}

func (s *app) view(c *ui.Context) {
	t := c.Theme()
	now := s.now(c)
	s.w, s.h = c.Size()
	s.dark = t.Dark
	// While the desktop asks for less motion, nothing flies: the pieces
	// would only show where they land, fading.
	still := c.Preferences().ReduceMotion

	// The demos that throw confetti for a while.
	emitters := s.emitters[:0]
	for _, e := range s.emitters {
		for e.i < e.n && !now.Before(e.next) {
			if !still {
				e.fire(e.i, e.n)
			}
			e.i++
			e.next = e.next.Add(e.every)
		}
		if e.i < e.n {
			emitters = append(emitters, e)
		}
	}
	s.emitters = emitters

	ui.Column(c).Fill().Padding(24).Gap(20).Children(func() {
		ui.Row(c).AlignItems(ui.End).Gap(12).Children(func() {
			ui.Column(c).Gap(4).Grow(1).Children(func() {
				ui.Text(c, "Confetti").FontSize(28).Bold()
				ui.Text(c, "canvas-confetti's demos in native UI, every piece moved by poses that only paint.").TextColor(t.TextMuted)
			})
			ui.Textf(c, "%d in the air", s.inAir()).FontSize(12).TextColor(t.TextMuted)
		})
		if still {
			ui.Text(c, "Confetti rests while the desktop asks for less motion.").TextColor(t.Warning)
		}
		ui.Row(c).Grow(1).Gap(16).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).Grow(1).Gap(16).Children(func() {
				s.presets(c, now, still)
				stage := ui.Column(c).Key("stage").Grow(1).Center().Gap(6).Radius(12).
					Border(1, t.Border).BorderStyle(ui.BorderDashed).Cursor(ui.CursorPointer)
				stage.Children(func() {
					ui.Text(c, "Click anywhere here").Bold()
					ui.Text(c, "to throw your burst from the pointer").FontSize(12).TextColor(t.TextMuted)
				})
				// Where the stage was pressed, noted as the press comes and
				// left to go on to Clicked: PointerPosition would build the
				// view, every piece in the air with it, as the pointer moves.
				stage.HandleInput(func(ev ui.InputEvent) bool {
					if ev.Kind == ui.InputPointerDown {
						s.pressX, s.pressY = ev.X, ev.Y
					}
					return false
				})
				if stage.Clicked() && !still {
					s.burst(s.custom(stage.LocalToWindow(s.pressX, s.pressY)))
				}
			})
			s.controls(c)
		})
	})

	// The pieces fly over everything, under the window's own (0, 0), and
	// let the pointer through. Bursts that faded away go.
	bursts := s.bursts[:0]
	for _, b := range s.bursts {
		if b.launched.IsZero() || now.Sub(b.launched) < b.life {
			bursts = append(bursts, b)
		}
	}
	clear(s.bursts[len(bursts):])
	s.bursts = bursts
	ui.Box(c).Key("confetti").Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Children(func() {
		for _, b := range s.bursts {
			b.build(c, now)
		}
	})
	// The view runs again once a burst faded, to let it go.
	var gone time.Duration
	for _, b := range s.bursts {
		if b.launched.IsZero() {
			continue
		}
		if d := b.life - now.Sub(b.launched); gone == 0 || d < gone {
			gone = d
		}
	}
	if gone > 0 {
		c.After(gone)
	}
	// And when a demo throws more.
	for _, e := range s.emitters {
		c.After(e.next.Sub(now))
	}
}

// presets builds a tile for each of canvas-confetti's demos, which lifts
// under the pointer and sinks as it presses.
func (s *app) presets(c *ui.Context, now time.Time, still bool) {
	t := c.Theme()
	ui.Grid(c).Columns(3).Gap(12).Children(func() {
		for i, p := range presets {
			tile := ui.ButtonBase(c.Key(i)).Label(p.name).Column().AlignItems(ui.Start).Gap(4).Padding(12, 14).
				Radius(10).Border(1, t.Border).Cursor(ui.CursorPointer)
			bg, lift := t.Surface, ui.Pose{}
			if tile.Pressed() {
				bg, lift = t.SurfacePressed, ui.Pose{Scale: 0.97}
			} else if tile.Hovered() {
				bg, lift = t.SurfaceHover, ui.Pose{Y: -2}
			}
			// The lift is a pose, which moves in frames that only paint; a
			// color that animated (AnimateColor) would build the view, and
			// every piece in the air, in each frame of its change.
			tile.Background(bg).AnimatePose(lift, ui.Snappy).
				Children(func() {
					ui.Text(c, p.name).Bold()
					ui.Text(c, p.about).FontSize(12).TextColor(t.TextMuted)
				})
			if tile.Clicked() && !still {
				p.run(s, now)
			}
		}
	})
}

// controls builds the sliders of the stage's bursts.
func (s *app) controls(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(280).Gap(10).Padding(16).Radius(12).Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Text(c, "Your burst").Bold()
		slider := func(label string, v *float64, lo, hi float64, format string) {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).FontSize(12).Width(62)
				ui.Slider(c.Key(label), v, lo, hi).Label(label).Grow(1)
				ui.Textf(c, format, *v).FontSize(12).Width(40).TextAlign(ui.End).TextColor(t.TextMuted)
			})
		}
		slider("Count", &s.count, 1, 400, "%.0f")
		slider("Angle", &s.angle, 0, 360, "%.0f°")
		slider("Spread", &s.spread, 0, 360, "%.0f°")
		slider("Velocity", &s.velocity, 0, 90, "%.0f")
		slider("Decay", &s.decay, 0.8, 0.99, "%.2f")
		slider("Gravity", &s.gravity, -1, 3, "%.1f")
		slider("Drift", &s.drift, -2, 2, "%.1f")
		slider("Size", &s.scalar, 0.4, 3, "%.1f×")
		ui.Spacer(c)
		if ui.Button(c, "Reset").Clicked() {
			s.resetOptions()
		}
	})
}

// custom returns the stage's options for a burst from (x, y) in window
// DIPs.
func (s *app) custom(x, y float32) Options {
	o := Defaults()
	o.Count = int(s.count)
	o.Angle, o.Spread, o.StartVelocity, o.Decay = s.angle, s.spread, s.velocity, s.decay
	o.Gravity, o.Drift, o.Scalar = s.gravity, s.drift, s.scalar
	o.X, o.Y = float64(x/s.w), float64(y/s.h)
	return o
}
