package ui

import (
	"testing"
	"time"
)

// benchTile and benchTitle are the identities the tiles of the Magic Move
// benchmarks share with their detail view.
type (
	benchTile  struct{ i int }
	benchTitle struct{ i int }
)

// Shared elements in the Magic Move benchmarks: none, a Match, a Match
// flying, or flying with the titles morphing.
const (
	unmatched = iota
	matched
	flying
	morphing
)

// magicMoveScene is a grid of n tiles with titles, in a scroll view, that
// opens one into a detail view every period frames and closes it as
// many after, as the gallery's Magic Move card does: with a Match, the
// tile and its title move into the view's hero and heading, flying, the
// title scaled, if they fly, while the rest fades; without, the views
// only fade.
func magicMoveScene(n, period, shared int) func() (func(c *Context), func()) {
	return func() (func(c *Context), func()) {
		frame, open := 0, -1
		motion := ElementTransition{Duration: 300 * time.Millisecond}
		fade := ElementTransition{Duration: 300 * time.Millisecond, Enter: &Motion{}, Exit: &Motion{}}
		transitions := func(i int) (tile, title ElementTransition) {
			tile, title = motion, motion
			if shared != unmatched {
				tile.Match, title.Match = benchTile{i}, benchTitle{i}
			}
			if shared >= flying {
				tile.Fly, title.Fly, title.ScaleContent = true, true, shared == flying
			}
			return tile, title
		}
		var morph ContentTransition
		if shared == morphing {
			morph = ContentTransition{Morph: true}
		}
		color := func(i int) Color { return RGB(uint8(40+i*37%180), uint8(80+i*13%150), uint8(120+i*29%120)) }
		view := func(c *Context) {
			Box(c).Fill().Background(RGB(246, 247, 249)).Children(func() {
				if open < 0 {
					Column(c.Key("grid")).Absolute().Fill().Transition(fade).Children(func() {
						Scroll(c).Fill().Children(func() {
							Grid(c).Columns(6).Gap(12).Padding(12).Children(func() {
								for i := range n {
									tile, title := transitions(i)
									Column(c.Key(i)).Height(90).Padding(8).Justify(End).Radius(12).Background(color(i)).Transition(tile).Children(func() {
										Textf(c, "Tile %d", i).FontSize(12).Bold().TextColor(RGB(255, 255, 255)).Transition(title).ContentTransition(morph)
									})
								}
							})
						})
					})
					return
				}
				tile, title := transitions(open)
				Column(c.Key("detail")).Absolute().Fill().Padding(24).Gap(16).Transition(fade).Children(func() {
					Box(c.Key("hero")).Height(320).Radius(16).Background(color(open)).Transition(tile)
					heading := "Tile %d"
					if shared == morphing {
						heading = "Tile %d at dusk"
					}
					Textf(c, heading, open).FontSize(28).Bold().Transition(title).ContentTransition(morph)
					Text(c, "The light goes from the sea to the sky as the sun sets behind the hills, and the boats come in.").Wrap()
				})
			})
		}
		step := func() {
			frame++
			if period > 0 && frame%period == 0 {
				if open < 0 {
					open = frame / period * 7 % n
				} else {
					open = -1
				}
			}
		}
		return view, step
	}
}

// BenchmarkMagicMove builds, lays out and paints the frames of a grid of
// 60 tiles opening one into a detail view and closing it: steady, with
// nothing moving, and toggling every 18 frames, the 300 ms the
// transitions take, with the views' shared elements moving (match),
// flying (fly), flying with the titles morphing (morph), or the views
// only fading.
func BenchmarkMagicMove(b *testing.B) {
	for _, sc := range []struct {
		name           string
		period, shared int
	}{
		{"steady", 0, unmatched},
		{"steady_match", 0, matched},
		{"steady_fly", 0, flying},
		{"toggle", 18, unmatched},
		{"toggle_match", 18, matched},
		{"toggle_fly", 18, flying},
		{"toggle_morph", 18, morphing},
	} {
		b.Run(sc.name, func(b *testing.B) {
			f := newStressFrames(magicMoveScene(60, sc.period, sc.shared), nil)
			b.ReportAllocs()
			for b.Loop() {
				f.next()
			}
		})
	}
}

// TestMagicMoveScene checks the benchmark's scene flies its elements.
func TestMagicMoveScene(t *testing.T) {
	f := newStressFrames(magicMoveScene(60, 18, flying), nil)
	for range 15 {
		f.next() // to the first toggle
	}
	flew := 0
	for range 36 {
		f.next()
		if m, _ := f.rt.matches.(*matches); m != nil && len(m.lifted) > 0 {
			flew++
		}
	}
	if flew < 30 {
		t.Errorf("elements flew in %d frames of 36", flew)
	}
}
