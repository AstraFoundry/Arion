package gpu

import (
	"fmt"
	"image"
	"testing"
)

func TestAges(t *testing.T) {
	var forgot []uintptr
	a := Ages{Forget: func(id uintptr) { forgot = append(forgot, id) }}
	caret := image.Rect(10, 10, 12, 30)
	clock := image.Rect(100, 0, 140, 20)
	both := fmt.Sprint([]image.Rectangle{caret, clock})
	// Three buffers in turn, each new: drawn whole.
	for id := uintptr(1); id <= 3; id++ {
		a.Add([]image.Rectangle{caret}, false)
		if _, known := a.Take(id); known {
			t.Fatalf("buffer %d, new, is known", id)
		}
	}
	// Buffer 1 lacks the two frames drawn since it was: the caret's, apart
	// from the clock's.
	a.Add([]image.Rectangle{clock}, false)
	if got, known := a.Take(1); !known || fmt.Sprint(got) != both {
		t.Errorf("buffer 1 lacks %v (known %v), want %v", got, known, both)
	}
	// Buffer 2 lacks the frame drawn into 3, the clock's and nothing.
	a.Add(nil, false)
	if got, known := a.Take(2); !known || fmt.Sprint(got) != both {
		t.Errorf("buffer 2 lacks %v (known %v), want %v", got, known, both)
	}
	// A frame changing everything makes every buffer drawn whole.
	a.Add(nil, true)
	if _, known := a.Take(3); known {
		t.Error("buffer 3 is known after a frame changing everything")
	}
	a.Add([]image.Rectangle{caret}, false)
	if _, known := a.Take(1); known {
		t.Error("buffer 1 is known after a frame changing everything")
	}
	// A fifth buffer takes the place of the one drawn least recently.
	a.Take(4)
	a.Take(5)
	if len(forgot) != 1 || forgot[0] != 2 {
		t.Errorf("forgot %v, want [2]", forgot)
	}
	a.Reset()
	if len(forgot) != 5 || a.Has(1) {
		t.Errorf("after Reset, forgot %v and tracks buffer 1: %v", forgot, a.Has(1))
	}
}

func TestAddRect(t *testing.T) {
	var rects []image.Rectangle
	// Rectangles apart stay apart; one meeting two takes them in.
	rects = addRect(rects, image.Rect(0, 0, 10, 10))
	rects = addRect(rects, image.Rect(50, 0, 60, 10))
	if len(rects) != 2 {
		t.Fatalf("two apart: %v", rects)
	}
	rects = addRect(rects, image.Rect(5, 5, 55, 8))
	if fmt.Sprint(rects) != fmt.Sprint([]image.Rectangle{image.Rect(0, 0, 60, 10)}) {
		t.Errorf("one meeting both: %v", rects)
	}
	// Past maxRects, the two nearest become one, and none overlap.
	rects = rects[:0]
	for i := range maxRects + 1 {
		rects = addRect(rects, image.Rect(i*100, 0, i*100+10, 10))
	}
	rects = addRect(rects, image.Rect(1000, 0, 1010, 10)) // nearest the last
	if len(rects) != maxRects {
		t.Fatalf("%d rectangles, want %d: %v", len(rects), maxRects, rects)
	}
	for i, r := range rects {
		for _, o := range rects[i+1:] {
			if r.Overlaps(o) {
				t.Errorf("%v overlaps %v", r, o)
			}
		}
	}
}

func TestPartial(t *testing.T) {
	var p Part
	// Within the frame.
	if !p.Set([]image.Rectangle{image.Rect(-5, 10, 20, 30)}, 100, 100, nil) || fmt.Sprint(p.Rects) != "[{0 10 20 30}]" {
		t.Errorf("Set = %v", p.Rects)
	}
	// Two corners stay apart.
	if !p.Set([]image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(90, 90, 100, 100)}, 100, 100, nil) || fmt.Sprint(p.Rects) != "[{0 0 10 10} {90 90 100 100}]" {
		t.Errorf("two corners: %v", p.Rects)
	}
	// Half the frame or more, together, is drawn whole.
	if p.Set([]image.Rectangle{image.Rect(0, 0, 100, 30), image.Rect(0, 60, 100, 80)}, 100, 100, nil) {
		t.Error("half the frame drawn in part")
	}
	// Nothing changed in the frame: nothing to draw.
	if !p.Set([]image.Rectangle{image.Rect(200, 0, 210, 10)}, 100, 100, nil) || len(p.Rects) != 0 {
		t.Errorf("outside: %v", p.Rects)
	}
	// Grown until what it grows by is in it: a backdrop from 40 to 60
	// meets the area, an effect reading one from 0 to 45 then meets that.
	grow := func(d []image.Rectangle) []image.Rectangle {
		for _, b := range []image.Rectangle{image.Rect(40, 0, 60, 10), image.Rect(0, 0, 45, 10)} {
			for _, r := range d {
				if r.Overlaps(b) {
					d = append(d, b)
					break
				}
			}
		}
		return d
	}
	if !p.Set([]image.Rectangle{image.Rect(50, 0, 55, 5), image.Rect(80, 80, 90, 90)}, 100, 100, grow) || fmt.Sprint(p.Rects) != "[{80 80 90 90} {0 0 60 10}]" {
		t.Errorf("grown: %v, want [{80 80 90 90} {0 0 60 10}]", p.Rects)
	}
}
