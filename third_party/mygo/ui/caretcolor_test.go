package ui

import (
	"image/color"
	"testing"
)

// TestCaretColor checks that the caret of a focused text input is the
// color CaretColor sets on it or on an element around it, and the theme's
// accent without.
func TestCaretColor(t *testing.T) {
	magenta := RGB(255, 0, 255)
	count := func(tt *Tester, want color.RGBA) int {
		img, n := tt.Image(), 0
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				if img.RGBAAt(x, y) == want {
					n++
				}
			}
		}
		return n
	}
	for name, build := range map[string]func(c *context, s *string){
		"on the input": func(c *context, s *string) { coreTextInput(c, s).Width(100).AutoFocus().CaretColor(magenta) },
		"inherited": func(c *context, s *string) {
			coreBox(c).CaretColor(magenta).Children(func() { coreTextInput(c, s).Width(100).AutoFocus() })
		},
		"on a text area": func(c *context, s *string) {
			coreTextArea(c, s).Width(100).Height(60).AutoFocus().CaretColor(magenta)
		},
	} {
		var s string
		tt := coreNewTester(func(c *context) { build(c, &s) }, 200, 100)
		if n := count(tt, color.RGBA{255, 0, 255, 255}); n == 0 {
			t.Errorf("%s: no pixel of the caret's color", name)
		}
	}
	var s string
	tt := coreNewTester(func(c *context) { coreTextInput(c, &s).Width(100).AutoFocus() }, 200, 100)
	if n := count(tt, color.RGBA{255, 0, 255, 255}); n != 0 {
		t.Errorf("without CaretColor: %d pixels of magenta", n)
	}
}
