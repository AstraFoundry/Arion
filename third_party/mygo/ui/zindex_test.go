package ui

import (
	"image/color"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

var (
	zRed   = RGB(255, 0, 0)
	zGreen = RGB(0, 255, 0)
	zBlue  = RGB(0, 0, 255)
)

// isColor reports whether c is the opaque color want.
func isColor(c color.RGBA, want Color) bool {
	return c.R == want.R && c.G == want.G && c.B == want.B && c.A == 255
}

// A stack of overlapping avatars, the first on top: each paints over the
// one after it and takes the pointer there, while the focus, assistive
// technology and the layout keep the tree's order.
func TestZIndexAvatars(t *testing.T) {
	var avatars [3]*node
	var clicks [3]int
	z := true
	tt := coreNewTester(func(c *context) {
		coreRow(c).Children(func() {
			for i, col := range []Color{zRed, zGreen, zBlue} {
				a := coreButtonBase(c).Size(32, 32).Shrink(0).Background(col).Label([]string{"Ada", "Bob", "Cy"}[i])
				if i > 0 {
					a.Margin(0, 0, 0, -16)
				}
				if z {
					a.ZIndex(3 - i)
				}
				if a.Clicked() {
					clicks[i]++
				}
				avatars[i] = a
			}
		})
	}, 100, 40)
	tt.Frame()
	layout := [3]float32{}
	for i, a := range avatars {
		layout[i] = a.x
	}
	if layout != [3]float32{0, 16, 32} {
		t.Fatalf("the avatars are at %v", layout)
	}
	img := tt.Image()
	for _, at := range []struct {
		x    int
		want Color
	}{{8, zRed}, {24, zRed}, {40, zGreen}, {56, zBlue}} {
		if c := img.RGBAAt(at.x, 16); !isColor(c, at.want) {
			t.Errorf("at %d the avatars paint %v, want %v", at.x, c, at.want)
		}
	}
	tt.ClickAt(24, 16) // Ada over Bob
	tt.ClickAt(40, 16) // Bob over Cy
	if clicks != [3]int{1, 1, 0} {
		t.Errorf("clicks where the avatars overlap: %v, want [1 1 0]", clicks)
	}

	order := slices.Clone(tt.rt.focusOrder)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	var read []string
	for _, n := range tt.h.access.Nodes {
		if n.Role == platform.RoleButton {
			read = append(read, n.Label)
		}
	}
	if want := []uint64{avatars[0].id, avatars[1].id, avatars[2].id}; !slices.Equal(order, want) {
		t.Errorf("Tab goes %v, want the tree's order %v", order, want)
	}
	if !slices.Equal(read, []string{"Ada", "Bob", "Cy"}) {
		t.Errorf("assistive technology reads %v", read)
	}

	// Without the z-index, the later one is on top, and nothing else
	// changes.
	z = false
	tt.Frame()
	for i, a := range avatars {
		if a.x != layout[i] {
			t.Errorf("avatar %d moved to %v without its z-index", i, a.x)
		}
	}
	if c := tt.Image().RGBAAt(24, 16); !isColor(c, zGreen) {
		t.Errorf("without a z-index, the overlap paints %v", c)
	}
	if !slices.Equal(tt.rt.focusOrder, order) {
		t.Errorf("Tab goes %v without the z-index, %v with", tt.rt.focusOrder, order)
	}
}

// A cell of a grid growing as the pointer is over it paints over the cell
// after it, and takes the pointer there.
func TestZIndexHoverGrows(t *testing.T) {
	var cells [2]*node
	var clicks [2]int
	tt := coreNewTester(func(c *context) {
		coreRow(c).Children(func() {
			for i, col := range []Color{zRed, zBlue} {
				cell := coreButtonBase(c).Size(40, 40).Shrink(0).Background(col)
				if cell.Hovered() {
					cell.Scale(1.5, 1.5).ZIndex(1)
				}
				if cell.Clicked() {
					clicks[i]++
				}
				cells[i] = cell
			}
		})
	}, 100, 60)
	tt.Move(10, 20)
	tt.Frame()
	if c := tt.Image().RGBAAt(45, 20); !isColor(c, zRed) {
		t.Errorf("the hovered cell grown over its neighbour paints %v there", c)
	}
	tt.ClickAt(45, 20)
	if clicks != [2]int{1, 0} {
		t.Errorf("clicks on the grown cell over its neighbour: %v", clicks)
	}
}

// Siblings with the same z-index paint as without one: those in the flow
// in tree order, then the absolute ones; a negative one goes under its
// siblings, over its parent's background.
func TestZIndexOrder(t *testing.T) {
	type kid struct {
		z   int
		abs bool
	}
	cases := []struct {
		name string
		kids []kid
		want Color // painted where all overlap
	}{
		{"equal, the last on top", []kid{{1, false}, {1, false}, {1, false}}, zBlue},
		{"equal, absolute over the flow", []kid{{0, false}, {0, true}, {0, false}}, zGreen},
		{"higher over absolute", []kid{{1, false}, {0, true}, {0, false}}, zRed},
		{"negative under the rest", []kid{{0, false}, {0, false}, {-1, false}}, zGreen},
		{"equal non-zero, absolute over the flow", []kid{{2, true}, {2, false}, {0, false}}, zRed},
	}
	for _, tc := range cases {
		var hits []int
		tt := coreNewTester(func(c *context) {
			coreBox(c).Size(60, 60).Background(RGB(9, 9, 9)).Children(func() {
				for i, k := range tc.kids {
					e := coreButtonBase(c).Size(40, 40).Background([]Color{zRed, zGreen, zBlue}[i]).ZIndex(k.z)
					if k.abs {
						e.Absolute().Top(0).Left(0)
					} else {
						e.Margin(0, 0, -40, 0) // all on top of each other
					}
					if e.Clicked() {
						hits = append(hits, i)
					}
				}
			})
		}, 60, 60)
		tt.Frame()
		if c := tt.Image().RGBAAt(20, 20); !isColor(c, tc.want) {
			t.Errorf("%s: paints %v, want %v", tc.name, c, tc.want)
		}
		tt.ClickAt(20, 20)
		want := slices.Index([]Color{zRed, zGreen, zBlue}, tc.want)
		if !slices.Equal(hits, []int{want}) {
			t.Errorf("%s: the click goes to %v, want %d, which paints on top", tc.name, hits, want)
		}
	}
	// Under all its siblings, still over its parent's background.
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(60, 60).Background(RGB(9, 9, 9)).Children(func() {
			coreBox(c).Size(40, 40).Background(zRed).ZIndex(-5)
		})
	}, 60, 60)
	tt.Frame()
	if c := tt.Image().RGBAAt(20, 20); !isColor(c, zRed) {
		t.Errorf("a negative z-index paints under its parent's background: %v", c)
	}
}

// A z-index orders the element among its siblings alone: what is inside a
// raised element moves with it, and does not reach past its parent's
// siblings.
func TestZIndexSiblingsOnly(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(60, 60).Children(func() {
			coreBox(c).Size(40, 40).Margin(0, 0, -40, 0).Children(func() {
				// Raised inside its parent, under its parent's later sibling.
				coreBox(c).Size(40, 40).Background(zRed).ZIndex(100)
			})
			coreBox(c).Size(40, 40).Background(zBlue)
		})
	}, 60, 60)
	tt.Frame()
	if c := tt.Image().RGBAAt(20, 20); !isColor(c, zBlue) {
		t.Errorf("a z-index reached past its parent: %v", c)
	}
}

// The window's content stays under its overlays, whatever its z-index.
func TestZIndexUnderOverlays(t *testing.T) {
	var clicks [2]int
	tt := coreNewTester(func(c *context) {
		b := coreButtonBase(c).Size(40, 40).Background(zRed).ZIndex(100)
		if b.Clicked() {
			clicks[0]++
		}
		coreOverlay(c, func() {
			o := coreButtonBase(c).Size(40, 40).Background(zBlue)
			if o.Clicked() {
				clicks[1]++
			}
		})
	}, 60, 60)
	tt.Frame()
	if c := tt.Image().RGBAAt(20, 20); !isColor(c, zBlue) {
		t.Errorf("the content raised paints over the overlay: %v", c)
	}
	tt.ClickAt(20, 20)
	if clicks != [2]int{0, 1} {
		t.Errorf("clicks: %v, want the overlay's", clicks)
	}
}

// A sticky child keeps painting above the flow, and a z-index orders it
// among the others as theirs.
func TestZIndexSticky(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			coreBox(c).Height(20).Shrink(0).Background(zRed).Sticky().Top(0)
			coreBox(c).Height(300).Shrink(0).Background(zBlue).ZIndex(1)
		})
	}, 100, 100)
	s.Y = 50
	tt.Frame()
	if c := tt.Image().RGBAAt(50, 5); !isColor(c, zBlue) {
		t.Errorf("the raised element in the flow paints under the sticky one: %v", c)
	}
}

// A cached parent paints its children anew as their z-indexes change.
func TestZIndexCache(t *testing.T) {
	first := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(40, 40).Cache().Children(func() {
			coreBox(c).Size(40, 40).Margin(0, 0, -40, 0).Background(zRed).ZIndex(map[bool]int{true: 1}[first])
			coreBox(c).Size(40, 40).Background(zBlue)
		})
	}, 60, 60)
	tt.Frame()
	if c := tt.Image().RGBAAt(20, 20); !isColor(c, zBlue) {
		t.Errorf("paints %v", c)
	}
	first = true
	tt.Frame()
	if c := tt.Image().RGBAAt(20, 20); !isColor(c, zRed) {
		t.Errorf("the cached parent paints %v once the first is raised", c)
	}
}

// Without a z-index other than 0, nothing changes: the elements paint the
// same, and are not ordered.
func TestZIndexUnused(t *testing.T) {
	build := func(z bool) *Tester {
		return coreNewTester(func(c *context) {
			coreColumn(c).Dividers(1, zGreen).Children(func() {
				for i := range 3 {
					e := coreButtonBase(c).Height(20).Background(zRed).Label("row")
					if i == 1 {
						e.Absolute().Top(5).Left(5).Size(20, 20)
					}
					if z {
						e.ZIndex(0)
					}
				}
			})
		}, 100, 100)
	}
	plain, zero := build(false), build(true)
	plain.Frame()
	zero.Frame()
	if d := sameOps(withoutEmptyClips(zero.rt.scene.Ops), withoutEmptyClips(plain.rt.scene.Ops)); d != "" {
		t.Errorf("a z-index of 0 changes the painting: %s", d)
	}
	for e := zero.rt.c.root; e != nil; e = e.first {
		if e.zOrder {
			t.Errorf("a z-index of 0 orders the children of element %x", e.id)
		}
	}
}

// Ordering children by z-index allocates nothing a frame once warm, as
// painting in tree order does not.
func TestZIndexAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("measures")
	}
	build := func(z bool) *Tester {
		return coreNewTester(func(c *context) {
			coreRow(c).Children(func() {
				for i := range 20 {
					e := coreBox(c).Size(20, 20).Margin(0, 0, 0, -5).Background(zRed)
					if z {
						e.ZIndex(20 - i)
					}
					coreBox(c).Size(5, 5).Absolute().Top(0).Left(float32(i))
				}
			})
		}, 300, 40)
	}
	plain, raised := build(false), build(true)
	for range 3 {
		plain.Frame()
		raised.Frame()
	}
	a, b := testing.AllocsPerRun(50, plain.Frame), testing.AllocsPerRun(50, raised.Frame)
	if b > a {
		t.Errorf("a frame allocates %.0f times with z-indexes, %.0f without", b, a)
	}
}

// What a z-index raises over a web view covers it, and a web view raised
// over its siblings, absolute ones too, is covered by none of them.
func TestZIndexWebView(t *testing.T) {
	page := &testWebView{"page"}
	raiseView := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 100).Children(func() {
			coreButtonBase(c).Size(40, 40).Margin(0, 0, -40, 0).Label("badge").ZIndex(map[bool]int{false: 1}[raiseView])
			coreWebView(c, page).Size(100, 100).ZIndex(map[bool]int{true: 1}[raiseView])
			coreButtonBase(c).Size(30, 30).Absolute().Top(60).Left(60).Label("corner")
		})
	}, 100, 100)
	tt.Frame()
	p, ok := placed(tt, page)
	if !ok || len(p.covers) != 2 || !p.covers[0].Contains(70, 70) || !p.covers[1].Contains(20, 20) {
		t.Errorf("the badge raised over the web view, and the corner after it: covers %+v", p.covers)
	}
	if tt.WebViewAt(20, 20) != nil || tt.WebViewAt(50, 50) != page {
		t.Error("the pointer goes to the web view under the badge raised over it")
	}
	raiseView = true
	tt.Frame()
	if p, _ := placed(tt, page); len(p.covers) != 0 {
		t.Errorf("the web view raised over the badge: covers %+v", p.covers)
	}
	if tt.WebViewAt(20, 20) != page || tt.WebViewAt(70, 70) != page {
		t.Error("the pointer misses the web view raised over the badge and the corner")
	}
}

// The copy of an element leaving with its Exit keeps the z-index the
// element had, as a dialog raised over the page fading out over it, and
// goes under its siblings of the same z-index alone.
func TestZIndexExit(t *testing.T) {
	cases := []struct {
		name     string
		z, blueZ int
		flow     bool // the element was in its parent's flow
		over     bool // the copy fades over blue, rather than under it
	}{
		{"raised over its sibling", 1, 0, false, true},
		{"raised over its sibling, from the flow", 1, 0, true, true},
		{"raised over a sibling lowered", 1, -1, false, true},
		{"at 0 over a sibling lowered", 0, -1, false, true},
		{"under its sibling of the same z-index", 2, 2, false, false},
		{"under its sibling raised higher", 1, 2, false, false},
		{"lowered under its sibling", -1, 0, false, false},
		{"at 0 under its sibling", 0, 0, true, false},
	}
	for _, tc := range cases {
		show := true
		tt := coreNewTester(func(c *context) {
			coreBox(c).Size(100, 100).Background(RGB(255, 255, 255)).Children(func() {
				coreBox(c).Absolute().Left(0).Top(0).Size(50, 100).Background(zBlue).ZIndex(tc.blueZ)
				if show {
					red := coreBox(c).Key("red").Size(100, 100).Background(zRed).ZIndex(tc.z).
						Transition(ElementTransition{Duration: time.Second, Ease: func(x float32) float32 { return x },
							Exit: &Motion{Opacity: 0}})
					if !tc.flow {
						red.Absolute().Left(0).Top(0)
					}
				}
			})
		}, 100, 100)
		tt.Frame()
		show = false
		tt.Frame()
		tt.Advance(500 * time.Millisecond)
		tt.Frame()
		img := tt.Image()
		if c := img.RGBAAt(75, 50); c.R != 255 || c.G < 100 || c.G > 155 {
			t.Errorf("%s: the copy, half faded over white, paints %v", tc.name, c)
		}
		c := img.RGBAAt(25, 50)
		if over := c.R > 100 && c.R < 155 && c.B > 100 && c.B < 155; over != tc.over {
			t.Errorf("%s: over blue, the copy half faded paints %v, want it over blue: %v", tc.name, c, tc.over)
		}
		if !tc.over && !isColor(c, zBlue) {
			t.Errorf("%s: blue over the copy paints %v", tc.name, c)
		}
	}
}
