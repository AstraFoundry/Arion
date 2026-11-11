package ui

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// The layout benchmarks lay out trees of boxes shaped as Taffy's flexbox
// benchmarks (taffy/benches/benches/flexbox.rs) build theirs, from scratch
// each time as a frame does: no text, so that they measure flexbox alone.

// benchBox is the style of a box of a benchmark's tree.
type benchBox struct {
	row           bool
	grow, shrink  float32
	width, height length
	margin        float32
	children      []*benchBox
}

// benchTree builds the boxes of a benchmark's tree from a seeded source,
// as Taffy's tree builders do.
type benchTree struct {
	rng   *rand.Rand
	style func(r *rand.Rand) *benchBox
	count int
}

func newBenchTree(style func(r *rand.Rand) *benchBox) *benchTree {
	return &benchTree{rng: rand.New(rand.NewPCG(12345, 0)), style: style}
}

func (t *benchTree) box(children []*benchBox) *benchBox {
	b := t.style(t.rng)
	b.children = children
	t.count++
	return b
}

// randomLength is Taffy's random_dimension: auto, a length up to 500 or a
// percentage.
func randomLength(r *rand.Rand) length {
	switch v := r.Float32(); {
	case v < 0.2:
		return length{}
	case v < 0.8:
		return px(r.Float32() * 500)
	default:
		return percent(r.Float32() * 100)
	}
}

func randomBox(r *rand.Rand) *benchBox {
	return &benchBox{row: true, shrink: 1, width: randomLength(r), height: randomLength(r)}
}

func fixedBox(b benchBox) func(*rand.Rand) *benchBox {
	return func(*rand.Rand) *benchBox { c := b; return &c }
}

// deep builds n boxes in levels of branching boxes (build_deep_tree).
func (t *benchTree) deep(n, branching int) []*benchBox {
	var out []*benchBox
	if n <= branching {
		for range n {
			out = append(out, t.box(nil))
		}
		return out
	}
	for range branching {
		out = append(out, t.box(t.deep((n-branching)/branching, branching)))
	}
	return out
}

// flat builds boxes of one to four boxes until there are n
// (build_flat_hierarchy).
func (t *benchTree) flat(n int) []*benchBox {
	var out []*benchBox
	for t.count < n {
		leaves := make([]*benchBox, 1+t.rng.IntN(4))
		for i := range leaves {
			leaves[i] = t.box(nil)
		}
		out = append(out, t.box(leaves))
	}
	return out
}

// superDeep builds depth levels of a box beside perLevel-1 leaves
// (build_super_deep_hierarchy).
func (t *benchTree) superDeep(depth, perLevel int) []*benchBox {
	var level []*benchBox
	for range depth {
		next := []*benchBox{t.box(level)}
		for range perLevel - 1 {
			next = append(next, t.box(nil))
		}
		level = next
	}
	return level
}

type layoutBench struct {
	name  string
	build func() []*benchBox
}

var layoutBenches = []layoutBench{
	{"HugeNested/10000", func() []*benchBox {
		return newBenchTree(fixedBox(benchBox{width: px(10), height: px(10), grow: 1, shrink: 1, row: true})).deep(10000, 10)
	}},
	{"Wide/1000", func() []*benchBox { return newBenchTree(randomBox).flat(1000) }},
	{"Wide/10000", func() []*benchBox { return newBenchTree(randomBox).flat(10000) }},
	{"DeepRandom/4000", func() []*benchBox { return newBenchTree(randomBox).deep(4000, 2) }},
	{"DeepRandom/10000", func() []*benchBox { return newBenchTree(randomBox).deep(10000, 2) }},
	{"DeepAuto/4000", func() []*benchBox {
		return newBenchTree(fixedBox(benchBox{row: true, grow: 1, shrink: 1, margin: 10})).deep(4000, 2)
	}},
	{"DeepAuto/10000", func() []*benchBox {
		return newBenchTree(fixedBox(benchBox{row: true, grow: 1, shrink: 1, margin: 10})).deep(10000, 2)
	}},
	{"SuperDeep/100", func() []*benchBox {
		return newBenchTree(fixedBox(benchBox{row: true, grow: 1, shrink: 1, margin: 10})).superDeep(100, 3)
	}},
}

// buildBench makes the elements of the boxes under a root w×h, a row as
// Taffy's, and returns the root.
func buildBench(boxes []*benchBox, w, h float32) (*Tester, *node) {
	var root *node
	var add func(c *context, b *benchBox)
	add = func(c *context, b *benchBox) {
		e := coreBox(c)
		e.row, e.grow, e.shrink = b.row, b.grow, b.shrink
		e.width, e.height = b.width, b.height
		e.margin = [4]float32{b.margin, b.margin, b.margin, b.margin}
		if len(b.children) > 0 {
			e.Children(func() {
				for _, ch := range b.children {
					add(c, ch)
				}
			})
		}
	}
	view := func(c *context) {
		root = coreRow(c).AlignItems(Stretch).Size(w, h).Children(func() {
			for _, b := range boxes {
				add(c, b)
			}
		})
	}
	tt := coreNewTester(view, int(w), int(h))
	return tt, root
}

// forgetLayout forgets what laying the tree out measured, and that it may
// reuse what it laid out before (layoutcache.go), as the elements of a
// frame that changed them all start.
func forgetLayout(e *node) {
	e.nmeasure, e.leafOK, e.intrOK, e.lay = 0, false, 0, 0
	for c := e.first; c != nil; c = c.next {
		forgetLayout(c)
	}
}

func BenchmarkLayout(b *testing.B) {
	for _, lb := range layoutBenches {
		b.Run(lb.name, func(b *testing.B) {
			_, root := buildBench(lb.build(), 1000, 800)
			b.ResetTimer()
			for b.Loop() {
				forgetLayout(root)
				layoutBox(root, 1000, 800)
			}
		})
	}
}

// TestLayoutBenchTrees checks the trees the benchmarks lay out have the
// size of Taffy's.
func TestLayoutBenchTrees(t *testing.T) {
	var count func(bs []*benchBox) int
	count = func(bs []*benchBox) int {
		n := len(bs)
		for _, b := range bs {
			n += count(b.children)
		}
		return n
	}
	for _, lb := range layoutBenches {
		if n := count(lb.build()); n < 300 {
			t.Errorf("%s: %d boxes", lb.name, n)
		} else {
			t.Log(lb.name, fmt.Sprint(n))
		}
	}
}
