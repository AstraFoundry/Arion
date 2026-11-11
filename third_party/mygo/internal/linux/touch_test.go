//go:build linux && (amd64 || arm64)

package linux

import (
	"math"
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestTouchGesture checks what fingers on a touchscreen do: taps click,
// a finger dragging scrolls the content along with it, two pinch and
// scroll (#797).
func TestTouchGesture(t *testing.T) {
	type touch struct {
		kind byte // b begin, m move, e end, c cancel
		seq  uintptr
		x, y float64
		ms   uint32
	}
	down := func(x, y float64, button int) platform.SurfaceEvent {
		return platform.SurfaceEvent{Kind: platform.PointerDown, X: x, Y: y, Button: button}
	}
	up := func(x, y float64, button int) platform.SurfaceEvent {
		return platform.SurfaceEvent{Kind: platform.PointerUp, X: x, Y: y, Button: button}
	}
	scroll := func(x, y, dx, dy float64) platform.SurfaceEvent {
		return platform.SurfaceEvent{Kind: platform.PointerScroll, X: x, Y: y, DX: dx, DY: dy, Precise: true}
	}
	zoom := func(x, y, scale float64) platform.SurfaceEvent {
		return platform.SurfaceEvent{Kind: platform.PointerZoom, X: x, Y: y, Scale: scale}
	}
	leave := platform.SurfaceEvent{Kind: platform.PointerLeave}
	for _, tc := range []struct {
		name    string
		touches []touch
		want    []platform.SurfaceEvent
	}{
		{"tap", []touch{{kind: 'b', seq: 1, x: 10, y: 20}, {kind: 'e', seq: 1, x: 10, y: 20, ms: 100}},
			[]platform.SurfaceEvent{down(10, 20, 0), up(10, 20, 0), leave}},
		{"tap within the slop", []touch{{kind: 'b', seq: 1, x: 10, y: 20}, {kind: 'm', seq: 1, x: 14, y: 24}, {kind: 'e', seq: 1, x: 15, y: 25, ms: 100}},
			[]platform.SurfaceEvent{down(10, 20, 0), up(10, 20, 0), leave}},
		{"long press", []touch{{kind: 'b', seq: 1, x: 10, y: 20, ms: 1000}, {kind: 'e', seq: 1, x: 10, y: 20, ms: 1000 + longPress}},
			[]platform.SurfaceEvent{down(10, 20, 1), up(10, 20, 1), leave}},
		{"long press as the clock wraps", []touch{{kind: 'b', seq: 1, x: 10, y: 20, ms: 1<<32 - 100}, {kind: 'e', seq: 1, x: 10, y: 20, ms: 1000}},
			[]platform.SurfaceEvent{down(10, 20, 1), up(10, 20, 1), leave}},
		{"canceled tap", []touch{{kind: 'b', seq: 1, x: 10, y: 20}, {kind: 'c', seq: 1, x: 10, y: 20, ms: 100}},
			[]platform.SurfaceEvent{leave}},
		// Past the slop the content under the finger follows it, from
		// where it landed: dragging up scrolls down the content.
		{"drag", []touch{{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'm', seq: 1, x: 100, y: 95}, {kind: 'm', seq: 1, x: 100, y: 90}, {kind: 'm', seq: 1, x: 103, y: 70}, {kind: 'e', seq: 1, x: 103, y: 70, ms: 100}},
			[]platform.SurfaceEvent{scroll(100, 100, 0, 10), scroll(100, 90, -3, 20), leave}},
		{"drag ending further", []touch{{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'm', seq: 1, x: 100, y: 80}, {kind: 'e', seq: 1, x: 100, y: 75}},
			[]platform.SurfaceEvent{scroll(100, 100, 0, 20), scroll(100, 80, 0, 5), leave}},
		{"pinch", []touch{
			{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'b', seq: 2, x: 200, y: 100},
			{kind: 'm', seq: 2, x: 300, y: 100},
			{kind: 'm', seq: 1, x: 0, y: 100},
			{kind: 'e', seq: 1, x: 0, y: 100}, {kind: 'e', seq: 2, x: 300, y: 100},
		}, []platform.SurfaceEvent{
			scroll(150, 100, -50, 0), zoom(200, 100, 2),
			scroll(200, 100, 50, 0), zoom(150, 100, 1.5),
			leave,
		}},
		{"two fingers scrolling", []touch{
			{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'b', seq: 2, x: 200, y: 100},
			{kind: 'm', seq: 1, x: 100, y: 60}, {kind: 'm', seq: 2, x: 200, y: 60},
			{kind: 'e', seq: 1, x: 100, y: 60}, {kind: 'e', seq: 2, x: 200, y: 60},
		}, []platform.SurfaceEvent{
			scroll(150, 100, 0, 20), zoom(150, 80, math.Hypot(100, 40)/100),
			scroll(150, 80, 0, 20), zoom(150, 60, 100/math.Hypot(100, 40)),
			leave,
		}},
		// The finger left scrolls on from where it is, without a jump.
		{"pinch, then one finger", []touch{
			{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'b', seq: 2, x: 200, y: 100},
			{kind: 'e', seq: 2, x: 200, y: 100},
			{kind: 'm', seq: 1, x: 100, y: 101},
			{kind: 'e', seq: 1, x: 100, y: 101},
		}, []platform.SurfaceEvent{scroll(100, 100, 0, -1), leave}},
		{"a second finger is no tap", []touch{
			{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'b', seq: 2, x: 200, y: 100},
			{kind: 'e', seq: 2, x: 200, y: 100}, {kind: 'e', seq: 1, x: 100, y: 100},
		}, []platform.SurfaceEvent{leave}},
		{"a third finger does nothing", []touch{
			{kind: 'b', seq: 1, x: 100, y: 100}, {kind: 'b', seq: 2, x: 200, y: 100}, {kind: 'b', seq: 3, x: 50, y: 50},
			{kind: 'm', seq: 3, x: 0, y: 0},
		}, nil},
		{"unknown sequences", []touch{{kind: 'm', seq: 9, x: 1, y: 1}, {kind: 'e', seq: 9, x: 1, y: 1}}, nil},
		{"taps in a row", []touch{
			{kind: 'b', seq: 1, x: 10, y: 20}, {kind: 'e', seq: 1, x: 10, y: 20, ms: 50},
			{kind: 'b', seq: 2, x: 12, y: 21, ms: 150}, {kind: 'e', seq: 2, x: 12, y: 21, ms: 200},
		}, []platform.SurfaceEvent{down(10, 20, 0), up(10, 20, 0), leave, down(12, 21, 0), up(12, 21, 0), leave}},
	} {
		var g touchGesture
		var got []platform.SurfaceEvent
		for _, tt := range tc.touches {
			switch tt.kind {
			case 'b':
				got = append(got, g.begin(tt.seq, tt.x, tt.y, tt.ms)...)
			case 'm':
				got = append(got, g.move(tt.seq, tt.x, tt.y)...)
			case 'e', 'c':
				got = append(got, g.end(tt.seq, tt.x, tt.y, tt.ms, tt.kind == 'c')...)
			}
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tc.name, got, tc.want)
		}
	}
}
