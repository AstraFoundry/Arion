package bootstrap

import (
	"github.com/egoist/mygo"
	"testing"
)

type accessLayoutWindow struct {
	bounds         mygo.Rectangle
	opacity        float64
	focusCount     int
	visibleResizes int
}

func (w *accessLayoutWindow) Bounds() mygo.Rectangle { return w.bounds }
func (w *accessLayoutWindow) SetBounds(bounds mygo.Rectangle) {
	if w.opacity > 0 {
		w.visibleResizes++
	}
	w.bounds = bounds
}
func (w *accessLayoutWindow) SetOpacity(opacity float64) { w.opacity = opacity }
func (w *accessLayoutWindow) Focus()                     { w.focusCount++ }

// CLP-A08: provisional layouts never become visible or take focus.
func TestClipboardFilesFirstVisibleLayoutIsSettled(t *testing.T) {
	window := &accessLayoutWindow{bounds: mygo.Rectangle{X: 100, Y: 100, Width: 360, Height: 200}}
	var layout clipboardAccessLayout
	for _, height := range []int{0, 214, 213} {
		layout.apply(window, height)
		if window.opacity != 0 || window.focusCount != 0 {
			t.Fatal("provisional layout was exposed")
		}
	}
	layout.apply(window, 213)
	if window.opacity != 1 || window.focusCount != 1 || window.bounds.Height != 213 || window.visibleResizes != 0 {
		t.Fatal("first visible geometry was not settled")
	}
	layout.apply(window, 213)
	if window.focusCount != 1 {
		t.Fatal("stable repaint stole focus again")
	}
}
