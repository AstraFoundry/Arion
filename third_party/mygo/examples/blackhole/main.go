// Black hole is a black hole and its accretion disk, drawn by shaders
// written in Go (package ui/shader) on the GPU: every pixel's ray of light
// is traced back through the hole's gravity, so the disk's far side shows
// above and below the shadow, the side turning toward the camera blazes
// brighter and bluer, and the stars behind smear into arcs around an
// Einstein ring.
//
//	go run ./examples/blackhole
//
// Drag to orbit, scroll to come closer. Three passes draw a frame: the
// scene into a canvas, its brightest light blurred into a smaller one, and
// both over the window with a lens's color fringes and a film's grain.
// The scene renders at a resolution of its own, which drops while frames
// come late and rises again once they keep up, as games do.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:           "Black Hole",
			Width:           1200,
			Height:          760,
			TitleBarStyle:   mygo.TitleBarHiddenInset,
			BackgroundColor: "#000000",
			Content:         ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
