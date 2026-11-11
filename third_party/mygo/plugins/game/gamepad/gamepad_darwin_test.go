package gamepad

import (
	"math"
	"testing"

	"github.com/ebitengine/purego/objc"
)

// A controller GameController makes for tests, of the extended gamepad
// profile, reads in this package's layout: its y axes turned to go down.
func TestGameController(t *testing.T) {
	if !loadGC() {
		t.Skip("no GameController.framework")
	}
	saved := gcPresent
	gcPresent = func(objc.ID) bool { return true }
	defer func() { gcPresent = saved }()
	var d *gcPad
	withPool(func() {
		c := objc.ID(objc.GetClass("GCController")).Send(objc.RegisterName("controllerWithExtendedGamepad"))
		if c == 0 {
			return
		}
		d = &gcPad{ctl: c.Send(selRetain)}
		gp := c.Send(selExtended)
		setValue := objc.RegisterName("setValue:")
		gp.Send(buttonSels[South]).Send(setValue, float32(1))
		gp.Send(buttonSels[RightTrigger]).Send(setValue, float32(0.6))
		setAxes := objc.RegisterName("setValueForXAxis:yAxis:")
		gp.Send(selLeftStick).Send(setAxes, float32(0.5), float32(0.25))
		gp.Send(selDpad).Send(setAxes, float32(0), float32(1))
	})
	if d == nil {
		t.Skip("no controllerWithExtendedGamepad")
	}
	defer d.close()
	var s State
	if !d.read(&s) {
		t.Fatal("read failed")
	}
	if !s.Pressed(South) || s.Pressed(East) || !s.Pressed(Up) || s.Pressed(Down) {
		t.Errorf("buttons %v", s.Buttons)
	}
	if math.Abs(float64(s.Value(RightTrigger))-0.6) > 1e-3 {
		t.Errorf("right trigger %v", s.Value(RightTrigger))
	}
	if math.Abs(float64(s.Axes[LeftX])-0.5) > 1e-3 || math.Abs(float64(s.Axes[LeftY])+0.25) > 1e-3 {
		t.Errorf("left stick %v, %v: y should go down", s.Axes[LeftX], s.Axes[LeftY])
	}
}
