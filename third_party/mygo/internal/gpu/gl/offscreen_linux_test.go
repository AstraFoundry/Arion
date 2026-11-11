//go:build linux && (amd64 || arm64)

package gl

import (
	"runtime"
	"testing"
)

// TestNewOffscreenFailsWithItsError checks that a setup failing after EGL
// loaded returns its error, rather than panicking as it undoes what it
// did: a framebuffer of 0×0 is incomplete, and without libepoxy loading
// fails, both after the display and the context.
func TestNewOffscreenFailsWithItsError(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	o, err := NewOffscreen(false, 0, 0)
	if err == nil {
		o.Release()
		t.Skip("a framebuffer of 0×0 is complete here")
	}
	if o != nil {
		t.Errorf("NewOffscreen returned %v with the error %v", o, err)
	}
	t.Log(err)
}
