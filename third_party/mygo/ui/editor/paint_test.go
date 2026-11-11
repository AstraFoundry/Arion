package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A deletion bar's stripes keep to the content: those of lines whose
// height isn't a multiple of 4 continue one another's, a stripe across
// a join split between the two, and scrolling moves them with the lines.
func TestBarStripes(t *testing.T) {
	// spans returns the stripes, as [top, bottom) in the content, of
	// lines of heights hs from top, at screenY on the screen.
	spans := func(top float64, screenY float32, hs ...float32) [][2]float32 {
		var out [][2]float32
		y := screenY
		for _, h := range hs {
			barStripes(ui.Rect{X: 0, Y: y, W: 4, H: h}, top+float64(y-screenY), func(r ui.Rect) {
				a, b := r.Y-screenY+float32(top), r.Y+r.H-screenY+float32(top)
				if n := len(out); n > 0 && out[n-1][1] == a {
					out[n-1][1] = b // a stripe split at a join
					return
				}
				out = append(out, [2]float32{a, b})
			})
			y += h
		}
		return out
	}
	check := func(name string, got [][2]float32, from, to float32) {
		t.Helper()
		var want [][2]float32
		for y := float32(0); y < to; y += 4 {
			if a, b := max(y, from), min(y+2, to); b > a {
				want = append(want, [2]float32{a, b})
			}
		}
		if len(got) != len(want) {
			t.Fatalf("%s: stripes %v, want %v", name, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: stripes %v, want %v", name, got, want)
			}
		}
	}
	check("18", spans(0, 10, 18, 18, 18), 0, 54)
	check("19.5", spans(0, 10, 19.5, 19.5, 19.5), 0, 58.5)
	// Lines from 37 in the content, scrolled by 11.25 and by 12.
	check("scrolled", spans(37, 25.75, 19.5, 19.5), 37, 76)
	check("scrolled more", spans(37, 25, 19.5, 19.5), 37, 76)
}
