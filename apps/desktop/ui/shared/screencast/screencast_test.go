package screencast

import (
	"testing"
	"time"
)

func TestShiftKeepsTheStageCovered(t *testing.T) {
	const width, height = 340, 500
	center := Point{width / 2, height / 2}
	for _, l := range []Lens{{1, center}, {1.5, Point{0, 0}}, {1.5, Point{width, height}}, {1.55, Point{257, 292}}, {1.2, Point{170, 274}}} {
		x, y := Shift(l, width, height)
		// Where the stage's corners land, scaled around its center.
		left := width/2 - width/2*l.Zoom + x
		top := height/2 - height/2*l.Zoom + y
		right := width/2 + width/2*l.Zoom + x
		bottom := height/2 + height/2*l.Zoom + y
		if left > 0.01 || top > 0.01 || right < width-0.01 || bottom < height-0.01 {
			t.Errorf("%+v uncovers the stage: %.1f %.1f %.1f %.1f", l, left, top, right, bottom)
		}
	}
	if x, y := Shift(Lens{1.5, center}, width, height); x != 0 || y != 0 {
		t.Errorf("centered lens shifts %v %v", x, y)
	}
}

func TestSampleHoldsAndEases(t *testing.T) {
	keys := []Key[Point]{{0, Point{0, 0}}, {100, Point{0, 0}}, {300, Point{20, 40}}}
	at := func(ms int) Point { return Sample(keys, time.Duration(ms)*time.Millisecond, MixPoint) }
	if at(50) != (Point{}) || at(100) != (Point{}) {
		t.Errorf("holds between equal keys: %+v %+v", at(50), at(100))
	}
	if mid := at(200); mid.X <= 0 || mid.X >= 20 || mid.Y <= 0 || mid.Y >= 40 {
		t.Errorf("eases between keys: %+v", mid)
	}
	if at(300) != (Point{20, 40}) || at(900) != (Point{20, 40}) {
		t.Errorf("stays on the last key: %+v %+v", at(300), at(900))
	}
}

func TestCursorClicks(t *testing.T) {
	path := []Key[Point]{{0, Point{10, 10}}, {400, Point{50, 30}}, {900, Point{50, 30}}}
	clicks := []time.Duration{500 * time.Millisecond}
	at := func(ms int) Cursor { return CursorAt(path, clicks, time.Duration(ms)*time.Millisecond) }
	if before := at(450); before.Press != 0 || before.Ripple >= 0 {
		t.Errorf("no click yet: %+v", before)
	}
	if during := at(550); during.Press <= 0 || during.Ripple <= 0 || during.RippleAt != (Point{50, 30}) {
		t.Errorf("pressed with a ring where it clicked: %+v", during)
	}
	if after := at(960); after.Press != 0 || after.Ripple >= 0 {
		t.Errorf("the ring is gone: %+v", after)
	}
}

func TestHead(t *testing.T) {
	if got := Head("语音输入", 0.5); got != "语音" {
		t.Errorf("Head = %q", got)
	}
}
