package usage

import (
	"math"

	"arion.local/desktop/ui/shared/iso"
	"github.com/egoist/mygo/ui"
)

// microphone draws the desk microphone: a weighted round base on a rubber
// ring with a switch, a stem through a knurled collar, and a capsule with a
// grille, a band that lights while it listens and a mesh on top.
func microphone(k *iso.Canvas, cam iso.Camera, ink iso.Ink, at trayPose, width float32,
	solid func(iso.Prism, iso.Palette, func()), line func([]iso.Vec, ui.Color), dot func(iso.Vec, float32, ui.Color),
	round func(iso.Vec, float32, float32) iso.Prism) {
	body := ink.Body
	foot := iso.Palette{Top: body.Shadow, Shadow: body.Shadow, Lit: body.Shadow, Edge: ink.Edge, Crease: ink.Edge}
	m := micAt
	// toward is the floor direction to the viewer, for parts on the
	// near side.
	a := float64(trayAzimuth) * math.Pi / 180
	toward := iso.Vec{float32(math.Cos(a)), float32(math.Sin(a)), 0}
	solid(round(iso.Vec{m[0], m[1], -1.5}, micBaseR-1, 1.5), foot, nil)
	solid(round(iso.Vec{m[0], m[1], 0}, micBaseR, micBaseH), body, func() {
		for i := 0; i < 3; i++ {
			angle := a + math.Pi/2 + float64(i)*2*math.Pi/3
			p := iso.Vec{m[0] + 15*float32(math.Cos(angle)), m[1] + 15*float32(math.Sin(angle)), micBaseH}
			dot(p, 1.1, body.Crease)
			dot(p, 0.5, ink.Edge)
		}
	})
	solid(round(iso.Vec{m[0], m[1], micBaseH}, micBaseR-2, 1), body, nil)
	// The switch sits on the near side of the base.
	sw := m.Add(toward.Scale(13))
	right := iso.Vec{-toward[1], toward[0], 0}
	ring := []iso.Vec{sw.Add(right.Scale(-3)).Add(toward.Scale(-1.8)), sw.Add(right.Scale(3)).Add(toward.Scale(-1.8)), sw.Add(right.Scale(3)).Add(toward.Scale(1.8)), sw.Add(right.Scale(-3)).Add(toward.Scale(1.8))}
	for i := range ring {
		ring[i][2] = micBaseH + 1
	}
	solid(iso.Prism{Ring: ring, Up: iso.Vec{0, 0, 1.4}}, body, func() {
		knob := sw.Add(right.Scale(at.live*1.6 - 0.8))
		knob[2] = micBaseH + 2.4
		dot(knob, 0.9, ink.Edge)
	})
	// The stem, through a knurled collar.
	solid(round(iso.Vec{m[0], m[1], micBaseH + 1}, stemR, capsuleZ0-micBaseH-1), body, nil)
	collarZ := float32(30)
	solid(round(iso.Vec{m[0], m[1], collarZ}, 3.6, 4), body, nil)
	for i := -6; i <= 6; i++ {
		angle := a + float64(i)*math.Pi/14
		p := iso.Vec{m[0] + 3.6*float32(math.Cos(angle)), m[1] + 3.6*float32(math.Sin(angle)), collarZ + 0.6}
		line([]iso.Vec{p, p.Add(iso.Vec{0, 0, 2.8})}, body.Crease)
	}
	// The capsule: a lower shell, the band, the grille, and a bevelled
	// top with its mesh.
	solid(round(iso.Vec{m[0], m[1], capsuleZ0}, capsuleR-1.2, 2), body, nil)
	solid(round(iso.Vec{m[0], m[1], capsuleZ0 + 2}, capsuleR, bandZ-capsuleZ0-2), body, nil)
	solid(round(iso.Vec{m[0], m[1], bandZ}, capsuleR+0.3, bandH), body.Mix(ink.Lit, at.live), nil)
	grille := float32(bandZ + bandH)
	solid(round(iso.Vec{m[0], m[1], grille}, capsuleR, capsuleZ1-grille-1), body, nil)
	// Grille rings on the near half, ending on the outline.
	for z := grille + 2; z < capsuleZ1-2; z += 2.2 {
		var arc []iso.Vec
		for s := 0; s <= 24; s++ {
			angle := a - math.Pi/2 + float64(s)/24*math.Pi
			arc = append(arc, iso.Vec{m[0] + capsuleR*float32(math.Cos(angle)), m[1] + capsuleR*float32(math.Sin(angle)), z})
		}
		line(arc, body.Crease)
	}
	solid(round(iso.Vec{m[0], m[1], capsuleZ1 - 1}, capsuleR-1, 1), body, func() {
		for _, o := range [][2]float32{{0, 0}, {3, 0}, {-3, 0}, {0, 3}, {0, -3}, {2.2, 2.2}, {-2.2, 2.2}, {2.2, -2.2}, {-2.2, -2.2}, {5.2, 0}, {-5.2, 0}, {0, 5.2}, {0, -5.2}} {
			dot(iso.Vec{m[0] + o[0], m[1] + o[1], capsuleZ1}, 0.45, ink.Edge)
		}
	})
}
