package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A Hidden element takes no room: its siblings close up, with one gap
// between them, in rows, columns and grids, and what sizes itself by its
// content does not count it.
func TestHiddenTakesNoRoom(t *testing.T) {
	hide := true
	view := func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreRow(c).Gap(10).Label("row").Children(func() {
				coreBox(c).Size(20, 20).Label("a")
				coreBox(c).Size(30, 30).Hidden(hide).Children(func() {
					coreBox(c).Size(5, 5).Label("inside")
				})
				coreBox(c).Size(20, 20).Label("c")
			})
			coreColumn(c).Gap(10).Children(func() {
				coreBox(c).Size(20, 20).Label("d")
				coreBox(c).Size(30, 30).Hidden(hide)
				coreBox(c).Size(20, 20).Label("f")
			})
			coreRow(c).Width(100).Justify(SpaceBetween).Children(func() {
				coreBox(c).Size(20, 20).Label("g")
				coreBox(c).Size(20, 20).Hidden(hide)
				coreBox(c).Size(20, 20).Label("i")
			})
			coreBox(c).Grid().Columns(2).Gap(10).Children(func() {
				coreBox(c).Size(20, 20).Label("j")
				coreBox(c).Size(20, 20).Hidden(hide)
				coreBox(c).Size(20, 20).Label("l")
			})
		})
	}
	tt := coreNewTester(view, 300, 300)
	b := func(label string) Rect {
		t.Helper()
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("no element %q", label)
		}
		return r
	}
	check := func() {
		t.Helper()
		if a, c := b("a"), b("c"); c.X != a.X+30 {
			t.Errorf("row: c at %v, a at %v: the hidden element took room or a gap", c.X, a.X)
		}
		if r := b("row"); r.W != 50 || r.H != 20 {
			t.Errorf("row sized by its content: %v, want 50×20", r)
		}
		if d, f := b("d"), b("f"); f.Y != d.Y+30 {
			t.Errorf("column: f at %v, d at %v", f.Y, d.Y)
		}
		if g, i := b("g"), b("i"); i.X != g.X+80 || i.Y != g.Y {
			t.Errorf("space between two, not three: g %v, i %v", g, i)
		}
		if j, l := b("j"), b("l"); l.X != j.X+30 || l.Y != j.Y {
			t.Errorf("grid: l at %v, j at %v: the hidden item took a cell", l, j)
		}
		if _, ok := tt.Find("inside"); ok {
			t.Error("an element inside a hidden one is found")
		}
	}
	check()
	// Shown, it takes its room again; hidden once more, it gives it up,
	// the layout reusing what it can between.
	hide = false
	tt.Frame()
	if a, c := b("a"), b("c"); c.X != a.X+70 {
		t.Errorf("shown: c at %v, a at %v", c.X, a.X)
	}
	if r := b("inside"); r.W != 5 {
		t.Errorf("shown: inside %v", r)
	}
	hide = true
	tt.Frame()
	check()
	tt.Frame()
	check()
}

// A Hidden element neither paints nor takes the pointer, Tab passes it,
// and the focus leaves it as it hides.
func TestHiddenInput(t *testing.T) {
	hideB, hideC := true, false
	clicked := map[string]int{}
	painted := 0
	tt := coreNewTester(func(c *context) {
		coreRow(c).Children(func() {
			for _, l := range []string{"a", "b", "c"} {
				e := coreButtonBase(c).Size(40, 20).Label(l).Background(RGB(255, 0, 0))
				switch l {
				case "b":
					e.Hidden(hideB).Draw(func(p *Painter, r Rect) {
						painted++
						p.AnimationFrame()
					})
				case "c":
					e.Hidden(hideC)
				}
				if e.Clicked() {
					clicked[l]++
				}
			}
		})
	}, 200, 100)
	if painted != 0 {
		t.Error("a hidden element painted")
	}
	if tt.FramePending() {
		t.Error("a hidden element's drawing asks for frames")
	}
	if c := tt.Image().RGBAAt(80+10, 10); c.R == 255 && c.G == 0 {
		t.Error("something shows where the hidden button would be after a and c")
	}
	// c is where b would be.
	tt.ClickAt(50, 10)
	if clicked["b"] != 0 || clicked["c"] != 1 {
		t.Errorf("clicks %v: want c clicked, in b's place", clicked)
	}
	tt.Key(0, KeyTab)
	tt.Key(0, KeyTab)
	if !tt.Focused("c") {
		t.Error("Tab did not pass the hidden button")
	}
	hideC = true
	tt.Frame()
	if tt.Focused("c") || tt.rt.focused != 0 {
		t.Error("the focus stayed in an element that hid")
	}
	tt.Key(0, KeyEnter)
	if clicked["c"] != 1 {
		t.Error("Enter pressed a hidden button")
	}
}

// Assistive technology does not read a Hidden element, nor what is inside
// it.
func TestHiddenAccessibility(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreButtonBase(c).Size(40, 20).Label("shown")
		coreColumn(c).Hidden(true).Children(func() {
			coreButtonBase(c).Size(40, 20).Label("gone")
			coreText(c, "a hidden text")
		})
	}, 200, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	accessNode(t, tree, platform.RoleButton, "shown")
	for _, n := range tree.Nodes {
		if n.Label == "gone" || n.Label == "a hidden text" {
			t.Errorf("a hidden element is in the tree: %+v", n)
		}
	}
}

// What is inside a Hidden element is still built, so its state lives on
// while it hides: a scroll offset, a local value.
func TestHiddenKeepsState(t *testing.T) {
	hide := false
	var scroll, local uint64
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Hidden(hide).Children(func() {
			sc := coreScroll(c).Size(100, 100).Label("scroll").Children(func() {
				coreBox(c).Size(100, 400)
			})
			scroll = sc.id
			n := coreLocal(sc, "n", func() int { return 0 })
			*n++
			local = uint64(*n)
		})
	}, 200, 200)
	tt.Scroll(50, 50, 0, 120)
	before := tt.rt.states[scroll].scrollY
	if before != 120 {
		t.Fatalf("scrolled to %v", before)
	}
	hide = true
	for range 3 {
		tt.Frame()
	}
	if _, ok := tt.Find("scroll"); ok {
		t.Error("the hidden scroll container is found")
	}
	if s := tt.rt.states[scroll]; s == nil || s.scrollY != before {
		t.Fatalf("hidden, the scroll container lost its offset: %+v", s)
	}
	counted := local
	hide = false
	tt.Frame()
	if s := tt.rt.states[scroll]; s.scrollY != before {
		t.Errorf("shown again at %v, want %v", s.scrollY, before)
	}
	if local <= counted {
		t.Errorf("the local value started again: %d after %d", local, counted)
	}
}

// What is attached to a Hidden element hides with it.
func TestHiddenPopover(t *testing.T) {
	hide := false
	tt := coreNewTester(func(c *context) {
		coreRow(c).Hidden(hide).Children(func() {
			b := coreButtonBase(c).Size(40, 20).Label("anchor")
			coreOverlay(c, func() {
				coreButtonBase(c).Size(60, 30).Label("panel").AttachTo(b, AnchorBottomLeft, AnchorTopLeft)
			})
		})
	}, 200, 200)
	if _, ok := tt.Find("panel"); !ok {
		t.Fatal("no panel")
	}
	hide = true
	tt.Frame()
	if _, ok := tt.Find("panel"); ok {
		t.Error("a panel attached to a hidden element shows")
	}
}

// A Hidden text inside another leaves the paragraph.
func TestHiddenInline(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreText(c, "Hello").Children(func() {
			coreText(c, " cruel").Hidden(true)
			coreText(c, " world")
		})
	}, 300, 100)
	if !tt.HasText("Hello world") || tt.HasText("cruel") {
		t.Errorf("texts %q", tt.Texts())
	}
}

// A Hidden dialog keeps neither the focus nor the window's elements to
// itself.
func TestHiddenDialog(t *testing.T) {
	hide := true
	tt := coreNewTester(func(c *context) {
		coreButtonBase(c).Size(40, 20).Label("outside")
		coreOverlay(c, func() {
			coreBox(c).Fill().Modal().Hidden(hide).Children(func() {
				coreButtonBase(c).Size(40, 20).Label("inside")
			})
		})
	}, 200, 200)
	tt.Key(0, KeyTab)
	if !tt.Focused("outside") {
		t.Error("a hidden dialog kept the focus from the window")
	}
	hide = false
	tt.Frame()
	if !tt.Focused("inside") {
		t.Error("the dialog shown did not take the focus")
	}
}
