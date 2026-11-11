package ui

import (
	"testing"
	"time"
)

// fadeInOut fades over a second, linearly, as it comes and as it goes.
var fadeInOut = ElementTransition{Duration: time.Second, Ease: Linear,
	Enter: &Motion{Opacity: 0}, Exit: &Motion{Opacity: 0}}

// TestOverlayEntersAndExits checks that a panel in the overlay layer plays
// its Enter and Exit as one in the window's tree does, though the layer
// comes with the first panel and goes with the last.
func TestOverlayEntersAndExits(t *testing.T) {
	panels := map[string]func(c *context, anchor *node, open *bool){
		"PopoverBase": func(c *context, anchor *node, open *bool) {
			corePopoverBase(c, anchor, open, func(p *node) {
				p.Width(60).Height(60).Background(RGB(255, 0, 0)).Label("panel").Transition(fadeInOut)
			})
		},
		"Overlay": func(c *context, anchor *node, open *bool) {
			if *open {
				coreOverlay(c, func() {
					coreBox(c).Absolute().Left(0).Top(20).Width(60).Height(60).Background(RGB(255, 0, 0)).Label("panel").Transition(fadeInOut)
				})
			}
		},
		"inline": func(c *context, anchor *node, open *bool) {
			if *open {
				coreBox(c).Width(60).Height(60).Background(RGB(255, 0, 0)).Label("panel").Transition(fadeInOut)
			}
		},
	}
	for name, panel := range panels {
		t.Run(name, func(t *testing.T) {
			open := false
			tt, now := clockTester(func(c *context) {
				coreBox(c).Fill().Background(RGB(255, 255, 255)).Children(func() {
					a := coreBox(c).Width(40).Height(20).Label("anchor")
					panel(c, a, &open)
				})
			}, 200, 200)
			tick := ticker(tt, now)
			// The centre of the panel, below the anchor.
			g := func() uint8 { return tt.Image().RGBAAt(30, 50).G }
			open = true
			tt.Frame()
			tick(500 * time.Millisecond)
			if v := g(); v < 110 || v > 145 {
				t.Errorf("halfway in, G=%d, want about 128", v)
			}
			tick(time.Second)
			if v := g(); v != 0 {
				t.Errorf("once in, G=%d", v)
			}
			open = false
			tt.Frame()
			tick(500 * time.Millisecond)
			if v := g(); v < 110 || v > 145 {
				t.Errorf("halfway out, G=%d, want about 128", v)
			}
			tick(time.Second)
			if v := g(); v != 255 {
				t.Errorf("once out, G=%d", v)
			}
			tick(time.Second)
			if ov := tt.rt.c.overlay; ov != nil {
				t.Errorf("the overlay layer stays once its last copy went, with %d children", ov.nchild)
			}
		})
	}
}

// TestOverlayExitLetsInputThrough checks that the layer a panel's copy
// leaves in takes neither the pointer nor the focus from the window.
func TestOverlayExitLetsInputThrough(t *testing.T) {
	open := true
	clicks := 0
	var under *node
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			a := coreBox(c).Width(40).Height(20).Label("anchor")
			under = coreButton(c, "Under").Label("under")
			if under.Clicked() {
				clicks++
			}
			corePopoverBase(c, a, &open, func(p *node) {
				p.Width(120).Height(60).Background(RGB(255, 0, 0)).Transition(fadeInOut)
			})
		})
	}, 200, 200)
	tick := ticker(tt, now)
	tick(2 * time.Second)
	open = false
	tt.Frame()
	tick(100 * time.Millisecond)
	if tt.rt.c.overlay == nil || tt.rt.c.overlay.first == nil {
		t.Fatal("no copy of the panel leaving")
	}
	r := at(t, tt, "under")
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tick(16 * time.Millisecond)
	if clicks != 1 {
		t.Errorf("a click through the panel leaving clicked the button under it %d times", clicks)
	}
	tt.Key(0, KeyTab)
	tick(16 * time.Millisecond)
	if !under.Focused() {
		t.Error("Tab does not focus the window's button while the panel leaves")
	}
}

// TestTooltipEntersAndExits checks a tip of TooltipBase, which comes and
// goes as the pointer rests on its anchor and leaves it.
func TestTooltipEntersAndExits(t *testing.T) {
	tt, now := clockTester(func(c *context) {
		coreBox(c).Fill().Background(RGB(255, 255, 255)).Padding(40).Children(func() {
			a := coreBox(c).Width(40).Height(20).Label("anchor")
			coreTooltipBase(c, a, func(tip *node) {
				tip.AttachTo(a, AnchorBottomStart, AnchorTopStart)
				tip.Width(60).Height(60).Background(RGB(255, 0, 0)).Transition(fadeInOut)
			})
		})
	}, 200, 200)
	tick := ticker(tt, now)
	g := func() uint8 { return tt.Image().RGBAAt(70, 90).G }
	tt.Move(60, 50)
	for i := 0; i < 50 && tt.rt.c.overlay == nil; i++ {
		tick(50 * time.Millisecond)
	}
	if tt.rt.c.overlay == nil {
		t.Fatal("no tip")
	}
	tick(500 * time.Millisecond)
	if v := g(); v < 110 || v > 145 {
		t.Errorf("halfway in, G=%d, want about 128", v)
	}
	tick(time.Second)
	tt.Move(190, 190)
	tick(16 * time.Millisecond)
	tick(500 * time.Millisecond)
	if v := g(); v < 110 || v > 145 {
		t.Errorf("halfway out, G=%d, want about 128", v)
	}
	tick(time.Second)
	if v := g(); v != 255 {
		t.Errorf("once out, G=%d", v)
	}
}

// TestAttachedPanelEntersWithTravel checks that the X and Y of an Enter
// show on a panel attached to its anchor (AttachTo), as they do on one
// placed by hand, and as a transform on the entering element would.
func TestAttachedPanelEntersWithTravel(t *testing.T) {
	enter := ElementTransition{Duration: time.Second, Ease: Linear, Enter: &Motion{X: 20, Y: 20}}
	panels := map[string]func(c *context, anchor *node, open *bool){
		"PopoverBase": func(c *context, anchor *node, open *bool) {
			corePopoverBase(c, anchor, open, func(p *node) {
				p.Width(60).Height(60).Label("panel").Transition(enter)
			})
		},
		"Overlay": func(c *context, anchor *node, open *bool) {
			if *open {
				coreOverlay(c, func() {
					coreBox(c).Absolute().Left(0).Top(20).Width(60).Height(60).Label("panel").Transition(enter)
				})
			}
		},
	}
	for name, panel := range panels {
		t.Run(name, func(t *testing.T) {
			open := false
			tt, now := clockTester(func(c *context) {
				coreBox(c).Fill().Children(func() {
					a := coreBox(c).Width(40).Height(20).Label("anchor")
					panel(c, a, &open)
				})
			}, 200, 200)
			tick := ticker(tt, now)
			open = true
			tt.Frame()
			tick(500 * time.Millisecond)
			half := at(t, tt, "panel")
			tick(time.Second)
			rest := at(t, tt, "panel")
			if d := half.Y - rest.Y; d != 10 {
				t.Errorf("halfway in, Y is %v from its place, want 10", d)
			}
			if d := half.X - rest.X; d != 10 {
				t.Errorf("halfway in, X is %v from its place, want 10", d)
			}
		})
	}
}

// TestAttachedPanelEntersEachAxis checks that an Enter's X alone moves a
// panel attached to its anchor (AttachTo) along X, and its Y alone along
// Y, ending at its place.
func TestAttachedPanelEntersEachAxis(t *testing.T) {
	axes := []struct {
		name   string
		motion Motion
		dx, dy float32
	}{
		{"X", Motion{X: 20}, 10, 0},
		{"Y", Motion{Y: 20}, 0, 10},
	}
	for _, tc := range axes {
		t.Run(tc.name, func(t *testing.T) {
			enter := ElementTransition{Duration: time.Second, Ease: Linear, Enter: &tc.motion}
			var open bool
			tt, now := clockTester(func(c *context) {
				coreBox(c).Fill().Children(func() {
					a := coreBox(c).Width(40).Height(20).Label("anchor")
					corePopoverBase(c, a, &open, func(p *node) {
						p.Width(60).Height(60).Label("panel").Transition(enter)
					})
				})
			}, 200, 200)
			tick := ticker(tt, now)
			open = true
			tt.Frame()
			tick(500 * time.Millisecond)
			half := at(t, tt, "panel")
			tick(time.Second)
			rest := at(t, tt, "panel")
			if d := half.X - rest.X; d != tc.dx {
				t.Errorf("halfway in, X is %v from its place, want %v", d, tc.dx)
			}
			if d := half.Y - rest.Y; d != tc.dy {
				t.Errorf("halfway in, Y is %v from its place, want %v", d, tc.dy)
			}
		})
	}
}

// TestAttachedPanelStaysOnMovingAnchor checks that a panel with an Enter
// transition stays on its anchor as the anchor moves, once it has come in:
// a move of the layout is not part of the Enter's travel.
func TestAttachedPanelStaysOnMovingAnchor(t *testing.T) {
	enter := ElementTransition{Duration: time.Second, Ease: Linear, Enter: &Motion{X: 20, Y: 20}}
	var open bool
	shift := float32(50)
	tt, now := clockTester(func(c *context) {
		coreBox(c).Fill().Children(func() {
			a := coreBox(c).Width(40).Height(20).Margin(0, 0, 0, shift).Label("anchor")
			corePopoverBase(c, a, &open, func(p *node) {
				p.Width(60).Height(60).Label("panel").Transition(enter)
			})
		})
	}, 300, 200)
	tick := ticker(tt, now)
	open = true
	tt.Frame()
	tick(2 * time.Second)
	gap := at(t, tt, "panel").X - at(t, tt, "anchor").X
	shift = 150
	tt.Frame()
	tick(100 * time.Millisecond)
	if d := at(t, tt, "panel").X - at(t, tt, "anchor").X; d != gap {
		t.Errorf("anchor moved: panel is %v from it, want %v", d, gap)
	}
}
