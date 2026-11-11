//go:build windows && (amd64 || arm64)

package windows

import "testing"

// TestWorkAreaMaximized checks that a window Windows maximizes over a
// whole monitor, as it does one without a maximize box, moves into the
// work area with the same invisible borders around it, and that other
// rectangles stay.
func TestWorkAreaMaximized(t *testing.T) {
	monitor := rect{0, 0, 3840, 2160}
	bottom := rect{0, 0, 3840, 2064} // a taskbar 96 px high at 200%
	for _, c := range []struct {
		name         string
		r, mon, work rect
		want         rect
		ok           bool
	}{
		{"over the monitor, taskbar at the bottom", rect{-16, -16, 3856, 2176}, monitor, bottom, rect{-16, -16, 3856, 2080}, true},
		{"over the monitor, taskbar on the left", rect{-16, -16, 3856, 2176}, monitor, rect{120, 0, 3840, 2160}, rect{104, -16, 3856, 2176}, true},
		{"over the monitor, no borders", rect{0, 0, 3840, 2160}, monitor, bottom, bottom, true},
		{"a secondary monitor", rect{3832, -88, 5768, 1008}, rect{3840, -80, 5760, 1000}, rect{3840, -80, 5760, 952}, rect{3832, -88, 5768, 960}, true},
		{"already in the work area", rect{-16, -16, 3856, 2080}, monitor, bottom, rect{-16, -16, 3856, 2080}, false},
		{"a work area as large as the monitor", rect{-16, -16, 3856, 2176}, monitor, monitor, rect{-16, -16, 3856, 2176}, false},
		{"a window smaller than the monitor", rect{100, 100, 900, 700}, monitor, bottom, rect{100, 100, 900, 700}, false},
	} {
		got, ok := workAreaMaximized(c.r, c.mon, c.work)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: workAreaMaximized(%+v) = %+v, %v; want %+v, %v", c.name, c.r, got, ok, c.want, c.ok)
		}
	}
}
