//go:build linux && (amd64 || arm64)

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
)

// TestGLProbeWindow checks that the probe of OpenGL leaves its window to
// GDK, which frees it once the X server says it destroyed it. Freed
// before, GDK handled that DestroyNotify on freed memory, and apps whose
// first window of native UI probed a GPU crashed as they started.
func TestGLProbeWindow(t *testing.T) {
	var (
		failure string
		freed   func() bool
		early   bool
	)
	mygo.RunOnMain(func() {
		if _, failure, freed = linux.TestProbeGL(); freed != nil {
			early = freed()
		}
	})
	if freed == nil {
		t.Skipf("no probe window to watch, off X11 or before it (%s)", failure)
	}
	if early {
		t.Fatal("the probe freed its window before the X server destroyed it")
	}
	eventually(t, "GDK freeing the probe's window", func() (ok bool) {
		mygo.RunOnMain(func() { ok = freed() })
		return ok
	})
}
