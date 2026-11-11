package ui

import (
	"fmt"
	"image"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// lcNode is an element of a random tree: its kind, a choice for each of
// the styles lcStyles sets, and its children.
type lcNode struct {
	key      int // keyed when > 0
	kind     int // lcBox, lcText, lcIcon, lcImage
	text     int
	styles   []uint8
	children []*lcNode
	value    string
	lists    [2]ListState // each Tester's
}

const (
	lcBox = iota
	lcText
	lcIcon
	lcImage
	// Those that lay out by more than they are built with (sameLayout),
	// holding their children as boxes do.
	lcInput
	lcList
	lcForm
	lcToolbar
	lcPopover
	lcParagraph
	lcRich
	lcAttached
)

var lcTexts = []string{"", "7", "Hello", "a line of text that wraps in a narrow box", "two\nlines", "مرحبا بالعالم", "1234567890"}

// lcStyles are the styles of the random trees' elements, each a list of
// choices, the first of which sets nothing.
var lcStyles = [][]func(e *node){
	{nil, func(e *node) { e.Row() }, func(e *node) { e.Grid().Columns(2) }, func(e *node) { e.Row().Wrap() },
		func(e *node) { e.Column().Reverse() }, func(e *node) { e.Row().WrapReverse() },
		func(e *node) { e.Grid().ColumnTracks(Fixed(40), Fr(1), Fr(2)) }},
	{nil, func(e *node) { e.Padding(4) }, func(e *node) { e.Padding(2, 8, 0, 3) }, func(e *node) { e.PaddingStart(9) }},
	{nil, func(e *node) { e.Margin(5) }, func(e *node) { e.Margin(0, Auto) }, func(e *node) { e.MarginStart(7) }, func(e *node) { e.Margin(Auto, 3) }},
	{nil, func(e *node) { e.Gap(6) }, func(e *node) { e.GapX(3).GapY(11) }},
	{nil, func(e *node) { e.Width(50) }, func(e *node) { e.Width(130) }, func(e *node) { e.WidthPercent(40) }, func(e *node) { e.FillWidth() }},
	{nil, func(e *node) { e.Height(30) }, func(e *node) { e.Height(90) }, func(e *node) { e.HeightPercent(50) }, func(e *node) { e.FillHeight() }},
	{nil, func(e *node) { e.MinWidth(60) }, func(e *node) { e.MaxWidth(80) }, func(e *node) { e.MinHeight(20) },
		func(e *node) { e.MaxHeight(40) }, func(e *node) { e.MaxWidthPercent(50) }, func(e *node) { e.MinHeightPercent(30) },
		func(e *node) { e.MaxWidthPercent(60).MaxWidth(90) }, func(e *node) { e.MinWidth(30).MinWidthPercent(40) }},
	{nil, func(e *node) { e.Grow(1) }, func(e *node) { e.Grow(2) }},
	{nil, func(e *node) { e.Shrink(0) }, func(e *node) { e.Shrink(3) }},
	{nil, func(e *node) { e.Basis(40) }, func(e *node) { e.BasisPercent(30) }},
	{nil, func(e *node) { e.Justify(Center) }, func(e *node) { e.Justify(End) }, func(e *node) { e.Justify(SpaceBetween) }, func(e *node) { e.Justify(SpaceEvenly) }},
	{nil, func(e *node) { e.AlignItems(Center) }, func(e *node) { e.AlignItems(End) }, func(e *node) { e.AlignItems(Start) },
		func(e *node) { e.AlignItems(Baseline) }, func(e *node) { e.AlignItems(Stretch) }},
	{nil, func(e *node) { e.AlignSelf(Center) }, func(e *node) { e.AlignSelf(End) }, func(e *node) { e.AlignSelf(Baseline) }, func(e *node) { e.JustifySelf(End) }},
	{nil, func(e *node) { e.Absolute().Top(4).Left(6) }, func(e *node) { e.Absolute().Right(3).Bottom(9) },
		func(e *node) { e.Absolute().LeftPercent(10).RightPercent(20) }, func(e *node) { e.Absolute().Start(5).Top(0).Bottom(0) }},
	{nil, func(e *node) { e.flags |= flagScrollY }, func(e *node) { e.flags |= flagScrollX }, func(e *node) { e.ClipX() }, func(e *node) { e.Clip() }},
	{nil, func(e *node) { e.AspectRatio(1.5) }},
	{nil, func(e *node) { e.FontSize(18) }, func(e *node) { e.Bold() }, func(e *node) { e.FontSize(11).LineHeight(1.6) },
		func(e *node) { e.LetterSpacing(1) }, func(e *node) { e.TextAlign(Center) }, func(e *node) { e.FixedLineHeight(24) }, func(e *node) { e.Italic() }},
	// What does not lay out.
	{nil, func(e *node) { e.Translate(5, 7) }, func(e *node) { e.Background(RGB(200, 0, 0)) }, func(e *node) { e.TextColor(RGB(0, 0, 200)) },
		func(e *node) { e.Opacity(0.5) }, func(e *node) { e.Scale(1.2, 1.2) }, func(e *node) { e.Radius(4).Shadow(0, 2, 4, 0, RGBA(0, 0, 0, 0.3)) }, func(e *node) { e.Underline() },
		func(e *node) { e.ClipInsetPercent(0, 30, 0, 0, 4) }, func(e *node) { e.MaskLinearGradient(LinearGradient{From: RGB(0, 0, 0), Angle: 90}) }},
	{nil, func(e *node) { e.Direction(RightToLeft) }, func(e *node) { e.Direction(LeftToRight) }},
	{nil, func(e *node) { e.MaxLines(1) }, func(e *node) { e.NoWrap() }, func(e *node) { e.SingleLine() }, func(e *node) { e.MaxLines(2).Ellipsis("~") },
		func(e *node) { e.Balance() }},
	{nil, func(e *node) { e.ColumnSpan(2) }, func(e *node) { e.ColumnStart(2) }, func(e *node) { e.RowStart(2).RowSpan(2) }},
	{nil, func(e *node) { e.BorderWidth(3) }, func(e *node) { e.BorderWidth(1, 0, 6, 2) }},
	{nil, func(e *node) { e.Top(5) }, func(e *node) { e.Left(-3) }, func(e *node) { e.End(4) },
		func(e *node) { e.Sticky().Top(5) }, func(e *node) { e.Sticky().Bottom(3).Start(2) }},
	{nil, func(e *node) { e.Invisible() }, func(e *node) { e.Hidden(true) }},
	{nil, func(e *node) {
		e.Transition(ElementTransition{Position: true, Size: true, Duration: 100 * time.Millisecond})
	}, func(e *node) {
		e.Transition(ElementTransition{Exit: &Motion{Opacity: 0}, Enter: &Motion{Opacity: 0}, Duration: 100 * time.Millisecond})
	}},
}

// lcModel is a random tree changing at random, with what the elements
// share.
type lcModel struct {
	rng     *rand.Rand
	root    *lcNode
	keys    int
	bitmaps [2]*Bitmap
	icon    *SVG
	w, h    int
	// extra styles an element further, when set.
	extra func(n *lcNode, e *node)
}

func newLCModel(seed uint64) *lcModel {
	m := &lcModel{rng: rand.New(rand.NewPCG(seed, 7)), w: 400, h: 300}
	m.bitmaps[0] = NewBitmap(image.NewRGBA(image.Rect(0, 0, 20, 10)))
	m.bitmaps[1] = NewBitmap(image.NewRGBA(image.Rect(0, 0, 8, 30)))
	icon, err := ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 12"><rect width="24" height="12"/></svg>`))
	if err != nil {
		panic(err)
	}
	m.icon = icon
	m.root = m.newNode(lcBox)
	for i := 0; i < 12; i++ {
		m.insert()
	}
	return m
}

func (m *lcModel) newNode(kind int) *lcNode {
	n := &lcNode{kind: kind, text: m.rng.IntN(len(lcTexts)), styles: make([]uint8, len(lcStyles))}
	if m.rng.IntN(3) == 0 {
		m.keys++
		n.key = m.keys
	}
	for i := range n.styles {
		if m.rng.IntN(4) == 0 {
			n.styles[i] = uint8(m.rng.IntN(len(lcStyles[i])))
		}
	}
	return n
}

// all returns the nodes of the tree, with their parents.
func (m *lcModel) all() (nodes, parents []*lcNode) {
	var walk func(n, p *lcNode)
	walk = func(n, p *lcNode) {
		nodes, parents = append(nodes, n), append(parents, p)
		for _, c := range n.children {
			walk(c, n)
		}
	}
	walk(m.root, nil)
	return nodes, parents
}

func (m *lcModel) boxes() []*lcNode {
	nodes, _ := m.all()
	var boxes []*lcNode
	for _, n := range nodes {
		if n.kind == lcBox || n.kind >= lcList && n.kind != lcRich {
			boxes = append(boxes, n)
		}
	}
	return boxes
}

func (m *lcModel) insert() {
	boxes := m.boxes()
	p := boxes[m.rng.IntN(len(boxes))]
	kind := lcBox
	switch v := m.rng.IntN(14); {
	case v < 4:
		kind = lcText
	case v == 4:
		kind = lcIcon
	case v == 5:
		kind = lcImage
	case v > 7:
		kind = lcInput + m.rng.IntN(lcAttached-lcInput+1)
	}
	if p.kind == lcParagraph {
		kind = lcText // only text goes inside text
	}
	at := m.rng.IntN(len(p.children) + 1)
	p.children = append(p.children[:at], append([]*lcNode{m.newNode(kind)}, p.children[at:]...)...)
}

// mutate changes the tree at random, and tells how.
func (m *lcModel) mutate() string {
	nodes, parents := m.all()
	i := m.rng.IntN(len(nodes))
	n, p := nodes[i], parents[i]
	switch v := m.rng.IntN(20); {
	case v < 9:
		s := m.rng.IntN(len(lcStyles))
		n.styles[s] = uint8(m.rng.IntN(len(lcStyles[s])))
		return fmt.Sprintf("style %d of %p to %d", s, n, n.styles[s])
	case v < 12:
		n.text = m.rng.IntN(len(lcTexts))
		return "text"
	case v < 14:
		m.insert()
		return "insert"
	case v < 15 && p != nil:
		for j, c := range p.children {
			if c == n {
				p.children = append(p.children[:j], p.children[j+1:]...)
				break
			}
		}
		return "remove"
	case v < 16 && p != nil && len(p.children) > 1:
		a, b := m.rng.IntN(len(p.children)), m.rng.IntN(len(p.children))
		p.children[a], p.children[b] = p.children[b], p.children[a]
		return "swap"
	case v < 17:
		if n.key > 0 {
			n.key = 0
		} else {
			m.keys++
			n.key = m.keys
		}
		return "key"
	case v < 18:
		sizes := [][2]int{{20, 10}, {8, 30}, {16, 16}, {1, 40}}
		s := sizes[m.rng.IntN(len(sizes))]
		m.bitmaps[m.rng.IntN(2)].Update(image.NewRGBA(image.Rect(0, 0, s[0], s[1])))
		return "bitmap"
	case v < 19:
		m.w, m.h = 200+m.rng.IntN(400), 150+m.rng.IntN(300)
		return "window"
	}
	return "nothing"
}

func (m *lcModel) build(c *context, tester int) {
	var build func(n *lcNode)
	build = func(n *lcNode) {
		var e *node
		if n.key > 0 {
			c.keySet, c.nextKey = true, n.key
		}
		switch n.kind {
		case lcBox:
			e = coreBox(c)
		case lcText:
			e = coreText(c, lcTexts[n.text])
		case lcIcon:
			e = coreIcon(c, m.icon)
		case lcImage:
			e = coreImage(c, m.bitmaps[n.text%2])
		case lcInput:
			n.value = lcTexts[n.text]
			e = coreTextInput(c, &n.value)
		case lcList:
			e = coreList(c, &n.lists[tester], 2+n.text, func(i int) {
				coreText(c, lcTexts[(i+n.text)%len(lcTexts)])
			})
		case lcForm:
			// Its labels as wide as the widest.
			e = coreForm(c, func() {
				coreField(c, lcTexts[n.text], func() {
					for _, ch := range n.children {
						build(ch)
					}
				})
				coreField(c, lcTexts[int(n.styles[1])%len(lcTexts)], func() {
					coreText(c, "value")
				})
			})
		case lcToolbar:
			e = coreToolbar(c, func() {
				for _, ch := range n.children {
					build(ch)
				}
			})
		case lcPopover:
			e = coreBox(c)
			target := e
			coreOverlay(c, func() {
				coreBox(c).Padding(3).AttachTo(target, AnchorBottom, AnchorTop).Children(func() {
					coreText(c, lcTexts[n.text])
				})
			})
		case lcParagraph:
			e = coreText(c, lcTexts[n.text])
		case lcRich:
			// Its spans' styles change apart from its text.
			e = coreRichText(c, Span{Text: lcTexts[n.text]}, Span{Text: " bold", Weight: 400 + 300*int(n.styles[2]%2), Size: float32(10 + n.styles[3])})
		case lcAttached:
			e = coreBox(c).Attach(AnchorTopRight, AnchorCenter)
		}
		for i, choice := range n.styles {
			if f := lcStyles[i][choice]; f != nil {
				f(e)
			}
		}
		if m.extra != nil {
			m.extra(n, e)
		}
		if n.kind == lcBox || n.kind == lcPopover || n.kind == lcParagraph || n.kind == lcAttached {
			e.Children(func() {
				for _, ch := range n.children {
					build(ch)
				}
			})
		}
	}
	build(m.root)
}

// sameBoxes reports where the elements of two frames differ in their
// boxes or in their text's layouts, copies of elements leaving with an
// exit transition included.
func sameBoxes(a, b *node, path string) string {
	switch {
	case a.id != b.id || a.kind != b.kind:
		return fmt.Sprintf("%s: not the same elements: kind %d/%d, %d/%d children, list %v, leaving %d/%d", path, a.kind, b.kind, a.nchild, b.nchild, a.list != nil, a.leaving, b.leaving)
	case a.x != b.x || a.y != b.y || a.w != b.w || a.h != b.h:
		return fmt.Sprintf("%s: box %v,%v %v×%v, want %v,%v %v×%v", path, a.x, a.y, a.w, a.h, b.x, b.y, b.w, b.h)
	case a.contentW != b.contentW || a.contentH != b.contentH:
		return fmt.Sprintf("%s: content %v×%v, want %v×%v", path, a.contentW, a.contentH, b.contentW, b.contentH)
	case (a.tl == nil) != (b.tl == nil) || a.tl != nil && (a.tl.Params != b.tl.Params || a.tl.Width != b.tl.Width || a.tl.Height != b.tl.Height):
		return path + ": text laid out otherwise"
	}
	ca, cb := a.first, b.first
	for i := 0; ca != nil && cb != nil; i++ {
		if d := sameBoxes(ca, cb, path+"/"+strconv.Itoa(i)); d != "" {
			return d
		}
		ca, cb = ca.next, cb.next
	}
	if ca != nil || cb != nil {
		return path + ": not the same children"
	}
	return ""
}

// lcTester is coreNewTester at the time now, for Testers that do the same
// things at the same times.
func lcTester(view func(c *context), width, height int, now time.Time) *Tester {
	h := &headless{w: float32(width), h: float32(height), scale: 1}
	t := &Tester{rt: newRuntime(view, h), h: h, now: now}
	t.rt.clock = t.Now
	h.clock = t.Now
	t.rt.collect = true
	t.rt.strict = true
	t.settle()
	return t
}

// The layout cache lays out what laying out anew does, as random trees
// change at random frame after frame: their elements' styles, texts,
// children, keys and images, and the window.
func TestLayoutCacheMatches(t *testing.T) {
	seeds, frames := 40, 80
	if testing.Short() {
		seeds = 15
	}
	var reused, computed uint64
	for seed := range uint64(seeds) {
		m := newLCModel(seed)
		view := func(tester int) func(c *context) {
			return func(c *context) {
				coreBox(c).Fill().Children(func() { m.build(c, tester) })
			}
		}
		start := time.Unix(1e9, 0)
		cached := lcTester(view(0), m.w, m.h, start)
		fresh := lcTester(view(1), m.w, m.h, start)
		fresh.rt.noLayoutCache = true
		var log []string
		for f := range frames {
			if m.rng.IntN(3) > 0 {
				for range 1 + m.rng.IntN(3) {
					log = append(log, m.mutate())
				}
			} else {
				log = append(log, "unchanged")
			}
			for _, tt := range []*Tester{cached, fresh} {
				tt.h.w, tt.h.h = float32(m.w), float32(m.h)
			}
			before := cached.rt.computed
			if m.rng.IntN(4) == 0 {
				// What moves takes its next step.
				cached.Advance(30 * time.Millisecond)
				fresh.Advance(30 * time.Millisecond)
			} else {
				cached.Frame()
				fresh.Frame()
			}
			computed += uint64(cached.rt.computed - before)
			reused += uint64(cached.rt.laySeq) - uint64(cached.rt.computed)
			if d := sameBoxes(cached.rt.c.root, fresh.rt.c.root, "root"); d != "" {
				if len(log) > 6 {
					log = log[len(log)-6:]
				}
				t.Fatalf("seed %d, frame %d, after %s: %s", seed, f, strings.Join(log, ", "), d)
			}
		}
	}
	if reused == 0 {
		t.Errorf("the cache reused no layout in %d laid out", computed)
	}
}

// The elements that a frame changes nothing of, or only moves, keep their
// layout; those whose style or text changed, or that inherit a text
// style changed, are laid out anew.
func TestLayoutCacheReuses(t *testing.T) {
	var dx float32
	size, label := float32(13), "Row"
	color := RGB(0, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().Clip().Children(func() {
			coreColumn(c).Translate(0, dx).FontSize(size).Children(func() {
				for i := range 100 {
					coreRow(c).Height(28).Gap(8).TextColor(color).Children(func() {
						coreText(c, label+" "+strconv.Itoa(i))
						coreSpacer(c)
						coreText(c, strconv.Itoa(i*7%100))
					})
				}
			})
		})
	}, 400, 300)
	frame := func(what string, want func(n uint32) bool) {
		t.Helper()
		before := tt.rt.computed
		tt.Frame()
		if n := tt.rt.computed - before; !want(n) {
			t.Errorf("%s: laid out %d elements anew", what, n)
		}
	}
	none := func(n uint32) bool { return n == 0 }
	frame("an unchanged frame", none)
	dx = -40
	frame("a translated column", none)
	color = RGB(255, 0, 0)
	frame("a text color", none)
	label = "Item"
	frame("the rows' first texts", func(n uint32) bool { return n >= 200 && n < 410 })
	size = 15
	frame("the column's font size", func(n uint32) bool { return n >= 400 })
	frame("an unchanged frame again", none)
	tt.h.w = 500
	// The rows widen; their texts keep their widths.
	frame("a wider window", func(n uint32) bool { return n >= 200 && n < 300 })
}

// What lays an element out is in its layoutStyle, or compared with it
// (sameLayout); node's other fields do not lay it out, or make it
// volatile there. A field added to node must be placed in one or the
// other.
func TestLayoutInputs(t *testing.T) {
	// Fields of node that the layout reads besides layoutStyle, compared
	// or made volatile by sameLayout, and those it writes.
	read := "flags ts cols rows spans image svg placeOn list form colFit overflow gridFit leaving ghosts attach popover " +
		"first last next nchild parent st c measured tl lay laid x y w h contentW contentH frags runes paragraph " +
		"ownText merged spansKey collapsed rowMinW field hidden placed flips originAttached originX originY originSet edgeMargin edgeMarginSet"
	// Fields the layout never reads.
	other := "epoch key valueInput id depth track transform translatePct world content contentSet " +
		"pose poseAnim poseSet poseM poseFade reposed ink noCull backface bg fill grad stripes borderC borderStyle radius " +
		"shadows material effect effecting cache grads blur opacity opacitySet undimmed cursor paintFn hitFn objectFn styleFn paintAfterFn " +
		"inputFn observeFn textClient caret takesText noIME copyData webView maximizeButton fit gray objectPos dismissed rotate label widget role " +
		"checked expanded busy current autoComplete orient tip highlighted focusHighlights keepOpen accValue accRange hasRange accFill accStep barInset scrollBase hideScrollbars keepScrollbars barsFrom snap snapAlign " +
		"listRow rowIndex rowsOf focusGroup groupSelects segment activeDescendant nameFrom nameJoin labelledBy focusTo " +
		"keyShortcuts accGrid accSet " +
		"setPos setSize choosesItems search description invalid describedBy errorMessage followX sort level expandable " +
		"chooseMany serial ignoreInput accessHidden accessDisabled themed lift ringOffset ringColor ringSet clipping zIndex zOrder " +
		"roleDescription controls hasPopup"
	known := map[string]bool{}
	for _, f := range strings.Fields(read + " " + other) {
		known[f] = true
	}
	ty := reflect.TypeFor[node]()
	for i := range ty.NumField() {
		f := ty.Field(i)
		if f.Name == "layoutStyle" {
			continue
		}
		if !known[f.Name] {
			t.Errorf("node.%s: say whether layout reads it (TestLayoutInputs, layoutcache.go)", f.Name)
		}
	}
}

// Each change lays out anew what it changes, as laying out without the
// cache does.
func TestLayoutCacheInvalidates(t *testing.T) {
	bitmap := NewBitmap(image.NewRGBA(image.Rect(0, 0, 20, 10)))
	cases := []struct {
		name   string
		build  func(c *context, v bool)
		change func()
	}{
		{"a span's weight", func(c *context, v bool) {
			coreRow(c).Children(func() {
				coreRichText(c, Span{Text: "plain "}, Span{Text: "bold", Weight: map[bool]int{false: 400, true: 800}[v]})
				coreText(c, "after")
			})
		}, nil},
		{"a bitmap's size", func(c *context, v bool) {
			coreRow(c).Children(func() { coreImage(c, bitmap); coreText(c, "after") })
		}, func() { bitmap.Update(image.NewRGBA(image.Rect(0, 0, 50, 30))) }},
		{"an inherited font size", func(c *context, v bool) {
			e := coreColumn(c).Children(func() {
				coreRow(c).Children(func() { coreText(c, "deep"); coreIcon(c, nil) })
			})
			if v {
				e.FontSize(20)
			}
		}, nil},
		{"a direction", func(c *context, v bool) {
			e := coreRow(c).Width(300)
			if v {
				e.Direction(RightToLeft)
			}
			e.Children(func() { coreText(c, "one"); coreText(c, "two") })
		}, nil},
		{"a grid's tracks", func(c *context, v bool) {
			coreGrid(c).ColumnTracks(Fixed(map[bool]float32{false: 40, true: 90}[v]), Fr(1)).Children(func() {
				coreText(c, "a")
				coreText(c, "b")
			})
		}, nil},
		{"keyed children reordered", func(c *context, v bool) {
			coreRow(c).Children(func() {
				order := []string{"wide one", "b"}
				if v {
					order = []string{"b", "wide one"}
				}
				for _, s := range order {
					coreText(c, s).Key(s)
				}
			})
		}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := false
			view := func(c *context) { coreColumn(c).Children(func() { tc.build(c, v) }) }
			start := time.Unix(1e9, 0)
			cached, fresh := lcTester(view, 300, 200, start), lcTester(view, 300, 200, start)
			fresh.rt.noLayoutCache = true
			cached.Frame()
			if tc.change != nil {
				tc.change()
			} else {
				v = true
			}
			before := cached.rt.computed
			cached.Frame()
			fresh.Frame()
			if d := sameBoxes(cached.rt.c.root, fresh.rt.c.root, "root"); d != "" {
				t.Error(d)
			}
			if cached.rt.computed == before {
				t.Error("nothing was laid out anew")
			}
		})
	}
}

// A text element laid out anew, as the rows of a List are each frame,
// takes the layout its state held of the same text rather than asking the
// text system, and lets it go as the text system's generation changes.
func TestHeldText(t *testing.T) {
	var s ListState
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 3, func(i int) {
			keep(i, coreTextf(c, "Row %d", i))
		}).Grow(1)
	}, 200, 200)
	e := keyed(tt, 1)
	held := e.st.text
	if held == nil || !slices.Contains(held.at[:], e.tl) {
		t.Fatalf("the state of the row's text holds %v, not its layout %p", held, e.tl)
	}
	// A copy of the layout, taller, which only the state holds.
	i := slices.Index(held.at[:], e.tl)
	copied := *e.tl
	copied.Height += 10
	held.at[i] = &copied
	tt.Frame()
	if e := keyed(tt, 1); e.tl != &copied || e.h != copied.Height {
		t.Fatalf("laid out again, the row's text took %p (%v high), not the layout held, %v high", e.tl, e.h, copied.Height)
	}
	held.gen++
	tt.Frame()
	if e := keyed(tt, 1); e.tl == &copied || e.h == copied.Height {
		t.Errorf("the text system's generation changed, and the row's text kept the layout held")
	}
}

// A grid measures the widths of its items only where its columns read
// them: those spanning a FitContent column, and those spanning an Fr
// column where the grid's width is not known, as CSS's minmax(0, 1fr), or
// a MinContent Fr column.
func TestGridMeasures(t *testing.T) {
	var fit bool
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			g := coreGrid(c).Width(300)
			if fit {
				g.ColumnTracks(FitContent(), Fr(1))
			} else {
				g.ColumnTracks(Fixed(40), Fr(1))
			}
			g.Children(func() {
				keep("first", coreText(c, "first"))
				keep("second", coreText(c, "second"))
			})
			// A grid as wide as its content asks.
			coreRow(c).Children(func() {
				coreGrid(c).Columns(2).Children(func() {
					keep("sized", coreText(c, "sized"))
					coreText(c, "by content")
				})
			})
		})
	}, 400, 300)
	measured := func(k string) bool { return keyed(tt, k).leafOK }
	if measured("first") || measured("second") || !measured("sized") {
		t.Errorf("fixed and Fr columns: measured the first %v, the second %v; Fr columns sized by content: %v", measured("first"), measured("second"), measured("sized"))
	}
	fit = true
	tt.Frame()
	if !measured("first") || measured("second") {
		t.Errorf("FitContent and Fr columns: measured the first %v, the second %v", measured("first"), measured("second"))
	}
	tt = coreNewTester(func(c *context) {
		coreGrid(c).Width(300).ColumnTracks(Fr(1).Min(40), Fr(1).MinContent()).Children(func() {
			keep("floor", coreText(c, "floor"))
			keep("content", coreText(c, "content"))
		})
	}, 400, 300)
	if measured("floor") || !measured("content") {
		t.Errorf("Fr columns with a minimum and with MinContent: measured the first %v, the second %v", measured("floor"), measured("content"))
	}
}
