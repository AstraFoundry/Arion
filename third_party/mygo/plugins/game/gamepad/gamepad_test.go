package gamepad

import (
	"math"
	"testing"
)

// fakeDevice is a pad whose state a test sets.
type fakeDevice struct {
	id     string
	state  State
	gone   bool
	closed bool
}

func (d *fakeDevice) key() string  { return d.id }
func (d *fakeDevice) name() string { return "Fake " + d.id }
func (d *fakeDevice) close()       { d.closed = true }
func (d *fakeDevice) read(s *State) bool {
	*s = d.state
	return !d.gone
}

// withDevices has scans find devs, which a test changes.
func withDevices(t *testing.T, devs *[]device) {
	t.Helper()
	saved, savedEach := scanDevices, scanEach
	scanDevices = func(known map[string]device) []device { return *devs }
	scanEach = 0
	mu.Lock()
	pads, scanned = nil, scanned.AddDate(-1, 0, 0)
	mu.Unlock()
	t.Cleanup(func() {
		scanDevices, scanEach = saved, savedEach
		mu.Lock()
		pads = nil
		mu.Unlock()
	})
}

func TestPadsConnectAndDisconnect(t *testing.T) {
	a, b := &fakeDevice{id: "a"}, &fakeDevice{id: "b"}
	devs := []device{a}
	withDevices(t, &devs)
	ps := Pads()
	if len(ps) != 1 || ps[0].Name() != "Fake a" || !ps[0].Connected() {
		t.Fatalf("pads %v", ps)
	}
	first := ps[0]
	devs = []device{b, a}
	ps = Pads()
	if len(ps) != 2 || ps[0] != first || ps[1].Name() != "Fake b" {
		t.Fatalf("pads in connecting order: %v", ps)
	}
	// Unplugged between scans: reading finds it gone.
	a.gone = true
	a.state.Buttons[South] = 1
	if first.State().Pressed(South) || first.Connected() || !a.closed {
		t.Error("a pad gone still reads, or is not closed")
	}
	// Gone from a scan.
	devs = nil
	if ps = Pads(); len(ps) != 0 {
		t.Errorf("pads after unplugging: %v", ps)
	}
}

func TestState(t *testing.T) {
	var last, s State
	s.Buttons[South] = 1
	s.Buttons[LeftTrigger] = 0.2
	s.Buttons[RightTrigger] = 0.3
	if !s.Pressed(South) || s.Pressed(LeftTrigger) || !s.Pressed(RightTrigger) {
		t.Error("Pressed")
	}
	if !s.JustPressed(last, South) || s.JustPressed(s, South) || !last.JustReleased(s, South) {
		t.Error("JustPressed, JustReleased")
	}
	// Within the dead zone, nothing; at the rim, all the way, in every
	// direction alike.
	s.Axes[LeftX], s.Axes[LeftY] = 0.1, 0.05
	if x, y := s.LeftStick(); x != 0 || y != 0 {
		t.Errorf("rest = %v, %v", x, y)
	}
	s.Axes[LeftX], s.Axes[LeftY] = 0.7071, 0.7071
	if x, y := s.LeftStick(); math.Abs(math.Hypot(float64(x), float64(y))-1) > 1e-3 {
		t.Errorf("diagonal rim = %v, %v", x, y)
	}
	s.Axes[RightX] = -(Deadzone + (1-Deadzone)/2)
	if x, _ := s.RightStick(); math.Abs(float64(x)+0.5) > 1e-4 {
		t.Errorf("half way past the dead zone = %v", x)
	}
	if s.Value(ButtonCount) != 0 || s.Axis(AxisCount) != 0 {
		t.Error("out of range")
	}
}
