//go:build !mygo_noinspector

package ui

import (
	"image/color"
	"slices"
	"testing"
)

// showLayout chooses the badge of the element keyed k.
func showLayout(t *testing.T, tt *Tester, k string) {
	t.Helper()
	in := &tt.rt.insp
	for id, key := range in.keys {
		if key == k {
			in.toggleLayout(tt.rt, id)
			tt.Frame()
			tt.Frame()
			return
		}
	}
	t.Fatalf("no element is keyed %q", k)
}

func overlayOf(t *testing.T, tt *Tester) *inspOverlay {
	t.Helper()
	if n := len(tt.rt.insp.overlays); n != 1 {
		t.Fatalf("%d layouts show", n)
	}
	return &tt.rt.insp.overlays[0]
}

func sameTracks(got []inspTrack, want ...inspTrack) bool {
	return slices.EqualFunc(got, want, func(a, b inspTrack) bool {
		return abs32(a.at-b.at) < 0.01 && abs32(a.size-b.size) < 0.01
	})
}

func gridView(extra func(*node)) func(*context) {
	return func(c *context) {
		g := coreGrid(c).Key("grid").Size(340, 220).ColumnTracks(Fixed(100), Fixed(100)).RowTracks(Fixed(50), Fixed(50)).GapX(20).GapY(10)
		if extra != nil {
			extra(g)
		}
		g.Children(func() {
			for range 4 {
				coreBox(c).Size(10, 10).Background(RGB(0, 128, 0))
			}
		})
	}
}

func TestInspectorShowsGridTracks(t *testing.T) {
	tt, _ := inspected(gridView(nil))
	openInspector(tt)
	showLayout(t, tt, "grid")
	o := overlayOf(t, tt)
	if !sameTracks(o.tracks.cols, inspTrack{0, 100}, inspTrack{120, 100}) {
		t.Errorf("columns: %v", o.tracks.cols)
	}
	if !sameTracks(o.tracks.rows, inspTrack{0, 50}, inspTrack{60, 50}) {
		t.Errorf("rows: %v", o.tracks.rows)
	}
	// The gaps, as tall and as wide as the grid.
	want := []Rect{{100, 0, 20, 220}, {0, 50, 340, 10}}
	if !slices.Equal(o.room, want) {
		t.Errorf("gaps: %v, want %v", o.room, want)
	}
	// They show, in the grid's color, over what was the white between cells.
	img := tt.Image()
	base := img.RGBAAt(110, 200)
	if base == (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("the gap isn't painted: %v", base)
	}
}

// Tracks move as the grid places them, not only as they are sized.
func TestInspectorGridTracksArePlaced(t *testing.T) {
	tt, _ := inspected(gridView(func(g *node) { g.Justify(Center).AlignContent(End) }))
	openInspector(tt)
	showLayout(t, tt, "grid")
	o := overlayOf(t, tt)
	if !sameTracks(o.tracks.cols, inspTrack{60, 100}, inspTrack{180, 100}) {
		t.Errorf("columns: %v", o.tracks.cols)
	}
	if !sameTracks(o.tracks.rows, inspTrack{110, 50}, inspTrack{170, 50}) {
		t.Errorf("rows: %v", o.tracks.rows)
	}
	// Where the layout put the cells' children says the same.
	var at [][2]float32
	for c := o.e.first; c != nil; c = c.next {
		at = append(at, [2]float32{c.x - o.e.x, c.y - o.e.y})
	}
	want := [][2]float32{{60, 110}, {180, 110}, {60, 170}, {180, 170}}
	if !slices.Equal(at, want) {
		t.Errorf("the children are at %v, want %v", at, want)
	}
}

// A layout cached before the badge is clicked still gives its tracks, and
// the grid's children stay where they were.
func TestInspectorGridTracksOfACachedLayout(t *testing.T) {
	var child *node
	tt, _ := inspected(func(c *context) {
		coreGrid(c).Key("grid").Size(300, 100).ColumnTracks(Fixed(100), Fixed(100)).AlignItems(Baseline).Children(func() {
			child = coreText(c, "One")
			coreText(c, "Two").FontSize(24)
		})
	})
	openInspector(tt)
	tt.Frame()
	tt.Frame()
	x, y := child.x, child.y
	showLayout(t, tt, "grid")
	if o := overlayOf(t, tt); !sameTracks(o.tracks.cols, inspTrack{0, 100}, inspTrack{100, 100}) {
		t.Errorf("columns: %v", o.tracks.cols)
	}
	if child.x != x || child.y != y {
		t.Errorf("the child moved from %v, %v to %v, %v", x, y, child.x, child.y)
	}
}

func TestInspectorGridTracksRightToLeft(t *testing.T) {
	tt, _ := inspected(gridView(func(g *node) { g.Direction(RightToLeft) }))
	openInspector(tt)
	showLayout(t, tt, "grid")
	o := overlayOf(t, tt)
	// From the left, as they paint: the first column is the rightmost.
	if !sameTracks(o.tracks.cols, inspTrack{120, 100}, inspTrack{240, 100}) {
		t.Errorf("columns: %v", o.tracks.cols)
	}
}

func TestInspectorFlexRoom(t *testing.T) {
	tt, _ := inspected(func(c *context) {
		coreRow(c).Key("row").Size(300, 40).Gap(10).Children(func() {
			coreBox(c).Size(50, 40)
			coreBox(c).Size(50, 40)
		})
	})
	openInspector(tt)
	showLayout(t, tt, "row")
	// The gap, then the free space after the children.
	want := []Rect{{50, 0, 10, 40}, {110, 0, 190, 40}}
	if o := overlayOf(t, tt); !slices.Equal(o.room, want) {
		t.Errorf("room: %v, want %v", o.room, want)
	}
}

func TestInspectorFlexRoomOfAColumn(t *testing.T) {
	tt, _ := inspected(func(c *context) {
		coreColumn(c).Key("col").Size(100, 100).Justify(End).Children(func() {
			coreBox(c).Size(100, 20)
			coreBox(c).Size(100, 20)
		})
	})
	openInspector(tt)
	showLayout(t, tt, "col")
	want := []Rect{{0, 0, 100, 60}}
	if o := overlayOf(t, tt); !slices.Equal(o.room, want) {
		t.Errorf("room: %v, want %v", o.room, want)
	}
}

func TestInspectorFlexRoomOfWrappedLines(t *testing.T) {
	tt, _ := inspected(func(c *context) {
		coreRow(c).Key("row").Size(100, 100).Wrap().GapX(10).GapY(5).AlignContent(Start).Children(func() {
			coreBox(c).Size(60, 20)
			coreBox(c).Size(60, 20)
		})
	})
	openInspector(tt)
	showLayout(t, tt, "row")
	want := []Rect{
		{60, 0, 40, 20},  // beside the first line's child
		{0, 20, 100, 5},  // between the lines
		{60, 25, 40, 20}, // beside the second's
		{0, 45, 100, 55}, // below them
	}
	if o := overlayOf(t, tt); !slices.Equal(o.room, want) {
		t.Errorf("room: %v, want %v", o.room, want)
	}
}

// What scrolls out of a container isn't shown outside it.
func TestInspectorFlexRoomWhileScrolled(t *testing.T) {
	var scroll ScrollState
	tt, _ := inspected(func(c *context) {
		coreScroll(c).Key("list").Size(100, 100).TrackScroll(&scroll).Gap(10).Children(func() {
			coreBox(c).Size(100, 80).Shrink(0)
			coreBox(c).Size(100, 80).Shrink(0)
		})
	})
	openInspector(tt)
	showLayout(t, tt, "list")
	scroll.Y = 40
	tt.Frame()
	tt.Frame()
	want := []Rect{{0, 40, 100, 10}}
	if o := overlayOf(t, tt); !slices.Equal(o.room, want) {
		t.Errorf("room: %v, want %v", o.room, want)
	}
}

func TestInspectorLayoutBadges(t *testing.T) {
	tt, _ := inspected(func(c *context) {
		coreGrid(c).Key("grid").Size(100, 100).Columns(2).Children(func() { coreBox(c).Size(10, 10) })
		coreRow(c).Key("row").Children(func() {
			coreBox(c).Size(10, 10)
			coreBox(c).Size(10, 10)
		})
		// One child says nothing of its layout.
		coreRow(c).Key("single").Children(func() { coreBox(c).Size(10, 10) })
	})
	openInspector(tt)
	tt.Frame()
	kinds := map[any]uint8{}
	for _, n := range tt.rt.insp.nodes {
		if k, ok := tt.rt.insp.keys[n.id]; ok {
			kinds[k] = n.layout
		}
	}
	if kinds["grid"] != inspGrid || kinds["row"] != inspFlex || kinds["single"] != inspNoLayout {
		t.Errorf("kinds: %v", kinds)
	}
	if !tt.HasText("grid") || !tt.HasText("flex") {
		t.Fatalf("no badges: %q", tt.Texts())
	}
	if err := tt.Click("grid"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if n := len(tt.rt.insp.overlays); n != 1 || !tt.rt.insp.overlays[0].e.grid {
		t.Fatalf("clicking the badge shows %d layouts", n)
	}
	if !tt.rt.noLayoutCache {
		t.Error("the layout cache stays on with a badge on")
	}
	if err := tt.Click("grid"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if n := len(tt.rt.insp.overlays); n != 0 {
		t.Errorf("clicking it again leaves %d layouts", n)
	}
	if tt.rt.noLayoutCache {
		t.Error("the layout cache stays off with no badge on")
	}
}

func TestInspectorClosingDropsLayouts(t *testing.T) {
	tt, _ := inspected(gridView(nil))
	openInspector(tt)
	showLayout(t, tt, "grid")
	tt.rt.toggleInspector()
	tt.Frame()
	if tt.rt.insp.layoutOn != nil || tt.rt.insp.overlays != nil || tt.rt.noLayoutCache {
		t.Errorf("after closing: %v, %v, %v", tt.rt.insp.layoutOn, tt.rt.insp.overlays, tt.rt.noLayoutCache)
	}
}
