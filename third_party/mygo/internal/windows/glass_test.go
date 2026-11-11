//go:build windows

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestGlassMargins checks that a frame extended from the bottom covers the
// whole client area, leaving DWM no top to draw caption buttons in, and
// that other windows extend it all over.
func TestGlassMargins(t *testing.T) {
	all := margins{-1, -1, -1, -1}
	if m := glassMargins(false, 760); m != all {
		t.Errorf("all over: %+v, want %+v", m, all)
	}
	if m := glassMargins(true, 1032); m != (margins{Bottom: 1032}) {
		t.Errorf("from the bottom: %+v, want the bottom over 1032 pixels", m)
	}
	if m := glassMargins(true, 0); m.Top != 0 || m.Bottom < 1 {
		t.Errorf("from the bottom without a client area: %+v, want a bottom and no top", m)
	}
}

// TestGlassFromBottom checks which windows extend their frame from the
// bottom: a frameless window, whose styles keep a caption whose buttons DWM
// would draw over its page, a maximized one, whose buttons DWM would draw
// past its title bar, and not a flyout, a popup without a caption.
func TestGlassFromBottom(t *testing.T) {
	frameless := &window{frameless: true, opts: &platform.WindowOptions{Frameless: true}}
	if !frameless.glassFromBottom() {
		t.Error("a frameless window extends its frame all over: DWM draws caption buttons over its page")
	}
	flyout := &window{frameless: true, opts: &platform.WindowOptions{Frameless: true, Flyout: &platform.Flyout{}}}
	if flyout.glassFromBottom() {
		t.Error("a flyout extends its frame from the bottom")
	}
	maximized := &window{showMaximized: true, opts: &platform.WindowOptions{}}
	if !maximized.glassFromBottom() {
		t.Error("a maximized window with the system's title bar extends its frame all over: DWM draws caption buttons past its title bar")
	}
	fullScreen := &window{hiddenTitleBar: true, fullScreen: true, opts: &platform.WindowOptions{}}
	if fullScreen.glassFromBottom() {
		t.Error("a full-screen window with a hidden title bar extends its frame from the bottom")
	}
}
