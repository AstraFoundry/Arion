package main

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A drag orbits the camera and keeps it turning after a throw, slowing
// to its own orbit; a scroll comes closer, no nearer than near; the hint
// fades once the user moved the camera.
func TestInput(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 1200, 760)
	if !tt.HasText("Drag to orbit · Scroll to zoom") {
		t.Fatal("no hint")
	}
	yaw, pitch := a.cam.yaw, a.cam.pitch
	tt.Press(500, 300)
	tt.Move(560, 330)
	tt.Move(620, 360)
	tt.Release(620, 360)
	if want := yaw - 120*0.005; math.Abs(a.cam.yaw-want) > 0.05 {
		t.Errorf("yaw %.3f after dragging 120 DIPs right, want about %.3f", a.cam.yaw, want)
	}
	if a.cam.pitch <= pitch {
		t.Errorf("pitch %.3f after dragging down from %.3f, want higher", a.cam.pitch, pitch)
	}
	for range 10 {
		a.step(0.1, false)
	}
	if a.hint != 0 {
		t.Errorf("hint %.2f after moving the camera, want faded", a.hint)
	}

	for range 40 {
		tt.Scroll(600, 380, 0, -200)
	}
	if a.cam.dist != near {
		t.Errorf("distance %.2f after scrolling in a long way, want %d", a.cam.dist, near)
	}
}

// The camera looks at the hole, its axes a right-handed frame.
func TestCamera(t *testing.T) {
	c := camera{yaw: 0.7, pitch: 0.3, dist: 20}
	eye, right, up, forward := c.axes()
	if d := math.Sqrt(eye.dot(eye)); math.Abs(d-20) > 1e-9 {
		t.Errorf("eye %.3f from the hole, want 20", d)
	}
	if math.Abs(forward.dot(eye.norm())+1) > 1e-9 {
		t.Error("the camera looks away from the hole")
	}
	if math.Abs(right.dot(up)) > 1e-9 || math.Abs(right.dot(forward)) > 1e-9 || up[1] <= 0 {
		t.Errorf("axes right %v up %v forward %v, want orthogonal, up upward", right, up, forward)
	}
}

// The resolution drops while frames come late, no lower than minQuality,
// and rises again once they keep up.
func TestQuality(t *testing.T) {
	a := newApp()
	for range 600 {
		a.step(1.0/20, false)
	}
	if a.quality != minQuality {
		t.Fatalf("quality %.3f after slow frames, want %.3f", a.quality, minQuality)
	}
	for range 2000 {
		a.step(1.0/120, false)
	}
	if a.quality != 1 {
		t.Fatalf("quality %.3f after fast frames, want 1", a.quality)
	}
}
