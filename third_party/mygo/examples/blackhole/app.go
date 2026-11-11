package main

import (
	_ "embed"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

var (
	//go:embed blackhole.kage
	holeSource string
	//go:embed bloom.kage
	bloomSource string
	//go:embed post.kage
	postSource string

	hole  = shader.MustCompile(holeSource)
	bloom = shader.MustCompile(bloomSource)
	post  = shader.MustCompile(postSource)
)

const (
	// The camera's distances from the hole, in its radii: nearest, at
	// rest and farthest (within the shader's escape, 42); and its pitch's
	// bounds, short of the poles.
	near, rest, far = 8, 22, 36
	maxPitch        = 1.4
	// orbitSpeed is how fast the camera circles on its own, in radians a
	// second, and settle how quickly a throw slows to it, a second.
	orbitSpeed, settle = 0.025, 1.2
	// The scene's resolution, a fraction of the window's pixels across,
	// from minQuality up: it drops by a step while frames come later than
	// slowFrame, and rises once they come sooner than fastFrame for a
	// while. The glow renders at glowScale of it.
	minQuality, qualityStep = 0.5, 0.125
	slowFrame, fastFrame    = 1.0 / 50, 1.0 / 90
	glowScale               = 4
)

var (
	white = ui.RGBA(255, 255, 255, 0.92)
	muted = ui.RGBA(255, 255, 255, 0.5)
)

// app is the window's state.
type app struct {
	cam camera
	// time is the disk's clock, which stands still while the desktop asks
	// for less motion; last is the frame's time.
	time float64
	last time.Time
	// The pointer's drag: whether the primary button is down, where it
	// last was and when; touched tells that the user moved the camera,
	// and the hint fades.
	down         bool
	lastX, lastY float32
	lastMove     time.Time
	touched      bool
	hint         float32
	// The passes' canvases: the scene, a fraction (quality) of the
	// window's pixels across, and its glow; frame is how long frames
	// take, smoothed, and slow and fast how long they took longer or
	// shorter than they should.
	scene, glow       *ui.Canvas
	quality           float64
	frame, slow, fast float64

	values                                           *shader.Values
	timeVar, eyeVar, rightVar, upVar, fwdVar, focVar shader.Uniform
	postValues                                       *shader.Values
	postTime                                         shader.Uniform
}

func newApp() *app {
	return &app{
		cam:     camera{yaw: 0.6, pitch: 0.1, dist: rest, spinYaw: orbitSpeed},
		hint:    1,
		quality: 1, frame: 1.0 / 60,
		values:  hole.NewValues(),
		timeVar: hole.Uniform("Time"), eyeVar: hole.Uniform("Eye"), rightVar: hole.Uniform("Right"),
		upVar: hole.Uniform("Up"), fwdVar: hole.Uniform("Forward"), focVar: hole.Uniform("Focal"),
		postValues: post.NewValues(), postTime: post.Uniform("Time"),
	}
}

func (a *app) view(c *ui.Context) {
	reduce := c.Preferences().ReduceMotion
	root := ui.Row(c).Fill().AlignItems(ui.End).Padding(0, 28, 26, 30).Background(ui.RGB(0, 0, 0))
	if a.down {
		root.Cursor(ui.CursorGrabbing)
	} else {
		root.Cursor(ui.CursorGrab)
	}
	root.Draw(func(p *ui.Painter, r ui.Rect) { a.draw(p, r, reduce) })
	root.HandleInput(a.input)
	root.Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, "Gargantua").FontSize(22).FontWeight(600).TextColor(white)
			ui.Text(c, "A black hole, its light traced through gravity by shaders written in Go").FontSize(12).TextColor(muted)
		})
		ui.Spacer(c).Grow(1)
		ui.Text(c, "Drag to orbit · Scroll to zoom").FontSize(12).TextColor(muted).Opacity(a.hint)
	})
}

// draw moves the scene on to the frame's time and draws it in its three
// passes.
func (a *app) draw(p *ui.Painter, r ui.Rect, reduce bool) {
	now := p.Now()
	dt := 0.0
	if !a.last.IsZero() {
		dt = min(now.Sub(a.last).Seconds(), 0.1)
	}
	a.last = now
	if !reduce {
		a.time += dt
	}
	a.step(dt, reduce)

	if !p.Accelerated() {
		// Seconds a frame on the CPU: the scene needs a GPU.
		p.Text(r.X+30, r.Y+60, "The black hole draws on a GPU.", 15, muted)
		return
	}
	scale := float64(p.Scale())
	w := max(1, int(math.Round(float64(r.W)*scale*a.quality)))
	h := max(1, int(math.Round(float64(r.H)*scale*a.quality)))
	a.scene = sized(a.scene, w, h)
	a.glow = sized(a.glow, max(1, w/glowScale), max(1, h/glowScale))

	eye, right, up, forward := a.cam.axes()
	a.values.Float(a.timeVar, float32(a.time)).Float(a.focVar, focal)
	for _, v := range [...]struct {
		u shader.Uniform
		d vec3
	}{{a.eyeVar, eye}, {a.rightVar, right}, {a.upVar, up}, {a.fwdVar, forward}} {
		x := v.d.f32()
		a.values.Vec3(v.u, x[0], x[1], x[2])
	}
	a.scene.Draw(p, func(cp *ui.Painter) {
		hole.Draw(cp, ui.Rect{W: float32(w), H: float32(h)}, shader.Options{Values: a.values})
	})
	gw, gh := a.glow.Size()
	a.glow.Draw(p, func(cp *ui.Painter) {
		bloom.Draw(cp, ui.Rect{W: float32(gw), H: float32(gh)}, shader.Options{Images: [4]*ui.Bitmap{a.scene.Bitmap()}})
	})
	post.Draw(p, r, shader.Options{
		Values: a.postValues.Float(a.postTime, float32(a.time)),
		Images: [4]*ui.Bitmap{a.scene.Bitmap(), a.glow.Bitmap()},
	})
	if !reduce || a.down || a.cam.spinning() || (a.touched && a.hint > 0) {
		p.AnimationFrame()
	}
}

// sized returns cv, or a canvas of w×h pixels where cv is not one.
func sized(cv *ui.Canvas, w, h int) *ui.Canvas {
	if cv != nil {
		if cw, ch := cv.Size(); cw == w && ch == h {
			return cv
		}
	}
	return ui.NewCanvas(w, h)
}

// step moves the camera on by dt seconds, fades the hint once the user
// moved it, and fits the resolution to how fast frames come.
func (a *app) step(dt float64, reduce bool) {
	cam := &a.cam
	if !a.down {
		goal := orbitSpeed
		if reduce {
			goal = 0
		}
		k := 1 - math.Exp(-dt*settle)
		cam.spinYaw += (goal - cam.spinYaw) * k
		cam.spinPitch -= cam.spinPitch * min(1, 2*k)
		cam.yaw += cam.spinYaw * dt
		cam.pitch = clamp(cam.pitch+cam.spinPitch*dt, -maxPitch, maxPitch)
	}
	if a.touched {
		a.hint = max(0, a.hint-float32(dt)/0.6)
	}

	if dt <= 0 {
		return
	}
	a.frame += (dt - a.frame) * 0.1
	if a.frame > slowFrame {
		a.slow += dt
	} else {
		a.slow = 0
	}
	if a.frame < fastFrame {
		a.fast += dt
	} else {
		a.fast = 0
	}
	switch {
	case a.slow > 0.5 && a.quality > minQuality:
		a.quality = max(minQuality, a.quality-qualityStep)
		a.slow, a.frame = 0, fastFrame
	case a.fast > 3 && a.quality < 1:
		a.quality = min(1, a.quality+qualityStep)
		a.fast = 0
	}
}

// spinning reports whether the camera still turns on its own.
func (c *camera) spinning() bool {
	return math.Abs(c.spinYaw) > 1e-3 || math.Abs(c.spinPitch) > 1e-3
}

// input takes the pointer's drags and scrolls: a drag turns the camera
// around the hole, and a throw keeps it turning, slowing down to its own
// orbit.
func (a *app) input(ev ui.InputEvent) bool {
	cam := &a.cam
	now := time.Now()
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		a.down, a.touched = true, true
		a.lastX, a.lastY, a.lastMove = ev.X, ev.Y, now
		cam.spinYaw, cam.spinPitch = 0, 0
		return true
	case ui.InputPointerMove:
		if !a.down {
			return false
		}
		dt := max(now.Sub(a.lastMove).Seconds(), 1.0/240)
		dyaw, dpitch := -float64(ev.X-a.lastX)*0.005, float64(ev.Y-a.lastY)*0.004
		cam.yaw += dyaw
		cam.pitch = clamp(cam.pitch+dpitch, -maxPitch, maxPitch)
		// The throw: the drag's speed, smoothed over its last moves.
		k := min(1, dt/0.06)
		cam.spinYaw += (dyaw/dt - cam.spinYaw) * k
		cam.spinPitch += (dpitch/dt - cam.spinPitch) * k
		a.lastX, a.lastY, a.lastMove = ev.X, ev.Y, now
		return true
	case ui.InputPointerUp:
		if !a.down {
			return false
		}
		a.down = false
		if now.Sub(a.lastMove) > 80*time.Millisecond {
			// Held still before letting go: no throw.
			cam.spinYaw, cam.spinPitch = 0, 0
		}
		return true
	case ui.InputScroll:
		a.touched = true
		cam.dist = clamp(cam.dist*math.Exp(float64(ev.DY)*0.003), near, far)
		return true
	}
	return false
}

func clamp(x, lo, hi float64) float64 { return max(lo, min(hi, x)) }
