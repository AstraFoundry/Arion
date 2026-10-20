// Confetti throws confetti in native UI, as canvas-confetti does on a web
// page: bursts of squares, circles, stars, hearts and emoji that fly out,
// slow down, fall, tumble and fade.
//
//	go run ./examples/confetti
//
// Pick one of canvas-confetti's demos, or click the stage to throw a burst
// of your own from the pointer. Each piece is three boxes posed one inside
// the other (AnimatePose), as canvas-confetti adds up its motions: a burst
// whose speed decays, along an easing of its own, a steady fall that fades
// it, and a spin in a plane turned away in 3D (RotationX), which makes it
// tumble. Poses move in frames that only paint, so hundreds of pieces fly
// without building the view again.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const windowW, windowH = 980, 700

func main() {
	s := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Confetti",
			Width:   windowW,
			Height:  windowH,
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
