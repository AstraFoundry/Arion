//go:build windows && (amd64 || arm64)

package windows

import "testing"

// TestShowsMaterial checks that a window tells its content a material
// shows only where DWM draws one: a flyout without a redirection bitmap
// shows none before Windows 11 22H2 (build 22621).
func TestShowsMaterial(t *testing.T) {
	build := windowsBuild
	t.Cleanup(func() { windowsBuild = build })
	for _, c := range []struct {
		build      uint32
		noRedirect bool
		vibrancy   string
		want       bool
	}{
		{19045, true, "popover", false}, // Windows 10
		{22000, true, "popover", false}, // Windows 11 21H2
		{22621, true, "popover", true},
		{26100, true, "popover", true},
		{26100, false, "popover", false}, // the bitmap covers it
		{26100, true, "", false},         // no material
	} {
		windowsBuild = func() uint32 { return c.build }
		s := &surface{w: &window{noRedirect: c.noRedirect, vibrancy: c.vibrancy}}
		if got := s.ShowsMaterial(); got != c.want {
			t.Errorf("build %d, noRedirect %v, vibrancy %q: %v, want %v", c.build, c.noRedirect, c.vibrancy, got, c.want)
		}
	}
}
