//go:build darwin

package darwin

import "testing"

// A press that has not moved when its drag is asked goes to the window
// server, which moves the window: the drags that still reach the app are
// the window's, and MyGo leaves the window alone.
func TestWindowDragNative(t *testing.T) {
	var d windowDrag
	d.down(1, NSPoint{100, 50})
	if got := d.start(1, NSPoint{10, 500}); got != startNative {
		t.Fatalf("start = %v, want the window server's move", got)
	}
	// The window server moved the window by the time the drag came.
	if eat, move := d.dragged(NSPoint{140, 70}, NSPoint{50, 480}); !eat || move {
		t.Errorf("dragged while the window server moves the window = eat %v, move %v; want eaten, no move", eat, move)
	}
	if d.phase != dragServer {
		t.Errorf("phase = %v, want the window server's", d.phase)
	}
	// Even back where it was, the window stays the window server's.
	if eat, move := d.dragged(NSPoint{108, 50}, NSPoint{10, 500}); !eat || move {
		t.Errorf("dragged back to the start = eat %v, move %v; want eaten, no move", eat, move)
	}
	d.up()
	if d.phase != dragIdle {
		t.Errorf("phase after the release = %v, want idle", d.phase)
	}
}

// Issue #528: a quick press and drag has moved by the time the page asks
// for the drag. MyGo moves the window itself, at once to where the pointer
// is, and then with it.
func TestWindowDragMovedBeforeAsked(t *testing.T) {
	var d windowDrag
	d.down(1, NSPoint{100, 50})
	for _, p := range []NSPoint{{104, 52}, {120, 60}} {
		if eat, move := d.dragged(p, NSPoint{10, 500}); eat || move {
			t.Fatalf("a drag before the drag is asked = eat %v, move %v; want the content's", eat, move)
		}
	}
	if got := d.start(1, NSPoint{10, 500}); got != startManual {
		t.Fatalf("start = %v, want MyGo's move", got)
	}
	// Right 20, down 10 in Core Graphics' space: down 10 in AppKit's.
	if got, want := d.target(), (NSPoint{30, 490}); got != want {
		t.Errorf("target = %v, want %v", got, want)
	}
	d.placed(d.target())
	eat, move := d.dragged(NSPoint{150, 40}, NSPoint{30, 490})
	if !eat || !move {
		t.Errorf("dragged = eat %v, move %v; want eaten and moved", eat, move)
	}
	if got, want := d.target(), (NSPoint{60, 510}); got != want {
		t.Errorf("target = %v, want %v", got, want)
	}
}

// The window server begins and says so before its frame moves: the frame
// AppKit gives lags a tenth of a second, and MyGo took the window too, which
// shook as both moved it. The press's drags stay the window server's.
func TestWindowDragNativeWillMove(t *testing.T) {
	var d windowDrag
	origin := NSPoint{10, 500}
	d.down(1, NSPoint{100, 50})
	d.start(1, origin)
	d.willMove(2)
	if d.phase != dragNative {
		t.Fatalf("phase after another window's move = %v, want native", d.phase)
	}
	d.willMove(1)
	for _, p := range []NSPoint{{108, 50}, {160, 70}} {
		if eat, move := d.dragged(p, origin); !eat || move {
			t.Errorf("dragged %v with the window's frame still = eat %v, move %v; want eaten, no move", p, eat, move)
		}
	}
	if d.phase != dragServer {
		t.Errorf("phase = %v, want the window server's", d.phase)
	}
}

// The window server did not begin: the press's drags still reach the app
// and the window stays. Past the slop, once the window server has had a few
// drags to say it begins, MyGo moves the window itself.
func TestWindowDragNativeDidNotBegin(t *testing.T) {
	var d windowDrag
	origin := NSPoint{10, 500}
	d.down(1, NSPoint{100, 50})
	if got := d.start(1, origin); got != startNative {
		t.Fatalf("start = %v, want the window server's move", got)
	}
	if eat, move := d.dragged(NSPoint{101, 51}, origin); !eat || move {
		t.Errorf("dragged within the slop = eat %v, move %v; want eaten, no move", eat, move)
	}
	if eat, move := d.dragged(NSPoint{104, 50}, origin); !eat || move {
		t.Errorf("dragged past the slop at once = eat %v, move %v; want eaten, no move", eat, move)
	}
	if eat, move := d.dragged(NSPoint{108, 50}, origin); !eat || !move {
		t.Errorf("dragged past the slop = eat %v, move %v; want eaten and moved", eat, move)
	}
	if d.phase != dragManual {
		t.Errorf("phase = %v, want manual", d.phase)
	}
	if got, want := d.target(), (NSPoint{18, 500}); got != want {
		t.Errorf("target = %v, want %v", got, want)
	}
}

// A quick hand is past the slop with the first drag after the ask, before
// the window server says it begins: MyGo took the window too, and the
// window shook as both moved it. MyGo waits, and the window server's
// windowWillMove: leaves the press to it.
func TestWindowDragNativeQuickBeforeWillMove(t *testing.T) {
	var d windowDrag
	origin := NSPoint{10, 500}
	d.down(1, NSPoint{100, 50})
	d.start(1, origin)
	if eat, move := d.dragged(NSPoint{112, 54}, origin); !eat || move {
		t.Errorf("first drag past the slop = eat %v, move %v; want eaten, no move", eat, move)
	}
	d.willMove(1)
	for _, p := range []NSPoint{{124, 58}, {136, 62}, {148, 66}} {
		if eat, move := d.dragged(p, origin); !eat || move {
			t.Errorf("dragged %v after the window server began = eat %v, move %v; want eaten, no move", p, eat, move)
		}
	}
}

// The window server says it begins after MyGo took over, and the drag goes
// on slowly: the window was where MyGo put it within the rounding at every
// drag, so MyGo kept moving it beside the window server, which shook it.
// windowWillMove: leaves the press to the window server.
func TestWindowDragManualWillMove(t *testing.T) {
	var d windowDrag
	origin := NSPoint{10, 500}
	d.down(1, NSPoint{100, 50})
	d.start(1, origin)
	for _, p := range []NSPoint{{102, 50}, {104, 50}, {106, 50}} {
		d.dragged(p, origin)
	}
	if d.phase != dragManual {
		t.Fatalf("phase = %v, want manual", d.phase)
	}
	d.placed(d.target())
	d.willMove(2)
	if d.phase != dragManual {
		t.Fatalf("phase after another window's move = %v, want manual", d.phase)
	}
	d.willMove(1)
	// A point a drag: the window server's frame is never 2 points from MyGo's.
	for i, p := range []NSPoint{{107, 50}, {108, 50}, {109, 50}} {
		if eat, move := d.dragged(p, NSPoint{16 + float64(i), 500}); !eat || move {
			t.Errorf("dragged %v after the window server began = eat %v, move %v; want eaten, no move", p, eat, move)
		}
	}
	if d.phase != dragServer {
		t.Errorf("phase = %v, want the window server's", d.phase)
	}
}

// The window server begins after MyGo took over: both would move the
// window, which shakes. Once the window is not where MyGo put it, MyGo
// leaves it to the window server.
func TestWindowDragNativeBeganLate(t *testing.T) {
	var d windowDrag
	origin := NSPoint{10, 500}
	d.down(1, NSPoint{100, 50})
	d.start(1, origin)
	d.dragged(NSPoint{102, 50}, origin)
	d.dragged(NSPoint{104, 50}, origin)
	if _, move := d.dragged(NSPoint{108, 50}, origin); !move {
		t.Fatal("dragged past the slop did not move the window")
	}
	d.placed(d.target())
	if _, move := d.dragged(NSPoint{120.4, 50}, NSPoint{18, 500}); !move {
		t.Fatal("dragged with the window where MyGo put it did not move the window")
	}
	d.placed(d.target())
	// AppKit rounds the origin MyGo put.
	if eat, move := d.dragged(NSPoint{130, 51}, NSPoint{30, 500}); !eat || !move {
		t.Errorf("dragged with the window's origin rounded = eat %v, move %v; want eaten and moved", eat, move)
	}
	d.placed(d.target())
	// The window server moved it to the pointer.
	for _, p := range []NSPoint{{160, 54}, {170, 55}} {
		if eat, move := d.dragged(p, NSPoint{70, 496}); !eat || move {
			t.Errorf("dragged with the window server moving the window = eat %v, move %v; want eaten, no move", eat, move)
		}
	}
	if d.phase != dragServer {
		t.Errorf("phase = %v, want the window server's", d.phase)
	}
}

// A drag is asked once a press, only for the window pressed, and only
// while the button is down.
func TestWindowDragAsked(t *testing.T) {
	var d windowDrag
	if got := d.start(1, NSPoint{}); got != startNone {
		t.Errorf("start without a press = %v, want none", got)
	}
	d.down(1, NSPoint{100, 50})
	if got := d.start(2, NSPoint{}); got != startNone {
		t.Errorf("start for another window = %v, want none", got)
	}
	if got := d.start(1, NSPoint{}); got != startNative {
		t.Errorf("start = %v, want the window server's move", got)
	}
	if got := d.start(1, NSPoint{}); got != startNone {
		t.Errorf("start again = %v, want none", got)
	}
	d.down(1, NSPoint{100, 50})
	d.up()
	if got := d.start(1, NSPoint{}); got != startNone {
		t.Errorf("start after the release = %v, want none", got)
	}
	// Drags of no press are the content's.
	if eat, move := d.dragged(NSPoint{200, 200}, NSPoint{}); eat || move {
		t.Errorf("dragged without a press = eat %v, move %v; want the content's", eat, move)
	}
	// A hand on a touchpad trembles: still enough for the window server.
	d.down(1, NSPoint{100, 50})
	d.dragged(NSPoint{100.5, 50.25}, NSPoint{})
	if got := d.start(1, NSPoint{}); got != startNative {
		t.Errorf("start after a tremble = %v, want the window server's move", got)
	}
}

func TestBelowMenuBar(t *testing.T) {
	for _, c := range []struct {
		origin NSPoint
		height float64
		want   NSPoint
	}{
		{NSPoint{10, 300}, 400, NSPoint{10, 300}}, // top at 700
		{NSPoint{10, 500}, 400, NSPoint{10, 475}}, // top at 900, past 875
		{NSPoint{-40, 475}, 400, NSPoint{-40, 475}},
	} {
		if got := belowMenuBar(c.origin, c.height, 875); got != c.want {
			t.Errorf("belowMenuBar(%v, %v, 875) = %v, want %v", c.origin, c.height, got, c.want)
		}
	}
}
