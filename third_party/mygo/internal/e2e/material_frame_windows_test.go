//go:build windows && (amd64 || arm64)

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// A window with a material and the system's title bar extends its frame all
// over while restored, and from the bottom while maximized: a frame extended
// all over has DWM draw the caption buttons of a maximized window past its
// title bar, under the translucent content.
func TestContentWindowMaximizedMaterialFrame(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{X: 80, Y: 80, Width: 480, Height: 320, Vibrancy: mygo.VibrancyMica,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "native") })})
	allOver := func() bool {
		top, _ := frameExtension(w)
		return top == -1
	}
	if !allOver() {
		t.Error("a restored window doesn't extend its frame all over")
	}
	w.Maximize()
	eventually(t, "the frame extended from the bottom", func() bool {
		top, bottom := frameExtension(w)
		return top == 0 && bottom > 0
	})
	w.Unmaximize()
	eventually(t, "the frame extended all over", allOver)
}
