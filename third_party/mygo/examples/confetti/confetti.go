package main

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// Options say how a burst of confetti flies, as canvas-confetti's: speeds
// and distances are in DIPs per tick, a sixtieth of a second, as it moves
// its pieces once per animation frame.
type Options struct {
	// Count is how many pieces the burst throws.
	Count int
	// Angle is the direction it throws them, in degrees counterclockwise
	// from the right (90 is up), and Spread how far they stray from it,
	// in degrees.
	Angle, Spread float64
	// StartVelocity is how fast a piece starts, from half of it to one and
	// a half times it; Decay is what its speed keeps every tick.
	StartVelocity, Decay float64
	// Gravity pulls the pieces down 3 DIPs every tick, times Gravity, and
	// Drift moves them sideways.
	Gravity, Drift float64
	// Ticks is how long the pieces last, fading as they go.
	Ticks int
	// X and Y are where the burst starts, as fractions of the window's
	// width and height.
	X, Y float64
	// Colors and Shapes are chosen from at random for each piece.
	Colors []ui.Color
	Shapes []Shape
	// Scalar scales the pieces.
	Scalar float64
	// Flat keeps the pieces facing the viewer instead of tumbling.
	Flat bool
}

// Shape is how a piece of confetti looks: a Square, a Circle, a Star, a
// Heart, or Text, an emoji.
type Shape struct {
	kind shapeKind
	text string
}

type shapeKind uint8

const (
	square shapeKind = iota
	circle
	star
	heart
	text
)

var (
	Square = Shape{kind: square}
	Circle = Shape{kind: circle}
	Star   = Shape{kind: star}
	Heart  = Shape{kind: heart}
)

// Text is a piece of confetti showing s, as an emoji.
func Text(s string) Shape { return Shape{kind: text, text: s} }

// defaultColors are canvas-confetti's.
var defaultColors = []ui.Color{
	ui.Hex("#26ccff"), ui.Hex("#a25afd"), ui.Hex("#ff5e7e"), ui.Hex("#88ff5a"),
	ui.Hex("#fcff42"), ui.Hex("#ffa62d"), ui.Hex("#ff36ff"),
}

// Defaults returns canvas-confetti's options: 50 squares and circles
// thrown up from the middle of the window.
func Defaults() Options {
	return Options{
		Count:         50,
		Angle:         90,
		Spread:        45,
		StartVelocity: 45,
		Decay:         0.9,
		Gravity:       1,
		Ticks:         200,
		X:             0.5,
		Y:             0.5,
		Colors:        defaultColors,
		Shapes:        []Shape{Square, Circle},
		Scalar:        1,
	}
}

// tick is how long canvas-confetti moves its pieces once.
const tick = time.Second / 60

// burst is a burst of confetti in the air. canvas-confetti adds up three
// motions for each piece: a burst that slows down as its speed decays, a
// fall at a steady speed under gravity and drift, which fades it, and a
// spin, seen in a tilted plane, which makes it tumble. Here each is a pose:
// the fall, the same for every piece of a burst, moves the whole burst,
// in which each piece bursts out, tilted, and spins.
type burst struct {
	id     uint64
	x, y   float32 // where it starts, in window DIPs
	life   time.Duration
	fall   ui.Pose
	ease   ui.Easing // of the pieces' burst
	pieces []piece
	// launched is when it headed for its poses, zero while it still shows
	// where it starts.
	launched time.Time
}

// piece is a piece of confetti in a burst.
type piece struct {
	id    int // its place in the burst, which keys it
	w, h  float32
	color ui.Color
	shape Shape
	// to is where the burst carries it, and spin how many degrees it turns
	// in its plane, which tilt turns away from the viewer around an axis
	// turned by turn degrees.
	to         ui.Pose
	spin       float32
	tilt, turn float32
}

// newBurst makes a burst starting at (x, y) in window DIPs.
func newBurst(o Options, x, y float32, id uint64) *burst {
	shapes, colors := o.Shapes, o.Colors
	if len(shapes) == 0 {
		shapes = []Shape{Square, Circle}
	}
	if len(colors) == 0 {
		colors = defaultColors
	}
	ticks := float64(max(o.Ticks, 1))
	b := &burst{
		id:     id,
		x:      x,
		y:      y,
		life:   time.Duration(ticks) * tick,
		fall:   ui.Pose{X: float32(o.Drift * ticks), Y: float32(3 * o.Gravity * ticks), Fade: 1},
		ease:   decayEase(o.Decay, ticks),
		pieces: make([]piece, o.Count),
	}
	// What the speed adds up to over the ticks, per DIP per tick of the
	// starting speed.
	reach := ticks
	if o.Decay != 1 {
		reach = (1 - math.Pow(o.Decay, ticks)) / (1 - o.Decay)
	}
	angle, spread := o.Angle*math.Pi/180, o.Spread*math.Pi/180
	for i := range b.pieces {
		dir := -angle + spread/2 - rand.Float64()*spread
		speed := o.StartVelocity*0.5 + rand.Float64()*o.StartVelocity
		shape := shapes[rand.IntN(len(shapes))]
		p := piece{
			id:    i,
			color: colors[rand.IntN(len(colors))],
			shape: shape,
			to:    ui.Pose{X: float32(math.Cos(dir) * speed * reach), Y: float32(math.Sin(dir) * speed * reach)},
		}
		s := float32(o.Scalar)
		switch shape.kind {
		case square:
			p.w, p.h = 10*s, 10*s*(0.5+0.4*rand.Float32())
		case circle:
			p.w, p.h = 8*s, 8*s
		case star:
			p.w, p.h = 16*s, 16*s
		case heart:
			p.w, p.h = 14*s, 13*s
		case text:
			p.w, p.h = 12*s, 12*s
		}
		if !o.Flat {
			p.tilt = 45 + 40*rand.Float32()
			p.turn = 360 * rand.Float32()
			if shape.kind != circle {
				// About a turn a second, either way, as canvas-confetti
				// turns its pieces' tilt 0.1 radians a tick. A circle looks
				// the same however it spins.
				turns := (0.6 + 0.8*rand.Float64()) * b.life.Seconds()
				if rand.IntN(2) == 0 {
					turns = -turns
				}
				p.spin = float32(turns * 360)
			}
		}
		b.pieces[i] = p
	}
	return b
}

// decayEase is how far a burst has gone, as a fraction of how far it goes,
// t of the way through ticks ticks, its speed keeping decay of itself
// every tick.
func decayEase(decay, ticks float64) ui.Easing {
	if decay == 1 {
		return ui.Linear
	}
	// decay^(t ticks), as an exponential, which is cheaper every frame.
	k := math.Log(decay) * ticks
	total := 1 - math.Exp(k)
	return func(t float32) float32 {
		return float32((1 - math.Exp(k*float64(t))) / total)
	}
}

// build builds the burst: a box that falls and fades, and in it a box for
// each piece, tilted, which bursts out, with what it shows spinning in it,
// or showing on it when it does not spin, as a circle. Every box costs
// each frame that paints it, but the poses move them in frames that only
// paint. A pose starts at the first one it gets, so a new burst shows
// where it starts until a frame showed it there, and then heads off. (A
// frame can build the view more than once, as after a click, and only the
// last build is posed.)
func (b *burst) build(c *ui.Context, now time.Time) {
	group := ui.Box(c.Key(b.id)).Absolute().Left(b.x).Top(b.y).Size(1, 1).PassThrough()
	if b.launched.IsZero() && group.Bounds().W > 0 {
		b.launched = now
	}
	launched := !b.launched.IsZero()
	if !launched {
		c.AnimationFrame()
	}
	linear := ui.Animation{Duration: b.life, Ease: ui.Linear}
	var fall ui.Pose
	if launched {
		fall = b.fall
	}
	group.AnimatePose(fall, linear).Children(func() {
		for i := range b.pieces {
			p := &b.pieces[i]
			var to, spin ui.Pose
			if launched {
				to, spin = p.to, ui.Pose{Rotate: p.spin}
			}
			var e ui.Element
			if p.spin == 0 {
				e = p.shapeBox(c.Key(p.id))
			} else {
				e = ui.Box(c.Key(p.id))
			}
			e.Absolute().Left(-p.w/2).Top(-p.h/2).Size(p.w, p.h).PassThrough()
			if p.tilt != 0 {
				// The pose spins the piece in its plane, which this turns
				// away: seen edge on as it spins, it tumbles. The burst
				// moves it after.
				e.Transform(ui.RotationX(p.tilt).Then(ui.Rotation(p.turn)))
			}
			e.AnimatePose(to, ui.Animation{Duration: b.life, Ease: b.ease})
			if p.spin != 0 {
				e.Children(func() {
					p.shapeBox(c).Size(p.w, p.h).PassThrough().AnimatePose(spin, linear)
				})
			}
		}
	})
}

// shapeBox builds what the piece looks like.
func (p *piece) shapeBox(c *ui.Context) ui.Element {
	switch p.shape.kind {
	case circle:
		return ui.Box(c).Radius(p.w / 2).Background(p.color)
	case star:
		return ui.Box(c).Draw(func(pt *ui.Painter, r ui.Rect) { pt.FillPath(starPath(r), p.color) })
	case heart:
		return ui.Box(c).Draw(func(pt *ui.Painter, r ui.Rect) { pt.FillPath(heartPath(r), p.color) })
	case text:
		return ui.Text(c, p.shape.text).FontSize(p.h).LineHeight(1).Center()
	}
	return ui.Box(c).Background(p.color)
}

// starPath is a five-pointed star filling r.
func starPath(r ui.Rect) *ui.Path {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	outer := r.W / 2
	inner := outer * 0.45
	path := &ui.Path{}
	for i := range 10 {
		rad := outer
		if i%2 == 1 {
			rad = inner
		}
		a := -math.Pi/2 + float64(i)*math.Pi/5
		x, y := cx+rad*float32(math.Cos(a)), cy+rad*float32(math.Sin(a))
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	return path.Close()
}

// heartPath is a heart filling r.
func heartPath(r ui.Rect) *ui.Path {
	x, y, w, h := r.X, r.Y, r.W, r.H
	return (&ui.Path{}).
		MoveTo(x+w/2, y+h).
		CubeTo(x-w*0.15, y+h*0.55, x+w*0.05, y-h*0.1, x+w/2, y+h*0.22).
		CubeTo(x+w*0.95, y-h*0.1, x+w*1.15, y+h*0.55, x+w/2, y+h).
		Close()
}
