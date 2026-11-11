package ui

import (
	"strings"
	"testing"
)

// fitTester tests view in a window that takes its content's size, between
// the widths lo and hi: told holds the sizes the content told, and the
// window takes each, as a flyout does.
func fitTester(view func(c *Context), lo, hi float64) (*Tester, *[][2]float64) {
	tt := NewTester(view, 300, 200)
	told := &[][2]float64{}
	tt.rt.fit = newFitter(func() (float64, float64) { return lo, hi }, func(w, h float64) {
		*told = append(*told, [2]float64{w, h})
	})
	tt.Frame()
	return tt, told
}

// take resizes the window to the size its content last told, and draws a
// frame at that size.
func take(tt *Tester, told [][2]float64) {
	s := told[len(told)-1]
	tt.SetSize(int(s[0]), int(s[1]))
}

func TestFitContentMeasures(t *testing.T) {
	rows, wide := 2, false
	tt, told := fitTester(func(c *Context) {
		Column(c).Fill().Padding(10).Gap(5).Children(func() {
			for i := 0; i < rows; i++ {
				w := float32(100)
				if wide {
					w = 150
				}
				Box(c).Size(w, 20)
			}
			// What is placed absolutely takes none of the size.
			Box(c).Absolute().Left(0).Top(0).Size(500, 500)
		})
	}, 0, 0)
	want := func(what string, w, h float64, n int) {
		t.Helper()
		if len(*told) != n || n > 0 && (*told)[n-1] != [2]float64{w, h} {
			t.Fatalf("%s: told %v, want %d sizes, the last %vx%v", what, *told, n, w, h)
		}
	}
	want("the first frame", 120, 65, 1)
	take(tt, *told)
	want("the frame at that size", 120, 65, 1)
	tt.Frame()
	want("the next", 120, 65, 1)

	rows = 4
	tt.Frame()
	want("more rows", 120, 115, 2)
	take(tt, *told)
	wide = true
	tt.Frame()
	want("wider rows", 170, 115, 3)

	// The window took another size, as SetSize gives: it holds while the
	// content measures the same.
	tt.SetSize(250, 250)
	tt.Frame()
	want("another size", 170, 115, 3)

	// Zoomed, the content tells the window's DIPs.
	tt.SetZoom(2)
	want("zoomed", 340, 230, 4)
}

// Text wraps at the most width, and its height is that of its lines at
// that width; below the least width, the content takes it.
func TestFitContentBounds(t *testing.T) {
	long := strings.Repeat("A line of text that wraps. ", 20)
	text := long
	_, told := fitTester(func(c *Context) {
		Column(c).Fill().Children(func() { Text(c, text) })
	}, 80, 200)
	if len(*told) != 1 {
		t.Fatalf("told %v", *told)
	}
	s := (*told)[0]
	if s[0] != 200 {
		t.Errorf("long text: width %v, want the most, 200", s[0])
	}
	oneLine := NewTester(func(c *Context) { Column(c).Children(func() { Text(c, "A") }) }, 300, 200)
	line := float64(oneLine.rt.c.root.first.h)
	if s[1] < 3*line {
		t.Errorf("long text: height %v, a line %v: not wrapped at 200", s[1], line)
	}

	_, told = fitTester(func(c *Context) { Box(c).Size(10, 10) }, 80, 200)
	if got := *told; len(got) != 1 || got[0] != [2]float64{80, 10} {
		t.Errorf("narrow content: told %v, want 80x10", got)
	}
}

// A view that builds its content after the window's size, which would
// have the window resize back and forth, holds once it measures again the
// size it told before.
func TestFitContentHoldsAViewFollowingItsSize(t *testing.T) {
	tt, told := fitTester(func(c *Context) {
		w, _ := c.Size()
		if w > 200 {
			Box(c).Size(150, 40)
		} else {
			Box(c).Size(250, 40)
		}
	}, 0, 0)
	take(tt, *told) // at 150, it measures 250
	take(tt, *told) // at 250, 150 again
	if got := *told; len(got) != 2 || got[0] != [2]float64{150, 40} || got[1] != [2]float64{250, 40} {
		t.Errorf("told %v, want 150x40 then 250x40", got)
	}
}

// A Scroll in content taller than the window may be scrolls, and what
// follows it shows: as a flyout's MaxHeight bounds it.
func TestFitContentScrollsBeyondTheMost(t *testing.T) {
	tt, told := fitTester(func(c *Context) {
		Column(c).Fill().Padding(12).Gap(8).Children(func() {
			Text(c, "Nodes")
			Scroll(c).Grow(1).Children(func() {
				for i := 0; i < 40; i++ {
					Box(c).Size(100, 20)
				}
			})
			Text(c, "an error")
		})
	}, 0, 0)
	if got := *told; len(got) != 1 || got[0][0] != 124 || got[0][1] < 40*20 {
		t.Fatalf("told %v, want 124 wide and the height of all the rows", got)
	}
	tt.SetSize(124, 300)
	if r, ok := tt.Find("an error"); !ok || r.Y+r.H > 300-12 {
		t.Errorf("the error line at %+v, %v, in a window 300 high", r, ok)
	}
}
