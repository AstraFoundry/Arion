//go:build linux && (amd64 || arm64)

package gl

import "testing"

func TestNeedsRestore(t *testing.T) {
	const area, other = 0x10, 0x20
	for _, c := range []struct {
		name          string
		area, current uintptr
		has, want     bool
	}{
		{"the area's, current", area, area, true, false},
		{"another window's", area, other, true, true},
		{"none, cleared", area, 0, false, true},
		{"GDK's, stale", area, area, false, true},
		{"none in GDK alone", area, 0, true, true},
		{"an area without a context", 0, other, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := needsRestore(c.area, c.current, c.has); got != c.want {
				t.Errorf("needsRestore = %v, want %v", got, c.want)
			}
		})
	}
}

// Without GTK, as in tests, MakeAreaCurrent changes nothing.
func TestMakeAreaCurrentWithoutGTK(t *testing.T) {
	if err := load(); err != nil {
		t.Skip(err)
	}
	if gdkClearCurrent != nil {
		t.Skip("GTK is loaded")
	}
	if MakeAreaCurrent(0x10) {
		t.Error("a context is current without one made")
	}
}
